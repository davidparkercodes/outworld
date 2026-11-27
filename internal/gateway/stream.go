package gateway

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/davidparkercodes/outworld/internal/config"
	"github.com/davidparkercodes/outworld/pkg/outworld"
)

type streamKey struct {
	choice int
	field  string
	tool   int
}

type streamState struct {
	sess     *outworld.Session
	raw      map[streamKey]*strings.Builder
	over     map[streamKey]bool
	rh       map[streamKey]*outworld.StreamRehydrator
	order    []streamKey
	template map[string]any
}

func (st *streamState) get(k streamKey) *outworld.StreamRehydrator {
	r, ok := st.rh[k]
	if !ok {
		r = st.sess.NewStreamRehydrator(k.field == "arguments")
		st.rh[k] = r
		st.raw[k] = &strings.Builder{}
		st.order = append(st.order, k)
	}
	return r
}

func (st *streamState) push(k streamKey, s string) string {
	r := st.get(k)
	if !st.over[k] {
		if b := st.raw[k]; b.Len()+len(s) > streamTextCap {
			st.over[k] = true
			b.Reset()
		} else {
			b.WriteString(s)
		}
	}
	return r.Push(s)
}

func (st *streamState) remember() {
	for _, k := range st.order {
		raw, out := st.raw[k].String(), st.rh[k].Text()
		if st.over[k] || raw == "" || len(out) >= streamTextCap {
			continue
		}
		if k.field == "arguments" {
			rememberArguments(st.sess, raw, out)
		} else {
			st.sess.Remember(out, raw)
		}
	}
}

var streamTextCap = 1 << 20

func toInt(v any) int {
	switch t := v.(type) {
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	case float64:
		return int(t)
	}
	return 0
}

var textFields = []string{"content", "reasoning_content", "reasoning", "text"}

func (st *streamState) process(chunk map[string]any) {
	choices, _ := chunk["choices"].([]any)
	for _, c := range choices {
		ch, ok := c.(map[string]any)
		if !ok {
			continue
		}
		idx := toInt(ch["index"])
		target := ch
		if d, ok := ch["delta"].(map[string]any); ok {
			target = d
		}
		for _, f := range textFields {
			if s, ok := target[f].(string); ok {
				target[f] = st.push(streamKey{idx, f, -1}, s)
			}
		}
		if tcs, ok := target["tool_calls"].([]any); ok {
			for _, tc := range tcs {
				m, ok := tc.(map[string]any)
				if !ok {
					continue
				}
				fn, ok := m["function"].(map[string]any)
				if !ok {
					continue
				}
				if args, ok := fn["arguments"].(string); ok {
					fn["arguments"] = st.push(streamKey{idx, "arguments", toInt(m["index"])}, args)
				}
			}
		}
		if fr, ok := ch["finish_reason"]; ok && fr != nil {
			st.flushInto(idx, target)
		}
	}
}

func (st *streamState) flushInto(choice int, target map[string]any) {
	for _, k := range st.order {
		if k.choice != choice {
			continue
		}
		rest := st.rh[k].Flush()
		if rest == "" {
			continue
		}
		if k.field != "arguments" {
			prev, _ := target[k.field].(string)
			target[k.field] = prev + rest
			continue
		}
		tcs, _ := target["tool_calls"].([]any)
		placed := false
		for _, tc := range tcs {
			m, ok := tc.(map[string]any)
			if !ok || toInt(m["index"]) != k.tool {
				continue
			}
			fn, _ := m["function"].(map[string]any)
			if fn == nil {
				fn = map[string]any{}
				m["function"] = fn
			}
			prev, _ := fn["arguments"].(string)
			fn["arguments"] = prev + rest
			placed = true
		}
		if !placed {
			target["tool_calls"] = append(tcs, map[string]any{"index": k.tool, "function": map[string]any{"arguments": rest}})
		}
	}
}

func (st *streamState) pendingChunk() map[string]any {
	choicesSeen := map[int]bool{}
	for _, k := range st.order {
		if st.rh[k].Pending() > 0 {
			choicesSeen[k.choice] = true
		}
	}
	if len(choicesSeen) == 0 {
		return nil
	}
	var idxs []int
	for i := range choicesSeen {
		idxs = append(idxs, i)
	}
	sort.Ints(idxs)
	var choices []any
	for _, i := range idxs {
		delta := map[string]any{}
		st.flushInto(i, delta)
		ch := map[string]any{"index": i, "finish_reason": nil}
		if _, isText := delta["text"]; isText {
			ch["text"] = delta["text"]
		} else {
			ch["delta"] = delta
		}
		choices = append(choices, ch)
	}
	out := map[string]any{}
	for _, k := range []string{"id", "object", "created", "model", "system_fingerprint"} {
		if v, ok := st.template[k]; ok {
			out[k] = v
		}
	}
	out["choices"] = choices
	return out
}

func (st *streamState) visibleText() string {
	var b strings.Builder
	for _, k := range st.order {
		if k.field == "arguments" {
			continue
		}
		b.WriteString(st.rh[k].Text())
		b.WriteByte('\n')
	}
	return b.String()
}

func (s *Server) streamRehydrate(w http.ResponseWriter, resp *http.Response, sess *outworld.Session) []string {
	copyHeaders(w.Header(), resp.Header)
	w.Header().Set("Trailer", HeaderUnknownNames)
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(resp.StatusCode)
	fl, _ := w.(http.Flusher)
	flush := func() {
		if fl != nil {
			fl.Flush()
		}
	}
	st := &streamState{sess: sess, rh: map[streamKey]*outworld.StreamRehydrator{}, raw: map[streamKey]*strings.Builder{}, over: map[streamKey]bool{}}
	finish := func() []string {
		if pc := st.pendingChunk(); pc != nil {
			io.WriteString(w, "data: "+string(encode(pc))+"\n\n")
		}
		st.remember()
		names := sess.UnknownNames(st.visibleText())
		if len(names) > 0 {
			io.WriteString(w, ": outworld unknown_names="+strings.Join(names, ",")+"\n\n")
		}
		return names
	}
	var names []string
	done := false
	event := func(lines []string) {
		var data []string
		for _, l := range lines {
			if d, ok := sseData(l); ok {
				data = append(data, d)
			}
		}
		payload := strings.Join(data, "\n")
		if len(data) > 0 && strings.TrimSpace(payload) == "[DONE]" {
			names = finish()
			done = true
			writeLines(w, lines)
			return
		}
		chunk, derr := decodeObject([]byte(payload))
		if len(data) == 0 || derr != nil {
			writeLines(w, lines)
			return
		}
		if st.template == nil {
			st.template = chunk
		}
		st.process(chunk)
		replaceData(w, lines, encode(chunk))
	}
	stop, drained := s.closeOnDrain(resp.Body)
	defer stop()
	br := bufio.NewReaderSize(resp.Body, 64*1024)
	var ev []string
	for {
		line, err := br.ReadString('\n')
		if line != "" {
			ev = append(ev, line)
			if strings.HasSuffix(line, "\n") && strings.TrimRight(line, "\r\n") == "" {
				event(ev)
				ev = nil
				flush()
			}
		}
		if err != nil {
			break
		}
	}
	if drained() && !done {
		writeShutdownEvent(w, config.APIOpenAI)
		flush()
		return nil
	}
	if len(ev) > 0 {
		event(ev)
		if !strings.HasSuffix(ev[len(ev)-1], "\n") {
			io.WriteString(w, "\n")
		}
		io.WriteString(w, "\n")
	}
	if !done {
		names = finish()
	}
	flush()
	if len(names) > 0 {
		w.Header().Set(HeaderUnknownNames, strings.Join(names, ", "))
	}
	return names
}

func writeLines(w io.Writer, lines []string) {
	for _, l := range lines {
		io.WriteString(w, l)
	}
}

func replaceData(w io.Writer, lines []string, payload []byte) {
	written := false
	for _, l := range lines {
		if _, ok := sseData(l); !ok {
			io.WriteString(w, l)
			continue
		}
		if written {
			continue
		}
		written = true
		io.WriteString(w, "data: "+string(payload)+l[len(strings.TrimRight(l, "\r\n")):])
	}
}
