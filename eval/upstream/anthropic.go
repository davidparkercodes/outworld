package upstream

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const MessageID = "msg_eval"

type MessagesRequest struct {
	Model    string          `json:"model"`
	Stream   bool            `json:"stream"`
	System   json.RawMessage `json:"system"`
	Messages []ChatMessage   `json:"messages"`
}

func (f *Fake) messages(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	f.Recorder.add(Record{Time: time.Now(), Path: r.URL.Path, Header: r.Header.Clone(), Body: body})
	var req MessagesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"type":"error","error":{"type":"invalid_request_error","message":"bad json"}}`)
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
		StreamMessage(w, model, content, len(body)/4)
		return
	}
	writeJSON(w, MessageBody(model, content, len(body)/4))
}

func MessageBody(model, content string, inputTokens int) map[string]any {
	return map[string]any{
		"id":            MessageID,
		"type":          "message",
		"role":          "assistant",
		"model":         model,
		"content":       []map[string]any{{"type": "text", "text": content}},
		"stop_reason":   "end_turn",
		"stop_sequence": nil,
		"usage":         map[string]any{"input_tokens": inputTokens, "output_tokens": len(content) / 4},
	}
}

func StreamMessage(w http.ResponseWriter, model, content string, inputTokens int) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	fl, _ := w.(http.Flusher)
	send := func(event string, data map[string]any) {
		data["type"] = event
		b, _ := json.Marshal(data)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
		if fl != nil {
			fl.Flush()
		}
	}
	send("message_start", map[string]any{"message": map[string]any{
		"id": MessageID, "type": "message", "role": "assistant", "model": model, "content": []any{},
		"stop_reason": nil, "stop_sequence": nil,
		"usage": map[string]any{"input_tokens": inputTokens, "output_tokens": 1},
	}})
	send("content_block_start", map[string]any{"index": 0, "content_block": map[string]any{"type": "text", "text": ""}})
	send("ping", map[string]any{})
	for _, c := range Chunks(content) {
		send("content_block_delta", map[string]any{"index": 0, "delta": map[string]any{"type": "text_delta", "text": c}})
	}
	send("content_block_stop", map[string]any{"index": 0})
	send("message_delta", map[string]any{
		"delta": map[string]any{"stop_reason": "end_turn", "stop_sequence": nil},
		"usage": map[string]any{"output_tokens": len(content) / 4},
	})
	send("message_stop", map[string]any{})
}
