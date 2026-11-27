package gateway

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func leakBody() map[string]any {
	return map[string]any{"model": "m", "messages": []any{
		map[string]any{"role": "system", "content": "You are helpful."},
		map[string]any{"role": "user", "name": "maria.garcia@corp.test", "content": "hello from John Smith"},
	}}
}

func TestLeakBlockExplainsItself(t *testing.T) {
	h := newHarness(t, "")
	resp := h.post(t, "/v1/chat/completions", leakBody(), nil)
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 422 || h.cloud.calls() != 0 || resp.Header.Get(HeaderRoute) != "blocked-leak" {
		t.Fatalf("status %d calls %d body %s", resp.StatusCode, h.cloud.calls(), b)
	}
	var out struct {
		Error struct {
			Message  string `json:"message"`
			Type     string `json:"type"`
			Code     string `json:"code"`
			Help     string `json:"help"`
			Findings []struct {
				Type    string `json:"type"`
				Source  string `json:"source"`
				Message *int   `json:"message_index"`
				Field   string `json:"field"`
				Start   *int   `json:"start"`
				End     *int   `json:"end"`
				Preview string `json:"preview"`
				Reason  string `json:"reason"`
				Hint    string `json:"hint"`
			} `json:"findings"`
		} `json:"error"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	e := out.Error
	if e.Code != "outworld_leak_blocked" || e.Type != "outworld_leak_blocked" || !strings.Contains(e.Message, "blocked outbound payload") || !strings.Contains(e.Help, HeaderDryRun) {
		t.Fatalf("error shape changed: %s", b)
	}
	if len(e.Findings) == 0 {
		t.Fatalf("no findings: %s", b)
	}
	sawEmail := false
	for _, f := range e.Findings {
		if f.Message == nil || *f.Message != 1 || f.Field != "messages[1].name" || f.Start == nil || f.End == nil || *f.End <= *f.Start {
			t.Fatalf("finding location: %+v in %s", f, b)
		}
		if f.Source == "" || f.Reason != "unmaskable_field" || f.Hint == "" || !strings.Contains(f.Preview, "***") {
			t.Fatalf("finding detail: %+v", f)
		}
		sawEmail = sawEmail || f.Type == "email"
	}
	if !sawEmail {
		t.Fatalf("expected an email finding: %s", b)
	}
	for _, real := range []string{"maria", "garcia", "corp.test", "John", "Smith"} {
		if strings.Contains(string(b), real) {
			t.Fatalf("422 body contains %q: %s", real, b)
		}
		if strings.Contains(h.logs.String(), real) {
			t.Fatalf("log contains %q: %s", real, h.logs.String())
		}
	}
	if strings.Contains(h.logs.String(), "***") || strings.Contains(h.logs.String(), "preview") {
		t.Fatalf("previews must not be logged: %s", h.logs.String())
	}
}

func TestAnthropicLeakBlockExplainsItself(t *testing.T) {
	h := newAnthHarness(t, "", "")
	body := map[string]any{"model": "m", "max_tokens": 16, "messages": []any{
		map[string]any{"role": "user", "content": []any{
			map[string]any{"type": "tool_use", "id": "t1", "name": "lookup", "input": map[string]any{"maria.garcia@corp.test": "x"}},
		}},
	}}
	resp, b := h.post(t, "/v1/messages", body, nil)
	assertAnthError(t, resp, b, 422, "outworld_leak_blocked")
	var out struct {
		Error struct {
			Findings []map[string]any `json:"findings"`
			Help     string           `json:"help"`
		} `json:"error"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Error.Findings) == 0 || out.Error.Help == "" {
		t.Fatalf("no findings: %s", b)
	}
	f := out.Error.Findings[0]
	if f["field"] != "messages[0].content[0].input.#0(key)" || f["message_index"] != float64(0) || f["hint"] == "" {
		t.Fatalf("finding: %v", f)
	}
	if strings.Contains(string(b), "maria") || strings.Contains(string(b), "corp.test") {
		t.Fatalf("422 body contains the value: %s", b)
	}
}

func decodeReport(t *testing.T, b []byte) DryRunReport {
	t.Helper()
	var rep DryRunReport
	if err := json.Unmarshal(b, &rep); err != nil {
		t.Fatalf("%v: %s", err, b)
	}
	return rep
}

func TestDryRunReturnsMaskedRequestWithoutCallingUpstream(t *testing.T) {
	h := newHarness(t, "")
	resp := h.post(t, "/v1/chat/completions", chatBody(userText, true), map[string]string{HeaderDryRun: "1"})
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || resp.Header.Get(HeaderDryRun) != "1" || resp.Header.Get(HeaderRoute) != "masked" {
		t.Fatalf("status %d headers %v body %s", resp.StatusCode, resp.Header, b)
	}
	if n := h.cloud.calls() + h.local.calls() + h.tee.calls(); n != 0 {
		t.Fatalf("dry run reached an upstream %d time(s)", n)
	}
	rep := decodeReport(t, b)
	if !rep.DryRun || !rep.WouldSend || rep.Blocked != "" || rep.Route != "masked" || rep.Upstream != "cloud" || len(rep.Findings) != 0 {
		t.Fatalf("report: %s", b)
	}
	for _, real := range realValues {
		if strings.Contains(string(b), real) {
			t.Fatalf("dry-run response contains real value %q: %s", real, b)
		}
		if strings.Contains(h.logs.String(), real) {
			t.Fatalf("log contains real value %q", real)
		}
	}
	if _, has := rep.Request["user"]; has {
		t.Fatal("the reported request must be what would be sent (user removed)")
	}
	msgs := rep.Request["messages"].([]any)
	sent := []rune(msgs[1].(map[string]any)["content"].(string))
	if len(rep.Replacements) < 6 || len(rep.Summary) == 0 {
		t.Fatalf("replacements: %s", b)
	}
	types := map[string]bool{}
	for _, r := range rep.Replacements {
		if r.Message != 1 || r.Field != "messages[1].content" || r.Source == "" || string(sent[r.Start:r.End]) != r.Replacement {
			t.Fatalf("replacement %+v does not match the masked request", r)
		}
		types[string(r.Type)] = true
		if strings.Contains(h.logs.String(), r.Replacement) && len(r.Replacement) > 3 {
			t.Fatalf("surrogate %q was logged", r.Replacement)
		}
	}
	for _, want := range []string{"person", "email", "phone", "date", "gov_id"} {
		if !types[want] {
			t.Fatalf("missing %s replacement: %s", want, b)
		}
	}
	if !strings.Contains(h.logs.String(), `"dry_run":true`) {
		t.Fatalf("dry run not marked in log: %s", h.logs.String())
	}

	resp = h.post(t, "/v1/chat/completions", chatBody(userText, false), nil)
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != 200 || h.cloud.calls() != 1 {
		t.Fatalf("a normal request after a dry run must go upstream: %d %d", resp.StatusCode, h.cloud.calls())
	}
	if got := h.cloud.lastUser(); got != string(sent) {
		t.Fatalf("dry run did not match what was sent:\n dry: %s\nsent: %s", string(sent), got)
	}
	resp = h.post(t, "/v1/chat/completions", chatBody(userText, false), map[string]string{HeaderDryRun: "0"})
	io.Copy(io.Discard, resp.Body)
	if h.cloud.calls() != 2 {
		t.Fatal("a false dry-run header must not turn dry run on")
	}
}

func TestDryRunShowsBlockFindings(t *testing.T) {
	h := newHarness(t, "")
	resp := h.post(t, "/v1/chat/completions", leakBody(), map[string]string{HeaderDryRun: "true"})
	b, _ := io.ReadAll(resp.Body)
	rep := decodeReport(t, b)
	if resp.StatusCode != 200 || resp.Header.Get(HeaderRoute) != "blocked-leak" || rep.WouldSend || rep.Blocked != "outworld_leak_blocked" || len(rep.Findings) == 0 {
		t.Fatalf("status %d report %s", resp.StatusCode, b)
	}
	if rep.Findings[0].Field != "messages[1].name" || rep.Findings[0].Hint == "" {
		t.Fatalf("finding: %+v", rep.Findings[0])
	}
	if h.cloud.calls() != 0 {
		t.Fatal("dry run reached the upstream")
	}
}

func TestDryRunOtherRoutes(t *testing.T) {
	h := newHarness(t, "")
	resp := h.post(t, "/v1/chat/completions", chatBody("this is top secret", false), map[string]string{HeaderDryRun: "1"})
	b, _ := io.ReadAll(resp.Body)
	rep := decodeReport(t, b)
	if resp.StatusCode != 200 || rep.WouldSend || rep.Blocked != "outworld_blocked" || rep.Request != nil {
		t.Fatalf("block route: %d %s", resp.StatusCode, b)
	}
	resp = h.post(t, "/v1/chat/completions", chatBody("my diagnosis for John Smith", false), map[string]string{HeaderDryRun: "1"})
	b, _ = io.ReadAll(resp.Body)
	rep = decodeReport(t, b)
	if !rep.WouldSend || rep.Route != "local" || len(rep.Replacements) != 0 || !strings.Contains(string(b), "John Smith") {
		t.Fatalf("local route: %s", b)
	}
	if n := h.cloud.calls() + h.local.calls() + h.tee.calls(); n != 0 {
		t.Fatalf("dry run reached an upstream %d time(s)", n)
	}
}

func TestDryRunRequiresGatewayToken(t *testing.T) {
	t.Setenv("GW_TOKEN", "s3cret-token")
	h := newHarness(t, "auth: {token_env: GW_TOKEN}\n")
	resp := h.post(t, "/v1/chat/completions", chatBody(userText, false), map[string]string{HeaderDryRun: "1"})
	if resp.StatusCode != 401 {
		t.Fatalf("dry run without the gateway token: %d", resp.StatusCode)
	}
}

func TestAnthropicDryRun(t *testing.T) {
	h := newAnthHarness(t, "", "")
	resp, b := h.post(t, "/v1/messages", anthBody(anthUserText, false), map[string]string{HeaderDryRun: "1"})
	rep := decodeReport(t, b)
	if resp.StatusCode != 200 || !rep.WouldSend || rep.API != "anthropic" || len(rep.Replacements) == 0 {
		t.Fatalf("status %d report %s", resp.StatusCode, b)
	}
	if h.claude.count() != 0 {
		t.Fatal("dry run reached the upstream")
	}
	assertNoReals(t, "dry-run report", string(b))
}
