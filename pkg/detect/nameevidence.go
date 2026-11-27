package detect

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/namedata"
	"github.com/davidparkercodes/outworld/internal/textutil"
)

var ordinaryObjects = wordSet(`as the a an my your our his her their its this that these those down up off all every each access permission permissions it them me us`)

var willSubjects = wordSet(`you we they he she i it this that the there these those anyone someone everyone`)

var whWords = wordSet(`who what when where why how which whose`)

var notThirdPerson = wordSet(`this these those us his its yes always perhaps thus`)

var monthPlaces = wordSet(`there home gone`)

var monthAbsent = wordSet(`ooo pto away traveling travelling`)

var monthPrepositions = wordSet(`on in with at out`)

var monthSeasonNouns = wordSet(`summer winter spring autumn fall q1 q2 q3 q4 h1 h2 season quarter year calendar month`)

var irregularPast = wordSet(`got was were had did said told took gave made sent ran saw thought brought bought found knew became began wrote spoke sat stood kept held felt lost won paid put read heard`)

var willCapsFunction = wordSet(`be do not our your a an anyone`)

var nonVerbAfterName = wordSet(`from on at in with for of and or but by about over under near via into onto across around per like than nor to is was has does`)

const (
	messageHeaderCue = " to "
	monthCopulaWords = 4
)

var roseToNouns = wordSet(`prominence power fame`)

var roseToTheNouns = wordSet(`occasion challenge top`)

var monthStrongCues = wordSet(`during since early late mid last next until till through in this every before after around of by due is as`)

var monthNouns = wordSet(`release releases timeline launch deadline sprint cycle quarter review update updates meeting schedule cohort batch newsletter issue edition report rollout milestone window plan roadmap build sale event payroll invoice close notes board offsite trip holiday holidays break term semester intake`)

var monthDeterminers = wordSet(`the this next last our that each every their your`)

var monthConnectors = wordSet(`and or to through thru until till vs`)

const (
	monthSepWindow  = 4
	monthPairWindow = 12
)

var monthPersonCues = [][2]string{{"loop", "in"}, {"waiting", "on"}, {"blocked", "on"}, {"reviewed", "by"}, {"approved", "by"}, {"assigned", "to"}, {"wait", "on"}, {"signed", "by"}}

var capsPersonCues = wordSet(`PING CC FROM WITH TO ASK TELL`)

var ordinaryNext = map[string]map[string]bool{
	"mark":  wordSet(`down my complete completed done resolved read unread fixed`),
	"hope":  wordSet(`you this that it we they all everyone`),
	"grant": wordSet(`writing proposal proposals application applications funding money deadline program programs`),
	"rose":  wordSet(`gold garden gardens petal petals water oil bush bushes quartz wine`),
	"grace": wordSet(`period periods note notes`),
	"bill":  wordSet(`of payment pay`),
	"frank": wordSet(`discussion conversation talk assessment opinion`),
	"may":   wordSet(`i we you be not`),
}

func OrdinaryWordUse(text string, start, end int) bool {
	if start < 0 || end > len(text) || start >= end {
		return false
	}
	w := text[start:end]
	if len(textutil.Tokens(w)) != 1 {
		return false
	}
	if whQuestionBefore(text, start) || coordinatedWithI(text, start, end) || coordinatedWithName(text, start, end) {
		return false
	}
	if IsMonthOrWeekday(w) {
		if c := monthCopula(text, end); c != 0 {
			return c > 0
		}
	}
	if personVerbAfter(text, end) || messageHeader(text, end) {
		return false
	}
	if IsMonthOrWeekday(w) && monthUse(text, start, end) {
		return true
	}
	switch {
	case textutil.IsAllLower(w):
		return !lowerNameContext(text, start, end)
	case textutil.IsAllUpper(w) && utf8.RuneCountInString(w) > 1:
		if capsPersonCues[rawWordBefore(text, start)] {
			return false
		}
		if capsNeighbour(text, start, end) {
			return true
		}
	}
	if NamedThingUse(text, start, end) || compoundAfter(text, end) || theNameOf(text, start, end) {
		return true
	}
	if !ambiguousStart(text, start, end) {
		return false
	}
	return ordinaryShape(text, start, end)
}

func capsNeighbour(text string, start, end int) bool {
	for _, n := range []string{rawWordBefore(text, start), rawWordAfter(text, end)} {
		if textutil.HasLetter(n) && textutil.IsAllUpper(n) {
			return true
		}
	}
	return false
}

func rawWordBefore(text string, i int) string {
	j := i
	for j > 0 && i-j < wordScanLimit && (text[j-1] == ' ' || text[j-1] == '\t') {
		j--
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

func rawWordAfter(text string, i int) string {
	j := i
	for j < len(text) && j-i < wordScanLimit && (text[j] == ' ' || text[j] == '\t') {
		j++
	}
	k := j
	for k < len(text) && k-j < wordScanLimit {
		r, n := utf8.DecodeRuneInString(text[k:])
		if !textutil.IsWordRune(r) {
			break
		}
		k += n
	}
	return text[j:k]
}

func ambiguousStart(text string, start, end int) bool {
	if sentenceStart(text, start) {
		return true
	}
	ls, le := lineBounds(text, start, end)
	lead := strings.TrimSpace(text[ls:start])
	if lead == "" || codeCommentLead(lead) || listMarker(lead) {
		return true
	}
	return titleLine(text[ls:le])
}

func codeCommentLead(lead string) bool {
	return lead == "//" || lead == "/*" || strings.HasSuffix(lead, " //") || strings.HasSuffix(lead, " /*") || strings.HasSuffix(lead, "\t//")
}

func listMarker(lead string) bool {
	if strings.Trim(lead, "#*-+>•") == "" {
		return true
	}
	for _, box := range []string{"[ ]", "[x]", "[X]"} {
		if b, ok := strings.CutSuffix(lead, box); ok && strings.Trim(strings.TrimSpace(b), "*-+") == "" && strings.TrimSpace(b) != "" {
			return true
		}
	}
	d := strings.TrimRight(lead, ".)")
	if d == lead || d == "" {
		return false
	}
	for _, r := range d {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func titleLine(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" || strings.ContainsAny(line[len(line)-1:], ".?!,;") {
		return false
	}
	words := 0
	for _, f := range strings.Fields(strings.TrimLeft(line, "#*-+> ")) {
		if !textutil.HasLetter(f) {
			continue
		}
		words++
		if utf8.RuneCountInString(f) > 3 && !textutil.StartsUpper(f) {
			return false
		}
	}
	return words >= 2 && words <= 12
}

func inCodeComment(text string, start, end int) bool {
	ls, _ := lineBounds(text, start, end)
	lead := strings.TrimSpace(text[ls:start])
	return strings.HasPrefix(lead, "//") || strings.HasPrefix(lead, "/*") || strings.Contains(lead, " //") || strings.Contains(lead, " /*")
}

func ordinaryShape(text string, start, end int) bool {
	rest := text[end:]
	for _, suf := range []string{"'s", "’s"} {
		if strings.HasPrefix(rest, suf) {
			return inCodeComment(text, start, end)
		}
	}
	next := wordsAfter(text, end, 2)
	if len(next) == 0 {
		return false
	}
	lw := strings.ToLower(text[start:end])
	n := next[0]
	if (lw == "rose" || lw == "fell") && (startsDigit(n) || strings.HasPrefix(text[end:], " %")) {
		return true
	}
	if lw == "will" {
		return willElided(text, end, n)
	}
	if lw == "rose" && n == "to" {
		more := wordsAfter(text, end, 3)
		return len(more) > 1 && (roseToNouns[more[1]] || (more[1] == "the" && len(more) > 2 && roseToTheNouns[more[2]]))
	}
	if ordinaryObjects[n] || ordinaryNext[lw][n] {
		return true
	}
	return n == "and" && len(next) > 1 && next[1] == "sweep"
}

func compoundAfter(text string, end int) bool {
	if end >= len(text) {
		return false
	}
	h, n := utf8.DecodeRuneInString(text[end:])
	if !textutil.IsHyphen(h) || end+n >= len(text) {
		return false
	}
	r, _ := utf8.DecodeRuneInString(text[end+n:])
	return unicode.IsLower(r)
}

var monthDashes = append([]string{"\u2013", "\u2014"}, runeStrings(textutil.HyphenVariants)...)

func runeStrings(rs []rune) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = string(r)
	}
	return out
}

func theNameOf(text string, start, end int) bool {
	if strings.ToLower(wordBefore(text, start)) != "the" {
		return false
	}
	w := wordsAfter(text, end, 1)
	return len(w) == 1 && (w[0] == "to" || w[0] == "of")
}

func monthUse(text string, start, end int) bool {
	if personDirectedBefore(text, start) || monthPersonPair(text, start) {
		return false
	}
	rest := text[end:]
	if strings.HasPrefix(rest, "'s") || strings.HasPrefix(rest, "’s") {
		return false
	}
	if next := rawWordAfter(text, end); strings.HasPrefix(rest, " ") && textutil.StartsUpper(next) && !IsMonthOrWeekday(next) && KnownName(next) {
		return false
	}
	if startsDigit(wordAfter(text, end)) || monthNeighbour(text, start, end) {
		return true
	}
	if next := wordsAfter(text, end, 1); len(next) == 1 && monthNouns[next[0]] {
		return true
	}
	prev := strings.ToLower(wordBefore(text, start))
	if monthDeterminers[prev] {
		return true
	}
	if nounSubjectIs(text, start) || (monthStrongCues[prev] && prev != "is") {
		return true
	}
	if prev == "for" {
		ls, le := lineBounds(text, start, end)
		return titleLine(text[ls:le])
	}
	return false
}

func monthPersonPair(text string, start int) bool {
	before := wordsBefore(text, start, 2)
	if len(before) < 2 {
		return false
	}
	for _, c := range monthPersonCues {
		if before[1] == c[0] && before[0] == c[1] {
			return true
		}
	}
	return before[0] == "by" && (strings.HasSuffix(before[1], "ed") || strings.HasSuffix(before[1], "en")) && before[1] != "scheduled" && before[1] != "planned" && before[1] != "expected" && before[1] != "needed"
}

func LowerNameCue(text string, start, end int, family bool) bool {
	if personDirectedBefore(text, start) || personVerbAfter(text, end) {
		return true
	}
	return !family && lowerNameContext(text, start, end)
}

func monthSep(c byte) bool { return c == ' ' || c == ',' || c == '-' || c == '/' }

func monthNeighbour(text string, start, end int) bool {
	j := start
	for j > 0 && j > start-monthSepWindow && monthSep(text[j-1]) {
		j--
	}
	if j < start {
		prev := rawWordBefore(text, j)
		if monthConnectors[prev] {
			j -= len(prev)
			for j > 0 && j > start-monthPairWindow && monthSep(text[j-1]) {
				j--
			}
			prev = rawWordBefore(text, j)
		}
		if IsMonthOrWeekday(prev) {
			return !PersonActionAfter(text, end)
		}
	}
	for _, dash := range monthDashes {
		if strings.HasPrefix(text[end:], dash) && IsMonthOrWeekday(rawWordAfter(text, end+len(dash))) {
			return true
		}
		if strings.HasSuffix(text[:start], dash) && IsMonthOrWeekday(rawWordBefore(text, start-len(dash))) {
			return true
		}
	}
	k := end
	for k < len(text) && k < end+monthSepWindow && monthSep(text[k]) {
		k++
	}
	next := rawWordAfter(text, k)
	if monthConnectors[next] {
		k += len(next)
		for k < len(text) && k < end+monthPairWindow && monthSep(text[k]) {
			k++
		}
		next = rawWordAfter(text, k)
	}
	if k > end && IsMonthOrWeekday(next) {
		return !PersonActionAfter(text, k+len(next))
	}
	return false
}

func BareTitlePair(text string, start, end int) bool {
	if start < 0 || end > len(text) || start >= end || strictPersonEvidence(text, start, end) {
		return false
	}
	ls, le := lineBounds(text, start, end)
	return titleLine(text[ls:le])
}

func IsHonorific(w string) bool { return honorificWords[strings.ToLower(strings.TrimSuffix(w, "."))] }

func messageHeader(text string, end int) bool {
	rest := text[end:]
	if !strings.HasPrefix(rest, messageHeaderCue) {
		return false
	}
	at := len(messageHeaderCue)
	name := rawWordAfter(rest, at)
	return textutil.StartsUpper(name) && strings.HasPrefix(rest[at+len(name):], ":")
}

func whQuestionBefore(text string, start int) bool {
	before := wordsBefore(text, start, 2)
	return len(before) == 2 && (before[0] == "is" || before[0] == "was" || before[0] == "does" || before[0] == "did") && whWords[before[1]]
}

func coordinatedWithI(text string, start, end int) bool {
	if strings.HasPrefix(text[end:], " and I ") || strings.HasPrefix(text[end:], " and I,") || strings.HasSuffix(text[:start], "I and ") {
		return true
	}
	return false
}

func nounSubjectIs(text string, start int) bool {
	before := wordsBefore(text, start, 2)
	if len(before) == 2 && before[0] == "it" && before[1] == "is" {
		return true
	}
	return len(before) == 2 && before[0] == "is" && !whWords[before[1]] && !willSubjects[before[1]]
}

func monthCopula(text string, end int) int {
	w := wordsAfter(text, end, monthCopulaWords)
	if len(w) < 2 || !monthLinkVerbs[w[0]] {
		return 0
	}
	if (w[0] == "will" || w[0] == "has" || w[0] == "had") && w[1] != "be" && w[1] != "been" {
		return 0
	}
	r := w[1:]
	for len(r) > 0 && (r[0] == "be" || r[0] == "been" || r[0] == "not") {
		r = r[1:]
	}
	if len(r) == 0 {
		return 0
	}
	switch {
	case willSubjects[r[0]] && r[0] != "the":
		return -1
	case monthAbsent[r[0]] || monthPlaces[r[0]]:
		return -1
	case thingPredicate(r):
		return 1
	case monthPrepositions[r[0]]:
		if len(r) > 1 && (monthSeasonNouns[r[1]] || (r[1] == "the" && len(r) > 2 && monthSeasonNouns[r[2]])) {
			return 1
		}
		return -1
	case monthPersonState(r):
		return -1
	}
	return 1
}

func willElided(text string, end int, n string) bool {
	if next := rawWordAfter(text, end); textutil.StartsUpper(next) {
		if willSubjects[n] || willCapsFunction[n] {
			return true
		}
		_, surname := namedata.LookupSurname(textutil.Fold(next))
		return !surname && BareTitlePair(text, end-len("will"), end+1+len(next))
	}
	if nonVerbAfterName[n] || (strings.HasSuffix(n, "ed") && !strings.HasSuffix(n, "eed")) || irregularPast[n] || PersonActionAfter(text, end) || personVerbs[n] {
		return false
	}
	if strings.HasSuffix(n, "s") && len(n) >= 3 && !notThirdPerson[n] && !willSubjects[n] {
		return false
	}
	return true
}

func coordinatedName(w string) bool {
	if !textutil.StartsUpper(w) || IsMonthOrWeekday(w) {
		return false
	}
	_, ok := namedata.LookupGiven(textutil.Fold(w))
	return ok
}

func coordinatedWithName(text string, start, end int) bool {
	for _, conj := range []string{" and ", " And ", " & "} {
		if strings.HasPrefix(text[end:], conj) && coordinatedName(rawWordAfter(text, end+len(conj))) {
			return true
		}
		if strings.HasSuffix(text[:start], conj) && coordinatedName(rawWordBefore(text, start-len(conj))) {
			return true
		}
	}
	return false
}

func thingPredicate(w []string) bool {
	return len(w) > 1 && ((w[0] == "out" && w[1] == "of") || (w[0] == "here" && w[1] == "already"))
}
