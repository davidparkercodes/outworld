package outworld

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/scaletest"
)

var gluedStreamCases = []string{
	"電話番号は090-1234-5678です",
	"电话13812345678",
	"전화010-1234-5678로",
	"連絡先は03-1234-5678まで",
	"call804-555-1234",
	"電話804-555-1234",
	"電話番号は804-555-1234です",
	"Tel804-555-1234",
	"Phone804.555.1234",
}

func streamed(s *Session, text string, sizes []int) string {
	r := s.NewStreamRehydrator(false)
	var b strings.Builder
	pos, k := 0, 0
	for pos < len(text) {
		n := sizes[k%len(sizes)]
		k++
		end := min(len(text), pos+n)
		b.WriteString(r.Push(text[pos:end]))
		pos = end
	}
	b.WriteString(r.Flush())
	return b.String()
}

func checkStreamMatches(t *testing.T, s *Session, masked, want string, rng *rand.Rand) int {
	t.Helper()
	bad := 0
	for size := 1; size <= 16; size++ {
		if got := streamed(s, masked, []int{size}); got != want {
			bad++
			t.Errorf("chunk %d: %q streamed to %q, want %q", size, masked, got, want)
		}
	}
	splits := 8
	if scaletest.Race {
		splits = 1
	}
	for k := 0; k < splits; k++ {
		sizes := make([]int, 12)
		for i := range sizes {
			sizes[i] = 1 + rng.Intn(9)
		}
		if got := streamed(s, masked, sizes); got != want {
			bad++
			t.Errorf("random split %v: %q streamed to %q, want %q", sizes, masked, got, want)
		}
	}
	return bad
}

func streamCorpus() []string {
	var out []string
	for _, r := range generatedPhoneSeparators() {
		sep := string(r)
		out = append(out, "Call 804"+sep+"555"+sep+"1234 now", "Call (804)"+sep+"555"+sep+"1234 now", "Call +1"+sep+"804"+sep+"555"+sep+"1234 now")
	}
	out = append(out, detectionPassRestoreCases...)
	out = append(out, exactPhoneCases...)
	out = append(out, gluedStreamCases...)
	for _, l := range []string{"", " ", "f", "x", "7", "_", ")", "]", "=", "[", "'", ":", "(", "電話", "전화"} {
		for _, body := range []string{"(804)-555-1234", "(804) 555-1234", "804-555-1234", "804–555–1234"} {
			for _, r := range []string{"", " ", ",", ".", ")", "_", "です", "a"} {
				out = append(out, "note "+l+body+r+" end")
			}
		}
	}
	return out
}

func TestStreamingMatchesUnstreamedRestore(t *testing.T) {
	rng := rand.New(rand.NewSource(20261125))
	corpus := streamCorpus()
	masked, bad := 0, 0
	step := 1
	if scaletest.Race {
		step = 10
	}
	for ti := 0; ti < len(corpus); ti += step {
		text := corpus[ti]
		e := newEngine(t, nil)
		s := e.NewSession("stream")
		out, err := s.Mask(ctx, text)
		if err != nil {
			t.Fatal(err)
		}
		if out == text {
			continue
		}
		masked++
		want := s.Rehydrate(out)
		bad += checkStreamMatches(t, s, out, want, rng)
		for _, suffix := range []string{"y", "7", "_", "%41"} {
			bad += checkStreamMatches(t, s, out+suffix, s.Rehydrate(out+suffix), rng)
		}
		standIn := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(out, "Call "), " now"))
		if strings.HasPrefix(text, "Call ") && strings.HasSuffix(text, " now") {
			for _, prefix := range []string{"1", "+1", "x", "%41"} {
				for _, suffix := range []string{" y", "y", "7", "0"} {
					echo := "x " + prefix + standIn + suffix
					bad += checkStreamMatches(t, s, echo, s.Rehydrate(echo), rng)
				}
			}
		}
		if bad > 20 {
			t.Fatal("too many stream mismatches")
		}
	}
	t.Logf("%d texts, %d masked and streamed at 16 chunk sizes and 8 random splits each", len(corpus), masked)
	if masked < 150/step {
		t.Errorf("only %d texts masked", masked)
	}
}

func TestGluedStandInsRestoreStreamedAndNot(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	for _, text := range gluedStreamCases {
		e := newEngine(t, nil)
		s := e.NewSession("glue")
		out, err := s.Mask(ctx, text)
		if err != nil {
			t.Fatal(err)
		}
		if out == text {
			t.Errorf("%q not masked", text)
			continue
		}
		if back := s.Rehydrate(out); back != text {
			t.Errorf("%q -> %q -> %q", text, out, back)
		}
		checkStreamMatches(t, s, out, text, rng)
	}
}

func TestNameAndInitialismStandInsWaitForTheirRightBoundary(t *testing.T) {
	rng := rand.New(rand.NewSource(17))
	e := newEngine(t, nil)
	s := e.NewSession("names")
	out, err := s.Mask(ctx, "John Smith (JS) met Maria Garcia and jane.doe@example.com on March 3, 2024.")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range strings.FieldsFunc(out, func(r rune) bool { return r == ' ' || r == '(' || r == ')' || r == ',' || r == '.' }) {
		for _, suffix := range []string{"", "y", "7", "_", "%41", "s"} {
			for _, prefix := range []string{"", "x", "%41", "_"} {
				text := "a " + prefix + w + suffix + " b"
				checkStreamMatches(t, s, text, s.Rehydrate(text), rng)
			}
		}
	}
}

func TestLongFormsStream(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	for _, gap := range []int{2, 4, 7} {
		filler := strings.Repeat("­", gap)
		digits := "18045551234"
		var b strings.Builder
		b.WriteString("+")
		for i, d := range digits {
			if i > 0 {
				b.WriteString(filler)
			}
			b.WriteRune(d)
		}
		phone := b.String()
		text := "Call " + phone + " now"
		e := newEngine(t, nil)
		if _, err := e.Vault().Assign("phone", phone, func(int) string { return "+1 555 867 5309" }); err != nil {
			t.Fatal(err)
		}
		s := e.NewSession("long")
		out, err := s.Mask(ctx, text)
		if err != nil {
			t.Fatal(err)
		}
		if out == text {
			t.Fatalf("%d-byte phone not masked", len(phone))
		}
		if back := s.Rehydrate(out); back != text {
			t.Fatalf("%d-byte phone: %q restores %q", len(phone), out, back)
		}
		t.Logf("%d-byte phone, %d-byte stand-in", len(phone), len(out)-len("Call  now"))
		checkStreamMatches(t, s, out, text, rng)
	}
}

func TestOtherStandInsGluedAfterWordsStream(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	texts := []string{
		"mailjane.doe@example.com now",
		"email jane.doe@example.com now",
		"\u4f1a\u8b70\u306f2024-03-05\u3067\u3059",
		"due 2024-03-05 ok",
		"\u62c5\u5f53\u306f John Smith \u3067\u3059",
		"Met John Smith yesterday.",
		"Call John Smith(JS) now",
	}
	for _, text := range texts {
		e := newEngine(t, nil)
		s := e.NewSession("other")
		out, err := s.Mask(ctx, text)
		if err != nil {
			t.Fatal(err)
		}
		if out == text {
			t.Errorf("%q no longer masks", text)
			continue
		}
		checkStreamMatches(t, s, out, s.Rehydrate(out), rng)
		for _, glue := range []string{"x", "\u96fb\u8a71", "7"} {
			checkStreamMatches(t, s, glue+out+glue, s.Rehydrate(glue+out+glue), rng)
		}
	}
}
