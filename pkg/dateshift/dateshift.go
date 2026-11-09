package dateshift

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Order int

const (
	MDY Order = iota
	DMY
)

const refYear = 2023

type Match struct {
	Start, End int
	Text       string
	date       time.Time
	hasYear    bool
	render     func(t time.Time) string
}

func (m Match) Date() (time.Time, bool) { return m.date, m.hasYear }

func (m Match) Shift(days int) string {
	if m.hasYear {
		return m.render(m.date.AddDate(0, 0, days))
	}
	doy := (m.date.YearDay() - 1 + ((days%365)+365)%365) % 365
	return m.render(time.Date(refYear, 1, 1, 12, 0, 0, 0, time.UTC).AddDate(0, 0, doy))
}

const monthAlt = `Jan(?:uary)?|Feb(?:ruary)?|Mar(?:ch)?|Apr(?:il)?|May|June?|July?|Aug(?:ust)?|Sept?(?:ember)?|Oct(?:ober)?|Nov(?:ember)?|Dec(?:ember)?`
const weekdayAlt = `Mon(?:day)?|Tue(?:s|sday)?|Wed(?:nesday)?|Thu(?:r|rs|rsday)?|Fri(?:day)?|Sat(?:urday)?|Sun(?:day)?`

var (
	monthAny   = monthAlt + "|" + strings.ToUpper(monthAlt)
	weekdayAny = weekdayAlt + "|" + strings.ToUpper(weekdayAlt)
	isoRE      = regexp.MustCompile(`\b(\d{4})([-/])(\d{1,2})([-/])(\d{1,2})`)
	numericRE  = regexp.MustCompile(`\b(\d{1,2})([/.\-])(\d{1,2})([/.\-])(\d{4}|\d{2})\b`)
	monthDayRE = regexp.MustCompile(`\b(?:(` + weekdayAny + `)\.?,?\s+)?(` + monthAny + `)(\.?)(\s+)(\d{1,2})(st|nd|rd|th)?\b(?:(,?\s+)(\d{4})\b)?`)
	dayMonthRE = regexp.MustCompile(`\b(?:(` + weekdayAny + `)\.?,?\s+)?(\d{1,2})(st|nd|rd|th)?(\s+(?:of\s+)?)(` + monthAny + `)(\.?)\b(?:(,?\s+)(\d{4})\b)?`)
)

var fullMonths = []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}

func monthIndex(s string) int {
	l := strings.ToLower(s)
	for i, m := range fullMonths {
		if strings.HasPrefix(strings.ToLower(m), l[:3]) {
			return i + 1
		}
	}
	return 0
}

func renderMonth(orig string, m time.Month) string {
	return matchUpper(orig, renderMonthTitle(orig, m))
}

func matchUpper(orig, s string) string {
	if len(orig) > 1 && orig == strings.ToUpper(orig) {
		return strings.ToUpper(s)
	}
	return s
}

func renderMonthTitle(orig string, m time.Month) string {
	full := fullMonths[m-1]
	if strings.EqualFold(orig, fullMonths[monthIndex(orig)-1]) {
		return full
	}
	if strings.EqualFold(orig, "Sept") && m == time.September {
		return "Sept"
	}
	return full[:3]
}

func renderWeekday(orig string, d time.Weekday) string {
	return matchUpper(orig, renderWeekdayTitle(orig, d))
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

func renderWeekdayTitle(orig string, d time.Weekday) string {
	full := d.String()
	if len(orig) > 4 {
		return full
	}
	switch titleCase(orig) {
	case "Tues", "Thur", "Thurs":
		if d == time.Tuesday {
			return "Tues"
		}
		if d == time.Thursday {
			return "Thurs"
		}
	}
	return full[:3]
}

func ordinal(n int) string {
	if n%100 >= 11 && n%100 <= 13 {
		return "th"
	}
	switch n % 10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	}
	return "th"
}

func pairWidth(a, b string) int {
	if strings.HasPrefix(a, "0") || strings.HasPrefix(b, "0") || (len(a) == 2 && len(b) == 2) {
		return 2
	}
	return 1
}

func width(s string) int {
	if strings.HasPrefix(s, "0") {
		return len(s)
	}
	return 1
}

func pad(n int, width int) string {
	s := strconv.Itoa(n)
	for len(s) < width {
		s = "0" + s
	}
	return s
}

func valid(y, m, d int) (time.Time, bool) {
	if m < 1 || m > 12 || d < 1 || d > 31 || y < 1 {
		return time.Time{}, false
	}
	t := time.Date(y, time.Month(m), d, 12, 0, 0, 0, time.UTC)
	if t.Day() != d || int(t.Month()) != m {
		return time.Time{}, false
	}
	return t, true
}

func Find(text string, order Order) []Match {
	var out []Match
	for _, g := range isoRE.FindAllStringSubmatchIndex(text, -1) {
		sub := groups(text, g)
		if sub[2] != sub[4] {
			continue
		}
		if g[1] < len(text) {
			if c := text[g[1]]; c >= '0' && c <= '9' || (c != 'T' && (c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z')) {
				continue
			}
		}
		y, _ := strconv.Atoi(sub[1])
		mo, _ := strconv.Atoi(sub[3])
		d, _ := strconv.Atoi(sub[5])
		t, ok := valid(y, mo, d)
		if !ok || y < 1800 || y > 2200 {
			continue
		}
		sep, mw := sub[2], pairWidth(sub[3], sub[5])
		dw := mw
		out = append(out, Match{Start: g[0], End: g[1], Text: text[g[0]:g[1]], date: t, hasYear: true, render: func(t time.Time) string {
			return fmt.Sprintf("%04d%s%s%s%s", t.Year(), sep, pad(int(t.Month()), mw), sep, pad(t.Day(), dw))
		}})
	}
	for _, g := range numericRE.FindAllStringSubmatchIndex(text, -1) {
		sub := groups(text, g)
		if sub[2] != sub[4] {
			continue
		}
		if g[1] < len(text) && (text[g[1]] == '.' || text[g[1]] == '-' || text[g[1]] == '/') && g[1]+1 < len(text) && text[g[1]+1] >= '0' && text[g[1]+1] <= '9' {
			continue
		}
		if g[0] > 0 && strings.IndexByte("./-", text[g[0]-1]) >= 0 {
			continue
		}
		a, _ := strconv.Atoi(sub[1])
		b, _ := strconv.Atoi(sub[3])
		y, _ := strconv.Atoi(sub[5])
		twoDigit := len(sub[5]) == 2
		if twoDigit {
			if sub[2] == "." {
				continue
			}
			if y < 70 {
				y += 2000
			} else {
				y += 1900
			}
		}
		dmy := order == DMY
		if a > 12 && b <= 12 {
			dmy = true
		} else if b > 12 && a <= 12 {
			dmy = false
		}
		mo, d := a, b
		if dmy {
			mo, d = b, a
		}
		t, ok := valid(y, mo, d)
		if !ok {
			continue
		}
		sep, aw := sub[2], pairWidth(sub[1], sub[3])
		bw := aw
		out = append(out, Match{Start: g[0], End: g[1], Text: text[g[0]:g[1]], date: t, hasYear: true, render: func(t time.Time) string {
			first, second := int(t.Month()), t.Day()
			if dmy {
				first, second = t.Day(), int(t.Month())
			}
			ys := pad(t.Year(), 4)
			if twoDigit {
				ys = pad(t.Year()%100, 2)
			}
			return pad(first, aw) + sep + pad(second, bw) + sep + ys
		}})
	}
	for _, g := range monthDayRE.FindAllStringSubmatchIndex(text, -1) {
		sub := groups(text, g)
		wd, mon, dot, sp, day, suf, ysep, year := sub[1], sub[2], sub[3], sub[4], sub[5], sub[6], sub[7], sub[8]
		if m, ok := build(text, g, wd, mon, day, year); ok {
			m.render = func(t time.Time) string {
				var b strings.Builder
				if wd != "" {
					b.WriteString(renderWeekday(wd, t.Weekday()))
					b.WriteString(text[g[3]:g[4]])
				}
				b.WriteString(renderMonth(mon, t.Month()))
				b.WriteString(dot + sp)
				b.WriteString(pad(t.Day(), width(day)))
				if suf != "" {
					b.WriteString(ordinal(t.Day()))
				}
				if year != "" {
					b.WriteString(ysep + strconv.Itoa(t.Year()))
				}
				return b.String()
			}
			out = append(out, m)
		}
	}
	for _, g := range dayMonthRE.FindAllStringSubmatchIndex(text, -1) {
		sub := groups(text, g)
		wd, day, suf, sp, mon, dot, ysep, year := sub[1], sub[2], sub[3], sub[4], sub[5], sub[6], sub[7], sub[8]
		if m, ok := build(text, g, wd, mon, day, year); ok {
			m.render = func(t time.Time) string {
				var b strings.Builder
				if wd != "" {
					b.WriteString(renderWeekday(wd, t.Weekday()))
					b.WriteString(text[g[3]:g[4]])
				}
				b.WriteString(pad(t.Day(), width(day)))
				if suf != "" {
					b.WriteString(ordinal(t.Day()))
				}
				b.WriteString(sp)
				b.WriteString(renderMonth(mon, t.Month()))
				b.WriteString(dot)
				if year != "" {
					b.WriteString(ysep + strconv.Itoa(t.Year()))
				}
				return b.String()
			}
			out = append(out, m)
		}
	}
	return resolve(out)
}

func build(text string, g []int, wd, mon, day, year string) (Match, bool) {
	mo := monthIndex(mon)
	d, _ := strconv.Atoi(day)
	y := refYear
	hasYear := year != ""
	if hasYear {
		y, _ = strconv.Atoi(year)
	} else if mo == 2 && d == 29 {
		return Match{}, false
	}
	t, ok := valid(y, mo, d)
	if !ok {
		return Match{}, false
	}
	return Match{Start: g[0], End: g[1], Text: text[g[0]:g[1]], date: t, hasYear: hasYear}, true
}

func groups(text string, g []int) []string {
	out := make([]string, len(g)/2)
	for i := range out {
		if g[2*i] >= 0 {
			out[i] = text[g[2*i]:g[2*i+1]]
		}
	}
	return out
}

func resolve(ms []Match) []Match {
	sort.SliceStable(ms, func(i, j int) bool {
		li, lj := ms[i].End-ms[i].Start, ms[j].End-ms[j].Start
		if li != lj {
			return li > lj
		}
		return ms[i].Start < ms[j].Start
	})
	var out []Match
	for _, m := range ms {
		clash := false
		for _, o := range out {
			if m.Start < o.End && o.Start < m.End {
				clash = true
				break
			}
		}
		if !clash {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	return out
}

func Shift(text string, order Order, days int) string {
	if days == 0 {
		return text
	}
	ms := Find(text, order)
	if len(ms) == 0 {
		return text
	}
	var b strings.Builder
	last := 0
	for _, m := range ms {
		b.WriteString(text[last:m.Start])
		b.WriteString(m.Shift(days))
		last = m.End
	}
	b.WriteString(text[last:])
	return b.String()
}
