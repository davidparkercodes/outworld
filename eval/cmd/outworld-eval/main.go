package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/davidparkercodes/outworld/eval/internal/app"
	"github.com/davidparkercodes/outworld/eval/internal/report"
)

func split(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func main() {
	var o app.Options
	var dictModes, modes, configs, apis string
	flag.StringVar(&o.CorpusPath, "corpus", "corpus/corpus.json", "corpus JSON")
	flag.StringVar(&o.PeoplePath, "people", "corpus/people.csv", "people dictionary CSV")
	flag.StringVar(&o.OutworldDir, "outworld-dir", "..", "outworld module root (used to build ./cmd/outworld)")
	flag.StringVar(&o.GatewayBin, "gateway-bin", "", "prebuilt gateway binary (skips go build)")
	flag.StringVar(&o.OutDir, "out", "results", "results directory")
	flag.StringVar(&configs, "configs", "", "detector configs to run, comma-separated: regex-only, dictionary, ner, dictionary+ner (default dictionary,regex-only, or dictionary+ner,ner with --ner-url)")
	flag.StringVar(&dictModes, "dictionary", "", "legacy selector used when --configs is not set: with, without, or both comma-separated")
	flag.StringVar(&apis, "api", "openai", "wire formats to exercise, comma-separated: openai, anthropic")
	flag.BoolVar(&o.Memo, "memo", false, "enable the gateway's cross-request mask memo (off by default so repeat requests are not cache hits)")
	flag.StringVar(&modes, "modes", "nonstream,stream", "request modes: nonstream, stream")
	flag.Float64Var(&o.MaxLeak, "max-leak", 0.05, "fail (exit 1) if any run's leak rate exceeds this fraction")
	flag.Float64Var(&o.MaxErrorRate, "max-error-rate", 0.02, "fail (exit 1) if any run's error rate exceeds this fraction")
	flag.IntVar(&o.Limit, "limit", 0, "only use the first N corpus items")
	flag.StringVar(&o.Model, "model", "fake-model", "model id sent in requests")
	flag.StringVar(&o.RealUpstream, "real-upstream", "", "OpenAI-compatible base URL (with /v1) of a real model; enables quality comparison")
	flag.StringVar(&o.NERURL, "ner-url", "", "base URL of a running NER sidecar (detectors/gliner); used by the ner and dictionary+ner configs")
	flag.StringVar(&o.APIKeyEnv, "api-key-env", "OUTWORLD_EVAL_API_KEY", "env var holding the real upstream API key")
	flag.DurationVar(&o.Timeout, "timeout", 2*time.Minute, "per-request timeout")
	flag.BoolVar(&o.KeepTmp, "keep-tmp", false, "keep the temporary config and vault directory")
	flag.Parse()
	o.Modes = split(modes)
	o.APIs = split(apis)
	o.Log = os.Stderr
	var err error
	if o.Configs, err = app.ResolveConfigs(split(configs), split(dictModes), o.NERURL); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if o.RealUpstream != "" && o.Model == "fake-model" {
		fmt.Fprintln(os.Stderr, "--real-upstream needs --model")
		os.Exit(2)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	rep, err := app.Execute(ctx, o)
	if err != nil {
		fmt.Fprintln(os.Stderr, "eval failed:", err)
		os.Exit(2)
	}
	mdPath, jsonPath, err := report.Write(rep)
	if err != nil {
		fmt.Fprintln(os.Stderr, "writing report:", err)
		os.Exit(2)
	}
	for _, r := range rep.Runs {
		s := r.Summary
		fmt.Printf("%-26s leak %.2f%% (%d/%d)  restore %.2f%%  over-mask %.2f%%  errors %d/%d %v\n", r.Label,
			100*s.Leak.Rate, s.Leak.Hits, s.Leak.N, 100*s.Restore.Rate, 100*s.OverMask.Rate, s.Errors, s.Requests, s.ErrorKinds)
	}
	fmt.Printf("report: %s\n        %s\n", mdPath, jsonPath)
	if !rep.Gate.Passed {
		fmt.Println("GATE FAILED: " + strings.Join(rep.Gate.Reasons, "; "))
		os.Exit(1)
	}
}
