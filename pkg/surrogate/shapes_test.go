package surrogate

import (
	"regexp"
	"strings"
	"testing"
)

func TestNationalPhoneShapes(t *testing.T) {
	g := NewGenerator([]byte("shape"))
	cases := []struct{ real, want string }{
		{"07911 123456", `^07\d{3} \d{6}$`},
		{"020 7123 4567", `^02\d \d{4} \d{4}$`},
		{"(020) 7123 4567", `^\(02\d\) \d{4} \d{4}$`},
		{"030 12345 678", `^03\d \d{5} \d{3}$`},
		{"01 23 45 67 89", `^01 \d\d \d\d \d\d \d\d$`},
		{"06.12.34.56.78", `^06\.\d\d\.\d\d\.\d\d\.\d\d$`},
		{"0412 345 678", `^04\d\d \d{3} \d{3}$`},
		{"0044 20 7123 4567", `^0044 \d\d \d{4} \d{4}$`},
		{"+49 30 12345 678", `^\+49 \d\d \d{5} \d{3}$`},
		{"+33 1 23 45 67 89", `^\+33 \d \d\d \d\d \d\d \d\d$`},
	}
	for _, c := range cases {
		for i := 0; i < 20; i++ {
			got := g.Phone(c.real, i)
			if !regexp.MustCompile(c.want).MatchString(got) {
				t.Fatalf("%q: surrogate %q does not match %s", c.real, got, c.want)
			}
		}
		if g.Phone(c.real, 0) == c.real {
			t.Fatalf("%q: surrogate equals real", c.real)
		}
	}
}

func TestSpokenPhoneShape(t *testing.T) {
	g := NewGenerator([]byte("shape"))
	word := `(?:zero|oh|one|two|three|four|five|six|seven|eight|nine)`
	real := "four one five, five five five, oh one four two"
	got := g.SpokenPhone(real, 0)
	if !regexp.MustCompile(`^` + word + ` ` + word + ` ` + word + `, ` + word + ` ` + word + ` ` + word + `, ` + word + ` ` + word + ` ` + word + ` ` + word + `$`).MatchString(got) {
		t.Fatalf("spoken %q", got)
	}
	if got == real || got != g.SpokenPhone(real, 0) {
		t.Fatalf("spoken surrogate must differ from the real value and be deterministic: %q", got)
	}
	if got := g.SpokenPhone("Seven DOUBLE five nine three one two", 0); !regexp.MustCompile(`^[A-Z][a-z]+ DOUBLE [a-z]+ [a-z]+ [a-z]+ [a-z]+ [a-z]+$`).MatchString(got) {
		t.Fatalf("spoken case %q", got)
	}
	if got := g.Phone("nine one seven five five five two zero four six", 0); strings.ContainsAny(got, "0123456789") || len(strings.Fields(got)) != 10 {
		t.Fatalf("digitless phone %q", got)
	}
}

func TestAddressShapesOutsideUSStyle(t *testing.T) {
	g := NewGenerator([]byte("shape"))
	cases := []struct{ real, want string }{
		{"14 rue des Lilas, 69003 Lyon", `^\d\d rue des \p{Lu}\p{L}+, \d{5} \p{Lu}\p{L}+$`},
		{"7 bis avenue de l'Europe", `^\d bis avenue de l'\p{Lu}\p{L}+$`},
		{"Calle Alcalde Moreno 21, 41004 Sevilla", `^Calle \p{Lu}\p{L}+ \d\d, \d{5} \p{Lu}\p{L}+$`},
		{"Carrer de Balmes 118, 08008 Barcelona", `^Carrer de \p{Lu}\p{L}+ \d{3}, \d{5} \p{Lu}\p{L}+$`},
		{"Lindenstraße 8, 50674 Köln", `^\p{Lu}\p{L}+straße \d, \d{5} \p{Lu}\p{L}+$`},
		{"Berliner Straße 44, 60311 Frankfurt am Main", `^\p{Lu}\p{L}+ Straße \d\d, \d{5} \p{Lu}\p{L}+$`},
		{"Am Kupfergraben 6, 10117 Berlin", `^Am \p{Lu}\p{L}+ \d, \d{5} \p{Lu}\p{L}+$`},
		{"Herengracht 401, 1017 BP Amsterdam", `^\p{Lu}\p{L}+gracht \d{3}, \d{4} [A-Z]{2} \p{Lu}\p{L}+$`},
		{"Via Garibaldi 30, 10122 Torino", `^Via \p{Lu}\p{L}+ \d\d, \d{5} \p{Lu}\p{L}+$`},
		{"Flat 4, 27 Glenorchy Road, Perth PH1 5XY", `^Flat \d, \d\d [A-Z][a-z]+ Road, [A-Z][a-z]+ [A-Z]{2}\d \d[A-Z]{2}$`},
		{"88 Canal Street\nManchester\nM1 3HE", `^\d\d [A-Z][a-z]+ Street\n[A-Z][a-z]+\n[A-Z]\d \d[A-Z]{2}$`},
		{"2207 HOLLOWAY RD STE 14\nRALEIGH NC 27603", `^\d{4} [A-Z]+ RD STE \d\d\n[A-Z]+ NC \d{5}$`},
		{"915 N ORCHARD AVE, BOISE ID 83706-1120", `^\d{3} N [A-Z]+ AVE, [A-Z]+ ID \d{5}-\d{4}$`},
	}
	for _, c := range cases {
		got := g.Address(c.real, 0)
		if !regexp.MustCompile(c.want).MatchString(got) {
			t.Errorf("%q: surrogate %q does not match %s", c.real, got, c.want)
		}
		for _, w := range regexp.MustCompile(`\p{L}{5,}`).FindAllString(c.real, -1) {
			if strings.Contains(got, w) && !addrStreetWords[strings.ToLower(w)] && !regexp.MustCompile(`^(?:Calle|Carrer|avenue|Straße)$`).MatchString(w) {
				t.Errorf("%q: surrogate %q keeps %q", c.real, got, w)
			}
		}
	}
	if a := g.Address("123 Main Street, Apt 4B, Richmond, VA 23220", 0); !regexp.MustCompile(`^\d{3} [A-Z][a-z]+ Street, Apt \d[A-Z], [A-Z][a-z]+, VA \d{5}$`).MatchString(a) {
		t.Fatalf("US style changed: %q", a)
	}
}
