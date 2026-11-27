package outworld

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/dateshift"
	"github.com/davidparkercodes/outworld/pkg/surrogate"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

func (s *Session) Rehydrate(text string) string {
	return s.rehydrate("", text, false)
}

func (s *Session) RehydrateJSONText(text string) string {
	return s.rehydrate("", text, true)
}

func realFor(rec vault.Record, matched string) string {
	return widthLike(kindCaseLike(plainRealFor(rec, matched), matched, rec.Kind), matched)
}

func kindCaseLike(real, matched, kind string) string {
	switch kindType(kind) {
	case "email", "ip", "url":
		switch {
		case textutil.IsAllUpper(matched) && !textutil.IsAllUpper(real):
			return strings.ToUpper(real)
		case textutil.IsAllLower(matched) && textutil.IsAllUpper(real):
			return strings.ToLower(real)
		}
	}
	return real
}

func fullWidthText(s string) bool {
	seen := false
	for _, r := range s {
		switch {
		case r >= 0xFF01 && r <= 0xFF5E:
			seen = true
		case r > ' ' && r < 0x7F:
			return false
		}
	}
	return seen
}

func toFullWidth(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r > ' ' && r < 0x7F {
			r += 0xFEE0
		}
		b.WriteRune(r)
	}
	return b.String()
}

func widthLike(real, matched string) string {
	switch {
	case fullWidthText(matched):
		return toFullWidth(real)
	case fullWidthText(real) && !strings.ContainsFunc(matched, func(r rune) bool { return r >= 0xFF01 && r <= 0xFF5E }):
		var b strings.Builder
		for _, r := range real {
			if r >= 0xFF01 && r <= 0xFF5E {
				r -= 0xFEE0
			}
			b.WriteRune(r)
		}
		return b.String()
	}
	return real
}

func plainRealFor(rec vault.Record, matched string) string {
	switch kindType(rec.Kind) {
	case "person", "org", "location", "address":
		if rec.Kind == "person.given" || rec.Kind == "person.family" || rec.Kind == "person.nick" || rec.Kind == residueKind {
			return textutil.MatchCase(matched, textutil.FoldForms(rec.Real))
		}
		return textutil.MatchCase(matched, rec.Real)
	case "phone":
		if rec.Kind == spokenPhoneKind {
			return textutil.MatchCase(matched, rec.Real)
		}
	}
	return rec.Real
}

func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return string(b[1 : len(b)-1])
}

func (s *Session) rehydrate(left, text string, escape bool) string {
	if text == "" {
		return text
	}
	return applyRestore(text, s.restorePlan(left, text, true), escape)
}

func applyRestore(text string, eds []restoreEdit, escape bool) string {
	if len(eds) == 0 {
		return text
	}
	var b strings.Builder
	last := 0
	for _, ed := range eds {
		b.WriteString(text[last:ed.start])
		if escape && ed.kind != phoneShapeEdit && ed.kind != shiftedDateEdit {
			b.WriteString(jsonEscape(ed.repl))
		} else {
			b.WriteString(ed.repl)
		}
		last = ed.end
	}
	b.WriteString(text[last:])
	return b.String()
}

func (s *Session) restorePlan(left, text string, record bool) []restoreEdit {
	if text == "" {
		return nil
	}
	full := left + text
	forms := s.e.vault.FindForms(full, standInFits)
	hits := s.e.surrogateHits(full)
	var out []restoreEdit
	last := len(left)
	gap := func(end int) {
		out = append(out, shiftEdits(s.unshiftEdits(full[last:end], record), last-len(left))...)
	}
	fi, hi := 0, 0
	for {
		for fi < len(forms) && forms[fi].Start < last {
			fi++
		}
		next := len(full) + 1
		if fi < len(forms) {
			next = forms[fi].Start
		}
		for hi < len(hits) && (hits[hi].Start < last || isInitialism(hits[hi].Record.Kind, hits[hi].Record.Surrogate) && full[hits[hi].Start:hits[hi].End] != hits[hi].Record.Surrogate) {
			hi++
		}
		if hi < len(hits) && hits[hi].End <= next {
			h := hits[hi]
			gap(h.Start)
			r := realFor(h.Record, full[h.Start:h.End])
			if h.Record.Kind == "phone" {
				if pr, ok := s.e.phoneReal(full[h.Start:h.End]); ok {
					r = pr
				}
			}
			out = append(out, restoreEdit{h.Start - len(left), h.End - len(left), r, h.Record.Kind})
			last = h.End
			hi++
			continue
		}
		if hi < len(hits) && hits[hi].Start < next {
			hi++
			continue
		}
		if fi >= len(forms) {
			break
		}
		f := forms[fi]
		gap(f.Start)
		out = append(out, restoreEdit{f.Start - len(left), f.End - len(left), restyleLike(f.Record.Real, full[f.Start:f.End]), f.Record.Kind})
		last = f.End
		fi++
	}
	gap(len(full))
	return out
}

func shiftEdits(eds []restoreEdit, by int) []restoreEdit {
	for i := range eds {
		eds[i].start += by
		eds[i].end += by
	}
	return eds
}

func standInFits(text string, start, end int) bool {
	if textutil.GluingDigit(textutil.RuneAt(text, start)) && textutil.GluingDigit(textutil.RuneBefore(text, start)) && !percentEscapeBefore(text, start) {
		return false
	}
	return !(textutil.GluingDigit(textutil.RuneBefore(text, end)) && textutil.GluingDigit(textutil.RuneAt(text, end)))
}

func percentEscapeBefore(text string, i int) bool {
	return i >= 3 && text[i-3] == '%' && isHexByte(text[i-2]) && isHexByte(text[i-1])
}

func (s *Session) rehydrateSurrogates(left, text string, escape bool) string {
	return applyRestore(text, s.surrogatePlan(left, text, true), escape)
}

func fullWidthIn(s string) bool {
	return strings.ContainsFunc(s, func(r rune) bool { return r >= 0xFF01 && r <= 0xFF5E })
}

func (e *Engine) surrogateHits(full string) []vault.Hit {
	if !fullWidthIn(full) {
		return e.vault.FindSurrogates(full)
	}
	folded, starts, ends := textutil.FoldFormsMap(full)
	var out []vault.Hit
	for _, h := range e.vault.FindSurrogates(folded) {
		if h.End <= h.Start || h.End > len(ends) {
			continue
		}
		a, b := starts[h.Start], ends[h.End-1]
		if a < b {
			out = append(out, vault.Hit{Start: a, End: b, Record: h.Record})
		}
	}
	return out
}

func (s *Session) surrogatePlan(left, text string, record bool) []restoreEdit {
	if text == "" {
		return nil
	}
	full := left + text
	var out []restoreEdit
	last := len(left)
	for _, h := range s.e.surrogateHits(full) {
		if h.Start < last || (isInitialism(h.Record.Kind, h.Record.Surrogate) && full[h.Start:h.End] != h.Record.Surrogate) {
			continue
		}
		out = append(out, shiftEdits(s.unshiftEdits(full[last:h.Start], record), last-len(left))...)
		r := realFor(h.Record, full[h.Start:h.End])
		if h.Record.Kind == "phone" {
			if pr, ok := s.e.phoneReal(full[h.Start:h.End]); ok {
				r = pr
			}
		}
		out = append(out, restoreEdit{h.Start - len(left), h.End - len(left), r, h.Record.Kind})
		last = h.End
	}
	return append(out, shiftEdits(s.unshiftEdits(full[last:], record), last-len(left))...)
}

var (
	phoneSep   = textutil.PhoneGapPattern()
	phoneOpen  = textutil.FoldClass('(')
	phoneClose = textutil.FoldClass(')')
	phonePlus  = textutil.FoldClass('+')
)

var phoneLikeRE = regexp.MustCompile(`(?:` + phonePlus + `?1` + phoneSep + `?)?(?:` + phoneOpen + `\d{3}` + phoneClose + phoneSep + `?|\d{3}` + phoneSep + `?)\d{3}` + phoneSep + `?\d{4}\b|(?:` + phonePlus + `\d{1,3}` + phoneSep + `?` + phoneOpen + `?|` + phoneOpen + `?0)\d{1,5}` + phoneClose + `?(?:` + phoneSep + `?\d{2,8}){1,5}\b`)

const (
	phoneShapeEdit  = "phone.shape"
	shiftedDateEdit = "date.shift"
)

type restoreEdit struct {
	start, end int
	repl       string
	kind       string
}

func (s *Session) unshift(seg string) string {
	return applyRestore(seg, s.unshiftEdits(seg, true), false)
}

func (s *Session) unshiftEdits(seg string, record bool) []restoreEdit {
	if seg == "" {
		return nil
	}
	eds := s.e.phoneEdits(seg)
	if s.offset != 0 {
		for _, m := range dateshift.FindLoose(seg, s.e.policy.DateOrder) {
			clash := false
			for _, ed := range eds {
				if m.Start < ed.end && ed.start < m.End {
					clash = true
					break
				}
			}
			if clash {
				continue
			}
			repl, ok := s.dateOriginal(m.Text)
			if !ok {
				if m.Weak {
					continue
				}
				repl = m.Shift(-s.offset)
			}
			if repl != m.Text && record {
				s.e.realDates.addRehydrated(s.scope, repl)
				if t, dated := m.Date(); dated {
					s.e.realDates.addDay(s.scope, t.AddDate(0, 0, -s.offset))
				}
				if ok || modelDateMissed(seg, m.Start, m.End, repl, s.e.policy.DateOrder) {
					s.e.realDates.add(s.scope, repl, ok)
				}
			}
			eds = append(eds, restoreEdit{m.Start, m.End, repl, shiftedDateEdit})
		}
		for _, m := range dateshift.FindVersioned(seg, s.e.policy.DateOrder, true) {
			repl, ok := s.dateOriginal(m.Text)
			if !ok && !m.Weak {
				t, dated := m.Date()
				if cand := m.Shift(-s.offset); s.e.realDates.isReal(s.scope, cand) || (dated && s.e.realDates.hasDayExact(s.scope, t.AddDate(0, 0, -s.offset))) {
					repl, ok = cand, true
				}
			}
			if ok && repl != m.Text {
				eds = append(eds, restoreEdit{m.Start, m.End, repl, shiftedDateEdit})
			}
		}
	}
	sort.Slice(eds, func(i, j int) bool { return eds[i].start < eds[j].start })
	return eds
}

const (
	minRunDigits = 7
	maxRunDigits = 15
	maxRunBytes  = 256
)

func (e *Engine) phoneEdits(seg string) []restoreEdit {
	var eds []restoreEdit
	if len(seg) > maxRunDigits && isDigits(seg) {
		return nil
	}
	for _, m := range phoneLikeRE.FindAllStringIndex(seg, -1) {
		if m[0] > 0 && textutil.IsWordRune(textutil.RuneBefore(seg, m[0])) && !bracketStart(seg[m[0]:]) {
			continue
		}
		if real, ok := e.phoneReal(seg[m[0]:m[1]]); ok {
			eds = append(eds, restoreEdit{m[0], m[1], real, phoneShapeEdit})
		}
	}
	for _, run := range phoneRuns(seg) {
		clash := false
		for _, ed := range eds {
			if run[0] < ed.end && ed.start < run[1] {
				clash = true
				break
			}
		}
		if clash {
			continue
		}
		if real, ok := e.phoneReal(seg[run[0]:run[1]]); ok {
			eds = append(eds, restoreEdit{run[0], run[1], real, phoneShapeEdit})
		}
	}
	for _, w := range phoneWindows(seg) {
		clash := false
		for _, ed := range eds {
			if w[0] < ed.end && ed.start < w[1] {
				clash = true
				break
			}
		}
		if clash {
			continue
		}
		if real, ok := e.phoneReal(seg[w[0]:w[1]]); ok {
			eds = append(eds, restoreEdit{w[0], w[1], real, phoneShapeEdit})
		}
	}
	sort.Slice(eds, func(i, j int) bool { return eds[i].start < eds[j].start })
	return eds
}

const (
	phoneGapBytes    = 32
	knownWindowBytes = 128
	standInWalkBytes = 8192
)

func knownRunRune(r rune) bool {
	return textutil.GluingDigit(r) || textutil.IsPhoneGapRune(r) || textutil.IsPhoneIgnorable(r)
}

func digitBefore(text string, i int) bool {
	for i > 0 {
		r, n := utf8.DecodeLastRuneInString(text[:i])
		if !textutil.IsPhoneIgnorable(r) {
			return textutil.GluingDigit(r)
		}
		i -= n
	}
	return false
}

func digitAfter(text string, i int) bool {
	for i < len(text) {
		r, n := utf8.DecodeRuneInString(text[i:])
		if !textutil.IsPhoneIgnorable(r) {
			return textutil.GluingDigit(r)
		}
		i += n
	}
	return false
}

func knownPhoneSpans(text string, a int, visit func(end, digits int) bool) {
	digits, gap, counted := 0, 0, 0
	for b, q := range text[a:] {
		if !textutil.IsPhoneIgnorable(q) {
			if counted += utf8.RuneLen(q); counted > knownWindowBytes {
				return
			}
		}
		switch {
		case textutil.IsPhoneIgnorable(q):
		case textutil.GluingDigit(q):
			gap = 0
			if digits++; digits > maxStandInDigits {
				return
			}
			end := a + b + utf8.RuneLen(q)
			if digits >= 10 && !digitAfter(text, end) && visit(end, digits) {
				return
			}
		case textutil.IsPhoneGapRune(q):
			if gap += utf8.RuneLen(q); gap > phoneGapBytes {
				return
			}
		default:
			return
		}
	}
}

func phoneWindows(seg string) [][2]int {
	var out [][2]int
	for a, r := range seg {
		if !textutil.GluingDigit(r) || digitBefore(seg, a) {
			continue
		}
		knownPhoneSpans(seg, a, func(end, _ int) bool {
			out = append(out, [2]int{a, end})
			return false
		})
	}
	return out
}

func (e *Engine) restorePhoneShapes(seg string) string {
	eds := e.phoneEdits(seg)
	var b strings.Builder
	last := 0
	for _, ed := range eds {
		b.WriteString(seg[last:ed.start])
		b.WriteString(ed.repl)
		last = ed.end
	}
	b.WriteString(seg[last:])
	return b.String()
}

func phoneRunRune(r rune) bool {
	if textutil.GluingDigit(r) || textutil.IsPhoneSeparator(r) || textutil.IsZeroWidth(r) {
		return true
	}
	switch r {
	case '+', '(', ')':
		return true
	}
	if r < utf8.RuneSelf {
		return false
	}
	f := textutil.FoldForms(string(r))
	return f == "+" || f == "(" || f == ")"
}

func phoneRunEdge(r rune, lead bool) bool {
	if textutil.GluingDigit(r) {
		return true
	}
	f := string(r)
	if r >= utf8.RuneSelf {
		f = textutil.FoldForms(f)
	}
	if lead {
		return f == "+" || f == "("
	}
	return f == ")"
}

func phoneRuns(seg string) [][2]int {
	var out [][2]int
	start := -1
	flush := func(end int) {
		if start < 0 {
			return
		}
		a, b := start, end
		for a < b {
			r, n := utf8.DecodeRuneInString(seg[a:b])
			if phoneRunEdge(r, true) {
				break
			}
			a += n
		}
		for b > a {
			r, n := utf8.DecodeLastRuneInString(seg[a:b])
			if phoneRunEdge(r, false) {
				break
			}
			b -= n
		}
		digits := 0
		for _, r := range seg[a:b] {
			if textutil.DigitLike(r) {
				digits++
			}
		}
		if digits >= minRunDigits && digits <= maxRunDigits && b-a <= maxRunBytes {
			out = append(out, [2]int{a, b})
		}
		start = -1
	}
	for i, r := range seg {
		if phoneRunRune(r) {
			if start < 0 {
				start = i
			}
			continue
		}
		flush(i)
	}
	flush(len(seg))
	return out
}

func digitsOf(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func bracketStart(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return textutil.FoldForms(string(r)) == "("
}

func (e *Engine) phoneReal(matched string) (string, bool) {
	md := digitValues(matched)
	rec, ok := e.vault.ReversePhone(md)
	if !ok {
		return "", false
	}
	var rd []string
	for _, c := range digitValues(rec.Real) {
		rd = append(rd, string(c))
	}
	switch {
	case len(md) == len(rd):
	case len(md) == 11 && len(rd) == 10:
		rd = append([]string{"1"}, rd...)
	case len(md) == 10 && len(rd) == 11 && textutil.FoldForms(rd[0]) == "1":
		rd = rd[1:]
	default:
		return "", false
	}
	if out, ok := putDigits(matched, rd); ok {
		return out, true
	}
	return "", false
}

func (s *Session) RehydrateValue(v any) any {
	switch t := v.(type) {
	case string:
		return s.Rehydrate(t)
	case []any:
		for i := range t {
			t[i] = s.RehydrateValue(t[i])
		}
		return t
	case map[string]any:
		for k, val := range t {
			t[k] = s.RehydrateValue(val)
		}
		return t
	}
	return v
}

func (s *Session) RehydrateArguments(args string) string {
	var v any
	if err := json.Unmarshal([]byte(args), &v); err != nil {
		return s.RehydrateJSONText(args)
	}
	out, err := json.Marshal(s.RehydrateValue(v))
	if err != nil {
		return s.RehydrateJSONText(args)
	}
	return string(out)
}

func (s *Session) UnknownNames(rehydrated string) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range textutil.Tokens(rehydrated) {
		w := rehydrated[t.Start:t.End]
		if !textutil.StartsUpper(w) {
			continue
		}
		if !surrogate.IsBundledGivenName(w) && !surrogate.IsBundledFamilyName(w) {
			continue
		}
		if s.e.vault.IsSurrogate(w) || s.e.vault.IsReal(w) || seen[w] {
			continue
		}
		seen[w] = true
		out = append(out, w)
	}
	sort.Strings(out)
	return out
}

type StreamRehydrator struct {
	s       *Session
	escape  bool
	buf     string
	left    string
	all     strings.Builder
	keep    int
	settled int
}

func (s *Session) NewStreamRehydrator(jsonEscaped bool) *StreamRehydrator {
	return &StreamRehydrator{s: s, escape: jsonEscaped, keep: 1 << 20}
}

const streamEvalStep = 32

func (r *StreamRehydrator) Push(chunk string) string {
	r.buf += chunk
	if len(r.buf) > r.wordWindow() && len(r.buf)-r.settled < streamEvalStep {
		return ""
	}
	held := partialRuneTail(r.buf)
	tail := r.buf[len(r.buf)-held:]
	r.buf = r.buf[:len(r.buf)-held]
	cut := r.avoidStraddle(r.safeCut())
	for round := 0; round < 4; round++ {
		next := r.runSafe(cut)
		if next == cut {
			break
		}
		cut = r.avoidStraddle(next)
	}
	out := r.emit(cut)
	r.buf += tail
	r.settled = len(r.buf)
	return out
}

func partialRuneTail(s string) int {
	for n := 1; n <= 3 && n <= len(s); n++ {
		c := s[len(s)-n]
		if !utf8.RuneStart(c) {
			continue
		}
		if c >= 0xC0 && !utf8.FullRuneInString(s[len(s)-n:]) {
			return n
		}
		return 0
	}
	return 0
}

func (r *StreamRehydrator) Flush() string { return r.emit(len(r.buf)) }

func (r *StreamRehydrator) Pending() int { return len(r.buf) }

func (r *StreamRehydrator) Text() string { return r.all.String() }

func (r *StreamRehydrator) emit(cut int) string {
	if cut <= 0 {
		return ""
	}
	seg := r.buf[:cut]
	r.buf = r.buf[cut:]
	out := r.s.rehydrate(r.left, seg, r.escape)
	ctx := r.left + seg
	from := max(0, len(ctx)-streamLeftBytes)
	for from < len(ctx) && !utf8.RuneStart(ctx[from]) {
		from++
	}
	r.left = ctx[from:]
	if r.all.Len() < r.keep {
		r.all.WriteString(out)
	}
	return out
}

func wordStart(buf string, i, limit int) int {
	stop := max(0, i-limit-utf8.UTFMax)
	for i > stop {
		r, n := utf8.DecodeLastRuneInString(buf[:i])
		if !textutil.IsWordRune(r) || !textutil.IsWordRune(textutil.RuneAt(buf, i)) {
			break
		}
		i -= n
	}
	return i
}

const (
	minWordWindow = 48
	shapeReach    = 64
)

func (r *StreamRehydrator) window() int {
	return max(r.wordWindow(), maxRunBytes)
}

func (r *StreamRehydrator) wordWindow() int {
	return max(r.s.e.vault.MaxSurrogateLen()+2, minWordWindow)
}

func (r *StreamRehydrator) midWordCut(cut int) int {
	window := r.wordWindow()
	for round := 0; round < 4 && cut > 0; round++ {
		lo := max(0, cut-2*window)
		for lo < cut && !utf8.RuneStart(r.buf[lo]) {
			lo++
		}
		seg := r.buf[lo:cut]
		end := len(seg)
		next := cut
		for _, h := range r.s.e.vault.FindSurrogates(seg) {
			if h.End == end {
				next = min(next, lo+h.Start)
			}
		}
		for _, h := range r.s.e.vault.FindForms(seg, standInFits) {
			if h.End == end {
				next = min(next, lo+h.Start)
			}
		}
		if insideLongDigitRun(r.buf, cut) {
			if next == cut {
				return cut
			}
			cut = next
			continue
		}
		for _, run := range phoneRuns(seg) {
			if _, ok := r.s.e.phoneReal(seg[run[0]:run[1]]); ok && run[1] == end {
				next = min(next, lo+run[0])
			}
		}
		if from := phoneCharsTail(seg); digitCount(seg[from:]) >= minPhoneLikeDigits {
			for _, m := range phoneLikeRE.FindAllStringIndex(seg[from:], -1) {
				if _, ok := r.s.e.phoneReal(seg[from+m[0] : from+m[1]]); ok && from+m[1] == end {
					next = min(next, lo+from+m[0])
				}
			}
		}
		if r.s.offset != 0 {
			for _, m := range dateshift.FindLoose(seg, r.s.e.policy.DateOrder) {
				if m.End == end {
					next = min(next, lo+m.Start)
				}
			}
		}
		if next == cut {
			return cut
		}
		cut = next
	}
	return cut
}

const longDigitRun = 24

const maxStandInDigits = 11

const (
	minPhoneLikeDigits = 4
	wordRunReach       = 32
)

func asciiAlnum(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func asciiLetter(c byte) bool { return c|0x20 >= 'a' && c|0x20 <= 'z' }

func insideWordRun(buf string, cut int) bool {
	if cut < wordRunReach || cut >= len(buf) || !asciiLetter(buf[cut-1]) && !asciiLetter(buf[cut]) {
		return false
	}
	for i := cut - wordRunReach; i < cut; i++ {
		if !asciiAlnum(buf[i]) {
			return false
		}
	}
	return true
}

var monthStems = []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"}

func mayHoldDate(s string) bool {
	if strings.ContainsFunc(s, textutil.DigitLike) {
		return true
	}
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return true
		}
	}
	for i := 0; i+3 <= len(s); i++ {
		if c := s[i] | 0x20; c < 'a' || c > 's' {
			continue
		}
		for _, m := range monthStems {
			if strings.EqualFold(s[i:i+3], m) {
				return true
			}
		}
	}
	return false
}

func digitCount(s string) int {
	n := 0
	for _, r := range s {
		if textutil.DigitLike(r) {
			n++
		}
	}
	return n
}

func phoneCharsTail(seg string) int {
	i := len(seg)
	for i > 0 {
		r, n := utf8.DecodeLastRuneInString(seg[:i])
		if !phoneRunRune(r) {
			break
		}
		i -= n
	}
	return i
}

func insideLongDigitRun(buf string, cut int) bool {
	if cut <= 0 || cut >= len(buf) || buf[cut-1] < '0' || buf[cut-1] > '9' || buf[cut] < '0' || buf[cut] > '9' {
		return false
	}
	a, b := cut, cut
	for a > 0 && cut-a < longDigitRun && buf[a-1] >= '0' && buf[a-1] <= '9' {
		a--
	}
	for b < len(buf) && b-cut < longDigitRun && buf[b] >= '0' && buf[b] <= '9' {
		b++
	}
	return cut-a >= longDigitRun && b-cut >= longDigitRun
}

func (r *StreamRehydrator) runSafe(cut int) int {
	buf := r.buf
	if cut <= 0 || cut > len(buf) || !knownRunRune(textutil.RuneBefore(buf, cut)) || cut < len(buf) && !knownRunRune(textutil.RuneAt(buf, cut)) || insideLongDigitRun(buf, cut) {
		return cut
	}
	start, first, gap := cut, -1, 0
	for start > 0 && cut-start <= maxRunBytes {
		q, n := utf8.DecodeLastRuneInString(buf[:start])
		if !knownRunRune(q) {
			break
		}
		switch {
		case textutil.GluingDigit(q):
			gap = 0
			first = start - n
		case textutil.IsPhoneGapRune(q):
			gap += n
		}
		if gap > phoneGapBytes {
			break
		}
		start -= n
	}
	switch {
	case cut-start > maxRunBytes:
		return r.standInBoundary(cut)
	case first < 0:
		return cut
	}
	return first
}

func (r *StreamRehydrator) standInBoundary(cut int) int {
	buf := r.buf
	var before, after []int
	for i := cut; i > 0 && cut-i < standInWalkBytes && len(before) < maxStandInDigits; {
		q, n := utf8.DecodeLastRuneInString(buf[:i])
		if !knownRunRune(q) {
			break
		}
		i -= n
		if textutil.GluingDigit(q) {
			before = append(before, i)
		}
	}
	end := cut
	for end < len(buf) && end-cut < standInWalkBytes && len(after) < maxStandInDigits {
		q, n := utf8.DecodeRuneInString(buf[end:])
		if !knownRunRune(q) {
			break
		}
		if textutil.GluingDigit(q) {
			after = append(after, end)
		}
		end += n
	}
	if end == len(buf) && len(after) < maxStandInDigits && len(before) > 0 {
		need := min(len(before), maxStandInDigits-len(after))
		cut = before[need-1]
	}
	best := cut
	for k := 1; k <= len(before); k++ {
		for _, size := range []int{10, 11} {
			if size-k < 1 || size-k > len(after) {
				continue
			}
			lo := before[k-1]
			last := after[size-k-1]
			_, n := utf8.DecodeRuneInString(buf[last:])
			if _, ok := r.s.e.vault.ReversePhone(digitValues(buf[lo : last+n])); ok && lo < best {
				best = lo
			}
		}
	}
	return best
}

func (r *StreamRehydrator) avoidStraddle(cut int) int {
	if cut > 0 && cut < len(r.buf) {
		ws := wordStart(r.buf, cut, r.wordWindow())
		before := textutil.RuneBefore(r.buf, ws)
		if ws == 0 {
			before = textutil.RuneBefore(r.left, len(r.left))
		}
		if cut-ws <= r.wordWindow() && !textutil.IsWordRune(before) {
			cut = ws
		} else {
			cut = r.midWordCut(cut)
		}
	}
	if cut <= 0 {
		return cut
	}
	type rng struct{ s, e int }
	var spans []rng
	lo, hi := max(0, cut-r.window()), min(len(r.buf), cut+r.window())
	for lo > 0 && !utf8.RuneStart(r.buf[lo]) {
		lo--
	}
	for hi < len(r.buf) && !utf8.RuneStart(r.buf[hi]) {
		hi++
	}
	region := r.buf[lo:hi]
	add := func(a, b int) { spans = append(spans, rng{lo + a, lo + b}) }
	for _, h := range r.s.e.vault.FindSurrogates(region) {
		add(h.Start, h.End)
	}
	slo, shi := max(lo, cut-shapeReach), min(hi, cut+shapeReach)
	for slo > lo && !utf8.RuneStart(r.buf[slo]) {
		slo--
	}
	for shi < hi && !utf8.RuneStart(r.buf[shi]) {
		shi++
	}
	near := r.buf[slo:shi]
	addNear := func(a, b int) { spans = append(spans, rng{slo + a, slo + b}) }
	if insideLongDigitRun(r.buf, cut) {
		near = ""
	}
	if insideWordRun(r.buf, cut) {
		near = ""
	}
	if r.s.offset != 0 && mayHoldDate(near) {
		for _, m := range dateshift.FindLoose(near, r.s.e.policy.DateOrder) {
			addNear(m.Start, m.End)
		}
	}
	if strings.ContainsAny(near, "0123456789") {
		for _, m := range phoneLikeRE.FindAllStringIndex(near, -1) {
			addNear(m[0], m[1])
		}
	}
	for _, h := range r.s.e.vault.FindForms(region, nil) {
		add(h.Start, h.End)
	}
	for _, run := range phoneRuns(region) {
		add(run[0], run[1])
	}
	ctxFrom := max(0, slo-streamLeftBytes)
	for ctxFrom > 0 && !utf8.RuneStart(r.buf[ctxFrom]) {
		ctxFrom--
	}
	leftCtx := r.buf[ctxFrom:slo]
	if slo == 0 {
		leftCtx = r.left
	}
	if near != "" && mayHoldDate(near) {
		for _, ed := range r.s.restorePlan(leftCtx, near, false) {
			addNear(ed.start, ed.end)
		}
	}
	if fullWidthIn(region) {
		for _, h := range r.s.e.surrogateHits(region) {
			add(h.Start, h.End)
		}
	}
	for changed := true; changed; {
		changed = false
		for _, sp := range spans {
			if sp.s < cut && cut < sp.e {
				cut = sp.s
				changed = true
			}
		}
	}
	return cut
}

const streamLeftBytes = 16

func openTail(buf string, window int) int {
	end := len(buf)
	if n := len(buf); n > 0 && buf[n-1] == '%' {
		end = n - 1
	} else if n > 1 && buf[n-2] == '%' && isHexByte(buf[n-1]) {
		end = n - 2
	}
	i := end
	for i > 0 {
		r, n := utf8.DecodeLastRuneInString(buf[:i])
		if !textutil.IsWordRune(r) {
			break
		}
		i -= n
	}
	if i == end && end == len(buf) {
		return end
	}
	if end-i > window {
		i = end - window
		for i < end && !utf8.RuneStart(buf[i]) {
			i++
		}
	}
	return i
}

func isHexByte(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

const (
	datePrefixBytes  = 40
	phonePrefixBytes = 20
)

func (r *StreamRehydrator) safeCut() int {
	buf := r.buf
	window := r.window()
	from := max(0, len(buf)-window)
	hold := openTail(buf, r.wordWindow())
	prefixFrom := len(buf) - r.s.e.vault.MaxSurrogateLen()
	widePrefixFrom := len(buf) - 3*r.s.e.vault.MaxSurrogateLen()
	wide := fullWidthIn(buf[max(0, widePrefixFrom):])
	runEnd := hold
	runStart, digits := runEnd, 0
	runDigits := map[int]int{}
	for runStart > from {
		rr, n := utf8.DecodeLastRuneInString(buf[:runStart])
		if !knownRunRune(rr) {
			break
		}
		runStart -= n
		if textutil.DigitLike(rr) {
			digits++
		}
		runDigits[runStart] = digits
	}
	for i := from; i < hold; i++ {
		if !utf8.RuneStart(buf[i]) {
			continue
		}
		before := textutil.RuneBefore(buf, i)
		if i == 0 {
			before = textutil.RuneBefore(r.left, len(r.left))
		}
		if i >= prefixFrom && (r.s.e.vault.SurrogatePrefixAt(buf, i) || r.s.e.vault.FormPrefixAt(buf, i)) {
			return i
		}
		if wide && i >= widePrefixFrom && r.s.e.vault.SurrogatePrefixAt(textutil.FoldForms(buf[i:]), 0) {
			return i
		}
		cur := textutil.RuneAt(buf, i)
		if asciiLetter(byte(cur)) && cur < utf8.RuneSelf && asciiLetter(byte(before)) && before < utf8.RuneSelf {
			continue
		}
		if i >= runStart && runDigits[i] <= maxRunDigits && phoneRunEdge(cur, true) {
			return i
		}
		if r.s.offset != 0 && i >= len(buf)-datePrefixBytes && datePrefix(buf[i:]) {
			return i
		}
		if i >= len(buf)-phonePrefixBytes && phonePrefix(buf[i:]) {
			return i
		}
	}
	return hold
}

var monthWeekdayNames = []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "Sept", "October", "November", "December", "Monday", "Tuesday", "Tues", "Wednesday", "Thursday", "Thurs", "Friday", "Saturday", "Sunday"}

func isMonthish(w string, partial bool) bool {
	if textutil.IsAllUpper(w) {
		w = w[:1] + strings.ToLower(w[1:])
	}
	for _, n := range monthWeekdayNames {
		if partial && strings.HasPrefix(n, w) {
			return true
		}
		if !partial && (w == n || (len(w) == 3 && strings.HasPrefix(n, w))) {
			return true
		}
	}
	return false
}

var compactDateTokenRE = regexp.MustCompile(`^(?:\d{2}[A-Za-z]{1,3}\d{0,4}|\d{8}T\d{0,6}Z?)$`)

func datePrefix(tail string) bool {
	if len(tail) > 40 || strings.ContainsAny(tail, "\n\r") {
		return false
	}
	toks := textutil.Tokens(tail)
	if len(toks) == 0 || toks[0].Start != 0 {
		return false
	}
	for _, ch := range tail {
		if !(textutil.IsWordRune(ch) || strings.ContainsRune(" ,./-º°", ch)) {
			return false
		}
	}
	for i, t := range toks {
		w := tail[t.Start:t.End]
		partial := i == len(toks)-1 && t.End == len(tail)
		if isDigits(w) {
			if len(w) > 8 {
				return false
			}
			continue
		}
		if compactDateTokenRE.MatchString(w) {
			continue
		}
		if num := strings.TrimRight(w, "stndrh"); isDigits(num) && len(w)-len(num) <= 2 {
			continue
		}
		if w == "of" || (partial && w == "o") {
			continue
		}
		if isMonthish(w, partial) || dateshift.Vocab(w, partial) {
			continue
		}
		return false
	}
	return true
}

func phonePrefix(tail string) bool {
	if len(tail) > 20 {
		return false
	}
	digits := 0
	for i := 0; i < len(tail); i++ {
		c := tail[i]
		switch {
		case c >= '0' && c <= '9':
			digits++
		case c == '+' && i == 0, c == '(' && i <= 3:
		case c == ' ' || c == '.' || c == '-' || c == ')':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return digits > 0 && digits <= 13
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

const dateContextBytes = 64

func modelDateMissed(seg string, start, end int, repl string, order dateshift.Order) bool {
	lo, hi := max(0, start-dateContextBytes), min(len(seg), end+dateContextBytes)
	for lo > 0 && !utf8.RuneStart(seg[lo]) {
		lo--
	}
	for hi < len(seg) && !utf8.RuneStart(seg[hi]) {
		hi++
	}
	around := seg[lo:start] + repl + seg[end:hi]
	return strictMissesIn(around, start-lo, start-lo+len(repl), order)
}
