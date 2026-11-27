package gateway

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/davidparkercodes/outworld/internal/config"
)

func TestClaimedValuesReachBlockRules(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "  allow_images: true\n  rules:\n    - {name: secret, match_regex: [\"(?i)topsecret\"], route: block}\n", tr)
	thinking := anthBody("hello", false)
	thinking["messages"] = append(thinking["messages"].([]any), map[string]any{"role": "assistant", "content": []any{
		map[string]any{"type": "thinking", "thinking": "the topsecret plan", "signature": sig}, map[string]any{"type": "text", "text": "ok"}}},
		map[string]any{"role": "user", "content": "go"})
	enc := searchTurn(webResult(map[string]any{"encrypted_content": "topsecretabc" + encContent}), nil)
	token := anthBody("hello", false)
	token["mcp_servers"] = []any{map[string]any{"type": "url", "url": "https://mcp.test/sse", "name": "m", "authorization_token": "topsecret"}}
	sigArg := anthBody("hello", false)
	sigArg["messages"] = append(sigArg["messages"].([]any), map[string]any{"role": "assistant", "content": []any{
		map[string]any{"type": "redacted_thinking", "data": "topsecretabc" + encContent}}}, map[string]any{"role": "user", "content": "go"})
	for name, b := range map[string]map[string]any{"thinking": thinking, "encrypted_content": enc, "authorization_token": token, "redacted_thinking": sigArg} {
		resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("%s: %d %s", name, resp.StatusCode, out)
		}
	}
	if len(tr.all()) != 0 {
		t.Fatal("blocked requests reached upstream")
	}
}

func TestThinkingTextGetsVaultCheck(t *testing.T) {
	gw, tr := seededGateway(t)
	b := anthBody("hello", false)
	b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{
		map[string]any{"type": "thinking", "thinking": "Kwame Asante wants the report.", "signature": sig}, map[string]any{"type": "text", "text": "ok"}}},
		map[string]any{"role": "user", "content": "go"})
	resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity || len(tr.all()) != 0 {
		t.Fatalf("vault-known real in thinking: %d %s", resp.StatusCode, out)
	}
	b["messages"].([]any)[1].(map[string]any)["content"].([]any)[0].(map[string]any)["thinking"] = "Tobiah Wrenfield wants the report."
	resp, out = postJSON(t, gw.URL+"/v1/messages", b, nil)
	if resp.StatusCode != 200 || !strings.Contains(tr.all()[0].body, "Tobiah Wrenfield wants the report.") {
		t.Fatalf("stand-ins in thinking must pass unchanged: %d %s", resp.StatusCode, out)
	}
}

func TestSingleTokenVaultRuleIsOnlyForNames(t *testing.T) {
	gw, tr := seededGateway(t)
	for _, b := range []map[string]any{
		func() map[string]any {
			b := anthBody("hello", false)
			b["extra"] = map[string]any{"user_id": "Asante"}
			return b
		}(),
		func() map[string]any {
			b := anthBody("hello", false)
			b["extra"] = map[string]any{"customer_id": "Asante"}
			return b
		}(),
		searchTurn(webResult(map[string]any{"title": "Asante"}), nil),
	} {
		resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("single-token vault record in a value: %d %s", resp.StatusCode, out)
		}
	}
	if len(tr.all()) != 0 {
		t.Fatal("blocked requests reached upstream")
	}
}

func TestOpaqueBase64Rule(t *testing.T) {
	smuggle := "MariaGarcia8045551234SSN123456789xx"
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	cases := map[string]map[string]any{}
	b := anthBody("hello", false)
	b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "redacted_thinking", "data": smuggle}}}, map[string]any{"role": "user", "content": "go"})
	cases["redacted_thinking"] = b
	cases["encrypted_content"] = searchTurn(webResult(map[string]any{"encrypted_content": smuggle}), nil)
	b = anthBody("hello", false)
	firstMsg(b)["content"] = []any{map[string]any{"type": "text", "text": "hi", "citations": []any{map[string]any{"type": "web_search_result_location", "cited_text": "x", "encrypted_index": shortEnc}}}}
	userIndex := b
	for name, c := range cases {
		resp, out := postJSON(t, gw.URL+"/v1/messages", c, nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("%s: %d %s", name, resp.StatusCode, out)
		}
	}
	if resp, out := postJSON(t, gw.URL+"/v1/messages", userIndex, nil); resp.StatusCode != 200 {
		t.Fatalf("binary encrypted_index in a user message must classify as binary: %d %s", resp.StatusCode, out)
	}
	resp, out := postJSON(t, gw.URL+"/v1/messages", searchTurn(webResult(map[string]any{"encrypted_content": shortEnc}), nil), nil)
	if resp.StatusCode != 200 || !strings.Contains(tr.all()[0].body, shortEnc) {
		t.Fatalf("short opaque encrypted_content: %d %s", resp.StatusCode, out)
	}
	if _, ok := stdBase64("EqoBCkgIARABGAIiQFk2sP9xK3a1c2d3e4f5g6h7i8j9k0lAbCdEfGhIjKlMnOpQrStUvWxYz"); ok {
		t.Fatal("length mod 4 = 1 is not base64")
	}
}

func TestIDShapes(t *testing.T) {
	good := []string{"toolu_01A09q90qw90lq917835lq9", "srvtoolu_01WYG3ziw53XMcoyKL4XcZmE", "call_Xk3n9QzL2pR7tV1wY5bA8cD0", "msg_01XFDUDYJgAACzvnptvVoYEL",
		"container_011CUfV7pDzLwV6qYH3m1Kb2", "chatcmpl-B9MBs8CjcvOU2jLn4n570S5qMJKcT", "resp_67ccd2bed1ec8190b14f964abc0542670bb6a6b452d3795b",
		"fc_67ca09c5efe0819096d0511c92b8c890096610f474011cc0", "file_011CNha8iCJcU1wXNR6q4V8w", "9a8b7c6d-1b2c-4d3e-8f00-112233445566",
		"8f14e45fceea167a5a36dedd4bea2543b2f1c9a0e1d2c3b4a5968778695a4b3c", "3e7c9b1d2a4f6e8c0b5d7a9f1e3c5b7d9a2c4e6f", "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"user_" + strings.Repeat("3f9a0c2b", 8) + "_account_7e1c2a90-1b2c-4d3e-8f00-112233445566_session_9a8b7c6d-1b2c-4d3e-8f00-112233445566"}
	for _, v := range good {
		if !idShaped(v) {
			t.Fatalf("real id rejected: %s", v)
		}
	}
	bad := []string{"user_elenapetrova", "user_ElenaPetrova", "call_ElenaPetrova12", "ab_ElenaPetrova19850314", "toolu-Elena-Petrova-1985-03-14",
		"user-Elena-Petrova-03-14-1985", "8045551234123456789", "456c656e6120506574726f76612c2062726f6f6b6c796e206e79",
		"Elena Petrova", "elena_petrova", "1985-03-14"}
	for _, v := range bad {
		if idShaped(v) {
			t.Fatalf("not an id: %s", v)
		}
	}
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, v := range []string{"toolu-Elena-Petrova-1985-03-14", "user-Elena-Petrova-03-14-1985", "user_ElenaPetrova", "call_ElenaPetrova12", "ab_ElenaPetrova19850314", "456c656e6120506574726f76612c2062726f6f6b6c796e206e79"} {
		b := chatBody("hello", false)
		b["safety_identifier"] = v
		resp, out := postJSON(t, gw.URL+"/v1/chat/completions", b, nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("%s: %d %s", v, resp.StatusCode, out)
		}
	}
}

func TestBlockedDryRunOmitsRequest(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, api := range []string{config.APIOpenAI, config.APIAnthropic} {
		b := chatBody("hello", false)
		if api == config.APIAnthropic {
			b = anthBody("hello", false)
		}
		b["safety_identifier"] = probe4
		resp, out := postJSON(t, gw.URL+pathFor(api), b, map[string]string{HeaderDryRun: "1"})
		rep := decodeReport(t, out)
		if resp.StatusCode != 200 || rep.Blocked != "outworld_leak_blocked" || rep.Request != nil {
			t.Fatalf("%s: %d %s", api, resp.StatusCode, out)
		}
		for _, real := range probe4Reals {
			if strings.Contains(string(out), real) {
				t.Fatalf("%s: dry run echoes %q: %s", api, real, out)
			}
		}
	}
}

func TestTextMediaRules(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "  allow_images: true\n", tr)
	u16 := utf16.Encode([]rune("notes for " + probe4))
	raw := []byte{0xff, 0xfe}
	for _, c := range u16 {
		raw = append(raw, byte(c), byte(c>>8))
	}
	for name, blk := range map[string]map[string]any{
		"utf16 document": {"type": "document", "source": map[string]any{"type": "base64", "media_type": "text/plain", "data": base64.StdEncoding.EncodeToString(raw)}},
		"text in image":  {"type": "image", "source": map[string]any{"type": "base64", "media_type": "text/plain", "data": base64.StdEncoding.EncodeToString([]byte("notes"))}},
	} {
		b := anthBody("hello", false)
		firstMsg(b)["content"] = []any{blk}
		resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
		if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(out), "outworld_unsupported_content") {
			t.Fatalf("%s: %d %s", name, resp.StatusCode, out)
		}
	}
}

func TestProviderResultMustFollowItsCall(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	use := map[string]any{"type": "server_tool_use", "id": "srvtoolu_01WYG3ziw53XMcoyKL4XcZmE", "name": "web_search", "input": map[string]any{"query": "q"}}
	res := map[string]any{"type": "web_search_tool_result", "tool_use_id": "srvtoolu_01WYG3ziw53XMcoyKL4XcZmE", "content": []any{webResult(map[string]any{"title": "Elena Petrova biography"})}}
	for name, msgs := range map[string][]any{
		"result before use":      {map[string]any{"role": "user", "content": "q"}, map[string]any{"role": "assistant", "content": []any{res, use}}, map[string]any{"role": "user", "content": "ok"}},
		"result in user message": {map[string]any{"role": "user", "content": "q"}, map[string]any{"role": "assistant", "content": []any{use}}, map[string]any{"role": "user", "content": []any{res}}},
	} {
		b := anthBody("hello", false)
		b["messages"] = msgs
		resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("%s: %d %s", name, resp.StatusCode, out)
		}
	}
	b := anthBody("hello", false)
	b["messages"] = []any{map[string]any{"role": "user", "content": "q"}, map[string]any{"role": "assistant", "content": []any{use, res}}, map[string]any{"role": "user", "content": "ok"}}
	resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("paired result: %d %s", resp.StatusCode, out)
	}
	var sent map[string]any
	json.Unmarshal([]byte(tr.all()[0].body), &sent)
	if !strings.Contains(tr.all()[0].body, "Elena Petrova biography") {
		t.Fatalf("paired public name must pass unchanged: %s", tr.all()[0].body)
	}
}

func TestBillingLineOnlyFirstAndUnique(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	good := "x-anthropic-billing-header: cc_version=2.1.3.abc; cc_entrypoint=cli; cch=0f3a9;"
	if billingLine("x-anthropic-billing-header: cch=0f3a9; cch=0f3a9;") {
		t.Fatal("repeated key accepted")
	}
	b := anthBody("hello", false)
	b["system"] = []any{map[string]any{"type": "text", "text": "be brief"}, map[string]any{"type": "text", "text": good}}
	resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("%d %s", resp.StatusCode, out)
	}
	b = anthBody("hello", false)
	b["system"] = []any{map[string]any{"type": "text", "text": good}}
	postJSON(t, gw.URL+"/v1/messages", b, nil)
	if !strings.Contains(tr.all()[1].body, good) {
		t.Fatal("billing line in system[0] must pass unchanged")
	}
}
