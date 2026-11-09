package corpus

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestGenerateDeterministic(t *testing.T) {
	a, da := Generate(DefaultSeed)
	b, db := Generate(DefaultSeed)
	ja, _ := a.Marshal()
	jb, _ := b.Marshal()
	if !bytes.Equal(ja, jb) || !bytes.Equal(MarshalDictionary(da), MarshalDictionary(db)) {
		t.Fatal("generator is not deterministic")
	}
}

func TestCommittedCorpusMatchesGenerator(t *testing.T) {
	c, dict := Generate(DefaultSeed)
	want, _ := c.Marshal()
	got, err := os.ReadFile("corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("corpus/corpus.json is stale: run `task gencorpus`")
	}
	gotDict, err := os.ReadFile("people.csv")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotDict, MarshalDictionary(dict)) {
		t.Fatal("corpus/people.csv is stale: run `task gencorpus`")
	}
}

func TestCorpusShape(t *testing.T) {
	c, dict := Generate(DefaultSeed)
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if n := len(c.Items); n < 140 || n > 170 {
		t.Fatalf("expected about 150 items, got %d", n)
	}
	types := map[string]int{}
	tags := map[string]bool{}
	variants := map[string]bool{}
	for _, it := range c.Items {
		for _, s := range it.Spans {
			types[s.Type]++
			if s.Type == TypePerson {
				variants[s.Variant] = true
			}
			if s.Type == TypeFinancial && strings.HasPrefix(s.Variant, "card-") && !Luhn(s.Value) {
				t.Errorf("%s: card %q fails Luhn", it.ID, s.Value)
			}
			if s.Type == TypeFinancial && s.Variant == "routing" && !abaValid(s.Value) {
				t.Errorf("%s: routing %q fails checksum", it.ID, s.Value)
			}
		}
		for _, tg := range it.Tags {
			tags[tg] = true
		}
	}
	for _, typ := range append(EntityTypes, TypeQuasi) {
		if types[typ] == 0 {
			t.Errorf("no spans of type %s", typ)
		}
	}
	for _, tg := range []string{"name-in-email", "name-in-url", "markdown-split", "all-caps", "common-word-name", "possessive", "nickname", "quasi-identifier", "multi-turn", "same-person-many-ways"} {
		if !tags[tg] {
			t.Errorf("no item tagged %s", tg)
		}
	}
	for _, v := range []string{"full", "first", "nickname", "possessive", "caps", "markdown", "url-slug", "title-last", "lower"} {
		if !variants[v] {
			t.Errorf("no person variant %s", v)
		}
	}
	people := People(DefaultSeed)
	if len(dict) == 0 || len(dict) >= len(people) {
		t.Fatalf("dictionary should cover some but not all people: %d of %d", len(dict), len(people))
	}
}

func abaValid(s string) bool {
	if len(s) != 9 {
		return false
	}
	w := []int{3, 7, 1, 3, 7, 1, 3, 7, 1}
	sum := 0
	for i := range 9 {
		sum += int(s[i]-'0') * w[i]
	}
	return sum%10 == 0
}

func TestLuhn(t *testing.T) {
	for _, n := range testCards {
		if !Luhn(n) {
			t.Errorf("%s should pass Luhn", n)
		}
	}
	if Luhn("4111111111111112") {
		t.Error("bad number passed Luhn")
	}
}

func TestDictionaryRoundTrip(t *testing.T) {
	_, dict := Generate(DefaultSeed)
	path := t.TempDir() + "/p.csv"
	if err := os.WriteFile(path, MarshalDictionary(dict), 0o644); err != nil {
		t.Fatal(err)
	}
	back, err := LoadDictionary(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(back) != len(dict) || back[0].Value != dict[0].Value || len(back[0].Aliases) != len(dict[0].Aliases) {
		t.Fatalf("round trip mismatch: %+v vs %+v", back[0], dict[0])
	}
}
