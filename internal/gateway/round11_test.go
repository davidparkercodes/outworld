package gateway

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/config"
)

var corpusWords = strings.Fields("the of and to in is for on that with as by this be are from at or an it not have which you was but can all your has more will one about new their its also other use may some these we time any only when what out most data system code file value request response model error test build server client user function type string return key token cache table index query result service config path version module package update release note")

func corpusEnglish(r *rand.Rand, n int) []byte {
	var b bytes.Buffer
	for b.Len() < n {
		b.WriteString(corpusWords[r.Intn(len(corpusWords))])
		if r.Intn(12) == 0 {
			b.WriteString(". ")
		} else {
			b.WriteByte(' ')
		}
	}
	return b.Bytes()[:n]
}

func reviewerProtobuf(r *rand.Rand, n int) []byte {
	var b []byte
	for len(b) < n {
		field := byte(1 + r.Intn(15))
		var s string
		switch r.Intn(9) {
		case 0:
			b = append(b, field<<3|0)
			b = binary.AppendUvarint(b, uint64(r.Int63n(1<<40)))
			continue
		case 1:
			b = append(b, field<<3|1)
			b = binary.LittleEndian.AppendUint64(b, r.Uint64())
			continue
		case 2:
			s = randTail(r, 8+r.Intn(32))
		case 3:
			s = fmt.Sprintf("%d", 1600000000+r.Int63n(200000000))
		case 4:
			s = fmt.Sprintf("%d", 1600000000000+r.Int63n(200000000000))
		case 5:
			s = fmt.Sprintf("%09d", r.Int63n(1000000000))
		case 6:
			s = fmt.Sprintf("%d", 1700000000000000+r.Int63n(100000000000000))
		case 7:
			x := make([]byte, 4+r.Intn(60))
			r.Read(x)
			s = string(x)
		case 8:
			s = string(corpusEnglish(r, 10+r.Intn(60)))
		}
		b = append(b, field<<3|2, byte(len(s)))
		b = append(b, s...)
	}
	return b
}

func separatorFormatted(r *rand.Rand, n int) []byte {
	var b []byte
	for len(b) < n {
		field := byte(1 + r.Intn(15))
		var s string
		switch r.Intn(8) {
		case 0:
			s = fmt.Sprintf("%d.%d.%d", r.Intn(20), r.Intn(40), r.Intn(200))
		case 1:
			s = fmt.Sprintf("%d.%d.%d.%d", r.Intn(256), r.Intn(256), r.Intn(256), r.Intn(256))
		case 2:
			s = fmt.Sprintf("%04d-%02d-%02d", 1990+r.Intn(40), 1+r.Intn(12), 1+r.Intn(28))
		case 3:
			s = fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02dZ", 2000+r.Intn(30), 1+r.Intn(12), 1+r.Intn(28), r.Intn(24), r.Intn(60), r.Intn(60))
		case 5:
			b = append(b, field<<3|1)
			b = binary.LittleEndian.AppendUint64(b, r.Uint64())
			continue
		case 6:
			x := make([]byte, 4+r.Intn(40))
			r.Read(x)
			s = string(x)
		default:
			s = fmt.Sprintf("v%d.%d.%d-rc.%d", r.Intn(5), r.Intn(30), r.Intn(100), r.Intn(9))
		}
		b = append(b, field<<3|2, byte(len(s)))
		b = append(b, s...)
	}
	return b
}

func uniformBytes(r *rand.Rand, n int) []byte {
	b := make([]byte, n)
	r.Read(b)
	return b
}

var corpusGenerators = map[string]func(*rand.Rand, int) []byte{
	"uniform":   uniformBytes,
	"zlib":      zlibEnglish,
	"aes":       aesLike,
	"protobuf":  reviewerProtobuf,
	"separator": separatorFormatted,
	"protobuf2": protobufLike,
}

var reviewerSeeds = []int64{8675309, 5550123, 909090, 20261123, 1123581321, 101010101, 424242, 13131313}

func TestCorpusIsBinaryAndClean(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	const perSeed, perRequest = 250 / corpusScale, min(25, 250/corpusScale)
	for name, gen := range corpusGenerators {
		for _, seed := range reviewerSeeds {
			name, gen, seed := name, gen, seed
			t.Run(fmt.Sprintf("%s/%d", name, seed), func(t *testing.T) {
				t.Parallel()
				r := rand.New(rand.NewSource(seed))
				bySource := map[string]int{}
				blocked := 0
				for done := 0; done < perSeed; done += perRequest {
					var blobs []string
					var pos []int
					for i := 0; i < perRequest; i++ {
						size := 16384
						if i%2 == 1 {
							size = 200 + r.Intn(1301)
						}
						enc := base64.StdEncoding.EncodeToString
						if r.Intn(3) == 0 {
							enc = base64.RawStdEncoding.EncodeToString
						}
						blobs = append(blobs, enc(gen(r, size)))
						pos = append(pos, done+i)
					}
					found, blobsHit := corpusFindingsByBlob(t, srv, blobs, pos)
					for src, n := range found {
						bySource[src] += n
					}
					blocked += blobsHit
				}
				t.Logf("%s seed %d findings by source: %v, blobs with findings %d of %d", name, seed, bySource, blocked, perSeed)
				if blocked > (blockedCeilings[name]*perSeed+249)/250 {
					t.Fatalf("%s seed %d: %d of %d blobs blocked, ceiling %d per 250", name, seed, blocked, perSeed, blockedCeilings[name])
				}
				ceilings := detectorCeilings[name]
				for src, n := range bySource {
					if n > ceilings[src] {
						t.Fatalf("%s seed %d: %d findings from %s, ceiling %d", name, seed, n, src, ceilings[src])
					}
				}
			})
		}
	}
}

var contentIndexRE = regexp.MustCompile(`^messages\[1\]\.content\[(\d+)\]`)

func corpusFindingsByBlob(t *testing.T, srv *Server, blobs []string, pos []int) (map[string]int, int) {
	t.Helper()
	sess := srv.engine.NewSession("")
	narrow, err := srv.maskTwice(context.Background(), sess, opaqueTurn(blobs, pos), config.APIAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]int{}
	hit := map[string]bool{}
	if lerr := leakOf(sess, narrow); lerr != nil {
		for _, v := range lerr.Violations {
			out[v.Source]++
			if m := contentIndexRE.FindStringSubmatch(v.Field); m != nil {
				hit[m[1]] = true
			} else {
				hit[v.Field] = true
			}
		}
	}
	return out, len(hit)
}

func corpusFindings(t *testing.T, srv *Server, blobs []string, pos []int) map[string]int {
	t.Helper()
	sess := srv.engine.NewSession("")
	narrow, err := srv.maskTwice(context.Background(), sess, opaqueTurn(blobs, pos), config.APIAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]int{}
	if lerr := leakOf(sess, narrow); lerr != nil {
		for _, v := range lerr.Violations {
			out[v.Source]++
		}
	}
	return out
}

func TestClassificationMargins(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewSource(10101))
	p := []byte(paragraph)
	every := func(k int, c byte) []byte {
		var out []byte
		for i, x := range p {
			if i > 0 && i%k == 0 {
				out = append(out, c)
			}
			out = append(out, x)
		}
		return out
	}
	sprinkle := func(share float64) []byte {
		var out []byte
		for _, c := range p {
			if r.Float64() < share {
				out = append(out, byte(r.Intn(256)))
			}
			out = append(out, c)
		}
		return out
	}
	for name, b := range map[string][]byte{
		"leading 0xff": append([]byte{0xff}, p...), "10% junk": sprinkle(0.1), "20% junk": sprinkle(0.2),
		"0xff every 6": every(6, 0xff), "NUL every 6": every(6, 0), "20% appended": append(append([]byte{}, p...), uniformBytes(r, len(p)/5)...),
	} {
		if !textual(b) {
			t.Fatalf("%s classified binary: printable %.3f letters %.3f", name, printableShare(b), letterShare(b))
		}
	}
	for name, gen := range corpusGenerators {
		for i := 0; i < 500/corpusScale; i++ {
			b := gen(r, 200+r.Intn(16000))
			if !parseFrame(b).framed && textual(b) {
				t.Fatalf("%s classified textual: printable %.3f letters %.3f", name, printableShare(b), letterShare(b))
			}
		}
	}
}

func toolStateBlob(r *rand.Rand, steps int) []byte {
	uuid := func() string {
		h := hex.EncodeToString(uniformBytes(r, 16))
		return h[:8] + "-" + h[8:12] + "-4" + h[13:16] + "-a" + h[17:20] + "-" + h[20:]
	}
	var st []map[string]any
	for i := 0; i < steps; i++ {
		st = append(st, map[string]any{
			"id": uuid(), "trace_id": hex.EncodeToString(uniformBytes(r, 16)), "user_id": fmt.Sprint(1700000000 + r.Intn(100000000)),
			"tool":       []string{"bash", "str_replace_based_edit_tool", "web_search", "code_execution"}[r.Intn(4)],
			"started_at": fmt.Sprintf("2026-%02d-%02dT%02d:%02d:%02d.%03dZ", 1+r.Intn(12), 1+r.Intn(28), r.Intn(24), r.Intn(60), r.Intn(60), r.Intn(1000)),
			"exit_code":  r.Intn(3), "duration_ms": r.Intn(120000), "seq": i,
		})
	}
	b, _ := json.Marshal(map[string]any{"schema": "tool-state/v3", "version": fmt.Sprintf("%d.%d.%d", r.Intn(5), r.Intn(40), r.Intn(200)),
		"session": uuid(), "created_at": "2026-03-01T10:00:00Z", "epoch_ms": 1700000000000 + r.Int63n(100000000000), "id": 1732345678, "steps": st})
	return b
}

func TestJSONToolStateBlobs(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	const shards, perShard, perRequest = 8, 250 / corpusScale, min(25, 250/corpusScale)
	for shard := 0; shard < shards; shard++ {
		seed := int64(909090 + shard)
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			t.Parallel()
			r := rand.New(rand.NewSource(seed))
			for done := 0; done < perShard; done += perRequest {
				var blobs []string
				var pos []int
				for i := 0; i < perRequest; i++ {
					blobs = append(blobs, base64.StdEncoding.EncodeToString(toolStateBlob(r, 1+r.Intn(20))))
					pos = append(pos, done+i)
				}
				sess := srv.engine.NewSession("")
				narrow, err := srv.maskTwice(context.Background(), sess, opaqueTurn(blobs, pos), config.APIAnthropic)
				if err != nil {
					t.Fatal(err)
				}
				if lerr := leakOf(sess, narrow); lerr != nil {
					t.Fatalf("tool state seed %d: %+v", seed, lerr.Violations[0])
				}
			}
		})
	}
}

func TestTextualCuesAndNegatives(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	wrap := func(phrase string) string {
		return base64.StdEncoding.EncodeToString([]byte("Notes from the meeting today. The client said " + phrase + " and asked for a follow up next week."))
	}
	for _, phrase := range []string{"reach me at 8045551234", "fax 8045551234", "acct 12345678901", "routing 021000021", "member 123456789", "text me at 8045551234",
		"my SSN is 123456789", "call me on 8045551234", "phone 8045551234", "passport number 123456789", "account 12345678901"} {
		for where, b := range allOpaquePositions(wrap(phrase)) {
			resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("%q in %s: %d %s", phrase, where, resp.StatusCode, out)
			}
		}
	}
	for _, phrase := range []string{"the hotel 8045551234 desk", "telemetry 8045551234 points", "cellar 8045551234 rows", "phoneme 8045551234 count", "Hôtel 8045551234 room",
		"order id 8045551234", "build number 123456789", "at 2026-03-14T10:00:00Z the job ran", `the payload {"id": 1732345678} arrived`} {
		for where, b := range allOpaquePositions(wrap(phrase)) {
			resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
			if resp.StatusCode != 200 {
				t.Fatalf("negative %q in %s: %d %s", phrase, where, resp.StatusCode, out)
			}
		}
	}
	for _, phrase := range []string{"born 1985-03-14", "date of birth 03/14/1985", "DOB: 1985-03-14"} {
		resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(wrap(phrase)), nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("cued date %q: %d %s", phrase, resp.StatusCode, out)
		}
	}
}

func TestNestedAndHexTextual(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	text := []byte("note " + probe4)
	for name, v := range map[string]string{
		"b64(b64)": base64.StdEncoding.EncodeToString([]byte(base64.StdEncoding.EncodeToString(text))),
		"b64(hex)": base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(text))),
		"raw hex":  hex.EncodeToString(text),
	} {
		for where, b := range allOpaquePositions(v) {
			resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("%s in %s: %d %s", name, where, resp.StatusCode, out)
			}
		}
	}
	for _, v := range []string{hex.EncodeToString([]byte("Elena Petrova")), strings.ToUpper(hex.EncodeToString([]byte("Elena Petrova"))),
		hex.EncodeToString([]byte("ElenaPetrova")), strings.ToUpper(hex.EncodeToString([]byte("ElenaPetrova"))), strings.ToUpper(hex.EncodeToString([]byte("Elena Petrova!")))} {
		for _, key := range []string{"safety_identifier", "prompt_cache_key"} {
			b := chatBody("hello", false)
			b[key] = v
			resp, out := postJSON(t, gw.URL+"/v1/chat/completions", b, nil)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("%s=%s: %d %s", key, v, resp.StatusCode, out)
			}
		}
		ab := anthBody("hello", false)
		ab["metadata"] = map[string]any{"user_id": v}
		gwk := newTestServer(t, "  keep_metadata: true\n", tr)
		resp, out := postJSON(t, gwk.URL+"/v1/messages", ab, nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("metadata.user_id=%s: %d %s", v, resp.StatusCode, out)
		}
	}
}

func TestRound11IDTails(t *testing.T) {
	for _, id := range []string{"toolu_01ELENAPETROVA9xQ2mZr7", "toolu_01ElenaPetrovaqz", "toolu_01ElenaPetrovaXk3", "toolu_01JohnSmith7kQ2mZx9Lp3"} {
		if idShaped(id) {
			t.Fatalf("name-bearing id accepted: %s", id)
		}
	}
	for _, id := range []string{"toolu_01RoseGoldXk3n9QzL2pR7", "call_WillPowerQ9x7Zt2LpW4v", "msg_01HopeFaith3kQ2zL8pW7T", "file-JadeStoneb7Kq2Zx9Lm4N"} {
		if !idShaped(id) {
			t.Fatalf("common-word pair rejected: %s", id)
		}
	}
}

func TestBinaryKeepsBlockTermsAndVendorKeys(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, blockRules, tr)
	resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(binaryWith("topsecret")), nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("block term in binary: %d %s", resp.StatusCode, out)
	}
	key := "sk-ant-api03-" + strings.Repeat("Ab3dE5gH7jK9mN1pQ3sT5vW7yZ", 3) + "-AbCdEfGh"
	for name, v := range map[string]string{
		"base64":     blobAround(" " + key + " "),
		"hex":        hex.EncodeToString(append(append(uniformBytes(rand.New(rand.NewSource(3)), 120), []byte(" "+key+" ")...), uniformBytes(rand.New(rand.NewSource(4)), 120)...)),
		"rawurl":     base64.RawURLEncoding.EncodeToString(append(append(uniformBytes(rand.New(rand.NewSource(5)), 120), []byte(" "+key+" ")...), uniformBytes(rand.New(rand.NewSource(6)), 120)...)),
		"b64 of b64": base64.StdEncoding.EncodeToString([]byte(base64.StdEncoding.EncodeToString(append(append(uniformBytes(rand.New(rand.NewSource(7)), 120), []byte(" "+key+" ")...), uniformBytes(rand.New(rand.NewSource(8)), 120)...)))),
	} {
		resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(v), nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("vendor key in binary %s: %d %s", name, resp.StatusCode, out)
		}
	}
}

func TestKnownGapPersonalDataInUnframedBinaryBytes(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, payload := range []string{"a804-555-1234b", "Elena Petrova born 03/14/1985"} {
		v := blobAround(payload)
		for where, b := range allOpaquePositions(v) {
			resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
			if resp.StatusCode != 200 {
				t.Fatalf("known gap changed: %q in %s: %d %s", payload, where, resp.StatusCode, out)
			}
		}
	}
	for _, payload := range []string{"x123-45-6789y", "pay 4111 1111 1111 1111 now", "xmaria.garcia@example.com7z"} {
		v := blobAround(payload)
		for where, b := range allOpaquePositions(v) {
			resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("closed gap reopened: %q in %s: %d %s", payload, where, resp.StatusCode, out)
			}
		}
	}
	rgw := newTestServer(t, "  rules:\n    - {name: secret, match_any: [\"topsecret\"], route: block}\n", tr)
	resp, out := postJSON(t, rgw.URL+"/v1/messages", redactedTurn(blobAround("x123-45-6789 topsecret y")), nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("block rule must still fire in binary: %d %s", resp.StatusCode, out)
	}
}

var detectorCeilings = map[string]map[string]int{
	"protobuf": {"names:handle": 5, "regex:assigned_secret": 2, "regex:context_token": 2, "regex:phone_intl": 2, "regex:phone_nanp": 2, "regex:email": 2},
}

var blockedCeilings = map[string]int{"protobuf": 5}
