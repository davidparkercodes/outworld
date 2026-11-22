package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/davidparkercodes/outworld/eval/corpus"
	"github.com/davidparkercodes/outworld/eval/upstream"
)

var msgs = []corpus.Message{
	{Role: "system", Content: "task"},
	{Role: "user", Content: "Call Marisol Quenby."},
	{Role: "assistant", Content: "Ok."},
	{Role: "user", Content: "Thanks."},
}

func TestMessagesClient(t *testing.T) {
	f := upstream.NewFake()
	var got map[string]any
	var hdr http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		hdr = r.Header.Clone()
		r.Body = io.NopCloser(bytes.NewReader(body))
		f.Handler().ServeHTTP(w, r)
	}))
	defer srv.Close()
	want := "Call Marisol Quenby." + upstream.MessageDivider + "Ok." + upstream.MessageDivider + "Thanks."
	for _, stream := range []bool{false, true} {
		res := Send(context.Background(), srv.Client(), APIAnthropic, srv.URL+"/v1", "k", "m", msgs, stream)
		if res.Err != nil {
			t.Fatalf("stream=%v: %v", stream, res.Err)
		}
		if upstream.EchoSection(res.Content) != want {
			t.Fatalf("stream=%v: echo %q", stream, upstream.EchoSection(res.Content))
		}
		if got["system"] != "task" || got["max_tokens"] == nil || len(got["messages"].([]any)) != 3 {
			t.Fatalf("payload not in Anthropic shape: %v", got)
		}
		if hdr.Get("anthropic-version") != "2023-06-01" || hdr.Get("x-api-key") != "k" {
			t.Fatalf("headers: %v", hdr)
		}
	}
}

func TestUnknownNamesReporting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Trailer", "X-Outworld-Unknown-Names")
		if r.URL.Path == "/v1/messages" {
			fmt.Fprint(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hi\"}}\n\n")
			fmt.Fprint(w, ": outworld unknown_names=Ada,Bo\n\n")
			fmt.Fprint(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
		} else {
			fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi\"}}]}\n\n")
			fmt.Fprint(w, ": outworld unknown_names=Ada,Bo\n\n")
			fmt.Fprint(w, "data: [DONE]\n\n")
		}
		w.Header().Set("X-Outworld-Unknown-Names", "Ada, Bo")
	}))
	defer srv.Close()
	for _, api := range []string{APIOpenAI, APIAnthropic} {
		res := Send(context.Background(), srv.Client(), api, srv.URL+"/v1", "", "m", msgs, true)
		if res.Err != nil || res.Content != "hi" || res.Unknown != 2 {
			t.Fatalf("%s: content=%q unknown=%d err=%v", api, res.Content, res.Unknown, res.Err)
		}
	}
}

func TestMessagesStreamNeedsStop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: ping\ndata: {\"type\":\"ping\"}\n\n")
	}))
	defer srv.Close()
	if res := Send(context.Background(), srv.Client(), APIAnthropic, srv.URL+"/v1", "", "m", msgs, true); res.Err == nil {
		t.Fatal("a stream without message_stop must be an error")
	}
}
