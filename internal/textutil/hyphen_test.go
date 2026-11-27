package textutil

import (
	"testing"
	"unicode"
)

func TestHyphenVariantListMatchesPredicate(t *testing.T) {
	listed := map[rune]bool{}
	for _, r := range HyphenVariants {
		if !IsHyphenVariant(r) {
			t.Errorf("%U is listed but IsHyphenVariant rejects it", r)
		}
		listed[r] = true
	}
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if IsHyphenVariant(r) && !listed[r] {
			t.Errorf("%U is a hyphen variant missing from HyphenVariants", r)
		}
	}
}

func TestIsHyphen(t *testing.T) {
	if !IsHyphen('-') {
		t.Error("ASCII hyphen rejected")
	}
	for _, r := range HyphenVariants {
		if !IsHyphen(r) {
			t.Errorf("%U rejected", r)
		}
	}
	for _, r := range []rune{'_', '.', ' ', 0x2013, 0x2014} {
		if IsHyphen(r) {
			t.Errorf("%U accepted", r)
		}
	}
}

func TestHyphenFields(t *testing.T) {
	got := HyphenFields("Jean‐Luc-Marie−Ann")
	if len(got) != 4 || got[0] != "Jean" || got[3] != "Ann" {
		t.Errorf("got %q", got)
	}
}

func TestRangeOrNameDashes(t *testing.T) {
	for _, r := range RangeOrNameDashes {
		if !IsRangeOrNameDash(r) || !IsNameJoiner(r) || IsHyphen(r) {
			t.Errorf("%U misclassified", r)
		}
	}
	for _, r := range []rune{'-', 0x2014, 0x2015, ' '} {
		if IsRangeOrNameDash(r) {
			t.Errorf("%U accepted as range or name dash", r)
		}
	}
	if got := NameJoinerFields("Lily–Rose-Ann‒Marie‐Jo"); len(got) != 5 {
		t.Errorf("got %q", got)
	}
}
