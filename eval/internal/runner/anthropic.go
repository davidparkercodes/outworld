package runner

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/davidparkercodes/outworld/eval/corpus"
)

const (
	APIOpenAI    = "openai"
	APIAnthropic = "anthropic"

	anthropicVersion   = "2023-06-01"
	anthropicMaxTokens = 4096
	unknownComment     = "outworld unknown_names="
)

func Send(ctx context.Context, c *http.Client, api, v1Base, apiKey, model string, msgs []corpus.Message, stream bool) Result {
	if api == APIAnthropic {
		return Messages(ctx, c, v1Base, apiKey, model, msgs, stream)
	}
	return Chat(ctx, c, v1Base, apiKey, model, msgs, stream)
}

func MessagesPayload(model string, msgs []corpus.Message, stream bool) map[string]any {
	var system []string
	turns := []map[string]any{}
	for _, m := range msgs {
		if m.Role == "system" || m.Role == "developer" {
			system = append(system, m.Content)
			continue
		}
		turns = append(turns, map[string]any{"role": m.Role, "content": m.Content})
	}
	payload := map[string]any{"model": model, "max_tokens": anthropicMaxTokens, "messages": turns, "stream": stream}
	if len(system) > 0 {
		payload["system"] = strings.Join(system, "\n\n")
	}
	if !stream {
		payload["temperature"] = 0
	}
	return payload
}

func Messages(ctx context.Context, c *http.Client, v1Base, apiKey, model string, msgs []corpus.Message, stream bool) Result {
	body, _ := json.Marshal(MessagesPayload(model, msgs, stream))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(v1Base, "/")+"/messages", bytes.NewReader(body))
	if err != nil {
		return Result{Err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", anthropicVersion)
	if apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
	}
	start := time.Now()
	resp, err := c.Do(req)
	if err != nil {
		return Result{Err: err, Latency: time.Since(start)}
	}
	defer resp.Body.Close()
	res := Result{Status: resp.StatusCode, Unknown: unknownFromHeaders(resp.Header)}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		res.Latency = time.Since(start)
		res.Err = fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
		return res
	}
	comment := 0
	if stream && strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		res.Content, res.Err = readMessageStream(resp.Body, &res.Unknown, &comment)
	} else {
		var out map[string]any
		b, err := io.ReadAll(resp.Body)
		if err == nil {
			err = json.Unmarshal(b, &out)
		}
		if err != nil {
			res.Err = fmt.Errorf("decode response: %w", err)
		} else if e, ok := out["error"]; ok && out["type"] == "error" {
			res.Err = fmt.Errorf("error response: %v", e)
		} else {
			res.Content = blocksText(out["content"])
			res.Unknown += unknownFromBody(out)
		}
	}
	res.Unknown += max(comment, unknownFromHeaders(resp.Trailer))
	res.Latency = time.Since(start)
	return res
}

func readMessageStream(r io.Reader, unknown, comment *int) (string, error) {
	var b strings.Builder
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	stopped := false
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if n, ok := unknownFromComment(line); ok {
			*comment += n
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		var ev map[string]any
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			return b.String(), fmt.Errorf("bad SSE event: %w", err)
		}
		*unknown += unknownFromBody(ev)
		switch ev["type"] {
		case "error":
			return b.String(), fmt.Errorf("stream error: %v", ev["error"])
		case "content_block_delta":
			d, _ := ev["delta"].(map[string]any)
			if d["type"] == "text_delta" {
				s, _ := d["text"].(string)
				b.WriteString(s)
			}
		case "content_block_start":
			cb, _ := ev["content_block"].(map[string]any)
			if cb["type"] == "text" {
				s, _ := cb["text"].(string)
				b.WriteString(s)
			}
		case "message_stop":
			stopped = true
		}
	}
	if err := sc.Err(); err != nil {
		return b.String(), err
	}
	if !stopped {
		return b.String(), fmt.Errorf("stream ended without message_stop")
	}
	return b.String(), nil
}

func blocksText(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	blocks, _ := v.([]any)
	var b strings.Builder
	for _, x := range blocks {
		m, _ := x.(map[string]any)
		if m["type"] == "text" {
			s, _ := m["text"].(string)
			b.WriteString(s)
		}
	}
	return b.String()
}

func unknownFromComment(line string) (int, bool) {
	if !strings.HasPrefix(line, ":") {
		return 0, false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, ":"))
	if !strings.HasPrefix(rest, unknownComment) {
		return 0, true
	}
	n := 0
	for _, p := range strings.Split(strings.TrimPrefix(rest, unknownComment), ",") {
		if strings.TrimSpace(p) != "" {
			n++
		}
	}
	return n, true
}
