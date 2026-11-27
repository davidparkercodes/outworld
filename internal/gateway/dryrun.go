package gateway

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/davidparkercodes/outworld/internal/config"
	"github.com/davidparkercodes/outworld/pkg/entity"
	"github.com/davidparkercodes/outworld/pkg/outworld"
)

const HeaderDryRun = "X-Outworld-Dry-Run"

const blockHelp = "Nothing was sent upstream. Each finding names a value the leak check still saw after masking; previews are redacted. Resend with the header " + HeaderDryRun + ": 1 to see the masked request without calling the upstream."

type ReplacementCount struct {
	Type   entity.Type   `json:"type"`
	Source string        `json:"source"`
	Action entity.Action `json:"action"`
	Count  int           `json:"count"`
}

type DryRunReport struct {
	DryRun       bool                   `json:"outworld_dry_run"`
	API          string                 `json:"api"`
	Route        string                 `json:"route"`
	RouteReason  string                 `json:"route_reason"`
	Upstream     string                 `json:"upstream,omitempty"`
	WouldSend    bool                   `json:"would_send"`
	Blocked      string                 `json:"blocked,omitempty"`
	Message      string                 `json:"message,omitempty"`
	Request      map[string]any         `json:"request,omitempty"`
	Summary      []ReplacementCount     `json:"summary"`
	Replacements []outworld.Replacement `json:"replacements"`
	Findings     []outworld.Violation   `json:"findings"`
	MemoHits     int                    `json:"memo_hits"`
}

func isDryRun(r *http.Request) bool {
	switch strings.ToLower(strings.TrimSpace(r.Header.Get(HeaderDryRun))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func (s *Server) maskTwice(ctx context.Context, sess *outworld.Session, req map[string]any, api string) (map[loc]leaf, error) {
	sess.SetPriming(true)
	_, err := s.maskFields(ctx, sess, req, api)
	sess.SetPriming(false)
	if err != nil {
		return nil, err
	}
	return s.maskFields(ctx, sess, req, api)
}

var containedTypes = map[[2]entity.Type]bool{
	{entity.Email, entity.Person}: true,
}

func folds(outer, inner outworld.Violation) bool {
	if outer.Message != inner.Message || outer.Field != inner.Field || outer.Start < 0 || inner.Start < 0 {
		return false
	}
	if outer.Start == inner.Start && outer.End == inner.End {
		return outer.Type == inner.Type || containedTypes[[2]entity.Type{outer.Type, inner.Type}] || containedTypes[[2]entity.Type{inner.Type, outer.Type}]
	}
	if outer.Start > inner.Start || inner.End > outer.End {
		return false
	}
	return outer.Type == inner.Type || containedTypes[[2]entity.Type{outer.Type, inner.Type}]
}

func better(a, b outworld.Violation) outworld.Violation {
	if a.Start == b.Start && a.End == b.End {
		if containedTypes[[2]entity.Type{b.Type, a.Type}] {
			return b
		}
		if a.Source == "vault" && b.Source != "vault" && a.Type == b.Type {
			return b
		}
	}
	return a
}

const redactedPreviewMark = "[***]"

const thinkingHint = "This value is in an earlier assistant thinking block, which is forwarded unchanged because the provider signs it. Drop that assistant turn, or rewind the conversation to before it, to continue."

const checkedFieldHint = "This value sits in a field the gateway checks but does not rewrite. Remove it from that field, or add it to policy.allow if it is not sensitive."

func DedupeViolations(vs []outworld.Violation) []outworld.Violation {
	var out []outworld.Violation
	for _, v := range vs {
		keep := true
		for i := 0; i < len(out); i++ {
			w := out[i]
			switch {
			case folds(w, v):
				out[i] = better(w, v)
				keep = false
			case folds(v, w):
				out = append(out[:i], out[i+1:]...)
				i--
			}
			if !keep {
				break
			}
		}
		if keep {
			out = append(out, v)
		}
	}
	return out
}

func SessionLeak(sess *outworld.Session) *outworld.LeakError { return leakOf(sess, nil) }

func nameRecordFinding(l leaf, v outworld.Violation) bool {
	return l.mode == modeName && (v.Source == "vault" || v.Source == "dictionary") && v.Type == entity.Person
}

func leakOf(sess *outworld.Session, narrow map[loc]leaf) *outworld.LeakError {
	vs := keptViolations(sess.Violations(), narrow)
	if len(vs) == 0 {
		return nil
	}
	return &outworld.LeakError{Violations: vs}
}

func keptViolations(all []outworld.Violation, narrow map[loc]leaf) []outworld.Violation {
	nameHits := map[loc][]outworld.Violation{}
	for _, v := range all {
		at := loc{v.Message, v.Field}
		if l, ok := narrow[at]; ok && nameRecordFinding(l, v) {
			nameHits[at] = append(nameHits[at], v)
		}
	}
	var kept []outworld.Violation
	seen := map[string]bool{}
	runes := map[loc][]rune{}
	runesAt := func(at loc, l leaf) []rune {
		rs, ok := runes[at]
		if !ok {
			rs = []rune(l.text)
			runes[at] = rs
		}
		return rs
	}
	for _, v := range all {
		at := loc{v.Message, v.Field}
		l, checked := narrow[at]
		if checked && (!keptFor(l, runesAt(at, l), v) || (nameRecordFinding(l, v) && vaultNameTokens(l.text, nameHits[at]) < 2)) {
			continue
		}
		v.Field = baseField(v.Field)
		if checked {
			key := spanKey(l, runesAt(at, l), v)
			if seen[key] {
				continue
			}
			seen[key] = true
		}
		if checked || strings.HasSuffix(v.Field, "(key)") {
			if v.Preview != "" {
				v.Preview = redactedPreviewMark
			}
			switch {
			case checked && l.mode == modeVault:
				v.Hint = thinkingHint
			case v.Reason == outworld.ReasonField && !strings.HasSuffix(v.Field, "(key)") && !strings.HasSuffix(v.Field, ".name"):
				v.Hint = checkedFieldHint
			}
		}
		kept = append(kept, v)
	}
	vs := DedupeViolations(kept)
	return append(vs, oversizeFindings(narrow)...)
}

const (
	oversizeKeyed = "keyed"
	keyedHint     = "This value holds more key and value pairs than the gateway checks, so it was not forwarded. Split the value or remove the secrets from it."
)

const oversizeHint = "This opaque value is larger than max_opaque_bytes, so it was not scanned and the request was blocked. Raise max_opaque_bytes in the config if values this large are expected."

func oversizeFindings(narrow map[loc]leaf) []outworld.Violation {
	var out []outworld.Violation
	for at, l := range narrow {
		if l.mode == modeOversize {
			source, hint := "gateway:opaque_too_large", oversizeHint
			switch l.text {
			case oversizeKeyed:
				source, hint = "gateway:too_many_keyed_values", keyedHint
			case oversizeCompressed:
				source, hint = "gateway:compressed_unreadable", compressedHint
			}
			out = append(out, outworld.Violation{Type: "oversize", Source: source, Message: at.msg, Field: baseField(at.path), Start: -1, End: -1, Preview: "[***]", Reason: outworld.ReasonField, Hint: hint})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Message != out[j].Message {
			return out[i].Message < out[j].Message
		}
		return out[i].Field < out[j].Field
	})
	return out
}

func spanKey(l leaf, rs []rune, v outworld.Violation) string {
	base := strings.TrimSuffix(v.Field, "(spaced)")
	span := ""
	if v.Start >= 0 && v.End <= len(rs) && v.Start < v.End {
		span = strings.TrimSpace(string(rs[v.Start:v.End]))
	}
	return strconv.Itoa(v.Message) + "|" + base + "|" + v.Source + "|" + span
}

func maskFailure(err error) (status int, code, msg, kind string) {
	if errors.Is(err, errBadMedia) {
		return http.StatusUnprocessableEntity, "outworld_unsupported_content", err.Error(), "bad_media"
	}
	if errors.Is(err, errNonText) {
		return http.StatusUnprocessableEntity, "outworld_unsupported_content", err.Error(), "non_text"
	}
	return http.StatusBadGateway, "outworld_detection_error", "detection failed; request not sent (fail closed)", "detect"
}

func (s *Server) finalizeRequest(req map[string]any, up *upstream, api string) {
	if api == config.APIAnthropic {
		if !s.cfg.Policy.KeepMetadata {
			if md, ok := req["metadata"].(map[string]any); ok {
				delete(md, "user_id")
			}
		}
	} else {
		delete(req, "user")
	}
	if up != nil && up.cfg.Model != "" {
		req["model"] = up.cfg.Model
	}
}

func writeLeakBlocked(w http.ResponseWriter, api, msg string, findings []outworld.Violation) {
	if findings == nil {
		findings = []outworld.Violation{}
	}
	const code = "outworld_leak_blocked"
	if api == config.APIAnthropic {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"type": "error", "error": map[string]any{
			"type": code, "message": msg, "findings": findings, "help": blockHelp,
		}})
		return
	}
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": map[string]any{
		"message": msg, "type": code, "code": code, "findings": findings, "help": blockHelp,
	}})
}

func summarize(reps []outworld.Replacement) []ReplacementCount {
	type key struct {
		t entity.Type
		s string
		a entity.Action
	}
	counts := map[key]int{}
	for _, r := range reps {
		counts[key{r.Type, r.Source, r.Action}]++
	}
	out := make([]ReplacementCount, 0, len(counts))
	for k, n := range counts {
		out = append(out, ReplacementCount{Type: k.t, Source: k.s, Action: k.a, Count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source
		}
		return out[i].Action < out[j].Action
	})
	return out
}

func (s *Server) dryRun(ctx context.Context, scope, api string, d decision, req map[string]any) (*DryRunReport, outworld.Stats) {
	rep := &DryRunReport{DryRun: true, API: api, Route: d.route, RouteReason: d.reason, Upstream: d.upstream,
		Summary: []ReplacementCount{}, Replacements: []outworld.Replacement{}, Findings: []outworld.Violation{}}
	switch d.route {
	case config.RouteBlock:
		rep.Blocked = "outworld_blocked"
		rep.Message = "request blocked by outworld policy (" + d.reason + ")"
		return rep, outworld.Stats{}
	case config.RouteLocal, config.RoutePassthrough:
		if up := s.upstreams[d.upstream]; up != nil && up.cfg.Model != "" {
			req["model"] = up.cfg.Model
		}
		rep.WouldSend = true
		rep.Request = req
		rep.Message = "this route forwards the request unchanged"
		return rep, outworld.Stats{}
	}
	sess := s.engine.NewSession(scope)
	sess.SetTrace(true)
	narrow, err := s.maskTwice(ctx, sess, req, api)
	stats := sess.Stats()
	if err != nil {
		_, rep.Blocked, rep.Message, _ = maskFailure(err)
		return rep, stats
	}
	for _, r := range sess.Replacements() {
		r.Field = baseField(r.Field)
		rep.Replacements = append(rep.Replacements, r)
	}
	rep.Summary = summarize(rep.Replacements)
	rep.MemoHits = sess.MemoHits()
	s.finalizeRequest(req, s.upstreams[d.upstream], api)
	rep.Request = req
	if lerr := leakOf(sess, narrow); lerr != nil {
		rep.Blocked = "outworld_leak_blocked"
		rep.Message = lerr.Error()
		rep.Findings = lerr.Violations
		rep.Request = nil
		return rep, stats
	}
	rep.WouldSend = true
	return rep, stats
}

func (s *Server) writeDryRun(w http.ResponseWriter, r *http.Request, api string, d decision, req map[string]any, li *logInfo) int {
	rep, stats := s.dryRun(r.Context(), s.scope(r), api, d, req)
	li.stats = stats
	li.violations = rep.Findings
	if rep.Blocked == "outworld_leak_blocked" {
		w.Header().Set(HeaderRoute, "blocked-leak")
	}
	w.Header().Set(HeaderDryRun, "1")
	writeJSON(w, http.StatusOK, rep)
	return http.StatusOK
}

func (s *Server) DryRun(ctx context.Context, api string, body []byte, header http.Header, scope string) (*DryRunReport, error) {
	req, err := decodeObject(body)
	if err != nil {
		return nil, errors.New("request body must be a JSON object")
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "/", nil)
	if err != nil {
		return nil, err
	}
	for k, v := range header {
		r.Header[k] = v
	}
	text := s.routingText(req, api)
	d := decision{route: config.RouteMasked, reason: "no-upstreams"}
	if len(s.upstreams) > 0 {
		if d, err = s.decide(ctx, r, text, api); err != nil {
			return nil, err
		}
	}
	if scope == "" {
		scope = s.scope(r)
	}
	rep, _ := s.dryRun(ctx, scope, api, d, req)
	return rep, nil
}
