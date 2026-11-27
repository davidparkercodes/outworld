package outworld

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/scaletest"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

var symmetrySeeds = []string{
	"Call 951 555 5490 now",
	"Email maria.garcia@example.com today",
	"Met Maria Garcia and John Smith today",
	"Ship to 742 Evergreen Terrace, Springfield, IL 62704",
	"Server 10.20.30.41 is down",
	"Meeting on March 3, 2024 at noon",
}

func symmetryEngine(t *testing.T, mode string) (*Engine, []string) {
	t.Helper()
	e := newEngine(t, nil)
	if mode == "history" {
		phoneHistory(t, e, 60)
	}
	var shifted []string
	for _, in := range symmetrySeeds {
		out, err := e.NewSession("s0").Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(in, "Meeting on ") {
			shifted = append(shifted, strings.TrimSuffix(strings.TrimPrefix(out, "Meeting on "), " at noon"))
		}
	}
	return e, shifted
}

func fullWidth(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '!' && r <= '~' {
			b.WriteRune(r + 0xFEE0)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func standInShapes(r *rand.Rand, rec vault.Record) string {
	s := rec.Surrogate
	switch r.Intn(9) {
	case 0:
		return " " + strings.ToUpper(s) + " "
	case 1:
		return " " + strings.ToLower(s) + " "
	case 2:
		return "x" + s + "y"
	case 3:
		return "(" + s + ")-+1 703 555 2468"
	case 4:
		return " " + fullWidth(s) + " "
	case 5:
		d := digitsOf(s)
		if len(d) == 10 {
			seps := []string{"\u2060", "-", " ", ".", "\u200b", "/", "--", " - "}
			return "(" + d[:3] + ")" + seps[r.Intn(len(seps))] + d[3:6] + seps[r.Intn(len(seps))] + d[6:]
		}
	case 6:
		if i := strings.IndexByte(s, '@'); i > 0 {
			return s[:i]
		}
	case 7:
		if f := strings.Fields(s); len(f) > 1 {
			return f[len(f)-1] + "," + f[0]
		}
	}
	return s
}

func TestRestoreNeverTouchesWhatMaskLeft(t *testing.T) {
	n := 400
	if scaletest.Race {
		n = 60
	}
	fillers := []string{"see ", "page 3, ", "re: ", "", "\n", "Call ", "q=%41", "\u00e9", "x"}
	for _, mode := range []string{"seed", "history"} {
		e, shifted := symmetryEngine(t, mode)
		var recs []vault.Record
		for _, rec := range e.Vault().Records() {
			if !strings.HasPrefix(rec.Kind, "vault.") {
				recs = append(recs, rec)
			}
		}
		r := rand.New(rand.NewSource(int64(len(mode)) * 7))
		bad := 0
		for i := 0; i < n && bad < 10; i++ {
			var b strings.Builder
			for k := 0; k < 1+r.Intn(3); k++ {
				b.WriteString(fillers[r.Intn(len(fillers))])
				if r.Intn(5) == 0 && len(shifted) > 0 {
					b.WriteString(shifted[r.Intn(len(shifted))])
				} else {
					b.WriteString(standInShapes(r, recs[r.Intn(len(recs))]))
				}
			}
			b.WriteString(fillers[r.Intn(len(fillers))])
			in := b.String()
			for _, scope := range []string{"s0", "s1"} {
				out, err := e.NewSession(scope).Mask(ctx, in)
				if err != nil {
					t.Fatal(err)
				}
				back := e.NewSession(scope).Rehydrate(out)
				if back != in && !strings.Contains(out, "[redacted") {
					bad++
					t.Errorf("%s %s: %q -> %q restores %q", mode, scope, in, out, back)
					continue
				}
				for _, rec := range recs {
					if len(rec.Real) > 3 && strings.Contains(back, rec.Real) && !strings.Contains(in, rec.Real) {
						t.Errorf("%s %s: %q restores another record's real %q", mode, scope, in, rec.Real)
					}
				}
				if got := streamed(e.NewSession(scope), out, []int{1 + i%9}); got != back {
					bad++
					t.Errorf("%s %s: %q -> %q streams %q, batch %q", mode, scope, in, out, got, back)
				}
			}
		}
	}
}

func TestStandInTypedInAnotherScopeStaysAsTyped(t *testing.T) {
	for _, mode := range []string{"seed", "history"} {
		e := newEngine(t, nil)
		if mode == "history" {
			phoneHistory(t, e, 60)
		}
		out, err := e.NewSession("s0").Mask(ctx, "Call 951 555 5490 now")
		if err != nil {
			t.Fatal(err)
		}
		d := digitsOf(out)
		in := "page 3, (" + d[:3] + ") " + d[3:6] + "\u2060" + d[6:] + "-+1 703 555 2468"
		for _, scope := range []string{"s1", "s0"} {
			masked, err := e.NewSession(scope).Mask(ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			if back := e.NewSession(scope).Rehydrate(masked); back != in {
				t.Errorf("%s %s: %q -> %q restores %q", mode, scope, in, masked, back)
			}
			for size := 1; size <= 12; size++ {
				if got := streamed(e.NewSession(scope), masked, []int{size}); got != in {
					t.Errorf("%s %s: %q -> %q streamed at %d gives %q", mode, scope, in, masked, size, got)
					break
				}
			}
			if strings.Contains(e.NewSession(scope).Rehydrate(masked), "5490") {
				t.Errorf("%s %s: the s0 number reaches %q", mode, scope, in)
			}
		}
	}
}
