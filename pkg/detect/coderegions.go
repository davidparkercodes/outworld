package detect

import (
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

type codeRegions struct {
	fences [][2]int
	spans  [][2]int
}

var regionCache struct {
	mu   sync.Mutex
	text string
	r    *codeRegions
}

func regionsFor(text string) *codeRegions {
	regionCache.mu.Lock()
	if regionCache.r != nil && len(regionCache.text) == len(text) && regionCache.text == text {
		r := regionCache.r
		regionCache.mu.Unlock()
		return r
	}
	regionCache.mu.Unlock()
	r := scanCodeRegions(text)
	regionCache.mu.Lock()
	regionCache.text, regionCache.r = text, r
	regionCache.mu.Unlock()
	return r
}

func inRegion(rs [][2]int, pos int) bool {
	i := sort.Search(len(rs), func(i int) bool { return rs[i][1] > pos })
	return i < len(rs) && rs[i][0] <= pos
}

func fenceRun(line string) (byte, int, int) {
	i := 0
	for i < len(line) && i < 3 && line[i] == ' ' {
		i++
	}
	if i >= len(line) || (line[i] != '`' && line[i] != '~') {
		return 0, 0, 0
	}
	c, j := line[i], i
	for j < len(line) && line[j] == c {
		j++
	}
	if j-i < 3 {
		return 0, 0, 0
	}
	return c, j - i, j
}

func blankLine(line string) bool {
	for i := 0; i < len(line); i++ {
		if line[i] != ' ' && line[i] != '\t' && line[i] != '\r' {
			return false
		}
	}
	return true
}

func scanCodeRegions(text string) *codeRegions {
	r := &codeRegions{}
	var fc byte
	fl, fstart, para := 0, -1, 0
	for pos := 0; pos <= len(text); {
		eol := pos
		for eol < len(text) && text[eol] != '\n' {
			eol++
		}
		line := text[pos:eol]
		next := eol + 1
		c, n, k := fenceRun(line)
		switch {
		case fstart >= 0:
			if c == fc && n >= fl && blankLine(line[k:]) {
				r.fences = append(r.fences, [2]int{fstart, min(next, len(text))})
				fstart, para = -1, min(next, len(text))
			}
		case c != 0 && !(c == '`' && containsByte(line[k:], '`')):
			r.spans = appendSpans(r.spans, text, para, pos)
			fc, fl, fstart = c, n, pos
		case blankLine(line):
			r.spans = appendSpans(r.spans, text, para, pos)
			para = min(next, len(text))
		}
		if eol >= len(text) {
			break
		}
		pos = next
	}
	if fstart < 0 {
		r.spans = appendSpans(r.spans, text, para, len(text))
	}
	return r
}

func containsByte(s string, b byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return true
		}
	}
	return false
}

func lineLeading(text string, i int) bool {
	for k := i - 1; k >= 0 && text[k] != '\n'; k-- {
		if text[k] != ' ' && text[k] != '\t' {
			return false
		}
	}
	return true
}

func wordBeforeTick(text string, i int) bool {
	if i == 0 {
		return false
	}
	r, _ := utf8.DecodeLastRuneInString(text[:i])
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func escapedAt(text string, i int) bool {
	n := 0
	for k := i - 1; k >= 0 && text[k] == '\\'; k-- {
		n++
	}
	return n%2 == 1
}

func quoteOpener(text string, n, j int) bool {
	if n > 2 {
		return false
	}
	k := j
	for k < len(text) && k-j < wordScanLimit {
		r, size := utf8.DecodeRuneInString(text[k:])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			break
		}
		k += size
	}
	if k == j || k >= len(text) {
		return false
	}
	if text[k] == '\'' || strings.HasPrefix(text[k:], "\u2019") {
		return true
	}
	for m := k; m < len(text) && m-j < 4*wordScanLimit && text[m] != '`' && text[m] != '\n'; {
		r, size := utf8.DecodeRuneInString(text[m:])
		if (r == '\'' || r == '\u2019') && m > 0 {
			p, _ := utf8.DecodeLastRuneInString(text[:m])
			if unicode.IsLetter(p) || unicode.IsDigit(p) {
				return true
			}
		}
		m += size
	}
	return false
}

func regionAt(rs [][2]int, pos int) (int, int, bool) {
	i := sort.Search(len(rs), func(i int) bool { return rs[i][1] > pos })
	if i < len(rs) && rs[i][0] <= pos {
		return rs[i][0], rs[i][1], true
	}
	return 0, 0, false
}

func apostropheTick(text string, i, n int) bool {
	if n != 1 || i == 0 || i+1 >= len(text) {
		return false
	}
	before, _ := utf8.DecodeLastRuneInString(text[:i])
	after, _ := utf8.DecodeRuneInString(text[i+1:])
	return unicode.IsLetter(before) && unicode.IsLetter(after)
}

func appendSpans(out [][2]int, text string, lo, hi int) [][2]int {
	if lo >= hi {
		return out
	}
	type run struct{ start, n int }
	var runs []run
	for i := lo; i < hi; {
		if text[i] != '`' {
			i++
			continue
		}
		j := i
		for j < hi && text[j] == '`' {
			j++
		}
		if !apostropheTick(text, i, j-i) && !(j-i >= 3 && lineLeading(text, i)) && !escapedAt(text, i) && !quoteOpener(text, j-i, j) {
			runs = append(runs, run{i, j - i})
		}
		i = j
	}
	nextSame := make([]int, len(runs))
	last := map[int]int{}
	for i := len(runs) - 1; i >= 0; i-- {
		nextSame[i] = -1
		if j, ok := last[runs[i].n]; ok {
			nextSame[i] = j
		}
		last[runs[i].n] = i
	}
	for i := 0; i < len(runs); {
		j := nextSame[i]
		if j < 0 || wordBeforeTick(text, runs[i].start) {
			i++
			continue
		}
		out = append(out, [2]int{runs[i].start, runs[j].start + runs[j].n})
		i = j + 1
	}
	return out
}
