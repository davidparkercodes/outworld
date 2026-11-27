package gateway

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/config"
	"github.com/davidparkercodes/outworld/pkg/outworld"
)

const probe = "Maria Garcia, phone (804) 555-1234"

var probeReals = []string{"Maria", "Garcia", "555-1234"}

const toolPII = "Email jane.doe@example.com or call John Smith, SSN 123-45-6789"

var toolReals = []string{"jane.doe@example.com", "John", "Smith", "123-45-6789"}

func openAITools(desc string) []any {
	return []any{map[string]any{"type": "function", "function": map[string]any{
		"name":        "send_note",
		"description": desc,
		"parameters": map[string]any{
			"type":     "object",
			"title":    "Note",
			"required": []any{"to", "description"},
			"properties": map[string]any{
				"to":          map[string]any{"type": "string", "description": desc, "format": "email", "default": "nobody"},
				"description": map[string]any{"type": "string", "description": desc, "enum": []any{"low", "high"}, "maxLength": 40},
				"nested":      map[string]any{"type": "array", "items": map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{"who": map[string]any{"type": "string", "description": desc, "examples": []any{"someone"}}}}},
			},
		},
	}}}
}

func anthropicTools(desc string) []any {
	return []any{map[string]any{
		"name":        "send_note",
		"description": desc,
		"input_schema": map[string]any{
			"type":     "object",
			"required": []any{"to"},
			"properties": map[string]any{
				"to":   map[string]any{"type": "string", "description": desc, "default": "nobody", "enum": []any{"a", "b"}},
				"type": map[string]any{"type": "string", "description": desc},
			},
		},
	}}
}

func blankMasked(t *testing.T, api string, v map[string]any) string {
	t.Helper()
	c := clone(t, v).(map[string]any)
	f := bareServer(config.PolicyConfig{AllowImages: true, KeepMetadata: true}).collect(c, api)
	for _, l := range f.leaves {
		if l.mode == modeMask {
			l.set("")
		}
	}
	for _, cm := range f.commits {
		cm()
	}
	return string(encode(c))
}

func maskedTexts(t *testing.T, api string, v map[string]any) []string {
	t.Helper()
	var out []string
	for _, l := range bareServer(config.PolicyConfig{AllowImages: true, KeepMetadata: true}).collect(clone(t, v).(map[string]any), api).leaves {
		if l.mode == modeMask {
			out = append(out, l.text)
		}
	}
	return out
}

func echoReply(host string, _ map[string]any) (*http.Response, error) {
	if host == "claude.test" {
		return jsonResponse(map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "text", "text": "ok"}}}), nil
	}
	return okReply(host, nil)
}

func TestToolDescriptionsAndMetadataMasked(t *testing.T) {
	for _, api := range []string{config.APIOpenAI, config.APIAnthropic} {
		t.Run(api, func(t *testing.T) {
			tr := &fnTransport{reply: echoReply}
			gw := newTestServer(t, "", tr)
			var path string
			var body map[string]any
			if api == config.APIOpenAI {
				path = "/v1/chat/completions"
				body = chatBody("Please write to John Smith", false)
				body["tools"] = openAITools(toolPII)
				body["metadata"] = map[string]any{"note": "for jane.doe@example.com", "ticket": "A-1"}
			} else {
				path = "/v1/messages"
				body = anthBody("Please write to John Smith", false)
				body["tools"] = anthropicTools(toolPII)
			}
			for range 2 {
				resp, out := postJSON(t, gw.URL+path, body, nil)
				if resp.StatusCode != 200 {
					t.Fatalf("status %d %s", resp.StatusCode, out)
				}
			}
			calls := tr.all()
			if len(calls) != 2 || calls[0].body != calls[1].body {
				t.Fatalf("identical requests must send identical bytes")
			}
			var sent map[string]any
			json.Unmarshal([]byte(calls[0].body), &sent)
			for _, real := range toolReals {
				if strings.Contains(calls[0].body, real) {
					t.Fatalf("upstream body holds %q: %s", real, calls[0].body)
				}
			}
			orig := clone(t, body).(map[string]any)
			delete(orig, "user")
			delete(sent, "user")
			if blankMasked(t, api, map[string]any{"tools": sent["tools"]}) != blankMasked(t, api, map[string]any{"tools": orig["tools"]}) {
				t.Fatal("structural schema changed")
			}
			msgs, _ := sent["messages"].([]any)
			last, _ := msgs[len(msgs)-1].(map[string]any)["content"].(string)
			surname := strings.Fields(last)[len(strings.Fields(last))-1]
			if !strings.Contains(calls[0].body, surname+", SSN") {
				t.Fatalf("tool description and message must share surrogates: %q", last)
			}
			if api == config.APIOpenAI {
				md := sent["metadata"].(map[string]any)
				if strings.Contains(md["note"].(string), "jane.doe@example.com") || md["ticket"] != "A-1" {
					t.Fatalf("metadata: %v", md)
				}
			}
		})
	}
}

type fieldCase struct {
	name string
	api  string
	body func(v string) map[string]any
}

func schemaTool(api string, schema map[string]any) []any {
	if api == config.APIAnthropic {
		return []any{map[string]any{"name": "t", "description": "d", "input_schema": schema}}
	}
	return []any{map[string]any{"type": "function", "function": map[string]any{"name": "t", "description": "d", "parameters": schema}}}
}

func prop(key string, v any) map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"p": map[string]any{"type": "string", key: v}}}
}

func fieldCases() []fieldCase {
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
	var cases []fieldCase
	for _, api := range []string{config.APIOpenAI, config.APIAnthropic} {
		wrap := oa
		if api == config.APIAnthropic {
			wrap = an
		}
		for _, k := range []string{"description", "title", "default", "const"} {
			cases = append(cases, fieldCase{"schema " + k, api, wrap(func(b map[string]any, v string) { b["tools"] = schemaTool(api, prop(k, v)) })})
		}
		for _, k := range []string{"enum", "examples"} {
			cases = append(cases, fieldCase{"schema " + k, api, wrap(func(b map[string]any, v string) { b["tools"] = schemaTool(api, prop(k, []any{"x", v})) })})
		}
		cases = append(cases, fieldCase{"schema title top", api, wrap(func(b map[string]any, v string) {
			b["tools"] = schemaTool(api, map[string]any{"type": "object", "title": v})
		})})
		cases = append(cases, fieldCase{"tool description", api, wrap(func(b map[string]any, v string) {
			b["tools"] = schemaTool(api, map[string]any{"type": "object"})
			tool := b["tools"].([]any)[0].(map[string]any)
			if fn, ok := tool["function"].(map[string]any); ok {
				fn["description"] = v
			} else {
				tool["description"] = v
			}
		})})
	}
	cases = append(cases,
		fieldCase{"prediction string", config.APIOpenAI, oa(func(b map[string]any, v string) {
			b["prediction"] = map[string]any{"type": "content", "content": v}
		})},
		fieldCase{"prediction parts", config.APIOpenAI, oa(func(b map[string]any, v string) {
			b["prediction"] = map[string]any{"type": "content", "content": []any{map[string]any{"type": "text", "text": v}}}
		})},
		fieldCase{"refusal part", config.APIOpenAI, oa(func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "refusal", "refusal": v}}})
		})},
		fieldCase{"response_format schema", config.APIOpenAI, oa(func(b map[string]any, v string) {
			b["response_format"] = map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "out", "description": "d", "schema": prop("enum", []any{v})}}
		})},
		fieldCase{"response_format description", config.APIOpenAI, oa(func(b map[string]any, v string) {
			b["response_format"] = map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "out", "description": v, "schema": map[string]any{"type": "object"}}}
		})},
		fieldCase{"stop string", config.APIOpenAI, oa(func(b map[string]any, v string) { b["stop"] = v })},
		fieldCase{"stop array", config.APIOpenAI, oa(func(b map[string]any, v string) { b["stop"] = []any{"\n\n", v} })},
		fieldCase{"web_search_options", config.APIOpenAI, oa(func(b map[string]any, v string) {
			b["web_search_options"] = map[string]any{"user_location": map[string]any{"type": "approximate", "approximate": map[string]any{"city": v}}}
		})},
		fieldCase{"metadata", config.APIOpenAI, oa(func(b map[string]any, v string) { b["metadata"] = map[string]any{"k": v} })},
		fieldCase{"input", config.APIOpenAI, oa(func(b map[string]any, v string) { b["input"] = v })},
		fieldCase{"suffix", config.APIOpenAI, oa(func(b map[string]any, v string) { b["suffix"] = v })},
		fieldCase{"reasoning", config.APIOpenAI, oa(func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": "ok", "reasoning": v})
		})},
		fieldCase{"annotation title", config.APIOpenAI, oa(func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": "ok", "annotations": []any{map[string]any{"type": "url_citation", "url_citation": map[string]any{"title": v, "url": "https://x.test", "start_index": 0, "end_index": 2}}}})
		})},
		fieldCase{"unknown schema keyword", config.APIAnthropic, an(func(b map[string]any, v string) {
			b["tools"] = schemaTool(config.APIAnthropic, map[string]any{"type": "object", "x-note": map[string]any{"k": []any{1, v}}})
		})},
		fieldCase{"output_format schema", config.APIAnthropic, an(func(b map[string]any, v string) {
			b["output_format"] = map[string]any{"type": "json_schema", "schema": prop("description", v)}
		})},
		fieldCase{"custom tool description", config.APIOpenAI, oa(func(b map[string]any, v string) {
			b["tools"] = []any{map[string]any{"type": "custom", "custom": map[string]any{"name": "Edit", "description": v}}}
		})},
		fieldCase{"custom tool call input", config.APIOpenAI, oa(func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"type": "custom", "id": "call_1", "custom": map[string]any{"name": "Edit", "input": v}}}})
		})},
		fieldCase{"stop_sequences", config.APIAnthropic, an(func(b map[string]any, v string) { b["stop_sequences"] = []any{v} })},
		fieldCase{"server tool user_location", config.APIAnthropic, an(func(b map[string]any, v string) {
			b["tools"] = []any{map[string]any{"type": "web_search_20250305", "name": "web_search", "user_location": map[string]any{"type": "approximate", "city": v}}}
		})},
		fieldCase{"citation cited_text", config.APIAnthropic, an(func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "text", "text": "ok",
				"citations": []any{map[string]any{"type": "char_location", "cited_text": v, "document_index": 0, "start_char_index": 0, "end_char_index": 5}}}}})
		})},
		fieldCase{"citation document_title", config.APIAnthropic, an(func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "text", "text": "ok",
				"citations": []any{map[string]any{"type": "char_location", "cited_text": "x", "document_title": v}}}}})
		})},
	)
	return cases
}

func pathFor(api string) string {
	if api == config.APIAnthropic {
		return "/v1/messages"
	}
	return "/v1/chat/completions"
}

func TestEveryTextFieldIsMasked(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, c := range fieldCases() {
		before := len(tr.all())
		resp, out := postJSON(t, gw.URL+pathFor(c.api), c.body(probe), nil)
		calls := tr.all()
		if resp.StatusCode != 200 || len(calls) != before+1 {
			t.Fatalf("%s/%s: status %d %s", c.api, c.name, resp.StatusCode, out)
		}
		for _, real := range probeReals {
			if strings.Contains(calls[len(calls)-1].body, real) {
				t.Fatalf("%s/%s: upstream holds %q: %s", c.api, c.name, real, calls[len(calls)-1].body)
			}
		}
	}
}

func TestEveryTextFieldIsSeenByRules(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, blockRules, tr)
	for _, c := range fieldCases() {
		for _, v := range []string{"this is topsecret", "SSN 123-45-6789"} {
			resp, out := postJSON(t, gw.URL+pathFor(c.api), c.body(v), nil)
			if resp.StatusCode != http.StatusForbidden {
				t.Fatalf("%s/%s %q: status %d %s", c.api, c.name, v, resp.StatusCode, out)
			}
		}
	}
	if n := len(tr.all()); n != 0 {
		t.Fatalf("blocked requests reached upstream %d times", n)
	}
}

func TestSchemaPropertyNamesAreCheckedNotRewritten(t *testing.T) {
	for _, api := range []string{config.APIOpenAI, config.APIAnthropic} {
		tr := &fnTransport{reply: echoReply}
		gw := newTestServer(t, "", tr)
		body := chatBody("hello", false)
		if api == config.APIAnthropic {
			body = anthBody("hello", false)
		}
		body["tools"] = schemaTool(api, map[string]any{"type": "object", "properties": map[string]any{"maria.garcia@example.com": map[string]any{"type": "string"}}})
		resp, out := postJSON(t, gw.URL+pathFor(api), body, nil)
		if resp.StatusCode != http.StatusUnprocessableEntity || len(tr.all()) != 0 || strings.Contains(string(out), "maria") {
			t.Fatalf("%s: sensitive property name: %d %s", api, resp.StatusCode, out)
		}
		body["tools"] = schemaTool(api, map[string]any{"type": "object", "properties": map[string]any{"recipient": map[string]any{"type": "string"}}})
		resp, _ = postJSON(t, gw.URL+pathFor(api), body, nil)
		if resp.StatusCode != 200 || !strings.Contains(tr.all()[0].body, `"recipient"`) {
			t.Fatalf("%s: plain property name must pass unchanged", api)
		}
	}
}

func TestMaskedEnumRoundTripsThroughToolCall(t *testing.T) {
	enumOf := func(req map[string]any, api string) string {
		tools := req["tools"].([]any)[0].(map[string]any)
		schema, _ := tools["input_schema"].(map[string]any)
		if fn, ok := tools["function"].(map[string]any); ok {
			schema = fn["parameters"].(map[string]any)
		}
		return schema["properties"].(map[string]any)["p"].(map[string]any)["enum"].([]any)[0].(string)
	}
	for _, api := range []string{config.APIOpenAI, config.APIAnthropic} {
		var seen string
		tr := &fnTransport{}
		tr.reply = func(_ string, req map[string]any) (*http.Response, error) {
			seen = enumOf(req, api)
			args, _ := json.Marshal(map[string]any{"p": seen})
			if api == config.APIAnthropic {
				return jsonResponse(map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "tool_use", "id": "t1", "name": "t", "input": map[string]any{"p": seen}}}}), nil
			}
			return jsonResponse(map[string]any{"id": "x", "choices": []any{map[string]any{"index": 0, "message": map[string]any{"role": "assistant",
				"tool_calls": []any{map[string]any{"id": "t1", "type": "function", "function": map[string]any{"name": "t", "arguments": string(args)}}}}}}}), nil
		}
		gw := newTestServer(t, "", tr)
		body := chatBody("hello", false)
		if api == config.APIAnthropic {
			body = anthBody("hello", false)
		}
		body["tools"] = schemaTool(api, prop("enum", []any{"Maria Garcia", "nobody"}))
		resp, out := postJSON(t, gw.URL+pathFor(api), body, nil)
		if resp.StatusCode != 200 || seen == "" || strings.Contains(seen, "Maria") {
			t.Fatalf("%s: enum sent as %q: %d %s", api, seen, resp.StatusCode, out)
		}
		if !strings.Contains(string(out), "Maria Garcia") || strings.Contains(string(out), seen) {
			t.Fatalf("%s: tool call not rehydrated (%q): %s", api, seen, out)
		}
	}
}

const pngB64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="

func TestImageDataURLsStayOutOfRouting(t *testing.T) {
	s := bareServer(config.PolicyConfig{AllowImages: true})
	req := map[string]any{"messages": []any{map[string]any{"role": "user", "content": []any{
		map[string]any{"type": "text", "text": "look"},
		map[string]any{"type": "image_url", "image_url": map[string]any{"url": "data:image/png;base64," + pngB64}},
	}}}}
	if got := s.routingText(req, config.APIOpenAI).full; got != "look" {
		t.Fatalf("openai routing text: %q", got)
	}
	areq := map[string]any{"messages": []any{map[string]any{"role": "user", "content": []any{
		map[string]any{"type": "text", "text": "look"},
		map[string]any{"type": "image", "source": map[string]any{"type": "base64", "media_type": "image/png", "data": pngB64}},
	}}}}
	if got := s.routingText(areq, config.APIAnthropic).full; got != "look" {
		t.Fatalf("anthropic routing text: %q", got)
	}
}

func searchResultBody(title string) map[string]any {
	body := anthBody("hello", false)
	body["messages"] = []any{
		map[string]any{"role": "user", "content": "search"},
		map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "server_tool_use", "id": "srvtoolu_01WYG3ziw53XMcoyKL4XcZmE", "name": "web_search", "input": map[string]any{"query": "biography"}},
			map[string]any{"type": "web_search_tool_result", "tool_use_id": "srvtoolu_01WYG3ziw53XMcoyKL4XcZmE", "content": []any{
				map[string]any{"type": "web_search_result", "title": title, "url": "https://x.test/page", "encrypted_content": shortEnc, "page_age": "2 days ago"}}}}},
		map[string]any{"role": "user", "content": "thanks"},
	}
	return body
}

func TestBillingHeaderExactShapeOnly(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, blockRules, tr)
	body := anthBody("hello", false)
	body["system"] = []any{map[string]any{"type": "text", "text": attributionText}, map[string]any{"type": "text", "text": "be brief"}}
	resp, out := postJSON(t, gw.URL+"/v1/messages", body, nil)
	if resp.StatusCode != 200 || !strings.Contains(tr.all()[0].body, attributionText) {
		t.Fatalf("exact billing header must pass unchanged: %d %s", resp.StatusCode, out)
	}
	body["system"] = []any{map[string]any{"type": "text", "text": attributionText + " topsecret"}}
	resp, out = postJSON(t, gw.URL+"/v1/messages", body, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("billing prefix must not hide text from rules: %d %s", resp.StatusCode, out)
	}
	body["system"] = []any{map[string]any{"type": "text", "text": "x-anthropic-billing-header: " + probe}}
	before := len(tr.all())
	resp, out = postJSON(t, gw.URL+"/v1/messages", body, nil)
	calls := tr.all()
	if resp.StatusCode != 200 || len(calls) != before+1 {
		t.Fatalf("status %d %s", resp.StatusCode, out)
	}
	for _, real := range probeReals {
		if strings.Contains(calls[len(calls)-1].body, real) {
			t.Fatalf("billing prefix must not skip masking: %s", calls[len(calls)-1].body)
		}
	}
}

func TestProviderToolResultsAreNarrowChecked(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, blockRules, tr)
	resp, out := postJSON(t, gw.URL+"/v1/messages", searchResultBody("Ada Lovelace biography"), nil)
	if resp.StatusCode != 200 || !strings.Contains(tr.all()[0].body, "Ada Lovelace biography") || !strings.Contains(tr.all()[0].body, shortEnc) {
		t.Fatalf("public names in provider results must pass unchanged: %d %s", resp.StatusCode, out)
	}
	resp, out = postJSON(t, gw.URL+"/v1/messages", searchResultBody("this is topsecret"), nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("block rule must see provider results: %d %s", resp.StatusCode, out)
	}
	resp, out = postJSON(t, gw.URL+"/v1/messages", searchResultBody("call (804) 555-1234 or mail maria.garcia@example.com"), nil)
	if resp.StatusCode != http.StatusUnprocessableEntity || strings.Contains(string(out), "555-1234") {
		t.Fatalf("regex detectors must see provider results: %d %s", resp.StatusCode, out)
	}
	if len(tr.all()) != 1 {
		t.Fatal("blocked requests reached upstream")
	}
}

func TestUnknownAnthropicBlocksAreLeakChecked(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	body := anthBody("hello", false)
	body["messages"] = []any{map[string]any{"role": "user", "content": []any{map[string]any{"type": "future_block", "note": "mail maria.garcia@example.com", "file_id": "f_abc"}}}}
	resp, out := postJSON(t, gw.URL+"/v1/messages", body, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity || len(tr.all()) != 0 {
		t.Fatalf("unknown block with PII: %d %s", resp.StatusCode, out)
	}
	body["messages"] = []any{map[string]any{"role": "user", "content": []any{map[string]any{"type": "future_block", "note": "nothing here", "file_id": "f_abc"}}}}
	resp, _ = postJSON(t, gw.URL+"/v1/messages", body, nil)
	if resp.StatusCode != 200 || !strings.Contains(tr.all()[0].body, `"note":"nothing here"`) {
		t.Fatalf("unknown block must pass unchanged: %d", resp.StatusCode)
	}
}

func TestKeptAnthropicUserIDIsChecked(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "  keep_metadata: true\n", tr)
	body := anthBody("hello", false)
	body["metadata"] = map[string]any{"user_id": "maria.garcia@example.com"}
	resp, out := postJSON(t, gw.URL+"/v1/messages", body, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity || len(tr.all()) != 0 {
		t.Fatalf("kept user_id: %d %s", resp.StatusCode, out)
	}
}

func TestDistinctValuesInOneObjectAreCountedSeparately(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	body := chatBody("hello", false)
	body["metadata"] = map[string]any{"maria.garcia@example.com": "a", "john.smith@gmail.com": "b", "tel 804-555-1234": "c"}
	resp, out := postJSON(t, gw.URL+"/v1/chat/completions", body, nil)
	var env struct {
		Error struct {
			Message  string               `json:"message"`
			Findings []outworld.Violation `json:"findings"`
		} `json:"error"`
	}
	json.Unmarshal(out, &env)
	types := map[string]int{}
	for _, f := range env.Error.Findings {
		types[string(f.Type)]++
	}
	if resp.StatusCode != http.StatusUnprocessableEntity || types["email"] != 2 || types["phone"] != 1 || len(env.Error.Findings) != 3 ||
		!strings.Contains(env.Error.Message, "3 sensitive value(s)") {
		t.Fatalf("findings: %d %s", resp.StatusCode, out)
	}
}

func TestDedupeKeepsSecretInsideURL(t *testing.T) {
	vs := []outworld.Violation{
		{Type: "url", Source: "regex:url", Field: "metadata.#0(key)", Start: 0, End: 40},
		{Type: "secret", Source: "regex:vendor_token", Field: "metadata.#0(key)", Start: 21, End: 40},
		{Type: "person", Source: "names:full", Field: "metadata.#1(key)", Start: 0, End: 10},
		{Type: "email", Source: "regex:email", Field: "metadata.#1(key)", Start: 0, End: 20},
		{Type: "email", Source: "vault", Field: "metadata.#2(key)", Start: -1, End: -1},
		{Type: "email", Source: "regex:email", Field: "metadata.#2(key)", Start: -1, End: -1},
	}
	got := DedupeViolations(vs)
	types := map[string]int{}
	for _, v := range got {
		types[string(v.Type)]++
	}
	if len(got) != 5 || types["url"] != 1 || types["secret"] != 1 || types["email"] != 3 || types["person"] != 0 {
		t.Fatalf("dedupe: %+v", got)
	}
}

func agentToolSpecs() []map[string]any {
	str := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
	return []map[string]any{
		{"name": "Bash", "description": "Executes a bash command in a persistent shell and returns its output.\n\nUsage:\n- Quote paths that contain spaces, for example cd \"/src/my project\".\n- Prefer rg over grep, and never run git push --force on main.\n- To commit, run git commit -m \"fix: handle empty input\" and include the trailer Co-Authored-By: Agent <noreply@example.com>.\n- Use the timeout parameter for long builds such as make -j8 or go test ./... -race.\n- Will fail if the command requires interactive input.",
			"schema": map[string]any{"type": "object", "required": []any{"command"}, "properties": map[string]any{
				"command":     str("The command to execute, for example ls -la /var/log or npm run build"),
				"timeout":     map[string]any{"type": "number", "description": "Optional timeout in milliseconds (max 600000)"},
				"description": str("Clear, concise description of what this command does in 5-10 words"),
			}}},
		{"name": "Read", "description": "Reads a file from the local filesystem. The file_path parameter must be an absolute path such as /home/user/project/src/main.go, not a relative path. Lines longer than 2000 characters are truncated. Mark any binary file as unsupported.",
			"schema": map[string]any{"type": "object", "required": []any{"file_path"}, "properties": map[string]any{
				"file_path": str("The absolute path to the file to read"),
				"offset":    map[string]any{"type": "integer", "description": "The line number to start reading from"},
				"limit":     map[string]any{"type": "integer", "description": "The number of lines to read"},
			}}},
		{"name": "Edit", "description": "Performs exact string replacements in files. Grant yourself a read of the file first; the edit will FAIL if old_string is not unique. Use replace_all to rename a variable such as billTotal across the file.",
			"schema": map[string]any{"type": "object", "required": []any{"file_path", "old_string", "new_string"}, "properties": map[string]any{
				"file_path":   str("The absolute path to the file to modify"),
				"old_string":  str("The text to replace"),
				"new_string":  str("The text to replace it with (must be different from old_string)"),
				"replace_all": map[string]any{"type": "boolean", "default": false, "description": "Replace all occurrences of old_string"},
			}}},
		{"name": "Grep", "description": "A powerful search tool built on ripgrep. Supports full regex syntax, e.g. \"log.*Error\" or \"function\\s+\\w+\". Filter files with glob (\"*.js\", \"**/*.tsx\") or type (\"py\", \"rust\"). Output modes: content, files_with_matches, count.",
			"schema": map[string]any{"type": "object", "required": []any{"pattern"}, "properties": map[string]any{
				"pattern":     str("The regular expression pattern to search for in file contents"),
				"path":        str("File or directory to search in. Defaults to the current working directory."),
				"output_mode": map[string]any{"type": "string", "enum": []any{"content", "files_with_matches", "count"}, "description": "Output mode"},
				"-i":          map[string]any{"type": "boolean", "description": "Case insensitive search"},
			}}},
		{"name": "WebFetch", "description": "Fetches content from a URL such as https://docs.example.com/guide and processes it with a prompt. HTTP URLs are upgraded to HTTPS. Results may be summarized if the content is very large. Bill only counts on fresh fetches.",
			"schema": map[string]any{"type": "object", "required": []any{"url", "prompt"}, "properties": map[string]any{
				"url":    map[string]any{"type": "string", "format": "uri", "description": "The URL to fetch content from"},
				"prompt": str("The prompt to run on the fetched content"),
			}}},
		{"name": "TodoWrite", "description": "Create and manage a structured task list for the current session. Mark each task as in_progress before starting and completed when done. Only one task may be in_progress at a time.",
			"schema": map[string]any{"type": "object", "required": []any{"todos"}, "properties": map[string]any{
				"todos": map[string]any{"type": "array", "items": map[string]any{"type": "object", "required": []any{"content", "status"}, "properties": map[string]any{
					"content": map[string]any{"type": "string", "minLength": 1},
					"status":  map[string]any{"type": "string", "enum": []any{"pending", "in_progress", "completed"}},
				}}},
			}}},
		{"name": "Task", "description": "Launch a new agent to handle complex, multi-step tasks autonomously. Available agent types: general-purpose, Explore, Plan. When the agent is done it returns a single message. May run in the background; use run_in_background for long work.",
			"schema": map[string]any{"type": "object", "required": []any{"description", "prompt"}, "properties": map[string]any{
				"description":       str("A short (3-5 word) description of the task"),
				"prompt":            str("The task for the agent to perform"),
				"subagent_type":     str("The type of specialized agent to use for this task"),
				"run_in_background": map[string]any{"type": "boolean"},
			}}},
	}
}

func agentTools(api string) []any {
	var out []any
	for _, t := range agentToolSpecs() {
		if api == config.APIAnthropic {
			out = append(out, map[string]any{"name": t["name"], "description": t["description"], "input_schema": t["schema"]})
		} else {
			out = append(out, map[string]any{"type": "function", "function": map[string]any{"name": t["name"], "description": t["description"], "parameters": t["schema"]}})
		}
	}
	return out
}

func TestLargeAgentToolListPassesCleanly(t *testing.T) {
	for _, api := range []string{config.APIOpenAI, config.APIAnthropic} {
		t.Run(api, func(t *testing.T) {
			tr := &fnTransport{reply: echoReply}
			gw := newTestServer(t, "", tr)
			body := chatBody("Fix the failing test in pkg/server", false)
			if api == config.APIAnthropic {
				body = anthBody("Fix the failing test in pkg/server", false)
			}
			body["tools"] = agentTools(api)
			for range 2 {
				resp, out := postJSON(t, gw.URL+pathFor(api), body, nil)
				if resp.StatusCode != 200 {
					t.Fatalf("status %d %s", resp.StatusCode, out)
				}
			}
			calls := tr.all()
			if len(calls) != 2 || calls[0].body != calls[1].body {
				t.Fatal("repeated request must send identical bytes")
			}
			var sent map[string]any
			json.Unmarshal([]byte(calls[0].body), &sent)
			if blankMasked(t, api, map[string]any{"tools": sent["tools"]}) != blankMasked(t, api, map[string]any{"tools": clone(t, body["tools"])}) {
				t.Fatal("tool structure changed")
			}
			want := maskedTexts(t, api, map[string]any{"tools": body["tools"]})
			got := maskedTexts(t, api, map[string]any{"tools": sent["tools"]})
			for i := range want {
				if want[i] != got[i] {
					t.Logf("changed: %q -> %q", want[i], got[i])
				}
			}
			_, out := postJSON(t, gw.URL+pathFor(api), body, map[string]string{HeaderDryRun: "1"})
			for _, r := range decodeReport(t, out).Replacements {
				if strings.HasPrefix(r.Field, "tools") {
					t.Logf("replacement %s %s %s at %s", r.Type, r.Source, r.Action, r.Field)
				}
			}
		})
	}
}

func TestSecretInsideURLKeyIsReported(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	body := chatBody("hello", false)
	body["metadata"] = map[string]any{"https://x.test/?token=sk" + "_live_4eC39HqLyjWDarjtT1zdp7dc": "a", "mail maria.garcia@example.com": "b"}
	resp, out := postJSON(t, gw.URL+"/v1/chat/completions", body, nil)
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
	if resp.StatusCode != http.StatusUnprocessableEntity || !types["secret"] || !types["email"] || types["person"] || strings.Contains(string(out), "sk_live") {
		t.Fatalf("findings: %d %s", resp.StatusCode, out)
	}
}
