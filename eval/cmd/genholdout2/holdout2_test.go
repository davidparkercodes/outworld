package main

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/eval/corpus"
	"github.com/davidparkercodes/outworld/eval/score"
)

var corpusDir = filepath.Join("..", "..", "corpus")

func mustBuild(t *testing.T) (*corpus.Corpus, []corpus.DictEntry, *registry) {
	t.Helper()
	c, dict, reg, err := build()
	if err != nil {
		t.Fatal(err)
	}
	return c, dict, reg
}

func TestCommittedHoldout2MatchesGenerator(t *testing.T) {
	c, dict, _ := mustBuild(t)
	want, err := c.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(corpusDir, corpusFile))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("corpus/%s is stale: run `go run ./cmd/genholdout2` from eval/", corpusFile)
	}
	gotDict, err := os.ReadFile(filepath.Join(corpusDir, peopleFile))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotDict, corpus.MarshalDictionary(dict)) {
		t.Fatalf("corpus/%s is stale: run `go run ./cmd/genholdout2` from eval/", peopleFile)
	}
}

func TestHoldout2Deterministic(t *testing.T) {
	a, da, _ := mustBuild(t)
	b, db, _ := mustBuild(t)
	ja, _ := a.Marshal()
	jb, _ := b.Marshal()
	if !bytes.Equal(ja, jb) || !bytes.Equal(corpus.MarshalDictionary(da), corpus.MarshalDictionary(db)) {
		t.Fatal("holdout2 generator is not deterministic")
	}
}

func TestCommittedHoldout2Spans(t *testing.T) {
	c, err := corpus.Load(filepath.Join(corpusDir, corpusFile))
	if err != nil {
		t.Fatal(err)
	}
	if n := len(c.Items); n < 110 || n > 130 {
		t.Fatalf("expected about 120 items, got %d", n)
	}
	pol := score.DefaultPolicy()
	types := map[string]int{}
	total := 0
	for _, it := range c.Items {
		sensitive := 0
		perMsg := map[int][]corpus.Span{}
		for _, s := range it.Spans {
			if got := it.Messages[s.Msg].Content[s.Start:s.End]; got != s.Value {
				t.Errorf("%s: slice %q != value %q", it.ID, got, s.Value)
			}
			if s.Canonical == "" {
				t.Errorf("%s: span %q has no canonical", it.ID, s.Value)
			}
			if s.Type == corpus.TypePerson && (s.Variant == "" || s.Person == "") {
				t.Errorf("%s: person span %q needs a variant and a person id", it.ID, s.Value)
			}
			if s.Type == corpus.TypeFinancial && strings.HasPrefix(s.Variant, "card-") && !corpus.Luhn(s.Value) {
				t.Errorf("%s: card %q fails Luhn", it.ID, s.Value)
			}
			if pol.Sensitive(s.Type) {
				sensitive++
			}
			types[s.Type]++
			total++
			perMsg[s.Msg] = append(perMsg[s.Msg], s)
		}
		for mi, spans := range perMsg {
			sort.Slice(spans, func(i, j int) bool { return spans[i].Start < spans[j].Start })
			for i := 1; i < len(spans); i++ {
				if spans[i].Start < spans[i-1].End {
					t.Errorf("%s msg %d: spans %q and %q overlap", it.ID, mi, spans[i-1].Value, spans[i].Value)
				}
			}
		}
		negative := hasTag(it, tagNegative)
		if sensitive == 0 && !negative {
			t.Errorf("%s: no sensitive span and not tagged %s", it.ID, tagNegative)
		}
		if sensitive > 0 && negative {
			t.Errorf("%s: tagged %s but has %d sensitive spans", it.ID, tagNegative, sensitive)
		}
	}
	if total < 1100 {
		t.Errorf("expected at least 1100 spans, got %d", total)
	}
	for _, typ := range append(append([]string{}, corpus.EntityTypes...), corpus.TypeQuasi) {
		if types[typ] == 0 {
			t.Errorf("no spans of type %s", typ)
		}
	}
}

func TestHoldout2Mix(t *testing.T) {
	c, _, _ := mustBuild(t)
	cats := map[string]int{}
	negatives, longs, lookalikes := 0, 0, 0
	for _, it := range c.Items {
		cats[it.Tags[0]]++
		if hasTag(it, tagNegative) {
			negatives++
		}
		if hasTag(it, tagLong) {
			longs++
			if wordCount(it) <= longWords {
				t.Errorf("%s: tagged %s with only %d words", it.ID, tagLong, wordCount(it))
			}
		}
		for _, s := range it.Spans {
			if strings.HasPrefix(s.Variant, lookalikePrefix) {
				lookalikes++
				if s.Type != corpus.TypeOrg {
					t.Errorf("%s: lookalike %q must use a kept type", it.ID, s.Value)
				}
			}
		}
	}
	n := float64(len(c.Items))
	for cat, share := range map[string]float64{catCode: 0.35, catBusiness: 0.25, catPersonal: 0.20, catLong: 0.20} {
		if got := float64(cats[cat]) / n; got < share-0.05 || got > share+0.05 {
			t.Errorf("category %s is %.0f%% of items, want about %.0f%%", cat, got*100, share*100)
		}
	}
	if negatives < 15 {
		t.Errorf("expected at least 15 items without a sensitive value, got %d", negatives)
	}
	if longs < 4 {
		t.Errorf("expected several items over %d words, got %d", longWords, longs)
	}
	if lookalikes == 0 {
		t.Error("no lookalike hard negatives")
	}
}

func TestHoldout2People(t *testing.T) {
	c, dict, reg := mustBuild(t)
	used := 0
	for _, p := range reg.order {
		if p.used {
			used++
		} else {
			t.Errorf("person %s is never referenced", p.full)
		}
	}
	if len(dict) < 2*used/5 || len(dict) > 3*used/5 {
		t.Errorf("dictionary should hold roughly half of the people: %d of %d", len(dict), used)
	}
	taken := map[string]bool{}
	for _, earlier := range []string{"people.csv", "holdout-people.csv"} {
		entries, err := corpus.LoadDictionary(filepath.Join(corpusDir, earlier))
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			taken[strings.ToLower(e.Value)] = true
		}
	}
	dictNames := map[string]bool{}
	for _, e := range dict {
		dictNames[e.Value] = true
	}
	inDict, outDict := 0, 0
	seen := map[string]bool{}
	for _, it := range c.Items {
		for _, s := range it.Spans {
			if s.Type != corpus.TypePerson || seen[s.Canonical] {
				continue
			}
			seen[s.Canonical] = true
			if taken[strings.ToLower(s.Canonical)] {
				t.Errorf("a person in this corpus is already in an earlier dictionary (person id %s)", s.Person)
			}
			if dictNames[s.Canonical] {
				inDict++
			} else {
				outDict++
			}
		}
	}
	if inDict == 0 || outDict == 0 {
		t.Errorf("need people on both sides of the dictionary: %d in, %d out", inDict, outDict)
	}
}

func TestHoldout2Hygiene(t *testing.T) {
	c, _, _ := mustBuild(t)
	allowed := []string{".example", "@example.com", "@example.org", "@example.net", "%40example.com"}
	for _, it := range c.Items {
		for _, m := range it.Messages {
			if strings.ContainsRune(m.Content, rune(0x2014)) {
				t.Errorf("%s: contains an em dash", it.ID)
			}
			if strings.Contains(m.Content, "{{p|") || strings.Contains(m.Content, "{{d|") {
				t.Errorf("%s: unparsed markup left in the text", it.ID)
			}
		}
		for _, s := range it.Spans {
			if s.Type != corpus.TypeEmail {
				continue
			}
			ok := false
			for _, suffix := range allowed {
				if strings.HasSuffix(strings.ToLower(s.Value), suffix) {
					ok = true
				}
			}
			if !ok {
				t.Errorf("%s: email %q is not on a reserved example domain", it.ID, s.Value)
			}
		}
	}
}
