package detect

import (
	"context"
	"regexp"
	"strings"

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
		re: regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9])(?:api[_-]?key|apikey|secret|token|access[_-]?key|client[_-]?secret|auth[_-]?token)\b["']?\s*[:=]\s*["']?([A-Za-z0-9._\-+/=]{12,})`)},
	{name: "password", typ: entity.Secret, label: "password", priority: 85, group: 1,
		re:       regexp.MustCompile(`(?i)\b(?:password|passwd|pwd|passphrase|passcode)s?\b(?:\s+(?:is|was|were|has|had|been|be|will|set|changed|updated|to|now|still|currently|temporarily|of|for|my|the|new|old|temporary))*\s*[:=]?\s*["'` + "`" + `]?([^\s"'` + "`" + `]{4,})`),
		trim:     ".,;:)",
		reject:   regexp.MustCompile(`\[redacted (?:password|passcode)`),
		validate: func(text string, s, e int) bool { return passwordLike(text[s:e]) }},
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
		re: regexp.MustCompile(`\b[A-CEGHJ-PR-TW-Z]{2}\s?\d{2}\s?\d{2}\s?\d{2}\s?[A-D]\b`)},
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
		re:       regexp.MustCompile(`\+\d{1,3}(?:[\s.\-]?\(?\d{1,4}\)?){2,5}\b`),
		validate: intlPhoneValid},
	{name: "street_address", typ: entity.Address, label: "address", priority: 60,
		re: reWithState(`\b\d{1,6}\s+(?:[NSEW]\.?\s+)?(?:[A-Z0-9][A-Za-z0-9'\-]*\.?\s+){1,4}` + streetSuffix + `\b\.?(?:,?\s+(?:Apt|Apartment|Suite|Ste|Unit|#)\.?\s*[A-Za-z0-9\-]+)?(?:,\s*[A-Z][A-Za-z]+(?:\s[A-Z][A-Za-z]+){0,2},\s*(?:STATES)\b(?:\s+\d{5}(?:-\d{4})?)?)?`)},
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
		re: regexp.MustCompile(`\b(?:[A-Z][A-Za-z0-9&'\-]*\s+(?:(?:&|and)\s+)?){1,4}(?:Inc|LLC|Ltd|Corp|Corporation|GmbH|PLC|LLP|Co)\b\.?`)},
	{name: "honorific_person", typ: entity.Person, label: "person", priority: 35, group: 1,
		re:       regexp.MustCompile(`\b(?:(?:Mr|Mrs|Ms|Mx|Dr|Prof|Rev)\.?\s+)+([A-Z][a-z'\-]+(?:\s+[A-Z][a-z'\-]+)?)`),
		validate: notHonorific},
	{name: "greeting_person", typ: entity.Person, label: "person", priority: 15, group: 1,
		re:       regexp.MustCompile(`(?m)^\s*(?:Hi|Hello|Dear|Hey)\s+([A-Z][a-z'\-]+(?:\s+[A-Z][a-z'\-]+)?)\s*[,!:\n]`),
		validate: notHonorific},
}

type RegexDetector struct {
	pats []pattern
}

func NewRegexDetector() *RegexDetector { return &RegexDetector{pats: patterns} }

func (d *RegexDetector) Name() string { return "regex" }

func (d *RegexDetector) Detect(_ context.Context, text string) ([]entity.Span, error) {
	var out []entity.Span
	for _, p := range d.pats {
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
	}
	return out, nil
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
