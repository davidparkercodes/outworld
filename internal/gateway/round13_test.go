package gateway

import (
	"encoding/base64"
	"encoding/binary"
	"math/rand"
	"net/http"
	"strings"
	"testing"
)

func ninePositions(v string) map[string]map[string]any {
	out := allOpaquePositions(v)
	fb := anthBody("hello", false)
	fb["extra"] = map[string]any{"signature": v}
	out["fallback signature"] = fb
	nested := anthBody("hello", false)
	nested["extra"] = map[string]any{"nested": map[string]any{"encrypted_content": v}}
	out["nested fallback"] = nested
	user := anthBody("hello", false)
	firstMsg(user)["content"] = []any{map[string]any{"type": "text", "text": "hi", "citations": []any{map[string]any{"type": "web_search_result_location", "cited_text": "x", "encrypted_index": v}}}}
	out["user encrypted_index"] = user
	userRedacted := anthBody("hello", false)
	firstMsg(userRedacted)["content"] = []any{map[string]any{"type": "redacted_thinking", "data": v}, map[string]any{"type": "text", "text": "hi"}}
	out["user redacted"] = userRedacted
	tool := anthBody("hello", false)
	tool["messages"] = append(tool["messages"].([]any), map[string]any{"role": "assistant", "content": []any{
		map[string]any{"type": "tool_use", "id": "toolu_01A09q90qw90lq917835lq9", "name": "t", "input": map[string]any{"state": map[string]any{"signature": v}}}}})
	out["tool input"] = tool
	return out
}

func expect422Everywhere(t *testing.T, gw string, name string, v string) {
	t.Helper()
	for where, b := range ninePositions(v) {
		resp, out := postJSON(t, gw+"/v1/messages", b, nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("%s in %s: %d %s", name, where, resp.StatusCode, out)
		}
	}
}

func TestAccidentallyFramedTextKeepsWholeCheck(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	base := "By doctor data shipping January? Other we file 7293 what error your are may will please config? Key result and i "
	ssn := []byte(base + "SSN 123-45-6789")
	if !parseFrame(ssn).framed {
		t.Fatal("repro no longer parses as a framed message")
	}
	for _, s := range []string{base + "SSN 123-45-6789", base + "phone 804-555-1234", base + "card 4111 1111 1111 1111"} {
		expect422Everywhere(t, gw.URL, s, base64.StdEncoding.EncodeToString([]byte(s)))
	}
}

func binaryOuter(fields ...[]byte) []byte {
	r := rand.New(rand.NewSource(13))
	out := field(9, uniformBytes(r, 200))
	for _, f := range fields {
		out = append(out, f...)
	}
	return append(out, field(10, uniformBytes(r, 200))...)
}

func TestCardsAndPhonesAreNotInertTokens(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, tok := range []string{"4111111111111111", "804.555.1234", "804-555-1234"} {
		v := base64.StdEncoding.EncodeToString(binaryOuter(field(2, []byte(tok))))
		resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(v), nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("token %s: %d %s", tok, resp.StatusCode, out)
		}
	}
	for _, tok := range []string{"1712345678", "171234567", "2026-03-14", "1.2.3"} {
		v := base64.StdEncoding.EncodeToString(binaryOuter(field(2, []byte(tok))))
		resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(v), nil)
		if resp.StatusCode != 200 {
			t.Fatalf("inert token %s: %d %s", tok, resp.StatusCode, out)
		}
	}
}

func TestFrameLimitsFailClosed(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	p := []byte(paragraph)
	nest := func(depth int, inner []byte) []byte {
		for i := 0; i < depth; i++ {
			inner = field(1, inner)
		}
		return inner
	}
	varints := func(n int) []byte {
		var b []byte
		for i := 0; i < n; i++ {
			b = binary.AppendUvarint(append(b, 0x08), uint64(i%100))
		}
		return b
	}
	r := rand.New(rand.NewSource(26))
	binaryLeaf := append(field(1, uniformBytes(r, 120)), field(2, p)...)
	cases := map[string][]byte{
		"depth 26":            nest(26, p),
		"depth 6 framed leaf": nest(6, binaryLeaf),
		"P in field 4097":     append(varints(4096), field(5, p)...),
		"4096 fields then P":  append(varints(4096), field(1, p)...),
		"P then 4096 varints": append(field(1, p), varints(4096)...),
	}
	for name, b := range cases {
		v := base64.StdEncoding.EncodeToString(b)
		resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(v), nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("%s: %d %s", name, resp.StatusCode, out)
		}
	}
}

func TestFormatCharactersDoNotHideText(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, sep := range []string{"\u200b", "\u2060", "\u00ad", "\ufeff"} {
		text := strings.ReplaceAll(paragraph, " ", sep+" "+sep)
		if !textual([]byte(text)) {
			t.Fatalf("format character %U makes text binary", []rune(sep)[0])
		}
		expect422Everywhere(t, gw.URL, "format "+sep, base64.StdEncoding.EncodeToString([]byte(text)))
	}
}

func TestHandlesInDecodedProse(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, s := range []string{"Please contact @elena.petrova for access to the shared drive.", "Questions go to @elena.petrova."} {
		resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(base64.StdEncoding.EncodeToString([]byte(s))), nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("%q: %d %s", s, resp.StatusCode, out)
		}
	}
}

func TestShortPayloadsAreNotProse(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	v := base64.StdEncoding.EncodeToString(binaryOuter(field(2, []byte("ed")), field(3, []byte("ViLA")), field(4, []byte("Ms")), field(5, []byte("Villa rd 12"))))
	resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(v), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("short payloads: %d %s", resp.StatusCode, out)
	}
	long := base64.StdEncoding.EncodeToString(binaryOuter(field(2, []byte("Please send the report to Maria Garcia today."))))
	resp, out = postJSON(t, gw.URL+"/v1/messages", redactedTurn(long), nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("prose payload: %d %s", resp.StatusCode, out)
	}
}
