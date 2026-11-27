package gateway

import (
	"encoding/base64"
	"encoding/binary"
	"math/rand"
	"net/http"
	"strings"
	"testing"
	"unicode"
)

func caseVariants(s string) []string {
	mixed := []rune(s)
	for i := range mixed {
		if i%2 == 0 {
			mixed[i] = unicode.ToUpper(mixed[i])
		} else {
			mixed[i] = unicode.ToLower(mixed[i])
		}
	}
	return []string{s, strings.ToUpper(s), strings.ToLower(s), string(mixed)}
}

var round14Cues = []string{"reach me at 8045551234", "fax 8045551234", "acct 12345678901", "routing 021000021", "member 123456789", "text me at 8045551234",
	"my SSN is 123456789", "call me on 8045551234", "phone 8045551234", "passport number 123456789", "account 12345678901",
	"TEL 8045551234", "PHONE: 8045551234", "CALL 8045551234", "CELL 8045551234", "MOBILE 8045551234", "FAX 8045551234", "CONTACT 8045551234", "pHone 8045551234"}

var shortCues = []string{"Ph 8045551234", "ph: 8045551234", "PhoneNumber 8045551234", "SMS 8045551234", "WhatsApp 8045551234"}

func TestCueCaseInsensitive(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, phrase := range append(append([]string{}, round14Cues...), shortCues...) {
		for _, variant := range caseVariants(phrase) {
			text := []byte("Notes from the meeting today. The client said " + variant + " and asked for a follow up next week.")
			for name, raw := range map[string][]byte{"decoded text": text, "prose in binary": binaryOuter(field(2, text))} {
				resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(base64.StdEncoding.EncodeToString(raw)), nil)
				if resp.StatusCode != http.StatusUnprocessableEntity {
					t.Fatalf("%q as %s: %d %s", variant, name, resp.StatusCode, out)
				}
			}
		}
	}
}

func TestSecretContextNeedsItsOwnKey(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	secret := "q7Zk2mN9xR4tW8vB1cD5fG3hJ6kL0pS2uY4aE7iO9"
	for _, s := range []string{"the api token: " + secret, "access token is " + secret, "password=" + secret, "secret \"" + secret + "\"",
		"use sk-ant-api03-" + strings.Repeat("Ab3dE5gH7jK9mN1pQ3sT5vW7yZ", 3) + "-AbCdEfGh now"} {
		resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(base64.StdEncoding.EncodeToString([]byte("Config notes: "+s+" for the build."))), nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("real secret %q: %d %s", s, resp.StatusCode, out)
		}
	}
	glued := []byte("the cache token release note for the user*(" + secret + " and the next build is ok for the team today and more words")
	resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(base64.StdEncoding.EncodeToString(glued)), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("framing bytes between a key word and a random id: %d %s", resp.StatusCode, out)
	}
}

func TestCardGluedToWordInDecodedText(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, s := range []string{"ship to 12 avenu4111 1111 1111 1111 please", "ref x4111-1111-1111-1111 ok"} {
		for name, raw := range map[string][]byte{"decoded text": []byte(s), "prose in binary": binaryOuter(field(2, []byte(s+" for the order")))} {
			resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(base64.StdEncoding.EncodeToString(raw)), nil)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("%q as %s: %d %s", s, name, resp.StatusCode, out)
			}
		}
	}
}

func TestTextualOuterNeverLosesFindingsWhenFramed(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	r := rand.New(rand.NewSource(1616))
	words := strings.Fields("by doctor data shipping january other we file what error your are may will please config key result and i the of to in for on")
	tails := []string{"DOB: 1985-03-14", "born 03/14/1985", "SSN 123-45-6789", "phone 804-555-1234", "card 4111 1111 1111 1111", "email maria.garcia@example.com"}
	framed := map[string]int{}
	for tries := 0; tries < 400000 && len(framed) < len(tails)*3; tries++ {
		var b strings.Builder
		for b.Len() < 80+r.Intn(60) {
			w := words[r.Intn(len(words))]
			if r.Intn(4) == 0 {
				w = strings.ToUpper(w[:1]) + w[1:]
			}
			b.WriteString(w)
			b.WriteString([]string{" ", " ", "? ", ". ", " 7293 "}[r.Intn(5)])
		}
		tail := tails[r.Intn(len(tails))]
		s := b.String() + tail
		if !parseFrame([]byte(s)).framed || framed[tail] >= 3 {
			continue
		}
		framed[tail]++
		resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(base64.StdEncoding.EncodeToString([]byte(s))), nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("framed textual value lost its finding: %q: %d %s", s, resp.StatusCode, out)
		}
	}
	if len(framed) == 0 {
		t.Fatal("no accidentally framed samples found")
	}
	t.Logf("accidentally framed samples checked: %v", framed)
}

var mixedWords = []string{"user", "session", "token", "key", "cache", "build", "status", "ok", "error", "id", "name", "path", "value", "type", "secret", "hash", "signature", "region", "model"}

func mixedProtobuf(r *rand.Rand, n int) []byte {
	var b []byte
	for len(b) < n {
		field := uint64(1 + r.Intn(30))
		switch r.Intn(6) {
		case 0:
			b = binary.AppendUvarint(b, field<<3)
			b = binary.AppendUvarint(b, r.Uint64()>>uint(r.Intn(60)))
		case 1:
			b = binary.AppendUvarint(b, field<<3|1)
			b = binary.LittleEndian.AppendUint64(b, r.Uint64())
		case 2:
			b = binary.AppendUvarint(b, field<<3|5)
			b = binary.LittleEndian.AppendUint32(b, r.Uint32())
		case 3:
			x := uniformBytes(r, 4+r.Intn(60))
			b = binary.AppendUvarint(b, field<<3|2)
			b = binary.AppendUvarint(b, uint64(len(x)))
			b = append(b, x...)
		case 4:
			s := mixedWords[r.Intn(len(mixedWords))]
			if r.Intn(2) == 0 {
				s += " " + mixedWords[r.Intn(len(mixedWords))]
			}
			b = binary.AppendUvarint(b, field<<3|2)
			b = binary.AppendUvarint(b, uint64(len(s)))
			b = append(b, s...)
		default:
			s := randTail(r, 8+r.Intn(48))
			b = binary.AppendUvarint(b, field<<3|2)
			b = binary.AppendUvarint(b, uint64(len(s)))
			b = append(b, s...)
		}
	}
	return b
}

func TestMixedProtobufRejectionRate(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	sizes := map[int]int{800: 200, 4096: 100, 16384: 50}
	if corpusScale > 1 {
		sizes = map[int]int{800: 50 * corpusScale}
	}
	for size, n := range sizes {
		r := rand.New(rand.NewSource(int64(4242 + size)))
		n = max(25, n/corpusScale)
		blocked := 0
		for done := 0; done < n; done += 25 {
			var blobs []string
			var pos []int
			for i := 0; i < 25; i++ {
				blobs = append(blobs, base64.StdEncoding.EncodeToString(mixedProtobuf(r, size)))
				pos = append(pos, done+i)
			}
			found, hit := corpusFindingsByBlob(t, srv, blobs, pos)
			if hit > 0 {
				t.Logf("size %d: %v", size, found)
			}
			blocked += hit
		}
		t.Logf("size %d: %d of %d blocked", size, blocked, n)
		if blocked*400 > 2*n {
			t.Fatalf("size %d: %d of %d blocked, ceiling 2 per 400", size, blocked, n)
		}
	}
}
