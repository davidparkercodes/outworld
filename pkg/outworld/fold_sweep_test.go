package outworld

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/davidparkercodes/outworld/internal/scaletest"
	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/detect"
	"github.com/davidparkercodes/outworld/pkg/entity"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

var foldSweep = []struct {
	text  string
	reals []string
}{
	{"phone ５５５-１２３-４５６７ today", []string{"１２３-４５６７", "123-4567", "4567"}},
	{"call (５５５) １２３-４５６７ now", []string{"１２３-４５６７", "123-4567", "4567"}},
	{"tel ０９０-１２３４-５６７８ please", []string{"１２３４-５６７８", "1234-5678", "5678"}},
	{"phone ８０４５５５１２３４ today", []string{"８０４５５５１２３４", "8045551234", "1234"}},
	{"call +４４ ２０ ７９４６ ０９５８ now", []string{"７９４６ ０９５８", "7946 0958", "0958"}},
	{"SSN １２３-４５-６７８９ on file", []string{"６７８９", "6789"}},
	{"card ４１１１ １１１１ １１１１ １１１１ exp", []string{"１１１１ １１１１", "1111 1111"}},
	{"IBAN ＤＥ８９ ３７０４ ００４４ ０５３２ ０１３０ ００ ok", []string{"０５３２ ０１３０", "0532 0130", "DE89"}},
	{"host １０.１.２.３ is up", []string{"１０.１.２.３", "10.1.2.3"}},
	{"mail jane\u200b.doe@example.com today", []string{"jane", "doe@example"}},
	{"on ２０２５-０３-０３ we met", []string{"２０２５-０３-０３", "2025-03-03"}},
	{"due ０３/１５/２０２４ latest", []string{"０３/１５/２０２４", "03/15/2024"}},
	{"Lieferung an Hauptstraße ５, １０１１５ Berlin bitte", []string{"１０１１５", "10115", "Berlin", "Hauptstraße"}},
	{"ship to 742 Evergreen Terrace, Springfield, IL ６２７０４ now", []string{"６２７０４", "62704", "Evergreen", "Springfield"}},
}

func TestFullWidthFormsMaskAndRestoreThroughAFileVault(t *testing.T) {
	for i, c := range foldSweep {
		path := filepath.Join(t.TempDir(), "v.vault")
		v, err := openTestVault(path, bytes.Repeat([]byte{byte(3 + i)}, 32))
		if err != nil {
			t.Fatal(err)
		}
		e, err := New(Options{Vault: v, Policy: DefaultPolicy()})
		if err != nil {
			t.Fatal(err)
		}
		s := e.NewSession("fold")
		masked, err := s.Mask(context.Background(), c.text)
		if err != nil {
			t.Errorf("%q: mask failed: %v", c.text, err)
			continue
		}
		if lerr := s.Err(); lerr != nil {
			t.Errorf("%q: blocked: %v", c.text, lerr)
			continue
		}
		folded := detect.FoldForms(masked)
		for _, r := range c.reals {
			if strings.Contains(masked, r) || strings.Contains(folded, detect.FoldForms(r)) {
				t.Errorf("%q: masked %q keeps %q", c.text, masked, r)
			}
		}
		if strings.Contains(masked, "[redacted") {
			continue
		}
		if back := s.Rehydrate(masked); back != c.text {
			t.Errorf("%q: masked %q rehydrates to %q", c.text, masked, back)
		}
		v.Close()
	}
}

func TestFullWidthPhonesGetDistinctRecords(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("fold")
	a, err := s.Mask(context.Background(), "phone ５５５-１２３-４５６７")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Mask(context.Background(), "phone ５５５-９８７-６５４３")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatalf("two different full-width phones share a surrogate: %q", a)
	}
	if got := s.Rehydrate(a + " / " + b); got != "phone ５５５-１２３-４５６７ / phone ５５５-９８７-６５４３" {
		t.Fatalf("rehydrate: %q", got)
	}
}

func TestStyledDigitPhonesKeepTheirOwnRecords(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("fold")
	mask := func(text string) string {
		t.Helper()
		m, err := s.Mask(context.Background(), text)
		if err != nil || s.Err() != nil {
			t.Fatalf("%q: %v %v", text, err, s.Err())
		}
		return m
	}
	ascii804, ascii900 := mask("call 804-555-1234 now"), mask("call 900-555-9999 now")
	cases := map[string]string{
		"call \U0001D7D6\U0001D7CE\U0001D7D2-\U0001D7D3\U0001D7D3\U0001D7D3-\U0001D7CF\U0001D7D0\U0001D7D1\U0001D7D2 now": ascii804,
		"call \U0001D7D7\U0001D7CE\U0001D7CE-\U0001D7D3\U0001D7D3\U0001D7D3-\U0001D7D7\U0001D7D7\U0001D7D7\U0001D7D7 now": ascii900,
		"call ⁸⁰⁴-⁵⁵⁵-¹²³⁴ now": ascii804,
		"call ⁹⁰⁰-⁵⁵⁵-⁹⁹⁹⁹ now": ascii900,
	}
	if ascii804 == ascii900 {
		t.Fatal("two numbers share a surrogate")
	}
	for text, want := range cases {
		got := mask(text)
		if got == want || detect.FoldForms(got) != want {
			t.Errorf("%q: masked %q, want the ASCII form's digits %q in the input's own digit style", text, got, want)
		}
		if back := s.Rehydrate(got); back != text {
			t.Errorf("%q: rehydrated to %q", text, back)
		}
	}
	for _, text := range []string{"call 804-555-1234 now", "call 900-555-9999 now"} {
		if got := mask(text); s.Rehydrate(got) != text {
			t.Errorf("%q after the styled forms: masked %q restores %q", text, got, s.Rehydrate(got))
		}
	}
}

func TestFoldedNamesAndEmailsAreMasked(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("fold")
	plain, err := s.Mask(context.Background(), "Met John Smith today")
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Met John\u00a0Smith today", "Met Ｊｏｈｎ Ｓｍｉｔｈ today", "Met Jo\u200bhn Smith today"} {
		m, err := s.Mask(context.Background(), text)
		if err != nil || s.Err() != nil {
			t.Fatalf("%q: %v %v", text, err, s.Err())
		}
		f := detect.FoldForms(m)
		if strings.Contains(f, "John") || strings.Contains(f, "Smith") || strings.Contains(f, "Jo") && strings.Contains(f, "hn Smith") {
			t.Errorf("%q: masked %q keeps the name", text, m)
		}
		if back := s.Rehydrate(m); detect.FoldForms(back) != detect.FoldForms(text) {
			t.Errorf("%q: rehydrates to %q", text, back)
		}
	}
	nbsp, _ := s.Mask(context.Background(), "Met John\u00a0Smith today")
	if strings.ReplaceAll(nbsp, "\u00a0", " ") != plain {
		t.Errorf("NBSP name %q should use the plain name's stand-in %q", nbsp, plain)
	}
	a, _ := s.Mask(context.Background(), "mail jane.roe@example.com")
	b, _ := s.Mask(context.Background(), "mail ｊａｎｅ.ｒｏｅ@example.com")
	if a != b {
		t.Errorf("full-width email %q should share the ASCII stand-in %q", b, a)
	}
}

func inScript(s string, zero rune) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(zero + r - '0')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TestOtherScriptDigitsMaskAndRestore(t *testing.T) {
	scripts := map[string]rune{"arabic-indic": 0x0660, "extended arabic": 0x06F0, "devanagari": 0x0966, "bengali": 0x09E6, "thai": 0x0E50, "math bold": 0x1D7CE}
	templates := []struct {
		text, real string
		restore    bool
	}{
		{"call 804-555-0110 now", "804-555-0110", true},
		{"SSN 123-45-6789 on file.", "123-45-6789", false},
		{"card 4111 1111 1111 1111 ok", "4111 1111 1111 1111", false},
		{"host 10.1.2.3 up", "10.1.2.3", true},
	}
	for name, zero := range scripts {
		for _, c := range templates {
			e := newEngine(t, nil)
			s := e.NewSession("fold")
			text, real := inScript(c.text, zero), inScript(c.real, zero)
			m, err := s.Mask(context.Background(), text)
			if err != nil || s.Err() != nil {
				t.Errorf("%s %q: %v %v", name, text, err, s.Err())
				continue
			}
			if strings.Contains(m, real) || strings.Contains(detect.FoldForms(m), c.real) {
				t.Errorf("%s %q: masked %q keeps the value", name, text, m)
			}
			if c.restore {
				if back := s.Rehydrate(m); back != text {
					t.Errorf("%s %q: rehydrates to %q", name, text, back)
				}
			}
		}
	}
	e := newEngine(t, nil)
	s := e.NewSession("fold")
	a, _ := s.Mask(context.Background(), "call ٨١٠-٥٥٥-٠١١٠")
	b, _ := s.Mask(context.Background(), "call ٨١٠ ٥٥٥ ٠١١٠")
	if detect.FoldForms(a)[5:] == "" || digitsOf(a) != digitsOf(b) {
		t.Errorf("two spellings of one Arabic-Indic number got different stand-ins: %q %q", a, b)
	}
}

func TestStyledLettersAndDashesRestoreTheirOwnBytes(t *testing.T) {
	cases := []struct{ text, real string }{
		{"Please email \U0001D404\U0001D425\U0001D41E\U0001D427\U0001D41A \U0001D40F\U0001D41E\U0001D42D\U0001D42B\U0001D428\U0001D42F\U0001D41A about the contract.", "Elena"},
		{"call 804\u2014555\u20141234 now", "1234"},
		{"Mary\u2014Jane said hi.", "Mary"},
	}
	for _, c := range cases {
		e := newEngine(t, nil)
		s := e.NewSession("fold")
		m, err := s.Mask(context.Background(), c.text)
		if err != nil || s.Err() != nil {
			t.Fatalf("%q: %v %v", c.text, err, s.Err())
		}
		if strings.Contains(detect.FoldForms(m), c.real) {
			t.Errorf("%q: masked %q keeps %q", c.text, m, c.real)
		}
		back := s.Rehydrate(m)
		if strings.Contains(c.text, "\U0001D404") {
			if detect.FoldForms(back) != detect.FoldForms(c.text) {
				t.Errorf("%q: rehydrates to %q", c.text, back)
			}
			continue
		}
		if back != c.text {
			t.Errorf("%q: rehydrates to %q, want the original bytes", c.text, back)
		}
	}
}

type emptySpanDetector struct{}

func (emptySpanDetector) Name() string { return "empty-span" }

func (emptySpanDetector) Detect(_ context.Context, text string) ([]entity.Span, error) {
	i := strings.Index(text, "​⁠")
	if i < 0 {
		return nil, nil
	}
	return []entity.Span{{Start: i, End: i + len("​⁠"), Type: entity.Email, Label: "email", Text: "​⁠", Source: "test", Priority: 99, Score: 1}}, nil
}

func TestEmptyKeySpanFailsClosed(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Extra = []detect.Detector{emptySpanDetector{}} })
	s := e.NewSession("fold")
	_, err := s.Mask(context.Background(), "hello ​⁠ there")
	if !errors.Is(err, vault.ErrEmptyReal) {
		t.Fatalf("want ErrEmptyReal, got %v", err)
	}
}

func TestEmailsAndPhonesInsideURLsMaskAndRestore(t *testing.T) {
	cases := []struct{ text, real, keep string }{
		{"Look at https://example.com/?email=jane.doe@example.com please", "jane.doe", "https://example.com/?email="},
		{"see https://mcp.example.com/jane.doe@example.com now", "jane.doe", "https://mcp.example.com/"},
		{"call https://example.com/u/555-867-5309 now", "867-5309", "https://example.com/u/"},
	}
	for _, c := range cases {
		e := newEngine(t, nil)
		s := e.NewSession("url")
		m, err := s.Mask(context.Background(), c.text)
		if err != nil || s.Err() != nil {
			t.Fatalf("%q: %v %v", c.text, err, s.Err())
		}
		if strings.Contains(m, c.real) || !strings.Contains(m, c.keep) || strings.Contains(m, "[redacted") {
			t.Errorf("%q: masked %q", c.text, m)
		}
		if back := s.Rehydrate(m); back != c.text {
			t.Errorf("%q: rehydrates to %q", c.text, back)
		}
	}
	e := newEngine(t, nil)
	s := e.NewSession("url")
	for _, text := range []string{"https://user@example.com/path", "https://example.com/items/123-456"} {
		if m, _ := s.Mask(context.Background(), text); m != text {
			t.Errorf("%q changed to %q", text, m)
		}
	}
}

func TestDashJoinedNamesMaskWholeAndRestore(t *testing.T) {
	for _, d := range []string{"\u2010", "\u2011", "\u2013", "\u2014", "\u2212", "\uff0d", "\ufe63", "-"} {
		cases := []struct {
			text  string
			reals []string
		}{
			{"Please email Elena Petrova" + d + "Ivanova today.", []string{"Elena", "Petrova", "Ivanova"}},
			{"Hi Jean" + d + "Luc,", []string{"Jean", "Luc"}},
			{"Mary" + d + "Jane said hi.", []string{"Mary", "Jane"}},
		}
		for _, c := range cases {
			e := newEngine(t, nil)
			s := e.NewSession("dash")
			m, err := s.Mask(context.Background(), c.text)
			if err != nil || s.Err() != nil {
				t.Fatalf("%q: %v %v", c.text, err, s.Err())
			}
			for _, r := range c.reals {
				if strings.Contains(m, r) {
					t.Errorf("%q: masked %q keeps %q", c.text, m, r)
				}
			}
			if back := s.Rehydrate(m); back != c.text {
				t.Errorf("%q: rehydrates to %q", c.text, back)
			}
		}
	}
}

func TestFancyNamesAndEncodedEmailsMask(t *testing.T) {
	cases := []struct {
		text, real string
		exact      bool
	}{
		{"Please email \u2130\U0001D4C1\u212F\U0001D4C3\U0001D4B6 \U0001D4AB\u212F\U0001D4C9\U0001D4C7\u2134\U0001D4CB\U0001D4B6 about it.", "Elena", false},
		{"Please email \U0001D53C\U0001D55D\U0001D556\U0001D55F\U0001D552 \u2119\U0001D556\U0001D565\U0001D563\U0001D560\U0001D567\U0001D552 about it.", "Petrova", false},
		{"Met \U0001D43D\U0001D45C\u210E\U0001D45B \U0001D446\U0001D45A\U0001D456\U0001D461\u210E today.", "Smith", false},
		{"see https://example.com/?email=elena.petrova%40example.com now", "elena.petrova", true},
	}
	for _, c := range cases {
		e := newEngine(t, nil)
		s := e.NewSession("fancy")
		m, err := s.Mask(context.Background(), c.text)
		if err != nil || s.Err() != nil {
			t.Fatalf("%q: %v %v", c.text, err, s.Err())
		}
		if strings.Contains(detect.FoldForms(m), c.real) {
			t.Errorf("%q: masked %q keeps %q", c.text, m, c.real)
		}
		back := s.Rehydrate(m)
		if c.exact && back != c.text || !c.exact && detect.FoldForms(back) != detect.FoldForms(c.text) {
			t.Errorf("%q: rehydrates to %q", c.text, back)
		}
		if c.exact && !strings.Contains(m, "%40") {
			t.Errorf("%q: stand-in %q is not percent-encoded", c.text, m)
		}
	}
}

func TestEmailSurrogatesAreClean(t *testing.T) {
	e := newEngine(t, nil)
	for _, real := range []string{"12345@example.com", "a@example.com", "x1@example.com", "jane+work@example.com", "jane.doe.smith@example.co.uk", "j@example.museum", "007@example.com", "jane_doe-99@sub.example.travel", "elena%2Bwork%40example.com", "12345%40example.com", "elena.petrova%2540example.com"} {
		plain := real
		if p, _, ok := detect.EncodedEmail(real); ok {
			plain = p
		}
		for attempt := 0; attempt < 8; attempt++ {
			sur := e.emailCandidate(plain, attempt, nil)
			if strings.IndexFunc(sur, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 || strings.Count(sur, "@") != 1 {
				t.Errorf("%q attempt %d: bad surrogate %q", real, attempt, sur)
			}
		}
	}
	s := e.NewSession("mail")
	for _, text := range []string{"mail 12345@example.com now", "go https://x.example/?e=12345%40example.com now", "go https://x.example/?e=elena%2Bwork%40example.com now", "go https://x.example/?e=elena.petrova%2540example.com now"} {
		m, err := s.Mask(context.Background(), text)
		if err != nil || s.Err() != nil {
			t.Fatalf("%q: %v %v", text, err, s.Err())
		}
		if strings.IndexFunc(m, func(r rune) bool { return r < 0x20 }) >= 0 || strings.Contains(m, "12345") || strings.Contains(m, "elena") {
			t.Errorf("%q: masked %q", text, m)
		}
		if back := s.Rehydrate(m); back != text {
			t.Errorf("%q: rehydrates to %q", text, back)
		}
	}
}

func TestHyphenClassJoinsLikeASCII(t *testing.T) {
	hyphens := []string{"\u2010", "\u2011", "\ufe63", "\uff0d", "\u2212"}
	for _, base := range []string{"Please email Elena Petrova-Quetzal today.", "Ask Priya Raman-Thistlewood now.", "Please email Elena Petrova-Ivanova today.", "Mary-Jane said hi.", "Hi Jean-Luc,"} {
		e := newEngine(t, nil)
		want, err := e.NewSession("h").Mask(context.Background(), base)
		if err != nil {
			t.Fatal(err)
		}
		if want == base {
			t.Fatalf("%q: the ASCII form must mask", base)
		}
		for _, h := range hyphens {
			e := newEngine(t, nil)
			s := e.NewSession("h")
			text := strings.ReplaceAll(base, "-", h)
			got, err := s.Mask(context.Background(), text)
			if err != nil || s.Err() != nil {
				t.Fatalf("%q: %v %v", text, err, s.Err())
			}
			if strings.ReplaceAll(got, h, "-") != want {
				t.Errorf("%q: %q, ASCII gives %q", text, got, want)
			}
			if back := s.Rehydrate(got); back != text {
				t.Errorf("%q: rehydrates to %q", text, back)
			}
		}
	}
	for _, base := range []string{"Please email Jean-Luc Picard today.", "Ask Siobhan-Nakamura now."} {
		for _, h := range hyphens {
			e := newEngine(t, nil)
			text := strings.ReplaceAll(base, "-", h)
			got, _ := e.NewSession("h").Mask(context.Background(), text)
			first := strings.Fields(strings.Split(text, h)[0])
			if strings.Contains(got, first[len(first)-1]+h) {
				t.Errorf("%q: masked %q keeps the first half", text, got)
			}
		}
	}
}

func TestUnicodeEmailsAndMultipleEncodedAddresses(t *testing.T) {
	cases := []struct{ text, real string }{
		{"mail \u00e9lena@example.com now", "lena@"},
		{"mail jos\u00e9.garc\u00eda@example.es now", "garc"},
		{"mail \u0435\u043b\u0435\u043d\u0430@\u043f\u0440\u0438\u043c\u0435\u0440.\u0440\u0444 now", "\u0435\u043b\u0435\u043d\u0430"},
		{"go https://x.example/?to=elena%40example.com%20john%40example.com now", "john"},
	}
	for _, c := range cases {
		e := newEngine(t, nil)
		s := e.NewSession("mail")
		m, err := s.Mask(context.Background(), c.text)
		if err != nil || s.Err() != nil {
			t.Fatalf("%q: %v %v", c.text, err, s.Err())
		}
		if strings.Contains(m, c.real) || strings.Contains(m, "elena") {
			t.Errorf("%q: masked %q", c.text, m)
		}
		for _, f := range strings.Fields(m) {
			if strings.Contains(f, "@") || strings.Contains(f, "%40") {
				for _, r := range f {
					if r > 0x7e {
						t.Errorf("%q: stand-in %q is not ASCII", c.text, f)
						break
					}
				}
			}
		}
		if back := s.Rehydrate(m); back != c.text {
			t.Errorf("%q: rehydrates to %q", c.text, back)
		}
	}
}

func TestAzureAPIVersionURLsRoundTrip(t *testing.T) {
	for _, text := range []string{
		"POST https://x.openai.azure.com/openai/deployments/gpt/chat?api-version=2024-02-01 now",
		"POST https://x.openai.azure.com/openai/deployments/gpt/chat/completions?api-version=2024-02-15-preview now",
	} {
		e := newEngine(t, nil)
		s := e.NewSession("azure")
		m, err := s.Mask(context.Background(), text)
		if err != nil || s.Err() != nil {
			t.Fatalf("%q: %v %v", text, err, s.Err())
		}
		if m != text {
			t.Errorf("%q: masked %q, want it unchanged", text, m)
		}
		if back := s.Rehydrate(m); back != text {
			t.Errorf("%q: masked %q rehydrates to %q", text, m, back)
		}
	}
}

func TestCueSeparatedPhonesRestoreTheirSeparators(t *testing.T) {
	for _, sep := range []string{"〜", "~", "*", "·", "/", "゠"} {
		text := "Call 804" + sep + "555" + sep + "1234 today"
		e := newEngine(t, nil)
		s := e.NewSession("sep")
		m, err := s.Mask(context.Background(), text)
		if err != nil || s.Err() != nil {
			t.Fatalf("%q: %v %v", text, err, s.Err())
		}
		if strings.Contains(m, "1234") || strings.Contains(m, "804") {
			t.Errorf("%q: masked %q", text, m)
		}
		if back := s.Rehydrate(m); back != text {
			t.Errorf("%q: masked %q rehydrates to %q", text, m, back)
		}
		ssn := "SSN 536" + sep + "22" + sep + "8471 on file"
		m, err = s.Mask(context.Background(), ssn)
		if err != nil || s.Err() != nil || strings.Contains(m, "8471") {
			t.Errorf("%q: masked %q %v %v", ssn, m, err, s.Err())
		}
	}
	e := newEngine(t, nil)
	s := e.NewSession("q")
	text := "auth_method=password&q=hello"
	if m, _ := s.Mask(context.Background(), text); m != text {
		t.Errorf("%q changed to %q", text, m)
	}
	text = "see https://example.com/login?auth_method=password&q=hello now"
	if m, _ := s.Mask(context.Background(), text); m != text {
		t.Errorf("%q changed to %q", text, m)
	}
}

var phoneMatrixSeps = []string{" ", "-", ".", "~", "〜", "〰", "゠", "/", "*", "·", "・", "•", "_", "|", ":", "\u2212", "\u2014", "\u2013", "\u2010", "\u2011", "\ufe58", "～", "\uff0f", "\uff0e", "\u3000", "\u00a0"}

func TestPhoneSeparatorMatrixRestoresEachOccurrence(t *testing.T) {
	failures := 0
	for _, a := range phoneMatrixSeps {
		for _, b := range phoneMatrixSeps {
			e := newEngine(t, nil)
			s := e.NewSession("matrix")
			for _, sep := range []string{a, b} {
				text := "Call 804" + sep + "555" + sep + "1234 now"
				m, err := s.Mask(context.Background(), text)
				if err != nil || s.Err() != nil {
					t.Fatalf("%q: %v %v", text, err, s.Err())
				}
				if m == text {
					failures++
					t.Errorf("first %q then %q: %q was not masked", a, b, text)
					continue
				}
				if back := s.Rehydrate(m); back != text {
					failures++
					t.Errorf("first %q then %q: %q masked %q restores %q", a, b, text, m, back)
				}
			}
		}
	}
	t.Logf("%d separators, %d pairs, %d failures", len(phoneMatrixSeps), len(phoneMatrixSeps)*len(phoneMatrixSeps), failures)
}

func TestMaskedPhonesAreAlwaysRestorable(t *testing.T) {
	for _, text := range []string{"Call ext804-555-1234 now", "Call a804-555-1234 now", "Contact x804*555*1234 now", "Call ext804〜555〜1234", "Call 804〜555〜1234x", "Call 804*555*1234 now", "SSN ref 536~22~8471"} {
		e := newEngine(t, nil)
		s := e.NewSession("glued")
		m, err := s.Mask(context.Background(), text)
		if err != nil || s.Err() != nil {
			t.Fatalf("%q: %v %v", text, err, s.Err())
		}
		if strings.Contains(m, "[redacted") {
			continue
		}
		if back := s.Rehydrate(m); back != text {
			t.Errorf("%q: masked %q but restores %q", text, m, back)
		}
	}
}

func generatedPhoneSeparators() []rune {
	seen := map[rune]bool{}
	var out []rune
	add := func(r rune) {
		if !seen[r] && r != '\n' && r != '\r' {
			seen[r] = true
			out = append(out, r)
		}
	}
	for _, r := range textutil.PhoneSeparators() {
		add(r)
	}
	for _, rg := range unicode.Pd.R16 {
		for r := rune(rg.Lo); r <= rune(rg.Hi); r += rune(rg.Stride) {
			add(r)
		}
	}
	for _, rg := range unicode.Pd.R32 {
		for r := rune(rg.Lo); r <= rune(rg.Hi); r += rune(rg.Stride) {
			add(r)
		}
	}
	for r := rune(0x80); r <= 0xFFFF; r++ {
		if f := textutil.FoldForms(string(r)); f == " " || len(f) == 1 && strings.Contains(" .-~/*_|:", f) {
			add(r)
		}
	}
	return out
}

var generatedSeparatorSkips = map[rune]string{}

func TestGeneratedPhoneSeparatorsRestore(t *testing.T) {
	seps := generatedPhoneSeparators()
	forms := 0
	for _, r := range seps {
		if why, skip := generatedSeparatorSkips[r]; skip {
			t.Logf("skip %U: %s", r, why)
			continue
		}
		sep := string(r)
		for _, text := range []string{"Call 804" + sep + "555" + sep + "1234 now", "Call (804)" + sep + "555" + sep + "1234 now", "Call +1" + sep + "804" + sep + "555" + sep + "1234 now"} {
			forms++
			e := newEngine(t, nil)
			s := e.NewSession("gen")
			m, err := s.Mask(context.Background(), text)
			if err != nil || s.Err() != nil {
				t.Fatalf("%U %q: %v %v", r, text, err, s.Err())
			}
			if strings.Contains(m, "1234") || m == text {
				t.Errorf("%U %q: not masked (%q)", r, text, m)
				continue
			}
			if back := s.Rehydrate(m); back != text {
				t.Errorf("%U %q: masked %q restores %q", r, text, m, back)
			}
		}
		if textutil.FoldForms(sep) == " " {
			text := "notes 804" + sep + "555" + sep + "1234 end"
			e := newEngine(t, nil)
			s := e.NewSession("space")
			m, _ := s.Mask(context.Background(), text)
			ascii, _ := newEngine(t, nil).NewSession("space").Mask(context.Background(), "notes 804 555 1234 end")
			if (m == text) != (ascii == "notes 804 555 1234 end") {
				t.Errorf("%U without a cue: %q vs ASCII space %q", r, m, ascii)
			}
			if m != text {
				if back := s.Rehydrate(m); back != text {
					t.Errorf("%U without a cue: restores %q", r, back)
				}
			}
		}
	}
	t.Logf("%d separator runes, %d forms, %d skipped", len(seps), forms, len(generatedSeparatorSkips))
	if len(seps) < 60 {
		t.Errorf("only %d separator runes generated", len(seps))
	}
}

func TestPlusOneBracketPhonesWithACue(t *testing.T) {
	for _, text := range []string{"Call +1(804)555/1234 now", "Call \uff0b1\uff08804\uff09555\uff0f1234 now", "phone: 1 (804)-555-1234", "Call +1 (804)\u301c555\u301c1234 now"} {
		e := newEngine(t, nil)
		s := e.NewSession("plus")
		m, err := s.Mask(context.Background(), text)
		if err != nil || s.Err() != nil {
			t.Fatalf("%q: %v %v", text, err, s.Err())
		}
		if strings.Contains(m, "1234") {
			t.Errorf("%q: masked %q keeps the number", text, m)
		}
		if back := s.Rehydrate(m); back != text {
			t.Errorf("%q: masked %q restores %q", text, m, back)
		}
	}
}

func TestFullWidthBracketAndPlusPhonesRestore(t *testing.T) {
	for _, text := range []string{
		"Call \uff08804\uff09 555-1234 now",
		"Call \uff08804\uff09\u3000555\uff0d1234 now",
		"Call \uff0b1 804 555 1234 now",
		"Call \uff0b1\uff08804\uff09555\uff0d1234 now",
		"Call +1 (804) 555-1234 now",
		"Call (804)\u00a0555-1234 now",
	} {
		e := newEngine(t, nil)
		s := e.NewSession("fw")
		m, err := s.Mask(context.Background(), text)
		if err != nil || s.Err() != nil {
			t.Fatalf("%q: %v %v", text, err, s.Err())
		}
		if strings.Contains(m, "1234") {
			t.Errorf("%q: masked %q keeps the number", text, m)
		}
		if back := s.Rehydrate(m); back != text {
			t.Errorf("%q: masked %q restores %q", text, m, back)
		}
	}
}

func TestKnownPhoneFormsMaskWithoutACue(t *testing.T) {
	text := "Call 804-555-1234, or 804\u301c555\u301c1234, or 804_555_1234 today."
	e := newEngine(t, nil)
	s := e.NewSession("known")
	m, err := s.Mask(context.Background(), text)
	if err != nil || s.Err() != nil {
		t.Fatalf("%v %v", err, s.Err())
	}
	if strings.Contains(m, "1234") {
		t.Errorf("same message: masked %q keeps a form", m)
	}
	if back := s.Rehydrate(m); back != text {
		t.Errorf("same message: restores %q", back)
	}
	for _, later := range []string{"or 804\u301c555\u301c1234 today", "or 804_555_1234 today", "or 804|555|1234 today"} {
		m, err := s.Mask(context.Background(), later)
		if err != nil || s.Err() != nil || strings.Contains(m, "1234") {
			t.Errorf("later message %q: masked %q %v %v", later, m, err, s.Err())
		}
		if back := s.Rehydrate(m); back != later {
			t.Errorf("later message %q: restores %q", later, back)
		}
	}
	fresh := newEngine(t, nil).NewSession("fresh")
	for _, unknown := range []string{"or 804_555_1234 today", "or 804\u301c555\u301c1234 today", "id 1804_555_12345 x"} {
		if m, _ := fresh.Mask(context.Background(), unknown); m != unknown {
			t.Errorf("unknown %q changed to %q on a fresh vault", unknown, m)
		}
	}
	for _, glued := range []string{"ref a804_555_1234", "x804\u301c555\u301c1234"} {
		m, err := s.Mask(context.Background(), glued)
		if err != nil || strings.Contains(m, "1234") {
			t.Errorf("letter-glued %q: masked %q %v", glued, m, err)
		}
		if back := s.Rehydrate(m); back != glued && !strings.Contains(m, "[redacted") {
			t.Errorf("letter-glued %q: restores %q", glued, back)
		}
	}
	for _, longer := range []string{"804_555_12345", "1804_555_12345", "98045551234567"} {
		if m, _ := s.Mask(context.Background(), longer); m != longer {
			t.Errorf("%q changed to %q", longer, m)
		}
	}
}

func TestKnownPhoneFormsSurviveAVaultReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.vault")
	key := bytes.Repeat([]byte{21}, 32)
	open := func() (*vault.Vault, *Engine) {
		v, err := openTestVault(path, key)
		if err != nil {
			t.Fatal(err)
		}
		e, err := New(Options{Vault: v, Policy: DefaultPolicy()})
		if err != nil {
			t.Fatal(err)
		}
		return v, e
	}
	v, e := open()
	first, err := e.NewSession("one").Mask(context.Background(), "Call 804-555-1234 now")
	if err != nil || strings.Contains(first, "1234") {
		t.Fatalf("first: %q %v", first, err)
	}
	v.Close()
	v, e = open()
	defer v.Close()
	s := e.NewSession("two")
	for _, text := range []string{"or 804\u301c555\u301c1234 today", "or 804_555_1234 today"} {
		m, err := s.Mask(context.Background(), text)
		if err != nil || s.Err() != nil {
			t.Fatalf("%q: %v %v", text, err, s.Err())
		}
		if strings.Contains(m, "1234") {
			t.Errorf("after reopen %q: masked %q keeps the number", text, m)
		}
		if back := s.Rehydrate(m); back != text {
			t.Errorf("after reopen %q: masked %q restores %q", text, m, back)
		}
	}
}

func TestBracketPhoneNeighboursKeepOrMaskAndRestore(t *testing.T) {
	lefts := []string{"", " ", "f", "x", "7", "_", ")", "]", "}", "=", "[", "'", "\"", "`", ",", ":", ";", "(", "{", "!", "?", "*", "#", "@", "&"}
	rights := []string{"", " ", ",", ".", ")", "]", "}", "'", "\"", "`", ";", "!", "a", "_", ".5", "-01"}
	keepLeft := map[string]bool{"f": true, "x": true, "7": true, ")": true, "]": true, "}": true, "=": true, "[": true}
	keepRight := map[string]bool{"a": true, ".5": true, "-01": true}
	bodies := []string{"(804)-555-1234", "(804).555.1234", "(804) 555-1234"}
	states := map[string]string{"fresh": "", "dashed": "Call 804-555-1234 now", "bracketed": "Call (804) 555-1234 now"}
	cases, pair := 0, 0
	for state, hold := range states {
		for _, body := range bodies {
			for _, l := range lefts {
				for _, r := range rights {
					if pair++; scaletest.Race && pair%3 != 0 {
						continue
					}
					text := "note " + l + body + r + " end"
					if l == "" {
						text = body + r + " end"
					}
					e := newEngine(t, nil)
					s := e.NewSession("brk")
					if hold != "" {
						if _, err := s.Mask(context.Background(), hold); err != nil {
							t.Fatal(err)
						}
					}
					m, err := s.Mask(context.Background(), text)
					if err != nil || s.Err() != nil {
						t.Fatalf("%s %q: %v %v", state, text, err, s.Err())
					}
					cases++
					masked := m != text
					if masked {
						if back := s.Rehydrate(m); back != text {
							t.Errorf("%s %q: masked %q restores %q", state, text, m, back)
						}
					}
					if state != "fresh" {
						continue
					}
					space := strings.Contains(body, ") ")
					enclosed := l == "[" && r == "]"
					want := !keepRight[r] && !(keepLeft[l] && !space && !enclosed) && l != "7"
					if space && r == ".5" && l != "7" {
						want = true
					}
					if masked != want {
						t.Errorf("fresh %q: masked=%v, want %v (%q)", text, masked, want, m)
					}
				}
			}
		}
		e := newEngine(t, nil)
		s := e.NewSession("mixed")
		if hold != "" {
			s.Mask(context.Background(), hold)
		}
		mixed := "f(804)-555-1234 and 804-555-1234"
		m, err := s.Mask(context.Background(), mixed)
		if err != nil || s.Err() != nil {
			t.Fatalf("%s mixed: %v %v", state, err, s.Err())
		}
		if !strings.Contains(m, " and ") || strings.HasSuffix(m, "804-555-1234") {
			t.Errorf("%s mixed: %q", state, m)
		}
		if back := s.Rehydrate(m); back != mixed {
			t.Errorf("%s mixed: masked %q restores %q", state, m, back)
		}
	}
	t.Logf("%d bracketed neighbour cases across %d vault states", cases, len(states))
}

func TestKnownNumbersOnlyMaskWhenHeld(t *testing.T) {
	for _, text := range []string{"order 8045551234 shipped", "ratio 804:555:1234", "id 804|555|1234", "ref 804_555_1234"} {
		fresh := newEngine(t, nil).NewSession("fresh")
		if m, _ := fresh.Mask(context.Background(), text); m != text {
			t.Errorf("fresh %q changed to %q", text, m)
		}
		e := newEngine(t, nil)
		s := e.NewSession("held")
		s.Mask(context.Background(), "Call 804-555-1234 now")
		m, err := s.Mask(context.Background(), text)
		if err != nil || s.Err() != nil || strings.Contains(m, "1234") {
			t.Errorf("held %q: masked %q %v %v", text, m, err, s.Err())
		}
		if back := s.Rehydrate(m); back != text {
			t.Errorf("held %q: masked %q restores %q", text, m, back)
		}
	}
}

func TestBracketNumbersInCodeAndLogsStayUnmasked(t *testing.T) {
	lines := []string{
		"x = sum(804)-555-1234",
		"t(804).555.1234",
		"f(x)=(804)-555-1234",
		"arr[(804)-555-1234]",
		"v(804).555.1234.2",
		"range (120)-130-1400 ms",
		"(100)-200-3000 units",
		"(304)-512-1024 bytes",
		"coords (812).553.1207",
		"offset (804)-555-1234",
		"size: (804)-555-1234",
		"len(804)-555-1234",
		"max(804).555.1234",
		"dict[(804)-555-1234]",
		"y=(804)-555-1234",
		"a_(804)-555-1234",
		"x2(804)-555-1234",
		"obj.method(804)-555-1234",
		"print(804)-555-1234",
		"math.pow(804)-555-1234",
		"SELECT id FROM t WHERE x=(804)-555-1234",
		"WHERE v IN(804)-555-1234",
		"SELECT AVG(804)-555-1234 FROM t",
		"SELECT COUNT(804).555.1234 FROM t",
		"count (804)-555-1234",
		"latency (804)-555-1234 ms",
		"(804)-555-1234 px",
		"(804)-555-1234 rows",
		"(804).555.1234 items",
		"(804)-555-1234 kb",
		"(804)-555-1234 mb",
		"(104)-555-1234",
		"(804)-155-1234",
		"x[(804).555.1234]",
		"(123).456.7890",
		"(911)-111-2222",
		"(804)-555-1234.5",
		"(804)-555-1234-01",
		"(804).555.1234.99",
		"(804)-555-1234abc",
		"ts=(804)-555-1234",
		"[INFO] f(804)-555-1234 ok",
		"k(804).555.1234 v",
		"2(804)-555-1234",
		"3(804).555.1234",
		"(x)(804)-555-1234",
		"WHERE a=(804)-555-1234",
		"m[i](804).555.1234",
		"run (804)-555-1234",
		"tbl[(804).555.1234]",
	}
	if len(lines) < 40 {
		t.Fatalf("only %d lines", len(lines))
	}
	for _, text := range lines {
		e := newEngine(t, nil)
		s := e.NewSession("code")
		m, err := s.Mask(context.Background(), text)
		if err != nil || s.Err() != nil {
			t.Fatalf("%q: %v %v", text, err, s.Err())
		}
		if m != text {
			t.Errorf("%q changed to %q", text, m)
		}
	}
}

func TestPlusOneWithAnySeparatorAfterACue(t *testing.T) {
	for _, sep := range []string{"/", "_", "~", "\uff0f", "\uff5e", "\u301c", "\u30fb", "\u00b7", "|", "\uff5c", ":", "\uff1a", "*", "\uff0a"} {
		for _, plus := range []string{"+", "\uff0b"} {
			text := "Call " + plus + "1" + sep + "804" + sep + "555" + sep + "1234 now"
			e := newEngine(t, nil)
			s := e.NewSession("plus1")
			m, err := s.Mask(context.Background(), text)
			if err != nil || s.Err() != nil {
				t.Fatalf("%q: %v %v", text, err, s.Err())
			}
			if strings.Contains(m, "1234") {
				t.Errorf("%q: masked %q keeps the number", text, m)
			}
			if back := s.Rehydrate(m); back != text {
				t.Errorf("%q: masked %q restores %q", text, m, back)
			}
		}
	}
}
