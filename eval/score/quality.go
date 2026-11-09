package score

import (
	"encoding/json"
	"strings"

	"github.com/davidparkercodes/outworld/eval/corpus"
)

type PRF struct {
	TP        int     `json:"tp"`
	FP        int     `json:"fp"`
	FN        int     `json:"fn"`
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
	F1        float64 `json:"f1"`
}

func (p *PRF) finish() {
	if p.TP+p.FP > 0 {
		p.Precision = float64(p.TP) / float64(p.TP+p.FP)
	}
	if p.TP+p.FN > 0 {
		p.Recall = float64(p.TP) / float64(p.TP+p.FN)
	}
	if p.Precision+p.Recall > 0 {
		p.F1 = 2 * p.Precision * p.Recall / (p.Precision + p.Recall)
	}
}

func (p *PRF) Add(o PRF) {
	p.TP += o.TP
	p.FP += o.FP
	p.FN += o.FN
	p.finish()
}

func ParseFacts(content string) []corpus.Fact {
	start := strings.Index(content, "[")
	for start >= 0 {
		depth := 0
		for i := start; i < len(content); i++ {
			switch content[i] {
			case '[':
				depth++
			case ']':
				depth--
			}
			if depth == 0 {
				var raw []map[string]any
				if err := json.Unmarshal([]byte(content[start:i+1]), &raw); err == nil {
					var out []corpus.Fact
					for _, r := range raw {
						d, _ := r["date"].(string)
						p, _ := r["person"].(string)
						if d != "" {
							out = append(out, corpus.Fact{Date: strings.TrimSpace(d), Person: strings.TrimSpace(p)})
						}
					}
					return out
				}
				break
			}
		}
		next := strings.Index(content[start+1:], "[")
		if next < 0 {
			return nil
		}
		start += next + 1
	}
	return nil
}

func personMatch(pred, gold string) bool {
	p, g := strings.ToLower(pred), strings.ToLower(gold)
	if p == g {
		return true
	}
	parts := strings.Fields(g)
	return len(parts) > 1 && strings.Contains(p, parts[len(parts)-1])
}

func MatchFacts(pred, gold []corpus.Fact) PRF {
	used := make([]bool, len(gold))
	var r PRF
	for _, p := range pred {
		hit := false
		for i, g := range gold {
			if !used[i] && p.Date == g.Date && personMatch(p.Person, g.Person) {
				used[i], hit = true, true
				break
			}
		}
		if hit {
			r.TP++
		} else {
			r.FP++
		}
	}
	for _, u := range used {
		if !u {
			r.FN++
		}
	}
	r.finish()
	return r
}
