package gateway

import (
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
)

func TestShortNamePayloadsInBinary(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for name, fields := range map[string][][]byte{
		"name":            {field(2, []byte("Elena Petrova"))},
		"label then name": {field(2, []byte("patient name")), field(3, []byte("Elena Petrova"))},
		"name colon":      {field(2, []byte("name:")), field(3, []byte("Elena Petrova"))},
		"name and date":   {field(2, []byte("Elena Petrova")), field(3, []byte("1985-03-14"))},
		"email at ip":     {field(2, []byte("elena@10.0.0.1"))},
	} {
		expect422Everywhere(t, gw.URL, name, base64.StdEncoding.EncodeToString(binaryOuter(fields...)))
	}
}

func TestFormatCharacterAsOnlySeparator(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, text := range []string{"SSN\u200b123-45-6789", "Contact\u200bElena\u200bPetrova\u200btoday", "Contact\u2060Elena\u2060Petrova\u2060today"} {
		expect422Everywhere(t, gw.URL, text, base64.StdEncoding.EncodeToString([]byte(text)))
	}
	b := anthBody("hello", false)
	b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "redacted_thinking",
		"data": base64.StdEncoding.EncodeToString([]byte("Write to maria.garcia@example.com\u200b today"))}}}, map[string]any{"role": "user", "content": "go"})
	resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity || strings.Count(string(out), `"source":"regex:email"`) != 1 {
		t.Fatalf("both variants must merge into one finding: %d %s", resp.StatusCode, out)
	}
}

func TestDetectorCardAndPhoneTokens(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, tok := range []string{"6250941006528599", "6759649826438453", "1.804.555.1234", "4111111111111111", "804-555-1234"} {
		v := base64.StdEncoding.EncodeToString(binaryOuter(field(2, []byte(tok))))
		resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(v), nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("token %s: %d %s", tok, resp.StatusCode, out)
		}
	}
}

func TestFramedOuterKeepsSplitValuesDropsCrossFieldCues(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	base := "By doctor data shipping January? Other we file 7293 what error your are may will please config? Key result and i "
	for _, s := range []string{base + "SSN 123-45-6789", base + "phone 804-555-1234", base + "card 4111 1111 1111 1111"} {
		expect422Everywhere(t, gw.URL, s, base64.StdEncoding.EncodeToString([]byte(s)))
	}
	cue := field(1, []byte("the access token for the service is stored in the next record of this message"))
	id := field(2, []byte("q7Zk2mN9xR4tW8vB1cD5fG3hJ6kL0pS2uY4aE7iO9"))
	filler := field(3, []byte(strings.Repeat("plain english words about the build and the release notes ", 3)))
	v := base64.StdEncoding.EncodeToString(append(append(cue, id...), filler...))
	resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(v), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("cue in one field, random id in the next: %d %s", resp.StatusCode, out)
	}
}

func TestProsePayloadMinimumAvoidsShortWordNoise(t *testing.T) {
	if proseShaped([]byte("ViLA")) || proseShaped([]byte("ed")) || !proseShaped([]byte("Please send the report to the team today.")) {
		t.Fatal("prose payload threshold")
	}
}
