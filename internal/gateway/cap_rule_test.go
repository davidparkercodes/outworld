package gateway

import (
	"encoding/base64"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func capPlacements(v string) map[string]map[string]any {
	assistant := func(blocks ...any) map[string]any {
		return map[string]any{"model": "claude-x", "max_tokens": 64, "messages": []any{map[string]any{"role": "user", "content": "hello"}, map[string]any{"role": "assistant", "content": blocks}, map[string]any{"role": "user", "content": "go"}}}
	}
	toolUse := func(input map[string]any) map[string]any {
		return assistant(map[string]any{"type": "tool_use", "id": "toolu_01AbCdEfGhIjKlMnOpQrStUv", "name": "f", "input": input})
	}
	toolResult := func(content any) map[string]any {
		b := assistant(map[string]any{"type": "tool_use", "id": "toolu_01AbCdEfGhIjKlMnOpQrStUv", "name": "f", "input": map[string]any{"q": "x"}})
		msgs := b["messages"].([]any)
		b["messages"] = append(msgs[:2], map[string]any{"role": "user", "content": []any{map[string]any{"type": "tool_result", "tool_use_id": "toolu_01AbCdEfGhIjKlMnOpQrStUv", "content": content}}})
		return b
	}
	id := "srvtoolu_01AbCdEfGhIjKlMnOpQrStUv"
	top := func(k string) map[string]any {
		b := map[string]any{"model": "claude-x", "max_tokens": 64, "messages": []any{map[string]any{"role": "user", "content": "hello"}}}
		b["x_extra"] = map[string]any{k: v}
		return b
	}
	userField := map[string]any{"model": "claude-x", "max_tokens": 64, "messages": []any{map[string]any{"role": "user", "content": "hello", "signature": v}}}
	return map[string]map[string]any{
		"thinking signature":             assistant(map[string]any{"type": "thinking", "thinking": "hmm", "signature": v}),
		"redacted_thinking data":         assistant(map[string]any{"type": "redacted_thinking", "data": v}),
		"web search encrypted_content":   assistant(map[string]any{"type": "server_tool_use", "id": id, "name": "web_search", "input": map[string]any{"query": "q"}}, map[string]any{"type": "web_search_tool_result", "tool_use_id": id, "content": []any{map[string]any{"type": "web_search_result", "title": "Guide", "url": "https://x.test", "encrypted_content": v}}}),
		"citation encrypted_index":       assistant(map[string]any{"type": "text", "text": "ok", "citations": []any{map[string]any{"type": "web_search_result_location", "url": "https://x.test", "title": "Guide", "cited_text": "c", "encrypted_index": v}}}),
		"tool_use input signature":       toolUse(map[string]any{"signature": v}),
		"tool_use input encrypted":       toolUse(map[string]any{"encrypted_content": v}),
		"tool_result content signature":  toolResult([]any{map[string]any{"type": "text", "text": "done", "signature": v}}),
		"tool_result string":             toolResult(v),
		"top-level signature":            top("signature"),
		"top-level encrypted_content":    top("encrypted_content"),
		"unknown block encrypted":        assistant(map[string]any{"type": "mystery_block", "encrypted_content": v}),
		"message field signature":        userField,
		"nested tool input signature":    toolUse(map[string]any{"a": map[string]any{"b": []any{map[string]any{"signature": v}}}}),
		"user mystery block signature":   map[string]any{"model": "claude-x", "max_tokens": 64, "messages": []any{map[string]any{"role": "user", "content": []any{map[string]any{"type": "mystery_block", "signature": v}}}}},
		"tool_use input encrypted_index": toolUse(map[string]any{"encrypted_index": v}),
		"tool_use input data":            toolUse(map[string]any{"data": v}),
	}
}

func TestCapRuleByPlacement(t *testing.T) {
	_, gw := gatewayWithCap(t, &fnTransport{reply: echoReply})
	r := rand.New(rand.NewSource(1991))
	v := base64.StdEncoding.EncodeToString(binaryWithPII(r, collisionCap*3/2, "clean"))
	capped := map[string]bool{
		"thinking signature": true, "redacted_thinking data": true, "web search encrypted_content": true, "citation encrypted_index": true,
		"tool_use input signature": true, "tool_use input encrypted": true, "tool_result content signature": true, "top-level signature": true,
		"top-level encrypted_content": true, "unknown block encrypted": true, "message field signature": true, "tool_result string": false,
		"nested tool input signature": true, "user mystery block signature": true, "tool_use input encrypted_index": true, "tool_use input data": false,
	}
	for name, body := range capPlacements(v) {
		resp, out := postJSON(t, gw.URL+"/v1/messages", body, nil)
		got := strings.Contains(string(out), "opaque_too_large")
		if got != capped[name] || got != (resp.StatusCode == 422) {
			t.Errorf("%s: capped %v (want %v), status %d", name, got, capped[name], resp.StatusCode)
		}
	}
}

var nativeOpaque = map[string]bool{"thinking signature": true, "redacted_thinking data": true, "web search encrypted_content": true, "citation encrypted_index": true}

func TestCapThresholdsByPlacement(t *testing.T) {
	if corpusScale > 1 {
		t.Skip("the default cap needs values near 1 MB")
	}
	tr := &fnTransport{reply: echoReply}
	srv := newTestGateway(t, "masked", "", tr)
	gw := httptest.NewServer(srv.Handler())
	t.Cleanup(gw.Close)
	r := rand.New(rand.NewSource(1992))
	under := base64.StdEncoding.EncodeToString(uniformBytes(r, 921600*3/4))
	over := base64.StdEncoding.EncodeToString(uniformBytes(r, 900<<10))
	for name := range capPlacements("x") {
		for size, v := range map[string]string{"encoded 921600": under, "raw 900 KiB": over} {
			resp, out := postJSON(t, gw.URL+"/v1/messages", capPlacements(v)[name], nil)
			got := strings.Contains(string(out), "opaque_too_large")
			want := false
			switch {
			case name == "tool_result string" || name == "tool_use input data":
			case nativeOpaque[name]:
				want = true
			default:
				want = size == "raw 900 KiB"
			}
			if got != want || got != (resp.StatusCode == http.StatusUnprocessableEntity) {
				t.Errorf("%s with %s: capped %v (want %v), status %d", name, size, got, want, resp.StatusCode)
			}
		}
	}
	srv.cfg.MaxBodyBytes = 1 << 20
	resp, _ := postJSON(t, gw.URL+"/v1/messages", capPlacements(over)["tool_use input data"], nil)
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("a body over max_body_bytes: %d", resp.StatusCode)
	}
}
