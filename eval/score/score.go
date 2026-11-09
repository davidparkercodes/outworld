package score

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/davidparkercodes/outworld/eval/corpus"
	"github.com/davidparkercodes/outworld/eval/upstream"
)

const (
	ActionMask  = "mask"
	ActionShift = "shift"
	ActionStrip = "strip"
	ActionKeep  = "keep"
)

type Policy map[string]string

func DefaultPolicy() Policy {
	return Policy{
		corpus.TypePerson: ActionMask, corpus.TypeEmail: ActionMask, corpus.TypePhone: ActionMask,
		corpus.TypeAddress: ActionMask, corpus.TypeDate: ActionShift, corpus.TypeOrg: ActionKeep,
		corpus.TypeLocation: ActionKeep, corpus.TypeSecret: ActionStrip, corpus.TypeGovID: ActionStrip,
		corpus.TypeFinancial: ActionStrip,
	}
}

func (p Policy) Sensitive(typ string) bool {
	switch p[typ] {
	case ActionMask, ActionShift, ActionStrip:
		return true
	}
	return false
}

type Observation struct {
	Item          *corpus.Item
	Mode          string
	Upstream      string
	UpstreamCalls int
	Final         string
	Err           string
	Latency       time.Duration
	DirectLatency time.Duration
	Unknown       int
	CheckRestore  bool
}

type EntityResult struct {
	Type       string `json:"type"`
	Action     string `json:"action"`
	Value      string `json:"value"`
	Variant    string `json:"variant,omitempty"`
	InDict     bool   `json:"in_dict,omitempty"`
	Leaked     bool   `json:"leaked"`
	LeakVia    string `json:"leak_via,omitempty"`
	Restorable bool   `json:"restorable"`
	Restored   bool   `json:"restored"`
	Reformat   bool   `json:"restored_reformatted,omitempty"`
	Altered    bool   `json:"altered,omitempty"`
}

type ItemResult struct {
	ID            string         `json:"id"`
	Mode          string         `json:"mode"`
	Err           string         `json:"error,omitempty"`
	UpstreamCalls int            `json:"upstream_calls"`
	LatencyMS     float64        `json:"latency_ms"`
	DirectMS      float64        `json:"direct_latency_ms"`
	Unknown       int            `json:"unknown_surrogates"`
	Residue       []string       `json:"surrogate_residue,omitempty"`
	Entities      []EntityResult `json:"entities"`
}

func messagesText(it *corpus.Item, blank func(corpus.Span) bool, includeSystem bool, sep string) string {
	var parts []string
	for mi, m := range it.Messages {
		if !includeSystem && (m.Role == "system" || m.Role == "developer") {
			continue
		}
		c := []byte(m.Content)
		if blank != nil {
			for _, s := range it.Spans {
				if s.Msg == mi && blank(s) {
					for k := s.Start; k < s.End; k++ {
						c[k] = 0
					}
				}
			}
		}
		parts = append(parts, string(c))
	}
	return strings.Join(parts, sep)
}

type unit struct {
	span  corpus.Span
	forms []string
}

func ScoreItem(o Observation, pol Policy, dict map[string]bool) ItemResult {
	it := o.Item
	res := ItemResult{ID: it.ID, Mode: o.Mode, Err: o.Err, UpstreamCalls: o.UpstreamCalls, Unknown: o.Unknown,
		LatencyMS: ms(o.Latency), DirectMS: ms(o.DirectLatency)}
	sensitive := func(s corpus.Span) bool { return pol.Sensitive(s.Type) }
	orig := messagesText(it, nil, true, "\n")
	bg := messagesText(it, sensitive, true, "\n")
	echoOrig := messagesText(it, nil, false, upstream.MessageDivider)
	up := o.Upstream
	upLower, bgLower := strings.ToLower(up), strings.ToLower(bg)
	upStrip, bgStrip := StripMarkdown(upLower), StripMarkdown(bgLower)
	upRuns, bgRuns := DigitRuns(up), DigitRuns(bg)
	finalEcho := strings.ToLower(upstream.EchoSection(o.Final))

	seen := map[string]bool{}
	for _, s := range it.Spans {
		key := s.Type + "\x00" + strings.ToLower(s.Value)
		if seen[key] {
			continue
		}
		seen[key] = true
		action := pol[s.Type]
		if s.Type == corpus.TypeQuasi {
			action = "report"
		}
		er := EntityResult{Type: s.Type, Action: action, Value: s.Value, Variant: s.Variant}
		if s.Type == corpus.TypePerson {
			er.InDict = dict[strings.ToLower(s.Canonical)]
		}
		val := strings.ToLower(s.Value)
		if !pol.Sensitive(s.Type) {
			er.Altered = o.Err == "" && CountWord(upLower, val) < CountWord(bgLower, val)
			res.Entities = append(res.Entities, er)
			continue
		}
		forms := append([]string{s.Value}, s.Needles...)
		for _, f := range forms {
			fl := strings.ToLower(f)
			if CountWord(upLower, fl) > CountWord(bgLower, fl) {
				er.Leaked = true
				switch {
				case f != s.Value:
					er.LeakVia = "variant"
				case CountWord(up, f) > CountWord(bg, f):
					er.LeakVia = "verbatim"
				default:
					er.LeakVia = "case-insensitive"
				}
				break
			}
		}
		if !er.Leaked && (strings.Contains(s.Value, "*") || strings.Contains(s.Value, "_")) {
			fs := StripMarkdown(val)
			if fs != val && CountWord(upStrip, fs) > CountWord(bgStrip, fs) {
				er.Leaked, er.LeakVia = true, "markdown-stripped"
			}
		}
		if !er.Leaked {
			switch s.Type {
			case corpus.TypePhone, corpus.TypeGovID, corpus.TypeFinancial:
				d := corpus.Digits(s.Value)
				if s.Type == corpus.TypePhone && len(d) > 10 {
					d = d[len(d)-10:]
				}
				if len(d) >= 7 && CountDigits(upRuns, d) > CountDigits(bgRuns, d) {
					er.Leaked, er.LeakVia = true, "digits"
				}
			}
		}
		if pol[s.Type] == ActionMask || pol[s.Type] == ActionShift {
			er.Restorable = o.CheckRestore && o.Err == "" && !er.Leaked
			if er.Restorable {
				need := CountWord(strings.ToLower(echoOrig), val)
				er.Restored = need > 0 && CountWord(finalEcho, val) >= need
				if !er.Restored && s.Type == corpus.TypeDate {
					for _, n := range s.Needles {
						if CountWord(finalEcho, strings.ToLower(n)) > 0 {
							er.Reformat = true
						}
					}
				}
			}
		}
		res.Entities = append(res.Entities, er)
	}
	if o.Err == "" && o.CheckRestore {
		origTok := Tokens(orig)
		upTok := Tokens(rePlaceholder.ReplaceAllString(up, " "))
		finTok := Tokens(upstream.EchoSection(o.Final))
		for t := range upTok {
			if !origTok[t] && finTok[t] {
				res.Residue = append(res.Residue, t)
			}
		}
		sort.Strings(res.Residue)
	}
	return res
}

var rePlaceholder = regexp.MustCompile(`\[[^\[\]\n]{1,40}\]`)

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }
