package outworld

import (
	"strings"
	"testing"
)

func streamAll(s *Session, text string, size int) string {
	r := s.NewStreamRehydrator(false)
	var b strings.Builder
	runes := []rune(text)
	for i := 0; i < len(runes); i += size {
		end := i + size
		if end > len(runes) {
			end = len(runes)
		}
		b.WriteString(r.Push(string(runes[i:end])))
	}
	b.WriteString(r.Flush())
	return b.String()
}

func TestNationalAndSpokenPhonesRoundTrip(t *testing.T) {
	cases := []string{
		"ring me on 07911 123456 tonight",
		"Office: 020 7123 4567.",
		"office line (020) 7123 4567 after nine",
		"Tel. 030 12345 678 bitte",
		"appelez le 01 23 45 67 89 demain",
		"mobile 0412 345 678, after six",
		"dial 0044 20 7123 4567 from abroad",
		"cell +61 412 345 678 or fax +33 1 23 45 67 89",
		"my number is four one five five five five oh one four two ok",
		"it's Seven double five nine three one two, thanks",
	}
	for _, text := range cases {
		e := newEngine(t, nil)
		s := e.NewSession("")
		masked, err := s.Mask(ctx, text)
		if err != nil || s.Err() != nil {
			t.Fatalf("%q: mask error %v %v", text, err, s.Err())
		}
		if masked == text {
			t.Errorf("%q: not masked", text)
			continue
		}
		if len(masked) != len(text) && strings.ContainsAny(text, "0123456789") {
			t.Errorf("%q: formatting changed in %q", text, masked)
		}
		if got := s.Rehydrate(masked); got != text {
			t.Errorf("rehydrate %q -> %q, want %q", masked, got, text)
		}
		for size := 1; size <= 7; size++ {
			if got := streamAll(s, masked, size); got != text {
				t.Errorf("stream size %d: %q -> %q, want %q", size, masked, got, text)
			}
		}
		again, _ := e.NewSession("").Mask(ctx, text)
		if again != masked {
			t.Errorf("%q: surrogate not stable: %q then %q", text, masked, again)
		}
	}
}

func TestNationalPhoneRestoredInOtherFormatting(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("")
	out, _ := s.Mask(ctx, "ring 07911 123456")
	d := digitsOf(out)
	if len(d) != 11 || !strings.HasPrefix(d, "07") || strings.Contains(out, "123456") {
		t.Fatalf("masked %q", out)
	}
	for answer, want := range map[string]string{
		d[:5] + " " + d[5:]:                 "07911 123456",
		d[:5] + "-" + d[5:]:                 "07911-123456",
		d[:4] + " " + d[4:7] + " " + d[7:]:  "0791 112 3456",
		"Call " + d[:5] + " " + d[5:] + ".": "Call 07911 123456.",
	} {
		if got := s.Rehydrate(answer); got != want {
			t.Errorf("%q -> %q, want %q", answer, got, want)
		}
		if got := streamAll(s, answer, 1); got != want {
			t.Errorf("stream %q -> %q, want %q", answer, got, want)
		}
	}
}

func TestSpokenPhonesGetDistinctSurrogates(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("")
	a, _ := s.Mask(ctx, "first is four one five five five five oh one four two")
	b, _ := s.Mask(ctx, "second is nine one seven five five five two zero four six")
	if strings.TrimPrefix(a, "first is ") == strings.TrimPrefix(b, "second is ") {
		t.Fatalf("two spoken numbers share one surrogate: %q %q", a, b)
	}
	if got := s.Rehydrate(a + " / " + b); got != "first is four one five five five five oh one four two / second is nine one seven five five five two zero four six" {
		t.Fatalf("rehydrate %q", got)
	}
}

func TestAddressesOutsideUSStyleRoundTrip(t *testing.T) {
	cases := []struct{ text, real string }{
		{"livré au 14 rue des Lilas, 69003 Lyon hier", "Lilas"},
		{"vivo en Calle Alcalde Moreno 21, 41004 Sevilla desde mayo", "Moreno"},
		{"wohnt in der Lindenstraße 8, 50674 Köln seit Mai", "Lindenstraße"},
		{"woont op Herengracht 401, 1017 BP Amsterdam nu", "Herengracht"},
		{"send it to Flat 4, 27 Glenorchy Road, Perth PH1 5XY please", "Glenorchy"},
		{"SHIP TO:\n2207 HOLLOWAY RD STE 14\nRALEIGH NC 27603\nthanks", "HOLLOWAY"},
	}
	for _, c := range cases {
		e := newEngine(t, func(o *Options) { o.Policy.DateShift = false })
		s := e.NewSession("")
		masked, err := s.Mask(ctx, c.text)
		if err != nil || s.Err() != nil {
			t.Fatalf("%q: mask error %v %v", c.text, err, s.Err())
		}
		if strings.Contains(masked, c.real) {
			t.Errorf("%q: real street leaked in %q", c.text, masked)
		}
		if got := s.Rehydrate(masked); got != c.text {
			t.Errorf("rehydrate %q -> %q, want %q", masked, got, c.text)
		}
		for size := 1; size <= 5; size++ {
			if got := streamAll(s, masked, size); got != c.text {
				t.Errorf("stream size %d: %q -> %q, want %q", size, masked, got, c.text)
			}
		}
	}
}

func TestIDsAndSecretsFollowStripPolicy(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("")
	text := "Passport no: 533401872, EIN 47-2918834, DNI 48211973K, token: a8Kq2mZx91, the secret is Zephyr$4411, raw 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12 here."
	masked, err := s.Mask(ctx, text)
	if err != nil || s.Err() != nil {
		t.Fatalf("mask error %v %v", err, s.Err())
	}
	for _, real := range []string{"533401872", "47-2918834", "48211973K", "a8Kq2mZx91", "Zephyr$4411", "2fd4e1c67a2d28fced849ee1bb76e7391b93eb12"} {
		if strings.Contains(masked, real) {
			t.Errorf("%q leaked in %q", real, masked)
		}
	}
	if strings.Count(masked, "[redacted ") != 6 {
		t.Errorf("expected six strip placeholders in %q", masked)
	}
	code := "commit 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12\ntoken = response.json()[\"token\"]\nthe password is required, v1.2.3 on host-01"
	if got, _ := e.NewSession("").Mask(ctx, code); got != code {
		t.Errorf("technical text changed: %q", got)
	}
}
