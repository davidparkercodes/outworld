package vault

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

var digitForms = map[string]rune{
	"full-width":    0xFF10,
	"math bold":     0x1D7CE,
	"double-struck": 0x1D7D8,
	"sans":          0x1D7E2,
	"sans bold":     0x1D7EC,
	"monospace":     0x1D7F6,
	"subscript":     0x2080,
}

var superscripts = []rune{0x2070, 0x00B9, 0x00B2, 0x00B3, 0x2074, 0x2075, 0x2076, 0x2077, 0x2078, 0x2079}

func inForm(s string, zero rune, super bool) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9' && super:
			b.WriteRune(superscripts[r-'0'])
		case r >= '0' && r <= '9':
			b.WriteRune(zero + r - '0')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TestFoldedDigitFormsShareTheASCIIRecord(t *testing.T) {
	kinds := map[string][2]string{
		"phone":   {"804-555-1234", "900-555-9999"},
		"ip":      {"10.1.2.3", "10.9.8.7"},
		"address": {"Hauptstraße 5, 10115 Berlin", "Hauptstraße 7, 10117 Berlin"},
		"id":      {"AB-123456", "AB-654321"},
		"email":   {"jane.1@example.com", "jane.2@example.com"},
	}
	forms := map[string]func(string) string{}
	for name, zero := range digitForms {
		zero := zero
		forms[name] = func(s string) string { return inForm(s, zero, false) }
	}
	forms["superscript"] = func(s string) string { return inForm(s, 0, true) }
	forms["zero width"] = func(s string) string { return s[:1] + "\u200b" + s[1:] }
	for kind, vals := range kinds {
		v, err := Open(NewMemoryBackend(bytes.Repeat([]byte{4}, 32)))
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		gen := func(int) string { n++; return fmt.Sprintf("SUR%d", n) }
		a, _ := v.Assign(kind, vals[0], gen)
		b, _ := v.Assign(kind, vals[1], gen)
		if a == b {
			t.Fatalf("%s: two ASCII values share %q", kind, a)
		}
		for name, f := range forms {
			fa, fb := f(vals[0]), f(vals[1])
			if fa == vals[0] {
				t.Fatalf("%s %s: form did not change the value", kind, name)
			}
			ga, err := v.Assign(kind, fa, gen)
			if err != nil {
				t.Fatal(err)
			}
			gb, _ := v.Assign(kind, fb, gen)
			if ga != a || gb != b {
				t.Errorf("%s %s: %q -> %q, %q -> %q, want %q and %q", kind, name, fa, ga, fb, gb, a, b)
			}
		}
	}
}

func TestPhoneWithoutDigitsNeverSharesAnEmptyKey(t *testing.T) {
	v, _ := Open(NewMemoryBackend(bytes.Repeat([]byte{4}, 32)))
	a, _ := v.Assign("phone", "five five five", func(int) string { return "A1" })
	b, _ := v.Assign("phone", "nine nine nine", func(int) string { return "B2" })
	if a == b || NormalizeReal("phone", "five five five") == "" {
		t.Fatalf("no-digit phones collide: %q %q", a, b)
	}
}

func TestMathBoldAndSuperscriptPhonesGetDistinctSurrogates(t *testing.T) {
	v, _ := Open(NewMemoryBackend(bytes.Repeat([]byte{4}, 32)))
	n := 0
	gen := func(int) string { n++; return fmt.Sprintf("555-01%02d", n) }
	got := map[string]string{}
	for _, real := range []string{"\U0001D7D6\U0001D7CE\U0001D7D2-\U0001D7D3\U0001D7D3\U0001D7D3-\U0001D7CF\U0001D7D0\U0001D7D1\U0001D7D2", "\U0001D7D7\U0001D7CE\U0001D7CE-\U0001D7D3\U0001D7D3\U0001D7D3-\U0001D7D7\U0001D7D7\U0001D7D7\U0001D7D7", "⁸⁰⁴-⁵⁵⁵-¹²³⁴", "⁹⁰⁰-⁵⁵⁵-⁹⁹⁹⁹"} {
		s, err := v.Assign("phone", real, gen)
		if err != nil {
			t.Fatal(err)
		}
		got[real] = s
		if r, ok := v.Reverse(s); !ok || NormalizeReal("phone", r.Real) != NormalizeReal("phone", real) {
			t.Errorf("%q: reverse gives %+v", real, r)
		}
	}
	seen := map[string]bool{}
	for _, s := range got {
		seen[s] = true
	}
	if len(seen) != 2 {
		t.Fatalf("want two records (804 and 900 numbers in two forms), got %v", got)
	}
}

func TestAssignRejectsValuesThatNormaliseToNothing(t *testing.T) {
	v, _ := Open(NewMemoryBackend(bytes.Repeat([]byte{4}, 32)))
	for _, real := range []string{"", "   ", "\u200b", "\u00ad", "\ufeff", "\u2060", " \u200b\u2060 "} {
		for _, kind := range []string{"phone", "person.given", "email", "address"} {
			if s, err := v.Assign(kind, real, func(int) string { return "X1" }); !errors.Is(err, ErrEmptyReal) {
				t.Errorf("%s %q: got %q, %v", kind, real, s, err)
			}
		}
	}
	if v.Len() != 0 {
		t.Fatalf("records written for empty values: %d", v.Len())
	}
}
