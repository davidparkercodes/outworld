package detect

import (
	"regexp"
	"strings"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

func isAlnumByte(c byte) bool { return isDigitByte(c) || isAlphaByte(c) }

func scanIBAN(text, _ string) []entity.Span {
	var out []entity.Span
	n := len(text)
	for i := 0; i+14 < n; i++ {
		if !isAlphaByte(text[i]) || !isAlphaByte(text[i+1]) || !isDigitByte(text[i+2]) || !isDigitByte(text[i+3]) {
			continue
		}
		if i > 0 && (isAlnumByte(text[i-1]) || text[i-1] >= 0x80 || text[i-1] == '_') {
			continue
		}
		want, ok := ibanLengths[strings.ToUpper(text[i:i+2])]
		if !ok {
			continue
		}
		j, count := i+4, 4
		for j < n && count < want {
			c := text[j]
			if isAlnumByte(c) {
				count++
				j++
				continue
			}
			if (c == ' ' || c == '-') && j+1 < n && isAlnumByte(text[j+1]) && isAlnumByte(text[j-1]) {
				j++
				continue
			}
			break
		}
		if count != want || j < n && (isAlnumByte(text[j]) || text[j] == '_') {
			continue
		}
		if !ValidIBAN(compactAlnum(text[i:j])) {
			continue
		}
		out = append(out, entity.Span{Start: i, End: j})
		i = j - 1
	}
	return out
}

func scanCrypto(text, _ string) []entity.Span {
	var out []entity.Span
	n := len(text)
	for i := 0; i < n; i++ {
		if !isAlnumByte(text[i]) {
			continue
		}
		j := i
		for j < n && isAlnumByte(text[j]) {
			j++
		}
		tok := text[i:j]
		clean := (i == 0 || strings.IndexByte("/_-.=@#", text[i-1]) < 0 && text[i-1] < 0x80) && (j >= n || strings.IndexByte("/_-=@", text[j]) < 0 && text[j] < 0x80)
		if clean && cryptoAddress(tok) {
			out = append(out, entity.Span{Start: i, End: j})
		}
		i = j
	}
	return out
}

func cryptoAddress(tok string) bool {
	switch {
	case len(tok) == 42 && tok[0] == '0' && tok[1] == 'x':
		digits, letters := 0, 0
		for i := 2; i < 42; i++ {
			c := tok[i]
			switch {
			case isDigitByte(c):
				digits++
			case c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F':
				letters++
			default:
				return false
			}
		}
		return digits >= 6 && letters >= 6
	case (len(tok) == 42 || len(tok) == 62) && (strings.HasPrefix(tok, "bc1") || strings.HasPrefix(tok, "tb1")):
		return strings.Trim(tok[3:], "023456789acdefghjklmnpqrstuvwxyz") == "" && hasDigit(tok[3:])
	case len(tok) >= 26 && len(tok) <= 35 && strings.IndexByte("13LMDT29mn", tok[0]) >= 0:
		return base58Check(tok)
	}
	return false
}

var cardRunRE = regexp.MustCompile(`\b(?:\d[ -]?){12,24}\d\b`)
var cardTailRE = regexp.MustCompile(`^[ \t]*[,;|/]?[ \t]*(?:(?i:exp(?:iry|ires|iration|\.)?|valid(?:[ ]thru)?|good[ ]thru)[ :.]*)?(\d{2}[ ]?/[ ]?(?:20)?\d{2})\b(?:[ \t]*[,;|/]?[ \t]*(?:(?i:cvv2?|cvc2?|csc|cvn|code)[ :.]*)?(\d{3,4})\b)?`)

const cardMaxBytes = 19 + 18

func isWordByte(c byte) bool { return isAlnumByte(c) || c == '_' }

func digitRunEnd(text string, s int) int {
	e := s
	for e < len(text) {
		c := text[e]
		if isDigitByte(c) || (c == ' ' || c == '-') && e+1 < len(text) && isDigitByte(text[e+1]) && isDigitByte(text[e-1]) {
			e++
			continue
		}
		break
	}
	return e
}

func cardAt(text string, a, s, e int) int {
	best := 0
	for b := a + 13; b <= e && b-a <= cardMaxBytes; b++ {
		if b < e && text[b] != ' ' && text[b] != '-' {
			continue
		}
		if validCard(text[a:b]) && cardGrouping(text[a:b], a != s || b != e) {
			best = b
		}
	}
	return best
}

func cardGrouping(v string, strict bool) bool {
	groups := strings.FieldsFunc(v, func(r rune) bool { return r == ' ' || r == '-' })
	if len(groups) == 1 {
		return true
	}
	if strict {
		shape := ""
		for _, g := range groups {
			shape += string(rune('0' + len(g)))
		}
		return shape == "4444" || shape == "465" || shape == "44443"
	}
	if strings.Contains(v, " ") && strings.Contains(v, "-") {
		return false
	}
	for _, g := range groups {
		if len(g) < 2 {
			return false
		}
	}
	return len(groups) <= 5
}

func cardStrictGrouped(v string) bool {
	groups := strings.FieldsFunc(v, func(r rune) bool { return r == ' ' || r == '-' })
	return len(groups) > 1 && cardGrouping(v, true)
}

func cardKeyCueAt(text string, c int, cue string) (int, bool) {
	e := c + len(cue)
	if e+1 < len(text) && text[e] == '_' && isDigitByte(text[e+1]) && (c == 0 || !isAlnumByte(text[c-1])) {
		return e, true
	}
	e, ok := cueAt(text, c, cue)
	return e, ok && e < len(text) && strings.IndexByte("=:#", text[e]) >= 0
}

func cardCueOnLine(text string, s int) bool {
	ls, _ := lineStartWithin(text, s, scanWindow)
	low := asciiLower(text[ls:s])
	return len(cueMatchesIn(text, low, ls, cardCueWords, cueAt)) > 0 || len(cueMatchesIn(text, low, ls, cardKeyCueWords, cardKeyCueAt)) > 0
}

func cardSpanOK(text string, s, e int) bool {
	lw := s > 0 && isAlnumByte(text[s-1])
	rw := e < len(text) && isAlnumByte(text[e])
	if lw && rw {
		return false
	}
	if s > 0 && (text[s-1] == '.' || text[s-1] == '/') {
		return false
	}
	special := s > 0 && strings.IndexByte("#=_", text[s-1]) >= 0 || e < len(text) && strings.IndexByte("_/=", text[e]) >= 0
	if lw || rw || special && !cardStrictGrouped(text[s:e]) {
		return cardCueOnLine(text, s)
	}
	return boundaryOK(text, s, e)
}

func scanCards(text, _ string) []entity.Span {
	var out []entity.Span
	n := len(text)
	for i := 0; i < n; i++ {
		if !isDigitByte(text[i]) {
			continue
		}
		s, e := i, digitRunEnd(text, i)
		i = e
		lw := s > 0 && isAlnumByte(text[s-1])
		if lw && !cardCueOnLine(text, s) {
			continue
		}
		if e < n && isWordByte(text[e]) && e > s && text[e] != '_' && (lw || !cardCueOnLine(text, s)) {
			for e < n && isWordByte(text[e]) && e > s {
				e--
				for e > s && isDigitByte(text[e-1]) {
					e--
				}
				for e > s && !isDigitByte(text[e-1]) {
					e--
				}
			}
		}
		for a := s; e-a >= 13; {
			b := cardAt(text, a, s, e)
			if b == 0 || !cardSpanOK(text, a, b) {
				for a < e && isDigitByte(text[a]) {
					a++
				}
				a = min(e, a+1)
				continue
			}
			out = append(out, entity.Span{Start: a, End: b})
			if t := cardTailRE.FindStringSubmatchIndex(text[b:min(n, b+48)]); t != nil {
				out = append(out, entity.Span{Start: b + t[2], End: b + t[3]})
				if t[4] >= 0 {
					out = append(out, entity.Span{Start: b + t[4], End: b + t[5]})
				}
			}
			a = min(e, b+1)
		}
	}
	return out
}

var bicCountries = wordSet(`ad ae al at au az ba be bg bh br by ca ch cl cn co cr cy cz de dk do ee eg es fi fr gb ge gi gr gt hk hr hu id ie il in is it jo jp ke kr kw kz lb li lt lu lv ma mc md me mk mt mu mx my ng nl no nz pe ph pk pl pt qa ro rs ru sa se sg si sk sm th tn tr tw ua us uy va vn za`)

func bicValid(text string, s, e int) bool {
	v := text[s:e]
	return bicCountries[strings.ToLower(v[4:6])]
}

var (
	cardCueWords    = strings.Split("card|cards|visa|mastercard|amex|american express|maestro|discover|cvv|cvc|carte|karte|kreditkarte|tarjeta|cartão|cartao|carta di credito|debit|credit|kaart|creditcard|debitcard|ccnum", "|")
	cardKeyCueWords = []string{"cc", "pan"}
	cardExpiryCues  = withoutWords(cardCueWords, "cvv", "cvc")
)

func withoutWords(list []string, drop ...string) []string {
	skip := wordSet(strings.Join(drop, " "))
	var out []string
	for _, w := range list {
		if !skip[w] {
			out = append(out, w)
		}
	}
	return out
}

func cardExpiryValid(text string, s, e int) bool {
	if mm := num2(text, s); mm < 1 || mm > 12 {
		return false
	}
	ls, _ := lineStartWithin(text, s, scanWindow)
	le, _ := lineEndWithin(text, e, scanWindow)
	ctx := asciiLower(text[ls:le])
	return hasCueWord(ctx, cardExpiryCues) || cardRunRE.MatchString(ctx)
}

func bankNumberValid(text string, s, e int) bool {
	v := text[s:e]
	d := digitsOnly(v)
	if len(d) < 5 || len(d) > 30 || sameDigits(d) {
		return false
	}
	return !idSlashDateRE.MatchString(v) && !idDateRE.MatchString(v) && !idPhoneRE.MatchString(v)
}

var bankLabels = newLabelSet(`kontonummer|konto-nr|kontonr|konto nr|kto-nr|kto.-nr|kto. nr|numéro de compte|numero de compte|n° de compte|compte bancaire|número de cuenta|numero de cuenta|nº de cuenta|n° de cuenta|no. de cuenta|cuenta bancaria|cuenta corriente|numero di conto|numero conto|conto corrente|rekeningnummer|rekeningnr|bankrekening|número da conta|numero da conta|conta corrente|conta bancária|numer konta|numer rachunku|nr konta|bank account|checking account|savings account|current account|bankleitzahl|bank code|code banque|code guichet|transit number|transit no|transit #|institution number|institution no|branch code|agência|bankgiro|plusgiro`, `CLABE|BSB|BLZ|RIB`)

var bankValueRE = regexp.MustCompile(`^[^\d\n]{0,20}(\d[\d \-./]{2,32}\d)`)

func scanBankNumbers(text, lower string) []entity.Span {
	var out []entity.Span
	bankLabels.each(text, sameLengthLower(text, lower), func(_, p int) {
		m := bankValueRE.FindStringSubmatchIndex(text[p:min(len(text), p+56)])
		if m == nil {
			return
		}
		s, e := p+m[2], p+m[3]
		if boundaryOK(text, s, e) && bankNumberValid(text, s, e) {
			out = append(out, entity.Span{Start: s, End: e})
		}
	})
	return out
}

var financialPatterns = []pattern{
	{name: "credit_card", typ: entity.Financial, label: "card", priority: 70, scan: scanCards},
	{name: "iban", typ: entity.Financial, label: "account", priority: 70, scan: scanIBAN},
	{name: "crypto_address", typ: entity.Financial, label: "wallet", priority: 70, scan: scanCrypto},
	{name: "bic", gate: gateLower("bic|swift"), typ: entity.Financial, label: "account", priority: 72, group: 1,
		re:       regexp.MustCompile(`(?i:\b(?:bic|swift)(?:[ /\-]?(?:code|bic|swift|nr\.?|no\.?))?)[ ]?[:#=]?[ ]?(?:(?i:is)[ ])?\b([A-Z]{6}[A-Z0-9]{2}(?:[A-Z0-9]{3})?)\b`),
		validate: bicValid},
	{name: "sort_code", gate: gateLower("sort code|sort-code|sortcode"), typ: entity.Financial, label: "account", priority: 73, group: 1,
		re:       regexp.MustCompile(`(?i)\bsort[ -]?code\b[^\d\n]{0,15}(\d{2}[- ]?\d{2}[- ]?\d{2})\b`),
		validate: func(text string, s, e int) bool { return !sameDigits(digitsOnly(text[s:e])) }},
	{name: "bank_number_intl", typ: entity.Financial, label: "account", priority: 72, scan: scanBankNumbers},
	{name: "ifsc", gate: gateExact("IFSC"), typ: entity.Financial, label: "account", priority: 72, group: 1,
		re: regexp.MustCompile(`\bIFSC\b[^\n]{0,15}?\b([A-Z]{4}0[A-Z0-9]{6})\b`)},
	{name: "card_cvv", gate: gateLower("cvv|cvc|csc|cvn|security code|cryptogramme|prüfnummer|de seguridad|di sicurezza"), typ: entity.Financial, label: "card", priority: 82, group: 1,
		re: regexp.MustCompile(`(?i)\b(?:cvv2?|cvc2?|csc|cvn|card security code|security code|cryptogramme(?: visuel)?|kartenprüfnummer|prüfnummer|c[óo]digo de seguridad|codice di sicurezza)\b[^\d\n]{0,12}(\d{3,4})\b`),
		validate: func(text string, s, e int) bool {
			ctx := cueContext(text, s, e)
			return hasCueWord(ctx, cardCueWords) || cardRunRE.MatchString(ctx)
		}},
	{name: "card_expiry", gate: gateLower("exp|valid|gültig|vencim|caduc|scad"), typ: entity.Financial, label: "card", priority: 72, group: 1,
		re:       regexp.MustCompile(`(?i)\b(?:exp(?:iry|ires|iration)?(?:[ ]date)?|valid[ ](?:thru|through|until)|good[ ]thru|gültig[ ]bis|vencimiento|vence|caducidad|caduca|scadenza|scade|validade|expira|expire)\b\.?[^\d\n]{0,10}(\d{2}[ ]?[/\-][ ]?(?:20)?\d{2})\b`),
		validate: cardExpiryValid},
}
