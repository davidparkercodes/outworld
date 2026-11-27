package outworld

import (
	"fmt"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/detect"
	"github.com/davidparkercodes/outworld/pkg/entity"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

var foldKinds = []string{"person.given", "person.family", "person.nick"}

type foldIndex struct {
	mu     sync.RWMutex
	n      int
	byKind map[string]map[string]vault.Record
	keys   nameKeys
}

type nameKeys struct {
	set   map[string]struct{}
	lens  []int
	max   int
	other bool
	pre2  [26 * 26]bool
	suf2  [26 * 26]bool
}

func pair2(a, b byte) int { return int(a-'a')*26 + int(b-'a') }

func (k *nameKeys) add(real string) {
	key := strings.ToLower(lettersOnly(textutil.Fold(textutil.FoldForms(real))))
	if key == "" {
		return
	}
	if !lowerASCII(key) {
		k.other = true
		return
	}
	if k.set == nil {
		k.set = map[string]struct{}{}
	}
	if _, ok := k.set[key]; ok {
		return
	}
	k.set[key] = struct{}{}
	known := false
	for _, l := range k.lens {
		if l == len(key) {
			known = true
		}
	}
	if !known {
		k.lens = append(k.lens, len(key))
	}
	k.max = max(k.max, len(key))
	if len(key) >= 2 {
		k.pre2[pair2(key[0], key[1])] = true
		k.suf2[pair2(key[len(key)-2], key[len(key)-1])] = true
	}
}

func lowerASCII(w string) bool {
	for i := 0; i < len(w); i++ {
		if w[i] < 'a' || w[i] > 'z' {
			return false
		}
	}
	return true
}

func (k *nameKeys) startsAt(f string, pos int, fn func(end int)) {
	if pos+2 > len(f) || !k.pre2[pair2(f[pos], f[pos+1])] {
		for _, l := range k.lens {
			if l == 1 && pos < len(f) {
				if _, ok := k.set[f[pos:pos+1]]; ok {
					fn(pos + 1)
				}
			}
		}
		return
	}
	for _, l := range k.lens {
		if pos+l <= len(f) {
			if _, ok := k.set[f[pos:pos+l]]; ok {
				fn(pos + l)
			}
		}
	}
}

func (e *Engine) nameKeySnapshot() (*nameKeys, func()) {
	e.refreshFolds()
	x := e.folds
	x.mu.RLock()
	return &x.keys, x.mu.RUnlock
}

func (e *Engine) hasNameKeys() bool {
	k, done := e.nameKeySnapshot()
	defer done()
	return len(k.set) > 0 || k.other
}

func (e *Engine) maybeNameKey(word string) bool {
	k, done := e.nameKeySnapshot()
	defer done()
	if k.other {
		return true
	}
	f := nameFoldKey(word)
	if !lowerASCII(f) {
		return true
	}
	_, ok := k.set[f]
	return ok
}

func (e *Engine) nameSpan(f string, edge int) (starts, ends []bool, any bool) {
	k, done := e.nameKeySnapshot()
	defer done()
	if k.other || !lowerASCII(f) {
		return nil, nil, true
	}
	starts, ends = make([]bool, len(f)+1), make([]bool, len(f)+1)
	for pos := 0; pos < len(f); pos++ {
		k.startsAt(f, pos, func(end int) {
			starts[pos], ends[end], any = true, true, true
		})
	}
	return starts, ends, any
}

func (e *Engine) nameRunPossible(f string) bool {
	k, done := e.nameKeySnapshot()
	defer done()
	if k.other || !lowerASCII(f) {
		return true
	}
	if len(k.set) == 0 {
		return false
	}
	head, tail := false, false
	for pos := 0; pos < len(f) && pos <= 9 && !head; pos++ {
		k.startsAt(f, pos, func(int) { head = true })
	}
	if !head {
		return false
	}
	for pos := max(0, len(f)-9-k.max); pos < len(f) && !tail; pos++ {
		k.startsAt(f, pos, func(end int) {
			if end >= len(f)-9 {
				tail = true
			}
		})
	}
	return tail
}

func (e *Engine) refreshFolds() {
	x := e.folds
	n := e.vault.Len()
	x.mu.RLock()
	same := n == x.n && x.byKind != nil
	x.mu.RUnlock()
	if same {
		return
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.byKind == nil || n < x.n {
		x.byKind, x.n, x.keys = map[string]map[string]vault.Record{}, 0, nameKeys{}
	}
	if n == x.n {
		return
	}
	recs := e.vault.RecordsFrom(x.n)
	for _, r := range recs {
		switch r.Kind {
		case "person.given", "person.family", "person.nick":
			x.keys.add(r.Real)
		}
		f := textutil.Fold(textutil.FoldForms(r.Real))
		for _, key := range []string{f, lettersOnly(f)} {
			if key == strings.ToLower(r.Real) || key == "" {
				continue
			}
			m := x.byKind[r.Kind]
			if m == nil {
				m = map[string]vault.Record{}
				x.byKind[r.Kind] = m
			}
			if _, ok := m[key]; !ok {
				m[key] = r
			}
		}
	}
	x.n += len(recs)
}

func (e *Engine) foldedRecord(kind, folded string) (vault.Record, bool) {
	x := e.folds
	x.mu.RLock()
	defer x.mu.RUnlock()
	r, ok := x.byKind[kind][folded]
	return r, ok
}

func (e *Engine) hasFolded() bool {
	x := e.folds
	x.mu.RLock()
	defer x.mu.RUnlock()
	for _, k := range foldKinds {
		if len(x.byKind[k]) > 0 {
			return true
		}
	}
	return false
}

func (e *Engine) nameReal(kinds []string, word string) (string, string, bool) {
	if !e.maybeNameKey(word) {
		return "", "", false
	}
	for _, k := range kinds {
		if _, ok := e.vault.Get(k, word); ok {
			return k, titleWord(word), true
		}
	}
	f := textutil.Fold(word)
	for _, k := range kinds {
		if r, ok := e.foldedRecord(k, f); ok {
			return k, r.Real, true
		}
	}
	if f != strings.ToLower(word) {
		for _, k := range kinds {
			if _, ok := e.vault.Get(k, f); ok {
				return k, titleWord(f), true
			}
		}
	}
	return "", "", false
}

func (e *Engine) foldedSurrogate(word string) (string, bool) {
	f := textutil.Fold(word)
	for _, k := range foldKinds {
		if r, ok := e.foldedRecord(k, f); ok {
			return r.Surrogate, true
		}
	}
	if f != strings.ToLower(word) {
		for _, k := range foldKinds {
			if sur, ok := e.vault.Get(k, f); ok {
				return sur, true
			}
		}
	}
	return "", false
}

func nonASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return true
		}
	}
	return false
}

func allLetters(w string) bool {
	for _, r := range w {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return w != ""
}

func markEnd(text string, end int) int {
	for end < len(text) {
		r, n := utf8.DecodeRuneInString(text[end:])
		if !unicode.Is(unicode.Mn, r) && !unicode.Is(unicode.Me, r) {
			break
		}
		end += n
	}
	return end
}

func lettersOnly(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) {
			return r
		}
		return -1
	}, s)
}

func asciiLetters(w string) bool {
	for i := 0; i < len(w); i++ {
		c := w[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
			return false
		}
	}
	return w != ""
}

func handleSep(r rune) bool { return r == '.' || r == '_' || textutil.IsHyphen(r) }

func handleSepBefore(text string, i int) int {
	r, n := utf8.DecodeLastRuneInString(text[:i])
	if !handleSep(r) {
		return 0
	}
	return n
}

func handleSepAt(text string, i int) int {
	r, n := utf8.DecodeRuneInString(text[i:])
	if !handleSep(r) {
		return 0
	}
	return n
}

func (s *Session) foldedNames(text string, ins []insertion) []finding {
	if !s.e.hasFolded() && !nonASCII(text) {
		return nil
	}
	var out []finding
	toks := textutil.Tokens(text)
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		w := text[t.Start:t.End]
		if len(w) < 3 || !allLetters(w) {
			continue
		}
		if _, ok := s.e.foldedSurrogate(w); !ok {
			continue
		}
		start, end, j := t.Start, t.End, i
		for start > 0 && j > 0 {
			n := handleSepBefore(text, start)
			if n == 0 || toks[j-1].End != start-n {
				break
			}
			start, j = toks[j-1].Start, j-1
		}
		k := i
		for k+1 < len(toks) && end < len(text) {
			n := handleSepAt(text, end)
			if n == 0 || toks[k+1].Start != end+n {
				break
			}
			end, k = toks[k+1].End, k+1
		}
		var given, family []string
		ok := true
		for _, pt := range toks[j : k+1] {
			kind, real, found := s.e.nameReal(foldKinds, text[pt.Start:pt.End])
			switch {
			case !found:
				ok = false
			case kind == "person.family":
				family = append(family, real)
			default:
				given = append(given, real)
			}
		}
		initialHandle := false
		for _, pt := range toks[j : k+1] {
			if pt.End-pt.Start == 1 && asciiLetters(text[pt.Start:pt.End]) {
				initialHandle = true
			}
		}
		if !ok || len(given) > 1 {
			start, end = t.Start, t.End
			kind, real, _ := s.e.nameReal(foldKinds, w)
			given, family = nil, nil
			if kind == "person.family" {
				family = []string{real}
			} else {
				given = []string{real}
			}
		}
		i = max(i, k)
		end = markEnd(text, end)
		if overlapsAny(ins, start, end) {
			continue
		}
		span := entity.Span{Start: start, End: end, Type: entity.Person, Label: "person", Text: text[start:end], Source: "vault:folded", Priority: 60, Score: 1}
		span.Canonical = strings.Join(append(given, family...), " ")
		if len(textutil.Tokens(span.Text)) == 1 && textutil.IsAllLower(span.Text) && !(initialHandle && !textutil.IsCommonWord(span.Text)) && !detect.LowerNameCue(text, start, end, len(family) > 0) {
			continue
		}
		if len(textutil.Tokens(span.Text)) == 1 {
			if !textutil.IsCommonWord(span.Text) && !detect.NameUseOK(text, start, end) {
				continue
			}
			if len(given) == 1 && textutil.IsCommonWord(span.Text) && detect.OrdinaryWordUse(text, start, end) {
				continue
			}
		}
		if strings.IndexFunc(span.Text, unicode.IsSpace) >= 0 {
			continue
		}
		out = append(out, finding{planned: planned{span: span}, source: "vault"})
	}
	return out
}

const minUserHandle = 4

func (s *Session) joinedUsernames(text string, ins []insertion) []finding {
	var out []finding
	for _, p := range detect.UsernamePositions(text) {
		start, end := p[0], p[1]
		w := text[start:end]
		if overlapsAny(ins, start, end) {
			continue
		}
		body := textutil.Fold(w)
		if len(body) < minUserHandle || !asciiLetters(body) {
			continue
		}
		if s.e.vault.IsReal(w) {
			if sur, ok := s.e.vault.Get("person.joined", w); ok {
				rec := vault.Record{Kind: "person.joined", Real: w, Surrogate: sur}
				sp := entity.Span{Start: start, End: end, Type: entity.Person, Label: "person", Text: w, Source: "vault", Priority: 60, Score: 1}
				out = append(out, finding{planned: planned{span: sp, rec: &rec}, source: "vault"})
			}
			continue
		}
		if f, ok := s.userFinding(w, body, start, end); ok {
			out = append(out, f)
		}
	}
	return out
}

func (s *Session) userFinding(w, body string, start, end int) (finding, bool) {
	given := []string{"person.given"}
	family := []string{"person.family"}
	for cut := 2; cut <= len(body)-2; cut++ {
		_, g, gok := s.e.nameReal(given, body[:cut])
		if !gok {
			continue
		}
		if _, f, fok := s.e.nameReal(family, body[cut:]); fok {
			sp := entity.Span{Start: start, End: end, Type: entity.Person, Label: "person", Text: w, Source: "vault:joined", Priority: 60, Score: 1, Canonical: g + " " + f}
			return finding{planned: planned{span: sp}, source: "vault"}, true
		}
	}
	var build string
	if _, f, ok := s.e.nameReal(family, body[1:]); ok && len(body) >= minUserHandle {
		if fs, ok := s.e.vault.Get("person.family", f); ok {
			build = s.e.gen.Initial(w, body[:1]) + strings.ToLower(fs)
		}
	}
	for cut := 3; build == "" && cut <= len(body)-2; cut++ {
		_, f, fok := s.e.nameReal(family, body[:cut])
		if !fok {
			continue
		}
		_, g, gok := s.e.nameReal(given, body[cut:])
		if !gok {
			continue
		}
		fs, ok1 := s.e.vault.Get("person.family", f)
		gs, ok2 := s.e.vault.Get("person.given", g)
		if ok1 && ok2 {
			build = strings.ToLower(fs) + strings.ToLower(gs)
		}
	}
	if build == "" {
		return finding{}, false
	}
	sur, err := s.e.vault.Assign("person.joined", w, func(i int) string {
		if i == 0 {
			return build
		}
		return fmt.Sprintf("%s%d", build, i)
	})
	sp := entity.Span{Start: start, End: end, Type: entity.Person, Label: "person", Text: w, Source: "vault", Priority: 60, Score: 1}
	if err != nil {
		return finding{planned: planned{span: sp}, source: "vault"}, true
	}
	rec := vault.Record{Kind: "person.joined", Real: w, Surrogate: sur}
	return finding{planned: planned{span: sp, rec: &rec}, source: "vault"}, true
}
