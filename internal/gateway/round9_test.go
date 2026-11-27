package gateway

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/config"
)

const base62 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func randTail(rng *rand.Rand, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = base62[rng.Intn(len(base62))]
	}
	return string(b)
}

var idPrefixes = []string{"toolu_", "srvtoolu_", "mcptoolu_", "msg_", "toolu_vrtx_", "toolu_bdrk_", "msg_vrtx_", "call_", "file_", "container_", "user_", "resp_",
	"fc_", "rs_", "item_", "req_", "run_", "thread_", "asst_", "batch_", "msgbatch_", "skill_", "ws_", "ctc_", "cs_", "mcp_", "srv_", "tool_", "chatcmpl-", "file-"}

func TestRandomProviderIDsAreAccepted(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(91))
	rejected := 0
	for _, p := range idPrefixes {
		for i := 0; i < 6000/corpusScale; i++ {
			id := p + randTail(rng, 16+rng.Intn(33))
			if !idShaped(id) {
				rejected++
				if rejected < 5 {
					t.Logf("rejected %s", id)
				}
			}
		}
	}
	if rejected != 0 {
		t.Fatalf("%d of %d random ids rejected", rejected, 6000/corpusScale*len(idPrefixes))
	}
}

func TestNameTails(t *testing.T) {
	for _, id := range []string{"call_Q9x7Zt2LpW4vN8kRm3", "file-b7Kq2Zx9Lm4Np8Wq", "msg_x93kQ2zL8pW7Tz5Yb"} {
		if !idShaped(id) {
			t.Fatalf("random id with name-like fragments rejected: %s", id)
		}
	}
	for _, id := range []string{"toolu_01JohnSmith12345678", "call_elenaPetrova19850314", "file-ELENAPETROVA1985031400", "msg_ElenaPetrova0000000000", "toolu_vrtx_JohnSmith1234567890"} {
		if idShaped(id) {
			t.Fatalf("name-bearing id accepted: %s", id)
		}
	}
}

func zlibEnglish(rng *rand.Rand, n int) []byte {
	words := strings.Fields("the quick brown fox jumps over a lazy dog while the build runs tests and reports results to the team every morning before lunch")
	var text strings.Builder
	for text.Len() < n*3 {
		text.WriteString(words[rng.Intn(len(words))])
		text.WriteByte(' ')
	}
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write([]byte(text.String()))
	w.Close()
	return buf.Bytes()
}

func aesLike(rng *rand.Rand, n int) []byte {
	key := make([]byte, 16)
	iv := make([]byte, 16)
	rng.Read(key)
	rng.Read(iv)
	blk, _ := aes.NewCipher(key)
	out := make([]byte, n)
	cipher.NewCTR(blk, iv).XORKeyStream(out, make([]byte, n))
	return out
}

func protobufLike(rng *rand.Rand, n int) []byte {
	var b []byte
	for len(b) < n {
		switch rng.Intn(4) {
		case 0:
			b = append(b, 0x08)
			b = binary.AppendUvarint(b, uint64(1700000000000+rng.Int63n(100000000000)))
		case 1:
			s := fmt.Sprintf("%d", 1700000000000000+rng.Int63n(100000000000000))[:13+rng.Intn(4)]
			b = append(b, 0x12, byte(len(s)))
			b = append(b, s...)
		case 2:
			now := int64(1700000000) + rng.Int63n(100000000)
			s := []string{fmt.Sprint(now), fmt.Sprint(now*1000 + rng.Int63n(1000)), fmt.Sprint(now*1000000 + rng.Int63n(1000000)), fmt.Sprint(rng.Int63n(1 << 40))}[rng.Intn(4)]
			b = append(b, 0x2a, byte(len(s)))
			b = append(b, s...)
			s = "session-" + randTail(rng, 10)
			b = append(b, 0x1a, byte(len(s)))
			b = append(b, s...)
		default:
			b = append(b, 0x20)
			b = binary.AppendUvarint(b, uint64(rng.Intn(1<<20)))
		}
	}
	return b
}

func opaqueTurn(blobs []string, positions []int) map[string]any {
	var content []any
	for i, v := range blobs {
		id := fmt.Sprintf("srvtoolu_%02dAbCdEfGh1JkLmNoPqRsT", i)
		switch positions[i] % 4 {
		case 0:
			content = append(content, map[string]any{"type": "thinking", "thinking": "hmm", "signature": v})
		case 1:
			content = append(content, map[string]any{"type": "redacted_thinking", "data": v})
		case 2:
			content = append(content, map[string]any{"type": "server_tool_use", "id": id, "name": "web_search", "input": map[string]any{"query": "q"}},
				map[string]any{"type": "web_search_tool_result", "tool_use_id": id, "content": []any{map[string]any{"type": "web_search_result", "title": "Guide", "url": "https://x.test", "encrypted_content": v}}})
		default:
			content = append(content, map[string]any{"type": "text", "text": "ok", "citations": []any{map[string]any{"type": "web_search_result_location", "url": "https://x.test", "cited_text": "x", "encrypted_index": v}}})
		}
	}
	return map[string]any{"model": "claude-x", "max_tokens": 16, "messages": []any{
		map[string]any{"role": "user", "content": "q"}, map[string]any{"role": "assistant", "content": content}, map[string]any{"role": "user", "content": "go"}}}
}

func TestRealisticBlobsHaveNoFindings(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	gens := map[string]func(*rand.Rand, int) []byte{
		"uniform":  func(r *rand.Rand, n int) []byte { b := make([]byte, n); r.Read(b); return b },
		"zlib":     zlibEnglish,
		"aes":      aesLike,
		"protobuf": protobufLike,
	}
	const shards, perShard, perRequest = 8, 250 / corpusScale, min(25, 250/corpusScale)
	for name, gen := range gens {
		for shard := 0; shard < shards; shard++ {
			t.Run(fmt.Sprintf("%s/%d", name, shard), func(t *testing.T) {
				t.Parallel()
				testBlobGenerator(t, srv, fmt.Sprintf("%s%d", name, shard), gen, perShard, perRequest)
			})
		}
	}
}

func testBlobGenerator(t *testing.T, srv *Server, name string, gen func(*rand.Rand, int) []byte, perGen, perRequest int) {
	{
		seed := int64(0)
		for _, c := range name {
			seed = seed*31 + int64(c)
		}
		rng := rand.New(rand.NewSource(seed))
		failures := 0
		for done := 0; done < perGen; done += perRequest {
			var blobs []string
			var pos []int
			for i := 0; i < perRequest; i++ {
				enc := base64.StdEncoding.EncodeToString
				if rng.Intn(3) == 0 {
					enc = base64.RawStdEncoding.EncodeToString
				}
				blobs = append(blobs, enc(gen(rng, 2048+rng.Intn(14<<10))))
				pos = append(pos, done+i)
			}
			sess := srv.engine.NewSession("")
			narrow, err := srv.maskTwice(context.Background(), sess, opaqueTurn(blobs, pos), config.APIAnthropic)
			if err != nil {
				t.Fatal(err)
			}
			if lerr := leakOf(sess, narrow); lerr != nil {
				failures++
				t.Logf("%s: %+v", name, lerr.Violations[0])
			}
		}
		if failures != 0 {
			t.Fatalf("%s: %d of %d batches had findings", name, failures, perGen/perRequest)
		}
	}
}

func blobAround(payload string) string {
	rng := rand.New(rand.NewSource(5))
	pre := make([]byte, 300)
	post := make([]byte, 300)
	rng.Read(pre)
	rng.Read(post)
	return base64.StdEncoding.EncodeToString(append(append(pre, payload...), post...))
}

func TestTextualProtobufEncryptedContent(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	rng := rand.New(rand.NewSource(17))
	var b []byte
	for len(b) < 3000 {
		s := "chunk " + randTail(rng, 40) + " of page text"
		b = append(b, 0x0a, byte(len(s)))
		b = append(b, s...)
	}
	enc := base64.StdEncoding.EncodeToString(b)
	resp, out := postJSON(t, gw.URL+"/v1/messages", searchTurn(webResult(map[string]any{"encrypted_content": enc}), nil), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("protobuf-like encrypted_content: %d %s", resp.StatusCode, out)
	}
}

func TestUnpaddedOpaqueValues(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, n := range []int{89, 120, 200, 301} {
		raw := randomBytes(fmt.Sprint("idx", n), n)
		for _, v := range []string{base64.RawStdEncoding.EncodeToString(raw), base64.RawURLEncoding.EncodeToString(raw)} {
			b := anthBody("hello", false)
			b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{
				map[string]any{"type": "text", "text": "ok", "citations": []any{map[string]any{"type": "web_search_result_location", "url": "https://x.test", "cited_text": "x", "encrypted_index": v}}},
				map[string]any{"type": "thinking", "thinking": "hmm", "signature": v}}}, map[string]any{"role": "user", "content": "go"})
			resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
			if resp.StatusCode != 200 {
				t.Fatalf("%d-char unpadded value: %d %s", len(v), resp.StatusCode, out)
			}
		}
	}
}

func TestNonASCIITextIsTextual(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	for _, text := range []string{
		"Паспорт 4510 123456, Maria Garcia, SSN 123-45-6789, родилась 1985-03-14",
		"护照号码 E12345678，姓名 Maria Garcia，电话 (804) 555-1234，出生日期 1985-03-14",
	} {
		if !textual([]byte(text)) {
			t.Fatalf("not textual: %s", text)
		}
		resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(base64.StdEncoding.EncodeToString([]byte(text))), nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("%s: %d %s", text, resp.StatusCode, out)
		}
	}
}

func TestRawHexInOpaqueFields(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	h := hex.EncodeToString([]byte("note " + probe4))
	cases := map[string]map[string]any{"redacted data": redactedTurn(h)}
	cases["encrypted_content"] = searchTurn(webResult(map[string]any{"encrypted_content": h}), nil)
	sigBody := anthBody("hello", false)
	sigBody["messages"] = append(sigBody["messages"].([]any), map[string]any{"role": "assistant", "content": []any{
		map[string]any{"type": "thinking", "thinking": "hmm", "signature": h}, map[string]any{"type": "text", "text": "ok", "citations": []any{
			map[string]any{"type": "web_search_result_location", "url": "https://x.test", "cited_text": "x", "encrypted_index": h}}}}}, map[string]any{"role": "user", "content": "go"})
	cases["signature and index"] = sigBody
	for name, b := range cases {
		resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("%s: %d %s", name, resp.StatusCode, out)
		}
	}
}
