package gateway

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"testing"
	"unicode/utf16"
)

const testInflateCaps = "  max_inflated_bytes: 4194304\n  max_inflated_request_bytes: 8388608\n"

const infPII = "Contact: call 804-555-1234 or mail jane.doe@example.com\n"

func infRows(r *rand.Rand, n int, pii bool) []byte {
	var b bytes.Buffer
	for i := 0; i < n; i++ {
		if pii && i == n/2 {
			b.WriteString(infPII)
			continue
		}
		fmt.Fprintf(&b, "build %d finished in %d ms, cache hit ratio %d%%, artefact %08x\n", i, r.Intn(9999), r.Intn(100), r.Uint32())
	}
	return b.Bytes()
}

func infText(r *rand.Rand, n int) []byte {
	var b bytes.Buffer
	for i := 0; b.Len() < n; i++ {
		fmt.Fprintf(&b, "build %d finished in %d ms, artefact %08x\n", i, r.Intn(9999), r.Uint32())
	}
	return b.Bytes()[:n]
}

func infGz(b []byte) []byte {
	var out bytes.Buffer
	w := gzip.NewWriter(&out)
	w.Write(b)
	w.Close()
	return out.Bytes()
}

func infZlib(b []byte) []byte {
	var out bytes.Buffer
	w := zlib.NewWriter(&out)
	w.Write(b)
	w.Close()
	return out.Bytes()
}

type infFile struct {
	name string
	data []byte
}

func infZip(method uint16, files ...infFile) []byte {
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	for _, f := range files {
		fw, _ := w.CreateHeader(&zip.FileHeader{Name: f.name, Method: method})
		fw.Write(f.data)
	}
	w.Close()
	return out.Bytes()
}

func infTar(files ...infFile) []byte {
	var out bytes.Buffer
	w := tar.NewWriter(&out)
	for _, f := range files {
		w.WriteHeader(&tar.Header{Name: f.name, Mode: 0o644, Size: int64(len(f.data)), Typeflag: tar.TypeReg})
		w.Write(f.data)
	}
	w.Close()
	return out.Bytes()
}

func infPatchZip(b []byte, local, central int, set func(uint16) uint16) []byte {
	b = append([]byte(nil), b...)
	for i := 0; i+4 <= len(b); i++ {
		at := -1
		switch string(b[i : i+4]) {
		case "PK\x03\x04":
			at = i + local
		case "PK\x01\x02":
			at = i + central
		}
		if at >= 0 {
			binary.LittleEndian.PutUint16(b[at:], set(binary.LittleEndian.Uint16(b[at:])))
		}
	}
	return b
}

func infUTF16(s string, order binary.AppendByteOrder) []byte {
	var b []byte
	for _, u := range utf16.Encode([]rune(s)) {
		b = order.AppendUint16(b, u)
	}
	return b
}

func infMagic(r *rand.Rand, magic string) []byte {
	b := make([]byte, 4096)
	r.Read(b)
	copy(b, magic)
	return b
}

func infFiles(n, dirtyAt int, dirty []byte) []infFile {
	fs := make([]infFile, n)
	for i := range fs {
		data := []byte(fmt.Sprintf("ok %d\n", i))
		if i == dirtyAt {
			data = dirty
		}
		fs[i] = infFile{fmt.Sprintf("f%d.txt", i), data}
	}
	return fs
}

func infInRandom(r *rand.Rand, pii []byte, at int) []byte {
	b := make([]byte, 4096)
	r.Read(b)
	copy(b[at:], pii)
	return b
}

func inflateCases() (clean, dirty map[string][]byte) {
	r := rand.New(rand.NewSource(54))
	c, d := infRows(r, 300, false), infRows(r, 300, true)
	var bomb bytes.Buffer
	bomb.Write(make([]byte, 20<<20))
	bomb.WriteString(infPII)
	gd := infGz(d)
	corrupt := append([]byte(nil), gd...)
	corrupt[40] ^= 0xff
	clean = map[string][]byte{
		"gzip clean log":             infGz(c),
		"zip deflate clean log":      infZip(zip.Deflate, infFile{"log.txt", c}),
		"zip stored clean log":       infZip(zip.Store, infFile{"log.txt", c}),
		"zlib clean log":             infZlib(c),
		"gzip repetitive 2 MB log":   infGz(bytes.Repeat([]byte("INFO request ok path=/health status=200\n"), 50000)),
		"zip 300 clean files":        infZip(zip.Deflate, infFiles(300, -1, nil)...),
		"zip 4096 tiny files":        infZip(zip.Deflate, infFiles(4096, -1, nil)...),
		"gzip 3 clean members":       append(append(infGz(infText(r, 9000)), infGz(infText(r, 9000))...), infGz(infText(r, 9000))...),
		"random 4 KB gzip magic":     infMagic(r, "\x1f\x8b"),
		"random 4 KB zlib magic":     infMagic(r, "\x78\x9c"),
		"random 4 KB zip magic":      infMagic(r, "PK\x03\x04"),
		"utf-16 neutral text":        infInRandom(r, infUTF16(" the release notes for the build ", binary.LittleEndian), 2048),
		"random blob with zip magic": infMagic(r, "PK\x03\x04\x14\x00"),
	}
	dirty = map[string][]byte{
		"gzip pii row":                   gd,
		"zip deflate pii":                infZip(zip.Deflate, infFile{"a.txt", d}),
		"zip stored pii":                 infZip(zip.Store, infFile{"a.txt", d}),
		"zlib pii":                       infZlib(d),
		"zip pii in entry 200 of 250":    infZip(zip.Deflate, infFiles(250, 199, d)...),
		"zip in zip pii":                 infZip(zip.Deflate, infFile{"inner.zip", infZip(zip.Deflate, infFile{"a.txt", d})}),
		"gzip in zip pii":                infZip(zip.Deflate, infFile{"a.txt.gz", gd}),
		"zip in zip in zip":              infZip(zip.Deflate, infFile{"l1.zip", infZip(zip.Deflate, infFile{"l2.zip", infZip(zip.Deflate, infFile{"a.txt", c})})}),
		"gzip of gzip of gzip":           infGz(infGz(infGz(c))),
		"zip encrypted flag":             infPatchZip(infZip(zip.Deflate, infFile{"a.txt", c}), 6, 8, func(v uint16) uint16 { return v | 1 }),
		"zip bzip2 method":               infPatchZip(infZip(zip.Store, infFile{"a.txt", c}), 8, 10, func(uint16) uint16 { return 12 }),
		"truncated gzip":                 gd[:len(gd)/2],
		"corrupt deflate gzip":           corrupt,
		"gzip bomb 20 MB zeros plus pii": infGz(bomb.Bytes()),
		"pii after 9 MiB of text":        infGz(append(infText(r, 9<<20), infPII...)),
		"multi-member gzip pii member 2": append(infGz(c), infGz(bytes.Repeat([]byte(infPII), 30))...),
		"gzip utf-16le pii":              infGz(infUTF16(string(d), binary.LittleEndian)),
		"zip 4097 entries":               infZip(zip.Deflate, infFiles(4097, -1, nil)...),
		"zip 4096 entries pii in last":   infZip(zip.Deflate, infFiles(4096, 4095, []byte(infPII))...),
		"gzip 4.5 MiB clean":             infGz(infText(r, 9<<19)),
		"email utf-16le in random":       infInRandom(r, infUTF16(" jane.doe@example.com ", binary.LittleEndian), 2049),
		"phone utf-16le in random":       infInRandom(r, infUTF16(" call 804-555-1234 ", binary.LittleEndian), 2048),
		"phone utf-16be in random":       infInRandom(r, infUTF16(" call 804-555-1234 ", binary.BigEndian), 2049),
		"email split by nul in random":   infInRandom(r, []byte(" jane.d\x00oe@exa\x00mple.com "), 2048),
		"full name in random":            infInRandom(r, []byte("\x00 Maria Gonzalez \x00"), 2048),
		"pii at 3.8 MiB of 3.9 MiB gzip": func() []byte {
			t := infText(r, 39<<20/10)
			i := 38 << 20 / 10
			return infGz(append(append(append([]byte(nil), t[:i]...), "\n"+infPII...), t[i:]...))
		}(),
	}
	return clean, dirty
}

func infBodies(v string) map[string]map[string]any {
	return map[string]map[string]any{
		"tool_use input": encodedAnthropic(v)["tool_use input data"],
		"user text":      {"model": "claude-x", "max_tokens": 64, "messages": []any{map[string]any{"role": "user", "content": "see " + v}}},
	}
}

func filesBody(blobs ...[]byte) (map[string]any, []string) {
	var vs []any
	var raw []string
	for _, b := range blobs {
		v := base64.StdEncoding.EncodeToString(b)
		vs = append(vs, v)
		raw = append(raw, v)
	}
	body := encodedAnthropic("x")["tool_use input data"]
	msgs := body["messages"].([]any)
	asst := msgs[len(msgs)-1].(map[string]any)
	asst["content"] = []any{map[string]any{"type": "tool_use", "id": "toolu_01AbCdEfGhIjKlMnOpQrStUv", "name": "f", "input": map[string]any{"files": vs}}}
	return body, raw
}

func TestInflatedBlobsThroughGateway(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, testInflateCaps, tr)
	clean, dirty := inflateCases()
	for name, raw := range clean {
		v := base64.StdEncoding.EncodeToString(raw)
		for where, body := range infBodies(v) {
			resp, out := postJSON(t, gw.URL+"/v1/messages", body, nil)
			if resp.StatusCode != http.StatusOK || !strings.Contains(lastForwarded(tr), v) {
				t.Errorf("clean %s in %s: %d, forwarded verbatim=%v %.200s", name, where, resp.StatusCode, strings.Contains(lastForwarded(tr), v), out)
			}
		}
	}
	for name, raw := range dirty {
		v := base64.StdEncoding.EncodeToString(raw)
		for where, body := range infBodies(v) {
			tr.mu.Lock()
			tr.calls = nil
			tr.mu.Unlock()
			resp, out := postJSON(t, gw.URL+"/v1/messages", body, nil)
			if resp.StatusCode == http.StatusOK && strings.Contains(lastForwarded(tr), v) {
				t.Errorf("%s in %s forwarded verbatim: %.200s", name, where, out)
			}
		}
	}
}

func TestCompressedUnreadableReason(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, testInflateCaps, tr)
	r := rand.New(rand.NewSource(9))
	gd := infGz(infRows(r, 300, false))
	v := base64.StdEncoding.EncodeToString(gd[:len(gd)/2])
	resp, out := postJSON(t, gw.URL+"/v1/messages", infBodies(v)["tool_use input"], nil)
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(out), "gateway:compressed_unreadable") {
		t.Errorf("truncated gzip: %d %.300s", resp.StatusCode, out)
	}
}

func TestInflateRequestBudgetRows(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, testInflateCaps, tr)
	r := rand.New(rand.NewSource(56))
	three := func(pii bool) []byte {
		t := infText(r, 3<<20)
		if pii {
			t = append(t, "\n"+infPII...)
		}
		return infGz(t)
	}
	for _, tc := range []struct {
		name  string
		blobs [][]byte
		clean bool
	}{
		{"one blob 3.9 MiB clean", [][]byte{infGz(infText(r, 39<<20/10))}, true},
		{"two blobs 3 MiB clean", [][]byte{three(false), three(false)}, true},
		{"three blobs 3 MiB clean", [][]byte{three(false), three(false), three(false)}, false},
		{"two blobs 3 MiB pii in second", [][]byte{three(false), three(true)}, false},
	} {
		body, vs := filesBody(tc.blobs...)
		tr.mu.Lock()
		tr.calls = nil
		tr.mu.Unlock()
		resp, out := postJSON(t, gw.URL+"/v1/messages", body, nil)
		sent := lastForwarded(tr)
		verbatim := resp.StatusCode == http.StatusOK
		for _, v := range vs {
			verbatim = verbatim && strings.Contains(sent, v)
		}
		if verbatim != tc.clean {
			t.Errorf("%s: status %d verbatim=%v want clean=%v %.200s", tc.name, resp.StatusCode, verbatim, tc.clean, out)
		}
	}
}
