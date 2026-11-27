package detect

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/namedata"
	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/entity"
)

var listLabelRE = regexp.MustCompile(`(?im)^[ \t>*#\-]*(?:[A-Za-z][A-Za-z/\-]*[ \t]){0,2}?(present|attendees?|attending|in attendance|participants?|cc|bcc|to|team|invited|invitees?|apologies|absent|regrets|members|guests|speakers|panell?ists|signatories|authors|reviewers|recipients|contributors|on the call)\*{0,2}[ \t]*:[ \t]*\*{0,2}`)

var strongListLabels = wordSet(`present attendees attendee attending attendance participants participant cc bcc invited invitees invitee apologies absent regrets panelists panellists signatories call`)

var listLabelWords = wordSet(`present attendees attendee attending participants participant cc bcc to from invited invitees apologies absent regrets members guests speakers panelists panellists signatories authors reviewers recipients contributors owner lead assignee reviewer author manager contact host organizer patient client customer employee name driver tenant landlord applicant supervisor physician doctor witness recipient sender attn attention`)

var honorificWords = wordSet(`mr mrs ms mx dr prof professor rev sir madam miss cllr senor senora senorita sr sra srta monsieur madame mademoiselle mme mlle herr frau signor signora signore signorina senhor senhora`)

var foreignHonorifics = wordSet(`senor senora senorita sr sra srta monsieur madame mademoiselle mme mlle herr frau signor signora signore signorina senhor senhora`)

var givenHonorifics = wordSet(`don dona`)

var suffixHonorifics = wordSet(`sr dr`)

var listNonPeople = wordSet(`design product research management board staff everyone committee council others guests members vendor vendors client clients customer customers contractor contractors partner partners chair secretary treasurer none nobody quorum`)

var surnameLeadOK = wordSet(`regarding about per see for with from to cc and or`)

var (
	elemInitialRE = regexp.MustCompile(`^(?:\p{Lu}\.){1,3}$`)
	elemNameRE    = regexp.MustCompile(`^` + nameWord + `$`)
	elemParenRE   = regexp.MustCompile(`[ \t]*(?:\([^()]*\)|<[^<>]*>|\[[^\[\]]*\])[ \t]*$`)
	elemSentRE    = regexp.MustCompile(`\p{L}{2,}\.[ \t]`)
)

type listElem struct {
	start, end int
	multi      bool
	strong     bool
	emit       bool
	shaped     bool
	known      bool
	given      bool
	initialism bool
}

type nameList struct {
	start, end int
	strong     bool
	commas     bool
	elems      []listElem
}

func splitList(seg string) (parts [][2]int, commas bool) {
	sep := byte(',')
	if strings.IndexByte(seg, ';') >= 0 {
		sep = ';'
	}
	commas = sep == ','
	start := 0
	flush := func(end int) {
		off := start
		nextAnd, nextAmp := -2, -2
		for {
			if nextAnd != -1 && nextAnd < off {
				if nextAnd = strings.Index(seg[off:end], " and "); nextAnd >= 0 {
					nextAnd += off
				}
			}
			if nextAmp != -1 && nextAmp < off {
				if nextAmp = strings.Index(seg[off:end], " & "); nextAmp >= 0 {
					nextAmp += off
				}
			}
			j, w := nextAnd, 5
			if j < 0 || (nextAmp >= 0 && nextAmp < j) {
				j, w = nextAmp, 3
			}
			if j < 0 {
				break
			}
			parts = append(parts, [2]int{off, j})
			off = j + w
		}
		parts = append(parts, [2]int{off, end})
	}
	for i := 0; i < len(seg); i++ {
		if seg[i] == sep {
			flush(i)
			start = i + 1
		}
	}
	flush(len(seg))
	return parts, commas
}

func trimElem(text string, s, e int) (int, int) {
	for s < e && (text[s] == ' ' || text[s] == '\t' || text[s] == '*') {
		s++
	}
	if strings.HasPrefix(text[s:e], "and ") {
		s += 4
	}
	for strips := 0; ; strips++ {
		for e > s && (text[e-1] == ' ' || text[e-1] == '\t' || text[e-1] == '*' || text[e-1] == '\r') {
			e--
		}
		if strips == 4 {
			break
		}
		ts := s
		if w := windowStart(text, e); w > ts {
			ts = w
		}
		if m := elemParenRE.FindStringIndex(text[ts:e]); m != nil && ts+m[0] > s {
			e = ts + m[0]
			continue
		}
		break
	}
	return s, e
}

func classifyElem(text string, s, e int, comps []companyMatch, commas bool) (listElem, bool) {
	el := listElem{start: s, end: e}
	body := text[s:e]
	if !commas {
		body = strings.Replace(body, ", ", "  ", 1)
	}
	if strings.HasSuffix(body, ".") && len(body) > 2 && !elemInitialRE.MatchString(body[strings.LastIndexByte(body, ' ')+1:]) {
		body = body[:len(body)-1]
		el.end--
	}
	if body == "" || strings.ContainsAny(body, "\t") {
		return el, false
	}
	var parts []string
	if commas {
		parts = strings.Split(body, " ")
	} else {
		parts = strings.Fields(body)
	}
	if len(parts) > 6 {
		return el, false
	}
	names, initials, nameish, uncommon, unknown := 0, 0, 0, 0, 0
	last := ""
	for i, p := range parts {
		switch {
		case p == "":
			return el, false
		case elemInitialRE.MatchString(p):
			initials++
		case nameParticles[p]:
			if i == len(parts)-1 {
				return el, false
			}
		case elemNameRE.MatchString(p):
			lw := strings.ToLower(p)
			if textutil.IsAllUpper(p) && utf8.RuneCountInString(p) <= 3 {
				if len(parts) == 1 && utf8.RuneCountInString(p) >= 2 {
					el.initialism = true
					return el, true
				}
				return el, false
			}
			if stopCap[lw] || techTerms[lw] || IsMonthOrWeekday(p) {
				return el, false
			}
			names++
			last = p
			if g, given := givenOf(p); (given && !g.Weak) || strongSurname(p) {
				nameish++
			}
			if !textutil.IsCommonWord(p) {
				uncommon++
			}
			if !KnownName(p) {
				unknown++
			}
		default:
			return el, false
		}
	}
	if names == 0 || names+initials > 5 || inCompany(comps, s, el.end) {
		return el, false
	}
	if llw := strings.ToLower(last); (notSurname[llw] || notSurnameExtra[llw] || thingNouns[llw]) && !(names > 1 && knownSurname(last)) {
		return el, false
	}
	el.multi = names+initials >= 2
	el.known = unknown == 0
	if el.multi {
		el.shaped = uncommon > 0 || initials > 0
		el.strong = el.shaped && (nameish > 0 || initials > 0)
		return el, true
	}
	lw := strings.ToLower(last)
	_, isState := StateName(last)
	_, given := givenOf(last)
	el.given = given
	el.shaped = shapeOf(last) == shapeTitle && utf8.RuneCountInString(last) >= 3 && !placeLike[lw] && !isState && !listNonPeople[lw] && (given || !textutil.IsCommonWord(last))
	return el, true
}

func nameLists(text string, comps []companyMatch) []nameList {
	var out []nameList
	for _, m := range listLabelRE.FindAllStringSubmatchIndex(text, -1) {
		labelWords := strings.Fields(strings.ToLower(text[m[2]:m[3]]))
		label := labelWords[len(labelWords)-1]
		pos := m[1]
		le := strings.IndexByte(text[pos:], '\n')
		if le < 0 {
			le = len(text) - pos
		}
		seg := text[pos : pos+le]
		parts, commas := splitList(seg)
		nl := nameList{start: pos, end: pos + le, strong: strongListLabels[label], commas: commas}
		valid, strong, multi := 0, 0, 0
		for pi, p := range parts {
			s, e := trimElem(text, pos+p[0], pos+p[1])
			if s >= e {
				continue
			}
			el, ok := classifyElem(text, s, e, comps, commas)
			if !ok && pi == len(parts)-1 {
				if cut := elemSentRE.FindStringIndex(text[s:e]); cut != nil {
					s2, e2 := trimElem(text, s, s+cut[1]-2)
					el, ok = classifyElem(text, s2, e2, comps, commas)
				}
			}
			if !ok {
				continue
			}
			nl.elems = append(nl.elems, el)
			if el.shaped || el.initialism {
				valid++
			}
			if el.strong {
				strong++
			}
			if el.multi {
				multi++
			}
		}
		qualifies := strong >= 1 && (nl.strong || strong >= 2 || (valid >= 2 && valid == len(parts)))
		if !qualifies {
			continue
		}
		if multi == 0 {
			nl.commas = false
		}
		dense := strong*2 >= multi
		for i := range nl.elems {
			el := &nl.elems[i]
			switch {
			case el.multi:
				el.emit = el.strong || (el.shaped && dense && (nl.strong || el.known))
			case el.shaped:
				el.emit = dense && (el.given || nl.strong)
			}
		}
		out = append(out, nl)
	}
	return out
}

var bulletNameRE = regexp.MustCompile(`(?m)^[ \t]*([-*]|\d{1,3}[.)])[ \t]+(` + nameWord + `(?: ` + nameWord + `)?)[ \t\r]*$`)

func bulletStyle(text string, m []int) byte {
	return text[m[3]-1]
}

func bulletNameSpans(text string, comps []companyMatch) []entity.Span {
	var out []entity.Span
	ms := bulletNameRE.FindAllStringSubmatchIndex(text, -1)
	for i := 0; i < len(ms); {
		j := i + 1
		for j < len(ms) && ms[j][0] == ms[j-1][1]+1 && bulletStyle(text, ms[j]) == bulletStyle(text, ms[i]) {
			j++
		}
		if j-i >= 2 && bulletPeople(text, ms[i:j], comps, bulletLabelled(text, ms[i][0])) {
			for _, m := range ms[i:j] {
				out = append(out, entity.Span{Start: m[4], End: m[5], Type: entity.Person, Label: "person", Text: text[m[4]:m[5]], Source: "names:list", Priority: 27, Score: 1})
			}
		}
		i = j
	}
	return out
}

func bulletLabelled(text string, s int) bool {
	if s == 0 || text[s-1] != '\n' {
		return false
	}
	e := s - 1
	if e > 0 && text[e-1] == '\r' {
		e--
	}
	ls := windowStart(text, e)
	if k := strings.LastIndexByte(text[ls:e], '\n'); k >= 0 {
		ls += k + 1
	} else if ls > 0 {
		return false
	}
	line := strings.TrimRight(text[ls:e], " \t")
	m := listLabelRE.FindStringIndex(line)
	return m != nil && m[1] == len(line)
}

func bulletPeople(text string, items [][]int, comps []companyMatch, labelled bool) bool {
	strong := labelled
	for _, m := range items {
		if w := text[m[4]:m[5]]; labelled && strings.IndexByte(w, ' ') < 0 {
			if _, given := givenOf(w); !given || shapeOf(w) != shapeTitle || inCompany(comps, m[4], m[5]) {
				return false
			}
			continue
		}
		el, ok := classifyElem(text, m[4], m[5], comps, false)
		if !ok || el.initialism || el.end != m[5] {
			return false
		}
		if el.multi {
			strong = strong || el.strong
			continue
		}
		w := text[m[4]:m[5]]
		g, given := givenOf(w)
		if !given {
			return false
		}
		if !g.Weak && !textutil.IsCommonWord(w) {
			strong = true
		}
	}
	return strong
}

func listSpans(text string, lists []nameList) []entity.Span {
	var out []entity.Span
	for _, nl := range lists {
		for _, el := range nl.elems {
			if !el.emit || embedded(text, el.start, el.end) {
				continue
			}
			out = append(out, entity.Span{Start: el.start, End: el.end, Type: entity.Person, Label: "person", Text: text[el.start:el.end], Source: "names:list", Priority: 27, Score: 1})
		}
	}
	return out
}

func inCommaList(lists []nameList, pos int) bool {
	i := sort.Search(len(lists), func(k int) bool { return lists[k].end > pos })
	return i < len(lists) && lists[i].commas && lists[i].start <= pos
}

func honorificNames(text string, toks []nameTok) ([]entity.Span, map[int]bool) {
	var out []entity.Span
	var claimed map[int]bool
	for i := 0; i+1 < len(toks); i++ {
		t := toks[i]
		key := textutil.Fold(t.text)
		givenLed := givenHonorifics[key] && t.shape == shapeLower
		switch {
		case givenLed:
		case foreignHonorifics[key] && t.shape != shapeOther:
		case honorificWords[key] && t.shape == shapeTitle:
		default:
			continue
		}
		if embedded(text, t.start, t.end) && !strings.HasPrefix(text[t.end:], ". ") {
			continue
		}
		if i > 0 && suffixHonorifics[key] && titleOrUpper(toks[i-1]) && oneSpace(text, toks[i-1].end, t.start) && (KnownName(toks[i-1].text) || (key == "dr" && !stopCap[strings.ToLower(toks[i-1].text)])) {
			continue
		}
		if gap := text[t.end:toks[i+1].start]; gap != " " && gap != ". " {
			continue
		}
		j, names, end := i+1, 0, -1
		lastJ := i
		lowerLed := t.shape == shapeLower && !givenLed && !suffixHonorifics[key]
		if w := wordsBefore(text, t.start, 1); lowerLed && len(w) > 0 && honorificArticles[w[0]] {
			lowerLed = false
		}
		lowerGiven := false
		for j < len(toks) && names < 4 {
			w := toks[j]
			if j > i+1 && !oneSpace(text, toks[j-1].end, w.start) {
				break
			}
			if w.shape == shapeLower && nameParticles[w.text] {
				j++
				continue
			}
			if w.shape == shapeLower && lowerLed {
				given := lowerHonorificGiven(w.text)
				if embedded(text, w.start, w.end) || honorificVerbs[textutil.Fold(w.text)] || !(names == 0 && (given || strongSurname(w.text)) || names == 1 && lowerGiven && strongSurname(w.text)) {
					break
				}
				lowerGiven = names == 0 && given
				names++
				end, lastJ = w.end, j
				j++
				continue
			}
			lw := strings.ToLower(w.text)
			if w.shape != shapeTitle || !surnameLike(w.text) || embedded(text, w.start, w.end) && !strings.HasPrefix(text[w.end:], ".") {
				break
			}
			known := KnownName(w.text)
			common := textutil.IsCommonWord(w.text)
			if honorificWords[textutil.Fold(lw)] || (common && !known) || (names >= 2 && (!known || common)) {
				break
			}
			if names == 1 && !known && continuesClause(text, w.end) {
				break
			}
			if givenLed && names == 0 && !known {
				break
			}
			names++
			end, lastJ = w.end, j
			j++
		}
		if names == 0 {
			continue
		}
		first := toks[i+1]
		src := "names:honorific"
		if givenLed {
			src = "names:don"
		}
		out = append(out, entity.Span{Start: first.start, End: end, Type: entity.Person, Label: "person", Text: text[first.start:end], Source: src, Priority: 30, Score: 1})
		if claimed == nil {
			claimed = map[int]bool{}
		}
		for k := i; k <= lastJ; k++ {
			claimed[k] = true
		}
		i = lastJ
	}
	return out, claimed
}

var honorificArticles = wordSet(`the a an this that die der den dem des eine einer einem einen diese dieser meine seine ihre unsere la el una un le les une esta esa mi su il lo uma o`)

var honorificVerbs = wordSet(`rief ruft sagte sagt kam ging hat hatte ist war wird will kann soll fragte meinte schrieb dijo llamo dice vino est dit avait`)

func lowerHonorificGiven(w string) bool {
	g, ok := givenOf(w)
	return ok && !g.Weak && !textutil.IsCommonWord(w) && utf8.RuneCountInString(w) >= 4
}

func titleName(w string) string {
	rs := []rune(strings.ToLower(w))
	up := true
	for i, r := range rs {
		if up && unicode.IsLetter(r) {
			rs[i] = unicode.ToUpper(r)
		}
		up = textutil.IsNameJoiner(r) || r == '\'' || r == '’'
	}
	return string(rs)
}

var capsLabelWords = wordSet(`present attendees attendee attending participants participant cc bcc to from invited invitees apologies absent regrets members guests speakers panelists panellists signatories authors reviewers recipients contributors owner lead assignee reviewer author manager contact host organizer patient client customer employee name driver tenant landlord applicant supervisor physician doctor witness recipient sender attn attention buyer seller provider beneficiary insured claimant signed invoice bill ship sold deliver delivered subject dear surname passenger traveler traveller guest member student resident defendant plaintiff account holder cardholder payee payer party candidate subscriber user staff officer agent nurse`)

func capsFamilyTok(text string, w nameTok) bool {
	lw := strings.ToLower(w.text)
	return w.shape == shapeUpper && utf8.RuneCountInString(w.text) >= 3 && knownSurname(w.text) && !stopCap[lw] && !techTerms[lw] && !capsLabelWords[lw] && !IsMonthOrWeekday(w.text) && !embedded(text, w.start, w.end)
}

func firmSurname(w string) bool {
	if parts := textutil.NameJoinerFields(w); len(parts) > 1 {
		for _, p := range parts {
			if firmSurname(p) {
				return true
			}
		}
		return false
	}
	weak, ok := namedata.LookupSurname(w)
	return ok && !weak && utf8.RuneCountInString(w) >= 4
}

func capsParticle(w nameTok) bool {
	return (w.shape == shapeUpper || w.shape == shapeLower) && nameParticles[strings.ToLower(w.text)]
}

func capsSurnameFirst(text string, toks []nameTok, i int) (end, n int, canonical string) {
	t := toks[i]
	if t.shape != shapeUpper {
		return 0, 0, ""
	}
	trimmed := false
	if i > 0 && oneSpace(text, toks[i-1].end, t.start) && toks[i-1].shape == shapeUpper {
		if capsFamilyTok(text, toks[i-1]) || nameParticles[strings.ToLower(toks[i-1].text)] {
			return 0, 0, ""
		}
		plw := strings.ToLower(toks[i-1].text)
		trimmed = stopCap[plw] || capsLabelWords[plw] || textutil.IsCommonWord(plw)
	}
	j, real, strong := i, 0, false
	var family []string
	for j < len(toks) && j-i < 5 {
		w := toks[j]
		if j > i && !oneSpace(text, toks[j-1].end, w.start) {
			break
		}
		if capsParticle(w) {
			family = append(family, strings.ToLower(w.text))
			j++
			continue
		}
		if w.shape != shapeUpper || embedded(text, w.start, w.end) {
			break
		}
		if !capsFamilyTok(text, w) {
			return 0, 0, ""
		}
		real++
		strong = strong || firmSurname(w.text)
		family = append(family, titleName(w.text))
		j++
	}
	if real == 0 || j >= len(toks) || (trimmed && !strong && real < 2) {
		return 0, 0, ""
	}
	lastFam := toks[j-1]
	if nameParticles[strings.ToLower(lastFam.text)] || text[lastFam.end:toks[j].start] != ", " {
		return 0, 0, ""
	}
	if real == 1 && len(family) == 1 && (utf8.RuneCountInString(t.text) < 4 || textutil.IsCommonWord(t.text) || placeLike[strings.ToLower(t.text)]) {
		return 0, 0, ""
	}
	var given []string
	k := j
	for k < len(toks) && k-j < 3 {
		w := toks[k]
		if w.shape != shapeTitle || (k > j && !oneSpace(text, toks[k-1].end, w.start)) || embedded(text, w.start, w.end) {
			break
		}
		lw := strings.ToLower(w.text)
		if _, ok := namedata.LookupGiven(w.text); !ok || stopCap[lw] || techTerms[lw] || textutil.IsCommonWord(w.text) || IsMonthOrWeekday(w.text) {
			break
		}
		given = append(given, w.text)
		k++
	}
	if len(given) == 0 {
		return 0, 0, ""
	}
	return toks[k-1].end, k - 1 - i, strings.Join(given, " ") + " " + strings.Join(family, " ")
}

var commonInitialisms = wordSet(`OK US UK EU IT HR PR QA AI ML PM AM ID TV PC OS DB CI CD UI UX CEO CTO CFO COO CIO CMO VP SVP EVP GM PA PS RE CC FYI TBD TBA ETA EOD ASAP FAQ API SDK URL PDF CSV SQL AWS GCP IBM USA UAE UN NA NB PO HQ RD AC DJ MC DR MR MS JR SR ST PT GP ER ICU HIV DNA RNA GDP KPI ROI SLA NDA RFP POC MVP LLC INC LTD PLC AGM EGM AOB TBC NYC LAX ATM PIN VAT GST IRS FBI CIA NSA DOJ DOD NHS BBC CNN NBA NFL MLB NHL UFC WWE GPS USB CPU GPU RAM ROM SSD HDD LAN WAN VPN DNS TCP UDP IP HTTP SSH SSL TLS XML HTML CSS PHP JS TS GO OP BTW IMO LOL OMG WTF IDK AKA DIY RIP VIP MIA POW AWOL ET PST EST CST MST GMT UTC BST CET`)

var isoCountryCodes = wordSet(`AD AE AF AG AI AL AM AO AQ AR AS AT AU AW AX AZ BA BB BD BE BF BG BH BI BJ BL BM BN BO BQ BR BS BT BV BW BY BZ CA CC CD CF CG CH CI CK CL CM CN CO CR CU CV CW CX CY CZ DE DJ DK DM DO DZ EC EE EG EH ER ES ET FI FJ FK FM FO FR GA GB GD GE GF GG GH GI GL GM GN GP GQ GR GS GT GU GW GY HK HM HN HR HT HU ID IE IL IM IN IO IQ IR IS IT JE JM JO JP KE KG KH KI KM KN KP KR KW KY KZ LA LB LC LI LK LR LS LT LU LV LY MA MC MD ME MF MG MH MK ML MM MN MO MP MQ MR MS MT MU MV MW MX MY MZ NA NC NE NF NG NI NL NO NP NR NU NZ OM PA PE PF PG PH PK PL PM PN PR PS PT PW PY QA RE RO RS RU RW SA SB SC SD SE SG SH SI SJ SK SL SM SN SO SR SS ST SV SX SY SZ TC TD TF TG TH TJ TK TL TM TN TO TR TT TV TW TZ UA UG UM US UY UZ VA VC VE VG VI VN VU WF WS YE YT ZA ZM ZW`)

var caProvinces = wordSet(`AB BC MB NB NL NS NT NU ON PE QC SK YT`)

var twoLetterWords = wordSet(`ah am an as at be by do go ha he hi if in is it me my no of oh ok on or ox so to up us we ye yo ad ex id pa ma mu nu pi xi la re ti fa mi ab ag ai al ar aw ax ay ba bi bo ed ef eh el em en er es et gi hm ho jo ka ki li lo mm mo na ne od oe oi om op os ow oy pe po qi sh ta te uh um un ut wo xu ya za`)

func CommonInitialism(w string) bool {
	up := strings.ToUpper(w)
	if commonInitialisms[up] || isoCountryCodes[up] || caProvinces[up] {
		return true
	}
	if _, ok := usStates[up]; ok {
		return true
	}
	lw := strings.ToLower(w)
	return twoLetterWords[lw] || textutil.IsCommonWord(lw)
}

func NameInitials(name string) []string {
	var all, core []rune
	for _, m := range wordRE.FindAllString(name, -1) {
		r, _ := utf8.DecodeRuneInString(textutil.Fold(m))
		if !unicode.IsLetter(r) {
			return nil
		}
		r = unicode.ToUpper(r)
		all = append(all, r)
		if !nameParticles[strings.ToLower(m)] {
			core = append(core, r)
		}
	}
	var out []string
	for _, rs := range [][]rune{all, core} {
		if len(rs) >= 2 && len(rs) <= 3 {
			if s := string(rs); len(out) == 0 || out[0] != s {
				out = append(out, s)
			}
		}
	}
	return out
}

func IsInitialismOf(text, name string) bool {
	if n := utf8.RuneCountInString(text); n < 2 || n > 3 || !textutil.IsAllUpper(text) {
		return false
	}
	for _, in := range NameInitials(name) {
		if strings.EqualFold(in, textutil.Fold(text)) {
			return true
		}
	}
	return false
}

var initialismLineStartRE = regexp.MustCompile(`^[ \t>*\-•]*(?:\d{1,3}[.)][ \t]+)?$`)

var initialismBefore = wordSet(`by from owner owners assignee assigned reviewer lead cc per`)

func initialismSlots(lists []nameList) map[int]int {
	slots := map[int]int{}
	for _, nl := range lists {
		for _, el := range nl.elems {
			if el.initialism {
				slots[el.start] = el.end
			}
		}
	}
	return slots
}

func initialismPosition(text string, s, e int, slots func() map[int]int) (ok, firm bool) {
	if embedded(text, s, e) && !(s > 0 && text[s-1] == '(') {
		return false, false
	}
	ls, le := lineBounds(text, s, e)
	rest := text[e:le]
	if speakerColon(rest) && initialismLineStartRE.MatchString(text[ls:s]) {
		return true, true
	}
	if end, listed := slots()[s]; listed && end == e {
		return true, true
	}
	if s > 0 && text[s-1] == '(' && strings.HasPrefix(rest, ")") {
		return true, false
	}
	prev := wordsBefore(text, s, 1)
	if len(prev) > 0 && (articles[prev[0]] || determiners[prev[0]]) {
		return false, false
	}
	if len(prev) > 0 && initialismBefore[prev[0]] {
		return true, false
	}
	after := wordsAfter(text, e, 2)
	if len(after) == 0 {
		return false, false
	}
	a := after[0]
	lowerNext := e+1 < len(text) && text[e+1] >= 'a' && text[e+1] <= 'z'
	switch {
	case !lowerNext:
		return false, false
	case a == "to" && len(after) > 1 && textutil.IsAllLower(after[1]):
		return true, false
	case personVerbAfter(text, e) || lowerNameAfter[a] || personModals[a]:
		return true, false
	case len(a) >= 5 && strings.HasSuffix(a, "ed"):
		return true, false
	}
	return false, false
}

func initialismSpans(text string, people []string, lists func() []nameList, source string, prio int) []entity.Span {
	if len(people) == 0 {
		return nil
	}
	byInitials := map[string]string{}
	for _, p := range people {
		for _, in := range NameInitials(p) {
			if _, ok := byInitials[in]; !ok {
				byInitials[in] = p
			}
		}
	}
	if len(byInitials) == 0 {
		return nil
	}
	var out []entity.Span
	var slotIndex map[int]int
	slots := func() map[int]int {
		if slotIndex == nil {
			slotIndex = initialismSlots(lists())
		}
		return slotIndex
	}
	for _, m := range wordRE.FindAllStringIndex(text, -1) {
		if m[1]-m[0] > 12 {
			continue
		}
		w := text[m[0]:m[1]]
		if n := utf8.RuneCountInString(w); n < 2 || n > 3 || !textutil.IsAllUpper(w) {
			continue
		}
		name, ok := byInitials[strings.ToUpper(textutil.Fold(w))]
		if !ok {
			continue
		}
		ok, firm := initialismPosition(text, m[0], m[1], slots)
		if !ok || (!firm && CommonInitialism(w)) {
			continue
		}
		out = append(out, entity.Span{Start: m[0], End: m[1], Type: entity.Person, Label: "person", Text: w, Source: source, Priority: prio, Score: 1, Canonical: name})
	}
	return out
}

var initialismSources = map[string]bool{"names:full": true, "names:label": true, "names:list": true, "names:email": true, "names:initial": true, "names:family-first": true, "names:honorific": true, "names:surname-first": true, "names:company-person": true, "names:signoff": true}

func initialismPeople(spans []entity.Span) []string {
	var out []string
	for _, sp := range spans {
		if sp.Type != entity.Person || !initialismSources[sp.Source] {
			continue
		}
		name := sp.Text
		if sp.Canonical != "" {
			name = sp.Canonical
		}
		if strings.ContainsAny(name, ",/\n") || textutil.IsAllLower(name) || len(wordRE.FindAllString(name, -1)) < 2 {
			continue
		}
		out = append(out, name)
	}
	return out
}
