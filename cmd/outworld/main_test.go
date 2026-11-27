package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/outworld"
)

func setup(t *testing.T) string {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "people.csv"), []byte("value,type,aliases\nJohn Smith,person,\n"), 0o600)
	cfg := filepath.Join(dir, "outworld.yaml")
	os.WriteFile(cfg, []byte("vault: {path: ./v.vault}\npolicy: {dictionary: ./people.csv}\n"), 0o600)
	t.Setenv("OUTWORLD_VAULT_KEY", strings.Repeat("cd", 32))
	return cfg
}

func TestMaskUnmaskCLI(t *testing.T) {
	cfg := setup(t)
	in := "John Smith (john@corp.test) met on 2024-03-15. OTP code 991234."
	var masked bytes.Buffer
	if err := maskCmd([]string{"--config", cfg}, strings.NewReader(in), &masked); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(masked.String(), "John") || strings.Contains(masked.String(), "2024-03-15") || strings.Contains(masked.String(), "991234") {
		t.Fatalf("masked: %s", masked.String())
	}
	var back bytes.Buffer
	if err := unmaskCmd([]string{"--config", cfg}, strings.NewReader(masked.String()), &back); err != nil {
		t.Fatal(err)
	}
	if back.String() != "John Smith (john@corp.test) met on 2024-03-15. OTP code [redacted code]." {
		t.Fatalf("unmasked: %s", back.String())
	}
	var other bytes.Buffer
	unmaskCmd([]string{"--config", cfg, "--scope", "elsewhere"}, strings.NewReader(masked.String()), &other)
	if other.String() == back.String() {
		t.Fatal("a different scope must use a different date offset")
	}
}

func TestEnvConfigAndMissingKey(t *testing.T) {
	cfg := setup(t)
	t.Setenv("OUTWORLD_CONFIG", cfg)
	var out bytes.Buffer
	if err := maskCmd(nil, strings.NewReader("hi John"), &out); err != nil || strings.Contains(out.String(), "John") {
		t.Fatalf("%v %q", err, out.String())
	}
	t.Setenv("OUTWORLD_VAULT_KEY", "")
	if err := maskCmd(nil, strings.NewReader("hi"), &out); err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("expected refusal without key, got %v", err)
	}
	if !errors.Is(&outworld.LeakError{}, outworld.ErrLeak) {
		t.Fatal("leak error must match ErrLeak for exit code 3")
	}
}

func TestDryRunCLI(t *testing.T) {
	cfg := setup(t)
	var out bytes.Buffer
	in := "John Smith (john@corp.test) met on 2024-03-15. OTP code 991234."
	if err := dryRunCmd([]string{"--config", cfg}, strings.NewReader(in), &out); err != nil {
		t.Fatal(err)
	}
	var rep struct {
		DryRun    bool `json:"outworld_dry_run"`
		WouldSend bool `json:"would_send"`
		Request   struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		} `json:"request"`
		Replacements []struct {
			Type        string `json:"type"`
			Source      string `json:"source"`
			Replacement string `json:"replacement"`
		} `json:"replacements"`
	}
	if err := json.Unmarshal(out.Bytes(), &rep); err != nil {
		t.Fatalf("%v: %s", err, out.String())
	}
	if !rep.DryRun || !rep.WouldSend || len(rep.Request.Messages) != 1 || len(rep.Replacements) < 3 {
		t.Fatalf("report: %s", out.String())
	}
	for _, real := range []string{"John", "Smith", "john@corp.test", "2024-03-15", "991234"} {
		if strings.Contains(out.String(), real) {
			t.Fatalf("report contains %q: %s", real, out.String())
		}
	}

	file := filepath.Join(t.TempDir(), "req.json")
	os.WriteFile(file, []byte(`{"model":"m","messages":[{"role":"user","name":"john@corp.test","content":"hello"}]}`), 0o600)
	out.Reset()
	err := dryRunCmd([]string{"--config", cfg, file}, strings.NewReader(""), &out)
	if !errors.Is(err, outworld.ErrLeak) {
		t.Fatalf("a blocked dry run must return the leak error for exit code 3, got %v", err)
	}
	if !strings.Contains(out.String(), `"blocked": "outworld_leak_blocked"`) || !strings.Contains(out.String(), `"hint"`) || !strings.Contains(out.String(), `"preview": "[***]"`) {
		t.Fatalf("report: %s", out.String())
	}
}

func TestMaskCLIMatchesGatewayDryRun(t *testing.T) {
	a := setup(t)
	var sink bytes.Buffer
	if err := maskCmd([]string{"--config", a}, strings.NewReader(""), &sink); err != nil {
		t.Fatal(err)
	}
	b := filepath.Join(t.TempDir(), "outworld.yaml")
	for _, f := range []string{"outworld.yaml", "people.csv", "v.vault"} {
		data, err := os.ReadFile(filepath.Join(filepath.Dir(a), f))
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(filepath.Dir(b), f), data, 0o600)
	}
	in := "Smith signed off. Later, Dana Whitfield wrote to John Smith and Dana on 2024-03-15 from dana.whitfield@corp.test."
	var rep bytes.Buffer
	if err := dryRunCmd([]string{"--config", a}, strings.NewReader(in), &rep); err != nil {
		t.Fatal(err)
	}
	var r struct {
		Request struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		} `json:"request"`
	}
	if err := json.Unmarshal(rep.Bytes(), &r); err != nil || len(r.Request.Messages) != 1 {
		t.Fatalf("%v: %s", err, rep.String())
	}
	var masked bytes.Buffer
	if err := maskCmd([]string{"--config", b}, strings.NewReader(in), &masked); err != nil {
		t.Fatal(err)
	}
	if masked.String() != r.Request.Messages[0].Content {
		t.Fatalf("cli mask differs from gateway:\ncli: %s\ngw:  %s", masked.String(), r.Request.Messages[0].Content)
	}
}

func TestErrorLinePrefixOnce(t *testing.T) {
	leak := &outworld.LeakError{Violations: []outworld.Violation{{Type: "email"}}}
	if got := errorLine(leak); strings.HasPrefix(got, "outworld: outworld:") || !strings.HasPrefix(got, "outworld: blocked") {
		t.Fatalf("leak line: %q", got)
	}
	if got := errorLine(errors.New("listen: busy")); got != "outworld: listen: busy" {
		t.Fatalf("plain line: %q", got)
	}
}

func TestMaskCLIDedupesFindings(t *testing.T) {
	cfg := setup(t)
	file := filepath.Join(t.TempDir(), "req.json")
	os.WriteFile(file, []byte(`{"model":"m","messages":[{"role":"user","name":"john@corp.test","content":"mail john@corp.test"}]}`), 0o600)
	var out bytes.Buffer
	err := dryRunCmd([]string{"--config", cfg, file}, strings.NewReader(""), &out)
	var le *outworld.LeakError
	if !errors.As(err, &le) {
		t.Fatalf("expected leak error, got %v", err)
	}
	seen := map[string]bool{}
	for _, v := range le.Violations {
		k := v.Field + ":" + strconv.Itoa(v.Start) + ":" + strconv.Itoa(v.End)
		if seen[k] {
			t.Fatalf("duplicate finding %s: %s", k, out.String())
		}
		seen[k] = true
	}
	if !strings.Contains(err.Error(), "1 sensitive value(s)") {
		t.Fatalf("count: %v %+v", err, le.Violations)
	}
}
