package detect

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"math/rand"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
)

const inflatePII = "Contact: call 804-555-1234 or mail jane.doe@example.com\n"

func logRows(r *rand.Rand, n int, pii bool) []byte {
	var b bytes.Buffer
	for i := 0; i < n; i++ {
		if pii && i == n/2 {
			b.WriteString(inflatePII)
			continue
		}
		fmt.Fprintf(&b, "build %d finished in %d ms, cache hit ratio %d%%, artefact %08x\n", i, r.Intn(9999), r.Intn(100), r.Uint32())
	}
	return b.Bytes()
}

func gz(b []byte) []byte {
	var out bytes.Buffer
	w := gzip.NewWriter(&out)
	w.Write(b)
	w.Close()
	return out.Bytes()
}

func zl(b []byte) []byte {
	var out bytes.Buffer
	w := zlib.NewWriter(&out)
	w.Write(b)
	w.Close()
	return out.Bytes()
}

type zipFile struct {
	name string
	data []byte
}

func zipOf(method uint16, files ...zipFile) []byte {
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	for _, f := range files {
		fw, _ := w.CreateHeader(&zip.FileHeader{Name: f.name, Method: method})
		fw.Write(f.data)
	}
	w.Close()
	return out.Bytes()
}

func patchZip(b []byte, patch func(local bool, h []byte)) []byte {
	b = append([]byte(nil), b...)
	for i := 0; i+4 <= len(b); i++ {
		switch string(b[i : i+4]) {
		case "PK\x03\x04":
			patch(true, b[i:])
		case "PK\x01\x02":
			patch(false, b[i:])
		}
	}
	return b
}

func encryptedZip(b []byte) []byte {
	return patchZip(b, func(local bool, h []byte) {
		at := 6
		if !local {
			at = 8
		}
		binary.LittleEndian.PutUint16(h[at:], binary.LittleEndian.Uint16(h[at:])|1)
	})
}

func methodZip(b []byte, method uint16) []byte {
	return patchZip(b, func(local bool, h []byte) {
		at := 8
		if !local {
			at = 10
		}
		binary.LittleEndian.PutUint16(h[at:], method)
	})
}

func tarOf(files ...zipFile) []byte {
	var out bytes.Buffer
	w := tar.NewWriter(&out)
	for _, f := range files {
		w.WriteHeader(&tar.Header{Name: f.name, Mode: 0o644, Size: int64(len(f.data)), Typeflag: tar.TypeReg})
		w.Write(f.data)
	}
	w.Close()
	return out.Bytes()
}

func utf16le(s string) []byte {
	var b []byte
	for _, u := range utf16.Encode([]rune(s)) {
		b = binary.LittleEndian.AppendUint16(b, u)
	}
	return b
}

func utf16be(s string) []byte {
	var b []byte
	for _, u := range utf16.Encode([]rune(s)) {
		b = binary.BigEndian.AppendUint16(b, u)
	}
	return b
}

func randomPNG(r *rand.Rand, side int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, side, side))
	r.Read(img.Pix)
	var b bytes.Buffer
	png.Encode(&b, img)
	return b.Bytes()
}

func withMagic(r *rand.Rand, magic string) []byte {
	b := make([]byte, 4096)
	r.Read(b)
	copy(b, magic)
	return b
}

type blobRow struct {
	name string
	raw  []byte
}

func inflateRows() (clean, dirty []blobRow) {
	r := rand.New(rand.NewSource(54))
	c, d := logRows(r, 300, false), logRows(r, 300, true)
	many := func(dirtyAt int) []zipFile {
		var fs []zipFile
		for i := 0; i < 300; i++ {
			data := []byte(fmt.Sprintf("ok %d\n", i))
			if i == dirtyAt {
				data = d
			}
			fs = append(fs, zipFile{fmt.Sprintf("f%d.txt", i), data})
		}
		return fs
	}
	var bomb bytes.Buffer
	bomb.Write(make([]byte, 20<<20))
	bomb.WriteString(inflatePII)
	clean = []blobRow{
		{"gzip clean log", gz(c)},
		{"zip deflate clean log", zipOf(zip.Deflate, zipFile{"log.txt", c})},
		{"zip stored clean log", zipOf(zip.Store, zipFile{"log.txt", c})},
		{"zlib clean log", zl(c)},
		{"gzip repetitive 2 MB log", gz(bytes.Repeat([]byte("INFO request ok path=/health status=200\n"), 50000))},
		{"zip 300 clean files", zipOf(zip.Deflate, many(-1)...)},
		{"zip in zip clean", zipOf(zip.Deflate, zipFile{"inner.zip", zipOf(zip.Deflate, zipFile{"a.txt", c})})},
		{"gzip in zip clean", zipOf(zip.Deflate, zipFile{"a.txt.gz", gz(c)})},
		{"tar.gz clean", gz(tarOf(zipFile{"a.txt", c}, zipFile{"b.txt", c[:1000]}))},
		{"random 4 KB with gzip magic", withMagic(r, "\x1f\x8b")},
		{"random 4 KB with zlib magic", withMagic(r, "\x78\x9c")},
		{"random 4 KB with zip magic", withMagic(r, "PK\x03\x04")},
		{"png", randomPNG(r, 48)},
	}
	dirty = []blobRow{
		{"gzip pii row", gz(d)},
		{"zip deflate pii", zipOf(zip.Deflate, zipFile{"a.txt", d})},
		{"zip stored pii", zipOf(zip.Store, zipFile{"a.txt", d})},
		{"zlib pii", zl(d)},
		{"zip pii in entry 200 of 250", zipOf(zip.Deflate, many(199)[:250]...)},
		{"zip in zip pii", zipOf(zip.Deflate, zipFile{"inner.zip", zipOf(zip.Deflate, zipFile{"a.txt", d})})},
		{"gzip in zip pii", zipOf(zip.Deflate, zipFile{"a.txt.gz", gz(d)})},
		{"zip in zip in zip", zipOf(zip.Deflate, zipFile{"l1.zip", zipOf(zip.Deflate, zipFile{"l2.zip", zipOf(zip.Deflate, zipFile{"a.txt", c})})})},
		{"gzip of gzip of gzip", gz(gz(gz(c)))},
		{"zip encrypted flag", encryptedZip(zipOf(zip.Deflate, zipFile{"a.txt", c}))},
		{"zip bzip2 method", methodZip(zipOf(zip.Store, zipFile{"a.txt", c}), 12)},
		{"truncated gzip", gz(d)[:len(gz(d))/2]},
		{"corrupt deflate gzip", func() []byte { b := gz(d); b[40] ^= 0xff; return b }()},
		{"gzip bomb 20 MB zeros plus pii", gz(bomb.Bytes())},
		{"pii after 9 MiB of text", gz(append(logRows(r, 160000, false), inflatePII...))},
		{"multi-member gzip pii in member 2", append(gz(c), gz(bytes.Repeat([]byte(inflatePII), 30))...)},
		{"gzip utf-16le pii", gz(utf16le(string(d)))},
		{"tar.gz pii", gz(tarOf(zipFile{"a.txt", c}, zipFile{"b.txt", d}))},
	}
	return clean, dirty
}

func blobVerdict(raw []byte) (clean, ok bool) {
	v := base64.StdEncoding.EncodeToString(raw)
	b, ok := binaryBlob(v)
	if !ok {
		return false, false
	}
	return blobClean(b, NewInflateBudgetFor(testLimits)), true
}

func TestInflateAcceptanceRows(t *testing.T) {
	clean, dirty := inflateRows()
	for _, row := range clean {
		v := base64.StdEncoding.EncodeToString(row.raw)
		if c, ok := blobVerdict(row.raw); !ok || !c {
			t.Errorf("%s: want clean blob, got clean=%v blob=%v", row.name, c, ok)
		}
		if got := secretSpans(t, "see "+v); len(got) != 0 {
			t.Errorf("%s: clean blob flagged by %s", row.name, got[0].Source)
		}
	}
	for _, row := range dirty {
		v := base64.StdEncoding.EncodeToString(row.raw)
		if c, ok := blobVerdict(row.raw); !ok || c {
			t.Errorf("%s: want not clean, got clean=%v blob=%v", row.name, c, ok)
		}
		for _, text := range []string{"see " + v, "see " + strings.TrimRight(strings.NewReplacer("+", "-", "/", "_").Replace(v), "=")} {
			whole := false
			for _, sp := range secretSpans(t, text) {
				whole = whole || sp.Text == text[4:]
			}
			if !whole {
				t.Errorf("%s: blob not stripped whole", row.name)
			}
		}
	}
}

func TestInflateStatuses(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	c := logRows(r, 300, false)
	cases := []struct {
		name string
		raw  []byte
		want InflateStatus
	}{
		{"gzip", gz(c), Inflated},
		{"zlib", zl(c), Inflated},
		{"zip", zipOf(zip.Deflate, zipFile{"a", c}), Inflated},
		{"plain", c, NotCompressed},
		{"gzip header random body", append([]byte{0x1f, 0x8b, 0x08, 0, 0, 0, 0, 0, 0, 0xff}, bytes.Repeat([]byte{0xa7, 0x13}, 600)...), Unreadable},
		{"zip local header no directory", methodZip(zipOf(zip.Store, zipFile{"a.txt", c}), 0)[:200], Unreadable},
		{"zip magic random body", withMagic(r, "PK\x03\x04"), NotCompressed},
		{"zlib truncated text", zl(c)[:len(zl(c))/2], Unreadable},
		{"over 4096 entries", zipOf(zip.Store, func() []zipFile {
			var fs []zipFile
			for i := 0; i < 4097; i++ {
				fs = append(fs, zipFile{fmt.Sprint(i), nil})
			}
			return fs
		}()...), Unreadable},
	}
	for _, tc := range cases {
		if _, st := Inflate(tc.raw, NewInflateBudgetFor(testLimits)); st != tc.want {
			t.Errorf("%s: status %d, want %d", tc.name, st, tc.want)
		}
	}
}

func TestInflateBlobCap(t *testing.T) {
	line := []byte("INFO request ok path=/health status=200\n")
	under := gz(bytes.Repeat(line, (39<<20/10)/len(line)))
	over := gz(bytes.Repeat(line, (9<<19)/len(line)))
	if _, st := Inflate(under, NewInflateBudgetFor(testLimits)); st != Inflated {
		t.Errorf("3.9 MiB inflated: status %d", st)
	}
	if _, st := Inflate(over, NewInflateBudgetFor(testLimits)); st != Unreadable {
		t.Errorf("4.5 MiB inflated: status %d", st)
	}
	if _, st := Inflate(zipOf(zip.Store, func() []zipFile {
		var fs []zipFile
		for i := 0; i < 4096; i++ {
			fs = append(fs, zipFile{fmt.Sprint(i), []byte("ok")})
		}
		return fs
	}()...), NewInflateBudgetFor(testLimits)); st != Inflated {
		t.Errorf("4096 entries: status %d", st)
	}
}

func TestInflateRequestBudget(t *testing.T) {
	part := gz(bytes.Repeat([]byte("INFO request ok path=/health status=200\n"), 75000))
	budget := NewInflateBudgetFor(testLimits)
	for i := 0; i < 2; i++ {
		if _, st := Inflate(part, budget); st != Inflated {
			t.Fatalf("blob %d of 3 MB: status %d", i, st)
		}
	}
	if _, st := Inflate(part, budget); st != Unreadable {
		t.Errorf("third 3 MB blob over the 8 MiB request budget: status %d", st)
	}
	text := strings.Repeat("see "+base64.StdEncoding.EncodeToString(part)+"\n", 3)
	if got := secretSpans(t, text); len(got) != 1 {
		t.Errorf("per-text budget: want exactly the third blob stripped, got %d spans", len(got))
	}
}

func TestInflateBombStopsAtCap(t *testing.T) {
	bomb := gz(gz(make([]byte, 100<<20)))
	if len(bomb) > 4<<10 {
		t.Fatalf("bomb is %d bytes", len(bomb))
	}
	start := time.Now()
	_, st := Inflate(bomb, NewInflateBudgetFor(testLimits))
	if st != Unreadable {
		t.Errorf("bomb status %d", st)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("bomb took %s", d)
	}
}

func TestUTF16AndNulSplitRuns(t *testing.T) {
	r := rand.New(rand.NewSource(16))
	for _, pii := range []struct {
		name string
		b    []byte
	}{
		{"email utf-16le", utf16le(" jane.doe@example.com ")},
		{"phone utf-16le", utf16le(" call 804-555-1234 ")},
		{"phone utf-16be", utf16be(" call 804-555-1234 ")},
		{"email utf-16be", utf16be(" jane.doe@example.com ")},
		{"email split by nul", []byte(" jane.d\x00oe@exa\x00mple.com ")},
		{"name utf-16le", utf16le(" Maria Gonzalez ")},
	} {
		for _, at := range []int{2048, 2049} {
			raw := make([]byte, 4096)
			r.Read(raw)
			copy(raw[at:], pii.b)
			if c, ok := blobVerdict(raw); !ok || c {
				t.Errorf("%s at offset %d: blob counted clean", pii.name, at)
			}
		}
	}
	for _, neutral := range [][]byte{utf16le(" the release notes for the build "), utf16be(" the release notes for the build ")} {
		raw := make([]byte, 4096)
		r.Read(raw)
		copy(raw[2048:], neutral)
		if c, _ := blobVerdict(raw); !c {
			t.Errorf("neutral utf-16 text made the blob dirty")
		}
	}
	if got := WideRuns([]byte("ab\x00cd\x00ef\x00gh\x00ij")); len(got) != 1 || got[0] != "abcdefghij" {
		t.Errorf("WideRuns joined %q", got)
	}
	if got := WideRuns([]byte("abcdefghij\x00\x00klmnopqr")); len(got) != 0 {
		t.Errorf("double nul joined %q", got)
	}
}

func TestBinaryNameRuns(t *testing.T) {
	r := rand.New(rand.NewSource(17))
	for _, tc := range []struct {
		text  string
		dirty bool
	}{
		{" Maria Gonzalez ", true},
		{"\nElena Petrova signed\n", true},
		{" Release Notes Draft ", false},
		{" Maria ", false},
		{" maria gonzalez ", false},
	} {
		raw := make([]byte, 4096)
		r.Read(raw)
		raw[2047], raw[2048+len(tc.text)] = 0, 0
		copy(raw[2048:], tc.text)
		if c, _ := blobVerdict(raw); c == tc.dirty {
			t.Errorf("%q: clean=%v, want dirty=%v", tc.text, c, tc.dirty)
		}
	}
}

func TestBinaryEmailShapeInRandomRuns(t *testing.T) {
	for _, s := range []string{"GT@d.BV", "H@x.CqE", "I@YS.wG"} {
		if binaryEmailRE.MatchString(s) {
			t.Errorf("%q accepted as a binary email", s)
		}
	}
	for _, s := range []string{"jane.doe@example.com", "ab@cd.org"} {
		if !binaryEmailRE.MatchString(s) {
			t.Errorf("%q rejected as a binary email", s)
		}
	}
}

func TestInflatedMarkupJudgedOnText(t *testing.T) {
	doc := `<?xml version="1.0"?><w:document><w:body><w:p w:rsidR="0001340F" w:rsidRDefault="0025371C"><w:r><w:t>The release plan covers build steps.</w:t></w:r></w:p></w:body></w:document>`
	if textHasFindings(doc) {
		t.Errorf("markup attributes counted as findings")
	}
	if !textHasFindings(strings.Replace(doc, "build steps.", "build steps. Mail jane.doe@example.com", 1)) {
		t.Errorf("email in markup text missed")
	}
	if !textHasFindings(strings.Replace(doc, "The release plan", "Maria Gonzalez wrote the release plan", 1)) {
		t.Errorf("person name in markup text missed")
	}
}
