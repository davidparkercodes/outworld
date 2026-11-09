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
	switch kindType(rec.Kind) {
	case "person", "org", "location", "address":
		return textutil.MatchCase(matched, rec.Real)
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
	full := left + text
	hits := s.e.vault.FindSurrogates(full)
	var b strings.Builder
	last := len(left)
	for _, h := range hits {
		if h.Start < last {
			continue
		}
		b.WriteString(s.unshift(full[last:h.Start]))
		r := realFor(h.Record, full[h.Start:h.End])
		if h.Record.Kind == "phone" {
			if pr, ok := s.e.phoneReal(full[h.Start:h.End]); ok {
				r = pr
			}
		}
		if escape {
			r = jsonEscape(r)
		}
		b.WriteString(r)
		last = h.End
	}
	b.WriteString(s.unshift(full[last:]))
	return b.String()
}

var phoneLikeRE = regexp.MustCompile(`(?:\+?1[\s.\-]?)?(?:\(\d{3}\)[\s.\-]?|\d{3}[\s.\-]?)\d{3}[\s.\-]?\d{4}\b`)

type restoreEdit struct {
	start, end int
	repl       string
}

func (s *Session) unshift(seg string) string {
	if seg == "" {
		return seg
	}
	var eds []restoreEdit
	for _, m := range phoneLikeRE.FindAllStringIndex(seg, -1) {
		if m[0] > 0 && textutil.IsWordRune(textutil.RuneBefore(seg, m[0])) {
			continue
		}
		if real, ok := s.e.phoneReal(seg[m[0]:m[1]]); ok {
			eds = append(eds, restoreEdit{m[0], m[1], real})
		}
	}
	if s.offset != 0 {
		for _, m := range dateshift.Find(seg, s.e.policy.DateOrder) {
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
				repl = m.Shift(-s.offset)
			}
			eds = append(eds, restoreEdit{m.Start, m.End, repl})
		}
	}
	if len(eds) == 0 {
		return seg
	}
	sort.Slice(eds, func(i, j int) bool { return eds[i].start < eds[j].start })
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

func digitsOf(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func (e *Engine) phoneReal(matched string) (string, bool) {
	rec, ok := e.vault.ReversePhone(digitsOf(matched))
	if !ok {
		return "", false
	}
	md, rd := digitsOf(matched), digitsOf(rec.Real)
	switch {
	case len(md) == len(rd):
	case len(md) == 11 && len(rd) == 10:
		rd = "1" + rd
	case len(md) == 10 && len(rd) == 11 && rd[0] == '1':
		rd = rd[1:]
	default:
		return rec.Real, true
	}
	return reformatDigits(matched, rd), true
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
	s      *Session
	escape bool
	buf    string
	left   string
	all    strings.Builder
	keep   int
}

func (s *Session) NewStreamRehydrator(jsonEscaped bool) *StreamRehydrator {
	return &StreamRehydrator{s: s, escape: jsonEscaped, keep: 1 << 20}
}

func (r *StreamRehydrator) Push(chunk string) string {
	r.buf += chunk
	cut := r.avoidStraddle(r.safeCut())
	return r.emit(cut)
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
	if rr, n := utf8.DecodeLastRuneInString(seg); rr != utf8.RuneError || n > 0 {
		r.left = seg[len(seg)-n:]
	}
	if r.all.Len() < r.keep {
		r.all.WriteString(out)
	}
	return out
}

func (r *StreamRehydrator) avoidStraddle(cut int) int {
	if cut <= 0 {
		return cut
	}
	type rng struct{ s, e int }
	var spans []rng
	for _, h := range r.s.e.vault.FindSurrogates(r.buf) {
		spans = append(spans, rng{h.Start, h.End})
	}
	if r.s.offset != 0 {
		for _, m := range dateshift.Find(r.buf, r.s.e.policy.DateOrder) {
			spans = append(spans, rng{m.Start, m.End})
		}
	}
	for _, m := range phoneLikeRE.FindAllStringIndex(r.buf, -1) {
		spans = append(spans, rng{m[0], m[1]})
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

func (r *StreamRehydrator) safeCut() int {
	buf := r.buf
	window := r.s.e.vault.MaxSurrogateLen() + 2
	if window < 48 {
		window = 48
	}
	from := len(buf) - window
	if from < 0 {
		from = 0
	}
	for i := from; i < len(buf); i++ {
		if !utf8.RuneStart(buf[i]) {
			continue
		}
		before := textutil.RuneBefore(buf, i)
		if i == 0 {
			before = textutil.RuneBefore(r.left, len(r.left))
		}
		cur := textutil.RuneAt(buf, i)
		if textutil.IsWordRune(cur) && textutil.IsWordRune(before) {
			continue
		}
		if r.s.e.vault.SurrogatePrefixAt(buf, i) {
			return i
		}
		if r.s.offset != 0 && datePrefix(buf[i:]) {
			return i
		}
		if phonePrefix(buf[i:]) {
			return i
		}
	}
	return len(buf)
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

func datePrefix(tail string) bool {
	if len(tail) > 40 || strings.ContainsAny(tail, "\n\r") {
		return false
	}
	toks := textutil.Tokens(tail)
	if len(toks) == 0 || toks[0].Start != 0 {
		return false
	}
	for _, ch := range tail {
		if !(textutil.IsWordRune(ch) || strings.ContainsRune(" ,./-", ch)) {
			return false
		}
	}
	for i, t := range toks {
		w := tail[t.Start:t.End]
		partial := i == len(toks)-1 && t.End == len(tail)
		if isDigits(w) {
			if len(w) > 4 {
				return false
			}
			continue
		}
		if num := strings.TrimRight(w, "stndrh"); isDigits(num) && len(w)-len(num) <= 2 {
			continue
		}
		if w == "of" || (partial && w == "o") {
			continue
		}
		if isMonthish(w, partial) {
			continue
		}
		return false
	}
	return true
}

func phonePrefix(tail string) bool {
	if len(tail) > 18 {
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
	return digits > 0 && digits <= 11
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
