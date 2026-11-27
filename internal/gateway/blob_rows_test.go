package gateway

import (
	"archive/tar"
	"bytes"
	"compress/flate"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
	mrand "math/rand"
	"net/http"
	"strings"
	"testing"
)

func rowsDeflate(b []byte) []byte {
	var o bytes.Buffer
	w, _ := flate.NewWriter(&o, 6)
	w.Write(b)
	w.Close()
	return o.Bytes()
}

func rowsTarTyped(flag byte, data []byte) []byte {
	var o bytes.Buffer
	w := tar.NewWriter(&o)
	w.WriteHeader(&tar.Header{Name: "entry", Mode: 0o644, Size: int64(len(data)), Typeflag: flag, Format: tar.FormatGNU})
	w.Write(data)
	w.Close()
	return o.Bytes()
}

func blobRowCases(seed int64) map[string]string {
	r := mrand.New(mrand.NewSource(seed))
	d := infRows(r, 120, true)
	fhcrc := append([]byte{0x1f, 0x8b, 0x08, 0x02, 0, 0, 0, 0, 0, 0xff, 0xde, 0xad}, rowsDeflate(d)...)
	fhcrc = binary.LittleEndian.AppendUint32(binary.LittleEndian.AppendUint32(fhcrc, crc32.ChecksumIEEE(d)), uint32(len(d)))
	pad := make([]byte, 1100)
	r.Read(pad)
	sfx := make([]byte, 16)
	r.Read(sfx)
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	der := x509.MarshalPKCS1PrivateKey(key)
	inRandom := append(append(infMagic(r, "")[:2048], der[:len(der)*7/10]...), infMagic(r, "")[:2048]...)
	run := base64.StdEncoding.EncodeToString(infMagic(r, ""))
	b64 := base64.StdEncoding.EncodeToString
	return map[string]string{
		"unreadable gzip of gzip of zlib": b64(infGz(infGz(infZlib(d)))),
		"unreadable xz magic":             b64(append([]byte{0xfd, '7', 'z', 'X', 'Z', 0}, pad...)),
		"unreadable zstd magic":           b64(append([]byte{0x28, 0xb5, 0x2f, 0xfd}, pad...)),
		"pii gzip fhcrc bad crc":          b64(fhcrc),
		"pii zip sfx prefix":              b64(append(sfx, infZip(8, infFile{"a.txt", d})...)),
		"pii tar.gz typeflag 7":           b64(infGz(rowsTarTyped('7', d))),
		"pii plain tar with gz entry":     b64(infTar(infFile{"a.gz", infGz(d)})),
		"strip rsa key cut inside 4 KB":   b64(inRandom),
		"strip vendor key mid base64 run": run[:700] + "AK" + "IAQWERTYUIOPASDFGH" + run[720:1500],
	}
}

func TestBlobRowsThroughGateway(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, testInflateCaps, tr)
	for _, seed := range []int64{18, 19, 20} {
		for name, v := range blobRowCases(seed) {
			for where, body := range infBodies(v) {
				tr.mu.Lock()
				tr.calls = nil
				tr.mu.Unlock()
				resp, out := postJSON(t, gw.URL+"/v1/messages", body, nil)
				verbatim := resp.StatusCode == http.StatusOK && strings.Contains(lastForwarded(tr), v)
				switch {
				case strings.HasPrefix(name, "unreadable") && (resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(out), "gateway:compressed_unreadable")):
					t.Errorf("seed %d %s in %s: want compressed_unreadable, got %d %.200s", seed, name, where, resp.StatusCode, out)
				case strings.HasPrefix(name, "pii") && (resp.StatusCode != http.StatusUnprocessableEntity || strings.Contains(string(out), "compressed_unreadable")):
					t.Errorf("seed %d %s in %s: want a personal data refusal, got %d %.200s", seed, name, where, resp.StatusCode, out)
				case verbatim:
					t.Errorf("seed %d %s in %s forwarded verbatim", seed, name, where)
				}
			}
		}
	}
}
