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
	corpusFile  = "holdout3.json"
	peopleFile  = "holdout3-people.csv"
	sourceName  = "holdout3"
	tagNegative = "negative"
	tagLong     = "long"
	tagMulti    = "multi-turn"
	longWords   = 2000

	famCode      = "code"
	famWork      = "work"
	famPersonal  = "personal"
	famData      = "data"
	famIntl      = "intl"
	famSecrets   = "secrets"
	famNegatives = "negatives"
	famLong      = "long"

	lookalikePrefix = "not-entity:"
	typeReviewed    = "reviewed"
)

var families = []string{famCode, famWork, famPersonal, famData, famIntl, famSecrets, famNegatives, famLong}

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
	fam   string
	genre string
	tags  []string
	msgs  []msg
	facts []string
}

func sys(t string) msg { return msg{"system", t} }
func usr(t string) msg { return msg{"user", t} }
func ast(t string) msg { return msg{"assistant", t} }

func item(fam, genre, tags string, msgs ...msg) def {
	d := def{fam: fam, genre: genre, msgs: msgs}
	for _, t := range strings.Split(tags, ",") {
		if t = strings.TrimSpace(t); t != "" {
			d.tags = append(d.tags, t)
		}
	}
	return d
}

func (d def) on(dateAndPersonKey ...string) def {
	d.facts = append(d.facts, dateAndPersonKey...)
	return d
}

func code(genre, tags string, msgs ...msg) def     { return item(famCode, genre, tags, msgs...) }
func work(genre, tags string, msgs ...msg) def     { return item(famWork, genre, tags, msgs...) }
func personal(genre, tags string, msgs ...msg) def { return item(famPersonal, genre, tags, msgs...) }
func data(genre, tags string, msgs ...msg) def     { return item(famData, genre, tags, msgs...) }
func intl(genre, tags string, msgs ...msg) def     { return item(famIntl, genre, tags, msgs...) }
func secrets(genre, tags string, msgs ...msg) def  { return item(famSecrets, genre, tags, msgs...) }
func negs(genre, tags string, msgs ...msg) def     { return item(famNegatives, genre, tags, msgs...) }
func long(genre, tags string, msgs ...msg) def     { return item(famLong, genre, tags, msgs...) }

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
		p.id = fmt.Sprintf("T%03d", i+1)
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
		p := &person{key: key, full: full, first: full, id: fmt.Sprintf("Z%03d", r.adhoc)}
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

func initialsOf(p *person) string {
	var sb strings.Builder
	for _, part := range []string{p.first, p.last} {
		if r, _ := utf8.DecodeRuneInString(part); r != utf8.RuneError {
			sb.WriteRune(unicode.ToUpper(r))
		}
	}
	return sb.String()
}

func personSurface(p *person, variant string) (string, error) {
	var s string
	switch variant {
	case "full":
		s = p.full
	case "first", "possessive":
		s = p.first
	case "last", "title-last":
		s = p.last
	case "nickname":
		s = p.nick
	case "caps":
		s = strings.ToUpper(p.full)
	case "lower":
		s = strings.ToLower(p.full)
	case "lower-first":
		s = strings.ToLower(p.first)
	case "lower-last":
		s = strings.ToLower(p.last)
	case "caps-first":
		s = strings.ToUpper(p.first)
	case "caps-last":
		s = strings.ToUpper(p.last)
	case "last-first":
		if p.first != "" && p.last != "" {
			s = p.last + ", " + p.first
		}
	case "initials":
		if p.first != "" && p.last != "" {
			s = initialsOf(p)
		}
	default:
		return "", fmt.Errorf("person %s: variant %q needs an explicit surface", p.key, variant)
	}
	if s == "" {
		return "", fmt.Errorf("person %s has no %s form", p.key, variant)
	}
	return s, nil
}

var separators = strings.NewReplacer(" ", "", "-", "", ".", "", "/", "")

var verbatimIDs = map[string]bool{"in-upi": true, "crypto-wallet": true, "br-pix": true, "ar-alias": true, "in-pan-spoken": true}

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
		return strings.TrimRight(addr[:i], "\\")
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
	case "x":
		if err := need(3); err != nil {
			return s, err
		}
		s = corpus.Span{Type: typeReviewed, Variant: fields[1], Value: fields[2], Canonical: fields[2]}
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
		if !verbatimIDs[fields[1]] {
			canon = separators.Replace(fields[2])
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

var fieldEscapes = strings.NewReplacer("[[lb]]", "{{", "[[rb]]", "}}", "[[pipe]]", "|")

func parse(text string, msgIdx int, reg *registry) (string, []corpus.Span, []corpus.Span, error) {
	var sb strings.Builder
	var spans, reviewed []corpus.Span
	for {
		i := strings.Index(text, "{{")
		if i < 0 {
			sb.WriteString(text)
			break
		}
		sb.WriteString(text[:i])
		j := strings.Index(text[i:], "}}")
		if j < 0 {
			return "", nil, nil, fmt.Errorf("unterminated markup near %q", text[i:min(len(text), i+40)])
		}
		body := text[i+2 : i+j]
		text = text[i+j+2:]
		if lit, ok := literals[body]; ok {
			sb.WriteString(lit)
			continue
		}
		fields := strings.Split(body, "|")
		for k := range fields {
			fields[k] = fieldEscapes.Replace(fields[k])
		}
		s, err := makeSpan(fields, reg)
		if err != nil {
			return "", nil, nil, err
		}
		s.Msg = msgIdx
		s.Start = sb.Len()
		sb.WriteString(s.Value)
		s.End = sb.Len()
		if s.Type == typeReviewed {
			reviewed = append(reviewed, s)
		} else {
			spans = append(spans, s)
		}
	}
	return sb.String(), spans, reviewed, nil
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
	dates, persons := map[string]bool{}, map[string]bool{}
	for _, s := range it.Spans {
		content := it.Messages[s.Msg].Content
		if !wordBounded(content, s) {
			return fmt.Errorf("%s: span %q is glued to a letter or digit, the scorer cannot count it", it.ID, s.Value)
		}
		if pol.Sensitive(s.Type) {
			sensitive++
		}
		switch s.Type {
		case corpus.TypeDate:
			dates[s.Canonical] = true
		case corpus.TypePerson:
			persons[s.Canonical] = true
		}
		if it.Messages[s.Msg].Role == "system" && restorable(pol, s.Type) &&
			score.CountWord(conversation.String(), strings.ToLower(s.Value)) == 0 {
			return fmt.Errorf("%s: %q appears only in the system message, restore accuracy cannot be scored for it", it.ID, s.Value)
		}
	}
	for _, f := range it.Facts {
		if !dates[f.Date] || !persons[f.Person] {
			return fmt.Errorf("%s: fact %s / %s has no matching date and person span", it.ID, f.Date, f.Person)
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
	for _, group := range itemGroups() {
		defs = append(defs, group...)
	}
	return defs
}

type built struct {
	corpus   *corpus.Corpus
	dict     []corpus.DictEntry
	reg      *registry
	reviewed map[string][]corpus.Span
}

func build() (*built, error) {
	reg, err := newRegistry()
	if err != nil {
		return nil, err
	}
	pol := score.DefaultPolicy()
	b := &built{corpus: &corpus.Corpus{Version: 1}, reg: reg, reviewed: map[string][]corpus.Span{}}
	for n, d := range allDefs() {
		it := corpus.Item{ID: fmt.Sprintf("h3-%03d", n+1), Source: sourceName, Template: d.genre, Tags: append([]string{d.fam}, d.tags...), Spans: []corpus.Span{}}
		turns := 0
		for mi, m := range d.msgs {
			content, spans, reviewed, err := parse(m.text, mi, reg)
			if err != nil {
				return nil, fmt.Errorf("%s (%s): %w", it.ID, d.genre, err)
			}
			it.Messages = append(it.Messages, corpus.Message{Role: m.role, Content: content})
			it.Spans = append(it.Spans, spans...)
			b.reviewed[it.ID] = append(b.reviewed[it.ID], reviewed...)
			if m.role != "system" {
				turns++
			}
		}
		if turns > 1 {
			it.Tags = append(it.Tags, tagMulti)
		}
		if len(d.facts)%2 != 0 {
			return nil, fmt.Errorf("%s (%s): facts must be date and person key pairs", it.ID, d.genre)
		}
		for i := 0; i+1 < len(d.facts); i += 2 {
			p, err := reg.lookup(d.facts[i+1])
			if err != nil {
				return nil, fmt.Errorf("%s (%s): %w", it.ID, d.genre, err)
			}
			it.Facts = append(it.Facts, corpus.Fact{Date: d.facts[i], Person: p.full})
		}
		if err := checkItem(it, pol); err != nil {
			return nil, fmt.Errorf("%w (%s)", err, d.genre)
		}
		b.corpus.Items = append(b.corpus.Items, it)
	}
	if err := b.corpus.Validate(); err != nil {
		return nil, err
	}
	for _, p := range reg.order {
		if p.used && p.dict {
			b.dict = append(b.dict, corpus.DictEntry{Value: p.full, Type: corpus.TypePerson, Aliases: p.aliases})
		}
	}
	return b, nil
}

const monthWords = `enero|febrero|marzo|abril|mayo|junio|julio|agosto|septiembre|setiembre|octubre|noviembre|diciembre|` +
	`janvier|février|mars|avril|mai|juin|juillet|août|septembre|octobre|novembre|décembre|` +
	`januar|februar|märz|juni|juli|oktober|dezember|` +
	`janeiro|fevereiro|março|maio|junho|julho|setembro|outubro|novembro|dezembro|` +
	`gennaio|febbraio|aprile|maggio|giugno|luglio|settembre|ottobre|dicembre|` +
	`januari|februari|maart|mei|augustus|` +
	`january|february|march|april|may|june|july|august|september|october|november|december`

type lookalike struct {
	kind string
	re   *regexp.Regexp
}

var unannotated = []lookalike{
	{"email", regexp.MustCompile(`[\w.+-]+@[\w-]+(\.[\w-]+)*\.[A-Za-z]{2,}\b`)},
	{"date", regexp.MustCompile(`\b(19|20)\d\d[-/.][01]\d[-/.][0-3]\d\b`)},
	{"date", regexp.MustCompile(`\b[0-3]?\d[/.-][01]?\d[/.-](19|20)\d\d\b`)},
	{"date", regexp.MustCompile(`\b[0-3]?\d/[01]?\d/\d\d\b`)},
	{"date", regexp.MustCompile(`\b[0-3]?\d/[01]?\d\b`)},
	{"date", regexp.MustCompile(`(?i)\b(jan|feb|mar|apr|may|jun|jul|aug|sep|sept|oct|nov|dec)[a-z]*\.? [0-3]?\d\b`)},
	{"date", regexp.MustCompile(`(?i)\b[0-3]?\d(st|nd|rd|th|er|\.|º)? (of |de |di )?(` + monthWords + `)`)},
	{"phone", regexp.MustCompile(`\+\d[\d ().-]{7,}\d`)},
	{"phone", regexp.MustCompile(`\(\d{2,4}\) ?\d{3,4}[ -]\d{3,4}\b`)},
	{"phone", regexp.MustCompile(`\b\d{3}[-. ]\d{3}[-. ]\d{4}\b`)},
	{"phone", regexp.MustCompile(`\b0\d{1,4}[ -]\d{3,4}[ -]\d{3,4}\b`)},
	{"number", regexp.MustCompile(`\b\d{7,}\b`)},
	{"number", regexp.MustCompile(`\b\d{3,4}[ -]\d{3,4}[ -]\d{3,4}\b`)},
	{"account", regexp.MustCompile(`\b[A-Z]{2}\d{2} ?[A-Z0-9]{4} ?\d{4}`)},
}

func blanked(it corpus.Item, mi int, extra []corpus.Span, keep func(corpus.Span) bool) string {
	b := []byte(it.Messages[mi].Content)
	for _, group := range [][]corpus.Span{it.Spans, extra} {
		for _, s := range group {
			if s.Msg == mi && keep(s) {
				for k := s.Start; k < s.End; k++ {
					b[k] = 0
				}
			}
		}
	}
	return string(b)
}

func nameTokens(p *person) []string {
	var toks []string
	for _, t := range []string{p.full, p.first, p.last, p.nick} {
		if utf8.RuneCountInString(t) >= 3 {
			toks = append(toks, strings.ToLower(t))
		}
	}
	for _, part := range strings.FieldsFunc(p.full, func(r rune) bool { return r == ' ' || r == '-' }) {
		if utf8.RuneCountInString(part) >= 4 {
			toks = append(toks, strings.ToLower(part))
		}
	}
	return toks
}

func lint(b *built) int {
	pol := score.DefaultPolicy()
	all := func(corpus.Span) bool { return true }
	warnings := 0
	warn := func(format string, args ...any) {
		warnings++
		fmt.Printf(format+"\n", args...)
	}
	for _, it := range b.corpus.Items {
		var bg strings.Builder
		for mi := range it.Messages {
			text := blanked(it, mi, b.reviewed[it.ID], all)
			bg.WriteString(strings.ToLower(text))
			bg.WriteByte('\n')
			for _, la := range unannotated {
				for _, m := range la.re.FindAllString(text, -1) {
					warn("%s (%s): %q looks like an unannotated %s", it.ID, it.Template, m, la.kind)
				}
			}
		}
		seen := map[string]bool{}
		present := map[string]bool{}
		for _, s := range it.Spans {
			if s.Type == corpus.TypePerson {
				present[s.Person] = true
			}
			v := strings.ToLower(s.Value)
			if !pol.Sensitive(s.Type) || seen[v] {
				continue
			}
			seen[v] = true
			if n := score.CountWord(bg.String(), v); n > 0 {
				warn("%s (%s): %q (%s) also occurs %d time(s) as plain text", it.ID, it.Template, s.Value, s.Type, n)
			}
		}
		for _, p := range b.reg.order {
			for _, tok := range nameTokens(p) {
				if seen[tok] || (commonNames[tok] && !present[p.id]) {
					continue
				}
				seen[tok] = true
				if n := score.CountWord(bg.String(), tok); n > 0 {
					warn("%s (%s): name %q from the people table occurs %d time(s) as plain text", it.ID, it.Template, tok, n)
				}
			}
		}
	}
	return warnings
}

func printCounts(title string, m map[string]int) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Println(title)
	for _, k := range keys {
		fmt.Printf("  %-34s %d\n", k, m[k])
	}
}

func stats(b *built) {
	c := b.corpus
	types, units, variants, kept, idVariants, genres, reasons := map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}
	type famStat struct {
		items, words, spans, long, negative int
		people                              map[string]bool
		types                               map[string]int
	}
	fams := map[string]*famStat{}
	negatives, multi, longs, words, withFacts, facts := 0, 0, 0, 0, 0, 0
	for _, it := range c.Items {
		fs := fams[it.Tags[0]]
		if fs == nil {
			fs = &famStat{people: map[string]bool{}, types: map[string]int{}}
			fams[it.Tags[0]] = fs
		}
		genres[it.Template]++
		wc := wordCount(it)
		words += wc
		fs.items++
		fs.words += wc
		fs.spans += len(it.Spans)
		if hasTag(it, tagNegative) {
			negatives++
			fs.negative++
		}
		if hasTag(it, tagMulti) {
			multi++
		}
		if hasTag(it, tagLong) {
			longs++
			fs.long++
		}
		if len(it.Facts) > 0 {
			withFacts++
			facts += len(it.Facts)
		}
		for _, r := range b.reviewed[it.ID] {
			reasons[r.Variant]++
		}
		seen := map[string]bool{}
		for _, s := range it.Spans {
			types[s.Type]++
			fs.types[s.Type]++
			if k := s.Type + "\x00" + strings.ToLower(s.Value); !seen[k] {
				seen[k] = true
				units[s.Type]++
			}
			switch s.Type {
			case corpus.TypePerson:
				variants[s.Variant]++
				fs.people[s.Person] = true
			case corpus.TypeOrg, corpus.TypeLocation:
				v := s.Variant
				if v == "" {
					v = "(plain)"
				}
				kept[s.Type+"/"+v]++
			case corpus.TypeGovID, corpus.TypeFinancial, corpus.TypeSecret:
				idVariants[s.Type+"/"+s.Variant]++
			}
		}
	}
	used := 0
	for _, p := range b.reg.order {
		if p.used {
			used++
		} else {
			fmt.Printf("unused person: %s\n", p.full)
		}
	}
	total := 0
	for _, n := range types {
		total += n
	}
	fmt.Printf("items %d (negative %d, multi-turn %d, long %d, with facts %d), facts %d, words %d, spans %d, people %d, in dictionary %d, countries %d\n",
		len(c.Items), negatives, multi, longs, withFacts, facts, words, total, used, len(b.dict), len(countries(c)))
	fmt.Println("by family")
	typeOrder := append(append([]string{}, corpus.EntityTypes...), corpus.TypeQuasi)
	fmt.Printf("  %-10s %5s %7s %6s %6s %4s %4s", "family", "items", "words", "spans", "people", "long", "neg")
	for _, t := range typeOrder {
		fmt.Printf(" %9s", t)
	}
	fmt.Println()
	for _, f := range families {
		fs := fams[f]
		if fs == nil {
			continue
		}
		fmt.Printf("  %-10s %5d %7d %6d %6d %4d %4d", f, fs.items, fs.words, fs.spans, len(fs.people), fs.long, fs.negative)
		for _, t := range typeOrder {
			fmt.Printf(" %9d", fs.types[t])
		}
		fmt.Println()
	}
	printCounts("spans by type", types)
	printCounts("scored units by type", units)
	printCounts("person spans by variant", variants)
	printCounts("kept spans by type and variant", kept)
	printCounts("secret, gov_id and financial spans by variant", idVariants)
	printCounts("reviewed plain text by reason", reasons)
	printCounts("items by genre", genres)
}

func countries(c *corpus.Corpus) map[string]bool {
	set := map[string]bool{}
	for _, it := range c.Items {
		for _, s := range it.Spans {
			if s.Type != corpus.TypeGovID && s.Type != corpus.TypeFinancial {
				continue
			}
			if len(s.Variant) > 3 && s.Variant[2] == '-' {
				set[s.Variant[:2]] = true
			}
		}
	}
	return set
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func main() {
	dir := flag.String("out", "corpus", "output directory for holdout3.json and holdout3-people.csv")
	doLint := flag.Bool("lint", false, "list sensitive values and people names that occur as plain text, and text that looks like an unannotated email, phone, date or long number")
	doStats := flag.Bool("stats", false, "print item, word, people and span counts by family, type and variant")
	doMint := flag.String("mint", "", "variant:pattern, fill each ? in the pattern so the identifier passes its checksum")
	flag.Parse()
	if *doMint != "" {
		if err := mint(*doMint); err != nil {
			fail(err)
		}
		return
	}
	b, err := build()
	if err != nil {
		fail(fmt.Errorf("invalid holdout3 corpus: %w", err))
	}
	if *doLint {
		fmt.Printf("lint warnings: %d\n", lint(b))
	}
	if *doStats {
		stats(b)
	}
	out, err := b.corpus.Marshal()
	if err != nil {
		fail(err)
	}
	if err := os.MkdirAll(*dir, 0o755); err != nil {
		fail(err)
	}
	if err := os.WriteFile(filepath.Join(*dir, corpusFile), out, 0o644); err != nil {
		fail(err)
	}
	if err := os.WriteFile(filepath.Join(*dir, peopleFile), corpus.MarshalDictionary(b.dict), 0o644); err != nil {
		fail(err)
	}
	spans := 0
	for _, it := range b.corpus.Items {
		spans += len(it.Spans)
	}
	fmt.Printf("wrote %d items, %d spans, %d dictionary entries to %s\n", len(b.corpus.Items), spans, len(b.dict), *dir)
}
