package gateway

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/config"
)

func searchTurn(result map[string]any, cite map[string]any) map[string]any {
	b := anthBody("hello", false)
	text := map[string]any{"type": "text", "text": "Per the article, it works."}
	if cite != nil {
		text["citations"] = []any{cite}
	}
	b["messages"] = []any{
		map[string]any{"role": "user", "content": "search"},
		map[string]any{"role": "assistant", "content": []any{
			map[string]any{"type": "server_tool_use", "id": "srvtoolu_01WYG3ziw53XMcoyKL4XcZmE", "name": "web_search", "input": map[string]any{"query": "storage"}},
			map[string]any{"type": "web_search_tool_result", "tool_use_id": "srvtoolu_01WYG3ziw53XMcoyKL4XcZmE", "content": []any{result}},
			text}},
		map[string]any{"role": "user", "content": "thanks"},
	}
	return b
}

func webResult(extra map[string]any) map[string]any {
	r := map[string]any{"type": "web_search_result", "title": "Storage guide", "url": "https://x.test/guide", "encrypted_content": encContent}
	for k, v := range extra {
		r[k] = v
	}
	return r
}

func TestPageAgeInProviderResults(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, age := range []string{"April 3, 2025", "2025-04-03", "3 days ago", "2 weeks ago"} {
		resp, out := postJSON(t, gw.URL+"/v1/messages", searchTurn(webResult(map[string]any{"page_age": age}), nil), nil)
		if resp.StatusCode != 200 || !strings.Contains(tr.all()[len(tr.all())-1].body, `"page_age":"`+age+`"`) {
			t.Fatalf("page_age %q: %d %s", age, resp.StatusCode, out)
		}
	}
	resp, out := postJSON(t, gw.URL+"/v1/messages", searchTurn(webResult(map[string]any{"page_age": "call (804) 555-1234"}), nil), nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("free text page_age: %d %s", resp.StatusCode, out)
	}
}

func TestCheatSheetURLInResultsAndCitations(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	u := "https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html"
	cite := map[string]any{"type": "web_search_result_location", "url": u, "title": "Password Storage", "cited_text": "Use a slow hash.", "encrypted_index": encContent}
	resp, out := postJSON(t, gw.URL+"/v1/messages", searchTurn(webResult(map[string]any{"url": u}), cite), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("cheat sheet url: %d %s", resp.StatusCode, out)
	}
}

func seededGateway(t *testing.T) (*httptest.Server, *fnTransport) {
	t.Helper()
	tr := &fnTransport{reply: echoReply}
	srv := newTestGateway(t, "masked", "", tr)
	v := srv.engine.Vault()
	for _, r := range [][3]string{{"person.given", "Edit", "Zorvath"}, {"person.given", "Kwame", "Tobiah"}, {"person.family", "Asante", "Wrenfield"}} {
		sur := r[2]
		if _, err := v.Assign(r[0], r[1], func(int) string { return sur }); err != nil {
			t.Fatal(err)
		}
	}
	gw := httptest.NewServer(srv.Handler())
	t.Cleanup(gw.Close)
	return gw, tr
}

func TestSingleTokenVaultRecordsDoNotBlockToolNames(t *testing.T) {
	gw, tr := seededGateway(t)
	named := func(api, name string) map[string]any {
		b := chatBody("hello", false)
		b["tools"] = []any{map[string]any{"type": "function", "function": map[string]any{"name": name, "parameters": map[string]any{"type": "object"}}}}
		if api == config.APIAnthropic {
			b = anthBody("hello", false)
			b["tools"] = []any{map[string]any{"name": name, "input_schema": map[string]any{"type": "object"}}}
			b["tool_choice"] = map[string]any{"type": "tool", "name": name}
		}
		return b
	}
	for _, api := range []string{config.APIOpenAI, config.APIAnthropic} {
		resp, out := postJSON(t, gw.URL+pathFor(api), named(api, "Edit"), nil)
		if resp.StatusCode != 200 || !strings.Contains(tr.all()[len(tr.all())-1].body, `"name":"Edit"`) {
			t.Fatalf("%s Edit with a stray vault record: %d %s", api, resp.StatusCode, out)
		}
		for _, name := range []string{"notify_maria_garcia", "notify_kwame_asante", "notifyKwameAsante"} {
			resp, out := postJSON(t, gw.URL+pathFor(api), named(api, name), nil)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("%s %s: %d %s", api, name, resp.StatusCode, out)
			}
		}
	}
	b := anthBody("Edit the file for Kwame", false)
	resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
	if resp.StatusCode != 200 || strings.Contains(tr.all()[len(tr.all())-1].body, "Kwame") {
		t.Fatalf("message text still masks single-token records: %d %s", resp.StatusCode, out)
	}
}

func TestPlainTextBase64AtOpaquePositions(t *testing.T) {
	plain := base64.StdEncoding.EncodeToString([]byte("Note 7: " + probe4))
	runBypassCases(t, "", []bypassCase{
		anCase("thinking signature", func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "thinking", "thinking": "hmm", "signature": b64Of(v)}}})
		}),
		anCase("redacted_thinking data", func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "redacted_thinking", "data": b64Of(v)}}})
		}),
		anCase("encrypted_index", func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "text", "text": "ok",
				"citations": []any{map[string]any{"type": "web_search_result_location", "cited_text": "x", "encrypted_index": b64Of(v)}}}}})
		}),
		anCase("encrypted_content", func(b map[string]any, v string) {
			sb := searchTurn(webResult(map[string]any{"encrypted_content": b64Of(v)}), nil)
			b["messages"] = sb["messages"]
		}),
		anCase("tool_use input signature", func(b map[string]any, v string) {
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "tool_use", "id": "toolu_01", "name": "t", "input": map[string]any{"signature": b64Of(v)}}}})
		}),
		anCase("unknown field signature", func(b map[string]any, v string) { b["extra"] = map[string]any{"signature": b64Of(v)} }),
	})
	if b, ok := stdBase64(plain); !ok || !textual(b) {
		t.Fatal("plain text base64 not recognized")
	}
	if b, ok := stdBase64(sig); !ok || textual(b) {
		t.Fatal("an opaque signature read as plain text")
	}
}

func b64Of(v string) string {
	return base64.StdEncoding.EncodeToString([]byte("Note 7: " + v + " Q9"))
}

func TestFullClaudeCodeStyleRequest(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "  keep_metadata: true\n  allow_images: true\n", tr)
	u := "https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html"
	b := realisticAgentRequest(config.APIAnthropic)
	b["metadata"] = map[string]any{"user_id": "user_" + strings.Repeat("3f9a0c2b", 8) + "_account_7e1c2a90-1b2c-4d3e-8f00-112233445566_session_9a8b7c6d-1b2c-4d3e-8f00-112233445566"}
	b["container"] = "container_011CUfV7pDzLwV6qYH3m1Kb2"
	b["context_management"] = map[string]any{"edits": []any{map[string]any{"type": "clear_thinking_20251015"}, map[string]any{"type": "clear_tool_uses_20250919"}}}
	msgs := b["messages"].([]any)
	msgs = append(msgs,
		map[string]any{"role": "assistant", "content": []any{
			map[string]any{"type": "server_tool_use", "id": "srvtoolu_01WYG3ziw53XMcoyKL4XcZmE", "name": "web_search", "input": map[string]any{"query": "password storage"}},
			map[string]any{"type": "web_search_tool_result", "tool_use_id": "srvtoolu_01WYG3ziw53XMcoyKL4XcZmE", "content": []any{
				map[string]any{"type": "web_search_result", "title": "Password Storage Cheat Sheet", "url": u, "page_age": "April 3, 2025", "encrypted_content": encContent}}},
			map[string]any{"type": "text", "text": "Use a slow hash.", "citations": []any{map[string]any{"type": "web_search_result_location", "url": u,
				"title": "Password Storage Cheat Sheet", "cited_text": "Use a slow password hash.", "encrypted_index": encContent}}}}},
		map[string]any{"role": "user", "content": []any{
			map[string]any{"type": "image", "source": map[string]any{"type": "base64", "media_type": "image/png", "data": pngB64}},
			map[string]any{"type": "text", "text": "Does this screenshot match?", "cache_control": map[string]any{"type": "ephemeral"}}}})
	b["messages"] = msgs
	for range 2 {
		resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
		if resp.StatusCode != 200 {
			t.Fatalf("status %d %s", resp.StatusCode, out)
		}
	}
	calls := tr.all()
	if calls[0].body != calls[1].body {
		t.Fatal("repeated request bytes differ")
	}
	for _, keep := range []string{sig, attributionText, encContent, "April 3, 2025", "toolu_01A09q90qw90lq917835lq9", "srvtoolu_01WYG3ziw53XMcoyKL4XcZmE", pngB64, "container_011CUfV7pDzLwV6qYH3m1Kb2"} {
		if !strings.Contains(calls[0].body, keep) {
			t.Fatalf("%q changed: %s", keep, calls[0].body)
		}
	}
}
