package outworld

import (
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/scaletest"
	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/entity"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

func contextLefts() []string {
	out := []string{"", " "}
	for _, d := range []string{"9", "1", "07"} {
		for _, sep := range []string{" ", "-", "", "\u00a0"} {
			out = append(out, d+sep)
		}
	}
	return append(out, "%41", "%2B", "%20", "x", "Dial ", "Dial 9 ", "\u96fb\u8a71", "\uc804\ud654", "(", "[", "q=%41", "_")
}

func contextRights() []string {
	out := []string{"", " "}
	for _, d := range []string{"9", "0", "12"} {
		for _, sep := range []string{" ", "-", "", "\u00a0"} {
			out = append(out, sep+d)
		}
	}
	return append(out, "%20", "%41", "x", " now", "\u3067\u3059", ")", "]", "_", ".")
}

func contextPhones(r *rand.Rand, n int) []string {
	seps := textutil.PhoneSeparators()
	zw := []string{"", "\u200b", "\u00ad", "\u2060"}
	out := []string{"804-555-1234", "804\u2013555\u20131234", "804\u200b555\u200b1234", "(804) 555-1234", "+1 804 555 1234"}
	for len(out) < n {
		a := string(seps[r.Intn(len(seps))]) + zw[r.Intn(len(zw))]
		b := string(seps[r.Intn(len(seps))]) + zw[r.Intn(len(zw))]
		out = append(out, "804"+a+"555"+b+"1234")
	}
	return out
}

type contextResult struct{ in, out string }

var unchangedOutputs, unchangedWithFindings int

func phoneFindings(t *testing.T, e *Engine, in string) []string {
	t.Helper()
	spans, err := e.Detect(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[[2]int]bool{}
	var out []string
	add := func(a, b int) {
		if !seen[[2]int{a, b}] {
			seen[[2]int{a, b}] = true
			out = append(out, in[a:b])
		}
	}
	for _, sp := range spans {
		if sp.Type == entity.Phone && strings.ContainsAny(textutil.FoldForms(sp.Text), "0123456789") {
			add(sp.Start, sp.End)
		}
	}
	for _, h := range e.Vault().FindReals(in) {
		if strings.HasPrefix(h.Record.Kind, "phone") {
			add(h.Start, h.End)
		}
	}
	for _, f := range e.NewSession("findings").knownPhones(in, nil) {
		add(f.span.Start, f.span.End)
	}
	return out
}

func runContexts(t *testing.T, e *Engine, phones []string) []contextResult {
	t.Helper()
	rng := rand.New(rand.NewSource(int64(len(phones))))
	var res []contextResult
	pair := 0
	for _, p := range phones {
		for _, l := range contextLefts() {
			for _, rt := range contextRights() {
				pair++
				if scaletest.Race && pair%3 != 0 {
					continue
				}
				in := l + p + rt
				findings := phoneFindings(t, e, in)
				s := e.NewSession("ctx")
				out, err := s.Mask(ctx, in)
				if err != nil {
					t.Fatal(err)
				}
				for f, k := range countFindings(findings) {
					if strings.Count(out, f)-(strings.Count(in, f)-k) > 0 && len(s.Violations()) == 0 {
						t.Errorf("finding %q left in place with no violation: %q -> %q", f, in, out)
					}
				}
				if out == in {
					unchangedOutputs++
					if len(findings) > 0 {
						unchangedWithFindings++
					}
					continue
				}
				if strings.Contains(out, "[redacted") {
					continue
				}
				if back := s.Rehydrate(out); back != in {
					t.Errorf("same session %q -> %q -> %q", in, out, back)
				}
				ns := e.NewSession("ctx")
				if back := ns.Rehydrate(out); back != in {
					t.Errorf("new session %q -> %q -> %q", in, out, back)
				}
				if rng.Intn(4) == 0 {
					size := 1 + rng.Intn(9)
					if got := streamed(ns, out, []int{size}); got != in {
						t.Errorf("streamed at %d %q -> %q -> %q", size, in, out, got)
					}
				}
				res = append(res, contextResult{in, out})
			}
		}
	}
	return res
}

func TestEveryMaskedPhoneRestoresInItsContext(t *testing.T) {
	r := rand.New(rand.NewSource(41))
	n := 12
	if scaletest.Race {
		n = 3
	}
	phones := contextPhones(r, n)
	for _, pastCap := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "v.bin")
		e := fileEngine(t, path)
		if _, err := e.NewSession("seed").Mask(ctx, "Call 804\u2013555\u20131234 now"); err != nil {
			t.Fatal(err)
		}
		if pastCap {
			seps := textutil.PhoneSeparators()
			for k := 0; !e.Vault().FormsFull("8045551234") && k < 4*vault.MaxPhoneForms; k++ {
				a, b := string(seps[k%len(seps)]), string(seps[(k*5+1)%len(seps)])
				if _, err := e.NewSession("fill").Mask(ctx, "Call 804"+a+"555"+b+"1234 now"); err != nil {
					t.Fatal(err)
				}
			}
			if !e.Vault().FormsFull("8045551234") {
				t.Fatal("cap not reached")
			}
		}
		res := runContexts(t, e, phones)
		e.Vault().Close()
		reopened := fileEngine(t, path)
		for _, c := range res {
			if back := reopened.NewSession("ctx").Rehydrate(c.out); back != c.in {
				t.Errorf("after reopen %q -> %q -> %q", c.in, c.out, back)
			}
		}
		t.Logf("past cap %v: %d masked occurrences checked, records %d, unchanged outputs %d (with a finding %d)", pastCap, len(res), reopened.Vault().Len(), unchangedOutputs, unchangedWithFindings)
		if unchangedWithFindings > 0 {
			t.Errorf("%d unchanged outputs still held a finding", unchangedWithFindings)
		}
		unchangedOutputs, unchangedWithFindings = 0, 0
	}
}

func TestReviewerContextRepros(t *testing.T) {
	e := newEngine(t, nil)
	if _, err := e.NewSession("seed").Mask(ctx, "Call 804\u2013555\u20131234 now"); err != nil {
		t.Fatal(err)
	}
	for _, in := range []string{"q=%41804\u2013555\u20131234%20x", "%41804\u2013555\u20131234"} {
		s := e.NewSession("p")
		out, err := s.Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if out != in {
			if back := s.Rehydrate(out); back != in {
				t.Errorf("%q -> %q -> %q", in, out, back)
			}
		}
	}
	seps := textutil.PhoneSeparators()
	for k := 0; !e.Vault().FormsFull("8045551234"); k++ {
		a, b := string(seps[k%len(seps)]), string(seps[(k*3+2)%len(seps)])
		e.NewSession("fill").Mask(ctx, "Call 804"+a+"555"+b+"1234 now")
	}
	for _, in := range []string{"Dial 9 804\u200b555\u200b1234 now", "9 804-\u200b555-\u200b1234 now", "\u96fb\u8a71\u756a\u53f7\u306f804-\u200b555\u22121234-9", "9 804\u2212555\u22121234_ ", "%41804\u2010555-1234%20"} {
		s := e.NewSession("p")
		out, err := s.Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if out != in {
			if back := s.Rehydrate(out); back != in {
				t.Errorf("past cap %q -> %q -> %q", in, out, back)
			}
			if got := streamed(e.NewSession("p"), out, []int{1}); got != in {
				t.Errorf("past cap streamed %q -> %q -> %q", in, out, got)
			}
		}
	}
}

func maxHeld(s *Session, text string, chunk int) (int, string) {
	r := s.NewStreamRehydrator(false)
	held := 0
	var b strings.Builder
	for pos := 0; pos < len(text); pos += chunk {
		b.WriteString(r.Push(text[pos:min(len(text), pos+chunk)]))
		held = max(held, r.Pending())
	}
	b.WriteString(r.Flush())
	return held, b.String()
}

func longRuns(n int) map[string]string {
	b64 := "QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVphYmNkZWZnaGlqa2xtbm9wcXJzdHV2d3h5ejAxMjM0NTY3ODkrLw"
	return map[string]string{
		"letters": strings.Repeat("a", n),
		"cjk":     strings.Repeat("\u96fb\u8a71\u756a\u53f7\u306f", n/15+1),
		"base64":  strings.Repeat(b64, n/len(b64)+1),
		"digits":  strings.Repeat("7", n),
	}
}

func TestStreamHoldbackIsBoundedOnLongRuns(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("run")
	r := s.NewStreamRehydrator(false)
	bound := 2*r.window() + streamEvalStep + utf8MaxChunk
	sizes := []int{1 << 10, 16 << 10, 40 << 10, 256 << 10}
	if scaletest.Race {
		sizes = []int{1 << 10, 4 << 10}
	}
	for _, n := range sizes {
		for name, text := range longRuns(n) {
			for _, chunk := range []int{1, 7, 64, 4096} {
				if chunk < 64 && n > 16<<10 {
					continue
				}
				held, out := maxHeld(s, text, chunk)
				if out != text {
					t.Fatalf("%s %d chunk %d: output changed", name, n, chunk)
				}
				if held > bound+chunk {
					t.Errorf("%s %d bytes at chunk %d: held %d bytes, bound %d", name, n, chunk, held, bound+chunk)
				}
			}
		}
	}
}

const utf8MaxChunk = 4

func TestStreamPushCostIsLinear(t *testing.T) {
	if testing.Short() {
		t.Skip("scale test skipped in short mode")
	}
	e := newEngine(t, nil)
	s := e.NewSession("run")
	for name := range longRuns(8) {
		name := name
		scaletest.Check(t, "stream "+name, scaletest.Size(128<<10), func(n int) func() {
			text := longRuns(n)[name]
			return func() { maxHeld(s, text, 64) }
		})
	}
}

func TestStandInsInsideLongRunsStream(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("run")
	masked, err := s.Mask(ctx, "\u96fb\u8a71804-555-1234 and Met John Smith")
	if err != nil {
		t.Fatal(err)
	}
	standIns := strings.Fields(masked)
	n := 40 << 10
	if scaletest.Race {
		n = 2 << 10
	}
	rng := rand.New(rand.NewSource(2026))
	for name, run := range longRuns(n) {
		for _, w := range []string{standIns[0], standIns[len(standIns)-1]} {
			for _, text := range []string{run[:len(run)/2] + w + run[len(run)/2:], run + w, run} {
				want := s.Rehydrate(text)
				for _, chunk := range []int{3, 64, 4096} {
					if _, got := maxHeld(s, text, chunk); got != want {
						t.Fatalf("%s run, chunk %d: streamed differs from unstreamed", name, chunk)
					}
				}
				if got := streamed(s, text, []int{1 + rng.Intn(50), 1 + rng.Intn(500)}); got != want {
					t.Fatalf("%s run, random split: streamed differs", name)
				}
			}
		}
	}
}
