package outworld

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/detect"
	"github.com/davidparkercodes/outworld/pkg/entity"
	"github.com/davidparkercodes/outworld/pkg/surrogate"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

type nameRole int

const (
	roleGiven nameRole = iota
	roleFamily
	roleNick
)

func (e *Engine) legacyPersonWord(w string) bool {
	for _, k := range nameKinds {
		if e.vault.Legacy(k, w) {
			return true
		}
	}
	return false
}

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
	orig := sp.Text
	var forms *formMap
	if textutil.NeedsFormFold(orig) {
		folded, starts, ends := textutil.FoldFormsMap(orig)
		forms = &formMap{orig: orig, starts: starts, ends: ends}
		sp.Text = folded
	}
	sp.Canonical = detect.FoldForms(sp.Canonical)
	text := sp.Text
	toks := textutil.Tokens(text)
	if len(toks) == 0 {
		return text, nil
	}
	if sp.Source == collisionInitials {
		return e.initialismCollisionSurrogate(text, avoid)
	}
	if sur, ok, err := e.initialsSurrogate(sp, avoid); ok || err != nil {
		return sur, err
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
	if len(toks) == 1 && len(canon) == 0 {
		if sur, ok, err := e.handleSurrogate(text, sp.Source, avoid); ok || err != nil {
			return sur, err
		}
	}
	east := eastPartsOf(sp)
	initials := handleLike(text, toks, sp, canon)
	if initials {
		if sur, ok := e.vault.Get("person.joined", text); ok {
			return sur, nil
		}
		initials = !e.legacyHandle(text, toks, canon)
	}
	var b strings.Builder
	last := 0
	foldedOnly := false
	preFamily := map[int]bool{}
	for k, t := range toks {
		if _, ok := e.vault.Get("person.family", text[t.Start:t.End]); ok {
			preFamily[k] = true
		}
	}
	for i, t := range toks {
		word := text[t.Start:t.End]
		b.WriteString(forms.slice(text, last, t.Start))
		last = t.End
		if utf8.RuneCountInString(word) == 1 {
			if initials && unicode.IsLetter([]rune(word)[0]) {
				letter, err := e.handleInitial(text, word, canon, gender, avoid)
				if err != nil {
					return "", err
				}
				word = textutil.MatchCase(word, letter)
			}
			b.WriteString(word)
			continue
		}
		if east == nil && isParticleWord(word) && !e.legacyPersonWord(word) && e.particleLiteral(text, toks, i, canon, preFamily) {
			b.WriteString(word)
			continue
		}
		var p string
		var err error
		real := realCasing(word, canon)
		if !strings.EqualFold(word, real) && sameName(word, real) {
			foldedOnly = true
		}
		role := e.role(word, i, len(toks), canon, sp, hyphenBefore(text, t.Start))
		lead := len(canon) > 0 && sameName(word, canon[0])
		switch {
		case east != nil && role == roleFamily:
			p, err = e.eastFamilySurrogate(real, east.origin, avoid)
		case east != nil && role == roleGiven && lead && e.vault.Legacy("person.given", real):
			p, _ = e.vault.Get("person.given", real)
		case east != nil && role == roleGiven && !lead && e.vault.Legacy("person.family", real):
			p, _ = e.vault.Get("person.family", real)
		case east != nil && role == roleGiven && east.marker != "" && sameName(word, east.marker):
			p = word
		case east != nil && role == roleGiven:
			p, err = e.eastGivenSurrogate(real, east.origin, avoid)
		case role == roleFamily:
			p, err = e.familySurrogate(real, avoid)
		case role == roleNick:
			p, err = e.nickSurrogate(real, canon[0], gender, avoid)
		default:
			g := gender
			if len(canon) == 0 || !sameName(word, canon[0]) {
				g = surrogate.GenderOf(word)
			}
			p, err = e.givenSurrogate(real, g, avoid)
		}
		if err != nil {
			return "", err
		}
		b.WriteString(textutil.MatchCase(word, p))
	}
	b.WriteString(forms.slice(text, last, len(text)))
	if initials || (foldedOnly && len(toks) >= 2 && strings.IndexFunc(text, unicode.IsSpace) < 0) {
		built := b.String()
		return e.vault.Assign("person.joined", text, func(i int) string {
			if i == 0 {
				return built
			}
			return fmt.Sprintf("%s%d", built, i)
		})
	}
	return b.String(), nil
}

type eastParts struct {
	origin surrogate.Origin
	marker string
}

func eastPartsOf(sp entity.Span) *eastParts {
	if sp.Source != "names:family-first" && sp.Source != learnedFamilyFirst {
		return nil
	}
	fields := strings.Fields(sp.Canonical)
	if len(fields) < 2 {
		return nil
	}
	given := fields[:len(fields)-1]
	parts := &eastParts{origin: surrogate.EastOrigin(fields[len(fields)-1], given)}
	if len(given) >= 2 && surrogate.IsEastMarker(given[0]) {
		parts.marker = given[0]
	}
	return parts
}

func handleLike(text string, toks []textutil.Token, sp entity.Span, canon []string) bool {
	if len(toks) < 2 || strings.ContainsAny(text, " \t\n") {
		return false
	}
	if sp.Source != "names:handle" && !(len(canon) >= 2 && textutil.IsAllLower(text)) {
		return false
	}
	for _, t := range toks {
		if t.End-t.Start == 1 && unicode.IsLetter(rune(text[t.Start])) {
			return true
		}
	}
	return false
}

func (e *Engine) handleInitial(handle, letter string, canon []string, gender string, avoid func(string) bool) (string, error) {
	if len(canon) >= 2 && strings.HasPrefix(textutil.Fold(canon[0]), textutil.Fold(letter)) {
		g, err := e.givenSurrogate(canon[0], gender, avoid)
		if err != nil {
			return "", err
		}
		return strings.ToLower(string([]rune(g)[:1])), nil
	}
	return e.gen.Initial(handle, letter), nil
}

func (e *Engine) legacyHandle(text string, toks []textutil.Token, canon []string) bool {
	found := false
	for _, t := range toks {
		word := text[t.Start:t.End]
		if utf8.RuneCountInString(word) == 1 {
			continue
		}
		real := realCasing(word, canon)
		if !e.vault.Legacy("person.family", real) && !e.vault.Legacy("person.given", real) {
			return false
		}
		found = true
	}
	return found
}

var nameKinds = []string{"person.family", "person.given", residueKind}

func (e *Engine) assignName(kind, word string, gen func(int) string) (string, error) {
	if s, ok := e.vault.Get(kind, word); ok {
		return s, nil
	}
	if asciiLetters(word) && len(word) >= 3 {
		e.refreshFolds()
		if r, ok := e.foldedRecord(kind, strings.ToLower(word)); ok {
			return r.Surrogate, nil
		}
	}
	for _, k := range nameKinds {
		if k == kind {
			continue
		}
		known, ok := e.vault.Get(k, word)
		if !ok {
			continue
		}
		if k != residueKind {
			return known, nil
		}
		return e.vault.Assign(kind, word, func(i int) string {
			if i == 0 {
				return known
			}
			return gen(i - 1)
		})
	}
	return e.vault.Assign(kind, word, gen)
}

func (e *Engine) eastFamilySurrogate(word string, origin surrogate.Origin, avoid func(string) bool) (string, error) {
	return e.assignName("person.family", word, func(i int) string {
		return avoided(e.gen.EastFamilyName(origin, word, i), avoid)
	})
}

func (e *Engine) eastGivenSurrogate(word string, origin surrogate.Origin, avoid func(string) bool) (string, error) {
	syllables := detect.EastSyllables(word)
	return e.assignName("person.given", word, func(i int) string {
		return avoided(e.gen.EastGivenName(origin, word, syllables, i), avoid)
	})
}

func sameName(a, b string) bool {
	return strings.EqualFold(a, b) || textutil.Fold(a) == textutil.Fold(b)
}

func (e *Engine) heldPersonWord(w string) bool {
	_, ok := e.heldPersonRecord(w)
	return ok
}

func (e *Engine) heldPersonRecord(w string) (vault.Record, bool) {
	w = strings.TrimSuffix(w, ".")
	for _, k := range []string{"person.given", "person.nick", "person.family"} {
		if sur, ok := e.vault.Get(k, w); ok {
			if r, ok := e.vault.Reverse(sur); ok && r.Kind == k {
				return r, true
			}
			return vault.Record{Kind: k, Real: w, Surrogate: sur}, true
		}
	}
	return vault.Record{}, false
}

func isParticleWord(w string) bool { return detect.IsNameParticle(w) || particleCompound(w) }

func (e *Engine) particleLiteral(text string, toks []textutil.Token, i int, canon []string, preFamily map[int]bool) bool {
	trail := len(toks)
	for trail > 0 && isParticleWord(text[toks[trail-1].Start:toks[trail-1].End]) {
		trail--
	}
	word := text[toks[i].Start:toks[i].End]
	lead := 0
	for lead < len(toks) && isParticleWord(text[toks[lead].Start:toks[lead].End]) {
		lead++
	}
	if (i < lead || i >= trail) && e.heldPersonWord(word) {
		return false
	}
	if i == len(toks)-1 && len(canon) > 1 && strings.EqualFold(canon[len(canon)-1], strings.TrimSuffix(word, ".")) {
		return false
	}
	if trail == 0 && len(toks) > 1 {
		return true
	}
	if i < trail {
		return i > 0 || textutil.IsAllLower(word) || innerParticle(word, canon) || joinerThenCapital(text, toks[i].End) || nextParticle(text, toks, i)
	}
	if i == len(toks)-1 && detect.StreetAbbrev(word) {
		return true
	}
	if trail == 0 || strings.ContainsFunc(text[toks[trail-1].End:toks[trail].Start], unicode.IsSpace) {
		return false
	}
	for k := range toks[:trail] {
		if preFamily[k] {
			return true
		}
	}
	return false
}

func nextParticle(text string, toks []textutil.Token, i int) bool {
	if i+1 >= len(toks) {
		return false
	}
	w := text[toks[i+1].Start:toks[i+1].End]
	return detect.IsNameParticle(w) || particleCompound(w)
}

func joinerThenCapital(text string, at int) bool {
	if at+1 >= len(text) || !strings.ContainsRune("-._", rune(text[at])) {
		return false
	}
	r, _ := utf8.DecodeRuneInString(text[at+1:])
	return unicode.IsUpper(r)
}

func particleCompound(w string) bool {
	l := strings.Map(func(r rune) rune {
		if textutil.IsHyphen(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, w)
	if l == "" {
		return false
	}
	for _, p := range leadParticles {
		if len(p) < len(l) && strings.HasPrefix(l, p) && (detect.IsNameParticle(l[len(p):]) || particleCompound(l[len(p):])) {
			return true
		}
	}
	return false
}

func innerParticle(word string, canon []string) bool {
	for k, c := range canon {
		if k > 0 && strings.EqualFold(c, word) {
			return true
		}
	}
	return false
}

func realCasing(word string, canon []string) string {
	for _, c := range canon {
		if sameName(word, c) {
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

func (e *Engine) role(word string, i, n int, canon []string, sp entity.Span, hyphenated bool) nameRole {
	if len(canon) >= 2 && sp.Source == "names:family-first" {
		if i == 0 {
			return roleFamily
		}
		return roleGiven
	}
	if len(canon) >= 2 && sp.Source == learnedFamilyFirst {
		if sameName(word, canon[len(canon)-1]) {
			return roleFamily
		}
		for _, c := range canon[:len(canon)-1] {
			if sameName(word, c) {
				return roleGiven
			}
		}
	}
	if len(canon) >= 2 {
		if sameName(word, canon[0]) {
			return roleGiven
		}
		for _, c := range canon[1:] {
			if sameName(word, c) {
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
		if i == 0 && detect.IsNameParticle(word) {
			if _, isGiven := e.vault.Get("person.given", word); isGiven {
				return roleGiven
			}
			return roleFamily
		}
		if i == 0 {
			return roleGiven
		}
		if hyphenated {
			_, isGiven := e.vault.Get("person.given", word)
			_, isFamily := e.vault.Get("person.family", word)
			if isGiven && !isFamily {
				return roleGiven
			}
		}
		return roleFamily
	}
	if strings.HasPrefix(sp.Source, "regex:honorific") || sp.Source == "names:honorific" || sp.Source == "collision:family" || sp.Source == "names:nick-surname" {
		return roleFamily
	}
	_, isGiven := e.vault.Get("person.given", word)
	_, isFamily := e.vault.Get("person.family", word)
	if isFamily && !isGiven {
		return roleFamily
	}
	return roleGiven
}

type formMap struct {
	orig         string
	starts, ends []int
}

func (m *formMap) origAt(text string, i int) int {
	if i >= len(text) {
		return len(m.orig)
	}
	return m.starts[i]
}

func (m *formMap) slice(text string, from, to int) string {
	if m == nil {
		return text[from:to]
	}
	if from >= to && from > 0 && to < len(text) {
		return ""
	}
	a := 0
	if from > 0 {
		a = m.ends[from-1]
	}
	return m.orig[a:m.origAt(text, to)]
}

func (e *Engine) givenSurrogate(word, gender string, avoid func(string) bool) (string, error) {
	return e.assignName("person.given", word, func(i int) string {
		return notContaining(avoided(e.gen.GivenNameFor(word, gender, i), avoid), word)
	})
}

func (e *Engine) familySurrogate(word string, avoid func(string) bool) (string, error) {
	return e.assignName("person.family", word, func(i int) string {
		return notContaining(avoided(e.gen.FamilyName(word, i), avoid), word)
	})
}

func notContaining(cand, real string) string {
	c := textutil.Fold(cand)
	for w := real; ; {
		r := textutil.Fold(w)
		if len(r) < 3 {
			break
		}
		if strings.Contains(c, r) {
			return ""
		}
		rest, ok := particlePrefixCut(w)
		if !ok {
			break
		}
		w = rest
	}
	ck := nameRootKey(cand)
	parts := strings.FieldsFunc(real, func(r rune) bool {
		return textutil.IsHyphen(r) || unicode.IsSpace(r) || isApostrophe(r) || r == '\u2018' || r == '`'
	})
	for _, base := range append([]string{real}, parts...) {
		for r := nameRootKey(base); len(r) >= 5; {
			if strings.Contains(ck, r) {
				return ""
			}
			cut := 0
			for _, p := range append(leadParticles, "o", "d", "l") {
				if strings.HasPrefix(r, p) && len(r)-len(p) >= 5 {
					cut = len(p)
					break
				}
			}
			if cut == 0 {
				break
			}
			r = r[cut:]
		}
	}
	return cand
}

func nameRootKey(w string) string {
	return strings.ToLower(lettersOnly(textutil.Fold(textutil.FoldForms(w))))
}

func particlePrefixCut(w string) (string, bool) {
	for _, p := range leadParticles {
		if len(w) > len(p) && strings.EqualFold(w[:len(p)], p) {
			next, _ := utf8.DecodeRuneInString(w[len(p):])
			if unicode.IsUpper(next) || len(p) >= 3 && len(w)-len(p) >= 4 {
				return w[len(p):], true
			}
		}
	}
	return "", false
}

func (e *Engine) nickSurrogate(word, first, gender string, avoid func(string) bool) (string, error) {
	g, err := e.givenSurrogate(first, gender, avoid)
	if err != nil {
		return "", err
	}
	return e.vault.Assign("person.nick", word, func(i int) string {
		return avoided(surrogate.Nickname(textutil.Fold(word), textutil.Fold(first), g, i), avoid)
	})
}

func avoided(cand string, avoid func(string) bool) string {
	if avoid != nil && avoid(cand) {
		return ""
	}
	return cand
}

func (e *Engine) joinedSurrogate(text string, canon []string, gender string, avoid func(string) bool) (string, bool, error) {
	strip := strings.NewReplacer("'", "", "\u2019", "", "-", "")
	lw := strings.NewReplacer("'", "", "\u2019", "").Replace(textutil.Fold(text))
	first := strip.Replace(textutil.Fold(canon[0]))
	var restWords []string
	for _, w := range canon[1:] {
		restWords = append(restWords, strip.Replace(textutil.Fold(w)))
	}
	rest := strings.Join(restWords, "")
	if lw == first || strings.Contains(" "+strings.Join(restWords, " ")+" ", " "+lw+" ") {
		return "", false, nil
	}
	firstInitial, restInitial := string([]rune(first)[:1]), string([]rune(rest)[:1])
	gap := ""
	if g := strings.TrimSuffix(strings.TrimPrefix(lw, first), rest); len(lw) == len(first)+len(g)+len(rest) && (g == "o" || g == "d" || g == "l") && strings.HasPrefix(lw, first) && strings.HasSuffix(lw, rest) {
		gap = g
	}
	var shape string
	switch lw {
	case first + gap + rest:
		shape = "full"
	case firstInitial + rest:
		shape = "initial-last"
	case first + restInitial:
		shape = "first-initial"
	case rest + first:
		shape = "reversed"
	default:
		return "", false, nil
	}
	g, err := e.givenSurrogate(canon[0], gender, avoid)
	if err != nil {
		return "", true, err
	}
	var fam strings.Builder
	for _, cw := range canon[1:] {
		for _, t := range textutil.Tokens(cw) {
			w := cw[t.Start:t.End]
			if utf8.RuneCountInString(w) == 1 || ((detect.IsNameParticle(w) || particleCompound(w)) && !e.legacyPersonWord(w)) {
				fam.WriteString(strings.ToLower(textutil.Fold(w)))
				continue
			}
			f, err := e.familySurrogate(w, avoid)
			if err != nil {
				return "", true, err
			}
			fam.WriteString(strings.ToLower(f))
		}
	}
	g, f := strings.ToLower(g), fam.String()
	var joined string
	switch shape {
	case "full":
		joined = g + gap + f
	case "initial-last":
		joined = string([]rune(g)[:1]) + f
	case "reversed":
		joined = f + g
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
	if (shape == "full" || shape == "reversed") && sur == joined && !textutil.StartsUpper(text) && strings.IndexFunc(text, unicode.IsUpper) >= 0 {
		a, b, k := g, f, utf8.RuneCountInString(first)
		if shape == "reversed" {
			a, b, k = f, g, utf8.RuneCountInString(rest)
		}
		if rs := []rune(text); k > 0 && k < len(rs) {
			return caseLike(string(rs[:k]), a) + caseLike(string(rs[k:]), b), true, nil
		}
	}
	return textutil.MatchCase(text, sur), true, nil
}

func (e *Engine) handleSurrogate(text, source string, avoid func(string) bool) (string, bool, error) {
	if sur, ok := e.vault.Get("person.joined", text); ok {
		return textutil.MatchCase(text, sur), true, nil
	}
	if source != "names:handle" {
		return "", false, nil
	}
	given, family, digits, ok := detect.SplitHandle(text)
	if !ok {
		first, initial, digits, ok := detect.SplitNameInitial(text)
		if !ok {
			return "", false, nil
		}
		g, err := e.givenSurrogate(titleWord(first), surrogate.GenderOf(first), avoid)
		if err != nil {
			return "", true, err
		}
		joined := strings.ToLower(g) + e.gen.Initial(text, initial)
		sur, err := e.vault.Assign("person.joined", text, func(i int) string {
			if i == 0 {
				return joined + digits
			}
			return fmt.Sprintf("%s%d%s", joined, i, digits)
		})
		if err != nil {
			return "", true, err
		}
		return textutil.MatchCase(text, sur), true, nil
	}
	f, err := e.familySurrogate(titleWord(family), avoid)
	if err != nil {
		return "", true, err
	}
	prefix := e.gen.Initial(text, given)
	if utf8.RuneCountInString(given) > 1 {
		g, err := e.givenSurrogate(titleWord(given), surrogate.GenderOf(given), avoid)
		if err != nil {
			return "", true, err
		}
		prefix = strings.ToLower(g)
	}
	joined := prefix + strings.ToLower(f)
	sur, err := e.vault.Assign("person.joined", text, func(i int) string {
		if i == 0 {
			return joined + digits
		}
		return fmt.Sprintf("%s%d%s", joined, i, digits)
	})
	if err != nil {
		return "", true, err
	}
	return textutil.MatchCase(text, sur), true, nil
}

func (e *Engine) initialsSurrogate(sp entity.Span, avoid func(string) bool) (string, bool, error) {
	text := sp.Text
	if sp.Canonical == "" || !detect.IsInitialismOf(text, sp.Canonical) {
		return "", false, nil
	}
	if sur, ok := e.vault.Get("person.joined", text); ok {
		return textutil.MatchCase(text, sur), true, nil
	}
	var words []string
	for _, t := range textutil.Tokens(sp.Canonical) {
		words = append(words, sp.Canonical[t.Start:t.End])
	}
	if len(words) != utf8.RuneCountInString(text) {
		var core []string
		for _, w := range words {
			if !detect.IsNameParticle(w) {
				core = append(core, w)
			}
		}
		words = core
	}
	var letters []rune
	for i, w := range words {
		r := []rune(strings.ToUpper(w))[0]
		if utf8.RuneCountInString(w) > 1 {
			kinds := []string{"person.given", "person.family"}
			if i > 0 {
				kinds[0], kinds[1] = kinds[1], kinds[0]
			}
			found := false
			for _, k := range kinds {
				if s, ok := e.vault.Get(k, w); ok && s != "" {
					r, found = []rune(strings.ToUpper(textutil.Fold(s)))[0], true
					break
				}
			}
			if !found {
				r = []rune(strings.ToUpper(e.gen.Initial(sp.Canonical+"|"+w, string(r))))[0]
			}
		}
		letters = append(letters, r)
	}
	sur, err := e.vault.Assign("person.joined", text, func(i int) string {
		cand := append([]rune(nil), letters...)
		for pos := range cand {
			if i > 0 && (utf8.RuneCountInString(words[pos]) > 1 || i > 60) {
				cand[pos] = []rune(strings.ToUpper(e.gen.Initial(fmt.Sprintf("%s|%d|%d", sp.Canonical, i, pos), "")))[0]
			}
		}
		c := string(cand)
		if strings.EqualFold(c, text) || detect.CommonInitialism(c) || (avoid != nil && avoid(c)) {
			return ""
		}
		return c
	})
	if err != nil {
		return "", true, err
	}
	return textutil.MatchCase(text, sur), true, nil
}

const collisionInitials = "collision:initials"

func isInitialism(kind, surrogate string) bool {
	if kind != "person.joined" {
		return false
	}
	n := 0
	for _, r := range surrogate {
		if !unicode.IsUpper(r) {
			return false
		}
		n++
	}
	return n >= 2 && n <= 3
}

func (e *Engine) initialismCollisionSurrogate(text string, avoid func(string) bool) (string, error) {
	n := utf8.RuneCountInString(text)
	return e.vault.Assign("person.joined", text, func(i int) string {
		var b strings.Builder
		for pos := 0; pos < n; pos++ {
			b.WriteString(strings.ToUpper(e.gen.Initial(fmt.Sprintf("%s|collision|%d|%d", text, i, pos), "")))
		}
		c := b.String()
		if c == text || detect.CommonInitialism(c) || (avoid != nil && avoid(c)) {
			return ""
		}
		return c
	})
}

func hyphenBefore(text string, i int) bool {
	if i <= 0 {
		return false
	}
	r, _ := utf8.DecodeLastRuneInString(text[:i])
	return textutil.IsHyphen(r)
}
