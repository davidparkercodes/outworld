package gateway

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/http"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/outworld"
)

var naturalSecrets = []string{
	"q7Zk2mN9xR4tW8vB1cD5fG3hJ6kL0pS2uY4aE7iO9",
	"Hx83Lm0QzP2vR7tK9sW4nB6cY1dF5gJ8aE3uI0oT",
	"pR4sT7uV0wX3yZ6aB9cD2eF5gH8iJ1kL4mN7oP0qR3",
	"bW9xZ2tQ8rT5yU1iO4pA7sD0fG3hJ6kL9zX2cV5bN8",
}

var secretPhrasings = []string{
	"Config notes: the secret key we use in prod is %s for the build.",
	"here is the production key, copy it: %s",
	"signing secret (rotate monthly): %s",
	"The API key is %s, please keep it safe.",
	"token=%s",
	"password: %s",
	"my access token %s expires friday",
	"Use this key: %s",
	"credential for the staging bucket -> %s",
	"Set the webhook secret to %s before deploying.",
	"apikey \"%s\"",
	"the password hash we computed was %s and it matched",
	"signature: '%s'",
	"Passphrase for the vault is %s",
	"the secret key we use in prod is %s",
	"Here is the password I set, %s",
	"token for the bot is %s ok",
	"our api key for staging: %s",
	"use this access token %s",
	"the client secret is %s",
	"my private key is %s",
	"Here's the token you asked for %s",
	"new key (do not share) %s",
	"KEY = %s",
	"bot token for slack, keep private: `%s`",
	"auth token [%s]",
	"Your temporary password is now %s; change it on login.",
	"the deploy key, as discussed yesterday: %s",
	"Secret: %s",
	"credentials -> user admin, %s",
	"Paste the key %s into the settings page.",
	"tokens rotated, current one %s",
}

var secretNegatives = []string{
	"the key is banana and the lock is blue",
	"token count was 4096 for that request",
	"Press the key labelled Enter to continue.",
	"the secret ingredient is cinnamon",
	"password reset link expired yesterday",
	"key 3f2a9c1e4b5d6a7f8e9d0c1b2a3f4e5d6c7b8a9f",
	"token id 550e8400-e29b-41d4-a716-446655440000",
	"signature version v4.2.17 is required",
	"hash algorithm sha256 is the default",
	"the API key field must not be empty",
	"secret santa list goes out monday",
	"key: value pairs are separated by commas",
	"Token budget is 200000 per day.",
	"credential helper is osxkeychain",
	"keyboard shortcuts are listed in the guide",
	"the hash 9b74c9897bac770ffc029102a200c5de is cached",
	"token request id 7c9e6679-7425-40de-944b-e07fc1f97466",
	"key version 2.14.3 shipped last week",
	"the password policy requires twelve characters",
	"Signature of the contract is due next Tuesday.",
	"keys are rotated every ninety days",
	"tokenizer vocabulary has fifty thousand entries",
	"key=enabled",
	"secret=true",
	"token: refresh",
	"the hash table resized at 75 percent load",
	"passphrase prompts are disabled in CI",
	"key ab12cd34ef56ab78cd90ef12ab34cd56",
	"apikey placeholder goes here",
	"signature check passed for release v1.0.3",
	"secrets manager region is us-east-1",
	"token expiry is 3600 seconds",
	"cache key for the page is q7Zk2mN9xR4tW8vB1cD5fG3hJ6kL0pS2uY4aE7iO9",
	"build hash Hx83Lm0QzP2vR7tK9sW4nB6cY1dF5gJ8aE3uI0oT",
	"the commit is pR4sT7uV0wX3yZ6aB9cD2eF5gH8iJ1kL4mN7oP0qR3",
}

func blobSources(t *testing.T, srv *Server, raw []byte) []string {
	t.Helper()
	found := corpusFindings(t, srv, []string{base64.StdEncoding.EncodeToString(raw)}, []int{1})
	var out []string
	for k := range found {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestNaturalSecretPhrasingsBlock(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	blocked, total := 0, 0
	for i, phrasing := range secretPhrasings {
		for j := 0; j < 3; j++ {
			s := fmt.Sprintf(phrasing, naturalSecrets[(i+j)%len(naturalSecrets)])
			for name, raw := range map[string][]byte{"decoded text": []byte(s), "prose in binary": binaryOuter(field(2, []byte(s)))} {
				total++
				resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(base64.StdEncoding.EncodeToString(raw)), nil)
				if resp.StatusCode != http.StatusUnprocessableEntity {
					t.Errorf("%q as %s: %d %s", s, name, resp.StatusCode, out)
					continue
				}
				blocked++
			}
		}
	}
	t.Logf("natural secret phrasings blocked %d of %d", blocked, total)
}

func TestKeyWordWithOrdinaryValuePasses(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, s := range secretNegatives {
		for name, raw := range map[string][]byte{"decoded text": []byte(s), "prose in binary": binaryOuter(field(2, []byte(s)))} {
			resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(base64.StdEncoding.EncodeToString(raw)), nil)
			if resp.StatusCode != 200 {
				t.Errorf("%q as %s: %d %s", s, name, resp.StatusCode, out)
			}
		}
	}
}

var keyedWords = strings.Fields("key token id secret session cache user request trace build status value type name path index")

func keyedProtobuf(r *rand.Rand, n int) []byte {
	var b []byte
	for len(b) < n {
		var s string
		switch r.Intn(3) {
		case 0:
			s = randTail(r, 32+r.Intn(8))
		default:
			var w strings.Builder
			for w.Len() < 40+r.Intn(50) {
				w.WriteString(keyedWords[r.Intn(len(keyedWords))])
				w.WriteString([]string{" ", " ", ": ", "_", " "}[r.Intn(5)])
			}
			s = w.String()
		}
		b = append(b, byte(4+r.Intn(12))<<3|2, byte(len(s)))
		b = append(b, s...)
	}
	return b
}

func TestProtobufKeyWordsNextToRandomIDs(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	ceilings := map[int]int{800: 11, 4096: 38}
	if corpusScale > 1 {
		delete(ceilings, 4096)
	}
	for size, ceiling := range ceilings {
		r := rand.New(rand.NewSource(int64(1717 + size)))
		const perRequest = 20
		const n = 100
		tokens, framedText := 0, 0
		for done := 0; done < n; done += perRequest {
			var blobs []string
			var pos []int
			for i := 0; i < perRequest; i++ {
				raw := keyedProtobuf(r, size)
				if textual(raw) && parseFrame(raw).framed {
					framedText++
				}
				blobs = append(blobs, base64.StdEncoding.EncodeToString(raw))
				pos = append(pos, done+i)
			}
			found, _ := corpusFindingsByBlob(t, srv, blobs, pos)
			tokens += found["regex:context_token"]
		}
		t.Logf("%d blobs of %d bytes, %d textual and framed: context_token %d", n, size, framedText, tokens)
		if tokens*100 > ceiling*n {
			t.Fatalf("context_token fired %d times on %d keyed protobuf blobs of %d bytes, ceiling %d per 100", tokens, n, size, ceiling)
		}
	}
}

var minedWords = strings.Fields("by doctor data shipping january other we file what error your are may will please config key result and i the of to in for on note team")

var minedTails = []string{"DOB: 1985-03-14", "born 03/14/1985", "SSN 123-45-6789", "phone 804-555-1234", "card 4111 1111 1111 1111",
	"email maria.garcia@example.com", "call 8045551234", "secret key: q7Zk2mN9xR4tW8vB1cD5fG3hJ6kL0pS2uY4aE7iO9", "meet at 10.0.0.12", "see you then", "patient Maria Garcia"}

var unframeSuffixes = []string{" .", " !", ".", " ok.", "\n"}

func minedSentence(r *rand.Rand, ansi bool) string {
	var b strings.Builder
	for b.Len() < 70+r.Intn(60) {
		w := minedWords[r.Intn(len(minedWords))]
		if r.Intn(4) == 0 {
			w = strings.ToUpper(w[:1]) + w[1:]
		}
		switch {
		case ansi && r.Intn(3) == 0:
			w = "\x1b[3" + string(rune('1'+r.Intn(6))) + "m" + w + "\x1b[0m"
		case ansi && r.Intn(12) == 0:
			w += string(rune([]byte{0x01, 0x07, 0x7f, 0x1b}[r.Intn(4)]))
		}
		b.WriteString(w)
		b.WriteString([]string{" ", " ", "? ", ". ", " 7293 ", ", "}[r.Intn(6)])
	}
	return b.String() + minedTails[r.Intn(len(minedTails))]
}

func TestFramingNeverChangesTheOutcome(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	for _, ansi := range []bool{false, true} {
		r := rand.New(rand.NewSource(1717))
		want := 300 / corpusScale
		mined, extra, noisy := 0, 0, 0
		for tries := 0; tries < 4000000 && mined < want; tries++ {
			s := minedSentence(r, ansi)
			if !textual([]byte(s)) || !parseFrame([]byte(s)).framed {
				continue
			}
			plain := ""
			for _, suf := range unframeSuffixes {
				if u := s + suf; !parseFrame([]byte(u)).framed {
					plain = u
					break
				}
			}
			if plain == "" {
				continue
			}
			mined++
			band := printableShare([]byte(s)) < cleanTextShare
			if band {
				noisy++
			}
			framed, unframed := blobSources(t, srv, []byte(s)), blobSources(t, srv, []byte(plain))
			var required []string
			for _, src := range unframed {
				if !band || !noisyFrameSources[src] {
					required = append(required, src)
				}
			}
			if len(required) > 0 && len(framed) == 0 || !band && len(unframed) == 0 && len(framed) > 0 {
				t.Errorf("status differs for %q: framed %v, unframed %v", s, framed, unframed)
			}
			for _, src := range required {
				if !slices.Contains(framed, src) {
					t.Errorf("framing removed %s from %q: framed %v, unframed %v", src, s, framed, unframed)
				}
			}
			if len(framed) > len(unframed) {
				extra++
			}
		}
		if mined < want || ansi && noisy < want/2 {
			t.Fatalf("ansi %v: mined only %d framed samples, %d below the clean-text share", ansi, mined, noisy)
		}
		t.Logf("ansi %v: %d framed samples, %d below the clean-text share, %d with extra field-level sources", ansi, mined, noisy, extra)
	}
}

func TestPhoneWordInsideIdentifierIsNotACue(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, s := range []string{`{"callId":"8045551234","state":"open"}`, `{"cellIndex":3105550123,"row":4}`, `getCellValue(8045551234) returned`,
		`{"call_id":"8045551234"}`, `{"mobileVersion":8045551234}`, `faxCount 8045551234`, `callId=8045551234`, `callID: 8045551234`,
		`phoneId=8045551234`, `smsId 8045551234`, `callSid=8045551234`, `telephoneId=8045551234`, `phonesCount=8045551234`,
		`cellphoneModel 8045551234`, `{"callId":"804~555~1234"}`} {
		resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(base64.StdEncoding.EncodeToString([]byte(s))), nil)
		if resp.StatusCode != 200 {
			t.Errorf("%q: %d %s", s, resp.StatusCode, out)
		}
	}
	for _, s := range []string{`{"phoneNumber":"8045551234"}`, `PhoneNumber 8045551234`, `{"phone_number":"8045551234"}`, `cellNo: 8045551234`,
		`mobilePhone: 8045551234`, `{"homePhone":"8045551234"}`, `callbackNumber=8045551234`, `userPhone 8045551234`} {
		resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(base64.StdEncoding.EncodeToString([]byte(s))), nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("%q: %d %s", s, resp.StatusCode, out)
		}
	}
}

var validatorPositives = []string{"the secret key we use in prod is %s", "Here is the password I set, %s", "token for the bot is %s ok",
	"our api key for staging: %s", "use this access token %s", "the client secret is %s", "my private key is %s"}

var validatorNegatives = []string{"the request id in prod is %s", "trace for this run: %s", "the commit is %s", "cache key for the page is %s",
	"context_token %s", "the session nonce is %s", "build hash %s", "commit hash %s", "file hash %s", "content hash %s",
	"primary key %s", "the idempotency key is %s", "lookup key: %s", "row key %s", "partition key for the table is %s"}

func signatureSources(t *testing.T, srv *Server, raw []byte) map[string]int {
	t.Helper()
	return corpusFindings(t, srv, []string{base64.StdEncoding.EncodeToString(raw)}, []int{0})
}

func TestValidatorSecretCasesInSignature(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	r := rand.New(rand.NewSource(1718))
	for i := 0; i < 20; i++ {
		secret := randTail(r, 40)
		for _, p := range validatorPositives {
			s := fmt.Sprintf(p, secret)
			if len(signatureSources(t, srv, field(1, []byte(s)))) == 0 {
				t.Errorf("one field %q: not blocked", s)
			}
		}
		for _, p := range validatorNegatives {
			s := fmt.Sprintf(p, secret)
			if found := signatureSources(t, srv, field(1, []byte(s))); len(found) > 0 {
				t.Errorf("one field %q: %v", s, found)
			}
		}
		for _, p := range validatorPositives {
			phrase, _, _ := strings.Cut(p, "%s")
			raw := append(field(1, []byte(phrase)), field(2, []byte(secret))...)
			if found := signatureSources(t, srv, raw); len(found) > 0 {
				t.Errorf("cue in field 1, random id in field 2 %q (textual %v): %v", phrase, textual(raw), found)
			}
		}
	}
}

func TestGluedCardsInShortBinaryFields(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, s := range []string{"avenu4111 1111 1111 1111", "x4111-1111-1111-1111", "ref x4111-1111-1111-1111"} {
		resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(base64.StdEncoding.EncodeToString(binaryOuter(field(2, []byte(s))))), nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("%q: %d %s", s, resp.StatusCode, out)
		}
	}
}

func TestPhoneQualifierSuffixKeepsTheCue(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, s := range []string{`{"phone_ext": 2025550123}`, `{"phoneExt":"2025550123"}`, `{"phone_extension":"2025550123"}`, `{"mobile_num":"2025550123"}`,
		`{"callbackNumber":"8045551234"}`, `{"whatsappNumber":"8045551234"}`, `{"emergencyContactPhoneNumber":"8045551234"}`} {
		for where, body := range allOpaquePositions(base64.StdEncoding.EncodeToString([]byte(s))) {
			resp, out := postJSON(t, gw.URL+"/v1/messages", body, nil)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Errorf("%s in %s: %d %s", s, where, resp.StatusCode, out)
			}
		}
	}
	for _, s := range []string{`{"call_ext_count": 2025550123}`, `{"textNumber":"2025550123"}`, `{"phoneIdNumber_x":"2025550123"}`} {
		resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(base64.StdEncoding.EncodeToString([]byte(s))), nil)
		if resp.StatusCode != 200 {
			t.Errorf("%s: %d %s", s, resp.StatusCode, out)
		}
	}
}

var identifierSecretKeys = []string{"api_key", "auth_token", "access_token", "client_secret", "db_password", "private_key", "secret_key", "refresh_token",
	"API_KEY", "authToken", "secretKey", "dbPassword", "session_token", "bot_token", "aws_secret_access_key", "gpg_passphrase", "aws_credential", "openai_apikey"}

var identifierSeparators = []string{" ", "\t", " -> ", " is ", ":\n  ", "=", ": "}

func mixedSecret(r *rand.Rand, n int) string {
	for {
		s := randTail(r, n)
		if strings.ContainsAny(s, "0123456789") && strings.ToLower(s) != s && strings.ToUpper(s) != s {
			return s
		}
	}
}

func TestIdentifierStyleSecretKeys(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	r := rand.New(rand.NewSource(1900))
	lengths := []int{16, 20, 24, 40}
	if corpusScale > 1 {
		lengths = []int{16, 40}
	}
	for _, n := range lengths {
		secret := mixedSecret(r, n)
		for _, key := range identifierSecretKeys {
			for _, sep := range identifierSeparators {
				s := "Notes: " + key + sep + secret + " thanks."
				for name, raw := range map[string][]byte{"decoded text": []byte(s), "prose in binary": binaryOuter(field(2, []byte(s))), "one field": field(1, []byte(key+sep+secret))} {
					if len(signatureSources(t, srv, raw)) == 0 {
						t.Errorf("%q as %s: not blocked", s, name)
					}
				}
			}
		}
		if len(signatureSources(t, srv, []byte("tokens:\n- "+secret))) == 0 {
			t.Errorf("yaml token list with %d chars: not blocked", n)
		}
		for _, s := range []string{"context_token " + secret, "next_page_token " + secret, "cache_key " + secret, "request_token_id " + secret,
			"idempotency_key -> " + secret, "partition_key is " + secret, "trace_id " + secret, "commit_hash " + secret, "key_id " + secret} {
			if found := signatureSources(t, srv, field(1, []byte(s))); len(found) > 0 {
				t.Errorf("%q: %v", s, found)
			}
		}
	}
}

func TestExemptQualifiersApplyInEveryForm(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	r := rand.New(rand.NewSource(1901))
	forms := []func(k, v string) string{
		func(k, v string) string { return k + " " + v },
		func(k, v string) string { return k + ": " + v },
		func(k, v string) string { return k + "=" + v },
		func(k, v string) string { return k + " -> " + v },
		func(k, v string) string { return k + " is " + v },
		func(k, v string) string { return `{"` + k + `":"` + v + `"}` },
	}
	for _, n := range []int{16, 24, 40} {
		v := mixedSecret(r, n)
		for _, k := range []string{"pageToken", "page_token", "nextPageToken", "next_page_token", "cursorToken", "cursor_token", "requestToken", "traceToken",
			"cache_token", "cacheKey", "cache_key", "primary_key", "lookupKey", "idempotency_key"} {
			for _, form := range forms {
				if found := signatureSources(t, srv, field(1, []byte(form(k, v)))); len(found) > 0 {
					t.Errorf("exempt %q: %v", form(k, v), found)
				}
			}
		}
		for _, k := range []string{"access_token", "auth_token", "refresh_token", "bearer_token", "api_token", "session_token", "id_token", "accessToken", "apiKey"} {
			for _, form := range forms {
				if len(signatureSources(t, srv, field(1, []byte(form(k, v))))) == 0 {
					t.Errorf("real secret %q: not blocked", form(k, v))
				}
			}
		}
	}
}

func TestQualifiedPhoneKeysCueEverySeparator(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	for _, k := range []string{"phone", "phone_ext", "phoneExt", "phone-ext", "PHONE_EXT", "mobile_ext", "tel_ext", "phone_extension", "callbackNumber", "whatsappNumber", "mobile_num"} {
		for _, v := range []string{"804~555~1234", "804-555-1234", "804.555.1234", "8045551234", "(804) 555-1234"} {
			s := `{"` + k + `":"` + v + `"}`
			if len(signatureSources(t, srv, []byte(s))) == 0 {
				t.Errorf("%s: not blocked", s)
			}
		}
	}
	for _, k := range []string{"phoneId", "call_count", "phone_ext_id", "textNumberCount"} {
		s := `{"` + k + `":"8045551234"}`
		if found := signatureSources(t, srv, []byte(s)); len(found) > 0 {
			t.Errorf("%s: %v", s, found)
		}
	}
}

func TestKeyWordOnAnEarlierLineDoesNotLabelTheValue(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	secret := naturalSecrets[0]
	for _, s := range []string{"api token\ncache key " + secret, "the secret\nlookup key: " + secret} {
		if found := signatureSources(t, srv, field(1, []byte(s))); len(found) > 0 {
			t.Errorf("%q: %v", s, found)
		}
	}
}

func digitsOf(r *rand.Rand, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte('0' + r.Intn(10))
	}
	return string(b)
}

func upperAlnum(r *rand.Rand, n int) string {
	const set = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	b := make([]byte, n)
	for i := range b {
		b[i] = set[r.Intn(len(set))]
	}
	return string(b)
}

func formatSecrets(r *rand.Rand) map[string]string {
	return map[string]string{
		"github":  "gh" + "p_" + mixedSecret(r, 36),
		"stripe":  "sk_" + "live_" + mixedSecret(r, 24),
		"aws":     "AK" + "IA" + upperAlnum(r, 16),
		"jwt":     "ey" + "JhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9" + "." + "ey" + "JzdWIiOiIxMjM0NTY3ODkwIn0" + "." + mixedSecret(r, 43),
		"slack":   "xo" + "xb-" + digitsOf(r, 12) + "-" + digitsOf(r, 13) + "-" + mixedSecret(r, 24),
		"gitlab":  "glp" + "at-" + mixedSecret(r, 20),
		"openai":  "sk-" + "proj-" + mixedSecret(r, 48),
		"pem":     "-----BEGIN " + "RSA PRIVATE KEY-----\nMIIEow" + mixedSecret(r, 60) + "\n-----END RSA PRIVATE KEY-----",
		"bearer":  "Bearer " + mixedSecret(r, 40),
		"urlcred": "postgres://admin:" + mixedSecret(r, 14) + "@db.example.com/app",
	}
}

func TestFormatSecretsAfterExemptKeysStillBlock(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	r := rand.New(rand.NewSource(1910))
	forms := []func(k, v string) string{
		func(k, v string) string { return k + ": " + v },
		func(k, v string) string { return `{"` + k + `":"` + v + `"}` },
		func(k, v string) string { return k + " " + v },
		func(k, v string) string { return k + "=" + v },
	}
	secrets := formatSecrets(r)
	missed, total := 0, 0
	for _, k := range []string{"cache_key", "next_page_token", "context_token", "request_token", "idempotency_key", "lookup_key", "pageToken", "sync_token"} {
		for name, v := range secrets {
			for _, form := range forms {
				s := form(k, v)
				for where, raw := range map[string][]byte{"decoded text": []byte(s), "protobuf field": field(1, []byte(s)), "prose in binary": binaryOuter(field(2, []byte("config "+s)))} {
					total++
					if len(signatureSources(t, srv, raw)) == 0 {
						missed++
						t.Errorf("%s %s after %s as %s: not blocked", name, form(k, "V"), k, where)
					}
				}
			}
		}
	}
	if len(signatureSources(t, srv, []byte("value "+secrets["github"]))) == 0 {
		t.Error("control: not blocked")
	}
	t.Logf("format secrets after exempt keys blocked %d of %d", total-missed, total)
}

func TestVendorValueIsNeverExemptEvenUnderAGenericSource(t *testing.T) {
	r := rand.New(rand.NewSource(1911))
	for name, v := range formatSecrets(r) {
		if name == "pem" || name == "bearer" || name == "urlcred" {
			continue
		}
		text := "pageToken=" + v
		viol := outworld.Violation{Source: "regex:kv_secret", Start: len([]rune("pageToken=")), End: len([]rune(text))}
		if exemptGenericSecret([]rune(text), viol) {
			t.Errorf("%s value after an exempt key was exempted as a generic finding", name)
		}
	}
	viol := outworld.Violation{Source: "regex:kv_secret", Start: len("pageToken="), End: len("pageToken=") + 24}
	if !exemptGenericSecret([]rune("pageToken="+mixedSecret(r, 24)), viol) {
		t.Error("a generic value after an exempt key must stay exempt")
	}
}

func TestPhoneCueKeyWithWordsBeforeTheValue(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	for _, s := range []string{"userPhone (home) 8045551234", "contactPhone (work) 8045551234", "callbackNumber (home) 8045551234", "whatsappNumber (main) 8045551234"} {
		for name, raw := range map[string][]byte{"decoded text": []byte(s), "one field": field(1, []byte(s))} {
			if len(signatureSources(t, srv, raw)) == 0 {
				t.Errorf("%q as %s: not blocked", s, name)
			}
		}
	}
	for _, s := range []string{"callId (home) 8045551234", "phoneCount (home) 8045551234"} {
		if found := signatureSources(t, srv, []byte(s)); len(found) > 0 {
			t.Errorf("%q: %v", s, found)
		}
	}
}

func TestCountAndDataStructureKeysAreNotSecrets(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	r := rand.New(rand.NewSource(1932))
	plain := []string{"max_tokens", "max_output_tokens", "max_completion_tokens", "input_tokens", "output_tokens", "prompt_tokens", "completion_tokens",
		"total_tokens", "cached_tokens", "num_tokens", "maxTokens", "maxOutputTokens", "inputTokens", "totalTokens", "shard_key", "dedup_key", "s3_key",
		"hash_key", "range_key", "public_key", "publicKey", "group_key", "composite_key", "unique_key", "natural_key", "surrogate_key", "object_key", "bucket_key"}
	forms := []func(k, v string) string{
		func(k, v string) string { return k + " " + v },
		func(k, v string) string { return k + ": " + v },
		func(k, v string) string { return k + "=" + v },
		func(k, v string) string { return `{"` + k + `":"` + v + `"}` },
	}
	for _, n := range []int{16, 24, 40} {
		v := mixedSecret(r, n)
		for _, k := range plain {
			for _, form := range forms {
				if found := signatureSources(t, srv, field(1, []byte(form(k, v)))); len(found) > 0 {
					t.Errorf("%q: %v", form(k, v), found)
				}
			}
		}
		for _, k := range []string{"private_key", "secret_key", "signing_key", "encryption_key", "master_key", "api_key", "privateKey", "signingKey"} {
			for _, form := range forms {
				if len(signatureSources(t, srv, field(1, []byte(form(k, v))))) == 0 {
					t.Errorf("real secret %q: not blocked", form(k, v))
				}
			}
		}
	}
	vendor := formatSecrets(r)
	for _, k := range []string{"max_tokens", "shard_key", "public_key"} {
		for name, v := range vendor {
			if len(signatureSources(t, srv, field(1, []byte(k+": "+v)))) == 0 {
				t.Errorf("%s after %s: not blocked", name, k)
			}
		}
	}
}

func TestPluralListKeysBlockAlike(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	r := rand.New(rand.NewSource(1933))
	for _, n := range []int{16, 20, 24} {
		v := mixedSecret(r, n)
		for _, k := range []string{"keys", "tokens", "secrets", "passwords", "credentials", "api_keys", "Keys", "TOKENS"} {
			s := k + ":\n- " + v
			for name, raw := range map[string][]byte{"decoded text": []byte(s), "one field": field(1, []byte(s))} {
				if len(signatureSources(t, srv, raw)) == 0 {
					t.Errorf("%q (%d chars) as %s: not blocked", k+":\\n- ...", n, name)
				}
			}
		}
	}
}

var keyValueForms = map[string]func(k, v, w string) string{
	"colon":        func(k, v, w string) string { return k + ": " + v },
	"equals":       func(k, v, w string) string { return k + "=" + v },
	"json":         func(k, v, w string) string { return `{"` + k + `": "` + v + `"}` },
	"json array":   func(k, v, w string) string { return `{"` + k + `": ["` + v + `"]}` },
	"json array 2": func(k, v, w string) string { return `{"` + k + `": ["abc", "` + v + `"]}` },
	"json lines":   func(k, v, w string) string { return "{\"" + k + "\": [\n  \"" + w + "\",\n  \"" + v + "\"\n]}" },
	"nested":       func(k, v, w string) string { return `{"creds": {"` + k + `": ["` + v + `"]}}` },
	"yaml list":    func(k, v, w string) string { return k + ":\n  - abc\n  - " + w + "\n  - " + v + "\n" },
	"yaml flow":    func(k, v, w string) string { return k + ": [" + w + ", " + v + "]" },
}

func TestSecretKeysInEveryFormAndNumber(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	r := rand.New(rand.NewSource(1934))
	v, w := mixedSecret(r, 20), mixedSecret(r, 20)
	keys := []string{"api_key", "client_secret", "db_password", "auth_token", "secret", "token", "password", "key",
		"tokens", "secrets", "keys", "passwords", "client_secrets", "api_tokens", "access_tokens", "apiKeys", "api_keys", "credentials"}
	for _, k := range keys {
		for name, form := range keyValueForms {
			s := form(k, v, w)
			for where, raw := range map[string][]byte{"decoded text": []byte(s), "protobuf field": field(1, []byte(s))} {
				if len(signatureSources(t, srv, raw)) == 0 {
					t.Errorf("%s %s as %s: not blocked: %q", k, name, where, s)
				}
			}
		}
	}
	for _, k := range []string{"page_tokens", "max_tokens", "cache_keys", "input_tokens", "next_page_tokens", "pageTokens", "cacheKeys"} {
		for name, form := range keyValueForms {
			if found := signatureSources(t, srv, []byte(form(k, v, w))); len(found) > 0 {
				t.Errorf("exempt %s %s: %v", k, name, found)
			}
		}
	}
	if found := signatureSources(t, srv, []byte("input_tokens: 512")); len(found) > 0 {
		t.Errorf("input_tokens: 512: %v", found)
	}
}

func nestedSecretShapes(v string) map[string]string {
	deep := func(n int) string { return strings.Repeat("[", n) + `"` + v + `"` + strings.Repeat("]", n) }
	var long strings.Builder
	for i := 0; i < 120; i++ {
		long.WriteString(`"abcdefgh",`)
	}
	var yaml strings.Builder
	yaml.WriteString("api_keys:\n")
	for i := 0; i < 5000; i++ {
		yaml.WriteString("  - item\n")
	}
	yaml.WriteString("  - " + v + "\n")
	return map[string]string{
		"nested pair":             `{"api_key": [["abc", "` + v + `"]]}`,
		"nested single":           `{"api_key": [["` + v + `"]]}`,
		"plural nested":           `{"secrets": [["` + v + `"]]}`,
		"plural pair":             `{"api_keys": [["abc", "` + v + `"]]}`,
		"yaml flow nested":        "api_key: [[abc, " + v + "]]",
		"array of objects":        `{"api_keys": [{"name": "prod", "value": "` + v + `"}]}`,
		"two levels":              `{"credentials": {"prod": {"primary": [{"data": "` + v + `"}]}}}`,
		"depth 60":                `{"api_key": ` + deep(60) + `}`,
		"depth 100":               `{"api_key": ` + deep(100) + `}`,
		"long array":              `{"api_key": [` + long.String() + `"` + v + `"]}`,
		"yaml item 5000":          yaml.String(),
		"secrets equals list":     "secrets = [" + v + "]",
		"secret_keys quoted list": "secret_keys = ['" + v + "']",
		"tokens equals list":      "tokens=[abc, " + v + "]",
		"tokens spaced list":      `tokens = ["abc", "` + v + `"]`,
		"multiline objects":       "{\n  \"tokens\": [\n    {\"id\": 1, \"value\": \"" + v + "\"}\n  ]\n}",
	}
}

func TestSecretValuesNestedUnderSecretKeys(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	v := mixedSecret(rand.New(rand.NewSource(1970)), 20)
	for name, s := range nestedSecretShapes(v) {
		for where, raw := range map[string][]byte{"decoded text": []byte(s), "protobuf field": field(1, []byte(s))} {
			if len(signatureSources(t, srv, raw)) == 0 {
				t.Errorf("%s as %s: not blocked", name, where)
			}
		}
	}
	for _, s := range []string{`{"cache_keys": [["` + v + `"]]}`, `{"max_tokens": [["` + v + `"]]}`, `{"page_tokens": [{"value": "` + v + `"}]}`,
		"cache_keys:\n  - " + v + "\n", `{"api_keys": [{"name": "prod", "value": "short"}]}`, `{"notes": [["` + v + `"]]}`} {
		if found := signatureSources(t, srv, []byte(s)); len(found) > 0 {
			t.Errorf("%q: %v", s, found)
		}
	}
}
