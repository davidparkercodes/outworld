package outworld

import (
	"encoding/base64"
	"github.com/davidparkercodes/outworld/internal/scaletest"
	"math/rand"
	"strings"
	"testing"
)

func TestMaskKeepsRandomThenProseBlob(t *testing.T) {
	words := strings.Fields("the build finished and the release notes describe several small fixes to the parser cache layer")
	e := newEngine(t, nil)
	sizes := []int{2 << 10, 32 << 10, 115000, 512 << 10}
	if scaletest.Race {
		sizes = sizes[:3]
	}
	for _, size := range sizes {
		r := rand.New(rand.NewSource(int64(size)))
		raw := make([]byte, 4608)
		r.Read(raw)
		var b strings.Builder
		for b.Len() < size {
			b.WriteString(words[r.Intn(len(words))])
			b.WriteByte(' ')
		}
		v := base64.StdEncoding.EncodeToString(append(raw, b.String()...))
		for _, in := range []string{"see " + v, `{"data": "` + v + `"}`} {
			out, err := e.NewSession("prose").Mask(ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			if out != in {
				t.Errorf("size %d: clean random plus prose blob changed", size)
			}
		}
	}
}
