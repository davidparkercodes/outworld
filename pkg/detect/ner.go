package detect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/pkg/dateshift"
	"github.com/davidparkercodes/outworld/pkg/entity"
)

type DateDetector struct {
	Order dateshift.Order
}

func (d DateDetector) Name() string { return "date" }

func (d DateDetector) Detect(_ context.Context, text string) ([]entity.Span, error) {
	var out []entity.Span
	for _, m := range dateshift.Find(text, d.Order) {
		out = append(out, entity.Span{Start: m.Start, End: m.End, Type: entity.Date, Label: "date", Text: m.Text, Source: "date", Priority: 50, Score: 1})
	}
	return out, nil
}

type NERDetector struct {
	URL       string
	Labels    map[string]entity.Type
	Threshold float64
	Client    *http.Client
}

func DefaultNERLabels() map[string]entity.Type {
	return map[string]entity.Type{
		"person":       entity.Person,
		"organization": entity.Org,
		"location":     entity.Location,
		"address":      entity.Address,
	}
}

func NewNERDetector(url string, timeout time.Duration) *NERDetector {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &NERDetector{URL: url, Labels: DefaultNERLabels(), Threshold: 0.5, Client: &http.Client{Timeout: timeout}}
}

func (d *NERDetector) Name() string { return "ner" }

type nerRequest struct {
	Text      string   `json:"text"`
	Labels    []string `json:"labels"`
	Threshold float64  `json:"threshold"`
}

type nerResponse struct {
	Entities []struct {
		Start int     `json:"start"`
		End   int     `json:"end"`
		Label string  `json:"label"`
		Score float64 `json:"score"`
	} `json:"entities"`
}

func (d *NERDetector) Detect(ctx context.Context, text string) ([]entity.Span, error) {
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	labels := make([]string, 0, len(d.Labels))
	for l := range d.Labels {
		labels = append(labels, l)
	}
	body, _ := json.Marshal(nerRequest{Text: TitleCaseShouting(text), Labels: labels, Threshold: d.Threshold})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(d.URL, "/")+"/detect", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := d.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ner sidecar: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("ner sidecar: status %d", resp.StatusCode)
	}
	var nr nerResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&nr); err != nil {
		return nil, fmt.Errorf("ner sidecar: decode: %w", err)
	}
	offsets := runeToByte(text)
	var out []entity.Span
	for _, e := range nr.Entities {
		t, ok := d.Labels[strings.ToLower(e.Label)]
		if !ok || e.Score < d.Threshold || e.Start < 0 || e.End <= e.Start || e.End >= len(offsets) {
			continue
		}
		s, en := offsets[e.Start], offsets[e.End]
		for en > s && strings.ContainsRune(" \t\n.,;:", rune(text[en-1])) {
			en--
		}
		if en <= s {
			continue
		}
		out = append(out, entity.Span{Start: s, End: en, Type: t, Label: string(t), Text: text[s:en], Source: "ner", Priority: 30, Score: e.Score})
	}
	return out, nil
}

func TitleCaseShouting(s string) string {
	orig := []rune(s)
	rs := []rune(s)
	changed := false
	for i := 0; i < len(rs); {
		if !unicode.IsLetter(rs[i]) {
			i++
			continue
		}
		j, upper := i, true
		for j < len(rs) && unicode.IsLetter(rs[j]) {
			if unicode.IsLower(rs[j]) {
				upper = false
			}
			j++
		}
		if upper && j-i >= 2 && shoutingRun(orig, i, j) {
			for k := i + 1; k < j; k++ {
				if l := unicode.ToLower(rs[k]); utf8.RuneLen(l) == utf8.RuneLen(rs[k]) {
					rs[k] = l
					changed = true
				}
			}
		}
		i = j
	}
	if !changed {
		return s
	}
	return string(rs)
}

func shoutingRun(rs []rune, i, j int) bool {
	neighbor := func(k, step int) bool {
		for k >= 0 && k < len(rs) && (rs[k] == ' ' || rs[k] == '-' || rs[k] == '\'' || rs[k] == ',' || rs[k] == ':') {
			k += step
		}
		if k < 0 || k >= len(rs) || !unicode.IsLetter(rs[k]) {
			return false
		}
		a, b := k, k
		for a > 0 && unicode.IsLetter(rs[a-1]) {
			a--
		}
		for b < len(rs)-1 && unicode.IsLetter(rs[b+1]) {
			b++
		}
		if b-a+1 < 2 {
			return false
		}
		for x := a; x <= b; x++ {
			if unicode.IsLower(rs[x]) {
				return false
			}
		}
		return true
	}
	return neighbor(i-1, -1) || neighbor(j, 1)
}

func runeToByte(s string) []int {
	out := make([]int, 0, utf8.RuneCountInString(s)+1)
	for i := range s {
		out = append(out, i)
	}
	return append(out, len(s))
}
