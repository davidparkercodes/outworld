package detect

import (
	"strings"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/namedata"
	"github.com/davidparkercodes/outworld/internal/textutil"
)

var thingNouns = wordSet(`ave blvd rd hwy pkwy room rooms hall house building suite lounge center centre theatre theater library wing tower bridge park street avenue road boulevard lane drive court square plaza terrace place way highway parkway trail award awards prize medal trophy cup scholarship fellowship fund foundation memorial project protocol method test theorem algorithm law principle effect equation scale index syndrome disease institute museum gallery garden gardens pavilion auditorium arena stadium chapel cathedral school college university academy hospital clinic lab labs laboratory kernel model distribution conjecture paradox machine cafe café bar restaurant hotel inn pub club estate station airport terminal campus conference summit festival tech falls beach river lake bay island islands city county mountain mountains valley canyon forest desert springs harbor harbour heights point glacier reef gulf sea ocean coast province district township village town market mall cemetery monument collection edition series cycle engine filter transform function curve diagram number code cipher salad sauce sandwich cocktail`)

var commemorativeNouns = wordSet(`award awards prize medal trophy cup scholarship fellowship memorial fund library building center centre theatre theater hall museum stadium arena airport bridge tower park institute school wing pavilion auditorium gallery lecture`)

var placeLeads = wordSet(`san santa santo sao são saint st ste mount mt fort ft port lake cape los las new north south east west upper lower puerto ciudad`)

var dottedLeads = wordSet(`st ste mt ft`)

var thingLeads = wordSet(`room suite hotel villa camp project operation cafe café casa chez`)

var stormLeads = wordSet(`hurricane storm cyclone typhoon`)

var eponymNouns = wordSet(`theorem method chain chains distribution constant equation algorithm effect principle paradox machine syndrome disease kernel series inequality conjecture lemma diagram completeness sims simulation simulations sampling estimator approximation interpolation polynomial matrix sequence coefficient correlation criterion integral operator notation cipher awards`)

var eponymNounsAfterArticle = wordSet(`test model rule process filter transform scale function curve cycle engine code graph plot chart score index number law award prize`)

var eponymPossessed = wordSet(`theorem law lemma conjecture paradox constant equation inequality principle razor algorithm syndrome disease method`)

var appositiveNouns = wordSet(`kernel language compiler library framework database tool app product company brand band city town river island ship movie film book novel song album game project protocol model system platform service store restaurant hotel satellite rover telescope hurricane storm crater planet mission spacecraft award awards program programme chatbot package module plugin distro distribution`)

var appositiveStops = wordSet(`and or is was has are were`)

var articles = wordSet(`the a an`)

var regionCodes = wordSet(`AK AR AZ CA CT DC FL GA IA IL KS KY MI MN MO MT NC ND NE NH NJ NM NV NY RI SC SD TN TX UT VA VT WA WI WV WY AB BC MB NB NL NS NT NU ON PE QC SK YT`)

var regionPreps = wordSet(`in near around outside across throughout via to from at through of`)

func regionShapeAfter(text string, s, e int, anyCode bool) bool {
	rest := text[e:]
	comma := strings.HasPrefix(rest, ",")
	if comma {
		rest = rest[1:]
	}
	if len(rest) < 3 || rest[0] != ' ' || rest[1] < 'A' || rest[1] > 'Z' || rest[2] < 'A' || rest[2] > 'Z' {
		return false
	}
	if !anyCode && !regionCodes[rest[1:3]] {
		return false
	}
	prev := ""
	if w := wordsBefore(text, s, 1); len(w) > 0 {
		prev = w[0]
	}
	if lowerNameBefore[prev] {
		return false
	}
	tail := rest[3:]
	if tail == "" || strings.IndexByte(",.;:)\r\n", tail[0]) >= 0 {
		return true
	}
	if tail[0] != ' ' || len(tail) < 2 {
		return false
	}
	if tail[1] >= '0' && tail[1] <= '9' {
		return true
	}
	return regionPreps[prev]
}

func regionCodeAfter(text string, s, e int) bool { return regionShapeAfter(text, s, e, false) }

func foldTok(w string) string {
	w = textutil.Fold(w)
	return strings.ReplaceAll(w, "’", "'")
}

func titleOrUpper(t nameTok) bool { return t.shape == shapeTitle || t.shape == shapeUpper }

func capitalized(w string) bool { return textutil.StartsUpper(w) }

func phraseCovers(text string, toks []nameTok, i int) bool {
	for j := i; j >= 0 && j > i-4; j-- {
		if j < i && !phraseGap(text[toks[j].end:toks[j+1].start]) {
			break
		}
		max := namedata.PhraseWords(foldTok(toks[j].text))
		if max == 0 {
			continue
		}
		parts := []string{foldTok(toks[j].text)}
		for k := j + 1; k < len(toks) && len(parts) < max; k++ {
			if !phraseGap(text[toks[k-1].end:toks[k].start]) {
				break
			}
			parts = append(parts, foldTok(toks[k].text))
			if k < i {
				continue
			}
			if phrase := strings.Join(parts, " "); namedata.IsPhrase(phrase) && !(namedata.PersonalPhrase(phrase) && phrasePerson(text, toks, j, k)) {
				return true
			}
		}
	}
	if parts := textutil.NameJoinerFields(foldTok(toks[i].text)); len(parts) > 1 {
		phrase := strings.Join(parts, " ")
		return namedata.IsPhrase(phrase) && !(namedata.PersonalPhrase(phrase) && phrasePerson(text, toks, i, i))
	}
	return false
}

var placeVerbs = wordSet(`visit visited saw see seeing`)

var personPossessed = wordSet(`wife husband son daughter mother father mom mum dad brother sister partner boss manager assistant team email emails phone number address desk office laptop calendar account review report feedback birthday family kids children wedding funeral resume cv salary doctor lawyer`)

func labelBefore(text string, s int) bool {
	j := s
	for j > 0 && (text[j-1] == ' ' || text[j-1] == '\t' || text[j-1] == '*') {
		j--
	}
	if j == 0 || text[j-1] != ':' {
		return false
	}
	for j > 0 && (text[j-1] == ':' || text[j-1] == '*') {
		j--
	}
	k := j
	for k > 0 && k > j-24 && text[k-1] >= 'A' && text[k-1] <= 'z' {
		k--
	}
	return listLabelWords[strings.ToLower(text[k:j])]
}

func directedBefore(text string, s int) bool {
	w := wordsBefore(text, s, 2)
	if len(w) == 0 {
		return false
	}
	if directedVerbs[w[0]] {
		return !placeVerbs[w[0]]
	}
	return len(w) > 1 && directedPreps[w[0]] && directedPrepVerbs[w[1]]
}

func strictPersonEvidence(text string, s, e int) bool {
	prev := ""
	if w := wordsBefore(text, s, 1); len(w) > 0 {
		prev = w[0]
	}
	if labelBefore(text, s) {
		return true
	}
	if determiners[prev] || prev == "at" {
		return false
	}
	if personVerbAfter(text, e) || lowerNameAfter[wordAfter(text, e)] || directedBefore(text, s) {
		return true
	}
	if roleBefore[prev] || lowerNameBefore[prev] || honorificWords[strings.Trim(prev, ".")] {
		return true
	}
	rest := text[e:]
	for _, suf := range []string{"'s", "’s"} {
		if strings.HasPrefix(rest, suf) {
			if w := wordsAfter(text, e+len(suf), 1); len(w) > 0 && personPossessed[w[0]] {
				return true
			}
		}
	}
	return false
}

func phrasePerson(text string, toks []nameTok, j, k int) bool {
	first, last := toks[j], toks[k]
	if k+1 < len(toks) {
		nx := toks[k+1]
		nlw := strings.ToLower(nx.text)
		if nx.shape == last.shape && titleOrUpper(nx) && oneSpace(text, last.end, nx.start) && knownSurname(nx.text) && surnameLike(nx.text) && !textutil.IsCommonWord(nx.text) && !thingNouns[nlw] && !commemorativeNouns[nlw] {
			return true
		}
	}
	if j > 0 {
		p := toks[j-1]
		if p.shape == shapeTitle && oneSpace(text, p.end, first.start) && !textutil.IsCommonWord(p.text) && !stopCap[strings.ToLower(p.text)] {
			if g, ok := givenOf(p.text); ok && !g.Weak {
				return true
			}
		}
	}
	return strictPersonEvidence(text, first.start, last.end)
}

func phraseGap(gap string) bool {
	if gap == " " {
		return true
	}
	r, n := utf8.DecodeRuneInString(gap)
	return n == len(gap) && textutil.IsHyphen(r)
}

func leadGap(lead, gap string) bool {
	return gap == " " || (gap == ". " && dottedLeads[lead])
}

func pairPersonEvidence(text string, s, e int, first string) bool {
	prev := ""
	if w := wordsBefore(text, s, 1); len(w) > 0 {
		prev = w[0]
	}
	if determiners[prev] || prev == "at" {
		return false
	}
	if personVerbAfter(text, e) || lowerNameAfter[wordAfter(text, e)] || personDirectedBefore(text, s) {
		return true
	}
	if roleBefore[prev] || lowerNameBefore[prev] || honorificWords[strings.Trim(prev, ".")] || listLabelWords[prev] || prev == "with" {
		return true
	}
	if placeBefore[prev] {
		return false
	}
	g, ok := givenOf(first)
	return ok && !g.Weak
}

func appositiveThing(text string, e int) bool {
	rest := text[e:]
	if !strings.HasPrefix(rest, " the ") {
		return false
	}
	rest = rest[5:]
	if len(rest) > 96 {
		rest = rest[:96]
	}
	if cut := strings.IndexAny(rest, ".,;:!?()\n"); cut >= 0 {
		rest = rest[:cut]
	}
	words := strings.Fields(rest)
	for i, w := range words {
		if appositiveStops[strings.ToLower(w)] {
			words = words[:i]
			break
		}
	}
	if len(words) == 0 || len(words) > 3 {
		return false
	}
	return appositiveNouns[strings.ToLower(words[len(words)-1])]
}

var platformNames = wordSet(`darwin`)

var platformLabels = wordSet(`platform platforms os version kernel system sysname uname goos target arch host`)

var platformTokens = wordSet(`linux win32 win64 windows arm64 amd64 x64 x86 aarch64 freebsd openbsd netbsd macos ios android unix posix wasm`)

func platformUse(text string, s, e int) bool {
	if !platformNames[strings.ToLower(text[s:e])] {
		return false
	}
	for _, w := range wordsBefore(text, s, 2) {
		if platformLabels[w] {
			return true
		}
	}
	if e+1 < len(text) && strings.IndexByte(" /-_", text[e]) >= 0 && text[e+1] >= '0' && text[e+1] <= '9' {
		return true
	}
	if platformTokens[strings.ToLower(wordBefore(text, s))] || platformTokens[wordAfter(text, e)] || platformTokens[wordAfterSep(text, e)] {
		return true
	}
	if w := wordsAfter(text, e, 2); len(w) == 2 && (w[0] == "and" || w[0] == "or") && platformTokens[w[1]] {
		return true
	}
	w := wordsBefore(text, s, 2)
	return len(w) == 2 && (w[0] == "and" || w[0] == "or") && platformTokens[w[1]]
}

const platformWordMax = 16

func wordAfterSep(text string, e int) string {
	if e >= len(text) || strings.IndexByte("/-_,", text[e]) < 0 {
		return ""
	}
	k := e + 1
	for k < len(text) && k-e < platformWordMax && (asciiLetter(text[k]) || (text[k] >= '0' && text[k] <= '9') || text[k] == '_') {
		k++
	}
	return strings.ToLower(text[e+1 : k])
}

func thingAt(text string, toks []nameTok, i int) bool {
	t := toks[i]
	if platformUse(text, t.start, t.end) {
		return true
	}
	if !titleOrUpper(t) && t.shape != shapeOther {
		return phraseCovers(text, toks, i)
	}
	if phraseCovers(text, toks, i) {
		return true
	}
	lw := strings.ToLower(t.text)
	if i > 0 {
		p := toks[i-1]
		plw := strings.ToLower(p.text)
		gap := text[p.end:t.start]
		if capitalized(p.text) && placeLeads[plw] && leadGap(plw, gap) {
			return true
		}
		if gap == " " && ((capitalized(p.text) && thingLeads[plw]) || stormLeads[plw]) {
			return true
		}
		if gap == " " && thingNouns[lw] && p.shape == t.shape && KnownName(p.text) && !stopCap[plw] && !pairPersonEvidence(text, p.start, t.end, p.text) {
			return true
		}
	}
	if i+1 < len(toks) {
		nx := toks[i+1]
		nlw := strings.ToLower(nx.text)
		gap := text[t.end:nx.start]
		if placeLeads[lw] && capitalized(nx.text) && leadGap(lw, gap) {
			return true
		}
		if gap == " " && capitalized(nx.text) && thingNouns[nlw] && utf8.RuneCountInString(nx.text) > 1 {
			if _, sur := namedata.LookupSurname(nx.text); sur && nx.shape == t.shape && pairPersonEvidence(text, t.start, nx.end, t.text) {
				return false
			}
			return true
		}
		if gap == " " && nx.shape == shapeLower {
			if eponymNouns[nlw] {
				return true
			}
			if eponymNounsAfterArticle[nlw] {
				if w := wordsBefore(text, t.start, 1); len(w) > 0 && articles[w[0]] {
					return true
				}
			}
		}
		if (gap == "'s " || gap == "’s " || gap == "' " || gap == "’ ") && eponymPossessed[nlw] {
			return true
		}
	}
	return regionCodeAfter(text, t.start, t.end) || appositiveThing(text, t.end)
}

func NamedThingUse(text string, start, end int) bool {
	if start < 0 || end > len(text) || start >= end {
		return false
	}
	ls, le := lineBounds(text, start, end)
	if start-ls > 96 {
		ls = start - 96
		for ls < start && !utf8.RuneStart(text[ls]) {
			ls++
		}
	}
	if le-end > 96 {
		le = end + 96
		for le > end && !utf8.RuneStart(text[le]) {
			le--
		}
	}
	var toks []nameTok
	at := -1
	for _, m := range wordRE.FindAllStringIndex(text[ls:le], -1) {
		s, e := ls+m[0], ls+m[1]
		w := text[s:e]
		if s <= start && end <= e {
			at = len(toks)
		}
		toks = append(toks, nameTok{s, e, w, shapeOf(w)})
	}
	if at < 0 {
		return false
	}
	return thingAt(text, toks, at)
}
