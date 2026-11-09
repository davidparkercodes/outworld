package upstream

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func post(t *testing.T, url string, stream bool) *http.Response {
	body, _ := json.Marshal(map[string]any{"model": "m", "stream": stream, "messages": []map[string]any{
		{"role": "system", "content": "task"},
		{"role": "user", "content": "Mail Marisol Quenby at m.q@mail.example on 2026-03-04."},
	}})
	resp, err := http.Post(url+"/v1/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestFakeServer(t *testing.T) {
	f := NewFake()
	s, err := Serve(f.Handler(), f.Recorder)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	resp := post(t, s.URL, false)
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	resp.Body.Close()
	content := out["choices"].([]any)[0].(map[string]any)["message"].(map[string]any)["content"].(string)
	for _, want := range []string{"Marisol Quenby\n", "[email] m.q@mail.example", "[date] 2026-03-04", EchoMarker} {
		if !strings.Contains(content, want) {
			t.Errorf("response missing %q:\n%s", want, content)
		}
	}
	if strings.Contains(EchoSection(content), "task") {
		t.Error("system prompt should not be echoed")
	}

	resp = post(t, s.URL, true)
	var sb strings.Builder
	sc := bufio.NewScanner(resp.Body)
	done := false
	for sc.Scan() {
		line := strings.TrimPrefix(sc.Text(), "data: ")
		if line == "[DONE]" {
			done = true
			break
		}
		var ch map[string]any
		if json.Unmarshal([]byte(line), &ch) == nil {
			d := ch["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)
			if c, ok := d["content"].(string); ok {
				sb.WriteString(c)
			}
		}
	}
	resp.Body.Close()
	if !done || sb.String() != content {
		t.Fatalf("stream content differs from non-stream (done=%v)", done)
	}
	if n := len(f.Recorder.All()); n != 2 {
		t.Fatalf("recorded %d requests", n)
	}
	if txt := RequestText(f.Recorder.All()[0].Body); !strings.Contains(txt, "Marisol Quenby") {
		t.Fatal("RequestText lost content")
	}
}

func TestChunksCover(t *testing.T) {
	s := strings.Repeat("abcdefghij", 13)
	if strings.Join(Chunks(s), "") != s {
		t.Fatal("chunks do not reassemble")
	}
}
