package outworld

import (
	"strings"
	"testing"
)

var dateFormatTexts = []string{
	"Nos vemos el 3 de diciembre de 2026 y otra vez el 15 de noviembre.",
	"Termin am 28. Oktober 2026, Abgabe 19.11.26.",
	"Rendez-vous le 9 novembre, puis op 3 december 2026.",
	"Issued 12-Nov-2026, outbound 02NOV26, return 14NOV.",
	"DTSTART:20261123T150000Z and stamp 20261127.",
	"We said december third, or the twenty-first of March.",
	"Rent is due 11/3, review by 10/14, week of 12/1.",
	"Slots: 11/3, 11/10, 11/17 and 12/1.",
}

func TestDateFormatsMaskAndRehydrate(t *testing.T) {
	e := newEngine(t, nil)
	for _, original := range dateFormatTexts {
		s := e.NewSession("")
		masked, err := s.Mask(ctx, original)
		if err != nil {
			t.Fatalf("%q: %v", original, err)
		}
		if masked == original {
			t.Errorf("not shifted: %q", original)
		}
		if got := s.Rehydrate(masked); got != original {
			t.Errorf("rehydrate\n got: %q\nwant: %q\n via: %q", got, original, masked)
		}
	}
}

func TestDateFormatsEchoedWithoutContext(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("")
	masked, _ := s.Mask(ctx, "Rent is due 11/3.")
	shifted := strings.TrimSuffix(strings.TrimPrefix(masked, "Rent is due "), ".")
	if shifted == "11/3" {
		t.Fatalf("not shifted: %q", masked)
	}
	if got := s.Rehydrate("Noted: " + shifted + " it is."); got != "Noted: 11/3 it is." {
		t.Errorf("bare echo not restored: %q", got)
	}
	if got := s.Rehydrate("Use 3/4 cup and a 5/6 majority."); got != "Use 3/4 cup and a 5/6 majority." {
		t.Errorf("unrelated fraction changed: %q", got)
	}
}

func TestDateFormatsStreamAllSplits(t *testing.T) {
	e := newEngine(t, nil)
	for _, original := range dateFormatTexts {
		s := e.NewSession("")
		masked, _ := s.Mask(ctx, original)
		for size := 1; size <= 7; size++ {
			r := s.NewStreamRehydrator(false)
			var b strings.Builder
			for i := 0; i < len(masked); i += size {
				end := i + size
				if end > len(masked) {
					end = len(masked)
				}
				b.WriteString(r.Push(masked[i:end]))
			}
			b.WriteString(r.Flush())
			if b.String() != original {
				t.Errorf("size=%d\n got: %q\nwant: %q\n via: %q", size, b.String(), original, masked)
			}
		}
	}
}
