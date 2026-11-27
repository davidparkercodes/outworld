package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davidparkercodes/outworld/internal/app"
	"github.com/davidparkercodes/outworld/internal/config"
	"github.com/davidparkercodes/outworld/pkg/outworld"
)

type recordedCall struct {
	host, path, query string
	body              string
}

type fnTransport struct {
	mu    sync.Mutex
	calls []recordedCall
	reply func(host string, req map[string]any) (*http.Response, error)
}

func (f *fnTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	f.calls = append(f.calls, recordedCall{host: r.URL.Host, path: r.URL.Path, query: r.URL.RawQuery, body: string(body)})
	f.mu.Unlock()
	var req map[string]any
	json.Unmarshal(body, &req)
	return f.reply(r.URL.Host, req)
}

func (f *fnTransport) all() []recordedCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]recordedCall(nil), f.calls...)
}

func jsonResponse(v any) *http.Response {
	b, _ := json.Marshal(v)
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(bytes.NewReader(b))}
}

func okReply(_ string, _ map[string]any) (*http.Response, error) {
	return jsonResponse(map[string]any{"id": "x", "choices": []any{map[string]any{"index": 0, "message": map[string]any{"role": "assistant", "content": "ok"}}}}), nil
}

func newTestServer(t *testing.T, policy string, tr *fnTransport) *httptest.Server {
	t.Helper()
	return newTestServerDefault(t, "masked", policy, tr)
}

func newTestServerDefault(t *testing.T, defaultRoute, policy string, tr *fnTransport) *httptest.Server {
	t.Helper()
	gw := httptest.NewServer(newTestGateway(t, defaultRoute, policy, tr).Handler())
	t.Cleanup(gw.Close)
	return gw
}

func newTestGateway(t *testing.T, defaultRoute, policy string, tr *fnTransport) *Server {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "people.csv"), []byte(people), 0o600)
	t.Setenv("OUTWORLD_VAULT_KEY", strings.Repeat("ef", 32))
	cfgText := `listen: 127.0.0.1:0
vault:
  path: ./outworld.vault
  key_env: OUTWORLD_VAULT_KEY
upstreams:
  cloud: {base_url: http://cloud.test/v1}
  local: {base_url: http://local.test/v1}
  MyUp: {base_url: http://myup.test/v1}
  claude: {api: anthropic, base_url: http://claude.test/v1}
policy:
  default_route: ` + defaultRoute + `
  masked_upstream: cloud
  local_upstream: local
  passthrough_upstream: local
  anthropic: {masked_upstream: claude}
  date_shift: true
  dictionary: ./people.csv
  entities: {person: mask, email: mask, phone: mask, address: mask, date: shift, org: keep, location: keep, secret: strip, gov_id: strip, financial: strip}
` + policy
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
	t.Cleanup(func() { eng.Vault().Close() })
	srv, err := New(cfg, eng, Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Transport: tr})
	if err != nil {
		t.Fatal(err)
	}
	return srv
}

func postJSON(t *testing.T, url string, body any, hdr map[string]string) (*http.Response, []byte) {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, out
}

const blockRules = `  rules:
    - {name: secret, match_any: ["topsecret"], route: block}
    - {name: ids, entities: ["gov_id"], route: block}
`

func TestBlockRulesSeeInputAndSuffix(t *testing.T) {
	tr := &fnTransport{reply: okReply}
	gw := newTestServer(t, blockRules, tr)
	for _, field := range []string{"input", "suffix", "prompt"} {
		for _, val := range []string{"this is topsecret", "my SSN is 123-45-6789"} {
			body := map[string]any{"model": "m", "prompt": "hello"}
			body[field] = val
			resp, out := postJSON(t, gw.URL+"/v1/completions", body, nil)
			if resp.StatusCode != http.StatusForbidden || resp.Header.Get(HeaderRoute) != config.RouteBlock {
				t.Fatalf("%s=%q: status %d %s", field, val, resp.StatusCode, out)
			}
		}
	}
	if n := len(tr.all()); n != 0 {
		t.Fatalf("blocked requests reached upstream %d times", n)
	}
}

func bareServer(policy config.PolicyConfig) *Server {
	return &Server{cfg: &config.Config{Policy: policy}}
}

func TestRoutingTextIsDeterministic(t *testing.T) {
	s := bareServer(config.PolicyConfig{})
	args := `{"zeta":"z","alpha":"x","mid":"m","beta":"b","gamma":"g","delta":"d"}`
	req := map[string]any{"messages": []any{map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"function": map[string]any{
		"name": "n", "arguments": args,
	}}}}}}
	first := s.routingText(req, config.APIOpenAI).full
	if first != "alpha\nx\nbeta\nb\ndelta\nd\ngamma\ng\nmid\nm\nzeta\nz" {
		t.Fatalf("order: %q", first)
	}
	areq := map[string]any{"messages": []any{map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "tool_use", "input": map[string]any{
		"zeta": "z", "alpha": "x", "mid": "m", "beta": "b", "gamma": "g", "delta": "d",
	}}}}}}
	afirst := s.routingText(areq, config.APIAnthropic).full
	if afirst != "alpha\nx\nbeta\nb\ndelta\nd\ngamma\ng\nmid\nm\nzeta\nz" {
		t.Fatalf("anthropic order: %q", afirst)
	}
	for range 50 {
		if s.routingText(req, config.APIOpenAI).full != first || s.routingText(areq, config.APIAnthropic).full != afirst {
			t.Fatal("routing text must not depend on map order")
		}
	}
}

func TestOpenAIForwardsQueryString(t *testing.T) {
	tr := &fnTransport{reply: okReply}
	gw := newTestServer(t, "  allow_route_header: true\n", tr)
	postJSON(t, gw.URL+"/v1/chat/completions?api-version=2024-10-01", chatBody("hi John Smith", false), nil)
	postJSON(t, gw.URL+"/v1/chat/completions?api-version=2024-10-01", chatBody("hi", false), map[string]string{HeaderRoute: "local"})
	calls := tr.all()
	if len(calls) != 2 {
		t.Fatalf("calls %d", len(calls))
	}
	for _, c := range calls {
		if c.query != "api-version=2024-10-01" || c.path != "/v1/chat/completions" {
			t.Fatalf("upstream got %s?%s", c.path, c.query)
		}
	}
}

func TestRouteHeaderKeepsUpstreamCase(t *testing.T) {
	tr := &fnTransport{reply: okReply}
	gw := newTestServer(t, "  allow_route_header: true\n", tr)
	resp, out := postJSON(t, gw.URL+"/v1/chat/completions", chatBody("hi", false), map[string]string{HeaderRoute: "PassThrough:MyUp"})
	if resp.StatusCode != 200 || resp.Header.Get(HeaderRoute) != config.RoutePassthrough {
		t.Fatalf("status %d %s", resp.StatusCode, out)
	}
	if calls := tr.all(); len(calls) != 1 || calls[0].host != "myup.test" {
		t.Fatalf("calls %+v", calls)
	}
}

func TestRouteHeaderDefaultOnlyTightens(t *testing.T) {
	tr := &fnTransport{reply: okReply}
	gw := newTestServer(t, "  rules:\n    - {match_any: [\"diagnosis\"], route: local}\n    - {match_any: [\"invoice\"], route: masked, upstream: MyUp}\n", tr)
	post := func(text, h string) (*http.Response, []byte) {
		return postJSON(t, gw.URL+"/v1/chat/completions", chatBody(text, false), map[string]string{HeaderRoute: h})
	}
	for _, c := range []struct {
		text, header string
		status       int
	}{
		{"hi John Smith", "bogus", http.StatusBadRequest},
		{"hi John Smith", "passthrough", http.StatusForbidden},
		{"hi John Smith", "masked:MyUp", http.StatusForbidden},
		{"hi John Smith", "local:MyUp", http.StatusForbidden},
		{"John Smith diagnosis", "masked", http.StatusForbidden},
		{"John Smith diagnosis", "passthrough", http.StatusForbidden},
	} {
		resp, out := post(c.text, c.header)
		if resp.StatusCode != c.status || !strings.Contains(string(out), "outworld_route_error") {
			t.Fatalf("%q + %s: status %d %s", c.text, c.header, resp.StatusCode, out)
		}
	}
	if n := len(tr.all()); n != 0 {
		t.Fatalf("refused header reached upstream %d times", n)
	}

	resp, _ := post("John Smith diagnosis", "local")
	if resp.StatusCode != 200 || resp.Header.Get(HeaderRoute) != config.RouteLocal || resp.Header.Get(HeaderRouteReason) != "rule:0" {
		t.Fatalf("equal rank keeps the rule decision: %d %q %q", resp.StatusCode, resp.Header.Get(HeaderRoute), resp.Header.Get(HeaderRouteReason))
	}
	resp, _ = post("invoice for John Smith", "masked")
	calls := tr.all()
	if resp.StatusCode != 200 || calls[len(calls)-1].host != "myup.test" || resp.Header.Get(HeaderRouteReason) != "rule:1" {
		t.Fatalf("equal rank keeps the rule's upstream: %d %+v", resp.StatusCode, calls[len(calls)-1])
	}
	resp, _ = post("hi John Smith", "local")
	calls = tr.all()
	if resp.StatusCode != 200 || resp.Header.Get(HeaderRoute) != config.RouteLocal || resp.Header.Get(HeaderRouteReason) != "header" || calls[len(calls)-1].host != "local.test" {
		t.Fatalf("masked to local is stricter: %d %q", resp.StatusCode, resp.Header.Get(HeaderRoute))
	}
	resp, _ = post("John Smith diagnosis", "block")
	if resp.StatusCode != http.StatusForbidden || resp.Header.Get(HeaderRoute) != config.RouteBlock {
		t.Fatalf("block header: %d", resp.StatusCode)
	}

	tr2 := &fnTransport{reply: okReply}
	gw2 := newTestServer(t, "  allow_route_header: false\n", tr2)
	resp, _ = postJSON(t, gw2.URL+"/v1/chat/completions", chatBody("hi John Smith", false), map[string]string{HeaderRoute: "passthrough"})
	if resp.StatusCode != 200 || resp.Header.Get(HeaderRoute) != config.RouteMasked || resp.Header.Get(HeaderRouteReason) != "default" {
		t.Fatalf("disabled header must be ignored: %d %q", resp.StatusCode, resp.Header.Get(HeaderRoute))
	}
}

func TestRouteHeaderCannotLoosenLocalDefault(t *testing.T) {
	trl := &fnTransport{reply: okReply}
	gwl := newTestServerDefault(t, "local", "", trl)
	resp, out := postJSON(t, gwl.URL+"/v1/chat/completions", chatBody("hi John Smith", false), map[string]string{HeaderRoute: "masked"})
	if resp.StatusCode != http.StatusForbidden || len(trl.all()) != 0 {
		t.Fatalf("masked header over local default: %d %s", resp.StatusCode, out)
	}
	resp, _ = postJSON(t, gwl.URL+"/v1/chat/completions", chatBody("hi John Smith", false), nil)
	if resp.StatusCode != 200 || trl.all()[0].host != "local.test" {
		t.Fatalf("local default: %d", resp.StatusCode)
	}
}

func TestOversizedResponseIs502(t *testing.T) {
	old := maxResponseBytes
	maxResponseBytes = 256
	t.Cleanup(func() { maxResponseBytes = old })
	size := 0
	tr := &fnTransport{reply: func(_ string, _ map[string]any) (*http.Response, error) {
		b := `{"type":"message","content":[{"type":"text","text":"` + strings.Repeat("a", size) + `"}]}`
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(b))}, nil
	}}
	gw := newTestServer(t, "", tr)
	for _, c := range []struct {
		path string
		body map[string]any
	}{
		{"/v1/chat/completions", chatBody("hi", false)},
		{"/v1/messages", anthBody("hi", false)},
	} {
		size = 400
		resp, out := postJSON(t, gw.URL+c.path, c.body, nil)
		if resp.StatusCode != http.StatusBadGateway || !strings.Contains(string(out), "exceeds 256 bytes") {
			t.Fatalf("%s oversized: %d %s", c.path, resp.StatusCode, out)
		}
		size = 256 - len(`{"type":"message","content":[{"type":"text","text":""}]}`)
		resp, out = postJSON(t, gw.URL+c.path, c.body, nil)
		if resp.StatusCode != 200 {
			t.Fatalf("%s at limit: %d %s", c.path, resp.StatusCode, out)
		}
	}
}

type chunkReader struct {
	data []byte
	n    int
}

func (c *chunkReader) Read(p []byte) (int, error) {
	if len(c.data) == 0 {
		return 0, io.EOF
	}
	k := min(c.n, len(p), len(c.data))
	copy(p, c.data[:k])
	c.data = c.data[k:]
	return k, nil
}

func sseFor(masked, nl string) string {
	var b strings.Builder
	b.WriteString(": keepalive" + nl + nl)
	b.WriteString(`data: {"id":"c1","object":"chat.completion.chunk",` + nl)
	b.WriteString(`data: "choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}` + nl + nl)
	text := []rune("ECHO: " + masked + " héllo 日本語 ✓")
	for i := 0; i < len(text); {
		end := min(i+5, len(text))
		piece, _ := json.Marshal(string(text[i:end]))
		b.WriteString(": tick" + nl)
		b.WriteString(`data: {"id":"c1","choices":[{"index":0,` + nl)
		b.WriteString(`data: "delta":{"content":` + string(piece) + `},` + nl)
		b.WriteString(`data: "finish_reason":null}]}` + nl + nl)
		i = end
	}
	b.WriteString(`data: {"id":"c1","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}` + nl + nl)
	b.WriteString("data: [DONE]" + nl + nl)
	return b.String()
}

func readEvents(t *testing.T, body string) (content string, comments []string, done bool) {
	t.Helper()
	var sb strings.Builder
	norm := strings.ReplaceAll(body, "\r\n", "\n")
	for _, ev := range strings.Split(norm, "\n\n") {
		var data []string
		for _, l := range strings.Split(ev, "\n") {
			switch {
			case strings.HasPrefix(l, ":"):
				comments = append(comments, l)
			case strings.HasPrefix(l, "data:"):
				data = append(data, strings.TrimPrefix(l[5:], " "))
			}
		}
		if len(data) == 0 {
			continue
		}
		payload := strings.Join(data, "\n")
		if payload == "[DONE]" {
			done = true
			continue
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			t.Fatalf("bad event %q: %v", payload, err)
		}
		for _, c := range chunk.Choices {
			sb.WriteString(c.Delta.Content)
		}
	}
	return sb.String(), comments, done
}

func TestOpenAIStreamMultiLineEvents(t *testing.T) {
	text := "Write to John Smith at john.smith@gmail.com about Maria Garcia"
	for _, nl := range []string{"\n", "\r\n"} {
		for _, n := range []int{1, 2, 3, 7} {
			t.Run(fmt.Sprintf("crlf=%v/chunk=%d", nl == "\r\n", n), func(t *testing.T) {
				tr := &fnTransport{}
				tr.reply = func(_ string, req map[string]any) (*http.Response, error) {
					msgs := req["messages"].([]any)
					masked := msgs[len(msgs)-1].(map[string]any)["content"].(string)
					return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}},
						Body: io.NopCloser(&chunkReader{data: []byte(sseFor(masked, nl)), n: n})}, nil
				}
				gw := newTestServer(t, "", tr)
				resp, out := postJSON(t, gw.URL+"/v1/chat/completions", chatBody(text, true), nil)
				if resp.StatusCode != 200 {
					t.Fatalf("status %d %s", resp.StatusCode, out)
				}
				content, comments, done := readEvents(t, string(out))
				if want := "ECHO: " + text + " héllo 日本語 ✓"; content != want {
					t.Fatalf("content\n got: %q\nwant: %q\nraw: %q", content, want, out)
				}
				if !done {
					t.Fatalf("missing [DONE]: %q", out)
				}
				ticks := 0
				for _, c := range comments {
					if c == ": tick" || c == ": keepalive" {
						ticks++
					}
				}
				if ticks < 2 {
					t.Fatalf("comment lines dropped: %v", comments)
				}
				if nl == "\r\n" && !strings.Contains(string(out), "\r\n") {
					t.Fatal("line endings must be preserved")
				}
			})
		}
	}
}

func TestStreamBuffersStopAtCap(t *testing.T) {
	old := streamTextCap
	streamTextCap = 64
	t.Cleanup(func() { streamTextCap = old })
	sess := testSession(t)
	st := &streamState{sess: sess, rh: map[streamKey]*outworld.StreamRehydrator{}, raw: map[streamKey]*strings.Builder{}, over: map[streamKey]bool{}}
	k := streamKey{0, "content", -1}
	var out strings.Builder
	for range 100 {
		out.WriteString(st.push(k, "0123456789"))
	}
	out.WriteString(st.rh[k].Flush())
	if out.String() != strings.Repeat("0123456789", 100) {
		t.Fatal("stream output changed past the cap")
	}
	if !st.over[k] || st.raw[k].Len() > streamTextCap {
		t.Fatalf("raw buffer kept %d bytes past cap", st.raw[k].Len())
	}
	small := streamKey{1, "content", -1}
	st.push(small, "short")
	if st.over[small] || st.raw[small].String() != "short" {
		t.Fatal("under-cap channel must keep its raw text")
	}

	c := &anthropicChannel{rh: sess.NewStreamRehydrator(false)}
	for range 100 {
		c.push("0123456789")
	}
	if !c.over || c.raw.Len() > streamTextCap || c.out.Len() > streamTextCap {
		t.Fatalf("anthropic buffers raw=%d out=%d", c.raw.Len(), c.out.Len())
	}
	c2 := &anthropicChannel{rh: sess.NewStreamRehydrator(false)}
	c2.push("abc")
	if c2.over || c2.raw.String() != "abc" {
		t.Fatal("under-cap anthropic channel must keep its text")
	}
}

func testSession(t *testing.T) *outworld.Session {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "outworld.yaml")
	t.Setenv("OUTWORLD_VAULT_KEY", strings.Repeat("ab", 32))
	os.WriteFile(cfgPath, []byte("vault: {path: ./v.vault, key_env: OUTWORLD_VAULT_KEY}\n"), 0o600)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	eng, err := app.BuildEngine(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { eng.Vault().Close() })
	return eng.NewSession("")
}

func TestServeShutsDownOnCancel(t *testing.T) {
	tr := &fnTransport{reply: okReply}
	_ = newTestServer(t, "", tr)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "outworld.yaml")
	os.WriteFile(cfgPath, []byte("vault: {path: ./v.vault, key_env: OUTWORLD_VAULT_KEY}\n"), 0o600)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	eng, err := app.BuildEngine(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Vault().Close()
	srv, _ := New(cfg, eng, Options{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ctx, ln) }()
	resp, err := http.Get("http://" + ln.Addr().String() + "/healthz")
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("health %v", err)
	}
	resp.Body.Close()
	cancel()
	select {
	case err := <-errc:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after cancel")
	}
	if _, err := http.Get("http://" + ln.Addr().String() + "/healthz"); err == nil {
		t.Fatal("server still accepting after shutdown")
	}
}

func TestDedupeViolations(t *testing.T) {
	vs := []outworld.Violation{
		{Type: "email", Source: "vault", Field: "messages[0].name", Start: 0, End: 14},
		{Type: "email", Source: "regex:email", Field: "messages[0].name", Start: 0, End: 14},
		{Type: "person", Source: "dictionary", Field: "messages[0].name", Start: 0, End: 4},
		{Type: "email", Source: "regex:email", Field: "messages[1].name", Start: 0, End: 14},
	}
	got := DedupeViolations(vs)
	if len(got) != 2 || got[0].Source != "regex:email" || got[1].Field != "messages[1].name" {
		t.Fatalf("dedupe: %+v", got)
	}
	if !strings.Contains((&outworld.LeakError{Violations: got}).Error(), "2 sensitive value(s)") {
		t.Fatal("count must be per value")
	}
}

func TestLeakFindingsCountedOncePerValue(t *testing.T) {
	tr := &fnTransport{reply: okReply}
	gw := newTestServer(t, "", tr)
	body := map[string]any{"model": "m", "messages": []any{
		map[string]any{"role": "user", "content": "mail john.smith@gmail.com"},
		map[string]any{"role": "user", "name": "john.smith@gmail.com", "content": "hi"},
	}}
	for _, hdr := range []map[string]string{nil, {HeaderDryRun: "1"}} {
		resp, out := postJSON(t, gw.URL+"/v1/chat/completions", body, hdr)
		var env struct {
			Error struct {
				Message  string               `json:"message"`
				Findings []outworld.Violation `json:"findings"`
			} `json:"error"`
			Message  string               `json:"message"`
			Findings []outworld.Violation `json:"findings"`
		}
		json.Unmarshal(out, &env)
		findings, msg := env.Error.Findings, env.Error.Message
		if hdr != nil {
			findings, msg = env.Findings, env.Message
		}
		if len(findings) == 0 {
			t.Fatalf("expected a block: %d %s", resp.StatusCode, out)
		}
		seen := map[string]bool{}
		for _, f := range findings {
			k := fmt.Sprintf("%d|%s|%d|%d", f.Message, f.Field, f.Start, f.End)
			if seen[k] {
				t.Fatalf("duplicate finding %s: %s", k, out)
			}
			seen[k] = true
		}
		if !strings.Contains(msg, fmt.Sprintf("%d sensitive value(s)", len(findings))) {
			t.Fatalf("message count does not match findings: %s", out)
		}
	}
}

func TestShutdownEndsOpenStreams(t *testing.T) {
	old := ShutdownTimeout
	ShutdownTimeout = 20 * time.Second
	t.Cleanup(func() { ShutdownTimeout = old })
	for _, api := range []string{config.APIOpenAI, config.APIAnthropic} {
		pr, pw := io.Pipe()
		t.Cleanup(func() { pw.Close() })
		var fragment string
		tr := &fnTransport{reply: func(_ string, req map[string]any) (*http.Response, error) {
			msgs := req["messages"].([]any)
			masked := msgs[len(msgs)-1].(map[string]any)["content"].(string)
			first := strings.Fields(masked)[1]
			fragment = first[:len(first)-1]
			text, _ := json.Marshal("ECHO: hi " + fragment)
			go func() {
				if api == config.APIAnthropic {
					io.WriteString(pw, "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{}}\n\n")
					io.WriteString(pw, "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n")
					io.WriteString(pw, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":"+string(text)+"}}\n\n")
				} else {
					io.WriteString(pw, "data: {\"id\":\"c1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":"+string(text)+"},\"finish_reason\":null}]}\n\n")
				}
			}()
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: pr}, nil
		}}
		srv := newTestGateway(t, "masked", "", tr)
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		errc := make(chan error, 1)
		go func() { errc <- srv.Serve(ctx, ln) }()
		body := chatBody("hi John Smith", true)
		if api == config.APIAnthropic {
			body = anthBody("hi John Smith", true)
		}
		b, _ := json.Marshal(body)
		resp, err := http.Post("http://"+ln.Addr().String()+pathFor(api), "application/json", bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		var got bytes.Buffer
		buf := make([]byte, 4096)
		for !strings.Contains(got.String(), "ECHO") {
			n, err := resp.Body.Read(buf)
			got.Write(buf[:n])
			if err != nil {
				t.Fatalf("%s: stream ended early: %v %q", api, err, got.String())
			}
		}
		start := time.Now()
		cancel()
		rest := make(chan error, 1)
		go func() {
			_, err := io.Copy(&got, resp.Body)
			rest <- err
		}()
		select {
		case <-rest:
		case <-time.After(5 * time.Second):
			t.Fatalf("%s: open stream held the shutdown", api)
		}
		select {
		case err := <-errc:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s: Serve did not return", api)
		}
		if time.Since(start) > 5*time.Second {
			t.Fatalf("%s: shutdown took %s", api, time.Since(start))
		}
		resp.Body.Close()
		out := got.String()
		if strings.Contains(out, "[DONE]") || strings.Contains(out, "message_stop") {
			t.Fatalf("%s: cut-off stream must not look complete: %q", api, out)
		}
		if api == config.APIAnthropic {
			if !strings.Contains(out, "event: error\ndata: {\"error\":{\"message\":") || !strings.Contains(out, `"type":"overloaded_error"`) {
				t.Fatalf("anthropic shutdown event missing: %q", out)
			}
		} else if !strings.Contains(out, `data: {"error":{"code":"outworld_shutdown"`) {
			t.Fatalf("openai shutdown event missing: %q", out)
		}
		if strings.Contains(out, fragment) {
			t.Fatalf("%s: held-back surrogate fragment %q flushed raw: %q", api, fragment, out)
		}
		if !strings.Contains(out, "ECHO: hi") {
			t.Fatalf("%s: text before the held tail lost: %q", api, out)
		}
	}
}
