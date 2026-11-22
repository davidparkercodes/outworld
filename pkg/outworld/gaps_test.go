package outworld

import (
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/detect"
)

func maskRoundTrip(t *testing.T, e *Engine, in string) string {
	t.Helper()
	s := e.NewSession("gaps")
	out, err := s.Mask(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Err(); err != nil {
		t.Fatalf("%q: %v", in, err)
	}
	if back := s.Rehydrate(out); back != in {
		t.Fatalf("round trip\n in  %q\n out %q\n back %q", in, out, back)
	}
	return out
}

func TestCodeHandlesMaskedAndRestored(t *testing.T) {
	cases := []struct{ in, handle string }{
		{"TODO(mwilliams): clean this up", "mwilliams"},
		{"FIXME(kchen)", "kchen"},
		{"I pushed the branch kchen/fix-auth last night", "kchen"},
		{"git checkout -b jsmith/retry-logic", "jsmith"},
		{"@mwilliams can you review", "mwilliams"},
		{"Reviewed-by: kchen", "kchen"},
		{"Signed-off-by: tbecker\nCo-authored-by: hnakamura", "tbecker"},
		{"see /home/jdoe/project and /Users/jdoe/Library", "jdoe"},
		{`open C:\Users\JDoe\Desktop\notes.txt`, "JDoe"},
		{"TODO(dthompson42): tidy", "dthompson42"},
		{"ls /home/xqzrt/.ssh", "xqzrt"},
	}
	for _, c := range cases {
		out := maskRoundTrip(t, newEngine(t, nil), c.in)
		if strings.Contains(out, c.handle) {
			t.Errorf("handle %q left in %q", c.handle, out)
		}
	}
}

func TestCodeHandleNegativesUntouched(t *testing.T) {
	in := "TODO(perf) feature/login release/1.2 fix/auth @types/node @param @Override @media /home/ubuntu/ /Users/Shared/ origin/main dependabot/npm"
	if out := maskRoundTrip(t, newEngine(t, nil), in); out != in {
		t.Fatalf("changed: %q", out)
	}
}

func TestCodeHandleConsistentWithSurname(t *testing.T) {
	e := newEngine(t, nil)
	in := "TODO(mwilliams): tidy up\nlogs are in /home/mwilliams/logs\nMr. Williams owns this; ping @mwilliams"
	out := maskRoundTrip(t, e, in)
	fam, ok := e.Vault().Get("person.family", "Williams")
	if !ok {
		t.Fatal("no family surrogate for Williams")
	}
	want := "m" + strings.ToLower(fam)
	if strings.Count(out, want) != 3 || !strings.Contains(out, "Mr. "+fam+" owns") {
		t.Fatalf("want %q three times in %q", want, out)
	}

	e = newEngine(t, nil)
	in = "TODO(mwilliams): tidy up\nlogs are in /home/mwilliams/logs\nMary Williams owns this; ping @mwilliams"
	out = maskRoundTrip(t, e, in)
	fam, _ = e.Vault().Get("person.family", "Williams")
	giv, _ := e.Vault().Get("person.given", "Mary")
	if fam == "" || giv == "" {
		t.Fatalf("missing surrogates in %q", out)
	}
	want = strings.ToLower(giv[:1] + fam)
	if strings.Count(out, want) != 3 || strings.Contains(strings.ToLower(out), "williams") {
		t.Fatalf("want %q three times in %q", want, out)
	}
}

func TestCodeHandleKnownAcrossTexts(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("gaps")
	first, err := s.Mask(ctx, "FIXME(kchen): flaky")
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Mask(ctx, "the fix from kchen is merged")
	if err != nil {
		t.Fatal(err)
	}
	sur := strings.TrimSuffix(strings.TrimPrefix(first, "FIXME("), "): flaky")
	if sur == "kchen" || !strings.Contains(second, sur) || strings.Contains(second, "kchen") {
		t.Fatalf("first %q second %q", first, second)
	}
	if s.Rehydrate(second) != "the fix from kchen is merged" {
		t.Fatalf("rehydrate %q", s.Rehydrate(second))
	}
}

func TestCompaniesPassThrough(t *testing.T) {
	for _, in := range []string{
		"Morgan Stanley reported earnings today.",
		"Goldman Sachs, Wells Fargo and Charles Schwab were downgraded.",
		"Johnson & Johnson hired Ernst & Young.",
		"Maria Okafor Holdings and Becker & Sons signed the lease.",
	} {
		if out := maskRoundTrip(t, newEngine(t, nil), in); out != in {
			t.Errorf("changed %q to %q", in, out)
		}
	}
}

func TestCompanyNamedPersonStillMasked(t *testing.T) {
	d, err := detect.ParseDictionaryCSV(strings.NewReader("value,type,aliases\nMorgan Stanley,person,\n"))
	if err != nil {
		t.Fatal(err)
	}
	e := newEngine(t, func(o *Options) { o.Dictionary = d })
	out := maskRoundTrip(t, e, "Morgan Stanley reported earnings today.")
	if strings.Contains(out, "Morgan") || strings.Contains(out, "Stanley") {
		t.Fatalf("dictionary person not masked: %q", out)
	}
	for _, in := range []string{"Contact: Morgan Stanley", "Mr. Edward Jones called about the lease.", "My brother Raymond James is visiting."} {
		out := maskRoundTrip(t, newEngine(t, nil), in)
		for _, w := range []string{"Morgan", "Stanley", "Edward", "Jones", "Raymond", "James"} {
			if strings.Contains(out, w) {
				t.Errorf("%q left in %q", w, out)
			}
		}
	}
}

func TestFamilyFirstMaskedWithFamilyRole(t *testing.T) {
	e := newEngine(t, nil)
	out := maskRoundTrip(t, e, "Please ask Nguyen Van An about the invoice.")
	if strings.Contains(out, "Nguyen") || strings.Contains(out, "Van An") {
		t.Fatalf("leak in %q", out)
	}
	fam, ok := e.Vault().Get("person.family", "Nguyen")
	if !ok || !strings.Contains(out, "ask "+fam+" ") {
		t.Fatalf("family surrogate %q not first in %q", fam, out)
	}
	if _, ok := e.Vault().Get("person.given", "Nguyen"); ok {
		t.Fatal("family name stored as a given name")
	}

	e = newEngine(t, nil)
	out = maskRoundTrip(t, e, "Zhang Wei sent it. Later Wei Zhang confirmed, and Zhang signed.")
	fam, _ = e.Vault().Get("person.family", "Zhang")
	giv, _ := e.Vault().Get("person.given", "Wei")
	if fam == "" || giv == "" || !strings.HasPrefix(out, fam+" "+giv+" sent") || !strings.Contains(out, giv+" "+fam+" confirmed") {
		t.Fatalf("roles not consistent: %q (family %q given %q)", out, fam, giv)
	}

	for _, in := range []string{"Kim Min-jun joined.", "Forward it to Li Xiaoming today.", "Park Ji Sung called."} {
		out := maskRoundTrip(t, newEngine(t, nil), in)
		for _, w := range []string{"Kim", "Min-jun", "Xiaoming", "Park", "Sung"} {
			if strings.Contains(out, w) {
				t.Errorf("%q left in %q", w, out)
			}
		}
	}
	in := "Do Not Disturb. He Said. Park Avenue. Sun Microsystems."
	if out := maskRoundTrip(t, newEngine(t, nil), in); out != in {
		t.Fatalf("changed: %q", out)
	}
}
