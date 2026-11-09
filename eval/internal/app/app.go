package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/davidparkercodes/outworld/eval/corpus"
	"github.com/davidparkercodes/outworld/eval/internal/gateway"
	"github.com/davidparkercodes/outworld/eval/internal/runner"
	"github.com/davidparkercodes/outworld/eval/score"
	"github.com/davidparkercodes/outworld/eval/upstream"
)

type Options struct {
	CorpusPath   string
	PeoplePath   string
	OutworldDir  string
	GatewayBin   string
	OutDir       string
	DictModes    []string
	Modes        []string
	MaxLeak      float64
	MaxErrorRate float64
	Limit        int
	Model        string
	RealUpstream string
	NERURL       string
	APIKeyEnv    string
	Timeout      time.Duration
	KeepTmp      bool
	Log          io.Writer
}

type RunReport struct {
	Label      string             `json:"label"`
	Dictionary string             `json:"dictionary,omitempty"`
	Config     string             `json:"config"`
	GatewayLog string             `json:"gateway_log,omitempty"`
	Summary    score.Summary      `json:"summary"`
	Quality    *QualitySummary    `json:"quality,omitempty"`
	Items      []score.ItemResult `json:"items"`
}

type QualitySummary struct {
	Items   int                  `json:"items"`
	Direct  score.PRF            `json:"direct"`
	Gateway score.PRF            `json:"gateway"`
	Errors  int                  `json:"errors"`
	Pairs   []runner.QualityPair `json:"pairs"`
}

type CorpusStats struct {
	Items       int            `json:"items"`
	Requests    int            `json:"requests_per_run"`
	Spans       int            `json:"spans"`
	SpansByType map[string]int `json:"spans_by_type"`
	DictPeople  int            `json:"dictionary_people"`
}

type Gate struct {
	MaxLeak      float64  `json:"max_leak"`
	MaxErrorRate float64  `json:"max_error_rate"`
	Passed       bool     `json:"passed"`
	Reasons      []string `json:"reasons,omitempty"`
}

type Report struct {
	RunDir   string      `json:"-"`
	Started  time.Time   `json:"started"`
	Duration string      `json:"duration"`
	Gateway  string      `json:"gateway"`
	Upstream string      `json:"upstream"`
	Model    string      `json:"model"`
	Modes    []string    `json:"modes"`
	Corpus   CorpusStats `json:"corpus"`
	Runs     []RunReport `json:"runs"`
	Gate     Gate        `json:"gate"`
}

func (o *Options) logf(format string, args ...any) {
	if o.Log != nil {
		fmt.Fprintf(o.Log, format+"\n", args...)
	}
}

func Execute(ctx context.Context, o Options) (*Report, error) {
	c, err := corpus.Load(o.CorpusPath)
	if err != nil {
		return nil, err
	}
	items := c.Items
	if o.Limit > 0 && o.Limit < len(items) {
		items = items[:o.Limit]
	}
	dictEntries, err := corpus.LoadDictionary(o.PeoplePath)
	if err != nil {
		return nil, err
	}
	dict := map[string]bool{}
	for _, e := range dictEntries {
		dict[strings.ToLower(e.Value)] = true
	}
	peopleAbs, err := filepath.Abs(o.PeoplePath)
	if err != nil {
		return nil, err
	}

	rep := &Report{Started: time.Now().UTC(), Model: o.Model, Modes: o.Modes, Gate: Gate{MaxLeak: o.MaxLeak, MaxErrorRate: o.MaxErrorRate, Passed: true}}
	rep.Corpus = CorpusStats{Items: len(items), Requests: len(items) * len(o.Modes), SpansByType: map[string]int{}, DictPeople: len(dictEntries)}
	for _, it := range items {
		rep.Corpus.Spans += len(it.Spans)
		for _, s := range it.Spans {
			rep.Corpus.SpansByType[s.Type]++
		}
	}

	rep.RunDir = filepath.Join(o.OutDir, rep.Started.Format("20060102-150405"))
	if err := os.MkdirAll(rep.RunDir, 0o755); err != nil {
		return nil, err
	}
	tmp, err := os.MkdirTemp("", "outworld-eval-")
	if err != nil {
		return nil, err
	}
	if !o.KeepTmp {
		defer os.RemoveAll(tmp)
	}

	bin := o.GatewayBin
	if bin == "" {
		o.logf("building gateway from %s", o.OutworldDir)
		bin, err = gateway.Build(ctx, o.OutworldDir, tmp)
		if err != nil {
			return nil, err
		}
		rep.Gateway = "built from " + filepath.Join(o.OutworldDir, "cmd", "outworld")
	} else {
		rep.Gateway = bin
	}

	real := o.RealUpstream != ""
	apiKey := ""
	if real {
		apiKey = os.Getenv(o.APIKeyEnv)
		if apiKey == "" {
			return nil, fmt.Errorf("real upstream mode needs an API key in $%s", o.APIKeyEnv)
		}
		rep.Upstream = "real (recording proxy in front of " + o.RealUpstream + ")"
	} else {
		rep.Upstream = "fake (deterministic echo)"
	}

	pol := score.DefaultPolicy()
	for _, dm := range o.DictModes {
		rr, err := runOnce(ctx, o, tmp, rep.RunDir, bin, dm, peopleAbs, items, pol, dict, real, apiKey)
		if err != nil {
			return nil, err
		}
		rep.Runs = append(rep.Runs, *rr)
		s := rr.Summary
		if s.Leak.N > 0 && s.Leak.Rate > o.MaxLeak {
			rep.Gate.Passed = false
			rep.Gate.Reasons = append(rep.Gate.Reasons, fmt.Sprintf("%s: leak rate %.2f%% > %.2f%%", rr.Label, 100*s.Leak.Rate, 100*o.MaxLeak))
		}
		if s.ErrorRate > o.MaxErrorRate {
			rep.Gate.Passed = false
			rep.Gate.Reasons = append(rep.Gate.Reasons, fmt.Sprintf("%s: error rate %.2f%% > %.2f%%", rr.Label, 100*s.ErrorRate, 100*o.MaxErrorRate))
		}
	}
	rep.Duration = time.Since(rep.Started).Round(time.Millisecond).String()
	return rep, nil
}

func runOnce(ctx context.Context, o Options, tmp, runDir, bin, dictMode, peopleAbs string, items []corpus.Item, pol score.Policy, dict map[string]bool, real bool, apiKey string) (*RunReport, error) {
	label := "with-dictionary"
	dictPath := peopleAbs
	if dictMode == "without" {
		label, dictPath = "without-dictionary", ""
	}
	if o.NERURL != "" {
		label = "dictionary+ner"
		if dictPath == "" {
			label = "ner-only"
		}
	}
	o.logf("== run %s", label)

	var rec *upstream.Recorder
	var gwUp, direct *upstream.Server
	var err error
	directV1, directKey := "", ""
	if real {
		p := upstream.NewProxy(o.RealUpstream, apiKey)
		rec = p.Recorder
		gwUp, err = upstream.Serve(p.Handler(), rec)
		directV1, directKey = o.RealUpstream, apiKey
	} else {
		f := upstream.NewFake()
		f.Model = o.Model
		rec = f.Recorder
		gwUp, err = upstream.Serve(f.Handler(), rec)
		if err == nil {
			d := upstream.NewFake()
			d.Model = o.Model
			direct, err = upstream.Serve(d.Handler(), d.Recorder)
			if direct != nil {
				directV1 = direct.URL + "/v1"
				defer direct.Close()
			}
		}
	}
	if err != nil {
		return nil, err
	}
	defer gwUp.Close()

	port, err := gateway.FreePort()
	if err != nil {
		return nil, err
	}
	listen := fmt.Sprintf("127.0.0.1:%d", port)
	cfg := gateway.Config{
		Listen: listen, VaultPath: filepath.Join(tmp, label+".vault"), UpstreamURL: gwUp.URL + "/v1",
		Dictionary: dictPath, NERURL: o.NERURL, DateShift: true, Entities: pol,
	}
	cfgPath := filepath.Join(tmp, label+".yaml")
	if err := os.WriteFile(cfgPath, []byte(cfg.YAML()), 0o600); err != nil {
		return nil, err
	}
	logPath := filepath.Join(runDir, label+".gateway.log")
	cloudKey := "eval-fake-key"
	if real {
		cloudKey = apiKey
	}
	proc, err := gateway.Start(ctx, bin, cfgPath, logPath, []string{"OUTWORLD_VAULT_KEY=" + gateway.RandomKey(), "CLOUD_API_KEY=" + cloudKey}, listen)
	if err != nil {
		return nil, err
	}
	defer proc.Stop()

	t := runner.Target{
		GatewayV1: proc.URL + "/v1", DirectV1: directV1, DirectKey: directKey, Recorder: rec,
		Model: o.Model, CheckRestore: !real, Timeout: o.Timeout,
	}
	obs := runner.Run(ctx, items, o.Modes, t, func(done, total int) {
		if done%50 == 0 || done == total {
			o.logf("  %d/%d requests", done, total)
		}
	})
	rr := &RunReport{Label: label, Dictionary: dictPath, Config: cfg.YAML(), GatewayLog: logPath}
	for _, ob := range obs {
		rr.Items = append(rr.Items, score.ScoreItem(ob, pol, dict))
	}
	rr.Summary = score.Aggregate(rr.Items)
	if real {
		pairs := runner.Quality(ctx, items, t)
		q := &QualitySummary{Pairs: pairs}
		for _, p := range pairs {
			q.Items++
			if p.DirectErr != "" || p.GatewayErr != "" {
				q.Errors++
				continue
			}
			q.Direct.Add(p.Direct)
			q.Gateway.Add(p.Gateway)
		}
		rr.Quality = q
	}
	return rr, nil
}
