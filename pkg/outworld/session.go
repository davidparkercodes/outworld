package outworld

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/dateshift"
	"github.com/davidparkercodes/outworld/pkg/detect"
	"github.com/davidparkercodes/outworld/pkg/entity"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

type Violation struct {
	Type   entity.Type `json:"type"`
	Source string      `json:"source"`
}

type Stats struct {
	Masked      int                 `json:"masked"`
	Stripped    int                 `json:"stripped"`
	Shifted     int                 `json:"shifted"`
	Generalized int                 `json:"generalized"`
	ByType      map[entity.Type]int `json:"by_type"`
}

type LeakError struct {
	Violations []Violation
}

func (e *LeakError) Error() string {
	types := map[entity.Type]bool{}
	for _, v := range e.Violations {
		types[v.Type] = true
	}
	var names []string
	for t := range types {
		names = append(names, string(t))
	}
	sort.Strings(names)
	return fmt.Sprintf("outworld: blocked outbound payload: %d sensitive value(s) would have left unmasked (types: %s)", len(e.Violations), strings.Join(names, ", "))
}

var ErrLeak = errors.New("outworld: leak detected")

func (e *LeakError) Is(target error) bool { return target == ErrLeak }

type Session struct {
	e          *Engine
	scope      string
	offset     int
	mu         sync.Mutex
	stats      Stats
	violations []Violation
	priming    bool
	detected   map[string][]entity.Span
	avoid      map[string]bool
	dates      map[string]string
}

func (e *Engine) NewSession(scope string) *Session {
	if scope == "" {
		scope = "global"
	}
	return &Session{e: e, scope: scope, offset: e.DateOffset(scope), stats: Stats{ByType: map[entity.Type]int{}}, detected: map[string][]entity.Span{}, avoid: map[string]bool{}, dates: map[string]string{}}
}

func (s *Session) DateOffset() int { return s.offset }

func (s *Session) Stats() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.stats
	out.ByType = map[entity.Type]int{}
	for k, v := range s.stats.ByType {
		out.ByType[k] = v
	}
	return out
}

func (s *Session) Violations() []Violation {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Violation(nil), s.violations...)
}

func (s *Session) Err() error {
	v := s.Violations()
	if len(v) == 0 {
		return nil
	}
	return &LeakError{Violations: v}
}

func (s *Session) SetPriming(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.priming = on
}

func (s *Session) isPriming() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.priming
}

func (s *Session) count(t entity.Type, a entity.Action) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.priming {
		return
	}
	s.stats.ByType[t]++
	switch a {
	case entity.Mask:
		s.stats.Masked++
	case entity.Strip:
		s.stats.Stripped++
	case entity.Shift:
		s.stats.Shifted++
	case entity.Generalize:
		s.stats.Generalized++
	}
}

func (s *Session) detect(ctx context.Context, text string) ([]entity.Span, error) {
	s.mu.Lock()
	spans, ok := s.detected[text]
	s.mu.Unlock()
	if ok {
		return spans, nil
	}
	spans, err := s.e.Detect(ctx, text)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.detected[text] = spans
	for _, t := range textutil.Tokens(text) {
		if w := text[t.Start:t.End]; textutil.StartsUpper(w) {
			s.avoid[strings.ToLower(w)] = true
		}
	}
	s.mu.Unlock()
	return spans, nil
}

func (s *Session) avoids(cand string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range textutil.Tokens(cand) {
		if s.avoid[strings.ToLower(cand[t.Start:t.End])] {
			return true
		}
	}
	return false
}

const maxRemediationRounds = 4

func (s *Session) Mask(ctx context.Context, text string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return text, nil
	}
	spans, err := s.detect(ctx, text)
	if err != nil {
		return "", err
	}
	plan := s.e.plan(spans)
	if s.isPriming() {
		for _, sp := range plan {
			if s.e.policy.Action(sp.Type) == entity.Mask {
				if _, err := s.surrogate(sp); err != nil {
					return "", err
				}
			}
		}
		return text, nil
	}
	items := make([]planned, len(plan))
	for i, sp := range plan {
		items[i] = planned{span: sp}
	}
	out, ins, err := s.apply(text, nil, items)
	if err != nil {
		return "", err
	}
	for round := 0; ; round++ {
		found, kept := s.scan(ctx, out, ins)
		collisions := s.collisions(out, ins, kept)
		if len(found) == 0 && len(collisions) == 0 {
			return out, nil
		}
		fix := s.remediable(append(found, collisions...), ins)
		if len(fix) == 0 || round == maxRemediationRounds {
			s.record(found)
			return out, nil
		}
		if out, ins, err = s.apply(out, ins, fix); err != nil {
			return "", err
		}
	}
}

func (s *Session) Check(ctx context.Context, text string) {
	if s.isPriming() || strings.TrimSpace(text) == "" {
		return
	}
	found, _ := s.scan(ctx, text, nil)
	s.record(found)
}

type planned struct {
	span entity.Span
	rec  *vault.Record
}

type finding struct {
	planned
	source string
}

func (s *Session) record(found []finding) {
	if len(found) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range found {
		s.violations = append(s.violations, Violation{Type: f.span.Type, Source: f.source})
	}
}

func (s *Session) apply(text string, ins []insertion, items []planned) (string, []insertion, error) {
	type edit struct {
		start, end int
		repl       string
		typ        entity.Type
	}
	eds := make([]edit, 0, len(ins)+len(items))
	for _, in := range ins {
		eds = append(eds, edit{in.start, in.end, text[in.start:in.end], in.typ})
	}
	for _, it := range items {
		repl, err := s.replacement(it)
		if err != nil {
			return "", nil, err
		}
		eds = append(eds, edit{it.span.Start, it.span.End, repl, it.span.Type})
	}
	sort.Slice(eds, func(i, j int) bool { return eds[i].start < eds[j].start })
	var b strings.Builder
	out := make([]insertion, 0, len(eds))
	last := 0
	for _, ed := range eds {
		b.WriteString(text[last:ed.start])
		start := b.Len()
		b.WriteString(ed.repl)
		out = append(out, insertion{start: start, end: b.Len(), typ: ed.typ})
		last = ed.end
	}
	b.WriteString(text[last:])
	return b.String(), out, nil
}

func (s *Session) scan(ctx context.Context, text string, ins []insertion) ([]finding, []entity.Span) {
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	sortInsertions(ins)
	var found []finding
	var all []entity.Span
	for _, d := range s.e.checkers {
		spans, err := d.Detect(ctx, text)
		if err != nil {
			found = append(found, finding{planned: planned{span: entity.Span{Type: "unknown", Start: -1, End: -1}}, source: d.Name() + ":error"})
			continue
		}
		all = append(all, spans...)
	}
	kept := s.e.keptOrgs(all)
	for _, sp := range all {
		sp = trimPossessive(sp)
		if sp.Len() <= 0 || s.e.policy.Action(sp.Type) == entity.Keep || insideKept(kept, sp.Type, sp.Start, sp.End) {
			continue
		}
		if covered(ins, sp.Start, sp.End) || s.isOwnOutput(sp) {
			continue
		}
		found = append(found, finding{planned: planned{span: sp}, source: sp.Source})
	}
	for _, h := range s.e.vault.FindReals(text) {
		t := kindType(h.Record.Kind)
		if s.e.policy.Action(t) == entity.Keep || covered(ins, h.Start, h.End) {
			continue
		}
		if t == entity.Person && (!detect.NameUseOK(text, h.Start, h.End) || insideKept(kept, t, h.Start, h.End)) {
			continue
		}
		rec := h.Record
		sp := entity.Span{Start: h.Start, End: h.End, Type: t, Label: string(t), Text: text[h.Start:h.End], Source: "vault", Priority: 60, Score: 1}
		found = append(found, finding{planned: planned{span: sp, rec: &rec}, source: "vault"})
		if ext, ok := s.extendName(text, ins, rec, sp); ok {
			found = append(found, finding{planned: planned{span: ext}, source: "vault"})
		}
	}
	return found, kept
}

func (s *Session) collisions(text string, ins []insertion, kept []entity.Span) []finding {
	var out []finding
	for _, h := range s.e.vault.FindSurrogates(text) {
		kind := h.Record.Kind
		if !strings.HasPrefix(kind, "person.") || kind == "person.joined" || covered(ins, h.Start, h.End) || insideKept(kept, entity.Person, h.Start, h.End) {
			continue
		}
		word := text[h.Start:h.End]
		if !textutil.StartsUpper(word) || s.e.policy.Action(entity.Person) != entity.Mask {
			continue
		}
		src := "collision"
		if kind == "person.family" {
			src = "collision:family"
		}
		sp := entity.Span{Start: h.Start, End: h.End, Type: entity.Person, Label: "person", Text: word, Source: src, Priority: 10, Score: 1}
		out = append(out, finding{planned: planned{span: sp}, source: src})
	}
	return out
}

func (s *Session) remediable(found []finding, ins []insertion) []planned {
	var cands []entity.Span
	recs := map[[2]int]*vault.Record{}
	for _, f := range found {
		sp := f.span
		if sp.Start < 0 || sp.Len() <= 0 || overlapsAny(ins, sp.Start, sp.End) {
			continue
		}
		cands = append(cands, sp)
		if f.rec != nil {
			recs[[2]int{sp.Start, sp.End}] = f.rec
		}
	}
	var out []planned
	for _, sp := range detect.Resolve(cands) {
		out = append(out, planned{span: sp, rec: recs[[2]int{sp.Start, sp.End}]})
	}
	return out
}

func overlapsAny(ins []insertion, start, end int) bool {
	for _, in := range ins {
		if in.start < end && start < in.end {
			return true
		}
	}
	return false
}

func (s *Session) extendName(text string, ins []insertion, rec vault.Record, sp entity.Span) (entity.Span, bool) {
	switch rec.Kind {
	case "person.given":
		end := detect.CapitalizedNameAfter(text, sp.End)
		if end <= sp.End || overlapsAny(ins, sp.End, end) {
			return sp, false
		}
		sp.End = end
	case "person.family":
		start, word := detect.CapitalizedWordBefore(text, sp.Start)
		if start >= sp.Start || overlapsAny(ins, start, sp.Start) || !s.e.knownGiven(word) {
			return sp, false
		}
		sp.Start = start
	default:
		return sp, false
	}
	sp.Text = text[sp.Start:sp.End]
	sp.Priority = 61
	return sp, true
}

func (s *Session) isOwnOutput(sp entity.Span) bool {
	if strings.HasPrefix(sp.Text, "[redacted") {
		return true
	}
	return s.e.vault.IsSurrogate(sp.Text)
}

func kindType(kind string) entity.Type {
	if i := strings.IndexByte(kind, '.'); i > 0 {
		kind = kind[:i]
	}
	return entity.Type(kind)
}

func stripLabel(sp entity.Span) string {
	l := sp.Label
	if l == "" {
		l = string(sp.Type)
	}
	switch sp.Type {
	case entity.Person, entity.Email, entity.Phone, entity.Address, entity.Org, entity.Location, entity.IP, entity.URL, entity.Date:
		l = string(sp.Type)
		if sp.Type == entity.IP {
			l = "ip address"
		}
	}
	return "[redacted " + l + "]"
}

func (s *Session) replacement(it planned) (string, error) {
	if it.rec == nil {
		return s.replace(it.span)
	}
	sp := it.span
	a := s.e.policy.Action(sp.Type)
	if a != entity.Mask {
		return s.replace(sp)
	}
	s.count(sp.Type, a)
	return recordSurrogate(*it.rec, sp.Text), nil
}

func recordSurrogate(rec vault.Record, matched string) string {
	switch kindType(rec.Kind) {
	case entity.Person, entity.Org, entity.Location, entity.Address:
		return textutil.MatchCase(matched, rec.Surrogate)
	case entity.Phone:
		return reformatDigits(matched, rec.Surrogate)
	case entity.Email:
		if textutil.IsAllUpper(matched) {
			return strings.ToUpper(rec.Surrogate)
		}
	}
	return rec.Surrogate
}

func (s *Session) replace(sp entity.Span) (string, error) {
	a := s.e.policy.Action(sp.Type)
	s.count(sp.Type, a)
	switch a {
	case entity.Strip:
		return stripLabel(sp), nil
	case entity.Generalize:
		return generalize(sp), nil
	case entity.Shift:
		return s.shiftDate(sp.Text), nil
	case entity.Mask:
		return s.surrogate(sp)
	}
	return sp.Text, nil
}

func (s *Session) shiftDate(text string) string {
	out := dateshift.Shift(text, s.e.policy.DateOrder, s.offset)
	if out != text {
		s.mu.Lock()
		if _, ok := s.dates[out]; !ok {
			s.dates[out] = text
		}
		s.mu.Unlock()
	}
	return out
}

func (s *Session) dateOriginal(shifted string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.dates[shifted]
	return o, ok
}

func generalize(sp entity.Span) string {
	if sp.Generalized != "" {
		return sp.Generalized
	}
	switch sp.Type {
	case entity.Location:
		if st, ok := detect.StateIn(sp.Text); ok {
			return "a city in " + st
		}
		if st, ok := detect.StateName(sp.Text); ok {
			return st
		}
		return "a place"
	case entity.Org:
		return "an organization"
	case entity.Person:
		return "a person"
	case entity.Address:
		if st, ok := detect.StateIn(sp.Text); ok {
			return "an address in " + st
		}
		return "an address"
	case entity.Email:
		return "an email address"
	case entity.Phone:
		return "a phone number"
	case entity.Date:
		for _, m := range dateshift.Find(sp.Text, dateshift.MDY) {
			if t, hasYear := m.Date(); hasYear {
				return t.Format("January 2006")
			}
		}
		return "a date"
	}
	return stripLabel(sp)
}

func (s *Session) surrogate(sp entity.Span) (string, error) {
	switch sp.Type {
	case entity.Person:
		return s.e.personSurrogate(sp, s.avoids)
	case entity.Email:
		names := s.e.emailNameTokens(sp.Text)
		return s.e.assignCase(sp.Text, "email", func(real string, i int) string { return s.e.emailCandidate(real, i, names) })
	case entity.Phone:
		sur, err := s.e.vault.Assign("phone", sp.Text, func(i int) string { return s.e.gen.Phone(sp.Text, i) })
		if err != nil {
			return "", err
		}
		return reformatDigits(sp.Text, sur), nil
	case entity.Address:
		return s.e.assignCase(sp.Text, "address", s.e.gen.Address)
	case entity.Org:
		return s.e.assignCase(sp.Text, "org", s.e.gen.Org)
	case entity.Location:
		return s.e.assignCase(sp.Text, "location", s.e.gen.Location)
	case entity.URL:
		return s.e.assignCase(sp.Text, "url", s.e.gen.URL)
	case entity.IP:
		return s.e.assignCase(sp.Text, "ip", s.e.gen.IP)
	}
	kind := string(sp.Type)
	return s.e.vault.Assign(kind, sp.Text, func(i int) string { return s.e.gen.Shape(kind, sp.Text, i) })
}

func (e *Engine) assignCase(text, kind string, gen func(string, int) string) (string, error) {
	sur, err := e.vault.Assign(kind, text, func(i int) string { return gen(text, i) })
	if err != nil {
		return "", err
	}
	switch kind {
	case "org", "location", "address":
		return textutil.MatchCase(text, sur), nil
	case "email":
		if textutil.IsAllUpper(text) {
			return strings.ToUpper(sur), nil
		}
	}
	return sur, nil
}

func (e *Engine) emailNameTokens(real string) map[string]string {
	out := map[string]string{}
	at := strings.LastIndex(real, "@")
	if at <= 0 {
		return out
	}
	local := strings.ToLower(real[:at])
	for _, t := range textutil.Tokens(local) {
		tok := local[t.Start:t.End]
		for _, kind := range []string{"person.given", "person.family", "person.nick"} {
			if s, ok := e.vault.Get(kind, tok); ok {
				out[tok] = strings.ToLower(s)
				break
			}
		}
	}
	return out
}

func (e *Engine) emailCandidate(real string, attempt int, names map[string]string) string {
	at := strings.LastIndex(real, "@")
	if at <= 0 {
		return e.gen.Email(real, attempt)
	}
	local := strings.ToLower(real[:at])
	var b strings.Builder
	last := 0
	for _, t := range textutil.Tokens(local) {
		b.WriteString(local[last:t.Start])
		last = t.End
		tok := local[t.Start:t.End]
		switch {
		case isDigits(tok):
			b.WriteString(e.gen.Shape("email-digits", real+"\x00"+tok, attempt))
		case len(tok) == 1:
			b.WriteString(tok)
		default:
			if s, ok := names[tok]; ok {
				b.WriteString(s)
			} else {
				b.WriteString(e.gen.Word(tok, attempt))
			}
		}
	}
	b.WriteString(local[last:])
	out := b.String()
	if attempt > 0 {
		out += fmt.Sprint(attempt)
	}
	return out + "@" + e.gen.EmailDomain(real)
}

func reformatDigits(format, sur string) string {
	var sd []byte
	for i := 0; i < len(sur); i++ {
		if sur[i] >= '0' && sur[i] <= '9' {
			sd = append(sd, sur[i])
		}
	}
	var pos []int
	for i := 0; i < len(format); i++ {
		if format[i] >= '0' && format[i] <= '9' {
			pos = append(pos, i)
		}
	}
	if len(pos) != len(sd) {
		return sur
	}
	out := []byte(format)
	for i, p := range pos {
		out[p] = sd[i]
	}
	return string(out)
}
