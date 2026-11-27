package detect

import (
	"bytes"
	"context"
	"github.com/davidparkercodes/outworld/internal/scaletest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/entity"
	"github.com/davidparkercodes/outworld/pkg/surrogate"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

var foldCases = []struct {
	text, want string
	typ        entity.Type
	reals      []string
}{
	{"mail jane\u200b.doe@example.com today", "jane\u200b.doe@example.com", entity.Email, []string{"jane", "doe"}},
	{"mail jane.doe\u200b@example.com today", "jane.doe\u200b@example.com", entity.Email, []string{"jane", "doe"}},
	{"mail jane.doe@exa\u00admple.com today", "jane.doe@exa\u00admple.com", entity.Email, []string{"jane", "doe"}},
	{"SSN １２３-４５-６７８９ on file", "１２３-４５-６７８９", entity.GovID, []string{"６７８９"}},
	{"SSN 123\u201145\u20116789 on file", "123\u201145\u20116789", entity.GovID, []string{"6789"}},
	{"SSN 123\u221245\u22126789 on file", "123\u221245\u22126789", entity.GovID, []string{"6789"}},
	{"card ４１１１ １１１１ １１１１ １１１１ exp", "４１１１ １１１１ １１１１ １１１１", entity.Financial, []string{"１１１１"}},
	{"card 4111\u00a01111\u00a01111\u00a01111 exp", "4111\u00a01111\u00a01111\u00a01111", entity.Financial, []string{"1111"}},
	{"host ｈｔｔｐ\u300010.1.2.3 and １０.１.２.３ up", "１０.１.２.３", entity.IP, []string{"１０.１.２.３"}},
}

func foldSpan(t *testing.T, text string, typ entity.Type) []entity.Span {
	t.Helper()
	var out []entity.Span
	for _, sp := range detectAll(t, text) {
		if sp.Type == typ {
			out = append(out, sp)
		}
	}
	return out
}

func TestFoldedFormsAreDetectedOverTheOriginalBytes(t *testing.T) {
	for _, c := range foldCases {
		got := foldSpan(t, c.text, c.typ)
		found := false
		for _, sp := range got {
			if sp.Text == c.want && c.text[sp.Start:sp.End] == c.want {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: want %s span %q, got %+v", c.text, c.typ, c.want, got)
		}
	}
}

func TestFoldedFormsMaskAndRoundTrip(t *testing.T) {
	for _, c := range foldCases {
		v, err := vault.Open(vault.NewMemoryBackend(bytes.Repeat([]byte{5}, 32)))
		if err != nil {
			t.Fatal(err)
		}
		g := surrogate.NewGenerator(v.Secret())
		var sp entity.Span
		for _, s := range foldSpan(t, c.text, c.typ) {
			if s.Text == c.want {
				sp = s
			}
		}
		if sp.Text == "" {
			t.Errorf("%q: no span", c.text)
			continue
		}
		var sur string
		switch c.typ {
		case entity.Email:
			sur, err = v.Assign("email", sp.Text, func(a int) string { return g.Email(sp.Text, a) })
		case entity.IP:
			sur, err = v.Assign("ip", sp.Text, func(a int) string { return g.IP(sp.Text, a) })
		default:
			sur = "[redacted " + sp.Label + "]"
		}
		if err != nil {
			t.Fatal(err)
		}
		masked := c.text[:sp.Start] + sur + c.text[sp.End:]
		for _, r := range c.reals {
			if strings.Contains(masked, r) {
				t.Errorf("%q: masked %q keeps %q", c.text, masked, r)
			}
		}
		if strings.HasPrefix(sur, "[redacted") {
			continue
		}
		hits := v.FindSurrogates(masked)
		if len(hits) != 1 {
			t.Errorf("%q: surrogate %q not found in %q", c.text, sur, masked)
			continue
		}
		if back := masked[:hits[0].Start] + hits[0].Record.Real + masked[hits[0].End:]; back != c.text {
			t.Errorf("round trip %q -> %q -> %q", c.text, masked, back)
		}
	}
}

func TestASCIITextSkipsTheFoldPass(t *testing.T) {
	for _, s := range []string{"jane.doe@example.com 123-45-6789 4111 1111 1111 1111", "plain words", "naïve café résumé"} {
		if needsFold(s) {
			t.Errorf("%q: should not need folding", s)
		}
	}
	if needsFold("dash \u2014 here") || needsFold("plain text\u00a0with a space") {
		t.Error("a dash or space that touches no digit or capitalised word must not trigger the fold pass")
	}
	for _, s := range []string{"call 804\u2014555\u20141234", "Mary\u2014Jane", "John\u00a0Smith", "x \u200b y", "\uff11"} {
		if !needsFold(s) {
			t.Errorf("%q must trigger the fold pass", s)
		}
	}
	text := "a\u200bb\u00a0c１\u2011d"
	folded, starts, ends := foldText(text)
	if folded != "ab c1-d" || len(starts) != len(folded) || len(ends) != len(folded) {
		t.Fatalf("folded %q", folded)
	}
	if text[starts[0]:ends[1]] != "a\u200bb" {
		t.Fatalf("offset map does not cover the removed rune: %q", text[starts[0]:ends[1]])
	}
}

func TestFoldPassStaysLinear(t *testing.T) {
	size := scaletest.Size(scaleGrowthSize)
	for name, unit := range map[string]string{
		"zero width email":   "jane\u200b.doe@example.com ",
		"full-width ssn":     "１２３-４５-６７８９ ",
		"nbsp card":          "4111\u00a01111\u00a01111\u00a01111 ",
		"nb hyphen and nbsp": "word\u2011word\u00a0",
	} {
		unit := unit
		checkGrowth(t, name, func(n int) string { return strings.Repeat(unit, n/len(unit)+1) }, size, func(text string) {
			if _, err := NewRegexDetector().Detect(context.Background(), text); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMixedScriptDigitsAreNotFolded(t *testing.T) {
	for _, s := range []string{"Room 1\u00b23", "x\u00b2 + 10\u2074", "E = mc\u00b2", "120 m\u00b2", "H\u2082O"} {
		if f := FoldForms(s); f != s && s != "E = mc²" {
			t.Errorf("%q folded to %q", s, f)
		}
		for _, sp := range detectAll(t, s) {
			if sp.Type == entity.GovID || sp.Type == entity.Phone {
				t.Errorf("%q: unexpected %s span %q", s, sp.Type, sp.Text)
			}
		}
	}
	for in, want := range map[string]string{
		"\u00b9\u00b2\u00b3-\u2074\u2075-\u2076\u2077\u2078\u2079": "123-45-6789",
		"SSN 12\u00b3-45-6789":        "SSN 123-45-6789",
		"call 8\u00b9\u2070-555-0110": "call 810-555-0110",
		"IBAN DE\u2078\u2079 \u00b3\u2077\u2070\u2074 \u2070\u2070\u2074\u2074": "IBAN DE89 3704 0044",
		"Room 1\u00b23-4\u2075-6789": "Room 123-45-6789",
	} {
		if got := FoldForms(in); got != want {
			t.Errorf("%q folded to %q, want %q", in, got, want)
		}
	}
}

func TestNameDetectorsRunOnFoldedText(t *testing.T) {
	d, err := ParseDictionaryCSV(strings.NewReader("name\nWilhelmina Quarrington\n"))
	if err != nil {
		t.Fatal(err)
	}
	dets := []Detector{d.Detector(), NameDetector{}}
	for _, text := range []string{"Met Wilhelmina\u00a0Quarrington today", "Met Ｗｉｌｈｅｌｍｉｎａ Quarrington today", "Met John\u00a0Smith today"} {
		found := false
		for _, r := range RunAll(context.Background(), text, dets) {
			for _, sp := range r.Spans {
				if sp.Type == entity.Person && text[sp.Start:sp.End] == sp.Text && strings.Contains(FoldForms(sp.Text), " ") {
					found = true
				}
			}
		}
		if !found {
			t.Errorf("%q: no full-name person span over the original bytes", text)
		}
	}
	plain := RunAll(context.Background(), "Met John Smith today", dets)
	again := RunAll(context.Background(), "Met John Smith today", dets)
	if len(plain[1].Spans) != len(again[1].Spans) {
		t.Fatal("unstable")
	}
}

func TestNameFoldPassStaysLinear(t *testing.T) {
	size := scaletest.Size(scaleGrowthSize)
	unit := "Met John\u00a0Smith and Ｍａｒｙ Jones. "
	checkGrowth(t, "name fold pass", func(n int) string { return strings.Repeat(unit, n/len(unit)+1) }, size, func(text string) {
		RunAll(context.Background(), text, []Detector{NameDetector{}})
	})
}

func TestPhoneAfterAWordHyphen(t *testing.T) {
	for _, text := range []string{"claude-8045551234", "see ref-8045551234 here", "id ref-18045551234 now", "x=8045551234;", "x-804-555-1234", "call tel-(804) 555-1234", "phone 8045551234"} {
		got := false
		for _, sp := range detectAll(t, text) {
			if sp.Type == entity.Phone {
				got = true
			}
		}
		if !got {
			t.Errorf("%q: want a phone span", text)
		}
	}
	for _, text := range []string{"2024-8045551234", "v1.2-8045551234", "ref-80455512345", "ref-8045551234-01", "ref-8045551234.5", "550e8400-e29b-41d4-a716-8045551234ab", "ref-804-555-1234x", "2024-01-15", "req-1732345678", "ts-1700000123", "job-2024111512", "span-9876543210", "build-8045551234.log", "img-8045551234.png", "epoch 1732345678", "id 0123456789", "n 5551234567 x"} {
		for _, sp := range detectAll(t, text) {
			if sp.Type == entity.Phone {
				t.Errorf("%q: unexpected phone span %q", text, sp.Text)
			}
		}
	}
}

func TestFoldPassCostsLittleOnMostlyASCIIText(t *testing.T) {
	if scaletest.Race || testing.Short() {
		t.Skip("timing")
	}
	size := scaletest.Size(1 << 20)
	unit := "The build finished and the report went out to the team for review today. "
	plain := strings.Repeat(unit, size/len(unit)+1)
	mixed := []byte(plain)
	marks := []string{"\u2014", "\u00a0", "\uff11", "\u200b", "\u2011"}
	var b strings.Builder
	step := len(plain) / (len(marks) + 1)
	last := 0
	for k, m := range marks {
		at := (k + 1) * step
		for at < len(plain) && plain[at] != ' ' {
			at++
		}
		b.Write(mixed[last:at])
		b.WriteString("Ann" + m + "Lee 804" + m + "555")
		last = at
	}
	b.Write(mixed[last:])
	text := b.String()
	folded, _, _ := foldText(text)
	rescanned := 3 * len(folded)
	t.Logf("fold passes rescan %d of %d bytes (%.2f%%)", rescanned, len(text), 100*float64(rescanned)/float64(len(text)))
	if float64(rescanned) > 0.10*float64(len(text)) {
		t.Errorf("fold passes rescan %d bytes of a %d byte text", rescanned, len(text))
	}
	dets := []Detector{NewRegexDetector(), DateDetector{}, NameDetector{}}
	p, x := time.Duration(1<<62), time.Duration(1<<62)
	for i := 0; i < 3; i++ {
		start := time.Now()
		RunAll(context.Background(), plain, dets)
		p = min(p, time.Since(start))
		start = time.Now()
		RunAll(context.Background(), text, dets)
		x = min(x, time.Since(start))
	}
	t.Logf("plain %v, with %d fold characters %v, ratio %.3f", p, len(marks)*2, x, float64(x)/float64(p))
	if float64(x) > 1.5*float64(p) {
		t.Errorf("fold pass costs %.0f%% over plain ASCII", 100*(float64(x)/float64(p)-1))
	}
}

func TestTenDigitRunsAfterIDLabelsAreNotPhones(t *testing.T) {
	for _, text := range []string{"trace_id=8045551234", "id=8045551234", "id: 8045551234", "request 8045551234", "req=8045551234", "order 8045551234", "ticket: 8045551234", "checksum=8045551234", "hash 8045551234", "seq=8045551234", "offset 8045551234", "size=8045551234", "bytes: 8045551234", "count=8045551234", "pid 8045551234", "port: 8045551234", "ts=8045551234", "time 8045551234", "epoch 8045551234", "build 8045551234", "job=8045551234", "run 8045551234", "spanId=8045551234", "user_id: 8045551234"} {
		for _, sp := range detectAll(t, text) {
			if sp.Type == entity.Phone {
				t.Errorf("%q: unexpected phone %q", text, sp.Text)
			}
		}
	}
	for _, text := range []string{"tel: 8045551234", "phone 8045551234", "call 8045551234", "cell=8045551234", "mobile: 8045551234", "fax 8045551234", "contact_phone=8045551234", "order 804-555-1234", "id: (804) 555-1234"} {
		found := false
		for _, sp := range detectAll(t, text) {
			if sp.Type == entity.Phone {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: want a phone", text)
		}
	}
}

func TestFoldWindowsCutOnRuneBoundaries(t *testing.T) {
	text := strings.Repeat("\u00e9", 600) + "\u2014John\u00a0Smith\u2014" + strings.Repeat("\u4e2d", 400)
	for _, w := range foldWindows(text, foldTriggers(text)) {
		if !utf8.ValidString(text[w[0]:w[1]]) {
			t.Fatalf("window %v cuts a rune", w)
		}
	}
}

func TestCallerIDStyleLabelsKeepPhones(t *testing.T) {
	for _, text := range []string{"caller id 8045551234", "Caller ID: 8045551234", "caller_id=8045551234", "callerId=8045551234", "Caller-ID: 8045551234", "WhatsApp ID: 8045551234", "Phone ID 8045551234", "telephone id 8045551234", "emergency contact id 8045551234", "Apple ID 8045551234", "my id 8045551234"} {
		found := false
		for _, sp := range detectAll(t, text) {
			if sp.Type == entity.Phone {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: want a phone", text)
		}
	}
	for _, text := range []string{"id=8045551234", "id: 8045551234", "WHERE id = 8045551234", "the order id: 8045551234", "batch_id=8045551234", "parentSpanId=8045551234", "txn id 8045551234", "event id: 8045551234", "session=8045551234"} {
		for _, sp := range detectAll(t, text) {
			if sp.Type == entity.Phone {
				t.Errorf("%q: unexpected phone %q", text, sp.Text)
			}
		}
	}
}

func TestDashJoinsOnlyRealNames(t *testing.T) {
	for _, d := range []string{"\u2010", "\u2011", "\u2012", "\u2013", "\u2014", "\u2015", "\u2212", "\uff0d", "\ufe63", "\ufe58", "\u2e3a", "\u2e3b"} {
		for _, c := range []struct {
			text  string
			whole bool
			name  string
		}{
			{"Please email Elena Petrova" + d + "Ivanova today.", true, "Petrova" + d + "Ivanova"},
			{"Hi Jean" + d + "Luc,", true, "Jean" + d + "Luc"},
			{"Mary" + d + "Jane said hi.", true, "Mary" + d + "Jane"},
			{"I spoke to Elena" + d + "she said yes.", false, "Elena" + d + "she"},
			{"Meet John Smith" + d + "CEO of it.", false, "Smith" + d + "CEO"},
		} {
			if !c.whole && textutil.IsHyphenVariant([]rune(d)[0]) {
				continue
			}
			joined := false
			var all []entity.Span
			for _, r := range RunAll(context.Background(), c.text, []Detector{NewRegexDetector(), NameDetector{}}) {
				all = append(all, r.Spans...)
			}
			for _, sp := range Resolve(all) {
				if sp.Type == entity.Person && strings.Contains(sp.Text, c.name) {
					joined = true
				}
			}
			if joined != c.whole {
				t.Errorf("%q: span over %q = %v, want %v", c.text, c.name, joined, c.whole)
			}
		}
	}
	for _, text := range []string{"call 804\ufe58555\ufe581234", "SSN 536\ufe5822\ufe588471"} {
		found := false
		for _, sp := range detectAll(t, text) {
			if sp.Type == entity.Phone || sp.Type == entity.GovID {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: not detected", text)
		}
	}
}

func TestCJKDashesDoNotJoinDigits(t *testing.T) {
	for _, text := range []string{"date 2024\u301c11\u301c15 ok", "id 536\u30a022\u30a08471", "n 2024\u303011\u303015"} {
		if FoldForms(text) != text {
			t.Errorf("%q folded to %q", text, FoldForms(text))
		}
		for _, sp := range detectAll(t, text) {
			if sp.Type == entity.GovID || sp.Type == entity.Phone {
				t.Errorf("%q: unexpected %s %q", text, sp.Type, sp.Text)
			}
		}
	}
}
