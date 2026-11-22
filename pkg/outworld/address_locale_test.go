package outworld

import (
	"strings"
	"testing"
)

var localeAddressCases = []struct{ locale, text, addr string }{
	{"fr", "Envoyer à Claire Dubois, 12 rue de Rivoli, 75001 Paris.", "12 rue de Rivoli, 75001 Paris"},
	{"fr", "livré au 7 bis avenue de l'Europe hier", "7 bis avenue de l'Europe"},
	{"es", "vivo en Calle Alcalde Moreno 21, 41004 Sevilla desde mayo", "Calle Alcalde Moreno 21, 41004 Sevilla"},
	{"pt", "mora na Rua das Flores 12, 4050-262 Porto desde maio", "Rua das Flores 12, 4050-262 Porto"},
	{"it", "abita in Via Garibaldi 30, 10122 Torino da maggio", "Via Garibaldi 30, 10122 Torino"},
	{"de", "wohnt in der Lindenstraße 8, 50674 Köln seit Mai", "Lindenstraße 8, 50674 Köln"},
	{"de", "Büro: Berliner Straße 44, 60311 Frankfurt am Main, zweiter Stock", "Berliner Straße 44, 60311 Frankfurt am Main"},
	{"at", "Lieferung an Mariahilfer Straße 88, 1070 Wien bitte", "Mariahilfer Straße 88, 1070 Wien"},
	{"nl", "woont op Herengracht 401, 1017 BP Amsterdam nu", "Herengracht 401, 1017 BP Amsterdam"},
	{"uk", "send it to Flat 4, 27 Glenorchy Road, Perth PH1 5XY please", "Flat 4, 27 Glenorchy Road, Perth PH1 5XY"},
}

func streamBytes(s *Session, text string, size, offset int) string {
	r := s.NewStreamRehydrator(false)
	var b strings.Builder
	first := min(offset, len(text))
	b.WriteString(r.Push(text[:first]))
	for i := first; i < len(text); i += size {
		b.WriteString(r.Push(text[i:min(i+size, len(text))]))
	}
	b.WriteString(r.Flush())
	return b.String()
}

func TestLocaleAddressesRoundTrip(t *testing.T) {
	for _, c := range localeAddressCases {
		e := newEngine(t, func(o *Options) { o.Policy.DateShift = false })
		s := e.NewSession("")
		masked, err := s.Mask(ctx, c.text)
		if err != nil || s.Err() != nil {
			t.Fatalf("%s %q: mask error %v %v", c.locale, c.text, err, s.Err())
		}
		sur, ok := e.Vault().Get("address", c.addr)
		if !ok {
			t.Errorf("%s: %q was not masked as one address in %q", c.locale, c.addr, masked)
			continue
		}
		if !strings.Contains(masked, sur) || strings.Contains(masked, c.addr) {
			t.Errorf("%s: masked text %q should carry %q", c.locale, masked, sur)
		}
		if got := s.Rehydrate(masked); got != c.text {
			t.Errorf("%s: rehydrate %q -> %q, want %q", c.locale, masked, got, c.text)
		}
		if got := e.NewSession("").Rehydrate("Echo: " + sur + "!"); got != "Echo: "+c.addr+"!" {
			t.Errorf("%s: echoed surrogate %q -> %q", c.locale, sur, got)
		}
		for size := 1; size <= 5; size++ {
			if got := streamAll(s, masked, size); got != c.text {
				t.Errorf("%s: rune stream size %d: %q -> %q, want %q", c.locale, size, masked, got, c.text)
			}
		}
		for size := 1; size <= 7; size++ {
			for offset := 0; offset < size; offset++ {
				if got := streamBytes(s, masked, size, offset); got != c.text {
					t.Errorf("%s: byte stream size %d offset %d: %q -> %q, want %q", c.locale, size, offset, masked, got, c.text)
				}
			}
		}
	}
}

func TestStreamRehydratorHoldsSplitCharacter(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Policy.DateShift = false })
	const real, sur = "Hauptstraße 5", "Müllerstraße 12"
	if _, err := e.Vault().Assign("address", real, func(int) string { return sur }); err != nil {
		t.Fatal(err)
	}
	s := e.NewSession("")
	cut := strings.Index(sur, "ü") + 1
	r := s.NewStreamRehydrator(false)
	got := r.Push("in "+sur[:cut]) + r.Push(sur[cut:]+" now") + r.Flush()
	if want := "in " + real + " now"; got != want {
		t.Fatalf("split inside a character: got %q, want %q", got, want)
	}
}

func TestLocaleAddressSurrogatesSurviveRemasking(t *testing.T) {
	for _, c := range localeAddressCases {
		e := newEngine(t, func(o *Options) { o.Policy.DateShift = false })
		s1 := e.NewSession("")
		masked, err := s1.Mask(ctx, c.text)
		if err != nil || s1.Err() != nil {
			t.Fatalf("%s: mask error %v %v", c.locale, err, s1.Err())
		}
		sur, _ := e.Vault().Get("address", c.addr)
		s2 := e.NewSession("")
		again, err := s2.Mask(ctx, masked)
		if err != nil || s2.Err() != nil {
			t.Fatalf("%s: re-mask of %q failed: %v %v", c.locale, masked, err, s2.Err())
		}
		if strings.Contains(again, sur) {
			t.Errorf("%s: surrogate %q was not recognised as an address again in %q", c.locale, sur, again)
		}
		back := s2.Rehydrate(again)
		if back != masked {
			t.Errorf("%s: re-masked %q rehydrates to %q, want %q", c.locale, again, back, masked)
		}
		if got := e.NewSession("").Rehydrate(back); got != c.text {
			t.Errorf("%s: second rehydration %q -> %q, want %q", c.locale, back, got, c.text)
		}
		for size := 1; size <= 4; size++ {
			if got := streamBytes(s2, again, size, 0); got != masked {
				t.Errorf("%s: stream of re-masked text size %d: %q, want %q", c.locale, size, got, masked)
			}
		}
	}
}
