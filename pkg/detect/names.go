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
	if NamedThingUse(text, start, end) {
		return false
	}
	if IsTechTerm(w) {
		rest := text[end:]
		return lowerNameContext(text, start, end) || personEvidence(text, start, end) || strings.HasPrefix(rest, "'s ") || strings.HasPrefix(rest, "’s ")
	}
	return nameUseOK(text, start, end)
}

func nameUseOK(text string, start, end int) bool {
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
	if speakerColon(rest) {
		before := strings.TrimSpace(text[ls:start])
		if before == "" || strings.HasSuffix(before, "]") || strings.HasSuffix(before, ">") {
			return true
		}
	}
	return false
}

const contextWindow = 320

func windowStart(text string, pos int) int {
	lo := pos - contextWindow
	if lo <= 0 {
		return 0
	}
	for lo < pos && !utf8.RuneStart(text[lo]) {
		lo++
	}
	return lo
}

func windowEnd(text string, pos int) int {
	hi := pos + contextWindow
	if hi >= len(text) {
		return len(text)
	}
	for hi > pos && !utf8.RuneStart(text[hi]) {
		hi--
	}
	return hi
}

func lineBounds(text string, start, end int) (int, int) {
	lo, hi := windowStart(text, start), windowEnd(text, end)
	ls := lo
	if i := strings.LastIndexByte(text[lo:start], '\n'); i >= 0 {
		ls = lo + i + 1
	}
	le := hi
	if i := strings.IndexByte(text[end:hi], '\n'); i >= 0 {
		le = end + i
	}
	return ls, le
}

const wordScanLimit = 64

func wordBefore(text string, i int) string {
	j := i
	for j > 0 && i-j < wordScanLimit {
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
	for k > 0 && j-k < wordScanLimit {
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
	for j < len(text) && j-i < wordScanLimit {
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
	for k < len(text) && k-j < wordScanLimit {
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
	if textutil.IsCommonWord(w) || IsMonthOrWeekday(w) || textutil.IsAllUpper(w) || notSurname[strings.ToLower(w)] || thingNouns[strings.ToLower(w)] || compoundHead(text, j, end) {
		return i
	}
	return end
}

func compoundHead(text string, j, end int) bool {
	r, n := utf8.DecodeRuneInString(text[end:])
	if !textutil.IsNameJoiner(r) {
		return false
	}
	k := end + n
	for k < len(text) && k-end < maxDashGap && text[k] >= 'a' && text[k] <= 'z' {
		k++
	}
	return k > end+n && particleCompound(text[j:k])
}

func nameWordEnd(text string, j int) int {
	r, _ := utf8.DecodeRuneInString(text[j:])
	if !unicode.IsUpper(r) {
		return j
	}
	k := j
	for k < len(text) {
		if k-j >= wordScanLimit {
			return j
		}
		r, n := utf8.DecodeRuneInString(text[k:])
		if unicode.IsLetter(r) {
			k += n
			continue
		}
		if (textutil.IsHyphen(r) || r == '\'' || r == '’') && k+n < len(text) {
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
		if i-1-k >= wordScanLimit {
			return i, ""
		}
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

func speakerColon(rest string) bool {
	if !strings.HasPrefix(rest, ":") {
		return false
	}
	return len(rest) == 1 || rest[1] == ' ' || rest[1] == '\t'
}

var personVerbs = wordSet(`said says called calls emailed emails texted texts replied replies asked asks told tells wrote writes joined joins messaged pinged mentioned thinks thought agreed suggested confirmed laughed smiled nodded waved phoned answered apologized apologised volunteered resigned retired`)

var personWorkVerbs = wordSet(`reviewed approved merged fixed pushed shipped deployed committed signed sent posted shared booked paid added opened closed filed reported built tested updated owns leads owned led commented rejected requested submitted noticed found`)

var personMotionVerbs = wordSet(`went came left met arrived visited stopped headed drove flew walked ate`)

var personVerbBase = wordSet(`join call email text reply ask tell write review present attend lead cover help handle follow send confirm meet visit drive bring host speak talk be know need want take come go start leave work run check look sign approve fix own share update`)

var personModals = wordSet(`will would can could should might must may won't can't couldn't wouldn't shouldn't didn't doesn't did does`)

var personStates = wordSet(`out sick away off back here late available free busy ooo traveling travelling joining leaving coming running covering reviewing presenting driving working visiting hosting attending leading calling waiting retiring on`)

var personLeftAfter = wordSet(`a me us him her early late today yesterday the`)

var directedVerbs = wordSet(`ask asked asking tell told telling email emailed emailing call called calling ping pinged pinging cc cced cc'd text texted texting message messaged messaging thank thanked thanks invite invited remind reminded notify notified phone phoned met meet meeting saw see seeing visit visited hired fired promoted introduce introduced congratulate congratulated`)

var directedPreps = wordSet(`with to from`)

var directedPrepVerbs = wordSet(`call calls called calling met meet meeting meets spoke speak speaking spoken talked talk talking chat chatted chatting heard hear lunch dinner coffee breakfast drinks sync synced replied reply replying wrote write writing listen listened`)

var roleBefore = wordSet(`colleague coworker teammate manager director engineer intern recruiter professor coach cousin uncle aunt nurse attorney lawyer neighbor neighbour friend contractor tenant landlord plumber dentist doctor teacher officer agent detective judge senator councilman councilwoman coordinator supervisor technician developer designer accountant mister`)

func wordsAfter(text string, e, n int) []string {
	var out []string
	j := e
	for len(out) < n {
		if j >= len(text) || text[j] != ' ' {
			break
		}
		j++
		k := j
		for k < len(text) && k-j < wordScanLimit {
			r, sz := utf8.DecodeRuneInString(text[k:])
			if !textutil.IsWordRune(r) && !((r == '\'' || r == '’') && k > j) {
				break
			}
			k += sz
		}
		if k == j {
			break
		}
		out = append(out, strings.ToLower(strings.TrimRight(text[j:k], "'’")))
		j = k
	}
	return out
}

func wordsBefore(text string, s, n int) []string {
	var out []string
	j := s
	for len(out) < n {
		if j <= 0 || text[j-1] != ' ' {
			break
		}
		j--
		if j > 0 && text[j-1] == ':' && len(out) == 0 {
			j--
		}
		k := j
		for k > 0 && j-k < wordScanLimit {
			r, sz := utf8.DecodeLastRuneInString(text[:k])
			if !textutil.IsWordRune(r) && r != '\'' {
				break
			}
			k -= sz
		}
		if k == j {
			break
		}
		out = append(out, strings.ToLower(text[k:j]))
		j = k
	}
	return out
}

func personVerbAfter(text string, e int) bool {
	w := wordsAfter(text, e, 3)
	if len(w) == 0 {
		return false
	}
	switch {
	case personVerbs[w[0]]:
		return true
	case w[0] == "left":
		return len(w) > 1 && personLeftAfter[w[1]]
	case personModals[w[0]]:
		if len(w) > 2 && (w[1] == "not" || w[1] == "also" || w[1] == "be") {
			return personVerbBase[w[2]] || personStates[w[2]]
		}
		return len(w) > 1 && personVerbBase[w[1]]
	case w[0] == "is" || w[0] == "was" || w[0] == "isn't" || w[0] == "wasn't":
		return len(w) > 1 && personStates[w[1]]
	case w[0] == "has" || w[0] == "had":
		return len(w) > 1 && (personVerbs[w[1]] || w[1] == "left" || w[1] == "been" && len(w) > 2 && personStates[w[2]])
	}
	return false
}

func PersonEvidence(text string, s, e int) bool {
	return PersonActionAfter(text, e) || personDirectedBefore(text, s)
}

func PersonActionAfter(text string, e int) bool {
	if personActionAt(text, e) {
		return true
	}
	if w := wordAfter(text, e); e < len(text) && text[e] == ' ' && nameParticles[strings.ToLower(w)] && w != "" {
		next := e + 1 + len(w)
		if StreetAbbrev(w) && next < len(text) && text[next] == '.' {
			next++
		}
		return personActionAt(text, next)
	}
	return false
}

func StreetAbbrev(w string) bool { return abbrevParticles[strings.ToLower(strings.TrimSuffix(w, "."))] }

func personActionAt(text string, e int) bool {
	w := wordAfter(text, e)
	return personVerbAfter(text, e) || personWorkVerbs[w] || personMotionVerbs[w] || lowerNameAfter[w]
}

func personDirectedBefore(text string, s int) bool {
	w := wordsBefore(text, s, 2)
	if len(w) == 0 {
		return false
	}
	if directedVerbs[w[0]] {
		return true
	}
	return len(w) > 1 && directedPreps[w[0]] && directedPrepVerbs[w[1]]
}

func sentenceBounds(text string, s, e int) (int, int) {
	lo, hi := windowStart(text, s), windowEnd(text, e)
	a := lo
	if i := strings.LastIndexAny(text[lo:s], ".!?\n"); i >= 0 {
		a = lo + i + 1
	}
	b := hi
	if i := strings.IndexAny(text[e:hi], ".!?\n"); i >= 0 {
		b = e + i
	}
	return a, b
}
