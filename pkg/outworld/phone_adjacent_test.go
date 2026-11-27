package outworld

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/scaletest"
	"github.com/davidparkercodes/outworld/internal/textutil"
)

var adjacentKnown = [][3]string{{"804", "555", "1234"}, {"703", "555", "2468"}, {"202", "555", "0199"}}

func adjacentGaps() []string {
	var out []string
	for _, r := range textutil.PhoneSeparators() {
		out = append(out, string(r))
	}
	for _, r := range gapSweepRunes(false, 120) {
		out = append(out, string(r))
	}
	return append(out, "    ", "  -  ", "----", " . . ", " -\r\n   ", "\u02bc", "\u0374", "\u02c8", ", ", " / ", "!?", "$", "\u034f-", "\ufe0f", "\U000e0041", "\u3164", "", "\u200b", "\u2060", "\u00ad", "-\u200b", "\u200b-", "\r\n", "-\n", "\n   ", " \u200b", "\u2028", "\u2029", "\u0085", "\v", "\f", "\u200e", "\u202a", "\u2066", "\u061c", "\u180e", "-\u2028")
}

var digitStyles = []rune{0xFF10, 0x0660, 0x0966, 0x1D7CE, 0x1D7D8, 0x09E6}

func styleDigits(r *rand.Rand, s string) string {
	var b strings.Builder
	base := digitStyles[r.Intn(len(digitStyles))]
	mixed := r.Intn(3) == 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			if mixed {
				base = digitStyles[r.Intn(len(digitStyles))]
			}
			b.WriteRune(base + c - '0')
			continue
		}
		b.WriteRune(c)
	}
	return b.String()
}

func adjacentPhoneText(r *rand.Rand, gaps []string) string {
	links := []string{" ", ", ", "", "\n", "; ", " or ", "/", "\u200b", " 9 ", "x", "\t"}
	edges := []string{"", "Call ", "9 ", "+1 ", "ext 2 ", "q=%41", "\u096f ", " now", "-9", ", thanks", "\u00b2", "%20", "Tel", "\u0422\u0435\u043b", "\u05d8\u05dc", "\u96fb\u8a71", "x", "\u00e9"}
	var b strings.Builder
	b.WriteString(edges[r.Intn(len(edges))])
	n := 1 + r.Intn(4)
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(links[r.Intn(len(links))])
		}
		p := adjacentKnown[r.Intn(len(adjacentKnown))]
		if r.Intn(5) == 0 {
			p = [3]string{"2" + string(rune('0'+r.Intn(10))) + "9", "555", "0" + string(rune('1'+r.Intn(8))) + "33"}
		}
		if r.Intn(6) == 0 {
			b.WriteString("1-")
		}
		phone := p[0] + gaps[r.Intn(len(gaps))] + p[1] + gaps[r.Intn(len(gaps))] + p[2]
		if r.Intn(8) == 0 {
			phone = styleDigits(r, phone)
		}
		b.WriteString(phone)
	}
	b.WriteString(edges[r.Intn(len(edges))])
	return b.String()
}

func randomSplit(r *rand.Rand, text string) []int {
	var sizes []int
	for left := len(text); left > 0; {
		k := 1 + r.Intn(12)
		sizes = append(sizes, k)
		left -= k
	}
	return sizes
}

func TestStreamEqualsBatchForAdjacentPhones(t *testing.T) {
	inputs := 1500
	if scaletest.Race {
		inputs = 150
	}
	gaps := adjacentGaps()
	for _, mode := range []string{"seed", "history"} {
		r := rand.New(rand.NewSource(int64(len(mode))))
		e := newEngine(t, nil)
		if _, err := e.NewSession("fill").Mask(ctx, "Call 804\u2013555\u20131234 now"); err != nil {
			t.Fatal(err)
		}
		if mode == "history" {
			phoneHistory(t, e, 300)
		}
		if _, err := e.NewSession("s0").Mask(ctx, "Call 703-555-2468 or 202-555-0199 now"); err != nil {
			t.Fatal(err)
		}
		bad, masked, redacted, inexact := 0, 0, 0, 0
		for i := 0; i < inputs && bad < 10; i++ {
			in := adjacentPhoneText(r, gaps)
			out, err := e.NewSession("s0").Mask(ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			if out == in {
				continue
			}
			masked++
			batch := e.NewSession("s0").Rehydrate(out)
			if strings.Contains(out, "[redacted") {
				redacted++
			} else if batch != in {
				inexact++
				t.Errorf("%s: %q -> %q restores %q", mode, in, out, batch)
			}
			var plans [][]int
			if i < 200 {
				for k := 1; k <= 40; k++ {
					plans = append(plans, []int{k})
				}
			} else {
				plans = append(plans, []int{1}, []int{2}, []int{3 + r.Intn(38)})
			}
			plans = append(plans, randomSplit(r, out), randomSplit(r, out))
			for _, plan := range plans {
				if got := streamed(e.NewSession("s0"), out, plan); got != batch {
					bad++
					t.Errorf("%s: %q -> %q: stream %v gives %q, batch %q", mode, in, out, plan, got, batch)
					break
				}
			}
		}
		t.Logf("%s: %d inputs, %d masked, %d redacted, %d batch restores differ from the input", mode, inputs, masked, redacted, inexact)
	}
}

func TestKnownPhoneWithLetterLikeSeparatorsMasksAndRestores(t *testing.T) {
	forms := []string{
		"1 202\u30fc555.0199 0",
		"804\u30fc555\u30fc1234",
		"804\uff70555\uff701234",
		"804\u2015555\u20151234",
		"804\u2043555\u20431234",
		"804\u2027555\u22c51234",
		"804\u2219555\u02d71234",
		"804\u2796555\uff651234",
		"804\u0387555\u2e311234",
		"804\u223c555\u02dc1234",
		"804\ufe4d555\u20241234",
		"Call 804\u30fc555\u30fc1234 and 703\u30fc555\u30fc2468",
	}
	for _, mode := range []string{"seed", "history"} {
		e := newEngine(t, nil)
		if _, err := e.NewSession("s0").Mask(ctx, "Call 804-555-1234 or 703-555-2468 or 202-555-0199 now"); err != nil {
			t.Fatal(err)
		}
		if mode == "history" {
			phoneHistory(t, e, 200)
		}
		for _, in := range forms {
			out, err := e.NewSession("s0").Mask(ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			for _, known := range []string{"8045551234", "7035552468", "2025550199"} {
				if strings.Contains(digitsOf(textutil.FoldForms(out)), known) {
					t.Errorf("%s: %q -> %q sends %s", mode, in, out, known)
				}
			}
			if back := e.NewSession("s0").Rehydrate(out); back != in {
				t.Errorf("%s: %q -> %q restores %q", mode, in, out, back)
				continue
			}
			for size := 1; size <= 24; size++ {
				if got := streamed(e.NewSession("s0"), out, []int{size}); got != in {
					t.Errorf("%s: %q -> %q streamed at %d gives %q", mode, in, out, size, got)
					break
				}
			}
		}
	}
}

func TestStrayDigitBeforeKnownNumberIsNotSwallowed(t *testing.T) {
	e := newEngine(t, nil)
	for _, seed := range []string{"Call 804-555-1234 now", "Dial \u096f 804-555-1234 now", "Call +1 804\u2014555\u20011234 now"} {
		if _, err := e.NewSession("s0").Mask(ctx, seed); err != nil {
			t.Fatal(err)
		}
	}
	for _, in := range []string{"9 804*555\u200b1234 ext 12", "9 9 9 804\u30fb555-\u200b1234 1234", "+9 804\u30fb555-\u200b1234 (9)", "\u096f 1-804\u2024555 \u200b1234\t202\u00b7555\u20080199 now", "Call 804\u2009555\u20091234 703\u00ad555\u00ad2468"} {
		out, err := e.NewSession("s0").Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(digitsOf(textutil.FoldForms(out)), "8045551234") {
			t.Errorf("%q -> %q sends the known number", in, out)
		}
		if back := e.NewSession("s0").Rehydrate(out); back != in && !strings.Contains(out, "[redacted") {
			t.Errorf("%q -> %q restores %q", in, out, back)
		}
	}
	mixed := "Call 306\u2009555\u20090172 306\u00ad555\u00ad0172"
	if _, err := e.NewSession("s0").Mask(ctx, "Call 804 555 1234 now"); err != nil {
		t.Fatal(err)
	}
	sur, _ := e.Vault().Get("phone", "804-555-1234")
	if sur != "" {
		d := digitsOf(sur)
		mixed = "Call " + d[:3] + "\u2009" + d[3:6] + "\u2009" + d[6:] + " " + d[:3] + "\u00ad" + d[3:6] + "\u00ad" + d[6:]
		want := "Call 804\u2009555\u20091234 804\u00ad555\u00ad1234"
		if got := e.NewSession("s0").Rehydrate(mixed); got != want {
			t.Errorf("model text %q restores %q, want %q", mixed, got, want)
		}
	}
}

func TestLongPhoneListStreamsWithoutSplittingAStandIn(t *testing.T) {
	gaps := []string{" ", "\u200b", "\u00ad", "\u2009", "-", "\u30fb", "\n"}
	e := newEngine(t, nil)
	if _, err := e.NewSession("s0").Mask(ctx, "Call 804-555-1234 or 703-555-2468 or 202-555-0199 now"); err != nil {
		t.Fatal(err)
	}
	r := rand.New(rand.NewSource(256))
	for k := 0; k < 12; k++ {
		var b strings.Builder
		for b.Len() < 300+40*k {
			if b.Len() > 0 {
				b.WriteString(gaps[r.Intn(len(gaps))])
			}
			p := adjacentKnown[r.Intn(len(adjacentKnown))]
			b.WriteString(p[0] + gaps[r.Intn(len(gaps))] + p[1] + gaps[r.Intn(len(gaps))] + p[2])
		}
		in := "List: " + b.String() + " end"
		out, err := e.NewSession("s0").Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		batch := e.NewSession("s0").Rehydrate(out)
		if batch != in && !strings.Contains(out, "[redacted") {
			t.Errorf("%q restores %q", out, batch)
		}
		for _, plan := range [][]int{{1}, {2}, {7}, {20}, {33}, randomSplit(r, out)} {
			if got := streamed(e.NewSession("s0"), out, plan); got != batch {
				t.Errorf("list %d at %v: stream %q, batch %q", k, plan, got, batch)
				break
			}
		}
	}
}

func TestEverySeparatorClassMasksAndRestores(t *testing.T) {
	seps := []string{" ", "\u00a0", "\u1680", "\u2002", "\u2009", "\u202f", "\u205f", "\u3000", "\u2028", "\u2029", "\u0085", "\v", "\f", "\t", "\r\n", "\u2010", "\u2013", "\u2014", "\u2e3a", "\u301c", "\ufe58", "\ufe63", "\uff0d", "\u058a", "\u1806", "\u200b", "\u200e", "\u200f", "\u2061", "\u2064", "\u202a", "\u202e", "\u2066", "\u2069", "\ufeff", "\u00ad", "\u061c", "\u180e", "\u30fc", "\uff70", "\u2015", "\u2043", "\u01c0", "\u02c9", "\u02ca", "\u02cc", "\u02d0", "\u1427", "\u2cba", "\u034f", "\ufe0f", "\U000e0041", "\u3164", "\u115f", "\u17b4", "\u180b"}
	styled := []string{
		"Tel\uff18\uff10\uff14\uff15\uff15\uff15\uff11\uff12\uff13\uff14",
		"x\u0668\u0660\u0664\u0665\u0665\u0665\u0661\u0662\u0663\u0664y",
		"\u0905\u096e\u0966\u096a\u096b\u096b\u096b\u0967\u0968\u0969\u096a\u0906",
		"a\U0001D7D6\U0001D7CE\U0001D7D2555\uff11\uff12\uff13\uff14b",
		"\u0422\u0435\u043b\uff18\uff10\uff14-\uff15\uff15\uff15-\uff11\uff12\uff13\uff14",
		"\u96fb\u8a71\u09ee\u09e6\u09ea\u09eb\u09eb\u09eb\u09e7\u09e8\u09e9\u09ea\u756a\u53f7",
	}
	for _, mode := range []string{"seed", "history"} {
		var cases []string
		for _, sep := range seps {
			cases = append(cases, "804"+sep+"555"+sep+"1234", "Call 804"+sep+"555"+sep+"1234 now", "x804"+sep+"555"+sep+"1234y 703"+sep+"555"+sep+"2468")
		}
		cases = append(cases, styled...)
		for _, in := range cases {
			e := newEngine(t, nil)
			if _, err := e.NewSession("s0").Mask(ctx, "Call 804-555-1234 or 703-555-2468 now"); err != nil {
				t.Fatal(err)
			}
			if mode == "history" {
				phoneHistory(t, e, 40)
			}
			out, err := e.NewSession("s0").Mask(ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			folded := digitsOf(textutil.FoldForms(out))
			if strings.Contains(folded, "8045551234") || strings.Contains(folded, "7035552468") {
				t.Errorf("%s: %q -> %q sends a known number", mode, in, out)
				continue
			}
			batch := e.NewSession("s0").Rehydrate(out)
			if batch != in {
				t.Errorf("%s: %q -> %q restores %q", mode, in, out, batch)
				continue
			}
			for size := 1; size <= 24; size += 1 + size/6 {
				if got := streamed(e.NewSession("s0"), out, []int{size}); got != batch {
					t.Errorf("%s: %q -> %q streamed at %d gives %q, batch %q", mode, in, out, size, got, batch)
					break
				}
			}
		}
	}
}
