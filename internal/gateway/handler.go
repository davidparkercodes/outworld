package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/davidparkercodes/outworld/internal/config"
	"github.com/davidparkercodes/outworld/pkg/entity"
	"github.com/davidparkercodes/outworld/pkg/outworld"
)

type decision struct {
	route    string
	upstream string
	reason   string
}

var errBadMedia = errors.New("request contains image, audio or file data that does not match its declared type; the gateway only forwards media it can verify")

var errNonText = errors.New("request contains non-text content (images, audio or files) which cannot be masked; set policy.allow_images or use the local route")

func (s *Server) chat(w http.ResponseWriter, r *http.Request) { s.handle(w, r, "/chat/completions") }

func (s *Server) completions(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, "/completions")
}

type logInfo struct {
	route, reason, upstream string
	status                  int
	stream                  bool
	stats                   outworld.Stats
	violations              []outworld.Violation
	dryRun                  bool
	unknownNames            int
	errKind                 string
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request, path string) {
	start := time.Now()
	li := &logInfo{}
	defer func() { s.logRequest(path, start, li) }()
	if s.scopeTooLong(r) {
		li.status, li.errKind = http.StatusBadRequest, "scope"
		writeError(w, li.status, "outworld_bad_request", HeaderScope+" is longer than 256 bytes")
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, s.cfg.MaxBodyBytes))
	if err != nil {
		li.status, li.errKind = http.StatusRequestEntityTooLarge, "body"
		writeError(w, li.status, "outworld_bad_request", "could not read request body")
		return
	}
	req, err := decodeObject(body)
	if err != nil {
		li.status, li.errKind = http.StatusBadRequest, "json"
		writeError(w, li.status, "outworld_bad_request", "request body must be a JSON object")
		return
	}
	li.stream, _ = req["stream"].(bool)

	target := path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	d, err := s.decide(r.Context(), r, s.routingText(req, config.APIOpenAI), config.APIOpenAI)
	if err != nil {
		li.status, li.errKind = routeStatus(err), "route"
		w.Header().Set(HeaderRoute, config.RouteBlock)
		writeError(w, li.status, "outworld_route_error", err.Error())
		return
	}
	li.route, li.reason, li.upstream = d.route, d.reason, d.upstream
	w.Header().Set(HeaderRoute, d.route)
	w.Header().Set(HeaderRouteReason, d.reason)

	if isDryRun(r) {
		li.dryRun = true
		li.status = s.writeDryRun(w, r, config.APIOpenAI, d, req, li)
		return
	}
	switch d.route {
	case config.RouteBlock:
		li.status = http.StatusForbidden
		writeError(w, li.status, "outworld_blocked", "request blocked by outworld policy ("+d.reason+")")
	case config.RouteLocal, config.RoutePassthrough:
		li.status = s.forwardRaw(w, r, s.upstreams[d.upstream], target, body, req, writeError)
	case config.RouteMasked:
		li.status = s.forwardMasked(w, r, s.upstreams[d.upstream], target, req, li)
	}
}

func (s *Server) logRequest(path string, start time.Time, li *logInfo) {
	attrs := []any{
		"path", path, "route", li.route, "reason", li.reason, "upstream", li.upstream,
		"status", li.status, "stream", li.stream, "duration_ms", time.Since(start).Milliseconds(),
	}
	if li.route == config.RouteMasked {
		types := map[string]int{}
		for t, n := range li.stats.ByType {
			types[string(t)] = n
		}
		attrs = append(attrs, "masked", li.stats.Masked, "stripped", li.stats.Stripped, "shifted", li.stats.Shifted,
			"generalized", li.stats.Generalized, "entities", types, "unknown_names", li.unknownNames)
	}
	if len(li.violations) > 0 {
		vt := map[string]int{}
		for _, v := range li.violations {
			vt[string(v.Type)]++
		}
		attrs = append(attrs, "leak_violations", vt)
	}
	if li.errKind != "" {
		attrs = append(attrs, "error", li.errKind)
	}
	if li.dryRun {
		attrs = append(attrs, "dry_run", true)
	}
	s.log.Info("request", attrs...)
}

func decodeObject(b []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errors.New("not an object")
	}
	return m, nil
}

func encode(v any) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.Encode(v)
	return bytes.TrimRight(buf.Bytes(), "\n")
}

func (s *Server) upstreamFor(route, explicit, api string) (string, error) {
	if route == config.RouteBlock {
		return "", nil
	}
	name := explicit
	if name == "" {
		ru := config.RouteUpstreams{
			MaskedUpstream:      s.cfg.Policy.MaskedUpstream,
			LocalUpstream:       s.cfg.Policy.LocalUpstream,
			PassthroughUpstream: s.cfg.Policy.PassthroughUpstream,
		}
		if api == config.APIAnthropic {
			ru = s.cfg.Policy.Anthropic
		}
		switch route {
		case config.RouteMasked:
			name = ru.MaskedUpstream
		case config.RouteLocal:
			name = ru.LocalUpstream
		case config.RoutePassthrough:
			name = ru.PassthroughUpstream
		}
	}
	if name == "" {
		return "", fmt.Errorf("route %q has no upstream configured for %s-format requests", route, api)
	}
	up, ok := s.upstreams[name]
	if !ok {
		return "", fmt.Errorf("route %q refers to unknown upstream %q", route, name)
	}
	if kind := up.cfg.APIKind(); kind != api {
		return "", fmt.Errorf("upstream %q speaks the %s api and cannot serve a %s-format request", name, kind, api)
	}
	return name, nil
}

func (s *Server) entityTypes(ctx context.Context, rt routeText) (map[entity.Type]bool, error) {
	found := map[entity.Type]bool{}
	scan := func(text string, keep func(entity.Span) bool) error {
		if text == "" {
			return nil
		}
		spans, err := s.engine.Detect(ctx, text)
		if err != nil {
			return err
		}
		for _, sp := range spans {
			if keep(sp) {
				found[sp.Type] = true
			}
		}
		return nil
	}
	all := func(entity.Span) bool { return true }
	if err := scan(rt.full, all); err != nil {
		return nil, err
	}
	if err := scan(rt.narrow, func(sp entity.Span) bool { return narrowSource(sp.Source) }); err != nil {
		return nil, err
	}
	if err := scan(rt.decoded, func(sp entity.Span) bool { return !opaqueNoise(sp.Source) }); err != nil {
		return nil, err
	}
	for _, run := range rt.binary {
		if err := scan(run, func(sp entity.Span) bool { return binarySources[sp.Source] }); err != nil {
			return nil, err
		}
	}
	return found, nil
}

type ruleMatcher func(rule *config.Rule) (bool, error)

func (s *Server) matcher(ctx context.Context, rt routeText) ruleMatcher {
	var detected map[entity.Type]bool
	var detectErr error
	return func(rule *config.Rule) (bool, error) {
		if rule.MatchText(rt.full) || rule.MatchText(rt.narrow) || rule.MatchText(rt.raw) || rule.MatchText(rt.decoded) || binaryBlocks(rt, rule) {
			return true, nil
		}
		if len(rule.EntityTypes()) == 0 {
			return false, nil
		}
		if detected == nil && detectErr == nil {
			detected, detectErr = s.entityTypes(ctx, rt)
		}
		if detectErr != nil {
			return false, detectErr
		}
		for _, t := range rule.EntityTypes() {
			if detected[t] {
				return true, nil
			}
		}
		return false, nil
	}
}

func (s *Server) decide(ctx context.Context, r *http.Request, rt routeText, api string) (decision, error) {
	ruleMatches := s.matcher(ctx, rt)
	rules := s.cfg.Policy.Rules
	for i := range rules {
		if rules[i].Route != config.RouteBlock {
			continue
		}
		ok, err := ruleMatches(&rules[i])
		if err != nil {
			return decision{}, err
		}
		if ok {
			return decision{route: config.RouteBlock, reason: ruleName(rules[i], i)}, nil
		}
	}
	h := strings.TrimSpace(r.Header.Get(HeaderRoute))
	if h == "" || !s.cfg.RouteHeaderAllowed() {
		return s.ruleDecision(rules, ruleMatches, api)
	}
	return s.headerDecision(h, rules, ruleMatches, api)
}

func (s *Server) headerDecision(h string, rules []config.Rule, ruleMatches ruleMatcher, api string) (decision, error) {
	route, up, _ := strings.Cut(h, ":")
	route = strings.ToLower(strings.TrimSpace(route))
	up = strings.TrimSpace(up)
	if _, ok := routeRank[route]; !ok {
		return decision{}, &routeError{http.StatusBadRequest, fmt.Sprintf("invalid %s header value", HeaderRoute)}
	}
	if s.cfg.RouteHeaderUnrestricted() {
		name, err := s.upstreamFor(route, up, api)
		if err != nil {
			return decision{}, &routeError{http.StatusBadRequest, err.Error()}
		}
		return decision{route: route, upstream: name, reason: "header"}, nil
	}
	if route == config.RouteBlock {
		return decision{route: config.RouteBlock, reason: "header"}, nil
	}
	d, err := s.ruleDecision(rules, ruleMatches, api)
	if err != nil {
		return decision{}, err
	}
	if up != "" && !(route == d.route && up == d.upstream) {
		def, err := s.upstreamFor(route, "", api)
		if err != nil {
			return decision{}, &routeError{http.StatusBadRequest, err.Error()}
		}
		if def != up {
			return decision{}, &routeError{http.StatusForbidden, fmt.Sprintf("%s may only name the upstream that route already uses unless policy.allow_route_header is true", HeaderRoute)}
		}
	}
	switch {
	case routeRank[route] < routeRank[d.route]:
		return decision{}, &routeError{http.StatusForbidden, fmt.Sprintf("%s may only make routing stricter (block, then local, then masked, then passthrough) unless policy.allow_route_header is true", HeaderRoute)}
	case routeRank[route] == routeRank[d.route]:
		return d, nil
	}
	name, err := s.upstreamFor(route, "", api)
	if err != nil {
		return decision{}, &routeError{http.StatusBadRequest, err.Error()}
	}
	return decision{route: route, upstream: name, reason: "header"}, nil
}

var routeRank = map[string]int{config.RouteBlock: 3, config.RouteLocal: 2, config.RouteMasked: 1, config.RoutePassthrough: 0}

type routeError struct {
	status int
	msg    string
}

func (e *routeError) Error() string { return e.msg }

func routeStatus(err error) int {
	var re *routeError
	if errors.As(err, &re) {
		return re.status
	}
	return http.StatusBadGateway
}

func (s *Server) ruleDecision(rules []config.Rule, ruleMatches ruleMatcher, api string) (decision, error) {
	for i := range rules {
		ok, err := ruleMatches(&rules[i])
		if err != nil {
			return decision{}, err
		}
		if ok {
			name, err := s.upstreamFor(rules[i].Route, rules[i].Upstream, api)
			if err != nil {
				return decision{}, err
			}
			return decision{route: rules[i].Route, upstream: name, reason: ruleName(rules[i], i)}, nil
		}
	}
	name, err := s.upstreamFor(s.cfg.Policy.DefaultRoute, "", api)
	if err != nil {
		return decision{}, err
	}
	return decision{route: s.cfg.Policy.DefaultRoute, upstream: name, reason: "default"}, nil
}

var safeRuleName = regexp.MustCompile(`[^A-Za-z0-9_.-]`)

func ruleName(r config.Rule, i int) string {
	if r.Name != "" {
		return "rule:" + safeRuleName.ReplaceAllString(r.Name, "_")
	}
	return "rule:" + strconv.Itoa(i)
}

func (s *Server) newUpstreamRequest(ctx context.Context, up *upstream, path string, body []byte, clientReq *http.Request) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, up.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if a := clientReq.Header.Get("Accept"); a != "" {
		req.Header.Set("Accept", a)
	}
	if up.cfg.APIKind() == config.APIAnthropic {
		forwardAnthropicHeaders(req.Header, clientReq.Header)
	}
	s.authorize(up, req, clientReq)
	return req, nil
}

func (s *Server) forwardRaw(w http.ResponseWriter, r *http.Request, up *upstream, path string, body []byte, req map[string]any, werr errorWriter) int {
	if up.cfg.Model != "" {
		req["model"] = up.cfg.Model
		body = encode(req)
	}
	ureq, err := s.newUpstreamRequest(r.Context(), up, path, body, r)
	if err != nil {
		werr(w, http.StatusInternalServerError, "outworld_internal", "could not build upstream request")
		return http.StatusInternalServerError
	}
	resp, err := up.client.Do(ureq)
	if err != nil {
		werr(w, http.StatusBadGateway, "outworld_upstream_error", "upstream unreachable")
		return http.StatusBadGateway
	}
	defer resp.Body.Close()
	drained := func() bool { return false }
	if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		var stop func()
		stop, drained = s.closeOnDrain(resp.Body)
		defer stop()
	}
	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	fl, _ := w.(http.Flusher)
	buf := make([]byte, 32*1024)
	done := false
	tail := []byte("\n")
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				break
			}
			tail = append(tail, buf[:n]...)
			if streamFinished(tail) {
				done = true
			}
			if len(tail) > 64 {
				tail = append(tail[:0], tail[len(tail)-64:]...)
			}
			if fl != nil {
				fl.Flush()
			}
		}
		if rerr != nil {
			break
		}
	}
	if drained() && !done {
		io.WriteString(w, "\n\n")
		writeShutdownEvent(w, up.cfg.APIKind())
		if fl != nil {
			fl.Flush()
		}
	}
	return resp.StatusCode
}

func streamFinished(b []byte) bool {
	return bytes.Contains(b, []byte("\ndata: [DONE]")) || bytes.Contains(b, []byte("\ndata:[DONE]")) || bytes.Contains(b, []byte("\nevent: message_stop"))
}

const maxScopeLen = 256

func (s *Server) scopeTooLong(r *http.Request) bool {
	return s.cfg.Policy.AllowScopeHeader && len(strings.TrimSpace(r.Header.Get(HeaderScope))) > maxScopeLen
}

func (s *Server) scope(r *http.Request) string {
	if s.cfg.Policy.AllowScopeHeader {
		if h := strings.TrimSpace(r.Header.Get(HeaderScope)); h != "" {
			return h
		}
	}
	return s.cfg.Policy.DateShiftScope
}

func (s *Server) forwardMasked(w http.ResponseWriter, r *http.Request, up *upstream, path string, req map[string]any, li *logInfo) int {
	sess := s.engine.NewSession(s.scope(r))
	narrow, err := s.maskTwice(r.Context(), sess, req, config.APIOpenAI)
	li.stats = sess.Stats()
	if err != nil {
		status, code, msg, kind := maskFailure(err)
		writeError(w, status, code, msg)
		li.errKind = kind
		return status
	}
	if lerr := leakOf(sess, narrow); lerr != nil {
		li.violations = lerr.Violations
		w.Header().Set(HeaderRoute, "blocked-leak")
		writeLeakBlocked(w, config.APIOpenAI, lerr.Error(), li.violations)
		return http.StatusUnprocessableEntity
	}
	s.finalizeRequest(req, up, config.APIOpenAI)
	stream, _ := req["stream"].(bool)
	ureq, err := s.newUpstreamRequest(r.Context(), up, path, encode(req), r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "outworld_internal", "could not build upstream request")
		return http.StatusInternalServerError
	}
	w.Header().Set(HeaderMasked, strconv.Itoa(li.stats.Masked+li.stats.Stripped+li.stats.Shifted+li.stats.Generalized))
	resp, err := up.client.Do(ureq)
	if err != nil {
		writeError(w, http.StatusBadGateway, "outworld_upstream_error", "upstream unreachable")
		return http.StatusBadGateway
	}
	defer resp.Body.Close()
	if stream && resp.StatusCode == http.StatusOK && strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		names := s.streamRehydrate(w, resp, sess)
		li.unknownNames = len(names)
		return resp.StatusCode
	}
	raw, err := readResponse(resp.Body)
	if err != nil {
		writeError(w, http.StatusBadGateway, "outworld_upstream_error", responseReadError(err))
		return http.StatusBadGateway
	}
	copyHeaders(w.Header(), resp.Header)
	out, names := rehydrateBody(sess, raw)
	li.unknownNames = len(names)
	if len(names) > 0 {
		w.Header().Set(HeaderUnknownNames, strings.Join(names, ", "))
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(out)
	return resp.StatusCode
}

var maxResponseBytes int64 = 64 << 20

var errResponseTooLarge = errors.New("upstream response too large")

func readResponse(r io.Reader) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(r, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxResponseBytes {
		return nil, errResponseTooLarge
	}
	return raw, nil
}

func responseReadError(err error) string {
	if errors.Is(err, errResponseTooLarge) {
		return "upstream response exceeds " + strconv.FormatInt(maxResponseBytes, 10) + " bytes"
	}
	return "upstream response truncated"
}

func rehydrateBody(sess *outworld.Session, raw []byte) ([]byte, []string) {
	obj, err := decodeObject(raw)
	if err != nil {
		text := sess.Rehydrate(string(raw))
		return []byte(text), sess.UnknownNames(text)
	}
	var texts []string
	remember := false
	var walk func(v any, key string) any
	walk = func(v any, key string) any {
		switch t := v.(type) {
		case string:
			if key == "arguments" {
				out := sess.RehydrateArguments(t)
				if remember {
					rememberArguments(sess, t, out)
				}
				return out
			}
			out := sess.Rehydrate(t)
			if key == "content" || key == "text" || key == "reasoning_content" || key == "message" {
				texts = append(texts, out)
			}
			if remember && (key == "content" || key == "text" || key == "reasoning_content" || key == "reasoning" || key == "refusal" || key == "input") {
				sess.Remember(out, t)
			}
			return out
		case []any:
			for i := range t {
				t[i] = walk(t[i], key)
			}
			return t
		case map[string]any:
			for k, x := range t {
				t[k] = walk(x, k)
			}
			return t
		}
		return v
	}
	for _, k := range []string{"choices", "error"} {
		if v, ok := obj[k]; ok {
			remember = k == "choices"
			obj[k] = walk(v, k)
		}
	}
	names := sess.UnknownNames(strings.Join(texts, "\n"))
	return encode(obj), names
}

func decodeAny(b []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	if dec.More() {
		return nil, errors.New("trailing data")
	}
	return v, nil
}
