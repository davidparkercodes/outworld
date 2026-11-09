package textutil

import "testing"

func TestMatcherBoundariesAndCase(t *testing.T) {
	m := NewMatcher()
	m.Add("John Smith", 1, AnyCase)
	m.Add("John", 2, Capitalized)
	m.Add("Ann", 3, AnyCase)

	cases := []struct {
		text string
		want []Match
	}{
		{"Call John Smith now", []Match{{5, 15, 1}}},
		{"JOHN SMITH called", []Match{{0, 10, 1}}},
		{"john smith called", []Match{{0, 10, 1}}},
		{"John's car", []Match{{0, 4, 2}}},
		{"john's car", nil},
		{"Johnson is here", nil},
		{"Anne and Ann", []Match{{9, 12, 3}}},
		{"(Ann)", []Match{{1, 4, 3}}},
		{"x.ann.y", []Match{{2, 5, 3}}},
		{"planner", nil},
	}
	for _, c := range cases {
		got := m.FindAll(c.text)
		if len(got) != len(c.want) {
			t.Fatalf("%q: got %v want %v", c.text, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("%q: got %v want %v", c.text, got, c.want)
			}
		}
	}
}

func TestMatcherIsPrefixAt(t *testing.T) {
	m := NewMatcher()
	m.Add("Dalia Rinaldi", 1, AnyCase)
	m.Add("(475) 555-0101", 2, AnyCase)
	m.Add("dalia.rinaldi@example.org", 3, AnyCase)
	for _, c := range []struct {
		s    string
		want bool
	}{
		{"D", true}, {"Dal", true}, {"dalia", true}, {"Dalia R", true}, {"Dalia Rinaldi", true},
		{"Dalia Rinaldis", false}, {"Dalx", false}, {"(", true}, {"(47", true}, {"(475) 555-01", true},
		{"dalia.rinaldi@ex", true}, {"x", false}, {"", false},
	} {
		if got := m.IsPrefixAt(c.s, 0); got != c.want {
			t.Errorf("IsPrefixAt(%q)=%v want %v", c.s, got, c.want)
		}
	}
}

func TestMatchCase(t *testing.T) {
	for _, c := range [][3]string{
		{"JOHN", "Dalia", "DALIA"},
		{"john", "Dalia", "dalia"},
		{"John", "Dalia", "Dalia"},
		{"John", "dalia", "Dalia"},
		{"McDonald", "Rinaldi", "Rinaldi"},
	} {
		if got := MatchCase(c[0], c[1]); got != c[2] {
			t.Errorf("MatchCase(%q,%q)=%q want %q", c[0], c[1], got, c[2])
		}
	}
}

func TestRuleFor(t *testing.T) {
	if RuleFor("Will") != Capitalized || RuleFor("Al") != Capitalized {
		t.Fatal("common or short single words must require capitalization")
	}
	if RuleFor("Josephine") != AnyCase || RuleFor("Will Smith") != AnyCase {
		t.Fatal("distinctive names should match any case")
	}
}
