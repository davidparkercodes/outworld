package detect

import (
	"context"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/namedata"
	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/entity"
)

const nameWord = `\p{Lu}\p{L}*(?:['’\-]\p{Lu}?\p{Ll}+)*`

var (
	wordRE = regexp.MustCompile(`\p{L}+(?:['’\-]\p{L}+)*`)

	labelRE = regexp.MustCompile(`(?im)^[ \t>*#\-]*(?:[A-Za-z][A-Za-z/\-]*[ \t]){0,3}?(on-?call|owner|lead|contact|attn|attention|from|to|cc|bcc|assignee|assigned to|reviewer|author|manager|patient|client|customer|name|signed|prepared by|approved by|requested by|reported by|submitted by|organizer|host|attendees?|driver|tenant|landlord|buyer|seller|applicant|employee|supervisor|physician|doctor|provider|beneficiary|insured|claimant|witness|recipient|sender)\*{0,2}[ \t]*:[ \t]*\*{0,2}`)

	emailNameRE = regexp.MustCompile(`((?:` + nameWord + `[ \t]){1,3})[<(\[][ \t]*(?:mailto:)?[A-Za-z0-9._%+\-]+@`)

	kinRE = regexp.MustCompile(`(?i:\b(?:my|his|her|our|their|your)\s+(?:(?:older|younger|little|big|late|step|twin|best|old|new|former|dear|ex)[\s\-]+)*(?:sister|brother|wife|husband|son|daughter|mother|father|mom|mum|dad|aunt|uncle|cousin|niece|nephew|grandmother|grandfather|grandma|grandpa|friend|girlfriend|boyfriend|partner|fianc[eé]e?|roommate|neighbou?r|colleague|co-?worker|boss|manager|assistant|doctor|therapist|lawyer|attorney|accountant|landlord|teacher|coach|mentor|buddy|kid|child|spouse|ex|stepmom|stepdad|(?:mother|father|sister|brother|son|daughter)-in-law))\s*,?\s+(` + nameWord + `(?: ` + nameWord + `)?)`)

	slugRE = regexp.MustCompile(`(?i)/(people|person|users?|u|profiles?|members?|staff|team|authors?|employees?|contacts?|in|by)/([a-z]+(?:[-._][a-z]+){1,3})(?:[/?#"'\s)>\]]|$)`)
)

var strongLabels = wordSet(`contact oncall on-call patient physician doctor driver tenant landlord applicant beneficiary insured claimant witness lead owner assignee reviewer author manager supervisor attn attention`)

var strongSlugPaths = wordSet(`people person profile profiles author authors staff in employees employee contacts contact`)

var stopCap = wordSet(`the a an and or but if of in on at to for with from by as is are was were be been it its this that these those i you he she we they my your his her our their not no yes do does did have has had will would can could should may might must shall who what when where why how which new old please thanks thank hello hi hey dear re fw fwd subject note todo fixme warning error info debug true false null none ok okay all any each every some here there today tomorrow yesterday now then also just very more most other such only own same so than too once etc mr mrs ms mx dr prof rev sir madam miss regards sincerely best cheers`)

var placeLike = wordSet(`monte paris london florence austin dallas houston sydney chelsea brooklyn israel india china kenya savannah charlotte phoenix madison jackson lincoln orlando milan vienna sierra dakota montana camden boston denver devon berlin ireland america asia trinity kingston santiago victoria adelaide alexandria odessa columbus cleveland raleigh eugene troy carmel geneva verona sofia holland britain egypt persia cheyenne indiana nevada georgia virginia carolina washington jordan chad`)

var techTerms = wordSet(`claude code anthropic openai chatgpt gpt gemini copilot cursor codex llama mistral siri alexa cortana watson java javascript typescript python ruby rust golang swift kotlin scala haskell julia ada pascal perl lua dart elixir erlang crystal basic cobol fortran lisp scheme racket clojure groovy bash zsh shell docker kubernetes helm jenkins travis github gitlab bitbucket jira confluence slack discord linux ubuntu debian fedora windows macos android node deno react vue angular svelte django flask rails laravel spring maven gradle npm yarn cargo redis postgres mysql sqlite mongo cassandra kafka spark hadoop airflow nginx apache tomcat jasmine mocha jest cypress selenium webpack vite babel eslint prettier adam bert elmo ernie apollo athena hermes atlas aurora lambda terraform ansible puppet chef vagrant jupyter pandas numpy homer mercury venus mars jupiter saturn neptune pluto hugo jekyll gatsby jade pug sass markdown unicode ascii json yaml toml sonnet opus haiku fable mythos`)

var notSurnameExtra = wordSet(`carlo river lake station airport bay island city county mountain language optimizer memorial bridge tower hall building library museum stadium arena theater theatre market mall plaza garden gardens beach harbor harbour port valley canyon forest desert ocean sea falls springs hotel inn cafe restaurant bar grill pizza kitchen studio studios media news times post journal daily weekly magazine press records music films pictures games api sdk cli app server client service services database cluster pipeline workflow dashboard report accounts account billing support sales legal team info admin office department dept desk help hr finance marketing engineering security operations ops meeting agenda summary update review notes plan budget invoice contract agreement policy act bill law code`)

var modalNames = wordSet(`will may`)

var dateAbbrev = wordSet(`jan feb mar apr jun jul aug sep sept oct nov dec mon tue tues wed thu thur thurs fri sat sun`)

var placeBefore = wordSet(`in near around outside across throughout via`)

var placeAfter = wordSet(`office offices region area branch campus site county city district province state location store plant facility warehouse hospital clinic airport station street road avenue metro suburbs downtown`)

var signoffRE = regexp.MustCompile(`(?im)(?:^|[.!?]\s+)(?:love|thanks|thank you|many thanks|best|best regards|kind regards|warm regards|regards|cheers|sincerely|yours|yours truly|talk soon|xo|xoxo|take care),?[ \t]*\n?[ \t]*(` + nameWord + `(?: ` + nameWord + `)?)[ \t]*$`)

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
	return true
}

func knownSurname(w string) bool {
	if i := strings.LastIndexAny(w, "-"); i > 0 {
		_, a := namedata.LookupSurname(w[:i])
		_, b := namedata.LookupSurname(w[i+1:])
		return a || b
	}
	_, ok := namedata.LookupSurname(w)
	return ok
}

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
	for j > 0 {
		r, n := utf8.DecodeLastRuneInString(text[:j])
		if r == ' ' || r == '\t' {
			j -= n
			continue
		}
		return r == '\n' || strings.ContainsRune(".:;!?,-–—>*|", r)
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
	var toks []nameTok
	for _, m := range wordRE.FindAllStringIndex(text, -1) {
		w := text[m[0]:m[1]]
		toks = append(toks, nameTok{m[0], m[1], w, shapeOf(w)})
	}
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		if embedded(text, t.start, t.end) {
			continue
		}
		if end, n, canonical := familyFirst(text, toks, i); n > 0 {
			out = append(out, entity.Span{Start: t.start, End: end, Type: entity.Person, Label: "person", Text: text[t.start:end], Source: "names:family-first", Priority: 30, Score: 1, Canonical: canonical})
			i += n
			continue
		}
		g, ok := givenOf(t.text)
		if !ok {
			if end, n := surnameLed(text, toks, i); n > 0 {
				emit(t.start, end, "full", 25)
				i += n
			} else if t.shape == shapeTitle && strongSurname(t.text) && lowerNameContext(text, t.start, t.end) && !placeUse(text, t.start, t.end) {
				emit(t.start, t.end, "last", 22)
			}
			continue
		}
		switch t.shape {
		case shapeTitle, shapeUpper:
			end, n := t.end, 0
			if pe, pn := particleSurname(text, toks, i); pn > 0 && (!g.Weak || NameUseOK(text, t.start, t.end)) {
				emit(t.start, pe, "full", 28)
				i += pn
				continue
			}
			if i+1 < len(toks) {
				nx := toks[i+1]
				if nx.shape == t.shape && oneSpace(text, t.end, nx.start) && !embedded(text, nx.start, nx.end) && surnameLike(nx.text) {
					known := knownSurname(nx.text)
					accept := false
					switch {
					case !g.Weak:
						accept = true
					case !NameUseOK(text, t.start, t.end):
					case known:
						accept = true
					case utf8.RuneCountInString(nx.text) >= 4 && !textutil.IsCommonWord(nx.text):
						accept = !(modalNames[strings.ToLower(t.text)] && sentenceStart(text, t.start) && continuesClause(text, nx.end)) && !headingBefore(text, toks, i)
					}
					if accept {
						end, n = nx.end, 1
						if i+2 < len(toks) {
							th := toks[i+2]
							_, midGiven := namedata.LookupGiven(nx.text)
							if th.shape == t.shape && oneSpace(text, nx.end, th.start) && surnameLike(th.text) && knownSurname(th.text) && (midGiven || known) {
								end, n = th.end, 2
							}
						}
					}
				}
			}
			if n > 0 {
				emit(t.start, end, "full", 28)
				i += n
				continue
			}
			if t.shape == shapeUpper || utf8.RuneCountInString(t.text) < 3 {
				continue
			}
			if !g.Weak {
				if NameUseOK(text, t.start, t.end) && !placeUse(text, t.start, t.end) {
					emit(t.start, t.end+trailingInitial(text, t.end), "first", 26)
				}
				continue
			}
			if !NameUseOK(text, t.start, t.end) {
				continue
			}
			if lowerNameContext(text, t.start, t.end) && weakContextOK(text, t.start, t.end) {
				emit(t.start, t.end, "first", 24)
				continue
			}
			lw := strings.ToLower(t.text)
			_, isState := StateName(t.text)
			if placeLike[lw] || isState || IsMonthOrWeekday(t.text) || sentenceStart(text, t.start) || placeUse(text, t.start, t.end) {
				continue
			}
			if prev := wordBefore(text, t.start); prev == "" || !textutil.IsAllLower(prev) || determiners[prev] {
				continue
			}
			if i+1 < len(toks) && oneSpace(text, t.end, toks[i+1].start) && toks[i+1].shape != shapeLower {
				continue
			}
			emit(t.start, t.end, "first", 23)
		case shapeLower:
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
			if !surnameLike(w) || IsTechTerm(w) || textutil.IsAllUpper(w) {
				ok = false
			}
		}
		if ok {
			emit(s, e, "signoff", 27)
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
	out = append(out, handleSpans(text)...)
	return guardOrgs(text, out, comps), nil
}

func particleSurname(text string, toks []nameTok, i int) (int, int) {
	if toks[i].shape != shapeTitle {
		return 0, 0
	}
	j := i + 1
	for j < len(toks) && j-i <= 3 && nameParticles[strings.ToLower(toks[j].text)] && oneSpace(text, toks[j-1].end, toks[j].start) {
		j++
	}
	if j == i+1 || j >= len(toks) {
		return 0, 0
	}
	last := toks[j]
	if last.shape != shapeTitle || !oneSpace(text, toks[j-1].end, last.start) || !surnameLike(last.text) || textutil.IsCommonWord(last.text) || embedded(text, last.start, last.end) {
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
	if i := strings.IndexByte(w, '-'); i > 0 && i < len(w)-1 {
		return strongSurname(w[:i]) || strongSurname(w[i+1:])
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

var cityStateAfter = regexp.MustCompile(`^, [A-Z]{2}\b`)

func placeUse(text string, s, e int) bool {
	return placeBefore[strings.ToLower(wordBefore(text, s))] || placeAfter[wordAfter(text, e)] || cityStateAfter.MatchString(text[e:])
}

func headingBefore(text string, toks []nameTok, i int) bool {
	if i == 0 {
		return false
	}
	p := toks[i-1]
	return oneSpace(text, p.end, toks[i].start) && (p.shape == shapeTitle || p.shape == shapeUpper) && !sentenceStart(text, toks[i].start)
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
		for {
			le := strings.IndexByte(text[pos:], '\n')
			if le < 0 {
				le = len(text) - pos
			}
			seg := text[pos : pos+le]
			var words [][]int
			for _, w := range wordRE.FindAllStringIndex(seg, -1) {
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
