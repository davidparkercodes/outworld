package detect

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/entity"
)

const (
	sepCueGap    = 25
	markdownRun  = 2
	cueSuffixMax = 8
)

var (
	ssnCueWords    = []string{"social security number", "social security no", "social security num", "social security", "taxpayer id", "tax id", "taxid", "ssns", "ssn", "ss#", "itin", "tin", "social"}
	cueKeySuffix   = []string{"number", "num", "nr", "no"}
	suffixOnlyCues = wordSet(`callback`)
	listJoiners    = []string{", ", ",", "; ", ";", " / ", "/", " & ", " and ", " or "}
	phoneCueWords  = sortedCues(phoneLabelWords)
)

func sortedCues(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for w := range set {
		out = append(out, w)
	}
	sort.Slice(out, func(a, b int) bool {
		if len(out[a]) != len(out[b]) {
			return len(out[a]) > len(out[b])
		}
		return out[a] < out[b]
	})
	return out
}

func gateCues(cues []string) func(string, string) bool { return gateLower(strings.Join(cues, "|")) }

func sepRune(r rune) bool { return r != '\n' && r != '\r' && textutil.IsPhoneSeparator(r) }

func wordLetter(r rune) bool {
	return (unicode.IsLetter(r) || unicode.IsDigit(r)) && !textutil.IsCJKLetter(r)
}

func markdownRune(r rune) bool { return r == '_' || r == '*' || r == '`' }

func edgeOK(text string, i int, left bool, reject string) bool {
	next := func(j int) (rune, int) {
		if left {
			if j <= 0 {
				return 0, 0
			}
			r, n := utf8.DecodeLastRuneInString(text[:j])
			return r, j - n
		}
		if j >= len(text) {
			return 0, j
		}
		r, n := utf8.DecodeRuneInString(text[j:])
		return r, j + n
	}
	r, j := next(i)
	if r == 0 {
		return true
	}
	if markdownRune(r) {
		for k := 1; k < markdownRun; k++ {
			if r2, j2 := next(j); r2 == r {
				j = j2
			}
		}
		r2, _ := next(j)
		return !wordLetter(r2) && r2 != '_'
	}
	return !wordLetter(r) && !strings.ContainsRune(reject, r)
}

func cueLeftOK(text string, c int) bool {
	if c == 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(text[:c])
	camel := unicode.IsLower(r) && text[c] >= 'A' && text[c] <= 'Z'
	if r == '/' {
		return slashLabelBefore(text, c-1)
	}
	return !wordLetter(r) || camel
}

func cuePluralEnd(text string, cue string, e int) (int, bool) {
	if e < len(text) && (text[e] == 's' || text[e] == 'S') && isAlphaByte(cue[len(cue)-1]) && (e+1 == len(text) || !isAlphaByte(text[e+1])) {
		return e + 1, true
	}
	return e, false
}

func cueSuffixEnd(text string, e int) (int, bool) {
	low := strings.ToLower(text[e:min(len(text), e+cueSuffixMax)])
	for _, suf := range cueKeySuffix {
		for _, join := range []string{"_", "-", ".", " ", ""} {
			if !strings.HasPrefix(low, join+suf) {
				continue
			}
			n := len(join + suf)
			if n < len(low) && low[n] == 's' {
				n++
			}
			if len(low) == n || !isAlphaByte(low[n]) {
				return e + n, true
			}
		}
	}
	return e, false
}

func cueRightOK(text string, e int) bool {
	if e >= len(text) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(text[e:])
	if wordLetter(r) && !unicode.IsDigit(r) || r == '_' || r == '/' {
		return false
	}
	return r != '-' || e+1 >= len(text) || !isAlphaByte(text[e+1])
}

func cueInURLOK(text string, c int) bool {
	if !insideURL(text, c) {
		return true
	}
	k := keyStart(text, c, 0)
	return k > 0 && strings.IndexByte("?&;", text[k-1]) >= 0
}

func cueSpecialOK(text string, c, e int, cue string, suffixed bool) bool {
	if suffixOnlyCues[strings.ToLower(cue)] && !suffixed {
		return false
	}
	switch strings.ToLower(text[c : c+len(cue)]) {
	case "tin":
		return text[c:c+len(cue)] == "TIN" || e < len(text) && (text[e] == ':' || text[e] == '#')
	case "ph":
		return e < len(text) && (text[e] == '.' || text[e] == ':' || text[e] == ' ' || text[e] == '\t')
	}
	return true
}

func cueAt(text string, c int, cue string) (int, bool) {
	if !cueLeftOK(text, c) {
		return 0, false
	}
	e, plural := cuePluralEnd(text, cue, c+len(cue))
	if plural && e < len(text) && text[e] == '=' {
		return 0, false
	}
	e, suffixed := cueSuffixEnd(text, e)
	if e < len(text) && text[e] == '/' {
		e = slashLabelAfter(text, e)
	}
	if !cueRightOK(text, e) || !cueInURLOK(text, c) || !cueSpecialOK(text, c, e, cue, suffixed) {
		return 0, false
	}
	return e, true
}

func slashLabelAfter(text string, slash int) int {
	j := slash + 1
	for j < len(text) && isAlphaByte(text[j]) {
		j++
	}
	if j > slash+1 && phoneLabelWords[strings.ToLower(text[slash+1:j])] {
		return j
	}
	return slash
}

func slashLabelBefore(text string, slash int) bool {
	k := slash
	for k > 0 && isAlphaByte(text[k-1]) {
		k--
	}
	return k < slash && phoneLabelWords[strings.ToLower(text[k:slash])] && (k == 0 || !isAlphaByte(text[k-1]))
}

func digitsAt(text string, i, n int) (int, bool) {
	for k := 0; k < n; k++ {
		if i >= len(text) || !isDigitByte(text[i]) {
			return 0, false
		}
		i++
	}
	return i, true
}

func sepAt(text string, i int) (rune, int) {
	if i >= len(text) {
		return 0, i
	}
	if r, size := utf8.DecodeRuneInString(text[i:]); sepRune(r) {
		return r, i + size
	}
	return 0, i
}

func groupsAt(text string, i int, sizes []int, sep rune) (int, bool) {
	for g, n := range sizes {
		j, ok := digitsAt(text, i, n)
		if !ok {
			return 0, false
		}
		i = j
		if g == len(sizes)-1 {
			break
		}
		r, j := sepAt(text, i)
		if r == 0 || sep != 0 && r != sep {
			return 0, false
		}
		sep, i = r, j
	}
	return i, true
}

func plusOneAt(text string, i int) (int, bool) {
	if i < len(text) && text[i] == '+' {
		i++
	}
	if i >= len(text) || text[i] != '1' {
		return 0, false
	}
	_, j := sepAt(text, i+1)
	return groupsAt(text, j, []int{3, 3, 4}, 0)
}

func bracketAt(text string, i int) (int, bool) {
	if i < len(text) && text[i] == '+' {
		if i+1 >= len(text) || text[i+1] != '1' {
			return 0, false
		}
		i++
	}
	if i < len(text) && text[i] == '1' {
		_, i = sepAt(text, i+1)
	}
	if i >= len(text) || text[i] != '(' {
		return 0, false
	}
	j, ok := digitsAt(text, i+1, 3)
	if !ok || j >= len(text) || text[j] != ')' {
		return 0, false
	}
	_, j = sepAt(text, j+1)
	if j, ok = digitsAt(text, j, 3); !ok {
		return 0, false
	}
	r, j := sepAt(text, j)
	if r == 0 {
		return 0, false
	}
	return digitsAt(text, j, 4)
}

func phoneShapeAt(text string, i int) (int, bool) {
	if e, ok := plusOneAt(text, i); ok {
		return e, true
	}
	if e, ok := bracketAt(text, i); ok {
		return e, true
	}
	return groupsAt(text, i, []int{3, 3, 4}, 0)
}

func ssnShapeAt(text string, i int) (int, bool) { return groupsAt(text, i, []int{3, 2, 4}, 0) }

type cueMatch struct {
	start, end int
	cue        string
}

type cueMatcher func(text string, c int, cue string) (int, bool)

func cueMatchesIn(text, low string, base int, cues []string, at cueMatcher) []cueMatch {
	var out []cueMatch
	seen := map[int]bool{}
	for _, cue := range cues {
		for from := 0; from < len(low); {
			k := strings.Index(low[from:], cue)
			if k < 0 {
				break
			}
			c := base + from + k
			from += k + len(cue)
			if seen[c] {
				continue
			}
			if e, ok := at(text, c, cue); ok {
				seen[c] = true
				out = append(out, cueMatch{start: c, end: e, cue: cue})
			}
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].start < out[b].start })
	return out
}

type spanSet []entity.Span

func (t spanSet) overlaps(s, e int) bool {
	k := sort.Search(len(t), func(i int) bool { return t[i].End > s })
	return k < len(t) && t[k].Start < e
}

func (t *spanSet) add(s, e int) {
	k := sort.Search(len(*t), func(i int) bool { return (*t)[i].Start >= s })
	*t = append(*t, entity.Span{})
	copy((*t)[k+1:], (*t)[k:])
	(*t)[k] = entity.Span{Start: s, End: e}
}

type cuedShape struct {
	startOK func(byte) bool
	shape   func(string, int) (int, bool)
	valid   func(string) bool
}

func (c cuedShape) at(text string, i int) (int, bool) {
	e, ok := c.shape(text, i)
	if !ok {
		return 0, false
	}
	return e, edgeOK(text, e, false, "") && c.valid(digitsOnly(text[i:e]))
}

func (c cuedShape) valueAfterCue(text string, ce int) (int, int, bool) {
	for i := ce; i < len(text) && i-ce < sepCueGap && text[i] != '\n'; i++ {
		if !c.startOK(text[i]) {
			continue
		}
		if i == ce || edgeOK(text, i, true, "_)]}") {
			if e, ok := c.at(text, i); ok {
				return i, e, true
			}
			if j, short := shortGroupAt(text, i); short && j < len(text) && c.startOK(text[j]) {
				if e, ok := c.at(text, j); ok {
					return j, e, true
				}
			}
		}
		if isDigitByte(text[i]) {
			return 0, 0, false
		}
	}
	return 0, 0, false
}

var continuationWords = wordSet(`work home office direct main desk`)

func continuationAt(text string, j int) int {
	k := j
	for k < len(text) && isAlphaByte(text[k]) {
		k++
	}
	if k == j {
		return j
	}
	w := strings.ToLower(text[j:k])
	if !continuationWords[w] && !phoneLabelWords[w] {
		return -1
	}
	if k < len(text) && text[k] == ':' {
		k++
	}
	for n := 0; k < len(text) && n < 2 && text[k] == ' '; n++ {
		k++
	}
	return k
}

func (c cuedShape) listAfter(text string, e int, taken *spanSet, out []entity.Span) []entity.Span {
	for {
		j := listNext(text, e)
		if j >= 0 && j < len(text) && isAlphaByte(text[j]) {
			j = continuationAt(text, j)
		}
		if j < 0 || j >= len(text) || !c.startOK(text[j]) {
			return out
		}
		e2, ok := c.at(text, j)
		if !ok || taken.overlaps(j, e2) {
			return out
		}
		out = append(out, entity.Span{Start: j, End: e2})
		taken.add(j, e2)
		e = e2
	}
}

func scanCued(text, lower string, cues []string, startOK func(byte) bool, shape func(string, int) (int, bool), valid func(string) bool) []entity.Span {
	c := cuedShape{startOK: startOK, shape: shape, valid: valid}
	var out []entity.Span
	var taken spanSet
	for _, m := range cueMatchesIn(text, sameLengthLower(text, lower), 0, cues, cueAt) {
		s, e, ok := c.valueAfterCue(text, m.end)
		if !ok || taken.overlaps(s, e) {
			continue
		}
		out = append(out, entity.Span{Start: s, End: e})
		taken.add(s, e)
		out = c.listAfter(text, e, &taken, out)
	}
	return out
}

func listNext(text string, e int) int {
	for _, j := range listJoiners {
		if strings.HasPrefix(text[e:], j) {
			return e + len(j)
		}
	}
	return -1
}

const shortGroupMax = 2

func shortGroupAt(text string, i int) (int, bool) {
	j := i
	for j < len(text) && j-i < shortGroupMax && isDigitByte(text[j]) {
		j++
	}
	if j == i || j >= len(text) || text[j] != ' ' {
		return 0, false
	}
	return j + 1, true
}

func phoneStart(c byte) bool { return isDigitByte(c) || c == '(' || c == '+' }

func scanSepSSN(text, lower string) []entity.Span {
	out := scanCued(text, lower, ssnCueWords, isDigitByte, ssnShapeAt, func(d string) bool { return len(d) == 9 })
	return append(out, scanGlued(text, lower, ssnCueWords, gluedSSNAt)...)
}

func scanSepPhone(text, lower string) []entity.Span {
	out := scanCued(text, lower, phoneCueWords, phoneStart, phoneShapeAt, nanpRun)
	return append(out, scanGlued(text, lower, phoneCueWords, gluedPhoneAt)...)
}

var nationalAtRE = regexp.MustCompile(`^\(?0\d{1,4}\)?(?:[ .\-]\d{2,8}){1,5}`)

func gluedSSNAt(text string, i int) (int, bool) {
	e, ok := ssnShapeAt(text, i)
	if !ok {
		e, ok = digitsAt(text, i, 9)
	}
	return e, ok && ssnValidDigits(digitsOnly(text[i:e]))
}

func gluedPhoneAt(text string, i int) (int, bool) {
	if e, ok := phoneShapeAt(text, i); ok && nanpRun(digitsOnly(text[i:e])) {
		return e, true
	}
	if m := nationalAtRE.FindStringIndex(text[i:]); m != nil && nationalPhoneValid(text[i:], 0, m[1]) {
		return i + m[1], true
	}
	return 0, false
}

func gluedCueAt(text string, c int, cue string) (int, bool) {
	e := c + len(cue)
	w := text[c:e]
	joined := e < len(text) && (text[e] == '_' || text[e] == '.')
	title := len(w) > 1 && w[:1] == strings.ToUpper(w[:1]) && w[1:] == strings.ToLower(w[1:])
	if w != strings.ToUpper(w) && w != strings.ToLower(w) && (!joined || !title) {
		return 0, false
	}
	if c > 0 {
		if r, _ := utf8.DecodeLastRuneInString(text[:c]); wordLetter(r) || r == '_' {
			return 0, false
		}
	}
	if joined {
		e++
	}
	if e >= len(text) || !isDigitByte(text[e]) {
		return 0, false
	}
	return e, true
}

func gluedAlphaCueAt(text string, c int, cue string) (int, bool) {
	if !isAlphaByte(cue[len(cue)-1]) {
		return 0, false
	}
	return gluedCueAt(text, c, cue)
}

func scanGlued(text, lower string, cues []string, shape func(string, int) (int, bool)) []entity.Span {
	var out []entity.Span
	var taken spanSet
	for _, m := range cueMatchesIn(text, sameLengthLower(text, lower), 0, cues, gluedAlphaCueAt) {
		e, ok := shape(text, m.end)
		if !ok || !edgeOK(text, e, false, "") || taken.overlaps(m.end, e) {
			continue
		}
		out = append(out, entity.Span{Start: m.end, End: e})
		taken.add(m.end, e)
	}
	return out
}

func scanPlusOne(text, _ string) []entity.Span {
	var out []entity.Span
	for from := 0; from < len(text); {
		k := strings.Index(text[from:], "+1")
		if k < 0 {
			break
		}
		i := from + k
		from = i + 2
		if !edgeOK(text, i, true, "_)]}=") {
			continue
		}
		e, ok := plusOneAt(text, i)
		if !ok || !edgeOK(text, e, false, "") || !nanpRun(digitsOnly(text[i:e])) {
			continue
		}
		out = append(out, entity.Span{Start: i, End: e})
		from = e
	}
	return out
}
