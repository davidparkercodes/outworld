package detect

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/namedata"
	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/entity"
)

func greetedGivenAt(text string, toks []nameTok, i int) (entity.Span, bool) {
	t := toks[i]
	if t.shape != shapeTitle || i+1 >= len(toks) || !lowerNameBefore[strings.ToLower(wordBefore(text, t.start))] {
		return entity.Span{}, false
	}
	if _, ok := givenOf(t.text); !ok || !oneSpace(text, t.end, toks[i+1].start) || !particleCompound(toks[i+1].text) {
		return entity.Span{}, false
	}
	return personSpan(text, t.start, t.end, "names:first", prioFirst), true
}

func greetingNameOK(text string, s, e int) bool {
	words := strings.Fields(text[s:e])
	switch len(words) {
	case 0:
		return true
	case 1:
		lw := strings.ToLower(words[0])
		return !greetingProduct(words[0]) && !greetingTargets[lw] && !stopCap[lw]
	}
	first, last := words[0], words[len(words)-1]
	if particleCompound(last) {
		return false
	}
	if ll := strings.ToLower(last); (ll == "team" || ll == "teams") && len(words) == 2 {
		if _, given := givenOf(first); initialismWord(first) || (!given && !knownSurname(first)) {
			return false
		}
	}
	roleLast := greetingRole(last) || (greetingGeneric(last) && !knownSurname(last) && !IsTechTerm(last))
	roleFirst := greetingRole(first) || (greetingGeneric(first) && !knownSurname(first) && !IsTechTerm(first))
	if roleFirst && roleLast {
		return false
	}
	productLast := greetingGeneric(last) || IsTechTerm(last) || utf8.RuneCountInString(last) < 2 || initialismWord(last)
	surname := knownSurname(last) && !greetingGeneric(last) && !initialismWord(last)
	return !((greetingProduct(first) || greetingBrands[strings.ToLower(first)] || techHead(first)) && productLast && !surname)
}

var assistantNames = wordSet(`claude chatgpt gpt gemini copilot siri alexa cortana bard codex llama mistral grok sonnet opus haiku`)

func IsAssistantName(w string) bool { return assistantNames[strings.ToLower(w)] }

func techHead(w string) bool {
	if !IsTechTerm(w) {
		return false
	}
	_, given := namedata.LookupGiven(w)
	return !given && !nameLikeTech[strings.ToLower(w)]
}

func greetingProduct(w string) bool {
	if IsAssistantName(w) {
		return true
	}
	if !IsTechTerm(w) {
		return false
	}
	_, given := namedata.LookupGiven(w)
	_, surname := namedata.LookupSurname(w)
	return !given && !surname && !nameLikeTech[strings.ToLower(w)]
}

func greetingRole(w string) bool {
	lw := strings.ToLower(w)
	for _, k := range []string{lw, strings.TrimSuffix(lw, "s")} {
		if greetingTargets[k] || greetingProducts[k] || greetingDepartments[k] || roleBefore[k] || listLabelWords[k] || capsLabelWords[k] {
			return true
		}
	}
	return false
}

func initialismWord(w string) bool {
	return utf8.RuneCountInString(w) == 2 && textutil.IsAllUpper(w)
}

var greetingTargets = wordSet(`all everyone everybody team teams folks there world guys friends colleagues manager customer customers support lead sir madam valued hiring dev staff people members crew gang`)

var greetingDepartments = wordSet(`sales press kitchen store property case night head worker chef`)

var greetingBrands = wordSet(`google microsoft apple amazon meta openai anthropic chatgpt`)

var greetingProducts = wordSet(`desktop studio agent shortcuts overflow stack visual assistant app bot plus`)

func greetingGeneric(w string) bool {
	lw := strings.ToLower(w)
	for _, k := range []string{lw, strings.TrimSuffix(lw, "s")} {
		if greetingTargets[k] || greetingProducts[k] || roleBefore[k] || listLabelWords[k] || capsLabelWords[k] || notSurname[k] || notSurnameExtra[k] || thingNouns[k] {
			return true
		}
	}
	return false
}

func TechNamesWithSurname(text string, family func(string) bool) []string {
	if !titleTechWordIn(text) {
		return nil
	}
	var out []string
	toks := wordRE.FindAllStringIndex(text, -1)
	for k, m := range toks {
		w := text[m[0]:m[1]]
		if !IsTechTerm(w) || shapeOf(w) != shapeTitle {
			continue
		}
		if m[1]+3 <= len(text) && text[m[1]] == ' ' && text[m[1]+1] >= 'A' && text[m[1]+1] <= 'Z' && text[m[1]+2] == '.' {
			out = append(out, w)
			continue
		}
		if k+1 >= len(toks) || toks[k+1][0] != m[1]+1 || text[m[1]] != ' ' {
			continue
		}
		nx := text[toks[k+1][0]:toks[k+1][1]]
		if shapeOf(nx) == shapeTitle && !IsTechTerm(nx) && surnameLike(nx) && (plainSurname(nx) || family(nx)) {
			out = append(out, w)
		}
	}
	return out
}

func titleTechWordIn(text string) bool {
	for i := 0; i < len(text); {
		r, n := utf8.DecodeRuneInString(text[i:])
		if !unicode.IsUpper(r) || i > 0 && unicode.IsLetter(textutil.RuneBefore(text, i)) {
			i += n
			continue
		}
		j := i + n
		for j < len(text) {
			r2, n2 := utf8.DecodeRuneInString(text[j:])
			if !unicode.IsLetter(r2) && !unicode.Is(unicode.Mn, r2) {
				break
			}
			j += n2
		}
		if IsTechTerm(text[i:j]) {
			return true
		}
		i = j
	}
	return false
}

func techGivenLoneAt(text string, toks []nameTok, i int) (entity.Span, bool) {
	t := toks[i]
	if t.shape != shapeTitle || !IsTechTerm(t.text) || IsAssistantName(t.text) || embedded(text, t.start, t.end) || identifierPart(text, t.start, t.end) {
		return entity.Span{}, false
	}
	if !techPersonHomonym(t.text) {
		return entity.Span{}, false
	}
	if i+1 < len(toks) && oneSpace(text, t.end, toks[i+1].start) && titleOrUpper(toks[i+1]) && !IsNameParticle(toks[i+1].text) {
		return entity.Span{}, false
	}
	prev := strings.ToLower(wordBefore(text, t.start))
	signed := t.start > 0 && text[t.start-1] == '\n' && signoffBefore(text, t.start-1) && clauseEndAfter(text, t.end)
	if !signed && !personDirectedBefore(text, t.start) && !PersonActionAfter(text, t.end) && !lowerNameBefore[prev] {
		return entity.Span{}, false
	}
	return personSpan(text, t.start, t.end, "names:first", prioFirst), true
}

var techCompoundNouns = wordSet(`gem gems crate crates package packages library libraries lang language compiler compilers runtime runtimes framework sdk script scripts test tests installer plugin plugins module modules version versions docs documentation api cli binary binaries interpreter syntax config configs configuration log logs deploy deploys deployment deployments compose topic topics container containers image images cluster clusters pipeline pipelines server servers service services daemon broker brokers queue queues pod pods node nodes instance instances throws threw exception exceptions error errors stack trace traces query queries schema schemas endpoint endpoints driver drivers kernel shell release releases repl app apps code build builds bindings extension extensions wrapper optimizer optimizers ci cd site sites client clients cache caches job jobs workflow workflows action actions chart charts operator operators binding linter linters formatter formatters parser parsers notebook notebooks kernels`)

var personContextNouns = wordSet(`meeting meetings call calls lunch dinner coffee visit visits account accounts contract contracts relationship onboarding feedback interview interviews appointment appointments session sessions party`)

var techVerbsBefore = wordSet(`install installing upgrade upgrading restart restarting reinstall reinstalling uninstall uninstalling configure configuring compile compiling deploy deploying`)

var techMoveVerbs = wordSet(`migrate migrating migrated switch switching switched port porting ported move moving moved`)

var caseStyles = wordSet(`pascal camel snake kebab`)

var relationBefore = wordSet(`daughter son wife husband mom mum dad mother father sister brother friend colleague coworker co-worker boss manager neighbour neighbor partner aunt uncle cousin girlfriend boyfriend fiancee fiance hire intern named niece nephew grandma grandpa grandmother grandfather kid child`)

var objectVerbsBefore = wordSet(`drove drive driving pay paid paying give gave giving tell told telling ask asked asking met meet meeting call called calling email emailed emailing thank thanked invite invited hired hire with cc forward send sent ping pinged text texted remind reminded saw see visit visited`)

var auxBefore = wordSet(`can could did does do is was will would has had should may might`)

var auxAfter = wordSet(`is was has had will would can could should did does isn't wasn't`)

func PersonShape(text string, start, end int) bool { return personShape(text, start, end) }

func hyphenNamePart(text string, start, end int) bool {
	if !nameShape(text[start:end]) {
		return false
	}
	if start > 1 && text[start-1] == '-' && nameShape(letterWordBefore(text, start-1)) {
		return true
	}
	return end+2 < len(text) && text[end] == '-' && nameShape(letterWordAfter(text, end+1))
}

func personShape(text string, start, end int) bool {
	if hyphenNamePart(text, start, end) {
		return true
	}
	rest := text[end:]
	if strings.HasPrefix(rest, "'s") || strings.HasPrefix(rest, "\u2019s") {
		return true
	}
	prev := strings.Trim(strings.ToLower(wordBefore(text, start)), ".")
	if honorificWords[prev] || relationBefore[prev] || objectVerbsBefore[prev] || personDirectedBefore(text, start) {
		return true
	}
	if auxBefore[prev] && sentenceStart(text, wordStartBefore(text, start)) {
		return true
	}
	if w := wordsBefore(text, start, 2); len(w) > 0 && w[0] == "to" && !(len(w) > 1 && techMoveVerbs[w[1]]) {
		return true
	}
	after := wordAfter(text, end)
	return auxAfter[after] || personVerbAfter(text, end) || lowerNameAfter[after] || personVerbs[after] || personWorkVerbs[after] || personMotionVerbs[after]
}

func wordStartBefore(text string, start int) int {
	j := start
	for j > 0 && start-j < wordScanLimit && (text[j-1] == ' ' || text[j-1] == '\t') {
		j--
	}
	for w := j; j > 0 && w-j < wordScanLimit && asciiLetter(text[j-1]); {
		j--
	}
	return j
}

func versionAfter(text string, end int) bool {
	k := end
	if k < len(text) && (text[k] == ' ' || text[k] == '-' || text[k] == '@') {
		k++
	}
	if k < len(text) && (text[k] == 'v' || text[k] == 'V') && k+1 < len(text) && text[k+1] >= '0' && text[k+1] <= '9' {
		return true
	}
	j := k
	for j < len(text) && text[j] >= '0' && text[j] <= '9' {
		j++
	}
	return j > k && j+1 < len(text) && text[j] == '.' && text[j+1] >= '0' && text[j+1] <= '9'
}

func identifierPart(text string, start, end int) bool {
	before, after := byte(0), byte(0)
	if start > 0 {
		before = text[start-1]
	}
	if end < len(text) {
		after = text[end]
	}
	prevAlnum := start > 1 && alnumRune(textutil.RuneBefore(text, start-1))
	nextAlnum := end+1 < len(text) && alnumRune(textutil.RuneAt(text, end+1))
	if prevAlnum && strings.IndexByte("_./:", before) >= 0 || nextAlnum && strings.IndexByte("_./:", after) >= 0 {
		return true
	}
	tok := text[start:end]
	if prevAlnum && before == '-' && !(nameShape(letterWordBefore(text, start-1)) && nameShape(tok)) {
		return true
	}
	return nextAlnum && after == '-' && !(nameShape(letterWordAfter(text, end+1)) && nameShape(tok))
}

func StopWord(w string) bool { return stopCap[strings.ToLower(w)] }

func strictIdentifier(text string, start, end int, known func(string) bool) bool {
	if strings.IndexFunc(text[start:end], unicode.IsUpper) >= 0 || vaultNameInToken(text, start, end, known) {
		return false
	}
	var before, after byte
	if start > 0 {
		before = text[start-1]
	}
	if end < len(text) {
		after = text[end]
	}
	prevAlnum := start > 1 && alnumRune(textutil.RuneBefore(text, start-1))
	nextAlnum := end+1 < len(text) && alnumRune(textutil.RuneAt(text, end+1))
	if (prevAlnum && (before == '-' || before == '_')) || (nextAlnum && (after == '-' || after == '_')) {
		return true
	}
	if !(prevAlnum && before == '.') && !(nextAlnum && after == '.') {
		return false
	}
	lo := start
	for lo > 0 && start-lo < wordScanLimit && (identByte(text[lo-1]) || text[lo-1] == '.') {
		lo--
	}
	hi := end
	for hi < len(text) && hi-end < wordScanLimit && (identByte(text[hi]) || text[hi] == '.') {
		hi++
	}
	tok := strings.Trim(text[lo:hi], ".")
	if strings.Count(tok, ".") >= 2 {
		return true
	}
	if !nextAlnum || after != '.' {
		return false
	}
	next := text[end+1 : hi]
	if k := strings.IndexByte(next, '.'); k >= 0 {
		next = next[:k]
	}
	return fileExtensions[next]
}

var fileExtensions = wordSet(`rb yml yaml json toml gemspec erb py js ts rs`)

func alnumRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

func identByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_'
}

func vaultNameInToken(text string, start, end int, known func(string) bool) bool {
	if known == nil {
		return false
	}
	lo := start
	for lo > 0 && start-lo < 2*wordScanLimit {
		r, n := utf8.DecodeLastRuneInString(text[:lo])
		if unicode.IsSpace(r) {
			break
		}
		lo -= n
	}
	hi := end
	for hi < len(text) && hi-end < 2*wordScanLimit {
		r, n := utf8.DecodeRuneInString(text[hi:])
		if unicode.IsSpace(r) {
			break
		}
		hi += n
	}
	name := text[start:end]
	for _, part := range strings.FieldsFunc(text[lo:hi], func(r rune) bool { return !unicode.IsLetter(r) }) {
		if len(part) < 2 || strings.EqualFold(part, name) {
			continue
		}
		if known(part) || known(titleCase(part)) {
			return true
		}
	}
	return false
}

func titleCase(w string) string {
	r, n := utf8.DecodeRuneInString(w)
	return string(unicode.ToUpper(r)) + strings.ToLower(w[n:])
}

func letterWordBefore(text string, i int) string {
	j := i
	for j > 0 && i-j < wordScanLimit {
		r, n := utf8.DecodeLastRuneInString(text[:j])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			break
		}
		j -= n
	}
	return text[j:i]
}

func letterWordAfter(text string, i int) string {
	j := i
	for j < len(text) && j-i < wordScanLimit {
		r, n := utf8.DecodeRuneInString(text[j:])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			break
		}
		j += n
	}
	return text[i:j]
}

func nameShape(w string) bool {
	rs := []rune(w)
	if len(rs) < 2 || !unicode.IsUpper(rs[0]) {
		return false
	}
	lower, upper := true, true
	for _, r := range rs[1:] {
		if !unicode.IsLetter(r) {
			return false
		}
		lower = lower && unicode.IsLower(r)
		upper = upper && unicode.IsUpper(r)
	}
	return lower || upper
}

func inFence(text string, start int) bool { return inRegion(regionsFor(text).fences, start) }

func InFence(text string, start int) bool { return inFence(text, start) }

func VaultNameKeep(text string, start, end int, known func(string) bool) bool {
	r := regionsFor(text)
	if a, b, ok := regionAt(r.spans, start); ok && possessiveBareSpan(text, a, b, start, end) {
		return false
	}
	for _, rs := range [][][2]int{r.spans, r.fences} {
		if a, b, ok := regionAt(rs, start); ok {
			return !otherPersonNear(text, max(a, start-contextWindow), min(b, end+contextWindow), text[start:end], known)
		}
	}
	return strictIdentifier(text, start, end, known)
}

func possessiveBareSpan(text string, a, b, start, end int) bool {
	if strings.Trim(text[a:b], "` ") != text[start:end] {
		return false
	}
	rest := strings.TrimLeft(text[b:], " \t")
	return strings.HasPrefix(rest, "'s") || strings.HasPrefix(rest, "\u2019s")
}

func otherPersonNear(text string, lo, hi int, name string, known func(string) bool) bool {
	for lo > 0 && !utf8.RuneStart(text[lo]) {
		lo++
	}
	for hi < len(text) && hi > lo && !utf8.RuneStart(text[hi]) {
		hi--
	}
	for _, w := range strings.FieldsFunc(text[lo:hi], func(r rune) bool { return !unicode.IsLetter(r) }) {
		if len(w) < 2 || strings.EqualFold(w, name) {
			continue
		}
		if known != nil && (known(w) || known(titleCase(w))) {
			return true
		}
		if _, given := namedata.LookupGiven(w); given && shapeOf(w) == shapeTitle && !IsTechTerm(w) {
			return true
		}
	}
	return false
}

func inCodeSpan(text string, start int) bool { return inRegion(regionsFor(text).spans, start) }

func CodeOrIdentifier(text string, start, end int) bool {
	return inCodeSpan(text, start) || identifierPart(text, start, end)
}

func TechContext(text string, start, end int) bool {
	if personShape(text, start, end) {
		return false
	}
	lw := strings.ToLower(text[start:end])
	after := wordsAfter(text, end, 2)
	switch {
	case inCodeSpan(text, start), identifierPart(text, start, end), versionAfter(text, end):
		return true
	case len(after) > 0 && caseStyles[lw] && after[0] == "case":
		return true
	case len(after) > 1 && after[0] == "on" && after[1] == "rails":
		return true
	case len(after) > 0 && techCompoundNouns[after[0]] && !(len(after) > 1 && personContextNouns[after[1]]):
		return true
	case len(after) > 1 && (after[0] == "or" || after[0] == "and") && IsTechTerm(after[1]):
		return true
	}
	before := wordsBefore(text, start, 2)
	if len(before) == 0 {
		return false
	}
	switch {
	case techVerbsBefore[before[0]]:
		return true
	case len(before) > 1 && (before[0] == "or" || before[0] == "and") && IsTechTerm(before[1]):
		return true
	case len(before) > 1 && (before[0] == "from" || before[0] == "to") && techMoveVerbs[before[1]]:
		return true
	case len(before) > 1 && before[0] == "in" && (before[1] == "written" || before[1] == "coded" || before[1] == "built" || before[1] == "implemented"):
		return true
	case len(before) > 1 && before[0] == "with" && before[1] == "built":
		return true
	case before[0] == "learn" || before[0] == "learning":
		return len(after) == 0 || !auxAfter[after[0]] && !personVerbs[after[0]]
	case before[0] == "use" || before[0] == "using":
		return len(after) > 0 && (after[0] == "for" || after[0] == "to" || after[0] == "with")
	case before[0] == "in":
		return len(after) == 0 || clauseEndAfter(text, end)
	}
	return false
}

func clauseEndAfter(text string, end int) bool {
	if end >= len(text) {
		return true
	}
	return strings.IndexByte(".,;:!?)\n", text[end]) >= 0
}

var nameLikeTech = wordSet(`mercury jupiter saturn neptune pluto`)

func techPersonHomonym(w string) bool {
	if !IsTechTerm(w) || IsAssistantName(w) {
		return false
	}
	if nameLikeTech[strings.ToLower(w)] {
		return true
	}
	_, given := namedata.LookupGiven(w)
	return given
}

func plainSurname(w string) bool {
	return knownSurname(w) && surnameLike(w) && !textutil.IsCommonWord(w) && !thingNouns[strings.ToLower(w)] && !greetingGeneric(w) && !greetingRole(w)
}

func dottedNameAt(text string, toks []nameTok, i int) (entity.Span, int, bool) {
	if i+1 >= len(toks) {
		return entity.Span{}, 0, false
	}
	t, nx := toks[i], toks[i+1]
	if t.end >= len(text) || text[t.end] != '.' || nx.start != t.end+1 || nx.shape != shapeTitle || (i > 0 && toks[i-1].end == t.start-1 && text[t.start-1] == '.') {
		return entity.Span{}, 0, false
	}
	initial := utf8.RuneCountInString(t.text) == 1 && t.shape == shapeTitle
	if _, given := namedata.LookupGiven(t.text); !initial && !(given && t.shape == shapeTitle) {
		return entity.Span{}, 0, false
	}
	if _, given := namedata.LookupGiven(nx.text); !given {
		return entity.Span{}, 0, false
	}
	if nx.end < len(text) && (text[nx.end] == '.' || text[nx.end] == '_' || text[nx.end] == '/') && nx.end+1 < len(text) && text[nx.end+1] != ' ' {
		return entity.Span{}, 0, false
	}
	if i+2 < len(toks) {
		f := toks[i+2]
		if f.shape == shapeTitle && oneSpace(text, nx.end, f.start) && plainSurname(f.text) {
			return personSpan(text, t.start, f.end, "names:full", prioFull), 2, true
		}
	}
	if PersonActionAfter(text, nx.end) || personDirectedBefore(text, t.start) || lowerNameBefore[strings.ToLower(wordBefore(text, t.start))] {
		return personSpan(text, t.start, nx.end, "names:first", prioFirst), 1, true
	}
	return entity.Span{}, 0, false
}
