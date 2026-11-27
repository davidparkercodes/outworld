package detect

import (
	"regexp"
	"strings"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

type idFormat struct {
	name   string
	re     *regexp.Regexp
	check  func(raw string) bool
	strong bool
	bare   bool
	cues   []string
}

func idf(name, re string, check func(string) bool, strong, bare bool, cues string) idFormat {
	return idFormat{name: name, re: regexp.MustCompile(`^(?:` + re + `)`), check: check, strong: strong, bare: bare, cues: strings.Split(cues, "|")}
}

var digitIDFormats = []idFormat{
	idf("cn_resident_id", `\d{17}[\dXx]`, cnResidentID, true, true, ""),
	idf("fr_nir", `[12378] \d{2} \d{2} (?:\d{2}|2[AB]) \d{3} \d{3} \d{2}|[12378]\d{4}(?:\d{2}|2[AB])\d{8}`, frNIR, true, true, ""),
	idf("br_cnpj", `\d{2}\.\d{3}\.\d{3}/\d{4}-\d{2}`, brCNPJ, true, false, ""),
	idf("br_cnpj", `\d{14}`, brCNPJ, false, true, "cnpj"),
	idf("ch_ahv", `756\.\d{4}\.\d{4}\.\d{2}`, chAHV, true, false, ""),
	idf("ch_ahv", `756\d{10}`, chAHV, false, true, "ahv|avs|ahv-nummer|ahv-nr|sozialversicherungsnummer"),
	idf("kr_rrn", `\d{6}-[1-8]\d{6}`, krRRN, true, false, ""),
	idf("za_id", `\d{13}`, zaID, true, true, ""),
	idf("be_nn", `\d{2}\.\d{2}\.\d{2}-\d{3}\.\d{2}`, beNN, true, false, ""),
	idf("br_cpf", `\d{3}\.\d{3}\.\d{3}-\d{2}`, brCPF, true, false, ""),
	idf("ar_cuil", `(?:20|23|24|27|30|33|34)-\d{8}-\d`, arCUIL, true, false, ""),
	idf("cl_rut", `\d{1,2}\.\d{3}\.\d{3}-[\dkK]`, clRUT, true, false, ""),
	idf("cl_rut", `\d{7,8}-[\dkK]`, clRUT, false, false, "rut|run|rol unico|rol único"),
	idf("nordic_id", `(?:(?:19|20)\d{6}|\d{6})[-+]\d{4}`, nordicID, true, false, ""),
	idf("fi_hetu", `\d{6}[-+A-FU-Y]\d{3}[0-9A-Y]`, fiHETU, true, false, ""),
	idf("ie_pps", `\d{7}[A-W][ABWTXZHI]?`, iePPS, true, false, ""),
	idf("be_nn", `\d{11}`, beNN, true, true, ""),
	idf("no_fnr", `\d{6} \d{5}|\d{11}`, noFnr, true, true, ""),
	idf("de_steuerid", `\d{2} \d{3} \d{3} \d{3}|\d{11}`, deSteuerID, true, true, ""),
	idf("at_svnr", `\d{4} \d{6}|\d{10}`, atSVNR, true, true, ""),
	idf("br_cpf", `\d{11}`, brCPF, false, true, "cpf"),
	idf("ar_cuil", `\d{11}`, arCUIL, false, true, "cuil|cuit"),
	idf("pl_pesel", `\d{11}`, plPESEL, false, true, "pesel"),
	idf("tr_tckn", `\d{11}`, trTCKN, false, true, "kimlik|tckn|tc no|t.c."),
	idf("se_personnummer", `(?:(?:19|20)\d{6}|\d{6})\d{4}`, sePersonnummer, false, true, "personnummer|personnr|pnr"),
	idf("dk_cpr", `\d{10}`, dkCPR, false, true, "cpr|cpr-nummer|cpr-nr|personnummer"),
	idf("in_aadhaar", `\d{4}[ -]?\d{4}[ -]?\d{4}`, inAadhaar, false, true, "aadhaar|aadhar|uidai|uid"),
	idf("jp_my_number", `\d{4}[ -]?\d{4}[ -]?\d{4}`, jpMyNumber, false, true, "my number|mynumber|individual number|マイナンバー|個人番号"),
	idf("uk_nhs", `\d{3}[ -]?\d{3}[ -]?\d{4}`, ukNHS, false, true, "nhs"),
	idf("au_medicare", `\d{4} ?\d{5} ?\d(?:[ /-]?\d)?`, auMedicare, false, true, "medicare"),
	idf("nl_bsn", `\d{9}`, nlBSN, false, true, "bsn|burgerservicenummer|sofinummer|sofi-nummer"),
	idf("pt_nif", `\d{3} ?\d{3} ?\d{3}`, ptNIF, false, true, "nif|contribuinte|nipc"),
	idf("luhn9_id", `\d{3}[- ]?\d{3}[- ]?\d{3}`, luhn9, false, true, "sin|social insurance|nas|assurance sociale|teudat zehut|zehut|israeli id|ת.ז"),
	idf("au_tfn", `\d{3} ?\d{3} ?\d{3}`, auTFN, false, true, "tfn|tax file"),
	idf("nz_ird", `\d{2,3}-?\d{3}-?\d{3}`, nzIRD, false, true, "ird|inland revenue"),
}

var letterIDFormats = []idFormat{
	idf("mx_curp", `[A-Z]{4}\d{6}[HMX][A-Z]{5}[A-Z0-9]\d`, mxCURP, true, false, ""),
	idf("it_codice_fiscale", `[A-Z]{6}\d{2}[A-EHLMPRST]\d{2}[A-Z]\d{3}[A-Z]`, itCF, true, false, ""),
	idf("mx_rfc", `[A-Z]{3,4}\d{6}[A-Z0-9]{3}`, mxRFC, false, false, "rfc|registro federal"),
	idf("in_pan", `[A-Z]{3}[ABCFGHJLPT][A-Z]\d{4}[A-Z]`, nil, true, false, ""),
	idf("sg_nric", `[STFG]\d{7}[A-Z]`, sgNRIC, true, false, ""),
	idf("hk_hkid", `[A-Z]{1,2}\d{6}\([0-9A]\)`, hkHKID, true, false, ""),
	idf("hk_hkid", `[A-Z]{1,2}\d{6}[0-9A]`, hkHKID, false, false, "hkid|hong kong id|hk id"),
	idf("es_cif", `[ABCDEFGHJNPQRSUVW]-?\d{7}-?[0-9A-J]`, esCIF, false, false, "cif|nif"),
	idf("de_personalausweis", `[CFGHJKLMNPRTVWXYZ][CFGHJKLMNPRTVWXYZ0-9]{8}\d`, deAusweis, false, false, "ausweis|personalausweis|ausweisnummer|reisepass|passnummer"),
}

const idNoBefore = "-_/.\\@#+%$&*~^"

func asciiLower(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			b := []byte(s)
			for j := i; j < len(b); j++ {
				if b[j] >= 'A' && b[j] <= 'Z' {
					b[j] += 32
				}
			}
			return string(b)
		}
	}
	return s
}

const scanWindow = 256

func lineStartWithin(text string, s, back int) (int, bool) {
	lo := max(0, s-back)
	if i := strings.LastIndexByte(text[lo:s], '\n'); i >= 0 {
		return lo + i + 1, true
	}
	return lo, lo == 0
}

func lineEndWithin(text string, e, fwd int) (int, bool) {
	hi := min(len(text), e+fwd)
	if i := strings.IndexByte(text[e:hi], '\n'); i >= 0 {
		return e + i, true
	}
	return hi, hi == len(text)
}

func cueContext(text string, s, e int) string {
	lo := max(0, s-240)
	ls := lo
	if i := strings.LastIndexByte(text[lo:s], '\n'); i > 0 {
		ls = lo + strings.LastIndexByte(text[lo:lo+i], '\n') + 1
	}
	le, _ := lineEndWithin(text, e, 120)
	return asciiLower(text[ls:le])
}

func hasCueWord(ctx string, cues []string) bool {
	for _, w := range cues {
		if w == "" {
			continue
		}
		for from := 0; ; {
			i := strings.Index(ctx[from:], w)
			if i < 0 {
				break
			}
			i += from
			from = i + len(w)
			before := i == 0 || !(isAlphaByte(ctx[i-1]) || isDigitByte(ctx[i-1]))
			after := from >= len(ctx) || !(isAlphaByte(ctx[from]) || isDigitByte(ctx[from]))
			if before && after {
				return true
			}
		}
	}
	return false
}

func idTokenStart(text string, i int) bool {
	if i == 0 {
		return true
	}
	p := text[i-1]
	if isDigitByte(p) || isAlphaByte(p) || p >= 0x80 || strings.IndexByte(idNoBefore, p) >= 0 {
		return false
	}
	if i >= 2 && (p == ' ' || p == ',') && isDigitByte(text[i-2]) && isDigitByte(text[i]) {
		return false
	}
	return true
}

func idTokenEnd(text string, e int) bool { return idEnd(text, e, " -./:,") }

func idValueEnd(text string, s, e int) bool {
	v := text[s:e]
	if strings.IndexByte(v, ' ') < 0 && strings.ContainsAny(v, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return idEnd(text, e, "-./:")
	}
	return idTokenEnd(text, e)
}

func idEnd(text string, e int, joiners string) bool {
	if e >= len(text) {
		return true
	}
	c := text[e]
	if isDigitByte(c) || isAlphaByte(c) || c == '_' || c >= 0x80 && !strings.HasPrefix(text[e:], "’") && !strings.HasPrefix(text[e:], "”") && !strings.HasPrefix(text[e:], " ") {
		return false
	}
	if e+1 < len(text) && strings.IndexByte(joiners, c) >= 0 && isDigitByte(text[e+1]) {
		return false
	}
	if e+1 < len(text) && strings.IndexByte("-/@\\=+", c) >= 0 && isAlphaByte(text[e+1]) {
		return false
	}
	return c != '%'
}

func digitsAhead(text string, i, span int) int {
	n := 0
	for j := i; j < len(text) && j < i+span; j++ {
		c := text[j]
		if isDigitByte(c) {
			n++
		} else if c == '\n' || c == ' ' && j+1 < len(text) && text[j+1] == ' ' {
			break
		}
	}
	return n
}

func letterIDCandidate(text string, i int) bool {
	j := i
	for j < len(text) && j < i+7 && text[j] >= 'A' && text[j] <= 'Z' {
		j++
	}
	if j < len(text) && text[j] == '-' {
		j++
	}
	return j > i && j < len(text) && isDigitByte(text[j])
}

func scanNationalIDs(text, _ string) []entity.Span {
	var out []entity.Span
	n := len(text)
	for i := 0; i < n; i++ {
		c := text[i]
		var formats []idFormat
		switch {
		case isDigitByte(c):
			if !idTokenStart(text, i) || digitsAhead(text, i, 26) < 7 {
				continue
			}
			formats = digitIDFormats
		case c >= 'A' && c <= 'Z':
			if !idTokenStart(text, i) || !letterIDCandidate(text, i) {
				continue
			}
			formats = letterIDFormats
		default:
			continue
		}
		window := text[i:min(n, i+40)]
		ctx := ""
		for k := range formats {
			f := &formats[k]
			loc := f.re.FindStringIndex(window)
			if loc == nil {
				continue
			}
			e := i + loc[1]
			if !idValueEnd(text, i, e) || f.check != nil && !f.check(text[i:e]) {
				continue
			}
			prio := 72
			if !f.strong {
				if ctx == "" {
					ctx = cueContext(text, i, e)
				}
				if !hasCueWord(ctx, f.cues) {
					continue
				}
				prio = 74
			} else if f.bare {
				prio = 52
			}
			out = append(out, entity.Span{Start: i, End: e, Priority: prio, Label: "id"})
			i = e - 1
			break
		}
	}
	return out
}

const intlIDWords = `passport|passeport|reisepass|reisepassnummer|passnummer|pass-nummer|pasaporte|passaporto|passaporte|paspoort|paspoortnummer|paszport|pasaport|` +
	`driver's license|driver's licence|drivers license|drivers licence|driver license|driver licence|driving license|driving licence|permis de conduire|führerschein|führerscheinnummer|fuhrerschein|fuehrerschein|carnet de conducir|licencia de conducir|permiso de conducir|patente di guida|rijbewijs|rijbewijsnummer|carta de condução|carteira de motorista|carteira de habilitação|prawo jazdy|körkort|kørekort|førerkort|ajokortti|ehliyet|` +
	`personalausweis|personalausweisnummer|ausweisnummer|ausweis-nr|ausweisnr|carte d'identité|carte d’identité|carte nationale d'identité|carte nationale d’identité|carta d'identità|carta d’identità|carta di identità|documento de identidad|documento de identidade|documento d'identità|documento nacional de identidad|cédula|cedula|cédula de ciudadanía|cedula de ciudadania|cédula de identidad|cedula de identidad|identiteitskaart|dowód osobisty|dowod osobisty|national id|national identity|national identification|identity card|id card|resident id|resident permit|residence permit|titre de séjour|aufenthaltstitel|permesso di soggiorno|` +
	`steuer-id|steuerid|steueridentifikationsnummer|steuer-identifikationsnummer|steuernummer|steuer-nr|steuernr|idnr|sozialversicherungsnummer|versicherungsnummer|sv-nummer|svnummer|rentenversicherungsnummer|krankenversicherungsnummer|numéro fiscal|numéro de sécurité sociale|n° de sécurité sociale|sécurité sociale|numero de securite sociale|codice fiscale|cod. fisc|partita iva|burgerservicenummer|rijksregisternummer|numéro national|numéro de registre national|número de identificación fiscal|numero de identificacion fiscal|número de la seguridad social|seguridad social|número de contribuinte|numero de contribuinte|personnummer|personnr|cpr-nummer|cpr-nr|cprnr|fødselsnummer|fodselsnummer|henkilötunnus|pps number|ahv-nummer|ahv-nr|numéro avs|social insurance|social security|tax file number|tax id|tax identification|taxpayer id|taxpayer identification|tax number|tax reference|aadhaar|aadhar|my number|resident registration|tc kimlik|kimlik no|teudat zehut|national insurance|nhs number|medicare|health card|voter id|alien registration|a-number`

const intlIDAcronyms = `DL|EIN|TIN|ITIN|SSN|DNI|NIE|NIF|CIF|NINO|SIN|MRN|CPF|CNPJ|CNH|NHS|BSN|NIR|PESEL|NIP|REGON|CPR|HETU|PPS|PPSN|AHV|AVS|SVNR|CURP|RFC|CUIL|CUIT|RUT|TFN|IRD|PAN|RRN|NRIC|HKID|CNI|TCKN|UTR|NSS|INSEE|JMBG|EGN|CNP|OIB`

type labelSet struct {
	words    map[[2]byte][]string
	acronyms map[[2]byte][]string
}

func labelIndex(list string) map[[2]byte][]string {
	if list == "" {
		return nil
	}
	m := map[[2]byte][]string{}
	for _, w := range strings.Split(list, "|") {
		k := [2]byte{w[0], w[1]}
		m[k] = append(m[k], w)
	}
	return m
}

func newLabelSet(words, acronyms string) labelSet {
	return labelSet{words: labelIndex(words), acronyms: labelIndex(acronyms)}
}

func sameLengthLower(text, lower string) string {
	if len(lower) == len(text) {
		return lower
	}
	return asciiLower(text)
}

func wordStartAt(s string, i int) bool {
	return i == 0 || !(isAlnumByte(s[i-1]) || s[i-1] == '_' || s[i-1] >= 0x80)
}

func wordEndAt(s string, e int) bool {
	return e >= len(s) || !(isAlnumByte(s[e]) || s[e] == '_' || s[e] >= 0x80 && !strings.HasPrefix(s[e:], "’"))
}

func (l labelSet) each(text, low string, fn func(s, e int)) {
	for i := 0; i+1 < len(text); i++ {
		if !wordStartAt(text, i) {
			continue
		}
		for _, w := range l.words[[2]byte{low[i], low[i+1]}] {
			if strings.HasPrefix(low[i:], w) {
				fn(i, i+len(w))
			}
		}
		if l.acronyms == nil {
			continue
		}
		for _, w := range l.acronyms[[2]byte{text[i], text[i+1]}] {
			if strings.HasPrefix(text[i:], w) {
				fn(i, i+len(w))
			}
		}
	}
}

var intlIDLabels = newLabelSet(intlIDWords, intlIDAcronyms)

var intlIDValueRE = regexp.MustCompile(`^(?:(?:[A-Z]{1,2} )?\d{1,6}(?: \d{2,6}| [A-Z]\b){1,7}(?: [A-Z0-9]\b)?|[A-Z]{1,3} \d{6,12}\b|[A-Z0-9]+(?:[.\-/][A-Z0-9]+)*(?:\([0-9A]\))?)`)
var idSlashDateRE = regexp.MustCompile(`^(?:\d{1,2}[/.\-]\d{1,2}[/.\-]\d{2,4}|\d{4}[/.\-]\d{1,2}[/.\-]\d{1,2}|\d{1,3}(?:\.\d{1,2})+|\d{1,2}[/.\-]\d{4}|(?:19|20)\d{2}[/\-](?:19|20)?\d{2})$`)

func labelledIDValueOK(text string, s, e, gap int) bool {
	v := text[s:e]
	c := compactAlnum(v)
	d := digitsOnly(v)
	if len(c) < 6 || len(c) > 32 || len(d) < 4 {
		return false
	}
	if gap > 12 && (len(d) < 5 || len(c) < 7) {
		return false
	}
	if strings.Contains(v, " ") && len(d) < 7 && !(len(d) == 6 && prefixedIDRE.MatchString(v)) {
		return false
	}
	if gap > 6 && allDigits(v) && (len(v) < 8 || strings.HasSuffix(v, "000")) {
		return false
	}
	if idDateRE.MatchString(v) || idPhoneRE.MatchString(v) || idSlashDateRE.MatchString(v) || sameDigits(d) {
		return false
	}
	if allDigits(v) && len(v) == 8 && (strings.HasPrefix(v, "19") || strings.HasPrefix(v, "20")) && monthDayOK(num2(v, 4), num2(v, 6)) {
		return false
	}
	return idValueEnd(text, s, e)
}

var prefixedIDRE = regexp.MustCompile(`^[A-Z]{1,3} \d+$`)
var idQualifiers = wordSet(`NO NR NUM NUMBER ID N`)

func upperRunEnd(text string, j int) int {
	for j < len(text) && text[j] >= 'A' && text[j] <= 'Z' {
		j++
	}
	return j
}

func idValueSkip(text string, j int) int {
	for j < len(text) && (isAlnumByte(text[j]) || strings.IndexByte("-./", text[j]) >= 0 && j+1 < len(text) && isAlnumByte(text[j+1])) {
		j++
	}
	return j
}

func scanLabelledIDs(text, lower string) []entity.Span {
	var out []entity.Span
	intlIDLabels.each(text, sameLengthLower(text, lower), func(ls, p int) {
		if !wordEndAt(text, p) {
			return
		}
		lim := min(len(text), p+48)
		for j := p; j < lim; {
			c := text[j]
			if c == '\n' || c == '[' && strings.HasPrefix(text[j:], "[redacted") {
				break
			}
			if (c == '.' || c == '!' || c == '?' || c == ';') && j+1 < len(text) && text[j+1] == ' ' && j-p > 4 {
				break
			}
			if !isDigitByte(c) && !(c >= 'A' && c <= 'Z') {
				j++
				continue
			}
			prev := text[j-1]
			ok := !(isDigitByte(prev) || isAlphaByte(prev) || prev == '_' || prev >= 0x80 || strings.IndexByte("$+", prev) >= 0)
			if u := upperRunEnd(text, j); ok && idQualifiers[text[j:u]] && (u >= len(text) || !isAlnumByte(text[u])) {
				j = u
				continue
			}
			if ok {
				if loc := intlIDValueRE.FindStringIndex(text[j:min(len(text), j+48)]); loc != nil {
					e := j + loc[1]
					if labelledIDValueOK(text, j, e, j-p) {
						if !(text[ls:p] == "PAN" && validCard(text[j:e])) {
							out = append(out, entity.Span{Start: j, End: e})
						}
						return
					}
					j = idValueSkip(text, e)
					continue
				}
			}
			for j < lim && (isDigitByte(text[j]) || isAlphaByte(text[j])) {
				j++
			}
		}
	})
	return out
}

var govIDPatterns = []pattern{
	{name: "national_id", typ: entity.GovID, label: "id", priority: 72, scan: scanNationalIDs},
	{name: "labelled_id_intl", typ: entity.GovID, label: "id", priority: 76, scan: scanLabelledIDs},
}
