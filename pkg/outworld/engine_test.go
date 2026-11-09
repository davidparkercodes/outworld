package outworld

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/detect"
	"github.com/davidparkercodes/outworld/pkg/entity"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

const dict = `value,type,aliases
John Smith,person,Johnny
Maria Garcia,person,
Priya Raman,person,P. Raman
Acme Health,org,
Will Turner,person,
`

var ctx = context.Background()

func newEngine(t *testing.T, mutate func(*Options)) *Engine {
	t.Helper()
	v, err := vault.Open(vault.NewMemoryBackend(bytes.Repeat([]byte{9}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	d, err := detect.ParseDictionaryCSV(strings.NewReader(dict))
	if err != nil {
		t.Fatal(err)
	}
	o := Options{Vault: v, Policy: DefaultPolicy(), Dictionary: d}
	if mutate != nil {
		mutate(&o)
	}
	e, err := New(o)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

var reals = []string{
	"John", "Smith", "Johnny", "Maria", "Garcia", "Priya", "Raman", "john.smith@gmail.com", "(804) 555-1234",
	"123 Main Street", "482913", "123-45-6789", "4111 1111 1111 1111", "March 15, 2024", "10.0.0.5", "sk-proj-",
}

func assertNoReals(t *testing.T, out string, extra ...string) {
	t.Helper()
	low := strings.ToLower(out)
	for _, r := range append(reals, extra...) {
		if strings.Contains(low, strings.ToLower(r)) {
			t.Fatalf("real value %q leaked in %q", r, out)
		}
	}
}

const sample = `Hi John,
Johnny's appointment at Acme Health is on March 15, 2024. JOHN SMITH and john smith are the same.
Email john.smith@gmail.com or call (804) 555-1234. Address: 123 Main Street, Apt 4B, Richmond, VA 23220.
Maria Garcia said your code is 482913. SSN 123-45-6789. Card 4111 1111 1111 1111. Key sk-proj-abcdefghijklmnopqrstuvwxyz123456.
Reset: https://accounts.example.com/reset?token=abc. Host 10.0.0.5. Profile facebook.com/john.smith. P. Raman and Priya agree.`

func TestMaskRoundTrip(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("")
	out, err := s.Mask(ctx, sample)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Err(); err != nil {
		t.Fatalf("unexpected leak: %v", err)
	}
	assertNoReals(t, out)
	if !strings.Contains(out, "Acme Health") {
		t.Fatal("org policy is keep")
	}
	for _, tok := range []string{"[redacted code]", "[redacted id]", "[redacted card]", "[redacted key]", "[redacted link]", "555-01"} {
		if !strings.Contains(out, tok) {
			t.Fatalf("missing %q in %s", tok, out)
		}
	}
	back := e.NewSession("").Rehydrate(out)
	want := sample
	for _, r := range []string{"482913", "123-45-6789", "4111 1111 1111 1111", "sk-proj-abcdefghijklmnopqrstuvwxyz123456", "https://accounts.example.com/reset?token=abc"} {
		want = strings.Replace(want, r, "", 1)
	}
	for _, r := range []string{"[redacted code]", "[redacted id]", "[redacted card]", "[redacted key]", "[redacted link]"} {
		back = strings.Replace(back, r, "", 1)
	}
	if back != want {
		t.Fatalf("roundtrip mismatch:\n got: %s\nwant: %s", back, want)
	}
}

func TestConsistencyAcrossSessionsAndShapes(t *testing.T) {
	e := newEngine(t, nil)
	a, _ := e.NewSession("").Mask(ctx, "John Smith wrote.")
	b, _ := e.NewSession("").Mask(ctx, "Later, John replied to Smith; JOHN SMITH signed; john.smith was cc'd.")
	full := strings.TrimSuffix(strings.TrimPrefix(a, ""), " wrote.")
	parts := strings.Fields(full)
	if len(parts) != 2 {
		t.Fatalf("full name shape not kept: %q", full)
	}
	if !strings.Contains(b, "Later, "+parts[0]+" replied to "+parts[1]) {
		t.Fatalf("first/last only not consistent: %q vs %q", a, b)
	}
	if !strings.Contains(b, strings.ToUpper(full)) || !strings.Contains(b, strings.ToLower(parts[0]+"."+parts[1])) {
		t.Fatalf("case variants not consistent: %q", b)
	}
}

func TestPossessivesAndOverlaps(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("")
	out, _ := s.Mask(ctx, "Maria's and Garcia's files; Maria Garcia's email maria.garcia@corp.test.")
	assertNoReals(t, out, "maria")
	if strings.Count(out, "'s") != 3 {
		t.Fatalf("possessives lost: %q", out)
	}
	if got := e.NewSession("").Rehydrate(out); got != "Maria's and Garcia's files; Maria Garcia's email maria.garcia@corp.test." {
		t.Fatalf("rehydrate: %q", got)
	}
}

func TestNamesInsideURLsAndEmails(t *testing.T) {
	e := newEngine(t, nil)
	in := "See https://social.example/john.smith/posts and johnsmith@corp.test and https://x.example/u/johnsmith"
	out, _ := e.NewSession("").Mask(ctx, in)
	assertNoReals(t, out, "johnsmith")
	if got := e.NewSession("").Rehydrate(out); got != in {
		t.Fatalf("rehydrate: %q", got)
	}
}

func TestCommonWordNamesNeedCapitalization(t *testing.T) {
	e := newEngine(t, nil)
	out, _ := e.NewSession("").Mask(ctx, "I will ask Will Turner. Will said yes.")
	if !strings.HasPrefix(out, "I will ask ") || strings.Contains(out, "Will Turner") || strings.Contains(out, "Will said") {
		t.Fatalf("got %q", out)
	}
}

func TestDateShiftScopes(t *testing.T) {
	e := newEngine(t, nil)
	a := e.NewSession("alpha")
	b := e.NewSession("beta")
	if a.DateOffset() == 0 || a.DateOffset() == b.DateOffset() {
		t.Fatalf("offsets %d %d", a.DateOffset(), b.DateOffset())
	}
	if e.NewSession("alpha").DateOffset() != a.DateOffset() {
		t.Fatal("scope offset not stable")
	}
	in := "Visit on March 15, 2024 and 2024-04-01; next week is fine; Tuesday too."
	out, _ := a.Mask(ctx, in)
	if strings.Contains(out, "March 15, 2024") || !strings.Contains(out, "next week is fine; Tuesday too.") {
		t.Fatalf("shift: %q", out)
	}
	if back := e.NewSession("alpha").Rehydrate(out); back != in {
		t.Fatalf("unshift: %q", back)
	}
	off := newEngine(t, func(o *Options) { o.Policy.DateShift = false })
	if out, _ := off.NewSession("").Mask(ctx, in); out != in {
		t.Fatalf("date shift disabled but changed: %q", out)
	}
}

func TestPolicyKeepGeneralizeStrip(t *testing.T) {
	e := newEngine(t, func(o *Options) {
		o.Policy.Actions[entity.Location] = entity.Generalize
		o.Policy.Actions[entity.Org] = entity.Generalize
		o.Policy.Actions[entity.Person] = entity.Keep
		o.Policy.Actions[entity.Email] = entity.Strip
	})
	out, _ := e.NewSession("").Mask(ctx, "John Smith of Acme Health moved to Norfolk, Virginia. Mail john@x.test")
	want := "John Smith of an organization moved to a city in Virginia. Mail [redacted email]"
	if out != want {
		t.Fatalf("got  %q\nwant %q", out, want)
	}
}

type fakeNER struct{ names []string }

func (f fakeNER) Name() string { return "fake" }
func (f fakeNER) Detect(_ context.Context, text string) ([]entity.Span, error) {
	var out []entity.Span
	for _, n := range f.names {
		if i := strings.Index(text, n); i >= 0 {
			out = append(out, entity.Span{Start: i, End: i + len(n), Type: entity.Person, Text: n, Source: "ner"})
		}
	}
	return out, nil
}

type failingNER struct{}

func (failingNER) Name() string { return "down" }
func (failingNER) Detect(context.Context, string) ([]entity.Span, error) {
	return nil, errors.New("down")
}

func TestOutboundLeakCheckRemediatesKnownRealValue(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.NER = fakeNER{names: []string{"Zebulon Quarry"}} })
	s := e.NewSession("")
	if _, err := s.Mask(ctx, "Zebulon Quarry called."); err != nil || s.Err() != nil {
		t.Fatalf("first pass: %v %v", err, s.Err())
	}
	e2, _ := New(Options{Vault: e.Vault(), Policy: DefaultPolicy()})
	s2 := e2.NewSession("")
	in := "Did zebulon quarry call back? Zebulon said Quarry would."
	out, _ := s2.Mask(ctx, in)
	if err := s2.Err(); err != nil {
		t.Fatalf("known values should be masked in place, got %v (out=%q)", err, out)
	}
	if strings.Contains(strings.ToLower(out), "zebulon") || strings.Contains(strings.ToLower(out), "quarry") {
		t.Fatalf("known value left in %q", out)
	}
	if back := s2.Rehydrate(out); back != in {
		t.Fatalf("rehydrate: %q", back)
	}
}

func TestLeakCheckBlocksUntransformableField(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("")
	s.Check(ctx, "Zed John Smith")
	err := s.Err()
	if err == nil || !errors.Is(err, ErrLeak) {
		t.Fatalf("expected leak block, got %v", err)
	}
	if strings.Contains(err.Error(), "Smith") {
		t.Fatal("leak error must not contain the value")
	}
}

func TestLeakCheckOnUntransformedField(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("")
	s.Check(ctx, "maria.garcia@corp.test")
	if s.Err() == nil {
		t.Fatal("check should flag unmasked email")
	}
}

func TestNERFailureFailsClosed(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.NER = failingNER{} })
	if _, err := e.NewSession("").Mask(ctx, "hello there"); err == nil {
		t.Fatal("expected error")
	}
	e2 := newEngine(t, func(o *Options) { o.NER = failingNER{}; o.NEROptional = true })
	if _, err := e2.NewSession("").Mask(ctx, "hello there"); err != nil {
		t.Fatal(err)
	}
}

func TestUnknownNames(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("")
	out, _ := s.Mask(ctx, "John Smith is here.")
	back := s.Rehydrate(out + " Also Bartholomew Vanterpool and Ignatius.")
	names := s.UnknownNames(back)
	if strings.Join(names, ",") != "Bartholomew,Ignatius,Vanterpool" {
		t.Fatalf("names=%v back=%q", names, back)
	}
}

func TestRehydrateArgumentsEscapesJSON(t *testing.T) {
	e := newEngine(t, func(o *Options) {
		d, _ := detect.ParseDictionaryCSV(strings.NewReader(`value,type,aliases` + "\n" + `"Ana ""The Hammer"" Lopes",person,`))
		o.Dictionary = d
	})
	s := e.NewSession("")
	masked, _ := s.Mask(ctx, `Ana "The Hammer" Lopes`)
	args, _ := json.Marshal(map[string]any{"to": masked, "n": 3})
	r := s.RehydrateArguments(string(args))
	var v map[string]any
	if err := json.Unmarshal([]byte(r), &v); err != nil {
		t.Fatalf("invalid json %q: %v", r, err)
	}
	if v["to"] != `Ana "The Hammer" Lopes` {
		t.Fatalf("got %v", v["to"])
	}
	raw := s.RehydrateJSONText(string(args))
	if err := json.Unmarshal([]byte(raw), &v); err != nil || v["to"] != `Ana "The Hammer" Lopes` {
		t.Fatalf("text mode %q %v", raw, err)
	}
}

func TestStreamRehydrationAllSplits(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("")
	original := "Dear Maria Garcia, John's call (804) 555-1234 on March 15, 2024 about john.smith@gmail.com. Bye Johnny!"
	masked, _ := s.Mask(ctx, original)
	assertNoReals(t, masked)
	for size := 1; size <= 9; size++ {
		for offset := 0; offset < size; offset++ {
			r := s.NewStreamRehydrator(false)
			var b strings.Builder
			first := offset
			if first > len(masked) {
				first = len(masked)
			}
			b.WriteString(r.Push(masked[:first]))
			for i := first; i < len(masked); i += size {
				end := i + size
				if end > len(masked) {
					end = len(masked)
				}
				b.WriteString(r.Push(masked[i:end]))
			}
			b.WriteString(r.Flush())
			if b.String() != original {
				t.Fatalf("size=%d offset=%d\n got: %q\nwant: %q", size, offset, b.String(), original)
			}
		}
	}
}

func TestStreamHoldsOnlyWhatItMust(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("")
	s.Mask(ctx, "John Smith")
	r := s.NewStreamRehydrator(false)
	if got := r.Push("The weather is nice "); got != "The weather is nice " {
		t.Fatalf("plain text held back: %q", got)
	}
	sur, _ := e.Vault().Get("person.given", "John")
	half := sur[:3]
	if got := r.Push("and " + half); got != "and " {
		t.Fatalf("expected hold of partial surrogate, got %q", got)
	}
	if got := r.Push(sur[3:] + " left."); got != "John left." {
		t.Fatalf("got %q", got)
	}
}

func TestMaskIsIdempotentOnSurrogates(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("")
	out, _ := s.Mask(ctx, "Email john.smith@gmail.com and call (804) 555-1234.")
	s2 := e.NewSession("")
	again, _ := s2.Mask(ctx, out)
	if s2.Err() != nil {
		t.Fatalf("surrogates should never trip the leak check: %v", s2.Err())
	}
	if e.NewSession("").Rehydrate(e.NewSession("").Rehydrate(again)) == "" {
		t.Fatal("empty")
	}
}
