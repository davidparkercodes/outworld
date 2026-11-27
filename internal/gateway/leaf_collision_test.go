package gateway

import (
	"encoding/base64"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/config"
)

const collisionCap = 16 << 10

var collisionPII = "SSN 123-45-6789 card 4111 1111 1111 1111 email maria.garcia@example.com"

func gatewayWithCap(t *testing.T, tr *fnTransport) (*Server, *httptest.Server) {
	t.Helper()
	srv := newTestGateway(t, "masked", "", tr)
	srv.cfg.MaxOpaqueBytes = collisionCap
	gw := httptest.NewServer(srv.Handler())
	t.Cleanup(gw.Close)
	return srv, gw
}

func binaryWithPII(r *rand.Rand, n int, mid string) []byte {
	b := uniformBytes(r, n/2)
	b[0] = 0
	b = append(b, "\x00\x01 "+mid+" \x00\x02"...)
	return append(b, uniformBytes(r, n-len(b))...)
}

func textWithPII(n int, mid string) []byte {
	filler := strings.Repeat("the build notes cover the release plan for the team. ", n/53+1)
	return []byte(filler[:n/2] + " " + mid + " " + filler[:n/2])
}

func anthropicPlacements(v string) map[string]map[string]any {
	out := map[string]map[string]any{}
	withAssistant := func(block map[string]any) map[string]any {
		b := anthBody("hello", false)
		b["messages"] = append(b["messages"].([]any), map[string]any{"role": "assistant", "content": []any{block}}, map[string]any{"role": "user", "content": "go"})
		return b
	}
	out["thinking signature"] = withAssistant(map[string]any{"type": "thinking", "thinking": "hmm", "signature": v})
	out["redacted thinking"] = withAssistant(map[string]any{"type": "redacted_thinking", "data": v})
	out["tool_use input signature"] = withAssistant(map[string]any{"type": "tool_use", "id": "toolu_01AbCdEfGhIjKlMnOpQrStUv", "name": "f", "input": map[string]any{"signature": v}})
	out["tool_use nested encrypted_content"] = withAssistant(map[string]any{"type": "tool_use", "id": "toolu_01AbCdEfGhIjKlMnOpQrStUv", "name": "f", "input": map[string]any{"x": map[string]any{"encrypted_content": v}}})
	top := anthBody("hello", false)
	top["metadata_extra"] = map[string]any{"signature": v}
	out["top-level unknown key"] = top
	out["unknown block"] = withAssistant(map[string]any{"type": "mystery_block", "encrypted_content": v})
	return out
}

func openAIPlacements(v string) map[string]map[string]any {
	out := map[string]map[string]any{}
	top := chatBody("hello", false)
	top["metadata_extra"] = map[string]any{"signature": v}
	out["top-level unknown key"] = top
	msg := chatBody("hello", false)
	msg["messages"] = append(msg["messages"].([]any), map[string]any{"role": "assistant", "content": "ok", "encrypted_content": v})
	out["unknown message field"] = msg
	return out
}

func TestOversizeOpaqueValueBlocksInEveryPlacement(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	_, gw := gatewayWithCap(t, tr)
	r := rand.New(rand.NewSource(1940))
	for _, n := range []int{collisionCap * 3 / 2, collisionCap / 2} {
		big := n > collisionCap
		for kind, raw := range map[string][]byte{"binary": binaryWithPII(r, n, collisionPII), "text": textWithPII(n, collisionPII), "clean": binaryWithPII(r, n, "nothing to see")} {
			v := base64.StdEncoding.EncodeToString(raw)
			for api, placements := range map[string]map[string]map[string]any{"/v1/messages": anthropicPlacements(v), "/v1/chat/completions": openAIPlacements(v)} {
				for where, body := range placements {
					resp, out := postJSON(t, gw.URL+api, body, nil)
					want := big || kind != "clean"
					if got := resp.StatusCode == http.StatusUnprocessableEntity; got != want {
						t.Errorf("%d bytes %s in %s %s: %d %.200s", n, kind, api, where, resp.StatusCode, out)
					}
					if big && !strings.Contains(string(out), "opaque_too_large") {
						t.Errorf("%d bytes %s in %s %s: no oversize finding: %.200s", n, kind, api, where, out)
					}
				}
			}
		}
	}
}

func TestLargePlainTextIsScannedNotRefused(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	_, gw := gatewayWithCap(t, tr)
	big, small := collisionCap*4, 2048
	bodies := func(text string) map[string]map[string]any {
		out := map[string]map[string]any{}
		out["message content"] = anthBody(text, false)
		sys := anthBody("hello", false)
		sys["system"] = text
		out["system prompt"] = sys
		tool := anthBody("hello", false)
		tool["messages"] = append(tool["messages"].([]any), map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "tool_use", "id": "toolu_01AbCdEfGhIjKlMnOpQrStUv", "name": "f", "input": map[string]any{"note": text}}}},
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "tool_result", "tool_use_id": "toolu_01AbCdEfGhIjKlMnOpQrStUv", "content": "done"}}})
		out["tool input text"] = tool
		top := anthBody("hello", false)
		top["metadata_extra"] = map[string]any{"note": text}
		out["unknown key text"] = top
		return out
	}
	for where := range bodies("x") {
		for _, pii := range []bool{false, true} {
			mid := "nothing personal here"
			if pii {
				mid = collisionPII
			}
			var statuses []int
			for _, n := range []int{small, big} {
				tr.mu.Lock()
				tr.calls = nil
				tr.mu.Unlock()
				resp, out := postJSON(t, gw.URL+"/v1/messages", bodies(string(textWithPII(n, mid)))[where], nil)
				statuses = append(statuses, resp.StatusCode)
				if strings.Contains(string(out), "opaque_too_large") {
					t.Errorf("%s with %d bytes of plain text was refused for size", where, n)
				}
				if resp.StatusCode == 200 {
					tr.mu.Lock()
					for _, c := range tr.calls {
						if strings.Contains(c.body, "maria.garcia@example.com") || strings.Contains(c.body, "123-45-6789") {
							t.Errorf("%s with %d bytes: personal data reached the upstream", where, n)
						}
					}
					tr.mu.Unlock()
				}
			}
			if statuses[0] != statuses[1] {
				t.Errorf("%s pii=%v: %d bytes gave %d, %d bytes gave %d", where, pii, small, statuses[0], big, statuses[1])
			}
			if !pii && statuses[1] != 200 {
				t.Errorf("%s: clean plain text of %d bytes gave %d", where, big, statuses[1])
			}
		}
	}
}

func leafAloneBlocks(t *testing.T, srv *Server, l leaf) bool {
	t.Helper()
	sess := srv.engine.NewSession("")
	sess.SetLocation(l.msg, l.path)
	if l.mode == modeMask {
		sess.Mask(t.Context(), l.text)
	} else if l.mode != modeRoute && l.mode != modeBinaryRoute {
		sess.Check(t.Context(), l.text)
	}
	narrow := map[loc]leaf{}
	if l.mode != modeMask && l.mode != modeRoute && l.mode != modeBinaryRoute {
		narrow[loc{l.msg, l.path}] = l
	}
	return len(keptViolations(sess.Violations(), narrow)) > 0
}

func TestLeafCollisionNeverLowersTheDecision(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	srv.cfg.MaxOpaqueBytes = collisionCap
	r := rand.New(rand.NewSource(1941))
	values := []func() string{
		func() string { return base64.StdEncoding.EncodeToString(binaryWithPII(r, 2048, collisionPII)) },
		func() string { return base64.StdEncoding.EncodeToString(binaryWithPII(r, collisionCap*3/2, "clean")) },
		func() string { return base64.StdEncoding.EncodeToString(textWithPII(2048, collisionPII)) },
		func() string { return base64.StdEncoding.EncodeToString(binaryWithPII(r, 2048, "clean")) },
		func() string { return "plain words " + collisionPII },
	}
	collided := 0
	for i := 0; i < 6*corpusScaleFloor(); i++ {
		v := values[i%len(values)]()
		for _, body := range anthropicPlacements(v) {
			f := srv.collect(body, config.APIAnthropic)
			groups := map[loc][]leaf{}
			for _, l := range f.leaves {
				at := loc{l.msg, baseField(l.path)}
				groups[at] = append(groups[at], l)
			}
			sess := srv.engine.NewSession("")
			narrow, err := srv.maskTwice(t.Context(), sess, body, config.APIAnthropic)
			if err != nil {
				t.Fatal(err)
			}
			blockedAt := map[loc]bool{}
			for _, v := range keptViolations(sess.Violations(), narrow) {
				blockedAt[loc{v.Message, v.Field}] = true
			}
			var keys []loc
			for at, ls := range groups {
				if len(ls) > 1 {
					keys = append(keys, at)
				}
			}
			sort.Slice(keys, func(a, b int) bool { return keys[a].path < keys[b].path })
			for _, at := range keys {
				collided++
				strictest := false
				for _, l := range groups[at] {
					strictest = strictest || leafAloneBlocks(t, srv, l)
				}
				if strictest && !blockedAt[at] {
					t.Errorf("%s: a leaf alone blocks but the request does not (%d leaves)", at.path, len(groups[at]))
				}
			}
		}
	}
	if collided == 0 {
		t.Fatal("no leaf collisions generated")
	}
}

func corpusScaleFloor() int { return max(1, 4/corpusScale) }

func TestCodeBuiltConfigKeepsTheDefaults(t *testing.T) {
	srv := &Server{cfg: &config.Config{}}
	r := rand.New(rand.NewSource(1990))
	big := base64.StdEncoding.EncodeToString(binaryWithPII(r, config.DefaultMaxOpaqueBytes+1024, "clean"))
	oversize := false
	for _, l := range srv.collect(anthropicPlacements(big)["thinking signature"], config.APIAnthropic).leaves {
		oversize = oversize || l.mode == modeOversize
	}
	if !oversize {
		t.Fatal("a zero max_opaque_bytes in a code-built config must still refuse an oversize opaque value")
	}
	small := base64.StdEncoding.EncodeToString(binaryWithPII(r, 2048, collisionPII))
	decoded := false
	for _, l := range srv.collect(encodedAnthropic(small)["tool_use input data"], config.APIAnthropic).leaves {
		decoded = decoded || strings.Contains(l.path, "(b64)")
	}
	if !decoded {
		t.Fatal("an empty encoded_values in a code-built config must still decode base64 in ordinary fields")
	}
}
