package gateway

import (
	"context"
	"encoding/base64"
	"math/rand"
	"net/http"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/config"
)

var proseWords = strings.Fields("the build finished and the release notes describe several small fixes to the parser cache layer")

func randomThenProse(seed int64, head, size int) []byte {
	r := rand.New(rand.NewSource(seed))
	raw := make([]byte, head)
	r.Read(raw)
	var b strings.Builder
	for b.Len() < size {
		b.WriteString(proseWords[r.Intn(len(proseWords))])
		b.WriteByte(' ')
	}
	return append(raw, b.String()...)
}

func TestRandomThenProseBlobPasses(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, testInflateCaps, tr)
	for seed := int64(1); seed <= 2; seed++ {
		v := base64.StdEncoding.EncodeToString(randomThenProse(seed, 4608, 115000))
		for where, body := range infBodies(v) {
			resp, out := postJSON(t, gw.URL+"/v1/messages", body, nil)
			if resp.StatusCode != http.StatusOK || !strings.Contains(lastForwarded(tr), v) {
				t.Errorf("seed %d in %s: %d %.300s", seed, where, resp.StatusCode, out)
			}
		}
	}
}

func TestRandomThenProseSweep(t *testing.T) {
	srv := newTestGateway(t, "masked", testInflateCaps, &fnTransport{reply: echoReply})
	for _, gzipped := range []bool{false, true} {
		for _, size := range []int{2 << 10, 8 << 10, 32 << 10, 128 << 10, 512 << 10} {
			for seed := int64(0); seed < 4; seed++ {
				head := 4608
				if gzipped {
					head = 64 << 10
				}
				raw := randomThenProse(seed, head, size)
				if gzipped {
					raw = infGz(raw)
				}
				sess := srv.engine.NewSession("")
				narrow, err := srv.maskTwice(context.Background(), sess, opaqueTurn([]string{base64.StdEncoding.EncodeToString(raw)}, []int{0}), config.APIAnthropic)
				if err != nil {
					t.Fatal(err)
				}
				if lerr := leakOf(sess, narrow); lerr != nil {
					t.Errorf("gzip=%v size %d seed %d: %d findings, first %s %s", gzipped, size, seed, len(lerr.Violations), lerr.Violations[0].Type, lerr.Violations[0].Source)
				}
			}
		}
	}
}

func TestDecodedPersonNeedsProseLine(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	for _, text := range []string{"Please call Maria Gonzalez about the invoice today.", "Dear Mr Smith, thanks for the build notes."} {
		v := base64.StdEncoding.EncodeToString([]byte(text))
		sess := srv.engine.NewSession("")
		narrow, err := srv.maskTwice(context.Background(), sess, opaqueTurn([]string{v}, []int{0}), config.APIAnthropic)
		if err != nil {
			t.Fatal(err)
		}
		if leakOf(sess, narrow) == nil {
			t.Errorf("%q in decoded text: person not found", text)
		}
	}
}
