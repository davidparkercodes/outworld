package outworld

import (
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

const (
	ReasonOverlap = "overlaps_replacement"
	ReasonRounds  = "rounds_exhausted"
	ReasonField   = "unmaskable_field"
	ReasonError   = "detector_error"
)

const previewContext = 32

type Replacement struct {
	Type        entity.Type   `json:"type"`
	Source      string        `json:"source"`
	Action      entity.Action `json:"action"`
	Replacement string        `json:"replacement"`
	Message     int           `json:"message_index"`
	Field       string        `json:"field,omitempty"`
	Start       int           `json:"start"`
	End         int           `json:"end"`
}

func (s *Session) SetLocation(message int, field string) {
	s.mu.Lock()
	s.locMsg, s.locField = message, field
	s.mu.Unlock()
}

func (s *Session) Location() (int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.locMsg, s.locField
}

func (s *Session) SetTrace(on bool) {
	s.mu.Lock()
	s.trace = on
	s.mu.Unlock()
}

func (s *Session) Replacements() []Replacement {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Replacement(nil), s.replaced...)
}

func (s *Session) MemoHits() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.memoHits
}

func (s *Session) noteMemoHit() {
	s.mu.Lock()
	if !s.priming {
		s.memoHits++
	}
	s.mu.Unlock()
}

func (s *Session) noteReplacements(text string, ins []insertion) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.trace || s.priming {
		return
	}
	for _, in := range ins {
		s.replaced = append(s.replaced, Replacement{
			Type:        in.typ,
			Source:      in.src,
			Action:      s.e.policy.Action(in.typ),
			Replacement: text[in.start:in.end],
			Message:     s.locMsg,
			Field:       s.locField,
			Start:       utf8.RuneCountInString(text[:in.start]),
			End:         utf8.RuneCountInString(text[:in.end]),
		})
	}
}

func (s *Session) record(found []finding, text, reason string) {
	if len(found) == 0 {
		return
	}
	spans := make([][2]int, 0, len(found))
	for _, f := range found {
		if f.span.Start >= 0 && f.span.End <= len(text) && f.span.Start < f.span.End {
			spans = append(spans, [2]int{f.span.Start, f.span.End})
		}
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i][0] < spans[j][0] })
	runes := runeOffsets(text, spans)
	maxLen := 0
	for _, sp := range spans {
		maxLen = max(maxLen, sp[1]-sp[0])
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := map[Violation]bool{}
	for _, f := range found {
		v := Violation{Type: f.span.Type, Source: f.source, Message: s.locMsg, Field: s.locField, Start: -1, End: -1, Reason: reason}
		if f.span.Start < 0 || f.span.End > len(text) || f.span.Start >= f.span.End {
			v.Reason = ReasonError
		} else {
			v.Start, v.End = runes[f.span.Start], runes[f.span.End]
			v.Preview = redactedPreview(text, f.span.Start, f.span.End, nearSpans(spans, f.span.Start, f.span.End, maxLen))
		}
		v.Hint = hintFor(v)
		if seen[v] {
			continue
		}
		seen[v] = true
		s.violations = append(s.violations, v)
	}
}

func runeOffsets(text string, spans [][2]int) map[int]int {
	pos := make([]int, 0, 2*len(spans))
	for _, sp := range spans {
		pos = append(pos, sp[0], sp[1])
	}
	sort.Ints(pos)
	out := make(map[int]int, len(pos))
	last, n := 0, 0
	for _, p := range pos {
		if _, ok := out[p]; ok {
			continue
		}
		n += utf8.RuneCountInString(text[last:p])
		last = p
		out[p] = n
	}
	return out
}

func nearSpans(spans [][2]int, start, end, maxLen int) [][2]int {
	lo := sort.Search(len(spans), func(i int) bool { return spans[i][0] >= start-previewContext-maxLen })
	hi := sort.Search(len(spans), func(i int) bool { return spans[i][0] >= end+previewContext })
	return spans[lo:hi]
}

const redactedMark = "[***]"

func redactedPreview(text string, start, end int, all [][2]int) string {
	ws := start - previewContext
	if ws < 0 {
		ws = 0
	}
	for ws > 0 && !utf8.RuneStart(text[ws]) {
		ws--
	}
	we := end + previewContext
	if we > len(text) {
		we = len(text)
	}
	for we < len(text) && !utf8.RuneStart(text[we]) {
		we++
	}
	type mark struct {
		start, end int
		text       string
	}
	marks := []mark{{start, end, redactedMark}}
	for _, sp := range all {
		clash := false
		for i := range marks {
			if sp[0] < marks[i].end && marks[i].start < sp[1] {
				marks[i].start, marks[i].end = min(marks[i].start, sp[0]), max(marks[i].end, sp[1])
				clash = true
				break
			}
		}
		if !clash && sp[0] < we && sp[1] > ws {
			marks = append(marks, mark{sp[0], sp[1], redactedMark})
		}
	}
	sort.Slice(marks, func(i, j int) bool { return marks[i].start < marks[j].start })
	var b strings.Builder
	if ws > 0 {
		b.WriteString("...")
	}
	pos := ws
	for _, m := range marks {
		if m.end <= pos && pos > ws {
			continue
		}
		if m.start > pos {
			b.WriteString(text[pos:m.start])
		}
		b.WriteString(m.text)
		pos = max(pos, m.end)
	}
	if pos < we {
		b.WriteString(text[pos:we])
	}
	if we < len(text) && pos < len(text) {
		b.WriteString("...")
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func hintFor(v Violation) string {
	if v.Reason == ReasonError {
		return "A detector failed during the leak check. Check that the NER sidecar (ner.url) is reachable, or set ner.optional: true."
	}
	if v.Reason == ReasonField {
		return "This value sits in a field the gateway cannot rewrite (a JSON object key or a message name). Move it into message text or a JSON value, or add it to policy.allow if it is not sensitive."
	}
	if v.Reason == ReasonRounds {
		return "New findings kept appearing after several masking rounds. Send less text per message, or add the names involved to the dictionary (policy.dictionary) so they are masked in the first pass."
	}
	src := v.Source
	switch {
	case src == "vault":
		return "A value the vault already knows appears next to or inside another replacement and could not be rewritten safely. Add the full form to the dictionary (policy.dictionary) so it is masked as one unit, or add it to policy.allow if it is not sensitive."
	case src == "ner":
		return "The NER model flagged this after masking. If it is a person, add the name to the dictionary (policy.dictionary); if it is not sensitive, add it to policy.allow or set policy.entities." + string(v.Type) + ": keep."
	case v.Type == entity.Person:
		return "Add this person to the dictionary (policy.dictionary) so every form of the name is masked, or turn on the NER sidecar (ner.url). If it is not a person, add the text to policy.allow."
	case v.Type == entity.Secret || v.Type == entity.GovID || v.Type == entity.Financial:
		return "A " + string(v.Type) + " pattern overlaps another replacement. Remove the value from the request, or if it is not sensitive add it to policy.allow or set policy.entities." + string(v.Type) + ": keep."
	}
	return "This " + string(v.Type) + " value overlaps another replacement. If it is not sensitive, add it to policy.allow or set policy.entities." + string(v.Type) + ": keep."
}
