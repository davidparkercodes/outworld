package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/davidparkercodes/outworld/internal/config"
	"github.com/davidparkercodes/outworld/pkg/outworld"
)

const (
	anthropicBillingPrefix  = "x-anthropic-billing-header"
	anthropicDefaultVersion = "2023-06-01"
)

func writeAnthropicError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{"type": "error", "error": map[string]any{"type": code, "message": msg}})
}

func (s *Server) messages(w http.ResponseWriter, r *http.Request) {
	s.handleAnthropic(w, r, "/messages", false)
}

func (s *Server) countTokens(w http.ResponseWriter, r *http.Request) {
	s.handleAnthropic(w, r, "/messages/count_tokens", true)
}

func forwardAnthropicHeaders(dst, src http.Header) {
	for k, vs := range src {
		l := strings.ToLower(k)
		if !(strings.HasPrefix(l, "anthropic-") || strings.HasPrefix(l, "x-stainless-") || strings.HasPrefix(l, "x-claude-code-") || l == "user-agent" || l == "x-app") {
			continue
		}
		dst.Del(k)
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
	if dst.Get("Anthropic-Version") == "" {
		dst.Set("Anthropic-Version", anthropicDefaultVersion)
	}
}

func (s *Server) handleAnthropic(w http.ResponseWriter, r *http.Request, path string, countOnly bool) {
	start := time.Now()
	li := &logInfo{}
	defer func() { s.logRequest(path, start, li) }()
	if s.scopeTooLong(r) {
		li.status, li.errKind = http.StatusBadRequest, "scope"
		writeAnthropicError(w, li.status, "outworld_bad_request", HeaderScope+" is longer than 256 bytes")
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, s.cfg.MaxBodyBytes))
	if err != nil {
		li.status, li.errKind = http.StatusRequestEntityTooLarge, "body"
		writeAnthropicError(w, li.status, "outworld_bad_request", "could not read request body")
		return
	}
	req, err := decodeObject(body)
	if err != nil {
		li.status, li.errKind = http.StatusBadRequest, "json"
		writeAnthropicError(w, li.status, "outworld_bad_request", "request body must be a JSON object")
		return
	}
	li.stream, _ = req["stream"].(bool)

	d, err := s.decide(r.Context(), r, s.routingText(req, config.APIAnthropic), config.APIAnthropic)
	if err != nil {
		li.status, li.errKind = routeStatus(err), "route"
		w.Header().Set(HeaderRoute, config.RouteBlock)
		writeAnthropicError(w, li.status, "outworld_route_error", err.Error())
		return
	}
	li.route, li.reason, li.upstream = d.route, d.reason, d.upstream
	w.Header().Set(HeaderRoute, d.route)
	w.Header().Set(HeaderRouteReason, d.reason)

	target := path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	if isDryRun(r) {
		li.dryRun = true
		li.status = s.writeDryRun(w, r, config.APIAnthropic, d, req, li)
		return
	}
	switch d.route {
	case config.RouteBlock:
		li.status = http.StatusForbidden
		writeAnthropicError(w, li.status, "outworld_blocked", "request blocked by outworld policy ("+d.reason+")")
	case config.RouteLocal, config.RoutePassthrough:
		li.status = s.forwardRaw(w, r, s.upstreams[d.upstream], target, body, req, writeAnthropicError)
	case config.RouteMasked:
		li.status = s.forwardMaskedAnthropic(w, r, s.upstreams[d.upstream], target, req, li, countOnly)
	}
}

func (s *Server) forwardMaskedAnthropic(w http.ResponseWriter, r *http.Request, up *upstream, target string, req map[string]any, li *logInfo, countOnly bool) int {
	sess := s.engine.NewSession(s.scope(r))
	narrow, err := s.maskTwice(r.Context(), sess, req, config.APIAnthropic)
	li.stats = sess.Stats()
	if err != nil {
		status, code, msg, kind := maskFailure(err)
		writeAnthropicError(w, status, code, msg)
		li.errKind = kind
		return status
	}
	if lerr := leakOf(sess, narrow); lerr != nil {
		li.violations = lerr.Violations
		w.Header().Set(HeaderRoute, "blocked-leak")
		writeLeakBlocked(w, config.APIAnthropic, lerr.Error(), li.violations)
		return http.StatusUnprocessableEntity
	}
	s.finalizeRequest(req, up, config.APIAnthropic)
	stream, _ := req["stream"].(bool)
	ureq, err := s.newUpstreamRequest(r.Context(), up, target, encode(req), r)
	if err != nil {
		writeAnthropicError(w, http.StatusInternalServerError, "outworld_internal", "could not build upstream request")
		return http.StatusInternalServerError
	}
	w.Header().Set(HeaderMasked, strconv.Itoa(li.stats.Masked+li.stats.Stripped+li.stats.Shifted+li.stats.Generalized))
	resp, err := up.client.Do(ureq)
	if err != nil {
		writeAnthropicError(w, http.StatusBadGateway, "outworld_upstream_error", "upstream unreachable")
		return http.StatusBadGateway
	}
	defer resp.Body.Close()
	ok := resp.StatusCode >= 200 && resp.StatusCode < 300
	if stream && !countOnly && ok && strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		names := s.streamAnthropic(w, resp, sess)
		li.unknownNames = len(names)
		return resp.StatusCode
	}
	raw, err := readResponse(resp.Body)
	if err != nil {
		writeAnthropicError(w, http.StatusBadGateway, "outworld_upstream_error", responseReadError(err))
		return http.StatusBadGateway
	}
	copyHeaders(w.Header(), resp.Header)
	out := raw
	var names []string
	switch {
	case !ok:
		out = rehydrateRawBody(sess, raw)
	case !countOnly:
		out, names = rehydrateAnthropicBody(sess, raw)
	}
	li.unknownNames = len(names)
	if len(names) > 0 {
		w.Header().Set(HeaderUnknownNames, strings.Join(names, ", "))
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(out)
	return resp.StatusCode
}

func rehydrateRawBody(sess *outworld.Session, raw []byte) []byte {
	text := string(raw)
	var out string
	if json.Valid(raw) {
		out = sess.RehydrateJSONText(text)
	} else {
		out = sess.Rehydrate(text)
	}
	if out == text {
		return raw
	}
	return []byte(out)
}

func isToolUseBlock(typ string) bool {
	return typ == "tool_use" || typ == "server_tool_use" || typ == "mcp_tool_use"
}

func rehydrateAnthropicBody(sess *outworld.Session, raw []byte) ([]byte, []string) {
	obj, err := decodeObject(raw)
	if err != nil {
		out := rehydrateRawBody(sess, raw)
		return out, sess.UnknownNames(string(out))
	}
	content, _ := obj["content"].([]any)
	var texts []string
	changed := false
	for _, b := range content {
		blk, ok := b.(map[string]any)
		if !ok {
			continue
		}
		typ, _ := blk["type"].(string)
		switch {
		case typ == "text":
			t, ok := blk["text"].(string)
			if !ok {
				continue
			}
			out := sess.Rehydrate(t)
			sess.Remember(out, t)
			texts = append(texts, out)
			if out != t {
				blk["text"] = out
				changed = true
			}
		case isToolUseBlock(typ):
			if in, ok := blk["input"]; ok {
				blk["input"] = rehydrateRemember(sess, in, &changed)
			}
		}
	}
	names := sess.UnknownNames(strings.Join(texts, "\n"))
	if !changed {
		return raw, names
	}
	return encode(obj), names
}

func rehydrateRemember(sess *outworld.Session, v any, changed *bool) any {
	switch t := v.(type) {
	case string:
		out := sess.Rehydrate(t)
		sess.Remember(out, t)
		if out != t {
			*changed = true
		}
		return out
	case []any:
		for i := range t {
			t[i] = rehydrateRemember(sess, t[i], changed)
		}
	case map[string]any:
		for k, x := range t {
			t[k] = rehydrateRemember(sess, x, changed)
		}
	}
	return v
}

func rememberLeaves(sess *outworld.Session, upstream, rehydrated any) {
	switch u := upstream.(type) {
	case string:
		if r, ok := rehydrated.(string); ok {
			sess.Remember(r, u)
		}
	case []any:
		r, ok := rehydrated.([]any)
		if !ok {
			return
		}
		for i := 0; i < len(u) && i < len(r); i++ {
			rememberLeaves(sess, u[i], r[i])
		}
	case map[string]any:
		r, ok := rehydrated.(map[string]any)
		if !ok {
			return
		}
		for k, x := range u {
			if y, ok := r[k]; ok {
				rememberLeaves(sess, x, y)
			}
		}
	}
}

func rememberJSON(sess *outworld.Session, upstream, rehydrated string) bool {
	u, err := decodeAny([]byte(upstream))
	if err != nil {
		return false
	}
	r, err := decodeAny([]byte(rehydrated))
	if err != nil {
		return false
	}
	rememberLeaves(sess, u, r)
	return true
}

func rememberArguments(sess *outworld.Session, upstream, rehydrated string) {
	if !rememberJSON(sess, upstream, rehydrated) {
		sess.Remember(rehydrated, upstream)
	}
}
