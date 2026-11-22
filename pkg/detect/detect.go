package detect

import (
	"context"
	"sort"
	"sync"

	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/entity"
)

type Detector interface {
	Name() string
	Detect(ctx context.Context, text string) ([]entity.Span, error)
}

type Chain []Detector

func (c Chain) Detect(ctx context.Context, text string) ([]entity.Span, error) {
	var all []entity.Span
	for _, d := range c {
		spans, err := d.Detect(ctx, text)
		if err != nil {
			return nil, err
		}
		all = append(all, spans...)
	}
	return all, nil
}

func (c Chain) Name() string { return "chain" }

func Resolve(spans []entity.Span) []entity.Span {
	if len(spans) == 0 {
		return nil
	}
	cands := make([]entity.Span, len(spans))
	copy(cands, spans)
	sort.SliceStable(cands, func(i, j int) bool {
		a, b := cands[i], cands[j]
		if a.Len() != b.Len() {
			return a.Len() > b.Len()
		}
		if a.Priority != b.Priority {
			return a.Priority > b.Priority
		}
		return a.Start < b.Start
	})
	var out []entity.Span
	for _, c := range cands {
		if c.Len() <= 0 {
			continue
		}
		clash := false
		for _, o := range out {
			if c.Overlaps(o) {
				clash = true
				break
			}
		}
		if !clash {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	return out
}

func boundaryOK(text string, start, end int) bool {
	if start < 0 || end > len(text) || start >= end {
		return false
	}
	if textutil.IsWordRune(textutil.RuneAt(text, start)) && !textutil.IsBoundaryBefore(text, start) {
		return false
	}
	if textutil.IsWordRune(textutil.RuneBefore(text, end)) && !textutil.IsBoundaryAfter(text, end) {
		return false
	}
	return true
}

type Result struct {
	Spans []entity.Span
	Err   error
}

func RunAll(ctx context.Context, text string, ds []Detector) []Result {
	out := make([]Result, len(ds))
	if len(ds) == 1 || len(text) < parallelMin {
		for i, d := range ds {
			out[i].Spans, out[i].Err = d.Detect(ctx, text)
		}
		return out
	}
	var wg sync.WaitGroup
	for i, d := range ds {
		wg.Add(1)
		go func(i int, d Detector) {
			defer wg.Done()
			out[i].Spans, out[i].Err = d.Detect(ctx, text)
		}(i, d)
	}
	wg.Wait()
	return out
}
