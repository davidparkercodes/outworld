package outworld

import (
	"context"
	"strings"
	"testing"
)

func TestGreetingGivenNameBeforeHyphenWord(t *testing.T) {
	cases := map[string][2]string{
		"Dear Lee Walk-in,":  {"Lee", "Walk-in"},
		"Dear Kim Tie-in,":   {"Kim", "Tie-in"},
		"Hi Kim Sign-in,":    {"Kim", "Sign-in"},
		"Dear Mark Sign-in,": {"Mark", "Sign-in"},
	}
	for in, c := range cases {
		s := newEngine(t, nil).NewSession("p")
		out, err := s.Mask(context.Background(), in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if strings.Contains(out, c[0]+" ") || !strings.Contains(out, " "+c[1]+",") {
			t.Errorf("%q: got %q, want %q replaced and %q kept", in, out, c[0], c[1])
		}
		if back := s.Rehydrate(out); back != in {
			t.Errorf("%q: restored %q", in, back)
		}
	}
	in := "Dear Park Walk-in,"
	if out, err := newEngine(t, nil).NewSession("p").Mask(context.Background(), in); err != nil || out != in {
		t.Errorf("%q: got %q (%v)", in, out, err)
	}
}

func TestGreetingFullNamesMaskAndRestore(t *testing.T) {
	cases := map[string][]string{
		"Dear Lily-Rose Depp,":      {"Lily", "Rose", "Depp"},
		"Dear Lily–Rose Depp,":      {"Lily", "Rose", "Depp"},
		"Hi Jean-Luc Picard,":       {"Jean", "Luc", "Picard"},
		"Dear Jean-Claude Juncker,": {"Jean", "Claude", "Juncker"},
		"Dear Mary Smith-Jones,":    {"Mary", "Smith", "Jones"},
		"Dear Kim Min-jun,":         {"Kim", "Min", "jun"},
		"Hi Lee Seo-yeon,":          {"Lee", "Seo", "yeon"},
	}
	split := func(r rune) bool { return r == ' ' || r == ',' || r == '-' || r == '–' }
	for in, parts := range cases {
		s := newEngine(t, nil).NewSession("p")
		out, err := s.Mask(context.Background(), in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		words := map[string]bool{}
		for _, w := range strings.FieldsFunc(out, split) {
			words[w] = true
		}
		for _, p := range parts {
			if words[p] {
				t.Errorf("%q: %q left raw in %q", in, p, out)
			}
		}
		if back := s.Rehydrate(out); back != in {
			t.Errorf("%q: restored %q", in, back)
		}
	}
}
