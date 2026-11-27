package gateway

import (
	"bufio"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/davidparkercodes/outworld/internal/config"
	"github.com/davidparkercodes/outworld/pkg/outworld"
)

type anthropicChannel struct {
	rh   *outworld.StreamRehydrator
	raw  strings.Builder
	out  strings.Builder
	over bool
}

func (c *anthropicChannel) push(s string) string {
	o := c.rh.Push(s)
	c.record(s, o)
	return o
}

func (c *anthropicChannel) record(raw, out string) {
	if c.over {
		return
	}
	if c.raw.Len()+len(raw) > streamTextCap || c.out.Len()+len(out) > streamTextCap {
		c.over = true
		c.raw.Reset()
		return
	}
	c.raw.WriteString(raw)
	c.out.WriteString(out)
}

type anthropicBlock struct {
	text *anthropicChannel
	json *anthropicChannel
}

type anthropicStream struct {
	sess    *outworld.Session
	w       io.Writer
	blocks  map[int]*anthropicBlock
	visible strings.Builder
	stopped bool
	names   []string
}

func (st *anthropicStream) block(idx int) *anthropicBlock {
	b, ok := st.blocks[idx]
	if !ok {
		b = &anthropicBlock{}
		st.blocks[idx] = b
	}
	return b
}

func (st *anthropicStream) textChannel(idx int) *anthropicChannel {
	b := st.block(idx)
	if b.text == nil {
		b.text = &anthropicChannel{rh: st.sess.NewStreamRehydrator(false)}
	}
	return b.text
}

func (st *anthropicStream) jsonChannel(idx int) *anthropicChannel {
	b := st.block(idx)
	if b.json == nil {
		b.json = &anthropicChannel{rh: st.sess.NewStreamRehydrator(true)}
	}
	return b.json
}

func (st *anthropicStream) emitDelta(idx int, deltaType, field, value string) {
	ev := map[string]any{"type": "content_block_delta", "index": idx, "delta": map[string]any{"type": deltaType, field: value}}
	io.WriteString(st.w, "event: content_block_delta\ndata: "+string(encode(ev))+"\n\n")
}

func (st *anthropicStream) closeBlock(idx int) {
	b, ok := st.blocks[idx]
	if !ok {
		return
	}
	delete(st.blocks, idx)
	if c := b.text; c != nil {
		if rest := c.rh.Flush(); rest != "" {
			c.record("", rest)
			st.emitDelta(idx, "text_delta", "text", rest)
		}
		if !c.over && c.raw.Len() > 0 {
			st.sess.Remember(c.out.String(), c.raw.String())
		}
		if st.visible.Len() < streamTextCap {
			st.visible.WriteString(c.out.String())
			st.visible.WriteByte('\n')
		}
	}
	if c := b.json; c != nil {
		if rest := c.rh.Flush(); rest != "" {
			c.record("", rest)
			st.emitDelta(idx, "input_json_delta", "partial_json", rest)
		}
		if !c.over && c.raw.Len() > 0 {
			rememberJSON(st.sess, c.raw.String(), c.out.String())
		}
	}
}

func (st *anthropicStream) closeAll() {
	idxs := make([]int, 0, len(st.blocks))
	for i := range st.blocks {
		idxs = append(idxs, i)
	}
	sort.Ints(idxs)
	for _, i := range idxs {
		st.closeBlock(i)
	}
}

func (st *anthropicStream) finish() {
	if st.stopped {
		return
	}
	st.stopped = true
	st.closeAll()
	st.names = st.sess.UnknownNames(st.visible.String())
	if len(st.names) > 0 {
		io.WriteString(st.w, ": outworld unknown_names="+strings.Join(st.names, ",")+"\n\n")
	}
}

func sseData(line string) (string, bool) {
	trimmed := strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(trimmed, "data:") {
		return "", false
	}
	return strings.TrimPrefix(trimmed[len("data:"):], " "), true
}

func (st *anthropicStream) event(lines []string) {
	var data []string
	for _, l := range lines {
		if d, ok := sseData(l); ok {
			data = append(data, d)
		}
	}
	var obj map[string]any
	if len(data) > 0 {
		obj, _ = decodeObject([]byte(strings.Join(data, "\n")))
	}
	modified := false
	if obj != nil {
		typ, _ := obj["type"].(string)
		idx := toInt(obj["index"])
		switch typ {
		case "content_block_start":
			cb, _ := obj["content_block"].(map[string]any)
			if t, ok := cb["text"].(string); ok && t != "" {
				cb["text"] = st.textChannel(idx).push(t)
				modified = true
			}
			switch in := cb["input"].(type) {
			case map[string]any:
				if len(in) > 0 {
					cb["input"] = rehydrateRemember(st.sess, in, &modified)
				}
			case []any:
				if len(in) > 0 {
					cb["input"] = rehydrateRemember(st.sess, in, &modified)
				}
			}
		case "content_block_delta":
			delta, _ := obj["delta"].(map[string]any)
			switch dt, _ := delta["type"].(string); dt {
			case "text_delta":
				if t, ok := delta["text"].(string); ok {
					delta["text"] = st.textChannel(idx).push(t)
					modified = true
				}
			case "input_json_delta":
				if t, ok := delta["partial_json"].(string); ok {
					delta["partial_json"] = st.jsonChannel(idx).push(t)
					modified = true
				}
			}
		case "content_block_stop":
			st.closeBlock(idx)
		case "message_stop":
			st.finish()
		}
	}
	if !modified {
		for _, l := range lines {
			io.WriteString(st.w, l)
		}
		return
	}
	written := false
	for _, l := range lines {
		if _, ok := sseData(l); !ok {
			io.WriteString(st.w, l)
			continue
		}
		if written {
			continue
		}
		written = true
		io.WriteString(st.w, "data: "+string(encode(obj))+l[len(strings.TrimRight(l, "\r\n")):])
	}
}

func (s *Server) streamAnthropic(w http.ResponseWriter, resp *http.Response, sess *outworld.Session) []string {
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
	st := &anthropicStream{sess: sess, w: w, blocks: map[int]*anthropicBlock{}}
	stop, drained := s.closeOnDrain(resp.Body)
	defer stop()
	br := bufio.NewReaderSize(resp.Body, 64*1024)
	var ev []string
	for {
		line, err := br.ReadString('\n')
		if line != "" {
			ev = append(ev, line)
			if strings.HasSuffix(line, "\n") && strings.TrimRight(line, "\r\n") == "" {
				st.event(ev)
				ev = nil
				flush()
			}
		}
		if err != nil {
			break
		}
	}
	if drained() && !st.stopped {
		st.stopped = true
		writeShutdownEvent(w, config.APIAnthropic)
		flush()
		return nil
	}
	if len(ev) > 0 {
		st.event(ev)
		if !strings.HasSuffix(ev[len(ev)-1], "\n") {
			io.WriteString(w, "\n")
		}
		io.WriteString(w, "\n")
	}
	st.finish()
	flush()
	if len(st.names) > 0 {
		w.Header().Set(HeaderUnknownNames, strings.Join(st.names, ", "))
	}
	return st.names
}
