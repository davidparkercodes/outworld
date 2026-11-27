package detect

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"compress/flate"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
)

const (
	InflateMinBytes   = cleanBlobMin
	InflateBlobMax    = 1 << 20
	InflateRequestMax = 2 << 20
	InflateMaxEntries = 4096
	inflateNesting    = 1
	zlibTextVariety   = 12
	meaningfulRun     = 32
)

type InflateStatus int

const (
	NotCompressed InflateStatus = iota
	Inflated
	Unreadable
)

type InflateLimits struct {
	Blob    int
	Request int
}

func (l InflateLimits) withDefaults() InflateLimits {
	if l.Blob <= 0 {
		l.Blob = InflateBlobMax
	}
	if l.Request <= 0 {
		l.Request = InflateRequestMax
	}
	return l
}

type InflateBudget struct {
	Left int
	Blob int
}

func NewInflateBudget() *InflateBudget { return NewInflateBudgetFor(InflateLimits{}) }

func NewInflateBudgetFor(l InflateLimits) *InflateBudget {
	l = l.withDefaults()
	return &InflateBudget{Left: l.Request, Blob: l.Blob}
}

func (b *InflateBudget) blobMax() int {
	if b.Blob <= 0 {
		return InflateBlobMax
	}
	return b.Blob
}

var errInflateCap = errors.New("detect: inflate cap exceeded")

type inflater struct {
	left    int
	entries int
}

func Inflate(raw []byte, budget *InflateBudget) ([][]byte, InflateStatus) {
	in := &inflater{left: min(budget.blobMax(), budget.Left)}
	start := in.left
	parts, st := in.open(raw, inflateNesting)
	budget.Left -= start - in.left
	return parts, st
}

func (in *inflater) open(raw []byte, nest int) ([][]byte, InflateStatus) {
	parts, st := in.layer(raw)
	if st != Inflated {
		return nil, st
	}
	var out [][]byte
	for _, p := range parts {
		if nest == 0 {
			if _, s := in.layer(p); s != NotCompressed {
				return nil, Unreadable
			}
			out = append(out, p)
			continue
		}
		sub, s := in.open(p, nest-1)
		switch s {
		case Unreadable:
			return nil, Unreadable
		case Inflated:
			out = append(out, sub...)
		default:
			out = append(out, p)
		}
	}
	return out, Inflated
}

var sealedMagics = [][]byte{{0xfd, '7', 'z', 'X', 'Z', 0x00}, {0x28, 0xb5, 0x2f, 0xfd}, {'7', 'z', 0xbc, 0xaf, 0x27, 0x1c}}

func (in *inflater) layer(raw []byte) ([][]byte, InflateStatus) {
	for _, m := range sealedMagics {
		if bytes.HasPrefix(raw, m) {
			return nil, Unreadable
		}
	}
	var parts [][]byte
	var st InflateStatus
	switch {
	case lzmaAloneHeader(raw):
		return nil, Unreadable
	case bzip2Header(raw):
		parts, st = in.bzip2(raw)
	case bytes.HasPrefix(raw, []byte{0x1f, 0x8b, 0x08}):
		parts, st = in.gzip(raw)
	case zipHeader(raw):
		return in.zip(raw)
	case tarHeader(raw):
		return in.tarEntries([][]byte{raw})
	case zlibHeader(raw):
		parts, st = in.zlib(raw)
	default:
		return nil, NotCompressed
	}
	if st != Inflated {
		return nil, st
	}
	return in.tarEntries(parts)
}

func zlibHeader(b []byte) bool {
	return len(b) >= 2 && b[0]&0x0f == 8 && b[0]>>4 <= 7 && b[1]&0x20 == 0 && (uint(b[0])<<8|uint(b[1]))%31 == 0
}

func bzip2Header(b []byte) bool {
	return len(b) >= 10 && string(b[:3]) == "BZh" && b[3] >= '1' && b[3] <= '9' && bytes.Equal(b[4:10], []byte{0x31, 0x41, 0x59, 0x26, 0x53, 0x59})
}

func lzmaAloneHeader(b []byte) bool {
	if len(b) < 13 || b[0] >= 225 {
		return false
	}
	dict := binary.LittleEndian.Uint32(b[1:5])
	plausible := false
	for n := 12; n <= 30; n++ {
		if dict == 1<<n || dict == 1<<n+1<<(n-1) {
			plausible = true
		}
	}
	size := b[5:13]
	known := size[5] == 0 && size[6] == 0 && size[7] == 0
	unknown := bytes.Equal(size, bytes.Repeat([]byte{0xff}, 8))
	return plausible && (known || unknown)
}

func tarHeader(b []byte) bool {
	if len(b) < 512 {
		return false
	}
	if string(b[257:262]) == "ustar" {
		return true
	}
	want, ok := octalField(b[148:156])
	if !ok || b[0] == 0 {
		return false
	}
	sum := 0
	for i, c := range b[:512] {
		if i >= 148 && i < 156 {
			c = ' '
		}
		sum += int(c)
	}
	return sum == want
}

func octalField(f []byte) (int, bool) {
	n, digits := 0, 0
	for _, c := range f {
		switch {
		case c >= '0' && c <= '7':
			n = n*8 + int(c-'0')
			digits++
		case c == ' ' || c == 0:
			if digits > 0 {
				return n, true
			}
		default:
			return 0, false
		}
	}
	return n, digits > 0
}

func (in *inflater) read(r io.Reader) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, int64(in.left)+1))
	if len(b) > in.left {
		in.left = 0
		return nil, errInflateCap
	}
	in.left -= len(b)
	return b, err
}

func (in *inflater) entry() bool {
	in.entries++
	return in.entries <= InflateMaxEntries
}

func (in *inflater) stream(r io.Reader, strong bool) ([][]byte, InflateStatus) {
	b, err := in.read(r)
	switch {
	case err == nil:
		return [][]byte{b}, Inflated
	case strong || errors.Is(err, errInflateCap) || meaningfulText(b):
		return nil, Unreadable
	}
	return nil, NotCompressed
}

func meaningfulText(b []byte) bool {
	start, distinct := -1, map[byte]bool{}
	for i := 0; i <= len(b); i++ {
		if i < len(b) && asciiTextByte(b[i]) {
			if start < 0 {
				start = i
				distinct = map[byte]bool{}
			}
			distinct[b[i]] = true
			if i-start+1 >= meaningfulRun && len(distinct) >= zlibTextVariety {
				return true
			}
			continue
		}
		start = -1
	}
	return false
}

func (in *inflater) bzip2(raw []byte) ([][]byte, InflateStatus) {
	return in.stream(bzip2.NewReader(bytes.NewReader(raw)), true)
}

func (in *inflater) zlib(raw []byte) ([][]byte, InflateStatus) {
	r, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, NotCompressed
	}
	return in.stream(r, false)
}

func gzipHeaderLen(b []byte) (int, bool) {
	if len(b) < 10 || b[0] != 0x1f || b[1] != 0x8b || b[2] != 8 || b[3]&0xe0 != 0 {
		return 0, false
	}
	flg, i := b[3], 10
	if flg&4 != 0 {
		if len(b) < i+2 {
			return 0, false
		}
		i += 2 + int(binary.LittleEndian.Uint16(b[i:]))
	}
	for _, bit := range []byte{8, 16} {
		if flg&bit == 0 {
			continue
		}
		end := bytes.IndexByte(b[min(i, len(b)):], 0)
		if end < 0 {
			return 0, false
		}
		i += end + 1
	}
	if flg&2 != 0 {
		i += 2
	}
	return i, i <= len(b)
}

func plausibleGzip(b []byte) bool {
	return (b[8] == 0 || b[8] == 2 || b[8] == 4) && (b[9] <= 13 || b[9] == 255)
}

func (in *inflater) gzip(raw []byte) ([][]byte, InflateStatus) {
	h, ok := gzipHeaderLen(raw)
	if !ok {
		if plausibleGzip(raw[:min(len(raw), 10)]) && len(raw) >= 10 {
			return nil, Unreadable
		}
		return nil, NotCompressed
	}
	strong := plausibleGzip(raw)
	var out []byte
	rest := raw
	for member := 0; ; member++ {
		if member > 0 {
			if len(rest) == 0 {
				break
			}
			if h, ok = gzipHeaderLen(rest); !ok {
				return nil, Unreadable
			}
		}
		br := bytes.NewReader(rest[h:])
		parts, st := in.stream(flate.NewReader(br), strong || member > 0)
		if st != Inflated {
			return nil, st
		}
		end := len(rest) - br.Len()
		if len(rest) < end+8 || crc32.ChecksumIEEE(parts[0]) != binary.LittleEndian.Uint32(rest[end:]) {
			if strong || member > 0 || meaningfulText(parts[0]) {
				return nil, Unreadable
			}
			return nil, NotCompressed
		}
		out = append(out, parts[0]...)
		rest = rest[end+8:]
	}
	return [][]byte{out}, Inflated
}

func zipHeader(b []byte) bool {
	if bytes.HasPrefix(b, []byte("PK\x03\x04")) {
		return true
	}
	tail := b[max(0, len(b)-(64<<10)-22):]
	return bytes.Contains(tail, []byte("PK\x05\x06"))
}

func (in *inflater) zip(raw []byte) ([][]byte, InflateStatus) {
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		if zipLocalHeader(raw) {
			return nil, Unreadable
		}
		return nil, NotCompressed
	}
	var out [][]byte
	seen := map[int64]bool{}
	for _, f := range zr.File {
		if !in.entry() {
			return nil, Unreadable
		}
		if off, err := f.DataOffset(); err == nil {
			seen[off] = true
		}
		if f.Mode().IsDir() {
			continue
		}
		if f.Flags&1 != 0 || f.Method != zip.Store && f.Method != zip.Deflate {
			return nil, Unreadable
		}
		rc, err := f.Open()
		if err != nil {
			return nil, Unreadable
		}
		b, err := in.read(rc)
		rc.Close()
		if err != nil {
			return nil, Unreadable
		}
		out = append(out, b)
	}
	hidden, ok := in.hiddenLocalEntries(raw, seen)
	if !ok {
		return nil, Unreadable
	}
	return append(out, hidden...), Inflated
}

func (in *inflater) hiddenLocalEntries(raw []byte, seen map[int64]bool) ([][]byte, bool) {
	var out [][]byte
	off := 0
	for off+30 <= len(raw) && string(raw[off:off+4]) == "PK\x03\x04" {
		h := raw[off:]
		flags := binary.LittleEndian.Uint16(h[6:])
		method := binary.LittleEndian.Uint16(h[8:])
		size := int(binary.LittleEndian.Uint32(h[18:]))
		data := off + 30 + int(binary.LittleEndian.Uint16(h[26:])) + int(binary.LittleEndian.Uint16(h[28:]))
		if data > len(raw) {
			return nil, false
		}
		if flags&8 != 0 && size == 0 {
			if seen[int64(data)] {
				return out, true
			}
			return nil, false
		}
		if data+size > len(raw) {
			return nil, false
		}
		if !seen[int64(data)] {
			if !in.entry() || flags&1 != 0 {
				return nil, false
			}
			var r io.Reader = bytes.NewReader(raw[data : data+size])
			switch method {
			case zip.Store:
			case zip.Deflate:
				r = flate.NewReader(r)
			default:
				return nil, false
			}
			b, err := in.read(r)
			if err != nil {
				return nil, false
			}
			out = append(out, b)
		}
		off = data + size
		if flags&8 != 0 {
			if off+4 <= len(raw) && string(raw[off:off+4]) == "PK\x07\x08" {
				off += 4
			}
			off += 12
		}
	}
	return out, true
}

func zipLocalHeader(b []byte) bool {
	if len(b) < 30 || string(b[:4]) != "PK\x03\x04" {
		return false
	}
	method := uint16(b[8]) | uint16(b[9])<<8
	n := int(b[26]) | int(b[27])<<8
	if method != zip.Store && method != zip.Deflate || n == 0 || n > 255 || 30+n > len(b) {
		return false
	}
	for _, c := range b[30 : 30+n] {
		if c < 0x20 || c > 0x7e {
			return false
		}
	}
	return true
}

func (in *inflater) tarEntries(parts [][]byte) ([][]byte, InflateStatus) {
	var out [][]byte
	for _, p := range parts {
		if !tarHeader(p) {
			out = append(out, p)
			continue
		}
		tr := tar.NewReader(bytes.NewReader(p))
		left := int64(len(p))
		for {
			h, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil || !in.entry() {
				return nil, Unreadable
			}
			switch h.Typeflag {
			case tar.TypeDir, tar.TypeSymlink, tar.TypeLink:
				continue
			}
			b, err := io.ReadAll(io.LimitReader(tr, left+1))
			if err != nil || int64(len(b)) > left {
				return nil, Unreadable
			}
			left -= int64(len(b))
			if len(b) > 0 {
				out = append(out, b)
			}
		}
	}
	return out, Inflated
}
