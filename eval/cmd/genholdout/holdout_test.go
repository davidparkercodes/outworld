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

func TestCommittedHoldoutMatchesGenerator(t *testing.T) {
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
		t.Fatalf("corpus/%s is stale: run `go run ./cmd/genholdout` from eval/", corpusFile)
	}
	gotDict, err := os.ReadFile(filepath.Join(corpusDir, peopleFile))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotDict, corpus.MarshalDictionary(dict)) {
		t.Fatalf("corpus/%s is stale: run `go run ./cmd/genholdout` from eval/", peopleFile)
	}
}

func TestHoldoutDeterministic(t *testing.T) {
	a, da, _ := mustBuild(t)
	b, db, _ := mustBuild(t)
	ja, _ := a.Marshal()
	jb, _ := b.Marshal()
	if !bytes.Equal(ja, jb) || !bytes.Equal(corpus.MarshalDictionary(da), corpus.MarshalDictionary(db)) {
		t.Fatal("holdout generator is not deterministic")
	}
}

func TestCommittedHoldoutSpans(t *testing.T) {
	c, err := corpus.Load(filepath.Join(corpusDir, corpusFile))
	if err != nil {
		t.Fatal(err)
	}
	if n := len(c.Items); n < 70 || n > 90 {
		t.Fatalf("expected 70 to 90 items, got %d", n)
	}
	pol := score.DefaultPolicy()
	types := map[string]int{}
	for _, item := range c.Items {
		sensitive := 0
		perMsg := map[int][]corpus.Span{}
		for _, s := range item.Spans {
			if got := item.Messages[s.Msg].Content[s.Start:s.End]; got != s.Value {
				t.Errorf("%s: slice %q != value %q", item.ID, got, s.Value)
			}
			if s.Canonical == "" {
				t.Errorf("%s: span %q has no canonical", item.ID, s.Value)
			}
			if s.Type == corpus.TypePerson && (s.Variant == "" || s.Person == "") {
				t.Errorf("%s: person span %q needs a variant and a person id", item.ID, s.Value)
			}
			if s.Type == corpus.TypeFinancial && strings.HasPrefix(s.Variant, "card-") && !corpus.Luhn(s.Value) {
				t.Errorf("%s: card %q fails Luhn", item.ID, s.Value)
			}
			if pol.Sensitive(s.Type) {
				sensitive++
			}
			types[s.Type]++
			perMsg[s.Msg] = append(perMsg[s.Msg], s)
		}
		for mi, spans := range perMsg {
			sort.Slice(spans, func(i, j int) bool { return spans[i].Start < spans[j].Start })
			for i := 1; i < len(spans); i++ {
				if spans[i].Start < spans[i-1].End {
					t.Errorf("%s msg %d: spans %q and %q overlap", item.ID, mi, spans[i-1].Value, spans[i].Value)
				}
			}
		}
		negative := hasTag(item, tagNegative)
		if sensitive == 0 && !negative {
			t.Errorf("%s: no sensitive span and not tagged %s", item.ID, tagNegative)
		}
		if sensitive > 0 && negative {
			t.Errorf("%s: tagged %s but has %d sensitive spans", item.ID, tagNegative, sensitive)
		}
	}
	for _, typ := range append(append([]string{}, corpus.EntityTypes...), corpus.TypeQuasi) {
		if types[typ] == 0 {
			t.Errorf("no spans of type %s", typ)
		}
	}
}

func TestHoldoutPeople(t *testing.T) {
	c, dict, reg := mustBuild(t)
	used := 0
	for _, p := range reg.order {
		if p.used {
			used++
		} else {
			t.Errorf("person %s is never referenced", p.full)
		}
	}
	if len(dict) < used/3 || len(dict) > 2*used/3 {
		t.Errorf("dictionary should hold roughly half of the people: %d of %d", len(dict), used)
	}
	main, err := corpus.LoadDictionary(filepath.Join(corpusDir, "people.csv"))
	if err != nil {
		t.Fatal(err)
	}
	taken := map[string]bool{}
	for _, e := range main {
		taken[strings.ToLower(e.Value)] = true
	}
	inDict, outDict := 0, 0
	dictNames := map[string]bool{}
	for _, e := range dict {
		dictNames[e.Value] = true
	}
	seen := map[string]bool{}
	for _, item := range c.Items {
		for _, s := range item.Spans {
			if s.Type != corpus.TypePerson || seen[s.Canonical] {
				continue
			}
			seen[s.Canonical] = true
			if taken[strings.ToLower(s.Canonical)] {
				t.Errorf("%s is already in the main corpus dictionary", s.Canonical)
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

func TestHoldoutHygiene(t *testing.T) {
	c, _, _ := mustBuild(t)
	allowed := []string{".example", "@example.com", "@example.org", "@example.net"}
	for _, item := range c.Items {
		for _, m := range item.Messages {
			if strings.ContainsRune(m.Content, rune(0x2014)) {
				t.Errorf("%s: contains an em dash", item.ID)
			}
		}
		for _, s := range item.Spans {
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
				t.Errorf("%s: email %q is not on a reserved example domain", item.ID, s.Value)
			}
		}
	}
}
