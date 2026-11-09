package runner

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/davidparkercodes/outworld/eval/corpus"
)

type Result struct {
	Content string
	Status  int
	Unknown int
	Latency time.Duration
	Err     error
}

func Chat(ctx context.Context, c *http.Client, v1Base, apiKey, model string, msgs []corpus.Message, stream bool) Result {
	payload := map[string]any{"model": model, "messages": msgs, "stream": stream}
	if !stream {
		payload["temperature"] = 0
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(v1Base, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Result{Err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
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
	if stream && strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		res.Content, res.Err = readStream(resp.Body, &res.Unknown)
	} else {
		var out map[string]any
		b, err := io.ReadAll(resp.Body)
		if err == nil {
			err = json.Unmarshal(b, &out)
		}
		if err != nil {
			res.Err = fmt.Errorf("decode response: %w", err)
		} else {
			res.Content = messageContent(out, "message")
			res.Unknown += unknownFromBody(out)
		}
	}
	res.Unknown += unknownFromHeaders(resp.Trailer)
	res.Latency = time.Since(start)
	return res
}

func readStream(r io.Reader, unknown *int) (string, error) {
	var b strings.Builder
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	sawDone := false
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			sawDone = true
			break
		}
		var chunk map[string]any
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return b.String(), fmt.Errorf("bad SSE chunk: %w", err)
		}
		if e, ok := chunk["error"]; ok {
			return b.String(), fmt.Errorf("stream error: %v", e)
		}
		b.WriteString(messageContent(chunk, "delta"))
		*unknown += unknownFromBody(chunk)
	}
	if err := sc.Err(); err != nil {
		return b.String(), err
	}
	if !sawDone {
		return b.String(), fmt.Errorf("stream ended without [DONE]")
	}
	return b.String(), nil
}

func messageContent(m map[string]any, field string) string {
	choices, _ := m["choices"].([]any)
	if len(choices) == 0 {
		return ""
	}
	ch, _ := choices[0].(map[string]any)
	msg, _ := ch[field].(map[string]any)
	s, _ := msg["content"].(string)
	return s
}

func isUnknownKey(k string) bool {
	k = strings.ToLower(k)
	return strings.Contains(k, "unknown") || strings.Contains(k, "hallucinat") || strings.Contains(k, "unresolved")
}

func unknownFromHeaders(h http.Header) int {
	n := 0
	for k, vs := range h {
		if !strings.Contains(strings.ToLower(k), "outworld") || !isUnknownKey(k) {
			continue
		}
		for _, v := range vs {
			if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				n += i
			} else if strings.TrimSpace(v) != "" {
				n += len(strings.Split(v, ","))
			}
		}
	}
	return n
}

func unknownFromBody(m map[string]any) int {
	ow, ok := m["outworld"].(map[string]any)
	if !ok {
		return 0
	}
	n := 0
	for k, v := range ow {
		if !isUnknownKey(k) {
			continue
		}
		switch t := v.(type) {
		case float64:
			n += int(t)
		case []any:
			n += len(t)
		}
	}
	return n
}
