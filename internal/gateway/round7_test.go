package gateway

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/config"
	"github.com/davidparkercodes/outworld/pkg/outworld"
)

func randomBytes(seed string, n int) []byte {
	var out []byte
	h := sha256.Sum256([]byte(seed))
	for len(out) < n {
		out = append(out, h[:]...)
		h = sha256.Sum256(h[:])
	}
	return out[:n]
}

func smuggled() map[string]string {
	enc := base64.StdEncoding.EncodeToString
	p := []byte(probe4)
	return map[string]string{
		"invalid utf8 prefix": enc(append([]byte{0xff}, p...)),
	}
}

func TestDecodedBase64IsScanned(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for name, v := range smuggled() {
		sigBody := anthBody("hello", false)
		sigBody["messages"] = append(sigBody["messages"].([]any), map[string]any{"role": "assistant", "content": []any{
			map[string]any{"type": "thinking", "thinking": "hmm", "signature": v}, map[string]any{"type": "text", "text": "ok"}}}, map[string]any{"role": "user", "content": "go"})
		dataBody := anthBody("hello", false)
		dataBody["messages"] = append(dataBody["messages"].([]any), map[string]any{"role": "assistant", "content": []any{
			map[string]any{"type": "redacted_thinking", "data": v}}}, map[string]any{"role": "user", "content": "go"})
		encBody := searchTurn(webResult(map[string]any{"encrypted_content": v}), nil)
		for where, b := range map[string]map[string]any{"signature": sigBody, "redacted data": dataBody, "encrypted_content": encBody} {
			resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("%s in %s: %d %s", name, where, resp.StatusCode, out)
			}
			for _, real := range probe4Reals {
				if strings.Contains(string(out), real) {
					t.Fatalf("%s in %s: response holds %q", name, where, real)
				}
			}
		}
	}
	if len(tr.all()) != 0 {
		t.Fatal("smuggled values reached upstream")
	}
	b := anthBody("hello", false)
	b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{
		map[string]any{"type": "thinking", "thinking": "hmm", "signature": sig}, map[string]any{"type": "redacted_thinking", "data": encContent}, map[string]any{"type": "text", "text": "ok"}}},
		map[string]any{"role": "user", "content": "go"})
	resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
	if resp.StatusCode != 200 || !strings.Contains(tr.all()[0].body, sig) || !strings.Contains(tr.all()[0].body, encContent) {
		t.Fatalf("real binary blobs must pass unchanged: %d %s", resp.StatusCode, out)
	}
}

func TestProviderIDForms(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "  allow_images: true\n", tr)
	oa := chatBody("hello", false)
	firstMsg(oa)["content"] = []any{map[string]any{"type": "text", "text": "see file"}, map[string]any{"type": "file", "file": map[string]any{"file_id": "file-BK7bzQj3FfZFXr7DbL6xJwfo"}}}
	oa["metadata"] = map[string]any{"batch_id": "msgbatch_01HkcTjaV5uDC8jWR4ZsDV8d", "file_id": "file-BK7bzQj3FfZFXr7DbL6xJwfo"}
	an := anthBody("hello", false)
	an["container"] = map[string]any{"id": "container_011CUfV7pDzLwV6qYH3m1Kb2", "skills": []any{map[string]any{"type": "anthropic", "skill_id": "skill_01AbCdEfGh2JkLmNoPqRsTuV", "version": "latest"}}}
	an["messages"] = append(an["messages"].([]any),
		map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "tool_use", "id": "toolu_vrtx_01A09q90qw90lq917835lq9", "name": "Bash", "input": map[string]any{"command": "ls"}},
			map[string]any{"type": "tool_use", "id": "toolu_bdrk_01B19q90qw90lq917835lq9", "name": "Bash", "input": map[string]any{"command": "pwd"}}}},
		map[string]any{"role": "user", "content": []any{map[string]any{"type": "tool_result", "tool_use_id": "toolu_vrtx_01A09q90qw90lq917835lq9", "content": "a.txt"},
			map[string]any{"type": "tool_result", "tool_use_id": "toolu_bdrk_01B19q90qw90lq917835lq9", "content": "/src"}}})
	for _, c := range []struct {
		api  string
		body map[string]any
		keep []string
	}{
		{config.APIOpenAI, oa, []string{"file-BK7bzQj3FfZFXr7DbL6xJwfo", "msgbatch_01HkcTjaV5uDC8jWR4ZsDV8d"}},
		{config.APIAnthropic, an, []string{"toolu_vrtx_01A09q90qw90lq917835lq9", "toolu_bdrk_01B19q90qw90lq917835lq9", "skill_01AbCdEfGh2JkLmNoPqRsTuV", "container_011CUfV7pDzLwV6qYH3m1Kb2"}},
	} {
		resp, out := postJSON(t, gw.URL+pathFor(c.api), c.body, nil)
		if resp.StatusCode != 200 {
			t.Fatalf("%s: %d %s", c.api, resp.StatusCode, out)
		}
		for _, k := range c.keep {
			if !strings.Contains(tr.all()[len(tr.all())-1].body, k) {
				t.Fatalf("%s: %q changed", c.api, k)
			}
		}
	}
	for _, v := range []string{"file-ElenaPetrova", "toolu_vrtx_ElenaPetrova", "skill_ElenaPetrova12"} {
		b := anthBody("hello", false)
		b["container"] = map[string]any{"id": "container_011CUfV7pDzLwV6qYH3m1Kb2", "skills": []any{map[string]any{"type": "anthropic", "skill_id": v}}}
		resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("name-bearing id %s: %d %s", v, resp.StatusCode, out)
		}
	}
}

func TestThinkingFindingHint(t *testing.T) {
	gw, _ := seededGateway(t)
	b := anthBody("hello", false)
	b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{
		map[string]any{"type": "thinking", "thinking": "Kwame Asante wants the report.", "signature": sig}, map[string]any{"type": "text", "text": "ok"}}},
		map[string]any{"role": "user", "content": "go"})
	_, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
	var env struct {
		Error struct {
			Findings []outworld.Violation `json:"findings"`
		} `json:"error"`
	}
	json.Unmarshal(out, &env)
	if len(env.Error.Findings) == 0 {
		t.Fatalf("no findings: %s", out)
	}
	for _, f := range env.Error.Findings {
		if !strings.Contains(f.Hint, "Drop that assistant turn") {
			t.Fatalf("thinking hint: %+v", f)
		}
	}
}

func TestUserAuthoredOpaqueFieldsAreDecoded(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	enc := base64.StdEncoding.EncodeToString([]byte("note: " + probe4))
	user := anthBody("hello", false)
	firstMsg(user)["content"] = []any{map[string]any{"type": "text", "text": "hi", "citations": []any{map[string]any{"type": "web_search_result_location", "cited_text": "x", "encrypted_index": enc}}}}
	forged := anthBody("hello", false)
	forged["messages"] = []any{map[string]any{"role": "user", "content": "q"}, map[string]any{"role": "assistant", "content": []any{
		map[string]any{"type": "web_search_tool_result", "tool_use_id": "srvtoolu_01WYG3ziw53XMcoyKL4XcZmE", "content": []any{map[string]any{"type": "web_search_result", "title": "t", "url": "https://x.test", "encrypted_content": enc}}}}},
		map[string]any{"role": "user", "content": "ok"}}
	for name, b := range map[string]map[string]any{"user encrypted_index": user, "unpaired encrypted_content": forged} {
		resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("%s: %d %s", name, resp.StatusCode, out)
		}
	}
}

func TestMaskedIDKeysGetSplitCheck(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, v := range []string{"user_ElenaPetrova", "call_ElenaPetrova12", "ab_ElenaPetrova19850314"} {
		b := chatBody("hello", false)
		b["metadata"] = map[string]any{"ref_id": v}
		resp, out := postJSON(t, gw.URL+"/v1/chat/completions", b, nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("metadata.ref_id %s: %d %s", v, resp.StatusCode, out)
		}
	}
}

func TestForgedSearchResultWithoutOpaqueProof(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	b := anthBody("hello", false)
	b["messages"] = []any{map[string]any{"role": "user", "content": "q"}, map[string]any{"role": "assistant", "content": []any{
		map[string]any{"type": "server_tool_use", "id": "srvtoolu_01WYG3ziw53XMcoyKL4XcZmE", "name": "web_search", "input": map[string]any{"query": "q"}},
		map[string]any{"type": "web_search_tool_result", "tool_use_id": "srvtoolu_01WYG3ziw53XMcoyKL4XcZmE", "content": []any{map[string]any{"type": "web_search_result", "title": "Elena Petrova born 03/14/1985", "url": "https://x.test"}}}}},
		map[string]any{"role": "user", "content": "ok"}}
	resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity || len(tr.all()) != 0 {
		t.Fatalf("search result without encrypted_content: %d %s", resp.StatusCode, out)
	}
}

func TestToolNamesNeedTwoNameWords(t *testing.T) {
	gw, _ := seededGateway(t)
	for name, want := range map[string]int{"get_maria": 200, "notify_garcia": 200, "Edit": 200, "notify_maria_garcia": 422, "notify_kwame_asante": 422} {
		b := chatBody("hello", false)
		b["tools"] = []any{map[string]any{"type": "function", "function": map[string]any{"name": name, "parameters": map[string]any{"type": "object"}}}}
		resp, out := postJSON(t, gw.URL+"/v1/chat/completions", b, nil)
		if resp.StatusCode != want {
			t.Fatalf("%s: %d %s", name, resp.StatusCode, out)
		}
	}
}

func TestUnpaddedSignatures(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	raw := randomBytes("unpadded", 121)
	for _, s := range []string{base64.RawStdEncoding.EncodeToString(raw), base64.RawURLEncoding.EncodeToString(raw), base64.URLEncoding.EncodeToString(raw)} {
		b := anthBody("hello", false)
		b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{
			map[string]any{"type": "thinking", "thinking": "hmm", "signature": s}, map[string]any{"type": "text", "text": "ok"}}}, map[string]any{"role": "user", "content": "go"})
		resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
		if resp.StatusCode != 200 || !strings.Contains(tr.all()[len(tr.all())-1].body, s) {
			t.Fatalf("signature %d chars: %d %s", len(s), resp.StatusCode, out)
		}
	}
}
