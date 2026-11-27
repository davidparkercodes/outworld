package gateway

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math/rand"
	"net/http"
	"strings"
	"testing"
)

func fidelityPDF(r *rand.Rand, n int) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n1 0 obj << /Length 1990 /Filter /FlateDecode >>\nstream\n")
	b.Write(uniformBytes(r, n))
	b.WriteString("\nendstream\nendobj\ntrailer << /Root 1 0 R >>\n%%EOF\n")
	return b.Bytes()
}

func fidelityImage(r *rand.Rand, enc func(*bytes.Buffer, image.Image)) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 48, 48))
	for i := range img.Pix {
		img.Pix[i] = byte(r.Intn(256))
	}
	img.Set(0, 0, color.RGBA{1, 2, 3, 255})
	var b bytes.Buffer
	enc(&b, img)
	return b.Bytes()
}

func fidelityGzip(r *rand.Rand, n int) []byte {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write(uniformBytes(r, n))
	w.Close()
	return b.Bytes()
}

func fidelityZip(r *rand.Rand, n int) []byte {
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	f, _ := w.CreateHeader(&zip.FileHeader{Name: "blob.bin", Method: zip.Store})
	f.Write(uniformBytes(r, n))
	w.Close()
	return b.Bytes()
}

func fidelityPayloads() map[string][]byte {
	out := map[string][]byte{}
	for _, kb := range []int{1, 3, 6, 12, 48} {
		for seed := 0; seed < 30; seed++ {
			out[fmt.Sprintf("random %dKB seed %d", kb, seed)] = uniformBytes(rand.New(rand.NewSource(int64(kb*1000+seed))), kb<<10)
		}
	}
	r := rand.New(rand.NewSource(4096))
	for i := 0; i < 5; i++ {
		out[fmt.Sprintf("pdf %d", i)] = fidelityPDF(r, 4<<10)
		out[fmt.Sprintf("png %d", i)] = fidelityImage(r, func(b *bytes.Buffer, img image.Image) { png.Encode(b, img) })
		out[fmt.Sprintf("jpeg %d", i)] = fidelityImage(r, func(b *bytes.Buffer, img image.Image) { jpeg.Encode(b, img, &jpeg.Options{Quality: 90}) })
		out[fmt.Sprintf("gzip %d", i)] = fidelityGzip(r, 3<<10)
		out[fmt.Sprintf("zip %d", i)] = fidelityZip(r, 3<<10)
	}
	return out
}

func fidelityBodies(v string) map[string]map[string]any {
	all := encodedAnthropic(v)
	return map[string]map[string]any{
		"tool_use input data": all["tool_use input data"],
		"user text":           {"model": "claude-x", "max_tokens": 64, "messages": []any{map[string]any{"role": "user", "content": "see " + v}}},
	}
}

func lastForwarded(tr *fnTransport) string {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if len(tr.calls) == 0 {
		return ""
	}
	return tr.calls[len(tr.calls)-1].body
}

func TestCleanBinaryBase64PassesUnchanged(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for name, raw := range fidelityPayloads() {
		if len(raw) < 1024 {
			t.Fatalf("%s: payload only %d bytes", name, len(raw))
		}
		v := base64.StdEncoding.EncodeToString(raw)
		for where, body := range fidelityBodies(v) {
			resp, out := postJSON(t, gw.URL+"/v1/messages", body, nil)
			if resp.StatusCode != http.StatusOK {
				t.Errorf("%s in %s: %d %.200s", name, where, resp.StatusCode, out)
				continue
			}
			if sent := lastForwarded(tr); !strings.Contains(sent, v) || strings.Contains(sent, redactedBlob) {
				t.Errorf("%s in %s: base64 value was changed before forwarding", name, where)
			}
		}
	}
}

func TestBinaryBase64WithPIIStillCaught(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	r := rand.New(rand.NewSource(77))
	for _, pii := range []string{" ssn 123-45-6789 ", " mail jane.doe@example.com ", " card 4111 1111 1111 1111 "} {
		for _, kb := range []int{1, 3, 12} {
			raw := uniformBytes(r, kb<<10)
			copy(raw[len(raw)/2:], pii)
			v := base64.StdEncoding.EncodeToString(raw)
			for where, body := range fidelityBodies(v) {
				resp, out := postJSON(t, gw.URL+"/v1/messages", body, nil)
				if resp.StatusCode == http.StatusOK && strings.Contains(lastForwarded(tr), v) {
					t.Errorf("%q in %dKB blob in %s passed unchanged: %.200s", pii, kb, where, out)
				}
			}
		}
	}
}
