package gateway

import "unicode/utf8"

const (
	cborMajorBytes = 2
	cborMajorText  = 3
	cborMajorArray = 4
	cborMajorMap   = 5
	cborMajorTag   = 6
	cborMajorOther = 7
	cborIndefinite = 31
	cborBreak      = 0xff
	cborMaxItems   = 4096
	cborMaxDepth   = 64
	cborMaxTags    = 8
	cborHopsPerB   = 32
	cborHopsFloor  = 1 << 16
	cborFailed     = -1
)

type cborPair struct{ key, value string }

type cborWalk struct {
	b      []byte
	memo   []int32
	hops   int
	budget int
	over   bool
	pairs  []cborPair
}

func cborMapPairs(b []byte) ([]cborPair, bool) {
	w := &cborWalk{b: b, memo: make([]int32, len(b)+1), budget: cborHopsPerB*len(b) + cborHopsFloor}
	for p := range b {
		if cborMapHeader(b[p]) {
			w.item(p, 0)
		}
		if w.over {
			return w.pairs, false
		}
	}
	return w.pairs, true
}

func cborMapHeader(h byte) bool {
	return h>>5 == cborMajorMap && (h&31 <= 27 || h&31 == cborIndefinite)
}

func (w *cborWalk) head(p int) (major byte, arg uint64, next int, indef, ok bool) {
	if p >= len(w.b) {
		return 0, 0, 0, false, false
	}
	h := w.b[p]
	major, ai := h>>5, h&31
	switch {
	case ai < 24:
		return major, uint64(ai), p + 1, false, true
	case ai <= 27:
		n := 1 << (ai - 24)
		if p+1+n > len(w.b) {
			return 0, 0, 0, false, false
		}
		var v uint64
		for _, c := range w.b[p+1 : p+1+n] {
			v = v<<8 | uint64(c)
		}
		return major, v, p + 1 + n, false, true
	case ai == cborIndefinite && major >= cborMajorBytes && major <= cborMajorMap:
		return major, 0, p + 1, true, true
	}
	return 0, 0, 0, false, false
}

func (w *cborWalk) item(p, depth int) int {
	if p >= len(w.b) || depth > cborMaxDepth {
		return -1
	}
	switch m := w.memo[p]; {
	case m == cborFailed:
		return -1
	case m != 0:
		return int(m) - 1
	}
	if !w.step() {
		return -1
	}
	end := w.parse(p, depth)
	if w.over {
		return -1
	}
	if end < 0 {
		w.memo[p] = cborFailed
	} else {
		w.memo[p] = int32(end + 1)
	}
	return end
}

func (w *cborWalk) step() bool {
	w.hops++
	if w.hops > w.budget {
		w.over = true
	}
	return !w.over
}

func (w *cborWalk) parse(p, depth int) int {
	major, arg, next, indef, ok := w.head(p)
	if !ok {
		return -1
	}
	switch major {
	case 0, 1:
		return next
	case cborMajorBytes, cborMajorText:
		if indef {
			_, end, ok := w.chunks(next, major)
			if !ok {
				return -1
			}
			return end
		}
		if arg > uint64(len(w.b)-next) {
			return -1
		}
		end := next + int(arg)
		if major == cborMajorText && !utf8.Valid(w.b[next:end]) {
			return -1
		}
		return end
	case cborMajorArray, cborMajorMap:
		per := 1
		if major == cborMajorMap {
			per = 2
		}
		if !indef && arg > cborMaxItems {
			return -1
		}
		var items []int
		q := next
		for n := 0; indef || n < int(arg)*per; n++ {
			if indef && q < len(w.b) && w.b[q] == cborBreak {
				if len(items)%per != 0 {
					return -1
				}
				q++
				break
			}
			if len(items) >= cborMaxItems*per || !w.step() {
				return -1
			}
			e := w.item(q, depth+1)
			if e < 0 {
				return -1
			}
			items = append(items, q)
			q = e
		}
		if major == cborMajorMap {
			for i := 0; i+1 < len(items); i += 2 {
				key, kok := w.text(items[i])
				value, vok := w.text(items[i+1])
				if kok && vok && key != "" && len(key) <= cborMaxKey {
					w.pairs = append(w.pairs, cborPair{key, value})
				}
			}
		}
		return q
	case cborMajorTag:
		return w.item(next, depth+1)
	case cborMajorOther:
		if indef || arg > 27 || p == len(w.b) || w.b[p] == cborBreak {
			return -1
		}
		return next
	}
	return -1
}

func (w *cborWalk) chunks(q int, major byte) (string, int, bool) {
	var s []byte
	for n := 0; ; n++ {
		if q >= len(w.b) || n > cborMaxItems {
			return "", 0, false
		}
		if w.b[q] == cborBreak {
			if major == cborMajorText && !utf8.Valid(s) {
				return "", 0, false
			}
			return string(s), q + 1, true
		}
		m, arg, next, indef, ok := w.head(q)
		if !ok || m != major || indef || arg > uint64(len(w.b)-next) {
			return "", 0, false
		}
		s = append(s, w.b[next:next+int(arg)]...)
		q = next + int(arg)
	}
}

func (w *cborWalk) text(p int) (string, bool) {
	for range cborMaxTags {
		major, arg, next, indef, ok := w.head(p)
		if !ok {
			return "", false
		}
		switch {
		case major == cborMajorTag:
			p = next
			continue
		case major != cborMajorText && major != cborMajorBytes:
			return "", false
		case indef:
			s, _, ok := w.chunks(next, major)
			return s, ok && cborPrintable(s)
		}
		if arg > uint64(len(w.b)-next) {
			return "", false
		}
		s := string(w.b[next : next+int(arg)])
		return s, cborPrintable(s)
	}
	return "", false
}

func cborPrintable(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if !printableRune(r) {
			return false
		}
	}
	return true
}
