package outworld

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/scaletest"
	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

var standInKey = bytes.Repeat([]byte{5}, 32)

var exactPhoneCases = []string{
	"WhatsApp+1 804\u2013555\u20131234",
	"Tel+1\u00a0804\u00a0555\u00a01234",
	"\u96fb\u8a71+1 804\u2013555\u20131234",
	"\u7535\u8bdd+1 804\uff0d555\uff0d1234",
	"\u96fb\u8a71+44 20 7946\u20130958",
	"Call+804~555~1234",
	"Call 11:(826)\u2212696\u20129421",
	"1\uff0b1545 345\uff0d6065",
	"Call 804\u200b555\u200b1234",
	"Call 804\u00ad555\u00ad1234",
	"Call 804\u2060555\u20601234",
	"Call 804\ufeff555\ufeff1234",
	"Call 804\u00a0555\u00a01234 or 804\u2009555\u20091234 or 804\u2013555\u20131234",
}

func fileEngine(t *testing.T, path string) *Engine {
	t.Helper()
	v, err := openTestVault(path, standInKey)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { v.Close() })
	e, err := New(Options{Vault: v, Policy: DefaultPolicy()})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestPhoneStandInsRestoreExactly(t *testing.T) {
	for _, in := range exactPhoneCases {
		path := filepath.Join(t.TempDir(), "v.bin")
		e := fileEngine(t, path)
		if _, err := e.Vault().Assign("phone", "804~555~1234", func(int) string { return "555 867 5309" }); err != nil {
			t.Fatal(err)
		}
		s := e.NewSession("p")
		out, err := s.Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if out == in {
			t.Errorf("%q not masked", in)
			continue
		}
		if back := s.Rehydrate(out); back != in {
			t.Errorf("same session %q -> %q -> %q", in, out, back)
		}
		if back := e.NewSession("p").Rehydrate(out); back != in {
			t.Errorf("later session %q -> %q", out, back)
		}
		e.Vault().Close()
		other := fileEngine(t, path)
		if back := other.NewSession("p").Rehydrate(out); back != in {
			t.Errorf("after reopen %q -> %q", out, back)
		}
	}
}

func TestPhoneMaskThenSeparateEngineUnmask(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.bin")
	masker := fileEngine(t, path)
	var outs []string
	for _, in := range exactPhoneCases {
		out, err := masker.NewSession("cli").Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		outs = append(outs, out)
	}
	masker.Vault().Close()
	unmasker := fileEngine(t, path)
	for i, out := range outs {
		if back := unmasker.NewSession("cli").Rehydrate(out); back != exactPhoneCases[i] {
			t.Errorf("%q -> %q -> %q", exactPhoneCases[i], out, back)
		}
	}
}

func randomPhone(r *rand.Rand) string {
	seps := textutil.PhoneSeparators()
	zw := []string{"\u200b", "\u00ad", "\u2060", "\ufeff"}
	sep := func() string {
		s := string(seps[r.Intn(len(seps))])
		if r.Intn(4) == 0 {
			s += zw[r.Intn(len(zw))]
		}
		return s
	}
	groups := [][]int{{3, 3, 4}, {3, 4}, {2, 4, 4}}[r.Intn(3)]
	var b strings.Builder
	if r.Intn(3) == 0 {
		b.WriteString([]string{"+1", "\uff0b1", "+44"}[r.Intn(3)])
		b.WriteString(sep())
	}
	for gi, n := range groups {
		if gi > 0 {
			b.WriteString(sep())
		}
		for k := 0; k < n; k++ {
			d := byte('2' + r.Intn(8))
			b.WriteByte(d)
			if r.Intn(20) == 0 {
				b.WriteString(zw[r.Intn(len(zw))])
			}
		}
	}
	return b.String()
}

func TestPhoneMaskedImpliesRestorable(t *testing.T) {
	n := scaletest.Size(20000)
	if testing.Short() {
		n = 2000
	}
	r := rand.New(rand.NewSource(20261124))
	neighbours := []string{"", " ", "a", "Call", "\u96fb\u8a71", "7", "(", ")", "[", "]", "+", ",", ".", ":", "\u00a0", "x"}
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	failures, checked, other := 0, 0, 0
	for i := 0; i < n && failures < 20; i++ {
		in := neighbours[r.Intn(len(neighbours))] + randomPhone(r) + neighbours[r.Intn(len(neighbours))]
		s := e.NewSession(fmt.Sprintf("prop-%d", i%50))
		s.SetTrace(true)
		out, err := s.Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if out == in || strings.Contains(out, "[redacted") {
			continue
		}
		if !onlyPhones(s.Replacements()) {
			other++
			continue
		}
		checked++
		if back := e.NewSession(fmt.Sprintf("prop-%d", i%50)).Rehydrate(out); back != in {
			failures++
			t.Errorf("case %d: %q -> %q -> %q", i, in, out, back)
		}
	}
	t.Logf("%d cases, %d phone-only masks checked, %d masks involving other types skipped", n, checked, other)
}

func onlyPhones(rs []Replacement) bool {
	for _, r := range rs {
		if r.Type != "phone" {
			return false
		}
	}
	return len(rs) > 0
}

func TestReformattedPhoneStandInFallsBack(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	s := e.NewSession("p")
	in := "Call 804\u2013555\u20131234 today"
	out, err := s.Mask(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	standIn := strings.TrimSuffix(strings.TrimPrefix(out, "Call "), " today")
	reformatted := strings.ReplaceAll(standIn, "\u2013", "-")
	back := s.Rehydrate("Dial " + reformatted + " now")
	if digitsOf(back) != "8045551234" || strings.Contains(back, digitsOf(standIn)) {
		t.Fatalf("reformatted stand-in %q restored as %q", reformatted, back)
	}
}

func TestOlderVaultPhoneRestoresThroughFallback(t *testing.T) {
	be := vault.NewMemoryBackend(standInKey)
	v, err := vault.Open(be)
	if err != nil {
		t.Fatal(err)
	}
	real := "804\u00a0555\u00a01234"
	sur, err := v.Assign("phone", real, func(int) string { return "555 867 5309" })
	if err != nil {
		t.Fatal(err)
	}
	e, err := New(Options{Vault: v, Policy: DefaultPolicy()})
	if err != nil {
		t.Fatal(err)
	}
	emitted := reformatDigits(real, sur)
	if back := e.NewSession("p").Rehydrate("Call " + emitted); digitsOf(back) != "8045551234" {
		t.Fatalf("older-vault stand-in %q restored as %q", emitted, back)
	}
}

func TestPhoneFormsAreBounded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.bin")
	e := fileEngine(t, path)
	r := rand.New(rand.NewSource(7))
	seps := textutil.PhoneSeparators()
	zw := []string{"", "\u200b", "\u00ad", "\u2060", "\ufeff"}
	variant := func() string {
		a := string(seps[r.Intn(len(seps))]) + zw[r.Intn(len(zw))]
		b := string(seps[r.Intn(len(seps))]) + zw[r.Intn(len(zw))]
		return "804" + a + "555" + b + "1234"
	}
	total := 10000
	if scaletest.Race {
		total = 1000
	}
	first, firstSize := 0, int64(0)
	type pair struct{ in, out string }
	var sent []pair
	for i := 0; i < total; i++ {
		in := "Call " + variant() + " now"
		s := e.NewSession("p")
		out, err := s.Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if out != in {
			if back := s.Rehydrate(out); back != in {
				t.Fatalf("variant %d same session: %q -> %q -> %q", i, in, out, back)
			}
			if back := e.NewSession("q").Rehydrate(out); back != in {
				t.Fatalf("variant %d new session: %q -> %q -> %q", i, in, out, back)
			}
			sent = append(sent, pair{in, out})
		}
		if i == 99 {
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			first, firstSize = e.Vault().Len(), info.Size()
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if n := e.Vault().Len(); n > vault.MaxPhoneForms+4 || n != first || info.Size() != firstSize {
		t.Fatalf("after %d variants: %d records, %d bytes; after 100: %d records, %d bytes", total, n, info.Size(), first, firstSize)
	}
	t.Logf("%d variants: records %d, file %d bytes, %d masked", total, e.Vault().Len(), info.Size(), len(sent))
	e.Vault().Close()
	reopened := fileEngine(t, path)
	for _, p := range sent {
		if back := reopened.NewSession("r").Rehydrate(p.out); back != p.in {
			t.Fatalf("after reopen: %q -> %q -> %q", p.in, p.out, back)
		}
	}
	text := strings.Repeat("Call 555 867 5309 and 804 555 1234 today. ", 2000)
	hits := e.Vault().FindForms(text, nil)
	if len(hits) > 2000 {
		t.Fatalf("%d form hits", len(hits))
	}
}

func TestPhoneStandInNeverEqualsARealValuesBytes(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	first, err := e.NewSession("p").Mask(ctx, "Call 804-555-1234 now")
	if err != nil {
		t.Fatal(err)
	}
	base := strings.TrimSuffix(strings.TrimPrefix(first, "Call "), " now")
	other := strings.ReplaceAll(base, "-", "\u2010")
	if _, err := e.NewSession("p").Mask(ctx, "Call "+other+" now"); err != nil {
		t.Fatal(err)
	}
	s := e.NewSession("p")
	third, err := s.Mask(ctx, "Call 804\u2010555\u20101234 now")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(third, other) {
		t.Fatalf("stand-in %q equals the real value %q byte for byte", third, other)
	}
	if back := s.Rehydrate(third); back != "Call 804\u2010555\u20101234 now" && third != "Call [redacted phone] now" {
		t.Fatalf("%q restored as %q", third, back)
	}
}

func TestPhoneStandInCollisionsNeverRestoreAnotherValue(t *testing.T) {
	for _, kind := range []string{"account", "phone.real"} {
		e := newEngine(t, func(o *Options) { o.Dictionary = nil })
		real := "804\u2013555\u20131234"
		sur, err := e.Vault().Assign("phone", real, func(int) string { return "555 867 5309" })
		if err != nil {
			t.Fatal(err)
		}
		emitted := reformatDigits(real, sur)
		other := "ACCT-77"
		if kind == "account" {
			if _, err := e.Vault().Assign("account", other, func(int) string { return emitted }); err != nil {
				t.Fatal(err)
			}
		} else {
			other = emitted
			if _, err := e.Vault().Assign("phone", emitted, func(int) string { return "212 555 0100" }); err != nil {
				t.Fatal(err)
			}
		}
		s := e.NewSession("p")
		out, err := s.Mask(ctx, "Call "+real+" now")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, emitted) {
			t.Fatalf("%s: stand-in reused a string that already means something else: %q", kind, out)
		}
		if back := s.Rehydrate(out); back != "Call "+real+" now" && out != "Call [redacted phone] now" {
			t.Fatalf("%s: %q restored as %q", kind, out, back)
		}
		if kind == "account" {
			if back := s.Rehydrate("ref " + emitted); back != "ref "+other {
				t.Fatalf("the other record's surrogate restored as %q", back)
			}
		}
	}
}

func TestDigitStylesGetSeparateStandIns(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	a, b := "Call \uff18\uff10\uff14-\uff15\uff15\uff15-\uff11\uff12\uff13\uff14 now", "Call 804-555-1234 now"
	outA, err := e.NewSession("p").Mask(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	outB, err := e.NewSession("p").Mask(ctx, b)
	if err != nil {
		t.Fatal(err)
	}
	if outA == a || outA == outB || textutil.FoldForms(outA) != outB {
		t.Fatalf("expected one number in two digit styles, got %q and %q", outA, outB)
	}
	for in, out := range map[string]string{a: outA, b: outB} {
		if back := e.NewSession("p").Rehydrate(out); back != in {
			t.Fatalf("%q restored as %q, want %q", out, back, in)
		}
	}
}

func TestLegacyStyledFormNeverStylesASCIIInput(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	if _, err := e.NewSession("p").Mask(ctx, "Call 804-555-1234 now"); err != nil {
		t.Fatal(err)
	}
	sur, _ := e.Vault().Get("phone", "804-555-1234")
	legacy := reformatDigits("8045551234", sur)
	if added, err := e.Vault().AddPhoneForm("\uff18\uff10\uff14\uff15\uff15\uff15\uff11\uff12\uff13\uff14", legacy); err != nil || !added {
		t.Fatalf("legacy form not stored: %v %v", added, err)
	}
	for _, in := range []string{"x 8045551234 y", "#8045551234", "Tel8045551234"} {
		out, err := e.NewSession("p").Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if back := e.NewSession("p").Rehydrate(out); back != in {
			t.Errorf("%q -> %q restores %q", in, out, back)
		}
	}
	if back := e.NewSession("p").Rehydrate("ref " + legacy); back != "ref 8045551234" {
		t.Errorf("the legacy stand-in restores %q", back)
	}
}

func TestStandInNotReplacedInsideLongerNumber(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	in := "Call 804\u2013555\u20131234 now"
	out, err := e.NewSession("p").Mask(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	standIn := strings.TrimSuffix(strings.TrimPrefix(out, "Call "), " now")
	for _, wrap := range []string{"9" + standIn, standIn + "7", "12" + standIn, standIn + "00"} {
		if hits := e.Vault().FindForms("x "+wrap+" y", standInFits); len(hits) != 0 {
			t.Errorf("exact replacement fired inside %q", wrap)
		}
	}
	for _, glue := range []string{"WhatsApp" + standIn, "(" + standIn + ")", "\u96fb\u8a71" + standIn, "+" + standIn, "+44\u00b7" + standIn, standIn + "\u00b73447"} {
		if hits := e.Vault().FindForms(glue, standInFits); len(hits) != 1 {
			t.Errorf("glued stand-in %q not found", glue)
		}
		if back := e.NewSession("p").Rehydrate(glue); !strings.Contains(back, "804\u2013555\u20131234") {
			t.Errorf("glued stand-in %q not restored: %q", glue, back)
		}
	}
}

func TestPhoneMaskedImpliesRestorablePastTheCap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.bin")
	e := fileEngine(t, path)
	seps := textutil.PhoneSeparators()
	numbers := [][3]string{{"804", "555", "1234"}, {"212", "555", "0199"}, {"415", "867", "5309"}}
	for _, n := range numbers {
		for k := 0; k < 2*vault.MaxPhoneForms; k++ {
			a, b := string(seps[k%len(seps)]), string(seps[(k*7+3)%len(seps)])
			if _, err := e.NewSession("fill").Mask(ctx, "Call "+n[0]+a+n[1]+b+n[2]+" now"); err != nil {
				t.Fatal(err)
			}
		}
		if !e.Vault().FormsFull(n[0] + n[1] + n[2]) {
			t.Fatalf("cap not reached for %v", n)
		}
	}
	type pair struct{ in, out string }
	var sent []pair
	check := func(in string) {
		s := e.NewSession("p")
		out, err := s.Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if out == in || strings.Contains(out, "[redacted") {
			return
		}
		if back := s.Rehydrate(out); back != in {
			t.Errorf("same session: %q -> %q -> %q", in, out, back)
		}
		if back := e.NewSession("q").Rehydrate(out); back != in {
			t.Errorf("new session: %q -> %q -> %q", in, out, back)
		}
		sent = append(sent, pair{in, out})
	}
	for _, in := range []string{"tel:804.555.1234_ ", "\u96fb\u8a71804-555-1234", "TIN: 804-555-1234_ ", "Call 804\u200b555\u200b1234 now", "Call 804 \u200b555\u30001234 now", "+14158675309_ ", "Contact: +1 212\u2012555\u20120199_ "} {
		check(in)
	}
	r := rand.New(rand.NewSource(99))
	zw := []string{"", "\u200b", "\u00ad", "\u2060", "\ufeff"}
	neighbours := []string{"", " ", "Call ", "\u96fb\u8a71", "tel:", "(", "_", "x"}
	for i := 0; i < scaletest.Size(3000); i++ {
		n := numbers[r.Intn(len(numbers))]
		a := string(seps[r.Intn(len(seps))]) + zw[r.Intn(len(zw))]
		b := string(seps[r.Intn(len(seps))]) + zw[r.Intn(len(zw))]
		check(neighbours[r.Intn(len(neighbours))] + n[0] + a + n[1] + b + n[2] + neighbours[r.Intn(len(neighbours))] + " end")
	}
	e.Vault().Close()
	reopened := fileEngine(t, path)
	for _, p := range sent {
		if back := reopened.NewSession("r").Rehydrate(p.out); back != p.in {
			t.Errorf("after reopen: %q -> %q -> %q", p.in, p.out, back)
		}
	}
	t.Logf("%d masked past the cap, records %d", len(sent), reopened.Vault().Len())
}
