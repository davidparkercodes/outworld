package dateshift

import (
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type lang struct {
	months [12]string
	abbr   [12]string
	days   [7]string
}

const (
	langDE = iota
	langES
	langFR
	langPT
	langIT
	langNL
)

var langs = []lang{
	langDE: {
		months: [12]string{"januar", "februar", "märz", "april", "mai", "juni", "juli", "august", "september", "oktober", "november", "dezember"},
		abbr:   [12]string{"jan", "feb", "mär", "apr", "mai", "jun", "jul", "aug", "sep", "okt", "nov", "dez"},
		days:   [7]string{"sonntag", "montag", "dienstag", "mittwoch", "donnerstag", "freitag", "samstag"},
	},
	langES: {
		months: [12]string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"},
		abbr:   [12]string{"ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sep", "oct", "nov", "dic"},
		days:   [7]string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"},
	},
	langFR: {
		months: [12]string{"janvier", "février", "mars", "avril", "mai", "juin", "juillet", "août", "septembre", "octobre", "novembre", "décembre"},
		abbr:   [12]string{"janv", "févr", "mars", "avr", "mai", "juin", "juil", "août", "sept", "oct", "nov", "déc"},
		days:   [7]string{"dimanche", "lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi"},
	},
	langPT: {
		months: [12]string{"janeiro", "fevereiro", "março", "abril", "maio", "junho", "julho", "agosto", "setembro", "outubro", "novembro", "dezembro"},
		abbr:   [12]string{"jan", "fev", "mar", "abr", "mai", "jun", "jul", "ago", "set", "out", "nov", "dez"},
		days:   [7]string{"domingo", "segunda-feira", "terça-feira", "quarta-feira", "quinta-feira", "sexta-feira", "sábado"},
	},
	langIT: {
		months: [12]string{"gennaio", "febbraio", "marzo", "aprile", "maggio", "giugno", "luglio", "agosto", "settembre", "ottobre", "novembre", "dicembre"},
		abbr:   [12]string{"gen", "feb", "mar", "apr", "mag", "giu", "lug", "ago", "set", "ott", "nov", "dic"},
		days:   [7]string{"domenica", "lunedì", "martedì", "mercoledì", "giovedì", "venerdì", "sabato"},
	},
	langNL: {
		months: [12]string{"januari", "februari", "maart", "april", "mei", "juni", "juli", "augustus", "september", "oktober", "november", "december"},
		abbr:   [12]string{"jan", "feb", "mrt", "apr", "mei", "jun", "jul", "aug", "sep", "okt", "nov", "dec"},
		days:   [7]string{"zondag", "maandag", "dinsdag", "woensdag", "donderdag", "vrijdag", "zaterdag"},
	},
}

var (
	orderDot       = []int{langDE, langNL, langIT, langFR, langES, langPT}
	orderConnector = []int{langES, langPT, langIT, langFR, langNL, langDE}
	orderLower     = []int{langNL, langFR, langIT, langES, langPT, langDE}
	orderTitle     = []int{langDE, langNL, langFR, langIT, langES, langPT}
)

type monthEntry struct {
	lang, month int
	abbr        bool
}

type dayEntry struct {
	lang, day int
}

var (
	foreignMonths = map[string][]monthEntry{}
	foreignDays   = map[string][]dayEntry{}
	englishMonths = map[string]bool{}
	vocab         = map[string]bool{}
	foreignRE     *regexp.Regexp
	foreignDayRE  *regexp.Regexp
)

var accentFold = strings.NewReplacer("á", "a", "à", "a", "â", "a", "ä", "a", "ã", "a", "é", "e", "è", "e", "ê", "e", "í", "i", "ì", "i", "î", "i", "ó", "o", "ò", "o", "ô", "o", "ö", "o", "õ", "o", "ú", "u", "ù", "u", "û", "u", "ü", "u", "ç", "c")

func fold(s string) string { return accentFold.Replace(strings.ToLower(s)) }

func alternation(set map[string]bool) string {
	words := make([]string, 0, len(set))
	for w := range set {
		words = append(words, regexp.QuoteMeta(w))
	}
	sort.Slice(words, func(i, j int) bool {
		if len(words[i]) != len(words[j]) {
			return len(words[i]) > len(words[j])
		}
		return words[i] < words[j]
	})
	return strings.Join(words, "|")
}

func init() {
	monthWords, dayWords := map[string]bool{}, map[string]bool{}
	for li, l := range langs {
		for i := range l.months {
			full, ab := l.months[i], l.abbr[i]
			foreignMonths[fold(full)] = append(foreignMonths[fold(full)], monthEntry{li, i + 1, false})
			monthWords[full], monthWords[fold(full)] = true, true
			if ab != full {
				foreignMonths[fold(ab)] = append(foreignMonths[fold(ab)], monthEntry{li, i + 1, true})
				monthWords[ab], monthWords[fold(ab)] = true, true
			}
		}
		for i, d := range l.days {
			foreignDays[fold(d)] = append(foreignDays[fold(d)], dayEntry{li, i})
			dayWords[d], dayWords[fold(d)] = true, true
		}
	}
	for _, m := range fullMonths {
		l := strings.ToLower(m)
		englishMonths[l], englishMonths[l[:3]] = true, true
	}
	englishMonths["sept"] = true
	for w := range monthWords {
		vocab[fold(w)] = true
	}
	for w := range dayWords {
		for _, part := range strings.Split(fold(w), "-") {
			vocab[part] = true
		}
	}
	for w := range englishMonths {
		vocab[w] = true
	}
	for d := time.Sunday; d <= time.Saturday; d++ {
		l := strings.ToLower(d.String())
		vocab[l], vocab[l[:3]] = true, true
	}
	for _, w := range []string{"tues", "thur", "thurs", "of", "the", "day", "de", "del", "do", "di", "twenty", "thirty"} {
		vocab[w] = true
	}
	for _, w := range ordinalWords {
		for _, part := range strings.Split(w, "-") {
			vocab[part] = true
		}
	}
	conn := `(?:de\s+|del\s+|do\s+|di\s+)?`
	foreignRE = regexp.MustCompile(`(?i)^(\d{1,2})(\.|º|°|er)?(\s+` + conn + `)(` + alternation(monthWords) + `)(\.?)(?:(,?\s+` + conn + `)(\d{4})\b)?`)
	foreignDayRE = regexp.MustCompile(`(?i)(?:^|[^\pL\pN])(` + alternation(dayWords) + `)(,?\s+)$`)
}

func Vocab(word string, partial bool) bool {
	w := fold(word)
	if w == "" {
		return false
	}
	if !partial {
		return vocab[w]
	}
	for v := range vocab {
		if strings.HasPrefix(v, w) {
			return true
		}
	}
	return false
}

const (
	caseLower = iota
	caseTitle
	caseUpper
)

func caseOf(s string) int {
	if utf8.RuneCountInString(s) > 1 && s == strings.ToUpper(s) && s != strings.ToLower(s) {
		return caseUpper
	}
	if r, _ := utf8.DecodeRuneInString(s); unicode.IsUpper(r) {
		return caseTitle
	}
	return caseLower
}

func applyCase(s string, c int) string {
	switch c {
	case caseUpper:
		return strings.ToUpper(s)
	case caseTitle:
		r, n := utf8.DecodeRuneInString(s)
		return string(unicode.ToUpper(r)) + s[n:]
	}
	return s
}

func hasConnector(s string) bool {
	return strings.TrimSpace(s) != ""
}

func pickLang(word, suffix, sep string, votes map[int]int) (monthEntry, bool) {
	cands := foreignMonths[fold(word)]
	if len(cands) == 0 {
		return monthEntry{}, false
	}
	var order []int
	lower := caseOf(word) == caseLower
	english := englishMonths[strings.ToLower(word)]
	switch {
	case suffix == ".":
		order = orderDot
	case hasConnector(sep):
		order = orderConnector
	case lower:
		order = orderLower
	default:
		order = orderTitle
	}
	if len(votes) > 0 {
		order = append([]int{}, order...)
		sort.SliceStable(order, func(i, j int) bool { return votes[order[i]] > votes[order[j]] })
	}
	for _, li := range order {
		for _, c := range cands {
			if c.lang != li {
				continue
			}
			if english && suffix != "." && !hasConnector(sep) {
				if !lower || c.abbr || c.lang == langDE {
					return monthEntry{}, false
				}
			}
			return c, true
		}
	}
	return monthEntry{}, false
}

func findForeign(text string) []Match {
	var out []Match
	var all [][]int
	for i := 0; i < len(text); i++ {
		if !isDigit(text[i]) {
			continue
		}
		start := i
		for i < len(text) && isDigit(text[i]) {
			i++
		}
		if i-start > 2 || letterBefore(text, start) {
			continue
		}
		end := start + 64
		if end > len(text) {
			end = len(text)
		}
		g := foreignRE.FindStringSubmatchIndex(text[start:end])
		if g == nil {
			continue
		}
		if g[1] == end-start && end < len(text) {
			g = foreignRE.FindStringSubmatchIndex(text[start:])
		}
		for k := range g {
			if g[k] >= 0 {
				g[k] += start
			}
		}
		all = append(all, g)
		i = g[1] - 1
	}
	votes := map[int]int{}
	for _, g := range all {
		if cands := foreignMonths[fold(text[g[8]:g[9]])]; len(cands) == 1 && !englishMonths[strings.ToLower(text[g[8]:g[9]])] {
			votes[cands[0].lang]++
		}
	}
	for _, g := range all {
		sub := groups(text, g)
		day, suf, sp, mon, dot, ysep, year := sub[1], sub[2], sub[3], sub[4], sub[5], sub[6], sub[7]
		if r, _ := utf8.DecodeRuneInString(text[g[9]:]); g[9] < len(text) && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			continue
		}
		e, ok := pickLang(mon, suf, sp, votes)
		if !ok {
			continue
		}
		if e.abbr && year == "" && !hasConnector(sp) && suf != "." {
			continue
		}
		start := g[0]
		wd, wsep := "", ""
		from := start - 24
		if from < 0 {
			from = 0
		}
		for from > 0 && !utf8.RuneStart(text[from]) {
			from--
		}
		if w := foreignDayRE.FindStringSubmatchIndex(text[from:start]); w != nil {
			wd, wsep = text[from+w[2]:from+w[3]], text[from+w[4]:from+w[5]]
			start = from + w[2]
		}
		d, _ := atoi(day)
		y := refYear
		hasYear := year != ""
		if hasYear {
			y, _ = atoi(year)
		} else if e.month == 2 && d == 29 {
			continue
		}
		t, ok := valid(y, e.month, d)
		if !ok {
			continue
		}
		l := langs[e.lang]
		monCase, wdCase, dw := caseOf(mon), caseOf(wd), width(day)
		abbr := e.abbr
		out = append(out, Match{Start: start, End: g[1], Text: text[start:g[1]], date: t, hasYear: hasYear, render: func(t time.Time) string {
			var b strings.Builder
			if wd != "" {
				b.WriteString(applyCase(l.days[t.Weekday()], wdCase))
				b.WriteString(wsep)
			}
			b.WriteString(pad(t.Day(), dw))
			if suf != "" && (!strings.EqualFold(suf, "er") || t.Day() == 1) {
				b.WriteString(suf)
			}
			b.WriteString(sp)
			name := l.months[t.Month()-1]
			if abbr {
				name = l.abbr[t.Month()-1]
			}
			b.WriteString(applyCase(name, monCase))
			b.WriteString(dot)
			if year != "" {
				b.WriteString(ysep + pad(t.Year(), 4))
			}
			return b.String()
		}})
	}
	return out
}
