package runner

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/davidparkercodes/outworld/eval/corpus"
	"github.com/davidparkercodes/outworld/eval/score"
	"github.com/davidparkercodes/outworld/eval/upstream"
)

const (
	ModeNonStream = "nonstream"
	ModeStream    = "stream"
)

type Target struct {
	GatewayV1    string
	GatewayKey   string
	DirectV1     string
	DirectKey    string
	Recorder     *upstream.Recorder
	Model        string
	CheckRestore bool
	Timeout      time.Duration
}

func Run(ctx context.Context, items []corpus.Item, modes []string, t Target, progress func(done, total int)) []score.Observation {
	client := &http.Client{Timeout: t.Timeout}
	total := len(items) * len(modes)
	var out []score.Observation
	for i := range items {
		it := &items[i]
		for _, mode := range modes {
			if ctx.Err() != nil {
				return out
			}
			stream := mode == ModeStream
			mark := t.Recorder.Mark()
			res := Chat(ctx, client, t.GatewayV1, t.GatewayKey, t.Model, it.Messages, stream)
			recs := t.Recorder.Since(mark)
			var up strings.Builder
			for _, r := range recs {
				up.WriteString(upstream.RequestText(r.Body))
				up.WriteString("\n")
			}
			o := score.Observation{
				Item: it, Mode: mode, Upstream: up.String(), UpstreamCalls: len(recs),
				Final: res.Content, Latency: res.Latency, Unknown: res.Unknown, CheckRestore: t.CheckRestore,
			}
			if res.Err != nil {
				o.Err = res.Err.Error()
			}
			if t.DirectV1 != "" {
				d := Chat(ctx, client, t.DirectV1, t.DirectKey, t.Model, it.Messages, stream)
				o.DirectLatency = d.Latency
			}
			out = append(out, o)
			if progress != nil {
				progress(len(out), total)
			}
		}
	}
	return out
}

type QualityPair struct {
	Item       string    `json:"item"`
	Direct     score.PRF `json:"direct"`
	Gateway    score.PRF `json:"gateway"`
	DirectErr  string    `json:"direct_error,omitempty"`
	GatewayErr string    `json:"gateway_error,omitempty"`
}

func Quality(ctx context.Context, items []corpus.Item, t Target) []QualityPair {
	client := &http.Client{Timeout: t.Timeout}
	var out []QualityPair
	for i := range items {
		it := &items[i]
		if len(it.Facts) == 0 || ctx.Err() != nil {
			continue
		}
		q := QualityPair{Item: it.ID}
		d := Chat(ctx, client, t.DirectV1, t.DirectKey, t.Model, it.Messages, false)
		if d.Err != nil {
			q.DirectErr = d.Err.Error()
		} else {
			q.Direct = score.MatchFacts(score.ParseFacts(d.Content), it.Facts)
		}
		g := Chat(ctx, client, t.GatewayV1, t.GatewayKey, t.Model, it.Messages, false)
		if g.Err != nil {
			q.GatewayErr = g.Err.Error()
		} else {
			q.Gateway = score.MatchFacts(score.ParseFacts(g.Content), it.Facts)
		}
		out = append(out, q)
	}
	return out
}
