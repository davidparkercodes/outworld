package scaletest

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const metaSize = 16 << 10

var sink int

func quadratic(n int) func() {
	s := strings.Repeat("a", n)
	return func() {
		for i := 0; i < len(s); i++ {
			sink += strings.Count(s[i:], "a")
		}
	}
}

func linear(n int) func() {
	s := strings.Repeat("a", n)
	return func() {
		for k := 0; k < 64; k++ {
			sink += strings.Count(s, "a")
		}
	}
}

type recorder struct {
	errors []string
	logs   []string
}

func (r *recorder) Helper() {}

func (r *recorder) Logf(format string, args ...any) {
	r.logs = append(r.logs, fmt.Sprintf(format, args...))
}

func (r *recorder) Errorf(format string, args ...any) {
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

func TestMeasureRejectsQuadraticAndAcceptsLinear(t *testing.T) {
	if r := Measure(quadratic(metaSize/Input), quadratic(metaSize)); r.OK || r.Ratio <= Limit {
		t.Errorf("a quadratic loop was accepted at %.1fx", r.Ratio)
	}
	if r := Measure(linear(metaSize/Input), linear(metaSize)); !r.OK {
		t.Errorf("a linear loop was rejected at %.1fx", r.Ratio)
	}
}

func TestCheckReportsQuadraticAndPassesLinear(t *testing.T) {
	var sizes []int
	record := func(work func(int) func()) func(int) func() {
		return func(n int) func() {
			sizes = append(sizes, n)
			return work(n)
		}
	}
	quad := &recorder{}
	r := Check(quad, "quadratic", metaSize, record(quadratic))
	if len(sizes) != 2 || sizes[0]*Input != sizes[1] || sizes[1] != metaSize {
		t.Fatalf("Check prepared sizes %v, want %d and %d", sizes, metaSize/Input, metaSize)
	}
	if r.OK || r.Ratio <= Limit {
		t.Errorf("Check accepted a quadratic loop at %.1fx", r.Ratio)
	}
	if !Race && len(quad.errors) == 0 {
		t.Errorf("Check reported no error for a quadratic loop at %.1fx", r.Ratio)
	}
	if len(quad.logs) == 0 {
		t.Error("Check logged nothing")
	}
	sizes = nil
	lin := &recorder{}
	if r := Check(lin, "linear", metaSize, record(linear)); !r.OK || len(lin.errors) != 0 {
		t.Errorf("Check rejected a linear loop at %.1fx: %v", r.Ratio, lin.errors)
	}
	if len(sizes) != 2 || sizes[0]*Input != sizes[1] {
		t.Fatalf("Check prepared sizes %v for the linear loop", sizes)
	}
}

func TestQuadraticFailsAfterEveryRetry(t *testing.T) {
	for attempt := 0; attempt < 3; attempt++ {
		r := Measure(quadratic(metaSize/Input), quadratic(metaSize))
		if r.OK || r.Ratio <= Limit {
			t.Fatalf("attempt %d: a quadratic loop was accepted at %.1fx", attempt, r.Ratio)
		}
		want := maxRounds
		if Race {
			want = 1
		}
		if r.Rounds != want {
			t.Fatalf("attempt %d: gave up after %d of %d rounds", attempt, r.Rounds, want)
		}
	}
	if r := Measure(linear(metaSize/Input), linear(metaSize)); r.Rounds < 1 || !r.OK {
		t.Fatalf("a linear loop: %+v", r)
	}
}

func TestGrowthConstants(t *testing.T) {
	if Input != 8 || Limit != 16 {
		t.Fatalf("Input %d, Limit %.0f: the growth check expects 8x input against a 16x bound", Input, Limit)
	}
}

func TestSizeScalesUnderRace(t *testing.T) {
	if want := 1024; !Race && Size(want) != want {
		t.Fatalf("Size changed input outside race: %d", Size(want))
	}
	if Race && Size(1024) != 1024/raceDivide {
		t.Fatalf("Size under race: %d", Size(1024))
	}
}

func mixed(fraction float64) func(int) func() {
	return func(n int) func() {
		s := strings.Repeat("a", n)
		quadPasses := int(fraction * float64(metaSize*Input) / float64(metaSize))
		return func() {
			for k := 0; k < 64*Input; k++ {
				sink += strings.Count(s, "a")
			}
			for p := 0; p < quadPasses; p++ {
				for i := 0; i < len(s); i += 64 {
					sink += strings.Count(s[i:], "a")
				}
			}
		}
	}
}

func TestDetectionLimit(t *testing.T) {
	if Race {
		t.Skip("timing")
	}
	if r := Measure(mixed(4)(metaSize/Input), mixed(4)(metaSize)); r.OK {
		t.Errorf("a quadratic part well above the linear part was accepted at %.1fx", r.Ratio)
	}
	if r := Measure(mixed(0.25)(metaSize/Input), mixed(0.25)(metaSize)); !r.OK {
		t.Errorf("a small quadratic part was rejected at %.1fx", r.Ratio)
	}
}

func TestClockIgnoresTimeSpentWaiting(t *testing.T) {
	if !CPUClock {
		t.Skip("no process CPU clock on this platform")
	}
	start := now()
	time.Sleep(200 * time.Millisecond)
	if d := now() - start; d > 50*time.Millisecond {
		t.Errorf("a 200ms sleep cost %v of measured time", d)
	}
	start = now()
	linear(metaSize)()
	busy := now() - start
	for k := 0; k < 50; k++ {
		linear(metaSize)()
	}
	if d := now() - start; d <= busy {
		t.Errorf("busy work did not advance the clock: %v then %v", busy, d)
	}
}

func TestMeasurementsDoNotOverlap(t *testing.T) {
	var active, overlaps atomic.Int32
	work := func(n int) func() {
		f := linear(n)
		return func() {
			if active.Add(1) > 1 {
				overlaps.Add(1)
			}
			f()
			active.Add(-1)
		}
	}
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Measure(work(metaSize/Input/4), work(metaSize/4))
		}()
	}
	wg.Wait()
	if n := overlaps.Load(); n > 0 {
		t.Errorf("%d measured calls overlapped another measurement", n)
	}
}
