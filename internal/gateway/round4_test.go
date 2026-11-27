package gateway

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
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

const probe4 = "Maria Garcia, phone (804) 555-1234, maria.garcia@example.com, SSN 123-45-6789"

var probe4Reals = []string{"Maria", "Garcia", "555-1234", "maria.garcia@example.com", "123-45-6789"}

type bypassCase struct {
	name string
	api  string
	body func(v string) map[string]any
}

func oaCase(name string, f func(b map[string]any, v string)) bypassCase {
	return bypassCase{name, config.APIOpenAI, func(v string) map[string]any {
		b := chatBody("hello", false)
		f(b, v)
		return b
	}}
}

func anCase(name string, f func(b map[string]any, v string)) bypassCase {
	return bypassCase{name, config.APIAnthropic, func(v string) map[string]any {
		b := anthBody("hello", false)
		f(b, v)
		return b
	}}
}

func firstMsg(b map[string]any) map[string]any {
	return b["messages"].([]any)[0].(map[string]any)
}

func runBypassCases(t *testing.T, policy string, cases []bypassCase) {
	t.Helper()
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, policy, tr)
	for _, c := range cases {
		before := len(tr.all())
		resp, out := postJSON(t, gw.URL+pathFor(c.api), c.body(probe4), nil)
		calls := tr.all()
		if resp.StatusCode == 200 {
			sent := calls[len(calls)-1].body
			for _, real := range probe4Reals {
				if strings.Contains(sent, real) {
					t.Fatalf("%s/%s: upstream holds %q: %s", c.api, c.name, real, sent)
				}
			}
		} else {
			if len(calls) != before {
				t.Fatalf("%s/%s: refused request reached upstream", c.api, c.name)
			}
			for _, real := range probe4Reals {
				if strings.Contains(string(out), real) {
					t.Fatalf("%s/%s: response holds %q: %s", c.api, c.name, real, out)
				}
			}
		}
	}
	btr := &fnTransport{reply: echoReply}
	bgw := newTestServer(t, policy+blockRules, btr)
	for _, c := range cases {
		resp, out := postJSON(t, bgw.URL+pathFor(c.api), c.body("topsecret"), nil)
		if resp.StatusCode != http.StatusForbidden && !(resp.StatusCode == http.StatusUnprocessableEntity && strings.Contains(string(out), "outworld_unsupported_content")) {
			t.Fatalf("%s/%s block rule: status %d %s", c.api, c.name, resp.StatusCode, out)
		}
	}
	if n := len(btr.all()); n != 0 {
		t.Fatalf("blocked requests reached upstream %d times", n)
	}
}

func TestProtocolFieldsNeedTheirShape(t *testing.T) {
	runBypassCases(t, "", []bypassCase{
		oaCase("model", func(b map[string]any, v string) { b["model"] = "gpt-x " + v }),
		oaCase("model word", func(b map[string]any, v string) { b["model"] = v }),
		oaCase("service_tier", func(b map[string]any, v string) { b["service_tier"] = v }),
		oaCase("reasoning_effort", func(b map[string]any, v string) { b["reasoning_effort"] = v }),
		oaCase("verbosity", func(b map[string]any, v string) { b["verbosity"] = v }),
		oaCase("verbosity array", func(b map[string]any, v string) { b["verbosity"] = []any{v} }),
		oaCase("tool_choice string", func(b map[string]any, v string) { b["tool_choice"] = v }),
		oaCase("role", func(b map[string]any, v string) { firstMsg(b)["role"] = v }),
		oaCase("tools type", func(b map[string]any, v string) {
			b["tools"] = []any{map[string]any{"type": v, "function": map[string]any{"name": "t"}}}
		}),
		oaCase("function strict", func(b map[string]any, v string) {
			b["tools"] = []any{map[string]any{"type": "function", "function": map[string]any{"name": "t", "strict": v}}}
		}),
		oaCase("tool_calls type", func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"type": v, "id": "call_1", "function": map[string]any{"name": "t", "arguments": "{}"}}}})
		}),
		oaCase("annotations type", func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": "ok", "annotations": []any{map[string]any{"type": v}}})
		}),
		oaCase("response_format type", func(b map[string]any, v string) { b["response_format"] = map[string]any{"type": v} }),
		oaCase("prediction type", func(b map[string]any, v string) { b["prediction"] = map[string]any{"type": v, "content": "x"} }),
		oaCase("custom format type", func(b map[string]any, v string) {
			b["tools"] = []any{map[string]any{"type": "custom", "custom": map[string]any{"name": "Edit", "format": map[string]any{"type": v}}}}
		}),
		oaCase("grammar syntax", func(b map[string]any, v string) {
			b["tools"] = []any{map[string]any{"type": "custom", "custom": map[string]any{"name": "Edit", "format": map[string]any{"type": "grammar", "grammar": map[string]any{"syntax": v, "definition": "start: WORD"}}}}}
		}),
		anCase("model", func(b map[string]any, v string) { b["model"] = "claude " + v }),
		anCase("model object", func(b map[string]any, v string) { b["model"] = map[string]any{"n": v} }),
		anCase("role object", func(b map[string]any, v string) { firstMsg(b)["role"] = map[string]any{"x": v} }),
		anCase("service_tier", func(b map[string]any, v string) { b["service_tier"] = v }),
		anCase("thinking type", func(b map[string]any, v string) { b["thinking"] = map[string]any{"type": v, "budget_tokens": 1024} }),
		anCase("tool_choice string", func(b map[string]any, v string) { b["tool_choice"] = v }),
		anCase("tool_choice type", func(b map[string]any, v string) { b["tool_choice"] = map[string]any{"type": v} }),
		anCase("tools type", func(b map[string]any, v string) { b["tools"] = []any{map[string]any{"type": v, "name": "t"}} }),
		anCase("tools strict", func(b map[string]any, v string) {
			b["tools"] = []any{map[string]any{"name": "t", "strict": v, "input_schema": map[string]any{"type": "object"}}}
		}),
		anCase("mcp server type", func(b map[string]any, v string) {
			b["mcp_servers"] = []any{map[string]any{"type": v, "url": "https://mcp.test/sse", "name": "m"}}
		}),
		anCase("output_format type", func(b map[string]any, v string) { b["output_format"] = map[string]any{"type": v} }),
		anCase("citation type", func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "text", "text": "ok", "citations": []any{map[string]any{"type": v, "cited_text": "x"}}}}})
		}),
		anCase("document media_type", func(b map[string]any, v string) {
			firstMsg(b)["content"] = []any{map[string]any{"type": "document", "source": map[string]any{"type": "text", "media_type": v, "data": "x"}}}
		}),
	})
}

func TestProtocolFieldsWithRealValuesPass(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	ob := chatBody("hello", false)
	ob["model"] = "gpt-5.1-codex"
	ob["service_tier"] = "flex"
	ob["reasoning_effort"] = "high"
	ob["verbosity"] = "low"
	ob["tool_choice"] = "required"
	ob["response_format"] = map[string]any{"type": "json_object"}
	ab := anthBody("hello", false)
	ab["model"] = "claude-opus-4-5@20251101"
	ab["service_tier"] = "auto"
	ab["thinking"] = map[string]any{"type": "adaptive"}
	ab["tool_choice"] = map[string]any{"type": "any"}
	ab["tools"] = []any{map[string]any{"type": "web_search_20250305", "name": "web_search", "max_uses": 3}, map[string]any{"type": "text_editor_20250728", "name": "str_replace_based_edit_tool"}}
	for _, c := range []struct {
		path string
		body map[string]any
	}{{"/v1/chat/completions", ob}, {"/v1/messages", ab}} {
		resp, out := postJSON(t, gw.URL+c.path, c.body, nil)
		if resp.StatusCode != 200 {
			t.Fatalf("%s: %d %s", c.path, resp.StatusCode, out)
		}
	}
}

func billingBody(line string) map[string]any {
	b := anthBody("hello", false)
	b["system"] = []any{map[string]any{"type": "text", "text": line}, map[string]any{"type": "text", "text": "be brief"}}
	return b
}

func TestBillingHeaderValueShapes(t *testing.T) {
	runBypassCases(t, "", []bypassCase{
		{"cch value", config.APIAnthropic, func(v string) map[string]any {
			return billingBody("x-anthropic-billing-header: cch=" + v + ";")
		}},
		{"unknown pair", config.APIAnthropic, func(v string) map[string]any {
			return billingBody("x-anthropic-billing-header: cc_version=1.0.0; cc_x=" + v + ";")
		}},
		{"entrypoint text", config.APIAnthropic, func(v string) map[string]any {
			return billingBody("x-anthropic-billing-header: cc_entrypoint=cli; " + v)
		}},
	})
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	good := "x-anthropic-billing-header: cc_version=2.1.3.abc; cc_entrypoint=cli; cch=0f3a9;"
	resp, out := postJSON(t, gw.URL+"/v1/messages", billingBody(good), nil)
	if resp.StatusCode != 200 || !strings.Contains(tr.all()[0].body, good) {
		t.Fatalf("real billing line: %d %s", resp.StatusCode, out)
	}
	if billingHeaderRE.MatchString("x-anthropic-billing-header: cch=maria.garcia;") {
		t.Fatal("cch must be hex")
	}
	resp, out = postJSON(t, gw.URL+"/v1/messages", billingBody("x-anthropic-billing-header: cc_entrypoint=mariagarcia;"), nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("known name inside a matching billing line: %d %s", resp.StatusCode, out)
	}
}

func TestSignatureAndEncryptedKeysAreNotExempt(t *testing.T) {
	runBypassCases(t, "", []bypassCase{
		oaCase("arguments signature", func(b map[string]any, v string) {
			a, _ := json.Marshal(map[string]any{"signature": v})
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"type": "function", "id": "call_1", "function": map[string]any{"name": "t", "arguments": string(a)}}}})
		}),
		oaCase("arguments encrypted_ssn", func(b map[string]any, v string) {
			a, _ := json.Marshal(map[string]any{"encrypted_ssn": v})
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"type": "function", "id": "call_1", "function": map[string]any{"name": "t", "arguments": string(a)}}}})
		}),
		oaCase("metadata encrypted_notes", func(b map[string]any, v string) { b["metadata"] = map[string]any{"encrypted_notes": v} }),
		anCase("tool_use input signature", func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "tool_use", "id": "toolu_01", "name": "t", "input": map[string]any{"signature": v}}}})
		}),
		anCase("tool_use input encrypted_notes", func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "tool_use", "id": "toolu_01", "name": "t", "input": map[string]any{"encrypted_notes": v}}}})
		}),
		anCase("schema default signature", func(b map[string]any, v string) {
			b["tools"] = schemaTool(config.APIAnthropic, map[string]any{"type": "object", "default": map[string]any{"signature": v}})
		}),
		anCase("text block signature", func(b map[string]any, v string) {
			firstMsg(b)["content"] = []any{map[string]any{"type": "text", "text": "hi", "signature": v}}
		}),
		anCase("metadata encrypted_notes", func(b map[string]any, v string) { b["metadata"] = map[string]any{"encrypted_notes": v} }),
		anCase("top-level encrypted_notes", func(b map[string]any, v string) { b["encrypted_notes"] = v }),
		anCase("citation encrypted_index text", func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "text", "text": "ok",
				"citations": []any{map[string]any{"type": "web_search_result_location", "cited_text": "x", "encrypted_index": v}}}}})
		}),
	})
}

func opaqueB64(seed string, n int) string {
	var out []byte
	h := sha256.Sum256([]byte(seed))
	for len(out) < n {
		out = append(out, h[:]...)
		h = sha256.Sum256(h[:])
	}
	return base64.StdEncoding.EncodeToString(out[:n])
}

var (
	sig        = opaqueB64("signature", 240)
	encContent = opaqueB64("encrypted", 300)
	shortEnc   = opaqueB64("short", 30)
)

func TestThinkingBlocksOnlyFromAssistant(t *testing.T) {
	runBypassCases(t, "", []bypassCase{
		anCase("thinking in user message", func(b map[string]any, v string) {
			firstMsg(b)["content"] = []any{map[string]any{"type": "thinking", "thinking": v, "signature": sig}, map[string]any{"type": "text", "text": "hi"}}
		}),
		anCase("assistant thinking extra key", func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "thinking", "thinking": "hmm", "signature": sig, "note": v}}})
		}),
		anCase("assistant thinking bad signature", func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "thinking", "thinking": v, "signature": "not base64 " + v}}})
		}),
		anCase("redacted_thinking data", func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "redacted_thinking", "data": v}}})
		}),
	})
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	b := anthBody("hello", false)
	b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{
		map[string]any{"type": "thinking", "thinking": "Kwame Asante asked about this.", "signature": sig},
		map[string]any{"type": "redacted_thinking", "data": sig},
		map[string]any{"type": "text", "text": "ok"}}}, map[string]any{"role": "user", "content": "go on"})
	resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
	if resp.StatusCode != 200 || !strings.Contains(tr.all()[0].body, "Kwame Asante asked about this.") {
		t.Fatalf("assistant thinking must pass unchanged: %d %s", resp.StatusCode, out)
	}
}

func TestIDLevelNeedsIDShapedValues(t *testing.T) {
	var cases []bypassCase
	for _, key := range []string{"user_id", "customer_id", "patient_id", "api_key", "license_key", "request_id", "userId", "user-id", "apiKey", "safety_identifier"} {
		for _, v := range []string{"Elena Petrova", "1985-03-14", "a8F3kL9qZ2xW7vB1nM4tY6uR0pE5sD3gH8jK2lQ9"} {
			key, v := key, v
			cases = append(cases, oaCase(key+"="+v, func(b map[string]any, _ string) { b["extra"] = map[string]any{key: v} }))
		}
	}
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, c := range cases {
		resp, out := postJSON(t, gw.URL+pathFor(c.api), c.body(""), nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("%s: status %d %s", c.name, resp.StatusCode, out)
		}
	}
	runBypassCases(t, "", []bypassCase{
		oaCase("metadata user_id", func(b map[string]any, v string) { b["metadata"] = map[string]any{"user_id": v} }),
		oaCase("tool_call_id", func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "tool", "tool_call_id": v, "content": "ok"})
		}),
		anCase("tool_use id", func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "tool_use", "id": v, "name": "t", "input": map[string]any{}}}})
		}),
	})
}

func pngWith(text string) string {
	raw, _ := base64.StdEncoding.DecodeString(pngB64)
	return base64.StdEncoding.EncodeToString(append(raw, []byte(text)...))
}

func TestMediaPathsWithImagesAllowed(t *testing.T) {
	b64 := base64.StdEncoding.EncodeToString
	runBypassCases(t, "  allow_images: true\n", []bypassCase{
		oaCase("made-up part type", func(b map[string]any, v string) {
			firstMsg(b)["content"] = []any{map[string]any{"type": "note", "text": v}}
		}),
		oaCase("image_url query", func(b map[string]any, v string) {
			firstMsg(b)["content"] = []any{map[string]any{"type": "image_url", "image_url": map[string]any{"url": "https://x.test/?q=" + v}}}
		}),
		oaCase("data url of text", func(b map[string]any, v string) {
			firstMsg(b)["content"] = []any{map[string]any{"type": "image_url", "image_url": map[string]any{"url": "data:image/png;base64," + b64([]byte(v))}}}
		}),
		oaCase("file filename", func(b map[string]any, v string) {
			firstMsg(b)["content"] = []any{map[string]any{"type": "file", "file": map[string]any{"filename": v, "file_data": "data:application/pdf;base64," + b64([]byte("%PDF-1.4\n"))}}}
		}),
		oaCase("audio of text", func(b map[string]any, v string) {
			firstMsg(b)["content"] = []any{map[string]any{"type": "input_audio", "input_audio": map[string]any{"format": "wav", "data": b64([]byte(v))}}}
		}),
		anCase("image url source", func(b map[string]any, v string) {
			firstMsg(b)["content"] = []any{map[string]any{"type": "image", "source": map[string]any{"type": "url", "url": "https://x.test/a.png?who=" + v}}}
		}),
		anCase("document url source", func(b map[string]any, v string) {
			firstMsg(b)["content"] = []any{map[string]any{"type": "document", "source": map[string]any{"type": "url", "url": "https://x.test/a.pdf?who=" + v}}}
		}),
		anCase("image data as plain text", func(b map[string]any, v string) {
			firstMsg(b)["content"] = []any{map[string]any{"type": "image", "source": map[string]any{"type": "base64", "media_type": "image/png", "data": v}}}
		}),
		anCase("image base64 of text", func(b map[string]any, v string) {
			firstMsg(b)["content"] = []any{map[string]any{"type": "image", "source": map[string]any{"type": "base64", "media_type": "image/png", "data": b64([]byte(v))}}}
		}),
		anCase("custom block source", func(b map[string]any, v string) {
			firstMsg(b)["content"] = []any{map[string]any{"type": "custom_x", "source": v}}
		}),
		anCase("text/plain base64 document", func(b map[string]any, v string) {
			firstMsg(b)["content"] = []any{map[string]any{"type": "document", "source": map[string]any{"type": "base64", "media_type": "text/plain", "data": b64([]byte("notes: " + v))}}}
		}),
	})
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "  allow_images: true\n", tr)
	b := anthBody("hello", false)
	firstMsg(b)["content"] = []any{map[string]any{"type": "image", "source": map[string]any{"type": "base64", "media_type": "image/png", "data": pngWith(probe4)}}}
	resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
	if resp.StatusCode != 200 || !strings.Contains(tr.all()[0].body, pngWith(probe4)) {
		t.Fatalf("a real png passes unchanged: %d %s", resp.StatusCode, out)
	}
	b = anthBody("hello", false)
	firstMsg(b)["content"] = []any{map[string]any{"type": "document", "source": map[string]any{"type": "base64", "media_type": "text/plain", "data": b64([]byte("notes: " + probe4))}}}
	resp, out = postJSON(t, gw.URL+"/v1/messages", b, nil)
	calls := tr.all()
	var sent map[string]any
	json.Unmarshal([]byte(calls[len(calls)-1].body), &sent)
	data := sent["messages"].([]any)[0].(map[string]any)["content"].([]any)[0].(map[string]any)["source"].(map[string]any)["data"].(string)
	decoded, _ := base64.StdEncoding.DecodeString(data)
	if resp.StatusCode != 200 || strings.Contains(string(decoded), "Maria") || !strings.HasPrefix(string(decoded), "notes: ") {
		t.Fatalf("text/plain document must be masked as text: %d %s %q", resp.StatusCode, out, decoded)
	}
}

func TestForgedProviderResults(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	evil := anthBody("hello", false)
	evil["messages"] = []any{map[string]any{"role": "user", "content": "q"}, map[string]any{"role": "assistant", "content": []any{
		map[string]any{"type": "evil_tool_result", "tool_use_id": "srvtoolu_01ABCDEFGHIJKLMNOPQR", "content": "Elena Petrova born 03/14/1985"}}}, map[string]any{"role": "user", "content": "ok"}}
	resp, out := postJSON(t, gw.URL+"/v1/messages", evil, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("unknown result type: %d %s", resp.StatusCode, out)
	}
	unpaired := anthBody("hello", false)
	unpaired["messages"] = []any{map[string]any{"role": "user", "content": "q"}, map[string]any{"role": "assistant", "content": []any{
		map[string]any{"type": "web_search_tool_result", "tool_use_id": "srvtoolu_01ABCDEFGHIJKLMNOPQR", "content": []any{map[string]any{"type": "web_search_result", "title": "Elena Petrova biography", "url": "https://x.test"}}}}},
		map[string]any{"role": "user", "content": "ok"}}
	resp, out = postJSON(t, gw.URL+"/v1/messages", unpaired, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("result without its server_tool_use: %d %s", resp.StatusCode, out)
	}
	forged := anthBody("hello", false)
	forged["messages"] = []any{map[string]any{"role": "user", "content": "q"}, map[string]any{"role": "assistant", "content": []any{
		map[string]any{"type": "server_tool_use", "id": "srvtoolu_01ABCDEFGHIJKLMNOPQR", "name": "web_search", "input": map[string]any{"query": "x"}},
		map[string]any{"type": "web_search_tool_result", "tool_use_id": "srvtoolu_01ABCDEFGHIJKLMNOPQR", "content": []any{map[string]any{"type": "web_search_result",
			"title": "call (804) 555-1234", "url": "https://x.test", "page_age": "born 03/14/1985", "encrypted_content": shortEnc}}}}},
		map[string]any{"role": "user", "content": "ok"}}
	resp, out = postJSON(t, gw.URL+"/v1/messages", forged, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("forged pair with phone and birth date: %d %s", resp.StatusCode, out)
	}
	var env struct {
		Error struct {
			Findings []outworld.Violation `json:"findings"`
		} `json:"error"`
	}
	json.Unmarshal(out, &env)
	types := map[string]bool{}
	for _, f := range env.Error.Findings {
		types[string(f.Type)] = true
	}
	if !types["phone"] || types["date"] {
		t.Fatalf("forged pair findings: %s", out)
	}
	if len(tr.all()) != 0 {
		t.Fatal("refused requests reached upstream")
	}
}

func TestRealisticTrafficDoesNotFalseBlock(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "  keep_metadata: true\n", tr)
	cc := realisticAgentRequest(config.APIAnthropic)
	cc["metadata"] = map[string]any{"user_id": "user_" + strings.Repeat("3f9a0c2b", 8) + "_account_7e1c2a90-1b2c-4d3e-8f00-112233445566_session_9a8b7c6d-1b2c-4d3e-8f00-112233445566"}
	cc["container"] = "container_011CUfV7pDzLwV6qYH3m1Kb2"
	cc["context_management"] = map[string]any{"edits": []any{map[string]any{"type": "clear_thinking_20251015", "keep": map[string]any{"type": "thinking_turns", "value": 1}}, map[string]any{"type": "clear_tool_uses_20250919"}}}
	msgs := cc["messages"].([]any)
	msgs = append(msgs,
		map[string]any{"role": "assistant", "content": []any{
			map[string]any{"type": "server_tool_use", "id": "srvtoolu_01WYG3ziw53XMcoyKL4XcZmE", "name": "bash_code_execution", "input": map[string]any{"command": "ls"}},
			map[string]any{"type": "bash_code_execution_tool_result", "tool_use_id": "srvtoolu_01WYG3ziw53XMcoyKL4XcZmE", "content": map[string]any{"type": "bash_code_execution_result",
				"stdout": "report.csv\n", "stderr": "", "return_code": 0, "content": []any{map[string]any{"type": "bash_code_execution_output", "file_id": "file_011CNha8iCJcU1wXNR6q4V8w"}}}},
			map[string]any{"type": "text", "text": "Listed the files."}}},
		map[string]any{"role": "user", "content": "thanks"})
	msgs = append(msgs,
		map[string]any{"role": "assistant", "content": []any{
			map[string]any{"type": "server_tool_use", "id": "srvtoolu_02AbCdEfGh1JkLmNoPqRsT", "name": "web_search", "input": map[string]any{"query": "release notes"}},
			map[string]any{"type": "web_search_tool_result", "tool_use_id": "srvtoolu_02AbCdEfGh1JkLmNoPqRsT", "content": []any{map[string]any{"type": "web_search_result",
				"title": "Release notes for March 14, 2025", "url": "https://x.test/notes", "page_age": "2025-03-14", "encrypted_content": encContent}}},
			map[string]any{"type": "server_tool_use", "id": "srvtoolu_03AbCdEfGh1JkLmNoPqRsT", "name": "web_fetch", "input": map[string]any{"url": "https://x.test/notes"}},
			map[string]any{"type": "web_fetch_tool_result", "tool_use_id": "srvtoolu_03AbCdEfGh1JkLmNoPqRsT", "content": map[string]any{"type": "web_fetch_result", "url": "https://x.test/notes",
				"retrieved_at": "2025-03-20T10:00:00Z", "content": map[string]any{"type": "document", "title": "Notes", "source": map[string]any{"type": "text", "media_type": "text/plain",
					"data": "Version 2.1 shipped on 2025-03-14. The previous release was on 03/01/2025."}}}},
			map[string]any{"type": "text", "text": "Found them."}}},
		map[string]any{"role": "user", "content": "good"})
	cc["messages"] = msgs
	codex := realisticAgentRequest(config.APIOpenAI)
	codex["prompt_cache_key"] = "9a8b7c6d-1b2c-4d3e-8f00-112233445566"
	codex["safety_identifier"] = "8f14e45fceea167a5a36dedd4bea2543b2f1c9a0e1d2c3b4a5968778695a4b3c"
	codex["user"] = "8f14e45fceea167a5a36dedd4bea2543b2f1c9a0e1d2c3b4a5968778695a4b3c"
	codex["metadata"] = map[string]any{"commit_id": "3e7c9b1d2a4f6e8c0b5d7a9f1e3c5b7d9a2c4e6f", "user_id": "8f14e45fceea167a5a36dedd4bea2543b2f1c9a0e1d2c3b4a5968778695a4b3c"}
	for _, c := range []struct {
		api  string
		body map[string]any
		keep []string
	}{
		{config.APIAnthropic, cc, []string{sig, attributionText, "toolu_01A09q90qw90lq917835lq9", "srvtoolu_01WYG3ziw53XMcoyKL4XcZmE", "file_011CNha8iCJcU1wXNR6q4V8w", "container_011CUfV7pDzLwV6qYH3m1Kb2", "clear_thinking_20251015"}},
		{config.APIOpenAI, codex, []string{"call_Xk3n9QzL2pR7tV1wY5bA8cD0", "9a8b7c6d-1b2c-4d3e-8f00-112233445566", "8f14e45fceea167a5a36dedd4bea2543b2f1c9a0e1d2c3b4a5968778695a4b3c", "3e7c9b1d2a4f6e8c0b5d7a9f1e3c5b7d9a2c4e6f", `{\"command\":\"go test ./pkg/server/...\",\"description\":\"Run server tests\"}`}},
	} {
		for range 2 {
			resp, out := postJSON(t, gw.URL+pathFor(c.api), c.body, nil)
			if resp.StatusCode != 200 {
				t.Fatalf("%s: status %d %s", c.api, resp.StatusCode, out)
			}
		}
		calls := tr.all()
		last, prev := calls[len(calls)-1].body, calls[len(calls)-2].body
		if last != prev {
			t.Fatalf("%s: repeated request bytes differ", c.api)
		}
		for _, k := range c.keep {
			if !strings.Contains(last, k) {
				t.Fatalf("%s: %q changed: %s", c.api, k, last)
			}
		}
	}
}

func TestLocalLocalWithoutUpstreamIs400(t *testing.T) {
	tr := &fnTransport{reply: okReply}
	gw := newTestServer(t, "", tr)
	resp, out := postJSON(t, gw.URL+"/v1/messages", anthBody("hi", false), map[string]string{HeaderRoute: "local:local"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("local:local without anthropic local upstream: %d %s", resp.StatusCode, out)
	}
}

func TestPassthroughDoneOnlyAtLineStart(t *testing.T) {
	pr, pw := io.Pipe()
	t.Cleanup(func() { pw.Close() })
	tr := &fnTransport{reply: func(_ string, _ map[string]any) (*http.Response, error) {
		go io.WriteString(pw, "data: {\"id\":\"c1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"say data: [DONE] or event: message_stop\"}}]}\n\n")
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
	for !strings.Contains(got.String(), "message_stop") {
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
	if !strings.Contains(got.String(), "outworld_shutdown") {
		t.Fatalf("an unfinished stream must get the shutdown event: %q", got.String())
	}
}

func TestCheckedValuePreviewAndHint(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	b := chatBody("hello", false)
	b["safety_identifier"] = "clear_thinking_for maria.garcia@example.com today"
	resp, out := postJSON(t, gw.URL+"/v1/chat/completions", b, nil)
	var env struct {
		Error struct {
			Findings []outworld.Violation `json:"findings"`
		} `json:"error"`
	}
	json.Unmarshal(out, &env)
	if resp.StatusCode != http.StatusUnprocessableEntity || len(env.Error.Findings) == 0 {
		t.Fatalf("%d %s", resp.StatusCode, out)
	}
	for _, f := range env.Error.Findings {
		if f.Preview != "[***]" || strings.Contains(f.Hint, "object key") {
			t.Fatalf("checked value finding: %+v", f)
		}
	}
}
