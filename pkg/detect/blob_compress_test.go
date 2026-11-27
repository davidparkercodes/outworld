package detect

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/flate"
	"encoding/binary"
	"encoding/hex"
	"hash/adler32"
	"hash/crc32"
	"math/rand"
	"testing"
)

const (
	bzip2PIIHex      = "425a6839314159265359c3411f9f00000b5d80001040037e5048002e37d44020004889927a40da6a1886c850340340002f741c4d06c0bc25157603a36ea4b67ff5afa35c37c72367d246404f3708825f4f8bb9229c284861a08fcf80"
	bzip2CleanHex    = "425a6839314159265359c3e3397200000fd980001042047ce03f679e002000488a3ca3d4d1834d4c13d4c534000000d112ed9703ccfc393da7159d4828a36af224d671761f2fb905971d4107fde4be37c9028444ee0aa07e2ee48a70a12187c672e4"
	tarBzip2PIIHex   = "425a6839314159265359e538084d0000835f81ca9040037fd0480000806e37de40002008082000741a4994fc94d30346a66a311e80d04a2819340080c8c1a01f2f4aa080c88404287d4d18e59101050ce0cfd2b5b72e26d63bda884ea043ecc420b7e0e842959b2a09ea7c4e9c6c9185bff75b31b9c550df8389a1f1cb13c4c24beaae4e8b1a9749de5226e936e791101d8bb9229c2848729c042680"
	tarBzip2CleanHex = "425a6839314159265359feb04697000089db82cc9042057de000807f679e40000208082000940d21953d329b28c87a47a990c4cf5068a0c4d320680680d0105fc6601009901004190a940d6a410040c07c4fe590ac3928522f76a3ba029c9a0855a42f76147e29f2a623658728fb47fd31945d725566ee835a61a080bcdf6d1bb326fc61fd1815873366f6a4564e976fd0d956ef44440702ee48a70a121fd608d2e0"
)

func unhex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func fillTo(b []byte, n int) []byte {
	var out []byte
	for len(out) < n {
		out = append(out, b...)
	}
	return out
}

func deflateRaw(b []byte) []byte {
	var o bytes.Buffer
	w, _ := flate.NewWriter(&o, 6)
	w.Write(b)
	w.Close()
	return o.Bytes()
}

func zlibCMF(b []byte, cmf byte, adler bool) []byte {
	flg := byte(0)
	for (uint(cmf)<<8|uint(flg))%31 != 0 {
		flg++
	}
	out := append([]byte{cmf, flg}, deflateRaw(b)...)
	if adler {
		out = binary.BigEndian.AppendUint32(out, adler32.Checksum(b))
	}
	return out
}

func gzipFHCRC(b []byte) []byte {
	out := []byte{0x1f, 0x8b, 0x08, 0x02, 0, 0, 0, 0, 0, 0xff, 0xde, 0xad}
	out = append(out, deflateRaw(b)...)
	out = binary.LittleEndian.AppendUint32(out, crc32.ChecksumIEEE(b))
	return binary.LittleEndian.AppendUint32(out, uint32(len(b)))
}

func tarTyped(flag byte, plain bool, data []byte) []byte {
	var o bytes.Buffer
	w := tar.NewWriter(&o)
	w.WriteHeader(&tar.Header{Name: "entry", Mode: 0o644, Size: int64(len(data)), Typeflag: flag, Format: tar.FormatGNU})
	w.Write(data)
	w.Close()
	if plain {
		return o.Bytes()
	}
	return gz(o.Bytes())
}

func localEntry(name string, data []byte) []byte {
	h := make([]byte, 30)
	copy(h, "PK\x03\x04")
	binary.LittleEndian.PutUint16(h[4:], 20)
	binary.LittleEndian.PutUint32(h[14:], crc32.ChecksumIEEE(data))
	binary.LittleEndian.PutUint32(h[18:], uint32(len(data)))
	binary.LittleEndian.PutUint32(h[22:], uint32(len(data)))
	binary.LittleEndian.PutUint16(h[26:], uint16(len(name)))
	return append(append(h, name...), data...)
}

func compressionRows(seed int64) (dirty, clean map[string][]byte) {
	r := rand.New(rand.NewSource(seed))
	c, d := logRows(r, 120, false), logRows(r, 120, true)
	bin := make([]byte, 64)
	r.Read(bin)
	sfx := make([]byte, 16)
	r.Read(sfx)
	pad := make([]byte, 1100)
	r.Read(pad)
	dirty = map[string][]byte{
		"gzip of gzip of zlib":        gz(gz(zl(d))),
		"zip of zip of zlib":          zipOf(zip.Deflate, zipFile{"a.zip", zipOf(zip.Deflate, zipFile{"b.z", zl(d)})}),
		"zlib of zlib of zlib":        zl(zl(zl(d))),
		"gzip of zip of zlib":         gz(zipOf(zip.Deflate, zipFile{"b.z", zl(d)})),
		"gzip fhcrc bad header crc":   gzipFHCRC(d),
		"zlib cmf 0x58":               zlibCMF(d, 0x58, true),
		"zlib cmf 0x28":               zlibCMF(d, 0x28, true),
		"zlib binary prefix no adler": zlibCMF(append(append([]byte(nil), bin...), d...), 0x78, false),
		"zlib binary prefix bad adler": func() []byte {
			b := zlibCMF(append(append([]byte(nil), bin...), d...), 0x78, true)
			b[len(b)-1] ^= 0xff
			return b
		}(),
		"zip sfx 16 byte prefix":           append(append([]byte(nil), sfx...), zipOf(zip.Deflate, zipFile{"a.txt", d})...),
		"zip hidden local entry":           append(localEntry("hidden.txt", d), zipOf(zip.Deflate, zipFile{"a.txt", c})...),
		"tar.gz typeflag 7":                tarTyped('7', false, d),
		"tar.gz typeflag Z":                tarTyped('Z', false, d),
		"tar.gz typeflag D":                tarTyped('D', false, d),
		"tar.gz typeflag V":                tarTyped('V', false, d),
		"tar.gz typeflag X":                tarTyped('X', false, d),
		"tar typeflag 7":                   tarTyped('7', true, d),
		"plain tar with gz entry":          tarOf(zipFile{"a.gz", gz(d)}),
		"plain tar with zlib entry":        tarOf(zipFile{"a.z", zl(d)}),
		"zip of tar.gz typeflag 7":         zipOf(zip.Store, zipFile{"a.tgz", tarTyped('7', false, d)}),
		"bzip2 pii":                        fillTo(append(unhex(bzip2PIIHex), unhex(bzip2CleanHex)...), 1100),
		"tar.bz2 pii":                      append(unhex(tarBzip2PIIHex), fillTo(unhex(tarBzip2CleanHex), 1000)...),
		"xz magic":                         append([]byte{0xfd, '7', 'z', 'X', 'Z', 0}, pad...),
		"zstd magic":                       append([]byte{0x28, 0xb5, 0x2f, 0xfd}, pad...),
		"7z magic":                         append([]byte{'7', 'z', 0xbc, 0xaf, 0x27, 0x1c}, pad...),
		"lzma alone header":                append([]byte{0x5d, 0, 0, 0x80, 0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, pad...),
		"gzip 2000 empty members then pii": append(bytes.Repeat(gz(nil), 2000), gz(d)...),
	}
	clean = map[string][]byte{
		"gzip fhcrc clean":              gzipFHCRC(fillTo(c, 3000)),
		"zlib cmf 0x58 clean":           zlibCMF(fillTo(c, 3000), 0x58, true),
		"zip sfx clean":                 append(append([]byte(nil), sfx...), zipOf(zip.Deflate, zipFile{"a.txt", fillTo(c, 3000)})...),
		"tar.gz typeflag 7 clean":       tarTyped('7', false, fillTo(c, 3000)),
		"plain tar with gz entry clean": tarOf(zipFile{"a.gz", gz(fillTo(c, 3000))}),
		"bzip2 clean":                   fillTo(unhex(bzip2CleanHex), 1100),
		"tar.bz2 clean":                 fillTo(unhex(tarBzip2CleanHex), 1100),
		"gzip of zlib clean":            gz(zl(fillTo(c, 3000))),
	}
	return dirty, clean
}

var unreadableRows = map[string]bool{
	"gzip of gzip of zlib": true, "zip of zip of zlib": true, "zlib of zlib of zlib": true, "gzip of zip of zlib": true,
	"zlib binary prefix no adler": true, "zlib binary prefix bad adler": true,
	"xz magic": true, "zstd magic": true, "7z magic": true, "lzma alone header": true,
}

func piiInParts(parts [][]byte) bool {
	for _, p := range parts {
		if !inflatedClean(p) {
			return true
		}
	}
	return false
}

func TestCompressionRows(t *testing.T) {
	for _, seed := range []int64{64, 65, 66} {
		dirty, clean := compressionRows(seed)
		for name, raw := range dirty {
			if len(raw) < cleanBlobMin {
				t.Fatalf("%s only %d bytes", name, len(raw))
			}
			parts, st := Inflate(raw, NewInflateBudgetFor(testLimits))
			switch {
			case unreadableRows[name] && st != Unreadable:
				t.Errorf("seed %d %s: status %d, want unreadable", seed, name, st)
			case !unreadableRows[name] && (st != Inflated || !piiInParts(parts)):
				t.Errorf("seed %d %s: status %d, personal data not found in the inflated parts", seed, name, st)
			}
			if !blobStripped(t, raw) {
				t.Errorf("seed %d %s: not stripped", seed, name)
			}
		}
		for name, raw := range clean {
			if len(raw) < cleanBlobMin {
				t.Fatalf("%s only %d bytes", name, len(raw))
			}
			if parts, st := Inflate(raw, NewInflateBudgetFor(testLimits)); st != Inflated || piiInParts(parts) {
				t.Errorf("seed %d %s: status %d on a clean archive", seed, name, st)
			}
			if blobStripped(t, raw) {
				t.Errorf("seed %d %s: clean archive stripped", seed, name)
			}
		}
	}
}

func TestRawDeflateResidualStaysRaw(t *testing.T) {
	r := rand.New(rand.NewSource(65))
	raw := deflateRaw(fillTo(logRows(r, 60, true), 4000))
	if _, st := Inflate(raw, NewInflateBudgetFor(testLimits)); st != NotCompressed {
		t.Errorf("raw deflate status %d; residual list is out of date", st)
	}
}

func TestRecognizedMagicRandomSafety(t *testing.T) {
	refused := map[string]int{}
	for _, size := range []int{1 << 10, 4 << 10, 48 << 10} {
		for seed := int64(0); seed < 2000; seed++ {
			if _, st := Inflate(randomDER(seed+int64(size), size), NewInflateBudgetFor(testLimits)); st == Unreadable {
				refused["random"]++
			}
		}
	}
	magics := map[string][]byte{"gzip": {0x1f, 0x8b, 0x08}, "zlib 78 9c": {0x78, 0x9c}, "zlib 78 01": {0x78, 0x01}, "zlib 58 85": {0x58, 0x85}, "zlib 28 9d": {0x28, 0x9d}, "zlib 08 1d": {0x08, 0x1d}, "bzip2 short": []byte("BZh9")}
	for name, m := range magics {
		for seed := int64(0); seed < 2000; seed++ {
			raw := randomDER(seed+50000, 4096)
			copy(raw, m)
			if _, st := Inflate(raw, NewInflateBudgetFor(testLimits)); st == Unreadable {
				refused[name]++
			}
		}
	}
	for name, n := range refused {
		if n != 0 {
			t.Errorf("%s: %d refusals", name, n)
		}
	}
}
