package textutil

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

func IsWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

func RuneBefore(s string, i int) rune {
	if i <= 0 {
		return 0
	}
	r, _ := utf8.DecodeLastRuneInString(s[:i])
	return r
}

func RuneAt(s string, i int) rune {
	if i >= len(s) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(s[i:])
	return r
}

func IsBoundaryBefore(s string, i int) bool { return !IsWordRune(RuneBefore(s, i)) }

func IsBoundaryAfter(s string, i int) bool { return !IsWordRune(RuneAt(s, i)) }

type Token struct {
	Start, End int
}

func Tokens(s string) []Token {
	var out []Token
	start := -1
	for i, r := range s {
		if IsWordRune(r) {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 {
			out = append(out, Token{start, i})
			start = -1
		}
	}
	if start >= 0 {
		out = append(out, Token{start, len(s)})
	}
	return out
}

func HasLetter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func IsAllUpper(s string) bool {
	seen := false
	for _, r := range s {
		if unicode.IsLower(r) {
			return false
		}
		if unicode.IsUpper(r) {
			seen = true
		}
	}
	return seen
}

func IsAllLower(s string) bool {
	seen := false
	for _, r := range s {
		if unicode.IsUpper(r) {
			return false
		}
		if unicode.IsLower(r) {
			seen = true
		}
	}
	return seen
}

func StartsUpper(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsUpper(r)
}

func MatchCase(src, repl string) string {
	switch {
	case IsAllUpper(src) && utf8.RuneCountInString(src) > 1:
		return strings.ToUpper(repl)
	case IsAllLower(src):
		return strings.ToLower(repl)
	case StartsUpper(src) && IsAllLower(repl):
		r, n := utf8.DecodeRuneInString(repl)
		return string(unicode.ToUpper(r)) + repl[n:]
	}
	return repl
}

func Normalize(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

type CaseRule int

const (
	AnyCase CaseRule = iota
	Capitalized
)

type Match struct {
	Start, End int
	ID         int
}

type phrase struct {
	value  string
	offset int
	id     int
	rule   CaseRule
}

type Matcher struct {
	byFirst map[string][]phrase
	maxLen  int
	n       int
	keys    []string
	dirty   bool
}

func NewMatcher() *Matcher { return &Matcher{byFirst: map[string][]phrase{}} }

func (m *Matcher) Len() int { return m.n }

func (m *Matcher) MaxLen() int { return m.maxLen }

func (m *Matcher) Add(value string, id int, rule CaseRule) {
	toks := Tokens(value)
	if len(toks) == 0 {
		return
	}
	key := strings.ToLower(value[toks[0].Start:toks[0].End])
	m.byFirst[key] = append(m.byFirst[key], phrase{value: value, offset: toks[0].Start, id: id, rule: rule})
	m.dirty = true
	if len(value) > m.maxLen {
		m.maxLen = len(value)
	}
	m.n++
}

func (m *Matcher) candidates(s string, t Token) []phrase {
	return m.byFirst[strings.ToLower(s[t.Start:t.End])]
}

func (m *Matcher) matchAt(s string, t Token, p phrase) (int, int, bool) {
	start := t.Start - p.offset
	end := start + len(p.value)
	if start < 0 || end > len(s) {
		return 0, 0, false
	}
	if !strings.EqualFold(s[start:end], p.value) {
		return 0, 0, false
	}
	first, _ := utf8.DecodeRuneInString(p.value)
	if IsWordRune(first) && !IsBoundaryBefore(s, start) {
		return 0, 0, false
	}
	last, _ := utf8.DecodeLastRuneInString(p.value)
	if IsWordRune(last) && !IsBoundaryAfter(s, end) {
		return 0, 0, false
	}
	if p.rule == Capitalized && !StartsUpper(s[start:end]) {
		return 0, 0, false
	}
	return start, end, true
}

func (m *Matcher) FindAll(s string) []Match {
	if m.n == 0 {
		return nil
	}
	best := map[int]Match{}
	for _, t := range Tokens(s) {
		for _, p := range m.candidates(s, t) {
			start, end, ok := m.matchAt(s, t, p)
			if !ok {
				continue
			}
			if cur, seen := best[start]; !seen || end > cur.End {
				best[start] = Match{start, end, p.id}
			}
		}
	}
	out := make([]Match, 0, len(best))
	for _, v := range best {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	return out
}

func (m *Matcher) sortedKeys() []string {
	if m.keys == nil || m.dirty {
		m.keys = m.keys[:0]
		for k := range m.byFirst {
			m.keys = append(m.keys, k)
		}
		sort.Strings(m.keys)
		m.dirty = false
	}
	return m.keys
}

func (m *Matcher) IsPrefixAt(s string, i int) bool {
	tail := s[i:]
	if tail == "" {
		return false
	}
	toks := Tokens(tail)
	if len(toks) == 0 {
		for _, ps := range m.byFirst {
			for _, p := range ps {
				if len(tail) <= len(p.value) && strings.EqualFold(p.value[:len(tail)], tail) {
					return true
				}
			}
		}
		return false
	}
	lead := tail[:toks[0].Start]
	firstTok := strings.ToLower(tail[toks[0].Start:toks[0].End])
	if len(toks) == 1 && toks[0].End == len(tail) {
		keys := m.sortedKeys()
		j := sort.SearchStrings(keys, firstTok)
		for ; j < len(keys) && strings.HasPrefix(keys[j], firstTok); j++ {
			for _, p := range m.byFirst[keys[j]] {
				if p.offset == len(lead) && strings.EqualFold(p.value[:len(lead)], lead) {
					return true
				}
			}
		}
		return false
	}
	for _, p := range m.byFirst[firstTok] {
		if p.offset != len(lead) || len(tail) > len(p.value) {
			continue
		}
		if strings.EqualFold(p.value[:len(tail)], tail) {
			return true
		}
	}
	return false
}

var commonWords = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`will may june april august mark rose grace hope faith joy bill art sue pat rob jack frank grant chase hunter lane dawn summer autumn winter ray drew gene guy max miles price rich young sky penny ivy lily amber crystal destiny harmony holly iris jasmine ruby star victor sunny honey king rex major dean bishop baker cook carter mason taylor page wood stone hill brown white green black gray grey long short young little rice lee may monday tuesday wednesday thursday friday saturday sunday january february march july september october november december christian angel justice royal sterling don ben van del de la le di da al el`) {
		commonWords[w] = true
	}
}

func IsCommonWord(s string) bool { return commonWords[strings.ToLower(s)] }

func RuleFor(value string) CaseRule {
	toks := Tokens(value)
	if len(toks) == 1 && toks[0].Start == 0 && toks[0].End == len(value) {
		if IsCommonWord(value) || len(value) <= 3 {
			return Capitalized
		}
	}
	return AnyCase
}
