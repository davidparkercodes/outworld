package detect

import (
	"context"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

type panicky struct{}

func (panicky) Name() string { return "panicky" }

func (panicky) Detect(context.Context, string) ([]entity.Span, error) { panic("boom") }

func TestRunAllRecoversPanics(t *testing.T) {
	for _, n := range []int{64, 10 << 10} {
		text := strings.Repeat("a", n-len("j@x.io ")) + "j@x.io "
		ds := []Detector{NewRegexDetector(), panicky{}, &NERDetector{}}
		res := RunAll(context.Background(), text, ds)
		if len(res) != len(ds) {
			t.Fatalf("size %d: %d results", n, len(res))
		}
		if res[0].Err != nil {
			t.Fatalf("size %d: healthy detector failed: %v", n, res[0].Err)
		}
		for _, i := range []int{1, 2} {
			if res[i].Err == nil || res[i].Spans != nil {
				t.Fatalf("size %d: detector %d: %+v", n, i, res[i])
			}
		}
		if strings.Contains(res[1].Err.Error(), "boom") {
			t.Fatalf("panic value leaked into the error: %v", res[1].Err)
		}
	}
}

func TestRunAllSingleDetectorPanic(t *testing.T) {
	res := RunAll(context.Background(), "hello", []Detector{panicky{}})
	if len(res) != 1 || res[0].Err == nil {
		t.Fatalf("%+v", res)
	}
}
