package outworld

import (
	"math/rand"
	"os"
	"strings"
	"testing"
	"unicode"

	"github.com/davidparkercodes/outworld/internal/scaletest"
	"github.com/davidparkercodes/outworld/internal/textutil"
)

func gapSweepRunes(full bool, sample int) []rune {
	var all, ignorable []rune
	for r := rune(1); r <= unicode.MaxRune; r++ {
		if r >= 0xD800 && r <= 0xDFFF || unicode.IsDigit(r) || textutil.GluingDigit(r) {
			continue
		}
		if textutil.IsPhoneIgnorable(r) || unicode.In(r, unicode.Lm, unicode.Sk) {
			ignorable = append(ignorable, r)
			continue
		}
		if unicode.IsLetter(r) || !unicode.In(r, unicode.M, unicode.N, unicode.P, unicode.S, unicode.Z, unicode.Cc, unicode.Cf, unicode.Co) {
			continue
		}
		all = append(all, r)
	}
	if full {
		return append(ignorable, all...)
	}
	r := rand.New(rand.NewSource(4040))
	r.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })
	return append(ignorable, all[:min(sample, len(all))]...)
}

func TestAnyGapRuneJoinsAKnownPhone(t *testing.T) {
	sample := 400
	if scaletest.Race {
		sample = 40
	}
	runes := gapSweepRunes(os.Getenv("OUTWORLD_FULL_GAP_SWEEP") != "", sample)
	for _, mode := range []string{"seed", "history"} {
		e := newEngine(t, nil)
		if _, err := e.NewSession("s0").Mask(ctx, "Call 804-555-1234 now"); err != nil {
			t.Fatal(err)
		}
		if mode == "history" {
			phoneHistory(t, e, 60)
		}
		bad := 0
		for i, g := range runes {
			if bad >= 20 {
				break
			}
			for _, in := range []string{"Call 804" + string(g) + "555" + string(g) + "1234 now", "804-555" + string(g) + "1234"} {
				out, err := e.NewSession("s0").Mask(ctx, in)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(digitsOf(textutil.FoldForms(out)), "8045551234") || out == in {
					bad++
					t.Errorf("%s: gap U+%04X: %q -> %q keeps the known number", mode, g, in, out)
					continue
				}
				batch := e.NewSession("s0").Rehydrate(out)
				if batch != in && !strings.Contains(out, "[redacted") {
					bad++
					t.Errorf("%s: gap U+%04X: %q -> %q restores %q", mode, g, in, out, batch)
					continue
				}
				size := 1 + i%7
				for _, plan := range [][]int{{size}, {1}} {
					if got := streamed(e.NewSession("s0"), out, plan); got != batch {
						bad++
						t.Errorf("%s: gap U+%04X: %q -> %q streams %q at %v, batch %q", mode, g, in, out, got, plan, batch)
						break
					}
				}
			}
		}
		t.Logf("%s: %d gap runes checked", mode, len(runes))
	}
}

func TestGapRuleLeavesLettersLongGapsAndOtherNumbersAlone(t *testing.T) {
	join := []string{
		"Call 804    555    1234 now",
		"Call 804    555-1234 now",
		"804      555      1234",
		"804  -  555  -  1234",
		"Tel:  804  -  555  -  1234",
		"(804)  -  555-1234",
		"804 -- 555 -- 1234",
		"804----555----1234",
		"804....555....1234",
		"804 . . 555 . . 1234",
		"804 // 555 // 1234",
		"804 \u2013 \u2013 555 \u2013 \u2013 1234",
		"| 804 |  555 |  1234 |",
		"804 \u200b - \u200b 555 1234",
		"804 -\r\n   555 -\r\n   1234",
		"804\n\n\n\n\n\n\n555\n1234",
		"804\u02bc555\u02bc1234",
		"804\u02b9555\u03741234",
		"804" + strings.Repeat(" ", 32) + "555-1234",
		"804\u034f\u034f\u034f\u034f\u034f-555-1234",
		"804 " + strings.Repeat("\u200e", 30) + " - " + strings.Repeat("\u200b", 30) + " 555 1234",
		"804 " + strings.Repeat("\u034f", 300) + " - " + strings.Repeat("\u200b", 1000) + " 555 1234",
		"804" + strings.Repeat("\ufe0f", 2000) + "555" + strings.Repeat("\U000e0041", 1000) + "1234",
	}
	raw := []string{
		"804a555b1234",
		"804x555x1234",
		"804\u4e94555\u516d1234",
		"804\u0627555\u06281234",
		"804" + strings.Repeat(" ", 33) + "555-1234",
		"804" + strings.Repeat("-", 17) + "\u2013" + strings.Repeat(" ", 13) + "555 1234",
		strings.Join(strings.Split("8045551234", ""), strings.Repeat(" ", 20)),
		"804!555?1235",
		"804-555-12345",
	}
	for _, mode := range []string{"seed", "history"} {
		e := newEngine(t, nil)
		if _, err := e.NewSession("s0").Mask(ctx, "Call 804\u2013555\u20131234 now"); err != nil {
			t.Fatal(err)
		}
		if mode == "history" {
			phoneHistory(t, e, 60)
		}
		for _, in := range join {
			text := "see " + in + " here"
			out, err := e.NewSession("s0").Mask(ctx, text)
			if err != nil || strings.Contains(digitValues(out), "8045551234") {
				t.Errorf("%s: %q goes out as %q (%v)", mode, in, out, err)
				continue
			}
			if back := e.NewSession("s0").Rehydrate(out); back != text {
				t.Errorf("%s: %q -> %q restores %q", mode, in, out, back)
				continue
			}
			for _, size := range []int{1, 2, 5, 13} {
				if got := streamed(e.NewSession("s0"), out, []int{size}); got != text {
					t.Errorf("%s: %q -> %q streamed at %d gives %q", mode, in, out, size, got)
					break
				}
			}
		}
		for _, in := range raw {
			if out, err := e.NewSession("s0").Mask(ctx, "see "+in+" here"); err != nil || out != "see "+in+" here" {
				t.Errorf("%s: %q changed to %q (%v)", mode, in, out, err)
			}
		}
	}
}

func TestPaddedDigitsScaleLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("scale test skipped in short mode")
	}
	e := newEngine(t, nil)
	if _, err := e.NewSession("s0").Mask(ctx, "Call 804-555-1234 now"); err != nil {
		t.Fatal(err)
	}
	scope := uniqueScope("pad")
	for name, unit := range map[string]string{
		"invisible": "8" + strings.Repeat("\u200b", 40) + "0",
		"gaps":      "80 - - 45" + strings.Repeat(" ", 20),
		"mixed":     "804" + strings.Repeat("\ufe0f", 30) + " . " + strings.Repeat("\u034f", 30) + "555",
	} {
		scaletest.Check(t, "mask padded "+name, scaletest.Size(16<<10), func(n int) func() {
			text := strings.Repeat(unit, n/len(unit)+1)
			return func() {
				if _, err := e.NewSession(scope()).Mask(ctx, text); err != nil {
					t.Fatal(err)
				}
			}
		})
		scaletest.Check(t, "stream padded "+name, scaletest.Size(32<<10), func(n int) func() {
			text := strings.Repeat(unit, n/len(unit)+1)
			return func() {
				streamed(e.NewSession("s0"), text, []int{7})
			}
		})
	}
}
