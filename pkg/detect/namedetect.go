package detect

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/namedata"
	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/entity"
)

const (
	prioFirst       = 26
	prioFull        = 28
	prioFamilyFirst = 30
	maxDashGap      = 8
)

var hyphenClass = hyphenClassOf(textutil.HyphenVariants)

var nameWord = `\p{Lu}[\p{L}\p{M}]*(?:['’` + hyphenClass + `]\p{Lu}?[\p{Ll}\p{M}]+)*`

func hyphenClassOf(rs []rune) string {
	var b strings.Builder
	b.WriteString(`\-`)
	for _, r := range rs {
		b.WriteString(fmt.Sprintf(`\x{%04X}`, r))
	}
	return b.String()
}

var (
	wordRE = regexp.MustCompile(`\p{L}[\p{L}\p{M}]*(?:['’` + hyphenClass + `]\p{L}[\p{L}\p{M}]*)*`)

	labelRE = regexp.MustCompile(`(?im)^[ \t>*#\-]*(?:[A-Za-z][A-Za-z/\-]*[ \t]){0,3}?(on-?call|owner|lead|contact|attn|attention|from|to|cc|bcc|assignee|assigned to|reviewer|author|manager|patient|client|customer|name|signed|prepared by|approved by|requested by|reported by|submitted by|organizer|host|attendees?|driver|tenant|landlord|buyer|seller|applicant|employee|supervisor|physician|doctor|provider|beneficiary|insured|claimant|witness|recipient|sender)\*{0,2}[ \t]*:[ \t]*\*{0,2}`)

	emailNameRE = regexp.MustCompile(`((?:` + nameWord + `[ \t]){1,3})[<(\[][ \t]*(?:mailto:)?[A-Za-z0-9._%+\-]+@`)

	kinRE = regexp.MustCompile(`(?i:\b(?:my|his|her|our|their|your)\s+(?:(?:older|younger|little|big|late|step|twin|best|old|new|former|dear|ex)[\s\-]+)*(?:sister|brother|wife|husband|son|daughter|mother|father|mom|mum|dad|aunt|uncle|cousin|niece|nephew|grandmother|grandfather|grandma|grandpa|friend|girlfriend|boyfriend|partner|fianc[eé]e?|roommate|neighbou?r|colleague|co-?worker|boss|manager|assistant|doctor|therapist|lawyer|attorney|accountant|landlord|teacher|coach|mentor|buddy|kid|child|spouse|ex|stepmom|stepdad|(?:mother|father|sister|brother|son|daughter)-in-law))\s*,?\s+(` + nameWord + `(?: ` + nameWord + `)?)`)

	slugRE = regexp.MustCompile(`(?i)/(people|person|users?|u|profiles?|members?|staff|team|authors?|employees?|contacts?|in|by)/([a-z]+(?:[-._][a-z]+){1,3})(?:[/?#"'\s)>\]]|$)`)
)

var strongLabels = wordSet(`contact oncall on-call patient physician doctor driver tenant landlord applicant beneficiary insured claimant witness lead owner assignee reviewer author manager supervisor attn attention`)

var strongSlugPaths = wordSet(`people person profile profiles author authors staff in employees employee contacts contact`)

var stopCap = wordSet(`the a an and or but if of in on at to for with from by as is are was were be been it its this that these those i you he she we they my your his her our their not no yes do does did have has had will would can could should may might must shall who what when where why how which new old please thanks thank hello hi hey dear re fw fwd subject note todo fixme warning error info debug true false null none ok okay all any each every some here there today tomorrow yesterday now then also just very more most other such only own same so than too once etc mr mrs ms mx dr prof rev sir madam miss regards sincerely best cheers`)

var placeLike = wordSet(`monte paris london florence austin dallas houston sydney chelsea brooklyn israel india china kenya savannah charlotte phoenix madison jackson lincoln orlando milan vienna sierra dakota montana camden boston denver devon berlin ireland america asia trinity kingston santiago victoria adelaide alexandria odessa columbus cleveland raleigh eugene troy carmel geneva verona sofia holland britain egypt persia cheyenne indiana nevada georgia virginia carolina washington jordan chad`)

var techTerms = wordSet(`claude code anthropic openai chatgpt gpt gemini copilot cursor codex llama mistral siri alexa cortana watson java javascript typescript python ruby rust golang swift kotlin scala haskell julia ada pascal perl lua dart elixir erlang crystal basic cobol fortran lisp scheme racket clojure groovy bash zsh shell docker kubernetes helm jenkins travis github gitlab bitbucket jira confluence slack discord linux ubuntu debian fedora windows macos android node deno react vue angular svelte django flask rails laravel spring maven gradle npm yarn cargo redis postgres mysql sqlite mongo cassandra kafka spark hadoop airflow nginx apache tomcat jasmine mocha jest cypress selenium webpack vite babel eslint prettier adam bert elmo ernie apollo athena hermes atlas aurora lambda terraform ansible puppet chef vagrant jupyter pandas numpy homer mercury venus mars jupiter saturn neptune pluto hugo jekyll gatsby jade pug sass markdown unicode ascii json yaml toml sonnet opus haiku fable mythos grafana prometheus kibana elasticsearch datadog splunk sentry vercel netlify supabase figma notion zapier heroku firebase pytorch tensorflow snowflake databricks airtable trello asana`)

var notSurnameExtra = wordSet(`carlo river lake station airport bay island city county mountain language optimizer memorial bridge tower hall building library museum stadium arena theater theatre market mall plaza garden gardens beach harbor harbour port valley canyon forest desert ocean sea falls springs hotel inn cafe restaurant bar grill pizza kitchen studio studios media news times post journal daily weekly magazine press records music films pictures games api sdk cli app server client service services database cluster pipeline workflow dashboard report accounts account billing support sales legal team info admin office department dept desk help hr finance marketing engineering security operations ops meeting agenda summary update review notes plan budget invoice contract agreement policy act bill law code`)

var modalNames = wordSet(`will may`)

var dateAbbrev = wordSet(`jan feb mar apr jun jul aug sep sept oct nov dec mon tue tues wed thu thur thurs fri sat sun`)

var placeBefore = wordSet(`in near around outside across throughout via`)

var placeAfter = wordSet(`office offices region area branch campus site county city district province state location store plant facility warehouse hospital clinic airport station street road avenue metro suburbs downtown`)

var signoffRE = regexp.MustCompile(`(?im)(?:^|[.!?]\s+)(?:love|thanks|thank you|many thanks|best|best regards|kind regards|warm regards|regards|cheers|sincerely|yours|yours truly|talk soon|xo|xoxo|take care),?[ \t]*\n?[ \t]*(` + nameWord + `(?: ` + nameWord + `)?)[ \t]*$`)

var dashSignRE = regexp.MustCompile(`(?m)^[ \t]*(--|-|\x{2013}|\x{2014})[ \t]+(` + nameWord + `(?: ` + nameWord + `)?)[ \t]*$`)

var commentPossessiveRE = regexp.MustCompile(`(?m)(?:^|[ \t])(?://+|#+|/\*+|\*)[ \t]*(\p{Lu}\p{Ll}+)['’]s\b`)

func commentPossessiveName(w string) bool {
	if _, ok := givenOf(w); !ok || textutil.IsCommonWord(w) || IsMonthOrWeekday(w) || !surnameLike(w) {
		return false
	}
	_, state := StateName(w)
	return !state
}

func lowerSignoffName(w string) bool {
	if _, ok := givenOf(w); ok {
		return true
	}
	return strongSurname(w)
}

func dashSignName(dash, name string) bool {
	words := strings.Fields(name)
	g, ok := givenOf(words[0])
	if !ok || textutil.IsCommonWord(words[0]) {
		return false
	}
	if _, state := StateName(words[0]); state || (g.Weak && dash == "-") {
		return false
	}
	for _, w := range words {
		if !surnameLike(w) || IsTechTerm(w) || textutil.IsAllUpper(w) {
			return false
		}
	}
	return true
}

var determiners = wordSet(`the a an this that these those my your his her our their its no every each any some of`)

var chatRE = regexp.MustCompile(`(?m)^(\[)?(?:\d{1,4}[/.\-]\d{1,2}[/.\-]\d{1,4},?[ T])?\d{1,2}:\d{2}(?::\d{2})?(?:[ ]?[APap][Mm])?\]?[ \t]*(?:-[ \t]*)?@?([\p{L}][\p{L}\p{N}._'’\- ]{1,40}?):[ \t]`)

var notSpeaker = wordSet(`error warn warning info debug trace fatal notice note system bot user assistant admin server client me you all everyone channel here todo fixme re fwd subject http https`)

func IsTechTerm(w string) bool { return techTerms[strings.ToLower(w)] }

func KnownName(w string) bool {
	if _, ok := namedata.LookupGiven(w); ok {
		return true
	}
	_, ok := namedata.LookupSurname(w)
	return ok
}

type NameDetector struct{}

func (NameDetector) Name() string { return "names" }

type nameTok struct {
	start, end int
	text       string
	shape      int
}

const (
	shapeOther = iota
	shapeTitle
	shapeUpper
	shapeLower
)

func shapeOf(w string) int {
	switch {
	case textutil.IsAllLower(w):
		return shapeLower
	case textutil.IsAllUpper(w) && utf8.RuneCountInString(w) > 1:
		return shapeUpper
	case textutil.StartsUpper(w):
		return shapeTitle
	}
	return shapeOther
}

func embedded(text string, s, e int) bool {
	if s > 0 {
		r, _ := utf8.DecodeLastRuneInString(text[:s])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune(`@./_\#$%&=+~`, r) {
			return true
		}
	}
	if e < len(text) {
		r, n := utf8.DecodeRuneInString(text[e:])
		if unicode.IsDigit(r) || strings.ContainsRune(`@(_\/=+~`, r) {
			return true
		}
		if r == '.' && e+n < len(text) {
			nr, _ := utf8.DecodeRuneInString(text[e+n:])
			if unicode.IsLetter(nr) || unicode.IsDigit(nr) {
				return true
			}
		}
	}
	return false
}

func givenOf(w string) (namedata.Given, bool) {
	if IsTechTerm(w) {
		return namedata.Given{}, false
	}
	g, ok := namedata.LookupGiven(w)
	lw := strings.ToLower(w)
	if !ok && strings.ContainsFunc(w, nonASCIIJoiner) {
		w = asciiHyphens(w)
		lw = strings.ToLower(w)
		g, ok = namedata.LookupGiven(w)
	}
	if !ok && strings.ContainsFunc(w, textutil.IsNameJoiner) {
		return hyphenGiven(w)
	}
	if !ok || dateAbbrev[lw] {
		return g, false
	}
	if placeLike[lw] || textutil.IsCommonWord(w) || IsMonthOrWeekday(w) {
		g.Weak = true
	}
	if _, isState := StateName(w); isState {
		g.Weak = true
	}
	return g, true
}

func surnameLike(w string) bool {
	lw := strings.ToLower(w)
	if utf8.RuneCountInString(w) < 2 || stopCap[lw] || IsMonthOrWeekday(w) || notSurname[lw] || notSurnameExtra[lw] || techTerms[lw] {
		return false
	}
	return !particleCompound(w)
}

var englishParticles = wordSet(`in on up off out to by over down through around away back`)

func particleCompound(w string) bool {
	parts := hyphenParts(w)
	if len(parts) != 2 || shapeOf(parts[1]) != shapeLower {
		return false
	}
	a, b := strings.ToLower(parts[0]), parts[1]
	return englishParticles[b] || a == b || stopCap[a] || joinedWordBlocked(a+b)
}

func knownSurname(w string) bool {
	if parts := hyphenParts(w); len(parts) > 1 {
		for _, p := range parts {
			if _, ok := namedata.LookupSurname(p); ok {
				return true
			}
		}
		return false
	}
	_, ok := namedata.LookupSurname(w)
	return ok
}

func techSurname(w string) bool {
	lw := strings.ToLower(w)
	return IsTechTerm(w) && knownSurname(w) && !stopCap[lw] && utf8.RuneCountInString(w) >= 3
}

func hyphenParts(w string) []string { return textutil.NameJoinerFields(w) }

func nonASCIIJoiner(r rune) bool { return r != '-' && textutil.IsNameJoiner(r) }

func asciiHyphens(w string) string {
	return strings.Map(func(r rune) rune {
		if nonASCIIJoiner(r) {
			return '-'
		}
		return r
	}, w)
}

func startsHyphen(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return textutil.IsHyphen(r)
}

func hyphenGiven(w string) (namedata.Given, bool) {
	parts := hyphenParts(w)
	if len(parts) < 2 || len(parts) > 3 {
		return namedata.Given{}, false
	}
	tech, lead := false, -1
	for i, p := range parts {
		lp := strings.ToLower(p)
		if shapeOf(p) != shapeTitle || utf8.RuneCountInString(p) < 2 || stopCap[lp] || IsMonthOrWeekday(p) {
			return namedata.Given{}, false
		}
		if techTerms[lp] {
			tech = true
		} else if _, ok := givenOf(p); ok && lead < 0 {
			lead = i
		}
	}
	if lead < 0 || (!tech && lead != 0) || placeHead(parts) {
		return namedata.Given{}, false
	}
	g, _ := givenOf(parts[lead])
	for i, p := range parts {
		if i != lead && (techTerms[strings.ToLower(p)] || !KnownName(p)) {
			g.Weak = true
		}
	}
	return g, true
}

func placeHead(parts []string) bool {
	_, state := StateName(parts[0])
	if !placeLike[strings.ToLower(parts[0])] && !state {
		return false
	}
	for _, p := range parts[1:] {
		if _, ok := namedata.LookupGiven(p); !ok {
			return true
		}
	}
	return false
}

func hyphenatedName(w string) bool { return strings.ContainsFunc(w, textutil.IsNameJoiner) }

func oneSpace(text string, a, b int) bool { return b == a+1 && text[a] == ' ' }

func continuesClause(text string, e int) bool {
	if e+1 >= len(text) || text[e] != ' ' {
		return false
	}
	r, _ := utf8.DecodeRuneInString(text[e+1:])
	return unicode.IsLower(r)
}

func clauseStart(text string, s int) bool {
	j := s
	for j > 0 && s-j < wordScanLimit {
		r, n := utf8.DecodeLastRuneInString(text[:j])
		if r == ' ' || r == '\t' {
			j -= n
			continue
		}
		return r == '\n' || strings.ContainsRune(".:;!?,-–\u2014>*|", r)
	}
	return true
}

func (NameDetector) Detect(_ context.Context, text string) ([]entity.Span, error) {
	var out []entity.Span
	emit := func(s, e int, src string, prio int) {
		if s < 0 || e <= s || e > len(text) {
			return
		}
		out = append(out, entity.Span{Start: s, End: e, Type: entity.Person, Label: "person", Text: text[s:e], Source: "names:" + src, Priority: prio, Score: 1})
	}
	comps := companyMatches(text)
	toks := nameTokens(text)
	var listed []entity.Span
	lists := nameLists(text, comps)
	out = append(out, listSpans(text, lists)...)
	out = append(out, bulletNameSpans(text, comps)...)
	honorific, claimed := honorificNames(text, toks)
	out = append(out, honorific...)
	claimed = claimListed(toks, lists, claimed)
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		if sp, n, ok := koreanNameAt(text, toks, i, claimed); ok {
			out = append(out, sp)
			i += n
			continue
		}
		if claimed[i] || embedded(text, t.start, t.end) || (t.shape != shapeLower && thingAt(text, toks, i) && !thingPersonPair(text, toks, i)) {
			continue
		}
		if end, n, canonical := capsSurnameFirst(text, toks, i); n > 0 {
			out = append(out, entity.Span{Start: t.start, End: end, Type: entity.Person, Label: "person", Text: text[t.start:end], Source: "names:surname-first", Priority: 29, Score: 1, Canonical: canonical})
			i += n
			continue
		}
		if end, n, canonical, pending := familyFirst(text, toks, i); n > 0 {
			ff := entity.Span{Start: t.start, End: end, Type: entity.Person, Label: "person", Text: text[t.start:end], Source: "names:family-first", Priority: prioFamilyFirst, Score: 1, Canonical: canonical}
			if !pending {
				out = append(out, ff)
				i += n
				continue
			}
			listed = append(listed, ff)
		}
		g, ok := givenOf(t.text)
		if !ok {
			if end, n := techGivenFull(text, toks, i); n > 0 {
				emit(t.start, end, "full", prioFull)
				i += n
			} else if end, n := surnameLed(text, toks, i); n > 0 && !(text[t.end] == ',' && inCommaList(lists, t.start)) {
				emit(t.start, end, "full", 25)
				i += n
			} else if t.shape == shapeTitle && strongSurname(t.text) && lowerNameContext(text, t.start, t.end) && !placeUse(text, t.start, t.end) {
				emit(t.start, t.end, "last", 22)
			}
			continue
		}
		switch t.shape {
		case shapeTitle, shapeUpper:
			if (g.Weak && cuePair(text, toks, i)) || thingPersonPair(text, toks, i) {
				emit(t.start, toks[i+1].end, "full", prioFull)
				i++
				continue
			}
			if t.shape == shapeTitle && IsMonthOrWeekday(t.text) && monthPerson(text, toks, i) {
				emit(t.start, t.end, "first", 24)
				continue
			}
			end, n := t.end, 0
			cue := g.Weak && t.shape == shapeTitle && pairCueBefore(text, t.start)
			if pe, pn := particleSurname(text, toks, i); pn > 0 && (!g.Weak || nameUseOK(text, t.start, t.end)) {
				emit(t.start, pe, "full", prioFull)
				i += pn
				continue
			}
			if i+1 < len(toks) && !cue {
				nx := toks[i+1]
				if nx.shape == t.shape && oneSpace(text, t.end, nx.start) && !embedded(text, nx.start, nx.end) && (surnameLike(nx.text) || thingSurname(nx.text) || (techSurname(nx.text) && (!g.Weak || PersonActionAfter(text, nx.end)))) && !dashParticleAfter(text, nx.text, nx.end) {
					known := knownSurname(nx.text)
					accept := false
					switch {
					case !g.Weak:
						accept = true
					case colourWords[strings.ToLower(nx.text)] || cueCompoundWords[strings.ToLower(nx.text)]:
						accept = compoundPersonEvidence(text, t.start, nx.end)
					case !nameUseOK(text, t.start, t.end):
					case known:
						accept = true
					case utf8.RuneCountInString(nx.text) >= 4 && !textutil.IsCommonWord(nx.text):
						accept = (!(modalNames[strings.ToLower(t.text)] && sentenceStart(text, t.start) && continuesClause(text, nx.end)) || PersonActionAfter(text, nx.end)) && (!headingBefore(text, toks, i) || (personDirectedBefore(text, t.start) && !indefinitePronouns[strings.ToLower(nx.text)]))
					}
					if accept {
						end, n = nx.end, 1
						if i+2 < len(toks) {
							th := toks[i+2]
							_, midGiven := namedata.LookupGiven(nx.text)
							if th.shape == t.shape && oneSpace(text, nx.end, th.start) && surnameLike(th.text) && knownSurname(th.text) && (midGiven || known) && !thingAt(text, toks, i+2) {
								end, n = th.end, 2
							}
						}
					}
				}
			}
			if n > 0 {
				if pe, pn := particleSurname(text, toks, i+n); pn > 0 && !thingAt(text, toks, i+n+1) {
					end, n = pe, n+pn
				}
				switch trailingThing(text, toks, i, i+n) {
				case trailSkip:
					i += n + 1
					continue
				case trailExtend:
					end, n = toks[i+n+1].end, n+1
				}
				emit(t.start, end, "full", prioFull)
				i += n
				continue
			}
			if t.shape == shapeUpper || utf8.RuneCountInString(t.text) < 3 || (g.Weak && wrappedToken(text, t.start, t.end)) {
				continue
			}
			if !g.Weak {
				if nameUseOK(text, t.start, t.end) && !placeUse(text, t.start, t.end) {
					emit(t.start, t.end+trailingInitial(text, t.end), "first", prioFirst)
				}
				continue
			}
			if !nameUseOK(text, t.start, t.end) {
				continue
			}
			if hyphenatedName(t.text) && (personDirectedBefore(text, t.start) || PersonActionAfter(text, t.end) || lowerNameBefore[strings.ToLower(wordBefore(text, t.start))]) {
				emit(t.start, t.end, "first", 24)
				continue
			}
			if lowerNameContext(text, t.start, t.end) && weakContextOK(text, t.start, t.end) {
				emit(t.start, t.end, "first", 24)
				continue
			}
			if dashBefore(text, t.start) && (personVerbAfter(text, t.end) || dashPersonWords[wordAfter(text, t.end)]) {
				emit(t.start, t.end, "first", 24)
				continue
			}
			lw := strings.ToLower(t.text)
			_, isState := StateName(t.text)
			if placeLike[lw] || isState || IsMonthOrWeekday(t.text) || sentenceStart(text, t.start) || placeUse(text, t.start, t.end) {
				continue
			}
			if prev := wordBefore(text, t.start); prev == "" || !textutil.IsAllLower(prev) || determiners[prev] || dashCommentBefore(text, t.start, t.end) {
				continue
			}
			if i+1 < len(toks) && oneSpace(text, t.end, toks[i+1].start) && toks[i+1].shape != shapeLower {
				continue
			}
			emit(t.start, t.end, "first", 23)
		case shapeLower:
			if personVerbs[t.text] || lowerNameAfter[t.text] {
				continue
			}
			if i+1 < len(toks) && !g.Weak && utf8.RuneCountInString(t.text) >= 3 {
				nx := toks[i+1]
				if nx.shape == shapeLower && oneSpace(text, t.end, nx.start) && !embedded(text, nx.start, nx.end) && strongSurname(nx.text) {
					if _, alsoGiven := namedata.LookupGiven(nx.text); !alsoGiven || lowerNameContext(text, t.start, nx.end) {
						emit(t.start, nx.end, "lower", 24)
						i++
						continue
					}
				}
			}
			if utf8.RuneCountInString(t.text) < 4 && !g.Weak {
				continue
			}
			if !g.Weak {
				if lowerNameContext(text, t.start, t.end) {
					emit(t.start, t.end, "lower", 24)
				}
				continue
			}
			if weakContextOK(text, t.start, t.end) && lowerNameContext(text, t.start, t.end) {
				emit(t.start, t.end, "lower", 22)
			}
		}
	}
	out = append(out, labelNames(text, comps)...)
	if len(listed) > 0 {
		starts, ends := map[int]bool{}, map[int]bool{}
		for _, p := range out {
			if p.Type == entity.Person && strings.Contains(p.Text, " ") {
				starts[p.Start], ends[p.End] = true, true
			}
		}
		for _, ff := range listed {
			given := ff.Text[strings.IndexByte(ff.Text, ' ')+1:]
			if !strings.Contains(given, " ") && familyListed(text, ff.Start, ff.End, given, starts, ends) {
				out = append(out, ff)
				starts[ff.Start], ends[ff.End] = true, true
			}
		}
	}
	out = append(out, coordinatedPairs(text, toks, out)...)
	for _, m := range initialSurnameRE.FindAllStringSubmatchIndex(text, -1) {
		if strongSurname(text[m[4]:m[5]]) {
			emit(m[2], m[3], "initial", 25)
		}
	}
	for _, m := range surnameInitialRE.FindAllStringSubmatchIndex(text, -1) {
		if strongSurname(text[m[2]:m[3]]) && !embedded(text, m[0], m[0]+1) {
			emit(m[0], m[1], "initial", 25)
		}
	}
	for _, m := range emailNameRE.FindAllStringSubmatchIndex(text, -1) {
		s, e := m[2], m[3]
		for e > s && (text[e-1] == ' ' || text[e-1] == '\t') {
			e--
		}
		words := wordRE.FindAllStringIndex(text[s:e], -1)
		if len(words) > 2 {
			if _, ok := givenOf(text[s+words[len(words)-3][0] : s+words[len(words)-3][1]]); !ok || len(words) > 3 {
				words = words[len(words)-2:]
			}
		}
		for len(words) > 0 && !surnameLike(text[s+words[0][0]:s+words[0][1]]) {
			words = words[1:]
		}
		if len(words) == 0 {
			continue
		}
		valid := true
		for _, w := range words {
			ww := text[s+w[0] : s+w[1]]
			if shapeOf(ww) != shapeTitle || !surnameLike(ww) {
				valid = false
			}
		}
		_, firstGiven := givenOf(text[s+words[0][0] : s+words[0][1]])
		if !valid || !(firstGiven || len(words) > 1) {
			continue
		}
		emit(s+words[0][0], e, "email", 27)
	}
	for _, m := range signoffRE.FindAllStringSubmatchIndex(text, -1) {
		s, e := m[2], m[3]
		ok := true
		for _, w := range strings.Fields(text[s:e]) {
			if !surnameLike(w) || IsTechTerm(w) || textutil.IsAllUpper(w) || (textutil.IsAllLower(w) && !lowerSignoffName(w)) {
				ok = false
			}
		}
		if ok {
			emit(s, e, "signoff", 27)
		}
	}
	for _, m := range commentPossessiveRE.FindAllStringSubmatchIndex(text, -1) {
		if commentPossessiveName(text[m[2]:m[3]]) {
			emit(m[2], m[3], "first", prioFirst)
		}
	}
	for _, m := range dashSignRE.FindAllStringSubmatchIndex(text, -1) {
		name := text[m[4]:m[5]]
		if dashSignName(text[m[2]:m[3]], name) || (m[0] > 0 && signoffBefore(text, m[0]-1) && looseSignName(name, true)) {
			emit(m[4], m[5], "signoff", 27)
		}
	}
	for _, m := range midDashSignRE.FindAllStringSubmatchIndex(text, -1) {
		if !codeDash(text, m[2], m[5]) && looseSignName(text[m[4]:m[5]], signoffBefore(text, m[2])) {
			emit(m[4], m[5], "signoff", 27)
		}
	}
	for _, m := range chatRE.FindAllStringSubmatchIndex(text, -1) {
		s, e := m[4], m[5]
		name := text[s:e]
		words := strings.Fields(name)
		if len(words) == 0 || len(words) > 3 {
			continue
		}
		ok := true
		known := false
		for _, w := range words {
			lw := strings.ToLower(strings.Trim(w, "._-"))
			if notSpeaker[lw] || stopCap[lw] || techTerms[lw] || utf8.RuneCountInString(lw) < 2 {
				ok = false
			}
			if _, g := namedata.LookupGiven(lw); g {
				known = true
			}
		}
		if !ok {
			continue
		}
		if len(words) == 1 && textutil.IsAllLower(name) && !known && m[2] < 0 {
			continue
		}
		emit(s, e, "chat", 27)
	}
	for _, m := range kinRE.FindAllStringSubmatchIndex(text, -1) {
		s, e := m[2], m[3]
		words := wordRE.FindAllStringIndex(text[s:e], -1)
		if len(words) == 0 || !surnameLike(text[s+words[0][0]:s+words[0][1]]) || IsTechTerm(text[s+words[0][0]:s+words[0][1]]) {
			continue
		}
		if len(words) == 2 && !surnameLike(text[s+words[1][0]:s+words[1][1]]) {
			e = s + words[0][1]
		}
		emit(s, e, "kin", 27)
	}
	for _, m := range slugRE.FindAllStringSubmatchIndex(text, -1) {
		path := strings.ToLower(text[m[2]:m[3]])
		s, e := m[4], m[5]
		parts := strings.FieldsFunc(text[s:e], func(r rune) bool { return r == '-' || r == '.' || r == '_' })
		if len(parts) < 2 {
			continue
		}
		_, first := givenOf(parts[0])
		_, last := namedata.LookupSurname(parts[len(parts)-1])
		ok := first || last
		if !ok && strongSlugPaths[path] {
			ok = true
			for _, p := range parts {
				if len(p) < 3 || techTerms[p] || notSurname[p] || notSurnameExtra[p] || stopCap[p] {
					ok = false
				}
			}
		}
		if ok {
			emit(s, e, "slug", 27)
		}
	}
	out = append(out, toHeaderSpans(text)...)
	out = append(out, nicknameSpans(text, out)...)
	out = append(out, handleSpans(text)...)
	out = append(out, particleTails(text, out)...)
	out = nickSurnames(text, out)
	out = guardOrgs(text, out, comps)
	return append(out, initialismSpans(text, initialismPeople(out), func() []nameList { return lists }, "names:initials", 27)...), nil
}

func claimListed(toks []nameTok, lists []nameList, claimed map[int]bool) map[int]bool {
	for _, nl := range lists {
		for _, el := range nl.elems {
			if !el.emit {
				continue
			}
			for i := sort.Search(len(toks), func(k int) bool { return toks[k].start >= el.start }); i < len(toks) && toks[i].start < el.end; i++ {
				if claimed == nil {
					claimed = map[int]bool{}
				}
				claimed[i] = true
			}
		}
	}
	return claimed
}

func thingSurname(w string) bool {
	return thingNouns[strings.ToLower(w)] && knownSurname(w)
}

const (
	trailNone = iota
	trailSkip
	trailExtend
)

func trailingThing(text string, toks []nameTok, first, last int) int {
	if last+1 >= len(toks) {
		return trailNone
	}
	c := toks[last+1]
	if c.shape != shapeTitle || !oneSpace(text, toks[last].end, c.start) {
		return trailNone
	}
	commemorative := commemorativeNouns[strings.ToLower(c.text)]
	surname := thingSurname(c.text)
	if !commemorative && !surname {
		return trailNone
	}
	s := toks[first].start
	prev := ""
	if w := wordsBefore(text, s, 1); len(w) > 0 {
		prev = w[0]
	}
	evidence := false
	if !articles[prev] {
		evidence = labelBefore(text, s) || directedBefore(text, s) || prev == "with" || roleBefore[prev] || lowerNameBefore[prev] || honorificWords[strings.Trim(prev, ".")]
		if !evidence && surname && !determiners[prev] && prev != "at" && !capitalAfter(text, toks, last+1) {
			evidence = strictPersonEvidence(text, s, c.end)
		}
	}
	switch {
	case evidence && surname && !capitalAfter(text, toks, last+1):
		return trailExtend
	case evidence || !commemorative:
		return trailNone
	}
	return trailSkip
}

func capitalAfter(text string, toks []nameTok, i int) bool {
	return i+1 < len(toks) && oneSpace(text, toks[i].end, toks[i+1].start) && titleOrUpper(toks[i+1])
}

func techGivenFull(text string, toks []nameTok, i int) (int, int) {
	t := toks[i]
	if t.shape != shapeTitle || i+1 >= len(toks) || !IsTechTerm(t.text) {
		return 0, 0
	}
	if _, ok := namedata.LookupGiven(t.text); !ok && !nameLikeTech[strings.ToLower(t.text)] && !IsAssistantName(t.text) {
		return 0, 0
	}
	if pe, pn := particleSurname(text, toks, i); pn > 0 && (PersonActionAfter(text, pe) || personDirectedBefore(text, t.start)) {
		return pe, pn
	}
	nx := toks[i+1]
	thing := !surnameLike(nx.text) && thingSurname(nx.text)
	if nx.shape != shapeTitle || !oneSpace(text, t.end, nx.start) || embedded(text, nx.start, nx.end) || !(surnameLike(nx.text) || thing) || !knownSurname(nx.text) {
		return 0, 0
	}
	evidence := personVerbAfter(text, nx.end) || lowerNameAfter[wordAfter(text, nx.end)] || personDirectedBefore(text, t.start)
	after := wordAfter(text, nx.end)
	explicit := personVerbs[after] || personWorkVerbs[after] || personMotionVerbs[after] || lowerNameAfter[after] || personDirectedBefore(text, t.start) || lowerNameBefore[strings.ToLower(wordBefore(text, t.start))]
	if thing {
		if !evidence && !explicit {
			return 0, 0
		}
	} else if !strongSurname(nx.text) && !(evidence && utf8.RuneCountInString(nx.text) >= 4 && !textutil.IsCommonWord(nx.text)) && !(explicit && surnameLike(nx.text)) && !plainSurname(nx.text) {
		return 0, 0
	}
	if _, isState := StateName(nx.text); isState {
		return 0, 0
	}
	return nx.end, 1
}

var abbrevParticles = wordSet(`st ste`)

func hyphenParticles(w string) bool {
	parts := hyphenParts(w)
	if len(parts) < 2 {
		return false
	}
	for _, p := range parts {
		if !nameParticles[strings.ToLower(p)] {
			return false
		}
	}
	return true
}

func particleGap(text string, prev nameTok, next int) bool {
	if oneSpace(text, prev.end, next) {
		return true
	}
	return abbrevParticles[strings.ToLower(prev.text)] && text[prev.end:next] == ". "
}

func particleSurname(text string, toks []nameTok, i int) (int, int) {
	shape := toks[i].shape
	if shape != shapeTitle && shape != shapeUpper {
		return 0, 0
	}
	j := i + 1
	for j < len(toks) && j-i <= 3 && (nameParticles[strings.ToLower(toks[j].text)] || abbrevParticles[strings.ToLower(toks[j].text)] || hyphenParticles(toks[j].text)) && particleGap(text, toks[j-1], toks[j].start) {
		j++
	}
	if j == i+1 || j >= len(toks) {
		return 0, 0
	}
	last := toks[j]
	if last.shape != shape || !particleGap(text, toks[j-1], last.start) || !surnameLike(last.text) || textutil.IsCommonWord(last.text) || embedded(text, last.start, last.end) {
		return 0, 0
	}
	return last.end, j - i
}

var initialSurnameRE = regexp.MustCompile(`(?:^|[^\p{L}\p{N}.])((?:\p{Lu}\.[ ]?){1,2}(` + nameWord + `))`)

var surnameInitialRE = regexp.MustCompile(`(` + nameWord + `), \p{Lu}\.(?:[ ]?\p{Lu}\.)?`)

func trailingInitial(text string, e int) int {
	if e+3 > len(text) || text[e] != ' ' || text[e+1] < 'A' || text[e+1] > 'Z' || text[e+2] != '.' {
		return 0
	}
	if e+3 < len(text) {
		r, _ := utf8.DecodeRuneInString(text[e+3:])
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return 0
		}
	}
	return 3
}

func strongSurname(w string) bool {
	if parts := hyphenParts(w); len(parts) > 1 {
		for _, p := range parts {
			if strongSurname(p) {
				return true
			}
		}
		return false
	}
	lw := strings.ToLower(w)
	weak, ok := namedata.LookupSurname(w)
	return ok && !weak && !placeLike[lw] && !textutil.IsCommonWord(w) && !techTerms[lw] && utf8.RuneCountInString(w) >= 4
}

func surnameLed(text string, toks []nameTok, i int) (int, int) {
	t := toks[i]
	if i+1 >= len(toks) || (t.shape != shapeTitle && t.shape != shapeUpper) || utf8.RuneCountInString(t.text) < 4 {
		return 0, 0
	}
	nx := toks[i+1]
	if nx.shape != t.shape || embedded(text, nx.start, nx.end) {
		return 0, 0
	}
	gap := text[t.end:nx.start]
	if gap == ", " && i > 0 {
		p := toks[i-1]
		plw := strings.ToLower(p.text)
		if p.shape == t.shape && oneSpace(text, p.end, t.start) && !stopCap[plw] && !listLabelWords[plw] && !surnameLeadOK[plw] {
			return 0, 0
		}
	}
	if strongSurname(t.text) {
		if g, ok := givenOf(nx.text); ok && !g.Weak && (gap == " " || gap == ", " || (gap == "/" && t.shape == shapeUpper)) {
			if gap == " " && (placeUse(text, t.start, nx.end) || strongSurname(nx.text)) {
				return 0, 0
			}
			return nx.end, 1
		}
		return 0, 0
	}
	if gap != " " || !strongSurname(nx.text) || IsTechTerm(t.text) {
		return 0, 0
	}
	lw := strings.ToLower(t.text)
	if _, sur := namedata.LookupSurname(t.text); sur || !surnameLike(t.text) || textutil.IsCommonWord(t.text) || placeLike[lw] || sentenceStart(text, t.start) || headingBefore(text, toks, i) || placeUse(text, t.start, nx.end) {
		return 0, 0
	}
	if _, isState := StateName(t.text); isState {
		return 0, 0
	}
	return nx.end, 1
}

func placeUse(text string, s, e int) bool {
	return placeBefore[strings.ToLower(wordBefore(text, s))] || placeAfter[wordAfter(text, e)] || regionShapeAfter(text, s, e, strings.HasPrefix(text[e:], ","))
}

var pairCues = wordSet(`from with by`)

func cuePair(text string, toks []nameTok, i int) bool {
	t := toks[i]
	if t.shape != shapeTitle || i+1 >= len(toks) {
		return false
	}
	nx := toks[i+1]
	if nx.shape != shapeTitle || !oneSpace(text, t.end, nx.start) || embedded(text, nx.start, nx.end) || utf8.RuneCountInString(nx.text) < 3 {
		return false
	}
	if !pairCueBefore(text, t.start) {
		return false
	}
	lw := strings.ToLower(nx.text)
	if stopCap[lw] || techTerms[lw] || notSurname[lw] || placeAfter[lw] || IsMonthOrWeekday(nx.text) || capitalAfter(text, toks, i+1) {
		return false
	}
	_, state := StateName(nx.text)
	compound := (colourWords[lw] || cueCompoundWords[lw]) && !meetingContext(text, t.start, nx.end)
	thing := thingNouns[lw] || notSurnameExtra[lw] || placeLike[lw] || state || compound || thingAt(text, toks, i+1)
	loose := !IsMonthOrWeekday(t.text) && !stopCap[strings.ToLower(t.text)] && utf8.RuneCountInString(nx.text) >= 4 && !textutil.IsCommonWord(nx.text)
	if !thing && (knownSurname(nx.text) || loose) {
		return true
	}
	return !state && !placeLike[lw] && personVerbAfter(text, nx.end)
}

func pairCueBefore(text string, s int) bool {
	w := wordsBefore(text, s, 1)
	return len(w) > 0 && pairCues[w[0]]
}

var colourWords = wordSet(`red blue gold silver pink rose violet amber ruby scarlet orange yellow purple navy teal coral ivory olive`)

var cueCompoundWords = wordSet(`power down talk tracing secret studio notes`)

var indefinitePronouns = wordSet(`anyone anybody someone somebody everyone everybody nobody noone anything something everything nothing`)

var meetingWords = wordSet(`lunch dinner coffee breakfast brunch drinks meeting meetings call calls sync chat catchup 1:1 standup interview`)

var timeWordsAfter = wordSet(`today tomorrow tonight later`)

func meetingContext(text string, s, e int) bool {
	second := text[s:e]
	if k := strings.LastIndexByte(second, ' '); k >= 0 {
		second = second[k+1:]
	}
	if w := wordsBefore(text, s, 2); len(w) == 2 && pairCues[w[0]] && meetingWords[w[1]] && strongSurname(second) {
		return true
	}
	if timeWordsAfter[wordAfter(text, e)] {
		return true
	}
	return strings.HasPrefix(text[e:], " at ") && e+4 < len(text) && text[e+4] >= '0' && text[e+4] <= '9'
}

func compoundPersonEvidence(text string, s, e int) bool {
	return PersonActionAfter(text, e) || personDirectedBefore(text, s) || strictPersonEvidence(text, s, e) || meetingContext(text, s, e)
}

func monthPerson(text string, toks []nameTok, i int) bool {
	t := toks[i]
	if _, ok := givenOf(t.text); !ok || embedded(text, t.start, t.end) {
		return false
	}
	if e := t.end; e < len(text) && (startsHyphen(text[e:]) || text[e] == '/' || (e+1 < len(text) && text[e] == ' ' && text[e+1] >= '0' && text[e+1] <= '9')) {
		return false
	}
	if w := wordsBefore(text, t.start, 2); len(w) > 0 && (directedBefore(text, t.start) || (len(w) == 2 && w[0] == "in" && w[1] == "loop")) {
		return true
	}
	if w := wordsAfter(text, t.end, 3); len(w) > 0 && monthLinkVerbs[w[0]] {
		return monthPersonState(w[1:]) && !sentenceInList(text, t.start)
	}
	if (personVerbAfter(text, t.end) || personWorkVerbs[wordAfter(text, t.end)]) && !sentenceInList(text, t.start) {
		return true
	}
	if i+2 < len(toks) && strings.HasPrefix(text[t.end:], " and ") && toks[i+2].start == t.end+5 && monthCoordName(toks[i+2]) && PersonActionAfter(text, toks[i+2].end) && !sentenceInList(text, t.start) {
		return true
	}
	return i >= 2 && toks[i-2].end+5 == t.start && text[toks[i-2].end:t.start] == " and " && monthCoordName(toks[i-2]) && PersonActionAfter(text, t.end) && !sentenceInList(text, toks[i-2].start)
}

var monthLinkVerbs = wordSet(`is was isn't wasn't will has had`)

var monthOnlyPersonStates = wordSet(`sick tired late happy upset out away free here back ill off available`)

func monthPersonState(w []string) bool {
	for len(w) > 0 && (w[0] == "be" || w[0] == "been" || w[0] == "not") {
		w = w[1:]
	}
	if len(w) == 0 {
		return false
	}
	if w[0] == "on" {
		return len(w) > 1 && (w[1] == "leave" || w[1] == "call")
	}
	if thingPredicate(w) {
		return false
	}
	return monthOnlyPersonStates[w[0]]
}

var toHeaderRE = regexp.MustCompile(`(?m)^[ \t]*(\p{Lu}\p{Ll}+) to (\p{Lu}\p{Ll}+):[ \t]`)

func headerName(w string) bool {
	_, state := StateName(w)
	return monthCoordName(nameTok{text: w, shape: shapeOf(w)}) && !placeLike[strings.ToLower(w)] && !state && !IsMonthOrWeekday(w)
}

func toHeaderSpans(text string) []entity.Span {
	var out []entity.Span
	for _, m := range toHeaderRE.FindAllStringSubmatchIndex(text, -1) {
		a, b := text[m[2]:m[3]], text[m[4]:m[5]]
		if !headerName(a) || !headerName(b) {
			continue
		}
		out = append(out, entity.Span{Start: m[2], End: m[3], Type: entity.Person, Label: "person", Text: a, Source: "names:chat", Priority: 27, Score: 1},
			entity.Span{Start: m[4], End: m[5], Type: entity.Person, Label: "person", Text: b, Source: "names:chat", Priority: 27, Score: 1})
	}
	return out
}

func monthCoordName(t nameTok) bool {
	_, ok := givenOf(t.text)
	return ok && t.shape == shapeTitle
}

func sentenceInList(text string, s int) bool {
	w := wordsBefore(text, s, 1)
	return len(w) > 0 && (dateBefore[w[0]] || w[0] == "to")
}

var appositiveDets = wordSet(`our the my a an your their his her`)

var appositiveRoles = wordSet(`hire manager engineer lead colleague client intern ceo cto cfo coo founder cofounder director designer developer friend neighbour neighbor teammate coworker contractor recruiter analyst consultant boss assistant mentor partner`)

func appositivePerson(text string, e int) bool {
	if !strings.HasPrefix(text[e:], ", ") {
		return false
	}
	w := wordsAfter(text, e+1, 3)
	if len(w) < 2 || !appositiveDets[w[0]] {
		return false
	}
	return appositiveRoles[w[1]] || (len(w) > 2 && appositiveRoles[w[2]])
}

func thingPersonPair(text string, toks []nameTok, i int) bool {
	if i+1 >= len(toks) {
		return false
	}
	t, nx := toks[i], toks[i+1]
	if t.shape != shapeTitle || nx.shape != shapeTitle || !oneSpace(text, t.end, nx.start) || embedded(text, nx.start, nx.end) {
		return false
	}
	lw := strings.ToLower(nx.text)
	if (!thingNouns[lw] && !notSurnameExtra[lw]) || (cueCompoundWords[lw] && !meetingContext(text, t.start, nx.end)) {
		return false
	}
	if _, ok := givenOf(t.text); !ok || IsMonthOrWeekday(t.text) {
		return false
	}
	if phraseCovers(text, toks, i) || phraseCovers(text, toks, i+1) || capitalAfter(text, toks, i+1) {
		return false
	}
	if i > 0 && oneSpace(text, toks[i-1].end, t.start) && titleOrUpper(toks[i-1]) && !sentenceStart(text, t.start) {
		return false
	}
	return strictPersonEvidence(text, t.start, nx.end) || appositivePerson(text, nx.end)
}

func wrappedToken(text string, s, e int) bool {
	if s == 0 || e >= len(text) {
		return false
	}
	b := s
	if r, n := utf8.DecodeLastRuneInString(text[:b]); r == '"' || r == '\'' || r == '\u201C' || r == '\u2018' {
		b -= n
		if b > 0 && text[b-1] != '(' && text[b-1] != ' ' && text[b-1] != '\t' && text[b-1] != '\n' {
			return false
		}
	} else if r != '(' {
		return false
	}
	r, n := utf8.DecodeRuneInString(text[e:])
	if r == '\'' || r == '\u2019' {
		if nr, _ := utf8.DecodeRuneInString(text[e+n:]); unicode.IsLetter(nr) {
			return false
		}
	}
	return r == ')' || r == '"' || r == '\'' || r == '\u201D' || r == '\u2019'
}

var nicknameRE = regexp.MustCompile(`^ ?(?:\([ \t]?["'\x{201C}\x{2018}]?|["'\x{201C}\x{2018}])(\p{Lu}\p{Ll}+)(?:["'\x{201D}\x{2019}]?[ \t]?\)|["'\x{201D}\x{2019}])`)

func nicknameSpans(text string, spans []entity.Span) []entity.Span {
	var out []entity.Span
	for _, p := range spans {
		if p.Type != entity.Person || p.End >= len(text) {
			continue
		}
		first := p.Text
		if k := strings.IndexByte(first, ' '); k > 0 {
			first = first[:k]
		}
		if _, ok := givenOf(first); !ok {
			continue
		}
		hi := p.End + 48
		if hi > len(text) {
			hi = len(text)
		}
		m := nicknameRE.FindStringSubmatchIndex(text[p.End:hi])
		if m == nil {
			continue
		}
		s, e := p.End+m[2], p.End+m[3]
		w := text[s:e]
		if _, ok := givenOf(w); !ok || IsMonthOrWeekday(w) {
			continue
		}
		if _, state := StateName(w); state {
			continue
		}
		if e+1 < len(text) {
			r, _ := utf8.DecodeRuneInString(text[p.End+m[1]:])
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				continue
			}
		}
		out = append(out, entity.Span{Start: s, End: e, Type: entity.Person, Label: "person", Text: w, Source: "names:nick", Priority: 26, Score: 1})
		if strings.Contains(p.Text, " ") {
			continue
		}
		if ss, se := surnameAfterNick(text, p.End+m[1]); se > 0 {
			out = append(out, entity.Span{Start: ss, End: se, Type: entity.Person, Label: "person", Text: text[ss:se], Source: "names:last", Priority: 26, Score: 1})
		}
	}
	return out
}

func surnameAfterNick(text string, pos int) (int, int) {
	if pos+2 >= len(text) || text[pos] != ' ' {
		return 0, 0
	}
	hi := pos + 64
	if hi > len(text) {
		hi = len(text)
	}
	m := wordRE.FindStringIndex(text[pos+1 : hi])
	if m == nil || m[0] != 0 {
		return 0, 0
	}
	s, e := pos+1, pos+1+m[1]
	w := text[s:e]
	if shapeOf(w) != shapeTitle || !surnameLike(w) || IsTechTerm(w) || thingNouns[strings.ToLower(w)] || embedded(text, s, e) {
		return 0, 0
	}
	if !knownSurname(w) && textutil.IsCommonWord(w) {
		return 0, 0
	}
	return s, e
}

func coordinatedPairs(text string, toks []nameTok, spans []entity.Span) []entity.Span {
	starts, ends := map[int]bool{}, map[int]bool{}
	for _, p := range spans {
		if p.Type == entity.Person && strings.Contains(p.Text, " ") {
			starts[p.Start], ends[p.End] = true, true
		}
	}
	if len(starts) == 0 {
		return nil
	}
	var out []entity.Span
	for i := 0; i+1 < len(toks); i++ {
		t, nx := toks[i], toks[i+1]
		if starts[t.start] || !coordPair(text, toks, i) || !familyListed(text, t.start, nx.end, "", starts, ends) {
			continue
		}
		out = append(out, entity.Span{Start: t.start, End: nx.end, Type: entity.Person, Label: "person", Text: text[t.start:nx.end], Source: "names:full", Priority: 28, Score: 1})
		starts[t.start], ends[nx.end] = true, true
		i++
	}
	return out
}

var coordSurnameNouns = wordSet(`park hall house`)

func coordPair(text string, toks []nameTok, i int) bool {
	t, nx := toks[i], toks[i+1]
	if t.shape != shapeTitle || nx.shape != shapeTitle || !oneSpace(text, t.end, nx.start) || embedded(text, t.start, t.end) || embedded(text, nx.start, nx.end) {
		return false
	}
	if _, ok := givenOf(t.text); !ok || IsMonthOrWeekday(t.text) {
		return false
	}
	lw := strings.ToLower(nx.text)
	if utf8.RuneCountInString(nx.text) < 2 || stopCap[lw] || techTerms[lw] || IsMonthOrWeekday(nx.text) {
		return false
	}
	if _, state := StateName(nx.text); state {
		return false
	}
	if (!surnameLike(nx.text) || thingNouns[lw]) && !(coordSurnameNouns[lw] && knownSurname(nx.text)) {
		return false
	}
	return !phraseCovers(text, toks, i) && !phraseCovers(text, toks, i+1)
}

func dashStart(text string, s int) int {
	j := s
	for j > 0 && s-j < maxDashGap && (text[j-1] == ' ' || text[j-1] == '\t') {
		j--
	}
	if j < 2 || text[j-2:j] != "--" {
		return -1
	}
	return j - 2
}

func dashBefore(text string, s int) bool {
	d := dashStart(text, s)
	return d > 0 && text[d-1] == ' ' && d >= 2 && text[d-2] != ' ' && text[d-2] != '\n' && text[d-2] != '-'
}

func dashCommentBefore(text string, s, e int) bool {
	d := dashStart(text, s)
	return d >= 0 && codeDash(text, d, e)
}

var dashPersonWords = wordSet(`was is were won wins said says has had will would`)

func codeDash(text string, dash, nameEnd int) bool {
	if personVerbAfter(text, nameEnd) || dashPersonWords[wordAfter(text, nameEnd)] {
		return false
	}
	return codeLine(text, dash)
}

var sqlStatementKeywords = wordSet(`select insert update delete create alter drop with grant revoke truncate`)

var sqlObjects = wordSet(`table index view database schema`)

var assignLineRE = regexp.MustCompile(`^[ \t]*(?:(?:let|var|const|int|float|double|string|bool|auto|val|local|export|my)[ \t]+)?[A-Za-z_][\w.]*[ \t]*:?=(?:[^=]|$)`)

func codeLine(text string, end int) bool {
	lo := end - 200
	if lo < 0 {
		lo = 0
	}
	if ls := strings.LastIndexByte(text[lo:end], '\n'); ls >= 0 {
		lo += ls + 1
	} else if lo > 0 {
		return false
	}
	seg := strings.TrimRight(text[lo:end], " \t")
	words := strings.Fields(seg)
	if len(words) == 0 {
		return false
	}
	if w := strings.TrimRight(words[0], ";,("); sqlStatementKeywords[strings.ToLower(w)] && (textutil.IsAllUpper(w) || sqlStatementShape(words)) {
		return true
	}
	if strings.HasSuffix(seg, ";") && statementBeforeSemicolon(strings.TrimRight(seg[:len(seg)-1], " \t"), words) {
		return true
	}
	return assignLineRE.MatchString(seg)
}

func statementBeforeSemicolon(body string, words []string) bool {
	if body == "" {
		return false
	}
	c := body[len(body)-1]
	code := c == ')' || c == ']' || c == '"' || c == '\'' || (c >= '0' && c <= '9')
	if !code && !strings.ContainsAny(body, " \t") && strings.ContainsAny(body, "_.(0123456789") {
		code = true
	}
	if !code {
		return false
	}
	run := 0
	for _, w := range words {
		if w = strings.TrimRight(w, ";,."); w != "" && textutil.IsAllLower(w) && strings.IndexFunc(w, func(r rune) bool { return !unicode.IsLetter(r) }) < 0 {
			if run++; run >= 3 {
				return false
			}
			continue
		}
		run = 0
	}
	return true
}

func sqlStatementShape(w []string) bool {
	if len(w) < 2 {
		return false
	}
	lw := func(i int) string { return strings.ToLower(strings.TrimRight(w[i], ";,")) }
	switch lw(0) {
	case "select":
		for i := 1; i < len(w); i++ {
			if lw(i) == "from" {
				return true
			}
		}
	case "insert":
		return lw(1) == "into"
	case "delete":
		return lw(1) == "from"
	case "update":
		return len(w) >= 3 && lw(2) == "set"
	case "create", "alter", "drop":
		return sqlObjects[lw(1)]
	}
	return false
}

var midDashSignRE = regexp.MustCompile(`(?m)[^\s][ \t]+(--|\x{2013}|\x{2014})[ \t]+(` + nameWord + `(?: ` + nameWord + `)?)[ \t\r]*$`)

var signoffLast = wordSet(`love thanks thx cheers best regards sincerely truly xo xoxo`)

var signoffPairs = wordSet(`you soon care`)

func signoffBefore(text string, p int) bool {
	lo := p - 32
	if lo < 0 {
		lo = 0
	}
	for p > lo && strings.IndexByte(" \t\r,", text[p-1]) >= 0 {
		p--
	}
	k := p
	for k > lo && asciiLetter(text[k-1]) {
		k--
	}
	if k == p || (k > lo && text[k-1] != ' ' && text[k-1] != '\t' && text[k-1] != '\n') {
		return false
	}
	w := strings.ToLower(text[k:p])
	if signoffLast[w] {
		return true
	}
	if !signoffPairs[w] || k-1 <= lo || text[k-1] != ' ' {
		return false
	}
	j := k - 1
	for j > lo && asciiLetter(text[j-1]) {
		j--
	}
	switch strings.ToLower(text[j : k-1]) {
	case "thank":
		return w == "you"
	case "talk":
		return w == "soon"
	case "take":
		return w == "care"
	}
	return false
}

func looseSignName(name string, signoff bool) bool {
	words := strings.Fields(name)
	if _, ok := givenOf(words[0]); !ok || IsMonthOrWeekday(words[0]) {
		return false
	}
	if _, state := StateName(words[0]); state && !signoff {
		return false
	}
	for i, w := range words {
		if IsTechTerm(w) || textutil.IsAllUpper(w) || (i > 0 && !surnameLike(w)) {
			return false
		}
	}
	return true
}

func asciiLetter(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

func headingBefore(text string, toks []nameTok, i int) bool {
	if i == 0 {
		return false
	}
	p := toks[i-1]
	return oneSpace(text, p.end, toks[i].start) && (p.shape == shapeTitle || p.shape == shapeUpper) && !sentenceStart(text, toks[i].start) && !lowerNameBefore[strings.ToLower(p.text)]
}

func weakContextOK(text string, s, e int) bool {
	if s > 0 && text[s-1] == '@' {
		return true
	}
	if lowerNameBefore[strings.ToLower(wordBefore(text, s))] {
		if e >= len(text) {
			return true
		}
		r, _ := utf8.DecodeRuneInString(text[e:])
		return r == '\n' || strings.ContainsRune(",.!?:;)", r)
	}
	if lowerNameAfter[wordAfter(text, e)] {
		return clauseStart(text, s)
	}
	ls, le := lineBounds(text, s, e)
	line := strings.TrimSpace(text[ls:le])
	if strings.TrimRight(line, ".,!:;-") == text[s:e] {
		return true
	}
	return speakerColon(strings.TrimLeft(text[e:le], " ")) && strings.TrimSpace(text[ls:s]) == ""
}

func labelNames(text string, comps []companyMatch) []entity.Span {
	var out []entity.Span
	for _, m := range labelRE.FindAllStringSubmatchIndex(text, -1) {
		label := strings.ToLower(text[m[2]:m[3]])
		pos := m[1]
		lineEnd := len(text)
		if le := strings.IndexByte(text[pos:], '\n'); le >= 0 {
			lineEnd = pos + le
		}
		for {
			seg := text[pos:lineEnd]
			var words [][]int
			for _, w := range wordRE.FindAllStringIndex(seg, 5) {
				if len(words) == 0 && w[0] != 0 {
					break
				}
				if len(words) > 0 && !(w[0] == words[len(words)-1][1]+1 && seg[w[0]-1] == ' ') {
					break
				}
				sh := shapeOf(seg[w[0]:w[1]])
				if sh != shapeTitle && sh != shapeUpper {
					break
				}
				words = append(words, w)
				if len(words) == 4 {
					break
				}
			}
			if len(words) == 0 {
				break
			}
			known, valid := false, true
			for wi, w := range words {
				ww := seg[w[0]:w[1]]
				lw := strings.ToLower(ww)
				if wi == 0 && len(words) > 1 {
					if stopCap[lw] || techTerms[lw] || IsMonthOrWeekday(ww) {
						valid = false
					}
				} else if !surnameLike(ww) {
					valid = false
				}
				if _, ok := givenOf(ww); ok {
					known = true
				}
				if knownSurname(ww) {
					known = true
				}
			}
			end := words[len(words)-1][1]
			if embedded(text, pos, pos+end) {
				valid = false
			}
			if valid && (known || (strongLabels[label] && len(words) >= 2)) && !(len(words) == 1 && !known) && (strongLabels[label] || !inCompany(comps, pos, pos+end)) {
				out = append(out, entity.Span{Start: pos, End: pos + end, Type: entity.Person, Label: "person", Text: seg[:end], Source: "names:label", Priority: 27, Score: 1})
			}
			rest := seg[end:]
			if !valid || !(strings.HasPrefix(rest, ", ") || strings.HasPrefix(rest, "; ") || strings.HasPrefix(rest, " and ") || strings.HasPrefix(rest, " & ")) {
				break
			}
			pos += end + strings.Index(rest, " ") + 1
			if strings.HasPrefix(text[pos:], "and ") {
				pos += 4
			}
		}
	}
	return out
}

var particleTailRE = regexp.MustCompile(`^((?: (?:de|del|della|la|las|los|y|e|da|do|dos|das|di|du|van|von|der|den|ten|ter|le|zu|St\.?|Ste\.?)){1,3}) (\p{Lu}[\p{Ll}\p{M}]+(?:-\p{Lu}[\p{Ll}\p{M}]+)?)`)

var particleEndRE = regexp.MustCompile(`^ ([Vv]an|[Dd]e|[Vv]on|[Dd]el|[Dd]a|[Dd]i|[Dd]u|[Ll]e|[Ll]a)\b`)

func particleTails(text string, spans []entity.Span) []entity.Span {
	var out []entity.Span
	for _, sp := range spans {
		if sp.Type != entity.Person || sp.End >= len(text) || text[sp.End] != ' ' {
			continue
		}
		window := text[sp.End:min(len(text), sp.End+wordScanLimit)]
		m := particleTailRE.FindStringSubmatchIndex(window)
		if m == nil {
			continue
		}
		w := window[m[4]:m[5]]
		if !surnameLike(w) || thingNouns[strings.ToLower(w)] || placeLike[strings.ToLower(w)] || (!knownSurname(w) && textutil.IsCommonWord(w)) {
			continue
		}
		end := sp.End + m[5]
		if end < len(text) && (unicode.IsLetter(textutil.RuneAt(text, end)) || unicode.IsDigit(textutil.RuneAt(text, end))) {
			continue
		}
		ext := sp
		ext.End, ext.Text, ext.Canonical = end, text[sp.Start:end], ""
		out = append(out, ext)
	}
	for _, sp := range spans {
		if sp.Type != entity.Person || sp.End >= len(text) || text[sp.End] != ' ' {
			continue
		}
		m := particleEndRE.FindStringSubmatchIndex(text[sp.End:min(len(text), sp.End+wordScanLimit)])
		if m == nil {
			continue
		}
		end := sp.End + m[3]
		if rest := wordsAfter(text, end, 1); len(rest) == 0 && end < len(text) && text[end] == ' ' {
			continue
		}
		if nx := wordAfter(text, end); end < len(text) && text[end] == ' ' && (nameParticles[strings.ToLower(nx)] || titleWordAt(text, end+1)) {
			continue
		}
		if !PersonActionAfter(text, end) && !personDirectedBefore(text, sp.Start) {
			continue
		}
		ext := sp
		ext.End, ext.Text, ext.Canonical = end, text[sp.Start:end], ""
		out = append(out, ext)
	}
	return out
}

func titleWordAt(text string, i int) bool {
	r, _ := utf8.DecodeRuneInString(text[i:])
	return i < len(text) && unicode.IsUpper(r)
}

func nickSurnames(text string, spans []entity.Span) []entity.Span {
	nickEnd := map[int]bool{}
	for _, sp := range spans {
		if sp.Source == "names:nick" && sp.End+2 < len(text) && strings.IndexByte("\"')", text[sp.End]) >= 0 && text[sp.End+1] == ' ' {
			nickEnd[sp.End+2] = true
		}
		if sp.Source == "names:nick" && sp.End+4 < len(text) && strings.HasPrefix(text[sp.End:], "\u201d ") {
			nickEnd[sp.End+4] = true
		}
	}
	if len(nickEnd) == 0 {
		return spans
	}
	out := spans[:0:0]
	seen := map[int]bool{}
	for _, sp := range spans {
		if sp.Type == entity.Person && nickEnd[sp.Start] && len(textutil.Tokens(sp.Text)) == 1 && (sp.Source == "names:first" || sp.Source == "names:last") {
			if !seen[sp.Start] && knownSurname(sp.Text) {
				seen[sp.Start] = true
				sp.Source = "names:nick-surname"
				sp.Priority = prioFamilyFirst
				out = append(out, sp)
			}
			if knownSurname(sp.Text) {
				continue
			}
		}
		out = append(out, sp)
	}
	return out
}
