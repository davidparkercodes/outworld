package upstream

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func postMessages(t *testing.T, url string, stream bool) *http.Response {
	body, _ := json.Marshal(map[string]any{"model": "m", "max_tokens": 64, "stream": stream,
		"system": []map[string]any{{"type": "text", "text": "task"}},
		"messages": []map[string]any{
			{"role": "user", "content": "Mail Marisol Quenby at m.q@mail.example on 2026-03-04."},
			{"role": "assistant", "content": []map[string]any{{"type": "text", "text": "Done."}}},
		}})
	resp, err := http.Post(url+"/v1/messages", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestFakeMessages(t *testing.T) {
	f := NewFake()
	s, err := Serve(f.Handler(), f.Recorder)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	resp := postMessages(t, s.URL, false)
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	resp.Body.Close()
	if out["type"] != "message" || out["role"] != "assistant" || out["stop_reason"] != "end_turn" || out["model"] != "m" {
		t.Fatalf("bad message envelope: %v", out)
	}
	if _, ok := out["stop_sequence"]; !ok {
		t.Fatal("stop_sequence should be present and null")
	}
	content := out["content"].([]any)[0].(map[string]any)["text"].(string)
	want := Respond([]ChatMessage{
		{Role: "user", Content: json.RawMessage(`"Mail Marisol Quenby at m.q@mail.example on 2026-03-04."`)},
		{Role: "assistant", Content: json.RawMessage(`"Done."`)},
	})
	if content != want {
		t.Fatalf("answer differs from the OpenAI fake's:\n%s\nwant:\n%s", content, want)
	}
	if strings.Contains(EchoSection(content), "task") {
		t.Error("system prompt should not be echoed")
	}

	resp = postMessages(t, s.URL, true)
	var sb strings.Builder
	var events []string
	deltas, event := 0, ""
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "event: ") {
			event = strings.TrimPrefix(line, "event: ")
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var ev map[string]any
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev); err != nil {
			t.Fatalf("bad event data: %v", err)
		}
		if ev["type"] != event {
			t.Fatalf("event line %q does not match data type %v", event, ev["type"])
		}
		if event == "content_block_delta" {
			d := ev["delta"].(map[string]any)
			if d["type"] != "text_delta" {
				t.Fatalf("unexpected delta %v", d)
			}
			sb.WriteString(d["text"].(string))
			deltas++
			continue
		}
		events = append(events, event)
	}
	resp.Body.Close()
	if got := strings.Join(events, " "); got != "message_start content_block_start ping content_block_stop message_delta message_stop" {
		t.Fatalf("event order: %s", got)
	}
	if sb.String() != content || deltas != len(Chunks(content)) {
		t.Fatalf("stream content differs from non-stream (%d deltas)", deltas)
	}
	recs := f.Recorder.All()
	if len(recs) != 2 || !strings.Contains(RequestText(recs[1].Body), "Marisol Quenby") {
		t.Fatalf("recorder missed requests: %d", len(recs))
	}
}
