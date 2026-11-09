package shim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/davidparkercodes/outworld/eval/corpus"
	"github.com/davidparkercodes/outworld/eval/score"
	"github.com/davidparkercodes/outworld/eval/upstream"
)

type Shim struct {
	UpstreamV1 string
	mask       *strings.Replacer
	unmask     *strings.Replacer
}

func Passthrough(upstreamV1 string) *Shim {
	return &Shim{UpstreamV1: upstreamV1}
}

func Oracle(upstreamV1 string, items []corpus.Item, pol score.Policy) *Shim {
	sur := map[string]string{}
	restore := map[string]string{}
	n := 0
	for _, it := range items {
		for _, s := range it.Spans {
			if !pol.Sensitive(s.Type) {
				continue
			}
			if _, ok := sur[s.Value]; ok {
				continue
			}
			if pol[s.Type] == score.ActionStrip {
				sur[s.Value] = "[REDACTED]"
				continue
			}
			n++
			tok := fmt.Sprintf("Zq%dqZ", n)
			sur[s.Value] = tok
			restore[tok] = s.Value
		}
	}
	keys := make([]string, 0, len(sur))
	for k := range sur {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) != len(keys[j]) {
			return len(keys[i]) > len(keys[j])
		}
		return keys[i] < keys[j]
	})
	var fwd, back []string
	for _, k := range keys {
		fwd = append(fwd, k, sur[k])
	}
	toks := make([]string, 0, len(restore))
	for k := range restore {
		toks = append(toks, k)
	}
	sort.Slice(toks, func(i, j int) bool {
		return len(toks[i]) > len(toks[j]) || (len(toks[i]) == len(toks[j]) && toks[i] < toks[j])
	})
	for _, t := range toks {
		back = append(back, t, restore[t])
	}
	return &Shim{UpstreamV1: upstreamV1, mask: strings.NewReplacer(fwd...), unmask: strings.NewReplacer(back...)}
}

func (s *Shim) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("POST /v1/chat/completions", s.chat)
	return mux
}

func (s *Shim) chat(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	stream, _ := req["stream"].(bool)
	req["stream"] = false
	if s.mask != nil {
		msgs, _ := req["messages"].([]any)
		for _, m := range msgs {
			mm, _ := m.(map[string]any)
			if c, ok := mm["content"].(string); ok {
				mm["content"] = s.mask.Replace(c)
			}
		}
	}
	body, _ := json.Marshal(req)
	resp, err := http.Post(strings.TrimRight(s.UpstreamV1, "/")+"/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		http.Error(w, "bad upstream response", http.StatusBadGateway)
		return
	}
	content := ""
	if ch, ok := out["choices"].([]any); ok && len(ch) > 0 {
		msg, _ := ch[0].(map[string]any)["message"].(map[string]any)
		content, _ = msg["content"].(string)
	}
	if s.unmask != nil {
		content = s.unmask.Replace(content)
	}
	if !stream {
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(map[string]any{
			"object":  "chat.completion",
			"choices": []any{map[string]any{"index": 0, "message": map[string]any{"role": "assistant", "content": content}, "finish_reason": "stop"}},
		})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	for _, c := range upstream.Chunks(content) {
		b, _ := json.Marshal(map[string]any{"object": "chat.completion.chunk", "choices": []any{map[string]any{"index": 0, "delta": map[string]any{"content": c}}}})
		fmt.Fprintf(w, "data: %s\n\n", b)
	}
	fmt.Fprint(w, "data: [DONE]\n\n")
}
