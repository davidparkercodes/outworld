package outworld

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/entity"
	"github.com/davidparkercodes/outworld/pkg/surrogate"
)

type nameRole int

const (
	roleGiven nameRole = iota
	roleFamily
	roleNick
)

func (e *Engine) knownGiven(word string) bool {
	if word == "" {
		return false
	}
	if _, ok := e.vault.Get("person.given", word); ok {
		return true
	}
	return surrogate.GenderOf(word) != "" && !textutil.IsCommonWord(word)
}

func canonicalWords(canonical string) []string {
	var out []string
	for _, t := range textutil.Tokens(canonical) {
		if w := canonical[t.Start:t.End]; utf8.RuneCountInString(w) > 1 {
			out = append(out, w)
		}
	}
	return out
}

func (e *Engine) personSurrogate(sp entity.Span, avoid func(string) bool) (string, error) {
	text := sp.Text
	toks := textutil.Tokens(text)
	if len(toks) == 0 {
		return text, nil
	}
	canon := canonicalWords(sp.Canonical)
	gender := sp.Gender
	if gender == "" && len(canon) > 0 {
		gender = surrogate.GenderOf(canon[0])
	}
	if len(toks) == 1 && len(canon) >= 2 {
		if sur, ok, err := e.joinedSurrogate(text, canon, gender, avoid); ok || err != nil {
			return sur, err
		}
	}
	var b strings.Builder
	last := 0
	for i, t := range toks {
		word := text[t.Start:t.End]
		b.WriteString(text[last:t.Start])
		last = t.End
		if utf8.RuneCountInString(word) == 1 {
			b.WriteString(word)
			continue
		}
		var p string
		var err error
		real := realCasing(word, canon)
		switch e.role(word, i, len(toks), canon, sp) {
		case roleFamily:
			p, err = e.familySurrogate(real, avoid)
		case roleNick:
			p, err = e.nickSurrogate(real, canon[0], gender, avoid)
		default:
			g := gender
			if len(canon) == 0 || !strings.EqualFold(word, canon[0]) {
				g = surrogate.GenderOf(word)
			}
			p, err = e.givenSurrogate(real, g, avoid)
		}
		if err != nil {
			return "", err
		}
		b.WriteString(textutil.MatchCase(word, p))
	}
	b.WriteString(text[last:])
	return b.String(), nil
}

func realCasing(word string, canon []string) string {
	for _, c := range canon {
		if strings.EqualFold(word, c) {
			return c
		}
	}
	if textutil.IsAllUpper(word) || textutil.IsAllLower(word) {
		r := []rune(strings.ToLower(word))
		r[0] = unicode.ToUpper(r[0])
		return string(r)
	}
	return word
}

func (e *Engine) role(word string, i, n int, canon []string, sp entity.Span) nameRole {
	if len(canon) >= 2 {
		if strings.EqualFold(word, canon[0]) {
			return roleGiven
		}
		for _, c := range canon[1:] {
			if strings.EqualFold(word, c) {
				return roleFamily
			}
		}
		if i == 0 {
			return roleNick
		}
		return roleFamily
	}
	if len(canon) == 1 {
		return roleGiven
	}
	if n >= 2 {
		if i == 0 {
			return roleGiven
		}
		return roleFamily
	}
	if strings.HasPrefix(sp.Source, "regex:honorific") || sp.Source == "collision:family" {
		return roleFamily
	}
	_, isGiven := e.vault.Get("person.given", word)
	_, isFamily := e.vault.Get("person.family", word)
	if isFamily && !isGiven {
		return roleFamily
	}
	return roleGiven
}

func (e *Engine) givenSurrogate(word, gender string, avoid func(string) bool) (string, error) {
	return e.vault.Assign("person.given", word, func(i int) string {
		return avoided(e.gen.GivenNameFor(word, gender, i), avoid)
	})
}

func (e *Engine) familySurrogate(word string, avoid func(string) bool) (string, error) {
	return e.vault.Assign("person.family", word, func(i int) string {
		return avoided(e.gen.FamilyName(word, i), avoid)
	})
}

func (e *Engine) nickSurrogate(word, first, gender string, avoid func(string) bool) (string, error) {
	g, err := e.givenSurrogate(first, gender, avoid)
	if err != nil {
		return "", err
	}
	return e.vault.Assign("person.nick", word, func(i int) string {
		return avoided(surrogate.Nickname(word, first, g, i), avoid)
	})
}

func avoided(cand string, avoid func(string) bool) string {
	if avoid != nil && avoid(cand) {
		return ""
	}
	return cand
}

func (e *Engine) joinedSurrogate(text string, canon []string, gender string, avoid func(string) bool) (string, bool, error) {
	lw := strings.ToLower(text)
	first := strings.ToLower(canon[0])
	var restWords []string
	for _, w := range canon[1:] {
		restWords = append(restWords, strings.ToLower(w))
	}
	rest := strings.Join(restWords, "")
	if lw == first || strings.Contains(" "+strings.Join(restWords, " ")+" ", " "+lw+" ") {
		return "", false, nil
	}
	firstInitial, restInitial := string([]rune(first)[:1]), string([]rune(rest)[:1])
	var shape string
	switch lw {
	case first + rest:
		shape = "full"
	case firstInitial + rest:
		shape = "initial-last"
	case first + restInitial:
		shape = "first-initial"
	default:
		return "", false, nil
	}
	g, err := e.givenSurrogate(canon[0], gender, avoid)
	if err != nil {
		return "", true, err
	}
	var fam strings.Builder
	for _, w := range canon[1:] {
		f, err := e.familySurrogate(w, avoid)
		if err != nil {
			return "", true, err
		}
		fam.WriteString(strings.ToLower(f))
	}
	g, f := strings.ToLower(g), fam.String()
	var joined string
	switch shape {
	case "full":
		joined = g + f
	case "initial-last":
		joined = string([]rune(g)[:1]) + f
	default:
		joined = g + string([]rune(f)[:1])
	}
	sur, err := e.vault.Assign("person.joined", text, func(i int) string {
		if i == 0 {
			return joined
		}
		return fmt.Sprintf("%s%d", joined, i)
	})
	if err != nil {
		return "", true, err
	}
	return textutil.MatchCase(text, sur), true, nil
}
