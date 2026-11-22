package detect

import (
	"context"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

type pattern struct {
	name     string
	typ      entity.Type
	label    string
	re       *regexp.Regexp
	group    int
	priority int
	validate func(text string, start, end int) bool
	reject   *regexp.Regexp
	post     func(sp *entity.Span)
	trim     string
	gate     func(text, lower string) bool
}

const streetSuffix = `(?:Street|St|Avenue|Ave|Road|Rd|Boulevard|Blvd|Lane|Ln|Drive|Dr|Court|Ct|Place|Pl|Way|Terrace|Ter|Circle|Cir|Parkway|Pkwy|Highway|Hwy|Square|Sq|Trail|Trl|Loop|Run|Pike|Alley|Row|Crescent|Close|Bend|Ridge|Crossing|Path|Point|Hollow|Glen|Heights|Grove|Park|View|Commons|Landing|Walk|Green|Meadows?|Cove|Creek|Hill|Pass|Mews|Gardens?)`

var linkSecretRE = regexp.MustCompile(`(?i)(reset|magic|verify|verification|confirm|activate|invite|sign-?in|log-?in|auth|token=|code=|otp|password|passwordless|one-?time|unsubscribe|[?&](?:key|sig|signature|t|k)=)`)

func reWithState(tmpl string) *regexp.Regexp {
	return regexp.MustCompile(strings.ReplaceAll(tmpl, "STATES", stateAlternation()))
}

func validator(f func(string) bool) func(string, int, int) bool {
	return func(text string, s, e int) bool { return f(text[s:e]) }
}

func notInsideDotted(text string, s, e int) bool {
	if s > 0 && (text[s-1] == '.' || text[s-1] == 'v' || text[s-1] == 'V') {
		return false
	}
	if e+1 < len(text) && text[e] == '.' && text[e+1] >= '0' && text[e+1] <= '9' {
		return false
	}
	return validIPv4(text[s:e])
}

func notURLUserinfo(text string, s, e int) bool {
	i := s
	for i > 0 && text[i-1] != ' ' && text[i-1] != '\n' && text[i-1] != '\t' {
		i--
	}
	return !strings.Contains(text[i:s], "://")
}

var honorifics = wordSet(`mr mrs ms mx dr prof rev sir madam miss`)

func notHonorific(text string, s, e int) bool {
	for _, w := range strings.Fields(strings.ToLower(text[s:e])) {
		if honorifics[strings.Trim(w, ".")] {
			return false
		}
	}
	return true
}

func passwordLike(s string) bool {
	if len(s) < 4 {
		return false
	}
	var lower, upper, digit, other bool
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			lower = true
		case r >= 'A' && r <= 'Z':
			if i > 0 {
				upper = true
			}
		case r >= '0' && r <= '9':
			digit = true
		default:
			other = true
		}
	}
	return digit || other || (lower && upper)
}

func tokenLike(s string) bool {
	hex := true
	for _, r := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			hex = false
			break
		}
	}
	if hex {
		return hasDigit(s) && strings.ContainsAny(strings.ToLower(s), "abcdef")
	}
	return hasDigit(s) && ShannonEntropy(s) >= 3.5
}

func hasDigit(s string) bool { return strings.ContainsAny(s, "0123456789") }

func phoneValid(text string, s, e int) bool {
	d := digitsOnly(text[s:e])
	if len(d) == 11 && d[0] != '1' {
		return false
	}
	if len(d) != 10 && len(d) != 11 {
		return false
	}
	if s > 0 && (text[s-1] >= '0' && text[s-1] <= '9' || text[s-1] == '-' || text[s-1] == '/') {
		return false
	}
	if e < len(text) && (text[e] >= '0' && text[e] <= '9' || text[e] == '-') {
		return false
	}
	return strings.Trim(d, d[:1]) != ""
}

func intlPhoneValid(text string, s, e int) bool {
	d := digitsOnly(text[s:e])
	return len(d) >= 8 && len(d) <= 15
}

func isDigitByte(c byte) bool { return c >= '0' && c <= '9' }

func isAlphaByte(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

func nationalPhoneValid(text string, s, e int) bool {
	m := text[s:e]
	d := digitsOnly(m)
	maxDigits := 13
	if strings.HasPrefix(d, "00") {
		maxDigits = 15
	}
	if len(d) < 9 || len(d) > maxDigits {
		return false
	}
	if strings.Count(m, "(") != strings.Count(m, ")") {
		return false
	}
	var sep byte
	for i := 0; i < len(m); i++ {
		c := m[i]
		if c == ' ' || c == '.' || c == '-' {
			if sep == 0 {
				sep = c
			} else if sep != c {
				return false
			}
		}
	}
	groups := strings.FieldsFunc(m, func(r rune) bool { return r < '0' || r > '9' })
	first := groups[0]
	if first[1] == '0' && (len(first) < 3 || first[2] == '0') {
		return false
	}
	if strings.Trim(d[1:], d[1:2]) == "" {
		return false
	}
	allTwo, allThree := true, true
	for _, g := range groups {
		if len(g) != 2 {
			allTwo = false
		}
		if len(g) != 3 {
			allThree = false
		}
	}
	if allThree {
		return false
	}
	if s > 0 {
		c := text[s-1]
		if strings.IndexByte(".,-/#$=_+*@\\", c) >= 0 || isDigitByte(c) || isAlphaByte(c) {
			return false
		}
		if strings.HasSuffix(text[:s], "€") || strings.HasSuffix(text[:s], "£") {
			return false
		}
		if s >= 2 && isDigitByte(text[s-2]) && (c == ':' || allTwo && c == sep) {
			return false
		}
	}
	if e < len(text) {
		c := text[e]
		if isDigitByte(c) || isAlphaByte(c) || c == '_' || c == '%' {
			return false
		}
		if e+1 < len(text) && isDigitByte(text[e+1]) && (strings.IndexByte(".,-/:", c) >= 0 || allTwo && c == sep) {
			return false
		}
	}
	if sep == '.' && !(allTwo && len(groups) >= 4) && !(len(groups) == 3 && len(d) == 10) {
		return false
	}
	if len(groups) >= 3 && len(groups[0]) == 2 && len(groups[1]) == 2 && len(groups[2]) == 4 &&
		(strings.HasPrefix(groups[2], "19") || strings.HasPrefix(groups[2], "20")) {
		return false
	}
	if len(groups) == 2 && (len(groups[0]) < 3 || sep == '-' && len(groups[0]) == 5 && len(groups[1]) == 4) {
		return false
	}
	if allTwo {
		ls := strings.LastIndexByte(text[:s], '\n') + 1
		le := len(text)
		if i := strings.IndexByte(text[e:], '\n'); i >= 0 {
			le = e + i
		}
		pairs := 0
		for _, f := range strings.Fields(text[ls:le]) {
			if len(f) == 2 && strings.Trim(f, "0123456789abcdefABCDEF") == "" {
				pairs++
			}
		}
		if pairs > len(groups)+1 || len(groups) != 5 {
			return false
		}
		a, _ := strconv.Atoi(groups[0])
		b, _ := strconv.Atoi(groups[1])
		progression := b-a >= -2 && b-a <= 2
		prev := b
		for _, g := range groups[2:] {
			n, _ := strconv.Atoi(g)
			if n-prev != b-a {
				progression = false
			}
			prev = n
		}
		if progression {
			return false
		}
	}
	return true
}

const digitWord = `(?:zero|oh|one|two|three|four|five|six|seven|eight|nine)`

var digitWordValue = map[string]int{"zero": 0, "oh": 0, "one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "six": 6, "seven": 7, "eight": 8, "nine": 9}

func spokenDigits(s string) []int {
	var out []int
	mult := 1
	for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return r < 'a' || r > 'z' }) {
		switch w {
		case "double":
			mult = 2
		case "triple":
			mult = 3
		default:
			if d, ok := digitWordValue[w]; ok {
				for ; mult > 0; mult-- {
					out = append(out, d)
				}
			}
			mult = 1
		}
	}
	return out
}

func spokenPhoneValid(text string, s, e int) bool {
	d := spokenDigits(text[s:e])
	if len(d) < 7 {
		return false
	}
	asc, desc, same := true, true, true
	for i := 1; i < len(d); i++ {
		if d[i] != d[i-1]+1 {
			asc = false
		}
		if d[i] != d[i-1]-1 {
			desc = false
		}
		if d[i] != d[0] {
			same = false
		}
	}
	return !asc && !desc && !same
}

var idDateRE = regexp.MustCompile(`^(?:\d{4}-\d{2}-\d{2}|\d{2}-\d{2}-\d{4}|(?:19|20)\d{2}-(?:19|20)\d{2})$`)
var idPhoneRE = regexp.MustCompile(`^(?:1-)?\d{3}-\d{3}-\d{4}$`)

func idValueValid(text string, s, e int) bool {
	v := text[s:e]
	d := digitsOnly(v)
	if len(v) < 6 || len(v) > 20 || len(d) < 4 {
		return false
	}
	if strings.Contains(v, " ") && len(d) < 7 {
		return false
	}
	return !idDateRE.MatchString(v) && !idPhoneRE.MatchString(v)
}

const dniLetters = "TRWAGMYFPDXBNJZSQVHLCKE"

func dniChecksumOK(v string) bool {
	v = strings.ReplaceAll(v, "-", "")
	num := v[:len(v)-1]
	switch num[0] {
	case 'X':
		num = "0" + num[1:]
	case 'Y':
		num = "1" + num[1:]
	case 'Z':
		num = "2" + num[1:]
	}
	n, err := strconv.Atoi(num)
	return err == nil && dniLetters[n%23] == v[len(v)-1]
}

func dniValid(text string, s, e int) bool {
	v := text[s:e]
	if dniChecksumOK(v) {
		return true
	}
	num := v[:8]
	if strings.Trim(num, num[:1]) == "" {
		return false
	}
	if y, _ := strconv.Atoi(num[:4]); y >= 1900 && y <= 2099 {
		mo, _ := strconv.Atoi(num[4:6])
		day, _ := strconv.Atoi(num[6:8])
		if mo >= 1 && mo <= 12 && day >= 1 && day <= 31 {
			return false
		}
	}
	if s > 0 && strings.IndexByte("-_/.#=:\\@", text[s-1]) >= 0 {
		return false
	}
	if e < len(text) && strings.IndexByte("-_/\\@=+", text[e]) >= 0 {
		return false
	}
	return true
}

var secretTechPrefixes = strings.Fields(`base64 sha1 sha2 sha3 sha5 sha- md5 utf8 utf-8 utf16 oauth hs256 rs256 es256 http2 ipv4 ipv6 x509 ed25519 bcrypt argon2 aes256 aes-256 rsa2048 rsa-2048 pbkdf2`)
var secretWordyRE = regexp.MustCompile(`^(?:[a-z]+|[A-Z]+)\d{0,3}(?:[\-_][a-z]+){1,2}$`)
var secretDottedRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)+$`)
var secretSnakeRE = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)+$`)
var secretNumericRE = regexp.MustCompile(`^(?:v?\d+(?:\.\d+)+|\d{4}-\d{2}-\d{2}.*)$`)

func secretValueValid(text string, s, e int) bool {
	v := text[s:e]
	if len(v) < 6 {
		return false
	}
	if strings.ContainsAny(v, "()[]{}<>|\\") || strings.Contains(v, "::") || strings.Contains(v, "->") || strings.Contains(v, "=>") || strings.Contains(v, "://") {
		return false
	}
	if strings.IndexByte("$%&*#/.@-~:?", v[0]) >= 0 || e < len(text) && text[e] == '(' {
		return false
	}
	if secretDottedRE.MatchString(v) || secretWordyRE.MatchString(v) || secretNumericRE.MatchString(v) {
		return false
	}
	low := strings.ToLower(v)
	for _, p := range secretTechPrefixes {
		if strings.HasPrefix(low, p) {
			return false
		}
	}
	var lower, upperInside, soft, hard bool
	digits := 0
	for i, r := range v {
		switch {
		case r >= 'a' && r <= 'z':
			lower = true
		case r >= 'A' && r <= 'Z':
			if i > 0 {
				upperInside = true
			}
		case r >= '0' && r <= '9':
			digits++
		case r == '-' || r == '_' || r == '.':
			soft = true
		case r < utf8.RuneSelf:
			hard = true
		}
	}
	before := strings.TrimRight(text[:s], " \t\"'`")
	prose := strings.HasSuffix(before, " is") || strings.HasSuffix(before, " now")
	label := strings.TrimRight(strings.TrimRight(before, ":= \t"), "\"'")
	if strings.HasSuffix(label, "PASS") || strings.HasSuffix(strings.ToLower(label), "pass") && digits == 0 && !hard {
		return false
	}
	if secretSnakeRE.MatchString(v) && digits <= 2 {
		return false
	}
	if digits > 0 || hard {
		return true
	}
	mixed := lower && upperInside
	if soft {
		return mixed || len(strings.FieldsFunc(v, func(r rune) bool { return r == '-' || r == '_' || r == '.' })) >= 4
	}
	if !mixed || v[0] >= 'a' && v[0] <= 'z' {
		return false
	}
	return !prose
}

var hexContextRE = regexp.MustCompile(`(?i)\b(?:commits?|merge|git|rev|revision|tree|parent|head|cherry-pick(?:ed)?|reverts?|bisect|blame|checkout|upstream|regression|backport(?:ed)?|changeset|fixes|squash|rebase|branch|tag|sha-?\d*|md5|\w*sums?|etag|integrity|trace|span|uuid|guid|oid|blob|object)\b`)
var hexIDLabelRE = regexp.MustCompile(`(?i)(?:id|uuid|guid)["']?\s*[:=]\s*["']?$`)

func bareHexValid(text string, s, e int) bool {
	v := text[s:e]
	if n := len(v); n != 32 && n != 40 && n != 64 {
		return false
	}
	if !hasDigit(v) || !strings.ContainsAny(strings.ToLower(v), "abcdef") {
		return false
	}
	if v != strings.ToLower(v) && v != strings.ToUpper(v) {
		return false
	}
	distinct := map[rune]bool{}
	run, prev := 0, rune(0)
	for _, r := range v {
		distinct[r] = true
		if r == prev {
			run++
			if run >= 5 {
				return false
			}
		} else {
			run, prev = 1, r
		}
	}
	if len(distinct) < 10 {
		return false
	}
	if s > 0 && strings.IndexByte("/\\._-@#+%", text[s-1]) >= 0 {
		return false
	}
	if e < len(text) && strings.IndexByte("/\\_-@+", text[e]) >= 0 {
		return false
	}
	if e+1 < len(text) && text[e] == '.' && (isDigitByte(text[e+1]) || isAlphaByte(text[e+1])) {
		return false
	}
	ls := strings.LastIndexByte(text[:s], '\n') + 1
	le := len(text)
	if i := strings.IndexByte(text[e:], '\n'); i >= 0 {
		le = e + i
	}
	before, after := text[ls:s], text[e:le]
	if hexContextRE.MatchString(before) || hexContextRE.MatchString(after) || hexIDLabelRE.MatchString(before) {
		return false
	}
	if strings.Trim(before, " \t#/*>|-") == "" && strings.TrimSpace(after) != "" && (after[0] == ' ' || after[0] == '\t') {
		return false
	}
	return true
}

func gateLower(words string) func(string, string) bool {
	list := strings.Split(words, "|")
	return func(_, lower string) bool {
		for _, w := range list {
			if strings.Contains(lower, w) {
				return true
			}
		}
		return false
	}
}

func gateExact(words string) func(string, string) bool {
	list := strings.Split(words, "|")
	return func(text, _ string) bool {
		for _, w := range list {
			if strings.Contains(text, w) {
				return true
			}
		}
		return false
	}
}

func gateBeforeDigit(words string) func(string, string) bool {
	list := strings.Split(words, "|")
	return func(_, lower string) bool {
		for _, w := range list {
			for from := 0; ; {
				i := strings.Index(lower[from:], w)
				if i < 0 {
					break
				}
				from += i + len(w)
				if from < len(lower) && isDigitByte(lower[from]) {
					return true
				}
			}
		}
		return false
	}
}

func gateAny(gates ...func(string, string) bool) func(string, string) bool {
	return func(text, lower string) bool {
		for _, g := range gates {
			if g(text, lower) {
				return true
			}
		}
		return false
	}
}

var spokenStarts = strings.Fields(`zero oh one two three four five six seven eight nine`)

func gateSpoken(_, lower string) bool {
	next := func(rest string) bool {
		rest = strings.TrimLeft(rest, " \t,-")
		for _, w := range append([]string{"double", "triple"}, spokenStarts...) {
			if strings.HasPrefix(rest, w) {
				return true
			}
		}
		return false
	}
	for _, w := range spokenStarts {
		for from := 0; ; {
			i := strings.Index(lower[from:], w)
			if i < 0 {
				break
			}
			from += i + len(w)
			if from < len(lower) && strings.IndexByte(" \t,-", lower[from]) >= 0 && next(lower[from:]) {
				return true
			}
		}
	}
	return false
}

func stateAbbrAlternation() string {
	var parts []string
	for abbr := range usStates {
		parts = append(parts, abbr)
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}

const (
	addrWord      = `\p{Lu}[\p{L}'’\-]*\.?`
	addrConn      = `(?:de la|de las|de los|de l['’]|del|des|de|du|d['’]|la|le|les|el|los|las|van der|van den|van de|van|der|den|di|della|delle|dei|do|da|dos|das)`
	addrName      = `(?:` + addrConn + ` ?)?` + addrWord + `(?:[ \-](?:` + addrConn + ` ?)?` + addrWord + `){0,3}`
	addrUKPost    = `[A-Z]{1,2}\d[A-Z\d]? ?\d[A-Z]{2}`
	addrBreak     = `(?:, ?| ?\n[ \t]*| )`
	addrCity      = `\p{Lu}[\p{L}'’\-]+(?: (?:(?:am|an der|de|del|la|sur|sous|en|op|upon) )?\p{Lu}[\p{L}'’\-]+)?`
	addrEUTail    = addrBreak + `(?:(?:[A-Z]{1,2}-)?\d{5}|\d{4} ?[A-Z]{2}|\d{4}-\d{3}|\d{4}) ` + addrCity
	addrUKTail    = addrBreak + `(?:` + addrWord + addrBreak + `){1,3}` + addrUKPost
	addrHouse     = `\d{1,4}[a-zA-Z]?\b(?:[\-/]\d{1,4})?`
	addrCapsSuffx = `(?:STREET|ST|AVENUE|AVE|ROAD|RD|BOULEVARD|BLVD|DRIVE|DR|LANE|LN|COURT|CT|PLACE|PL|PARKWAY|PKWY|HIGHWAY|HWY|WAY|CIRCLE|CIR|TERRACE|TER|TRAIL|TRL|SQUARE|SQ|PLAZA|PLZ|LOOP|PIKE)`
)

func addrRE(tmpl string) *regexp.Regexp {
	return regexp.MustCompile(strings.ReplaceAll(tmpl, "ABBR", stateAbbrAlternation()))
}

const idLabel = `(?:(?i:passport|driver'?s?[ ]licen[cs]e|driving[ ]licen[cs]e|licen[cs]e[ ](?:no\.?|number|num\.?|#)|tax(?:payer)?[ ]?id(?:entification)?|national[ ]id(?:entity|entification)?|national[ ]insurance|social[ ]insurance|alien(?:[ ]registration)?|a-number|medicare|health[ ]card|medical[ ]record|residen(?:t|ce)[ ]permit|identity[ ]card|id[ ]card|voter[ ]id)|DL|EIN|TIN|ITIN|DNI|NIE|NIF|NINO|SIN|MRN|CPF|NHS|A#)`
const idQualifier = `(?:[ ]?(?i:no\.?|number|num\.?|nr\.?|id|card|#))*`
const idConnector = `(?:[ ]?[:=#\-][ ]?|[ ](?i:is|was)[ ](?:(?i:now)[ ])?|[ ])["'` + "`" + `*]{0,2}`
const idValue = `(\d{2,6}(?: \d{2,6}){1,2}(?: \d)?|[A-Z0-9][A-Z0-9\-]{4,18}[A-Z0-9])`

var patterns = []pattern{
	{name: "email", typ: entity.Email, label: "email", priority: 50,
		re:       regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9\-]+(?:\.[A-Za-z0-9\-]+)*\.[A-Za-z]{2,}`),
		validate: notURLUserinfo},
	{name: "url", typ: entity.URL, label: "url", priority: 40,
		re: regexp.MustCompile(`(?i)\b(?:https?://|www\.)[^\s<>"'` + "`" + `]+`),
		post: func(sp *entity.Span) {
			if linkSecretRE.MatchString(sp.Text) {
				sp.Type, sp.Label = entity.Secret, "link"
			}
		}},
	{name: "private_key", typ: entity.Secret, label: "key", priority: 100,
		re: regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`)},
	{name: "jwt", typ: entity.Secret, label: "token", priority: 90,
		re: regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`)},
	{name: "api_key_prefixed", typ: entity.Secret, label: "key", priority: 90,
		re: regexp.MustCompile(`\b(?:sk-(?:ant-|proj-|live-|test-)?[A-Za-z0-9_\-]{20,}|(?:ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{40,}|xox[abposr]-[A-Za-z0-9-]{10,}|(?:AKIA|ASIA)[0-9A-Z]{16}|AIza[0-9A-Za-z_\-]{35}|glpat-[A-Za-z0-9_\-]{20,}|hf_[A-Za-z0-9]{30,}|(?:sk|pk|rk)_(?:live|test)_[A-Za-z0-9]{16,}|npm_[A-Za-z0-9]{36}|SG\.[A-Za-z0-9_-]{16,}\.[A-Za-z0-9_-]{16,})`)},
	{name: "bearer", typ: entity.Secret, label: "token", priority: 85, group: 1,
		re: regexp.MustCompile(`(?i)\bbearer\s+([A-Za-z0-9._~+/=\-]{16,})`)},
	{name: "assigned_secret", typ: entity.Secret, label: "secret", priority: 85, group: 1,
		re: regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])(?:api[_-]?key|apikey|secret|token|access[_-]?key|client[_-]?secret|auth[_-]?token)\b["']?\s*[:=]\s*["']?([A-Za-z0-9._\-+/=]{12,})`),
		validate: func(text string, s, e int) bool {
			v := text[s:e]
			return !(e < len(text) && text[e] == '(') && !secretDottedRE.MatchString(v) && !(secretSnakeRE.MatchString(v) && len(digitsOnly(v)) <= 2)
		}},
	{name: "password", typ: entity.Secret, label: "password", priority: 85, group: 1,
		re:     regexp.MustCompile(`(?i)\b(?:password|passwd|pwd|passphrase|passcode)s?\b(?:\s+(?:is|was|were|has|had|been|be|will|set|changed|updated|to|now|still|currently|temporarily|of|for|my|the|new|old|temporary))*\s*[:=]?\s*["'` + "`" + `]?([^\s"'` + "`" + `]{4,})`),
		trim:   ".,;:)",
		reject: regexp.MustCompile(`\[redacted (?:password|passcode)`),
		validate: func(text string, s, e int) bool {
			return passwordLike(text[s:e]) && !secretWordyRE.MatchString(text[s:e]) && strings.Trim(text[s:e], "*.xX-_#") != ""
		}},
	{name: "labelled_secret", typ: entity.Secret, label: "secret", priority: 85, group: 1,
		re:       regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])(?:pass|secret(?:[ _\-]key)?|token|api[ _\-]?key|access[ _\-]?(?:token|key)|auth[ _\-]?token)\b["']?(?:[ \t]*[:=][ \t]*|[ \t]+is[ \t]+(?:now[ \t]+)?)["'` + "`" + `]?([^\s"'` + "`" + `]{6,})`),
		trim:     ".,;:)",
		reject:   regexp.MustCompile(`\[redacted `),
		validate: secretValueValid},
	{name: "bare_hex", typ: entity.Secret, label: "key", priority: 30,
		re: regexp.MustCompile(`[0-9a-fA-F]{32,64}`), validate: bareHexValid},
	{name: "env_secret", typ: entity.Secret, label: "secret", priority: 86, group: 1,
		re: regexp.MustCompile(`\b[A-Z][A-Z0-9_]*(?:KEY|SECRET|TOKEN|PASSWORD|PASSWD|PASS|PWD|CREDENTIALS?|AUTH|APIKEY)[A-Z0-9_]*\s*[:=]\s*["']?([^\s"'` + "`" + `]{6,})`),
		validate: func(text string, s, e int) bool {
			return hasDigit(text[s:e]) || passwordLike(text[s:e]) || len(text[s:e]) >= 16
		}},
	{name: "context_token", typ: entity.Secret, label: "secret", priority: 84, group: 1,
		re:       regexp.MustCompile(`(?i)\b(?:credentials?|keys?|tokens?|secrets?|passwords?|hash|signature|apikey|api key)\b[^\n]{0,40}?(?:^|[^A-Za-z0-9+/_\-])([A-Za-z0-9+/_\-]{32,}={0,2})`),
		validate: validator(tokenLike)},
	{name: "url_credentials", typ: entity.Secret, label: "password", priority: 95, group: 1,
		re: regexp.MustCompile(`\b[a-zA-Z][a-zA-Z0-9+.\-]*://[^\s:/@]+:([^\s@/]+)@`)},
	{name: "otp_after", typ: entity.Secret, label: "code", priority: 80, group: 1,
		re:     regexp.MustCompile(`(?i)\b(?:code|otp|passcode|pin|one[- ]time|2fa|mfa|verification|security code|login code)\b[^\d\n]{0,30}?\b([A-Z]-?\d{4,8}|\d{3}[- ]\d{3}|\d{4,8})\b`),
		reject: regexp.MustCompile(`(?i)\b(?:zip|postal|post|area|country|promo|discount|coupon|source|error|status|http|dial|exit|return|response|tax|reference|ref|product|item|sku|order)\s+code\b`)},
	{name: "otp_before", typ: entity.Secret, label: "code", priority: 80, group: 1,
		re: regexp.MustCompile(`(?i)\b(\d{4,8})\b\s*(?:is\s+your|is\s+the|-)?\s*(?:verification|security|login|sign-?in|one[- ]time|confirmation|access|2fa|otp)\s+code`)},
	{name: "high_entropy", typ: entity.Secret, label: "secret", priority: 20,
		re:       regexp.MustCompile(`[A-Za-z0-9_\-+/]{24,}={0,2}`),
		validate: validator(looksRandomToken)},
	{name: "ssn", typ: entity.GovID, label: "id", priority: 70,
		re: regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)},
	{name: "ssn_context", typ: entity.GovID, label: "id", priority: 75, group: 1,
		re: regexp.MustCompile(`(?i)(?:\b(?:ssn|ssns|social security(?:\s+(?:number|no|num))?|social|tax\s?id|taxpayer id|itin|tin)\b|\bss#)[^\d\n]{0,25}(\d{3}[\s.\-]?\d{2}[\s.\-]?\d{4})\b`)},
	{name: "id_document", typ: entity.GovID, label: "id", priority: 70, group: 1,
		re:       regexp.MustCompile(`(?i)\b(?:passport|driver'?s?\s+licen[cs]e|licen[cs]e\s+(?:no|number)|dl)\b(?:\s*(?:no\.?|number|#))?[\s:#]{0,4}([A-Z0-9]{6,12})\b`),
		validate: func(text string, s, e int) bool { return hasDigit(text[s:e]) }},
	{name: "uk_nino", typ: entity.GovID, label: "id", priority: 70,
		re: regexp.MustCompile(`\b[A-CEGHJ-PR-TW-Z]{2}[ \-]?\d{2}[ \-]?\d{2}[ \-]?\d{2}[ \-]?[A-D]\b`)},
	{name: "labelled_id", gate: gateAny(gateLower("passport|licen|tax|national i|insurance|alien|a-number|medicare|health card|medical record|permit|identity card|id card|voter id"), gateExact("DL|EIN|TIN|DNI|NIE|NIF|NINO|SIN|MRN|CPF|NHS|A#")), typ: entity.GovID, label: "id", priority: 76, group: 1,
		re:       regexp.MustCompile(`\b` + idLabel + idQualifier + idConnector + idValue),
		validate: idValueValid},
	{name: "labelled_policy", gate: gateLower("policy|member|subscriber|claim|group"), typ: entity.Financial, label: "policy number", priority: 73, group: 1,
		re:       regexp.MustCompile(`(?i:\b(?:policy|member(?:ship)?|subscriber|claim|group)[ ](?:no\.?|number|num\.?|id|#))` + idConnector + idValue),
		validate: idValueValid},
	{name: "es_dni", typ: entity.GovID, label: "id", priority: 70,
		re: regexp.MustCompile(`\b\d{8}-?[A-Z]\b`), validate: dniValid},
	{name: "es_nie", typ: entity.GovID, label: "id", priority: 70,
		re: regexp.MustCompile(`\b[XYZ]-?\d{7}-?[A-Z]\b`), validate: validator(dniChecksumOK)},
	{name: "credit_card", typ: entity.Financial, label: "card", priority: 70,
		re: regexp.MustCompile(`\b(?:\d[ -]?){12,18}\d\b`), validate: validator(validCard)},
	{name: "iban", typ: entity.Financial, label: "account", priority: 70,
		re: regexp.MustCompile(`\b[A-Z]{2}\d{2}(?: ?[A-Z0-9]{4}){2,7}(?: ?[A-Z0-9]{1,4})?\b`), validate: validator(ValidIBAN)},
	{name: "routing", typ: entity.Financial, label: "account", priority: 75, group: 1,
		re: regexp.MustCompile(`(?i)\b(?:routing|aba|rtn)\b[^\d\n]{0,20}(\d{9})\b`), validate: validator(ValidABA)},
	{name: "account", typ: entity.Financial, label: "account", priority: 72, group: 1,
		re: regexp.MustCompile(`(?i)\b(?:account|acct|a/c)\b(?:\s*(?:number|no\.?|#))?[^\d\n]{0,12}(\d[\d -]{3,20}\d)\b`)},
	{name: "policy_number", typ: entity.Financial, label: "policy number", priority: 71, group: 1,
		re:       regexp.MustCompile(`(?i)\b(?:policy|claim|member|subscriber|group|contract|certificate)\s*(?:number|no\.?|num|#|id)?\s*[:#]?\s*([A-Z0-9][A-Z0-9\-]{5,24})\b`),
		validate: func(text string, s, e int) bool { return hasDigit(text[s:e]) }},
	{name: "ipv4", typ: entity.IP, label: "ip", priority: 45,
		re: regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`), validate: notInsideDotted},
	{name: "ipv6", typ: entity.IP, label: "ip", priority: 45,
		re:       regexp.MustCompile(`(?:[0-9A-Fa-f]{1,4}:){7}[0-9A-Fa-f]{1,4}|(?:[0-9A-Fa-f]{1,4}:){1,7}:(?:[0-9A-Fa-f]{1,4}(?::[0-9A-Fa-f]{1,4}){0,5})?|::(?:[0-9A-Fa-f]{1,4}:){0,6}[0-9A-Fa-f]{1,4}`),
		validate: validator(validIPv6)},
	{name: "phone_nanp", typ: entity.Phone, label: "phone", priority: 55,
		re:       regexp.MustCompile(`(?:\+?1[\s.\-]?)?(?:\(\d{3}\)\s?|\d{3}[\s.\-]?)\d{3}[\s.\-]?\d{4}\b`),
		validate: phoneValid},
	{name: "phone_intl", typ: entity.Phone, label: "phone", priority: 54,
		re:       regexp.MustCompile(`\+\d{1,3}(?:[\s.\-]?\(?\d{1,4}\)?){2,6}\b`),
		validate: intlPhoneValid},
	{name: "phone_national", typ: entity.Phone, label: "phone", priority: 53,
		re:       regexp.MustCompile(`\(?0\d{1,4}\)?(?:[ .\-]\d{2,8}){1,5}`),
		validate: nationalPhoneValid},
	{name: "phone_spoken", gate: gateSpoken, typ: entity.Phone, label: "phone", priority: 53,
		re:       regexp.MustCompile(`(?i)\b(?:(?:double|triple)[ \-])?` + digitWord + `(?:(?:[ \t]*[,\-][ \t]*|[ \t]+)(?:(?:double|triple)[ \-])?` + digitWord + `){3,}\b`),
		validate: spokenPhoneValid},
	{name: "street_address", typ: entity.Address, label: "address", priority: 60,
		re: reWithState(`\b\d{1,6}\s+(?:[NSEW]\.?\s+)?(?:[A-Z0-9][A-Za-z0-9'\-]*\.?\s+){1,4}` + streetSuffix + `\b\.?(?:,?\s+(?:Apt|Apartment|Suite|Ste|Unit|#)\.?\s*[A-Za-z0-9\-]+)?(?:,\s*[A-Z][A-Za-z]+(?:\s[A-Z][A-Za-z]+){0,2},\s*(?:STATES)\b(?:\s+\d{5}(?:-\d{4})?)?)?`)},
	{name: "address_fr", typ: entity.Address, label: "address", priority: 61, trim: ".,",
		re: regexp.MustCompile(`\b\d{1,4}(?: ?(?:bis|ter))?,? (?i:rue|boulevard|bd\.?|chemin|allée|allee|impasse|quai|avenue|av\.) ` + addrName + `(?:` + addrEUTail + `)?`)},
	{name: "address_fr_postal", typ: entity.Address, label: "address", priority: 61, trim: ".,",
		re: regexp.MustCompile(`\b\d{1,4}(?: ?(?:bis|ter))?,? (?i:place|cours|route|passage|square) ` + addrName + addrEUTail)},
	{name: "address_street_first", gate: gateLower("calle |carrer |avenida |avda|avinguda |paseo |passeig |rua |travessa |viale |piazza |ronda |camino |carretera |rambla "), typ: entity.Address, label: "address", priority: 61, trim: ".,",
		re: regexp.MustCompile(`(?i:calle|carrer|avenida|avda\.?|avinguda|paseo|passeig|rua|travessa|viale|piazza|ronda|camino|carretera|rambla) ` + addrName + `,? (?:(?i:n[ºo°]\.?|num\.?|número) ?)?\d{1,4}[A-Za-z]?\b(?:, ?\d{1,2}[ºª°]? ?[A-Za-z]{0,3}\b)?(?:` + addrEUTail + `)?`)},
	{name: "address_street_first_postal", gate: gateExact("Via |Plaza |Plaça |Corso |Largo |Vicolo "), typ: entity.Address, label: "address", priority: 61, trim: ".,",
		re: regexp.MustCompile(`(?:Via|Plaza|Plaça|Corso|Largo|Vicolo) ` + addrName + `,? (?:(?i:n[ºo°]\.?) ?)?\d{1,4}[A-Za-z]?\b` + addrEUTail)},
	{name: "address_germanic", gate: gateBeforeDigit("straße |strasse |str. |gasse |allee |gracht |straat |laan |plein |singel |weg |platz |dijk |steeg |dreef |markt "), typ: entity.Address, label: "address", priority: 61, trim: ".,",
		re: regexp.MustCompile(`(?:` + addrWord + ` )?\p{Lu}\p{L}+(?:straße|strasse|str\.|gasse|allee|gracht|straat|laan|plein|singel|weg|platz|dijk|steeg|dreef|markt) ` + addrHouse + `(?:` + addrEUTail + `)?`)},
	{name: "address_germanic_split", gate: gateBeforeDigit("straße |strasse |str. |allee |platz |weg |gasse |chaussee |laan |straat "), typ: entity.Address, label: "address", priority: 61, trim: ".,",
		re: regexp.MustCompile(`(?:` + addrWord + ` ){1,2}(?:Straße|Strasse|Str\.|Allee|Platz|Weg|Gasse|Chaussee|Laan|Straat) ` + addrHouse + `(?:` + addrEUTail + `)?`)},
	{name: "address_germanic_postal", gate: gateAny(gateExact("Am |An der |An den |Im |Zum |Zur |Auf der |Auf dem |Unter den |In der |Op de |Aan de "), gateBeforeDigit("ring |damm |ufer |kade ")), typ: entity.Address, label: "address", priority: 61, trim: ".,",
		re: regexp.MustCompile(`(?:(?:Am|An der|An den|Im|Zum|Zur|Auf der|Auf dem|Unter den|In der|Op de|Aan de) ` + addrWord + `(?: ` + addrWord + `)?|(?:` + addrWord + ` )?(?:\p{Lu}\p{L}+(?:ring|damm|ufer|kade)|Ring|Damm|Ufer)) ` + addrHouse + addrEUTail)},
	{name: "address_uk_flat", typ: entity.Address, label: "address", priority: 61, trim: ".,",
		re: regexp.MustCompile(`\b(?:Flat|Apartment|Apt\.?|Unit|Suite|Room) [A-Za-z0-9]{1,4},? \d{1,5}[A-Za-z]? (?:` + addrWord + ` ){1,3}` + streetSuffix + `\b\.?(?:` + addrUKTail + `)?`)},
	{name: "address_uk_postcode", typ: entity.Address, label: "address", priority: 61,
		re: regexp.MustCompile(`\b(?:(?:Flat|Apartment|Apt\.?|Unit|Suite|Room) [A-Za-z0-9]{1,4},? )?\d{1,5}[A-Za-z]? ` + addrWord + `(?: ` + addrWord + `){0,3}` + addrUKTail + `\b`)},
	{name: "address_us_caps", typ: entity.Address, label: "address", priority: 61,
		re: addrRE(`\b\d{1,6} (?:[NSEW]{1,2} )?(?:[A-Z0-9]+ ){1,4}` + addrCapsSuffx + `\b(?: (?:APT|STE|SUITE|UNIT|BLDG|FL|RM|#) ?[A-Z0-9\-]+)?(?: ?, ?|[ \t]*\n[ \t]*)(?:[A-Z]+ ){0,2}[A-Z]+,? (?:ABBR) {1,2}\d{5}(?:-\d{4})?\b`)},
	{name: "po_box", typ: entity.Address, label: "address", priority: 60,
		re: regexp.MustCompile(`(?i)\bP\.?\s?O\.?\s+Box\s+\d+\b`)},
	{name: "city_state", typ: entity.Location, label: "location", priority: 30,
		re: reWithState(`\b[A-Z][a-z]+(?:[ \-][A-Z][a-z]+){0,2},\s+(?:STATES)\b(?:\s+\d{5}(?:-\d{4})?)?`),
		post: func(sp *entity.Span) {
			if st, ok := StateIn(sp.Text); ok {
				sp.Generalized = "a city in " + st
			}
		}},
	{name: "org_suffix", typ: entity.Org, label: "organization", priority: 25,
		re: regexp.MustCompile(`\b(?:[A-Z][A-Za-z0-9&'\-]*\s+(?:(?:&|and)\s+)?){1,4}(?:Inc|LLC|Ltd|Corp|Corporation|CORPORATION|GmbH|PLC|LLP|Co|AB|ApS|AG|SA|SAS|SARL|BV|NV|Oy|AS|Pty|Pte|Srl|SpA|KG|LP|Holdings|HOLDINGS|Industries|INDUSTRIES|Enterprises|ENTERPRISES|Technologies|TECHNOLOGIES|Logistics|LOGISTICS|Consulting|CONSULTING|Freight|FREIGHT|Fabrics|FABRICS|Hardware|HARDWARE|Brewing|BREWING|Brewery|BREWERY|Bakery|BAKERY|Motors|MOTORS|Labs|Laboratories|LABORATORIES|Ventures|VENTURES|University|UNIVERSITY|College|COLLEGE|Institute|INSTITUTE|Hospital|HOSPITAL|Clinic|CLINIC|Pharmacy|PHARMACY|Construction|CONSTRUCTION|Plumbing|PLUMBING|Realty|REALTY|Insurance|INSURANCE|Airlines|AIRLINES|Airways|AIRWAYS|Foods|FOODS|Farms|FARMS|Architects|ARCHITECTS|Arkitekter|ARKITEKTER|Byg|Bank|Pharmaceuticals|PHARMACEUTICALS|Textiles|TEXTILES|Shipping|SHIPPING|Trucking|TRUCKING|Roofing|ROOFING|Landscaping|LANDSCAPING|Automotive|AUTOMOTIVE|Electronics|ELECTRONICS|Software|SOFTWARE|Analytics|ANALYTICS|Robotics|ROBOTICS|Biotech|BIOTECH|Therapeutics|THERAPEUTICS|Networks|NETWORKS|Outfitters|OUTFITTERS|Distillery|DISTILLERY|Winery|WINERY|Vineyards|VINEYARDS|Jewelers|JEWELERS|Florist|FLORIST|Dental|DENTAL|Orthodontics|ORTHODONTICS|Chiropractic|CHIROPRACTIC|Veterinary|VETERINARY)\b\.?`)},
	{name: "honorific_person", typ: entity.Person, label: "person", priority: 35, group: 1,
		re:       regexp.MustCompile(`\b(?:(?:Mr|Mrs|Ms|Mx|Dr|Prof|Rev|Herr|Frau|Señor|Señora|Sr|Sra|Mme|Mlle|Monsieur|Madame|Signor|Signora|Mevrouw|Meneer|Pan|Pani)\.?\s+)+((?:(?:de|del|della|di|da|dos|van|von|der|den|ten|ter|la|le|el|al)\s+){0,3}` + nameWord + `(?:\s+` + nameWord + `)?)`),
		validate: notHonorific},
	{name: "greeting_person", typ: entity.Person, label: "person", priority: 15, group: 1,
		re:       regexp.MustCompile(`(?m)^\s*(?:Hi|Hello|Dear|Hey)\s+(` + nameWord + `(?:\s+` + nameWord + `)?)\s*[,!:\n]`),
		validate: notHonorific},
}

type RegexDetector struct {
	pats []pattern
}

func NewRegexDetector() *RegexDetector { return &RegexDetector{pats: patterns} }

func (d *RegexDetector) Name() string { return "regex" }

func (d *RegexDetector) Detect(_ context.Context, text string) ([]entity.Span, error) {
	lower := ""
	for _, p := range d.pats {
		if p.gate != nil {
			lower = strings.ToLower(text)
			break
		}
	}
	if len(text) < parallelMin {
		var out []entity.Span
		for i := range d.pats {
			out = append(out, d.pats[i].find(text, lower)...)
		}
		return out, nil
	}
	results := make([][]entity.Span, len(d.pats))
	sem := make(chan struct{}, runtime.GOMAXPROCS(0))
	var wg sync.WaitGroup
	for i := range d.pats {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			results[i] = d.pats[i].find(text, lower)
			<-sem
		}(i)
	}
	wg.Wait()
	var out []entity.Span
	for _, r := range results {
		out = append(out, r...)
	}
	return out, nil
}

const parallelMin = 8 << 10

func (p *pattern) find(text, lower string) []entity.Span {
	if p.gate != nil && !p.gate(text, lower) {
		return nil
	}
	var out []entity.Span
	for _, m := range p.re.FindAllStringSubmatchIndex(text, -1) {
		s, e := m[2*p.group], m[2*p.group+1]
		if s < 0 {
			continue
		}
		if p.typ == entity.URL {
			e = trimURL(text, s, e)
		}
		for p.trim != "" && e > s && strings.IndexByte(p.trim, text[e-1]) >= 0 {
			e--
		}
		if !boundaryOK(text, s, e) {
			continue
		}
		if p.validate != nil && !p.validate(text, s, e) {
			continue
		}
		if p.reject != nil && p.reject.MatchString(text[max(0, m[0]-16):m[1]]) {
			continue
		}
		sp := entity.Span{Start: s, End: e, Type: p.typ, Label: p.label, Text: text[s:e], Source: "regex:" + p.name, Priority: p.priority, Score: 1}
		if p.post != nil {
			p.post(&sp)
		}
		out = append(out, sp)
	}
	return out
}

func trimURL(text string, s, e int) int {
	for e > s {
		c := text[e-1]
		if strings.IndexByte(".,;:!?'\"", c) >= 0 {
			e--
			continue
		}
		if c == ')' && strings.Count(text[s:e], "(") < strings.Count(text[s:e], ")") {
			e--
			continue
		}
		if c == ']' && strings.Count(text[s:e], "[") < strings.Count(text[s:e], "]") {
			e--
			continue
		}
		break
	}
	return e
}
