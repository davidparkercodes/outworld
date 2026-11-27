package outworld

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/scaletest"
	"github.com/davidparkercodes/outworld/internal/textutil"
)

func phoneHistory(t *testing.T, e *Engine, n int) {
	t.Helper()
	r := rand.New(rand.NewSource(12))
	lefts := []string{"", "page 3, ", "x=9;", "Call ", "1,", "703-555-2468, ", "ext 2 ", "q=%41", "Rooms "}
	rights := []string{"", " 703 555 2468", " or 703-555-2468", "%41", " ext 12", " now", "-9", "\nbye"}
	seps := []string{"-", " ", ".", "\u2010", "*", "~", "\u2007", "\u00ad", "-\u200b", "\u2060", "\u2009", "\n", " \n"}
	for i := 0; i < n; i++ {
		in := lefts[r.Intn(len(lefts))] + "804" + seps[r.Intn(len(seps))] + "555" + seps[r.Intn(len(seps))] + "1234" + rights[r.Intn(len(rights))]
		if _, err := e.NewSession("s0").Mask(ctx, in); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPhoneAcrossLineBreaksMasksAndStreams(t *testing.T) {
	forms := []string{
		"804\n555\n1234",
		"804 555\n1234",
		"Rooms 804 555\n1234 apples",
		"Call 804-555-1234\nthanks",
		"Call me:\n804\n555\n1234\nbye",
		"804\r\n555\r\n1234",
		"804-555-\n1234",
		"804\r555\r1234",
		"804\t555\t1234",
		"804-\r\n555-\r\n1234",
		"804\n    555\n    1234",
		"804 \n555 \n1234",
		"Call me:\r\n  804.\r\n  555.\r\n  1234\r\nbye",
	}
	untouched := []string{"804\n555\n12345", "1804\n555\n12340", "8045\n555\n1234", "804\n5550\n1234"}
	columns := []string{"804\n555\n9999", "104\n555\n9999", "x\n804\n555\n1234567", "703\n555\n2468"}
	fresh := newEngine(t, nil)
	history := 400
	if scaletest.Race {
		history = 60
	}
	for _, mode := range []string{"seed", "history"} {
		e := newEngine(t, nil)
		if _, err := e.NewSession("seed").Mask(ctx, "Call 804-555-1234 now"); err != nil {
			t.Fatal(err)
		}
		if mode == "history" {
			phoneHistory(t, e, history)
		}
		for _, in := range forms {
			for _, scope := range []string{"seed", "other"} {
				out, err := e.NewSession(scope).Mask(ctx, in)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(digitsOf(textutil.FoldForms(out)), "8045551234") {
					t.Errorf("%s %s: %q -> %q sends the known number", mode, scope, in, out)
					continue
				}
				if back := e.NewSession(scope).Rehydrate(out); back != in {
					t.Errorf("%s %s: %q -> %q restores %q", mode, scope, in, out, back)
					continue
				}
				for size := 1; size <= 24; size++ {
					if got := streamed(e.NewSession(scope), out, []int{size}); got != in {
						t.Errorf("%s %s: %q -> %q streamed at %d gives %q", mode, scope, in, out, size, got)
						break
					}
				}
			}
		}
		for _, in := range untouched {
			if out, err := e.NewSession("seed").Mask(ctx, in); err != nil || out != in {
				t.Errorf("%s: column %q changed to %q (%v)", mode, in, out, err)
			}
		}
		for _, in := range columns {
			want, err := fresh.NewSession("col").Mask(ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			out, err := e.NewSession("col").Mask(ctx, in)
			if err != nil || (out == in) != (want == in) {
				t.Errorf("%s: column %q became %q, an empty vault gives %q (%v)", mode, in, out, want, err)
			}
		}
	}
}
