package score

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

func isWordByte(s string, i int, forward bool) bool {
	var r rune
	if forward {
		r, _ = utf8.DecodeRuneInString(s[i:])
	} else {
		r, _ = utf8.DecodeLastRuneInString(s[:i])
	}
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func wordEdge(s string) (bool, bool) {
	if s == "" {
		return false, false
	}
	first, _ := utf8.DecodeRuneInString(s)
	last, _ := utf8.DecodeLastRuneInString(s)
	alnum := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }
	return alnum(first), alnum(last)
}

func CountWord(haystack, needle string) int {
	if needle == "" {
		return 0
	}
	needStart, needEnd := wordEdge(needle)
	n := 0
	for off := 0; off <= len(haystack)-len(needle); {
		i := strings.Index(haystack[off:], needle)
		if i < 0 {
			break
		}
		i += off
		j := i + len(needle)
		ok := true
		if needStart && i > 0 && isWordByte(haystack, i, false) {
			ok = false
		}
		if needEnd && j < len(haystack) && isWordByte(haystack, j, true) {
			ok = false
		}
		if ok {
			n++
		}
		off = i + 1
	}
	return n
}

func CountFold(haystack, needle string) int {
	return CountWord(strings.ToLower(haystack), strings.ToLower(needle))
}

var markdownStripper = strings.NewReplacer("**", "", "__", "", "*", "", "_", "", "`", "")

func StripMarkdown(s string) string { return markdownStripper.Replace(s) }

var reDigitRun = regexp.MustCompile(`\d(?:[\d \-.()/]*\d)?`)

func DigitRuns(s string) []string {
	var out []string
	for _, m := range reDigitRun.FindAllString(s, -1) {
		var b strings.Builder
		for i := 0; i < len(m); i++ {
			if m[i] >= '0' && m[i] <= '9' {
				b.WriteByte(m[i])
			}
		}
		out = append(out, b.String())
	}
	return out
}

func CountDigits(runs []string, digits string) int {
	n := 0
	for _, r := range runs {
		n += strings.Count(r, digits)
	}
	return n
}

var reToken = regexp.MustCompile(`[\p{L}\p{N}][\p{L}\p{N}_\-]*[\p{L}\p{N}]`)

func Tokens(s string) map[string]bool {
	out := map[string]bool{}
	for _, t := range reToken.FindAllString(strings.ToLower(s), -1) {
		if len(t) >= 3 {
			out[t] = true
		}
	}
	return out
}
