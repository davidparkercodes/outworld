package detect

import (
	"strings"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/namedata"
	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/entity"
)

var extraGivenSyllables = wordSet(`do so ga jung woong neul byul bit sol ram saem heung youl`)

var familyOnlySyllables = wordSet(`in ki gum`)

var englishSyllableNames = wordSet(`do young moon song soon won sang sung ah ye ha`)

func englishWordSyllable(s string) bool {
	return syllableWords[s] || englishSyllables[s] || stopCap[s] || textutil.IsCommonWord(s)
}

func englishFirstSyllable(s string) bool { return englishWordSyllable(s) && !englishSyllableNames[s] }

var koreanFoodWords = wordSet(`kimchi mushu miso soju banchan bibimbap bulgogi japchae ramyeon ramyun gochujang makgeolli tteokbokki kimbap gimbap mandu galbi sundae hoeddeok yuja omija dangmyeon jjajang jjamppong tofu`)

var koreanPlaceWords = wordSet(`busan daegu incheon gwangju daejeon ulsan suwon jeju seoul gangnam itaewon myeongdong sinchon hongdae jeonju cheonan hongkong`)

var koreanCultureWords = wordSet(`taebo kungfu taekwondo hapkido pingpong hanbok hangul hangeul kpop kdrama manhwa mukbang aegyo hallyu`)

var japaneseLoanWords = wordSet(`sushi ramen mochi sake tempura taiko kendo judo aikido karate wasabi tamago yakuza bonsai ikebana`)

var englishLookalikeWords = wordSet(`boho`)

var asianBrandWords = wordSet(`hyundai samsung daewoo kia hankook kumho ssangyong lotte pokemon nintendo toyota honda mazda subaru nissan suzuki kawasaki yamaha sony`)

func joinedWordBlocked(j string) bool {
	return koreanFoodWords[j] || koreanPlaceWords[j] || koreanCultureWords[j] || japaneseLoanWords[j] || asianBrandWords[j] || englishLookalikeWords[j] ||
		englishSyllablePairs[j] || syllablePairWords[j] || textutil.IsCommonWord(j) || namedata.IsHandleWord(j) || techTerms[j]
}

func syllableGivenIn(parts []string, familyPos bool) bool {
	if len(parts) != 2 || shapeOf(parts[0]) != shapeTitle || (shapeOf(parts[1]) != shapeLower && shapeOf(parts[1]) != shapeTitle) {
		return false
	}
	a, b := strings.ToLower(parts[0]), strings.ToLower(parts[1])
	for i, p := range []string{a, b} {
		if !givenSyllables[p] && !extraGivenSyllables[p] && !(familyPos && familyOnlySyllables[p] && shapeOf(parts[1]) == shapeLower && (i == 0 || p != a)) {
			return false
		}
	}
	if familyOnlySyllables[b] && englishFirstSyllable(a) {
		return false
	}
	return a != b && !joinedWordBlocked(a+b)
}

func syllableGivenAt(text string, toks []nameTok, j int, familyPos bool) (end, used int, ok bool) {
	if j >= len(toks) {
		return 0, 0, false
	}
	t := toks[j]
	if t.shape != shapeTitle || embedded(text, t.start, t.end) {
		return 0, 0, false
	}
	if parts := hyphenParts(t.text); len(parts) > 1 {
		return t.end, 1, syllableGivenIn(parts, familyPos)
	}
	return 0, 0, false
}

func koreanFamilyFirstAt(text string, toks []nameTok, i int) (entity.Span, int, bool) {
	t := toks[i]
	if t.shape != shapeTitle || !koreanFamilyNames[strings.ToLower(t.text)] || embedded(text, t.start, t.end) || i+1 >= len(toks) || !oneSpace(text, t.end, toks[i+1].start) {
		return entity.Span{}, 0, false
	}
	end, used, ok := syllableGivenAt(text, toks, i+1, true)
	if !ok {
		return entity.Span{}, 0, false
	}
	given := text[toks[i+1].start:end]
	return entity.Span{Start: t.start, End: end, Type: entity.Person, Label: "person", Text: text[t.start:end], Source: "names:family-first", Priority: prioFamilyFirst, Score: 1, Canonical: given + " " + t.text}, used, true
}

func syllableNameAt(text string, toks []nameTok, i int) (entity.Span, int, bool) {
	end, used, ok := syllableGivenAt(text, toks, i, false)
	if !ok {
		return entity.Span{}, 0, false
	}
	t := toks[i]
	if k := i + used; k < len(toks) {
		nx := toks[k]
		if nx.shape == shapeTitle && oneSpace(text, end, nx.start) && !embedded(text, nx.start, nx.end) && (pairSurname(nx.text) || (knownSurname(nx.text) && surnameLike(nx.text) && !thingNouns[strings.ToLower(nx.text)] && PersonActionAfter(text, nx.end))) {
			return personSpan(text, t.start, nx.end, "names:full", prioFull), used, true
		}
	}
	prev := strings.ToLower(wordBefore(text, t.start))
	if honorificWords[strings.Trim(prev, ".")] || lowerNameBefore[prev] || personDirectedBefore(text, t.start) || PersonActionAfter(text, end) {
		return personSpan(text, t.start, end, "names:first", prioFirst), used - 1, true
	}
	return entity.Span{}, 0, false
}

func koreanNameAt(text string, toks []nameTok, i int, claimed map[int]bool) (entity.Span, int, bool) {
	if claimed[i] {
		return entity.Span{}, 0, false
	}
	if sp, n, ok := koreanFamilyFirstAt(text, toks, i); ok {
		return sp, n, true
	}
	if sp, n, ok := koreanThreePartAt(text, toks, i); ok {
		return sp, n, true
	}
	if sp, n, ok := dottedNameAt(text, toks, i); ok {
		return sp, n, true
	}
	if sp, ok := greetedGivenAt(text, toks, i); ok {
		return sp, 0, true
	}
	if sp, ok := techGivenLoneAt(text, toks, i); ok {
		return sp, 0, true
	}
	return syllableNameAt(text, toks, i)
}

func koreanThreePartAt(text string, toks []nameTok, i int) (entity.Span, int, bool) {
	if i+2 >= len(toks) {
		return entity.Span{}, 0, false
	}
	t, g, f := toks[i], toks[i+1], toks[i+2]
	lw := strings.ToLower(t.text)
	if t.shape != shapeTitle || !koreanFamilyNames[lw] || textutil.IsCommonWord(t.text) || stopCap[lw] || embedded(text, t.start, t.end) {
		return entity.Span{}, 0, false
	}
	if g.shape != shapeTitle || f.shape != shapeTitle || !oneSpace(text, t.end, g.start) || !oneSpace(text, g.end, f.start) || embedded(text, f.start, f.end) {
		return entity.Span{}, 0, false
	}
	if _, ok := givenOf(g.text); !ok || !surnameLike(f.text) || capitalAfter(text, toks, i+2) || sentenceStart(text, t.start) && !PersonActionAfter(text, f.end) && !knownSurname(f.text) {
		return entity.Span{}, 0, false
	}
	return personSpan(text, t.start, f.end, "names:full", prioFull), 2, true
}

func compoundGivenPart(w string) bool {
	lw := strings.ToLower(w)
	if _, ok := namedata.LookupGiven(w); !ok || placeLike[lw] || stopCap[lw] || techTerms[lw] || IsMonthOrWeekday(w) || utf8.RuneCountInString(w) < 2 {
		return false
	}
	_, state := StateName(w)
	return !state
}

func personSpan(text string, s, e int, source string, prio int) entity.Span {
	return entity.Span{Start: s, End: e, Type: entity.Person, Label: "person", Text: text[s:e], Source: source, Priority: prio, Score: 1}
}

func pairSurname(w string) bool {
	lw := strings.ToLower(w)
	if koreanFamilyNames[lw] {
		return true
	}
	return strongSurname(w) && surnameLike(w) && !thingNouns[lw]
}

func dashParticleAfter(text, word string, e int) bool {
	for _, r := range textutil.RangeOrNameDashes {
		d := string(r)
		if !strings.HasPrefix(text[e:], d) {
			continue
		}
		k := e + len(d)
		j := k
		for j < len(text) && j-k < maxDashGap && text[j] >= 'a' && text[j] <= 'z' {
			j++
		}
		if j == k {
			return false
		}
		next := text[k:j]
		return englishParticles[next] || joinedWordBlocked(strings.ToLower(word)+next)
	}
	return false
}

func nameTokens(text string) []nameTok {
	var raw []nameTok
	for _, m := range wordRE.FindAllStringIndex(text, -1) {
		w := text[m[0]:m[1]]
		raw = append(raw, nameTok{m[0], m[1], w, shapeOf(w)})
	}
	out := raw[:0:0]
	for i := 0; i < len(raw); i++ {
		if i+1 < len(raw) && joinsAsName(text, raw, i) {
			t := raw[i]
			w := text[t.start:raw[i+1].end]
			out = append(out, nameTok{t.start, raw[i+1].end, w, shapeOf(w)})
			i++
			continue
		}
		out = append(out, raw[i])
	}
	return out
}

func joinsAsName(text string, toks []nameTok, i int) bool {
	t, nx := toks[i], toks[i+1]
	r, n := utf8.DecodeRuneInString(text[t.end:])
	if !textutil.IsRangeOrNameDash(r) || nx.start != t.end+n || t.shape != shapeTitle {
		return false
	}
	if nx.shape == shapeLower {
		return koreanSyllable(t.text) && koreanSyllable(nx.text)
	}
	if nx.shape != shapeTitle || !compoundGivenPart(t.text) {
		return false
	}
	if i+2 < len(toks) {
		f := toks[i+2]
		if f.shape == shapeTitle && oneSpace(text, nx.end, f.start) && (surnameLike(f.text) || techSurname(f.text)) {
			return true
		}
	}
	prev := strings.ToLower(wordBefore(text, t.start))
	return PersonActionAfter(text, nx.end) || personDirectedBefore(text, t.start) || lowerNameBefore[prev] || honorificWords[strings.Trim(prev, ".")]
}

func koreanSyllable(w string) bool {
	lw := strings.ToLower(w)
	return givenSyllables[lw] || extraGivenSyllables[lw] || familyOnlySyllables[lw]
}
