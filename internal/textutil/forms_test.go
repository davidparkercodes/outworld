package textutil

import "testing"

func TestLetterlikeAndMathLettersFold(t *testing.T) {
	cases := map[string]string{
		"\u2130\U0001D4C1\u212F\U0001D4C3\U0001D4B6 \U0001D4AB\u212F\U0001D4C9\U0001D4C7\u2134\U0001D4CB\U0001D4B6":             "Elena Petrova",
		"\U0001D53C\U0001D55D\U0001D556\U0001D55F\U0001D552 \u2119\U0001D556\U0001D565\U0001D563\U0001D560\U0001D567\U0001D552": "Elena Petrova",
		"\U0001D43D\U0001D45C\u210E\U0001D45B \U0001D446\U0001D45A\U0001D456\U0001D461\u210E":                                   "John Smith",
		"\u210B\u2112\u2133\u211B\u212C\u2131\u2110\u2102\u2115\u211A\u2124\u211D\u210D\u210C\u2111\u211C\u2128\u212D":          "HLMRBFICNQZRHHIRZC",
		"\u210A\u2113\u212F\u2134\u210E": "gleoh",
	}
	for in, want := range cases {
		if got := FoldForms(in); got != want {
			t.Errorf("%q folded to %q, want %q", in, got, want)
		}
	}
}

func TestDashAndZeroWidthSetsAreShared(t *testing.T) {
	for _, r := range []rune{0x2010, 0x2011, 0x2012, 0x2013, 0x2014, 0x2015, 0x2212, 0xFE63, 0xFF0D} {
		if !IsDashVariant(r) {
			t.Errorf("%U is not a dash", r)
		}
		if f, ok := formRune(r); !ok || f != '-' {
			t.Errorf("%U folds to %q", r, f)
		}
	}
	for _, r := range []rune{0x200B, 0x200C, 0x200D, 0x2060, 0xFEFF, 0x00AD} {
		if !IsZeroWidth(r) {
			t.Errorf("%U is not zero width", r)
		}
		if f, ok := formRune(r); !ok || f != -1 {
			t.Errorf("%U folds to %q", r, f)
		}
	}
}
