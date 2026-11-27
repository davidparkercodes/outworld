package outworld

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/namedata"
	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/dateshift"
	"github.com/davidparkercodes/outworld/pkg/detect"
	"github.com/davidparkercodes/outworld/pkg/entity"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

type Violation struct {
	Type    entity.Type `json:"type"`
	Source  string      `json:"source"`
	Message int         `json:"message_index"`
	Field   string      `json:"field,omitempty"`
	Start   int         `json:"start"`
	End     int         `json:"end"`
	Preview string      `json:"preview,omitempty"`
	Reason  string      `json:"reason,omitempty"`
	Hint    string      `json:"hint,omitempty"`
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
	techAnchors map[string]bool
	anchorText  string
	anchorWords map[string]bool
	paraText    string
	paraIndex   *paraNames
	formsReal   map[string]string
	formsSur    map[string]bool
	formsBase   map[string]int
	formsNorm   map[string]string
	formsQueue  []vault.Record
	e           *Engine
	scope       string
	offset      int
	mu          sync.Mutex
	stats       Stats
	violations  []Violation
	priming     bool
	detected    map[string][]entity.Span
	avoid       map[string]bool
	dates       map[string]string
	learned     *detect.DictionaryDetector
	learnedSet  map[string]bool
	familyFirst map[string]bool
	learnedDic  *detect.Dictionary
	pending     []entity.Span
	locMsg      int
	locField    string
	trace       bool
	replaced    []Replacement
	memoHits    int
}

func (e *Engine) NewSession(scope string) *Session {
	if scope == "" {
		scope = "global"
	}
	return &Session{e: e, scope: scope, offset: e.DateOffset(scope), stats: Stats{ByType: map[entity.Type]int{}}, detected: map[string][]entity.Span{}, avoid: map[string]bool{}, dates: map[string]string{}, locMsg: -1}
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
	was := s.priming
	s.priming = on
	pending := s.pending
	if !on {
		s.pending = nil
	}
	s.mu.Unlock()
	if was && !on {
		var failed []finding
		for _, sp := range pending {
			if _, err := s.surrogate(sp); err != nil {
				failed = append(failed, finding{planned: planned{span: entity.Span{Type: sp.Type, Start: -1, End: -1}}, source: "surrogate:error"})
			}
		}
		s.record(failed, "", ReasonError)
	}
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
	s.mu.Unlock()
	s.noteAvoid(text)
	return spans, nil
}

func (s *Session) noteAvoid(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range textutil.Tokens(text) {
		if w := text[t.Start:t.End]; len(w) >= 3 || (len(w) == 2 && textutil.IsAllUpper(w)) {
			s.avoid[strings.ToLower(w)] = true
		}
	}
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

const realPieceSource = "real-piece"

func (s *Session) Mask(ctx context.Context, text string) (string, error) {
	out, err := s.mask(ctx, text)
	if ferr := s.commitForms(); err == nil && ferr != nil {
		return "", ferr
	}
	return out, err
}

func (s *Session) mask(ctx context.Context, text string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return text, nil
	}
	if out, ok := s.memoLookup(text); ok {
		s.noteAvoid(text)
		s.noteMemoHit()
		return out, nil
	}
	mark := s.memoMark()
	s.noteTechAnchors(text)
	spans, err := s.detect(ctx, text)
	if err != nil {
		return "", err
	}
	names := s.learn(spans)
	learned, err := s.learnedSpans(ctx, text)
	if err != nil {
		return "", err
	}
	spans = append(spans[:len(spans):len(spans)], learned...)
	spans = append(spans, s.versionedDates(text, nil)...)
	pairRecs := map[[2]int]*vault.Record{}
	for _, f := range append(s.pairedVaultWords(text, nil), s.gluedVaultNames(text, nil)...) {
		if f.rec == nil {
			continue
		}
		f.span.Source = pairSource
		spans = append(spans, f.span)
		pairRecs[[2]int{f.span.Start, f.span.End}] = f.rec
	}
	act := s.e.actionable(spans)
	plan := detect.Resolve(act)
	if s.isPriming() {
		s.mu.Lock()
		for _, sp := range plan {
			if s.e.policy.Action(sp.Type) == entity.Mask {
				s.pending = append(s.pending, sp)
			}
		}
		s.mu.Unlock()
		return text, nil
	}
	items := make([]planned, len(plan))
	for i, sp := range plan {
		items[i] = planned{span: sp}
		if r, ok := pairRecs[[2]int{sp.Start, sp.End}]; ok && sp.Source == pairSource {
			items[i].rec = r
		}
	}
	out, ins, err := s.apply(text, nil, items)
	if err != nil {
		return "", err
	}
	lost := shiftSpans(s.losers(act, plan), editRanges(nil, items), ins, out)
	for round := 0; ; round++ {
		found, _ := s.scan(ctx, out, ins)
		collisions := s.collisions(out, ins)
		collisions = append(collisions, s.restoreCollisions(out, ins, append(found, collisions...))...)
		var fix []planned
		if len(found) == 0 && len(collisions) == 0 {
			fix, lost = s.lostRepairs(out, lost, ins), nil
			if len(fix) == 0 {
				s.memoStore(text, out, names, mark)
				s.noteReplacements(out, ins)
				return out, nil
			}
			if round == maxRemediationRounds {
				for _, it := range fix {
					found = append(found, finding{planned: it, source: it.span.Source})
				}
				s.record(found, out, ReasonRounds)
				s.noteReplacements(out, ins)
				return out, nil
			}
		} else {
			fix = s.remediable(out, append(found, collisions...), ins)
			if len(fix) == 0 || round == maxRemediationRounds {
				reason := ReasonOverlap
				if len(fix) > 0 {
					reason = ReasonRounds
				}
				s.record(found, out, reason)
				s.noteReplacements(out, ins)
				return out, nil
			}
		}
		olds := editRanges(ins, fix)
		if out, ins, err = s.apply(out, ins, fix); err != nil {
			return "", err
		}
		lost = shiftSpans(lost, olds, ins, out)
	}
}

func (s *Session) Check(ctx context.Context, text string) {
	if s.isPriming() || strings.TrimSpace(text) == "" {
		return
	}
	found, _ := s.scan(ctx, text, nil)
	kept := found[:0]
	for _, f := range found {
		if f.source != knownWeakDateSource {
			kept = append(kept, f)
		}
	}
	s.record(kept, text, ReasonField)
}

type planned struct {
	span entity.Span
	rec  *vault.Record
}

type finding struct {
	planned
	source string
}

func (s *Session) apply(text string, ins []insertion, items []planned) (string, []insertion, error) {
	type edit struct {
		start, end int
		repl       string
		typ        entity.Type
		src        string
		realPiece  bool
	}
	eds := make([]edit, 0, len(ins)+len(items))
	for _, in := range ins {
		eds = append(eds, edit{in.start, in.end, text[in.start:in.end], in.typ, in.src, in.realPiece})
	}
	phones := map[int]planned{}
	for _, it := range items {
		repl, err := s.replacement(it)
		if err != nil {
			return "", nil, err
		}
		src := it.span.Source
		if it.rec != nil {
			src = "vault"
		}
		piece := it.span.Type == entity.Person && s.e.keepsRealPiece(it.span, repl)
		if s.provablePhone(it) {
			phones[it.span.Start] = it
		}
		eds = append(eds, edit{it.span.Start, it.span.End, repl, it.span.Type, src, piece})
	}
	sort.Slice(eds, func(i, j int) bool { return eds[i].start < eds[j].start })
	if len(phones) > 0 {
		starts, ends, repls := make([]int, len(eds)), make([]int, len(eds)), make([]string, len(eds))
		for i, ed := range eds {
			starts[i], ends[i], repls[i] = ed.start, ed.end, ed.repl
		}
		for pass := 0; pass < phoneSettlePasses; pass++ {
			changed := false
			for i := range eds {
				it, ok := phones[starts[i]]
				if !ok || ends[i] != it.span.End {
					continue
				}
				out, os, oe := renderEdits(text, starts, ends, repls)
				before := repls[i]
				lo, hi := phoneRunBounds(out, os[i], oe[i])
				c, err := s.settlePhone(it, repls[i], trimRuneEdges(out[max(0, lo-phoneContextBytes):os[i]], true), trimRuneEdges(out[oe[i]:min(len(out), hi+phoneContextBytes)], false))
				if err != nil {
					return "", nil, err
				}
				repls[i] = c
				changed = changed || c != before
			}
			if !changed {
				break
			}
		}
		for i := range eds {
			eds[i].repl = repls[i]
		}
	}
	var b strings.Builder
	out := make([]insertion, 0, len(eds))
	last := 0
	for _, ed := range eds {
		b.WriteString(text[last:ed.start])
		start := b.Len()
		b.WriteString(ed.repl)
		out = append(out, insertion{start: start, end: b.Len(), typ: ed.typ, src: ed.src, realPiece: ed.realPiece})
		last = ed.end
	}
	b.WriteString(text[last:])
	return b.String(), out, nil
}

func (e *Engine) keepsRealPiece(sp entity.Span, repl string) bool {
	if strings.IndexFunc(sp.Text, unicode.IsSpace) >= 0 {
		return false
	}
	toks := textutil.Tokens(sp.Text)
	if len(toks) == 0 {
		return false
	}
	real, sur := textutil.Fold(sp.Text), textutil.Fold(repl)
	if real == sur {
		return false
	}
	var words []string
	for _, w := range canonicalWords(sp.Canonical) {
		if w = textutil.Fold(w); utf8.RuneCountInString(w) >= 3 {
			words = append(words, w)
		}
	}
	if len(toks) == 1 {
		for _, w := range words {
			if len(w) >= len(real) {
				continue
			}
			if (strings.HasPrefix(real, w) && strings.HasPrefix(sur, w)) || (strings.HasSuffix(real, w) && strings.HasSuffix(sur, w)) {
				return true
			}
		}
		return false
	}
	inWords := map[string]bool{}
	for _, w := range words {
		rs := []rune(w)
		for i := 0; i+3 <= len(rs); i++ {
			inWords[string(rs[i:i+3])] = true
		}
	}
	if len(inWords) == 0 {
		return false
	}
	surPre, surSuf := map[string]bool{}, map[string]bool{}
	for _, st := range textutil.Tokens(sur) {
		piece := sur[st.Start:st.End]
		pp := []rune(piece)
		if len(pp) < 3 || e.vault.IsSurrogate(piece) {
			continue
		}
		surPre[string(pp[:3])] = true
		surSuf[string(pp[len(pp)-3:])] = true
	}
	for _, rt := range textutil.Tokens(real) {
		rp := []rune(real[rt.Start:rt.End])
		if len(rp) < 3 {
			continue
		}
		pre, suf := string(rp[:3]), string(rp[len(rp)-3:])
		if (surPre[pre] && inWords[pre]) || (surSuf[suf] && inWords[suf]) {
			return true
		}
	}
	return false
}

func significant(seg string) bool {
	letters, digits := 0, 0
	for _, r := range seg {
		switch {
		case unicode.IsLetter(r):
			letters++
		case unicode.IsDigit(r):
			digits++
		default:
			letters, digits = 0, 0
		}
		if letters >= 2 || digits >= 4 {
			return true
		}
	}
	return false
}

func shiftSpans(spans []entity.Span, olds [][2]int, ins []insertion, out string) []entity.Span {
	if len(spans) == 0 || len(olds) != len(ins) {
		return nil
	}
	shift := func(p int, end bool) int {
		delta := 0
		for i, o := range olds {
			if p <= o[0] {
				break
			}
			if p < o[1] {
				if end {
					return ins[i].end
				}
				return ins[i].start
			}
			delta = ins[i].end - o[1]
		}
		return p + delta
	}
	var moved []entity.Span
	for _, sp := range spans {
		sp.Start, sp.End = shift(sp.Start, false), shift(sp.End, true)
		if sp.Start >= sp.End || sp.End > len(out) {
			continue
		}
		sp.Text = out[sp.Start:sp.End]
		moved = append(moved, sp)
	}
	return moved
}

func editRanges(ins []insertion, items []planned) [][2]int {
	olds := make([][2]int, 0, len(ins)+len(items))
	for _, in := range ins {
		olds = append(olds, [2]int{in.start, in.end})
	}
	for _, it := range items {
		olds = append(olds, [2]int{it.span.Start, it.span.End})
	}
	sort.Slice(olds, func(i, j int) bool { return olds[i][0] < olds[j][0] })
	return olds
}

func (s *Session) losers(act, plan []entity.Span) []entity.Span {
	if len(act) == len(plan) {
		return nil
	}
	wins := make([]insertion, len(plan))
	for i, w := range plan {
		wins[i] = insertion{start: w.Start, end: w.End}
	}
	var lost []entity.Span
	for _, sp := range act {
		if !covered(wins, sp.Start, sp.End) && s.e.policy.Action(sp.Type) != entity.Shift {
			lost = append(lost, sp)
		}
	}
	return lost
}

func (s *Session) lostRepairs(out string, lost []entity.Span, ins []insertion) []planned {
	var cands []entity.Span
	for _, m := range lost {
		if covered(ins, m.Start, m.End) {
			continue
		}
		if m.Type == entity.Person && s.e.policy.Action(m.Type) == entity.Mask {
			for _, r := range s.residue(out, m, ins) {
				if significant(r.Text) {
					cands = append(cands, r)
				}
			}
			continue
		}
		gap := func(start, end int) {
			seg := out[start:end]
			const edge = " \t\r\n,;:"
			lead := len(seg) - len(strings.TrimLeft(seg, edge))
			start, end = start+lead, start+len(strings.TrimRight(seg, edge))
			if end > start && significant(out[start:end]) {
				cands = append(cands, entity.Span{Start: start, End: end, Type: m.Type, Label: m.Label, Text: out[start:end], Source: residueSource, Priority: m.Priority, Score: m.Score})
			}
		}
		pos := m.Start
		for _, in := range insFrom(ins, pos) {
			if in.start >= m.End {
				break
			}
			if in.end <= pos {
				continue
			}
			if in.start > pos {
				gap(pos, in.start)
			}
			pos = max(pos, in.end)
		}
		if pos < m.End {
			gap(pos, m.End)
		}
	}
	var fix []planned
	for _, sp := range detect.Resolve(cands) {
		fix = append(fix, planned{span: sp})
	}
	return fix
}

func (s *Session) scan(ctx context.Context, text string, ins []insertion) ([]finding, []entity.Span) {
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	sortInsertions(ins)
	s.e.refreshFolds()
	var found []finding
	var all []entity.Span
	for i, r := range detect.RunAll(ctx, text, s.e.checkers) {
		if r.Err != nil {
			found = append(found, finding{planned: planned{span: entity.Span{Type: "unknown", Start: -1, End: -1}}, source: s.e.checkers[i].Name() + ":error"})
			continue
		}
		all = append(all, r.Spans...)
	}
	all = s.e.splitTitlePairs(text, all)
	learned, err := s.learnedSpans(ctx, text)
	if err != nil {
		found = append(found, finding{planned: planned{span: entity.Span{Type: "unknown", Start: -1, End: -1}}, source: "learned:error"})
	}
	all = append(all, learned...)
	for _, in := range ins {
		if in.realPiece {
			sp := entity.Span{Start: in.start, End: in.end, Type: in.typ, Label: string(in.typ), Text: text[in.start:in.end], Source: realPieceSource, Priority: 60, Score: 1}
			found = append(found, finding{planned: planned{span: sp}, source: realPieceSource})
		}
	}
	found = append(found, s.knownDates(text, ins)...)
	found = append(found, s.knownPhones(text, ins)...)
	for _, sp := range s.versionedDates(text, ins) {
		found = append(found, finding{planned: planned{span: sp}, source: sp.Source})
	}
	kept := s.e.keptOrgs(all)
	for _, sp := range all {
		sp = trimPossessive(sp)
		if sp.Len() <= 0 || s.e.allowed(sp) || s.e.policy.Action(sp.Type) == entity.Keep || insideKept(kept, sp.Type, sp.Start, sp.End) {
			continue
		}
		if covered(ins, sp.Start, sp.End) || s.isOwnOutput(sp) || wordsCovered(ins, text, sp) || onlySeparatorsOpen(ins, text, sp) {
			continue
		}
		found = append(found, finding{planned: planned{span: sp}, source: sp.Source})
	}
	var glued []finding
	if s.e.policy.Action(entity.Person) != entity.Keep {
		found = append(found, s.foldedNames(text, ins)...)
		found = append(found, s.joinedUsernames(text, ins)...)
		if s.e.hasNameKeys() {
			glued = s.gluedVaultNames(text, ins)
			found = append(found, glued...)
			found = append(found, s.pairedVaultWords(text, ins)...)
			found = append(found, s.runRemainders(text, ins)...)
			found = append(found, s.adjacentVaultNames(text, ins)...)
			found = append(found, s.heldParticleNeighbors(text, ins)...)
			found = append(found, s.initialRuns(text, ins)...)
			found = append(found, s.digitAffixedJoined(text, ins)...)
		}
	}
	hits := s.e.vault.FindReals(text)
	for i := 0; i < len(hits); i++ {
		h := hits[i]
		if overlapsFinding(glued, h.Start, h.End) {
			continue
		}
		rec, ok := s.realHit(text, h, ins, kept)
		if !ok {
			if h.Record.Kind == "person.joined" {
				hits = append(hits, s.innerReals(text, h)...)
			}
			continue
		}
		t := kindType(rec.Kind)
		hEnd := markEnd(text, h.End)
		sp := entity.Span{Start: h.Start, End: hEnd, Type: t, Label: string(t), Text: text[h.Start:hEnd], Source: "vault", Priority: 60, Score: 1}
		found = append(found, finding{planned: planned{span: sp, rec: &rec}, source: "vault"})
		if ext, ok := s.extendName(text, ins, rec, sp); ok {
			found = append(found, finding{planned: planned{span: ext}, source: "vault"})
		}
	}
	return append(found, s.formFoldedReals(text, ins, kept)...), kept
}

const maxGluedHandle = 40

func (s *Session) innerReals(text string, h vault.Hit) []vault.Hit {
	var out []vault.Hit
	_, first := utf8.DecodeRuneInString(text[h.Start:h.End])
	_, last := utf8.DecodeLastRuneInString(text[h.Start:h.End])
	for _, w := range [][2]int{{h.Start, h.End - last}, {h.Start + first, h.End}} {
		if w[1]-w[0] < 2 {
			continue
		}
		for _, in := range s.e.vault.FindReals(text[w[0]:w[1]]) {
			in.Start, in.End = in.Start+w[0], in.End+w[0]
			if in.Record.Kind == "person.joined" || wordRuneAt(text, in.End) || in.Start > 0 && wordRuneBefore(text, in.Start) {
				continue
			}
			out = append(out, in)
		}
	}
	return out
}

func wordRuneAt(text string, i int) bool {
	r := textutil.RuneAt(text, i)
	return i < len(text) && (unicode.IsLetter(r) || unicode.IsDigit(r))
}

func wordRuneBefore(text string, i int) bool {
	r := textutil.RuneBefore(text, i)
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func (s *Session) gluedVaultNames(text string, ins []insertion) []finding {
	s.e.refreshFolds()
	var out []finding
	joinedRuns(text, s.particleJoint, func(i, j, runes int) {
		if runes >= 6 && runes <= maxGluedHandle && !overlapsAny(ins, i, j) {
			if f, ok := s.gluedRun(text, i, j); ok {
				out = append(out, f)
			}
		}
	})
	return out
}

var jointParticles = []string{"della", "delle", "ste", "van", "von", "der", "den", "del", "ter", "ten", "dos", "das", "st", "de", "la", "le", "da", "di", "du"}

func possessiveAt(text string, j, n int) bool {
	k := j + n
	if k >= len(text) || (text[k] != 's' && text[k] != 'S') {
		return false
	}
	return !letterOrMark(textutil.RuneAt(text, k+1))
}

func oneLetterParticle(text string, seg, j int) bool {
	if j-seg < 2 {
		return false
	}
	r, n := utf8.DecodeLastRuneInString(text[:j])
	return strings.ContainsRune("odlODL", r) && unicode.IsLetter(textutil.RuneBefore(text, j-n))
}

func (s *Session) givenAfterParticles(w string, depth int) bool {
	if depth > 3 {
		return false
	}
	for _, p := range jointParticles {
		if len(w) > len(p)+1 && strings.HasSuffix(w, p) {
			rest := w[:len(w)-len(p)]
			if _, _, ok := s.nameRealLoose("person.given", rest); ok {
				return true
			}
			if s.givenAfterParticles(rest, depth+1) {
				return true
			}
		}
	}
	return false
}

var trailParticles = append(append([]string{}, jointParticles...), "o", "d", "l")

func (s *Session) particleToken(text string, j int) bool {
	lo, hi := j, j
	for lo > 0 && j-lo < 4*maxGluedHandle {
		r := textutil.RuneBefore(text, lo)
		if !letterOrMark(r) && !strings.ContainsRune(".-_", r) {
			break
		}
		lo -= utf8.RuneLen(r)
	}
	for hi < len(text) && hi-j < 4*maxGluedHandle {
		r := textutil.RuneAt(text, hi)
		if !letterOrMark(r) && !strings.ContainsRune(".-_", r) {
			break
		}
		hi += utf8.RuneLen(r)
	}
	token := strings.Trim(text[lo:hi], ".-_")
	segs := strings.FieldsFunc(token, func(r rune) bool { return strings.ContainsRune(".-_", r) })
	if len(segs) < 2 || len(segs) > 5 {
		return false
	}
	evidence := false
	for _, sg := range segs {
		if particleWord(sg) || s.particleAffixed(sg) {
			evidence = true
			break
		}
	}
	if !evidence {
		return false
	}
	_, ok := s.gluedSplit(token)
	return ok
}

func (s *Session) particleAffixed(w string) bool {
	l := strings.ToLower(w)
	name := func(x string) bool {
		if _, _, ok := s.nameRealLoose("person.family", x); ok {
			return true
		}
		_, _, ok := s.e.nameReal([]string{"person.given", "person.nick"}, x)
		return ok
	}
	for _, p := range trailParticles {
		if len(l) < len(p)+3 {
			continue
		}
		if strings.HasSuffix(l, p) && name(l[:len(l)-len(p)]) {
			return true
		}
		if strings.HasPrefix(l, p) && name(l[len(p):]) {
			return true
		}
	}
	return false
}

func (s *Session) particleJoint(text string, seg, j, next int) bool {
	if s.particleToken(text, j) {
		return true
	}
	before := strings.ToLower(text[seg:j])
	k := next
	for k < len(text) && k-next < maxGluedHandle && unicode.IsLetter(textutil.RuneAt(text, k)) {
		k += utf8.RuneLen(textutil.RuneAt(text, k))
	}
	after := strings.ToLower(text[next:k])
	if detect.IsNameParticle(before) {
		return true
	}
	if len(before) >= 2 && strings.ContainsRune("odl", rune(before[len(before)-1])) && unicode.IsLetter(rune(before[len(before)-2])) {
		return true
	}
	if s.givenAfterParticles(before, 0) {
		return true
	}
	for _, p := range jointParticles {
		if len(after) > len(p)+1 && strings.HasPrefix(after, p) {
			if _, _, ok := s.nameRealLoose("person.family", after[len(p):]); ok {
				return true
			}
		}
	}
	return false
}

func letterRuns(text string, fn func(a, b, runes int)) { joinedRuns(text, nil, fn) }

func joinedRuns(text string, joint func(text string, seg, j, next int) bool, fn func(a, b, runes int)) {
	hyphens := joint != nil
	for i := 0; i < len(text); {
		r, n := utf8.DecodeRuneInString(text[i:])
		if !letterOrMark(r) {
			i += n
			continue
		}
		j, runes, seg := i, 0, i
		for j < len(text) {
			r2, n2 := utf8.DecodeRuneInString(text[j:])
			joiner := isApostrophe(r2) && !possessiveAt(text, j, n2) || hyphens && unicode.Is(unicode.Lm, r2) && oneLetterParticle(text, seg, j) || hyphens && (textutil.IsHyphen(r2) || r2 == '.' || r2 == '_') && joint(text, seg, j, j+n2) || hyphens && oneLetterParticle(text, seg, j) && !unicode.IsLetter(r2) && !unicode.IsDigit(r2) && !unicode.IsSpace(r2)
			if joiner && j > i && letterOrMark(textutil.RuneAt(text, j+n2)) {
				j += n2
				seg = j
				continue
			}
			if !letterOrMark(r2) {
				break
			}
			j += n2
			runes++
		}
		fn(i, j, runes)
		i = j
	}
}

const pairSource = "vault:pair"

func digitBounds(text string, a, b int) (int, int) {
	for a > 0 && text[a-1] >= '0' && text[a-1] <= '9' {
		a--
	}
	for b < len(text) && text[b] >= '0' && text[b] <= '9' {
		b++
	}
	return a, b
}

func (s *Session) runName(run string) (string, string, int, bool) {
	if utf8.RuneCountInString(run) > maxGluedHandle {
		return "", "", 0, false
	}
	if f := nameFoldKey(run); lowerASCII(f) && !s.e.nameRunPossible(f) {
		return "", "", 0, false
	}
	if textutil.IsAllLower(run) && detect.IsNameParticle(run) {
		return "", "", 0, false
	}
	cores := []string{run}
	if len(run) > 3 && (run[len(run)-1] == 's' || run[len(run)-1] == 'S') {
		cores = append(cores, run[:len(run)-1])
	}
	if _, _, exact := s.e.nameReal([]string{"person.given", "person.nick", "person.family"}, run); !exact {
		if real, ok := s.syllableGiven(run); ok {
			return "person.given", real, len(run), true
		}
	}
	for _, core := range cores {
		for _, kind := range []string{"person.given", "person.nick", "person.family"} {
			if _, real, ok := s.nameRealLoose(kind, core); ok {
				return kind, real, len(core), true
			}
		}
	}
	return "", "", 0, false
}

func (s *Session) syllableTokens(text string, fn func(a, b, runes int)) {
	for i := 0; i < len(text); {
		r, n := utf8.DecodeRuneInString(text[i:])
		if !textutil.IsHyphen(r) || i == 0 || !unicode.IsLetter(textutil.RuneBefore(text, i)) || i+n >= len(text) || !unicode.IsLetter(textutil.RuneAt(text, i+n)) {
			i += n
			continue
		}
		a, b := i, i+n
		for a > 0 && i-a < 16 && unicode.IsLetter(textutil.RuneBefore(text, a)) {
			a -= utf8.RuneLen(textutil.RuneBefore(text, a))
		}
		for b < len(text) && b-i < 16 && unicode.IsLetter(textutil.RuneAt(text, b)) {
			b += utf8.RuneLen(textutil.RuneAt(text, b))
		}
		if _, ok := s.syllableGiven(text[a:i] + text[i+n:b]); ok && !textutil.StartsUpper(text[a:b]) {
			fn(a, b, utf8.RuneCountInString(text[a:b]))
		}
		i = b
	}
}

func (s *Session) syllableGiven(run string) (string, bool) {
	if h := strings.IndexFunc(run, textutil.IsHyphen); h > 0 {
		_, n := utf8.DecodeRuneInString(run[h:])
		run = run[:h] + run[h+n:]
	}
	if len(run) < 4 || len(run) > 16 || !asciiLetters(run) {
		return "", false
	}
	for cut := 2; cut <= len(run)-2; cut++ {
		_, r1, ok := s.e.nameReal([]string{"person.given"}, run[:cut])
		if !ok {
			continue
		}
		_, r2, ok := s.e.nameReal([]string{"person.family", "person.given"}, run[cut:])
		if ok && cut <= 4 && detect.EastSyllables(run[:cut]) == 1 {
			return r1 + "-" + strings.ToLower(r2), true
		}
	}
	return "", false
}

var leadParticles = []string{"della", "delle", "del", "dos", "das", "van", "von", "der", "den", "ter", "ten", "ste", "st", "de", "la", "le", "da", "di", "du"}

func (s *Session) nameRealLoose(kind, w string) (string, string, bool) {
	w = stripApostrophes(w)
	if k, r, ok := s.e.nameReal([]string{kind}, w); ok {
		return k, r, true
	}
	if kind != "person.family" || len(w) < 4 {
		return "", "", false
	}
	for _, p := range leadParticles {
		if len(w) >= len(p)+3 && strings.EqualFold(w[:len(p)], p) {
			if k, r, ok := s.nameRealLoose(kind, w[len(p):]); ok {
				return k, r, true
			}
		}
	}
	if strings.IndexByte("oOdD", w[0]) >= 0 {
		if k, r, ok := s.e.nameReal([]string{kind}, w[1:]); ok {
			return k, r, true
		}
	}
	for _, p := range trailParticles {
		if len(w) >= len(p)+3 && strings.EqualFold(w[len(w)-len(p):], p) {
			if k, r, ok := s.nameRealLoose(kind, w[:len(w)-len(p)]); ok {
				return k, r, true
			}
		}
	}
	if len(w) <= 2*maxGluedHandle {
		for cut := 3; cut <= len(w)-3; cut++ {
			_, r1, ok1 := s.e.nameReal([]string{kind}, w[:cut])
			if !ok1 {
				continue
			}
			if _, r2, ok2 := s.e.nameReal([]string{kind}, w[cut:]); ok2 {
				return kind, r1 + "-" + r2, true
			}
		}
	}
	return "", "", false
}

const maxJoinerBytes = 16

func (s *Session) joinedToPartner(text string, a, b int, kind string) bool {
	other := "person.family"
	if kind == "person.family" {
		other = "person.given"
	}
	partner := func(w string) bool {
		if k, _, _, ok := s.runName(w); ok && k == other {
			return true
		}
		r, ok := s.e.vault.Reverse(w)
		return ok && (r.Kind == other || r.Kind == "person.joined")
	}
	j := a
	for j > 0 && a-j < maxJoinerBytes {
		r, n := utf8.DecodeLastRuneInString(text[:j])
		if letterOrMark(r) || unicode.IsDigit(r) {
			break
		}
		j -= n
	}
	if j < a && j > 0 {
		k := j
		for k > 0 && j-k < 4*maxJoinerBytes {
			r, n := utf8.DecodeLastRuneInString(text[:k])
			if !letterOrMark(r) {
				break
			}
			k -= n
		}
		if k < j && partner(text[k:j]) && (kind == "person.family" || surnameFirstJoined(text[k:j], text[j:a], text[a:b])) {
			return true
		}
	}
	j = b
	for j < len(text) && j-b < maxJoinerBytes {
		r, n := utf8.DecodeRuneInString(text[j:])
		if letterOrMark(r) || unicode.IsDigit(r) {
			break
		}
		j += n
	}
	if j > b && j < len(text) {
		k := j
		for k < len(text) && k-j < 4*maxJoinerBytes {
			r, n := utf8.DecodeRuneInString(text[k:])
			if !letterOrMark(r) {
				break
			}
			k += n
		}
		if k > j && partner(text[j:k]) {
			return true
		}
	}
	return false
}

func surnameFirstJoined(surname, joiner, given string) bool {
	if strings.ContainsRune(joiner, ',') || !strings.ContainsFunc(joiner, unicode.IsSpace) {
		return true
	}
	return textutil.StartsUpper(surname) && textutil.StartsUpper(given)
}

func (s *Session) adjacentVaultNames(text string, ins []insertion) []finding {
	var out []finding
	try := func(a, b int) {
		if a >= b || overlapsAny(ins, a, b) {
			return
		}
		kind, real, n, ok := s.runName(text[a:b])
		w := text[a:b]
		if !ok || n != b-a || detect.StopWord(w) || asciiLetters(w) && (textutil.IsAllUpper(w) || textutil.StartsUpper(w) && textutil.IsAllLower(w[1:])) {
			return
		}
		sur, ok := s.baseSurrogate(kind, real)
		if !ok {
			return
		}
		rec := vault.Record{Kind: kind, Real: real, Surrogate: sur}
		sp := entity.Span{Start: a, End: b, Type: entity.Person, Label: "person", Text: text[a:b], Source: "vault", Priority: 60, Score: 1}
		out = append(out, finding{planned: planned{span: sp, rec: &rec}, source: "vault"})
	}
	for _, in := range ins {
		if in.typ != entity.Person {
			continue
		}
		j := in.end
		for j < len(text) && j-in.end < maxJoinerBytes && unicode.IsSpace(textutil.RuneAt(text, j)) && textutil.RuneAt(text, j) != '\n' {
			j += utf8.RuneLen(textutil.RuneAt(text, j))
		}
		if j > in.end {
			k := j
			for k < len(text) && k-j < 4*maxGluedHandle && letterOrMark(textutil.RuneAt(text, k)) {
				k += utf8.RuneLen(textutil.RuneAt(text, k))
			}
			try(j, k)
		}
		j = in.start
		for j > 0 && in.start-j < maxJoinerBytes && unicode.IsSpace(textutil.RuneBefore(text, j)) && textutil.RuneBefore(text, j) != '\n' {
			j -= utf8.RuneLen(textutil.RuneBefore(text, j))
		}
		if j < in.start {
			k := j
			for k > 0 && j-k < 4*maxGluedHandle && letterOrMark(textutil.RuneBefore(text, k)) {
				k -= utf8.RuneLen(textutil.RuneBefore(text, k))
			}
			try(k, j)
		}
	}
	return out
}

func (s *Session) heldParticleNeighbors(text string, ins []insertion) []finding {
	var out []finding
	wordAt := func(a int, forward bool) (int, int) {
		b := a
		if forward {
			for b < len(text) && b-a < 16 && unicode.IsLetter(textutil.RuneAt(text, b)) {
				b += utf8.RuneLen(textutil.RuneAt(text, b))
			}
			return a, b
		}
		for b > 0 && a-b < 16 && unicode.IsLetter(textutil.RuneBefore(text, b)) {
			b -= utf8.RuneLen(textutil.RuneBefore(text, b))
		}
		return b, a
	}
	nameLike := func(w string) bool {
		if w == "" {
			return false
		}
		if particleWord(w) {
			return true
		}
		_, _, _, ok := s.runName(w)
		return ok
	}
	emit := func(a, b int, outer string) {
		w := text[a:b]
		if a >= b || !particleWord(w) || overlapsAny(ins, a, b) || nameLike(outer) {
			return
		}
		rec, ok := s.e.heldPersonRecord(w)
		if !ok {
			return
		}
		sp := entity.Span{Start: a, End: b, Type: entity.Person, Label: "person", Text: w, Source: "vault", Priority: 60, Score: 1}
		out = append(out, finding{planned: planned{span: sp, rec: &rec}, source: "vault"})
	}
	for _, in := range ins {
		if in.typ != entity.Person {
			continue
		}
		if in.start > 1 && (text[in.start-1] == ' ' || text[in.start-1] == '-') {
			a, b := wordAt(in.start-1, false)
			oa, ob := 0, 0
			if a > 1 && text[a-1] == ' ' {
				oa, ob = wordAt(a-1, false)
			}
			emit(a, b, text[oa:ob])
		}
		if in.end+1 < len(text) && (text[in.end] == ' ' || text[in.end] == '-') {
			a, b := wordAt(in.end+1, true)
			oa, ob := 0, 0
			if b+1 < len(text) && text[b] == ' ' {
				oa, ob = wordAt(b+1, true)
			}
			emit(a, b, text[oa:ob])
		}
	}
	return out
}

func overlapsPerson(ins []insertion, a, b int) bool {
	for _, in := range ins {
		if in.typ == entity.Person && in.start < b && a < in.end {
			return true
		}
	}
	return false
}

func handleTokenAt(text string, a, b int) (int, int) {
	lo, hi := a, b
	for lo > 0 && a-lo < 4*maxGluedHandle {
		r := textutil.RuneBefore(text, lo)
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("._-", r) {
			break
		}
		lo -= utf8.RuneLen(r)
	}
	for hi < len(text) && hi-b < 4*maxGluedHandle {
		r := textutil.RuneAt(text, hi)
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("._-", r) {
			break
		}
		hi += utf8.RuneLen(r)
	}
	return lo, hi
}

func (s *Session) otherNameInToken(text string, ins []insertion, a, b int) bool {
	lo, hi := handleTokenAt(text, a, b)
	found := false
	letterRuns(text[lo:hi], func(x, y, runes int) {
		if found || lo+x == a && lo+y == b || runes < 2 {
			return
		}
		if overlapsPerson(ins, lo+x, lo+y) {
			found = true
			return
		}
		if _, _, n, ok := s.runName(text[lo+x : lo+y]); ok && n == y-x {
			found = true
		}
	})
	return found
}

func (s *Session) initialRuns(text string, ins []insertion) []finding {
	var out []finding
	letterRuns(text, func(a, b, runes int) {
		if runes < 4 || runes > maxGluedHandle || overlapsAny(ins, a, b) || !asciiLetters(text[a:b]) {
			return
		}
		camel := detect.CamelCaseIdentifier(text[a:b])
		for _, cut := range [][2]int{{a, b - 1}, {a + 1, b}} {
			core := text[cut[0]:cut[1]]
			kind, real, n, ok := s.runName(core)
			if !ok || n != len(core) || kind == "" {
				continue
			}
			initial := text[a:cut[0]] + text[cut[1]:b]
			name := strings.ToLower(core)
			joinedForm := false
			for _, sep := range []string{"_", ".", ""} {
				for _, form := range []string{name + sep + strings.ToLower(initial), strings.ToLower(initial) + sep + name} {
					if _, ok := s.e.vault.Get("person.joined", form); ok {
						joinedForm = true
					}
				}
			}
			if camel || !joinedForm {
				if !s.otherNameInToken(text, ins, a, b) {
					continue
				}
			}
			sur, ok := s.baseSurrogate(kind, real)
			if !ok {
				continue
			}
			if rec, ok := s.affixedJoined(text[a:cut[0]], core, text[cut[1]:b], sur); ok {
				sp := entity.Span{Start: a, End: b, Type: entity.Person, Label: "person", Text: text[a:b], Source: "vault", Priority: 60, Score: 1}
				out = append(out, finding{planned: planned{span: sp, rec: &rec}, source: "vault"})
				return
			}
		}
	})
	return out
}

func (s *Session) digitAffixedJoined(text string, ins []insertion) []finding {
	var out []finding
	for i := 0; i < len(text); {
		r, n := utf8.DecodeRuneInString(text[i:])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			i += n
			continue
		}
		lo, hi := handleTokenAt(text, i, i)
		i = max(hi, i+n)
		tok := text[lo:hi]
		a, b := 0, len(tok)
		for a < b && tok[a] >= '0' && tok[a] <= '9' {
			a++
		}
		for b > a && tok[b-1] >= '0' && tok[b-1] <= '9' {
			b--
		}
		if a == 0 && b == len(tok) || b-a < 3 || overlapsAny(ins, lo, hi) {
			continue
		}
		core := tok[a:b]
		for _, h := range s.e.vault.FindReals(core) {
			if h.Start != 0 || h.End != len(core) || h.Record.Kind != "person.joined" || !strings.EqualFold(h.Record.Real, core) {
				continue
			}
			whole := tok
			base := tok[:a] + strings.ToLower(h.Record.Surrogate) + tok[b:]
			js, err := s.e.vault.Assign("person.joined", whole, func(k int) string {
				if k == 0 {
					return base
				}
				return fmt.Sprintf("%s%d", base, k)
			})
			if err != nil {
				continue
			}
			rec := vault.Record{Kind: "person.joined", Real: whole, Surrogate: js}
			if stored, ok := s.e.vault.Reverse(js); ok && stored.Kind == "person.joined" {
				rec.Real = stored.Real
			}
			sp := entity.Span{Start: lo, End: hi, Type: entity.Person, Label: "person", Text: whole, Source: "vault", Priority: 60, Score: 1}
			out = append(out, finding{planned: planned{span: sp, rec: &rec}, source: "vault"})
		}
	}
	return out
}

func (s *Session) runRemainders(text string, ins []insertion) []finding {
	var out []finding
	affixed := func(a, na, nb, b int) {
		if overlapsAny(ins, a, b) {
			return
		}
		kind, real, _, _ := s.runName(text[na:nb])
		sur, ok := s.baseSurrogate(kind, real)
		if !ok {
			return
		}
		if rec, ok := s.affixedJoined(text[a:na], text[na:nb], text[nb:b], sur); ok {
			sp := entity.Span{Start: a, End: b, Type: entity.Person, Label: "person", Text: text[a:b], Source: "vault", Priority: 60, Score: 1}
			out = append(out, finding{planned: planned{span: sp, rec: &rec}, source: "vault"})
		}
	}
	emit := func(a, b int, trimOnly bool) {
		if b-a < 3 || overlapsAny(ins, a, b) {
			return
		}
		kind, real, n, ok := s.runName(text[a:b])
		if ok && n == b-a && !trimOnly {
			sur, ok := s.baseSurrogate(kind, real)
			if !ok {
				return
			}
			rec := vault.Record{Kind: kind, Real: real, Surrogate: sur}
			sp := entity.Span{Start: a, End: b, Type: entity.Person, Label: "person", Text: text[a:b], Source: "vault", Priority: 60, Score: 1}
			out = append(out, finding{planned: planned{span: sp, rec: &rec}, source: "vault"})
			return
		}
		na, nb, ok := s.particleTrimmed(text, a, b)
		if !ok {
			return
		}
		affixed(a, na, nb, b)
	}
	for _, in := range ins {
		if in.typ != entity.Person {
			continue
		}
		hi := in.end
		for hi < len(text) && hi-in.end < 4*maxGluedHandle && letterOrMark(textutil.RuneAt(text, hi)) {
			hi += utf8.RuneLen(textutil.RuneAt(text, hi))
		}
		if hi > in.end && in.end > 0 && letterOrMark(textutil.RuneBefore(text, in.end)) {
			emit(in.end, hi, false)
		}
		lo := in.start
		for lo > 0 && in.start-lo < 4*maxGluedHandle && letterOrMark(textutil.RuneBefore(text, lo)) {
			lo -= utf8.RuneLen(textutil.RuneBefore(text, lo))
		}
		if lo < in.start && in.start < len(text) && letterOrMark(textutil.RuneAt(text, in.start)) {
			emit(lo, in.start, false)
		}
		s.forwardParticleRun(text, in, affixed)
		j := in.start
		if j > 0 && letterOrMark(textutil.RuneBefore(text, j)) {
			for lo = j; lo > 0 && j-lo < 4*maxGluedHandle && letterOrMark(textutil.RuneBefore(text, lo)); lo -= utf8.RuneLen(textutil.RuneBefore(text, lo)) {
			}
			if !particleWord(text[lo:j]) {
				continue
			}
			j = lo
		}
		end := j
		for hop := 0; hop < 4 && j > 1 && strings.ContainsRune(".-_", rune(text[j-1])) && letterOrMark(textutil.RuneBefore(text, j-1)); hop++ {
			lo = j - 1
			for lo > 0 && j-lo < 4*maxGluedHandle && letterOrMark(textutil.RuneBefore(text, lo)) {
				lo -= utf8.RuneLen(textutil.RuneBefore(text, lo))
			}
			if particleWord(text[lo : j-1]) {
				j = lo
				continue
			}
			na, nb, ok := s.particleTrimmed(text, lo, j-1)
			if !ok && (hop > 0 || end < in.start) {
				if _, _, n, whole := s.runName(text[lo : j-1]); whole && n == j-1-lo {
					na, nb, ok = lo, j-1, true
				}
			}
			if ok {
				affixed(lo, na, nb, end-1)
			}
			break
		}
	}
	return out
}

func (s *Session) affixedJoined(prefix, core, suffix, sur string) (vault.Record, bool) {
	whole := prefix + core + suffix
	base := strings.ToLower(prefix + sur + suffix)
	js, err := s.e.vault.Assign("person.joined", whole, func(i int) string {
		if i == 0 {
			return base
		}
		return fmt.Sprintf("%s%d", base, i)
	})
	if err != nil {
		return vault.Record{}, false
	}
	rec := vault.Record{Kind: "person.joined", Real: whole, Surrogate: js}
	if stored, ok := s.e.vault.Reverse(js); ok && stored.Kind == "person.joined" {
		rec.Real = stored.Real
	}
	if strings.EqualFold(js, base) {
		rec.Surrogate = prefix + caseLike(core, sur) + suffix
	}
	return rec, true
}

func (s *Session) forwardParticleRun(text string, in insertion, affixed func(a, na, nb, b int)) {
	runEnd := func(x int) int {
		for y := x; y < len(text) && y-x < 4*maxGluedHandle; {
			r := textutil.RuneAt(text, y)
			if !letterOrMark(r) {
				return y
			}
			y += utf8.RuneLen(r)
		}
		return min(len(text), x+4*maxGluedHandle)
	}
	j := in.end
	if j < len(text) && letterOrMark(textutil.RuneAt(text, j)) {
		hi := runEnd(j)
		if !particleWord(text[j:hi]) {
			return
		}
		j = hi
	}
	start := j
	for hop := 0; hop < 4 && j+1 < len(text) && strings.ContainsRune(".-_", rune(text[j])) && letterOrMark(textutil.RuneAt(text, j+1)); hop++ {
		hi := runEnd(j + 1)
		if particleWord(text[j+1 : hi]) {
			j = hi
			continue
		}
		na, nb, ok := s.particleTrimmed(text, j+1, hi)
		if !ok && (hop > 0 || start > in.end) {
			if _, _, n, whole := s.runName(text[j+1 : hi]); whole && n == hi-j-1 {
				na, nb, ok = j+1, hi, true
			}
		}
		if ok {
			affixed(start+1, na, nb, hi)
		}
		return
	}
}

func (s *Session) particleTrimmed(text string, a, b int) (int, int, bool) {
	return s.particleTrimmedDepth(text, a, b, 3)
}

func (s *Session) particleTrimmedDepth(text string, a, b, depth int) (int, int, bool) {
	whole := func(x, y int) bool {
		if y-x < 2 {
			return false
		}
		_, _, n, ok := s.runName(text[x:y])
		return ok && n == y-x
	}
	if depth == 0 {
		return a, b, false
	}
	for k := 1; k <= 5 && k < b-a-1; k++ {
		if tail := text[b-k : b]; particleWord(tail) {
			if whole(a, b-k) {
				return a, b - k, true
			}
			if x, y, ok := s.particleTrimmedDepth(text, a, b-k, depth-1); ok {
				return x, y, true
			}
		}
		if head := text[a : a+k]; particleWord(head) {
			if whole(a+k, b) {
				return a + k, b, true
			}
			if x, y, ok := s.particleTrimmedDepth(text, a+k, b, depth-1); ok {
				return x, y, true
			}
		}
	}
	return a, b, false
}

func particleWord(w string) bool {
	l := strings.ToLower(w)
	return asciiLetters(w) && (detect.IsNameParticle(l) || l == "o" || l == "d" || l == "l" || particleCompound(l))
}

func (s *Session) pairedVaultWords(text string, ins []insertion) []finding {
	if s.e.policy.Action(entity.Person) == entity.Keep {
		return nil
	}
	p := s.paraNamesFor(text)
	if len(p.paired) == 0 && len(p.masked) == 0 {
		return nil
	}
	var out []finding
	visit := func(a, b, runes int) {
		if runes < 2 || overlapsAny(ins, a, b) {
			return
		}
		kind, real, n, ok := s.runName(text[a:b])
		if !ok {
			return
		}
		core := text[a : a+n]
		masked := p.masked[textutil.Fold(textutil.FoldForms(core))]
		if !masked && !p.paired[p.at(a)] {
			return
		}
		if detect.StopWord(core) && !s.joinedToPartner(text, a, a+n, kind) {
			return
		}
		pa, pb := digitBounds(text, a, b)
		if pa == a && pb == b && n == b-a && asciiLetters(core) && strings.EqualFold(core, real) && (textutil.IsAllUpper(core) || textutil.StartsUpper(core) && textutil.IsAllLower(text[a+1:b])) {
			return
		}
		if detect.IsTechTerm(core) && !masked && !s.homonymPersonUse(text, a, a+n) {
			return
		}
		sur, ok := s.baseSurrogate(kind, real)
		if !ok {
			return
		}
		rec := vault.Record{Kind: kind, Real: real, Surrogate: sur}
		if pa < a || pb > a+n {
			prefix, suffix, whole := text[pa:a], text[a+n:pb], text[pa:pb]
			base := prefix + strings.ToLower(sur) + strings.ToLower(suffix)
			js, err := s.e.vault.Assign("person.joined", whole, func(i int) string {
				if i == 0 {
					return base
				}
				return fmt.Sprintf("%s%d", base, i)
			})
			if err != nil {
				return
			}
			rec = vault.Record{Kind: "person.joined", Real: whole, Surrogate: js}
			if stored, ok := s.e.vault.Reverse(js); ok && stored.Kind == "person.joined" {
				rec.Real = stored.Real
			}
			if strings.EqualFold(js, base) {
				rec.Surrogate = prefix + caseLike(text[a:a+n], sur) + suffix
			}
		}
		sp := entity.Span{Start: pa, End: pb, Type: entity.Person, Label: "person", Text: text[pa:pb], Source: pairSource, Priority: 61, Score: 1}
		out = append(out, finding{planned: planned{span: sp, rec: &rec}, source: "vault"})
	}
	letterRuns(text, visit)
	s.syllableTokens(text, visit)
	return out
}

func letterOrMark(r rune) bool { return unicode.IsLetter(r) || unicode.Is(unicode.Mn, r) }

func (s *Session) particleSurnameEvidence(text string, start, end int) bool {
	prev := strings.TrimRight(lastWordBefore(text, start), ".")
	if prev != "" && (detect.IsHonorific(prev) || s.vaultHas(prev, "person.given", "person.nick")) {
		return true
	}
	p := s.paraNamesFor(text)
	return p.given[p.at(start)] || detect.PersonEvidence(text, start, end)
}

func afterAbbrevParticle(text string, start int) bool {
	for _, p := range []string{"St. ", "St ", "Ste. ", "Ste "} {
		if strings.HasSuffix(text[:start], p) {
			return true
		}
	}
	return false
}

func isApostrophe(r rune) bool { return r == '\'' || r == '\u2019' || r == '\u02bc' }

func stripApostrophes(s string) string {
	return strings.Map(func(r rune) rune {
		if isApostrophe(r) {
			return -1
		}
		return r
	}, s)
}

func (s *Session) gluedRun(text string, a, b int) (finding, bool) {
	pa, pb := a, b
	for pa > 0 && text[pa-1] >= '0' && text[pa-1] <= '9' {
		pa--
	}
	for pb < len(text) && text[pb] >= '0' && text[pb] <= '9' {
		pb++
	}
	if f, ok := s.gluedPair(text, pa, a, b, pb); ok {
		return f, true
	}
	if b-a > 6 && (text[b-1] == 's' || text[b-1] == 'S') {
		return s.gluedPair(text, pa, a, b-1, pb)
	}
	return finding{}, false
}

func foldedWithMap(letters string) (string, []int) {
	var body strings.Builder
	at := []int{0}
	for i, r := range letters {
		f := textutil.Fold(textutil.FoldForms(string(r)))
		if !unicode.IsLetter(r) && !unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Lm, r) {
			f = ""
		}
		end := i + utf8.RuneLen(r)
		if f == "" && len(at) > 0 {
			at[len(at)-1] = end
			continue
		}
		body.WriteString(f)
		for k := 0; k < len(f); k++ {
			at = append(at, end)
		}
	}
	return body.String(), at
}

func caseLike(src, sur string) string {
	switch {
	case textutil.IsAllUpper(src) && utf8.RuneCountInString(src) > 1:
		return strings.ToUpper(sur)
	case textutil.StartsUpper(src):
		return titleWord(sur)
	}
	return strings.ToLower(sur)
}

func nameFoldKey(w string) string {
	if asciiLetters(w) {
		return strings.ToLower(w)
	}
	return strings.ToLower(lettersOnly(textutil.Fold(textutil.FoldForms(w))))
}

func (s *Session) gluedPair(text string, pa, a, b, pb int) (finding, bool) {
	letters := text[a:b]
	if f := nameFoldKey(letters); lowerASCII(f) && !s.e.nameRunPossible(f) {
		return finding{}, false
	}
	body, at := foldedWithMap(letters)
	if !asciiLetters(body) || len(body) < 6 {
		return finding{}, false
	}
	sp, ok := s.gluedSplitBody(body)
	if !ok {
		return finding{}, false
	}
	return s.gluedFinding(text, pa, a, b, pb, letters, at, sp)
}

type gluedCut struct {
	cut          int
	k1, k2       string
	real1, real2 string
}

func (s *Session) gluedSplit(run string) (gluedCut, bool) {
	body, _ := foldedWithMap(run)
	if !asciiLetters(body) || len(body) < 6 {
		return gluedCut{}, false
	}
	if c, ok := s.gluedSplitBody(body); ok {
		return c, true
	}
	if body[len(body)-1] == 's' {
		return s.gluedSplitBody(body[:len(body)-1])
	}
	return gluedCut{}, false
}

func (s *Session) surnamePart(w string) (string, string, bool) {
	if k, r, ok := s.nameRealLoose("person.family", w); ok {
		return k, r, true
	}
	if _, r, ok := s.e.nameReal([]string{"person.given"}, w); ok {
		if _, sur := namedata.LookupSurname(w); sur {
			return "person.given", r, true
		}
	}
	return "", "", false
}

func (s *Session) partReal(kind, w string) (string, string) {
	if kind == "person.family" {
		k, r, _ := s.surnamePart(w)
		return k, r
	}
	_, r, _ := s.nameRealLoose(kind, w)
	return kind, r
}

func (s *Session) gluedSplitBody(body string) (gluedCut, bool) {
	if !s.e.nameRunPossible(body) {
		return gluedCut{}, false
	}
	starts, ends, _ := s.e.nameSpan(body, 0)
	if c, ok := s.gluedSplitPlain(body, 0, starts, ends); ok {
		return c, true
	}
	for a := 0; a <= 8 && a < len(body); a++ {
		if a > 0 && !particleWord(body[:a]) {
			continue
		}
		for b := 0; b <= 8 && a+b < len(body); b++ {
			if a == 0 && b == 0 || b > 0 && !particleWord(body[len(body)-b:]) || len(body)-a-b < 6 {
				continue
			}
			if c, ok := s.gluedSplitPlain(body[a:len(body)-b], a, starts, ends); ok {
				c.cut += a
				return c, true
			}
		}
	}
	return gluedCut{}, false
}

func anyMarked(marks []bool, lo, hi int) bool {
	for i := max(lo, 0); i <= hi && i < len(marks); i++ {
		if marks[i] {
			return true
		}
	}
	return false
}

func (s *Session) gluedSplitPlain(body string, off int, starts, ends []bool) (gluedCut, bool) {
	given, family := []string{"person.given"}, []string{"person.family"}
	for cut := 3; cut <= len(body)-3; cut++ {
		if starts != nil && !(anyMarked(ends, off+cut-8, off+cut) && anyMarked(starts, off+cut, off+cut+9)) {
			continue
		}
		var firstKind, secondKind []string
		for _, gk := range [][]string{given, {"person.nick"}} {
			if firstKind != nil {
				break
			}
			if _, _, ok := s.nameRealLoose(gk[0], body[:cut]); ok {
				if _, _, ok := s.surnamePart(body[cut:]); ok {
					firstKind, secondKind = gk, family
				}
			}
			if firstKind == nil {
				if _, _, ok := s.surnamePart(body[:cut]); ok {
					if _, _, ok := s.nameRealLoose(gk[0], body[cut:]); ok {
						firstKind, secondKind = family, gk
					}
				}
			}
		}
		if firstKind == nil {
			continue
		}
		k1, r1 := s.partReal(firstKind[0], body[:cut])
		k2, r2 := s.partReal(secondKind[0], body[cut:])
		return gluedCut{cut: cut, k1: k1, k2: k2, real1: r1, real2: r2}, true
	}
	return gluedCut{}, false
}

func (s *Session) baseSurrogate(kind, real string) (string, bool) {
	if sur, ok := s.e.vault.Get(kind, real); ok {
		return sur, true
	}
	parts := strings.Split(real, "-")
	if len(parts) < 2 {
		return "", false
	}
	out := make([]string, len(parts))
	for i, p := range parts {
		sur, ok := s.e.vault.Get(kind, p)
		if !ok && i > 0 {
			sur, ok = s.e.vault.Get("person.family", p)
		}
		if !ok {
			return "", false
		}
		out[i] = sur
	}
	return strings.Join(out, "-"), true
}

func (s *Session) gluedFinding(text string, pa, a, b, pb int, letters string, at []int, c gluedCut) (finding, bool) {
	s1, ok1 := s.baseSurrogate(c.k1, c.real1)
	s2, ok2 := s.baseSurrogate(c.k2, c.real2)
	if !ok1 || !ok2 {
		return finding{}, false
	}
	cut := c.cut
	prefix, suffix := text[pa:a], text[b:pb]
	whole := text[pa:pb]
	s1, s2 = strings.ReplaceAll(s1, "-", ""), strings.ReplaceAll(s2, "-", "")
	base := prefix + strings.ToLower(s1) + strings.ToLower(s2) + strings.ToLower(suffix)
	sur, err := s.e.vault.Assign("person.joined", whole, func(i int) string {
		if i == 0 {
			return base
		}
		return fmt.Sprintf("%s%d", base, i)
	})
	sp := entity.Span{Start: pa, End: pb, Type: entity.Person, Label: "person", Text: whole, Source: "vault", Priority: 60, Score: 1}
	if err != nil {
		return finding{planned: planned{span: sp}, source: "vault"}, true
	}
	rec := vault.Record{Kind: "person.joined", Real: whole, Surrogate: sur}
	if stored, ok := s.e.vault.Reverse(sur); ok && stored.Kind == "person.joined" {
		rec.Real = stored.Real
	}
	mixedLower := !textutil.StartsUpper(letters) && strings.IndexFunc(letters, unicode.IsUpper) >= 0
	if (prefix != "" || suffix != "" || mixedLower || rec.Real != whole && innerCapital(letters)) && strings.EqualFold(sur, base) {
		k := at[cut]
		rec.Surrogate = prefix + caseLike(letters[:k], s1) + caseLike(letters[k:], s2) + suffix
	}
	return finding{planned: planned{span: sp, rec: &rec}, source: "vault"}, true
}

func (s *Session) formFoldedReals(text string, ins []insertion, kept []entity.Span) []finding {
	if !textutil.NeedsFormFold(text) {
		return nil
	}
	folded, starts, ends := textutil.FoldFormsMap(text)
	var out []finding
	for _, h := range s.e.vault.FindReals(folded) {
		if h.End <= h.Start || h.End > len(ends) {
			continue
		}
		a, b := starts[h.Start], ends[h.End-1]
		t := kindType(h.Record.Kind)
		if text[a:b] == folded[h.Start:h.End] || covered(ins, a, b) || insideKept(kept, t, a, b) {
			continue
		}
		rec, ok := s.realHit(folded, h, nil, nil)
		if !ok {
			continue
		}
		sp := entity.Span{Start: a, End: b, Type: t, Label: string(t), Text: text[a:b], Source: "vault", Priority: 60, Score: 1}
		out = append(out, finding{planned: planned{span: sp, rec: &rec}, source: "vault"})
	}
	return out
}

func (s *Session) realHit(text string, h vault.Hit, ins []insertion, kept []entity.Span) (vault.Record, bool) {
	t := kindType(h.Record.Kind)
	if s.e.policy.Action(t) == entity.Keep || covered(ins, h.Start, h.End) {
		return vault.Record{}, false
	}
	rec := h.Record
	if rec.Kind == residueKind {
		named := false
		for _, kind := range []string{"person.family", "person.given", "person.nick"} {
			if sur, ok := s.e.vault.Get(kind, rec.Real); ok {
				rec, named = vault.Record{Kind: kind, Real: rec.Real, Surrogate: sur}, true
				break
			}
		}
		if !named {
			return vault.Record{}, false
		}
	}
	word := text[h.Start:h.End]
	if t == entity.Person {
		if insideKept(kept, t, h.Start, h.End) {
			return vault.Record{}, false
		}
		if s.e.lowerFamilyOnly(rec, text, h.Start, h.End) && !detect.LowerNameCue(text, h.Start, h.End, true) && !s.e.givenBefore(text, h.Start) {
			return vault.Record{}, false
		}
		if detect.IsAssistantName(word) {
			if !s.techPersonUse(text, word, h.Start, h.End) {
				return vault.Record{}, false
			}
		} else if detect.IsTechTerm(word) {
			if !personInsertionInToken(text, ins, h.Start, h.End) && !s.maskedInText(text, word) && !s.homonymPersonUse(text, h.Start, h.End) {
				return vault.Record{}, false
			}
		} else if weakLoneName(rec, word) {
			if !s.e.capsWithFamily(text, h.Start, h.End) && detect.OrdinaryWordUse(text, h.Start, h.End) {
				return vault.Record{}, false
			}
		} else if !detect.NameUseOK(text, h.Start, h.End) && !(rec.Kind == "person.family" && afterAbbrevParticle(text, h.Start)) {
			return vault.Record{}, false
		}
		if rec.Kind == "person.family" && detect.IsNameParticle(textutil.Fold(rec.Real)) && !s.particleSurnameEvidence(text, h.Start, h.End) {
			return vault.Record{}, false
		}
	}
	if len(s.e.allow) > 0 && s.e.allow[textutil.Normalize(word)] {
		return vault.Record{}, false
	}
	if rec.Kind == "person.joined" && word != rec.Real && detect.CamelCaseIdentifier(word) {
		return vault.Record{}, false
	}
	return rec, true
}

func (s *Session) noteTechAnchors(text string) {
	words := s.techAnchorWords(text)
	if len(words) == 0 {
		return
	}
	s.mu.Lock()
	if s.techAnchors == nil {
		s.techAnchors = map[string]bool{}
	}
	for w := range words {
		s.techAnchors[w] = true
	}
	s.mu.Unlock()
}

func (s *Session) techAnchorWords(text string) map[string]bool {
	words := map[string]bool{}
	for _, w := range detect.TechNamesWithSurname(text, s.familyKnown) {
		words[w] = true
	}
	return words
}

func (s *Session) vaultHas(w string, kinds ...string) bool {
	for _, kind := range kinds {
		if _, ok := s.e.vault.Get(kind, w); ok {
			return true
		}
	}
	return false
}

func (s *Session) familyKnown(w string) bool { return s.vaultHas(w, "person.family") }

func (s *Session) techPersonUse(text, word string, start, end int) bool {
	if detect.CodeOrIdentifier(text, start, end) {
		return false
	}
	if s.techNameAnchored(text, word) {
		return true
	}
	return !detect.TechContext(text, start, end) && detect.PersonShape(text, start, end)
}

func (s *Session) homonymPersonUse(text string, start, end int) bool {
	return (!detect.InFence(text, start) && s.pairInParagraph(text, text[start:end], start)) || !detect.VaultNameKeep(text, start, end, s.knownPersonOrStandIn)
}

type paraNames struct {
	starts                []int
	given, family, paired map[int]bool
	masked                map[string]bool
}

func (s *Session) paraNamesFor(text string) *paraNames {
	s.mu.Lock()
	if s.paraIndex != nil && len(s.paraText) == len(text) && s.paraText == text {
		p := s.paraIndex
		s.mu.Unlock()
		return p
	}
	s.mu.Unlock()
	p := &paraNames{starts: []int{0}, given: map[int]bool{}, family: map[int]bool{}, paired: map[int]bool{}, masked: map[string]bool{}}
	blank := true
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			if blank && i > 0 {
				p.starts = append(p.starts, i+1)
			}
			blank = true
			continue
		}
		if text[i] != ' ' && text[i] != '\t' && text[i] != '\r' {
			blank = false
		}
	}
	s.e.refreshFolds()
	letterRuns(text, func(a, b, runes int) {
		i := p.at(a)
		if kind, _, _, ok := s.runName(text[a:b]); ok {
			if kind == "person.given" || kind == "person.nick" {
				p.given[i] = true
			} else {
				p.family[i] = true
			}
		}
		if runes >= 6 && runes <= maxGluedHandle {
			if _, ok := s.gluedSplit(text[a:b]); ok {
				p.given[i], p.family[i] = true, true
			}
		}
	})
	s.syllableTokens(text, func(a, b, runes int) {
		p.given[p.at(a)] = true
	})
	for _, h := range s.e.vault.FindSurrogates(text) {
		i := p.at(h.Start)
		if kindType(h.Record.Kind) == entity.Person && h.Record.Kind != "person.joined" {
			p.masked[textutil.Fold(textutil.FoldForms(h.Record.Real))] = true
		}
		switch h.Record.Kind {
		case "person.given", "person.nick":
			p.given[i] = true
		case "person.family":
			p.family[i] = true
		case "person.joined":
			p.given[i], p.family[i] = true, true
		}
	}
	for i := range p.given {
		if p.family[i] {
			p.paired[i] = true
		}
	}
	s.mu.Lock()
	s.paraText, s.paraIndex = text, p
	s.mu.Unlock()
	return p
}

func (s *Session) maskedInText(text, word string) bool {
	return s.paraNamesFor(text).masked[textutil.Fold(textutil.FoldForms(word))]
}

func (p *paraNames) at(pos int) int {
	return sort.Search(len(p.starts), func(i int) bool { return p.starts[i] > pos }) - 1
}

func (s *Session) pairInParagraph(text, word string, pos int) bool {
	_, isGiven := s.e.vault.Get("person.given", word)
	_, isFamily := s.e.vault.Get("person.family", word)
	if !isGiven && !isFamily {
		return false
	}
	p := s.paraNamesFor(text)
	i := p.at(pos)
	return (isGiven && p.family[i]) || (isFamily && p.given[i])
}

func personInsertionInToken(text string, ins []insertion, start, end int) bool {
	lo := start
	for lo > 0 && start-lo < 2*maxWordBytes && !unicode.IsSpace(textutil.RuneBefore(text, lo)) {
		lo--
	}
	hi := end
	for hi < len(text) && hi-end < 2*maxWordBytes && !unicode.IsSpace(textutil.RuneAt(text, hi)) {
		hi++
	}
	for _, in := range insFrom(ins, lo) {
		if in.start >= hi {
			break
		}
		if in.typ == entity.Person && (in.end <= start || in.start >= end) {
			return true
		}
	}
	return false
}

func (s *Session) knownPersonOrStandIn(w string) bool {
	return s.knownPersonWord(w) || s.e.vault.IsSurrogate(w)
}

func (s *Session) knownPersonWord(w string) bool {
	return s.vaultHas(w, "person.given", "person.family")
}

func (s *Session) techNameAnchored(text, word string) bool {
	s.mu.Lock()
	anchored := s.techAnchors[word]
	s.mu.Unlock()
	return anchored || s.textAnchored(text, word)
}

func (s *Session) textAnchored(text, word string) bool {
	s.mu.Lock()
	cached := text == s.anchorText && s.anchorWords != nil
	words := s.anchorWords
	s.mu.Unlock()
	if !cached {
		words = s.techAnchorWords(text)
		s.mu.Lock()
		s.anchorText, s.anchorWords = text, words
		s.mu.Unlock()
	}
	return words[word]
}

const handleJoiners = "._-/@\\"

const (
	capsNeighbourBytes = 40
	maxWordBytes       = 64
)

func (e *Engine) capsWithFamily(text string, start, end int) bool {
	if !textutil.IsAllUpper(text[start:end]) {
		return false
	}
	lo, hi := max(0, start-capsNeighbourBytes), min(len(text), end+capsNeighbourBytes)
	for lo > 0 && !utf8.RuneStart(text[lo]) {
		lo--
	}
	for hi < len(text) && !utf8.RuneStart(text[hi]) {
		hi++
	}
	toks := textutil.Tokens(text[lo:hi])
	for i, t := range toks {
		if lo+t.Start != start {
			continue
		}
		for _, j := range []int{i - 1, i + 1} {
			if j < 0 || j >= len(toks) {
				continue
			}
			w := text[lo+toks[j].Start : lo+toks[j].End]
			if !textutil.IsAllUpper(w) {
				continue
			}
			if _, ok := e.vault.Get("person.family", w); ok {
				return true
			}
		}
	}
	return false
}

func (e *Engine) lowerFamilyOnly(rec vault.Record, text string, start, end int) bool {
	word := text[start:end]
	if rec.Kind != "person.family" || !textutil.IsAllLower(word) || len(textutil.Tokens(word)) != 1 {
		return false
	}
	if !textutil.IsCommonWord(word) && !namedata.IsHandleWord(word) {
		return false
	}
	sentenceEnd := end < len(text) && text[end] == '.' && (end+1 == len(text) || text[end+1] == ' ' || text[end+1] == '\n')
	if (start > 0 && strings.ContainsRune(handleJoiners, rune(text[start-1]))) || (end < len(text) && strings.ContainsRune(handleJoiners, rune(text[end])) && !sentenceEnd) {
		return false
	}
	for _, k := range []string{"person.given", "person.nick"} {
		if _, ok := e.vault.Get(k, word); ok {
			return false
		}
	}
	return true
}

func (e *Engine) givenBefore(text string, start int) bool {
	j := start
	if j == 0 || text[j-1] != ' ' {
		return false
	}
	j--
	k := j
	for k > 0 && j-k < maxWordBytes {
		r, n := utf8.DecodeLastRuneInString(text[:k])
		if !textutil.IsWordRune(r) {
			break
		}
		k -= n
	}
	if k == j {
		return false
	}
	w := text[k:j]
	if _, ok := e.vault.Get("person.given", w); ok {
		return true
	}
	return detect.IsHonorific(w)
}

func weakLoneName(rec vault.Record, word string) bool {
	if rec.Kind != "person.given" && rec.Kind != "person.nick" {
		return false
	}
	return len(textutil.Tokens(word)) == 1 && textutil.IsCommonWord(word)
}

func (s *Session) leavesKnownReal(out string) bool {
	s.e.refreshFolds()
	for _, h := range s.e.vault.FindReals(out) {
		if kindType(h.Record.Kind) != entity.Person {
			continue
		}
		if _, ok := s.realHit(out, h, nil, nil); ok {
			return true
		}
	}
	return len(s.foldedNames(out, nil)) > 0 || len(s.joinedUsernames(out, nil)) > 0
}

func (s *Session) versionedDates(text string, ins []insertion) []entity.Span {
	if s.offset == 0 || s.e.policy.Action(entity.Date) == entity.Keep {
		return nil
	}
	vms := dateshift.FindVersioned(text, s.e.policy.DateOrder, false)
	if len(vms) == 0 {
		return nil
	}
	plain := map[string]bool{}
	for _, m := range dateshift.FindLoose(text, s.e.policy.DateOrder) {
		plain[m.Text] = true
		if t, ok := m.Date(); ok {
			plain[dayKey(t)] = true
		}
	}
	var out []entity.Span
	for _, m := range vms {
		if overlapsAny(ins, m.Start, m.End) {
			continue
		}
		t, dated := m.Date()
		if plain[m.Text] || s.e.realDates.isReal(s.scope, m.Text) || (dated && (plain[dayKey(t)] || s.e.realDates.hasDay(s.scope, t))) {
			out = append(out, entity.Span{Start: m.Start, End: m.End, Type: entity.Date, Label: "date", Text: m.Text, Source: knownDateSource, Priority: 50, Score: 1})
		}
	}
	return out
}

const knownPhoneSource = "vault:phone"

func (s *Session) knownPhones(text string, ins []insertion) []finding {
	if s.e.policy.Action(entity.Phone) == entity.Keep || !strings.ContainsAny(text, "0123456789") && !strings.ContainsFunc(text, textutil.GluingDigit) {
		return nil
	}
	var out []finding
	for _, m := range phoneLikeRE.FindAllStringIndex(text, -1) {
		if overlapsAny(ins, m[0], m[1]) {
			continue
		}
		if r := textutil.RuneBefore(text, m[0]); (textutil.IsWordRune(r) || r == '_') && !bracketStart(text[m[0]:]) {
			continue
		}
		if r := textutil.RuneAt(text, m[1]); textutil.IsWordRune(r) || r == '_' {
			continue
		}
		if _, ok := s.e.vault.Get("phone", text[m[0]:m[1]]); !ok {
			continue
		}
		out = append(out, knownPhoneFinding(text, m[0], m[1]))
	}
	return append(out, s.digitWindowPhones(text, ins, out)...)
}

func knownPhoneFinding(text string, a, b int) finding {
	sp := entity.Span{Start: a, End: b, Type: entity.Phone, Label: "phone", Text: text[a:b], Source: knownPhoneSource, Priority: 55, Score: 1}
	return finding{planned: planned{span: sp}, source: knownPhoneSource}
}

func (s *Session) digitWindowPhones(text string, ins []insertion, found []finding) []finding {
	var out []finding
	for a, r := range text {
		if !textutil.GluingDigit(r) || digitBefore(text, a) && !percentEscapeBefore(text, a) {
			continue
		}
		leadingOne := textutil.FoldForms(string(r)) == "1"
		knownPhoneSpans(text, a, func(end, digits int) bool {
			if digits == 11 && !leadingOne {
				return true
			}
			if overlapsAny(ins, a, end) || overlapsFinding(found, a, end) || overlapsFinding(out, a, end) {
				return true
			}
			start, ok := a, false
			if _, ok = s.e.vault.Get("phone", gluingDigits(text[a:end])); !ok && digits == 11 {
				start = a + utf8.RuneLen(r)
				for start < end && !textutil.GluingDigit(textutil.RuneAt(text, start)) {
					start += utf8.RuneLen(textutil.RuneAt(text, start))
				}
				_, ok = s.e.vault.Get("phone", gluingDigits(text[start:end]))
			}
			if ok {
				out = append(out, knownPhoneFinding(text, start, end))
			}
			return ok
		})
	}
	return out
}

func gluingDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if textutil.GluingDigit(r) {
			b.WriteString(textutil.FoldForms(string(r)))
		}
	}
	return b.String()
}

func overlapsFinding(found []finding, a, b int) bool {
	for _, f := range found {
		if f.span.Start < b && a < f.span.End {
			return true
		}
	}
	return false
}

func (s *Session) knownDates(text string, ins []insertion) []finding {
	return s.knownDatesIn(text, ins, false)
}

func (s *Session) knownDatesIn(text string, ins []insertion, ignoreSaturation bool) []finding {
	if s.offset == 0 || s.e.policy.Action(entity.Date) == entity.Keep || !s.e.realDates.worthScanning(s.scope, text) {
		return nil
	}
	var out []finding
	for _, m := range dateshift.FindLoose(text, s.e.policy.DateOrder) {
		if overlapsAny(ins, m.Start, m.End) {
			continue
		}
		src := knownDateSource
		got := s.e.realDates.lookup(s.scope, m.Text, m.Weak)
		if ignoreSaturation {
			got.saturated = false
		}
		if m.Weak {
			src = knownWeakDateSource
			cue := weakDateCues[strings.ToLower(lastWordBefore(text, m.Start))]
			if !got.rehydrated && strings.IndexFunc(m.Text, unicode.IsLetter) < 0 && !cue {
				continue
			}
			if !got.known && !(got.saturated && cue) {
				continue
			}
		} else if !got.known && !got.saturated {
			continue
		}
		sp := entity.Span{Start: m.Start, End: m.End, Type: entity.Date, Label: "date", Text: m.Text, Source: src, Priority: 50, Score: 1}
		out = append(out, finding{planned: planned{span: sp}, source: src})
	}
	return out
}

func lastWordBefore(text string, i int) string {
	j := i
	for j > 0 && (text[j-1] == ' ' || text[j-1] == '\t' || text[j-1] == '(') {
		j--
	}
	k := j
	for k > 0 {
		r, n := utf8.DecodeLastRuneInString(text[:k])
		if !unicode.IsLetter(r) {
			break
		}
		k -= n
	}
	return text[k:j]
}

func onlySeparatorsOpen(ins []insertion, text string, sp entity.Span) bool {
	hit := false
	pos := sp.Start
	for _, in := range insFrom(ins, pos) {
		if in.start >= sp.End {
			break
		}
		if in.end <= pos {
			continue
		}
		if in.start > pos && strings.IndexFunc(text[pos:in.start], textutil.IsWordRune) >= 0 {
			return false
		}
		hit = true
		pos = max(pos, in.end)
	}
	return hit && (pos >= sp.End || strings.IndexFunc(text[pos:sp.End], textutil.IsWordRune) < 0)
}

func wordsCovered(ins []insertion, text string, sp entity.Span) bool {
	if sp.Type != entity.Person || len(ins) == 0 {
		return false
	}
	toks := textutil.Tokens(text[sp.Start:sp.End])
	if len(toks) < 2 {
		return false
	}
	hit := false
	for _, t := range toks {
		if covered(ins, sp.Start+t.Start, sp.Start+t.End) {
			hit = true
			continue
		}
		w := text[sp.Start+t.Start : sp.Start+t.End]
		if utf8.RuneCountInString(w) > 1 && !detect.IsNameParticle(w) && !particleCompound(w) {
			return false
		}
	}
	return hit
}

func (s *Session) collisions(text string, ins []insertion) []finding {
	var out []finding
	for _, h := range s.e.vault.FindSurrogates(text) {
		kind := h.Record.Kind
		if !strings.HasPrefix(kind, "person.") || covered(ins, h.Start, h.End) {
			continue
		}
		word := text[h.Start:h.End]
		if !textutil.StartsUpper(word) || s.e.policy.Action(entity.Person) != entity.Mask {
			continue
		}
		src := "collision"
		switch {
		case kind == "person.family":
			src = "collision:family"
		case isInitialism(kind, h.Record.Surrogate):
			if word != h.Record.Surrogate {
				continue
			}
			src = collisionInitials
		}
		sp := entity.Span{Start: h.Start, End: h.End, Type: entity.Person, Label: "person", Text: word, Source: src, Priority: 10, Score: 1}
		out = append(out, finding{planned: planned{span: sp}, source: src})
	}
	return out
}

func (s *Session) restoreCollisions(text string, ins []insertion, have []finding) []finding {
	var out []finding
	for _, ed := range s.restorePlan("", text, false) {
		if ed.repl == text[ed.start:ed.end] || overlapsAny(ins, ed.start, ed.end) || overlapsFinding(have, ed.start, ed.end) {
			continue
		}
		typ := restoredType(ed.kind)
		sp := entity.Span{Start: ed.start, End: ed.end, Type: typ, Label: string(typ), Text: text[ed.start:ed.end], Source: restoreCollisionSource, Priority: 10, Score: 1}
		out = append(out, finding{planned: planned{span: sp}, source: restoreCollisionSource})
	}
	return out
}

const restoreCollisionSource = "collision:restore"

func restoredType(kind string) entity.Type {
	switch kind {
	case phoneShapeEdit:
		return entity.Phone
	case shiftedDateEdit:
		return entity.Date
	}
	if strings.HasPrefix(kind, "name.") {
		return entity.Person
	}
	return kindType(kind)
}

func (s *Session) remediable(text string, found []finding, ins []insertion) []planned {
	var cands []entity.Span
	recs := map[[2]int]*vault.Record{}
	for _, f := range found {
		sp := f.span
		if sp.Start < 0 || sp.Len() <= 0 {
			continue
		}
		if overlapsAny(ins, sp.Start, sp.End) {
			cands = append(cands, s.residue(text, sp, ins)...)
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
	for _, in := range insFrom(ins, start) {
		if in.start >= end {
			break
		}
		if start < in.end {
			return true
		}
	}
	return false
}

func (s *Session) extendName(text string, ins []insertion, rec vault.Record, sp entity.Span) (entity.Span, bool) {
	switch rec.Kind {
	case "person.given":
		end := detect.CapitalizedNameAfter(text, sp.End)
		if end <= sp.End || overlapsAny(ins, sp.End, end) || !s.e.surnameWords(text[sp.End:end]) {
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

func (e *Engine) surnameWords(tail string) bool {
	toks := textutil.Tokens(tail)
	if len(toks) == 0 {
		return false
	}
	for _, t := range toks {
		w := tail[t.Start:t.End]
		if _, ok := e.vault.Get("person.family", w); ok {
			continue
		}
		if _, ok := namedata.LookupSurname(textutil.Fold(w)); !ok || textutil.IsCommonWord(w) {
			return false
		}
	}
	return true
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
	r, err := s.plainReplacement(it)
	if err == nil && fullWidthText(it.span.Text) && !strings.HasPrefix(r, "[redacted") {
		r = toFullWidth(r)
	}
	return r, err
}

func (s *Session) plainReplacement(it planned) (string, error) {
	if it.rec == nil {
		r, err := s.replace(it.span)
		if err == nil && it.span.Type == entity.Person && hasMark(it.span.Text) {
			r = stripMarks(r)
		}
		if err == nil && it.span.Type == entity.Person && s.e.policy.Action(entity.Person) == entity.Mask && r != it.span.Text && !strings.HasPrefix(r, "[redacted") && s.rehydrateSurrogates("", r, false) != it.span.Text {
			if f, ok := s.spellingForm(it.span.Text, r); ok {
				return f, nil
			}
		}
		return r, err
	}
	sp := it.span
	a := s.e.policy.Action(sp.Type)
	if a != entity.Mask {
		return s.replace(sp)
	}
	s.count(sp.Type, a)
	if it.rec.Kind == "phone" && strings.ContainsAny(detect.FoldForms(sp.Text), "0123456789") {
		return s.e.phoneStandIn(sp.Text, it.rec.Surrogate)
	}
	r := recordSurrogate(*it.rec, sp.Text)
	if kindType(it.rec.Kind) == entity.Person && realFor(*it.rec, r) != sp.Text {
		if f, ok := s.spellingForm(sp.Text, it.rec.Surrogate); ok {
			return f, nil
		}
	}
	return r, nil
}

const MaxNewFormsPerRequest = 64

func (s *Session) commitForms() error {
	s.mu.Lock()
	queue := s.formsQueue
	s.formsQueue, s.formsReal, s.formsSur, s.formsBase, s.formsNorm = nil, nil, nil, nil, nil
	s.mu.Unlock()
	_, err := s.e.vault.AddNameForms(queue)
	return err
}

func (s *Session) spellingForm(text, base string) (string, bool) {
	if f, ok := s.e.vault.NameForm(text); ok {
		return f, true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if f, ok := s.formsReal[text]; ok {
		return f, true
	}
	if len(s.formsQueue) >= MaxNewFormsPerRequest {
		return "", false
	}
	key := textutil.Fold(textutil.FoldForms(text))
	plain := map[string]bool{strings.ToLower(base): true, strings.ToUpper(base): true, titleWord(base): true, base: true}
	shaped := applyShape(text, base)
	if innerCapital(text) && innerCapital(base) {
		shaped = base
		delete(plain, base)
	}
	for k := 0; k < 12; k++ {
		cand := disambiguate(shaped, k)
		norm := strings.ToLower(textutil.FoldForms(cand))
		if cand == "" || plain[cand] || s.formsSur[cand] || (s.formsNorm[norm] != "" && s.formsNorm[norm] != key) || !s.e.vault.NameFormFits(text, cand, s.formsBase[key]) {
			continue
		}
		if s.formsReal == nil {
			s.formsReal, s.formsSur, s.formsBase, s.formsNorm = map[string]string{}, map[string]bool{}, map[string]int{}, map[string]string{}
		}
		s.formsReal[text], s.formsSur[cand], s.formsNorm[norm] = cand, true, key
		s.formsBase[key]++
		s.formsQueue = append(s.formsQueue, vault.Record{Real: text, Surrogate: cand})
		return cand, true
	}
	return "", false
}

func innerCapital(w string) bool {
	_, n := utf8.DecodeRuneInString(w)
	return n < len(w) && strings.IndexFunc(w[n:], unicode.IsUpper) >= 0 && strings.IndexFunc(w, unicode.IsLower) >= 0 && strings.IndexFunc(w, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) < 0
}

func applyShape(text, sur string) string {
	tw, sw := strings.Fields(text), strings.Fields(sur)
	if len(tw) > 1 && len(tw) == len(sw) {
		var b strings.Builder
		k := 0
		for _, r := range sur {
			if unicode.IsSpace(r) {
				b.WriteRune(r)
				continue
			}
			if b.Len() == 0 || unicode.IsSpace(lastRune(b.String())) {
				b.WriteString(shapeWord(tw[k], sw[k]))
				k++
			}
		}
		return b.String()
	}
	notLetter := func(r rune) bool { return !unicode.IsLetter(r) && !unicode.Is(unicode.Mn, r) }
	tp, sp := strings.FieldsFunc(text, notLetter), strings.FieldsFunc(sur, notLetter)
	if len(tp) > 1 && len(tp) == len(sp) {
		var b strings.Builder
		k, i := 0, 0
		for i < len(sur) {
			r, n := utf8.DecodeRuneInString(sur[i:])
			if notLetter(r) {
				b.WriteRune(r)
				i += n
				continue
			}
			b.WriteString(shapeWord(tp[k], sp[k]))
			i += len(sp[k])
			k++
		}
		return b.String()
	}
	return shapeWord(text, sur)
}

func lastRune(s string) rune {
	r, _ := utf8.DecodeLastRuneInString(s)
	return r
}

func shapeWord(text, sur string) string {
	var styles []rune
	for _, r := range text {
		if unicode.IsLetter(r) {
			styles = append(styles, r)
		}
	}
	if len(styles) == 0 {
		return sur
	}
	letters := 0
	for _, r := range sur {
		if unicode.IsLetter(r) {
			letters++
		}
	}
	var b strings.Builder
	i := 0
	for _, r := range sur {
		if !unicode.IsLetter(r) {
			b.WriteRune(r)
			continue
		}
		idx := 0
		if i > 0 && letters > 1 && len(styles) > 1 {
			idx = 1 + (i-1)*(len(styles)-1)/(letters-1)
		}
		st := styles[min(idx, len(styles)-1)]
		i++
		c := unicode.ToLower(r)
		if unicode.IsUpper(st) {
			c = unicode.ToUpper(r)
		}
		if st >= 0xFF21 && st <= 0xFF5A && c < 0x80 && unicode.IsLetter(c) {
			c += 0xFF21 - 'A'
		}
		b.WriteRune(c)
	}
	return b.String()
}

func disambiguate(s string, k int) string {
	if k == 0 {
		return s
	}
	rs := []rune(s)
	var vowels []int
	for i, r := range rs {
		if i > 0 && strings.ContainsRune("aeiouAEIOU", r) || strings.ContainsRune("\uff41\uff45\uff49\uff4f\uff55\uff21\uff25\uff29\uff2f\uff35", r) {
			vowels = append(vowels, i)
		}
	}
	if k > len(vowels) {
		return ""
	}
	i := vowels[k-1]
	out := append(append(append([]rune{}, rs[:i+1]...), rs[i]), rs[i+1:]...)
	return string(out)
}

func hasMark(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) {
			return true
		}
	}
	return false
}

func stripMarks(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) {
			return -1
		}
		return r
	}, s)
}

const phoneContextBytes = 16

func phoneRunBounds(text string, start, end int) (int, int) {
	lo, hi := start, end
	for lo > 0 && start-lo < maxRunBytes {
		r, n := utf8.DecodeLastRuneInString(text[:lo])
		if !phoneRunRune(r) {
			break
		}
		lo -= n
	}
	for hi < len(text) && hi-end < maxRunBytes {
		r, n := utf8.DecodeRuneInString(text[hi:])
		if !phoneRunRune(r) {
			break
		}
		hi += n
	}
	return lo, hi
}

const phoneSettlePasses = 3

func (s *Session) provablePhone(it planned) bool {
	sp := it.span
	return sp.Type == entity.Phone && s.e.policy.Action(sp.Type) == entity.Mask && !(it.rec != nil && it.rec.Kind == spokenPhoneKind) && strings.ContainsAny(detect.FoldForms(sp.Text), "0123456789")
}

func renderEdits(text string, starts, ends []int, repls []string) (string, []int, []int) {
	var b strings.Builder
	os, oe := make([]int, len(starts)), make([]int, len(starts))
	last := 0
	for i := range starts {
		b.WriteString(text[last:starts[i]])
		os[i] = b.Len()
		b.WriteString(repls[i])
		oe[i] = b.Len()
		last = ends[i]
	}
	b.WriteString(text[last:])
	return b.String(), os, oe
}

func trimRuneEdges(s string, left bool) string {
	if left {
		i := 0
		for i < len(s) && !utf8.RuneStart(s[i]) {
			i++
		}
		return s[i:]
	}
	j := len(s)
	for j > 0 && !utf8.FullRuneInString(s[lastRuneStart(s[:j]):j]) {
		j = lastRuneStart(s[:j])
	}
	return s[:j]
}

func lastRuneStart(s string) int {
	i := len(s) - 1
	for i > 0 && !utf8.RuneStart(s[i]) {
		i--
	}
	return max(i, 0)
}

func (s *Session) settlePhone(it planned, current, left, right string) (string, error) {
	sp := it.span
	if s.restoresIn(left, current, right, sp.Text) {
		return current, nil
	}
	sur, haveSur := s.e.vault.Get("phone", sp.Text)
	var cands []string
	if f, ok := s.e.vault.PhoneForm(sp.Text); ok {
		cands = append(cands, f)
	}
	if haveSur {
		cands = append(cands, sur)
	}
	for _, c := range cands {
		if c != current && s.restoresIn(left, c, right, sp.Text) {
			return c, nil
		}
	}
	if haveSur {
		emitted := reformatDigits(sp.Text, sur)
		if s.restoresIn(left, emitted, right, sp.Text) || s.formWouldRestore(left, emitted, right) {
			if added, err := s.e.vault.AddPhoneFormInContext(sp.Text, emitted); err != nil {
				return "", err
			} else if added && s.restoresIn(left, emitted, right, sp.Text) {
				return emitted, nil
			}
		}
	}
	for _, c := range []string{current, sur} {
		if c != "" && (c == current || haveSur) && s.restoresNumber(left, c, right, sp.Text) {
			return c, nil
		}
	}
	return stripLabel(sp), nil
}

func (s *Session) restoresNumber(left, cand, right, real string) bool {
	if strings.HasPrefix(cand, "[redacted") {
		return false
	}
	whole := s.rehydrate("", left+cand+right, false)
	l, r := s.rehydrate("", left, false), s.rehydrate("", right, false)
	if !strings.HasPrefix(whole, l) || !strings.HasSuffix(whole[len(l):], r) || len(whole) < len(l)+len(r) {
		return false
	}
	mid := whole[len(l) : len(whole)-len(r)]
	styled := func(c rune) bool { return c >= utf8.RuneSelf && textutil.DigitLike(c) }
	return mid != cand && (strings.ContainsFunc(real, styled) || strings.ContainsFunc(mid, styled)) && textutil.FoldForms(mid) == textutil.FoldForms(real)
}

func (s *Session) formWouldRestore(left, emitted, right string) bool {
	_, recorded := s.e.vault.FormOwner(emitted)
	return !recorded && standInFits(left+emitted+right, len(left), len(left)+len(emitted))
}

func (s *Session) restoresIn(left, cand, right, real string) bool {
	if strings.HasPrefix(cand, "[redacted") {
		return true
	}
	return s.rehydrate("", left+cand+right, false) == s.rehydrate("", left, false)+real+s.rehydrate("", right, false)
}

func recordSurrogate(rec vault.Record, matched string) string {
	switch kindType(rec.Kind) {
	case entity.Person, entity.Org, entity.Location, entity.Address:
		return textutil.MatchCase(matched, rec.Surrogate)
	case entity.Phone:
		if rec.Kind == spokenPhoneKind {
			return textutil.MatchCase(matched, rec.Surrogate)
		}
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
	folded := detect.FoldForms(text)
	out := dateshift.ShiftLoose(folded, s.e.policy.DateOrder, s.offset)
	if out == folded {
		return text
	}
	if out != text {
		s.e.realDates.add(s.scope, text, true)
		for _, m := range dateshift.FindLoose(folded, s.e.policy.DateOrder) {
			if t, ok := m.Date(); ok {
				s.e.realDates.addDay(s.scope, t)
			}
		}
		if folded != text {
			s.e.realDates.add(s.scope, folded, true)
		}
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
		if sp.Source == residueSource {
			return s.residueSurrogate(sp)
		}
		return s.e.personSurrogate(sp, s.avoids)
	case entity.Email:
		if plain, encode, ok := detect.EncodedEmail(sp.Text); ok {
			names := s.e.emailNameTokens(detect.FoldForms(plain))
			return s.e.vault.Assign("email", sp.Text, func(i int) string {
				return encode(s.e.emailCandidate(plain, i, names))
			})
		}
		names := s.e.emailNameTokens(detect.FoldForms(sp.Text))
		return s.e.assignCase(sp.Text, "email", func(real string, i int) string { return s.e.emailCandidate(real, i, names) })
	case entity.Phone:
		folded := detect.FoldForms(sp.Text)
		if !strings.ContainsAny(folded, "0123456789") {
			return s.e.assignCase(sp.Text, spokenPhoneKind, s.e.gen.SpokenPhone)
		}
		sur, err := s.e.vault.Assign("phone", sp.Text, func(i int) string { return s.e.gen.Phone(folded, i) })
		if err != nil {
			return "", err
		}
		return s.e.phoneStandIn(sp.Text, sur)
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
	folded := detect.FoldForms(sp.Text)
	return s.e.vault.Assign(kind, sp.Text, func(i int) string { return s.e.gen.Shape(kind, folded, i) })
}

const spokenPhoneKind = "phone.spoken"

func (e *Engine) phoneStandIn(real, sur string) (string, error) {
	emitted := reformatDigits(real, sur)
	if f, ok := e.vault.PhoneForm(real); ok {
		return f, nil
	}
	added, err := e.vault.AddPhoneForm(real, emitted)
	if err != nil {
		return "", err
	}
	if added {
		return emitted, nil
	}
	digits := vault.NormalizeReal("phone", real)
	if owner, ok := e.vault.FormOwner(emitted); ok {
		if vault.NormalizeReal("phone", owner) == digits {
			return emitted, nil
		}
		return sur, nil
	}
	if rec, ok := e.vault.Reverse(emitted); ok && (rec.Kind != "phone" || vault.NormalizeReal("phone", rec.Real) != digits) {
		return sur, nil
	}
	if e.vault.IsReal(emitted) {
		return sur, nil
	}
	if e.restorePhoneShapes(emitted) == real {
		return emitted, nil
	}
	return sur, nil
}

func (e *Engine) assignCase(real, kind string, gen func(string, int) string) (string, error) {
	text := detect.FoldForms(real)
	sur, err := e.vault.Assign(kind, real, func(i int) string { return gen(text, i) })
	if err != nil {
		return "", err
	}
	switch kind {
	case "org", "location", "address", spokenPhoneKind:
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
	e.refreshFolds()
	for _, t := range textutil.Tokens(local) {
		tok := local[t.Start:t.End]
		for _, kind := range []string{"person.given", "person.family", "person.nick"} {
			if s, ok := e.vault.Get(kind, tok); ok {
				out[tok] = strings.ToLower(s)
				break
			}
		}
		if _, ok := out[tok]; !ok && len(tok) >= 3 && asciiLetters(tok) {
			if s, ok := e.foldedSurrogate(tok); ok {
				out[tok] = strings.ToLower(s)
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
			b.WriteString(e.gen.Shape("email-digits|"+real, tok, attempt))
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
	var sd []string
	for i := 0; i < len(sur); i++ {
		if sur[i] >= '0' && sur[i] <= '9' {
			sd = append(sd, sur[i:i+1])
		}
	}
	if out, ok := putDigits(format, sd); ok {
		return out
	}
	return sur
}

func putDigits(format string, digits []string) (string, bool) {
	n := 0
	for _, r := range format {
		if textutil.DigitLike(r) {
			n++
		}
	}
	if n != len(digits) {
		return "", false
	}
	var b strings.Builder
	k := 0
	for _, r := range format {
		if textutil.DigitLike(r) {
			v, _ := textutil.DigitValue([]rune(digits[k])[0])
			b.WriteRune(textutil.StyleDigit(r, v))
			k++
			continue
		}
		b.WriteRune(r)
	}
	return b.String(), true
}

func digitValues(s string) string {
	var b strings.Builder
	for _, r := range s {
		if v, ok := textutil.DigitValue(r); ok {
			b.WriteByte(byte('0' + v))
		}
	}
	return b.String()
}

func restyleLike(real, standIn string) string {
	var styles []rune
	for _, r := range standIn {
		if textutil.DigitLike(r) {
			styles = append(styles, r)
		}
	}
	var b strings.Builder
	k := 0
	for _, r := range real {
		v, ok := textutil.DigitValue(r)
		if !ok {
			b.WriteRune(r)
			continue
		}
		if k >= len(styles) {
			return real
		}
		b.WriteRune(textutil.StyleDigit(styles[k], v))
		k++
	}
	if k != len(styles) {
		return real
	}
	return b.String()
}

const (
	residueSource = "residue"
	residueKind   = "person.residue"
)

func (s *Session) residueSurrogate(sp entity.Span) (string, error) {
	text := sp.Text
	var b strings.Builder
	last := 0
	for _, t := range textutil.Tokens(text) {
		word := text[t.Start:t.End]
		b.WriteString(text[last:t.Start])
		last = t.End
		if utf8.RuneCountInString(word) <= 1 || detect.IsNameParticle(word) {
			b.WriteString(word)
			continue
		}
		real := realCasing(word, nil)
		sur, ok := "", false
		for _, kind := range []string{"person.family", "person.given", "person.nick"} {
			if sur, ok = s.e.vault.Get(kind, word); ok {
				break
			}
		}
		if !ok {
			var err error
			sur, err = s.e.vault.Assign(residueKind, real, func(i int) string {
				return avoided(s.e.gen.FamilyName(word, i), s.avoids)
			})
			if err != nil {
				return "", err
			}
		}
		b.WriteString(textutil.MatchCase(word, sur))
	}
	b.WriteString(text[last:])
	return b.String(), nil
}

func (s *Session) residue(text string, sp entity.Span, ins []insertion) []entity.Span {
	if sp.End > len(text) {
		return nil
	}
	person := sp.Type == entity.Person && s.e.policy.Action(sp.Type) == entity.Mask
	if !person && s.e.policy.Action(sp.Type) != entity.Strip {
		return nil
	}
	var out []entity.Span
	add := func(start, end int) {
		if end <= start {
			return
		}
		out = append(out, entity.Span{Start: start, End: end, Type: sp.Type, Label: sp.Label, Text: text[start:end], Source: residueSource, Priority: sp.Priority, Score: sp.Score})
	}
	gap := func(start, end int) {
		if !person {
			seg := text[start:end]
			lead := len(seg) - len(strings.TrimLeft(seg, " \t\r\n"))
			add(start+lead, start+len(strings.TrimRight(seg, " \t\r\n")))
			return
		}
		runStart, runEnd := -1, -1
		for _, t := range textutil.Tokens(text[start:end]) {
			w := text[start+t.Start : start+t.End]
			if utf8.RuneCountInString(w) <= 1 || detect.IsNameParticle(w) || particleCompound(w) || sp.Type == entity.Person && strings.IndexFunc(w, unicode.IsLetter) < 0 {
				continue
			}
			if runStart >= 0 && text[runEnd:start+t.Start] == " " {
				runEnd = start + t.End
				continue
			}
			add(runStart, runEnd)
			runStart, runEnd = start+t.Start, start+t.End
		}
		add(runStart, runEnd)
	}
	pos := sp.Start
	for _, in := range insFrom(ins, pos) {
		if in.start >= sp.End {
			break
		}
		if in.end <= pos {
			continue
		}
		if in.start > pos {
			gap(pos, in.start)
		}
		if in.end > pos {
			pos = in.end
		}
	}
	if pos < sp.End {
		gap(pos, sp.End)
	}
	return out
}
