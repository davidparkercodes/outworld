package corpus

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	TypePerson    = "person"
	TypeEmail     = "email"
	TypePhone     = "phone"
	TypeAddress   = "address"
	TypeDate      = "date"
	TypeOrg       = "org"
	TypeLocation  = "location"
	TypeSecret    = "secret"
	TypeGovID     = "gov_id"
	TypeFinancial = "financial"
	TypeQuasi     = "quasi"
)

var EntityTypes = []string{
	TypePerson, TypeEmail, TypePhone, TypeAddress, TypeDate,
	TypeOrg, TypeLocation, TypeSecret, TypeGovID, TypeFinancial,
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Span struct {
	Msg       int      `json:"msg"`
	Start     int      `json:"start"`
	End       int      `json:"end"`
	Type      string   `json:"type"`
	Value     string   `json:"value"`
	Canonical string   `json:"canonical"`
	Variant   string   `json:"variant,omitempty"`
	Person    string   `json:"person,omitempty"`
	Needles   []string `json:"needles,omitempty"`
}

type Fact struct {
	Date   string `json:"date"`
	Person string `json:"person"`
}

type Item struct {
	ID       string    `json:"id"`
	Source   string    `json:"source"`
	Template string    `json:"template"`
	Tags     []string  `json:"tags,omitempty"`
	Messages []Message `json:"messages"`
	Spans    []Span    `json:"spans"`
	Facts    []Fact    `json:"facts,omitempty"`
}

type Corpus struct {
	Version int    `json:"version"`
	Seed    uint64 `json:"seed"`
	Items   []Item `json:"items"`
}

type DictEntry struct {
	Value   string
	Type    string
	Aliases []string
}

func Load(path string) (*Corpus, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Corpus
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Corpus) Validate() error {
	seen := map[string]bool{}
	for _, it := range c.Items {
		if seen[it.ID] {
			return fmt.Errorf("duplicate item id %s", it.ID)
		}
		seen[it.ID] = true
		for i, s := range it.Spans {
			if s.Msg < 0 || s.Msg >= len(it.Messages) {
				return fmt.Errorf("%s span %d: bad message index %d", it.ID, i, s.Msg)
			}
			content := it.Messages[s.Msg].Content
			if s.Start < 0 || s.End > len(content) || s.Start >= s.End {
				return fmt.Errorf("%s span %d: bad range %d-%d", it.ID, i, s.Start, s.End)
			}
			if content[s.Start:s.End] != s.Value {
				return fmt.Errorf("%s span %d: text %q != value %q", it.ID, i, content[s.Start:s.End], s.Value)
			}
		}
	}
	return nil
}

func (c *Corpus) Marshal() ([]byte, error) {
	var sb strings.Builder
	enc := json.NewEncoder(&sb)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(c); err != nil {
		return nil, err
	}
	return []byte(sb.String()), nil
}

func LoadDictionary(path string) ([]DictEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 || len(rows[0]) < 3 || rows[0][0] != "value" {
		return nil, fmt.Errorf("%s: expected header value,type,aliases", path)
	}
	var out []DictEntry
	for _, r := range rows[1:] {
		e := DictEntry{Value: r[0], Type: r[1]}
		if r[2] != "" {
			e.Aliases = strings.Split(r[2], "|")
		}
		out = append(out, e)
	}
	return out, nil
}

func MarshalDictionary(entries []DictEntry) []byte {
	var sb strings.Builder
	w := csv.NewWriter(&sb)
	_ = w.Write([]string{"value", "type", "aliases"})
	for _, e := range entries {
		_ = w.Write([]string{e.Value, e.Type, strings.Join(e.Aliases, "|")})
	}
	w.Flush()
	return []byte(sb.String())
}

func Luhn(number string) bool {
	sum, n := 0, 0
	for i := len(number) - 1; i >= 0; i-- {
		c := number[i]
		if c < '0' || c > '9' {
			continue
		}
		d := int(c - '0')
		if n%2 == 1 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		n++
	}
	return n > 0 && sum%10 == 0
}

func Digits(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
