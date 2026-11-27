package detect

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/textutil"
)

const (
	foldWindow     = 512
	foldWordSlack  = 64
	foldNeighbourN = 32
	foldSep        = "\n\x00\n"
)

func foldTriggers(text string) []int {
	var out []int
	for i := 0; i < len(text); {
		if text[i] < utf8.RuneSelf {
			i++
			continue
		}
		f, size, ok := textutil.FoldRuneAt(text, i)
		if ok && (f != '-' && f != ' ' || foldTouchesToken(text, i, size)) {
			out = append(out, i)
		}
		i += size
	}
	return out
}

func foldTouchesToken(text string, i, size int) bool {
	if i > 0 {
		r, _ := utf8.DecodeLastRuneInString(text[:i])
		if textutil.DigitLike(r) {
			return true
		}
		j := i
		for j > 0 && i-j < foldNeighbourN {
			r, n := utf8.DecodeLastRuneInString(text[:j])
			if !unicode.IsLetter(r) {
				break
			}
			j -= n
		}
		if j < i {
			if r, _ := utf8.DecodeRuneInString(text[j:]); unicode.IsUpper(r) {
				return true
			}
		}
	}
	if i+size < len(text) {
		r, _ := utf8.DecodeRuneInString(text[i+size:])
		if textutil.DigitLike(r) || unicode.IsUpper(r) {
			return true
		}
		if f, _, ok := textutil.FoldRuneAt(text, i+size); ok && (unicode.IsUpper(f) || f >= '0' && f <= '9') {
			return true
		}
	}
	return false
}

func needsFold(text string) bool { return len(foldTriggers(text)) > 0 }

func foldWindows(text string, triggers []int) [][2]int {
	var out [][2]int
	for _, t := range triggers {
		s, e := max(0, t-foldWindow), min(len(text), t+foldWindow)
		if k := strings.LastIndexByte(text[s:t], '\n'); k >= 0 {
			s += k + 1
		}
		if k := strings.IndexByte(text[t:e], '\n'); k >= 0 {
			e = t + k
		}
		for n := 0; s > 0 && n < foldWordSlack && !isSpaceByte(text[s-1]); n++ {
			s--
		}
		for n := 0; e < len(text) && n < foldWordSlack && !isSpaceByte(text[e]); n++ {
			e++
		}
		for s > 0 && !utf8.RuneStart(text[s]) {
			s--
		}
		for e < len(text) && !utf8.RuneStart(text[e]) {
			e++
		}
		if len(out) > 0 && s <= out[len(out)-1][1] {
			out[len(out)-1][1] = max(out[len(out)-1][1], e)
			continue
		}
		out = append(out, [2]int{s, e})
	}
	return out
}

func foldText(text string) (string, []int, []int) {
	triggers := foldTriggers(text)
	var b strings.Builder
	var starts, ends []int
	for k, w := range foldWindows(text, triggers) {
		if k > 0 {
			b.WriteString(foldSep)
			for range foldSep {
				starts = append(starts, w[0])
				ends = append(ends, w[0])
			}
		}
		f, ss, ee := textutil.FoldFormsMap(text[w[0]:w[1]])
		b.WriteString(f)
		for i := range ss {
			starts = append(starts, ss[i]+w[0])
			ends = append(ends, ee[i]+w[0])
		}
	}
	return b.String(), starts, ends
}

func foldSpanOK(folded string, s, e int) bool {
	return s >= 0 && e <= len(folded) && s < e && !strings.Contains(folded[s:e], "\x00")
}

func FoldForms(s string) string { return textutil.FoldForms(s) }
