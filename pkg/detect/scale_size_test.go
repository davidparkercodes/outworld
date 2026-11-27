package detect

import (
	"testing"

	"github.com/davidparkercodes/outworld/internal/scaletest"
)

const (
	scaleGrowthSize = 128 << 10
	nameGrowthSize  = 64 << 10
)

func checkGrowth(t *testing.T, name string, build func(size int) string, size int, run func(string)) {
	t.Helper()
	scaletest.Check(t, name, size, func(n int) func() {
		text := build(n)
		return func() { run(text) }
	})
}
