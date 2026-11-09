package detect

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/entity"
)

type Entry struct {
	Value      string      `json:"value"`
	Type       entity.Type `json:"type"`
	Aliases    []string    `json:"aliases,omitempty"`
	Generalize string      `json:"generalize,omitempty"`
	Gender     string      `json:"gender,omitempty"`
}

type Dictionary struct {
	Entries []Entry
}

func LoadDictionary(path string) (*Dictionary, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open dictionary: %w", err)
	}
	defer f.Close()
	if strings.EqualFold(filepath.Ext(path), ".json") {
		return ParseDictionaryJSON(f)
	}
	return ParseDictionaryCSV(f)
}

func ParseDictionaryJSON(r io.Reader) (*Dictionary, error) {
	var raw []struct {
		Value      string          `json:"value"`
		Type       string          `json:"type"`
		Aliases    json.RawMessage `json:"aliases"`
		Generalize string          `json:"generalize"`
		Gender     string          `json:"gender"`
	}
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("parse dictionary json: %w", err)
	}
	d := &Dictionary{}
	for i, e := range raw {
		var aliases []string
		if len(e.Aliases) > 0 {
			var list []string
			var one string
			if json.Unmarshal(e.Aliases, &list) == nil {
				aliases = list
			} else if json.Unmarshal(e.Aliases, &one) == nil {
				aliases = splitAliases(one)
			}
		}
		if err := d.add(e.Value, e.Type, aliases, e.Generalize, e.Gender); err != nil {
			return nil, fmt.Errorf("dictionary entry %d: %w", i, err)
		}
	}
	return d, nil
}

func ParseDictionaryCSV(r io.Reader) (*Dictionary, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cr.TrimLeadingSpace = true
	cr.Comment = '#'
	d := &Dictionary{}
	cols := map[string]int{"value": 0, "type": 1, "aliases": 2, "generalize": 3, "gender": 4}
	line := 0
	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse dictionary csv: %w", err)
		}
		line++
		if len(rec) == 0 || strings.TrimSpace(rec[0]) == "" {
			continue
		}
		if line == 1 && strings.EqualFold(strings.TrimSpace(rec[0]), "value") {
			cols = headerColumns(rec, cols)
			continue
		}
		field := func(name string) string {
			if i, ok := cols[name]; ok && i < len(rec) {
				return strings.TrimSpace(rec[i])
			}
			return ""
		}
		typ := field("type")
		if typ == "" {
			typ = "person"
		}
		if err := d.add(field("value"), typ, splitAliases(field("aliases")), field("generalize"), field("gender")); err != nil {
			return nil, fmt.Errorf("dictionary line %d: %w", line, err)
		}
	}
	return d, nil
}

func headerColumns(rec []string, defaults map[string]int) map[string]int {
	cols := map[string]int{}
	claimed := map[int]bool{}
	for i, h := range rec {
		cols[strings.ToLower(strings.TrimSpace(h))] = i
		claimed[i] = true
	}
	for name, i := range defaults {
		if _, ok := cols[name]; !ok && !claimed[i] {
			cols[name] = i
			claimed[i] = true
		}
	}
	return cols
}

func splitAliases(s string) []string {
	var out []string
	for _, a := range strings.Split(s, "|") {
		if a = strings.TrimSpace(a); a != "" {
			out = append(out, a)
		}
	}
	return out
}

func (d *Dictionary) add(value, typ string, aliases []string, gen, gender string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("empty value")
	}
	t, err := entity.ParseType(typ)
	if err != nil {
		return err
	}
	g := strings.ToLower(strings.TrimSpace(gender))
	switch g {
	case "", "f", "m", "n", "female", "male", "neutral", "nonbinary", "x":
	default:
		return fmt.Errorf("unknown gender %q (use f, m or n)", gender)
	}
	d.Entries = append(d.Entries, Entry{Value: value, Type: t, Aliases: aliases, Generalize: gen, Gender: g})
	return nil
}

func (d *Dictionary) Values() []string {
	var out []string
	for _, e := range d.Entries {
		out = append(out, e.Value)
		out = append(out, e.Aliases...)
	}
	return out
}

type dictPhrase struct {
	entry int
	text  string
	weak  bool
}

type DictionaryDetector struct {
	dict    *Dictionary
	matcher *textutil.Matcher
	phrases []dictPhrase
}

func (d *Dictionary) Detector() *DictionaryDetector {
	dd := &DictionaryDetector{dict: d, matcher: textutil.NewMatcher()}
	seen := map[string]bool{}
	add := func(entry int, text string, person bool, rules ...textutil.CaseRule) {
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}
		key := strings.ToLower(text)
		if seen[key] {
			return
		}
		seen[key] = true
		single := len(textutil.Tokens(text)) == 1
		weak := single && (textutil.IsCommonWord(text) || utf8.RuneCountInString(text) <= 3)
		rule := textutil.AnyCase
		if single && !person {
			rule = textutil.RuleFor(text)
		}
		if len(rules) > 0 {
			rule = rules[0]
		}
		dd.phrases = append(dd.phrases, dictPhrase{entry: entry, text: text, weak: weak && person})
		dd.matcher.Add(text, len(dd.phrases)-1, rule)
	}
	for i, e := range d.Entries {
		person := e.Type == entity.Person
		add(i, e.Value, person)
		for _, a := range e.Aliases {
			add(i, a, person)
		}
		if !person {
			continue
		}
		for _, form := range append([]string{e.Value}, e.Aliases...) {
			toks := textutil.Tokens(form)
			if len(toks) < 2 {
				continue
			}
			var words []string
			for _, t := range toks {
				words = append(words, strings.ToLower(form[t.Start:t.End]))
			}
			first, rest := words[0], words[1:]
			if len(first) >= 2 {
				add(i, first, true)
			}
			if last := words[len(words)-1]; len(last) >= 2 {
				add(i, last, true, textutil.Capitalized)
			}
			if len(rest) > 1 {
				add(i, strings.Join(rest, "-"), true)
			}
			for _, sep := range []string{".", "_", "-", ""} {
				add(i, strings.Join(words, sep), true)
				add(i, first+sep+strings.Join(rest, ""), true)
				add(i, first+sep+strings.Join(rest, sep), true)
			}
			if len(first) < 2 {
				continue
			}
			restJoined := strings.Join(rest, "")
			fi, ri := string([]rune(first)[:1]), string([]rune(restJoined)[:1])
			for _, h := range []string{fi + restJoined, fi + "." + restJoined, fi + "_" + restJoined, first + ri, first + "_" + ri, first + "." + ri} {
				if utf8.RuneCountInString(h) >= 5 {
					add(i, h, true)
				}
			}
		}
	}
	return dd
}

func (d *DictionaryDetector) Name() string { return "dictionary" }

func (d *DictionaryDetector) Detect(_ context.Context, text string) ([]entity.Span, error) {
	var out []entity.Span
	for _, m := range d.matcher.FindAll(text) {
		p := d.phrases[m.ID]
		if p.weak && !NameUseOK(text, m.Start, m.End) {
			continue
		}
		e := d.dict.Entries[p.entry]
		out = append(out, entity.Span{
			Start: m.Start, End: m.End, Type: e.Type, Label: string(e.Type), Text: text[m.Start:m.End],
			Source: "dictionary", Priority: 65, Score: 1, Canonical: e.Value, Generalized: e.Generalize, Gender: genderCode(e.Gender),
		})
	}
	return out, nil
}

func genderCode(g string) string {
	switch g {
	case "f", "female":
		return "f"
	case "m", "male":
		return "m"
	case "n", "neutral", "nonbinary", "x":
		return "n"
	}
	return ""
}

func (d *DictionaryDetector) Phrases() []string {
	out := make([]string, 0, len(d.phrases))
	for _, p := range d.phrases {
		out = append(out, p.text)
	}
	return out
}
