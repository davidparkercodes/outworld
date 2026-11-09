package outworld

import (
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/detect"
	"github.com/davidparkercodes/outworld/pkg/surrogate"
)

func dictEngine(t *testing.T, csv string) *Engine {
	t.Helper()
	return newEngine(t, func(o *Options) {
		d, err := detect.ParseDictionaryCSV(strings.NewReader(csv))
		if err != nil {
			t.Fatal(err)
		}
		o.Dictionary = d
	})
}

func TestAliasesShareOneIdentity(t *testing.T) {
	e := dictEngine(t, "value,type,aliases\nMarcus Delaney,person,Marc|Delaney\n")
	s := e.NewSession("")
	in := "Marcus Delaney called. Marc said Delaney's car is fixed."
	out, _ := s.Mask(ctx, in)
	full := strings.Fields(strings.SplitN(out, " called", 2)[0])
	if len(full) != 2 {
		t.Fatalf("full name shape: %q", out)
	}
	nick := strings.Fields(strings.SplitN(out, ". ", 2)[1])[0]
	if nick == full[0] || len(nick) < 2 || !strings.HasPrefix(full[0], nick[:2]) {
		t.Fatalf("nickname %q should derive from first-name surrogate %q: %q", nick, full[0], out)
	}
	if !strings.Contains(out, full[1]+"'s") {
		t.Fatalf("last name not consistent: %q", out)
	}
	if back := s.Rehydrate(out); back != in {
		t.Fatalf("rehydrate: %q", back)
	}
}

func TestSurrogateMatchesGender(t *testing.T) {
	e := dictEngine(t, "value,type,aliases,generalize,gender\nPriya Raman,person,,,\nAlex Doe,person,,,f\nJordan Roe,person,,,m\n")
	cases := map[string]string{"Priya": surrogate.Female, "Alex": surrogate.Female, "Jordan": surrogate.Male}
	for real, want := range cases {
		out, _ := e.NewSession("").Mask(ctx, real+" called.")
		first := strings.Fields(out)[0]
		if got := surrogate.GenderOf(first); got != want {
			t.Errorf("%s -> %s has gender %q, want %q", real, first, got, want)
		}
	}
	out, _ := e.NewSession("").Mask(ctx, "Dr. Zorblax called.")
	if g := surrogate.GenderOf(strings.Fields(out)[1]); g != surrogate.Neutral {
		t.Errorf("unknown name should get a neutral first name, got %q", out)
	}
}

func TestCaseInsensitiveFirstNamesWithCommonWordGuard(t *testing.T) {
	e := dictEngine(t, "value,type,aliases\nIngrid Achterberg,person,\nWill Ashgrove,person,\nMay Ostrander,person,\nChase Nwankwo,person,\n")
	for _, c := range []struct {
		in       string
		gone     []string
		mustKeep []string
	}{
		{"ingrid said the deploy failed", []string{"ingrid"}, nil},
		{"hey will!! the cake?", []string{"will!!"}, nil},
		{"I will go. Will you come?", nil, []string{"I will go. Will you come?"}},
		{"See you in May or on May 4.", nil, []string{"in May or on "}},
		{"May said yes.", []string{"May said"}, nil},
		{"Chase, I keep getting the error.", []string{"Chase,"}, nil},
		{"No need to chase this one.", nil, []string{"chase this"}},
		{"[09:15] chase_n: on it", []string{"chase_n"}, nil},
		{"wiki/people/ingrid-achterberg and ingrid.achterberg", []string{"ingrid"}, nil},
	} {
		out, _ := e.NewSession("").Mask(ctx, c.in)
		for _, g := range c.gone {
			if strings.Contains(out, g) {
				t.Errorf("%q: %q should be masked, got %q", c.in, g, out)
			}
		}
		for _, k := range c.mustKeep {
			if !strings.Contains(out, k) {
				t.Errorf("%q: %q should be kept, got %q", c.in, k, out)
			}
		}
	}
}

func TestKnownGivenNameExtendsToSurname(t *testing.T) {
	e, _ := New(Options{Vault: newEngine(t, nil).Vault(), Policy: DefaultPolicy()})
	s := e.NewSession("")
	in := "Hi Yusra,\nthanks.\nYusra Pellucidi"
	out, _ := s.Mask(ctx, in)
	if strings.Contains(out, "Yusra") || strings.Contains(out, "Pellucidi") {
		t.Fatalf("got %q", out)
	}
	if s.Err() != nil {
		t.Fatal(s.Err())
	}
	if back := s.Rehydrate(out); back != in {
		t.Fatalf("rehydrate %q", back)
	}
}

func TestPrimingMasksEarlierMessages(t *testing.T) {
	e, _ := New(Options{Vault: newEngine(t, nil).Vault(), Policy: DefaultPolicy()})
	msgs := []string{"Kwabena owes me $40.", "Dear Mr. Kwabena Asante,"}
	s := e.NewSession("")
	s.SetPriming(true)
	for _, m := range msgs {
		if _, err := s.Mask(ctx, m); err != nil {
			t.Fatal(err)
		}
	}
	s.SetPriming(false)
	for _, m := range msgs {
		out, _ := s.Mask(ctx, m)
		if strings.Contains(out, "Kwabena") || strings.Contains(out, "Asante") {
			t.Fatalf("got %q", out)
		}
	}
	if st := s.Stats(); st.Masked == 0 || s.Err() != nil {
		t.Fatalf("stats %+v err %v", st, s.Err())
	}
}

func TestPhoneRestoredInAnyFormat(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("")
	out, _ := s.Mask(ctx, "call (804) 555-1234")
	d := digitsOf(out)
	if len(d) != 10 || strings.Contains(out, "804") {
		t.Fatalf("masked %q", out)
	}
	area, ex, line := d[:3], d[3:6], d[6:]
	for answer, want := range map[string]string{
		area + "-" + ex + "-" + line:         "804-555-1234",
		area + "." + ex + "." + line:         "804.555.1234",
		"+1 " + area + " " + ex + " " + line: "+1 804 555 1234",
		"1" + d:                              "18045551234",
		d:                                    "8045551234",
		"(" + area + ")" + ex + "-" + line:   "(804)555-1234",
		"Call " + area + "-" + ex + "-" + line + ".": "Call 804-555-1234.",
	} {
		if got := s.Rehydrate(answer); got != want {
			t.Errorf("%q -> %q, want %q", answer, got, want)
		}
		r := s.NewStreamRehydrator(false)
		var b strings.Builder
		for i := 0; i < len(answer); i++ {
			b.WriteString(r.Push(answer[i : i+1]))
		}
		b.WriteString(r.Flush())
		if b.String() != want {
			t.Errorf("stream %q -> %q, want %q", answer, b.String(), want)
		}
	}
}

func TestDatesRestoreExactlyAsWritten(t *testing.T) {
	e := newEngine(t, nil)
	for _, in := range []string{"Sat, 6 Jun 2026 13:17", "on Jun 15.", "Wednesday, December 2 works", "MADE ON MARCH 1, 2026 BETWEEN", "the 2nd of March", "Sept 9"} {
		s := e.NewSession("x")
		out, _ := s.Mask(ctx, in)
		if out == in {
			t.Errorf("%q not shifted", in)
		}
		if back := s.Rehydrate(out); back != in {
			t.Errorf("%q -> %q -> %q", in, out, back)
		}
	}
}

func TestUndetectedRealEqualToSurrogateIsRemasked(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("")
	out, _ := s.Mask(ctx, "Maria Garcia called.")
	sur := strings.Fields(out)[1]
	s2 := e.NewSession("")
	in := "Maria Garcia met Dr. Okafor and " + sur + " Industries."
	out2, _ := s2.Mask(ctx, in)
	if back := s2.Rehydrate(out2); back != in {
		t.Fatalf("rehydrate %q -> %q -> %q", in, out2, back)
	}
}
