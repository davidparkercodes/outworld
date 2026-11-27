package gateway

import (
	"encoding/base64"
	"math/rand"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/davidparkercodes/outworld/internal/config"
	"github.com/davidparkercodes/outworld/internal/scaletest"
	"github.com/davidparkercodes/outworld/pkg/outworld"
)

func blobKinds() map[string]func(*rand.Rand, int) []byte {
	return map[string]func(*rand.Rand, int) []byte{
		"reviewer": reviewerProtobuf,
		"nonframe": nonFrameBytes,
		"random":   uniformBytes,
		"keys": func(r *rand.Rand, n int) []byte {
			return []byte(strings.Repeat("api_key auth_token secretKey ", n/29+1)[:n])
		},
		"framedtext": func(r *rand.Rand, n int) []byte {
			var b []byte
			for len(b) < n {
				b = append(b, field(1+r.Intn(15), []byte(minedSentence(r, false)))...)
			}
			return b
		},
		"cbor": func(r *rand.Rand, n int) []byte {
			var b []byte
			for len(b) < n {
				b = append(b, cborRecord(r, "phoneNumber", "id"+randTail(r, 8), nil, nil)...)
			}
			return b
		},
	}
}

func TestOpaqueBlobScanScalesLinearly(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	rounds := 3
	if scaletest.Race {
		rounds = 1
	}
	for name, gen := range blobKinds() {
		cost := func(size int) time.Duration {
			v := base64.StdEncoding.EncodeToString(gen(rand.New(rand.NewSource(7)), size))
			return bestCPU(rounds, func() { corpusFindingsByBlob(t, srv, []string{v}, []int{0}) })
		}
		base := 16 << 10
		if scaletest.Race {
			base = 4 << 10
		}
		small, large := cost(base), cost(4*base)
		if scaletest.Race {
			t.Logf("%s: 16 KiB %v, 64 KiB %v", name, small, large)
			continue
		}
		if large > 8*small+20*time.Millisecond {
			t.Errorf("%s: 4x the blob took %v after %v", name, large, small)
		}
	}
}

func TestOversizeOpaqueValueIsBlockedUnscanned(t *testing.T) {
	if config.DefaultMaxOpaqueBytes < 256<<10 {
		t.Fatalf("default max_opaque_bytes %d would block ordinary signatures", config.DefaultMaxOpaqueBytes)
	}
	tr := &fnTransport{reply: echoReply}
	srv := newTestGateway(t, "masked", "", tr)
	srv.cfg.MaxOpaqueBytes = 8 << 10
	r := rand.New(rand.NewSource(1930))
	big := base64.StdEncoding.EncodeToString(uniformBytes(r, 16<<10))
	small := base64.StdEncoding.EncodeToString(uniformBytes(r, 4<<10))
	sess := srv.engine.NewSession("")
	narrow, err := srv.maskTwice(t.Context(), sess, opaqueTurn([]string{big, small}, []int{0, 1}), config.APIAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	lerr := leakOf(sess, narrow)
	if lerr == nil || len(lerr.Violations) != 1 {
		t.Fatalf("want exactly one oversize finding, got %v", lerr)
	}
	v := lerr.Violations[0]
	if v.Source != "gateway:opaque_too_large" || v.Type != "oversize" || !strings.Contains(v.Hint, "max_opaque_bytes") || !strings.Contains(v.Field, "content[0]") {
		t.Fatalf("oversize finding: %+v", v)
	}
	gw := newTestServer(t, "", tr)
	resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(base64.StdEncoding.EncodeToString(uniformBytes(r, config.DefaultMaxOpaqueBytes+1))), nil)
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(out), "opaque_too_large") {
		t.Fatalf("over the default cap: %d %s", resp.StatusCode, out)
	}
	resp, out = postJSON(t, gw.URL+"/v1/messages", redactedTurn(base64.StdEncoding.EncodeToString(uniformBytes(r, 256<<10))), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("a 256 KiB random signature must pass: %d %.300s", resp.StatusCode, out)
	}
}

func TestRawIDWindowsCoverEveryShortSpan(t *testing.T) {
	s := strings.Repeat("abcdefghij", 500)
	f := &fields{frames: map[frameKey]frame{}, decoded: map[string][]byte{}}
	f.rawID(0, "p", s)
	if len(f.leaves) < 2 {
		t.Fatal("a long raw value must be split")
	}
	for start := 0; start+rawIDOverlap <= len(s); start++ {
		covered := false
		for _, l := range f.leaves {
			if len(l.text) > rawIDWindow {
				t.Fatalf("window of %d bytes", len(l.text))
			}
			if strings.Contains(l.text, s[start:start+rawIDOverlap]) {
				covered = true
				break
			}
		}
		if !covered {
			t.Fatalf("span at %d is in no window", start)
		}
	}
}

func TestFormatHiddenValueDeepInALongText(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	filler := strings.Repeat("the build notes cover the release plan for the team. ", 300)
	for _, hidden := range []string{"SSN\u200b123-45-6789", "Contact\u200bElena\u200bPetrova\u200btoday", "Contact\u2060Elena\u2060Petrova\u2060today", "4111\u200b1111\u200b1111\u200b1111"} {
		text := filler + hidden + " " + filler
		resp, out := postJSON(t, gw.URL+"/v1/messages", redactedTurn(base64.StdEncoding.EncodeToString([]byte(text))), nil)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("%q in the middle of %d bytes: %d %.200s", hidden, len(text), resp.StatusCode, out)
		}
	}
}

func TestKeptFindingsScaleWithLeafSize(t *testing.T) {
	cost := func(n int) time.Duration {
		text := strings.Repeat("abcdefgh ", n/9)
		l := leaf{msg: 1, path: "p", mode: modeDecoded, text: text}
		narrow := map[loc]leaf{{1, "p"}: l}
		var all []outworld.Violation
		for at := 0; at+8 < len(text); at += 40 {
			all = append(all, outworld.Violation{Type: "email", Source: "regex:email", Message: 1, Field: "p", Start: at, End: at + 8})
		}
		return bestCPU(3, func() { keptViolations(all, narrow) })
	}
	small, large := cost(64<<10), cost(256<<10)
	if scaletest.Race {
		t.Logf("64 KiB %v, 256 KiB %v", small, large)
		return
	}
	if large > 8*small+10*time.Millisecond {
		t.Fatalf("4x the leaf and its findings took %v after %v", large, small)
	}
}

const reviewerBlobCostLimit = 3.0

func TestReviewerBlobCostAgainstPlainText(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	r := rand.New(rand.NewSource(1931))
	const size = 4096
	n, rounds := 50, 2
	if scaletest.Race {
		n, rounds = 5, 1
	}
	var blobs []string
	for i := 0; i < n; i++ {
		blobs = append(blobs, base64.StdEncoding.EncodeToString(reviewerProtobuf(r, size)))
	}
	plain := string(corpusEnglish(r, size))
	text := bestCPU(rounds+1, func() {
		for i := 0; i < n; i++ {
			srv.engine.NewSession("").Check(t.Context(), plain)
		}
	})
	pos := make([]int, 25)
	for i := range pos {
		pos[i] = i
	}
	scan := bestCPU(rounds, func() {
		for i := 0; i < n; i += 25 {
			end := min(n, i+25)
			corpusFindingsByBlob(t, srv, blobs[i:end], pos[:end-i])
		}
	})
	ratio := float64(scan) / float64(max(text, time.Millisecond))
	if scaletest.Race {
		t.Logf("%d reviewer blobs of %d bytes: %v, plain text of the same size: %v, %.1fx", n, size, scan, text, ratio)
		return
	}
	t.Logf("blob scan %.1fx a plain-text check", ratio)
	if ratio > reviewerBlobCostLimit {
		t.Fatalf("%d reviewer blobs cost %v, %.1fx a plain-text check of the same size (%v), limit %.1fx", n, scan, ratio, text, reviewerBlobCostLimit)
	}
}

func TestWindowCutsNeverCreateFindings(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	head := strings.Repeat("7", 50) + "12345678Z " + strings.Repeat("x", formatWindow-10)
	text := head + "\u200b" + " and the rest of the notes for the team today"
	if found := sigSources(t, srv, []byte(text)); len(found) > 0 {
		t.Fatalf("a window cut created a finding: %v", found)
	}
	f := &fields{frames: map[frameKey]frame{}, decoded: map[string][]byte{}}
	f.rawID(0, "p", strings.Repeat("abcdefghij", 300))
	for i, l := range f.leaves {
		if want := (i > 0) && (i < len(f.leaves)-1); want != (len(l.cuts) == 2) {
			t.Fatalf("window %d of %d has cuts %v", i, len(f.leaves), l.cuts)
		}
	}
}

func TestNoisyFrameKeyCopiesStayOnOneLine(t *testing.T) {
	count := func(mode leafMode, text string) int {
		f := &fields{frames: map[frameKey]frame{}, decoded: map[string][]byte{}}
		f.keyLeaves(0, "p", mode, text)
		return len(f.leaves)
	}
	value := "abcdefghij1234567890Kq"
	for _, text := range []string{"notes token:\r" + value, "notes token:\t" + value, "notes key:\n" + value, "the phone\n8045551234"} {
		if n := count(modeNoisyFrame, text); n != 0 {
			t.Errorf("%q across frame glue in the noisy band: %d copies", text, n)
		}
		if n := count(modeDecoded, text); n == 0 && !strings.Contains(text, "\r") && !strings.Contains(text, "\t") {
			t.Errorf("%q in clean decoded text: no copy", text)
		}
	}
	for _, text := range []string{"api_key " + value, "phone_ext 8045551234", "notes key: " + value, "tokens=" + value, "tokens:\n- " + value, "api_keys: [" + value + "]"} {
		if n := count(modeNoisyFrame, text); n == 0 {
			t.Errorf("%q on one line or in a list in the noisy band: no copy", text)
		}
	}
}
