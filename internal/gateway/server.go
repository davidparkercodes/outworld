package gateway

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/davidparkercodes/outworld/internal/config"
	"github.com/davidparkercodes/outworld/pkg/outworld"
)

const (
	HeaderRoute        = "X-Outworld-Route"
	HeaderRouteReason  = "X-Outworld-Route-Reason"
	HeaderScope        = "X-Outworld-Scope"
	HeaderMasked       = "X-Outworld-Masked"
	HeaderUnknownNames = "X-Outworld-Unknown-Names"
	HeaderToken        = "X-Outworld-Token"
)

type upstream struct {
	name    string
	cfg     config.Upstream
	apiKey  string
	client  *http.Client
	baseURL string
}

type Server struct {
	cfg       *config.Config
	engine    *outworld.Engine
	upstreams map[string]*upstream
	log       *slog.Logger
	token     string
	draining  chan struct{}
	drainOnce sync.Once
	inflight  sync.WaitGroup
	cborScan  func([]byte) ([]string, []string, bool)
}

type Options struct {
	Logger    *slog.Logger
	Transport http.RoundTripper
}

func New(cfg *config.Config, engine *outworld.Engine, o Options) (*Server, error) {
	if o.Logger == nil {
		o.Logger = slog.New(slog.NewJSONHandler(os.Stderr, nil))
	}
	s := &Server{cfg: cfg, engine: engine, upstreams: map[string]*upstream{}, log: o.Logger, draining: make(chan struct{})}
	if cfg.Auth.TokenEnv != "" {
		s.token = strings.TrimSpace(os.Getenv(cfg.Auth.TokenEnv))
	}
	for name, u := range cfg.Upstreams {
		key := ""
		if u.APIKeyEnv != "" {
			key = os.Getenv(u.APIKeyEnv)
			if key == "" {
				o.Logger.Warn("upstream api key env var is empty", "upstream", name, "env", u.APIKeyEnv)
			}
		}
		tr := o.Transport
		if tr == nil {
			ht := time.Duration(u.HeaderTimeout)
			if ht <= 0 {
				ht = 180 * time.Second
			}
			tr = &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				DialContext:           (&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
				ResponseHeaderTimeout: ht,
				IdleConnTimeout:       90 * time.Second,
				MaxIdleConnsPerHost:   16,
				ForceAttemptHTTP2:     true,
			}
		}
		s.upstreams[name] = &upstream{name: name, cfg: u, apiKey: key, client: &http.Client{Transport: tr}, baseURL: strings.TrimRight(u.BaseURL, "/")}
	}
	return s, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("HEAD /api/hello", s.hello)
	mux.HandleFunc("POST /v1/chat/completions", s.auth(writeError, s.chat))
	mux.HandleFunc("POST /v1/completions", s.auth(writeError, s.completions))
	mux.HandleFunc("GET /v1/models", s.auth(writeError, s.models))
	mux.HandleFunc("POST /v1/messages", s.auth(writeAnthropicError, s.messages))
	mux.HandleFunc("POST /v1/messages/count_tokens", s.auth(writeAnthropicError, s.countTokens))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.inflight.Add(1)
		defer s.inflight.Done()
		mux.ServeHTTP(w, r)
	})
}

func (s *Server) closeOnDrain(body io.Closer) (stop func(), drained func() bool) {
	var hit atomic.Bool
	stop = s.onDrain(func() {
		hit.Store(true)
		body.Close()
	})
	return stop, hit.Load
}

const shutdownMessage = "outworld is shutting down; this response was cut off before the upstream finished"

func writeShutdownEvent(w io.Writer, api string) {
	if api == config.APIAnthropic {
		io.WriteString(w, "event: error\ndata: "+string(encode(map[string]any{"type": "error", "error": map[string]any{"type": "overloaded_error", "message": shutdownMessage}}))+"\n\n")
		return
	}
	io.WriteString(w, "data: "+string(encode(map[string]any{"error": map[string]any{"message": shutdownMessage, "type": "outworld_shutdown", "code": "outworld_shutdown"}}))+"\n\n")
}

func (s *Server) onDrain(fn func()) func() {
	done := make(chan struct{})
	go func() {
		select {
		case <-s.draining:
			fn()
		case <-done:
		}
	}()
	return func() { close(done) }
}

func (s *Server) hello(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "vault_entries": s.engine.Vault().Len(), "memo": s.engine.MemoStats()})
}

type errorWriter func(w http.ResponseWriter, status int, code, msg string)

func bearerValue(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 7 && strings.EqualFold(v[:7], "Bearer ") {
		v = strings.TrimSpace(v[7:])
	}
	return v
}

func (s *Server) isGatewayToken(v string) bool {
	if s.token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(bearerValue(v)), []byte(s.token)) == 1
}

func (s *Server) auth(werr errorWriter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.token != "" {
			ok := false
			for _, h := range []string{"Authorization", "X-Api-Key", HeaderToken} {
				if s.isGatewayToken(r.Header.Get(h)) {
					ok = true
				}
			}
			if !ok {
				werr(w, http.StatusUnauthorized, "outworld_unauthorized", "missing or invalid gateway token")
				return
			}
		}
		next(w, r)
	}
}

func (s *Server) models(w http.ResponseWriter, r *http.Request) {
	name := s.cfg.Policy.MaskedUpstream
	if s.cfg.Policy.DefaultRoute == config.RouteLocal && s.cfg.Policy.LocalUpstream != "" {
		name = s.cfg.Policy.LocalUpstream
	}
	anthropic := r.Header.Get("Anthropic-Version") != "" || name == ""
	if anthropic && s.cfg.Policy.Anthropic.MaskedUpstream != "" {
		name = s.cfg.Policy.Anthropic.MaskedUpstream
	}
	up, ok := s.upstreams[name]
	if !ok {
		writeError(w, http.StatusBadGateway, "outworld_no_upstream", "no upstream configured")
		return
	}
	target := up.baseURL + "/models"
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "outworld_internal", "could not build request")
		return
	}
	if up.cfg.APIKind() == config.APIAnthropic {
		forwardAnthropicHeaders(req.Header, r.Header)
	}
	s.authorize(up, req, r)
	resp, err := up.client.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "outworld_upstream_error", "upstream unreachable")
		return
	}
	defer resp.Body.Close()
	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		s.log.Warn("models response copy failed", "upstream", name, "error", err.Error())
	}
}

func (s *Server) authorize(up *upstream, req, client *http.Request) {
	switch {
	case up.cfg.ForwardAuth:
		for _, h := range []string{"Authorization", "X-Api-Key"} {
			if v := client.Header.Get(h); v != "" && !s.isGatewayToken(v) {
				req.Header.Set(h, v)
			}
		}
	case up.apiKey == "":
	case up.cfg.APIKind() == config.APIAnthropic:
		req.Header.Set("X-Api-Key", up.apiKey)
	default:
		req.Header.Set("Authorization", "Bearer "+up.apiKey)
	}
	for k, v := range up.cfg.Headers {
		req.Header.Set(k, v)
	}
}

var hopHeaders = map[string]bool{
	"Connection": true, "Keep-Alive": true, "Proxy-Authenticate": true, "Proxy-Authorization": true,
	"Te": true, "Trailer": true, "Transfer-Encoding": true, "Upgrade": true, "Content-Length": true,
	"Content-Encoding": true, "Set-Cookie": true,
}

func copyHeaders(dst, src http.Header) {
	for k, vs := range src {
		if hopHeaders[http.CanonicalHeaderKey(k)] {
			continue
		}
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"message": msg, "type": code, "code": code}})
}

var (
	MaxHeaderBytes  = 64 << 10
	ShutdownTimeout = 30 * time.Second
	HandlerWait     = 10 * time.Second
)

func (s *Server) ListenAndServe(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.cfg.Listen)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	return s.Serve(ctx, ln)
}

func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	base, cancelBase := context.WithCancel(context.Background())
	defer cancelBase()
	srv := &http.Server{Handler: s.Handler(), ReadHeaderTimeout: 30 * time.Second, IdleTimeout: 120 * time.Second, MaxHeaderBytes: MaxHeaderBytes,
		BaseContext: func(net.Listener) context.Context { return base }}
	s.log.Info("outworld listening", "addr", ln.Addr().String(), "default_route", s.cfg.Policy.DefaultRoute, "auth", s.token != "")
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
	}
	s.log.Info("outworld shutting down")
	s.drainOnce.Do(func() { close(s.draining) })
	sctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil {
		cancelBase()
		srv.Close()
		s.log.Warn("shutdown deadline passed, open requests cancelled", "error", err.Error())
	}
	<-errc
	done := make(chan struct{})
	go func() {
		s.inflight.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(HandlerWait):
		s.log.Warn("handlers still running after shutdown")
	}
	return nil
}
