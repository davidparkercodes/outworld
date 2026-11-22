package gateway

import (
	"context"
	"encoding/json"
	"errors"
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

	d, err := s.decide(r.Context(), r, anthropicText(req), config.APIAnthropic)
	if err != nil {
		li.status, li.errKind = http.StatusBadGateway, "route"
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

func anthropicText(req map[string]any) string {
	var parts []string
	add := func(v any) {
		if s, ok := v.(string); ok && s != "" {
			parts = append(parts, s)
		}
	}
	var leaves func(v any)
	leaves = func(v any) {
		switch t := v.(type) {
		case string:
			add(t)
		case []any:
			for _, x := range t {
				leaves(x)
			}
		case map[string]any:
			for _, x := range t {
				leaves(x)
			}
		}
	}
	var content func(v any)
	content = func(v any) {
		blocks, ok := v.([]any)
		if !ok {
			add(v)
			return
		}
		for _, b := range blocks {
			m, ok := b.(map[string]any)
			if !ok {
				continue
			}
			typ, _ := m["type"].(string)
			switch typ {
			case "thinking", "redacted_thinking", "image":
			case "tool_use", "server_tool_use", "mcp_tool_use":
				leaves(m["input"])
			default:
				add(m["text"])
				add(m["title"])
				add(m["context"])
				content(m["content"])
				switch src := m["source"].(type) {
				case string:
					add(src)
				case map[string]any:
					if st, _ := src["type"].(string); st == "text" {
						add(src["data"])
					}
					if c, ok := src["content"]; ok {
						content(c)
					}
				}
			}
		}
	}
	content(req["system"])
	if msgs, ok := req["messages"].([]any); ok {
		for _, m := range msgs {
			if msg, ok := m.(map[string]any); ok {
				content(msg["content"])
			}
		}
	}
	return strings.Join(parts, "\n")
}

func maskStringField(ctx context.Context, sess *outworld.Session, m map[string]any, key string) error {
	t, ok := m[key].(string)
	if !ok {
		return nil
	}
	out, err := sess.Mask(ctx, t)
	if err != nil {
		return err
	}
	m[key] = out
	return nil
}

func (s *Server) nonText() error {
	if s.cfg.Policy.AllowImages {
		return nil
	}
	return errNonText
}

func (s *Server) maskAnthropicRequest(ctx context.Context, sess *outworld.Session, req map[string]any) error {
	switch sys := req["system"].(type) {
	case string:
		if err := maskStringField(ctx, sess, req, "system"); err != nil {
			return err
		}
	case []any:
		for _, b := range sys {
			blk, ok := b.(map[string]any)
			if !ok {
				continue
			}
			if typ, _ := blk["type"].(string); typ != "text" {
				continue
			}
			if t, _ := blk["text"].(string); strings.HasPrefix(t, anthropicBillingPrefix) {
				continue
			}
			if err := maskStringField(ctx, sess, blk, "text"); err != nil {
				return err
			}
		}
	}
	msgs, _ := req["messages"].([]any)
	for _, m := range msgs {
		msg, ok := m.(map[string]any)
		if !ok {
			continue
		}
		if err := s.maskAnthropicContent(ctx, sess, msg, "content"); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) maskAnthropicContent(ctx context.Context, sess *outworld.Session, holder map[string]any, key string) error {
	switch c := holder[key].(type) {
	case string:
		return maskStringField(ctx, sess, holder, key)
	case []any:
		for _, b := range c {
			blk, ok := b.(map[string]any)
			if !ok {
				continue
			}
			if err := s.maskAnthropicBlock(ctx, sess, blk); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Server) maskAnthropicBlock(ctx context.Context, sess *outworld.Session, blk map[string]any) error {
	typ, _ := blk["type"].(string)
	switch typ {
	case "thinking", "redacted_thinking":
		return nil
	case "text":
		return maskStringField(ctx, sess, blk, "text")
	case "tool_use", "server_tool_use", "mcp_tool_use":
		in, ok := blk["input"]
		if !ok {
			return nil
		}
		nv, err := maskValue(ctx, sess, in)
		if err != nil {
			return err
		}
		blk["input"] = nv
		return nil
	case "tool_result", "mcp_tool_result":
		return s.maskAnthropicContent(ctx, sess, blk, "content")
	case "search_result":
		for _, k := range []string{"title", "source"} {
			if err := maskStringField(ctx, sess, blk, k); err != nil {
				return err
			}
		}
		return s.maskAnthropicContent(ctx, sess, blk, "content")
	case "document":
		for _, k := range []string{"title", "context"} {
			if err := maskStringField(ctx, sess, blk, k); err != nil {
				return err
			}
		}
		src, ok := blk["source"].(map[string]any)
		if !ok {
			return s.nonText()
		}
		switch st, _ := src["type"].(string); st {
		case "text":
			return maskStringField(ctx, sess, src, "data")
		case "content":
			return s.maskAnthropicContent(ctx, sess, src, "content")
		}
		return s.nonText()
	case "image":
		return s.nonText()
	}
	if _, has := blk["source"]; has {
		return s.nonText()
	}
	return nil
}

func (s *Server) forwardMaskedAnthropic(w http.ResponseWriter, r *http.Request, up *upstream, target string, req map[string]any, li *logInfo, countOnly bool) int {
	sess := s.engine.NewSession(s.scope(r))
	sess.SetPriming(true)
	err := s.maskAnthropicRequest(r.Context(), sess, req)
	sess.SetPriming(false)
	if err == nil {
		err = s.maskAnthropicRequest(r.Context(), sess, req)
	}
	li.stats = sess.Stats()
	if err != nil {
		if errors.Is(err, errNonText) {
			writeAnthropicError(w, http.StatusUnprocessableEntity, "outworld_unsupported_content", err.Error())
			li.errKind = "non_text"
			return http.StatusUnprocessableEntity
		}
		writeAnthropicError(w, http.StatusBadGateway, "outworld_detection_error", "detection failed; request not sent (fail closed)")
		li.errKind = "detect"
		return http.StatusBadGateway
	}
	if lerr := sess.Err(); lerr != nil {
		li.violations = sess.Violations()
		w.Header().Set(HeaderRoute, "blocked-leak")
		writeAnthropicError(w, http.StatusUnprocessableEntity, "outworld_leak_blocked", lerr.Error())
		return http.StatusUnprocessableEntity
	}
	if !s.cfg.Policy.KeepMetadata {
		if md, ok := req["metadata"].(map[string]any); ok {
			delete(md, "user_id")
		}
	}
	if up.cfg.Model != "" {
		req["model"] = up.cfg.Model
	}
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
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		writeAnthropicError(w, http.StatusBadGateway, "outworld_upstream_error", "upstream response truncated")
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
