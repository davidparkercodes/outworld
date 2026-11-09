package detect

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/textutil"
)

var lowerNameBefore = wordSet(`hi hey hello dear thanks thank thx ty cc ping love yo morning congrats bye sorry welcome xo xoxo`)
var lowerNameAfter = wordSet(`said says wrote writes asked asks found replied mentioned called texted emailed messaged pinged told tells thinks thought suggested noted confirmed agreed owes sent added posted shared booked paid joined left signed approved fixed pushed turns turned celebrates joins starts leaves arrives arrived`)
var functionAfter = wordSet(`you i we he she they it the a an be not this that your our my his her their there someone anyone to if all any me us them also please access down up out permission`)
var dateBefore = wordSet(`in on of since until till by early late mid last next this every from through before after during around`)
var notSurname = wordSet(`street st avenue ave road rd lane ln drive dr court ct place way boulevard blvd terrace circle parkway highway trail park square inc llc ltd corp co group labs health clinic hospital school university college bank software systems partners holdings associates foundation institute center centre team project`)
var monthsAndDays = wordSet(`january february march april may june july august september october november december monday tuesday wednesday thursday friday saturday sunday`)

func wordSet(s string) map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(s) {
		m[w] = true
	}
	return m
}

func IsMonthOrWeekday(w string) bool { return monthsAndDays[strings.ToLower(w)] }

func NameUseOK(text string, start, end int) bool {
	w := text[start:end]
	if len(textutil.Tokens(w)) != 1 {
		return true
	}
	weak := textutil.IsCommonWord(w) || utf8.RuneCountInString(w) <= 3
	if !weak {
		return true
	}
	switch {
	case textutil.IsAllLower(w):
		return lowerNameContext(text, start, end)
	case textutil.IsAllUpper(w) && utf8.RuneCountInString(w) > 1:
		return lowerNameContext(text, start, end)
	}
	prev, next := wordBefore(text, start), wordAfter(text, end)
	if IsMonthOrWeekday(w) && (dateBefore[strings.ToLower(prev)] || startsDigit(next) || startsDigit(prev)) {
		return false
	}
	if sentenceStart(text, start) && functionAfter[next] {
		return false
	}
	return true
}

func lowerNameContext(text string, start, end int) bool {
	if start > 0 && text[start-1] == '@' {
		return true
	}
	if lowerNameBefore[strings.ToLower(wordBefore(text, start))] {
		return true
	}
	if lowerNameAfter[strings.ToLower(wordAfter(text, end))] {
		return true
	}
	ls, le := lineBounds(text, start, end)
	line := strings.TrimSpace(text[ls:le])
	w := text[start:end]
	if strings.TrimRight(line, ".,!:;-") == w {
		return true
	}
	rest := strings.TrimLeft(text[end:le], " ")
	if strings.HasPrefix(rest, ":") {
		before := strings.TrimSpace(text[ls:start])
		if before == "" || strings.HasSuffix(before, "]") || strings.HasSuffix(before, ">") {
			return true
		}
	}
	return false
}

func lineBounds(text string, start, end int) (int, int) {
	ls := strings.LastIndexByte(text[:start], '\n') + 1
	le := strings.IndexByte(text[end:], '\n')
	if le < 0 {
		return ls, len(text)
	}
	return ls, end + le
}

func wordBefore(text string, i int) string {
	j := i
	for j > 0 {
		r, n := utf8.DecodeLastRuneInString(text[:j])
		if r == '\n' || r == '.' || r == '!' || r == '?' {
			return ""
		}
		if textutil.IsWordRune(r) {
			break
		}
		j -= n
	}
	k := j
	for k > 0 {
		r, n := utf8.DecodeLastRuneInString(text[:k])
		if !textutil.IsWordRune(r) {
			break
		}
		k -= n
	}
	return text[k:j]
}

func wordAfter(text string, i int) string {
	j := i
	for j < len(text) {
		r, n := utf8.DecodeRuneInString(text[j:])
		if r == '\n' {
			return ""
		}
		if textutil.IsWordRune(r) {
			break
		}
		if r != ' ' && r != '\t' {
			return ""
		}
		j += n
	}
	k := j
	for k < len(text) {
		r, n := utf8.DecodeRuneInString(text[k:])
		if !textutil.IsWordRune(r) {
			break
		}
		k += n
	}
	return strings.ToLower(text[j:k])
}

func startsDigit(s string) bool { return s != "" && s[0] >= '0' && s[0] <= '9' }

func sentenceStart(text string, i int) bool {
	for j := i; j > 0; {
		r, n := utf8.DecodeLastRuneInString(text[:j])
		switch {
		case r == '\n':
			return true
		case unicode.IsSpace(r), r == '"', r == '\'', r == '(', r == '*', r == '_', r == '>', r == '-':
			j -= n
			continue
		}
		return r == '.' || r == '!' || r == '?' || r == ':'
	}
	return true
}

func CapitalizedNameAfter(text string, i int) int {
	if i >= len(text) || text[i] != ' ' {
		return i
	}
	j := i + 1
	end := nameWordEnd(text, j)
	if end <= j {
		return i
	}
	w := text[j:end]
	if textutil.IsCommonWord(w) || IsMonthOrWeekday(w) || textutil.IsAllUpper(w) || notSurname[strings.ToLower(w)] {
		return i
	}
	return end
}

func nameWordEnd(text string, j int) int {
	r, _ := utf8.DecodeRuneInString(text[j:])
	if !unicode.IsUpper(r) {
		return j
	}
	k := j
	for k < len(text) {
		r, n := utf8.DecodeRuneInString(text[k:])
		if unicode.IsLetter(r) {
			k += n
			continue
		}
		if (r == '-' || r == '\'' || r == '’') && k+n < len(text) {
			nr, _ := utf8.DecodeRuneInString(text[k+n:])
			if unicode.IsUpper(nr) {
				k += n
				continue
			}
		}
		break
	}
	if utf8.RuneCountInString(text[j:k]) < 2 {
		return j
	}
	return k
}

func CapitalizedWordBefore(text string, i int) (int, string) {
	if i < 2 || text[i-1] != ' ' {
		return i, ""
	}
	k := i - 1
	for k > 0 {
		r, n := utf8.DecodeLastRuneInString(text[:k])
		if !unicode.IsLetter(r) {
			break
		}
		k -= n
	}
	if k > 0 {
		r, _ := utf8.DecodeLastRuneInString(text[:k])
		if textutil.IsWordRune(r) {
			return i, ""
		}
	}
	if nameWordEnd(text, k) != i-1 {
		return i, ""
	}
	return k, text[k : i-1]
}
