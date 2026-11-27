package detect

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/namedata"
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

const bitmapResolveMin = 64

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
	var taken []bool
	if len(cands) > bitmapResolveMin {
		maxEnd := 0
		for _, c := range cands {
			if c.Start < 0 {
				maxEnd = -1
				break
			}
			maxEnd = max(maxEnd, c.End)
		}
		if maxEnd >= 0 {
			taken = make([]bool, maxEnd)
		}
	}
	for _, c := range cands {
		if c.Len() <= 0 {
			continue
		}
		clash := false
		if taken != nil {
			for i := c.Start; i < c.End; i++ {
				if taken[i] {
					clash = true
					break
				}
			}
		} else {
			for _, o := range out {
				if c.Overlaps(o) {
					clash = true
					break
				}
			}
		}
		if !clash {
			out = append(out, c)
			for i := c.Start; taken != nil && i < c.End; i++ {
				taken[i] = true
			}
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
	out := runAll(ctx, text, ds)
	if !needsFold(text) {
		return out
	}
	var idx []int
	var rest []Detector
	for i, d := range ds {
		if foldsItself(d) || out[i].Err != nil {
			continue
		}
		idx = append(idx, i)
		rest = append(rest, d)
	}
	if len(rest) == 0 {
		return out
	}
	folded, starts, ends := foldText(text)
	for j, r := range runAll(ctx, folded, rest) {
		i := idx[j]
		if r.Err != nil {
			out[i] = Result{Err: r.Err}
			continue
		}
		seen := map[[2]int]entity.Type{}
		for _, sp := range out[i].Spans {
			seen[[2]int{sp.Start, sp.End}] = sp.Type
		}
		for _, sp := range r.Spans {
			if !foldSpanOK(folded, sp.Start, sp.End) {
				continue
			}
			s, e := starts[sp.Start], ends[sp.End-1]
			if t, ok := seen[[2]int{s, e}]; ok && t == sp.Type {
				continue
			}
			if sp.Type == entity.Person && !dashJoinedName(text[s:e]) {
				continue
			}
			seen[[2]int{s, e}] = sp.Type
			sp.Start, sp.End, sp.Text = s, e, text[s:e]
			out[i].Spans = append(out[i].Spans, sp)
		}
	}
	return out
}

func dashJoinedName(span string) bool {
	for i := 0; i < len(span); {
		r, size := utf8.DecodeRuneInString(span[i:])
		if !textutil.IsDashVariant(r) || textutil.IsHyphenVariant(r) {
			i += size
			continue
		}
		before, after := lettersBefore(span, i), lettersAfter(span, i+size)
		if !dashNamePair(before, after) {
			return false
		}
		i += size
	}
	return true
}

func lettersBefore(s string, i int) string {
	j := i
	for j > 0 {
		r, n := utf8.DecodeLastRuneInString(s[:j])
		if !unicode.IsLetter(r) {
			break
		}
		j -= n
	}
	return s[j:i]
}

func lettersAfter(s string, i int) string {
	j := i
	for j < len(s) {
		r, n := utf8.DecodeRuneInString(s[j:])
		if !unicode.IsLetter(r) {
			break
		}
		j += n
	}
	return s[i:j]
}

func dashNamePair(before, after string) bool {
	if after == "" || before == "" {
		return false
	}
	if r, _ := utf8.DecodeRuneInString(after); !unicode.IsUpper(r) {
		return false
	}
	if textutil.IsAllUpper(after) && utf8.RuneCountInString(after) > 1 && !textutil.IsAllUpper(before) {
		return false
	}
	if textutil.IsCommonWord(after) {
		return false
	}
	f := textutil.Fold(after)
	if _, ok := namedata.LookupGiven(f); ok {
		return true
	}
	_, ok := namedata.LookupSurname(f)
	return ok
}

func foldsItself(d Detector) bool {
	switch d.(type) {
	case *RegexDetector, DateDetector, *DateDetector, *NERDetector:
		return true
	}
	return false
}

func runAll(ctx context.Context, text string, ds []Detector) []Result {
	out := make([]Result, len(ds))
	if len(ds) == 1 || len(text) < parallelMin {
		for i, d := range ds {
			out[i] = runOne(ctx, text, d)
		}
		return out
	}
	var wg sync.WaitGroup
	for i, d := range ds {
		wg.Add(1)
		go func(i int, d Detector) {
			defer wg.Done()
			out[i] = runOne(ctx, text, d)
		}(i, d)
	}
	wg.Wait()
	return out
}

func runOne(ctx context.Context, text string, d Detector) (r Result) {
	defer func() {
		if recover() != nil {
			r = Result{Err: fmt.Errorf("detect: detector %s panicked", detectorName(d))}
		}
	}()
	r.Spans, r.Err = d.Detect(ctx, text)
	return r
}

func detectorName(d Detector) (name string) {
	defer func() {
		if recover() != nil {
			name = fmt.Sprintf("%T", d)
		}
	}()
	return d.Name()
}
