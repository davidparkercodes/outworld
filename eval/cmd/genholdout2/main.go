package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/eval/corpus"
	"github.com/davidparkercodes/outworld/eval/score"
)

const (
	corpusFile  = "holdout2.json"
	peopleFile  = "holdout2-people.csv"
	sourceName  = "holdout2"
	tagNegative = "negative"
	tagLong     = "long"
	longWords   = 2000

	catCode     = "code"
	catBusiness = "business"
	catPersonal = "personal"
	catLong     = "long-or-multi-turn"

	lookalikePrefix = "not-entity:"
)

type person struct {
	key     string
	full    string
	first   string
	last    string
	nick    string
	dict    bool
	aliases []string
	id      string
	used    bool
}

func in(key, full, first, last, nick string, aliases ...string) person {
	return person{key: key, full: full, first: first, last: last, nick: nick, dict: true, aliases: aliases}
}

func out(key, full, first, last, nick string) person {
	return person{key: key, full: full, first: first, last: last, nick: nick}
}

type msg struct {
	role string
	text string
}

type def struct {
	cat   string
	genre string
	tags  []string
	msgs  []msg
}

func sys(t string) msg { return msg{"system", t} }
func usr(t string) msg { return msg{"user", t} }
func ast(t string) msg { return msg{"assistant", t} }

func item(cat, genre, tags string, msgs ...msg) def {
	d := def{cat: cat, genre: genre, msgs: msgs}
	for _, t := range strings.Split(tags, ",") {
		if t = strings.TrimSpace(t); t != "" {
			d.tags = append(d.tags, t)
		}
	}
	return d
}

func code(genre, tags string, msgs ...msg) def     { return item(catCode, genre, tags, msgs...) }
func business(genre, tags string, msgs ...msg) def { return item(catBusiness, genre, tags, msgs...) }
func personal(genre, tags string, msgs ...msg) def { return item(catPersonal, genre, tags, msgs...) }
func long(genre, tags string, msgs ...msg) def     { return item(catLong, genre, tags, msgs...) }

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
		p.id = fmt.Sprintf("S%03d", i+1)
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
		full := key[1:]
		p := &person{key: key, full: full, first: full, id: fmt.Sprintf("Y%03d", r.adhoc)}
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

func personSurface(p *person, variant string) (string, error) {
	var s string
	switch variant {
	case "full":
		s = p.full
	case "first", "possessive":
		s = p.first
	case "last":
		s = p.last
	case "nickname":
		s = p.nick
	case "caps":
		s = strings.ToUpper(p.full)
	case "lower":
		s = strings.ToLower(p.full)
	case "lower-first":
		s = strings.ToLower(p.first)
	default:
		return "", fmt.Errorf("person %s: variant %q needs an explicit surface", p.key, variant)
	}
	if s == "" {
		return "", fmt.Errorf("person %s has no %s form", p.key, variant)
	}
	return s, nil
}

var separators = strings.NewReplacer(" ", "", "-", "", ".", "", "/", "")

var separatedIDs = []string{"ssn", "card", "iban", "sin", "aadhaar", "nhs", "sort-code", "cpf", "nino", "tfn", "routing"}

func dateNeedles(iso, surface string) ([]string, error) {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return nil, fmt.Errorf("bad ISO date %q", iso)
	}
	if !strings.Contains(surface, iso[:4]) {
		return nil, nil
	}
	var needles []string
	for _, f := range []string{iso, t.Format("January 2, 2006"), t.Format("01/02/2006")} {
		if f != surface {
			needles = append(needles, f)
		}
	}
	return needles, nil
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
		if err := need(2); err != nil {
			return s, err
		}
		p, err := reg.lookup(fields[1])
		if err != nil {
			return s, err
		}
		p.used = true
		variant := "full"
		if len(fields) > 2 {
			variant = fields[2]
		}
		var surface string
		if len(fields) > 3 {
			surface = fields[3]
		} else if surface, err = personSurface(p, variant); err != nil {
			if !strings.HasPrefix(p.key, "+") {
				return s, err
			}
			surface = p.full
		}
		s = corpus.Span{Type: corpus.TypePerson, Variant: variant, Value: surface, Canonical: p.full, Person: p.id}
	case "e":
		if err := need(2); err != nil {
			return s, err
		}
		s = corpus.Span{Type: corpus.TypeEmail, Value: fields[1], Canonical: strings.ToLower(fields[1])}
	case "ph":
		if err := need(2); err != nil {
			return s, err
		}
		d := corpus.Digits(fields[1])
		canon := fields[1]
		if len(d) > 0 {
			canon = d
			if strings.HasPrefix(strings.TrimSpace(fields[1]), "+") {
				canon = "+" + d
			}
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
		if len(fields) > 2 {
			s.Variant = fields[2]
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
	case "n":
		if err := need(3); err != nil {
			return s, err
		}
		s = corpus.Span{Type: corpus.TypeOrg, Variant: lookalikePrefix + fields[1], Value: fields[2], Canonical: fields[2]}
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
		for _, pre := range separatedIDs {
			if strings.HasPrefix(fields[1], pre) {
				canon = separators.Replace(fields[2])
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

var literals = map[string]string{"bt": "`", "fence": "```", "lb": "{{", "rb": "}}", "pipe": "|"}

func parse(text string, msgIdx int, reg *registry) (string, []corpus.Span, error) {
	var sb strings.Builder
	var spans []corpus.Span
	for {
		i := strings.Index(text, "{{")
		if i < 0 {
			sb.WriteString(text)
			break
		}
		sb.WriteString(text[:i])
		j := strings.Index(text[i:], "}}")
		if j < 0 {
			return "", nil, fmt.Errorf("unterminated markup near %q", text[i:min(len(text), i+40)])
		}
		body := text[i+2 : i+j]
		text = text[i+j+2:]
		if lit, ok := literals[body]; ok {
			sb.WriteString(lit)
			continue
		}
		s, err := makeSpan(strings.Split(body, "|"), reg)
		if err != nil {
			return "", nil, err
		}
		s.Msg = msgIdx
		s.Start = sb.Len()
		sb.WriteString(s.Value)
		s.End = sb.Len()
		spans = append(spans, s)
	}
	return sb.String(), spans, nil
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

func hasTag(it corpus.Item, tag string) bool {
	for _, t := range it.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

func wordCount(it corpus.Item) int {
	n := 0
	for _, m := range it.Messages {
		n += len(strings.Fields(m.Content))
	}
	return n
}

func checkItem(it corpus.Item, pol score.Policy) error {
	sensitive := 0
	var conversation strings.Builder
	want := "user"
	for _, m := range it.Messages {
		if m.Role == "system" {
			continue
		}
		if m.Role != want {
			return fmt.Errorf("%s: turns must alternate user and assistant, starting with user", it.ID)
		}
		if want == "user" {
			want = "assistant"
		} else {
			want = "user"
		}
		conversation.WriteString(strings.ToLower(m.Content))
		conversation.WriteByte('\n')
	}
	if want != "assistant" {
		return fmt.Errorf("%s: the last turn must be a user turn", it.ID)
	}
	for _, s := range it.Spans {
		content := it.Messages[s.Msg].Content
		if !wordBounded(content, s) {
			return fmt.Errorf("%s: span %q is glued to a letter or digit, the scorer cannot count it", it.ID, s.Value)
		}
		if pol.Sensitive(s.Type) {
			sensitive++
		}
		if it.Messages[s.Msg].Role == "system" && restorable(pol, s.Type) &&
			score.CountWord(conversation.String(), strings.ToLower(s.Value)) == 0 {
			return fmt.Errorf("%s: %q appears only in the system message, restore accuracy cannot be scored for it", it.ID, s.Value)
		}
	}
	if neg := hasTag(it, tagNegative); neg != (sensitive == 0) {
		return fmt.Errorf("%s: negative tag is %v but the item has %d sensitive spans", it.ID, neg, sensitive)
	}
	if long := hasTag(it, tagLong); long != (wordCount(it) > longWords) {
		return fmt.Errorf("%s: long tag is %v but the item has %d words", it.ID, long, wordCount(it))
	}
	return nil
}

func allDefs() []def {
	var defs []def
	for _, group := range [][]def{itemsCode, itemsCodeData, itemsBusiness, itemsPersonal, itemsConversations, longItems()} {
		defs = append(defs, group...)
	}
	return defs
}

func build() (*corpus.Corpus, []corpus.DictEntry, *registry, error) {
	reg, err := newRegistry()
	if err != nil {
		return nil, nil, nil, err
	}
	pol := score.DefaultPolicy()
	c := &corpus.Corpus{Version: 1}
	for n, d := range allDefs() {
		it := corpus.Item{ID: fmt.Sprintf("h2-%03d", n+1), Source: sourceName, Template: d.genre, Tags: append([]string{d.cat}, d.tags...), Spans: []corpus.Span{}}
		turns := 0
		for mi, m := range d.msgs {
			content, spans, err := parse(m.text, mi, reg)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("%s (%s): %w", it.ID, d.genre, err)
			}
			it.Messages = append(it.Messages, corpus.Message{Role: m.role, Content: content})
			it.Spans = append(it.Spans, spans...)
			if m.role != "system" {
				turns++
			}
		}
		if turns > 1 {
			it.Tags = append(it.Tags, "multi-turn")
		}
		if err := checkItem(it, pol); err != nil {
			return nil, nil, nil, fmt.Errorf("%w (%s)", err, d.genre)
		}
		c.Items = append(c.Items, it)
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

var unannotated = []*regexp.Regexp{
	regexp.MustCompile(`[\w.+-]+@[\w-]+(\.[\w-]+)*\.[A-Za-z]{2,}\b`),
	regexp.MustCompile(`\b(19|20)\d\d-[01]\d-[0-3]\d\b`),
	regexp.MustCompile(`\b[0-3]?\d[/.][01]?\d[/.](19|20)\d\d\b`),
	regexp.MustCompile(`(?i)\b(jan|feb|mar|apr|may|jun|jul|aug|sep|sept|oct|nov|dec)[a-z]*\.? [0-3]?\d\b`),
	regexp.MustCompile(`(?i)\b[0-3]?\d(st|nd|rd|th)? (of )?(january|february|march|april|may|june|july|august|september|october|november|december)\b`),
}

func blanked(it corpus.Item, mi int, keep func(corpus.Span) bool) string {
	b := []byte(it.Messages[mi].Content)
	for _, s := range it.Spans {
		if s.Msg == mi && keep(s) {
			for k := s.Start; k < s.End; k++ {
				b[k] = 0
			}
		}
	}
	return string(b)
}

func lint(c *corpus.Corpus) {
	pol := score.DefaultPolicy()
	sensitive := func(s corpus.Span) bool { return pol.Sensitive(s.Type) }
	for _, it := range c.Items {
		var bg strings.Builder
		for mi := range it.Messages {
			text := blanked(it, mi, sensitive)
			bg.WriteString(strings.ToLower(text))
			bg.WriteByte('\n')
			for _, re := range unannotated {
				for _, m := range re.FindAllString(text, -1) {
					fmt.Printf("%s: %q looks like an unannotated email or date\n", it.ID, m)
				}
			}
		}
		seen := map[string]bool{}
		for _, s := range it.Spans {
			v := strings.ToLower(s.Value)
			if !pol.Sensitive(s.Type) || seen[v] {
				continue
			}
			seen[v] = true
			if n := score.CountWord(bg.String(), v); n > 0 {
				fmt.Printf("%s: %q (%s) also occurs %d time(s) outside sensitive spans\n", it.ID, s.Value, s.Type, n)
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
		fmt.Printf("  %-28s %d\n", k, m[k])
	}
}

func stats(c *corpus.Corpus, dict []corpus.DictEntry, reg *registry) {
	types, units, variants, kept, genres, cats := map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}
	negatives, multi, longs, words := 0, 0, 0, 0
	for _, it := range c.Items {
		genres[it.Template]++
		cats[it.Tags[0]]++
		words += wordCount(it)
		if hasTag(it, tagNegative) {
			negatives++
		}
		if hasTag(it, "multi-turn") {
			multi++
		}
		if hasTag(it, tagLong) {
			longs++
		}
		seen := map[string]bool{}
		for _, s := range it.Spans {
			types[s.Type]++
			if k := s.Type + "\x00" + strings.ToLower(s.Value); !seen[k] {
				seen[k] = true
				units[s.Type]++
			}
			switch s.Type {
			case corpus.TypePerson:
				variants[s.Variant]++
			case corpus.TypeOrg, corpus.TypeLocation:
				v := s.Variant
				if v == "" {
					v = "(plain)"
				}
				kept[s.Type+"/"+v]++
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
	fmt.Printf("items %d (negative %d, multi-turn %d, long %d), words %d, people %d, in dictionary %d\n", len(c.Items), negatives, multi, longs, words, used, len(dict))
	printCounts("items by category", cats)
	printCounts("spans by type", types)
	printCounts("scored units by type", units)
	printCounts("person spans by variant", variants)
	printCounts("kept spans by type and variant", kept)
	printCounts("items by genre", genres)
}

func main() {
	dir := flag.String("out", "corpus", "output directory for holdout2.json and holdout2-people.csv")
	doLint := flag.Bool("lint", false, "list sensitive values that also occur outside sensitive spans, and text that looks like an unannotated email or date")
	doStats := flag.Bool("stats", false, "print span counts by type, person variant, kept variant and genre")
	flag.Parse()
	c, dict, reg, err := build()
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid holdout2 corpus:", err)
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
	if err := os.MkdirAll(*dir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(*dir, corpusFile), data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(*dir, peopleFile), corpus.MarshalDictionary(dict), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	spans := 0
	for _, it := range c.Items {
		spans += len(it.Spans)
	}
	fmt.Printf("wrote %d items, %d spans, %d dictionary entries to %s\n", len(c.Items), spans, len(dict), *dir)
}
