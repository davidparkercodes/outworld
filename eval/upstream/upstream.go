package upstream

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Record struct {
	Time   time.Time
	Path   string
	Header http.Header
	Body   []byte
}

type Recorder struct {
	mu   sync.Mutex
	recs []Record
}

func (r *Recorder) add(rec Record) {
	r.mu.Lock()
	r.recs = append(r.recs, rec)
	r.mu.Unlock()
}

func (r *Recorder) Mark() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.recs)
}

func (r *Recorder) Since(mark int) []Record {
	r.mu.Lock()
	defer r.mu.Unlock()
	if mark >= len(r.recs) {
		return nil
	}
	return append([]Record(nil), r.recs[mark:]...)
}

func (r *Recorder) All() []Record { return r.Since(0) }

type ChatRequest struct {
	Model    string        `json:"model"`
	Stream   bool          `json:"stream"`
	Messages []ChatMessage `json:"messages"`
}

type ChatMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

func (m ChatMessage) Text() string {
	var s string
	if err := json.Unmarshal(m.Content, &s); err == nil {
		return s
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(m.Content, &parts); err == nil {
		var b strings.Builder
		for _, p := range parts {
			b.WriteString(p.Text)
		}
		return b.String()
	}
	return string(m.Content)
}

const (
	EchoMarker     = "ECHO"
	TimelineMarker = "TIMELINE"
	MessageDivider = "\n<<<msg>>>\n"
)

type Fake struct {
	Recorder *Recorder
	Model    string
	Delay    time.Duration
}

func NewFake() *Fake {
	return &Fake{Recorder: &Recorder{}, Model: "fake-model"}
}

func (f *Fake) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/models", f.models)
	mux.HandleFunc("POST /v1/chat/completions", f.chat)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	return mux
}

func (f *Fake) models(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{
		"object": "list",
		"data":   []map[string]any{{"id": f.Model, "object": "model", "created": 0, "owned_by": "outworld-eval"}},
	})
}

func (f *Fake) chat(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	f.Recorder.add(Record{Time: time.Now(), Path: r.URL.Path, Header: r.Header.Clone(), Body: body})
	var req ChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, `{"error":{"message":"bad json"}}`, http.StatusBadRequest)
		return
	}
	if f.Delay > 0 {
		time.Sleep(f.Delay)
	}
	content := Respond(req.Messages)
	model := req.Model
	if model == "" {
		model = f.Model
	}
	if req.Stream {
		streamContent(w, model, content)
		return
	}
	writeJSON(w, map[string]any{
		"id":      "chatcmpl-eval",
		"object":  "chat.completion",
		"created": 0,
		"model":   model,
		"choices": []map[string]any{{
			"index":         0,
			"message":       map[string]any{"role": "assistant", "content": content},
			"finish_reason": "stop",
		}},
		"usage": map[string]any{"prompt_tokens": len(body) / 4, "completion_tokens": len(content) / 4, "total_tokens": (len(body) + len(content)) / 4},
	})
}

var chunkSizes = []int{3, 7, 1, 5, 9, 2, 11, 4, 6, 8}

func Chunks(content string) []string {
	var out []string
	for i, k := 0, 0; i < len(content); k++ {
		n := chunkSizes[k%len(chunkSizes)]
		if i+n > len(content) {
			n = len(content) - i
		}
		out = append(out, content[i:i+n])
		i += n
	}
	return out
}

func streamContent(w http.ResponseWriter, model, content string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	fl, _ := w.(http.Flusher)
	send := func(delta map[string]any, finish any) {
		b, _ := json.Marshal(map[string]any{
			"id": "chatcmpl-eval", "object": "chat.completion.chunk", "created": 0, "model": model,
			"choices": []map[string]any{{"index": 0, "delta": delta, "finish_reason": finish}},
		})
		fmt.Fprintf(w, "data: %s\n\n", b)
		if fl != nil {
			fl.Flush()
		}
	}
	send(map[string]any{"role": "assistant", "content": ""}, nil)
	for _, c := range Chunks(content) {
		send(map[string]any{"content": c}, nil)
	}
	send(map[string]any{}, "stop")
	fmt.Fprint(w, "data: [DONE]\n\n")
	if fl != nil {
		fl.Flush()
	}
}

var (
	reEmail = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	reDate  = regexp.MustCompile(`\b(?:\d{4}-\d{2}-\d{2}|\d{1,2}/\d{1,2}/\d{4}|\d{2}\.\d{2}\.\d{4}|(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\.? \d{1,2}(?:, \d{4})?|\d{1,2} (?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]* \d{4})\b`)
	reName  = regexp.MustCompile(`\b[A-Z][a-z]+(?:[-'][A-Z][a-z]+)?(?: [A-Z][a-z]+(?:[-'][A-Z][a-z]+)?){1,2}\b`)
)

type found struct {
	pos  int
	kind string
	text string
}

func Timeline(text string) []string {
	var fs []found
	for kind, re := range map[string]*regexp.Regexp{"email": reEmail, "date": reDate, "name": reName} {
		for _, m := range re.FindAllStringIndex(text, -1) {
			fs = append(fs, found{m[0], kind, text[m[0]:m[1]]})
		}
	}
	for i := 1; i < len(fs); i++ {
		for j := i; j > 0 && (fs[j].pos < fs[j-1].pos || (fs[j].pos == fs[j-1].pos && fs[j].kind < fs[j-1].kind)); j-- {
			fs[j], fs[j-1] = fs[j-1], fs[j]
		}
	}
	seen := map[string]bool{}
	var out []string
	for _, f := range fs {
		key := f.kind + "\x00" + f.text
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, "["+f.kind+"] "+f.text)
	}
	return out
}

func Respond(msgs []ChatMessage) string {
	var parts []string
	for _, m := range msgs {
		if m.Role == "system" || m.Role == "developer" {
			continue
		}
		parts = append(parts, m.Text())
	}
	all := strings.Join(parts, MessageDivider)
	var b strings.Builder
	b.WriteString(TimelineMarker + "\n")
	for i, line := range Timeline(all) {
		fmt.Fprintf(&b, "%d. %s\n", i+1, line)
	}
	b.WriteString("\n" + EchoMarker + "\n")
	b.WriteString(all)
	return b.String()
}

func EchoSection(response string) string {
	if i := strings.Index(response, "\n"+EchoMarker+"\n"); i >= 0 {
		return response[i+len(EchoMarker)+2:]
	}
	return response
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

type Server struct {
	URL      string
	srv      *http.Server
	Recorder *Recorder
}

func Serve(h http.Handler, rec *Recorder) (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &Server{URL: "http://" + ln.Addr().String(), srv: &http.Server{Handler: h}, Recorder: rec}
	go func() { _ = s.srv.Serve(ln) }()
	return s, nil
}

func (s *Server) Close() { _ = s.srv.Close() }

type Proxy struct {
	Recorder *Recorder
	BaseURL  string
	APIKey   string
	Client   *http.Client
}

func NewProxy(baseURL, apiKey string) *Proxy {
	return &Proxy{Recorder: &Recorder{}, BaseURL: strings.TrimRight(baseURL, "/"), APIKey: apiKey, Client: &http.Client{Timeout: 5 * time.Minute}}
}

func (p *Proxy) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if r.Method == http.MethodPost {
			p.Recorder.add(Record{Time: time.Now(), Path: r.URL.Path, Header: r.Header.Clone(), Body: body})
		}
		path := strings.TrimPrefix(r.URL.Path, "/v1")
		req, err := http.NewRequestWithContext(r.Context(), r.Method, p.BaseURL+path, bytes.NewReader(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		if p.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+p.APIKey)
		}
		resp, err := p.Client.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		for k, vs := range resp.Header {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		fl, _ := w.(http.Flusher)
		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				_, _ = w.Write(buf[:n])
				if fl != nil {
					fl.Flush()
				}
			}
			if err != nil {
				return
			}
		}
	})
}

func RequestText(body []byte) string {
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return string(body)
	}
	var b strings.Builder
	var walk func(any)
	walk = func(x any) {
		switch t := x.(type) {
		case string:
			b.WriteString(t)
			b.WriteString("\n")
		case []any:
			for _, e := range t {
				walk(e)
			}
		case map[string]any:
			for _, e := range t {
				walk(e)
			}
		}
	}
	walk(v)
	return b.String()
}
