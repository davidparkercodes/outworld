package gateway

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/config"
	"github.com/davidparkercodes/outworld/pkg/detect"
	"github.com/davidparkercodes/outworld/pkg/outworld"
)

const (
	textShare       = 0.80
	textLetterShare = 0.40
	opaqueMinLen    = 128
	looseOpaqueMin  = 96
	minDecodeLen    = 16
	minBinaryRun    = 8
	hexTextMinLen   = 16
	shortHexBytes   = 16
	blockSubstrMin  = 6
	secretContext   = 16
)

const runSeparator = "\x00"

const tokenClass = `A-Za-z0-9_+/=.\-`

var (
	secretCandidateRE = regexp.MustCompile(`[` + tokenClass + `]{20,}`)
	vendorPrefixRE    = regexp.MustCompile(`(?:sk-|sk_|pk_|rk_|ghp_|gho_|ghu_|ghs_|ghr_|github_pat_|xox[abposr]-|AKIA|ASIA|AIza|glpat-|hf_|npm_|SG\.|eyJ)[` + tokenClass + `]{12,}`)
	tokenPayloadRE    = regexp.MustCompile(`^[` + tokenClass + `:]+$`)
	hexRE             = regexp.MustCompile(`^[0-9a-fA-F]+$`)
	looseBase64RE     = regexp.MustCompile(`^[A-Za-z0-9+/_-]+={0,2}$`)
)

var binarySources = map[string]bool{
	"regex:vendor_token": true, "regex:api_key_prefixed": true, "regex:private_key": true,
	"regex:private_key_open": true, "regex:jwt": true,
}

func binaryFinding(v outworld.Violation) bool { return binarySources[v.Source] }

func formatRune(r rune) bool { return unicode.Is(unicode.Cf, r) }

func printableRune(r rune) bool {
	return r != utf8.RuneError && (unicode.IsPrint(r) || formatRune(r) || r == '\n' || r == '\r' || r == '\t')
}

func printableShare(b []byte) float64 {
	if len(b) == 0 {
		return 0
	}
	good := 0
	for i := 0; i < len(b); {
		r, n := utf8.DecodeRune(b[i:])
		if printableRune(r) {
			good += n
		}
		i += n
	}
	return float64(good) / float64(len(b))
}

func letterShare(b []byte) float64 {
	printable, letters := 0, 0
	for i := 0; i < len(b); {
		r, n := utf8.DecodeRune(b[i:])
		i += n
		if !printableRune(r) || formatRune(r) {
			continue
		}
		printable++
		if unicode.IsLetter(r) || r == ' ' {
			letters++
		}
	}
	if printable == 0 {
		return 0
	}
	return float64(letters) / float64(printable)
}

const (
	cleanTextShare = 0.97
	frameMinLen    = 64
	frameMaxField  = 65535
	fixedFieldMax  = 0.5
	frameMaxDepth  = 4
	frameMaxFields = 4096
)

type frame struct {
	payloads [][]byte
	framed   bool
	limit    bool
}

func parseFrame(b []byte) frame {
	payloads, ok, limit := walkFrame(b, frameMinLen)
	return frame{payloads: payloads, framed: ok, limit: limit}
}

func walkFrame(b []byte, minLen int) ([][]byte, bool, bool) {
	if len(b) < minLen {
		return nil, false, false
	}
	var payloads [][]byte
	fields, fixed := 0, 0
	for i := 0; i < len(b); {
		key, n := binary.Uvarint(b[i:])
		if n <= 0 || key>>3 == 0 || key>>3 > frameMaxField {
			return nil, false, false
		}
		i += n
		fields++
		if fields > frameMaxFields {
			return nil, false, true
		}
		switch key & 7 {
		case 0:
			_, m := binary.Uvarint(b[i:])
			if m <= 0 {
				return nil, false, false
			}
			i += m
		case 1:
			i += 8
			fixed++
		case 2:
			l, m := binary.Uvarint(b[i:])
			if m <= 0 || l > uint64(len(b)-i-m) {
				return nil, false, false
			}
			payloads = append(payloads, b[i+m:i+m+int(l)])
			i += m + int(l)
		case 5:
			i += 4
			fixed++
		default:
			return nil, false, false
		}
		if i > len(b) {
			return nil, false, false
		}
	}
	return payloads, float64(fixed) <= fixedFieldMax*float64(fields), false
}

func textual(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	ps := printableShare(b)
	if len(b) < frameMinLen {
		return ps >= textShare
	}
	return ps >= cleanTextShare || ps >= textShare && letterShare(b) >= textLetterShare
}

type frameKey struct {
	data *byte
	size int
}

func (f *fields) frameOf(b []byte) frame {
	if len(b) == 0 {
		return frame{}
	}
	key := frameKey{&b[0], len(b)}
	if fr, ok := f.frames[key]; ok {
		return fr
	}
	fr := parseFrame(b)
	f.frames[key] = fr
	return fr
}

func (f *fields) binaryBlob(b []byte) bool { return f.frameOf(b).framed || !textual(b) }

func stdBase64(v any) ([]byte, bool) {
	s, ok := v.(string)
	if !ok || len(s) < minDecodeLen || len(s)%4 != 0 || !stdAlphabet(s) {
		return nil, false
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil || len(b) == 0 {
		return nil, false
	}
	return b, true
}

func stdAlphabet(s string) bool {
	pad := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '=':
			pad++
		case pad > 0:
			return false
		case isUpperASCII(c), isLowerASCII(c), isDigitASCII(c), c == '+', c == '/':
		default:
			return false
		}
	}
	return pad <= 2
}

func base64Decode(v any, looseMin int) ([]byte, bool) {
	if b, ok := stdBase64(v); ok {
		return b, true
	}
	s, ok := v.(string)
	if !ok || len(s) < looseMin || !looseBase64RE.MatchString(s) {
		return nil, false
	}
	for _, enc := range []*base64.Encoding{base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if b, err := enc.DecodeString(s); err == nil && len(b) > 0 {
			return b, true
		}
	}
	return nil, false
}

func hexDecode(s string) ([]byte, bool) {
	if len(s) < hexTextMinLen || len(s)%2 != 0 || !hexRE.MatchString(s) {
		return nil, false
	}
	b, err := hex.DecodeString(s)
	return b, err == nil
}

func asciiPrintable(b []byte) bool {
	for _, c := range b {
		if c < 0x20 || c >= 0x7f {
			return false
		}
	}
	return true
}

func hexTextual(s string) ([]byte, bool) {
	b, ok := hexDecode(s)
	return b, ok && printableShare(b) >= textShare && (len(b) >= shortHexBytes || asciiPrintable(b))
}

func cleanText(b []byte) string { return cleanTextAt(b, false) }

func cleanTextAt(b []byte, spaceFormat bool) string {
	var out strings.Builder
	broken := false
	for i := 0; i < len(b); {
		r, n := utf8.DecodeRune(b[i:])
		i += n
		switch {
		case formatRune(r):
			if spaceFormat && !broken {
				out.WriteByte(' ')
			}
		case printableRune(r):
			out.WriteRune(r)
			broken = false
		case !broken:
			out.WriteByte('\n')
			broken = true
		}
	}
	return out.String()
}

const formatWindow = 128

func spacedWindows(b []byte) (string, []int) {
	var parts []string
	var cuts []int
	runes := 0
	lo, hi := -1, -1
	flush := func() {
		if lo < 0 {
			return
		}
		for lo > 0 && !utf8.RuneStart(b[lo]) {
			lo--
		}
		for hi < len(b) && !utf8.RuneStart(b[hi]) {
			hi++
		}
		if len(parts) > 0 {
			runes += utf8.RuneCountInString(payloadSeparator)
		}
		part := cleanTextAt(b[lo:hi], true)
		if lo > 0 {
			cuts = append(cuts, runes)
		}
		runes += utf8.RuneCountInString(part)
		if hi < len(b) {
			cuts = append(cuts, runes)
		}
		parts = append(parts, part)
	}
	for i := 0; i < len(b); {
		r, n := utf8.DecodeRune(b[i:])
		if formatRune(r) {
			s, e := max(0, i-formatWindow), min(len(b), i+n+formatWindow)
			if lo >= 0 && s <= hi {
				hi = e
			} else {
				flush()
				lo, hi = s, e
			}
		}
		i += n
	}
	flush()
	return strings.Join(parts, payloadSeparator), cuts
}

var groupedCardRE = regexp.MustCompile(`[0-9]{4}[ -][0-9]{4}[ -][0-9]{4}[ -][0-9]{1,7}|[0-9]{4}[ -][0-9]{6}[ -][0-9]{4,5}`)

const (
	cardMinDigits = 13
	cardMaxDigits = 19
	hexIDMinLen   = 24
)

const cardEdgeTrim = 2

func gluedCards(text string) []string { return cardsIn(text, 0) }

func cardsIn(text string, trim int) []string {
	var out []string
	for _, m := range groupedCardRE.FindAllString(text, -1) {
		for cut := len(m); cut > 0 && isDigitASCII(m[cut-1]); cut-- {
			if detect.CardNumber(digitsIn(m[:cut])) {
				out = append(out, " "+m[:cut]+" ")
				break
			}
		}
	}
	for i := 0; i < len(text); {
		if !isAlnumASCII(text[i]) {
			i++
			continue
		}
		j, hex := i, true
		for j < len(text) && isAlnumASCII(text[j]) {
			hex = hex && isHexASCII(text[j])
			j++
		}
		if j-i < hexIDMinLen || !hex {
			out = append(out, cardsInToken(text[i:j], trim)...)
		}
		i = j
	}
	return out
}

func cardsInToken(token string, trim int) []string {
	var out []string
	for k := 0; k < len(token); {
		if !isDigitASCII(token[k]) {
			k++
			continue
		}
		e := k
		for e < len(token) && isDigitASCII(token[e]) {
			e++
		}
		if e-k >= cardMinDigits {
			if d, ok := cardWithin(token[k:e], trim); ok {
				out = append(out, " "+d+" ")
			}
		}
		k = e
	}
	return out
}

func isHexASCII(c byte) bool { return isDigitASCII(c) || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F' }

var dashedSSNRE = regexp.MustCompile(`[0-9]{3}-[0-9]{2}-[0-9]{4}`)

func gluedSSNs(text string) []string {
	var out []string
	for _, m := range dashedSSNRE.FindAllStringIndex(text, -1) {
		before := m[0] > 0 && text[m[0]-1] == '-'
		after := m[1] < len(text) && text[m[1]] == '-'
		if !before && !after {
			out = append(out, " "+text[m[0]:m[1]]+" ")
		}
	}
	return out
}

func cardWithin(run string, trim int) (string, bool) {
	for a := 0; a <= trim; a++ {
		for b := 0; a+b <= len(run) && b <= trim; b++ {
			d := run[a : len(run)-b]
			if len(d) >= cardMinDigits && len(d) <= cardMaxDigits && detect.CardNumber(d) {
				return d, true
			}
		}
	}
	return "", false
}

var separatedNANPRE = regexp.MustCompile(`(?:\([0-9]{3}\) ?|[0-9]{3}[-. ])[0-9]{3}[-. ][0-9]{4}`)

const phoneContext = 32

func gluedPhones(text string) []string {
	var out []string
	for _, m := range separatedNANPRE.FindAllStringIndex(text, -1) {
		if !detect.SeparatedPhone(text[m[0]:m[1]]) {
			continue
		}
		lo := max(0, m[0]-phoneContext)
		if i := strings.LastIndexByte(text[lo:m[0]], 0); i >= 0 {
			lo += i + 1
		}
		out = append(out, text[lo:m[0]]+" "+text[m[0]:m[1]]+" ")
	}
	return out
}

func isAlnumASCII(c byte) bool { return isDigitASCII(c) || isLowerASCII(c) || isUpperASCII(c) }

func digitsIn(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if isDigitASCII(s[i]) {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func (f *fields) textLeaves(msg int, path string, mode leafMode, b []byte) {
	text := cleanTextAt(b, false)
	f.add(msg, path, mode, text, nil)
	for _, c := range gluedCards(text) {
		f.add(msg, path+"(card)", mode, c, nil)
	}
	f.keyLeaves(msg, path, mode, text)
	if spaced, cuts := spacedWindows(b); spaced != "" {
		f.addCut(msg, path+"(spaced)", mode, spaced, cuts)
	}
}

var (
	phoneKeyValueRE  = regexp.MustCompile(identifierKey + keyValueSep + `([+(]?[0-9][0-9()+ .~*/\-]{6,18}[0-9])`)
	secretKeyValueRE = regexp.MustCompile(identifierKey + `(\s{0,8}->\s{0,8}|\s{1,8}is\s{1,8}|\s{0,8}:\s{0,8}\n\s{0,8}(?:-\s{1,8})?|[ \t]{1,8}|["'\x60]?\s{0,8}[:=]\s{0,8}["'\x60]?)` + secretValue)
	secretValueRE    = regexp.MustCompile(secretValue)
)

const (
	secretValue    = `([A-Za-z0-9+/_\-.]{16,512})`
	keyedCopiesMax = 1 << 16
)

func (f *fields) keyLeaves(msg int, path string, mode leafMode, text string) {
	var copies []string
	seen := map[string]bool{}
	add := func(c string) bool {
		if seen[c] {
			return true
		}
		seen[c] = true
		copies = append(copies, c)
		return len(copies) <= keyedCopiesMax
	}
	crossesFrame := func(m []string) bool { return mode == modeNoisyFrame && strings.ContainsRune(m[0], '\n') }
	eachKeyValue(phoneKeyValueRE, text, func(m []string, _ int) bool {
		if !phoneKey(m[1]) || crossesFrame(m) {
			return false
		}
		return add("phone " + m[2] + " ")
	})
	eachKeyValue(secretKeyValueRE, text, func(m []string, at int) bool {
		if !secretIdentifier(m[1], m[2]) || crossesFrame(m) || plainWordBefore(text, at, m[1]) || mode == modeNoisyFrame && strings.ContainsAny(m[2], ":=") {
			return false
		}
		return add(assignedKey(m[1]) + "=" + m[3] + " ")
	})
	structuredSecrets(text, mode == modeNoisyFrame, func(key, value string) bool { return add(key + "=" + value + " ") })
	if len(copies) > keyedCopiesMax {
		f.add(msg, path, modeOversize, oversizeKeyed, nil)
		return
	}
	if len(copies) > 0 {
		f.add(msg, path+"(keyed)", mode, strings.Join(copies, payloadSeparator), nil)
	}
}

func keyOwner(key, prev, inherit string) string {
	parts := keyParts(key)
	if len(parts) == 0 || !secretKeyWords[strings.ToLower(parts[len(parts)-1])] {
		return inherit
	}
	if !secretIdentifier(key, ":") || len(parts) == 1 && plainKeyRE.MatchString(prev+" ") {
		return ""
	}
	return assignedKey(key)
}

type ownerRun struct {
	owner string
	count int
}

type ownerStack []ownerRun

func (st *ownerStack) push(owner string) {
	if n := len(*st); n > 0 && (*st)[n-1].owner == owner {
		(*st)[n-1].count++
		return
	}
	*st = append(*st, ownerRun{owner, 1})
}

func (st *ownerStack) pop() {
	n := len(*st)
	if n == 0 || n == 1 && (*st)[0].count == 1 {
		return
	}
	if (*st)[n-1].count > 1 {
		(*st)[n-1].count--
		return
	}
	*st = (*st)[:n-1]
}

func (st ownerStack) top() string { return st[len(st)-1].owner }

var tableHeaderRE = regexp.MustCompile(`^\[{1,2}[ \t]{0,8}([A-Za-z0-9_\-]{1,64}(?:\.[A-Za-z0-9_\-]{1,64}){0,8})[ \t]{0,8}\]{1,2}[ \t]{0,8}(?:\r?\n|$)`)

func tableHeader(text string, at int) (string, int, bool) {
	if at >= len(text) || text[at] != '[' {
		return "", 0, false
	}
	m := tableHeaderRE.FindStringSubmatchIndex(text[at:min(len(text), at+640)])
	if m == nil {
		return "", 0, false
	}
	end := at + m[1]
	if end > at && text[end-1] == '\n' {
		end--
	}
	return text[at+m[2] : at+m[3]], end, true
}

func opensScalar(prev byte) bool { return prev == 0 || strings.IndexByte(":=[{,-\n", prev) >= 0 }

func structuredSecrets(text string, glueBreaks bool, emit func(key, value string) bool) {
	stack := ownerStack{{"", 1}}
	scalar, hasScalar, prev := "", false, ""
	pending, pendingSet := "", false
	type blockFrame struct {
		indent int
		owner  string
	}
	var blocks []blockFrame
	lineStart, itemLine, indent := true, false, 0
	stopped := false
	var last byte
	inherited := func() string {
		if len(stack) == 1 && stack[0].count == 1 && len(blocks) > 0 {
			return blocks[len(blocks)-1].owner
		}
		return stack.top()
	}
	owner := func() string {
		if pendingSet {
			return pending
		}
		return inherited()
	}
	flush := func() {
		if !hasScalar {
			return
		}
		if o := owner(); o != "" && !stopped {
			for _, v := range secretValueRE.FindAllString(scalar, -1) {
				if !emit(o, v) {
					stopped = true
					break
				}
			}
		}
		prev, hasScalar, pendingSet = scalar, false, false
	}
	newline := func() {
		flush()
		if pendingSet {
			blocks = append(blocks, blockFrame{indent, pending})
			pendingSet = false
		}
		lineStart, itemLine, last = true, false, '\n'
	}
	for i := 0; i < len(text); {
		if lineStart {
			j := i
			for j < len(text) && (text[j] == ' ' || text[j] == '\t' && !glueBreaks) {
				j++
			}
			lineStart = false
			if j < len(text) && (text[j] == '\n' || text[j] == '\r') {
				i = j
				continue
			}
			indent = j - i
			if name, end, ok := tableHeader(text, j); ok {
				parts := strings.Split(name, ".")
				blocks = []blockFrame{{-1, keyOwner(parts[len(parts)-1], prev, "")}}
				i, last = end, ']'
				continue
			}
			itemLine = j+1 < len(text) && text[j] == '-' && (text[j+1] == ' ' || text[j+1] == '\t')
			for len(blocks) > 0 && (blocks[len(blocks)-1].indent > indent || blocks[len(blocks)-1].indent == indent && !itemLine) {
				blocks = blocks[:len(blocks)-1]
			}
			if itemLine {
				i, last = j+2, '-'
				continue
			}
			i = j
			continue
		}
		c := text[i]
		switch {
		case c == '\n' || glueBreaks && (c == '\r' || c == '\t'):
			newline()
			i++
			continue
		case c == ' ' || c == '\t' || c == '\r':
			i++
			continue
		case c == ':' || c == '=':
			if hasScalar {
				pending, pendingSet = keyOwner(scalar, prev, inherited()), true
				prev, hasScalar = scalar, false
			}
		case c == '[' || c == '{':
			flush()
			o := owner()
			pendingSet = false
			stack.push(o)
		case c == ']' || c == '}':
			flush()
			pendingSet = false
			stack.pop()
		case c == ',':
			flush()
			pendingSet = false
		case (c == '"' || c == '\'') && opensScalar(last):
			flush()
			j := i + 1
			for j < len(text) && text[j] != c && text[j] != '\n' {
				if text[j] == '\\' {
					j++
				}
				j++
			}
			scalar, hasScalar = text[i+1:min(j, len(text))], true
			last = c
			if j < len(text) && text[j] == c {
				j++
			}
			i = min(j, len(text))
			continue
		default:
			flush()
			j := i
			for j < len(text) && !strings.ContainsRune(wordStop, rune(text[j])) {
				j++
			}
			scalar, hasScalar = text[i:max(j, i+1)], true
			i = max(j, i+1)
			last = 'a'
			continue
		}
		last = c
		i++
	}
	flush()
}

const wordStop = " \t\r\n:=[]{},"

func plainWordBefore(text string, at int, key string) bool {
	return len(keyParts(key)) == 1 && plainKeyRE.MatchString(text[max(0, at-secretKeyWindow):at])
}

func eachKeyValue(re *regexp.Regexp, text string, use func([]string, int) bool) {
	for pos := 0; pos < len(text); {
		loc := re.FindStringSubmatchIndex(text[pos:])
		if loc == nil {
			return
		}
		groups := make([]string, len(loc)/2)
		for i := range groups {
			groups[i] = text[pos+loc[2*i] : pos+loc[2*i+1]]
		}
		if keyStart(text, pos+loc[2]) && use(groups, pos+loc[2]) {
			pos += loc[1]
		} else {
			pos += loc[3]
		}
	}
}

func keyStart(text string, at int) bool {
	r, _ := utf8.DecodeLastRuneInString(text[:at])
	return at == 0 || !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

func assignedKey(k string) string {
	if strings.HasSuffix(strings.ToLower(k), "s") {
		k = k[:len(k)-1]
	}
	if len(keyParts(k)) > 1 {
		return k
	}
	if one := strings.ToLower(k); one != "key" {
		return one
	}
	return "api_key"
}

func phoneKey(k string) bool {
	return detect.PhoneLabelWord(k) || detect.PhoneLabelWord(numberSuffixTrimmed(k)) || detect.PhoneLabelWord(lastKeyPart(k))
}

func printableRuns(b []byte) []string {
	var runs []string
	start, count := -1, 0
	for i := 0; i <= len(b); {
		r, n := rune(0), 1
		if i < len(b) {
			r, n = utf8.DecodeRune(b[i:])
		}
		ok := i < len(b) && printableRune(r)
		if ok && start < 0 {
			start, count = i, 0
		}
		if ok {
			count++
		}
		if !ok && start >= 0 {
			if count >= minBinaryRun {
				runs = append(runs, string(b[start:i]))
			}
			start = -1
		}
		i += n
	}
	return append(runs, detect.WideRuns(b)...)
}

const (
	rawIDWindow  = 1024
	rawIDOverlap = 128
)

func (f *fields) rawID(msg int, path, s string) {
	if len(s) <= rawIDWindow {
		f.add(msg, path, modeID, s, nil)
		return
	}
	for k, at := 0, 0; at < len(s); k, at = k+1, at+rawIDWindow-rawIDOverlap {
		end := min(len(s), at+rawIDWindow)
		piece := s[at:end]
		var cuts []int
		if at > 0 {
			cuts = append(cuts, 0)
		}
		if end < len(s) {
			cuts = append(cuts, utf8.RuneCountInString(piece))
		}
		f.addCut(msg, fmt.Sprintf("%s(part %d)", path, k), modeID, piece, cuts)
		if end == len(s) {
			return
		}
	}
}

func (f *fields) oversize(msg int, path string, n int) bool {
	if f.maxOpaque <= 0 || n <= f.maxOpaque {
		return false
	}
	f.add(msg, path, modeOversize, "", nil)
	return true
}

const (
	blobMinLen   = 24
	blobFullScan = 1024
	redactedBlob = "[redacted secret]"
)

var blobTokenRE = regexp.MustCompile(`[A-Za-z0-9+/_\-]{24,}={0,2}`)

func (f *fields) encodedValues(msg int, path, text string, rewrite bool) string {
	if f.encoded == "" || len(text) < blobMinLen {
		return text
	}
	var out strings.Builder
	var texts, runs []string
	last := 0
	for _, m := range blobTokenRE.FindAllStringIndex(text, -1) {
		token := text[m[0]:m[1]]
		b, ok := base64Decode(token, blobMinLen)
		if !ok || idShaped(token) {
			continue
		}
		switch {
		case f.encoded == config.EncodedRedact && rewrite:
			out.WriteString(text[last:m[0]])
			out.WriteString(redactedBlob)
			last = m[1]
		case len(b) >= blobFullScan:
			f.scanBlob(msg, path+"(b64)", b)
		case textual(b):
			texts = append(texts, cleanText(b))
		default:
			runs = append(runs, printableRuns(b)...)
		}
	}
	if len(texts) > 0 {
		joined := strings.Join(texts, payloadSeparator)
		f.add(msg, path+"(b64)", modeDecoded, joined, nil)
		for _, c := range gluedCards(joined) {
			f.add(msg, path+"(b64)(card)", modeDecoded, c, nil)
		}
		f.keyLeaves(msg, path+"(b64)", modeDecoded, joined)
	}
	f.binaryPII(msg, path+"(b64)", runs)
	if last == 0 {
		return text
	}
	out.WriteString(text[last:])
	return out.String()
}

func (f *fields) scanBlob(msg int, path string, b []byte) {
	saved := f.maxOpaque
	f.maxOpaque = 0
	f.scanBytes(msg, path, b, 0)
	f.maxOpaque = saved
}

func (f *fields) scanBytes(msg int, path string, b []byte, depth int) {
	if depth == 0 && f.oversize(msg, path, len(b)) {
		return
	}
	if depth == 0 {
		f.scanInflated(msg, path, b)
	}
	if depth == 0 && printableShare(b) >= textShare {
		f.scanEncoded(msg, path+"(decoded)", strings.TrimSpace(string(b)), depth+1)
	}
	fr := f.frameOf(b)
	text := textual(b)
	noisy := fr.framed && printableShare(b) < cleanTextShare
	if fr.framed {
		f.scanFramed(msg, path+"(decoded)", fr.payloads)
		f.framePairLeaf(msg, path+"(decoded)", b)
	} else if payloads, ok, _ := walkFrame(b, 1); ok {
		f.scanFramed(msg, path+"(decoded)", payloads)
		f.framePairLeaf(msg, path+"(decoded)", b)
	}
	switch {
	case text && noisy:
		f.textLeaves(msg, path+"(decoded)", modeNoisyFrame, b)
	case text || fr.limit:
		f.textLeaves(msg, path+"(decoded)", modeDecoded, b)
	}
	if !text || noisy {
		f.scanBinary(msg, path, b)
	}
	pairs, keyed, complete := f.pairs(b)
	if len(pairs) > 0 {
		f.add(msg, path+"(cbor)", modeCBORPII, strings.Join(pairs, runSeparator), nil)
	}
	if !complete {
		f.add(msg, path+"(cbor)", modeOversize, oversizeKeyed, nil)
		return
	}
	f.keyedLeaf(msg, path+"(cbor)", keyed, keyedBytesFor(len(b)))
}

const keyedBytesFloor = 4096

func keyedBytesFor(n int) int { return 6*n + keyedBytesFloor }

func (f *fields) keyedLeaf(msg int, path string, copies []string, maxBytes int) {
	seen := map[string]bool{}
	var unique []string
	size := 0
	for _, c := range copies {
		if !seen[c] {
			seen[c] = true
			unique = append(unique, c)
			size += len(c)
		}
	}
	if len(unique) > keyedCopiesMax || size > maxBytes {
		f.add(msg, path, modeOversize, oversizeKeyed, nil)
		return
	}
	if len(unique) > 0 {
		f.add(msg, path+"(keyed)", modeDecoded, strings.Join(unique, payloadSeparator), nil)
	}
}

var inertTokenRE = regexp.MustCompile(`^(?:[0-9.:]+|[0-9]{4}-[0-9]{2}-[0-9]{2}(?:T[0-9:.]+(?:Z|[+-][0-9:]+)?)?|v?[0-9]+(?:\.[0-9]+)+(?:-[a-z]+\.?[0-9]*)?)$`)

const (
	payloadSeparator = "\n\x00\n"
	minProsePayload  = 24
	proseLetterShare = 0.40
)

func cardToken(p []byte) bool {
	d := string(p)
	return strings.Trim(d, digitChars) == "" && detect.CardNumber(d)
}

func inertToken(p []byte) bool {
	return inertTokenRE.Match(p) && !cardToken(p) && !detect.SeparatedPhone(string(p))
}

func proseShaped(p []byte) bool {
	return len(p) >= minProsePayload && letterShare(p) >= proseLetterShare
}

func framedText(payloads [][]byte, depth int, prose, tokens, binary *[]string) {
	for _, p := range payloads {
		fr := parseFrame(p)
		if fr.limit || fr.framed && depth >= frameMaxDepth {
			*prose = append(*prose, cleanText(p))
			continue
		}
		if fr.framed {
			framedText(fr.payloads, depth+1, prose, tokens, binary)
		}
		if !textual(p) {
			if !fr.framed {
				*binary = append(*binary, printableRuns(p)...)
			}
			continue
		}
		switch {
		case tokenPayloadRE.Match(p) && inertToken(p):
		case !tokenPayloadRE.Match(p) && proseShaped(p):
			*prose = append(*prose, cleanText(p))
		default:
			*tokens = append(*tokens, cleanText(p))
		}
	}
}

func (f *fields) scanFramed(msg int, path string, payloads [][]byte) {
	var prose, tokens, binary []string
	framedText(payloads, 0, &prose, &tokens, &binary)
	f.binaryPII(msg, path, binary)
	joined := strings.Join(prose, payloadSeparator)
	if len(prose) > 0 {
		f.add(msg, path+"(fields)", modeDecoded, joined, nil)
		f.keyLeaves(msg, path+"(fields)", modeDecoded, joined)
	}
	shorts := strings.Join(tokens, payloadSeparator)
	if len(tokens) > 0 {
		f.add(msg, path+"(tokens)", modeToken, shorts, nil)
	}
	for _, c := range gluedCards(joined + payloadSeparator + shorts) {
		f.add(msg, path+"(card)", modeDecoded, c, nil)
	}
}

func secretCandidates(run string) []string {
	var out []string
	for _, m := range secretCandidateRE.FindAllStringIndex(run, -1) {
		out = append(out, run[max(0, m[0]-secretContext):m[1]])
	}
	out = append(out, pemBlocks(run)...)
	for _, m := range vendorPrefixRE.FindAllStringIndex(run, -1) {
		tok := run[m[0]:m[1]]
		out = append(out, " "+tok+" ")
		if strings.HasPrefix(tok, "AKIA") || strings.HasPrefix(tok, "ASIA") {
			out = append(out, " "+tok[:min(len(tok), awsKeyLen)]+" ")
		}
	}
	return out
}

const (
	awsKeyLen     = 20
	pemMaxBlock   = 16384
	pemOpenWindow = 4096
)

const pemTailMax = 128

func pemBlocks(run string) []string {
	var out []string
	emitted, endAt := 0, -1
	for i := strings.Index(run, "-----BEGIN "); i >= 0; {
		if i >= emitted {
			if endAt < i {
				endAt = -1
				if j := strings.Index(run[i:], "-----END "); j >= 0 {
					endAt = i + j
				}
			}
			end := min(len(run), i+pemOpenWindow)
			if endAt >= 0 && endAt-i < pemMaxBlock {
				tail := run[endAt+len("-----END ") : min(len(run), endAt+len("-----END ")+pemTailMax)]
				if k := strings.Index(tail, "-----"); k >= 0 {
					end = endAt + len("-----END ") + k + len("-----")
				}
			}
			out = append(out, run[i:end])
			emitted = end
		}
		next := strings.Index(run[i+1:], "-----BEGIN ")
		if next < 0 {
			break
		}
		i += next + 1
	}
	return out
}

const (
	cborTextMin   = 0x60
	cborTextShort = 0x77
	cborText8     = 0x78
	cborText16    = 0x79
	cborMinString = 2
	cborMaxKey    = 64
	cborMaxValue  = 32
	cborMaxSecret = 512
)

func printablePrefix(b []byte) []int32 {
	good := make([]int32, len(b)+1)
	for i := 0; i < len(b); {
		r, n := utf8.DecodeRune(b[i:])
		for k := 0; k < n; k++ {
			good[i+k+1] = good[i+k]
			if printableRune(r) && r != '\n' && r != '\r' {
				good[i+k+1]++
			}
		}
		i += n
	}
	return good
}

func cborItemAt(b []byte, good []int32, i int) (int, int, bool) {
	if i >= len(b) {
		return 0, 0, false
	}
	var n, start int
	switch h := b[i]; {
	case h >= cborTextMin && h <= cborTextShort:
		n, start = int(h-cborTextMin), i+1
	case h == cborText8 && i+1 < len(b):
		n, start = int(b[i+1]), i+2
	case h == cborText16 && i+2 < len(b):
		n, start = int(b[i+1])<<8|int(b[i+2]), i+3
	default:
		return 0, 0, false
	}
	end := start + n
	if n < cborMinString || end > len(b) || int(good[end]-good[start]) != n || !utf8.RuneStart(b[start]) || lastRuneBroken(b[start:end]) {
		return 0, 0, false
	}
	return start, end, true
}

func cborAnchored(b []byte, good []int32, i int) bool {
	return i > 0 && good[i] == good[i-1] && b[i-1] != '\n' && b[i-1] != '\r' && b[i-1] != '\t'
}

func lastRuneBroken(b []byte) bool {
	r, _ := utf8.DecodeLastRune(b)
	return r == utf8.RuneError
}

func (f *fields) pairs(b []byte) ([]string, []string, bool) {
	if f.cborScan != nil {
		return f.cborScan(b)
	}
	return cborPairs(b)
}

func cborPairs(b []byte) ([]string, []string, bool) {
	good := printablePrefix(b)
	var out, keyed []string
	emit := func(c string) bool {
		keyed = append(keyed, c)
		return true
	}
	mapped, complete := cborMapPairs(b)
	for _, p := range mapped {
		if secretIdentifier(p.key, ":") {
			pairCopies(p.key, p.value, emit)
		} else if phoneKey(p.key) && strings.ContainsAny(p.value, "0123456789") {
			out = append(out, p.key+" "+p.value[:min(len(p.value), cborMaxKey)]+" ")
		}
	}
	chained := map[int]bool{}
	for i := range b {
		ks, ke, ok := cborItemAt(b, good, i)
		if !ok || ke-ks > cborMaxKey {
			continue
		}
		vs, ve, ok := cborItemAt(b, good, ke)
		if !ok {
			continue
		}
		if cborAnchored(b, good, i) || chained[i] {
			chained[ve] = true
			if key := string(b[ks:ke]); ve-vs <= cborMaxSecret && secretIdentifier(key, ":") {
				pairCopies(key, string(b[vs:ve]), emit)
			}
		}
		if ve-vs > cborMaxValue || !bytes.ContainsAny(b[vs:ve], "0123456789@") {
			continue
		}
		if key := string(b[ks:ke]); phoneKey(key) {
			out = append(out, key+" "+string(b[vs:ve])+" ")
		} else {
			out = append(out, " "+string(b[vs:ve])+" ")
		}
	}
	return out, keyed, complete
}

func (f *fields) scanBinary(msg int, path string, b []byte) {
	runs := printableRuns(b)
	if !f.frameOf(b).framed {
		f.binaryPII(msg, path+"(decoded)", runs)
	}
	if len(runs) == 0 {
		return
	}
	f.add(msg, path+"(decoded)", modeBinaryRoute, strings.Join(runs, runSeparator), nil)
	var candidates []string
	for _, run := range runs {
		candidates = append(candidates, secretCandidates(run)...)
	}
	if len(candidates) > 0 {
		f.add(msg, path+"(decoded)", modeBinary, strings.Join(candidates, runSeparator), nil)
	}
}

type frameItem struct {
	field uint64
	wire  uint64
	data  []byte
}

func frameItems(b []byte) ([]frameItem, bool) {
	var items []frameItem
	for i := 0; i < len(b); {
		key, n := binary.Uvarint(b[i:])
		if n <= 0 || key>>3 == 0 || key>>3 > frameMaxField || len(items) >= frameMaxFields {
			return nil, false
		}
		i += n
		var data []byte
		switch key & 7 {
		case 0:
			_, m := binary.Uvarint(b[i:])
			if m <= 0 {
				return nil, false
			}
			i += m
			continue
		case 1, 5:
			w := 8
			if key&7 == 5 {
				w = 4
			}
			if i+w > len(b) {
				return nil, false
			}
			data = b[i : i+w]
			i += w
		case 2:
			l, m := binary.Uvarint(b[i:])
			if m <= 0 || l > uint64(len(b)-i-m) {
				return nil, false
			}
			data = b[i+m : i+m+int(l)]
			i += m + int(l)
		default:
			return nil, false
		}
		items = append(items, frameItem{key >> 3, key & 7, data})
	}
	return items, len(items) > 0
}

var pairKeyRE = regexp.MustCompile(identifierKey + `[ \t]{0,8}[:=][ \t]{0,8}$`)

func pairCopies(key, value string, emit func(string) bool) bool {
	if phoneKey(key) {
		return emit("phone " + value[:min(len(value), cborMaxKey)] + " ")
	}
	if !secretIdentifier(key, ":") {
		return true
	}
	for _, v := range secretValueRE.FindAllString(value, -1) {
		if !emit(assignedKey(key) + "=" + v + " ") {
			return false
		}
	}
	return true
}

func framePairs(b []byte, depth int, emit func(string) bool) bool {
	items, ok := frameItems(b)
	if !ok {
		return true
	}
	for i, it := range items {
		if it.wire == 2 {
			if sub, ok := frameItems(it.data); ok && len(sub) == 2 && sub[0].field == 1 && sub[0].wire == 2 && sub[1].field == 2 && sub[1].wire == 2 {
				if !pairCopies(string(sub[0].data), string(sub[1].data), emit) {
					return false
				}
			} else if depth < frameMaxDepth && !framePairs(it.data, depth+1, emit) {
				return false
			}
		}
		if i+1 < len(items) && items[i+1].wire == 2 {
			if m := pairKeyRE.FindStringSubmatch(string(it.data)); m != nil && !pairCopies(m[1], string(items[i+1].data), emit) {
				return false
			}
		}
	}
	return true
}

func (f *fields) framePairLeaf(msg int, path string, b []byte) {
	var copies []string
	complete := framePairs(b, 0, func(c string) bool {
		copies = append(copies, c)
		return len(copies) <= keyedCopiesMax
	})
	if !complete {
		f.add(msg, path, modeOversize, oversizeKeyed, nil)
		return
	}
	if len(copies) > 0 {
		f.add(msg, path+"(pairs)", modeDecoded, strings.Join(copies, payloadSeparator), nil)
	}
}

func (f *fields) binaryPII(msg int, path string, runs []string) {
	if len(runs) == 0 {
		return
	}
	text := strings.Join(runs, runSeparator)
	f.keyLeaves(msg, path+"(runs)", modeDecoded, text)
	f.add(msg, path+"(runs)", modeBinaryPII, text, nil)
	for _, c := range cardsIn(text, cardEdgeTrim) {
		f.add(msg, path+"(card)", modeBinaryPII, c, nil)
	}
	for _, c := range gluedSSNs(text) {
		f.add(msg, path+"(ssn)", modeBinaryPII, c, nil)
	}
	for _, m := range emailShapeRE.FindAllString(text, -1) {
		f.add(msg, path+"(email)", modeBinaryPII, " "+m+" ", nil)
	}
	for _, c := range gluedPhones(text) {
		f.add(msg, path+"(phone)", modeBinaryPII, c, nil)
	}
	if split := splitLetterDigit(text); split != text {
		f.add(msg, path+"(split)", modeBinaryPII, split, nil)
		for _, c := range cardsIn(split, cardEdgeTrim) {
			f.add(msg, path+"(split)(card)", modeBinaryPII, c, nil)
		}
	}
}

func splitLetterDigit(s string) string {
	var b strings.Builder
	prev := rune(-1)
	for _, r := range s {
		if prev >= 0 && (unicode.IsDigit(r) && unicode.IsLetter(prev) || unicode.IsLetter(r) && unicode.IsDigit(prev)) {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
		prev = r
	}
	return b.String()
}

func (f *fields) scanEncoded(msg int, path, s string, depth int) {
	if b, ok := hexDecode(s); ok {
		f.scanBytes(msg, path+"(hex)", b, depth)
	}
	if b, ok := base64Decode(s, minDecodeLen); ok {
		f.scanBytes(msg, path, b, depth)
	}
}

func (f *fields) decodeOnce(s string) ([]byte, bool) {
	if b, ok := f.decoded[s]; ok {
		return b, b != nil
	}
	b, ok := base64Decode(s, looseOpaqueMin)
	if !ok {
		b = nil
	}
	f.decoded[s] = b
	return b, ok
}

func (f *fields) opaque(msg int, path string, m map[string]any, key string) bool {
	s, ok := m[key].(string)
	if !ok {
		return false
	}
	b, ok := f.decodeOnce(s)
	if !ok {
		return false
	}
	f.claim(m, key)
	if f.oversize(msg, path, len(s)) {
		return true
	}
	if len(s) < opaqueMinLen || !f.binaryBlob(b) {
		f.rawID(msg, path, s)
	}
	f.scanBytes(msg, path, b, 0)
	if hb, ok := hexDecode(s); ok {
		f.scanBytes(msg, path+"(hex)", hb, 0)
	}
	return true
}

func encodedKey(m map[string]any, k string) bool {
	switch strings.ToLower(k) {
	case "signature", "encrypted_content", "encrypted_index":
		return true
	case "data":
		return m["type"] == "redacted_thinking"
	}
	return false
}

func (f *fields) decodeKnown(msg int, path string, m map[string]any, k string) bool {
	s, ok := m[k].(string)
	if !ok || !encodedKey(m, k) {
		return false
	}
	f.scanEncoded(msg, path, s, 0)
	b, ok := f.decodeOnce(s)
	return ok && f.binaryBlob(b)
}

func binaryBlocks(rt routeText, rule *config.Rule) bool {
	var runs []string
	for _, joined := range rt.binary {
		runs = append(runs, strings.Split(joined, runSeparator)...)
	}
	for _, run := range runs {
		if rule.MatchText(run) {
			return true
		}
		low := strings.ToLower(run)
		for _, t := range rule.MatchAny {
			if len(t) >= blockSubstrMin && strings.Contains(low, strings.ToLower(t)) {
				return true
			}
		}
	}
	return false
}
