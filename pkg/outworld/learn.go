package outworld

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/detect"
	"github.com/davidparkercodes/outworld/pkg/entity"
)

var learnHonorifics = map[string]bool{"mr": true, "mrs": true, "ms": true, "mx": true, "dr": true, "prof": true, "rev": true, "sir": true, "madam": true, "miss": true}

func learnableName(sp entity.Span) (string, bool) {
	if sp.Type != entity.Person || sp.Source == "dictionary" || sp.Source == "vault" || strings.HasPrefix(sp.Source, "collision") {
		return "", false
	}
	text := sp.Text
	switch sp.Source {
	case "names:slug":
		text = strings.NewReplacer("-", " ", ".", " ", "_", " ").Replace(text)
	case "names:handle":
		if _, _, _, split := detect.SplitHandle(text); split {
			return "", false
		}
		text = strings.NewReplacer(".", " ", "_", " ").Replace(text)
	case "names:family-first":
		if sp.Canonical != "" {
			text = sp.Canonical
		}
	}
	var words []string
	for _, w := range strings.Fields(text) {
		w = strings.Trim(w, ".,;:()[]*_\"")
		if w == "" || learnHonorifics[strings.ToLower(w)] {
			continue
		}
		letters := 0
		for _, r := range w {
			switch {
			case unicode.IsLetter(r):
				letters++
			case r == '-' || r == '\'' || r == '’':
			default:
				return "", false
			}
		}
		if letters < 2 {
			continue
		}
		if textutil.IsAllUpper(w) || textutil.IsAllLower(w) {
			w = titleWord(w)
		}
		words = append(words, w)
	}
	if len(words) == 0 || len(words) > 4 {
		return "", false
	}
	if len(words) == 1 && utf8.RuneCountInString(words[0]) < 3 {
		return "", false
	}
	return strings.Join(words, " "), true
}

var mailboxWords = map[string]bool{"info": true, "support": true, "admin": true, "sales": true, "team": true, "help": true, "noreply": true, "no": true, "reply": true, "contact": true, "billing": true, "hello": true, "office": true, "mail": true, "service": true, "services": true, "dev": true, "ops": true, "security": true, "alerts": true, "notifications": true, "bot": true, "test": true, "user": true, "customer": true, "accounts": true, "jobs": true, "press": true, "legal": true, "privacy": true, "abuse": true, "postmaster": true, "webmaster": true, "root": true, "system": true, "api": true, "app": true, "data": true, "news": true, "marketing": true, "hr": true, "it": true}

func emailName(addr string) (string, bool) {
	local, _, ok := strings.Cut(addr, "@")
	if !ok {
		return "", false
	}
	if i := strings.IndexByte(local, '+'); i > 0 {
		local = local[:i]
	}
	parts := strings.FieldsFunc(local, func(r rune) bool { return r == '.' || r == '_' || r == '-' })
	if len(parts) < 2 || len(parts) > 3 {
		return "", false
	}
	known := false
	for i, p := range parts {
		if utf8.RuneCountInString(p) < 2 || mailboxWords[strings.ToLower(p)] || detect.IsTechTerm(p) || textutil.IsCommonWord(p) {
			return "", false
		}
		for _, r := range p {
			if !unicode.IsLetter(r) {
				return "", false
			}
		}
		if detect.KnownName(p) {
			known = true
		}
		parts[i] = titleWord(p)
	}
	if !known {
		return "", false
	}
	return strings.Join(parts, " "), true
}

func titleWord(w string) string {
	rs := []rune(strings.ToLower(w))
	up := true
	for i, r := range rs {
		if up && unicode.IsLetter(r) {
			rs[i] = unicode.ToUpper(r)
		}
		up = r == '-' || r == '\'' || r == '’'
	}
	return string(rs)
}

func (s *Session) learn(spans []entity.Span) []string {
	kept := s.e.keptOrgs(spans)
	var add []string
	for _, sp := range spans {
		sp = trimPossessive(sp)
		if s.e.allowed(sp) || s.e.policy.Action(entity.Person) != entity.Mask || insideKept(kept, sp.Type, sp.Start, sp.End) {
			continue
		}
		if sp.Type == entity.Email {
			if name, ok := emailName(sp.Text); ok {
				add = append(add, name)
			}
			continue
		}
		if name, ok := learnableName(sp); ok {
			add = append(add, name)
		}
	}
	s.learnNames(add)
	return add
}

func (s *Session) learnNames(add []string) {
	if len(add) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.learnedSet == nil {
		s.learnedSet = map[string]bool{}
		s.learnedDic = &detect.Dictionary{}
	}
	changed := false
	for _, name := range add {
		k := strings.ToLower(name)
		if s.learnedSet[k] {
			continue
		}
		s.learnedSet[k] = true
		s.learnedDic.Entries = append(s.learnedDic.Entries, detect.Entry{Value: name, Type: entity.Person})
		changed = true
	}
	if changed {
		s.learned = s.learnedDic.Detector()
	}
}

func (s *Session) learnedSpans(ctx context.Context, text string) []entity.Span {
	s.mu.Lock()
	d := s.learned
	s.mu.Unlock()
	if d == nil {
		return nil
	}
	spans, _ := d.Detect(ctx, text)
	var ext []entity.Span
	for i := range spans {
		spans[i].Source = "learned"
		spans[i].Priority = 29
		if len(strings.Fields(spans[i].Canonical)) >= 2 {
			continue
		}
		spans[i].Canonical = ""
		sp := spans[i]
		if end := detect.CapitalizedNameAfter(text, sp.End); end > sp.End && textutil.StartsUpper(sp.Text) && !textutil.IsAllUpper(sp.Text) {
			sp.End, sp.Text = end, text[sp.Start:end]
			ext = append(ext, sp)
		}
	}
	return append(spans, ext...)
}
