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
	switch {
	case len(digits) == 10:
		repl = fmt.Sprintf("%d%s555%s", 2+r.intn(7), r.digits(2, false), "01"+r.digits(2, false))
	case len(digits) == 11 && real[digits[0]] == '1':
		repl = fmt.Sprintf("1%d%s555%s", 2+r.intn(7), r.digits(2, false), "01"+r.digits(2, false))
	default:
		keep := 0
		if strings.HasPrefix(strings.TrimSpace(real), "+") {
			keep = min(2, len(digits))
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

var addrNumRE = regexp.MustCompile(`^\d+`)
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
	if num == "" {
		return fmt.Sprintf("%s %s Lane", r.digits(3, true), r.pick(streetNames))
	}
	suffix := "Street"
	sm := addrSuffixRE.FindStringSubmatchIndex(real)
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
