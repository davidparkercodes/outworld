package detect

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

const (
	blobRunChars = 1366
	cleanBlobMin = 1024
	blobRunMin   = 8
)

var blobFindingTypes = map[entity.Type]bool{entity.Email: true, entity.Phone: true, entity.GovID: true, entity.Financial: true, entity.Secret: true}

var blobProbe = struct{ regex *RegexDetector }{}

func init() {
	blobProbe.regex = &RegexDetector{pats: allPatterns}
}

const (
	nameRunMin    = 12
	nameRunLetter = 0.8
)

var binaryFindingSources = map[string]bool{
	"regex:credit_card": true, "regex:iban": true, "regex:vendor_token": true, "regex:api_key_prefixed": true, "regex:jwt": true,
	"regex:private_key": true, "regex:private_key_open": true, "regex:kv_secret": true, "regex:env_secret": true,
	"regex:auth_header": true, "regex:bearer": true, "regex:basic_auth": true, "regex:url_credentials": true,
}

var binaryEmailRE = regexp.MustCompile(`^[A-Za-z0-9._%+\-]{2,}@(?:[A-Za-z0-9\-]+\.)*[A-Za-z0-9\-]{2,}\.[a-z]{2,}$`)

func binaryRunsHaveFindings(runs []string) bool {
	spans, err := blobProbe.regex.Detect(context.Background(), strings.Join(runs, "\n\n"))
	if err != nil {
		return true
	}
	for _, sp := range spans {
		if !asciiOnly(sp.Text) {
			continue
		}
		switch {
		case sp.Type == entity.Email:
			if binaryEmailRE.MatchString(sp.Text) {
				return true
			}
		case sp.Type == entity.Phone, binaryFindingSources[sp.Source], strings.HasPrefix(sp.Source, "regex:ssn"):
			return true
		}
	}
	for _, run := range runs {
		if nameRun(run) || printableRunVendorKey(run) {
			return true
		}
	}
	return false
}

func nameRun(run string) bool {
	if len(run) < nameRunMin {
		return false
	}
	letters, total := 0, 0
	for _, r := range run {
		total++
		if unicode.IsLetter(r) || r == ' ' {
			letters++
		}
	}
	if float64(letters) < nameRunLetter*float64(total) {
		return false
	}
	spans, err := NameDetector{}.Detect(context.Background(), run)
	if err != nil {
		return true
	}
	for _, sp := range spans {
		if sp.Type == entity.Person && len(strings.Fields(sp.Text)) >= 2 {
			return true
		}
	}
	return false
}

func asciiOnly(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

func textHasFindings(text string) bool {
	text = stripMarkup(text)
	spans, err := blobProbe.regex.Detect(context.Background(), text)
	if err != nil {
		return true
	}
	for _, sp := range spans {
		if blobFindingTypes[sp.Type] && sp.Source != "regex:high_entropy" || sp.Type == entity.Person {
			return true
		}
	}
	people, err := NameDetector{}.Detect(context.Background(), text)
	return err != nil || len(people) > 0
}

func stripMarkup(text string) string {
	if !strings.HasPrefix(strings.TrimSpace(text), "<") {
		return text
	}
	var b strings.Builder
	b.Grow(len(text))
	depth := 0
	for i := 0; i < len(text); i++ {
		switch c := text[i]; {
		case c == '<':
			depth++
			b.WriteByte(' ')
		case c == '>' && depth > 0:
			depth--
		case depth == 0:
			b.WriteByte(c)
		}
	}
	return b.String()
}

var blobRunRE = regexp.MustCompile(`[A-Za-z0-9+/_\-]{24,}`)

const (
	vendorEdgeWindow  = 256
	runVendorBudget   = 1.8e-11
	keyBodyDistinct   = 8
	keyBodyRepeat     = 6
	alnumClass        = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	upperDigitClass   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	digitClass        = "0123456789"
	bech32UpperClass  = "QPZRY9X8GF2TVDW0S3JN54KHCE6MUA7L"
	stdBase64Alphabet = alnumClass + "+/"
	urlBase64Alphabet = alnumClass + "-_"
)

type keySeg struct {
	class    string
	min, max int
}

type keyShape struct {
	name   string
	prefix string
	segs   []keySeg
}

func lit(s string) []keySeg {
	var out []keySeg
	for _, r := range s {
		out = append(out, keySeg{string(r), 1, 1})
	}
	return out
}

var vendorKeyShapes = func() []keyShape {
	var out []keyShape
	add := func(name, prefix string, segs ...keySeg) {
		out = append(out, keyShape{name, prefix, segs})
	}
	for _, p := range []string{"AKIA", "ASIA", "ABIA", "ACCA"} {
		add("aws "+p, p, keySeg{upperDigitClass, 16, 16})
	}
	add("google AIzaSy", "AIzaSy", keySeg{alnumClass + "_-", 33, 33})
	add("google AIza", "AIza", keySeg{alnumClass + "_-", 35, 35})
	for _, p := range []string{"ghp_", "gho_", "ghu_", "ghs_", "ghr_"} {
		add("github "+p, p, keySeg{alnumClass, 36, 36})
	}
	add("github_pat_", "github_pat_", keySeg{alnumClass + "_", 40, 0})
	add("gitlab glpat-", "glpat-", keySeg{alnumClass + "_-", 20, 0})
	add("anthropic sk-ant-api03-", "sk-ant-api03-", keySeg{alnumClass + "_-", 32, 0})
	add("openai sk-proj-", "sk-proj-", keySeg{alnumClass + "_-", 32, 0})
	add("stripe sk-live-", "sk-live-", keySeg{alnumClass, 20, 0})
	add("stripe sk-test-", "sk-test-", keySeg{alnumClass, 20, 0})
	for _, p := range []string{"xoxb-", "xoxp-", "xoxa-", "xoxo-", "xoxs-", "xoxr-"} {
		add("slack "+p, p, append([]keySeg{{digitClass, 10, 13}}, append(lit("-"), keySeg{alnumClass + "-", 20, 0})...)...)
	}
	add("npm npm_", "npm_", keySeg{alnumClass, 36, 36})
	add("age AGE-SECRET-KEY-1", "AGE-SECRET-KEY-1", keySeg{bech32UpperClass, 58, 58})
	add("meta EAA", "EAA", keySeg{alnumClass, 60, 0})
	return out
}()

func (k keyShape) pattern() string {
	var b strings.Builder
	b.WriteString(regexp.QuoteMeta(k.prefix))
	for _, s := range k.segs {
		if s.min == 1 && s.max == 1 && len(s.class) == 1 {
			b.WriteString(regexp.QuoteMeta(s.class))
			continue
		}
		b.WriteString("[" + regexClass(s.class) + "]")
		switch {
		case s.max == 0:
			fmt.Fprintf(&b, "{%d,}", s.min)
		case s.max == s.min:
			fmt.Fprintf(&b, "{%d}", s.min)
		default:
			fmt.Fprintf(&b, "{%d,%d}", s.min, s.max)
		}
	}
	return b.String()
}

func regexClass(chars string) string {
	var b strings.Builder
	for _, r := range chars {
		if r == '-' || r == ']' || r == '\\' || r == '^' {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func classShare(class, alphabet string) float64 {
	n := 0
	for _, r := range alphabet {
		if strings.ContainsRune(class, r) {
			n++
		}
	}
	return float64(n) / float64(len(alphabet))
}

func (k keyShape) chance(alphabet string) float64 {
	p := 1.0
	for _, r := range k.prefix {
		p *= classShare(string(r), alphabet)
	}
	for _, s := range k.segs {
		p *= math.Pow(classShare(s.class, alphabet), float64(s.min))
	}
	return p
}

func (k keyShape) runChance() float64 {
	return max(k.chance(stdBase64Alphabet), k.chance(urlBase64Alphabet))
}

func shapesWhere(keep func(keyShape) bool) []keyShape {
	var out []keyShape
	for _, k := range vendorKeyShapes {
		if keep(k) {
			out = append(out, k)
		}
	}
	return out
}

func shapesRE(ks []keyShape, start, end bool) *regexp.Regexp {
	var alts []string
	for _, k := range ks {
		alts = append(alts, k.pattern())
	}
	re := "(?:" + strings.Join(alts, "|") + ")"
	if start {
		re = "^" + re
	}
	if end {
		re += "$"
	}
	return regexp.MustCompile(re)
}

var (
	runVendorShapes    = shapesWhere(func(k keyShape) bool { return k.runChance() <= runVendorBudget })
	runVendorRE        = shapesRE(runVendorShapes, false, false)
	gluedVendorStartRE = shapesRE(vendorKeyShapes, true, false)
	gluedVendorEndRE   = shapesRE(vendorKeyShapes, false, true)
	printableVendorRE  = shapesRE(vendorKeyShapes, false, false)
	blobVendorGate     = func() string {
		var ps []string
		for _, k := range vendorKeyShapes {
			ps = append(ps, k.prefix)
		}
		return strings.Join(ps, "|")
	}()
)

func vendorBody(m string) string {
	for _, k := range vendorKeyShapes {
		if strings.HasPrefix(m, k.prefix) {
			return m[len(k.prefix):]
		}
	}
	return m
}

func degenerateBody(body string) bool {
	counts := map[rune]int{}
	top, streak, longest := 0, 0, 0
	var prev rune
	for i, r := range body {
		counts[r]++
		top = max(top, counts[r])
		if i > 0 && r == prev {
			streak++
		} else {
			streak = 1
		}
		longest = max(longest, streak)
		prev = r
	}
	return len(counts) < keyBodyDistinct || top*2 > len(body) || longest >= keyBodyRepeat
}

func vendorMatches(re *regexp.Regexp, text string, base int) []entity.Span {
	var out []entity.Span
	for _, m := range re.FindAllStringIndex(text, -1) {
		if degenerateBody(vendorBody(text[m[0]:m[1]])) {
			continue
		}
		out = append(out, entity.Span{Start: base + m[0], End: base + m[1]})
	}
	return out
}

func scanBlobVendorKeys(text, _ string) []entity.Span {
	var out []entity.Span
	for _, run := range blobRunRE.FindAllStringIndex(text, -1) {
		s, e := run[0], run[1]
		out = append(out, vendorMatches(runVendorRE, text[s:e], s)...)
		if e-s < blobRunChars {
			continue
		}
		var spans []entity.Span
		spans = append(spans, vendorMatches(gluedVendorStartRE, text[s:min(e, s+vendorEdgeWindow)], s)...)
		tail := max(s, e-vendorEdgeWindow)
		spans = append(spans, vendorMatches(gluedVendorEndRE, text[tail:e], tail)...)
		out = append(out, spans...)
	}
	return out
}

func printableRunVendorKey(run string) bool {
	return len(vendorMatches(printableVendorRE, run, 0)) > 0
}

var blobEncodings = []*base64.Encoding{base64.StdEncoding.Strict(), base64.RawStdEncoding.Strict(), base64.URLEncoding.Strict(), base64.RawURLEncoding.Strict()}

var highEntropyRE = regexp.MustCompile(`[A-Za-z0-9_\-+/]{24,}={0,2}`)

func highEntropyBlobValid(text string, s, e int) bool {
	v := text[s:e]
	if _, ok := binaryBlob(v); ok {
		return false
	}
	if raw, ok := shortDecoded(v); ok {
		switch {
		case keyMaterial(raw):
			return !slugPath(v) && highEntropyContextOK(text, s, e)
		case publicDER(raw):
			return false
		}
	}
	return highEntropyValid(text, s, e)
}

func shortDecoded(v string) ([]byte, bool) {
	for _, enc := range blobEncodings {
		if raw, err := enc.DecodeString(v); err == nil && len(raw) > 0 && len(raw) < cleanBlobMin {
			return raw, true
		}
	}
	return nil, false
}

var oidSignedData = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}

func publicDERPrefix(raw []byte) bool {
	outer, _, ok := readTLV(raw)
	if !ok || outer.tag != 0x30 {
		return false
	}
	el := derChildren(outer.body, 2)
	if len(el) == 0 {
		return false
	}
	switch el[0].tag {
	case 0x06:
		return oidOf(el[0]).Equal(oidSignedData)
	case 0x30:
		inner := derChildren(el[0].body, 2)
		if len(inner) == 0 {
			return false
		}
		if inner[0].tag == 0xa0 {
			v := derChildren(inner[0].body, 1)
			n, ok := 0, false
			if len(v) == 1 {
				n, ok = smallInt(v[0])
			}
			return ok && n <= 2
		}
		if oidIn(oidOf(inner[0]), oidKeyAlgorithms) {
			return len(el) < 2 || el[1].tag == 0x03
		}
		if n, ok := smallInt(inner[0]); ok && n == 0 && len(inner) > 1 {
			return inner[1].tag == 0x30
		}
	}
	return false
}

func publicDER(raw []byte) bool {
	if len(raw) == 0 || raw[0] != 0x30 {
		return false
	}
	if publicDERPrefix(raw) {
		return true
	}
	if _, err := x509.ParseCertificate(raw); err == nil {
		return true
	}
	if _, err := x509.ParseCertificateRequest(raw); err == nil {
		return true
	}
	if _, err := x509.ParsePKIXPublicKey(raw); err == nil {
		return true
	}
	var ci struct {
		Type    asn1.ObjectIdentifier
		Content asn1.RawValue `asn1:"explicit,tag:0"`
	}
	rest, err := asn1.Unmarshal(raw, &ci)
	return err == nil && len(rest) == 0 && ci.Type.Equal(oidSignedData)
}

func blobRunSpans(text string) [][2]int {
	if len(text) < blobRunChars {
		return nil
	}
	var runs [][2]int
	for _, m := range highEntropyRE.FindAllStringIndex(text, -1) {
		if m[1]-m[0] < blobRunChars {
			continue
		}
		if _, ok := binaryBlob(text[m[0]:m[1]]); ok {
			runs = append(runs, [2]int{m[0], m[1]})
		}
	}
	return runs
}

func (p *pattern) outsideBlobs(spans []entity.Span, runs [][2]int) []entity.Span {
	if p.inBlob || len(runs) == 0 {
		return spans
	}
	out := spans[:0]
	for _, sp := range spans {
		if !insideBlobRun(sp, runs) {
			out = append(out, sp)
		}
	}
	return out
}

func insideBlobRun(sp entity.Span, runs [][2]int) bool {
	i := sort.Search(len(runs), func(i int) bool { return runs[i][1] > sp.Start })
	if i == len(runs) {
		return false
	}
	r := runs[i]
	return sp.Start >= r[0] && sp.End <= r[1] && (sp.Start != r[0] || sp.End != r[1])
}

func scanBlobSecrets(text string, l InflateLimits) []entity.Span {
	var out []entity.Span
	budget := NewInflateBudgetFor(l)
	for _, m := range highEntropyRE.FindAllStringIndex(text, -1) {
		s, e := m[0], m[1]
		raw, ok := binaryBlob(text[s:e])
		if !ok || !boundaryOK(text, s, e) || slugPath(text[s:e]) || !highEntropyContextOK(text, s, e) {
			continue
		}
		if !blobClean(raw, budget) {
			out = append(out, entity.Span{Start: s, End: e})
		}
	}
	return out
}

func binaryBlob(v string) ([]byte, bool) {
	if base64.RawStdEncoding.DecodedLen(len(strings.TrimRight(v, "="))) < cleanBlobMin {
		return nil, false
	}
	for _, enc := range blobEncodings {
		raw, err := enc.DecodeString(v)
		if err != nil {
			continue
		}
		if len(raw) < cleanBlobMin || printableBytes(raw) {
			return nil, false
		}
		return raw, true
	}
	return nil, false
}

func blobClean(raw []byte, budget *InflateBudget) bool {
	key := blobMemoKey{sha256.Sum256(raw), budget.blobMax()}
	v, ok := blobOutcomes.get(key)
	if !ok {
		v = judgeBlob(raw, budget.blobMax())
		blobOutcomes.put(key, v)
	}
	if v.inflated > budget.Left {
		budget.Left = 0
		return false
	}
	budget.Left -= v.inflated
	return v.clean
}

func judgeBlob(raw []byte, blobMax int) blobOutcome {
	if keyMaterial(raw) {
		return blobOutcome{}
	}
	own := &InflateBudget{Left: blobMax, Blob: blobMax}
	parts, st := Inflate(raw, own)
	v := blobOutcome{inflated: blobMax - own.Left}
	if st == Unreadable {
		return v
	}
	for _, p := range parts {
		if !inflatedClean(p) {
			return v
		}
	}
	v.clean = !binaryRunsHaveFindings(blobRuns(raw))
	return v
}

const blobMemoSize = 64

type blobMemoKey struct {
	sum  [32]byte
	blob int
}

type blobOutcome struct {
	clean    bool
	inflated int
}

type blobMemo struct {
	mu    sync.Mutex
	m     map[blobMemoKey]blobOutcome
	order []blobMemoKey
}

var blobOutcomes = &blobMemo{m: map[blobMemoKey]blobOutcome{}}

func (c *blobMemo) get(k blobMemoKey) (blobOutcome, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.m[k]
	return v, ok
}

func (c *blobMemo) put(k blobMemoKey, v blobOutcome) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.m[k]; ok {
		return
	}
	if len(c.order) >= blobMemoSize {
		delete(c.m, c.order[0])
		c.order = c.order[1:]
	}
	c.m[k] = v
	c.order = append(c.order, k)
}

func inflatedClean(p []byte) bool {
	if keyMaterial(p) {
		return false
	}
	if utf8.Valid(p) && textShare(p) {
		return !textHasFindings(string(p))
	}
	return !binaryRunsHaveFindings(blobRuns(p))
}

func textShare(p []byte) bool {
	text := 0
	for _, r := range string(p) {
		if unicode.IsPrint(r) || r == '\t' || r == '\n' || r == '\r' {
			text++
		}
	}
	return text*10 >= utf8.RuneCount(p)*9
}

func printableBytes(raw []byte) bool {
	for _, b := range raw {
		if b != '\n' && b != '\r' && b != '\t' && (b < 0x20 || b > 0x7e) {
			return false
		}
	}
	return true
}

var opensshKeyMagic = []byte("openssh-key-v1\x00")

const (
	keyJunkMax     = 4
	keyIntMin      = 64
	sec1KeyMin     = 28
	sec1KeyMax     = 66
	sshMagicWindow = 16
	keyContentMin  = 64
)

var (
	oidPKCS7Data      = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}
	oidPKCS7Encrypted = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 6}
	oidPBEFamilies    = []asn1.ObjectIdentifier{{1, 2, 840, 113549, 1, 5}, {1, 2, 840, 113549, 1, 12, 1}}
	oidKeyAlgorithms  = []asn1.ObjectIdentifier{
		{1, 2, 840, 113549, 1, 1, 1}, {1, 2, 840, 10045, 2, 1}, {1, 3, 101, 110}, {1, 3, 101, 111}, {1, 3, 101, 112}, {1, 3, 101, 113},
		{1, 2, 840, 10040, 4, 1}, {1, 2, 840, 113549, 1, 3, 1}, {1, 2, 840, 10046, 2, 1}, {1, 2, 840, 113549, 1, 1, 10},
	}
)

func keyMaterial(raw []byte) bool {
	if bytes.Contains(raw, sshKeyAnchor) {
		return true
	}
	for off := 0; off <= keyJunkMax && off < len(raw); off++ {
		if raw[off] == 0x30 && pkcs12Shape(raw[off:]) {
			return true
		}
	}
	for _, a := range keyAnchors {
		for i := 0; ; {
			j := bytes.Index(raw[i:], a.prefix)
			if j < 0 {
				break
			}
			if a.check(raw, i+j) {
				return true
			}
			i += j + 1
		}
	}
	return false
}

var sshKeyAnchor = opensshKeyMagic[4:]

type keyAnchor struct {
	name   string
	prefix []byte
	check  func(b []byte, at int) bool
	fixed  int
}

var rsaModulusHeaders = [][]byte{{0x81, 0x80}, {0x81, 0x81}, {0x82, 0x01, 0x00}, {0x82, 0x01, 0x01}, {0x82, 0x01, 0x80}, {0x82, 0x01, 0x81}, {0x82, 0x02, 0x00}, {0x82, 0x02, 0x01}, {0x82, 0x03, 0x00}, {0x82, 0x03, 0x01}, {0x82, 0x04, 0x00}, {0x82, 0x04, 0x01}}

func tlvLen(h []byte) int {
	if h[0] < 0x80 {
		return int(h[0])
	}
	n := 0
	for _, c := range h[1:] {
		n = n<<8 | int(c)
	}
	return n
}

func nextTagAt(b []byte, at int, tags ...byte) bool {
	if at >= len(b) {
		return true
	}
	for _, t := range tags {
		if b[at] == t {
			return true
		}
	}
	return false
}

func pkcs1At(b []byte, at int) bool {
	rest := b[at+4:]
	for _, h := range rsaModulusHeaders {
		if bytes.HasPrefix(rest, h) {
			return nextTagAt(b, at+4+len(h)+tlvLen(h), 0x02)
		}
	}
	return false
}

func keyOIDBodies(oids []asn1.ObjectIdentifier) [][]byte {
	var out [][]byte
	for _, o := range oids {
		der, _ := asn1.Marshal(o)
		out = append(out, der)
	}
	return out
}

var keyAlgorithmDER = keyOIDBodies(oidKeyAlgorithms)

func pkcs8At(b []byte, at int) bool {
	seq := b[at+3:]
	if len(seq) < 2 {
		return false
	}
	hl := 2
	if seq[1] == 0x81 {
		hl = 3
	}
	if len(seq) < hl || seq[1] > 0x81 {
		return false
	}
	body := seq[hl:]
	for _, oid := range keyAlgorithmDER {
		if bytes.HasPrefix(body, oid) {
			return nextTagAt(b, at+3+hl+tlvLen(seq[1:hl]), 0x04)
		}
	}
	return false
}

func sec1At(b []byte, at int) bool {
	if len(b) < at+5 {
		return false
	}
	n := int(b[at+4])
	if n != 0x20 && n != 0x30 && n != 0x42 {
		return false
	}
	return nextTagAt(b, at+5+n, 0xa0, 0xa1)
}

func encryptedPKCS8At(b []byte, at int) bool {
	if at < 2 {
		return false
	}
	var seqStart int
	switch {
	case b[at-2] == 0x30:
		seqStart = at - 2
	case at >= 3 && b[at-3] == 0x30 && b[at-2] == 0x81:
		seqStart = at - 3
	default:
		return false
	}
	hl := at - seqStart
	return nextTagAt(b, seqStart+hl+tlvLen(b[seqStart+1:at]), 0x04)
}

var keyAnchors = []keyAnchor{
	{"pkcs1 rsa or dsa", []byte{0x02, 0x01, 0x00, 0x02}, pkcs1At, 4 + 2 + 1},
	{"pkcs8 v1", []byte{0x02, 0x01, 0x00, 0x30}, pkcs8At, 4 + 1 + 5 + 1},
	{"pkcs8 v2", []byte{0x02, 0x01, 0x01, 0x30}, pkcs8At, 4 + 1 + 5 + 1},
	{"sec1 ec", []byte{0x02, 0x01, 0x01, 0x04}, sec1At, 4 + 1 + 1},
	{"encrypted pkcs8 pbes", []byte{0x06, 0x09, 0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x01, 0x05}, encryptedPKCS8At, 10 + 1 + 1 + 1},
	{"encrypted pkcs8 pkcs12 pbe", []byte{0x06, 0x0a, 0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x01, 0x0c, 0x01}, encryptedPKCS8At, 11 + 1 + 1 + 1},
}

type derTLV struct {
	tag      byte
	body     []byte
	declared int
	whole    bool
}

func readTLV(b []byte) (derTLV, []byte, bool) {
	if len(b) < 2 {
		return derTLV{}, nil, false
	}
	t := derTLV{tag: b[0]}
	n, i := int(b[1]), 2
	if n&0x80 != 0 {
		k := n & 0x7f
		if k == 0 || k > 4 || len(b) < 2+k {
			return derTLV{}, nil, false
		}
		n = 0
		for _, c := range b[2 : 2+k] {
			n = n<<8 | int(c)
		}
		i += k
	}
	t.declared = n
	if len(b)-i < n {
		t.body = b[i:]
		return t, nil, true
	}
	t.body, t.whole = b[i:i+n], true
	return t, b[i+n:], true
}

func derChildren(b []byte, want int) []derTLV {
	var out []derTLV
	for len(b) > 0 && len(out) < want {
		t, rest, ok := readTLV(b)
		if !ok {
			break
		}
		out = append(out, t)
		if !t.whole {
			break
		}
		b = rest
	}
	return out
}

func smallInt(t derTLV) (int, bool) {
	if t.tag != 0x02 || t.declared != 1 || len(t.body) != 1 {
		return 0, false
	}
	return int(t.body[0]), true
}

func oidOf(t derTLV) asn1.ObjectIdentifier {
	if t.tag != 0x06 || !t.whole {
		return nil
	}
	var oid asn1.ObjectIdentifier
	full := append([]byte{0x06, byte(len(t.body))}, t.body...)
	if len(t.body) > 127 {
		return nil
	}
	if _, err := asn1.Unmarshal(full, &oid); err != nil {
		return nil
	}
	return oid
}

func oidUnder(oid asn1.ObjectIdentifier, families []asn1.ObjectIdentifier) bool {
	for _, f := range families {
		if len(oid) > len(f) && oid[:len(f)].Equal(f) {
			return true
		}
	}
	return false
}

func oidIn(oid asn1.ObjectIdentifier, set []asn1.ObjectIdentifier) bool {
	for _, o := range set {
		if oid.Equal(o) {
			return true
		}
	}
	return false
}

func algOID(t derTLV) asn1.ObjectIdentifier {
	if t.tag != 0x30 {
		return nil
	}
	el := derChildren(t.body, 1)
	if len(el) == 0 {
		return nil
	}
	return oidOf(el[0])
}

func pkcs12Shape(b []byte) bool {
	outer, _, ok := readTLV(b)
	if !ok || outer.tag != 0x30 {
		return false
	}
	el := derChildren(outer.body, 2)
	if len(el) < 2 {
		return false
	}
	if v, ok := smallInt(el[0]); !ok || v != 3 || el[1].tag != 0x30 {
		return false
	}
	ci := derChildren(el[1].body, 2)
	if len(ci) < 2 || ci[1].tag != 0xa0 || ci[1].declared < keyContentMin {
		return false
	}
	oid := oidOf(ci[0])
	return oid.Equal(oidPKCS7Data) || oid.Equal(oidPKCS7Encrypted)
}

func blobRuns(raw []byte) []string {
	var runs []string
	start, count := -1, 0
	for i := 0; i <= len(raw); {
		r, n := utf8.RuneError, 1
		if i < len(raw) {
			r, n = utf8.DecodeRune(raw[i:])
		}
		ok := r != utf8.RuneError && (unicode.IsPrint(r) || r == '\t' || r == '\n' || r == '\r')
		switch {
		case ok && start < 0:
			start, count = i, 1
		case ok:
			count++
		case start >= 0:
			if count >= blobRunMin {
				runs = append(runs, string(raw[start:i]))
			}
			start = -1
		}
		i += n
	}
	return append(runs, WideRuns(raw)...)
}

func asciiTextByte(c byte) bool {
	return c >= 0x20 && c <= 0x7e || c == '\t' || c == '\n' || c == '\r'
}

func WideRuns(raw []byte) []string {
	var out []string
	var b strings.Builder
	count, joined := 0, false
	flush := func() {
		if joined && count >= blobRunMin {
			out = append(out, b.String())
		}
		b.Reset()
		count, joined = 0, false
	}
	for i, c := range raw {
		switch {
		case asciiTextByte(c):
			b.WriteByte(c)
			count++
		case c == 0 && count > 0 && i+1 < len(raw) && asciiTextByte(raw[i+1]):
			joined = true
		default:
			flush()
		}
	}
	flush()
	return out
}
