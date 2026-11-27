package gateway

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
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

const encodedPII = " card 4111 1111 1111 1111 jane.doe@example.com call 804-555-1234 today "

func encodedSamples(r *rand.Rand) map[string][2]string {
	short := []byte("Please call Maria Gonzalez at 804-555-1234 or mail jane.doe@example.com")
	long := func(mid string) []byte {
		b := uniformBytes(r, 64<<10)
		b[0] = 0
		copy(b[32<<10:], mid)
		return b
	}
	return map[string][2]string{
		"long":           {base64.StdEncoding.EncodeToString(long(encodedPII)), base64.StdEncoding.EncodeToString(long(strings.Repeat("x", len(encodedPII))))},
		"short":          {base64.StdEncoding.EncodeToString(short), base64.StdEncoding.EncodeToString([]byte("Please call the front desk at the usual time for the team"))},
		"short unpadded": {base64.RawStdEncoding.EncodeToString(short), base64.RawStdEncoding.EncodeToString([]byte("The build notes cover the release plan for the team!"))},
	}
}

func encodedAnthropic(v string) map[string]map[string]any {
	assistant := func(blocks ...any) map[string]any {
		b := map[string]any{"model": "claude-x", "max_tokens": 64, "messages": []any{map[string]any{"role": "user", "content": "hello"}, map[string]any{"role": "assistant", "content": blocks}}}
		return b
	}
	withResult := func(content any) map[string]any {
		b := assistant(map[string]any{"type": "tool_use", "id": "toolu_01AbCdEfGhIjKlMnOpQrStUv", "name": "f", "input": map[string]any{"q": "x"}})
		b["messages"] = append(b["messages"].([]any), map[string]any{"role": "user", "content": []any{map[string]any{"type": "tool_result", "tool_use_id": "toolu_01AbCdEfGhIjKlMnOpQrStUv", "content": content}}})
		return b
	}
	top := map[string]any{"model": "claude-x", "max_tokens": 64, "messages": []any{map[string]any{"role": "user", "content": "hello"}}, "x_extra": v}
	return map[string]map[string]any{
		"tool_use input data":    assistant(map[string]any{"type": "tool_use", "id": "toolu_01AbCdEfGhIjKlMnOpQrStUv", "name": "f", "input": map[string]any{"data": v}}),
		"nested tool input":      assistant(map[string]any{"type": "tool_use", "id": "toolu_01AbCdEfGhIjKlMnOpQrStUv", "name": "f", "input": map[string]any{"x": map[string]any{"blob": v}}}),
		"tool_result string":     withResult(v),
		"tool_result text block": withResult([]any{map[string]any{"type": "text", "text": v}}),
		"top-level unknown key":  top,
		"unknown block":          assistant(map[string]any{"type": "mystery_block", "data": v}),
		"lone token in text":     {"model": "claude-x", "max_tokens": 64, "messages": []any{map[string]any{"role": "user", "content": "see this: " + v + " thanks"}}},
	}
}

func encodedOpenAI(v string) map[string]map[string]any {
	base := func(msgs ...any) map[string]any {
		return map[string]any{"model": "gpt-x", "messages": append([]any{map[string]any{"role": "user", "content": "hello"}}, msgs...)}
	}
	args, _ := json.Marshal(map[string]any{"data": v})
	top := base()
	top["x_extra"] = v
	return map[string]map[string]any{
		"tool call arguments":   base(map[string]any{"role": "assistant", "content": nil, "tool_calls": []any{map[string]any{"id": "call_AbCdEfGhIjKlMnOpQrStUv12", "type": "function", "function": map[string]any{"name": "f", "arguments": string(args)}}}}, map[string]any{"role": "tool", "tool_call_id": "call_AbCdEfGhIjKlMnOpQrStUv12", "content": "done"}),
		"tool message":          base(map[string]any{"role": "assistant", "content": nil, "tool_calls": []any{map[string]any{"id": "call_AbCdEfGhIjKlMnOpQrStUv12", "type": "function", "function": map[string]any{"name": "f", "arguments": "{}"}}}}, map[string]any{"role": "tool", "tool_call_id": "call_AbCdEfGhIjKlMnOpQrStUv12", "content": v}),
		"top-level unknown key": top,
		"unknown content part":  base(map[string]any{"role": "user", "content": []any{map[string]any{"type": "mystery", "data": v}}}),
		"lone token in text":    base(map[string]any{"role": "user", "content": "see this: " + v + " thanks"}),
	}
}

func TestEncodedValuesAreScannedInEveryPlacement(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	r := rand.New(rand.NewSource(1980))
	for size, pair := range encodedSamples(r) {
		for api, build := range map[string]func(string) map[string]map[string]any{"/v1/messages": encodedAnthropic, "/v1/chat/completions": encodedOpenAI} {
			for i, v := range pair {
				for where, body := range build(v) {
					resp, out := postJSON(t, gw.URL+api, body, nil)
					want := http.StatusUnprocessableEntity
					if i == 1 && where != "unknown content part" {
						want = 200
					}
					if resp.StatusCode != want {
						t.Errorf("%s %s %s with pii=%v: %d %.200s", api, where, size, i == 0, resp.StatusCode, out)
					}
				}
			}
		}
	}
}

func TestRedactModeReplacesEncodedValues(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "  encoded_values: redact\n", tr)
	r := rand.New(rand.NewSource(1981))
	for size, pair := range encodedSamples(r) {
		v := pair[0]
		body := encodedAnthropic(v)["tool_use input data"]
		tr.mu.Lock()
		tr.calls = nil
		tr.mu.Unlock()
		resp, out := postJSON(t, gw.URL+"/v1/messages", body, nil)
		if resp.StatusCode != 200 {
			t.Fatalf("redact %s: %d %.200s", size, resp.StatusCode, out)
		}
		tr.mu.Lock()
		sent := tr.calls[len(tr.calls)-1].body
		tr.mu.Unlock()
		if strings.Contains(sent, v[:40]) || !strings.Contains(sent, redactedBlob) {
			t.Errorf("redact %s: the value was not replaced", size)
		}
	}
}

func TestHighEntropyTokensNeverGoUpstreamUnchanged(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	r := rand.New(rand.NewSource(1982))
	for _, n := range []int{32, 64, 96} {
		token := mixedSecret(r, n-2) + "!~"
		tr.mu.Lock()
		tr.calls = nil
		tr.mu.Unlock()
		resp, out := postJSON(t, gw.URL+"/v1/messages", encodedAnthropic(token)["tool_use input data"], nil)
		if resp.StatusCode == 200 {
			tr.mu.Lock()
			sent := tr.calls[len(tr.calls)-1].body
			tr.mu.Unlock()
			if strings.Contains(sent, token[:n-2]) {
				t.Errorf("%d-character token went upstream unchanged", n)
			}
		} else if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("%d-character token: %d %.200s", n, resp.StatusCode, out)
		}
	}
}

func mediaKinds(r *rand.Rand, size int) map[string][]byte {
	side := 8
	for side*side*3 < size {
		side *= 2
	}
	img := image.NewRGBA(image.Rect(0, 0, side, side))
	for i := range img.Pix {
		img.Pix[i] = byte(r.Intn(256))
	}
	var pngBuf, jpgBuf bytes.Buffer
	png.Encode(&pngBuf, img)
	small := image.NewRGBA(image.Rect(0, 0, side, side))
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			small.Set(x, y, color.RGBA{byte(x), byte(y), byte(x ^ y), 255})
		}
	}
	jpeg.Encode(&jpgBuf, img, &jpeg.Options{Quality: 90})
	text := []byte(strings.Repeat("the build notes cover the release plan for the team. ", size/53+1))
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	zw.Write(uniformBytes(r, size))
	zw.Close()
	var zb bytes.Buffer
	z := zip.NewWriter(&zb)
	w, _ := z.Create("notes.bin")
	w.Write(uniformBytes(r, size))
	w.Write(text[:size/4])
	z.Close()
	var stream bytes.Buffer
	zl := zlib.NewWriter(&stream)
	zl.Write(uniformBytes(r, size))
	zl.Close()
	pdf := fmt.Sprintf("%%PDF-1.4\n1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj\n3 0 obj << /Type /Page /Parent 2 0 R /Contents 4 0 R >> endobj\n4 0 obj << /Length %d /Filter /FlateDecode >> stream\n", stream.Len())
	pdfBytes := append(append([]byte(pdf), stream.Bytes()...), "\nendstream endobj\ntrailer << /Root 1 0 R >>\n%%EOF\n"...)
	return map[string][]byte{"png": pngBuf.Bytes(), "jpeg": jpgBuf.Bytes(), "pdf": pdfBytes, "gzip": gz.Bytes(), "zip": zb.Bytes(), "random": uniformBytes(r, size), "protobuf": reviewerProtobuf(r, size)}
}

func TestFilePayloadsInToolDataPass(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	r := rand.New(rand.NewSource(1983))
	sizes := []int{4 << 10, 64 << 10, 512 << 10}
	if corpusScale > 1 {
		sizes = []int{4 << 10}
	}
	refused := map[string]int{}
	total := map[string]int{}
	for _, size := range sizes {
		for kind, raw := range mediaKinds(r, size) {
			v := base64.StdEncoding.EncodeToString(raw)
			for _, where := range []string{"tool_use input data", "tool_result string"} {
				resp, out := postJSON(t, gw.URL+"/v1/messages", encodedAnthropic(v)[where], nil)
				total[kind]++
				if resp.StatusCode != 200 {
					refused[kind]++
					if kind != "protobuf" {
						t.Errorf("%s of %d bytes in %s: %d %.200s", kind, size, where, resp.StatusCode, out)
					}
				}
			}
		}
	}
	t.Logf("refused by kind %v of %v", refused, total)
}

func TestTypedMediaAndLargeOrdinaryValues(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "  allow_images: true\n", tr)
	r := rand.New(rand.NewSource(1984))
	big := 2 << 20
	if corpusScale > 1 {
		big = 256 << 10
	}
	pngBytes := mediaKinds(r, big)["png"]
	pdfBytes := mediaKinds(r, big)["pdf"]
	imageBlock := anthBody([]any{map[string]any{"type": "image", "source": map[string]any{"type": "base64", "media_type": "image/png", "data": base64.StdEncoding.EncodeToString(pngBytes)}}, map[string]any{"type": "text", "text": "what is this"}}, false)
	docBlock := anthBody([]any{map[string]any{"type": "document", "source": map[string]any{"type": "base64", "media_type": "application/pdf", "data": base64.StdEncoding.EncodeToString(pdfBytes)}}, map[string]any{"type": "text", "text": "summarize"}}, false)
	dataURL := map[string]any{"model": "gpt-x", "messages": []any{map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "what is this"},
		map[string]any{"type": "image_url", "image_url": map[string]any{"url": "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)}}}}}}
	for name, c := range map[string]struct {
		api  string
		body map[string]any
	}{"image block": {"/v1/messages", imageBlock}, "document block": {"/v1/messages", docBlock}, "openai image data url": {"/v1/chat/completions", dataURL}} {
		resp, out := postJSON(t, gw.URL+c.api, c.body, nil)
		if resp.StatusCode != 200 {
			t.Errorf("%s of %d bytes: %d %.200s", name, big, resp.StatusCode, out)
		}
	}
	n := 900 << 10
	if corpusScale > 1 {
		n = 64 << 10
	}
	clean := uniformBytes(r, n)
	clean[0] = 0
	withPII := append([]byte{}, clean...)
	copy(withPII[n/2:], encodedPII)
	for i, raw := range [][]byte{clean, withPII} {
		resp, out := postJSON(t, gw.URL+"/v1/messages", encodedAnthropic(base64.StdEncoding.EncodeToString(raw))["tool_use input data"], nil)
		want := 200
		if i == 1 {
			want = http.StatusUnprocessableEntity
		}
		if resp.StatusCode != want || strings.Contains(string(out), "opaque_too_large") {
			t.Errorf("%d bytes base64 under an ordinary field, pii=%v: %d %.200s", n, i == 1, resp.StatusCode, out)
		}
	}
}
