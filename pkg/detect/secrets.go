package detect

import (
	"encoding/base64"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

const vendorTokenGate = "glpat-|gldt-|glrt-|glcbt-|glptt-|glsoat-|glimt-|glft-|GR1348941|xapp-|xoxe|whsec_|ya29.|GOCSPX-|pypi-AgE|rubygems_|dckr_pat_|_v1_|shpat_|shpca_|shppa_|shpss_|sq0atp-|sq0csp-|dapi|lin_api_|ntn_|secret_|PMAK-|sntrys_|sntryu_|glc_|glsa_|NRAK-|NRII-|hvs.|hvb.|.atlasv1.|dp.pt.|dp.st.|dp.ct.|dp.sa.|sbp_|sb_secret_|pscale_|ATATT3|figd_|r8_|gsk_|xai-|pplx-|tskey-|AGE-SECRET-KEY-1|key-|xkeysib-|-us|EAA|oy2|ops_eyJ|A3-|fo1_|fm1_|fm2_|HRKU-|ABIA|ACCA|SK|:AA|hooks.slack.com|/api/webhooks/|webhook.office.com|ingest."

var vendorTokenRE = regexp.MustCompile(`\b(?:` +
	`gl(?:pat|dt|rt|cbt|ptt|soat|imt|ft)-[A-Za-z0-9_\-]{20,}|GR1348941[A-Za-z0-9_\-]{20,}|` +
	`xapp-\d-[A-Z0-9]+-\d+-[A-Za-z0-9]+|xoxe(?:\.xox[bp])?-\d-[A-Za-z0-9\-]{20,}|whsec_[A-Za-z0-9+/=]{24,}|` +
	`ya29\.[A-Za-z0-9_\-]{20,}|GOCSPX-[A-Za-z0-9_\-]{20,}|pypi-AgE[A-Za-z0-9_\-]{50,}|rubygems_[a-f0-9]{48}|dckr_pat_[A-Za-z0-9_\-]{20,}|` +
	`do[opr]_v1_[a-f0-9]{64}|shp(?:at|ca|pa|ss)_[a-fA-F0-9]{32}|sq0(?:atp|csp)-[A-Za-z0-9_\-]{22,}|dapi[a-f0-9]{32}|lin_api_[A-Za-z0-9]{40}|` +
	`ntn_[A-Za-z0-9]{40,}|secret_[A-Za-z0-9]{43}|PMAK-[a-f0-9]{24}-[a-f0-9]{34}|sntry[su]_[A-Za-z0-9+/=_\-]{40,}|glc_[A-Za-z0-9+/=]{32,}|glsa_[A-Za-z0-9]{32}_[a-f0-9]{8}|` +
	`NR(?:AK|II)-[A-Za-z0-9_\-]{27,}|hv[sb]\.[A-Za-z0-9_\-]{24,}|[A-Za-z0-9]{14}\.atlasv1\.[A-Za-z0-9_\-=]{60,}|dp\.(?:pt|st|ct|sa)\.[A-Za-z0-9]{40,}|` +
	`sbp_[a-f0-9]{40}|sb_secret_[A-Za-z0-9_\-]{20,}|pscale_(?:tkn|pw|oauth)_[A-Za-z0-9_\-.]{32,}|ATATT3[A-Za-z0-9_\-=]{50,}|figd_[A-Za-z0-9_\-]{40,}|` +
	`r8_[A-Za-z0-9]{37}|gsk_[A-Za-z0-9]{40,}|xai-[A-Za-z0-9]{40,}|pplx-[A-Za-z0-9]{40,}|tskey-[a-z]+-[A-Za-z0-9]{8,}(?:-[A-Za-z0-9]{8,})?|` +
	`AGE-SECRET-KEY-1[QPZRY9X8GF2TVDW0S3JN54KHCE6MUA7L]{58}\b|key-[0-9a-f]{32}|xkeysib-[a-f0-9]{64}-[A-Za-z0-9]{16}|[0-9a-f]{32}-us\d{1,2}|EAA[A-Za-z0-9_\-]{60,}|oy2[a-z0-9]{43}|` +
	`ops_eyJ[A-Za-z0-9+/=_\-]{100,}|A3-[A-Z0-9]{6}-(?:[A-Z0-9]{11}|[A-Z0-9]{6}-[A-Z0-9]{5})-[A-Z0-9]{5}-[A-Z0-9]{5}-[A-Z0-9]{5}|` +
	`fo1_[A-Za-z0-9_\-]{40,}|fm[12]_[A-Za-z0-9+/=_\-]{40,}|HRKU-[A-Za-z0-9_\-]{30,}|(?:ABIA|ACCA)[0-9A-Z]{16}|SK[0-9a-f]{32}|\d{8,10}:AA[A-Za-z0-9_\-]{33}|` +
	`https://hooks\.slack\.com/(?:services|workflows|triggers)/[A-Za-z0-9/_\-]{20,}|https://(?:canary\.|ptb\.)?discord(?:app)?\.com/api/webhooks/\d+/[A-Za-z0-9_\-]{20,}|` +
	`https://[a-z0-9\-]+\.webhook\.office\.com/[^\s"'<>` + "`" + `]{20,}|https://[a-f0-9]{32}(?::[a-f0-9]{32})?@[a-z0-9.\-]*ingest[a-z0-9.\-]*/\d+` +
	`)`)

var discordTokenRE = regexp.MustCompile(`\b[MNO][A-Za-z\d_\-]{23,25}\.[A-Za-z\d_\-]{6}\.[A-Za-z\d_\-]{27,38}\b`)

func tokenish(v string) bool {
	if len(v) < 8 || placeholderValue(v) || secretDottedRE.MatchString(v) && !hasDigit(v) {
		return false
	}
	var lower, upperInside bool
	for i := 0; i < len(v); i++ {
		c := v[i]
		if c >= 'a' && c <= 'z' {
			lower = true
		} else if c >= 'A' && c <= 'Z' && i > 0 {
			upperInside = true
		}
	}
	return hasDigit(v) || lower && upperInside && len(v) >= 16 || strings.HasSuffix(v, "=")
}

var placeholderParts = strings.Fields(`xxxx **** changeme change-me change_me placeholder redacted your_ your- yourpass yourkey dummy foobar`)
var placeholderExact = wordSet(`todo null none nil undefined password passwort secret pass test insert replace sample string true false required optional`)

func placeholderValue(v string) bool {
	low := asciiLower(v)
	if strings.Trim(low, "x*.-_#•?0") == "" {
		return true
	}
	if strings.ContainsAny(v, "<>{}") || strings.HasPrefix(v, "$") || strings.HasPrefix(v, "%") || strings.HasPrefix(low, "env.") || strings.HasPrefix(low, "process.") || strings.HasPrefix(low, "os.") {
		return true
	}
	for _, w := range placeholderParts {
		if strings.Contains(low, w) {
			return true
		}
	}
	return placeholderExact[low]
}

func kvSecretValid(text string, s, e int) bool {
	v := text[s:e]
	if placeholderValue(v) || !secretValueValid(text, s, e) {
		return false
	}
	if strings.EqualFold(v, "true") || strings.EqualFold(v, "false") {
		return false
	}
	return hasDigit(v) || strings.ContainsAny(v, "!@#$%^&*+?~")
}

var kvNames = strings.Fields(`secret token password passwort passwd pwd credential credentials passphrase sharedaccesssignature key`)
var kvKeyPrefixes = strings.Fields(`api access secret private signing encryption master auth license account sharedaccess`)
var kvHeadRE = regexp.MustCompile(`^(?:[_\-]?(?:value|v\d|\d))?["']?[ \t]*[:=][ \t]*["']?`)

const kvValueStops = " \t\n\r\f\"'`,;&"

func kvValueEnd(text string, s int) int {
	e := s
	for e < len(text) && strings.IndexByte(kvValueStops, text[e]) < 0 {
		e++
	}
	return e
}

func scanKeyValueSecrets(text, lower string) []entity.Span {
	var out []entity.Span
	low := sameLengthLower(text, lower)
	find := func(words []string) {
		for _, w := range words {
			for from := 0; from < len(low); {
				i := strings.Index(low[from:], w)
				if i < 0 {
					break
				}
				ks := from + i
				p := ks + len(w)
				from = p
				if p >= len(text) || strings.IndexByte("_-\"' \t:=0123456789v", low[p]) < 0 {
					continue
				}
				if w == "key" {
					before := strings.TrimRight(low[max(0, ks-14):ks], "_-")
					ok := false
					for _, pre := range kvKeyPrefixes {
						ok = ok || strings.HasSuffix(before, pre)
					}
					if !ok {
						continue
					}
				}
				m := kvHeadRE.FindStringIndex(text[p:min(len(text), p+scanWindow)])
				if m == nil {
					continue
				}
				s := p + m[1]
				e := kvValueEnd(text, s)
				if utf8.RuneCountInString(text[s:e]) < 8 {
					continue
				}
				from = e
				for e > s && (text[e-1] == '.' || text[e-1] == ')') {
					e--
				}
				if boundaryOK(text, s, e) && kvSecretValid(text, s, e) {
					out = append(out, entity.Span{Start: s, End: e})
				}
			}
		}
	}
	find(kvNames)
	return out
}

var pctEncodedRE = regexp.MustCompile(`%[0-9A-Fa-f]{2}`)
var wordNumberRE = regexp.MustCompile(`^[A-Za-z][A-Za-z.\-]*[#@]\d+$`)
var fileExtRE = regexp.MustCompile(`\.(?:[a-z]{2,5}|[A-Z]{2,4})$`)
var tildeRevRE = regexp.MustCompile(`[~^]\d*$`)

var referenceWords = wordSet(`issue issues pr mr cl ticket bug room section item case ref no nr step page task story jira build run job id order invoice version release chapter figure fig table line row level floor gate lane unit apt suite batch lot track channel thread post comment commit rev part vol volume episode season week day slide note appendix exhibit clause article rule test`)

const hardSymbols = "!@#$%^*?~"

func credScore(tok string, loose bool) int {
	n := len(tok)
	if n < 6 || n > 64 || !loose && n < 8 {
		return 0
	}
	if strings.ContainsAny(tok, "/\\()[]{}<>|=,;:\"'` \t") || strings.IndexByte("$%&*#/.@-~:?+^!_", tok[0]) >= 0 {
		return 0
	}
	if placeholderValue(tok) || pctEncodedRE.MatchString(tok) || secretDottedRE.MatchString(tok) || secretWordyRE.MatchString(tok) || secretNumericRE.MatchString(tok) || fileExtRE.MatchString(tok) || tildeRevRE.MatchString(tok) {
		return 0
	}
	low := asciiLower(tok)
	for _, p := range secretTechPrefixes {
		if strings.HasPrefix(low, p) {
			return 0
		}
	}
	if at := strings.IndexByte(tok, '@'); at > 0 {
		after := tok[at+1:]
		if after == asciiLower(after) || strings.Contains(after, ".") {
			return 0
		}
	}
	if h := strings.IndexByte(tok, '#'); h > 0 && strings.Contains(tok[:h], ".") {
		return 0
	}
	if wordNumberRE.MatchString(tok) && referenceWords[strings.TrimRight(low, "0123456789#@")] {
		return 0
	}
	if !loose && (wordNumberRE.MatchString(tok) || dollarDigit(tok)) {
		return 0
	}
	end := n
	for end > 0 && strings.IndexByte("!?.", tok[end-1]) >= 0 {
		end--
	}
	var lower, upper, upperInside, digit, soft, hard bool
	for i := 0; i < n; i++ {
		c := tok[i]
		switch {
		case c >= 'a' && c <= 'z' || c >= 0x80:
			lower = true
		case c >= 'A' && c <= 'Z':
			upper = true
			if i > 0 {
				upperInside = true
			}
		case isDigitByte(c):
			digit = true
		case strings.IndexByte(hardSymbols, c) >= 0:
			if i < end {
				hard = true
			}
		default:
			soft = true
		}
	}
	letter := lower || upper
	switch {
	case !letter:
		return 0
	case digit && upper && lower && hard:
		return 4
	case digit && hard:
		return 3
	case hard && upper && lower, digit && upper && lower && soft, digit && upperInside && lower:
		return 2
	case digit:
		return 1
	}
	return 0
}

func dollarDigit(tok string) bool {
	i := strings.IndexByte(tok, '$')
	return i >= 0 && i+1 < len(tok) && isDigitByte(tok[i+1])
}

func isBlankByte(c byte) bool { return c == ' ' || c == '\t' || c == '\r' }

func stripTokenWrap(text string, s, e int) (int, int) {
	for s < e && strings.IndexByte("\"'`([<*", text[s]) >= 0 {
		s++
	}
	for e > s && strings.IndexByte("\"'`)]>*.,;:", text[e-1]) >= 0 {
		e--
	}
	return s, e
}

func nextToken(text string, from, lim int) (int, int) {
	s := from
	for s < lim && isBlankByte(text[s]) {
		s++
	}
	e := s
	for e < lim && !isBlankByte(text[e]) {
		e++
	}
	return s, e
}

var credNounSet = newLabelSet(`password|passwords|passwd|pwd|pw|passphrase|passcode|passwort|kennwort|mot de passe|mdp|contraseña|contrasena|clave de acceso|senha|wachtwoord|hasło|haslo|lösenord|losenord|salasana|heslo|creds|credential|credentials|secret answer|security answer|break-glass`, "")
var credVerbSet = newLabelSet(`login|log in|sign in|logon|wifi|wi-fi|wlan`, "")

func wordEdges(low string, s, e int) bool {
	if s > 0 && (isAlnumByte(low[s-1]) || low[s-1] == '_' || low[s-1] >= 0x80) {
		return false
	}
	return e >= len(low) || !(isAlnumByte(low[e]) || low[e] == '_' || low[e] >= 0x80)
}

var credGate = gateLower("pass|pwd|pw|mdp|contrase|clave|senha|wachtwoord|hasł|haslo|senord|salasana|heslo|creds|credential|answer|break-glass|login|log in|sign in|logon|wifi|wi-fi|wlan")
var copulas = wordSet(`is ist est es é è lautet was are = -> → sera será jest är to als`)
var pairUserRE = regexp.MustCompile(`^[A-Za-z0-9._@\-]{2,32}$`)

func credCandidate(text string, s, e, min int, pair, loose bool) (int, int, bool) {
	s, e = stripTokenWrap(text, s, e)
	if e-s < 4 {
		return 0, 0, false
	}
	tok := text[s:e]
	if pair {
		if c := strings.IndexByte(tok, ':'); c > 0 && c < len(tok)-4 && pairUserRE.MatchString(tok[:c]) && !isDigitByte(tok[c-1]) {
			need := min - 1
			if need < 1 {
				need = 1
			}
			if credScore(tok[c+1:], true) >= need {
				return s + c + 1, e, true
			}
			return 0, 0, false
		}
	}
	return s, e, credScore(tok, loose) >= min
}

func scanCredentials(text, lower string) []entity.Span {
	var out []entity.Span
	low := sameLengthLower(text, lower)
	visit := func(cs, ce int, noun bool) {
		if !wordEdges(low, cs, ce) || strings.Contains(low[max(0, cs-12):cs], "[redacted") {
			return
		}
		le, wholeLine := lineEndWithin(text, ce, scanWindow)
		min := 3
		found := false
		lim := ce + 72
		if lim > le {
			lim = le
		}
		prevWord := ""
		for from := ce; from < lim && !found; {
			ts, te := nextToken(text, from, le)
			if ts >= te || ts >= lim || strings.HasPrefix(text[ts:], "[redacted") {
				break
			}
			from = te
			need := min
			switch {
			case noun && (copulas[prevWord] || strings.HasSuffix(prevWord, ":") || strings.HasSuffix(prevWord, "=")), prevWord == "/":
				need = 1
			case noun && prevWord == "":
				need = 2
			}
			if s, e, ok := credCandidate(text, ts, te, need, true, noun); ok {
				out = append(out, entity.Span{Start: s, End: e})
				found = true
				break
			}
			raw := text[ts:te]
			if c := raw[len(raw)-1]; (c == '.' || c == '?' || c == '!') && te+1 < le && text[te+1] >= 'A' && text[te+1] <= 'Z' {
				break
			}
			prevWord = low[ts:te]
		}
		if found {
			return
		}
		rest := strings.Trim(text[ce:le], " \t\r:=-")
		if noun && rest == "" && wholeLine && le < len(text) {
			ns := le + 1
			ne, wholeNext := lineEndWithin(text, ns, scanWindow)
			if fields := strings.Fields(text[ns:ne]); wholeNext && len(fields) >= 1 && len(fields) <= 2 {
				ts, te := nextToken(text, ns, ne)
				if s, e, ok := credCandidate(text, ts, te, 1, true, true); ok {
					out = append(out, entity.Span{Start: s, End: e})
					return
				}
			}
		}
		start, atLineStart := lineStartWithin(text, cs, 56)
		for !atLineStart && start < cs && text[start-1] != ' ' {
			start++
		}
		for from := start; from < cs; {
			ts, te := nextToken(text, from, cs)
			if ts >= te {
				break
			}
			from = te
			if s, e, ok := credCandidate(text, ts, te, 3, false, false); ok {
				out = append(out, entity.Span{Start: s, End: e})
			}
		}
	}
	credNounSet.each(text, low, func(s, e int) { visit(s, e, true) })
	credVerbSet.each(text, low, func(s, e int) { visit(s, e, false) })
	return out
}

func passwordShapeGate(text, _ string) bool { return strings.ContainsAny(text, hardSymbols) }

func scanPasswordShape(text, _ string) []entity.Span {
	var out []entity.Span
	n := len(text)
	for i := 1; i+1 < n; i++ {
		if strings.IndexByte(hardSymbols, text[i]) < 0 || !isAlnumByte(text[i-1]) {
			continue
		}
		s := i
		for s > 0 && !isSpaceByte(text[s-1]) {
			s--
		}
		e := i
		for e < n && !isSpaceByte(text[e]) {
			e++
		}
		ts, te := stripTokenWrap(text, s, e)
		if te-ts >= 10 && !strings.Contains(text[ts:te], "@") && len(digitsOnly(text[ts:te])) >= 2 && credScore(text[ts:te], false) == 4 {
			out = append(out, entity.Span{Start: ts, End: te})
		}
		i = e
	}
	return out
}

func isSpaceByte(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

var recoveryCueRE = regexp.MustCompile(`(?:backup|recovery|rescue|scratch|emergency|single-use)[ -](?:codes?|keys?)|codes de (?:secours|récupération|recuperation)|wiederherstellungscodes?|backup-codes?|códigos de (?:respaldo|recuperación|recuperacion)|codici di (?:backup|recupero)|herstelcodes?`)
var recoveryCodeRE = regexp.MustCompile(`^(?:[A-Za-z0-9]{4,6}(?:-[A-Za-z0-9]{4,6}){1,3}|[a-z0-9]{8,12}|[A-Z0-9]{8,12}|\d{6,10})$`)

func recoveryCode(tok string) bool {
	if !recoveryCodeRE.MatchString(tok) || !hasDigit(tok) || secretNumericRE.MatchString(tok) {
		return false
	}
	return !(allDigits(tok) && len(tok) == 8 && (strings.HasPrefix(tok, "19") || strings.HasPrefix(tok, "20")))
}

func recoveryLine(text string, ls, le int, strict bool) ([]entity.Span, bool) {
	var out []entity.Span
	for from := ls; from < le; {
		rs, re := nextToken(text, from, le)
		if rs >= re {
			break
		}
		from = re
		if strings.HasPrefix(text[rs:], "[redacted") {
			return out, true
		}
		ts, te := stripTokenWrap(text, rs, re)
		tok := text[ts:te]
		if len(tok) == 4 && allDigits(tok) && re+5 <= le && text[re] == ' ' && allDigits(text[re+1:re+5]) && (re+5 == le || !isAlnumByte(text[re+5])) {
			out = append(out, entity.Span{Start: ts, End: re + 5})
			from = re + 5
			continue
		}
		if recoveryCode(tok) {
			out = append(out, entity.Span{Start: ts, End: te})
			continue
		}
		marker := tok == "" || tok == "-" || tok == "•" || tok == "·" || len(tok) <= 2 && allDigits(tok)
		if strict && !marker {
			return nil, false
		}
	}
	return out, len(out) > 0
}

const recoveryWindow = 512

func scanRecoveryCodes(text, lower string) []entity.Span {
	var out []entity.Span
	low := sameLengthLower(text, lower)
	done, listed := 0, -1
	for _, m := range recoveryCueRE.FindAllStringIndex(low, -1) {
		le, whole := lineEndWithin(text, m[1], recoveryWindow)
		for !whole && le > m[1] && !isBlankByte(text[le-1]) {
			le--
		}
		if from := max(m[1], done); from < le {
			found, _ := recoveryLine(text, from, le, false)
			out = append(out, found...)
			done = le
		}
		if !whole || le == listed {
			continue
		}
		listed = le
		blanks := 0
		for lines := 0; le < len(text) && lines < 24; lines++ {
			ls := le + 1
			if le, whole = lineEndWithin(text, ls, recoveryWindow); !whole {
				break
			}
			if strings.TrimSpace(text[ls:le]) == "" {
				if blanks++; blanks > 1 {
					break
				}
				continue
			}
			codes, ok := recoveryLine(text, ls, le, true)
			if !ok {
				break
			}
			out = append(out, codes...)
		}
	}
	return out
}

var cookieLineRE = regexp.MustCompile(`(?i)\b(?:set-)?cookie["']?[ \t]*:[ \t]*["']?([^\n"']{8,})`)
var cookiePairRE = regexp.MustCompile(`([A-Za-z0-9_.\-]+)=([^;\s,]+)`)
var cookieAttrs = wordSet(`path domain expires max-age samesite secure httponly priority version comment`)

func scanCookies(text, _ string) []entity.Span {
	var out []entity.Span
	for _, m := range cookieLineRE.FindAllStringSubmatchIndex(text, -1) {
		for _, p := range cookiePairRE.FindAllStringSubmatchIndex(text[m[2]:m[3]], -1) {
			name := strings.ToLower(text[m[2]+p[2] : m[2]+p[3]])
			s, e := m[2]+p[4], m[2]+p[5]
			if cookieAttrs[name] || e-s < 8 || placeholderValue(text[s:e]) {
				continue
			}
			sensitive := strings.Contains(name, "sess") || strings.Contains(name, "sid") || strings.Contains(name, "token") || strings.Contains(name, "auth") || strings.Contains(name, "jwt") || strings.Contains(name, "csrf") || strings.Contains(name, "xsrf")
			if sensitive || e-s >= 16 && hasDigit(text[s:e]) {
				out = append(out, entity.Span{Start: s, End: e})
			}
		}
	}
	return out
}

var otpLabels = newLabelSet(`code de vérification|code de verification|code de sécurité|code de securite|code de confirmation|code de validation|code à usage unique|bestätigungscode|sicherheitscode|verifizierungscode|einmalcode|freischaltcode|código de verificación|codigo de verificacion|código de seguridad|codigo de seguridad|código de confirmación|codigo de confirmacion|código de acceso|codigo de acceso|código de verificação|código de segurança|código de confirmação|codice di verifica|codice di sicurezza|codice di conferma|verificatiecode|beveiligingscode|bevestigingscode|kod weryfikacyjny|kod sms|sms code|sms-code|auth code|authentication code|authenticator code|backup code|recovery code`, "")
var otpValueRE = regexp.MustCompile(`^[^\d\n]{0,30}?\b([A-Z]-?\d{4,8}|\d{3}[- ]\d{3}|\d{4,8})\b`)

func scanIntlOTP(text, lower string) []entity.Span {
	var out []entity.Span
	otpLabels.each(text, sameLengthLower(text, lower), func(_, p int) {
		m := otpValueRE.FindStringSubmatchIndex(text[p:min(len(text), p+48)])
		if m == nil {
			return
		}
		s, e := p+m[2], p+m[3]
		if boundaryOK(text, s, e) && idTokenEnd(text, e) && text[s-1] != '+' {
			out = append(out, entity.Span{Start: s, End: e})
		}
	})
	return out
}

func seedPhraseValid(text string, s, e int) bool {
	switch len(strings.FieldsFunc(text[s:e], func(r rune) bool { return r == ' ' || r == ',' })) {
	case 12, 15, 18, 21, 24:
		return true
	}
	return false
}

func base64Text(v string) bool {
	v = strings.TrimRight(v, "=")
	for _, enc := range []*base64.Encoding{base64.RawStdEncoding, base64.RawURLEncoding} {
		raw, err := enc.DecodeString(v)
		if err != nil || len(raw) < 12 {
			continue
		}
		printable := true
		for _, b := range raw {
			if b != '\n' && b != '\r' && b != '\t' && (b < 0x20 || b > 0x7e) {
				printable = false
				break
			}
		}
		if printable {
			return true
		}
	}
	return false
}

var blobContextRE = regexp.MustCompile(`(?i)(?:\b(?:integrity|checksum|digest|resolved|sha(?:1|256|384|512)|md5|fingerprint|etag|content-md5|x-amz-request-id|base64|ssh-(?:rsa|ed25519|dss)|ecdsa-sha2-nistp\d+)\b|\bh1:$)`)
var cloudResourceRE = regexp.MustCompile(`^(?:subnet|vpc|sg|ami|vol|snap|eni|igw|rtb|acl|i|eipalloc|nat|tgw|lt|fs|pcx|vpce)-[0-9a-f]{8,}`)
var camelWordRE = regexp.MustCompile(`[A-Z]?[a-z]{2,}`)

func identifierLike(v string) bool {
	if len(v) < 8 {
		return false
	}
	covered := 0
	for _, m := range camelWordRE.FindAllStringIndex(v, -1) {
		covered += m[1] - m[0]
	}
	return covered*4 >= len(v)*3
}

func insidePublicPEM(text string, s int) bool {
	from := max(0, s-4096)
	i := strings.LastIndex(text[from:s], "-----BEGIN ")
	if i < 0 {
		return false
	}
	head := text[from+i : s]
	if strings.Contains(head, "-----END ") {
		return false
	}
	nl := strings.IndexByte(head, '\n')
	return nl > 0 && !strings.Contains(head[:nl], "PRIVATE")
}

func highEntropyValid(text string, s, e int) bool {
	v := text[s:e]
	if !looksRandomToken(strings.TrimPrefix(v, "0x")) || placeholderValue(v) || identifierLike(v) || cloudResourceRE.MatchString(v) {
		return false
	}
	for _, seg := range strings.Split(v, "/") {
		if identifierLike(seg) {
			return false
		}
	}
	if !highEntropyContextOK(text, s, e) {
		return false
	}
	if strings.Count(v, "/") >= 2 {
		wordy := 0
		segs := strings.Split(v, "/")
		for _, seg := range segs {
			if len(seg) <= 24 && len(digitsOnly(seg))*4 <= len(seg) {
				wordy++
			}
		}
		if wordy == len(segs) {
			return false
		}
	}
	return true
}

func highEntropyContextOK(text string, s, e int) bool {
	v := text[s:e]
	if base64Text(v) || insidePublicPEM(text, s) {
		return false
	}
	low := asciiLower(v)
	for _, p := range []string{"sha1-", "sha256-", "sha384-", "sha512-", "md5-", "data:"} {
		if strings.HasPrefix(low, p) {
			return false
		}
	}
	if s > 0 && strings.IndexByte("/\\.@#", text[s-1]) >= 0 {
		return false
	}
	ls, wholeBefore := lineStartWithin(text, s, scanWindow)
	le, wholeAfter := lineEndWithin(text, e, scanWindow)
	if blobContextRE.MatchString(text[max(ls, s-200):s]) {
		return false
	}
	if e-s >= 60 && wholeBefore && wholeAfter && strings.TrimSpace(text[ls:s]) == "" && strings.TrimSpace(text[e:le]) == "" {
		same := func(a, b int) bool {
			l := strings.TrimSpace(text[a:b])
			return len(l) == e-s && strings.Trim(l, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=") == ""
		}
		reach := e - s + scanWindow
		if ls > 0 {
			if ps, ok := lineStartWithin(text, ls-1, reach); ok && same(ps, ls-1) {
				return false
			}
		}
		if le < len(text) {
			if ne, ok := lineEndWithin(text, le+1, reach); ok && same(le+1, ne) {
				return false
			}
		}
	}
	return true
}

func curlUserValid(text string, s, e int) bool {
	v := text[s:e]
	return !placeholderValue(v) && !strings.HasPrefix(v, "$") && asciiLower(v) != "pass"
}

var privateKeyOpenRE = regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY(?: BLOCK)?-----(?:\\[nr]|\s)*(?:(?:[A-Za-z][A-Za-z\-]*: [^\n]*\n)*(?:\\[nr]|\s)*)?(?:[A-Za-z0-9+/=]{16,}(?:\\[nr]|\s)*)+`)

const base64Bytes = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="

func pemBreakBefore(text string, e int) int {
	switch {
	case strings.HasSuffix(text[:e], "\r\n"):
		return e - 2
	case strings.HasSuffix(text[:e], "\n"):
		return e - 1
	case strings.HasSuffix(text[:e], `\n`):
		if strings.HasSuffix(text[:e], `\r\n`) {
			return e - 4
		}
		return e - 2
	}
	return -1
}

func pemTailEnd(text string, e int) int {
	b := pemBreakBefore(text, e)
	if b <= 0 || strings.IndexByte(base64Bytes, text[b-1]) < 0 {
		return e
	}
	t := e
	for t < len(text) && t-e < 16 && strings.IndexByte(base64Bytes, text[t]) >= 0 {
		t++
	}
	tail, rest := text[e:t], text[t:]
	if tail == "" || len(tail) > 15 {
		return e
	}
	if rest != "" && rest[0] != '\n' && rest[0] != '"' && !strings.HasPrefix(rest, "\r\n") && !strings.HasPrefix(rest, `\n`) && !strings.HasPrefix(rest, `\r`) {
		return e
	}
	var lower, upperInside bool
	for i := 0; i < len(tail); i++ {
		lower = lower || tail[i] >= 'a' && tail[i] <= 'z'
		upperInside = upperInside || i > 0 && tail[i] >= 'A' && tail[i] <= 'Z'
	}
	if strings.ContainsAny(tail, "0123456789+/=") || lower && upperInside {
		return t
	}
	return e
}

func scanOpenPrivateKeys(text, _ string) []entity.Span {
	var out []entity.Span
	for _, m := range privateKeyOpenRE.FindAllStringIndex(text, -1) {
		out = append(out, entity.Span{Start: m[0], End: pemTailEnd(text, m[1])})
	}
	return out
}

var secretPatterns = []pattern{
	{name: "vendor_token", gate: gateExact(vendorTokenGate), typ: entity.Secret, label: "key", priority: 91, trim: ".,;:!?'\")",
		re: vendorTokenRE},
	{name: "vendor_token", gate: gateLower("discord|bot|token"), typ: entity.Secret, label: "token", priority: 91,
		re: discordTokenRE},
	{name: "private_key_open", gate: gateExact("PRIVATE KEY"), typ: entity.Secret, label: "key", priority: 99, scan: scanOpenPrivateKeys},
	{name: "private_key", gate: gateExact("Private-Lines:"), typ: entity.Secret, label: "key", priority: 99, group: 1,
		re: regexp.MustCompile(`Private-Lines: \d+\s+((?:[A-Za-z0-9+/=]{16,}\s*)+)`)},
	{name: "auth_header", gate: gateLower("authorization"), typ: entity.Secret, label: "token", priority: 86, group: 1,
		re:       regexp.MustCompile(`(?i)\b(?:proxy-)?authorization["']?[ \t]*[:=][ \t]*["']?(?:(?:basic|bearer|token|apikey|negotiate|ntlm)[ \t]+)?([A-Za-z0-9._~+/=\-]{8,})`),
		validate: validator(tokenish)},
	{name: "basic_auth", gate: gateLower("basic "), typ: entity.Secret, label: "token", priority: 85, group: 1,
		re: regexp.MustCompile(`\bBasic[ \t]+([A-Za-z0-9+/]{16,}={0,2})`),
		validate: func(text string, s, e int) bool {
			return base64Text(text[s:e]) || strings.HasSuffix(text[s:e], "=")
		}},
	{name: "key_header", gate: gateLower("key|token"), typ: entity.Secret, label: "key", priority: 86, group: 1,
		re:       regexp.MustCompile(`(?i)\b(?:x-api-key|x-auth-token|x-access-token|x-amz-security-token|x-vault-token|x-goog-api-key|api-key|private-token|x-csrf-token|x-xsrf-token|ocp-apim-subscription-key|x-functions-key|x-gitlab-token|x-hub-signature(?:-256)?)["']?[ \t]*[:=][ \t]*["']?([A-Za-z0-9._~+/=\-]{8,})`),
		validate: validator(tokenish)},
	{name: "cookie", gate: gateLower("cookie"), typ: entity.Secret, label: "token", priority: 86, scan: scanCookies},
	{name: "session_id", gate: gateLower("sess|sid=|csrf|xsrf"), typ: entity.Secret, label: "token", priority: 85, group: 1,
		re:       regexp.MustCompile(`(?i)\b(?:phpsessid|jsessionid|asp\.net_sessionid|sessionid|session_id|session-id|connect\.sid|sid|csrftoken|csrf_token|xsrf-token|_session|sessiontoken)["']?[ \t]*[:=][ \t]*["']?([A-Za-z0-9%._~+/\-]{12,})`),
		validate: validator(tokenish)},
	{name: "kv_secret", gate: gateLower("secret|token|passw|pwd|key|credential|passphrase"), typ: entity.Secret, label: "secret", priority: 85, scan: scanKeyValueSecrets},
	{name: "query_secret", gate: gateLower("token=|key=|secret=|sig=|signature=|password=|passwd=|pwd="), typ: entity.Secret, label: "secret", priority: 86, group: 1,
		re:       regexp.MustCompile(`(?i)[?&;](?:access_token|refresh_token|id_token|token|api_key|apikey|key|client_secret|secret|sig|signature|password|passwd|pwd|auth)=([^&\s"'<>#` + "`" + `]+)`),
		validate: querySecretValid},
	{name: "cli_credentials", gate: gateLower("curl|wget|--user|-u |http "), typ: entity.Secret, label: "password", priority: 88, group: 1,
		re:       regexp.MustCompile(`(?:\b(?:curl|wget|http|https|httpie)\b[^\n|;]*?[ \t](?:-u|--user|--proxy-user)|--user|--proxy-user|--http-user)[ \t=]+["']?[^\s:"']+:([^\s"']{3,})`),
		validate: curlUserValid},
	{name: "cli_credentials", gate: gateLower("--pass|--http-password|-p|redis-cli"), typ: entity.Secret, label: "password", priority: 88, group: 1, glued: true,
		re: regexp.MustCompile(`(?:(?:--password|--passwd|--pass|--http-password)[ \t=]+["']?|\b(?:mysql|mysqldump|mariadb|mysqladmin)\b[^\n|;]*?[ \t]-p|\bsshpass[ \t]+-p[ \t]*["']?|\bredis-cli\b[^\n|;]*?[ \t]-a[ \t]+["']?|\bmongo(?:sh|dump|restore)?\b[^\n|;]*?[ \t]-p[ \t]+["']?)([^\s"']{4,})`),
		validate: func(text string, s, e int) bool {
			return curlUserValid(text, s, e) && passwordLike(text[s:e]) && !strings.HasPrefix(text[s:e], "-")
		}},
	{name: "credential_cue", gate: credGate, typ: entity.Secret, label: "password", priority: 83, scan: scanCredentials},
	{name: "high_entropy", gate: passwordShapeGate, typ: entity.Secret, label: "password", priority: 21, scan: scanPasswordShape},
	{name: "recovery_codes", gate: gateLower("backup|recovery|rescue|scratch|emergency|single-use|secours|récupération|recuperation|wiederherstellung|respaldo|recuperación|recupero|herstelcode"), typ: entity.Secret, label: "code", priority: 81, scan: scanRecoveryCodes},
	{name: "recovery_key", gate: gateLower("bitlocker|recovery key|recovery-key|wiederherstellungsschl"), typ: entity.Secret, label: "key", priority: 90,
		re: regexp.MustCompile(`\b\d{6}(?:-\d{6}){7}\b`)},
	{name: "seed_phrase", gate: gateLower("seed|mnemonic|recovery phrase|secret phrase|backup phrase|wallet"), typ: entity.Secret, label: "seed phrase", priority: 90, group: 1,
		re:       regexp.MustCompile(`(?i:\b(?:seed(?:[ ]phrase|[ ]words)?|mnemonic(?:[ ]phrase)?|recovery[ ]phrase|secret[ ](?:recovery[ ])?phrase|backup[ ]phrase|wallet[ ](?:phrase|words))\b)[^\n]{0,40}?[:\n][ \t]*["'` + "`" + `]?((?:[a-z]{3,8}[ ,]+){11,23}[a-z]{3,8})\b`),
		validate: seedPhraseValid},
	{name: "otp_intl", gate: gateLower("code|kod|codice|código|codigo"), typ: entity.Secret, label: "code", priority: 80, scan: scanIntlOTP},
}
