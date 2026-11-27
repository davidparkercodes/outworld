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

var (
	corpusDir        = filepath.Join("..", "..", "corpus")
	earlierPeopleDir = filepath.Join("..", "..", "corpus")
	earlierPeople    = []string{"people.csv", "holdout-people.csv", "holdout2-people.csv"}
)

func mustBuild(t *testing.T) *built {
	t.Helper()
	b, err := build()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCommittedHoldout3MatchesGenerator(t *testing.T) {
	b := mustBuild(t)
	want, err := b.corpus.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(corpusDir, corpusFile))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("corpus/%s is stale: run `go run ./cmd/genholdout3` from eval/", corpusFile)
	}
	gotDict, err := os.ReadFile(filepath.Join(corpusDir, peopleFile))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotDict, corpus.MarshalDictionary(b.dict)) {
		t.Fatalf("corpus/%s is stale: run `go run ./cmd/genholdout3` from eval/", peopleFile)
	}
}

func TestHoldout3Deterministic(t *testing.T) {
	a, b := mustBuild(t), mustBuild(t)
	ja, _ := a.corpus.Marshal()
	jb, _ := b.corpus.Marshal()
	if !bytes.Equal(ja, jb) || !bytes.Equal(corpus.MarshalDictionary(a.dict), corpus.MarshalDictionary(b.dict)) {
		t.Fatal("holdout3 generator is not deterministic")
	}
}

func TestCommittedHoldout3Spans(t *testing.T) {
	c, err := corpus.Load(filepath.Join(corpusDir, corpusFile))
	if err != nil {
		t.Fatal(err)
	}
	if n := len(c.Items); n < 110 || n > 135 {
		t.Fatalf("expected about 120 items, got %d", n)
	}
	pol := score.DefaultPolicy()
	types := map[string]int{}
	total, checked := 0, 0
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
			if s.Type == corpus.TypeGovID || s.Type == corpus.TypeFinancial {
				if s.Variant == "" {
					t.Errorf("%s: identifier %q needs a variant", it.ID, s.Value)
				}
				if valid := validatorFor(s.Variant); valid != nil {
					checked++
					if !valid(s.Value) {
						t.Errorf("%s: %s %q fails its checksum", it.ID, s.Variant, s.Value)
					}
				}
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
	if total < 1800 {
		t.Errorf("expected at least 1800 spans, got %d", total)
	}
	if checked < 100 {
		t.Errorf("expected at least 100 checksum-validated identifiers, got %d", checked)
	}
	for _, typ := range corpus.EntityTypes {
		if types[typ] == 0 {
			t.Errorf("no spans of type %s", typ)
		}
	}
	if n := len(countries(c)); n < 20 {
		t.Errorf("expected identifiers from at least 20 countries, got %d", n)
	}
}

func TestHoldout3Mix(t *testing.T) {
	b := mustBuild(t)
	fams := map[string]int{}
	negatives, longs, lookalikes, withFacts, multi := 0, 0, 0, 0, 0
	for _, it := range b.corpus.Items {
		fams[it.Tags[0]]++
		if hasTag(it, tagNegative) {
			negatives++
		}
		if hasTag(it, tagMulti) {
			multi++
		}
		if len(it.Facts) > 0 {
			withFacts++
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
	for fam, least := range map[string]int{famCode: 18, famWork: 14, famPersonal: 12, famData: 10, famIntl: 20, famSecrets: 10, famNegatives: 8, famLong: 8} {
		if fams[fam] < least {
			t.Errorf("family %s has %d items, want at least %d", fam, fams[fam], least)
		}
	}
	if len(fams) != len(families) {
		t.Errorf("items use %d families, want %d", len(fams), len(families))
	}
	if negatives < 10 {
		t.Errorf("expected at least 10 items without a sensitive value, got %d", negatives)
	}
	if longs < 8 {
		t.Errorf("expected at least 8 items over %d words, got %d", longWords, longs)
	}
	if lookalikes < 200 {
		t.Errorf("expected at least 200 lookalike hard negatives, got %d", lookalikes)
	}
	if withFacts < 15 {
		t.Errorf("expected at least 15 items with facts, got %d", withFacts)
	}
	if multi < 5 {
		t.Errorf("expected at least 5 multi-turn items, got %d", multi)
	}
}

func loadEarlierNames(t *testing.T) map[string]bool {
	t.Helper()
	var paths []string
	if entries, err := os.ReadDir(earlierPeopleDir); err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".csv") && e.Name() != peopleFile {
				paths = append(paths, filepath.Join(earlierPeopleDir, e.Name()))
			}
		}
	} else {
		for _, name := range earlierPeople {
			paths = append(paths, filepath.Join(corpusDir, name))
		}
	}
	if len(paths) == 0 {
		t.Fatal("no earlier people files to compare against")
	}
	taken := map[string]bool{}
	for _, p := range paths {
		entries, err := corpus.LoadDictionary(p)
		if err != nil {
			t.Fatalf("cannot load an earlier people file: %v", err)
		}
		for _, e := range entries {
			taken[strings.ToLower(e.Value)] = true
			for _, a := range e.Aliases {
				if strings.Contains(a, " ") {
					taken[strings.ToLower(a)] = true
				}
			}
		}
	}
	return taken
}

func TestHoldout3People(t *testing.T) {
	b := mustBuild(t)
	used := 0
	for _, p := range b.reg.order {
		if p.used {
			used++
		} else {
			t.Errorf("person %s is never referenced", p.full)
		}
	}
	if used < 140 {
		t.Errorf("expected about 150 people, got %d", used)
	}
	if len(b.dict) < 2*used/5 || len(b.dict) > 3*used/5 {
		t.Errorf("dictionary should hold roughly half of the people: %d of %d", len(b.dict), used)
	}
	taken := loadEarlierNames(t)
	for _, p := range b.reg.order {
		names := []string{p.full}
		for _, a := range p.aliases {
			if strings.Contains(a, " ") {
				names = append(names, a)
			}
		}
		for _, n := range names {
			if taken[strings.ToLower(n)] {
				t.Errorf("%q is already used in an earlier people file", n)
			}
		}
	}
	dictNames := map[string]bool{}
	for _, e := range b.dict {
		dictNames[e.Value] = true
		if e.Type != corpus.TypePerson {
			t.Errorf("dictionary entry %q has type %s", e.Value, e.Type)
		}
	}
	inDict, outDict := 0, 0
	seen := map[string]bool{}
	items := map[string]map[string]bool{}
	for _, it := range b.corpus.Items {
		for _, s := range it.Spans {
			if s.Type != corpus.TypePerson {
				continue
			}
			if items[s.Person] == nil {
				items[s.Person] = map[string]bool{}
			}
			items[s.Person][it.ID] = true
			if seen[s.Canonical] {
				continue
			}
			seen[s.Canonical] = true
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
	recurring := 0
	for _, in := range items {
		if len(in) >= 3 {
			recurring++
		}
	}
	if recurring < 50 {
		t.Errorf("expected at least 50 people who appear in three or more items, got %d", recurring)
	}
}

func TestHoldout3Hygiene(t *testing.T) {
	b := mustBuild(t)
	allowed := []string{".example", "@example.com", "@example.org", "@example.net", " dot example"}
	for _, it := range b.corpus.Items {
		for _, m := range it.Messages {
			if strings.ContainsRune(m.Content, rune(0x2014)) {
				t.Errorf("%s: contains an em dash", it.ID)
			}
			for _, left := range []string{"{{p|", "{{d|", "{{n|", "{{x|", "[[lb]]", "[[rb]]"} {
				if strings.Contains(m.Content, left) {
					t.Errorf("%s: unparsed markup %q left in the text", it.ID, left)
				}
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
	if n := lint(b); n != 0 {
		t.Errorf("lint reports %d warnings, run `go run ./cmd/genholdout3 -lint`", n)
	}
}
