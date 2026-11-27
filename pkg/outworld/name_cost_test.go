package outworld

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/davidparkercodes/outworld/internal/scaletest"
)

func letterRunKeys(n int) string {
	r := rand.New(rand.NewSource(7))
	var b strings.Builder
	b.WriteString("keys:\n")
	for b.Len() < n {
		b.WriteString("- ")
		for i := 0; i < 22; i++ {
			b.WriteByte(byte('a' + r.Intn(26)))
		}
		b.WriteByte('\n')
	}
	return b.String()[:n]
}

func plainEnglish(n int) string {
	words := strings.Fields("the build finished and the release notes describe several small fixes to the parser cache layer while the team met on tuesday to review open items")
	r := rand.New(rand.NewSource(8))
	var b strings.Builder
	for b.Len() < n {
		b.WriteString(words[r.Intn(len(words))])
		if r.Intn(12) == 0 {
			b.WriteString(". ")
		} else {
			b.WriteByte(' ')
		}
	}
	return b.String()[:n]
}

func costEngine(t *testing.T, people int) *Engine {
	t.Helper()
	ctx := context.Background()
	e := newEngine(t, nil)
	first := strings.Fields("Ruby Anna Maria Zoe Liam Noah Emma Olivia Ava Mia Ella Grace Lily Chloe Ivy Hugo Felix Oscar Leo Max Sam Ben Tom Jack Luke")
	last := strings.Fields("Kelly Moss Smith Jones Brown Taylor Wilson Davies Evans Thomas Johnson Roberts Walker Wright Thompson White Hughes Edwards Green Hall Wood Harris Lewis Martin Jackson")
	for i := 0; i < people; i++ {
		if _, err := e.NewSession("seed").Mask(ctx, fmt.Sprintf("%s %s called about the invoice.", first[i%len(first)], last[(i*7)%len(last)])); err != nil {
			t.Fatal(err)
		}
	}
	return e
}

func bestCPU(reps int, fn func()) time.Duration {
	best := time.Duration(1 << 62)
	for i := 0; i < reps; i++ {
		start := scaletest.CPU()
		fn()
		best = min(best, scaletest.CPU()-start)
	}
	return max(best, time.Millisecond)
}

func vaultNamePass(e *Engine, text string) {
	s := e.NewSession(uniqueScope("names")())
	if !e.hasNameKeys() {
		return
	}
	s.gluedVaultNames(text, nil)
	s.pairedVaultWords(text, nil)
	s.runRemainders(text, nil)
	s.adjacentVaultNames(text, nil)
}

func TestNameMatchingCostOnLongLetterRuns(t *testing.T) {
	if testing.Short() {
		t.Skip("cost test skipped in short mode")
	}
	ctx := context.Background()
	size := scaletest.Size(256 << 10)
	keys, prose := letterRunKeys(size), plainEnglish(size)
	check := func(e *Engine, text string) time.Duration {
		n := 0
		return bestCPU(3, func() {
			n++
			e.NewSession(fmt.Sprintf("cost%d", n)).Check(ctx, text)
		})
	}
	bound := func(what string, ratio float64) {
		t.Logf("%-44s %.2fx", what, ratio)
		if !scaletest.Race && ratio > 1.3 {
			t.Errorf("%s: %.2fx (limit 1.3x)", what, ratio)
		}
	}
	empty, full := costEngine(t, 0), costEngine(t, 50)
	checks := map[string]time.Duration{}
	for name, e := range map[string]*Engine{"empty": empty, "50 people": full} {
		pk := bestCPU(3, func() { vaultNamePass(e, keys) })
		pp := bestCPU(3, func() { vaultNamePass(e, prose) })
		bound("name pass, letter runs vs English, "+name, float64(pk)/float64(pp))
		checks[name+" keys"], checks[name+" prose"] = check(e, keys), check(e, prose)
		t.Logf("check %s: letter runs %v, English %v", name, checks[name+" keys"], checks[name+" prose"])
	}
	bound("check letter runs, 50 people vs empty", float64(checks["50 people keys"])/float64(checks["empty keys"]))
	bound("check English, 50 people vs empty", float64(checks["50 people prose"])/float64(checks["empty prose"]))
	prev := time.Duration(0)
	for _, kb := range []int{64, 256, 1024} {
		text := letterRunKeys(scaletest.Size(kb << 10))
		d := bestCPU(2, func() { vaultNamePass(full, text) })
		t.Logf("name pass on %4d KiB of letter runs: %v", kb, d)
		if prev > 0 && !scaletest.Race && float64(d) > 8*float64(prev) {
			t.Errorf("name pass grew %.1fx for 4x input", float64(d)/float64(prev))
		}
		prev = d
	}
}
