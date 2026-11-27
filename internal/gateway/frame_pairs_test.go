package gateway

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/davidparkercodes/outworld/internal/config"
	"github.com/davidparkercodes/outworld/internal/scaletest"
)

func alnumSecret(r *rand.Rand, n int) string {
	const a = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = a[r.Intn(len(a))]
	}
	return string(b)
}

func controlBytes(r *rand.Rand, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(1 + r.Intn(8))
	}
	return b
}

func noisyBand(r *rand.Rand, parts ...[]byte) ([]byte, bool) {
	out := field(1, corpusEnglish(r, 500))
	for _, p := range parts {
		out = append(out, p...)
	}
	for k := 0; k < 200 && printableShare(out) >= 0.95; k++ {
		out = append(out, field(15, controlBytes(r, 6))...)
	}
	ps := printableShare(out)
	return out, parseFrame(out).framed && textual(out) && ps < cleanTextShare && ps >= textShare
}

func fixed64(num int, s string) []byte { return append([]byte{byte(num<<3 | 1)}, []byte(s)[:8]...) }

func nestObjects(n int, inner string) string {
	return strings.Repeat(`{"a": `, n) + inner + strings.Repeat("}", n)
}

func TestStructuredSecretsSurviveProseAndDepth(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	r := rand.New(rand.NewSource(2000))
	s := func() string { return alnumSecret(r, 32) }
	cases := map[string]string{
		"apostrophe before key":  fmt.Sprintf("Here's the yaml: api_keys: [%s, %s]", s(), s()),
		"quote after a word":     fmt.Sprintf("the note said 'hold on secrets:\n  prod: %s", s()),
		"apostrophe line before": fmt.Sprintf("Don't commit these.\ntokens:\n  - %s", s()),
		"unterminated quote":     fmt.Sprintf("it said 'hold on\napi_keys: [%s]", s()),
		"63 open brackets":       strings.Repeat("[", 63) + " " + fmt.Sprintf("api_keys: [%s, %s]", s(), s()),
		"64 open brackets":       strings.Repeat("[", 64) + " " + fmt.Sprintf("api_keys: [%s, %s]", s(), s()),
		"200 open brackets":      strings.Repeat("[", 200) + " " + fmt.Sprintf("api_keys: [%s]", s()),
		"nested 70 array":        nestObjects(70, fmt.Sprintf(`{"api_keys": ["%s"]}`, s())),
		"nested 70 object":       nestObjects(70, fmt.Sprintf(`{"secrets": {"prod": "%s"}}`, s())),
		"value 70 deep":          `{"secrets": ` + strings.Repeat("[", 70) + `"` + s() + `"` + strings.Repeat("]", 70) + `}`,
		"owner after deep":       `{"secrets": {"deep": ` + nestObjects(70, "1") + `, "prod": "` + s() + `"}}`,
		"yaml map":               fmt.Sprintf("secrets:\n  prod: %s\n  dev: %s", s(), s()),
		"equals array":           fmt.Sprintf(`secrets = ["%s"]`, s()),
		"long value":             "api_key: " + alnumSecret(r, 600),
		"nine space gap":         "api_key:" + strings.Repeat(" ", 9) + s(),
		"toml table":             fmt.Sprintf("[secrets]\nprod = \"%s\"", s()),
		"toml dotted table":      fmt.Sprintf("[env.api_keys]\nprimary = \"%s\"\nbackup = \"%s\"", s(), s()),
		"toml array table":       fmt.Sprintf("[[secrets]]\nname = \"prod\"\nvalue = \"%s\"", s()),
	}
	for _, text := range []string{fmt.Sprintf("[server]\nhost = \"%s\"", s()), fmt.Sprintf("[secrets]\nprod = \"%s\"\n[server]\nhost = \"%s\"", "short", s()), fmt.Sprintf("[cache_keys]\nmain = \"%s\"", s())} {
		if found := blobSources(t, srv, []byte(text)); len(found) > 0 {
			t.Errorf("%q: %v", text, found)
		}
	}
	for name, text := range cases {
		for _, wrap := range []string{"bare", "prose"} {
			body := text
			if wrap == "prose" {
				body = "the deploy notes for the service follow below and are long enough to be prose.\n" + text + "\nend of notes."
			}
			if len(blobSources(t, srv, []byte(body))) == 0 {
				t.Errorf("%s (%s): not blocked", name, wrap)
			}
		}
	}
}

func TestKeysAndValuesSplitAcrossFields(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	r := rand.New(rand.NewSource(2001))
	keys := []string{"password", "api_key", "secret", "token", "access_token", "OPENAI_API_KEY", "DB_PASSWORD", "client_secret", "apiKey", "auth_token"}
	shapes := map[string]func(k, v string) [][]byte{
		"straddle colon":  func(k, v string) [][]byte { return [][]byte{field(1, []byte(k+":")), field(1, []byte(v))} },
		"straddle equals": func(k, v string) [][]byte { return [][]byte{field(1, []byte(k+"=")), field(1, []byte(v))} },
		"map entry": func(k, v string) [][]byte {
			return [][]byte{field(5, append(field(1, []byte(k)), field(2, []byte(v))...))}
		},
		"map entry colon": func(k, v string) [][]byte {
			return [][]byte{field(5, append(field(1, []byte(k+":")), field(2, []byte(v))...))}
		},
		"binary payload": func(k, v string) [][]byte {
			return [][]byte{field(6, append([]byte(k+":\t"+v), controlBytes(r, 40)...))}
		},
		"binary payload newline": func(k, v string) [][]byte {
			return [][]byte{field(6, append([]byte(k+":\n"+v), controlBytes(r, 40)...))}
		},
	}
	for _, k := range keys {
		for name, build := range shapes {
			blob, band := noisyBand(r, build(k, alnumSecret(r, 32))...)
			if !band {
				t.Fatalf("%s/%s is not in the noisy band", name, k)
			}
			if len(blobSources(t, srv, blob)) == 0 {
				t.Errorf("%s/%s: not blocked", name, k)
			}
		}
	}
	for _, k8 := range []string{"api_key:", "api_key=", "secret: ", "token = "} {
		blob, _ := noisyBand(r, fixed64(1, k8), field(1, []byte(alnumSecret(r, 32))))
		if len(blobSources(t, srv, blob)) == 0 {
			t.Errorf("fixed64 %q then a field: not blocked", k8)
		}
	}
	for _, f := range []string{"api_keys:[%s,%s]", "{\"secrets\":{\"prod\":\"%s\",\"dev\":\"%s\"}}"} {
		blob, _ := noisyBand(r, field(6, []byte(fmt.Sprintf(f, alnumSecret(r, 32), alnumSecret(r, 32)))))
		if len(blobSources(t, srv, blob)) == 0 {
			t.Errorf("token-shaped struct %q in the noisy band: not blocked", f)
		}
	}
	for _, f := range []string{"api_keys: [%s, %s]", "{\"secrets\": {\"prod\": \"%s\", \"dev\": \"%s\"}}", "tokens:\n  - %s\n  - %s"} {
		v := fmt.Sprintf(f, alnumSecret(r, 32), alnumSecret(r, 32))
		for name, part := range map[string][]byte{"struct in field": field(6, []byte("the deploy notes say "+v+" for now")), "struct in binary": field(6, append([]byte(v), controlBytes(r, 40)...))} {
			blob, _ := noisyBand(r, part)
			if len(blobSources(t, srv, blob)) == 0 {
				t.Errorf("%s %q: not blocked", name, f)
			}
		}
	}
	for _, k := range []string{"tokens", "api_keys", "secrets"} {
		blob, _ := noisyBand(r, field(6, []byte(k+":")), field(6, []byte("- "+alnumSecret(r, 32))))
		if len(blobSources(t, srv, blob)) == 0 {
			t.Errorf("yaml list %s split by framing in the noisy band: not blocked", k)
		}
	}
	blob, _ := noisyBand(r, field(1, []byte("phone:")), field(1, []byte("8045551234")))
	if len(blobSources(t, srv, blob)) == 0 {
		t.Error("phone label then number in the next field: not blocked")
	}
	for _, neg := range [][][]byte{{field(1, []byte("time update by these token")), field(1, []byte(alnumSecret(r, 32)))}, {field(5, append(field(1, []byte("cache_key")), field(2, []byte(alnumSecret(r, 32)))...))}} {
		blob, _ := noisyBand(r, neg...)
		if found := blobSources(t, srv, blob); len(found) > 0 {
			t.Errorf("negative pair blocked: %v", found)
		}
	}
}

func cborTextItem(s string) []byte {
	switch {
	case len(s) < 24:
		return append([]byte{byte(0x60 | len(s))}, s...)
	case len(s) < 256:
		return append([]byte{0x78, byte(len(s))}, s...)
	}
	return append([]byte{0x79, byte(len(s) >> 8), byte(len(s))}, s...)
}

type cborEntry struct {
	k string
	v any
}

func cborAny(x any) []byte {
	switch x := x.(type) {
	case string:
		return cborTextItem(x)
	case []cborEntry:
		b := []byte{byte(0xa0 | len(x))}
		for _, e := range x {
			b = append(append(b, cborTextItem(e.k)...), cborAny(e.v)...)
		}
		return b
	case []any:
		b := []byte{byte(0x80 | len(x))}
		for _, e := range x {
			b = append(b, cborAny(e)...)
		}
		return b
	}
	panic("unsupported cbor value")
}

func repeatTo(unit []byte, n int) []byte {
	return bytes.Repeat(unit, n/len(unit)+1)
}

func lowerFiller(r *rand.Rand, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte('a' + r.Intn(26))
	}
	return string(b)
}

type cborKV struct{ k, v string }

func cborTextMap(kvs ...cborKV) []byte {
	b := []byte{byte(0xa0 | len(kvs))}
	for _, kv := range kvs {
		b = append(append(b, cborTextItem(kv.k)...), cborTextItem(kv.v)...)
	}
	return b
}

func protoFiller(r *rand.Rand, n int) []byte {
	var b []byte
	for len(b) < n {
		b = append(b, field(3+r.Intn(12), []byte(fmt.Sprintf("build step %d finished ok", r.Intn(1000))))...)
		b = append(b, varintField(3+r.Intn(12), uint64(r.Intn(1<<20)))...)
	}
	return b
}

func toolDataStatus(t *testing.T, url string, raw []byte) int {
	t.Helper()
	body := anthBody("hello", false)
	body["messages"] = append(body["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "tool_use", "id": "toolu_01AbCdEfGhIjKlMnOpQrStUv", "name": "f", "input": map[string]any{"data": base64.StdEncoding.EncodeToString(raw)}}}}, map[string]any{"role": "user", "content": "go"})
	resp, _ := postJSON(t, url+"/v1/messages", body, nil)
	return resp.StatusCode
}

func TestSecretKeysInCBORMaps(t *testing.T) {
	gw := newTestServer(t, "", &fnTransport{reply: echoReply})
	r := rand.New(rand.NewSource(68))
	v := "q8Zr2LmP4vX9tB7nK3wY6cH1dF5gJ0sA"
	long := "Wm3nQ7rT1yU5iO9pA2sD6fG8hJ4kL0zXc4Vb7Nm2Qw9Er5Ty"
	city := cborKV{"city", "Richmond"}
	nested := func(inner []byte) []byte { return append(append([]byte{0xa1}, cborTextItem("config")...), inner...) }
	array := func(items ...[]byte) []byte {
		return append([]byte{byte(0x80 | len(items))}, bytes.Join(items, nil)...)
	}
	positives := map[string][]byte{
		"key first":        bytes.Repeat(cborTextMap(cborKV{"api_key", v}, city), 30),
		"key alone":        bytes.Repeat(cborTextMap(cborKV{"api_key", v}), 30),
		"key last":         bytes.Repeat(cborTextMap(city, cborKV{"api_key", v}), 30),
		"key middle":       bytes.Repeat(cborTextMap(city, cborKV{"secret", v}, cborKV{"zip", "23220"}), 30),
		"secret first":     bytes.Repeat(cborTextMap(cborKV{"secret", v}, city), 30),
		"once with filler": append(cborTextMap(cborKV{"api_key", v}, city), protoFiller(r, 1500)...),
		"nested map":       bytes.Repeat(nested(cborTextMap(city, cborKV{"token", v})), 20),
		"array of maps":    bytes.Repeat(array(cborTextMap(city), cborTextMap(cborKV{"apiKey", v}), cborTextMap(city)), 15),
		"long value":       bytes.Repeat(cborTextMap(cborKV{"client_secret", long}, city), 20),
		"upper key":        bytes.Repeat(cborTextMap(cborKV{"API_KEY", v}, city), 30),
	}
	for name, raw := range positives {
		if code := toolDataStatus(t, gw.URL, raw); code != http.StatusUnprocessableEntity {
			t.Errorf("%s: %d, want 422", name, code)
		}
	}
	negatives := map[string][]byte{
		"plain cbor map":       bytes.Repeat(cborTextMap(city, cborKV{"street", "Main Street"}), 30),
		"secret key, no value": bytes.Repeat(cborTextMap(cborKV{"token", "not set"}, city), 30),
		"cache key":            bytes.Repeat(cborTextMap(cborKV{"cache_key", "users-by-region"}, city), 30),
		"bare word, plain":     append(append(protoFiller(r, 1500), append(field(1, []byte("token")), field(2, []byte("expired yesterday"))...)...), protoFiller(r, 1500)...),
		"filler only":          protoFiller(r, 3000),
	}
	for name, raw := range negatives {
		if code := toolDataStatus(t, gw.URL, raw); code != http.StatusOK {
			t.Errorf("negative %s: %d, want 200", name, code)
		}
	}
}

func TestCBORSecretCopiesStayBounded(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	r := rand.New(rand.NewSource(69))
	var overlap []byte
	for len(overlap) < 64<<10 {
		overlap = append(overlap, "etokenxz"+alnumSecret(r, 2)...)
	}
	if _, keyed, _ := cborPairs(overlap); len(keyed) > 0 {
		t.Errorf("text with no CBOR structure produced %d secret copies", len(keyed))
	}
	key := strings.Repeat("a", 56) + "_api_key"
	var heavy []byte
	for len(heavy) < 64<<10 {
		var tokens []string
		for range 13 {
			tokens = append(tokens, alnumSecret(r, 16))
		}
		heavy = append(heavy, 0xa1)
		heavy = append(heavy, cborTextItem(key)...)
		heavy = append(heavy, cborTextItem(strings.Join(tokens, " ")[:200])...)
	}
	_, keyed, _ := cborPairs(heavy)
	size, seen := 0, map[string]bool{}
	for _, c := range keyed {
		if !seen[c] {
			seen[c] = true
			size += len(c)
		}
	}
	if len(keyed) == 0 || size > keyedBytesFor(len(heavy)) {
		t.Errorf("anchored long-key pairs: %d copies, %d bytes for a %d byte value, budget %d", len(keyed), size, len(heavy), keyedBytesFor(len(heavy)))
	}
	f := &fields{}
	f.keyedLeaf(0, "x", []string{"token=" + alnumSecret(r, 32) + " ", "token=" + alnumSecret(r, 32) + " "}, 40)
	if len(f.leaves) != 1 || f.leaves[0].mode != modeOversize || f.leaves[0].text != oversizeKeyed {
		t.Errorf("copies over the byte budget were not refused: %+v", f.leaves)
	}
	small := bytes.Repeat(cborTextMap(cborKV{"api_key", "q8Zr2LmP4vX9tB7nK3wY6cH1dF5gJ0sA"}, cborKV{"city", "Richmond"}), 2000)
	g := srv.collect(opaqueTurn([]string{base64.StdEncoding.EncodeToString(small)}, []int{0}), config.APIAnthropic)
	for _, l := range g.leaves {
		if l.mode == modeOversize {
			t.Errorf("repeated ordinary pairs hit the copy budget at %s", l.path)
		}
	}
}

func TestSecretKeysInCBORMapsWithShortItems(t *testing.T) {
	gw := newTestServer(t, "", &fnTransport{reply: echoReply})
	v, v2 := "q8Zr2LmP4vX9tB7nK3wY6cH1dF5gJ0sA", "Wm3nQ7rT1yU5iO9pA2sD6fG8hJ4kL0zX"
	rows := map[string]any{
		"middle of 5, client_secret": []cborEntry{{"a", "x"}, {"b", "y"}, {"client_secret", v}, {"c", "z"}, {"d", "w"}},
		"middle of 5, api_key":       []cborEntry{{"a", "x"}, {"b", "y"}, {"api_key", v}, {"c", "z"}, {"d", "w"}},
		"middle of 3, api_key":       []cborEntry{{"a", "x"}, {"api_key", v}, {"c", "z"}},
		"array of maps token":        []any{[]cborEntry{{"name", "a"}, {"token", v}}, []cborEntry{{"name", "b"}, {"token", v2}}},
		"array of maps api_key":      []any{[]cborEntry{{"name", "a"}, {"api_key", v}}, []cborEntry{{"name", "b"}, {"api_key", v2}}},
		"array of maps, key first":   []any{[]cborEntry{{"api_key", v}, {"name", "a"}}},
		"name a then token":          []cborEntry{{"name", "a"}, {"token", v}},
		"key n then token":           []cborEntry{{"n", "Richmond"}, {"token", v}},
		"empty value then key":       []cborEntry{{"name", ""}, {"api_key", v}},
		"empty key then key":         []cborEntry{{"", "x"}, {"password", v}},
	}
	for name, x := range rows {
		if code := toolDataStatus(t, gw.URL, repeatTo(cborAny(x), 1500)); code != http.StatusUnprocessableEntity {
			t.Errorf("%s: %d, want 422", name, code)
		}
	}
	lengths := []int{0, 1, 2, 3, 23, 24, 255, 256}
	secrets := []cborEntry{{"token", v}, {"api_key", v}, {"password", v}, {"client_secret", v}}
	missed, total, posted := 0, 0, 0
	for _, seed := range []int64{1, 2, 3} {
		r := rand.New(rand.NewSource(seed))
		for n := 1; n <= 8; n++ {
			for pos := 0; pos < n; pos++ {
				for _, l := range lengths {
					secret := secrets[r.Intn(len(secrets))]
					flat := make([]cborEntry, 0, n)
					for i := 0; i < n; i++ {
						if i == pos {
							flat = append(flat, secret)
							continue
						}
						flat = append(flat, cborEntry{lowerFiller(r, l), lowerFiller(r, l)})
					}
					f := lowerFiller(r, l)
					shapes := map[string]any{
						"flat":   flat,
						"nested": []cborEntry{{"x", f}, {"cfg", flat}},
						"array":  []any{[]cborEntry{{"n", f}, secret}, flat, []cborEntry{{"n", f}}},
					}
					for shape, x := range shapes {
						total++
						blob := repeatTo(cborAny(x), 1500)
						_, keyed, complete := cborPairs(blob)
						if !complete || !strings.Contains(strings.Join(keyed, " "), "="+secret.v.(string)) {
							missed++
							t.Errorf("seed %d, %d pairs, secret at %d, filler length %d, %s: no copy", seed, n, pos, l, shape)
							continue
						}
						if seed == 1 && l == lengths[(n+pos)%len(lengths)] {
							posted++
							if code := toolDataStatus(t, gw.URL, blob); code != http.StatusUnprocessableEntity {
								t.Errorf("seed %d, %d pairs, secret at %d, filler length %d, %s: %d", seed, n, pos, l, shape, code)
							}
						}
					}
				}
			}
		}
	}
	t.Logf("generated maps copied %d of %d, %d posted end to end", total-missed, total, posted)
	clean := map[string]any{
		"plain map":        []cborEntry{{"city", "Richmond"}, {"state", "VA"}, {"build", "ok"}},
		"token not set":    []cborEntry{{"token", "not set"}, {"city", "Richmond"}},
		"cache key":        []cborEntry{{"cache_key", "users-by-region"}, {"city", "Richmond"}},
		"count keys":       []cborEntry{{"usage", "completion_tokens 12 prompt_tokens 30"}, {"a", "x"}},
		"short items only": []cborEntry{{"a", ""}, {"b", "y"}, {"", "n"}},
		"array of names":   []any{[]cborEntry{{"name", "a"}}, []cborEntry{{"name", "bb"}}},
	}
	for name, x := range clean {
		if code := toolDataStatus(t, gw.URL, repeatTo(cborAny(x), 1500)); code != http.StatusOK {
			t.Errorf("clean %s: %d, want 200", name, code)
		}
	}
}

func TestCBORMapWalkStaysBounded(t *testing.T) {
	r := rand.New(rand.NewSource(5))
	shapes := map[string]func(n int) []byte{
		"random":        func(n int) []byte { return uniformBytes(r, n) },
		"english":       func(n int) []byte { return corpusEnglish(r, n) },
		"all bf":        func(n int) []byte { return bytes.Repeat([]byte{0xbf}, n) },
		"all a1":        func(n int) []byte { return bytes.Repeat([]byte{0xa1}, n) },
		"bf every 2":    func(n int) []byte { return bytes.Repeat([]byte{0xbf, 0x00}, n/2) },
		"b7 ints":       func(n int) []byte { return bytes.Repeat(append([]byte{0xb7}, make([]byte, 45)...), n/46) },
		"hidden b7":     func(n int) []byte { return bytes.Repeat(append([]byte{0x50, 0xb7}, make([]byte, 15)...), n/17) },
		"one char maps": func(n int) []byte { return bytes.Repeat([]byte{0xa2, 0x61, 'a', 0x61, 'b', 0x60, 0x61, 'c'}, n/8) },
	}
	small, large := 128<<10, 512<<10
	if scaletest.Race {
		small, large = 8<<10, 32<<10
	}
	for name, gen := range shapes {
		sb, lb := gen(small), gen(large)
		if _, ok := cborMapPairs(lb); !ok {
			t.Errorf("%s: walk budget exhausted", name)
		}
		s := bestCPU(2, func() { cborMapPairs(sb) })
		l := bestCPU(2, func() { cborMapPairs(lb) })
		if !scaletest.Race && l > 8*s+50*time.Millisecond {
			t.Errorf("%s: 4x the input took %v after %v", name, l, s)
		}
	}
	hidden := bytes.Repeat(append([]byte{0x50, 0xbf}, make([]byte, 15)...), large/17)
	sb64 := hidden[:4096]
	if _, ok := cborMapPairs(hidden); ok {
		t.Error("indefinite maps hidden in byte strings did not hit the walk budget")
	}
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	srv.cborScan = func([]byte) ([]string, []string, bool) { return nil, nil, false }
	f := srv.collect(opaqueTurn([]string{base64.StdEncoding.EncodeToString(sb64)}, []int{0}), config.APIAnthropic)
	refused := false
	for _, l := range f.leaves {
		refused = refused || l.mode == modeOversize && l.text == oversizeKeyed
	}
	if !refused {
		t.Error("an exhausted walk was not refused")
	}
}
