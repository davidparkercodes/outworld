package corpus

import (
	"fmt"
	"strings"
	"time"
)

type builder struct {
	item  Item
	cur   strings.Builder
	role  string
	open  bool
	spans []Span
}

func newBuilder(id, source, template string) *builder {
	return &builder{item: Item{ID: id, Source: source, Template: template}}
}

func (b *builder) system(s string) *builder {
	b.item.Messages = append(b.item.Messages, Message{Role: "system", Content: s})
	return b
}

func (b *builder) begin(role string) *builder {
	if b.open {
		b.end()
	}
	b.role = role
	b.open = true
	b.cur.Reset()
	b.spans = nil
	return b
}

func (b *builder) end() *builder {
	if !b.open {
		return b
	}
	idx := len(b.item.Messages)
	for _, s := range b.spans {
		s.Msg = idx
		b.item.Spans = append(b.item.Spans, s)
	}
	b.item.Messages = append(b.item.Messages, Message{Role: b.role, Content: b.cur.String()})
	b.open = false
	return b
}

func (b *builder) t(parts ...string) *builder {
	if !b.open {
		b.begin("user")
	}
	for _, p := range parts {
		b.cur.WriteString(p)
	}
	return b
}

func (b *builder) e(typ, value, canonical, variant string, needles ...string) *builder {
	return b.ep(typ, value, canonical, variant, "", needles...)
}

func (b *builder) ep(typ, value, canonical, variant, person string, needles ...string) *builder {
	if value == "" {
		panic(fmt.Sprintf("%s: empty %s span", b.item.ID, typ))
	}
	if !b.open {
		b.begin("user")
	}
	start := b.cur.Len()
	b.cur.WriteString(value)
	var ns []string
	for _, n := range needles {
		if n != "" && !strings.EqualFold(n, value) {
			ns = append(ns, n)
		}
	}
	b.spans = append(b.spans, Span{
		Start: start, End: b.cur.Len(), Type: typ, Value: value,
		Canonical: canonical, Variant: variant, Person: person, Needles: ns,
	})
	return b
}

func (b *builder) tag(tags ...string) *builder {
	for _, t := range tags {
		found := false
		for _, x := range b.item.Tags {
			if x == t {
				found = true
			}
		}
		if !found {
			b.item.Tags = append(b.item.Tags, t)
		}
	}
	return b
}

func (b *builder) fact(d time.Time, p *Person) *builder {
	b.item.Facts = append(b.item.Facts, Fact{Date: d.Format("2006-01-02"), Person: p.Full()})
	return b
}

func (b *builder) build() Item {
	b.end()
	return b.item
}

func (b *builder) p(p *Person, variant string) *builder {
	full := p.Full()
	switch variant {
	case "full":
		b.ep(TypePerson, full, full, variant, p.ID)
	case "first":
		b.ep(TypePerson, p.First, full, variant, p.ID)
		if p.CommonWord {
			b.tag("common-word-name")
		}
	case "nickname":
		if p.Nick == "" {
			return b.p(p, "first")
		}
		b.ep(TypePerson, p.Nick, full, variant, p.ID)
		b.tag("nickname")
	case "possessive":
		b.ep(TypePerson, p.First, full, variant, p.ID)
		b.t("'s")
		b.tag("possessive")
	case "full-possessive":
		b.ep(TypePerson, full, full, "possessive", p.ID)
		b.t("'s")
		b.tag("possessive")
	case "caps":
		b.ep(TypePerson, strings.ToUpper(full), full, variant, p.ID)
		b.tag("all-caps")
	case "lower":
		b.ep(TypePerson, strings.ToLower(p.First), full, variant, p.ID)
		b.tag("lowercase-name")
	case "title-last":
		b.t(p.Title + " ")
		b.ep(TypePerson, p.Last, full, variant, p.ID)
	case "last-first":
		b.ep(TypePerson, p.Last+", "+p.First, full, variant, p.ID)
	case "markdown":
		b.ep(TypePerson, "**"+p.First+"** "+p.Last, full, variant, p.ID)
		b.tag("markdown-split")
	case "markdown-mid":
		b.ep(TypePerson, p.First+" _"+p.Last+"_", full, "markdown", p.ID)
		b.tag("markdown-split")
	case "handle":
		b.ep(TypePerson, strings.ToLower(p.First)+"_"+strings.ToLower(p.Last[:1]), full, variant, p.ID)
		b.tag("chat-handles")
	case "url-slug":
		b.ep(TypePerson, p.Slug(), full, variant, p.ID)
		b.tag("name-in-url")
	default:
		panic("unknown person variant " + variant)
	}
	return b
}

func (b *builder) email(p *Person, i int) *builder {
	addr := p.Emails[i%len(p.Emails)]
	b.ep(TypeEmail, addr, strings.ToLower(addr), "", p.ID)
	if strings.Contains(strings.ToLower(addr), strings.ToLower(p.First)) || strings.Contains(strings.ToLower(addr), strings.ToLower(p.Last)) {
		b.tag("name-in-email")
	}
	return b
}

func (b *builder) phone(p *Person, format int) *builder {
	v := formatPhone(p.PhoneArea, p.PhoneLine, format)
	b.ep(TypePhone, v, "+1"+p.PhoneArea+"555"+p.PhoneLine, phoneFormats[format%len(phoneFormats)], p.ID)
	return b
}

func (b *builder) address(p *Person, withUnit bool) *builder {
	a := p.Address
	street := a.Street()
	full := street
	if withUnit && a.Unit != "" {
		full += ", " + a.Unit
	}
	full += ", " + a.City + ", " + a.State + " " + a.Zip
	b.ep(TypeAddress, full, full, "", p.ID, street)
	return b
}

func (b *builder) date(d time.Time, format string) *builder {
	v := FormatDate(d, format)
	iso := d.Format("2006-01-02")
	var needles []string
	if dateHasYear(format) {
		needles = []string{iso, d.Format("January 2, 2006"), d.Format("01/02/2006")}
	}
	b.e(TypeDate, v, iso, format, needles...)
	return b
}

func (b *builder) org(o Org) *builder {
	return b.e(TypeOrg, o.Name, o.Name, "")
}

func (b *builder) loc(l string) *builder {
	return b.e(TypeLocation, l, l, "")
}

func (b *builder) secret(v, variant string, needles ...string) *builder {
	return b.e(TypeSecret, v, v, variant, needles...)
}

func (b *builder) quasi(v string) *builder {
	b.tag("quasi-identifier")
	return b.e(TypeQuasi, v, v, "")
}

var phoneFormats = []string{"paren", "dash", "intl", "dot"}

func formatPhone(area, line string, format int) string {
	switch format % len(phoneFormats) {
	case 0:
		return "(" + area + ") 555-" + line
	case 1:
		return area + "-555-" + line
	case 2:
		return "+1 " + area + " 555 " + line
	default:
		return area + ".555." + line
	}
}

var DateFormats = []string{"iso", "long", "short", "us", "eu", "dotted", "weekday", "monthday", "ordinal", "header"}

func dateHasYear(format string) bool {
	switch format {
	case "weekday", "monthday", "ordinal":
		return false
	}
	return true
}

func FormatDate(d time.Time, format string) string {
	switch format {
	case "iso":
		return d.Format("2006-01-02")
	case "long":
		return d.Format("January 2, 2006")
	case "short":
		return d.Format("Jan 2, 2006")
	case "us":
		return d.Format("01/02/2006")
	case "eu":
		return d.Format("2 January 2006")
	case "dotted":
		return d.Format("02.01.2006")
	case "weekday":
		return d.Format("Monday, January 2")
	case "monthday":
		return d.Format("Jan 2")
	case "ordinal":
		return "the " + ordinal(d.Day()) + " of " + d.Format("January")
	case "header":
		return d.Format("Mon, 2 Jan 2006 15:04")
	}
	panic("unknown date format " + format)
}

func ordinal(n int) string {
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
}
