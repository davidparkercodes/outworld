package gateway

import (
	"encoding/base64"
	"encoding/hex"
	"math/rand"
	"net/http"
	"strings"
	"testing"
)

const paragraph = "Patient record for Maria Garcia, born 03/14/1985, living at 123 Main Street, Richmond, VA 23220. Passport X12345678, SSN 123-45-6789, phone (804) 555-1234."

func allOpaquePositions(v string) map[string]map[string]any {
	sig := anthBody("hello", false)
	sig["messages"] = append(sig["messages"].([]any), map[string]any{"role": "assistant", "content": []any{
		map[string]any{"type": "thinking", "thinking": "hmm", "signature": v}, map[string]any{"type": "text", "text": "ok"}}}, map[string]any{"role": "user", "content": "go"})
	idx := anthBody("hello", false)
	idx["messages"] = append(idx["messages"].([]any), map[string]any{"role": "assistant", "content": []any{
		map[string]any{"type": "text", "text": "ok", "citations": []any{map[string]any{"type": "web_search_result_location", "url": "https://x.test", "cited_text": "x", "encrypted_index": v}}}}},
		map[string]any{"role": "user", "content": "go"})
	return map[string]map[string]any{
		"signature":         sig,
		"redacted data":     redactedTurn(v),
		"encrypted_content": searchTurn(webResult(map[string]any{"encrypted_content": v}), nil),
		"encrypted_index":   idx,
	}
}

func TestInvalidByteDoesNotDisableTextChecks(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	rng := rand.New(rand.NewSource(23))
	mixed := func(share float64) []byte {
		var out []byte
		for _, c := range []byte(paragraph + " " + paragraph) {
			if rng.Float64() < share {
				out = append(out, byte(rng.Intn(256)))
			}
			out = append(out, c)
		}
		return out
	}
	for name, raw := range map[string][]byte{
		"leading 0xff": append([]byte{0xff}, paragraph...),
		"10% binary":   mixed(0.1),
		"20% binary":   mixed(0.2),
	} {
		for where, b := range allOpaquePositions(base64.StdEncoding.EncodeToString(raw)) {
			resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("%s in %s: %d %s", name, where, resp.StatusCode, out)
			}
		}
	}
	if len(tr.all()) != 0 {
		t.Fatal("leaking requests reached upstream")
	}
}

func TestNamesInsideIDTails(t *testing.T) {
	for _, id := range []string{"toolu_01ElenaPetrovaXk3", "toolu_01ElenaPetrova7kQ2mZx9Lp3", "call_ElenaPetrovaXkQmZrWp", "toolu_01MariaGarciaXk3q",
		"toolu_01JohnSmith7kQ2mZx9Lp3", "toolu_01elenaPetrova9xQ2mZr7Lp", "toolu_01Elena9Petrova9xQ2mZr7", "toolu_01ElenaXPetrova9xQ2mZ"} {
		if idShaped(id) {
			t.Fatalf("name-bearing id accepted: %s", id)
		}
	}
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	b := anthBody("hello", false)
	b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "tool_use", "id": "toolu_01ElenaPetrovaXk3", "name": "Bash", "input": map[string]any{}}}})
	resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("tool_use id with a name: %d %s", resp.StatusCode, out)
	}
}

func TestHexOfTextInIDFields(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, v := range []string{hex.EncodeToString([]byte("Elena Petrova")), hex.EncodeToString([]byte("123-45-6789"))} {
		b := chatBody("hello", false)
		b["metadata"] = map[string]any{"user_id": v}
		resp, out := postJSON(t, gw.URL+"/v1/chat/completions", b, nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("hex of text %s: %d %s", v, resp.StatusCode, out)
		}
	}
	for _, v := range []string{"8f14e45fceea167a5a36dedd4bea2543b2f1c9a0e1d2c3b4a5968778695a4b3c", "3e7c9b1d2a4f6e8c", "a1b2c3d4e5f60718293a4b5c"} {
		b := chatBody("hello", false)
		b["metadata"] = map[string]any{"user_id": v}
		resp, out := postJSON(t, gw.URL+"/v1/chat/completions", b, nil)
		if resp.StatusCode != 200 {
			t.Fatalf("real hex id %s: %d %s", v, resp.StatusCode, out)
		}
	}
}

func TestCuedNumbersInDecodedText(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, phrase := range []string{"my SSN is 123456789", "call me on 8045551234", "phone 8045551234", "passport number 123456789", "account 12345678901"} {
		text := "Notes from the meeting today. The client said " + phrase + " and asked for a follow up next week."
		enc := base64.StdEncoding.EncodeToString([]byte(text))
		cases := allOpaquePositions(enc)
		fb := anthBody("hello", false)
		fb["extra"] = map[string]any{"signature": enc}
		cases["fallback signature"] = fb
		oa := chatBody("hello", false)
		oa["extra"] = map[string]any{"encrypted_content": enc}
		for where, b := range cases {
			resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("%q in %s: %d %s", phrase, where, resp.StatusCode, out)
			}
		}
		resp, out := postJSON(t, gw.URL+"/v1/chat/completions", oa, nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("%q in openai fallback: %d %s", phrase, resp.StatusCode, out)
		}
	}
	if len(tr.all()) != 0 {
		t.Fatal("leaking requests reached upstream")
	}
}

func TestPlainTextNumbersUnaffected(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	resp, out := postJSON(t, gw.URL+"/v1/chat/completions", chatBody("call me on 8045551234 tomorrow", false), nil)
	if resp.StatusCode != 200 || strings.Contains(tr.all()[0].body, "8045551234") {
		t.Fatalf("plain message phone must be masked: %d %s %s", resp.StatusCode, out, tr.all()[0].body)
	}
	b := anthBody("hello", false)
	firstMsg(b)["content"] = []any{map[string]any{"type": "tool_result", "tool_use_id": "toolu_01A09q90qw90lq917835lq9", "content": "call me on 8045551234"}}
	resp, out = postJSON(t, gw.URL+"/v1/messages", b, nil)
	if resp.StatusCode != 200 || strings.Contains(tr.all()[1].body, "8045551234") {
		t.Fatalf("tool result phone must be masked: %d %s", resp.StatusCode, out)
	}
}
