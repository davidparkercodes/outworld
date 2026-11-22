package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/eval/corpus"
	"github.com/davidparkercodes/outworld/eval/score"
)

const (
	corpusFile  = "holdout.json"
	peopleFile  = "holdout-people.csv"
	tagNegative = "negative"
)

type person struct {
	key     string
	full    string
	dict    bool
	aliases []string
	id      string
	used    bool
}

type msg struct {
	role string
	text string
}

type def struct {
	genre string
	tags  []string
	msgs  []msg
}

func sys(t string) msg { return msg{"system", t} }
func usr(t string) msg { return msg{"user", t} }
func ast(t string) msg { return msg{"assistant", t} }

func it(genre, tags string, msgs ...msg) def {
	d := def{genre: genre, msgs: msgs}
	for _, t := range strings.Split(tags, ",") {
		if t = strings.TrimSpace(t); t != "" {
			d.tags = append(d.tags, t)
		}
	}
	return d
}

type registry struct {
	byKey map[string]*person
	order []*person
	adhoc int
}

func newRegistry() (*registry, error) {
	r := &registry{byKey: map[string]*person{}}
	for i := range people {
		p := people[i]
		if r.byKey[p.key] != nil {
			return nil, fmt.Errorf("duplicate person key %s", p.key)
		}
		p.id = fmt.Sprintf("H%02d", i+1)
		r.byKey[p.key] = &p
		r.order = append(r.order, &p)
	}
	return r, nil
}

func (r *registry) lookup(key string) (*person, error) {
	if strings.HasPrefix(key, "+") {
		if p := r.byKey[key]; p != nil {
			return p, nil
		}
		r.adhoc++
		p := &person{key: key, full: key[1:], id: fmt.Sprintf("X%02d", r.adhoc)}
		r.byKey[key] = p
		r.order = append(r.order, p)
		return p, nil
	}
	p := r.byKey[key]
	if p == nil {
		return nil, fmt.Errorf("unknown person key %q", key)
	}
	return p, nil
}

func stripSeparators(s string) string {
	return strings.NewReplacer(" ", "", "-", "").Replace(s)
}

func dateNeedles(iso, surface string) ([]string, error) {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return nil, fmt.Errorf("bad ISO date %q", iso)
	}
	if !strings.Contains(surface, iso[:4]) {
		return nil, nil
	}
	var out []string
	for _, f := range []string{iso, t.Format("January 2, 2006"), t.Format("01/02/2006")} {
		if f != surface {
			out = append(out, f)
		}
	}
	return out, nil
}

func streetLine(addr string) string {
	if i := strings.IndexAny(addr, ",\n"); i > 0 {
		return addr[:i]
	}
	return ""
}

func makeSpan(fields []string, reg *registry) (corpus.Span, error) {
	need := func(n int) error {
		if len(fields) < n {
			return fmt.Errorf("markup %q needs %d fields", strings.Join(fields, "|"), n)
		}
		return nil
	}
	var s corpus.Span
	switch fields[0] {
	case "p":
		if err := need(4); err != nil {
			return s, err
		}
		p, err := reg.lookup(fields[1])
		if err != nil {
			return s, err
		}
		p.used = true
		s = corpus.Span{Type: corpus.TypePerson, Variant: fields[2], Value: fields[3], Canonical: p.full, Person: p.id}
	case "e":
		if err := need(2); err != nil {
			return s, err
		}
		s = corpus.Span{Type: corpus.TypeEmail, Value: fields[1], Canonical: fields[1]}
	case "ph":
		if err := need(2); err != nil {
			return s, err
		}
		d := corpus.Digits(fields[1])
		canon := fields[1]
		switch {
		case len(d) == 10:
			canon = "+1" + d
		case len(d) > 0:
			canon = "+" + d
		}
		s = corpus.Span{Type: corpus.TypePhone, Value: fields[1], Canonical: canon}
		if len(fields) > 2 {
			s.Variant = fields[2]
		}
	case "a":
		if err := need(2); err != nil {
			return s, err
		}
		s = corpus.Span{Type: corpus.TypeAddress, Value: fields[1], Canonical: fields[1]}
		if st := streetLine(fields[1]); st != "" {
			s.Needles = []string{st}
		}
	case "d":
		if err := need(2); err != nil {
			return s, err
		}
		surface := fields[1]
		if len(fields) > 2 {
			surface = fields[2]
		}
		needles, err := dateNeedles(fields[1], surface)
		if err != nil {
			return s, err
		}
		s = corpus.Span{Type: corpus.TypeDate, Value: surface, Canonical: fields[1], Needles: needles}
		if len(fields) > 3 {
			s.Variant = fields[3]
		}
	case "o", "l", "q":
		if err := need(2); err != nil {
			return s, err
		}
		typ := map[string]string{"o": corpus.TypeOrg, "l": corpus.TypeLocation, "q": corpus.TypeQuasi}[fields[0]]
		s = corpus.Span{Type: typ, Value: fields[1], Canonical: fields[1]}
		if len(fields) > 2 {
			s.Variant = fields[2]
		}
	case "s":
		if err := need(3); err != nil {
			return s, err
		}
		s = corpus.Span{Type: corpus.TypeSecret, Variant: fields[1], Value: fields[2], Canonical: fields[2]}
		if len(fields) > 3 {
			s.Needles = fields[3:]
		}
	case "g", "f":
		if err := need(3); err != nil {
			return s, err
		}
		typ := corpus.TypeGovID
		if fields[0] == "f" {
			typ = corpus.TypeFinancial
		}
		canon := fields[2]
		for _, pre := range []string{"ssn", "card", "iban", "sin", "aadhaar", "nhs", "sort-code"} {
			if strings.HasPrefix(fields[1], pre) {
				canon = stripSeparators(fields[2])
			}
		}
		s = corpus.Span{Type: typ, Variant: fields[1], Value: fields[2], Canonical: canon}
		if canon != fields[2] {
			s.Needles = []string{canon}
		}
	default:
		return s, fmt.Errorf("unknown markup kind %q", fields[0])
	}
	if s.Value == "" {
		return s, fmt.Errorf("empty value in markup %q", strings.Join(fields, "|"))
	}
	return s, nil
}

func parse(text string, msgIdx int, reg *registry) (string, []corpus.Span, error) {
	var out strings.Builder
	var spans []corpus.Span
	for {
		i := strings.Index(text, "{{")
		if i < 0 {
			out.WriteString(text)
			break
		}
		out.WriteString(text[:i])
		j := strings.Index(text[i:], "}}")
		if j < 0 {
			return "", nil, fmt.Errorf("unterminated markup near %q", text[i:min(len(text), i+40)])
		}
		body := text[i+2 : i+j]
		text = text[i+j+2:]
		if body == "bt" {
			out.WriteByte('`')
			continue
		}
		s, err := makeSpan(strings.Split(body, "|"), reg)
		if err != nil {
			return "", nil, err
		}
		s.Msg = msgIdx
		s.Start = out.Len()
		out.WriteString(s.Value)
		s.End = out.Len()
		spans = append(spans, s)
	}
	return out.String(), spans, nil
}

func alnum(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

func wordBounded(content string, s corpus.Span) bool {
	first, _ := utf8.DecodeRuneInString(s.Value)
	last, _ := utf8.DecodeLastRuneInString(s.Value)
	if alnum(first) && s.Start > 0 {
		if r, _ := utf8.DecodeLastRuneInString(content[:s.Start]); alnum(r) {
			return false
		}
	}
	if alnum(last) && s.End < len(content) {
		if r, _ := utf8.DecodeRuneInString(content[s.End:]); alnum(r) {
			return false
		}
	}
	return true
}

func restorable(pol score.Policy, typ string) bool {
	return pol[typ] == score.ActionMask || pol[typ] == score.ActionShift
}

func hasTag(item corpus.Item, tag string) bool {
	for _, t := range item.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

func checkItem(item corpus.Item, pol score.Policy) error {
	sensitive := 0
	var conversation strings.Builder
	for _, m := range item.Messages {
		if m.Role != "system" {
			conversation.WriteString(strings.ToLower(m.Content))
			conversation.WriteByte('\n')
		}
	}
	for _, s := range item.Spans {
		content := item.Messages[s.Msg].Content
		if !wordBounded(content, s) {
			return fmt.Errorf("%s: span %q is glued to a letter or digit, the scorer cannot count it", item.ID, s.Value)
		}
		if pol.Sensitive(s.Type) {
			sensitive++
		}
		if item.Messages[s.Msg].Role == "system" && restorable(pol, s.Type) &&
			score.CountWord(conversation.String(), strings.ToLower(s.Value)) == 0 {
			return fmt.Errorf("%s: %q appears only in the system message, restore accuracy cannot be scored for it", item.ID, s.Value)
		}
	}
	if neg := hasTag(item, tagNegative); neg != (sensitive == 0) {
		return fmt.Errorf("%s: negative tag is %v but the item has %d sensitive spans", item.ID, neg, sensitive)
	}
	return nil
}

func allDefs() []def {
	var out []def
	for _, group := range [][]def{itemsChat, itemsWork, itemsTech, itemsDocs, itemsNegative, itemsMultiTurn} {
		out = append(out, group...)
	}
	return out
}

func build() (*corpus.Corpus, []corpus.DictEntry, *registry, error) {
	reg, err := newRegistry()
	if err != nil {
		return nil, nil, nil, err
	}
	pol := score.DefaultPolicy()
	c := &corpus.Corpus{Version: 1}
	for n, d := range allDefs() {
		item := corpus.Item{ID: fmt.Sprintf("ho-%03d", n+1), Source: "holdout", Template: d.genre, Tags: append([]string{}, d.tags...), Spans: []corpus.Span{}}
		turns := 0
		for mi, m := range d.msgs {
			content, spans, err := parse(m.text, mi, reg)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("%s: %w", item.ID, err)
			}
			item.Messages = append(item.Messages, corpus.Message{Role: m.role, Content: content})
			item.Spans = append(item.Spans, spans...)
			if m.role != "system" {
				turns++
			}
		}
		if turns > 1 {
			item.Tags = append(item.Tags, "multi-turn")
		}
		if err := checkItem(item, pol); err != nil {
			return nil, nil, nil, err
		}
		c.Items = append(c.Items, item)
	}
	if err := c.Validate(); err != nil {
		return nil, nil, nil, err
	}
	var dict []corpus.DictEntry
	for _, p := range reg.order {
		if p.used && p.dict {
			dict = append(dict, corpus.DictEntry{Value: p.full, Type: corpus.TypePerson, Aliases: p.aliases})
		}
	}
	return c, dict, reg, nil
}

func lint(c *corpus.Corpus) {
	pol := score.DefaultPolicy()
	for _, item := range c.Items {
		var bg strings.Builder
		for mi, m := range item.Messages {
			b := []byte(m.Content)
			for _, s := range item.Spans {
				if s.Msg == mi && pol.Sensitive(s.Type) {
					for k := s.Start; k < s.End; k++ {
						b[k] = 0
					}
				}
			}
			bg.WriteString(strings.ToLower(string(b)))
			bg.WriteByte('\n')
		}
		seen := map[string]bool{}
		for _, s := range item.Spans {
			v := strings.ToLower(s.Value)
			if !pol.Sensitive(s.Type) || seen[v] {
				continue
			}
			seen[v] = true
			if n := score.CountWord(bg.String(), v); n > 0 {
				fmt.Printf("%s: %q (%s) also occurs %d time(s) outside sensitive spans\n", item.ID, s.Value, s.Type, n)
			}
		}
	}
}

func printCounts(title string, m map[string]int) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Println(title)
	for _, k := range keys {
		fmt.Printf("  %-16s %d\n", k, m[k])
	}
}

func stats(c *corpus.Corpus, dict []corpus.DictEntry, reg *registry) {
	types, variants, genres := map[string]int{}, map[string]int{}, map[string]int{}
	negatives, multi := 0, 0
	for _, item := range c.Items {
		genres[item.Template]++
		if hasTag(item, tagNegative) {
			negatives++
		}
		if hasTag(item, "multi-turn") {
			multi++
		}
		for _, s := range item.Spans {
			types[s.Type]++
			if s.Type == corpus.TypePerson {
				variants[s.Variant]++
			}
		}
	}
	used := 0
	for _, p := range reg.order {
		if p.used {
			used++
		} else {
			fmt.Printf("unused person: %s\n", p.full)
		}
	}
	fmt.Printf("items %d (negative %d, multi-turn %d), people %d, in dictionary %d\n", len(c.Items), negatives, multi, used, len(dict))
	printCounts("spans by type", types)
	printCounts("person spans by variant", variants)
	printCounts("items by genre", genres)
}

func main() {
	out := flag.String("out", "corpus", "output directory for holdout.json and holdout-people.csv")
	doLint := flag.Bool("lint", false, "list sensitive values that also occur outside sensitive spans")
	doStats := flag.Bool("stats", false, "print span counts by type, person variant and genre")
	flag.Parse()
	c, dict, reg, err := build()
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid holdout corpus:", err)
		os.Exit(1)
	}
	if *doLint {
		lint(c)
	}
	if *doStats {
		stats(c, dict, reg)
	}
	data, err := c.Marshal()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(*out, corpusFile), data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(*out, peopleFile), corpus.MarshalDictionary(dict), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	spans := 0
	for _, item := range c.Items {
		spans += len(item.Spans)
	}
	fmt.Printf("wrote %d items, %d spans, %d dictionary entries to %s\n", len(c.Items), spans, len(dict), *out)
}
