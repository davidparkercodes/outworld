package namedata

import (
	"sort"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/textutil"
)

func TestPhrasesAreFoldedSortedAndMultiWord(t *testing.T) {
	lines := strings.Split(strings.TrimSpace(phrasesRaw), "\n")
	if !sort.StringsAreSorted(lines) {
		t.Error("phrases.txt is not in byte order")
	}
	for _, l := range lines {
		phrase, personal := strings.CutSuffix(l, " *")
		if phrase != textutil.Fold(phrase) || strings.Contains(phrase, "*") || len(strings.Fields(phrase)) < 2 || strings.Join(strings.Fields(phrase), " ") != phrase {
			t.Errorf("bad phrase line %q", l)
		}
		if !IsPhrase(phrase) || PhraseWords(strings.Fields(phrase)[0]) < len(strings.Fields(phrase)) {
			t.Errorf("phrase %q not loaded", phrase)
		}
		if PersonalPhrase(phrase) != personal {
			t.Errorf("phrase %q has the wrong personal flag", phrase)
		}
	}
	if IsPhrase("florence nightingale") || IsPhrase("paris hilton") {
		t.Error("a person is listed as a phrase")
	}
}

func TestPersonShapedPhrasesAreFlaggedOrAbsent(t *testing.T) {
	for _, p := range []string{"jane doe", "john doe", "jane roe", "lou gehrig"} {
		if IsPhrase(p) {
			t.Errorf("%q is an ordinary personal name and must not be a phrase", p)
		}
	}
	for _, p := range []string{"tom collins", "jack russell", "glen allen", "glen ellen", "mary sue", "george mason", "vera cruz", "alice springs", "carol stream", "john hancock", "mason dixon"} {
		if !PersonalPhrase(p) {
			t.Errorf("%q must carry the personal flag", p)
		}
	}
	for _, p := range []string{"monte carlo", "hong kong", "turing test", "markov chain", "kansas city"} {
		if !IsPhrase(p) || PersonalPhrase(p) {
			t.Errorf("%q must be a plain phrase", p)
		}
	}
}
