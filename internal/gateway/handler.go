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
	"sort"
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
	unknownNames            int
	errKind                 string
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request, path string) {
	start := time.Now()
	li := &logInfo{}
	defer func() { s.logRequest(path, start, li) }()

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

	d, err := s.decide(r.Context(), r, req)
	if err != nil {
		li.status, li.errKind = http.StatusBadGateway, "route"
		w.Header().Set(HeaderRoute, config.RouteBlock)
		writeError(w, li.status, "outworld_route_error", err.Error())
		return
	}
	li.route, li.reason, li.upstream = d.route, d.reason, d.upstream
	w.Header().Set(HeaderRoute, d.route)
	w.Header().Set(HeaderRouteReason, d.reason)

	switch d.route {
	case config.RouteBlock:
		li.status = http.StatusForbidden
		writeError(w, li.status, "outworld_blocked", "request blocked by outworld policy ("+d.reason+")")
	case config.RouteLocal, config.RoutePassthrough:
		li.status = s.forwardRaw(w, r, s.upstreams[d.upstream], path, body, req)
	case config.RouteMasked:
		li.status = s.forwardMasked(w, r, s.upstreams[d.upstream], path, req, li)
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

func requestText(req map[string]any) string {
	var parts []string
	var walk func(v any)
	walk = func(v any) {
		switch t := v.(type) {
		case string:
			parts = append(parts, t)
		case []any:
			for _, x := range t {
				walk(x)
			}
		case map[string]any:
			for k, x := range t {
				if k == "role" || k == "type" || k == "tool_call_id" || k == "id" {
					continue
				}
				walk(x)
			}
		}
	}
	walk(req["messages"])
	walk(req["prompt"])
	return strings.Join(parts, "\n")
}

func (s *Server) upstreamFor(route, explicit string) (string, error) {
	name := explicit
	if name == "" {
		switch route {
		case config.RouteMasked:
			name = s.cfg.Policy.MaskedUpstream
		case config.RouteLocal:
			name = s.cfg.Policy.LocalUpstream
		case config.RoutePassthrough:
			name = s.cfg.Policy.PassthroughUpstream
		case config.RouteBlock:
			return "", nil
		}
	}
	if name == "" {
		return "", fmt.Errorf("route %q has no upstream configured", route)
	}
	if _, ok := s.upstreams[name]; !ok {
		return "", fmt.Errorf("route %q refers to unknown upstream %q", route, name)
	}
	return name, nil
}

func (s *Server) decide(ctx context.Context, r *http.Request, req map[string]any) (decision, error) {
	text := requestText(req)
	var detected map[entity.Type]bool
	var detectErr error
	detectTypes := func() (map[entity.Type]bool, error) {
		if detected != nil || detectErr != nil {
			return detected, detectErr
		}
		spans, err := s.engine.Detect(ctx, text)
		if err != nil {
			detectErr = err
			return nil, err
		}
		detected = map[entity.Type]bool{}
		for _, sp := range spans {
			detected[sp.Type] = true
		}
		return detected, nil
	}
	ruleMatches := func(rule *config.Rule) (bool, error) {
		if rule.MatchText(text) {
			return true, nil
		}
		if len(rule.EntityTypes()) == 0 {
			return false, nil
		}
		types, err := detectTypes()
		if err != nil {
			return false, err
		}
		for _, t := range rule.EntityTypes() {
			if types[t] {
				return true, nil
			}
		}
		return false, nil
	}
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
	if h := strings.TrimSpace(r.Header.Get(HeaderRoute)); h != "" && s.cfg.RouteHeaderAllowed() {
		route, up, _ := strings.Cut(strings.ToLower(h), ":")
		route = strings.TrimSpace(route)
		switch route {
		case config.RouteMasked, config.RouteLocal, config.RoutePassthrough, config.RouteBlock:
		default:
			return decision{}, fmt.Errorf("invalid %s header value", HeaderRoute)
		}
		name, err := s.upstreamFor(route, strings.TrimSpace(up))
		if err != nil {
			return decision{}, err
		}
		return decision{route: route, upstream: name, reason: "header"}, nil
	}
	for i := range rules {
		ok, err := ruleMatches(&rules[i])
		if err != nil {
			return decision{}, err
		}
		if ok {
			name, err := s.upstreamFor(rules[i].Route, rules[i].Upstream)
			if err != nil {
				return decision{}, err
			}
			return decision{route: rules[i].Route, upstream: name, reason: ruleName(rules[i], i)}, nil
		}
	}
	name, err := s.upstreamFor(s.cfg.Policy.DefaultRoute, "")
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
	up.authorize(req)
	return req, nil
}

func (s *Server) forwardRaw(w http.ResponseWriter, r *http.Request, up *upstream, path string, body []byte, req map[string]any) int {
	if up.cfg.Model != "" {
		req["model"] = up.cfg.Model
		body = encode(req)
	}
	ureq, err := s.newUpstreamRequest(r.Context(), up, path, body, r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "outworld_internal", "could not build upstream request")
		return http.StatusInternalServerError
	}
	resp, err := up.client.Do(ureq)
	if err != nil {
		writeError(w, http.StatusBadGateway, "outworld_upstream_error", "upstream unreachable")
		return http.StatusBadGateway
	}
	defer resp.Body.Close()
	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	fl, _ := w.(http.Flusher)
	buf := make([]byte, 32*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				break
			}
			if fl != nil {
				fl.Flush()
			}
		}
		if rerr != nil {
			break
		}
	}
	return resp.StatusCode
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
	sess.SetPriming(true)
	err := s.maskRequest(r.Context(), sess, req)
	sess.SetPriming(false)
	if err == nil {
		err = s.maskRequest(r.Context(), sess, req)
	}
	li.stats = sess.Stats()
	if err != nil {
		if errors.Is(err, errNonText) {
			writeError(w, http.StatusUnprocessableEntity, "outworld_unsupported_content", err.Error())
			li.errKind = "non_text"
			return http.StatusUnprocessableEntity
		}
		writeError(w, http.StatusBadGateway, "outworld_detection_error", "detection failed; request not sent (fail closed)")
		li.errKind = "detect"
		return http.StatusBadGateway
	}
	if lerr := sess.Err(); lerr != nil {
		li.violations = sess.Violations()
		w.Header().Set(HeaderRoute, "blocked-leak")
		writeError(w, http.StatusUnprocessableEntity, "outworld_leak_blocked", lerr.Error())
		return http.StatusUnprocessableEntity
	}
	delete(req, "user")
	if up.cfg.Model != "" {
		req["model"] = up.cfg.Model
	}
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
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		writeError(w, http.StatusBadGateway, "outworld_upstream_error", "upstream response truncated")
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

func rehydrateBody(sess *outworld.Session, raw []byte) ([]byte, []string) {
	obj, err := decodeObject(raw)
	if err != nil {
		text := sess.Rehydrate(string(raw))
		return []byte(text), sess.UnknownNames(text)
	}
	var texts []string
	var walk func(v any, key string) any
	walk = func(v any, key string) any {
		switch t := v.(type) {
		case string:
			if key == "arguments" {
				return sess.RehydrateArguments(t)
			}
			out := sess.Rehydrate(t)
			if key == "content" || key == "text" || key == "reasoning_content" || key == "message" {
				texts = append(texts, out)
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
			obj[k] = walk(v, k)
		}
	}
	names := sess.UnknownNames(strings.Join(texts, "\n"))
	return encode(obj), names
}

func (s *Server) maskRequest(ctx context.Context, sess *outworld.Session, req map[string]any) error {
	if msgs, ok := req["messages"].([]any); ok {
		for _, m := range msgs {
			msg, ok := m.(map[string]any)
			if !ok {
				continue
			}
			if err := s.maskMessage(ctx, sess, msg); err != nil {
				return err
			}
		}
	}
	for _, k := range []string{"prompt", "suffix", "input"} {
		v, ok := req[k]
		if !ok {
			continue
		}
		nv, err := maskValue(ctx, sess, v)
		if err != nil {
			return err
		}
		req[k] = nv
	}
	return nil
}

func (s *Server) maskMessage(ctx context.Context, sess *outworld.Session, msg map[string]any) error {
	switch c := msg["content"].(type) {
	case string:
		out, err := sess.Mask(ctx, c)
		if err != nil {
			return err
		}
		msg["content"] = out
	case []any:
		for _, p := range c {
			part, ok := p.(map[string]any)
			if !ok {
				continue
			}
			typ, _ := part["type"].(string)
			switch typ {
			case "text", "input_text", "output_text":
				if t, ok := part["text"].(string); ok {
					out, err := sess.Mask(ctx, t)
					if err != nil {
						return err
					}
					part["text"] = out
				}
			case "refusal":
			default:
				if !s.cfg.Policy.AllowImages {
					return errNonText
				}
			}
		}
	}
	for _, k := range []string{"reasoning_content", "refusal"} {
		if t, ok := msg[k].(string); ok {
			out, err := sess.Mask(ctx, t)
			if err != nil {
				return err
			}
			msg[k] = out
		}
	}
	if name, ok := msg["name"].(string); ok {
		sess.Check(ctx, name)
	}
	if tcs, ok := msg["tool_calls"].([]any); ok {
		for _, tc := range tcs {
			if m, ok := tc.(map[string]any); ok {
				if err := maskFunction(ctx, sess, m["function"]); err != nil {
					return err
				}
			}
		}
	}
	return maskFunction(ctx, sess, msg["function_call"])
}

func maskFunction(ctx context.Context, sess *outworld.Session, f any) error {
	fn, ok := f.(map[string]any)
	if !ok {
		return nil
	}
	args, ok := fn["arguments"].(string)
	if !ok {
		return nil
	}
	out, err := maskArguments(ctx, sess, args)
	if err != nil {
		return err
	}
	fn["arguments"] = out
	return nil
}

func maskArguments(ctx context.Context, sess *outworld.Session, args string) (string, error) {
	v, err := decodeAny([]byte(args))
	if err != nil {
		return sess.Mask(ctx, args)
	}
	nv, err := maskValue(ctx, sess, v)
	if err != nil {
		return "", err
	}
	return string(encode(nv)), nil
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

func maskValue(ctx context.Context, sess *outworld.Session, v any) (any, error) {
	switch t := v.(type) {
	case string:
		return sess.Mask(ctx, t)
	case []any:
		for i := range t {
			nv, err := maskValue(ctx, sess, t[i])
			if err != nil {
				return nil, err
			}
			t[i] = nv
		}
		return t, nil
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sess.Check(ctx, k)
			nv, err := maskValue(ctx, sess, t[k])
			if err != nil {
				return nil, err
			}
			t[k] = nv
		}
		return t, nil
	}
	return v, nil
}
