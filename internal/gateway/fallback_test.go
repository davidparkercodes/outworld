package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/davidparkercodes/outworld/internal/config"
	"github.com/davidparkercodes/outworld/pkg/outworld"
)

func uncoveredCases() []fieldCase {
	oa := func(f func(b map[string]any, v string)) func(string) map[string]any {
		return func(v string) map[string]any {
			b := chatBody("hello", false)
			f(b, v)
			return b
		}
	}
	an := func(f func(b map[string]any, v string)) func(string) map[string]any {
		return func(v string) map[string]any {
			b := anthBody("hello", false)
			f(b, v)
			return b
		}
	}
	userMsg := func(b map[string]any) map[string]any { return b["messages"].([]any)[0].(map[string]any) }
	return []fieldCase{
		{"guided_choice", config.APIOpenAI, oa(func(b map[string]any, v string) { b["guided_choice"] = []any{"a", v} })},
		{"documents", config.APIOpenAI, oa(func(b map[string]any, v string) { b["documents"] = []any{map[string]any{"text": v}} })},
		{"safety_identifier", config.APIOpenAI, oa(func(b map[string]any, v string) { b["safety_identifier"] = v })},
		{"prompt_cache_key", config.APIOpenAI, oa(func(b map[string]any, v string) { b["prompt_cache_key"] = v })},
		{"tool_choice function name", config.APIOpenAI, oa(func(b map[string]any, v string) {
			b["tool_choice"] = map[string]any{"type": "function", "function": map[string]any{"name": v}}
		})},
		{"response_format extra key", config.APIOpenAI, oa(func(b map[string]any, v string) {
			b["response_format"] = map[string]any{"type": "json_schema", "note": v, "json_schema": map[string]any{"name": "o", "schema": map[string]any{"type": "object"}}}
		})},
		{"json_schema extra key", config.APIOpenAI, oa(func(b map[string]any, v string) {
			b["response_format"] = map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "o", "extra": v, "schema": map[string]any{"type": "object"}}}
		})},
		{"tool_call_id", config.APIOpenAI, oa(func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "tool", "tool_call_id": v, "content": "ok"})
		})},
		{"text part extra key", config.APIOpenAI, oa(func(b map[string]any, v string) {
			userMsg(b)["content"] = []any{map[string]any{"type": "text", "text": "hi", "note": v}}
		})},
		{"content object", config.APIOpenAI, oa(func(b map[string]any, v string) { userMsg(b)["content"] = map[string]any{"text": v} })},
		{"reasoning object", config.APIOpenAI, oa(func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": "ok", "reasoning": map[string]any{"text": v}})
		})},
		{"custom grammar definition", config.APIOpenAI, oa(func(b map[string]any, v string) {
			b["tools"] = []any{map[string]any{"type": "custom", "custom": map[string]any{"name": "Edit", "description": "d",
				"format": map[string]any{"type": "grammar", "grammar": map[string]any{"syntax": "lark", "definition": v}}}}}
		})},
		{"top-level unknown", config.APIAnthropic, an(func(b map[string]any, v string) { b["extra"] = map[string]any{"note": v} })},
		{"tool_choice name", config.APIAnthropic, an(func(b map[string]any, v string) { b["tool_choice"] = map[string]any{"type": "tool", "name": v} })},
		{"mcp server name", config.APIAnthropic, an(func(b map[string]any, v string) {
			b["mcp_servers"] = []any{map[string]any{"type": "url", "url": "https://mcp.test/sse", "name": v}}
		})},
		{"container", config.APIAnthropic, an(func(b map[string]any, v string) { b["container"] = v })},
		{"unknown message key", config.APIAnthropic, an(func(b map[string]any, v string) { userMsg(b)["note"] = v })},
		{"metadata other key", config.APIAnthropic, an(func(b map[string]any, v string) { b["metadata"] = map[string]any{"user_id": "u1", "note": v} })},
		{"text block extra key", config.APIAnthropic, an(func(b map[string]any, v string) {
			userMsg(b)["content"] = []any{map[string]any{"type": "text", "text": "hi", "note": v}}
		})},
		{"tool_result extra key", config.APIAnthropic, an(func(b map[string]any, v string) {
			userMsg(b)["content"] = []any{map[string]any{"type": "tool_result", "tool_use_id": "toolu_01", "content": "ok", "note": v}}
		})},
		{"document extra key", config.APIAnthropic, an(func(b map[string]any, v string) {
			userMsg(b)["content"] = []any{map[string]any{"type": "document", "source": map[string]any{"type": "text", "media_type": "text/plain", "data": "x"}, "note": v}}
		})},
		{"search_result extra key", config.APIAnthropic, an(func(b map[string]any, v string) {
			userMsg(b)["content"] = []any{map[string]any{"type": "search_result", "title": "t", "source": "s", "content": []any{}, "note": v}}
		})},
		{"citation url", config.APIAnthropic, an(func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "text", "text": "ok",
				"citations": []any{map[string]any{"type": "web_search_result_location", "cited_text": "x", "url": v}}}}})
		})},
		{"system non-text block", config.APIAnthropic, an(func(b map[string]any, v string) { b["system"] = []any{map[string]any{"type": "note", "text": v}} })},
		{"system text extra key", config.APIAnthropic, an(func(b map[string]any, v string) {
			b["system"] = []any{map[string]any{"type": "text", "text": "be brief", "note": v}}
		})},
		{"unknown block id field", config.APIAnthropic, an(func(b map[string]any, v string) {
			userMsg(b)["content"] = []any{map[string]any{"type": "custom_x", "patient_id": v}}
		})},
		{"unknown block data field", config.APIAnthropic, an(func(b map[string]any, v string) {
			userMsg(b)["content"] = []any{map[string]any{"type": "custom_x", "data": v}}
		})},
		{"schema format", config.APIOpenAI, oa(func(b map[string]any, v string) {
			b["tools"] = schemaTool(config.APIOpenAI, map[string]any{"type": "object", "properties": map[string]any{"p": map[string]any{"type": "string", "format": v}}})
		})},
		{"schema type", config.APIAnthropic, an(func(b map[string]any, v string) {
			b["tools"] = schemaTool(config.APIAnthropic, map[string]any{"type": v})
		})},
		{"tool name", config.APIAnthropic, an(func(b map[string]any, v string) {
			b["tools"] = []any{map[string]any{"name": v, "description": "d", "input_schema": map[string]any{"type": "object"}}}
		})},
		{"tool name", config.APIOpenAI, oa(func(b map[string]any, v string) {
			b["tools"] = []any{map[string]any{"type": "function", "function": map[string]any{"name": v, "parameters": map[string]any{"type": "object"}}}}
		})},
	}
}

func TestUncoveredFieldsAreChecked(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, c := range uncoveredCases() {
		resp, out := postJSON(t, gw.URL+pathFor(c.api), c.body(probe), nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("%s/%s: status %d %s", c.api, c.name, resp.StatusCode, out)
		}
		for _, real := range probeReals {
			if strings.Contains(string(out), real) {
				t.Fatalf("%s/%s: 422 body holds %q: %s", c.api, c.name, real, out)
			}
		}
	}
	if n := len(tr.all()); n != 0 {
		t.Fatalf("leaking requests reached upstream %d times", n)
	}
	for _, c := range uncoveredCases() {
		before := len(tr.all())
		resp, out := postJSON(t, gw.URL+pathFor(c.api), c.body("plain words"), nil)
		calls := tr.all()
		if resp.StatusCode != 200 || len(calls) != before+1 || !strings.Contains(calls[len(calls)-1].body, "plain words") {
			t.Fatalf("%s/%s clean: status %d %s", c.api, c.name, resp.StatusCode, out)
		}
	}
}

func TestUncoveredFieldsAreSeenByRules(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, blockRules, tr)
	for _, c := range uncoveredCases() {
		resp, out := postJSON(t, gw.URL+pathFor(c.api), c.body("topsecret"), nil)
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("%s/%s: status %d %s", c.api, c.name, resp.StatusCode, out)
		}
	}
	if n := len(tr.all()); n != 0 {
		t.Fatalf("blocked requests reached upstream %d times", n)
	}
}

func TestToolNamesUseNarrowCheck(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	named := func(api, name string) map[string]any {
		b := chatBody("hello", false)
		b["tools"] = []any{map[string]any{"type": "function", "function": map[string]any{"name": name, "parameters": map[string]any{"type": "object"}}}}
		if api == config.APIAnthropic {
			b = anthBody("hello", false)
			b["tools"] = []any{map[string]any{"name": name, "input_schema": map[string]any{"type": "object"}}}
		}
		return b
	}
	for _, api := range []string{config.APIOpenAI, config.APIAnthropic} {
		for _, name := range []string{"Edit", "Bash", "Will", "Grant"} {
			resp, out := postJSON(t, gw.URL+pathFor(api), named(api, name), nil)
			if resp.StatusCode != 200 || !strings.Contains(tr.all()[len(tr.all())-1].body, `"name":"`+name+`"`) {
				t.Fatalf("%s %s: %d %s", api, name, resp.StatusCode, out)
			}
		}
		for _, name := range []string{"call_8045551234", "notify_maria_garcia"} {
			resp, out := postJSON(t, gw.URL+pathFor(api), named(api, name), nil)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("%s %s: %d %s", api, name, resp.StatusCode, out)
			}
		}
	}
}

func TestCustomToolsOpenAI(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	body := chatBody("hello", false)
	grammar := `start: "apply" WORD`
	body["tools"] = []any{map[string]any{"type": "custom", "custom": map[string]any{"name": "Edit", "description": "Edit for " + probe,
		"format": map[string]any{"type": "grammar", "grammar": map[string]any{"syntax": "lark", "definition": grammar}}}}}
	body["messages"] = append(body["messages"].([]any),
		map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"type": "custom", "id": "call_abc123", "custom": map[string]any{"name": "Edit", "input": "write to " + probe}}}},
		map[string]any{"role": "tool", "tool_call_id": "call_abc123", "content": "done"})
	resp, out := postJSON(t, gw.URL+"/v1/chat/completions", body, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d %s", resp.StatusCode, out)
	}
	sent := tr.all()[0].body
	for _, real := range probeReals {
		if strings.Contains(sent, real) {
			t.Fatalf("custom tool text leaked %q: %s", real, sent)
		}
	}
	gb, _ := json.Marshal(grammar)
	if !strings.Contains(sent, `"name":"Edit"`) || !strings.Contains(sent, string(gb)) || !strings.Contains(sent, `"call_abc123"`) {
		t.Fatalf("custom names, grammar and ids must pass unchanged: %s", sent)
	}
}

func TestCustomToolInputRoundTrips(t *testing.T) {
	var seen string
	tr := &fnTransport{}
	tr.reply = func(_ string, req map[string]any) (*http.Response, error) {
		msgs := req["messages"].([]any)
		seen = msgs[len(msgs)-1].(map[string]any)["content"].(string)
		return jsonResponse(map[string]any{"id": "x", "choices": []any{map[string]any{"index": 0, "message": map[string]any{"role": "assistant",
			"tool_calls": []any{map[string]any{"type": "custom", "id": "call_1", "custom": map[string]any{"name": "Edit", "input": seen}}}}}}}), nil
	}
	gw := newTestServer(t, "", tr)
	resp, out := postJSON(t, gw.URL+"/v1/chat/completions", chatBody("note for "+probe, false), nil)
	if resp.StatusCode != 200 || !strings.Contains(string(out), "Maria Garcia") || strings.Contains(seen, "Maria") {
		t.Fatalf("custom input not rehydrated: %d %s (sent %q)", resp.StatusCode, out, seen)
	}
	var reply struct {
		Choices []struct {
			Message map[string]any `json:"message"`
		} `json:"choices"`
	}
	json.Unmarshal(out, &reply)
	first := seen
	next := chatBody("note for "+probe, false)
	next["messages"] = append(next["messages"].([]any), reply.Choices[0].Message, map[string]any{"role": "tool", "tool_call_id": "call_1", "content": "ok"})
	if resp, out := postJSON(t, gw.URL+"/v1/chat/completions", next, nil); resp.StatusCode != 200 {
		t.Fatalf("resend: %d %s", resp.StatusCode, out)
	}
	calls := tr.all()
	var sent map[string]any
	json.Unmarshal([]byte(calls[1].body), &sent)
	call := sent["messages"].([]any)[2].(map[string]any)["tool_calls"].([]any)[0].(map[string]any)["custom"].(map[string]any)
	if call["input"] != first {
		t.Fatalf("resent custom input must be the bytes the model produced: %q vs %q", call["input"], first)
	}
}

func TestKeyTextNeverPrinted(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	key := "maria.garcia@example.com_notes"
	cases := []map[string]any{}
	b := chatBody("hello", false)
	b["tools"] = []any{map[string]any{"type": "function", key: "x", "function": map[string]any{"name": "t", "parameters": map[string]any{"type": "object"}}}}
	cases = append(cases, b)
	b = chatBody("hello", false)
	b["messages"].([]any)[0].(map[string]any)[key] = "x"
	cases = append(cases, b)
	b = chatBody("hello", false)
	b["tools"] = schemaTool(config.APIOpenAI, map[string]any{"type": "object", key: "x"})
	cases = append(cases, b)
	b = anthBody("hello", false)
	b["tools"] = []any{map[string]any{"type": "web_search_20250305", "name": "web_search", key: "x"}}
	cases = append(cases, b)
	for i, body := range cases {
		path := "/v1/chat/completions"
		if _, ok := body["max_tokens"]; ok {
			path = "/v1/messages"
		}
		resp, out := postJSON(t, gw.URL+path, body, nil)
		var env struct {
			Error struct {
				Findings []outworld.Violation `json:"findings"`
			} `json:"error"`
		}
		json.Unmarshal(out, &env)
		if resp.StatusCode != http.StatusUnprocessableEntity || len(env.Error.Findings) == 0 {
			t.Fatalf("case %d: %d %s", i, resp.StatusCode, out)
		}
		if strings.Contains(string(out), "maria") || strings.Contains(string(out), "notes") || strings.Contains(string(out), "example") {
			t.Fatalf("case %d: key text printed: %s", i, out)
		}
		for _, f := range env.Error.Findings {
			if strings.HasSuffix(f.Field, "(key)") && f.Preview != "[***]" {
				t.Fatalf("case %d: key preview %q", i, f.Preview)
			}
		}
	}
}

func TestUnchangedArgumentsKeepTheirBytes(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	args := `{"z":"hi","a":1.50}`
	body := chatBody("hello", false)
	body["messages"] = append(body["messages"].([]any), map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"id": "call_1", "type": "function", "function": map[string]any{"name": "t", "arguments": args}}}})
	postJSON(t, gw.URL+"/v1/chat/completions", body, nil)
	ab, _ := json.Marshal(args)
	if !strings.Contains(tr.all()[0].body, string(ab)) {
		t.Fatalf("unchanged arguments were rewritten: %s", tr.all()[0].body)
	}
	body["messages"].([]any)[2].(map[string]any)["tool_calls"].([]any)[0].(map[string]any)["function"].(map[string]any)["arguments"] = `{"z":"for Maria Garcia","a":1.50}`
	postJSON(t, gw.URL+"/v1/chat/completions", body, nil)
	if strings.Contains(tr.all()[1].body, "Maria") {
		t.Fatal("changed arguments must be masked")
	}
}

func TestRouteHeaderNamedUpstreamAndMissingRoute(t *testing.T) {
	tr := &fnTransport{reply: okReply}
	gw := newTestServer(t, "", tr)
	resp, out := postJSON(t, gw.URL+"/v1/chat/completions", chatBody("hi", false), map[string]string{HeaderRoute: "local:local"})
	if resp.StatusCode != 200 || tr.all()[0].host != "local.test" {
		t.Fatalf("local:local: %d %s", resp.StatusCode, out)
	}
	resp, out = postJSON(t, gw.URL+"/v1/chat/completions", chatBody("hi", false), map[string]string{HeaderRoute: "masked:cloud"})
	if resp.StatusCode != 200 {
		t.Fatalf("masked:cloud: %d %s", resp.StatusCode, out)
	}
	resp, out = postJSON(t, gw.URL+"/v1/messages", anthBody("hi", false), map[string]string{HeaderRoute: "local"})
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(out), "outworld_route_error") {
		t.Fatalf("stricter route without upstream: %d %s", resp.StatusCode, out)
	}
}

func TestPassthroughDrainAfterDoneAddsNothing(t *testing.T) {
	pr, pw := io.Pipe()
	t.Cleanup(func() { pw.Close() })
	tr := &fnTransport{reply: func(_ string, _ map[string]any) (*http.Response, error) {
		go io.WriteString(pw, "data: {\"id\":\"c1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi\"}}]}\n\ndata: [DONE]\n\n")
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: pr}, nil
	}}
	srv := newTestGateway(t, "local", "", tr)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ctx, ln) }()
	b, _ := json.Marshal(chatBody("hi", true))
	resp, err := http.Post("http://"+ln.Addr().String()+"/v1/chat/completions", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	buf := make([]byte, 4096)
	for !strings.Contains(got.String(), "[DONE]") {
		n, err := resp.Body.Read(buf)
		got.Write(buf[:n])
		if err != nil {
			t.Fatal(err)
		}
	}
	cancel()
	io.Copy(&got, resp.Body)
	resp.Body.Close()
	select {
	case <-errc:
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return")
	}
	if strings.Contains(got.String(), "outworld_shutdown") {
		t.Fatalf("finished stream got a shutdown event: %q", got.String())
	}
}

func realisticAgentRequest(api string) map[string]any {
	if api == config.APIAnthropic {
		return map[string]any{
			"model": "claude-x", "max_tokens": 32000, "stream": false,
			"metadata": map[string]any{"user_id": "user_3f9a0c2b1d_account_7e1c2a90-1b2c-4d3e-8f00-112233445566_session_9a8b7c6d"},
			"thinking": map[string]any{"type": "enabled", "budget_tokens": 31999},
			"system": []any{
				map[string]any{"type": "text", "text": attributionText},
				map[string]any{"type": "text", "text": "You are a coding agent. Use the tools to read and edit files in the repository.", "cache_control": map[string]any{"type": "ephemeral"}},
			},
			"tools":       agentTools(api),
			"tool_choice": map[string]any{"type": "auto"},
			"messages": []any{
				map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "Fix the failing test in pkg/server", "cache_control": map[string]any{"type": "ephemeral", "ttl": "5m"}}}},
				map[string]any{"role": "assistant", "content": []any{
					map[string]any{"type": "thinking", "thinking": "I should run the tests first.", "signature": sig},
					map[string]any{"type": "text", "text": "Running the tests."},
					map[string]any{"type": "tool_use", "id": "toolu_01A09q90qw90lq917835lq9", "name": "Bash", "input": map[string]any{"command": "go test ./pkg/server/...", "description": "Run server tests"}},
				}},
				map[string]any{"role": "user", "content": []any{
					map[string]any{"type": "tool_result", "tool_use_id": "toolu_01A09q90qw90lq917835lq9", "content": "--- FAIL: TestHandler (0.00s)\n    handler_test.go:42: got 500, want 200\nFAIL", "is_error": true},
				}},
			},
		}
	}
	return map[string]any{
		"model": "gpt-x", "stream": false, "temperature": 0.2, "parallel_tool_calls": true,
		"prompt_cache_key": "repo-3f9a0c2b1d-main",
		"metadata":         map[string]any{"client": "coding-agent", "session": "9a8b7c6d-1b2c-4d3e-8f00-112233445566"},
		"tools":            agentTools(api),
		"tool_choice":      "auto",
		"stream_options":   map[string]any{"include_usage": true},
		"messages": []any{
			map[string]any{"role": "system", "content": "You are a coding agent. Use the tools to read and edit files in the repository."},
			map[string]any{"role": "user", "content": "Fix the failing test in pkg/server"},
			map[string]any{"role": "assistant", "content": nil, "tool_calls": []any{map[string]any{"id": "call_Xk3n9QzL2pR7tV1wY5bA8cD0", "type": "function",
				"function": map[string]any{"name": "Bash", "arguments": `{"command":"go test ./pkg/server/...","description":"Run server tests"}`}}}},
			map[string]any{"role": "tool", "tool_call_id": "call_Xk3n9QzL2pR7tV1wY5bA8cD0", "content": "--- FAIL: TestHandler (0.00s)\n    handler_test.go:42: got 500, want 200\nFAIL"},
		},
	}
}

func TestRealisticAgentRequestPasses(t *testing.T) {
	for _, api := range []string{config.APIOpenAI, config.APIAnthropic} {
		tr := &fnTransport{reply: echoReply}
		gw := newTestServer(t, "", tr)
		body := realisticAgentRequest(api)
		for range 2 {
			resp, out := postJSON(t, gw.URL+pathFor(api), body, nil)
			if resp.StatusCode != 200 {
				t.Fatalf("%s: status %d %s", api, resp.StatusCode, out)
			}
		}
		calls := tr.all()
		if calls[0].body != calls[1].body {
			t.Fatalf("%s: repeated request bytes differ", api)
		}
		keeps := []string{"call_Xk3n9QzL2pR7tV1wY5bA8cD0", "repo-3f9a0c2b1d-main"}
		if api == config.APIAnthropic {
			keeps = []string{"toolu_01A09q90qw90lq917835lq9", sig, attributionText}
		}
		for _, keep := range keeps {
			kb, _ := json.Marshal(keep)
			if !strings.Contains(calls[0].body, string(kb)) {
				t.Fatalf("%s: %q changed: %s", api, keep, calls[0].body)
			}
		}
	}
}

func TestScopeHeaderLengthCapped(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "  allow_scope_header: true\n", tr)
	for _, api := range []string{config.APIOpenAI, config.APIAnthropic} {
		body := chatBody("hi", false)
		if api == config.APIAnthropic {
			body = anthBody("hi", false)
		}
		resp, out := postJSON(t, gw.URL+pathFor(api), body, map[string]string{HeaderScope: strings.Repeat("s", 257)})
		if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(out), "outworld_bad_request") {
			t.Fatalf("%s long scope: %d %s", api, resp.StatusCode, out)
		}
		resp, _ = postJSON(t, gw.URL+pathFor(api), body, map[string]string{HeaderScope: strings.Repeat("s", 256)})
		if resp.StatusCode != 200 {
			t.Fatalf("%s 256-byte scope: %d", api, resp.StatusCode)
		}
	}
	if len(tr.all()) != 2 {
		t.Fatal("over-long scope reached upstream")
	}
}

func TestMaxHeaderBytes(t *testing.T) {
	tr := &fnTransport{reply: okReply}
	srv := newTestGateway(t, "masked", "", tr)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ctx, ln) }()
	defer func() {
		cancel()
		<-errc
	}()
	get := func(size int) int {
		req, _ := http.NewRequest(http.MethodGet, "http://"+ln.Addr().String()+"/healthz", nil)
		req.Header.Set("X-Pad", strings.Repeat("a", size))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	if code := get(32 << 10); code != 200 {
		t.Fatalf("32 KiB header: %d", code)
	}
	if code := get(200 << 10); code != http.StatusRequestHeaderFieldsTooLarge {
		t.Fatalf("200 KiB header: %d", code)
	}
}
