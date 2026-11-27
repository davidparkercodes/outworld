package outworld

import (
	"runtime"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/scaletest"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

func formEngine(t *testing.T, forms int) (*Engine, []string) {
	t.Helper()
	e := newEngine(t, nil)
	var recs []vault.Record
	var emitted []string
	for i := 0; i < forms; i++ {
		tag := string(rune('a'+i%26)) + string(rune('a'+i/26%26)) + string(rune('a'+i/676%26))
		real, sur := "Qu"+tag+"x", "Zo"+tag+"y"
		got, err := e.Vault().Assign("person.given", real, func(int) string { return sur })
		if err != nil || got != sur {
			t.Fatalf("assign %q: %q %v", real, got, err)
		}
		recs = append(recs, vault.Record{Real: "QU" + tag + "x", Surrogate: "ZO" + tag + "y"})
	}
	n, err := e.Vault().AddNameForms(recs)
	if err != nil || n == 0 {
		t.Fatalf("name forms: %d %v", n, err)
	}
	for _, r := range recs[:n] {
		emitted = append(emitted, r.Surrogate)
	}
	return e, emitted
}

func formText(emitted []string, n int) string {
	var b strings.Builder
	for i := 0; b.Len() < n; i++ {
		b.WriteString("Hello " + emitted[i%len(emitted)] + ", see Zaaaay and words here. ")
	}
	return b.String()
}

func TestRehydrateWithFormsScalesLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("scale test skipped in short mode")
	}
	e, emitted := formEngine(t, 100)
	s := e.NewSession("forms")
	if got := s.Rehydrate("Hello " + emitted[0] + ", ok"); got != "Hello QUaaax, ok" {
		t.Fatalf("form not restored: %q", got)
	}
	scaletest.Check(t, "rehydrate with forms", scaletest.Size(64<<10), func(n int) func() {
		text := formText(emitted, n)
		return func() { s.Rehydrate(text) }
	})
	scaletest.Check(t, "stream with forms", scaletest.Size(64<<10), func(n int) func() {
		text := formText(emitted, n)
		return func() { streamed(e.NewSession("forms"), text, []int{20}) }
	})
	alloc := func(text string) float64 {
		var a, b runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&a)
		s.Rehydrate(text)
		runtime.ReadMemStats(&b)
		return float64(b.TotalAlloc-a.TotalAlloc) / float64(len(text))
	}
	small, large := alloc(formText(emitted, scaletest.Size(16<<10))), alloc(formText(emitted, scaletest.Size(256<<10)))
	t.Logf("allocated bytes per input byte: %.1f at 16 KB, %.1f at 256 KB", small, large)
	if !scaletest.Race && large > 2*small+64 {
		t.Errorf("allocations grow with text length: %.1f then %.1f bytes per input byte", small, large)
	}
}
