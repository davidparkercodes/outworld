package detect

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const (
	urlStrongMinLen     = 8
	urlMixedMinLen      = 4
	urlTrivialMaxDigits = 6
	botIDMinDigits      = 6
	botIDMaxDigits      = 12
	botSecretMinLen     = 30
)

var (
	urlStrongKeys    = wordSet(`token access_token refresh_token id_token auth_token authtoken session_token sessiontoken api_key apikey api-key secret client_secret password passwd pwd pass passcode sig signature x-amz-signature x-amz-credential x-amz-security-token x-goog-signature x-goog-credential sas_token private_token session_id sessionid`)
	urlWeakKeys      = wordSet(`key code auth otp t k`)
	urlMagicWords    = wordSet(`reset reset-password password-reset verify verification confirm activate invite magic magic-link signin sign-in login log-in auth unsubscribe onetime one-time`)
	urlPlaceholderRE = regexp.MustCompile(`(?i)^(?:<[^>]*>|\{+[^}]*\}+|\[[^\]]*\]|%[a-z0-9_]+%|:[a-z_]+|\$\{?[a-z0-9_]+\}?|x+|\*+|\.+|_+|-+|your[_-]?[a-z0-9_-]*|my[_-]?[a-z0-9_-]*|example|sample|test|dummy|placeholder|null|nil|none|undefined|true|false|yes|no|on|off|changeme|redacted|secret|token|password|value|string)$`)
	urlTokenShapeRE  = regexp.MustCompile(`^[A-Za-z0-9_\-.~+/=]{16,}$`)
	urlPasswordKeys  = wordSet(`password passwd pwd passcode passphrase`)
	urlBarePlaceRE   = regexp.MustCompile(`(?i)^(?:<[^>]*>|\{+[^}]*\}+|\$\{?[a-z0-9_]+\}?|%[a-z0-9_]+%|x{3,}|\*{3,}|\.{3,}|true|false|null|none)$`)
	urlTrivialRE     = regexp.MustCompile(`(?i)^(?:v?\d{1,` + strconv.Itoa(urlTrivialMaxDigits) + `}(?:\.\d+)*|v\d+[a-z]?|(?:hmac[-_]?)?(?:sha|sha-)(?:1|224|256|384|512)|md5|aws4-hmac-sha256|hs256|rs256|es256|plain|basic|bearer|none)$`)
	urlBase64RE      = regexp.MustCompile(`^[A-Za-z0-9+/]{8,}={0,2}$`)
	urlPathKeys      = wordSet(`token tokens key keys secret secrets apikey api-key api_key access-token access_token`)
	urlVendorShapes  []*regexp.Regexp
	urlSlackHookRE   = regexp.MustCompile(`hooks\.slack\.com/services/T[A-Z0-9]+/B[A-Z0-9]+/[A-Za-z0-9]{16,}`)
	urlBotTokenRE    = regexp.MustCompile(`/bot\d{` + strconv.Itoa(botIDMinDigits) + `,` + strconv.Itoa(botIDMaxDigits) + `}:[A-Za-z0-9_-]{` + strconv.Itoa(botSecretMinLen) + `,}(?:[/?#]|$)`)
)

func init() {
	for _, p := range patterns {
		if p.name == "jwt" || p.name == "api_key_prefixed" {
			urlVendorShapes = append(urlVendorShapes, p.re)
		}
	}
}

func linkSecret(u string) bool {
	rest := u
	if i := strings.Index(rest, "://"); i >= 0 {
		rest = rest[i+3:]
	}
	auth := rest
	if i := strings.IndexAny(auth, "/?#"); i >= 0 {
		auth = auth[:i]
	}
	if at := strings.LastIndexByte(auth, '@'); at >= 0 {
		if c := strings.IndexByte(auth[:at], ':'); c >= 0 && urlCredentialValue(auth[c+1:at], true) {
			return true
		}
	}
	for _, re := range urlVendorShapes {
		if re.MatchString(u) {
			return true
		}
	}
	if urlBotTokenRE.MatchString(u) || urlSlackHookRE.MatchString(u) {
		return true
	}
	params, path := "", rest
	if i := strings.IndexAny(rest, "?#"); i >= 0 {
		params, path = rest[i+1:], rest[:i]
	}
	segs := strings.Split(path, "/")
	magic := false
	for _, seg := range segs[1:] {
		if urlMagicWords[strings.ToLower(seg)] {
			magic = true
		}
	}
	for _, kv := range strings.FieldsFunc(params, func(r rune) bool { return r == '&' || r == ';' || r == '#' || r == '?' }) {
		eq := strings.IndexByte(kv, '=')
		if eq <= 0 {
			continue
		}
		key := kv[:eq]
		if k, err := url.QueryUnescape(key); err == nil {
			key = k
		}
		if urlParamSecret(key, kv[eq+1:], magic) {
			return true
		}
	}
	for i := 1; i+1 < len(segs); i++ {
		if urlPathKeys[strings.ToLower(segs[i])] && urlCredentialValue(segs[i+1], false) {
			return true
		}
		if !urlMagicWords[strings.ToLower(segs[i])] {
			continue
		}
		next := segs[i+1]
		if urlCredentialValue(next, false) {
			return true
		}
		if w := strings.ToLower(segs[i]); (w == "magic" || w == "magic-link") && urlCredentialValue(next, true) && !urlMagicWords[strings.ToLower(next)] {
			return true
		}
	}
	return false
}

func keyParts(k string) []string {
	var parts []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			parts = append(parts, strings.ToLower(cur.String()))
			cur.Reset()
		}
	}
	for i := 0; i < len(k); i++ {
		c := k[i]
		switch {
		case c == '_' || c == '-' || c == '.' || c == ' ':
			flush()
		case c >= 'A' && c <= 'Z' && i > 0 && k[i-1] >= 'a' && k[i-1] <= 'z':
			flush()
			cur.WriteByte(c)
		default:
			cur.WriteByte(c)
		}
	}
	flush()
	return parts
}

var urlStrongParts = wordSet(`api app secret private token password passwd pwd auth sig signature session credential credentials license encryption master signing client consumer subscription access bearer apikey`)

func urlStrongKey(key string) bool {
	parts := keyParts(key)
	for _, p := range parts {
		if urlStrongParts[p] {
			return true
		}
	}
	return len(parts) > 1 && parts[len(parts)-1] == "key"
}

const urlIdentifierMax = 24

var (
	urlPlainParts     = wordSet(`public sort primary cache partition foreign`)
	urlPlainLastParts = wordSet(`version ver type method mode format name id url uri host endpoint base region scope count limit timeout enabled`)
	urlIdentifierRE   = regexp.MustCompile(`^[A-Za-z_]+$`)
)

func urlCredentialKey(parts []string) bool {
	for _, p := range parts {
		if urlStrongParts[p] {
			return true
		}
	}
	return false
}

func urlPlainKey(parts []string) bool {
	for _, p := range parts {
		if urlPlainParts[p] {
			return true
		}
	}
	return false
}

func urlKeyNamesValue(parts []string, val string) bool {
	if len(parts) < 2 || parts[len(parts)-1] != "key" {
		return false
	}
	v := urlUnescape(val)
	return len(v) < urlIdentifierMax && urlIdentifierRE.MatchString(v)
}

func urlParamSecret(key, val string, magic bool) bool {
	orig := key
	key = strings.ToLower(key)
	parts := keyParts(orig)
	if len(parts) > 1 && urlPlainLastParts[parts[len(parts)-1]] && !urlStrongKeys[key] {
		return false
	}
	if !urlCredentialKey(parts) && (urlPlainKey(parts) || urlKeyNamesValue(parts, val)) {
		return false
	}
	switch {
	case urlPasswordKeys[key] || strings.HasSuffix(key, "_password"):
		return urlPasswordValue(val)
	case urlStrongKeys[key] || urlStrongKey(orig) && !urlWeakKeys[key]:
		return magic && urlCredentialValue(val, true) || urlStrongValue(val)
	case urlWeakKeys[key]:
		return urlCredentialValue(val, magic) || key == "auth" && urlBase64Value(val)
	}
	return false
}

func urlMagicPath(u string) bool {
	rest := u
	if i := strings.Index(rest, "://"); i >= 0 {
		rest = rest[i+3:]
	}
	if i := strings.IndexAny(rest, "?#"); i >= 0 {
		rest = rest[:i]
	}
	for _, seg := range strings.Split(rest, "/")[1:] {
		if urlMagicWords[strings.ToLower(seg)] {
			return true
		}
	}
	return false
}

func urlTokenAround(text string, s int) (int, int) {
	a := s
	for a > 0 && s-a < 2048 && !isURLStop(text[a-1]) {
		a--
	}
	b := s
	for b < len(text) && b-s < 2048 && !isURLStop(text[b]) {
		b++
	}
	return a, b
}

func isURLStop(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '"' || c == '\'' || c == '<' || c == '>' || c == '`'
}

func insideURL(text string, s int) bool {
	a, _ := urlTokenAround(text, s)
	tok := strings.ToLower(text[a:s])
	return strings.Contains(tok, "://") || strings.HasPrefix(tok, "www.")
}

const queryKeyLimit = 256

func querySecretValid(text string, s, e int) bool {
	if placeholderValue(text[s:e]) {
		return false
	}
	k := s - 1
	for k > 0 && s-1-k < queryKeyLimit && text[k-1] != '?' && text[k-1] != '&' && text[k-1] != ';' {
		k--
	}
	key := text[k : s-1]
	a, b := urlTokenAround(text, s)
	return urlParamSecret(key, text[s:e], urlMagicPath(text[a:b]))
}

func urlUnescape(v string) string {
	if d, err := url.QueryUnescape(v); err == nil {
		v = d
	}
	return strings.TrimRight(v, ".,;:!)")
}

func urlPasswordValue(v string) bool {
	v = urlUnescape(v)
	return v != "" && !strings.HasPrefix(v, "/") && !strings.Contains(v, "://") && !urlBarePlaceRE.MatchString(v)
}

func urlStrongValue(v string) bool {
	v = urlUnescape(v)
	if v == "" || strings.HasPrefix(v, "/") || strings.Contains(v, "://") || urlPlaceholderRE.MatchString(v) || urlTrivialRE.MatchString(v) {
		return false
	}
	letter := strings.IndexFunc(v, func(r rune) bool { return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' }) >= 0
	digit := strings.ContainsAny(v, "0123456789")
	return len(v) >= urlStrongMinLen || len(v) >= urlMixedMinLen && letter && digit
}

func urlBase64Value(v string) bool {
	v = strings.TrimRight(v, ".,;:!)")
	if d, err := url.QueryUnescape(v); err == nil {
		v = d
	}
	return urlBase64RE.MatchString(v) && (strings.HasSuffix(v, "=") || strings.ContainsAny(v, "0123456789")) && !slugPath(strings.ToLower(v))
}

func urlCredentialValue(v string, strong bool) bool {
	if d, err := url.QueryUnescape(v); err == nil {
		v = d
	}
	v = strings.TrimRight(v, ".,;:!)")
	if v == "" || strings.HasPrefix(v, "/") || strings.Contains(v, "://") || urlPlaceholderRE.MatchString(v) {
		return false
	}
	if strong {
		return true
	}
	if !urlTokenShapeRE.MatchString(v) {
		return false
	}
	hasDigit, hasLetter := false, false
	for i := 0; i < len(v); i++ {
		switch c := v[i]; {
		case c >= '0' && c <= '9':
			hasDigit = true
		case c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z':
			hasLetter = true
		}
	}
	return hasDigit && hasLetter && !slugPath(v)
}
