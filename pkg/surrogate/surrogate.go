package surrogate

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Generator struct {
	secret []byte
}

func NewGenerator(secret []byte) *Generator {
	return &Generator{secret: append([]byte(nil), secret...)}
}

type rng struct {
	buf []byte
	pos int
	g   *Generator
	key string
	ctr int
}

func (g *Generator) stream(kind, real string, attempt int) *rng {
	return &rng{g: g, key: fmt.Sprintf("%s\x00%s\x00%d", kind, strings.ToLower(real), attempt)}
}

func (r *rng) next() uint64 {
	if r.pos+8 > len(r.buf) {
		m := hmac.New(sha256.New, r.g.secret)
		fmt.Fprintf(m, "%s\x00%d", r.key, r.ctr)
		r.ctr++
		r.buf = m.Sum(nil)
		r.pos = 0
	}
	v := binary.BigEndian.Uint64(r.buf[r.pos:])
	r.pos += 8
	return v
}

func (r *rng) intn(n int) int { return int(r.next() % uint64(n)) }

func (r *rng) pick(list []string) string { return list[r.intn(len(list))] }

func (r *rng) digits(n int, firstNonZero bool) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		d := r.intn(10)
		if i == 0 && firstNonZero && d == 0 {
			d = 1 + r.intn(9)
		}
		b.WriteByte(byte('0' + d))
	}
	return b.String()
}

func (r *rng) syllableName() string {
	n := 2 + r.intn(2)
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString(r.pick(syllOnset))
		b.WriteString(r.pick(syllVowel))
		if i == n-1 {
			b.WriteString(r.pick(syllCoda))
		}
	}
	s := b.String()
	return strings.ToUpper(s[:1]) + s[1:]
}

func (g *Generator) GivenName(real string, attempt int) string {
	return g.GivenNameFor(real, GenderOf(real), attempt)
}

func (g *Generator) FamilyName(real string, attempt int) string {
	r := g.stream("family", real, attempt)
	if attempt < 3*len(familyNames) {
		return r.pick(familyNames)
	}
	return r.syllableName() + r.pick([]string{"son", "ley", "wood", "ford", "man", "er", "ton"})
}

var exampleDomains = []string{"example.com", "example.org", "example.net"}

func (g *Generator) Email(real string, attempt int) string {
	r := g.stream("email", real, attempt)
	local := strings.ToLower(r.pick(givenNames) + "." + r.pick(familyNames))
	if attempt > 0 {
		local += strconv.Itoa(r.intn(1000))
	}
	if at := strings.LastIndex(real, "@"); at > 0 && strings.Contains(real[:at], "+") {
		local += "+" + strings.ToLower(r.pick(orgKinds))
	}
	return local + "@" + r.pick(exampleDomains)
}

func (g *Generator) Word(real string, attempt int) string {
	return strings.ToLower(g.stream("word", real, attempt).syllableName())
}

func (g *Generator) EmailDomain(real string) string {
	return g.stream("domain", real, 0).pick(exampleDomains)
}

func (g *Generator) Phone(real string, attempt int) string {
	r := g.stream("phone", real, attempt)
	var digits []int
	for i, ch := range real {
		if ch >= '0' && ch <= '9' {
			digits = append(digits, i)
		}
	}
	var repl string
	trunk := len(digits) > 2 && real[digits[0]] == '0' && !strings.HasPrefix(strings.TrimSpace(real), "+")
	switch {
	case len(digits) == 0:
		return g.SpokenPhone(real, attempt)
	case len(digits) == 10 && !trunk:
		repl = fmt.Sprintf("%d%s555%s", 2+r.intn(7), r.digits(2, false), "01"+r.digits(2, false))
	case len(digits) == 11 && real[digits[0]] == '1':
		repl = fmt.Sprintf("1%d%s555%s", 2+r.intn(7), r.digits(2, false), "01"+r.digits(2, false))
	default:
		keep := 0
		if strings.HasPrefix(strings.TrimSpace(real), "+") {
			keep = min(2, len(digits))
		}
		if trunk {
			keep = 2
			if real[digits[1]] == '0' && len(digits) > 4 {
				keep = 4
			}
		}
		var b strings.Builder
		for i := 0; i < keep; i++ {
			b.WriteByte(real[digits[i]])
		}
		b.WriteString(r.digits(len(digits)-keep, keep == 0))
		repl = b.String()
	}
	out := []byte(real)
	for i, pos := range digits {
		out[pos] = repl[i]
	}
	return string(out)
}

var spokenWordRE = regexp.MustCompile(`(?i)\b(?:zero|oh|one|two|three|four|five|six|seven|eight|nine)\b`)
var spokenWords = []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine"}

func (g *Generator) SpokenPhone(real string, attempt int) string {
	r := g.stream("phone.spoken", real, attempt)
	return spokenWordRE.ReplaceAllStringFunc(real, func(w string) string {
		d := r.intn(10)
		out := spokenWords[d]
		if d == 0 && strings.EqualFold(w, "oh") {
			out = "oh"
		}
		switch {
		case w == strings.ToUpper(w):
			return strings.ToUpper(out)
		case w[:1] == strings.ToUpper(w[:1]):
			return strings.ToUpper(out[:1]) + out[1:]
		}
		return out
	})
}

var addrNumRE = regexp.MustCompile(`^\d+`)
var addrForeignRE = regexp.MustCompile(`\n|[A-Z]{1,2}\d[A-Z\d]? ?\d[A-Z]{2}\b|\b\d{4} ?[A-Z]{2}\b`)
var addrTokenRE = regexp.MustCompile(`[\p{L}'’]+|\d+|[^\p{L}\d]+`)
var addrLetterPostRE = regexp.MustCompile(`[A-Z]{1,2}\d[A-Z\d]? ?\d[A-Z]{2}\b|\b\d{4} ?[A-Z]{2}\b`)
var addrAttachedRE = regexp.MustCompile(`(?i)^\p{L}{2,}?(straße|strasse|gasse|allee|gracht|straat|laan|plein|singel|weg|platz|dijk|steeg|dreef|markt|ring|damm|ufer|kade|str)$`)
var addrElisionRE = regexp.MustCompile(`^(?i:[dl])['’]`)

func addrWordSet(s string) map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(s) {
		m[w] = true
	}
	return m
}

var addrStreetWords = addrWordSet(`
rue boulevard bd chemin allée allee impasse quai avenue av place cours route passage
calle carrer avenida avda avinguda paseo passeig rua travessa viale piazza ronda camino carretera rambla via plaza plaça corso largo vicolo
straße strasse str platz weg gasse ring damm ufer chaussee laan straat
am an im zum zur auf unter
street st ave road rd blvd lane ln drive dr court ct pl way terrace ter circle cir parkway pkwy highway hwy square sq trail trl loop run pike
alley row crescent close bend ridge crossing path point hollow glen heights grove park view commons landing walk green meadow meadows cove creek hill pass mews garden gardens plz`)

var addrKeepWords = addrWordSet(`
flat apartment apt unit suite ste room floor bldg fl rm po box bis ter no num número nr
in op aan der den dem de la las los del des du le les el van di della delle dei do da dos das the
n s e w ne nw se sw north south east west`)

func (g *Generator) shapedAddress(r *rng, real string) string {
	post := map[int]int{}
	for _, m := range addrLetterPostRE.FindAllStringIndex(real, -1) {
		post[m[0]] = m[1]
	}
	toks := addrTokenRE.FindAllStringIndex(real, -1)
	var b strings.Builder
	seenStreet, seenNumber, city := false, false, false
	name := func(src string) string {
		list := streetNames
		if city {
			list = cityNames
		}
		return matchWordCase(src, r.pick(list))
	}
	skip := 0
	for i, t := range toks {
		if t[0] < skip {
			continue
		}
		if end, ok := post[t[0]]; ok {
			b.WriteString(shapeLike(r, real[t[0]:end]))
			skip = end
			city = true
			continue
		}
		tok := real[t[0]:t[1]]
		first, _ := utf8.DecodeRuneInString(tok)
		switch {
		case unicode.IsDigit(first):
			if len(tok) >= 4 && seenStreet && seenNumber {
				city = true
			}
			b.WriteString(r.digits(len(tok), tok[0] != '0' && len(tok) < 5))
			seenNumber = true
		case !unicode.IsLetter(first) && first != '\'' && first != '’':
			if seenStreet && seenNumber && strings.ContainsAny(tok, ",\n") {
				city = true
			}
			b.WriteString(tok)
		default:
			low := strings.ToLower(tok)
			prevDigit := i > 0 && toks[i-1][1] == t[0] && real[t[0]-1] >= '0' && real[t[0]-1] <= '9'
			nextZip := city && len(tok) == 2 && tok == strings.ToUpper(tok) && i+2 < len(toks) && toks[i+2][1]-toks[i+2][0] == 5 && unicode.IsDigit(rune(real[toks[i+2][0]]))
			switch {
			case addrStreetWords[low]:
				seenStreet = true
				b.WriteString(tok)
			case addrKeepWords[low] || nextZip:
				b.WriteString(tok)
			case utf8.RuneCountInString(tok) == 1:
				if prevDigit || seenNumber && first < utf8.RuneSelf && unicode.IsUpper(first) {
					b.WriteString(shapeLike(r, tok))
				} else {
					b.WriteString(tok)
				}
			case addrElisionRE.MatchString(tok):
				n := strings.IndexAny(tok, "'’")
				_, size := utf8.DecodeRuneInString(tok[n:])
				b.WriteString(tok[:n+size] + name(tok[n+size:]))
			case !unicode.IsUpper(first):
				b.WriteString(tok)
			default:
				if m := addrAttachedRE.FindStringSubmatch(tok); m != nil && !city {
					seenStreet = true
					b.WriteString(matchWordCase(tok, r.pick(streetNames)+strings.ToLower(m[1])))
				} else {
					b.WriteString(name(tok))
				}
			}
		}
	}
	return b.String()
}

func matchWordCase(src, repl string) string {
	if utf8.RuneCountInString(src) > 1 && src == strings.ToUpper(src) {
		return strings.ToUpper(repl)
	}
	return repl
}

var addrUnitRE = regexp.MustCompile(`(?i)((?:Apt|Apartment|Suite|Ste|Unit|#)\.?\s*)([A-Za-z0-9\-]+)`)
var addrSuffixRE = regexp.MustCompile(`\b(Street|St|Avenue|Ave|Road|Rd|Boulevard|Blvd|Lane|Ln|Drive|Dr|Court|Ct|Place|Pl|Way|Terrace|Ter|Circle|Cir|Parkway|Pkwy|Highway|Hwy|Square|Sq|Trail|Trl|Loop|Run|Pike|Alley|Row|Crescent|Close|Bend|Ridge|Crossing|Path|Point|Hollow|Glen|Heights|Grove|Park|View|Commons|Landing|Walk|Green|Meadows?|Cove|Creek|Hill|Pass|Mews|Gardens?)\b(\.?)`)
var addrTailRE = regexp.MustCompile(`,\s*([A-Z][A-Za-z]+(?:\s[A-Z][A-Za-z]+){0,2}),\s*([A-Za-z][A-Za-z\s]*?)(?:\s+(\d{5})(-\d{4})?)?$`)
var poBoxRE = regexp.MustCompile(`(?i)^(P\.?\s?O\.?\s+Box\s+)(\d+)$`)

func (g *Generator) Address(real string, attempt int) string {
	r := g.stream("address", real, attempt)
	if m := poBoxRE.FindStringSubmatch(real); m != nil {
		return m[1] + r.digits(len(m[2]), true)
	}
	num := addrNumRE.FindString(real)
	sm := addrSuffixRE.FindStringSubmatchIndex(real)
	if strings.ContainsAny(real, "0123456789") && (num == "" || sm == nil || addrForeignRE.MatchString(real)) {
		if out := g.shapedAddress(r, real); out != real {
			return out
		}
	}
	if num == "" {
		return fmt.Sprintf("%s %s Lane", r.digits(3, true), r.pick(streetNames))
	}
	suffix := "Street"
	if sm != nil {
		suffix = real[sm[2]:sm[5]]
	}
	out := r.digits(len(num), true) + " " + r.pick(streetNames) + " " + suffix
	rest := ""
	if sm != nil {
		rest = real[sm[1]:]
	}
	if um := addrUnitRE.FindStringSubmatchIndex(rest); um != nil {
		sep := rest[:um[0]]
		out += sep + rest[um[2]:um[3]] + shapeLike(r, rest[um[4]:um[5]])
	}
	if tm := addrTailRE.FindStringSubmatch(rest); tm != nil {
		out += ", " + r.pick(cityNames) + ", " + strings.TrimSpace(tm[2])
		if tm[3] != "" {
			out += " " + r.digits(5, false)
			if tm[4] != "" {
				out += "-" + r.digits(4, false)
			}
		}
	}
	return out
}

func (g *Generator) Location(real string, attempt int) string {
	r := g.stream("location", real, attempt)
	city := r.pick(cityNames)
	if attempt >= 3*len(cityNames) {
		city = r.syllableName() + r.pick([]string{"ton", "field", "burg", "ville", "mere"})
	}
	if i := strings.Index(real, ","); i > 0 {
		return city + real[i:]
	}
	return city
}

var orgSuffixRE = regexp.MustCompile(`(?i)\s*,?\s+(Inc|LLC|Ltd|Corp|Corporation|GmbH|PLC|LLP|Co)\.?$`)

func (g *Generator) Org(real string, attempt int) string {
	r := g.stream("org", real, attempt)
	stem := r.pick(orgStems)
	if attempt >= 3*len(orgStems) {
		stem = r.syllableName()
	}
	if m := orgSuffixRE.FindString(real); m != "" {
		return stem + " " + r.pick(orgKinds) + m
	}
	return stem + " " + r.pick(orgKinds)
}

func (g *Generator) URL(real string, attempt int) string {
	r := g.stream("url", real, attempt)
	scheme := "https://"
	if strings.HasPrefix(strings.ToLower(real), "http://") {
		scheme = "http://"
	}
	return fmt.Sprintf("%swww.%s/r/%s", scheme, r.pick(exampleDomains), strconv.FormatUint(r.next()%(1<<40), 36))
}

func (g *Generator) IP(real string, attempt int) string {
	r := g.stream("ip", real, attempt)
	if strings.Contains(real, ":") {
		return fmt.Sprintf("2001:db8::%x:%x", r.intn(0xffff), r.intn(0xffff))
	}
	nets := []string{"192.0.2", "198.51.100", "203.0.113"}
	return fmt.Sprintf("%s.%d", r.pick(nets), 1+r.intn(254))
}

func (g *Generator) Shape(kind, real string, attempt int) string {
	r := g.stream("shape:"+kind, real, attempt)
	return shapeLike(r, real)
}

func shapeLike(r *rng, real string) string {
	var b strings.Builder
	for _, ch := range real {
		switch {
		case unicode.IsDigit(ch):
			b.WriteByte(byte('0' + r.intn(10)))
		case unicode.IsUpper(ch):
			b.WriteByte(byte('A' + r.intn(26)))
		case unicode.IsLower(ch):
			b.WriteByte(byte('a' + r.intn(26)))
		default:
			b.WriteRune(ch)
		}
	}
	return b.String()
}

func (g *Generator) Uint64(kind string) uint64 { return g.stream(kind, "", 0).next() }
