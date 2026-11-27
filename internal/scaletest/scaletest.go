package scaletest

import (
	"runtime"
	"sync"
	"time"
)

var (
	wallStart = time.Now()
	measuring sync.Mutex
)

func wallNow() time.Duration { return time.Since(wallStart) }

const (
	Input      = 8
	Limit      = 16.0
	runs       = 5
	maxRounds  = 6
	timerFloor = 100 * time.Microsecond
	batchMin   = 20 * time.Millisecond
	batchMax   = 256
	raceDivide = 16
)

func Size(n int) int {
	if Race {
		return n / raceDivide
	}
	return n
}

type Reporter interface {
	Helper()
	Logf(format string, args ...any)
	Errorf(format string, args ...any)
}

type Result struct {
	OK           bool
	Ratio        float64
	Small, Large time.Duration
	Batch        int
	Rounds       int
}

func batchFor(small func()) int {
	start := now()
	small()
	one := now() - start
	if one >= batchMin {
		return 1
	}
	return min(int(batchMin/max(one, time.Microsecond))+1, batchMax)
}

func once(small, large func()) (time.Duration, time.Duration, int) {
	batch, n := Input, 1
	if !Race {
		batch, n = max(batchFor(small), Input), runs
	}
	ts, tl := time.Duration(1<<62), time.Duration(1<<62)
	under := 0
	for i := 0; i < n; i++ {
		runtime.GC()
		start := now()
		for k := 0; k < batch; k++ {
			small()
		}
		rs := (now() - start) / time.Duration(batch)
		ts = min(ts, rs)
		runtime.GC()
		start = now()
		large()
		rl := now() - start
		tl = min(tl, rl)
		if float64(rl) <= float64(max(rs, timerFloor))*Limit*3/4 {
			under++
		} else {
			under = 0
		}
		if i >= 2 && under >= 2 {
			break
		}
	}
	return ts, tl, batch
}

func Measure(small, large func()) Result {
	measuring.Lock()
	defer measuring.Unlock()
	best := Result{Small: time.Duration(1 << 62), Large: time.Duration(1 << 62)}
	for round := 1; round <= maxRounds; round++ {
		ts, tl, batch := once(small, large)
		best.Small, best.Large, best.Batch, best.Rounds = min(best.Small, ts), min(best.Large, tl), batch, round
		best.Ratio = float64(best.Large) / float64(max(best.Small, timerFloor))
		best.OK = best.Ratio <= Limit
		if best.OK || Race {
			break
		}
	}
	return best
}

func Check(t Reporter, name string, size int, prepare func(n int) func()) Result {
	t.Helper()
	r := Measure(prepare(size/Input), prepare(size))
	t.Logf("%-40s size %8d: small %v (x%d), large %v, growth %.1fx in %d rounds", name, size, r.Small.Round(time.Microsecond), r.Batch, r.Large.Round(time.Millisecond), r.Ratio, r.Rounds)
	if !Race && !r.OK {
		t.Errorf("%s: %dx the input took %.1fx the time (limit %.0fx)", name, Input, r.Ratio, Limit)
	}
	return r
}

func CPU() time.Duration { return now() }
