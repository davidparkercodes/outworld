package gateway

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"math/rand"
	"net/http"
	"strings"
	"testing"
)

func field(num int, payload []byte) []byte {
	b := binary.AppendUvarint(nil, uint64(num<<3|2))
	b = binary.AppendUvarint(b, uint64(len(payload)))
	return append(b, payload...)
}

func varintField(num int, v uint64) []byte {
	return binary.AppendUvarint(binary.AppendUvarint(nil, uint64(num<<3)), v)
}

func chunked(p []byte, size int) []byte {
	var out []byte
	for i := 0; i < len(p); i += size {
		out = append(out, field(1+i/size%15, p[i:min(len(p), i+size)])...)
	}
	return out
}

func framedVariants(p []byte) map[string][]byte {
	wrap := corpusEnglish(rand.New(rand.NewSource(41)), 4096)
	return map[string][]byte{
		"field 1":       field(1, p),
		"field 2":       field(2, p),
		"nested":        field(3, field(1, field(2, p))),
		"split 40":      chunked(p, 40),
		"split 100":     chunked(p, 100),
		"split 127":     chunked(p, 127),
		"extra varints": append(append(varintField(1, 1732345678), field(2, p)...), varintField(3, 7)...),
		"english wrap":  append(append(field(1, wrap[:2048]), field(2, p)...), field(3, wrap[2048:])...),
	}
}

func encodings(b []byte) map[string]string {
	return map[string]string{"base64": base64.StdEncoding.EncodeToString(b), "hex": hex.EncodeToString(b), "rawurl": base64.RawURLEncoding.EncodeToString(b)}
}

func TestFramedTextIsChecked(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	p := []byte(paragraph + " " + paragraph)
	for name, b := range framedVariants(p) {
		for encName, v := range encodings(b) {
			for where, body := range allOpaquePositions(v) {
				resp, out := postJSON(t, gw.URL+"/v1/messages", body, nil)
				if resp.StatusCode != http.StatusUnprocessableEntity {
					t.Fatalf("%s %s in %s: %d %s", name, encName, where, resp.StatusCode, out)
				}
			}
		}
	}
}

func TestShortTextIsNotFramed(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, text := range []string{"Maria Smith 123-45-6789", "Maria Garcia maria@example.com", "Maria Petrova 804-555-1234"} {
		for where, body := range allOpaquePositions(base64.StdEncoding.EncodeToString([]byte(text))) {
			resp, out := postJSON(t, gw.URL+"/v1/messages", body, nil)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("%q in %s: %d %s", text, where, resp.StatusCode, out)
			}
		}
	}
}

func TestNumericPIIListsAreText(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, text := range []string{
		"SSN 123-45-6789, 987-65-4321, 234-56-7890; phone (804) 555-1234; card 4111 1111 1111 1111",
		"name,ssn,phone\nMaria Garcia,123-45-6789,804-555-1234\nJohn Smith,234-56-7890,(212) 555-0199\n",
		"123-45-6789\n234-56-7890\n345-67-8901\n456-78-9012\n",
		"(804) 555-1234\n(212) 555-0199\n(415) 555-0123\n",
	} {
		for where, body := range allOpaquePositions(base64.StdEncoding.EncodeToString([]byte(text))) {
			resp, out := postJSON(t, gw.URL+"/v1/messages", body, nil)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("%q in %s: %d %s", text, where, resp.StatusCode, out)
			}
		}
	}
}

const pemKey = "--" + "---BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA1Xk7v8Y2rQ9mZ3pL6tW4sN0bH5cJ8dF2gK7jM1nP3qR5sT9u\nV1wX3yZ5aB7cD9eF1gH3iJ5kL7mN9oP1qR3sT5uV7wX9yZ1aB3cD5eF7gH9iJ1kL3\n-----END RSA PRIVATE KEY-----\n"

func TestSecretsInsideBinary(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	r := rand.New(rand.NewSource(77))
	glued := func(secret string) []byte {
		pre, post := uniformBytes(r, 150), uniformBytes(r, 150)
		pre[len(pre)-1], post[0] = 'x', '7'
		return append(append(pre, secret...), post...)
	}
	secrets := map[string]string{
		"pem":    pemKey,
		"sk-ant": "sk-ant-api03-" + strings.Repeat("Ab3dE5gH7jK9mN1pQ3sT5vW7yZ", 3) + "-AbCdEfGh",
		"ghp":    "ghp_" + strings.Repeat("Ab3dE5gH7j", 4),
		"jwt":    "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
		"akia":   "AK" + "IAIOSFODNN7EXAMPLE",
	}
	for name, secret := range secrets {
		for encName, v := range encodings(glued(secret)) {
			resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(v), nil)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("%s glued in binary (%s): %d %s", name, encName, resp.StatusCode, out)
			}
		}
	}
}

func TestBinaryNeverHighEntropyInFallbackPositions(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	blob := base64.StdEncoding.EncodeToString(uniformBytes(rand.New(rand.NewSource(9)), 600))
	user := anthBody("hello", false)
	firstMsg(user)["content"] = []any{map[string]any{"type": "web_search_tool_result", "tool_use_id": "srvtoolu_01WYG3ziw53XMcoyKL4XcZmE", "content": []any{
		map[string]any{"type": "web_search_result", "title": "Guide", "url": "https://x.test", "encrypted_content": blob}}}, map[string]any{"type": "text", "text": "hi"}}
	extra := anthBody("hello", false)
	extra["extra"] = map[string]any{"nested": map[string]any{"signature": blob}}
	tool := anthBody("hello", false)
	tool["messages"] = append(tool["messages"].([]any), map[string]any{"role": "assistant", "content": []any{
		map[string]any{"type": "tool_use", "id": "toolu_01A09q90qw90lq917835lq9", "name": "t", "input": map[string]any{"state": map[string]any{"encrypted_content": blob}}}}})
	for name, b := range map[string]map[string]any{"user web search": user, "nested fallback": extra, "tool input": tool} {
		resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
		if resp.StatusCode != 200 {
			t.Fatalf("%s: %d %s", name, resp.StatusCode, out)
		}
	}
}

func TestMixedCaseNamesInIDs(t *testing.T) {
	for _, id := range []string{"toolu_01MariaGARCIA9xQ2mZr7", "toolu_01MARIAgarcia9xQ2mZr7", "toolu_01ElenaPETROVA9xQ2mZr7", "toolu_01JohnSMITH9xQ2mZr7", "toolu_01JOHNsmith9xQ2mZr7"} {
		if idShaped(id) {
			t.Fatalf("name-bearing id accepted: %s", id)
		}
	}
	if !idShaped("toolu_01CrystalBall9xQ2mZr7Lp") {
		t.Fatal("CrystalBall is two common words")
	}
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	b := chatBody("hello", false)
	b["metadata"] = map[string]any{"session_id": "toolu_01CrystalBall9xQ2mZr7Lp"}
	if resp, out := postJSON(t, gw.URL+"/v1/chat/completions", b, nil); resp.StatusCode != 200 {
		t.Fatalf("CrystalBall id: %d %s", resp.StatusCode, out)
	}
}

func TestDateCuesWiderAndMultilingual(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, phrase := range []string{"date of birth is listed below as 1985-03-14", "d.o.b 1985-03-14", "дата рождения 14.03.1985", "fecha de nacimiento 14/03/1985",
		"Geburtsdatum 14.03.1985", "date de naissance 14/03/1985", "nato il 14/03/1985", "data de nascimento 14/03/1985"} {
		v := base64.StdEncoding.EncodeToString([]byte("Record notes. The " + phrase + " for the file."))
		resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(v), nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("%q: %d %s", phrase, resp.StatusCode, out)
		}
	}
}
