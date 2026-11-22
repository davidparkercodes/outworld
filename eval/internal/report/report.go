package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/davidparkercodes/outworld/eval/corpus"
	"github.com/davidparkercodes/outworld/eval/internal/app"
	"github.com/davidparkercodes/outworld/eval/score"
)

func Write(rep *app.Report) (string, string, error) {
	js, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return "", "", err
	}
	md := Markdown(rep)
	jsonPath := filepath.Join(rep.RunDir, "report.json")
	mdPath := filepath.Join(rep.RunDir, "report.md")
	if err := os.WriteFile(jsonPath, js, 0o644); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(mdPath, []byte(md), 0o644); err != nil {
		return "", "", err
	}
	parent := filepath.Dir(rep.RunDir)
	_ = os.WriteFile(filepath.Join(parent, "latest.json"), js, 0o644)
	_ = os.WriteFile(filepath.Join(parent, "latest.md"), []byte(md), 0o644)
	return mdPath, jsonPath, nil
}

func pct(r *score.Rate) string {
	if r == nil || r.N == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f%% (%d/%d)", 100*r.Rate, r.Hits, r.N)
}

func pctInv(r *score.Rate) string {
	if r == nil || r.N == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f%% (%d/%d)", 100*(1-r.Rate), r.N-r.Hits, r.N)
}

func keys(ms ...map[string]*score.Rate) []string {
	set := map[string]bool{}
	for _, m := range ms {
		for k := range m {
			set[k] = true
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func table(b *strings.Builder, title string, runs []app.RunReport, pick func(score.Summary) map[string]*score.Rate, order []string) {
	var ms []map[string]*score.Rate
	for _, r := range runs {
		ms = append(ms, pick(r.Summary))
	}
	ks := order
	if ks == nil {
		ks = keys(ms...)
	} else {
		var filtered []string
		for _, k := range ks {
			for _, m := range ms {
				if m[k] != nil {
					filtered = append(filtered, k)
					break
				}
			}
		}
		ks = filtered
	}
	if len(ks) == 0 {
		return
	}
	fmt.Fprintf(b, "\n## %s\n\n| | ", title)
	for _, r := range runs {
		fmt.Fprintf(b, "%s | ", r.Label)
	}
	b.WriteString("\n|---|")
	for range runs {
		b.WriteString("---|")
	}
	b.WriteString("\n")
	for _, k := range ks {
		fmt.Fprintf(b, "| %s | ", k)
		for _, m := range ms {
			fmt.Fprintf(b, "%s | ", pct(m[k]))
		}
		b.WriteString("\n")
	}
}

func Markdown(rep *app.Report) string {
	var b strings.Builder
	b.WriteString("# Outworld eval report\n\n")
	fmt.Fprintf(&b, "- Started: %s (took %s)\n", rep.Started.Format("2006-01-02 15:04:05 MST"), rep.Duration)
	fmt.Fprintf(&b, "- Gateway: %s\n- Upstream: %s\n- Model: `%s`\n- Modes: %s\n", rep.Gateway, rep.Upstream, rep.Model, strings.Join(rep.Modes, ", "))
	fmt.Fprintf(&b, "- APIs: %s\n- Detector configs: %s\n", strings.Join(rep.APIs, ", "), strings.Join(rep.Configs, ", "))
	fmt.Fprintf(&b, "- Host: %d CPU cores\n- NER: %s\n- Memo: %s\n", rep.CPUCores, nerNote(rep), memoNote(rep))
	fmt.Fprintf(&b, "- Corpus: %d items, %d spans, %d requests per run, %d people in dictionary\n", rep.Corpus.Items, rep.Corpus.Spans, rep.Corpus.Requests, rep.Corpus.DictPeople)
	status := "PASS"
	if !rep.Gate.Passed {
		status = "FAIL: " + strings.Join(rep.Gate.Reasons, "; ")
	}
	fmt.Fprintf(&b, "- Gate (leak <= %.2f%%, errors <= %.2f%%): **%s**\n", 100*rep.Gate.MaxLeak, 100*rep.Gate.MaxErrorRate, status)

	runs := rep.Runs
	b.WriteString("\n## Summary\n\n| Metric | ")
	for _, r := range runs {
		fmt.Fprintf(&b, "%s | ", r.Label)
	}
	b.WriteString("\n|---|")
	for range runs {
		b.WriteString("---|")
	}
	b.WriteString("\n")
	row := func(name string, f func(s score.Summary) string) {
		fmt.Fprintf(&b, "| %s | ", name)
		for _, r := range runs {
			fmt.Fprintf(&b, "%s | ", f(r.Summary))
		}
		b.WriteString("\n")
	}
	row("Leak rate, all requests (sensitive values seen upstream)", func(s score.Summary) string { return pct(&s.Leak) })
	row("Leak rate, answered requests only", func(s score.Summary) string { return pct(&s.LeakAnswered) })
	row("Restore accuracy", func(s score.Summary) string { return pct(&s.Restore) })
	row("Dates restored only in another format", func(s score.Summary) string { return fmt.Sprint(s.DateReformatted) })
	row("Over-masking (kept types altered)", func(s score.Summary) string { return pct(&s.OverMask) })
	row("Quasi-identifiers sent intact", func(s score.Summary) string { return pctInv(&s.QuasiAltered) })
	row("Unknown surrogates reported by gateway", func(s score.Summary) string { return fmt.Sprint(s.UnknownReported) })
	row("Responses with surrogate residue", func(s score.Summary) string {
		return fmt.Sprintf("%d (%d tokens)", s.ResidueItems, s.ResidueTokens)
	})
	row("Placeholder tokens (frequent, excluded from residue)", func(s score.Summary) string {
		if len(s.Placeholders) == 0 {
			return "none"
		}
		return "`" + strings.Join(s.Placeholders, "`, `") + "`"
	})
	row("Errors", func(s score.Summary) string { return fmt.Sprintf("%d/%d", s.Errors, s.Requests) })
	row("Errors by kind", func(s score.Summary) string {
		if len(s.ErrorKinds) == 0 {
			return "none"
		}
		var ks []string
		for k := range s.ErrorKinds {
			ks = append(ks, k)
		}
		sort.Strings(ks)
		var parts []string
		for _, k := range ks {
			parts = append(parts, fmt.Sprintf("%s: %d", k, s.ErrorKinds[k]))
		}
		return strings.Join(parts, ", ")
	})
	row("Upstream calls", func(s score.Summary) string { return fmt.Sprint(s.UpstreamCalls) })

	table(&b, "Leak rate by entity type", runs, func(s score.Summary) map[string]*score.Rate { return s.LeakByType }, corpus.EntityTypes)
	table(&b, "Person leak rate by reference variant (answered requests)", runs, func(s score.Summary) map[string]*score.Rate { return s.PersonByVariant }, nil)
	table(&b, "Person leak rate by dictionary coverage (answered requests)", runs, func(s score.Summary) map[string]*score.Rate { return s.PersonByDict }, nil)
	modeTable(&b, runs)
	table(&b, "Restore accuracy by entity type (answered requests)", runs, func(s score.Summary) map[string]*score.Rate { return s.RestoreByType }, corpus.EntityTypes)
	table(&b, "Over-masking by kept type (answered requests)", runs, func(s score.Summary) map[string]*score.Rate { return s.OverMaskByType }, nil)

	latencyTables(&b, rep)

	for _, r := range runs {
		if r.Quality == nil {
			continue
		}
		q := r.Quality
		fmt.Fprintf(&b, "\n## Task quality, %s (timeline extraction vs ground truth)\n\n", r.Label)
		fmt.Fprintf(&b, "%d items, %d with errors.\n\n| Path | precision | recall | F1 |\n|---|---|---|---|\n", q.Items, q.Errors)
		fmt.Fprintf(&b, "| direct | %.3f | %.3f | %.3f |\n| via Outworld | %.3f | %.3f | %.3f |\n", q.Direct.Precision, q.Direct.Recall, q.Direct.F1, q.Gateway.Precision, q.Gateway.Recall, q.Gateway.F1)
	}

	for _, r := range runs {
		s := r.Summary
		if len(s.ErrorSamples) > 0 {
			fmt.Fprintf(&b, "\n## Errors, %s\n\n", r.Label)
			for _, e := range s.ErrorSamples {
				fmt.Fprintf(&b, "- `%s`\n", oneLine(e, 300))
			}
		}
		if len(s.ResidueSamples) > 0 {
			fmt.Fprintf(&b, "\n## Surrogate residue samples, %s\n\n%s\n", r.Label, "`"+strings.Join(s.ResidueSamples, "`, `")+"`")
		}
		if len(s.Leaks) > 0 {
			fmt.Fprintf(&b, "\n## Leak samples, %s (first 40 of %d)\n\n| Item | Mode | Type | Variant | Via | Value |\n|---|---|---|---|---|---|\n", r.Label, len(s.Leaks))
			for i, l := range s.Leaks {
				if i == 40 {
					break
				}
				fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | `%s` |\n", l.Item, l.Mode, l.Type, l.Variant, l.Via, oneLine(l.Value, 80))
			}
		}
		fmt.Fprintf(&b, "\nGateway log: `%s`\n", r.GatewayLog)
	}
	return b.String()
}

func nerNote(rep *app.Report) string {
	var with []string
	for _, r := range rep.Runs {
		if r.NER {
			with = append(with, r.Label)
		}
	}
	if len(with) == 0 {
		return "not in use"
	}
	return "in use (runs: " + strings.Join(with, ", ") + ")"
}

func memoNote(rep *app.Report) string {
	if rep.Memo {
		return "ON (`--memo`): repeated text is served from the gateway's mask memo, so the second mode of each item is a cache hit and its latency is not a cold measurement"
	}
	return "off (every request is masked from scratch)"
}

func uniq(runs []app.RunReport, pick func(app.RunReport) string) []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range runs {
		if k := pick(r); !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	return out
}

func find(runs []app.RunReport, api, cfg string) *app.RunReport {
	for i := range runs {
		if runs[i].API == api && runs[i].Detectors == cfg {
			return &runs[i]
		}
	}
	return nil
}

func pivotHead(b *strings.Builder, lead []string, cfgs []string) {
	fmt.Fprintf(b, "| %s | %s |\n|", strings.Join(lead, " | "), strings.Join(cfgs, " | "))
	for range len(lead) + len(cfgs) {
		b.WriteString("---|")
	}
	b.WriteString("\n")
}

func runModes(runs []app.RunReport, api string, has func(app.RunReport, string) bool, all func(app.RunReport) []string) []string {
	set := map[string]bool{}
	for _, r := range runs {
		if r.API != api {
			continue
		}
		for _, m := range all(r) {
			if has(r, m) {
				set[m] = true
			}
		}
	}
	out := make([]string, 0, len(set))
	for m := range set {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}

func modeTable(b *strings.Builder, runs []app.RunReport) {
	cfgs := uniq(runs, func(r app.RunReport) string { return r.Detectors })
	b.WriteString("\n## Leak rate by API and request mode\n\n")
	pivotHead(b, []string{"API", "Mode"}, cfgs)
	for _, api := range uniq(runs, func(r app.RunReport) string { return r.API }) {
		modes := runModes(runs, api, func(r app.RunReport, m string) bool { return r.Summary.LeakByMode[m] != nil }, func(r app.RunReport) []string {
			return keys(r.Summary.LeakByMode)
		})
		for _, m := range modes {
			fmt.Fprintf(b, "| %s | %s | ", api, m)
			for _, c := range cfgs {
				var rate *score.Rate
				if r := find(runs, api, c); r != nil {
					rate = r.Summary.LeakByMode[m]
				}
				fmt.Fprintf(b, "%s | ", pct(rate))
			}
			b.WriteString("\n")
		}
	}
}

func latencyTables(b *strings.Builder, rep *app.Report) {
	runs := rep.Runs
	cfgs := uniq(runs, func(r app.RunReport) string { return r.Detectors })
	apis := uniq(runs, func(r app.RunReport) string { return r.API })
	b.WriteString("\n## Latency (ms)\n\n")
	fmt.Fprintf(b, "Host has %d CPU cores. NER: %s. Memo: %s. Answered requests only; overhead is gateway time minus the same request sent straight to the baseline fake.\n", rep.CPUCores, nerNote(rep), memoNote(rep))
	b.WriteString("\n### By detector config\n\n")
	pivotHead(b, []string{"API", "Mode", "Metric"}, cfgs)
	metrics := []struct {
		name string
		f    func(score.Latency) string
	}{
		{"n", func(l score.Latency) string { return fmt.Sprint(l.N) }},
		{"gateway p50", func(l score.Latency) string { return fmt.Sprintf("%.2f", l.GatewayP50) }},
		{"gateway p95", func(l score.Latency) string { return fmt.Sprintf("%.2f", l.GatewayP95) }},
		{"overhead p50", func(l score.Latency) string { return fmt.Sprintf("%.2f", l.OverP50) }},
		{"overhead p95", func(l score.Latency) string { return fmt.Sprintf("%.2f", l.OverP95) }},
		{"overhead > 1 s", func(l score.Latency) string { return fmt.Sprintf("%d (%.1f%%)", l.OverSlow, 100*l.SlowShare) }},
	}
	latModes := func(api string) []string {
		return runModes(runs, api, func(r app.RunReport, m string) bool { _, ok := r.Summary.Latency[m]; return ok }, func(r app.RunReport) []string {
			var ms []string
			for m := range r.Summary.Latency {
				ms = append(ms, m)
			}
			return ms
		})
	}
	for _, api := range apis {
		for _, m := range latModes(api) {
			for _, mt := range metrics {
				fmt.Fprintf(b, "| %s | %s | %s | ", api, m, mt.name)
				for _, c := range cfgs {
					cell := "n/a"
					if r := find(runs, api, c); r != nil {
						if l, ok := r.Summary.Latency[m]; ok {
							cell = mt.f(l)
						}
					}
					fmt.Fprintf(b, "%s | ", cell)
				}
				b.WriteString("\n")
			}
		}
	}
	b.WriteString("\n### Detail\n\n| Config | API | Mode | n | gateway p50 | gateway p95 | direct p50 | direct p95 | overhead p50 | overhead p95 | overhead > 1 s | share > 1 s |\n|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, r := range runs {
		for _, m := range latModes(r.API) {
			l, ok := r.Summary.Latency[m]
			if !ok {
				continue
			}
			fmt.Fprintf(b, "| %s | %s | %s | %d | %.2f | %.2f | %.2f | %.2f | %.2f | %.2f | %d | %.1f%% |\n", r.Detectors, r.API, m, l.N, l.GatewayP50, l.GatewayP95, l.DirectP50, l.DirectP95, l.OverP50, l.OverP95, l.OverSlow, 100*l.SlowShare)
		}
	}
}

func oneLine(s string, n int) string {
	s = strings.NewReplacer("\n", " ", "|", "\\|", "`", "'").Replace(s)
	if len(s) > n {
		s = s[:n] + " [cut, see report.json]"
	}
	return s
}
