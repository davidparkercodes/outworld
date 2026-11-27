package detect

import (
	"sort"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/entity"
)

func propertySeparators() []string {
	var out []string
	for _, r := range textutil.PhoneSeparators() {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		out = append(out, string(r))
	}
	return out
}

func prefixPhoneCues() []string {
	var out []string
	for w := range phoneLabelWords {
		if suffixOnlyCues[w] {
			w += " number"
		}
		out = append(out, w)
	}
	sort.Strings(out)
	return out
}

func permutations(items []string) [][]string {
	if len(items) <= 1 {
		return [][]string{append([]string(nil), items...)}
	}
	var out [][]string
	for i := range items {
		rest := append(append([]string(nil), items[:i]...), items[i+1:]...)
		for _, p := range permutations(rest) {
			out = append(out, append([]string{items[i]}, p...))
		}
	}
	return out
}

var scanOnly bool

func maskedValues(t *testing.T, text string, typ entity.Type) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	if scanOnly {
		lower := strings.ToLower(text)
		var spans []entity.Span
		if typ == entity.Phone {
			spans = scanSepPhone(text, lower)
		} else {
			spans = scanSepSSN(text, lower)
		}
		for _, sp := range spans {
			out[text[sp.Start:sp.End]] = true
		}
		return out
	}
	for _, sp := range regexSpans(t, text) {
		if sp.Type == typ {
			out[sp.Text] = true
		}
	}
	return out
}

var aloneCache = map[string]bool{}

func aloneKey(typ entity.Type, line string) string {
	if scanOnly {
		return "scan\x00" + string(typ) + "\x00" + line
	}
	return string(typ) + "\x00" + line
}

func masksAlone(t *testing.T, line, value string, typ entity.Type) bool {
	t.Helper()
	key := aloneKey(typ, line)
	if v, ok := aloneCache[key]; ok {
		return v
	}
	v := maskedValues(t, line, typ)[value]
	aloneCache[key] = v
	return v
}

func checkCombined(t *testing.T, label string, items []struct{ cue, value string }, typs []entity.Type, joiner string) bool {
	t.Helper()
	parts := make([]string, len(items))
	for k, it := range items {
		parts[k] = it.cue + " " + it.value
	}
	line := strings.Join(parts, joiner)
	ok := true
	for k, it := range items {
		alone := masksAlone(t, it.cue+" "+it.value, it.value, typs[k])
		if alone && !maskedValues(t, line, typs[k])[it.value] {
			t.Errorf("%s: %q masks alone but not in %q", label, it.value, line)
			ok = false
		}
	}
	return ok
}

func TestCuedPhonesAllMaskOnOneLine(t *testing.T) {
	scanOnly = true
	defer func() { scanOnly = false }()
	numbers := []string{"804~555~1234", "703~555~2468", "202~555~0147", "415~555~0199"}
	cues := prefixPhoneCues()
	seps := propertySeparators()
	lines, failures := 0, 0
	for _, sep := range seps {
		for n := 1; n <= 4; n++ {
			step := 1
			for start := 0; start < len(cues); start += step {
				tuple := make([]string, n)
				for k := range tuple {
					tuple[k] = cues[(start+k*5)%len(cues)]
				}
				for _, order := range permutations(tuple) {
					items := make([]struct{ cue, value string }, n)
					typs := make([]entity.Type, n)
					for k, cue := range order {
						items[k] = struct{ cue, value string }{cue, strings.ReplaceAll(numbers[k], "~", sep)}
						typs[k] = entity.Phone
					}
					lines++
					if !checkCombined(t, "phones", items, typs, " and ") {
						failures++
						if failures > 20 {
							t.Fatalf("too many failures")
						}
					}
				}
			}
		}
	}
	t.Logf("%d cues, %d separators, %d combined lines, %d failures", len(cues), len(seps), lines, failures)
}

func TestCuedSSNsAndMixedOnOneLine(t *testing.T) {
	ssns := []string{"536~22~8471", "219~09~9999"}
	var cues []string
	for _, c := range ssnCueWords {
		if c != "tin" {
			cues = append(cues, c)
		}
	}
	phoneCues := prefixPhoneCues()
	for _, sep := range propertySeparators() {
		w := func(s string) string { return strings.ReplaceAll(s, "~", sep) }
		for a := range cues {
			b := (a + 3) % len(cues)
			for _, order := range [][]int{{a, b}, {b, a}} {
				items := []struct{ cue, value string }{{cues[order[0]], w(ssns[0])}, {cues[order[1]], w(ssns[1])}}
				checkCombined(t, "ssns", items, []entity.Type{entity.GovID, entity.GovID}, ", ")
			}
			p := phoneCues[a%len(phoneCues)]
			checkCombined(t, "mixed", []struct{ cue, value string }{{p, w("804~555~1234")}, {cues[a], w(ssns[0])}}, []entity.Type{entity.Phone, entity.GovID}, " and ")
			checkCombined(t, "mixed", []struct{ cue, value string }{{cues[a], w(ssns[0])}, {p, w("804~555~1234")}}, []entity.Type{entity.GovID, entity.Phone}, " and ")
		}
	}
}

func TestCueBeforePlusOrBracketRetriesTheDigits(t *testing.T) {
	for _, sep := range []string{"~", "〜", "/", "・"} {
		w := func(s string) string { return strings.ReplaceAll(s, "~", sep) }
		for _, c := range []struct{ text, want string }{
			{"Call+804~555~1234", "804~555~1234"},
			{"Call +804~555~1234", "804~555~1234"},
			{"call: +804~555~1234", "804~555~1234"},
			{"Call(804~555~1234", "804~555~1234"},
			{"Call (804~555~1234)", "804~555~1234"},
			{"phone (804~555~1234) today", "804~555~1234"},
		} {
			text, want := w(c.text), w(c.want)
			if got := spanOf(t, text, entity.Phone); got != want {
				t.Errorf("%q: phone %q, want %q", text, got, want)
			}
		}
	}
	for _, text := range []string{"call(804, 555, 1234)", "fn (1+804-555)", "call (804) 55", "call +80~45~55"} {
		if got := spanOf(t, text, entity.Phone); got != "" {
			t.Errorf("%q: unexpected phone %q", text, got)
		}
	}
}

func TestPhCueAndDotGluedLabels(t *testing.T) {
	for _, c := range []struct{ text, want string }{
		{"ph 804/555/1234", "804/555/1234"},
		{"Ph 804/555/1234", "804/555/1234"},
		{"ph 804~555~1234", "804~555~1234"},
		{"PH 804\u301c555\u301c1234", "804\u301c555\u301c1234"},
		{"pH 804/555/1234", "804/555/1234"},
		{"ph: 804/555/1234", "804/555/1234"},
		{"ph. 804~555~1234", "804~555~1234"},
		{"Tel.090-1234-5678", "090-1234-5678"},
		{"tel.804-555-1234", "804-555-1234"},
		{"Ph.804~555~1234", "804~555~1234"},
		{"phone.804-555-1234", "804-555-1234"},
		{"TEL.804/555/1234", "804/555/1234"},
		{"ph 804~555~1234", "804~555~1234"},
	} {
		if got := spanOf(t, c.text, entity.Phone); got != c.want {
			t.Errorf("%q: phone %q, want %q", c.text, got, c.want)
		}
	}
	line := "ph 804/555/1234 fax 703/555/2468"
	got := maskedValues(t, line, entity.Phone)
	if !got["804/555/1234"] || !got["703/555/2468"] {
		t.Errorf("%q: masked %v, want both numbers", line, got)
	}
	for _, text := range []string{"pH 7.4", "ph 6.5/7.0", "pH 7", "ph 12/14", "PH 3~4", "pH 6/7 and 7/8", "ph 7.4 buffer", "example.tel.8045551234.com", "host.804-555-1234.example.com", "v.090-1234", "graph.804~555~1234"} {
		if got := spanOf(t, text, entity.Phone); got != "" {
			t.Errorf("%q: unexpected phone %q", text, got)
		}
	}
}

func TestContinuationWordsAfterACuedPhone(t *testing.T) {
	scanOnly = true
	defer func() { scanOnly = false }()
	numbers := []string{"804~555~1234", "703~555~2468", "202~555~0147", "415~555~0199"}
	words := []string{"work", "home", "office", "direct", "main", "desk", "cell", "fax", "mobile"}
	joiners := []string{" and ", ", ", "; ", " / ", " or ", ","}
	cues := prefixPhoneCues()
	lines := 0
	for _, sep := range propertySeparators() {
		w := func(s string) string { return strings.ReplaceAll(s, "~", sep) }
		for ci, cue := range cues {
			tuple := []string{words[ci%len(words)], words[(ci+3)%len(words)], words[(ci+5)%len(words)]}
			for _, order := range permutations(tuple) {
				joiner := joiners[(ci+len(order[0]))%len(joiners)]
				parts := []string{cue + " " + w(numbers[0])}
				for k, word := range order {
					parts = append(parts, word+" "+w(numbers[k+1]))
				}
				line := strings.Join(parts, joiner)
				lines++
				got := maskedValues(t, line, entity.Phone)
				for k := range parts {
					if !got[w(numbers[k])] && masksAlone(t, cue+" "+w(numbers[0]), w(numbers[0]), entity.Phone) {
						t.Fatalf("%q: %q not masked (%v)", line, w(numbers[k]), got)
					}
				}
			}
		}
	}
	t.Logf("%d continuation lines", lines)
	scanOnly = false
	for _, sep := range []string{"~", "\u301c", "/"} {
		w := func(s string) string { return strings.ReplaceAll(s, "~", sep) }
		for _, c := range []struct{ text, want string }{
			{"phone 804~555~1234 and work 703~555~2468", "703~555~2468"},
			{"phone 804~555~1234, work 703~555~2468", "703~555~2468"},
			{"phone 804~555~1234, work 703~555~2468, home 202~555~0147", "202~555~0147"},
			{"tel: 804~555~1234; office: 703~555~2468", "703~555~2468"},
		} {
			if !maskedValues(t, w(c.text), entity.Phone)[w(c.want)] {
				t.Errorf("%q: %q not masked", w(c.text), w(c.want))
			}
		}
		for _, text := range []string{"work 804~555~1234", "phone 804~555~1234 and work 804 hours", "phone 804~555~1234 and work 2024~2025", "work/home 804~555~1234", "phone 804~555~1234\nwork 703~555~2468", "phone 804~555~1234 and worker 703~555~2468"} {
			text = w(text)
			for v := range maskedValues(t, text, entity.Phone) {
				if v != w("804~555~1234") || strings.HasPrefix(text, "work") {
					t.Errorf("%q: unexpected phone %q", text, v)
				}
			}
		}
	}
}
