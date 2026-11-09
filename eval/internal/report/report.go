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
	table(&b, "Leak rate by request mode", runs, func(s score.Summary) map[string]*score.Rate { return s.LeakByMode }, nil)
	table(&b, "Restore accuracy by entity type (answered requests)", runs, func(s score.Summary) map[string]*score.Rate { return s.RestoreByType }, corpus.EntityTypes)
	table(&b, "Over-masking by kept type (answered requests)", runs, func(s score.Summary) map[string]*score.Rate { return s.OverMaskByType }, nil)

	b.WriteString("\n## Latency (ms)\n\n| Run | Mode | n | gateway p50 | gateway p95 | direct p50 | direct p95 | overhead p50 | overhead p95 |\n|---|---|---|---|---|---|---|---|---|\n")
	for _, r := range runs {
		var modes []string
		for m := range r.Summary.Latency {
			modes = append(modes, m)
		}
		sort.Strings(modes)
		for _, m := range modes {
			l := r.Summary.Latency[m]
			fmt.Fprintf(&b, "| %s | %s | %d | %.2f | %.2f | %.2f | %.2f | %.2f | %.2f |\n", r.Label, m, l.N, l.GatewayP50, l.GatewayP95, l.DirectP50, l.DirectP95, l.OverP50, l.OverP95)
		}
	}

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

func oneLine(s string, n int) string {
	s = strings.NewReplacer("\n", " ", "|", "\\|", "`", "'").Replace(s)
	if len(s) > n {
		s = s[:n] + " [cut, see report.json]"
	}
	return s
}
