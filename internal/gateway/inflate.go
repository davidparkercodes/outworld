package gateway

import (
	"crypto/sha256"
	"strings"
	"sync"
	"unicode"

	"github.com/davidparkercodes/outworld/pkg/detect"
	"github.com/davidparkercodes/outworld/pkg/outworld"
)

const (
	oversizeCompressed = "compressed"
	compressedHint     = "This value is compressed data the gateway could not fully read (corrupt, encrypted, too deeply nested or over the inflate limit), so it was not forwarded. Send it uncompressed or remove it."
	inflatedMemoBytes  = 16 << 20
)

func (f *fields) scanInflated(msg int, path string, b []byte) {
	if len(b) < detect.InflateMinBytes {
		return
	}
	if f.inflate == nil {
		f.inflate = detect.NewInflateBudget()
	}
	parts, st := detect.Inflate(b, f.inflate)
	switch st {
	case detect.Unreadable:
		f.add(msg, path+"(inflate)", modeOversize, oversizeCompressed, nil)
	case detect.Inflated:
		for _, p := range parts {
			if textual(p) {
				f.inflatedText(msg, path+"(inflated)", p)
				continue
			}
			f.scanBytes(msg, path+"(inflated)", p, 1)
		}
	}
}

func (f *fields) inflatedText(msg int, path string, p []byte) {
	key := sha256.Sum256(p)
	leaves, ok := inflatedLeaves.get(key)
	if !ok {
		tmp := &fields{frames: map[frameKey]frame{}, decoded: map[string][]byte{}}
		tmp.textLeaves(0, "", modeDecoded, p)
		leaves = tmp.leaves
		inflatedLeaves.put(key, leaves)
	}
	for _, l := range leaves {
		f.leaves = append(f.leaves, leaf{msg: msg, path: f.place(msg, path+l.path), mode: l.mode, text: l.text, cuts: l.cuts})
	}
}

type leafMemo struct {
	mu    sync.Mutex
	m     map[[32]byte][]leaf
	order [][32]byte
	bytes int
}

func leafBytes(ls []leaf) int {
	n := 0
	for _, l := range ls {
		n += len(l.text)
	}
	return n
}

var inflatedLeaves = &leafMemo{m: map[[32]byte][]leaf{}}

func (c *leafMemo) get(k [32]byte) ([]leaf, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.m[k]
	return v, ok
}

func (c *leafMemo) put(k [32]byte, v []leaf) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.m[k]; ok {
		return
	}
	size := leafBytes(v)
	if size > inflatedMemoBytes {
		return
	}
	for c.bytes+size > inflatedMemoBytes && len(c.order) > 0 {
		c.bytes -= leafBytes(c.m[c.order[0]])
		delete(c.m, c.order[0])
		c.order = c.order[1:]
	}
	c.m[k] = v
	c.order = append(c.order, k)
	c.bytes += size
}

const (
	personLineMin     = 12
	personLineLetters = 6
)

func decodedPersonLine(rs []rune, v outworld.Violation) bool {
	if !strings.HasPrefix(v.Source, "names:") && !strings.HasPrefix(v.Source, "regex:") {
		return true
	}
	if strings.ContainsAny(string(rs[v.Start:v.End]), "\n\r") {
		return false
	}
	lo, hi := v.Start, v.End
	for lo > 0 && rs[lo-1] != '\n' && rs[lo-1] != '\r' {
		lo--
	}
	for hi < len(rs) && rs[hi] != '\n' && rs[hi] != '\r' {
		hi++
	}
	letters := 0
	for _, r := range rs[lo:hi] {
		if unicode.IsLetter(r) {
			letters++
		}
	}
	return hi-lo >= personLineMin && letters >= personLineLetters
}
