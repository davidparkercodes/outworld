package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/davidparkercodes/outworld/internal/app"
	"github.com/davidparkercodes/outworld/internal/config"
)

const anthUserText = "Hi, I'm John Smith (Johnny to friends). My email is john.smith@gmail.com, phone (804) 555-1234, " +
	"I live at 123 Main Street. Maria Garcia's visit is on March 15, 2024."

const attributionText = "x-anthropic-billing-header: cc_version=9.9.9; cc_entrypoint=cli; cch=abc12;"

type anthCall struct {
	path   string
	query  string
	header http.Header
	body   string
}

type anthFake struct {
	mu       sync.Mutex
	calls    []anthCall
	produced [][]any
	srv      *httptest.Server
	respond  func(req map[string]any, last string) []any
	failBody string
	failCode int
}

func newAnthFake(t *testing.T) *anthFake {
	f := &anthFake{}
	f.respond = func(_ map[string]any, last string) []any {
		return []any{map[string]any{"type": "text", "text": "ECHO: " + last}}
	}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *anthFake) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func (f *anthFake) call(i int) anthCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	if i < 0 {
		i = len(f.calls) + i
	}
	return f.calls[i]
}

func (f *anthFake) made(i int) []any {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.produced[i]
}

func (c anthCall) json(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(c.body), &m); err != nil {
		t.Fatalf("upstream body is not json: %v", err)
	}
	return m
}

func lastText(req map[string]any) string {
	msgs, _ := req["messages"].([]any)
	if len(msgs) == 0 {
		return ""
	}
	msg, _ := msgs[len(msgs)-1].(map[string]any)
	switch c := msg["content"].(type) {
	case string:
		return c
	case []any:
		out := ""
		for _, b := range c {
			if blk, ok := b.(map[string]any); ok && blk["type"] == "text" {
				out, _ = blk["text"].(string)
			}
		}
		return out
	}
	return ""
}

func clone(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	json.Unmarshal(b, &out)
	return out
}

func (f *anthFake) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	f.calls = append(f.calls, anthCall{path: r.URL.Path, query: r.URL.RawQuery, header: r.Header.Clone(), body: string(body)})
	failBody, failCode := f.failBody, f.failCode
	f.mu.Unlock()
	if failCode != 0 {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "7")
		w.Header().Set("Request-Id", "req_fake")
		w.Header().Set("X-Should-Retry", "false")
		w.Header().Set("Anthropic-Ratelimit-Requests-Remaining", "3")
		w.WriteHeader(failCode)
		io.WriteString(w, failBody)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/count_tokens") {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{ "input_tokens": 42 }`)
		return
	}
	var req map[string]any
	json.Unmarshal(body, &req)
	content := f.respond(req, lastText(req))
	b, _ := json.Marshal(content)
	var kept []any
	json.Unmarshal(b, &kept)
	f.mu.Lock()
	f.produced = append(f.produced, kept)
	f.mu.Unlock()
	if stream, _ := req["stream"].(bool); stream {
		writeAnthSSE(w, content)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Request-Id", "req_fake")
	json.NewEncoder(w).Encode(map[string]any{"id": "msg_1", "type": "message", "role": "assistant", "model": "claude-x",
		"content": content, "stop_reason": "end_turn", "usage": map[string]any{"input_tokens": 5, "output_tokens": 7}})
}

func writeAnthSSE(w http.ResponseWriter, content []any) {
	w.Header().Set("Content-Type", "text/event-stream")
	fl := w.(http.Flusher)
	send := func(event string, data map[string]any) {
		data["type"] = event
		b, _ := json.Marshal(data)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
		fl.Flush()
	}
	send("message_start", map[string]any{"message": map[string]any{"id": "msg_1", "type": "message", "role": "assistant", "content": []any{}, "usage": map[string]any{"input_tokens": 5}}})
	send("ping", map[string]any{})
	for i, b := range content {
		blk := b.(map[string]any)
		switch blk["type"] {
		case "text":
			text := blk["text"].(string)
			send("content_block_start", map[string]any{"index": i, "content_block": map[string]any{"type": "text", "text": ""}})
			for j, step := 0, 1; j < len(text); j, step = j+step, step%3+1 {
				end := min(j+step, len(text))
				send("content_block_delta", map[string]any{"index": i, "delta": map[string]any{"type": "text_delta", "text": text[j:end]}})
				if j == 0 {
					send("ping", map[string]any{})
				}
			}
		case "thinking":
			send("content_block_start", map[string]any{"index": i, "content_block": map[string]any{"type": "thinking", "thinking": "", "signature": ""}})
			send("content_block_delta", map[string]any{"index": i, "delta": map[string]any{"type": "thinking_delta", "thinking": blk["thinking"]}})
			send("content_block_delta", map[string]any{"index": i, "delta": map[string]any{"type": "signature_delta", "signature": blk["signature"]}})
		case "tool_use":
			send("content_block_start", map[string]any{"index": i, "content_block": map[string]any{"type": "tool_use", "id": blk["id"], "name": blk["name"], "input": map[string]any{}}})
			in, _ := json.Marshal(blk["input"])
			for j := 0; j < len(in); j += 2 {
				end := min(j+2, len(in))
				send("content_block_delta", map[string]any{"index": i, "delta": map[string]any{"type": "input_json_delta", "partial_json": string(in[j:end])}})
			}
		}
		send("content_block_stop", map[string]any{"index": i})
	}
	send("message_delta", map[string]any{"delta": map[string]any{"stop_reason": "end_turn"}, "usage": map[string]any{"output_tokens": 9}})
	send("message_stop", map[string]any{})
}

type anthHarness struct {
	gw     *httptest.Server
	claude *anthFake
	raw    *anthFake
	oai    *fakeUpstream
	logs   *bytes.Buffer
}

func newAnthHarness(t *testing.T, upstreamOpts, policyExtra string) *anthHarness {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "people.csv"), []byte(people), 0o600)
	h := &anthHarness{claude: newAnthFake(t), raw: newAnthFake(t), oai: newFake(t, "oai"), logs: &bytes.Buffer{}}
	t.Setenv("OUTWORLD_VAULT_KEY", strings.Repeat("cd", 32))
	t.Setenv("CLAUDE_KEY", "sk-upstream-env-key")
	if upstreamOpts == "" {
		upstreamOpts = "api_key_env: CLAUDE_KEY"
	}
	cfgText := fmt.Sprintf(`listen: 127.0.0.1:0
vault:
  path: ./outworld.vault
  key_env: OUTWORLD_VAULT_KEY
upstreams:
  claude: {api: anthropic, base_url: %s/v1, %s}
  raw: {api: anthropic, base_url: %s/v1}
  oai: {base_url: %s/v1}
policy:
  default_route: masked
  masked_upstream: oai
  passthrough_upstream: oai
  anthropic:
    masked_upstream: claude
    passthrough_upstream: raw
  date_shift: true
  dictionary: ./people.csv
  entities:
    person: mask
    email: mask
    phone: mask
    address: mask
    date: shift
    org: keep
    location: keep
%s`, h.claude.srv.URL, upstreamOpts, h.raw.srv.URL, h.oai.srv.URL, policyExtra)
	cfgPath := filepath.Join(dir, "outworld.yaml")
	os.WriteFile(cfgPath, []byte(cfgText), 0o600)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	eng, err := app.BuildEngine(cfg)
	if err != nil {
		t.Fatal(err)
	}
	srv, err := New(cfg, eng, Options{Logger: slog.New(slog.NewJSONHandler(h.logs, nil))})
	if err != nil {
		t.Fatal(err)
	}
	h.gw = httptest.NewServer(srv.Handler())
	t.Cleanup(h.gw.Close)
	return h
}

func (h *anthHarness) post(t *testing.T, path string, body any, hdr map[string]string) (*http.Response, []byte) {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, h.gw.URL+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp, out
}

func assertNoReals(t *testing.T, what, body string) {
	t.Helper()
	for _, r := range realValues {
		if strings.Contains(body, r) {
			t.Fatalf("%s: real value %q present: %s", what, r, body)
		}
	}
}

func anthBody(content any, stream bool) map[string]any {
	return map[string]any{"model": "claude-x", "max_tokens": 64, "stream": stream,
		"messages": []any{map[string]any{"role": "user", "content": content}}}
}

func assertAnthError(t *testing.T, resp *http.Response, body []byte, status int, code string) {
	t.Helper()
	var env struct {
		Type  string `json:"type"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	json.Unmarshal(body, &env)
	if resp.StatusCode != status || env.Type != "error" || env.Error.Type != code || env.Error.Message == "" {
		t.Fatalf("want %d %s in anthropic envelope, got %d %s", status, code, resp.StatusCode, body)
	}
}

type sseEvent struct {
	name string
	data map[string]any
	raw  string
}

func parseAnthSSE(t *testing.T, body []byte) []sseEvent {
	t.Helper()
	var out []sseEvent
	for _, chunk := range strings.Split(string(body), "\n\n") {
		var ev sseEvent
		for _, line := range strings.Split(chunk, "\n") {
			switch {
			case strings.HasPrefix(line, "event: "):
				ev.name = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				ev.raw = strings.TrimPrefix(line, "data: ")
				if err := json.Unmarshal([]byte(ev.raw), &ev.data); err != nil {
					t.Fatalf("bad sse data %q: %v", ev.raw, err)
				}
			}
		}
		if ev.name == "" && ev.data == nil {
			continue
		}
		if ev.data["type"] != ev.name {
			t.Fatalf("event line %q does not match data type %v", ev.name, ev.data["type"])
		}
		out = append(out, ev)
	}
	return out
}

func collapse(names []string) []string {
	var out []string
	for _, n := range names {
		if len(out) == 0 || out[len(out)-1] != n {
			out = append(out, n)
		}
	}
	return out
}

func assembleAnth(t *testing.T, events []sseEvent) []any {
	t.Helper()
	blocks := map[int]map[string]any{}
	partial := map[int]string{}
	max := -1
	for _, ev := range events {
		idx := int(asFloat(ev.data["index"]))
		switch ev.name {
		case "content_block_start":
			blocks[idx] = ev.data["content_block"].(map[string]any)
			if idx > max {
				max = idx
			}
		case "content_block_delta":
			d := ev.data["delta"].(map[string]any)
			b := blocks[idx]
			switch d["type"] {
			case "text_delta":
				b["text"] = b["text"].(string) + d["text"].(string)
			case "thinking_delta":
				b["thinking"] = b["thinking"].(string) + d["thinking"].(string)
			case "signature_delta":
				b["signature"] = b["signature"].(string) + d["signature"].(string)
			case "input_json_delta":
				partial[idx] += d["partial_json"].(string)
			}
		case "content_block_stop":
			if p, ok := partial[idx]; ok {
				var in any
				if err := json.Unmarshal([]byte(p), &in); err != nil {
					t.Fatalf("reassembled tool input is not json: %q", p)
				}
				blocks[idx]["input"] = in
			}
		}
	}
	var out []any
	for i := 0; i <= max; i++ {
		out = append(out, blocks[i])
	}
	return out
}

func asFloat(v any) float64 {
	f, _ := v.(float64)
	return f
}

func TestAnthropicNonStreamRoundTrip(t *testing.T) {
	thinking := map[string]any{"type": "thinking", "thinking": "The user <asked> about a visit & more.", "signature": "c2lnbmF0dXJl+/=="}
	redacted := map[string]any{"type": "redacted_thinking", "data": "b3BhcXVl+/=="}
	outputConfig := map[string]any{"effort": "high", "format": map[string]any{"type": "json_schema"}}
	cases := []struct {
		name   string
		system any
	}{
		{"system string", "You assist John Smith of 123 Main Street."},
		{"system blocks", []any{
			map[string]any{"type": "text", "text": attributionText},
			map[string]any{"type": "text", "text": "You assist John Smith of 123 Main Street.", "cache_control": map[string]any{"type": "ephemeral", "ttl": "1h"}},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newAnthHarness(t, "", "")
			h.claude.respond = func(_ map[string]any, last string) []any {
				return []any{
					map[string]any{"type": "thinking", "thinking": "thinking about " + last, "signature": "c2ln"},
					map[string]any{"type": "text", "text": "ECHO: " + last},
					map[string]any{"type": "tool_use", "id": "toolu_2", "name": "send", "input": map[string]any{"body": last, "count": 3, "nested": []any{last}}},
				}
			}
			body := map[string]any{
				"model": "claude-x", "max_tokens": 64, "system": tc.system,
				"output_config":      outputConfig,
				"context_management": map[string]any{"edits": []any{map[string]any{"type": "clear_tool_uses"}}},
				"thinking":           map[string]any{"type": "adaptive"},
				"tool_choice":        map[string]any{"type": "auto"},
				"metadata":           map[string]any{"user_id": "john.smith@gmail.com"},
				"messages": []any{
					map[string]any{"role": "user", "content": "Look up Maria Garcia please."},
					map[string]any{"role": "assistant", "content": []any{
						thinking, redacted,
						map[string]any{"type": "text", "text": "Looking up Maria Garcia."},
						map[string]any{"type": "tool_use", "id": "toolu_1", "name": "lookup", "input": map[string]any{"name": "Maria Garcia", "phone": "(804) 555-1234"}},
					}},
					map[string]any{"role": "user", "content": []any{
						map[string]any{"type": "tool_result", "tool_use_id": "toolu_1", "content": "email john.smith@gmail.com"},
						map[string]any{"type": "tool_result", "tool_use_id": "toolu_1", "content": []any{map[string]any{"type": "text", "text": "lives at 123 Main Street"}}},
					}},
					map[string]any{"role": "system", "content": "", "note": "kept"},
					map[string]any{"role": "user", "content": []any{
						map[string]any{"type": "text", "text": anthUserText, "cache_control": map[string]any{"type": "ephemeral"}},
					}},
				},
			}
			resp, out := h.post(t, "/v1/messages?beta=true", body, map[string]string{
				"anthropic-version": "2099-01-01", "anthropic-beta": "feature-a,feature-b", "anthropic-workspace-id": "ws1",
				"x-app": "cli", "x-stainless-lang": "js", "x-claude-code-session-id": "s1", "User-Agent": "claude-cli/9.9",
				"Cookie": "a=b", "X-Outworld-Scope": "ignored", "x-api-key": "client-key",
			})
			if resp.StatusCode != 200 {
				t.Fatalf("status %d: %s", resp.StatusCode, out)
			}
			call := h.claude.call(-1)
			assertNoReals(t, "upstream body", call.body)
			if call.path != "/v1/messages" || call.query != "beta=true" {
				t.Fatalf("upstream url %s?%s", call.path, call.query)
			}
			want := map[string]string{"Anthropic-Version": "2099-01-01", "Anthropic-Beta": "feature-a,feature-b", "Anthropic-Workspace-Id": "ws1",
				"X-App": "cli", "X-Stainless-Lang": "js", "X-Claude-Code-Session-Id": "s1", "User-Agent": "claude-cli/9.9", "X-Api-Key": "sk-upstream-env-key"}
			for k, v := range want {
				if call.header.Get(k) != v {
					t.Fatalf("upstream header %s = %q, want %q", k, call.header.Get(k), v)
				}
			}
			for _, k := range []string{"Cookie", "X-Outworld-Scope", "Authorization"} {
				if call.header.Get(k) != "" {
					t.Fatalf("header %s must not reach upstream", k)
				}
			}
			up := call.json(t)
			if !reflect.DeepEqual(up["output_config"], outputConfig) || up["context_management"] == nil || up["thinking"] == nil || up["tool_choice"] == nil {
				t.Fatalf("unknown top-level fields not preserved: %s", call.body)
			}
			if md, _ := up["metadata"].(map[string]any); md["user_id"] != nil {
				t.Fatal("metadata.user_id must be dropped")
			}
			if sys, ok := up["system"].([]any); ok {
				if sys[0].(map[string]any)["text"] != attributionText {
					t.Fatalf("attribution block changed: %v", sys[0])
				}
				if !reflect.DeepEqual(sys[1].(map[string]any)["cache_control"], map[string]any{"type": "ephemeral", "ttl": "1h"}) {
					t.Fatal("system cache_control lost")
				}
			} else if _, ok := up["system"].(string); !ok {
				t.Fatal("system string must stay a string")
			}
			msgs := up["messages"].([]any)
			asst := msgs[1].(map[string]any)["content"].([]any)
			if !reflect.DeepEqual(asst[0], thinking) || !reflect.DeepEqual(asst[1], redacted) {
				t.Fatalf("thinking blocks changed: %v %v", asst[0], asst[1])
			}
			if !strings.Contains(call.body, `"thinking":"The user <asked> about a visit & more."`) {
				t.Fatal("thinking text must not be html-escaped")
			}
			if sysMsg := msgs[3].(map[string]any); sysMsg["note"] != "kept" || sysMsg["content"] != "" {
				t.Fatalf("mid-conversation system entry changed: %v", sysMsg)
			}
			last := msgs[4].(map[string]any)["content"].([]any)[0].(map[string]any)
			if !reflect.DeepEqual(last["cache_control"], map[string]any{"type": "ephemeral"}) {
				t.Fatal("message cache_control lost")
			}
			if _, ok := msgs[2].(map[string]any)["content"].([]any)[0].(map[string]any)["content"].(string); !ok {
				t.Fatal("string tool_result content must stay a string")
			}

			var got struct {
				Content []map[string]any `json:"content"`
			}
			if err := json.Unmarshal(out, &got); err != nil {
				t.Fatal(err)
			}
			made := h.claude.made(0)
			if !reflect.DeepEqual(map[string]any(got.Content[0]), made[0]) {
				t.Fatalf("response thinking block changed: %v", got.Content[0])
			}
			if got.Content[1]["text"] != "ECHO: "+anthUserText {
				t.Fatalf("text not restored: %v", got.Content[1]["text"])
			}
			in := got.Content[2]["input"].(map[string]any)
			if in["body"] != anthUserText || in["nested"].([]any)[0] != anthUserText || in["count"] != float64(3) {
				t.Fatalf("tool input not restored: %v", in)
			}
			if resp.Header.Get("Request-Id") != "req_fake" || resp.Header.Get(HeaderRoute) != "masked" || resp.Header.Get(HeaderMasked) == "" {
				t.Fatalf("response headers: %v", resp.Header)
			}
			assertLogsClean(t, h.logs.String())
		})
	}
}

func TestAnthropicStreaming(t *testing.T) {
	h := newAnthHarness(t, "", "")
	h.claude.respond = func(_ map[string]any, last string) []any {
		return []any{
			map[string]any{"type": "thinking", "thinking": "considering " + last, "signature": "c2lnMg=="},
			map[string]any{"type": "text", "text": "ECHO: " + last},
			map[string]any{"type": "tool_use", "id": "toolu_9", "name": "send", "input": map[string]any{"body": last, "quote": "say \"hi\""}},
		}
	}
	resp, out := h.post(t, "/v1/messages?beta=true", anthBody(anthUserText, true), nil)
	if resp.StatusCode != 200 || !strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("status %d %s", resp.StatusCode, out)
	}
	call := h.claude.call(-1)
	assertNoReals(t, "upstream body", call.body)
	if call.header.Get("Anthropic-Version") != anthropicDefaultVersion {
		t.Fatalf("default anthropic-version not set: %q", call.header.Get("Anthropic-Version"))
	}

	rec := httptest.NewRecorder()
	made := h.claude.made(0)
	writeAnthSSE(rec, made)
	var wantNames, gotNames []string
	for _, ev := range parseAnthSSE(t, rec.Body.Bytes()) {
		wantNames = append(wantNames, ev.name)
	}
	events := parseAnthSSE(t, out)
	pings := 0
	for _, ev := range events {
		gotNames = append(gotNames, ev.name)
		if ev.name == "ping" {
			pings++
		}
	}
	if !reflect.DeepEqual(collapse(gotNames), collapse(wantNames)) {
		t.Fatalf("event order changed:\n got %v\nwant %v", collapse(gotNames), collapse(wantNames))
	}
	if pings != 2 {
		t.Fatalf("pings forwarded: %d", pings)
	}
	content := assembleAnth(t, events)
	if !reflect.DeepEqual(content[0], made[0]) {
		t.Fatalf("thinking block changed in stream: %v", content[0])
	}
	if got := content[1].(map[string]any)["text"]; got != "ECHO: "+anthUserText {
		t.Fatalf("stream text not restored: %q", got)
	}
	in := content[2].(map[string]any)["input"].(map[string]any)
	if in["body"] != anthUserText || in["quote"] != `say "hi"` {
		t.Fatalf("stream tool input not restored: %v", in)
	}
	assertLogsClean(t, h.logs.String())
}

func TestAnthropicStreamWithoutMessageStop(t *testing.T) {
	h := newAnthHarness(t, "", "")
	h.claude.srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		json.Unmarshal(body, &req)
		masked := lastText(req)
		w.Header().Set("Content-Type", "text/event-stream")
		start, _ := json.Marshal(map[string]any{"type": "content_block_start", "index": 0, "content_block": map[string]any{"type": "text", "text": "A: "}})
		delta, _ := json.Marshal(map[string]any{"type": "content_block_delta", "index": 0, "delta": map[string]any{"type": "text_delta", "text": masked}})
		fmt.Fprintf(w, "event: content_block_start\ndata: %s\n\nevent: future_event\ndata: {\"type\":\"future_event\",\"x\":  1}\n\nevent: content_block_delta\ndata: %s\n\n", start, delta)
	})
	resp, out := h.post(t, "/v1/messages", anthBody("Call John Smith", true), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d %s", resp.StatusCode, out)
	}
	if !strings.Contains(string(out), "event: future_event\ndata: {\"type\":\"future_event\",\"x\":  1}\n\n") {
		t.Fatalf("unknown event must pass through byte for byte: %s", out)
	}
	content := assembleAnth(t, parseAnthSSE(t, out))
	if got := content[0].(map[string]any)["text"]; got != "A: Call John Smith" {
		t.Fatalf("pending text not flushed at end of stream: %q", got)
	}
}

func TestAnthropicStability(t *testing.T) {
	const invented = "Sure. I will CC Priscilla Zyxwell (priscilla.v@example.org, 555-867-5309) on it. "
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprintf("stream=%v", stream), func(t *testing.T) {
			h := newAnthHarness(t, "", "")
			h.claude.respond = func(_ map[string]any, last string) []any {
				return []any{
					map[string]any{"type": "thinking", "thinking": "plan", "signature": "c2lnMw=="},
					map[string]any{"type": "text", "text": invented + "Regarding: " + last},
					map[string]any{"type": "tool_use", "id": "toolu_5", "name": "send", "input": map[string]any{"cc": "priscilla.v@example.org", "who": "Priscilla Zyxwell", "body": last}},
				}
			}
			first := map[string]any{"role": "user", "content": anthUserText}
			body := map[string]any{"model": "claude-x", "max_tokens": 64, "stream": stream, "messages": []any{first}}
			resp, out := h.post(t, "/v1/messages", body, nil)
			if resp.StatusCode != 200 {
				t.Fatalf("turn 1 status %d %s", resp.StatusCode, out)
			}
			var assistant []any
			if stream {
				assistant = assembleAnth(t, parseAnthSSE(t, out))
			} else {
				var got map[string]any
				json.Unmarshal(out, &got)
				assistant = got["content"].([]any)
			}
			if text := assistant[1].(map[string]any)["text"].(string); text != invented+"Regarding: "+anthUserText {
				t.Fatalf("turn 1 text not restored: %q", text)
			}
			body["messages"] = []any{first,
				map[string]any{"role": "assistant", "content": assistant},
				map[string]any{"role": "user", "content": []any{
					map[string]any{"type": "tool_result", "tool_use_id": "toolu_5", "content": "sent to John Smith"},
					map[string]any{"type": "text", "text": "Thanks, now tell Maria Garcia."},
				}},
			}
			resp, out = h.post(t, "/v1/messages", body, nil)
			if resp.StatusCode != 200 {
				t.Fatalf("turn 2 status %d %s", resp.StatusCode, out)
			}
			one, two := h.claude.call(0), h.claude.call(1)
			assertNoReals(t, "turn 2 upstream body", two.body)
			m1 := one.json(t)["messages"].([]any)
			m2 := two.json(t)["messages"].([]any)
			u1, _ := json.Marshal(m1[0])
			u2, _ := json.Marshal(m2[0])
			if !bytes.Equal(u1, u2) {
				t.Fatalf("user turn drifted:\n%s\n%s", u1, u2)
			}
			produced, _ := json.Marshal(h.claude.made(0))
			replayed, _ := json.Marshal(m2[1].(map[string]any)["content"])
			if !bytes.Equal(produced, replayed) {
				t.Fatalf("assistant turn drifted:\nproduced %s\nreplayed %s", produced, replayed)
			}
		})
	}
}

func TestAnthropicAuth(t *testing.T) {
	t.Run("forward_auth off", func(t *testing.T) {
		h := newAnthHarness(t, "", "")
		resp, out := h.post(t, "/v1/messages", anthBody("hi", false), map[string]string{"x-api-key": "client-key", "Authorization": "Bearer client-token"})
		if resp.StatusCode != 200 {
			t.Fatalf("status %d %s", resp.StatusCode, out)
		}
		call := h.claude.call(-1)
		if call.header.Get("X-Api-Key") != "sk-upstream-env-key" || call.header.Get("Authorization") != "" {
			t.Fatalf("upstream auth: x-api-key=%q authorization=%q", call.header.Get("X-Api-Key"), call.header.Get("Authorization"))
		}
		if strings.Contains(fmt.Sprint(call.header), "client-") {
			t.Fatal("client credentials reached upstream")
		}
	})
	t.Run("forward_auth on", func(t *testing.T) {
		t.Setenv("OUTWORLD_TOKEN", "gw-token")
		h := newAnthHarness(t, "forward_auth: true, api_key_env: CLAUDE_KEY", "")
		resp, out := h.post(t, "/v1/messages", anthBody("hi", false), map[string]string{"Authorization": "Bearer sub-token", "X-Outworld-Token": "gw-token"})
		if resp.StatusCode != 200 {
			t.Fatalf("status %d %s", resp.StatusCode, out)
		}
		call := h.claude.call(-1)
		if call.header.Get("Authorization") != "Bearer sub-token" || call.header.Get("X-Api-Key") != "" || call.header.Get("X-Outworld-Token") != "" {
			t.Fatalf("upstream headers: %v", call.header)
		}
		resp, _ = h.post(t, "/v1/messages", anthBody("hi", false), map[string]string{"x-api-key": "client-key", "X-Outworld-Token": "gw-token"})
		if call = h.claude.call(-1); resp.StatusCode != 200 || call.header.Get("X-Api-Key") != "client-key" {
			t.Fatalf("x-api-key not forwarded: %v", call.header)
		}
		resp, _ = h.post(t, "/v1/messages", anthBody("hi", false), map[string]string{"x-api-key": "gw-token"})
		if call = h.claude.call(-1); resp.StatusCode != 200 || strings.Contains(fmt.Sprint(call.header), "gw-token") {
			t.Fatalf("gateway token must never reach upstream: %v", call.header)
		}
	})
	t.Run("gateway token", func(t *testing.T) {
		t.Setenv("OUTWORLD_TOKEN", "gw-token")
		h := newAnthHarness(t, "", "")
		for _, hdr := range []map[string]string{
			{"Authorization": "Bearer gw-token"},
			{"x-api-key": "gw-token"},
			{"X-Outworld-Token": "gw-token"},
		} {
			if resp, out := h.post(t, "/v1/messages", anthBody("hi", false), hdr); resp.StatusCode != 200 {
				t.Fatalf("token via %v rejected: %d %s", hdr, resp.StatusCode, out)
			}
			if resp, _ := h.post(t, "/v1/chat/completions", chatBody("hi", false), hdr); resp.StatusCode != 200 {
				t.Fatalf("openai endpoint rejected token via %v", hdr)
			}
		}
		before := h.claude.count()
		for _, hdr := range []map[string]string{nil, {"x-api-key": "wrong"}, {"X-Outworld-Token": "gw-toke"}} {
			resp, out := h.post(t, "/v1/messages", anthBody("hi", false), hdr)
			assertAnthError(t, resp, out, 401, "outworld_unauthorized")
		}
		if h.claude.count() != before {
			t.Fatal("unauthorized request reached upstream")
		}
	})
}

func TestAnthropicNonTextContent(t *testing.T) {
	cases := []struct {
		name   string
		block  map[string]any
		status int
	}{
		{"image", map[string]any{"type": "image", "source": map[string]any{"type": "base64", "media_type": "image/png", "data": "AAAA"}}, 422},
		{"pdf document", map[string]any{"type": "document", "source": map[string]any{"type": "base64", "media_type": "application/pdf", "data": "AAAA"}}, 422},
		{"unknown block with source", map[string]any{"type": "future_media", "source": map[string]any{"type": "url", "url": "https://example.org/x"}}, 422},
		{"nested image in tool_result", map[string]any{"type": "tool_result", "tool_use_id": "t", "content": []any{map[string]any{"type": "image", "source": map[string]any{"type": "base64", "data": "AAAA"}}}}, 422},
		{"text document", map[string]any{"type": "document", "title": "Notes on John Smith", "source": map[string]any{"type": "text", "media_type": "text/plain", "data": "John Smith lives at 123 Main Street"}}, 200},
		{"unknown block without source", map[string]any{"type": "future_block", "payload": map[string]any{"k": "v"}}, 200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newAnthHarness(t, "", "")
			content := []any{map[string]any{"type": "text", "text": "what is this, John?"}, tc.block}
			resp, out := h.post(t, "/v1/messages", anthBody(content, false), nil)
			if tc.status == 422 {
				assertAnthError(t, resp, out, 422, "outworld_unsupported_content")
				if h.claude.count() != 0 {
					t.Fatal("refused request reached upstream")
				}
				return
			}
			if resp.StatusCode != 200 {
				t.Fatalf("status %d %s", resp.StatusCode, out)
			}
			assertNoReals(t, "upstream body", h.claude.call(-1).body)
			if tc.name == "unknown block without source" && !strings.Contains(h.claude.call(-1).body, `"payload":{"k":"v"}`) {
				t.Fatal("unknown block not passed through")
			}
		})
	}
	t.Run("allow_images", func(t *testing.T) {
		h := newAnthHarness(t, "", "  allow_images: true\n")
		png := map[string]any{"type": "image", "source": map[string]any{"type": "base64", "media_type": "image/png", "data": pngB64}}
		resp, out := h.post(t, "/v1/messages", anthBody([]any{png}, false), nil)
		if resp.StatusCode != 200 || !strings.Contains(h.claude.call(-1).body, pngB64) {
			t.Fatalf("status %d %s", resp.StatusCode, out)
		}
		resp, out = h.post(t, "/v1/messages", anthBody([]any{cases[0].block}, false), nil)
		assertAnthError(t, resp, out, 422, "outworld_unsupported_content")
	})
}

func TestAnthropicRouting(t *testing.T) {
	h := newAnthHarness(t, "", "  rules:\n    - {match_regex: [\"(?i)top[- ]secret\"], route: block}\n  allow_route_header: true\n")
	resp, out := h.post(t, "/v1/messages", anthBody("hi", false), map[string]string{HeaderRoute: "passthrough:oai"})
	assertAnthError(t, resp, out, 400, "outworld_route_error")
	resp, out = h.post(t, "/v1/messages", anthBody("hi", false), map[string]string{HeaderRoute: "local"})
	assertAnthError(t, resp, out, 400, "outworld_route_error")
	if h.oai.calls() != 0 || h.claude.count() != 0 || h.raw.count() != 0 {
		t.Fatal("mismatched route reached an upstream")
	}
	oresp := (&harness{gw: h.gw}).post(t, "/v1/chat/completions", chatBody("hi", false), map[string]string{HeaderRoute: "passthrough:claude"})
	ob, _ := io.ReadAll(oresp.Body)
	if oresp.StatusCode != 400 || !strings.Contains(string(ob), "outworld_route_error") || h.claude.count() != 0 {
		t.Fatalf("openai request to anthropic upstream: %d %s", oresp.StatusCode, ob)
	}

	resp, out = h.post(t, "/v1/messages", map[string]any{"model": "m", "max_tokens": 1, "system": []any{map[string]any{"type": "text", "text": "this is top secret"}},
		"messages": []any{map[string]any{"role": "user", "content": "hi"}}}, nil)
	assertAnthError(t, resp, out, 403, "outworld_blocked")

	resp, out = h.post(t, "/v1/messages?beta=true", anthBody("hello John Smith", false), map[string]string{HeaderRoute: "passthrough", "anthropic-beta": "b1", "x-api-key": "client-key"})
	if resp.StatusCode != 200 || resp.Header.Get(HeaderRoute) != "passthrough" || h.raw.count() != 1 {
		t.Fatalf("passthrough: %d %s", resp.StatusCode, out)
	}
	call := h.raw.call(-1)
	if !strings.Contains(call.body, "John Smith") || call.query != "beta=true" || call.header.Get("Anthropic-Beta") != "b1" || call.header.Get("X-Api-Key") != "" {
		t.Fatalf("passthrough upstream request: %s %v", call.body, call.header)
	}
	if !strings.Contains(string(out), "ECHO: hello John Smith") {
		t.Fatalf("passthrough response: %s", out)
	}
}

func TestAnthropicCountTokens(t *testing.T) {
	h := newAnthHarness(t, "", "")
	body := map[string]any{"model": "claude-x", "system": "You assist John Smith.", "messages": []any{map[string]any{"role": "user", "content": anthUserText}}}
	resp, out := h.post(t, "/v1/messages/count_tokens?beta=true", body, nil)
	if resp.StatusCode != 200 || string(out) != `{ "input_tokens": 42 }` {
		t.Fatalf("status %d body %q", resp.StatusCode, out)
	}
	call := h.claude.call(-1)
	if call.path != "/v1/messages/count_tokens" || call.query != "beta=true" {
		t.Fatalf("upstream url %s?%s", call.path, call.query)
	}
	assertNoReals(t, "count_tokens upstream body", call.body)
}

func TestAnthropicUpstreamErrorPassthrough(t *testing.T) {
	h := newAnthHarness(t, "", "")
	const errBody = `{"type":"error",  "error":{"type":"invalid_request_error","message":"messages.1.content.0: thinking blocks <cannot> be modified & are bound to a different conversation"},"request_id":"req_fake"}` + "\n"
	h.claude.failBody, h.claude.failCode = errBody, 400
	for _, stream := range []bool{false, true} {
		resp, out := h.post(t, "/v1/messages", anthBody("Call John Smith", stream), nil)
		if resp.StatusCode != 400 || string(out) != errBody {
			t.Fatalf("stream=%v: status %d body %q", stream, resp.StatusCode, out)
		}
		for k, v := range map[string]string{"Retry-After": "7", "Request-Id": "req_fake", "X-Should-Retry": "false", "Anthropic-Ratelimit-Requests-Remaining": "3"} {
			if resp.Header.Get(k) != v {
				t.Fatalf("response header %s = %q", k, resp.Header.Get(k))
			}
		}
	}
}

func TestOpenAIStability(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprintf("stream=%v", stream), func(t *testing.T) {
			h := newHarness(t, "")
			h.cloud.toolArgs = func(string) string { return `{"cc": "priscilla.v@example.org"}` }
			first := map[string]any{"role": "user", "content": "Email John Smith at john.smith@gmail.com"}
			body := map[string]any{"model": "m", "stream": stream, "messages": []any{first}}
			resp := h.post(t, "/v1/chat/completions", body, nil)
			var content, args string
			if stream {
				content, args, _ = readSSE(t, resp.Body)
			} else {
				var got struct {
					Choices []struct {
						Message struct {
							Content   string `json:"content"`
							ToolCalls []struct {
								Function struct {
									Arguments string `json:"arguments"`
								} `json:"function"`
							} `json:"tool_calls"`
						} `json:"message"`
					} `json:"choices"`
				}
				json.NewDecoder(resp.Body).Decode(&got)
				content, args = got.Choices[0].Message.Content, got.Choices[0].Message.ToolCalls[0].Function.Arguments
			}
			upstreamContent := "ECHO: " + h.cloud.lastUser()
			if !strings.Contains(content, "John Smith") {
				t.Fatalf("turn 1 not restored: %q", content)
			}
			body["messages"] = []any{first,
				map[string]any{"role": "assistant", "content": content, "tool_calls": []any{map[string]any{"id": "t1", "type": "function", "function": map[string]any{"name": "send_email", "arguments": args}}}},
				map[string]any{"role": "tool", "tool_call_id": "t1", "content": "ok"},
			}
			resp = h.post(t, "/v1/chat/completions", body, nil)
			if resp.StatusCode != 200 {
				t.Fatalf("turn 2 status %d", resp.StatusCode)
			}
			h.cloud.mu.Lock()
			var req map[string]any
			json.Unmarshal([]byte(h.cloud.bodies[len(h.cloud.bodies)-1]), &req)
			h.cloud.mu.Unlock()
			asst := req["messages"].([]any)[1].(map[string]any)
			if asst["content"] != upstreamContent {
				t.Fatalf("assistant content drifted: %q vs %q", asst["content"], upstreamContent)
			}
			gotArgs := asst["tool_calls"].([]any)[0].(map[string]any)["function"].(map[string]any)["arguments"].(string)
			if !strings.Contains(gotArgs, "priscilla.v@example.org") {
				t.Fatalf("model-invented argument was re-masked: %s", gotArgs)
			}
		})
	}
}

func TestHelloProbe(t *testing.T) {
	t.Setenv("OUTWORLD_TOKEN", "gw-token")
	h := newAnthHarness(t, "", "")
	resp, err := http.Head(h.gw.URL + "/api/hello")
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("hello probe: %v %v", resp, err)
	}
	b, _ := io.ReadAll(resp.Body)
	if len(b) != 0 {
		t.Fatal("hello probe must have no body")
	}
}
