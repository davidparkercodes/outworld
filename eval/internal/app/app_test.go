package app

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fakeGateway(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "fake-outworld")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/fake-outworld")
	cmd.Dir = "../.."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fake gateway: %v\n%s", err, out)
	}
	return bin
}

func opts(t *testing.T, bin string) Options {
	return Options{
		CorpusPath: "../../corpus/corpus.json", PeoplePath: "../../corpus/people.csv", GatewayBin: bin,
		OutDir: t.TempDir(), Configs: []string{"dictionary"}, Modes: []string{"nonstream", "stream"},
		MaxLeak: 0.05, MaxErrorRate: 0, Limit: 25, Model: "fake-model", Timeout: 30 * time.Second,
	}
}

func TestEndToEndPassthroughLeaksEverything(t *testing.T) {
	testPassthrough(t, "openai")
}

func TestEndToEndPassthroughLeaksEverythingAnthropic(t *testing.T) {
	testPassthrough(t, "anthropic")
}

func TestEndToEndOracleLeaksNothing(t *testing.T) {
	testOracle(t, "openai")
}

func TestEndToEndOracleLeaksNothingAnthropic(t *testing.T) {
	testOracle(t, "anthropic")
}

func testPassthrough(t *testing.T, api string) {
	o := opts(t, fakeGateway(t))
	o.APIs = []string{api}
	rep, err := Execute(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Runs[0].API != api || rep.Runs[0].Label != "dictionary" {
		t.Fatalf("run labelled %q api %q", rep.Runs[0].Label, rep.Runs[0].API)
	}
	s := rep.Runs[0].Summary
	if s.Errors != 0 || s.Leak.N == 0 || s.Leak.Rate != 1 {
		t.Fatalf("passthrough should leak every sensitive value: %+v", s.Leak)
	}
	if rep.Gate.Passed {
		t.Fatal("gate should fail on a passthrough gateway")
	}
	if s.OverMask.Hits != 0 {
		t.Fatalf("passthrough cannot over-mask: %+v", s.OverMask)
	}
}

func testOracle(t *testing.T, api string) {
	bin := fakeGateway(t)
	abs, _ := filepath.Abs("../../corpus/corpus.json")
	t.Setenv("FAKE_OUTWORLD_MODE", "oracle")
	t.Setenv("FAKE_OUTWORLD_CORPUS", abs)
	o := opts(t, bin)
	o.APIs = []string{api}
	rep, err := Execute(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	s := rep.Runs[0].Summary
	if s.Errors != 0 || s.Leak.Hits != 0 || s.Restore.Rate != 1 || s.ResidueTokens != 0 {
		t.Fatalf("oracle should leak nothing and restore everything: leak=%+v restore=%+v residue=%d errors=%v", s.Leak, s.Restore, s.ResidueTokens, s.ErrorSamples)
	}
	if !rep.Gate.Passed {
		t.Fatalf("gate should pass: %v", rep.Gate.Reasons)
	}
}

func TestEndToEndMatrix(t *testing.T) {
	bin := fakeGateway(t)
	abs, _ := filepath.Abs("../../corpus/corpus.json")
	t.Setenv("FAKE_OUTWORLD_MODE", "oracle")
	t.Setenv("FAKE_OUTWORLD_CORPUS", abs)
	o := opts(t, bin)
	o.Limit = 5
	o.APIs = []string{"openai", "anthropic"}
	o.Configs = []string{"regex-only", "dictionary"}
	rep, err := Execute(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	var labels []string
	for _, r := range rep.Runs {
		labels = append(labels, r.Label)
		if r.Summary.Errors != 0 || r.Summary.Leak.Hits != 0 || r.Summary.Restore.Rate != 1 {
			t.Fatalf("%s: leak=%+v restore=%+v errors=%v", r.Label, r.Summary.Leak, r.Summary.Restore, r.Summary.ErrorSamples)
		}
		if !strings.Contains(r.Config, "memo: {enabled: false}") {
			t.Fatalf("%s: memo should be off by default:\n%s", r.Label, r.Config)
		}
		if (r.API == "anthropic") != strings.Contains(r.Config, "anthropic: {masked_upstream: cloud}") {
			t.Fatalf("%s: wrong upstream wiring:\n%s", r.Label, r.Config)
		}
		if (r.Detectors == "dictionary") != strings.Contains(r.Config, "dictionary:") {
			t.Fatalf("%s: wrong dictionary wiring:\n%s", r.Label, r.Config)
		}
	}
	want := "regex-only/openai dictionary/openai regex-only/anthropic dictionary/anthropic"
	if got := strings.Join(labels, " "); got != want {
		t.Fatalf("labels %q, want %q", got, want)
	}
	if rep.CPUCores < 1 || rep.NER || rep.Memo {
		t.Fatalf("header facts wrong: cores=%d ner=%v memo=%v", rep.CPUCores, rep.NER, rep.Memo)
	}
}

func TestRealUpstreamRejectsAnthropic(t *testing.T) {
	o := opts(t, "unused")
	o.APIs = []string{"anthropic"}
	o.RealUpstream = "http://127.0.0.1:1/v1"
	if _, err := Execute(context.Background(), o); err == nil || !strings.Contains(err.Error(), "not supported with --api anthropic") {
		t.Fatalf("want a clear refusal, got %v", err)
	}
}

func TestResolveConfigs(t *testing.T) {
	cases := []struct {
		configs, dict []string
		ner           string
		want          string
	}{
		{nil, nil, "", "dictionary regex-only"},
		{nil, nil, "http://x", "dictionary+ner ner"},
		{nil, []string{"without"}, "", "regex-only"},
		{nil, []string{"with"}, "http://x", "dictionary+ner"},
		{[]string{"regex-only", "dictionary", "ner", "dictionary+ner"}, nil, "http://x", "regex-only dictionary ner dictionary+ner"},
		{[]string{"with-dictionary", "without-dictionary", "ner-only", "dictionary"}, nil, "http://x", "dictionary regex-only ner"},
	}
	for _, c := range cases {
		got, err := ResolveConfigs(c.configs, c.dict, c.ner)
		if err != nil || strings.Join(got, " ") != c.want {
			t.Errorf("ResolveConfigs(%v, %v, %q) = %v, %v; want %s", c.configs, c.dict, c.ner, got, err, c.want)
		}
	}
	for _, bad := range [][]string{{"ner"}, {"dictionary+ner"}, {"bogus"}} {
		if _, err := ResolveConfigs(bad, nil, ""); err == nil {
			t.Errorf("ResolveConfigs(%v) without NER should fail", bad)
		}
	}
}
