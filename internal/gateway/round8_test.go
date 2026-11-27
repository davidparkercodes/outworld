package gateway

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"math/rand"
	"net/http"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/config"
)

func TestRandomBinaryBlobsHaveNoFindings(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	rng := rand.New(rand.NewSource(7))
	const blobs, perRequest, size = 3000 / corpusScale, min(50, 3000/corpusScale), 16 << 10
	for done := 0; done < blobs; done += perRequest {
		var content []any
		for i := 0; i < perRequest; i++ {
			b := make([]byte, size)
			rng.Read(b)
			content = append(content, map[string]any{"type": "redacted_thinking", "data": base64.StdEncoding.EncodeToString(b)})
		}
		req := map[string]any{"model": "claude-x", "max_tokens": 16, "messages": []any{
			map[string]any{"role": "user", "content": "q"}, map[string]any{"role": "assistant", "content": content}, map[string]any{"role": "user", "content": "go"}}}
		sess := srv.engine.NewSession("")
		narrow, err := srv.maskTwice(context.Background(), sess, req, config.APIAnthropic)
		if err != nil {
			t.Fatal(err)
		}
		if lerr := leakOf(sess, narrow); lerr != nil {
			t.Fatalf("random blobs %d..%d: %+v", done, done+perRequest, lerr.Violations)
		}
	}
}

func binaryWith(text string) string {
	rng := rand.New(rand.NewSource(11))
	pre, post := make([]byte, 200), make([]byte, 200)
	rng.Read(pre)
	rng.Read(post)
	pre[len(pre)-1], post[0] = 'r', '#'
	return base64.StdEncoding.EncodeToString(append(append(pre, []byte(text)...), post...))
}

func redactedTurn(data string) map[string]any {
	b := anthBody("hello", false)
	b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "redacted_thinking", "data": data}}},
		map[string]any{"role": "user", "content": "go"})
	return b
}

func TestBinaryRunsBlockTermsAndCards(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, blockRules, tr)
	resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(binaryWith("topsecret")), nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("block term glued inside binary: %d %s", resp.StatusCode, out)
	}
	if len(tr.all()) != 0 {
		t.Fatal("refused requests reached upstream")
	}
}

func pairedTurn(useName, resultType string, content any) map[string]any {
	b := anthBody("hello", false)
	b["messages"] = []any{map[string]any{"role": "user", "content": "q"}, map[string]any{"role": "assistant", "content": []any{
		map[string]any{"type": "server_tool_use", "id": "srvtoolu_01WYG3ziw53XMcoyKL4XcZmE", "name": useName, "input": map[string]any{"q": "x"}},
		map[string]any{"type": resultType, "tool_use_id": "srvtoolu_01WYG3ziw53XMcoyKL4XcZmE", "content": content}}},
		map[string]any{"role": "user", "content": "ok"}}
	return b
}

func TestProviderResultTypeMustMatchCall(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	fetch := map[string]any{"type": "web_fetch_result", "url": "https://x.test", "content": map[string]any{"type": "document", "source": map[string]any{"type": "text", "media_type": "text/plain", "data": "Elena Petrova wrote this."}}}
	resp, out := postJSON(t, gw.URL+"/v1/messages", pairedTurn("web_search", "web_fetch_tool_result", fetch), nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("web_fetch result paired with web_search: %d %s", resp.StatusCode, out)
	}
	resp, out = postJSON(t, gw.URL+"/v1/messages", pairedTurn("web_fetch", "web_fetch_tool_result", fetch), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("matching pair: %d %s", resp.StatusCode, out)
	}
	bash := map[string]any{"type": "bash_code_execution_result", "stdout": "ok", "stderr": "", "return_code": 0,
		"content": []any{map[string]any{"type": "bash_code_execution_output", "file_id": "file_ElenaPetrova"}}}
	resp, out = postJSON(t, gw.URL+"/v1/messages", pairedTurn("bash_code_execution", "bash_code_execution_tool_result", bash), nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("name-bearing file_id in a paired result: %d %s", resp.StatusCode, out)
	}
}

func TestNestedEncodingIsDecoded(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	inner := base64.StdEncoding.EncodeToString([]byte("note " + probe4))
	hexed := hex.EncodeToString([]byte("note " + probe4))
	for name, v := range map[string]string{
		"base64 of base64": base64.StdEncoding.EncodeToString([]byte(inner)),
		"base64 of hex":    base64.StdEncoding.EncodeToString([]byte(hexed)),
	} {
		for where, b := range map[string]map[string]any{
			"redacted data": redactedTurn(v),
			"unknown signature": func() map[string]any {
				b := anthBody("hello", false)
				b["extra"] = map[string]any{"signature": v}
				return b
			}(),
		} {
			resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("%s in %s: %d %s", name, where, resp.StatusCode, out)
			}
		}
	}
}

func TestShortLooseBase64IsDecoded(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	raw := []byte("x " + probe4)
	for _, v := range []string{base64.RawURLEncoding.EncodeToString(raw), base64.RawStdEncoding.EncodeToString(raw)} {
		sigBody := anthBody("hello", false)
		sigBody["messages"] = append(sigBody["messages"].([]any), map[string]any{"role": "assistant", "content": []any{
			map[string]any{"type": "thinking", "thinking": "hmm", "signature": v}, map[string]any{"type": "text", "text": "ok"}}}, map[string]any{"role": "user", "content": "go"})
		for where, b := range map[string]map[string]any{"signature": sigBody, "redacted data": redactedTurn(v)} {
			resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("%d-char loose base64 in %s: %d %s", len(v), where, resp.StatusCode, out)
			}
		}
	}
	resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(shortEnc), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("real short redacted data: %d %s", resp.StatusCode, out)
	}
}

func TestEncodedKeysAnyCaseAndUserRedacted(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	enc := base64.StdEncoding.EncodeToString([]byte("note " + probe4))
	user := anthBody("hello", false)
	firstMsg(user)["content"] = []any{map[string]any{"type": "redacted_thinking", "data": enc}, map[string]any{"type": "text", "text": "hi"}}
	capital := anthBody("hello", false)
	capital["extra"] = map[string]any{"Signature": enc}
	oaCap := chatBody("hello", false)
	oaCap["extra"] = map[string]any{"Encrypted_Content": enc}
	for name, c := range map[string]struct {
		path string
		body map[string]any
	}{"user redacted_thinking": {"/v1/messages", user}, "Signature": {"/v1/messages", capital}, "Encrypted_Content": {"/v1/chat/completions", oaCap}} {
		resp, out := postJSON(t, gw.URL+c.path, c.body, nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("%s: %d %s", name, resp.StatusCode, out)
		}
	}
}

func TestCamelNameIDTails(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "  allow_images: true\n", tr)
	for _, id := range []string{"file-ElenaPetrova19850314", "file_ElenaPetrova19850314", "toolu_vrtx_ElenaPetrova19850314", "toolu_bdrk_ElenaPetrova19850314", "msgbatch_ElenaPetrova19850314"} {
		if idShaped(id) {
			t.Fatalf("name-bearing id accepted: %s", id)
		}
		oa := chatBody("hello", false)
		firstMsg(oa)["content"] = []any{map[string]any{"type": "text", "text": "see"}, map[string]any{"type": "file", "file": map[string]any{"file_id": id}}}
		md := chatBody("hello", false)
		md["metadata"] = map[string]any{"file_id": id}
		tu := anthBody("hello", false)
		tu["messages"] = append(tu["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "tool_use", "id": id, "name": "Bash", "input": map[string]any{}}}})
		doc := anthBody("hello", false)
		firstMsg(doc)["content"] = []any{map[string]any{"type": "document", "source": map[string]any{"type": "file", "file_id": id}}}
		for where, c := range map[string]struct {
			path string
			body map[string]any
		}{"file part": {"/v1/chat/completions", oa}, "metadata": {"/v1/chat/completions", md}, "tool_use id": {"/v1/messages", tu}, "document source": {"/v1/messages", doc}} {
			resp, out := postJSON(t, gw.URL+c.path, c.body, nil)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("%s in %s: %d %s", id, where, resp.StatusCode, out)
			}
		}
	}
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	rng := rand.New(rand.NewSource(3))
	for i := 0; i < 5000; i++ {
		tail := make([]byte, 24)
		for j := range tail {
			tail[j] = alphabet[rng.Intn(len(alphabet))]
		}
		if digitCount(string(tail)) < 2 {
			continue
		}
		for _, prefix := range []string{"file-", "toolu_", "toolu_vrtx_", "msgbatch_", "call_"} {
			if !idShaped(prefix + string(tail)) {
				t.Fatalf("random id rejected: %s%s", prefix, tail)
			}
		}
	}
}

func TestUnpaddedEncryptedContentIsProof(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	raw := base64.RawStdEncoding.EncodeToString(randomBytes("enc", 301))
	b := searchTurn(webResult(map[string]any{"title": "Ada Lovelace biography", "encrypted_content": raw}), nil)
	resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
	if resp.StatusCode != 200 || !strings.Contains(tr.all()[0].body, raw) {
		t.Fatalf("unpadded encrypted_content: %d %s", resp.StatusCode, out)
	}
}
