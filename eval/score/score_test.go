package score

import (
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/eval/corpus"
	"github.com/davidparkercodes/outworld/eval/upstream"
)

func TestCountWord(t *testing.T) {
	cases := []struct {
		h, n string
		want int
	}{
		{"will you ask will.ashgrove@x.example", "will", 2},
		{"willow will", "will", 1},
		{"marisol's desk", "marisol", 1},
		{"a **marisol** quenby", "**marisol** quenby", 1},
		{"(312) 555-0147", "(312) 555-0147", 1},
	}
	for _, c := range cases {
		if got := CountWord(c.h, c.n); got != c.want {
			t.Errorf("CountWord(%q,%q)=%d want %d", c.h, c.n, got, c.want)
		}
	}
}

func TestDigitRuns(t *testing.T) {
	runs := DigitRuns("card 4111-1111-1111-1111, phone (312) 555.0147")
	if CountDigits(runs, "4111111111111111") != 1 || CountDigits(runs, "3125550147") != 1 {
		t.Fatalf("runs = %v", runs)
	}
}

func item() *corpus.Item {
	content := "Will you ask Will Ashgrove? Will's card is 4242 4242 4242 4242 at Copperline Logistics."
	idx := func(s string, nth int) int {
		off := 0
		for range nth {
			off += strings.Index(content[off:], s) + 1
		}
		return off + strings.Index(content[off:], s)
	}
	sp := func(typ, v string, nth int) corpus.Span {
		i := idx(v, nth)
		return corpus.Span{Msg: 1, Start: i, End: i + len(v), Type: typ, Value: v, Canonical: v}
	}
	return &corpus.Item{
		ID:       "t",
		Messages: []corpus.Message{{Role: "system", Content: "task"}, {Role: "user", Content: content}},
		Spans: []corpus.Span{
			sp("person", "Will Ashgrove", 0),
			sp("person", "Will", 2),
			sp("financial", "4242 4242 4242 4242", 0),
			sp("org", "Copperline Logistics", 0),
		},
	}
}

func TestBackgroundSubtraction(t *testing.T) {
	it := item()
	if err := (&corpus.Corpus{Items: []corpus.Item{*it}}).Validate(); err != nil {
		t.Fatal(err)
	}
	masked := "Will you ask PERSON_1? PERSON_2's card is [REDACTED] at Copperline Logistics."
	final := "TIMELINE\n\n" + upstream.EchoMarker + "\nWill you ask Will Ashgrove? Will's card is [REDACTED] at Copperline Logistics."
	r := ScoreItem(Observation{Item: it, Upstream: masked, Final: final, CheckRestore: true}, DefaultPolicy(), nil)
	for _, e := range r.Entities {
		if e.Leaked {
			t.Errorf("%s %q reported leaked", e.Type, e.Value)
		}
		if e.Restorable && !e.Restored {
			t.Errorf("%q not restored", e.Value)
		}
		if e.Altered {
			t.Errorf("%q reported altered", e.Value)
		}
	}
	if len(r.Residue) != 0 {
		t.Errorf("bracketed strip placeholders are not residue, got %v", r.Residue)
	}
	r = ScoreItem(Observation{Item: it, Upstream: masked + " Ezra", Final: final + " Ezra", CheckRestore: true}, DefaultPolicy(), nil)
	if strings.Join(r.Residue, ",") != "ezra" {
		t.Errorf("unrestored surrogate should be residue, got %v", r.Residue)
	}

	leaky := "Will you ask PERSON_1? Will's card is 4242-4242-4242-4242 at Acme."
	r = ScoreItem(Observation{Item: it, Upstream: leaky, Final: "", CheckRestore: true}, DefaultPolicy(), nil)
	got := map[string]EntityResult{}
	for _, e := range r.Entities {
		got[e.Value] = e
	}
	if !got["Will"].Leaked || got["Will Ashgrove"].Leaked {
		t.Errorf("first-name leak detection wrong: %+v", got)
	}
	if !got["4242 4242 4242 4242"].Leaked || got["4242 4242 4242 4242"].LeakVia != "digits" {
		t.Errorf("card leak via digits not detected: %+v", got["4242 4242 4242 4242"])
	}
	if !got["Copperline Logistics"].Altered {
		t.Error("org replacement should count as over-masking")
	}
	if got["Will Ashgrove"].Restored {
		t.Error("empty final answer cannot restore anything")
	}
}

func TestAggregateAndPercentile(t *testing.T) {
	rs := []ItemResult{
		{ID: "a", Mode: "nonstream", LatencyMS: 10, DirectMS: 4, Entities: []EntityResult{{Type: "person", Action: ActionMask, Leaked: true, LeakVia: "verbatim", Variant: "first"}, {Type: "email", Action: ActionMask, Restorable: true, Restored: true}}},
		{ID: "b", Mode: "nonstream", Err: "boom"},
	}
	s := Aggregate(rs)
	if s.Leak.N != 2 || s.Leak.Hits != 1 || s.Errors != 1 || s.Restore.Rate != 1 {
		t.Fatalf("bad summary %+v", s)
	}
	if Percentile([]float64{1, 2, 3, 4, 100}, 95) != 100 || Percentile([]float64{1, 2, 3, 4}, 50) != 2 {
		t.Fatal("percentile")
	}
}

func TestFacts(t *testing.T) {
	content := "Here you go:\n```json\n[{\"date\":\"2026-03-02\",\"person\":\"Rose Thornbury\"},{\"date\":\"2026-01-01\",\"person\":\"Nobody\"}]\n```"
	pred := ParseFacts(content)
	if len(pred) != 2 {
		t.Fatalf("parsed %v", pred)
	}
	m := MatchFacts(pred, []corpus.Fact{{Date: "2026-03-02", Person: "Rose Thornbury"}, {Date: "2026-05-05", Person: "Will Ashgrove"}})
	if m.TP != 1 || m.FP != 1 || m.FN != 1 {
		t.Fatalf("match %+v", m)
	}
}
