package textutil

import (
	"regexp"
	"testing"
	"unicode"
)

func TestPhoneSeparatorClassMatchesTheRunePredicate(t *testing.T) {
	class := regexp.MustCompile(`^` + PhoneSeparatorClass() + `$`)
	for r := rune(0); r <= 0xFFFF; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue
		}
		if got, want := class.MatchString(string(r)), IsPhoneSeparator(r); got != want {
			t.Errorf("U+%04X: class %v, IsPhoneSeparator %v", r, got, want)
		}
	}
	for _, r := range "\n\r\t " {
		if !IsPhoneSeparator(r) {
			t.Errorf("%q is not a phone separator", r)
		}
	}
}

func TestPhoneGapTakesLineBreaks(t *testing.T) {
	gap := regexp.MustCompile(`^\d{3}` + PhoneGapPattern() + `\d{3}` + PhoneGapPattern() + `\d{4}$`)
	for _, in := range []string{"804\n555\n1234", "804\r\n555\r\n1234", "804-555-\n1234", "804-\r\n555-\r\n1234", "804\n    555\n    1234", "804 \n555 \n1234", "804\r555\r1234", "804\t555\t1234", "804-555-1234"} {
		if !gap.MatchString(in) {
			t.Errorf("%q is not read as one number", in)
		}
	}
	for _, in := range []string{"804\u2028555\u20291234", "804\v555\f1234", "804-\u0085555-\u20281234", "804 \u2029  555\r\n1234"} {
		if !gap.MatchString(in) {
			t.Errorf("%q is not read as one number", in)
		}
	}
	for _, in := range []string{"804\n\n555\n1234", "804--555-1234", "804x555x1234"} {
		if gap.MatchString(in) {
			t.Errorf("%q is read as one number", in)
		}
	}
}

func TestPhoneSeparatorsCoverWholeUnicodeClasses(t *testing.T) {
	missing := 0
	check := func(class string, r rune) {
		if !IsPhoneSeparator(r) {
			missing++
			if missing < 20 {
				t.Errorf("%s U+%04X is not a phone separator", class, r)
			}
		}
	}
	for name, tab := range map[string]*unicode.RangeTable{"Zs": unicode.Zs, "Zl": unicode.Zl, "Zp": unicode.Zp, "Pd": unicode.Pd, "White_Space": unicode.White_Space} {
		forEachRune(tab, func(r rune) { check(name, r) })
	}
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if unicode.IsSpace(r) {
			check("IsSpace", r)
		}
	}
	invisible := []rune{0x00AD, 0x061C, 0x180E, 0xFEFF}
	for _, rg := range [][2]rune{{0x200B, 0x200F}, {0x2060, 0x2064}, {0x202A, 0x202E}, {0x2066, 0x2069}} {
		for r := rg[0]; r <= rg[1]; r++ {
			invisible = append(invisible, r)
		}
	}
	for _, r := range invisible {
		if !IsZeroWidth(r) {
			t.Errorf("U+%04X is not treated as invisible", r)
		}
		check("Cf", r)
	}
	for _, r := range "\n\r\v\f\u0085\u2028\u2029" {
		if !IsLineBreak(r) {
			t.Errorf("U+%04X is not a line break", r)
		}
	}
	if missing > 0 {
		t.Errorf("%d runes missing in all", missing)
	}
}
