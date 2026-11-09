package gateway

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/davidparkercodes/outworld/internal/app"
	"github.com/davidparkercodes/outworld/internal/config"
)

const people = `value,type,aliases
John Smith,person,Johnny
Maria Garcia,person,
Acme Health,org,
`

var realValues = []string{"John", "Smith", "Johnny", "Maria", "Garcia", "john.smith@gmail.com", "804) 555-1234", "123 Main Street", "482913", "123-45-6789", "March 15, 2024"}

const userText = "Hi, I'm John Smith (Johnny to friends). My email is john.smith@gmail.com, phone (804) 555-1234, " +
	"I live at 123 Main Street. Maria Garcia's visit is on March 15, 2024. Code 482913, SSN 123-45-6789. I work at Acme Health."

type fakeUpstream struct {
	mu       sync.Mutex
	bodies   []string
	auth     []string
	srv      *httptest.Server
	name     string
	toolArgs func(user string) string
}

func newFake(t *testing.T, name string) *fakeUpstream {
	f := &fakeUpstream{name: name}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeUpstream) lastUser() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var req map[string]any
	json.Unmarshal([]byte(f.bodies[len(f.bodies)-1]), &req)
	msgs := req["messages"].([]any)
	return msgs[len(msgs)-1].(map[string]any)["content"].(string)
}

func (f *fakeUpstream) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.bodies)
}

func (f *fakeUpstream) serve(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/v1/models" {
		json.NewEncoder(w).Encode(map[string]any{"object": "list", "data": []any{map[string]any{"id": f.name + "-model"}}})
		return
	}
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	f.bodies = append(f.bodies, string(body))
	f.auth = append(f.auth, r.Header.Get("Authorization"))
	f.mu.Unlock()
	var req map[string]any
	json.Unmarshal(body, &req)
	user := ""
	if msgs, ok := req["messages"].([]any); ok && len(msgs) > 0 {
		user, _ = msgs[len(msgs)-1].(map[string]any)["content"].(string)
	}
	reply := "ECHO: " + user
	args := ""
	if f.toolArgs != nil {
		args = f.toolArgs(user)
	}
	if stream, _ := req["stream"].(bool); stream {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		send := func(delta map[string]any, finish any) {
			chunk := map[string]any{"id": "c1", "object": "chat.completion.chunk", "created": 1, "model": "m",
				"choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": finish}}}
			b, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "data: %s\n\n", b)
			fl.Flush()
		}
		send(map[string]any{"role": "assistant"}, nil)
		for i, step := 0, 1; i < len(reply); i, step = i+step, step%3+1 {
			end := min(i+step, len(reply))
			send(map[string]any{"content": reply[i:end]}, nil)
		}
		if args != "" {
			send(map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": "t1", "type": "function", "function": map[string]any{"name": "send_email", "arguments": ""}}}}, nil)
			for i := 0; i < len(args); i += 2 {
				end := min(i+2, len(args))
				send(map[string]any{"tool_calls": []any{map[string]any{"index": 0, "function": map[string]any{"arguments": args[i:end]}}}}, nil)
			}
		}
		send(map[string]any{}, "stop")
		fmt.Fprint(w, "data: [DONE]\n\n")
		return
	}
	msg := map[string]any{"role": "assistant", "content": reply}
	if args != "" {
		msg["tool_calls"] = []any{map[string]any{"id": "t1", "type": "function", "function": map[string]any{"name": "send_email", "arguments": args}}}
	}
	json.NewEncoder(w).Encode(map[string]any{"id": "x", "object": "chat.completion", "choices": []any{map[string]any{"index": 0, "message": msg, "finish_reason": "stop"}}})
}

type harness struct {
	gw    *httptest.Server
	cloud *fakeUpstream
	local *fakeUpstream
	tee   *fakeUpstream
	logs  *bytes.Buffer
}

func newHarness(t *testing.T, extra string) *harness {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "people.csv"), []byte(people), 0o600)
	h := &harness{cloud: newFake(t, "cloud"), local: newFake(t, "local"), tee: newFake(t, "tee"), logs: &bytes.Buffer{}}
	t.Setenv("OUTWORLD_VAULT_KEY", strings.Repeat("ab", 32))
	t.Setenv("CLOUD_API_KEY", "cloud-secret-key")
	cfgText := fmt.Sprintf(`listen: 127.0.0.1:0
vault:
  path: ./outworld.vault
  key_env: OUTWORLD_VAULT_KEY
upstreams:
  cloud: {base_url: %s/v1, api_key_env: CLOUD_API_KEY}
  local: {base_url: %s/v1}
  tee: {base_url: %s/v1}
policy:
  default_route: masked
  masked_upstream: cloud
  local_upstream: local
  passthrough_upstream: tee
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
    secret: strip
    gov_id: strip
    financial: strip
  rules:
    - {match_any: ["diagnosis", "lawsuit"], route: local}
    - {match_regex: ["(?i)top[- ]secret"], route: block}
    - {entities: ["financial"], route: passthrough, name: "money"}
%s`, h.cloud.srv.URL, h.local.srv.URL, h.tee.srv.URL, extra)
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

func (h *harness) post(t *testing.T, path string, body any, hdr map[string]string) *http.Response {
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
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func chatBody(content string, stream bool) map[string]any {
	return map[string]any{"model": "gpt-x", "stream": stream, "user": "john.smith@gmail.com", "messages": []any{
		map[string]any{"role": "system", "content": "You are helpful."},
		map[string]any{"role": "user", "content": content},
	}}
}

func assertClean(t *testing.T, f *fakeUpstream) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, b := range f.bodies {
		for _, r := range realValues {
			if strings.Contains(b, r) {
				t.Fatalf("real value %q reached upstream: %s", r, b)
			}
		}
	}
}

func assertLogsClean(t *testing.T, logs string) {
	t.Helper()
	for _, r := range append(realValues, "Hi, I'm", "ECHO") {
		if strings.Contains(logs, r) {
			t.Fatalf("logs contain %q: %s", r, logs)
		}
	}
}

func TestMaskedNonStreaming(t *testing.T) {
	h := newHarness(t, "")
	h.cloud.toolArgs = func(user string) string {
		i := strings.Index(user, "email is ")
		email := strings.SplitN(user[i+len("email is "):], ",", 2)[0]
		b, _ := json.Marshal(map[string]any{"to": email, "note": "for " + strings.Fields(user)[2]})
		return string(b)
	}
	resp := h.post(t, "/v1/chat/completions", chatBody(userText, false), nil)
	if resp.StatusCode != 200 || resp.Header.Get(HeaderRoute) != "masked" {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d route %q body %s", resp.StatusCode, resp.Header.Get(HeaderRoute), b)
	}
	assertClean(t, h.cloud)
	if h.cloud.auth[0] != "Bearer cloud-secret-key" {
		t.Fatalf("upstream auth %q", h.cloud.auth[0])
	}
	if strings.Contains(h.cloud.bodies[0], `"user":"`) {
		t.Fatal("user field must be dropped")
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					Function struct{ Arguments string } `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	want := "ECHO: " + strings.NewReplacer("482913", "[redacted code]", "123-45-6789", "[redacted id]").Replace(userText)
	if got := out.Choices[0].Message.Content; got != want {
		t.Fatalf("rehydrated content\n got: %s\nwant: %s", got, want)
	}
	var args map[string]string
	json.Unmarshal([]byte(out.Choices[0].Message.ToolCalls[0].Function.Arguments), &args)
	if args["to"] != "john.smith@gmail.com" || args["note"] != "for John" {
		t.Fatalf("tool args %v", args)
	}
	assertLogsClean(t, h.logs.String())
}

func readSSE(t *testing.T, r io.Reader) (string, string, []string) {
	t.Helper()
	var content, args strings.Builder
	var comments []string
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, ":") {
			comments = append(comments, line)
			continue
		}
		if !strings.HasPrefix(line, "data: ") || line == "data: [DONE]" {
			continue
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Function struct{ Arguments string } `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(line[6:]), &chunk); err != nil {
			t.Fatalf("bad chunk %q: %v", line, err)
		}
		for _, c := range chunk.Choices {
			content.WriteString(c.Delta.Content)
			for _, tc := range c.Delta.ToolCalls {
				args.WriteString(tc.Function.Arguments)
			}
		}
	}
	return content.String(), args.String(), comments
}

func TestMaskedStreaming(t *testing.T) {
	h := newHarness(t, "")
	h.cloud.toolArgs = func(user string) string {
		b, _ := json.Marshal(map[string]any{"who": strings.Fields(user)[2] + " " + strings.TrimSuffix(strings.Fields(user)[3], "")})
		return string(b)
	}
	text := userText + " Ignatius Wetherby was mentioned by the model."
	resp := h.post(t, "/v1/chat/completions", chatBody(text, true), nil)
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d %s", resp.StatusCode, b)
	}
	content, args, comments := readSSE(t, resp.Body)
	assertClean(t, h.cloud)
	want := "ECHO: " + strings.NewReplacer("482913", "[redacted code]", "123-45-6789", "[redacted id]").Replace(text)
	if content != want {
		t.Fatalf("stream content\n got: %s\nwant: %s", content, want)
	}
	var a map[string]string
	if err := json.Unmarshal([]byte(args), &a); err != nil || a["who"] != "John Smith" {
		t.Fatalf("stream args %q %v", args, err)
	}
	if len(comments) != 1 || !strings.Contains(comments[0], "Ignatius") || !strings.Contains(comments[0], "Wetherby") {
		t.Fatalf("unknown names comment: %v", comments)
	}
	if resp.Trailer.Get(HeaderUnknownNames) == "" {
		t.Fatal("expected unknown names trailer")
	}
}

func TestRouting(t *testing.T) {
	h := newHarness(t, "")
	resp := h.post(t, "/v1/chat/completions", chatBody("John Smith has a diagnosis of flu", false), nil)
	if resp.Header.Get(HeaderRoute) != "local" || h.local.calls() != 1 || h.cloud.calls() != 0 {
		t.Fatalf("route %q local=%d cloud=%d", resp.Header.Get(HeaderRoute), h.local.calls(), h.cloud.calls())
	}
	if !strings.Contains(h.local.lastUser(), "John Smith") {
		t.Fatal("local route should receive raw text")
	}

	resp = h.post(t, "/v1/chat/completions", chatBody("this is TOP SECRET from John", false), map[string]string{HeaderRoute: "passthrough"})
	if resp.StatusCode != 403 || resp.Header.Get(HeaderRoute) != "block" || h.tee.calls() != 0 {
		t.Fatalf("block rule must win over header: %d %q", resp.StatusCode, resp.Header.Get(HeaderRoute))
	}

	resp = h.post(t, "/v1/chat/completions", chatBody("hello John", false), map[string]string{HeaderRoute: "passthrough"})
	if resp.Header.Get(HeaderRoute) != "passthrough" || resp.Header.Get(HeaderRouteReason) != "header" || h.tee.calls() != 1 {
		t.Fatalf("header passthrough: %q", resp.Header.Get(HeaderRoute))
	}

	resp = h.post(t, "/v1/chat/completions", chatBody("pay with 4111 1111 1111 1111", false), nil)
	if resp.Header.Get(HeaderRoute) != "passthrough" || resp.Header.Get(HeaderRouteReason) != "rule:money" {
		t.Fatalf("entity rule: %q %q", resp.Header.Get(HeaderRoute), resp.Header.Get(HeaderRouteReason))
	}

	resp = h.post(t, "/v1/chat/completions", chatBody("hi", false), map[string]string{HeaderRoute: "bogus"})
	if resp.StatusCode == 200 {
		t.Fatal("invalid header route must fail")
	}
	resp = h.post(t, "/v1/chat/completions", chatBody("hi", false), map[string]string{HeaderRoute: "passthrough:nope"})
	if resp.StatusCode == 200 {
		t.Fatal("unknown upstream must fail")
	}
}

func TestLeakBlocksRequest(t *testing.T) {
	h := newHarness(t, "")
	body := map[string]any{"model": "m", "messages": []any{
		map[string]any{"role": "user", "name": "maria.garcia@corp.test", "content": "hello"},
	}}
	resp := h.post(t, "/v1/chat/completions", body, nil)
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 422 || h.cloud.calls() != 0 || !strings.Contains(string(b), "outworld_leak_blocked") {
		t.Fatalf("status %d calls %d body %s", resp.StatusCode, h.cloud.calls(), b)
	}
	if strings.Contains(string(b), "maria") || strings.Contains(h.logs.String(), "maria") {
		t.Fatal("leak error leaked the value")
	}
}

func TestImagesBlockedOnMaskedRoute(t *testing.T) {
	h := newHarness(t, "")
	body := map[string]any{"model": "m", "messages": []any{map[string]any{"role": "user", "content": []any{
		map[string]any{"type": "text", "text": "what is this, John?"},
		map[string]any{"type": "image_url", "image_url": map[string]any{"url": "data:image/png;base64,AAAA"}},
	}}}}
	resp := h.post(t, "/v1/chat/completions", body, nil)
	if resp.StatusCode != 422 || h.cloud.calls() != 0 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	body["messages"].([]any)[0].(map[string]any)["content"] = []any{map[string]any{"type": "text", "text": "what is this, John?"}}
	resp = h.post(t, "/v1/chat/completions", body, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("text parts should pass: %d", resp.StatusCode)
	}
	assertClean(t, h.cloud)
}

func TestToolMessagesAndArgumentsMasked(t *testing.T) {
	h := newHarness(t, "")
	body := map[string]any{"model": "m", "messages": []any{
		map[string]any{"role": "user", "content": "email Maria"},
		map[string]any{"role": "assistant", "content": nil, "tool_calls": []any{map[string]any{"id": "c", "type": "function",
			"function": map[string]any{"name": "lookup", "arguments": `{"name":"Maria Garcia","phone":"(804) 555-1234"}`}}}},
		map[string]any{"role": "tool", "tool_call_id": "c", "content": `{"email":"john.smith@gmail.com","address":"123 Main Street"}`},
	}}
	resp := h.post(t, "/v1/chat/completions", body, nil)
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d %s", resp.StatusCode, b)
	}
	assertClean(t, h.cloud)
}

func TestAuthAndHealthAndModels(t *testing.T) {
	t.Setenv("OUTWORLD_TOKEN", "letmein")
	h := newHarness(t, "")
	if resp, _ := http.Get(h.gw.URL + "/healthz"); resp.StatusCode != 200 {
		t.Fatal("healthz")
	}
	resp := h.post(t, "/v1/chat/completions", chatBody("hi", false), nil)
	if resp.StatusCode != 401 || h.cloud.calls() != 0 {
		t.Fatalf("auth not enforced: %d", resp.StatusCode)
	}
	resp = h.post(t, "/v1/chat/completions", chatBody("hi", false), map[string]string{"Authorization": "Bearer wrong"})
	if resp.StatusCode != 401 {
		t.Fatal("wrong token accepted")
	}
	resp = h.post(t, "/v1/chat/completions", chatBody("hi", false), map[string]string{"Authorization": "Bearer letmein"})
	if resp.StatusCode != 200 || h.cloud.auth[0] != "Bearer cloud-secret-key" {
		t.Fatalf("authorized request: %d, upstream saw %q", resp.StatusCode, h.cloud.auth[0])
	}
	req, _ := http.NewRequest(http.MethodGet, h.gw.URL+"/v1/models", nil)
	req.Header.Set("Authorization", "Bearer letmein")
	mr, err := http.DefaultClient.Do(req)
	if err != nil || mr.StatusCode != 200 {
		t.Fatalf("models %v", err)
	}
	b, _ := io.ReadAll(mr.Body)
	if !strings.Contains(string(b), "cloud-model") {
		t.Fatalf("models body %s", b)
	}
}

func TestCompletionsPromptMasked(t *testing.T) {
	h := newHarness(t, "")
	resp := h.post(t, "/v1/completions", map[string]any{"model": "m", "prompt": "Write to John Smith at john.smith@gmail.com"}, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	assertClean(t, h.cloud)
}
