package app

import (
	"context"
	"os/exec"
	"path/filepath"
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
		OutDir: t.TempDir(), DictModes: []string{"with"}, Modes: []string{"nonstream", "stream"},
		MaxLeak: 0.05, MaxErrorRate: 0, Limit: 25, Model: "fake-model", Timeout: 30 * time.Second,
	}
}

func TestEndToEndPassthroughLeaksEverything(t *testing.T) {
	rep, err := Execute(context.Background(), opts(t, fakeGateway(t)))
	if err != nil {
		t.Fatal(err)
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

func TestEndToEndOracleLeaksNothing(t *testing.T) {
	bin := fakeGateway(t)
	abs, _ := filepath.Abs("../../corpus/corpus.json")
	t.Setenv("FAKE_OUTWORLD_MODE", "oracle")
	t.Setenv("FAKE_OUTWORLD_CORPUS", abs)
	rep, err := Execute(context.Background(), opts(t, bin))
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
