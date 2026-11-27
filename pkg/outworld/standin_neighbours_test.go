package outworld

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/scaletest"
	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/dateshift"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

func neighbourPieces(r *rand.Rand) string {
	seps := textutil.PhoneSeparators()
	sep := func() string { return string(seps[r.Intn(len(seps))]) }
	pieces := []func() string{
		func() string { return "" },
		func() string { return " " },
		func() string { return string(rune('0' + r.Intn(10))) },
		func() string { return string(rune('0'+r.Intn(10))) + sep() },
		func() string { return "\u0669 " },
		func() string { return "703" + sep() + "555" + sep() + "2468" },
		func() string { return " or 703-555-2468" },
		func() string { return "703 555 2468 " },
		func() string { return "%41" },
		func() string { return "%20" },
		func() string { return "q=%41" },
		func() string { return "\u200b" },
		func() string { return "\u2060" },
		func() string { return "No1 " },
		func() string { return "_" },
		func() string { return "-" },
		func() string { return "(" },
		func() string { return ")" },
		func() string { return "\u96fb\u8a71" },
		func() string { return strings.Repeat("9 ", 1+r.Intn(9)) },
		func() string { return "Call " },
		func() string { return " now" },
		func() string { return "\u00b2" },
		func() string { return "\u00b9" },
		func() string { return "\u00b3\u2070" },
		func() string { return "\u2074\u2079" },
		func() string { return "\u2083" },
		func() string { return "\u2462" },
		func() string { return "\u00ad" },
		func() string { return "x" },
		func() string { return "\u00e9" },
		func() string { return "ref" },
	}
	n := 1 + r.Intn(2)
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString(pieces[r.Intn(len(pieces))]())
	}
	return b.String()
}

func neighbourPhone(r *rand.Rand) string {
	seps := textutil.PhoneSeparators()
	zw := []string{"", "", "\u200b", "\u2060"}
	zw = append(zw, "\u00ad")
	a := string(seps[r.Intn(len(seps))]) + zw[r.Intn(len(zw))]
	b := string(seps[r.Intn(len(seps))]) + zw[r.Intn(len(zw))]
	return "804" + a + "555" + b + "1234"
}

func fillAllCaps(t *testing.T, e *Engine) {
	t.Helper()
	seps := textutil.PhoneSeparators()
	for k := 0; !e.Vault().FormsFull("8045551234") && k < 4*vault.MaxPhoneForms; k++ {
		a, b := string(seps[k%len(seps)]), string(seps[(k*5+1)%len(seps)])
		if _, err := e.NewSession("fill").Mask(ctx, "Call 804"+a+"555"+b+"1234 now"); err != nil {
			t.Fatal(err)
		}
	}
	for k := 0; k < vault.MaxPhoneFormsInContext; k++ {
		zw := strings.Repeat("\u200b", k+1)
		e.Vault().AddPhoneFormInContext("804"+zw+"555 1234", "306"+zw+"555 0172")
	}
}

func TestMaskNeverSendsAFindingRaw(t *testing.T) {
	n := 300
	if scaletest.Race {
		n = 60
	}
	r := rand.New(rand.NewSource(1124))
	redacted := map[string]int{}
	capped := newEngine(t, func(o *Options) { o.Dictionary = nil })
	capped.NewSession("seed").Mask(ctx, "Call 804-555-1234 now")
	fillAllCaps(t, capped)
	for _, mode := range []string{"fresh", "seeded", "seeded-zw", "capped"} {
		bad := 0
		for i := 0; i < n && bad < 10; i++ {
			phone := neighbourPhone(r)
			in := neighbourPieces(r) + phone + neighbourPieces(r)
			e := capped
			if mode != "capped" {
				e = newEngine(t, func(o *Options) { o.Dictionary = nil })
			}
			switch mode {
			case "seeded":
				e.NewSession("seed").Mask(ctx, "Call "+phone+" now")
			case "seeded-zw":
				e.NewSession("seed").Mask(ctx, "Call "+strings.ReplaceAll(phone, "555", "\u200b555\u200b")+" now")
				e.NewSession("seed").Mask(ctx, "Call "+phone+" now")
			case "capped":
				e.NewSession("seed").Mask(ctx, "Call "+phone+" now")
			}
			findings := phoneFindings(t, e, in)
			s := e.NewSession("n")
			s.SetTrace(true)
			out, err := s.Mask(ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			for f, k := range countFindings(findings) {
				if strings.Count(out, f)-(strings.Count(in, f)-k) > 0 {
					bad++
					t.Errorf("%s: %q -> %q leaves the finding %q in place", mode, in, out, f)
				}
			}
			leaked := false
			for _, rep := range s.Replacements() {
				if rep.Replacement == phone || strings.Contains(digitsOf(textutil.FoldForms(rep.Replacement)), "8045551234") {
					leaked = true
				}
			}
			if leaked {
				bad++
				t.Errorf("%s: %q -> %q sends a finding's real number", mode, in, out)
				continue
			}
			if out == in {
				continue
			}
			if strings.Contains(out, "[redacted") {
				redacted[mode+": "+shapeOf(in, phone)]++
				continue
			}
			if back := e.NewSession("n").Rehydrate(out); back != in {
				bad++
				t.Errorf("%s: %q -> %q restores %q", mode, in, out, back)
			}
		}
	}
	total := 0
	for k, v := range redacted {
		total += v
		t.Logf("redacted %d x %s", v, k)
	}
	t.Logf("%d contexts per mode, %d ended in a redaction token", n, total)
}

func shapeOf(in, phone string) string {
	i := strings.Index(in, phone)
	if i < 0 {
		return "?"
	}
	return "left=" + shapeClass(in[:i]) + " right=" + shapeClass(in[i+len(phone):])
}

func shapeClass(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case textutil.DigitLike(r):
			b.WriteByte('9')
		case textutil.IsZeroWidth(r):
			b.WriteByte('z')
		case r == '%':
			b.WriteByte('%')
		case textutil.IsPhoneSeparator(r):
			b.WriteByte('-')
		case textutil.IsWordRune(r):
			b.WriteByte('a')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TestTwoAdjacentPhonesBothRestore(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	e.NewSession("seed").Mask(ctx, "Call 804-555-1234 now")
	for _, in := range []string{
		"Call 804\u00b7555\u20101234 or 703-555-2468",
		"703 555 2468 804\uff0d555\u00b71234-0000",
		"\u0669 804 555\u00b71234 703 555 2468",
		"9 9 9 9 9 9 9 9 9 804\u2014555~1234 or 703-555-2468",
		"804-555-1234 703-555-2468",
	} {
		s := e.NewSession("p")
		out, err := s.Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(digitsOf(textutil.FoldForms(out)), "8045551234") || strings.Contains(digitsOf(textutil.FoldForms(out)), "7035552468") {
			t.Errorf("%q -> %q keeps a real number", in, out)
		}
		if back := e.NewSession("p").Rehydrate(out); back != in && !strings.Contains(out, "[redacted") {
			t.Errorf("%q -> %q restores %q", in, out, back)
		}
		if got := streamed(e.NewSession("p"), out, []int{1}); got != e.NewSession("p").Rehydrate(out) {
			t.Errorf("%q streamed differs: %q", in, got)
		}
	}
}

func TestKnownNumberBetweenPunctuationMasks(t *testing.T) {
	for _, c := range [][2]string{{"Call 804\u2013555\u20131234 now", "q=%41804\u2013555\u20131234%20x"}, {"Call 8045551234 now", "_8045551234-"}, {"Call 804.555.1234 now", "-804.555.1234_"}, {"Call (804) 555-1234 now", "/(804) 555-1234_"}, {"Call 804 555 1234 now", "1-804 555 1234-"}} {
		for _, capped := range []bool{false, true} {
			e := newEngine(t, func(o *Options) { o.Dictionary = nil })
			e.NewSession("seed").Mask(ctx, c[0])
			if capped {
				fillAllCaps(t, e)
			}
			out, err := e.NewSession("p").Mask(ctx, c[1])
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(digitsOf(textutil.FoldForms(out)), "8045551234") {
				t.Errorf("capped=%v %q -> %q keeps the known number", capped, c[1], out)
				continue
			}
			if back := e.NewSession("p").Rehydrate(out); back != c[1] && !strings.Contains(out, "[redacted") {
				t.Errorf("capped=%v %q -> %q restores %q", capped, c[1], out, back)
			}
		}
	}
}

func streamBlob(alpha string, n int, seed int64) string {
	r := rand.New(rand.NewSource(seed))
	var b strings.Builder
	b.WriteString("Here is the blob ")
	for b.Len() < n {
		b.WriteByte(alpha[r.Intn(len(alpha))])
	}
	b.WriteString(" done.")
	return b.String()
}

func TestLongRunStreamingCostsLikePlainText(t *testing.T) {
	if testing.Short() {
		t.Skip("timing test skipped in short mode")
	}
	e := newEngine(t, nil)
	s := e.NewSession("blob")
	if _, err := s.Mask(ctx, "Call Maria Garcia at 804-555-1234."); err != nil {
		t.Fatal(err)
	}
	le := newEngine(t, nil)
	ls := le.NewSession("blob")
	if _, err := ls.Mask(ctx, "Call Maria Garcia at 804-555-1234."); err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("Long Stand In Avenue ", 10)
	if _, err := le.Vault().Assign("address", "real "+long, func(int) string { return long }); err != nil {
		t.Fatal(err)
	}
	if le.Vault().MaxSurrogateLen() < 200 {
		t.Fatalf("longest stand-in %d bytes", le.Vault().MaxSurrogateLen())
	}
	n := scaletest.Size(256 << 10)
	inputs := map[string]string{
		"plain":  streamBlob("the quick brown fox jumps ", n, 1),
		"digits": streamBlob("0123456789", n, 2),
		"alnum":  streamBlob("abcdefghijklmnopqrstuvwxyz0123456789", n, 3),
	}
	run := func(s *Session, text string) time.Duration {
		start := time.Now()
		r := s.NewStreamRehydrator(true)
		for i := 0; i < len(text); i += 20 {
			r.Push(text[i:min(len(text), i+20)])
		}
		r.Flush()
		return time.Since(start)
	}
	cases := []string{"plain", "digits", "alnum", "long plain", "long alnum"}
	best := map[string]time.Duration{}
	rounds := 3
	if scaletest.Race {
		rounds = 1
	}
	for round := 0; round < rounds; round++ {
		for _, k := range cases {
			sess, text := s, inputs[k]
			if name, ok := strings.CutPrefix(k, "long "); ok {
				sess, text = ls, inputs[name]
			}
			d := run(sess, text)
			if cur, ok := best[k]; !ok || d < cur {
				best[k] = d
			}
		}
	}
	t.Logf("256 KB in 20-byte chunks: plain %v, digits %v, alnum %v; with a %d-byte stand-in: plain %v, alnum %v", best["plain"], best["digits"], best["alnum"], le.Vault().MaxSurrogateLen(), best["long plain"], best["long alnum"])
	for _, c := range [][2]string{{"digits", "plain"}, {"alnum", "plain"}, {"long alnum", "alnum"}, {"long alnum", "long plain"}} {
		if !scaletest.Race && 2*best[c[0]] > 3*best[c[1]] {
			t.Errorf("%s streams %v, more than 1.5 times the %s %v", c[0], best[c[0]], c[1], best[c[1]])
		}
	}
}

func TestNoDetectedPhoneIsLeftInPlace(t *testing.T) {
	n := 1500
	if scaletest.Race {
		n = 150
	}
	r := rand.New(rand.NewSource(77))
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	e.NewSession("seed").Mask(ctx, "Call 415-867-5309 now")
	found, masked, redacted, left := 0, 0, 0, 0
	others := []string{"x_536-22-8471\u00e9 ", "\u96fb\u8a71 \u00e9536/22/8471: ", "id 536.22.8471\n ", "tel:\u00e9536/22/8471. ", ""}
	for i := 0; i < n; i++ {
		phone := neighbourPhone(r)
		if r.Intn(3) == 0 {
			phone = strings.NewReplacer("804", "415", "555", "867", "1234", "5309").Replace(phone)
		}
		in := others[r.Intn(len(others))] + neighbourPieces(r) + phone + neighbourPieces(r)
		findings := phoneFindings(t, e, in)
		s := e.NewSession("fz")
		out, err := s.Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		for f, k := range countFindings(findings) {
			found += k
			kept := strings.Count(out, f) - (strings.Count(in, f) - k)
			if kept > 0 {
				left += kept
				t.Errorf("%q -> %q leaves the finding %q in place", in, out, f)
			}
			masked += k - max(kept, 0)
		}
		if strings.Contains(out, "[redacted phone]") {
			redacted++
		}
	}
	t.Logf("%d inputs, %d phone findings, %d masked, %d left in place, %d outputs with a phone redaction token", n, found, masked, left, redacted)
}

func countFindings(fs []string) map[string]int {
	out := map[string]int{}
	for _, f := range fs {
		out[f]++
	}
	return out
}

func TestKnownNumberBesideOtherIDsIsMasked(t *testing.T) {
	for _, in := range []string{
		"x_536-22-8471\u00e9 tin 415_867\u20225309 ext 12 ",
		"id 536.22.8471\n sms (415\u2014867/5309/ ",
		"\u96fb\u8a71 \u00e9536/22/8471: ITIN 1 212\u2022555\u20220199 now ",
		"phone: \u00e9536.22.8471\n cell +1 912\u202270\u22121234: ",
		"tel:\u00e9536/22/8471. Call 1-703\u00b7555\u20120142\n ",
	} {
		e := newEngine(t, nil)
		if _, err := e.NewSession("seed").Mask(ctx, "Call 415-867-5309 now"); err != nil {
			t.Fatal(err)
		}
		findings := phoneFindings(t, e, in)
		s := e.NewSession("ids")
		out, err := s.Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, "5309") {
			t.Errorf("%q -> %q sends the known number", in, out)
		}
		for _, f := range findings {
			if strings.Contains(out, f) {
				t.Errorf("%q -> %q leaves %q in place", in, out, f)
			}
		}
		if back := e.NewSession("ids").Rehydrate(out); back != in {
			t.Errorf("%q -> %q -> %q", in, out, back)
		}
	}
}

func TestScriptDigitBesideNumberRestoresExactly(t *testing.T) {
	for _, in := range []string{
		"Dial 9 804\u2014555\u00ad1234\u00b2",
		"Call 804-555\u00ad1234\u00b2",
		"Call 804-555-1234\u00b2",
		"Call 804-555-1234\u00b2\u00b3 x",
		"Call 804-555-1234\u2462 x",
		"Call 804-555-1234\u2083 x",
		"x \u2079804-555-1234\u2070 x",
	} {
		e := newEngine(t, nil)
		out, err := e.NewSession("p").Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(digitsOf(textutil.FoldForms(out)), "8045551234") {
			t.Errorf("%q -> %q keeps the number", in, out)
		}
		if back := e.NewSession("p").Rehydrate(out); back != in {
			t.Errorf("%q -> %q -> %q", in, out, back)
		}
		if got := streamed(e.NewSession("p"), out, []int{1, 3}); got != in {
			t.Errorf("streamed %q -> %q -> %q", in, out, got)
		}
	}
}

func cutCheckText(r *rand.Rand) string {
	pieces := []string{"abc", "Zq", "x9", "7", "42", "2024", "20240512", "12MAY2024", "May 5", "Sept 3rd", "the fifth of May", "Wednesday, May 5, 2024", "804-555-1234", "(804) 555 1234", "+44 20 7946 0958", "5/12", "12/05/2024", "2024-05-12", " ", ", ", "-", "/", ".", "\u00a0", "\u017fep 5", "Junk", "marmot", "oct0ber", "x", "y", "zz", "the fifth", "fifth of ", "Tuesday the ", "on the ", "Ninth ", "ME ", "Octo", "0", "555", "01234567"}
	var b strings.Builder
	for b.Len() < 200 {
		if r.Intn(3) == 0 {
			b.WriteString(pieces[r.Intn(len(pieces))])
			continue
		}
		b.WriteByte("abcdefghijklmnopqrstuvwxyz0123456789"[r.Intn(36)])
	}
	return b.String()
}

func TestCutChecksSkipOnlyWindowsWithoutMatches(t *testing.T) {
	r := rand.New(rand.NewSource(5))
	skippedWord, skippedDate := 0, 0
	texts := 1500
	if scaletest.Race {
		texts = 100
	}
	for k := 0; k < texts; k++ {
		text := cutCheckText(r)
		words := strings.Map(func(c rune) rune {
			if c >= '0' && c <= '9' {
				return rune('a' + r.Intn(26))
			}
			return c
		}, text)
		for i := 0; i+40 <= len(words); i += 8 {
			w := words[i : i+40]
			if !utf8.ValidString(w) || mayHoldDate(w) {
				continue
			}
			skippedDate++
			for _, order := range []dateshift.Order{dateshift.MDY, dateshift.DMY} {
				if ms := dateshift.FindLoose(w, order); len(ms) > 0 {
					t.Fatalf("%q holds the date %q but was judged date-free", w, ms[0].Text)
				}
			}
		}
		for _, order := range []dateshift.Order{dateshift.MDY, dateshift.DMY} {
			if !mayHoldDate(text) {
				skippedDate++
				if ms := dateshift.FindLoose(text, order); len(ms) > 0 {
					t.Fatalf("%q holds the date %q but was judged date-free", text, ms[0].Text)
				}
			}
		}
		for cut := 1; cut < len(text); cut++ {
			if !utf8.RuneStart(text[cut]) || !insideWordRun(text, cut) {
				continue
			}
			skippedWord++
			lo, hi := max(0, cut-shapeReach), min(len(text), cut+shapeReach)
			for lo > 0 && !utf8.RuneStart(text[lo]) {
				lo--
			}
			for hi < len(text) && !utf8.RuneStart(text[hi]) {
				hi++
			}
			near := text[lo:hi]
			for _, order := range []dateshift.Order{dateshift.MDY, dateshift.DMY} {
				for _, m := range dateshift.FindLoose(near, order) {
					if lo+m.Start < cut && cut < lo+m.End {
						t.Fatalf("cut %d in %q splits the date %q", cut, text, m.Text)
					}
				}
			}
			for _, m := range phoneLikeRE.FindAllStringIndex(near, -1) {
				if lo+m[0] < cut && cut < lo+m[1] {
					t.Fatalf("cut %d in %q splits the phone %q", cut, text, near[m[0]:m[1]])
				}
			}
		}
		for end := 1; end <= len(text); end++ {
			if end < len(text) && !utf8.RuneStart(text[end]) {
				continue
			}
			seg := text[:end]
			from := phoneCharsTail(seg)
			var want, got []int
			for _, m := range phoneLikeRE.FindAllStringIndex(seg, -1) {
				if m[1] == end {
					want = append(want, m[0])
				}
			}
			if digitCount(seg[from:]) >= minPhoneLikeDigits {
				for _, m := range phoneLikeRE.FindAllStringIndex(seg[from:], -1) {
					if from+m[1] == end {
						got = append(got, from+m[0])
					}
				}
			}
			if fmt.Sprint(want) != fmt.Sprint(got) {
				t.Fatalf("%q: phone shapes ending at the cut %v, tail scan found %v", seg, want, got)
			}
		}
	}
	if skippedWord == 0 || skippedDate == 0 {
		t.Fatalf("the generator never reached a skip: word %d, date %d", skippedWord, skippedDate)
	}
	t.Logf("checked %d word-run cuts and %d date-free texts", skippedWord, skippedDate)
}

func TestVaultKnownPhoneGluedToLettersIsMasked(t *testing.T) {
	glued := []string{"x804-555-1234", "804-555-1234xyz", "x804-555-1234y", "ref804-555-1234b", "\u00e9804-555-1234", "804-555-1234\u00e9", "\u00e9804-555-1234\u00df", "\u96fb804-555-1234\u8a71", "abc8045551234def", "a(804) 555-1234b", "id_804.555.1234_x", "x804-555-1234 and y804-555-1234z"}
	digitRuns := []string{"98045551234567", "x98045551234567y", "8045551234567b", "a1804555123499"}
	for _, capped := range []bool{false, true} {
		e := newEngine(t, func(o *Options) { o.Dictionary = nil })
		if _, err := e.NewSession("seed").Mask(ctx, "Call 804-555-1234 now"); err != nil {
			t.Fatal(err)
		}
		if capped {
			fillAllCaps(t, e)
		}
		for _, in := range glued {
			s := e.NewSession("g")
			out, err := s.Mask(ctx, "ref "+in+" end")
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(digitsOf(textutil.FoldForms(out)), "8045551234") {
				t.Errorf("capped=%v %q -> %q sends the known number", capped, in, out)
				continue
			}
			if strings.Contains(out, "[redacted") {
				continue
			}
			if back := e.NewSession("g").Rehydrate(out); back != "ref "+in+" end" {
				t.Errorf("capped=%v %q -> %q restores %q", capped, in, out, back)
			}
			for _, size := range []int{1, 3, 7} {
				if got := streamed(e.NewSession("g"), out, []int{size}); got != "ref "+in+" end" {
					t.Errorf("capped=%v streamed at %d %q -> %q -> %q", capped, size, in, out, got)
				}
			}
		}
		for _, in := range digitRuns {
			if out, err := e.NewSession("g").Mask(ctx, "order "+in+" shipped"); err != nil || out != "order "+in+" shipped" {
				t.Errorf("capped=%v digit run %q changed to %q (%v)", capped, in, out, err)
			}
		}
	}
	fresh := newEngine(t, func(o *Options) { o.Dictionary = nil })
	for _, in := range glued[:4] {
		if out, err := fresh.NewSession("f").Mask(ctx, in); err != nil || out != in {
			t.Errorf("fresh engine %q changed to %q (%v)", in, out, err)
		}
	}
}

func TestStreamMatchesBatchAfterHistory(t *testing.T) {
	r := rand.New(rand.NewSource(655))
	lefts := []string{"", "page 3, ", "x=9;", "Call ", "1,", "703-555-2468, ", "703 555 2468 ", "ext 2 ", "%34", "q=%41"}
	rights := []string{"", " 703 555 2468", " or 703-555-2468", "%41", " ext 12", " now", "-9"}
	seps := []string{"-", " ", ".", "\u2010", "*", "~", "\u2007", "\u00ad", "-\u200b", "\u200b-", "\u2060", "\u2009", "\u00a0"}
	line := func() string {
		return lefts[r.Intn(len(lefts))] + "804" + seps[r.Intn(len(seps))] + "555" + seps[r.Intn(len(seps))] + "1234" + rights[r.Intn(len(rights))]
	}
	n := 700
	if scaletest.Race {
		n = 150
	}
	e := newEngine(t, nil)
	if _, err := e.NewSession("s0").Mask(ctx, "Call 804\u2013555\u20131234 now"); err != nil {
		t.Fatal(err)
	}
	bad, checked := 0, 0
	for i := 0; i < n && bad < 10; i++ {
		in := line()
		out, err := e.NewSession("s0").Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if out == in {
			continue
		}
		checked++
		batch := e.NewSession("s0").Rehydrate(out)
		if batch != in && !strings.Contains(out, "[redacted") {
			bad++
			t.Errorf("%q -> %q restores %q", in, out, batch)
			continue
		}
		for size := 1; size <= 40; size++ {
			if scaletest.Race && size%7 != 1 {
				continue
			}
			if got := streamed(e.NewSession("s0"), out, []int{size}); got != batch {
				bad++
				t.Errorf("line %d %q -> %q: chunks of %d stream %q, batch %q", i, in, out, size, got, batch)
				break
			}
		}
	}
	t.Logf("%d masked lines compared at every chunk size", checked)
}

func TestUnprovablePhoneIsRedactedNeverSentRaw(t *testing.T) {
	r := rand.New(rand.NewSource(1124))
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	if _, err := e.NewSession("seed").Mask(ctx, "Call 804-555-1234 now"); err != nil {
		t.Fatal(err)
	}
	fillAllCaps(t, e)
	redacted, tried := 0, 0
	for ; tried < 2000 && redacted < 5; tried++ {
		phone := neighbourPhone(r)
		in := neighbourPieces(r) + phone + neighbourPieces(r)
		if _, err := e.NewSession("seed").Mask(ctx, "Call "+phone+" now"); err != nil {
			t.Fatal(err)
		}
		out, err := e.NewSession("n").Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, "[redacted phone]") {
			continue
		}
		redacted++
		if strings.Contains(out, phone) || strings.Contains(digitsOf(textutil.FoldForms(out)), "8045551234") {
			t.Errorf("%q -> %q sends the known number next to a redaction", in, out)
		}
		if back := e.NewSession("n").Rehydrate(out); back != out {
			t.Errorf("%q -> %q: the redaction token rehydrated to %q", in, out, back)
		}
	}
	if redacted == 0 {
		t.Fatalf("no input in %d reached the redaction fallback", tried)
	}
	t.Logf("%d inputs reached the redaction fallback within %d tries", redacted, tried)
}

func TestKnownNumberBesideEscapesAndDigitGroupsMasks(t *testing.T) {
	forms := []string{"q=%41804\u2010555*1234%41", "q=%41804\u301c555\uff0d1234-9", "+9 804~555 1234 1234", "12 34 56 804-\u200b555-1234-0000", "No1 804.555\u22121234 1234", "%2B1804\u00a0555~1234-9"}
	longer := []string{"98045551234567", "order 8045551234567 shipped", "1804555123499", "804_555_12345"}
	for _, capped := range []bool{false, true} {
		e := newEngine(t, func(o *Options) { o.Dictionary = nil })
		if _, err := e.NewSession("s0").Mask(ctx, "Call 804\u2013555\u20131234 now"); err != nil {
			t.Fatal(err)
		}
		if capped {
			fillAllCaps(t, e)
		}
		for _, in := range forms {
			out, err := e.NewSession("s0").Mask(ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(digitsOf(textutil.FoldForms(out)), "8045551234") {
				t.Errorf("capped=%v %q -> %q sends the known number", capped, in, out)
				continue
			}
			if strings.Contains(out, "[redacted") {
				continue
			}
			if back := e.NewSession("s0").Rehydrate(out); back != in {
				t.Errorf("capped=%v %q -> %q restores %q", capped, in, out, back)
			}
			for _, size := range []int{1, 2, 5} {
				if got := streamed(e.NewSession("s0"), out, []int{size}); got != in {
					t.Errorf("capped=%v streamed at %d %q -> %q -> %q", capped, size, in, out, got)
				}
			}
		}
		for _, in := range longer {
			if out, err := e.NewSession("s0").Mask(ctx, in); err != nil || out != in {
				t.Errorf("capped=%v longer run %q changed to %q (%v)", capped, in, out, err)
			}
		}
	}
}
