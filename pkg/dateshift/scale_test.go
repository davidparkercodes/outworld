package dateshift

import (
	"fmt"
	"github.com/davidparkercodes/outworld/internal/scaletest"
	"strings"
	"testing"
)

func denseDates(size int) string {
	var b strings.Builder
	for i := 0; b.Len() < size; i++ {
		fmt.Fprintf(&b, "on %d/%d and may %d ", 1+i%12, 1+i%28, 1+i%28)
	}
	return b.String()
}

func countSteps(text string) int {
	_, strict := findCounted(text, MDY, false)
	_, loose := findCounted(text, MDY, true)
	return strict + loose
}

func TestDenseFindScalesLinearly(t *testing.T) {
	small, large := countSteps(denseDates(scaletest.Size(64<<10))), countSteps(denseDates(scaletest.Size(512<<10)))
	t.Logf("resolve and settle steps: 64 KiB %d, 512 KiB %d", small, large)
	if small == 0 || large > 12*small {
		t.Errorf("date finding steps grow faster than linear: %d then %d", small, large)
	}
}
