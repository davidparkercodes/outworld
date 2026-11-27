package detect

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/namedata"
	"github.com/davidparkercodes/outworld/pkg/entity"
)

type addrTailKind int

const (
	addrTailEU addrTailKind = iota
	addrTailEURequired
	addrTailUK
	addrTailUKPostcode
	addrTailUKCountry
)

type addrCore struct {
	re          *regexp.Regexp
	tail        addrTailKind
	numberFirst bool
	ok          func(text string, s, e int) bool
}

const (
	addrRomanceStrong = `calle|carrer|avenida|avda\.?|avinguda|paseo|passeig|rua|travessa|viale|piazza|ronda|camino|carretera|rambla`
	addrRomanceWeak   = `via|plaza|plaça|corso|largo|vicolo|praça|praca|piazzale|alameda|glorieta|estrada`
	addrRomanceFirst  = `calle|carrer|avenida|avda\.?|avinguda|passeig|rua|travessa|viale|carretera`
	addrRomanceSecond = `via|plaza|plaça|corso|largo|vicolo|praça|praca|piazzale|alameda|glorieta|estrada|paseo|piazza|ronda|camino|rambla`
	addrAttached      = `straße|strasse|str\.|gasse|allee|gracht|straat|laan|plein|singel|weg|platz|dijk|steeg|dreef|markt|gatan|gata|gade|vej|vägen|veien`
	addrAttachedSafe  = `straße|strasse|straat|laan|gracht|gasse`
	addrSplitWord     = `Straße|Strasse|STRASSE|Str\.|Allee|ALLEE|Platz|PLATZ|Weg|WEG|Gasse|GASSE|Chaussee|Laan|LAAN|Straat|STRAAT`
	addrHyphenWord    = `Straße|Strasse|STRASSE|Str\.|Allee|Platz|Weg|Gasse|Ring|Damm|Ufer|Chaussee|Brücke|Promenade|Steig|Pfad`
	addrHyphenStreet  = `(?:\p{Lu}[\p{L}.]*-)+(?:` + addrHyphenWord + `)`
	addrTitle         = `(?:Burgemeester|Burg\.|Prof\.|Professor|Dr\.|Doctor|Dokter|Mr\.|Meester|Sint|St\.|Koningin|Koning|Prins|Prinses|Pastoor|Ds\.|Dominee|Generaal|Graaf|President|Minister|Jonkheer|Jhr\.|Ir\.|Ing\.|Kardinaal|Bisschop|Monseigneur|Mgr\.|Pater|Kapitein|Admiraal|Wethouder|Notaris)`
	addrParticle      = `(?:van der|van den|van de|van|de|den|der|ter|ten|te)`
	addrTitled        = addrTitle + ` (?:` + addrWord + ` ){0,2}(?:` + addrParticle + ` )?`
	addrNumber        = `(?:(?i:n[ºo°]\.?|num\.?|número) ?)?\d{1,4}[A-Za-z]?\b(?:[\-/]\d{1,4}[A-Za-z]?\b)?`
	addrUKUnit        = `(?:Flat|Apartment|Apt\.?|Unit|Suite|Room|FLAT|APARTMENT|APT\.?|UNIT|SUITE) [A-Za-z0-9]{1,4},? `
)

var (
	addrUKPostAtRE   = regexp.MustCompile(`^[A-Z]{1,2}\d[A-Z\d]? ?\d[A-Z]{2}`)
	addrUKPostFindRE = regexp.MustCompile(`[A-Z]{1,2}\d[A-Z\d]? \d[A-Z]{2}`)
	addrCueLineRE    = regexp.MustCompile(`(?i)(?:^|[^\p{L}])(?:adresse|anschrift|lieferadresse|postanschrift|address|dirección|direccion|domicilio|indirizzo|morada|endereço|adres|ship to|deliver to|c/o|postfach|postbus|apartado|boîte postale|casella postale|p\.?o\.? box)\b[^\n]{0,40}$`)
	addrStreetLineRE = regexp.MustCompile(`^(?:\p{L}[\p{L}.'’\-]*,? ){0,4}\p{Lu}[\p{L}.'’\-]{2,},? (?:(?i:n[ºo°]\.?) ?)?\d{1,3} ?[a-zA-Z]?$`)
	addrNumLineRE    = regexp.MustCompile(`^\d{1,4}[a-zA-Z]?(?: bis| ter)?,? (?:\p{L}[\p{L}.'’\-]* ){0,4}\p{Lu}[\p{L}.'’\-]*$`)
	addrUSTailRE     = reWithState(`^,?[ \t]*\n?[ \t]*[A-Z][A-Za-z.'\-]+(?: [A-Z][A-Za-z.'\-]+){0,3},?[ \t]*\n?[ \t]*(?:STATES)\b`)
	addrPostcodeCue  = regexp.MustCompile(`(?i)post ?code[ \t]*(?:is|:)?[ \t]*$`)
)

var addrCores = map[string]addrCore{
	"address_fr": {numberFirst: true,
		re: regexp.MustCompile(`\b\d{1,4}(?: ?(?i:bis|ter))?,? (?i:rue|boulevard|bd\.?|chemin|allée|allee|impasse|quai|avenue|av\.) ` + addrName)},
	"address_fr_postal": {numberFirst: true, tail: addrTailEURequired,
		re: regexp.MustCompile(`\b\d{1,4}(?: ?(?i:bis|ter))?,? (?i:place|cours|route|passage|square) ` + addrName)},
	"address_street_first": {ok: addrNotPostalSplit,
		re: regexp.MustCompile(`(?i:` + addrRomanceStrong + `) ` + addrName + `,? (?:` + addrNumber + `|s/n)`)},
	"address_street_first_postal": {tail: addrTailEURequired, ok: addrNotPostalSplit,
		re: regexp.MustCompile(`(?i:` + addrRomanceWeak + `) ` + addrName + `,? (?:` + addrNumber + `|s/n)`)},
	"address_number_first": {numberFirst: true,
		re: regexp.MustCompile(`\b\d{1,4}[a-zA-Z]?,? (?i:` + addrRomanceFirst + `) ` + addrName)},
	"address_number_first_postal": {numberFirst: true, tail: addrTailEURequired,
		re: regexp.MustCompile(`\b\d{1,4}[a-zA-Z]?,? (?i:` + addrRomanceSecond + `) ` + addrName)},
	"address_number_first_germanic": {numberFirst: true,
		re: regexp.MustCompile(`\b\d{1,4}[a-zA-Z]?,? (?:` + addrTitled + `|` + addrWord + ` )?\p{Lu}\p{L}+(?i:` + addrAttachedSafe + `)`)},
	"address_number_first_germanic_postal": {numberFirst: true, tail: addrTailEURequired,
		re: regexp.MustCompile(`\b\d{1,4}[a-zA-Z]?,? (?:(?:` + addrTitled + `|` + addrWord + ` )?\p{Lu}\p{L}+(?i:` + addrAttached + `)|(?:` + addrWord + ` ){1,2}(?:` + addrSplitWord + `)|` + addrHyphenStreet + `)`)},
	"address_germanic": {
		re: regexp.MustCompile(`(?:` + addrTitled + `|` + addrWord + ` )?\p{Lu}\p{L}+(?i:` + addrAttached + `) ` + addrHouse)},
	"address_germanic_split": {ok: addrSplitNamed,
		re: regexp.MustCompile(`(?:` + addrWord + ` ){1,2}(?:` + addrSplitWord + `) ` + addrHouse)},
	"address_germanic_hyphen": {
		re: regexp.MustCompile(addrHyphenStreet + ` ` + addrHouse)},
	"address_germanic_postal": {tail: addrTailEURequired,
		re: regexp.MustCompile(`(?:(?:Am|An der|An den|Im|Zum|Zur|Auf der|Auf dem|Unter den|In der|Op de|Aan de) ` + addrWord + `(?: ` + addrWord + `)?|(?:` + addrWord + ` )?(?:\p{Lu}\p{L}+(?:ring|damm|ufer|kade)|Ring|Damm|Ufer)) ` + addrHouse)},
	"address_uk_flat": {tail: addrTailUK,
		re: regexp.MustCompile(`\b` + addrUKUnit + `\d{1,5}[A-Za-z]? (?:` + addrWord + ` ){1,3}(?:` + streetSuffix + `|` + addrCapsSuffx + `)\b\.?`)},
	"address_uk_postcode": {tail: addrTailUKPostcode,
		re: regexp.MustCompile(`\b(?:` + addrUKUnit + `)?\d{1,5}[A-Za-z]? ` + addrWord)},
	"address_uk_country": {tail: addrTailUKCountry,
		re: regexp.MustCompile(`\b\d{1,5}[A-Za-z]? (?:` + addrWord + ` ){1,3}(?i:` + streetSuffix + `)\b\.?`)},
}

var addressPatterns = []pattern{
	addrPattern("address_fr", gateLower("rue |boulevard |bd |bd. |chemin |allée |allee |impasse |quai |avenue |av. ")),
	addrPattern("address_fr_postal", gateLower("place |cours |route |passage |square ")),
	addrPattern("address_street_first", gateLower("calle |carrer |avenida |avda|avinguda |paseo |passeig |rua |travessa |viale |piazza |ronda |camino |carretera |rambla ")),
	addrPattern("address_street_first_postal", gateLower("via |plaza |plaça |corso |largo |vicolo |praça |praca |piazzale |alameda |glorieta |estrada ")),
	addrPattern("address_number_first", gateLower("calle |carrer |avenida |avda|avinguda |passeig |rua |travessa |viale |carretera ")),
	addrPattern("address_number_first_postal", gateLower("via |plaza |plaça |corso |largo |vicolo |praça |praca |piazzale |alameda |glorieta |estrada |paseo |piazza |ronda |camino |rambla ")),
	addrPattern("address_number_first_germanic", gateLower("straße|strasse|straat|laan|gracht|gasse")),
	addrPattern("address_number_first_germanic_postal", gateBeforeBreak("straße|strasse|str.|gasse|allee|gracht|straat|laan|plein|singel|weg|platz|dijk|steeg|dreef|markt|ring|damm|ufer|chaussee|brücke|promenade|steig|pfad")),
	addrPattern("address_germanic", gateBeforeDigit("straße |strasse |str. |gasse |allee |gracht |straat |laan |plein |singel |weg |platz |dijk |steeg |dreef |markt |gatan |gata |gade |vej |vägen |veien ")),
	addrPattern("address_germanic_split", gateBeforeDigit("straße |strasse |str. |allee |platz |weg |gasse |chaussee |laan |straat ")),
	addrPattern("address_germanic_hyphen", gateBeforeDigit("-straße |-strasse |-str. |-allee |-platz |-weg |-gasse |-ring |-damm |-ufer |-chaussee |-brücke |-promenade |-steig |-pfad ")),
	addrPattern("address_germanic_postal", gateAny(gateExact("Am |An der |An den |Im |Zum |Zur |Auf der |Auf dem |Unter den |In der |Op de |Aan de "), gateBeforeDigit("ring |damm |ufer |kade "))),
	addrPattern("address_uk_flat", gateLower("flat |apartment |apt |apt. |unit |suite |room ")),
	addrPattern("address_uk_postcode", gateUKPostcode),
	addrPattern("address_uk_country", gateExact("UK|U.K.|United Kingdom|England|Scotland|Wales|Northern Ireland|Great Britain|UNITED KINGDOM|ENGLAND|SCOTLAND|WALES")),
	addrSpacedPattern(pattern{name: "address_postal_city", typ: entity.Address, label: "address", priority: 61, scan: addrPostalCityScan}),
	addrSpacedPattern(pattern{name: "address_uk_town", gate: gateUKPostcode, typ: entity.Address, label: "address", priority: 61, scan: addrUKTownScan}),
}

func addrPattern(name string, gate func(string, string) bool) pattern {
	core := addrCores[name]
	return addrSpacedPattern(pattern{name: name, gate: gate, typ: entity.Address, label: "address", priority: 61,
		scan: func(text, _ string) []entity.Span { return core.scan(text) }})
}

func addrOddSpacing(text string) bool {
	return strings.Contains(text, "  ") || strings.IndexByte(text, '\t') >= 0 || strings.Contains(text, " ,")
}

func addrNormalize(text string) (string, []int) {
	var b strings.Builder
	idx := make([]int, 0, len(text))
	for i := 0; i < len(text); {
		c := text[i]
		if c != ' ' && c != '\t' {
			b.WriteByte(c)
			idx = append(idx, i)
			i++
			continue
		}
		j := i
		for j < len(text) && (text[j] == ' ' || text[j] == '\t') {
			j++
		}
		if j < len(text) && text[j] == ',' {
			i = j
			continue
		}
		b.WriteByte(' ')
		idx = append(idx, i)
		i = j
	}
	return b.String(), idx
}

func addrSpacedPattern(p pattern) pattern {
	gate, scan := p.gate, p.scan
	p.gate = func(text, lower string) bool {
		return gate == nil || gate(text, lower) || addrOddSpacing(text)
	}
	p.scan = func(text, lower string) []entity.Span {
		var out []entity.Span
		if gate == nil || gate(text, lower) {
			out = scan(text, lower)
		}
		if !addrOddSpacing(text) {
			return out
		}
		norm, idx := addrNormalize(text)
		nl := strings.ToLower(norm)
		if gate != nil && !gate(norm, nl) {
			return out
		}
		for _, sp := range scan(norm, nl) {
			if sp.Start < 0 || sp.End > len(norm) || sp.Start >= sp.End {
				continue
			}
			s, e := idx[sp.Start], idx[sp.End-1]+1
			if text[s:e] == norm[sp.Start:sp.End] {
				continue
			}
			sp.Start, sp.End = s, e
			out = append(out, sp)
		}
		return out
	}
	return p
}

func gateBeforeBreak(words string) func(string, string) bool {
	list := strings.Split(words, "|")
	return func(_, lower string) bool {
		for _, w := range list {
			for from := 0; ; {
				i := strings.Index(lower[from:], w)
				if i < 0 {
					break
				}
				from += i + len(w)
				if from >= len(lower) {
					break
				}
				switch c := lower[from]; {
				case c == ',' || c == '\n' || c == '\r':
					return true
				case c == ' ' && from+4 < len(lower):
					next := lower[from+1 : from+5]
					if isDigitByte(next[0]) || next[1] == '-' && isDigitByte(next[2]) || next[2] == '-' && isDigitByte(next[3]) {
						return true
					}
				}
			}
		}
		return false
	}
}

func gateUKPostcode(text, _ string) bool {
	for i := 2; i+2 < len(text); i++ {
		c := text[i]
		if c < '0' || c > '9' {
			continue
		}
		if a, b := text[i+1], text[i+2]; a < 'A' || a > 'Z' || b < 'A' || b > 'Z' {
			continue
		}
		p := text[i-1]
		if p == ' ' {
			p = text[i-2]
		}
		if isDigitByte(p) || p >= 'A' && p <= 'Z' {
			return true
		}
	}
	return false
}

func (c addrCore) scan(text string) []entity.Span {
	var out []entity.Span
	for pos := 0; pos < len(text); {
		m := c.re.FindStringIndex(text[pos:])
		if m == nil {
			break
		}
		s, e := pos+m[0], pos+m[1]
		start, end, next, ok := c.accept(text, s, e)
		if ok {
			out = append(out, entity.Span{Start: start, End: end})
		}
		pos = max(next, s+1)
	}
	return out
}

func (c addrCore) accept(text string, s, e int) (start, end, next int, ok bool) {
	start = s
	next = addrRetryAt(text, s, e)
	cut := addrSentenceBreak(text, s, e)
	switch {
	case cut >= 0 && (!c.numberFirst || c.tail != addrTailEU):
		return 0, 0, cut + 2, false
	case cut >= 0:
		end, next = cut, cut+2
	default:
		if !addrCoreEnds(text, e, c.numberFirst) {
			return 0, 0, next, false
		}
		if c.ok != nil && !c.ok(text, s, e) {
			return 0, 0, next, false
		}
		tail, found := addrTail(text, e, c.tail)
		if c.tail == addrTailEU || c.tail == addrTailEURequired {
			if p := addrPostalBefore(text, s); p >= 0 {
				start, found = p, true
			}
		}
		if !found && (c.tail != addrTailEU && c.tail != addrTailUK || c.tail == addrTailUK && addrUSTailRE.MatchString(text[e:min(len(text), e+addrContextWindow)])) {
			return 0, 0, next, false
		}
		end = tail
	}
	for end > s && (text[end-1] == '.' || text[end-1] == ',') {
		end--
	}
	if !boundaryOK(text, start, end) {
		return 0, 0, next, false
	}
	if cut < 0 {
		next = e
	}
	return start, end, next, true
}

const addrPostalBeforeWindow = 64

var (
	addrStatusWords = wordSet(`done worker workers fix fixed fixes merged merge units unit error errors failed failure fail fails passed pass success succeeded ok warning warnings warn info debug trace fatal items item rows row records record total totals build builds job jobs task tasks tests test step steps retry retries bytes files file lines line users events requests results issues commits points tickets orders hits misses calls ms sec secs seconds minutes hours days`)
	addrHouseEndRE  = regexp.MustCompile(`\p{L}[\p{L}.'’\-]* (?:(?i:n[ºo°]\.?) ?)?\d{1,4} ?[a-zA-Z]?$`)
)

func addrPostalBefore(text string, s int) int {
	j := s
	for j > 0 && (text[j-1] == ' ' || text[j-1] == '\t') {
		j--
	}
	if j == 0 || text[j-1] != ',' {
		return -1
	}
	k := j - 1
	for k > 0 && (text[k-1] == ' ' || text[k-1] == '\t') {
		k--
	}
	for i := max(0, k-addrPostalBeforeWindow); i < k; i++ {
		if !isDigitByte(text[i]) && (text[i] < 'A' || text[i] > 'Z') {
			continue
		}
		if i > 0 && (isDigitByte(text[i-1]) || isAlphaByte(text[i-1])) {
			continue
		}
		if strings.IndexByte(text[i:k], '\n') >= 0 {
			continue
		}
		if end, ok := addrPostalCity(text, i); ok && end == k && !addrStatusCity(text, i, k) && addrPostalLeadOK(text, i) {
			return i
		}
	}
	return -1
}

func addrStatusCity(text string, i, k int) bool {
	p := addrPostalAt(text, i)
	if p < 0 {
		return false
	}
	for _, w := range strings.Fields(text[p:k]) {
		if !addrStatusWords[strings.ToLower(strings.Trim(w, ".:;"))] {
			return false
		}
	}
	return true
}

func addrPostalLeadOK(text string, i int) bool {
	j := i
	for j > 0 && (text[j-1] == ' ' || text[j-1] == '\t') {
		j--
	}
	if j == 0 {
		return true
	}
	if text[j-1] != ',' {
		return true
	}
	k := j - 1
	for k > 0 && (text[k-1] == ' ' || text[k-1] == '\t') {
		k--
	}
	from := max(0, k-addrPostalBeforeWindow)
	if b := strings.LastIndexAny(text[from:k], ",\n:;"); b >= 0 {
		from += b + 1
	}
	seg := text[from:k]
	return !addrHouseEndRE.MatchString(seg) && !addrStreetLike(seg)
}

func addrRetryAt(text string, s, e int) int {
	for k := s + 1; k < e; k++ {
		if text[k-1] == ' ' || text[k-1] == '\n' {
			return k
		}
	}
	return e
}

func addrCoreEnds(text string, e int, wordLast bool) bool {
	if e >= len(text) {
		return true
	}
	r, size := utf8.DecodeRuneInString(text[e:])
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return false
	}
	if r == '-' && wordLast {
		next, _ := utf8.DecodeRuneInString(text[e+size:])
		return !unicode.IsLetter(next)
	}
	return true
}

var addrAbbreviations = wordSet(`st ste sta sto dr av avda bd gen prof mr mme pres cdt mal str dott ing pza ctra burg ds jhr ir mgr apt no num`)

func addrSentenceBreak(text string, s, e int) int {
	for i := s; i+2 < e; i++ {
		if text[i] != '.' || text[i+1] != ' ' {
			continue
		}
		if r, _ := utf8.DecodeRuneInString(text[i+2:]); !unicode.IsUpper(r) {
			continue
		}
		j := i
		for j > s {
			r, size := utf8.DecodeLastRuneInString(text[:j])
			if !unicode.IsLetter(r) {
				break
			}
			j -= size
		}
		word := strings.ToLower(text[j:i])
		if utf8.RuneCountInString(word) > 1 && !addrAbbreviations[word] {
			return i
		}
	}
	return -1
}

func addrNotPostalSplit(text string, _, e int) bool {
	return !(e+3 < len(text) && text[e] == '-' && isDigitByte(text[e+1]) && isDigitByte(text[e+2]) && isDigitByte(text[e+3]))
}

var addrArticles = wordSet(`die der den dem das des eine einer einem diese dieser diesem unsere ihre meine seine the de het een onze deze`)

func addrSplitNamed(text string, s, e int) bool {
	f := strings.Fields(text[s:e])
	return len(f) > 3 || len(f) == 3 && !addrArticles[strings.ToLower(f[0])]
}

func addrTail(text string, e int, kind addrTailKind) (int, bool) {
	switch kind {
	case addrTailUKCountry:
		return addrUKCountryTail(text, e)
	case addrTailUKPostcode:
		return addrUKPostTail(text, e)
	case addrTailUK:
		if end, ok := addrUKPostTail(text, e); ok {
			return end, true
		}
		return addrUKCountryTail(text, e)
	}
	unitEnd, strongEnd := addrUnits(text, e)
	for _, from := range []int{unitEnd, e} {
		b := addrBreak(text, from)
		if b < 0 {
			continue
		}
		if end, ok := addrPostalCity(text, b); ok {
			return end, true
		}
		if from == e {
			break
		}
	}
	return strongEnd, false
}

func addrBreak(text string, i int) int {
	j := i
	if j < len(text) && text[j] == ',' {
		j++
	}
	for _, sep := range []string{" - ", " | ", " · "} {
		if j == i && strings.HasPrefix(text[j:], sep) {
			return j + len(sep)
		}
	}
	k := j
	for k < len(text) && (text[k] == ' ' || text[k] == '\t' || text[k] == '\r') {
		k++
	}
	if k < len(text) && text[k] == '\n' {
		k++
		for k < len(text) && (text[k] == ' ' || text[k] == '\t') {
			k++
		}
		return k
	}
	if j < len(text) && text[j] == ' ' {
		return j + 1
	}
	if j > i {
		return j
	}
	return -1
}

func addrCapWord(text string, i int) int {
	if i >= len(text) {
		return -1
	}
	r, size := utf8.DecodeRuneInString(text[i:])
	if !unicode.IsUpper(r) {
		return -1
	}
	j, n := i+size, 1
	for j < len(text) {
		r, size = utf8.DecodeRuneInString(text[j:])
		switch {
		case unicode.IsLetter(r):
		case r == '\'' || r == '’' || r == '-':
			if next, _ := utf8.DecodeRuneInString(text[j+size:]); !unicode.IsLetter(next) {
				return j
			}
		case r == '.':
			if j+2 < len(text) && text[j+1] == ' ' && addrDotLeads[strings.ToLower(text[i:j])] {
				if next, _ := utf8.DecodeRuneInString(text[j+2:]); unicode.IsUpper(next) {
					return j + 1
				}
			}
			return j
		default:
			return j
		}
		j += size
		n++
	}
	return j
}

func addrPostalAt(text string, i int) int {
	if i > 0 && (isDigitByte(text[i-1]) || isAlphaByte(text[i-1]) || text[i-1] == '.' || text[i-1] == '-') {
		return -1
	}
	j := i
	for j < len(text) && j-i < 2 && text[j] >= 'A' && text[j] <= 'Z' {
		j++
	}
	prefixed := j > i
	if prefixed {
		if j >= len(text) || text[j] != '-' {
			return -1
		}
		j++
	}
	d := j
	for d < len(text) && isDigitByte(text[d]) {
		d++
	}
	digit := func(k int) bool { return k < len(text) && isDigitByte(text[k]) }
	upper := func(k int) bool { return k < len(text) && text[k] >= 'A' && text[k] <= 'Z' }
	end := -1
	switch d - j {
	case 5:
		end = d
	case 4:
		end = d
		k := d
		if k < len(text) && text[k] == ' ' {
			k++
		}
		switch {
		case d < len(text) && text[d] == '-' && digit(d+1) && digit(d+2) && digit(d+3) && !prefixed:
			end = d + 4
		case !prefixed && upper(k) && upper(k+1) && k+2 < len(text) && text[k+2] == ' ':
			end = k + 2
		}
	case 3:
		if !prefixed && d < len(text) && text[d] == ' ' && digit(d+1) && digit(d+2) && !digit(d+3) {
			end = d + 3
		}
	case 2:
		if !prefixed && d < len(text) && text[d] == '-' && digit(d+1) && digit(d+2) && digit(d+3) {
			end = d + 4
		}
	}
	if end < 0 || end >= len(text) || text[end] != ' ' {
		return -1
	}
	return end
}

var addrCityConns = strings.Split("de las |de los |de la |de l'|de l’|del |de |des |du |d'|d’|sur |sous |lès |lez |le |la |les |l'|l’|di |della |delle |dei |do |da |dos |das ", "|")

var addrRiverConns = strings.Split("an der |an den |am |im |in der |ob der |vor der |bei |op |aan den |aan de |aan |bij ", "|")

func addrConnAt(text string, i int, conns []string, caps bool) int {
	rest := text[i:]
	for _, c := range conns {
		if len(rest) < len(c) {
			continue
		}
		if strings.HasPrefix(rest, c) || caps && rest[:len(c)] == strings.ToUpper(c) {
			return i + len(c)
		}
	}
	return -1
}

func addrCity(text string, i int) int {
	e := addrCapWord(text, i)
	if e < 0 || utf8.RuneCountInString(text[i:e]) < 2 || addrAfterCity[strings.ToLower(text[i:e])] {
		return -1
	}
	caps := text[i:e] == strings.ToUpper(text[i:e])
	last := strings.TrimSuffix(strings.ToLower(text[i:e]), ".")
	bare, conns := 0, 0
	for e < len(text) && text[e] == ' ' {
		j := e + 1
		if conns < 2 {
			if c := addrConnAt(text, j, addrCityConns, caps); c > 0 {
				if w := addrCapWord(text, c); w > 0 {
					e, last = w, strings.ToLower(text[c:w])
					conns++
					continue
				}
			}
			if c := addrConnAt(text, j, addrRiverConns, caps); c > 0 {
				if w := addrCapWord(text, c); w > 0 && addrRivers[strings.ToLower(text[c:w])] {
					e, last = w, ""
					conns++
					continue
				}
				break
			}
		}
		w := addrCapWord(text, j)
		if w < 0 || utf8.RuneCountInString(text[j:w]) < 2 {
			break
		}
		word := strings.ToLower(text[j:w])
		if addrCountryAt(text, j) > 0 || addrAfterCity[word] || w < len(text) && text[w] == ':' {
			break
		}
		lead := addrCityLeads[last]
		if !(bare == 0 && conns == 0) && !(lead && bare < 2) {
			break
		}
		if (!lead || bare > 0 || addrLoneLeads[last] && addrNameWord(text[j:w])) && addrCapFollows(text, w) && !addrTwoWordTowns[strings.ToLower(text[i:w])] {
			break
		}
		e, last = w, word
		bare++
	}
	return e
}

func addrNameWord(w string) bool {
	for _, part := range strings.FieldsFunc(w, func(r rune) bool { return r == '-' || r == '\'' || r == '’' }) {
		if _, ok := namedata.LookupGiven(part); ok {
			return true
		}
		if _, ok := namedata.LookupSurname(part); ok {
			return true
		}
	}
	return false
}

func addrCapFollows(text string, e int) bool {
	if e+1 >= len(text) || text[e] != ' ' {
		return false
	}
	n := addrCapWord(text, e+1)
	if n < 0 || addrCountryAt(text, e+1) > 0 || n < len(text) && text[n] == ':' {
		return false
	}
	return !addrAfterCity[strings.TrimSuffix(strings.ToLower(text[e+1:n]), ".")]
}

func addrCountryAt(text string, i int) int {
	ends := make([]int, 0, 3)
	j := i
	for len(ends) < 3 {
		w := addrCapWord(text, j)
		if w < 0 {
			break
		}
		ends = append(ends, w)
		if w >= len(text) || text[w] != ' ' {
			break
		}
		j = w + 1
	}
	for k := len(ends) - 1; k >= 0; k-- {
		name := text[i:ends[k]]
		if !addrCountries[strings.ToLower(name)] {
			continue
		}
		if len(name) <= 2 && name != strings.ToUpper(name) {
			continue
		}
		if r, _ := utf8.DecodeRuneInString(text[ends[k]:]); ends[k] < len(text) && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			continue
		}
		return ends[k]
	}
	return -1
}

func addrCountryTail(text string, e int) int {
	if b := addrBreak(text, e); b >= 0 {
		if c := addrCountryAt(text, b); c > 0 {
			return c
		}
	}
	return e
}

func addrPostalCity(text string, i int) (int, bool) {
	p := addrPostalAt(text, i)
	if p < 0 {
		return 0, false
	}
	c := p + 1
	if c < len(text) && text[c] == ' ' {
		c++
	}
	if post := text[i:p]; strings.HasPrefix(text[c:], post+" ") {
		c += len(post) + 1
	}
	e := addrCity(text, c)
	if e < 0 {
		return 0, false
	}
	if p-i == 4 && e+1 < len(text) && text[e] == ' ' {
		if r, size := utf8.DecodeRuneInString(text[e+1:]); unicode.IsUpper(r) {
			if next, _ := utf8.DecodeRuneInString(text[e+1+size:]); !unicode.IsLetter(next) && !unicode.IsDigit(next) && next != '.' {
				e += 1 + size
			}
		}
	}
	if e+4 <= len(text) && text[e] == ' ' && text[e+1] == '(' && text[e+2] >= 'A' && text[e+2] <= 'Z' && text[e+3] >= 'A' && text[e+3] <= 'Z' && e+4 < len(text) && text[e+4] == ')' {
		e += 5
	}
	return addrCountryTail(text, e), true
}

var (
	addrUnitValued = wordSet(`appt apt appartement apartment apartamento bât bat bâtiment batiment escalier esc escalera étage etage porte piso planta puerta pta portal bloque bloco lote andar scala interno int piano top stiege tür whg wohnung flat unit suite ste room floor bus huis`)
	addrUnitBare   = wordSet(`bajo ático atico entresuelo hinterhaus vorderhaus eg og dg esq dto izq izda dcha dcho drcha frente fte ctro centro rc`)
	addrFloorWords = wordSet(`og stock etage étage obergeschoss piano andar verdieping piso planta`)
)

var addrOrdinalEnds = wordSet(`n r t a e è er ère ème eme on o`)

func addrLowerWord(text string, i int) (string, int) {
	if i > len(text) {
		return "", i
	}
	j := i
	for j < len(text) {
		r, size := utf8.DecodeRuneInString(text[j:])
		if !unicode.IsLetter(r) {
			break
		}
		j += size
	}
	return strings.ToLower(text[i:j]), j
}

func addrUnitAt(text string, i int) (int, bool) {
	d := i
	for d < len(text) && d-i < 2 && isDigitByte(text[d]) {
		d++
	}
	if d > i {
		for _, mark := range []string{"º", "ª", "°"} {
			if !strings.HasPrefix(text[d:], mark) {
				continue
			}
			end := d + len(mark)
			if end < len(text) && text[end] == '.' {
				end++
			}
			k := end
			if k < len(text) && text[k] == ' ' {
				k++
			}
			if w, we := addrLowerWord(text, k); we > k && (addrUnitBare[w] || addrUnitValued[w] || we-k == 1 && text[k] >= 'A' && text[k] <= 'Z') {
				end = we
				if end < len(text) && text[end] == '.' && addrUnitBare[w] {
					end++
				}
			}
			return end, true
		}
		if w, we := addrLowerWord(text, d); we > d && addrOrdinalEnds[w] && w == text[d:we] {
			if f, fe := addrLowerWord(text, we+1); we < len(text) && text[we] == ' ' && addrFloorWords[f] {
				return fe, false
			}
			return we, false
		}
		if d < len(text) && text[d] == '.' && d+1 < len(text) && text[d+1] == ' ' {
			if w, we := addrLowerWord(text, d+2); addrFloorWords[w] {
				return we, false
			}
		}
		k := d
		if k < len(text) && text[k] == ' ' {
			k++
		}
		if k < len(text) && isAlphaByte(text[k]) && (k+1 == len(text) || text[k+1] == ',' || text[k+1] == '\n') {
			return k + 1, false
		}
		return -1, false
	}
	w, we := addrLowerWord(text, i)
	if we == i {
		return -1, false
	}
	if addrUnitBare[w] {
		if we < len(text) && text[we] == '.' {
			we++
		}
		return we, false
	}
	if !addrUnitValued[w] {
		return -1, false
	}
	k := we
	if k < len(text) && text[k] == '.' {
		k++
	}
	if k < len(text) && text[k] == ' ' {
		k++
	}
	v := k
	hasDigit := false
	for v < len(text) && v-k < 4 && (isDigitByte(text[v]) || isAlphaByte(text[v])) {
		hasDigit = hasDigit || isDigitByte(text[v])
		v++
	}
	if v == k || v < len(text) && (isDigitByte(text[v]) || isAlphaByte(text[v])) || !hasDigit && !(v-k == 1 && text[k] >= 'A' && text[k] <= 'Z') {
		return -1, false
	}
	return v, false
}

func addrUnits(text string, e int) (int, int) {
	end, strongEnd, allStrong := e, e, true
	for n := 0; n < 3; n++ {
		rest := text[end:]
		if w, we := addrLowerWord(text, end+1); strings.HasPrefix(rest, " ") && (w == "bis" || w == "ter") && we-end == 4 {
			end = we
			allStrong = false
			continue
		}
		at := -1
		for _, sep := range []string{", ", " - ", " ", "/"} {
			if strings.HasPrefix(rest, sep) {
				at = end + len(sep)
				break
			}
		}
		if at < 0 {
			break
		}
		u, strong := addrUnitAt(text, at)
		if u < 0 {
			break
		}
		end = u
		allStrong = allStrong && strong
		if allStrong {
			strongEnd = end
		}
	}
	return end, strongEnd
}

var addrUKConns = wordSet(`on upon under le in by the next de la cum super juxta`)

func addrUKBreak(text string, i int) (int, bool) {
	j := i
	if j < len(text) && text[j] == '.' {
		n := 0
		for j-n > 0 && isAlphaByte(text[j-n-1]) {
			n++
		}
		if n > 0 && n <= 4 {
			j++
		}
	}
	if j < len(text) && text[j] == ',' {
		j++
	}
	k := j
	for k < len(text) && (text[k] == ' ' || text[k] == '\t' || text[k] == '\r') {
		k++
	}
	if k < len(text) && text[k] == '\n' {
		k++
		for k < len(text) && (text[k] == ' ' || text[k] == '\t') {
			k++
		}
		return k, true
	}
	if j < len(text) && text[j] == ' ' && j == k-1 {
		return k, false
	}
	return -1, false
}

func addrUKPostAt(text string, i int) int {
	if i >= len(text) {
		return -1
	}
	m := addrUKPostAtRE.FindStringIndex(text[i:min(len(text), i+8)])
	if m == nil {
		return -1
	}
	end := i + m[1]
	if end < len(text) && (isDigitByte(text[end]) || isAlphaByte(text[end])) {
		return -1
	}
	return end
}

func addrUKPostTail(text string, e int) (int, bool) {
	pos, words, lines := e, 0, 0
	unit := false
	for words <= 8 {
		b, nl := addrUKBreak(text, pos)
		if b < 0 {
			return e, false
		}
		if nl {
			if lines++; lines > 3 {
				return e, false
			}
		}
		if p := addrUKPostAt(text, b); p > 0 {
			return addrCountryTail(text, p), true
		}
		if w := addrCapWord(text, b); w > 0 {
			word, _ := addrLowerWord(text, b)
			unit = addrUnitValued[word]
			pos = w
			words++
			continue
		}
		if word, we := addrLowerWord(text, b); we > b && addrUKConns[word] && we < len(text) && text[we] == ' ' && !nl {
			pos = we
			unit = false
			continue
		}
		if unit {
			v := b
			for v < len(text) && v-b < 4 && (isDigitByte(text[v]) || isAlphaByte(text[v])) {
				v++
			}
			if v > b && isDigitByte(text[b]) {
				pos = v
				unit = false
				continue
			}
		}
		return e, false
	}
	return e, false
}

func addrUKTown(text string, i int) int {
	e := addrCapWord(text, i)
	if e < 0 || utf8.RuneCountInString(text[i:e]) < 2 {
		return -1
	}
	for n := 0; n < 3 && e < len(text) && text[e] == ' '; n++ {
		j := e + 1
		if word, we := addrLowerWord(text, j); we > j && addrUKConns[word] && we < len(text) && text[we] == ' ' {
			j = we + 1
		}
		w := addrCapWord(text, j)
		if w < 0 || addrCountryAt(text, j) > 0 {
			break
		}
		e = w
	}
	return e
}

func addrUKCountryTail(text string, e int) (int, bool) {
	pos := e
	for n := 0; n < 3; n++ {
		b, _ := addrUKBreak(text, pos)
		if b < 0 || b == pos+1 && text[pos] == ' ' {
			return e, false
		}
		if c := addrCountryAt(text, b); c > 0 && n > 0 {
			if !addrUKCountries[strings.ToLower(text[b:c])] {
				return e, false
			}
			return c, true
		}
		t := addrUKTown(text, b)
		if t < 0 {
			return e, false
		}
		pos = t
	}
	return e, false
}

const addrContextWindow = 256

func addrWindowStart(text string, i int) (int, bool) {
	lo := max(0, i-addrContextWindow)
	if k := strings.LastIndexByte(text[lo:i], '\n'); k >= 0 {
		return lo + k + 1, false
	}
	if lo == 0 {
		return 0, false
	}
	for lo < i && !utf8.RuneStart(text[lo]) {
		lo++
	}
	return lo, true
}

func addrStreetLike(seg string) bool {
	seg = strings.TrimSpace(seg)
	if len(seg) < 4 || len(seg) > 60 || !addrStreetLineRE.MatchString(seg) && !addrNumLineRE.MatchString(seg) {
		return false
	}
	for _, w := range strings.Fields(seg) {
		if IsMonthOrWeekday(strings.Trim(w, ",.")) {
			return false
		}
	}
	return true
}

var addrCues = strings.Split("adresse|anschrift|lieferadresse|postanschrift|address|dirección|direccion|domicilio|indirizzo|morada|endereço|adres|plz|wohnhaft in|wohnt in|lives in|living in|vive en|habite à|mora em|woont in|abita a|abita in", "|")

func addrCueEnds(before string) bool {
	t := strings.TrimRight(before, " \t")
	if strings.HasSuffix(t, ":") || strings.HasSuffix(t, "-") {
		t = strings.TrimRight(t[:len(t)-1], " \t")
	}
	cut := max(0, len(t)-32)
	for cut < len(t) && !utf8.RuneStart(t[cut]) {
		cut++
	}
	tail := strings.ToLower(t[cut:])
	for _, cue := range addrCues {
		if !strings.HasSuffix(tail, cue) {
			continue
		}
		if r, _ := utf8.DecodeLastRuneInString(tail[:len(tail)-len(cue)]); !unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func addrPrevLine(text string, ls int) string {
	if ls == 0 {
		return ""
	}
	end := ls - 1
	for end > 0 && (text[end-1] == ' ' || text[end-1] == '\t' || text[end-1] == '\r') {
		end--
	}
	start, _ := addrWindowStart(text, end)
	return strings.TrimSpace(text[start:end])
}

func addrAdjacent(text string, s, e int) bool {
	ls, clipped := addrWindowStart(text, s)
	before := text[ls:s]
	if !clipped && strings.TrimSpace(before) == "" {
		for k := e; k < len(text) && text[k] != '\n'; k++ {
			if strings.IndexByte(" \t\r.,;)", text[k]) < 0 {
				return false
			}
		}
		line := addrPrevLine(text, ls)
		return addrStreetLike(strings.TrimSuffix(line, ",")) || addrCueLineRE.MatchString(line)
	}
	if addrCueEnds(before) {
		return true
	}
	trimmed := strings.TrimRight(before, " \t")
	if !strings.HasSuffix(trimmed, ",") {
		return false
	}
	segs := strings.Split(strings.TrimSuffix(trimmed, ","), ",")
	first := 0
	if clipped {
		first = 1
	}
	for k := len(segs) - 1; k >= first && k >= len(segs)-2; k-- {
		if addrStreetLike(segs[k]) {
			return true
		}
	}
	return false
}

func addrPostalCityScan(text, _ string) []entity.Span {
	var out []entity.Span
	for i := 0; i+6 < len(text); i++ {
		if !isDigitByte(text[i]) {
			continue
		}
		n := 1
		for i+n < len(text) && isDigitByte(text[i+n]) {
			n++
		}
		s := i
		i += n - 1
		if n != 5 && n != 4 && n != 2 {
			continue
		}
		if s >= 2 && text[s-1] == '-' && text[s-2] >= 'A' && text[s-2] <= 'Z' {
			s -= 2
			if s >= 1 && text[s-1] >= 'A' && text[s-1] <= 'Z' {
				s--
			}
		}
		e, ok := addrPostalCity(text, s)
		if !ok || !addrAdjacent(text, s, e) || !boundaryOK(text, s, e) {
			continue
		}
		out = append(out, entity.Span{Start: s, End: e})
		i = e - 1
	}
	return out
}

func addrUKTownScan(text, _ string) []entity.Span {
	var out []entity.Span
	for _, m := range addrUKPostFindRE.FindAllStringIndex(text, -1) {
		s, e := m[0], m[1]
		if e < len(text) && (isDigitByte(text[e]) || isAlphaByte(text[e])) || s > 0 && (isDigitByte(text[s-1]) || isAlphaByte(text[s-1]) || text[s-1] == '-' || text[s-1] == '_' || text[s-1] == '/') {
			continue
		}
		area := 1
		if text[s+1] >= 'A' && text[s+1] <= 'Z' {
			area = 2
		}
		if !ukPostcodeAreas[text[s:s+area]] {
			continue
		}
		ls, clipped := addrWindowStart(text, s)
		if clipped {
			k := strings.IndexByte(text[ls:s], ' ')
			if k < 0 {
				k = s - ls - 1
			}
			ls += k + 1
		}
		head := text[ls:s]
		head = strings.TrimSuffix(head, " ")
		head = strings.TrimSuffix(head, ",")
		words := strings.Split(head, " ")
		start, caps := s, 0
		pos := len(head)
		for k := len(words) - 1; k >= 0 && caps < 3; k-- {
			w := words[k]
			wordStart := ls + pos - len(w)
			if w == "" {
				break
			}
			if addrUKConns[w] && k > 0 {
				pos -= len(w) + 1
				continue
			}
			if end := addrCapWord(text, wordStart); end != wordStart+len(w) || utf8.RuneCountInString(w) < 2 || addrAfterCity[strings.ToLower(w)] {
				break
			}
			start = wordStart
			caps++
			pos -= len(w) + 1
		}
		if caps == 0 && !addrPostcodeCue.MatchString(text[max(ls, s-24):s]) {
			continue
		}
		e = addrCountryTail(text, e)
		if boundaryOK(text, start, e) {
			out = append(out, entity.Span{Start: start, End: e})
		}
	}
	return out
}
