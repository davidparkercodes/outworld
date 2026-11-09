package gateway

import (
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
}

type Options struct {
	Logger    *slog.Logger
	Transport http.RoundTripper
}

func New(cfg *config.Config, engine *outworld.Engine, o Options) (*Server, error) {
	if o.Logger == nil {
		o.Logger = slog.New(slog.NewJSONHandler(os.Stderr, nil))
	}
	s := &Server{cfg: cfg, engine: engine, upstreams: map[string]*upstream{}, log: o.Logger}
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
	mux.HandleFunc("POST /v1/chat/completions", s.auth(s.chat))
	mux.HandleFunc("POST /v1/completions", s.auth(s.completions))
	mux.HandleFunc("GET /v1/models", s.auth(s.models))
	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "vault_entries": s.engine.Vault().Len()})
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.token != "" {
			got := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
			if subtle.ConstantTimeCompare([]byte(got), []byte(s.token)) != 1 {
				writeError(w, http.StatusUnauthorized, "outworld_unauthorized", "missing or invalid bearer token")
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
	up, ok := s.upstreams[name]
	if !ok {
		writeError(w, http.StatusBadGateway, "outworld_no_upstream", "no upstream configured")
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, up.baseURL+"/models", nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "outworld_internal", "could not build request")
		return
	}
	up.authorize(req)
	resp, err := up.client.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "outworld_upstream_error", "upstream unreachable")
		return
	}
	defer resp.Body.Close()
	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (u *upstream) authorize(req *http.Request) {
	if u.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+u.apiKey)
	}
	for k, v := range u.cfg.Headers {
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

func (s *Server) ListenAndServe() error {
	srv := &http.Server{Addr: s.cfg.Listen, Handler: s.Handler(), ReadHeaderTimeout: 30 * time.Second}
	s.log.Info("outworld listening", "addr", s.cfg.Listen, "default_route", s.cfg.Policy.DefaultRoute, "auth", s.token != "")
	err := srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return fmt.Errorf("listen: %w", err)
}
