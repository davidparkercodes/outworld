package dateshift

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

func atoi(s string) (int, error) { return strconv.Atoi(s) }

var ordinalWords = func() []string {
	base := []string{"first", "second", "third", "fourth", "fifth", "sixth", "seventh", "eighth", "ninth", "tenth", "eleventh", "twelfth", "thirteenth", "fourteenth", "fifteenth", "sixteenth", "seventeenth", "eighteenth", "nineteenth", "twentieth"}
	out := append([]string{}, base...)
	for _, w := range base[:9] {
		out = append(out, "twenty-"+w)
	}
	return append(out, "thirtieth", "thirty-first")
}()

var ordinalValue = func() map[string]int {
	m := map[string]int{}
	for i, w := range ordinalWords {
		m[w] = i + 1
	}
	return m
}()

const (
	unitOrd  = `first|second|third|fourth|fifth|sixth|seventh|eighth|ninth`
	ordAlt   = `(?:twenty|thirty)[- ](?:` + unitOrd + `)|` + unitOrd + `|tenth|eleventh|twelfth|thirteenth|fourteenth|fifteenth|sixteenth|seventeenth|eighteenth|nineteenth|twentieth|thirtieth`
	abbrAlt  = `Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec`
	slashTok = `\d{1,2}/\d{1,2}`
)

var (
	spokenMonthDayRE = regexp.MustCompile(`(?i)\b(?:(` + weekdayAlt + `)\.?,?\s+)?(` + monthAlt + `)(\s+(?:the\s+)?)(` + ordAlt + `)\b(?:(,?\s+)(\d{4})\b)?`)
	spokenDayMonthRE = regexp.MustCompile(`(?i)\b(?:(` + weekdayAlt + `)\.?,?\s+(?:the\s+)?)?(` + ordAlt + `)(\s+(?:day\s+)?of\s+)(` + monthAlt + `)\b(?:(,?\s+)(\d{4})\b)?`)
	dayMonYearRE     = regexp.MustCompile(`(?i)\b(\d{1,2})([-/])(` + monthAlt + `)(?:([-/])(\d{4}|\d{2}))?\b`)
	airlineRE        = regexp.MustCompile(`\b(\d{2})(` + abbrAlt + `|` + strings.ToUpper(abbrAlt) + `)(\d{4}|\d{2})?\b`)
	compactRE        = regexp.MustCompile(`\d{8}`)
	icalTimeRE       = regexp.MustCompile(`^T\d{4,6}Z?(?:[^0-9A-Za-z]|$)`)
	slashRE          = regexp.MustCompile(slashTok)
	slashRangeAfter  = regexp.MustCompile(`^\s*[-\x{2013}]\s*` + slashTok + `(?:[^0-9/]|$)`)
	slashRangeBefore = regexp.MustCompile(slashTok + `\s*[-\x{2013}]\s*$`)
	timeAfterRE      = regexp.MustCompile(`(?i)^\s*(?:(?:at|@)\s*\d|,?\s*\(?(?:` + weekdayAlt + `)\b|,?\s+\d{1,2}(?::\d{2})?\s*[ap]\.?m\b)`)
)

var cueWords = wordSet(`on by due until till til since from through thru before after starting starts start begins begin ending ends dated date dates appt appointment appointments scheduled rescheduled reschedule deadline eta effective expires expiring exp arriving arrives arrive departing departs depart returning returns return born dob week weekend weekdays between mon monday tue tues tuesday wed wednesday thu thur thurs thursday fri friday sat saturday sun sunday booked moved postponed delivery delivered ships shipping shipped visit meeting`)

var softCues = wordSet(`by from on`)

var quantityWords = wordSet(`increase increased increases decrease decreased decreases reduce reduced reduces divide divided multiply multiplied cut up down off grew fell rose dropped scale scaled ratio odds score scored won lost lead led`)

var labelWords = wordSet(`version ver v chapter ch page pg p step part ratio score odds room rm apt suite unit lot size model no number gauge about roughly approximately nearly than a an every per x release build rev firmware`)

var idWords = wordSet(`isbn id no number order invoice ticket ref acct account sku serial phone tel fax zip pin code tracking case po item part model build`)

var unitWords = wordSet(`cup cups tsp tbsp teaspoon teaspoons tablespoon tablespoons inch inches oz ounce ounces lb lbs pound pounds mile miles of full empty star stars point points pts odds ratio hp time times split scale way done complete majority vote mm cm ft yd th ths pint pints gallon gallons liter liters litre litres stick sticks tank teaspoonful`)

func wordSet(s string) map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(s) {
		m[w] = true
	}
	return m
}

func wordsBefore(text string, start, n int) []string {
	from := start - 48
	if from < 0 {
		from = 0
	}
	seg := text[from:start]
	if i := strings.LastIndexAny(seg, "\n\r"); i >= 0 {
		seg = seg[i+1:]
	}
	if from > 0 {
		if i := strings.IndexFunc(seg, func(r rune) bool { return !unicode.IsLetter(r) }); i >= 0 {
			seg = seg[i:]
		} else {
			seg = ""
		}
	}
	words := strings.FieldsFunc(strings.ToLower(seg), func(r rune) bool { return !unicode.IsLetter(r) })
	if len(words) > n {
		words = words[len(words)-n:]
	}
	return words
}

func wordAfter(text string, end int) string {
	rest := strings.TrimLeft(text[end:], " \t")
	i := strings.IndexFunc(rest, func(r rune) bool { return !unicode.IsLetter(r) })
	if i < 0 {
		i = len(rest)
	}
	return strings.ToLower(rest[:i])
}

func lastWord(text string, start int) string {
	seg := strings.TrimRight(text[:start], " \t:#")
	if len(seg) != start && strings.ContainsAny(text[len(seg):start], "#") {
		return "no"
	}
	i := strings.LastIndexFunc(seg, func(r rune) bool { return !unicode.IsLetter(r) })
	return strings.ToLower(seg[i+1:])
}

func hasCue(text string, start, end int) bool {
	words := wordsBefore(text, start, 3)
	quantity := false
	for _, w := range words {
		if quantityWords[w] {
			quantity = true
		}
	}
	for i, w := range words {
		if !cueWords[w] {
			continue
		}
		if softCues[w] && quantity {
			continue
		}
		if !softCues[w] || i >= len(words)-2 {
			return true
		}
	}
	if n := len(words); n >= 2 && words[n-2] == "as" && words[n-1] == "of" {
		return true
	}
	return timeAfterRE.MatchString(text[end:])
}

func byteAt(text string, i int) byte {
	if i < 0 || i >= len(text) {
		return 0
	}
	return text[i]
}

func isDigit(c byte) bool  { return c >= '0' && c <= '9' }
func isLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

func letterBefore(text string, i int) bool {
	r, _ := utf8.DecodeLastRuneInString(text[:i])
	return i > 0 && (unicode.IsLetter(r) || unicode.IsDigit(r))
}

func letterAfter(text string, i int) bool {
	r, _ := utf8.DecodeRuneInString(text[i:])
	return i < len(text) && (unicode.IsLetter(r) || unicode.IsDigit(r))
}

func ambiguousMonthWord(mon string) bool {
	l := strings.ToLower(mon)
	return l == "may" || (mon == l && (l == "march" || l == "mar"))
}

func plainFollow(text string, end int) bool {
	return wordAfter(text, end) == ""
}

func renderOrdinal(orig string, day int) string {
	w := ordinalWords[day-1]
	if strings.Contains(orig, " ") {
		w = strings.Replace(w, "-", " ", 1)
	}
	return applyCase(w, caseOf(orig))
}

var ordinalStems = []string{"first", "second", "third", "fourth", "fifth", "sixth", "seventh", "eighth", "ninth", "tenth", "eleventh", "twelfth", "teenth", "tieth"}

func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

func isSpace(c byte) bool { return c == ' ' || c == '\n' || c == '\r' || c == '\t' }

func ordinalWindows(text string) [][2]int {
	lower := asciiLower(text)
	var hits []int
	for _, stem := range ordinalStems {
		for from := 0; ; {
			i := strings.Index(lower[from:], stem)
			if i < 0 {
				break
			}
			hits = append(hits, from+i)
			from += i + len(stem)
		}
	}
	sort.Ints(hits)
	var wins [][2]int
	for _, h := range hits {
		s, e := h-48, h+48
		if s < 0 {
			s = 0
		}
		if e > len(text) {
			e = len(text)
		}
		for s > 0 && !isSpace(text[s-1]) {
			s--
		}
		for e < len(text) && !isSpace(text[e]) {
			e++
		}
		if n := len(wins); n > 0 && s <= wins[n-1][1] {
			if e > wins[n-1][1] {
				wins[n-1][1] = e
			}
			continue
		}
		wins = append(wins, [2]int{s, e})
	}
	return wins
}

func findInWindows(re *regexp.Regexp, text string, wins [][2]int) [][]int {
	var out [][]int
	for _, w := range wins {
		for _, g := range re.FindAllStringSubmatchIndex(text[w[0]:w[1]], -1) {
			for k := range g {
				if g[k] >= 0 {
					g[k] += w[0]
				}
			}
			out = append(out, g)
		}
	}
	return out
}

func findSpoken(text string) []Match {
	var out []Match
	emit := func(g []int, wd, mon, ord, year string, needCtx bool, render func(t time.Time) string) {
		d, ok := ordinalValue[strings.ReplaceAll(strings.ToLower(ord), " ", "-")]
		if !ok {
			return
		}
		m, ok := build(text, g, wd, mon, strconv.Itoa(d), year)
		if !ok {
			return
		}
		m.render = render
		m.needCtx = needCtx
		out = append(out, m)
	}
	wins := ordinalWindows(text)
	if len(wins) == 0 {
		return nil
	}
	for _, g := range findInWindows(spokenMonthDayRE, text, wins) {
		sub := groups(text, g)
		wd, mon, sp, ord, ysep, year := sub[1], sub[2], sub[3], sub[4], sub[5], sub[6]
		needCtx := ambiguousMonthWord(mon) && year == "" && wd == "" && !hasCue(text, g[0], g[1])
		emit(g, wd, mon, ord, year, needCtx, func(t time.Time) string {
			var b strings.Builder
			if wd != "" {
				b.WriteString(renderWeekday(wd, t.Weekday()))
				b.WriteString(text[g[3]:g[4]])
			}
			b.WriteString(renderMonth(mon, t.Month()))
			b.WriteString(sp)
			b.WriteString(renderOrdinal(ord, t.Day()))
			if year != "" {
				b.WriteString(ysep + pad(t.Year(), 4))
			}
			return b.String()
		})
	}
	for _, g := range findInWindows(spokenDayMonthRE, text, wins) {
		sub := groups(text, g)
		wd, ord, sp, mon, ysep, year := sub[1], sub[2], sub[3], sub[4], sub[5], sub[6]
		emit(g, wd, mon, ord, year, false, func(t time.Time) string {
			var b strings.Builder
			if wd != "" {
				b.WriteString(renderWeekday(wd, t.Weekday()))
				b.WriteString(text[g[3]:g[4]])
			}
			b.WriteString(renderOrdinal(ord, t.Day()))
			b.WriteString(sp)
			b.WriteString(renderMonth(mon, t.Month()))
			if year != "" {
				b.WriteString(ysep + pad(t.Year(), 4))
			}
			return b.String()
		})
	}
	return out
}

func expandYear(s string) int {
	y, _ := strconv.Atoi(s)
	if len(s) == 2 {
		if y < 70 {
			return y + 2000
		}
		return y + 1900
	}
	return y
}

func renderYear(orig string, t time.Time) string {
	if len(orig) == 2 {
		return pad(t.Year()%100, 2)
	}
	return pad(t.Year(), 4)
}

func renderShortMonth(orig string, m time.Month) string {
	if len(orig) == 3 {
		return matchUpper(orig, fullMonths[m-1][:3])
	}
	return renderMonth(orig, m)
}

func findDayMonYear(text string) []Match {
	var out []Match
	for _, g := range dayMonYearRE.FindAllStringSubmatchIndex(text, -1) {
		sub := groups(text, g)
		day, sep, mon, sep2, year := sub[1], sub[2], sub[3], sub[4], sub[5]
		if year != "" && sep != sep2 {
			continue
		}
		if year == "" && caseOf(mon) == caseLower {
			continue
		}
		if g[0] > 0 && strings.IndexByte("./-", text[g[0]-1]) >= 0 {
			continue
		}
		if c := byteAt(text, g[1]); (c == '-' || c == '/') && (isDigit(byteAt(text, g[1]+1)) || isLetter(byteAt(text, g[1]+1))) {
			continue
		}
		ys := ""
		if year != "" {
			ys = strconv.Itoa(expandYear(year))
		}
		m, ok := build(text, g, "", mon, day, ys)
		if !ok {
			continue
		}
		dw := width(day)
		m.render = func(t time.Time) string {
			s := pad(t.Day(), dw) + sep + renderShortMonth(mon, t.Month())
			if year != "" {
				s += sep2 + renderYear(year, t)
			}
			return s
		}
		out = append(out, m)
	}
	return out
}

func findAirline(text string) []Match {
	var out []Match
	for _, g := range airlineRE.FindAllStringSubmatchIndex(text, -1) {
		sub := groups(text, g)
		day, mon, year := sub[1], sub[2], sub[3]
		if year == "" && caseOf(mon) != caseUpper {
			continue
		}
		if g[0] > 0 && strings.IndexByte("./-#", text[g[0]-1]) >= 0 {
			continue
		}
		ys := ""
		if year != "" {
			ys = strconv.Itoa(expandYear(year))
		}
		m, ok := build(text, g, "", mon, day, ys)
		if !ok {
			continue
		}
		m.render = func(t time.Time) string {
			s := pad(t.Day(), 2) + renderShortMonth(mon, t.Month())
			if year != "" {
				s += renderYear(year, t)
			}
			return s
		}
		out = append(out, m)
	}
	return out
}

func findCompact(text string) []Match {
	var out []Match
	for _, g := range compactRE.FindAllStringIndex(text, -1) {
		s, e := g[0], g[1]
		if letterBefore(text, s) || isDigit(byteAt(text, e)) {
			continue
		}
		if p := byteAt(text, s-1); p == '#' || p == '+' || p == '.' || p == '$' || ((p == '-' || p == '/') && isDigit(byteAt(text, s-2))) {
			continue
		}
		n := byteAt(text, e)
		if (n == '.' || n == '-' || n == '/') && isDigit(byteAt(text, e+1)) {
			continue
		}
		if letterAfter(text, e) && !icalTimeRE.MatchString(text[e:]) {
			continue
		}
		if idWords[lastWord(text, s)] {
			continue
		}
		y, _ := strconv.Atoi(text[s : s+4])
		mo, _ := strconv.Atoi(text[s+4 : s+6])
		d, _ := strconv.Atoi(text[s+6 : e])
		t, ok := valid(y, mo, d)
		if !ok || y < 1900 || y > 2099 {
			continue
		}
		out = append(out, Match{Start: s, End: e, Text: text[s:e], date: t, hasYear: true, render: func(t time.Time) string {
			return pad(t.Year(), 4) + pad(int(t.Month()), 2) + pad(t.Day(), 2)
		}})
	}
	return out
}

func findSlash(text string, order Order) []Match {
	var out []Match
	for _, g := range slashRE.FindAllStringSubmatchIndex(text, -1) {
		s, e := g[0], g[1]
		if letterBefore(text, s) || letterAfter(text, e) {
			continue
		}
		p := byteAt(text, s-1)
		if p != 0 && strings.IndexByte(`/.\$#=~^_+*<>`, p) >= 0 {
			continue
		}
		from := s - 12
		if from < 0 {
			from = 0
		}
		rangeBefore := slashRangeBefore.MatchString(text[from:s])
		if p == '-' && !rangeBefore {
			continue
		}
		n := byteAt(text, e)
		if n != 0 && strings.IndexByte(`/\"'%_+*=<>`, n) >= 0 {
			continue
		}
		rangeAfter := slashRangeAfter.MatchString(text[e:])
		if (n == '.' || n == ',' || n == ':') && isDigit(byteAt(text, e+1)) {
			continue
		}
		if n == '-' && !rangeAfter && (isDigit(byteAt(text, e+1)) || isLetter(byteAt(text, e+1))) {
			continue
		}
		tok := text[s:e]
		if tok == "24/7" {
			continue
		}
		if unitWords[wordAfter(text, e)] {
			continue
		}
		if labelWords[lastWord(text, s)] {
			continue
		}
		slash := strings.IndexByte(tok, '/')
		as, bs := tok[:slash], tok[slash+1:]
		a, _ := strconv.Atoi(as)
		b, _ := strconv.Atoi(bs)
		dmy := order == DMY
		mo, d := a, b
		if dmy {
			mo, d = b, a
		}
		if mo == 2 && d == 29 {
			continue
		}
		t, ok := valid(refYear, mo, d)
		if !ok {
			continue
		}
		aw := 1
		if strings.HasPrefix(as, "0") || strings.HasPrefix(bs, "0") {
			aw = 2
		}
		bw := aw
		ctx := rangeBefore || rangeAfter || hasCue(text, s, e)
		out = append(out, Match{Start: s, End: e, Text: tok, date: t, needCtx: !ctx, slash: true, render: func(t time.Time) string {
			first, second := int(t.Month()), t.Day()
			if dmy {
				first, second = t.Day(), int(t.Month())
			}
			return pad(first, aw) + "/" + pad(second, bw)
		}})
	}
	return out
}

const nearBytes = 160

func settleContext(ms []Match, loose bool) []Match {
	slashes := 0
	for _, m := range ms {
		if m.slash {
			slashes++
		}
	}
	out := make([]Match, 0, len(ms))
	for i, m := range ms {
		if m.needCtx {
			ok := m.slash && slashes >= 3
			for j := range ms {
				if ok {
					break
				}
				o := ms[j]
				if j == i || o.needCtx {
					continue
				}
				if o.Start-m.End < nearBytes && m.Start-o.End < nearBytes {
					ok = true
				}
			}
			if !ok {
				if !loose {
					continue
				}
				m.Weak = true
			}
		}
		out = append(out, m)
	}
	return out
}
