package detect

import (
	"bytes"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/surrogate"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

var postcodeFirstCases = []struct{ text, want, post, city string }{
	{"livré au 75001 Paris, 12 rue de Rivoli hier", "75001 Paris, 12 rue de Rivoli", "75001", "Paris"},
	{"il habite 69003 Lyon, 14 rue des Lilas depuis mai", "69003 Lyon, 14 rue des Lilas", "69003", "Lyon"},
	{"vivo en 28013 Madrid, Calle Mayor 12 desde mayo", "28013 Madrid, Calle Mayor 12", "28013", "Madrid"},
	{"vivo en 41013 Sevilla, Plaza de España 3 desde mayo", "41013 Sevilla, Plaza de España 3", "41013", "Sevilla"},
	{"mora na 1250-146 Lisboa, Avenida da Liberdade 110 desde maio", "1250-146 Lisboa, Avenida da Liberdade 110", "1250-146", "Lisboa"},
	{"abita in 20121 Milano, Via Roma 12 da maggio", "20121 Milano, Via Roma 12", "20121", "Milano"},
	{"Adresse: 10115 Berlin, Hauptstraße 5", "10115 Berlin, Hauptstraße 5", "10115", "Berlin"},
	{"Lieferung an 1070 Wien, Mariahilfer Straße 88 bitte", "1070 Wien, Mariahilfer Straße 88", "1070", "Wien"},
	{"Lieferung an CH-8001 Zürich, Bahnhofstrasse 21 bitte", "CH-8001 Zürich, Bahnhofstrasse 21", "8001", "Zürich"},
	{"woont op 1017 BP Amsterdam, Herengracht 401 nu", "1017 BP Amsterdam, Herengracht 401", "1017 BP", "Amsterdam"},
}

func TestPostcodeFirstAddressSpansAreWhole(t *testing.T) {
	for _, c := range postcodeFirstCases {
		got := addressSpans(t, c.text)
		if len(got) != 1 || got[0].Text != c.want {
			t.Errorf("%q: want one address %q, got %+v", c.text, c.want, got)
		}
	}
}

func TestPostcodeFirstMaskedOutputDropsRealPostcodeAndCity(t *testing.T) {
	g := surrogate.NewGenerator([]byte("postcode-first"))
	for _, c := range postcodeFirstCases {
		spans := addressSpans(t, c.text)
		if len(spans) != 1 {
			t.Errorf("%q: want one address, got %+v", c.text, spans)
			continue
		}
		s := spans[0]
		for attempt := 0; attempt < 20; attempt++ {
			sur := g.Address(s.Text, attempt)
			masked := c.text[:s.Start] + sur + c.text[s.End:]
			if strings.Contains(masked, c.post) || strings.Contains(masked, c.city) {
				t.Errorf("%q: masked %q keeps the real postcode or city", c.text, masked)
				break
			}
			k := strings.Index(sur, ", ")
			if k < 0 || !strings.ContainsAny(sur[:k], "0123456789") {
				t.Errorf("%q: surrogate %q lost the postcode-first shape", c.text, sur)
				break
			}
			again := addressSpans(t, masked)
			if len(again) != 1 || again[0].Text != sur {
				t.Errorf("%q: surrogate %q is not re-detected as one address, got %+v", c.text, sur, again)
				break
			}
		}
	}
}

func TestPostcodeFirstNeedsAStreet(t *testing.T) {
	for _, text := range []string{"I moved to 75001 Paris last year", "Order 28013 Madrid, shipped today", "10115 Berlin, Hauptbahnhof", "Mara Ellison\n10117 Berlin, Hauptstraße"} {
		for _, s := range addressSpans(t, text) {
			if strings.Contains(s.Text, ", ") {
				t.Errorf("%q: unexpected postcode-first address %+v", text, s)
			}
		}
	}
}

func maskAddresses(t *testing.T, g *surrogate.Generator, text string, attempt int) (string, [][2]int) {
	t.Helper()
	var b strings.Builder
	var ins [][2]int
	last := 0
	for _, sp := range addressSpans(t, text) {
		b.WriteString(text[last:sp.Start])
		sur := g.Address(sp.Text, attempt)
		ins = append(ins, [2]int{b.Len(), b.Len() + len(sur)})
		b.WriteString(sur)
		last = sp.End
	}
	b.WriteString(text[last:])
	return b.String(), ins
}

func checkMasked(t *testing.T, text string, reals []string) {
	t.Helper()
	g := surrogate.NewGenerator([]byte("review"))
	for attempt := 0; attempt < 20; attempt++ {
		masked, ins := maskAddresses(t, g, text, attempt)
		for _, r := range reals {
			if strings.Contains(masked, r) {
				t.Errorf("%q: masked %q keeps %q", text, masked, r)
				return
			}
		}
		for _, sp := range addressSpans(t, masked) {
			inside := false
			for _, in := range ins {
				if sp.Start >= in[0] && sp.End <= in[1] {
					inside = true
				}
			}
			if !inside {
				t.Errorf("%q: masked %q has address residue %q outside the stand-ins", text, masked, sp.Text)
				return
			}
		}
	}
}

func TestBackToBackAddressesKeepTheirOwnPostcode(t *testing.T) {
	cases := []struct {
		text  string
		want  []string
		reals []string
	}{
		{"Hauptstraße 5, 10115 Berlin, Lindenstraße 8", []string{"Hauptstraße 5, 10115 Berlin", "Lindenstraße 8"}, []string{"10115", "Berlin"}},
		{"12 rue de Rivoli, 75001 Paris, 14 rue des Lilas", []string{"12 rue de Rivoli, 75001 Paris", "14 rue des Lilas"}, []string{"75001", "Paris"}},
		{"Calle Mayor 4, 28013 Madrid, Calle del Olmo 5", []string{"Calle Mayor 4, 28013 Madrid", "Calle del Olmo 5"}, []string{"28013", "Madrid"}},
	}
	for _, c := range cases {
		got := addressSpans(t, c.text)
		if len(got) != len(c.want) {
			t.Errorf("%q: want %q, got %+v", c.text, c.want, got)
			continue
		}
		for i := range got {
			if got[i].Text != c.want[i] {
				t.Errorf("%q: span %d = %q, want %q", c.text, i, got[i].Text, c.want[i])
			}
		}
		checkMasked(t, c.text, c.reals)
	}
}

func TestPostcodeFirstOddShapesAreMaskedWhole(t *testing.T) {
	cases := []struct {
		text  string
		want  string
		reals []string
	}{
		{"10115 Berlin, Hauptstraße 5, 10117 Berlin", "10115 Berlin, Hauptstraße 5, 10117 Berlin", []string{"10115", "10117", "Berlin", "Hauptstraße"}},
		{"75001 Paris, 12 rue de Rivoli, 75002 Paris", "75001 Paris, 12 rue de Rivoli, 75002 Paris", []string{"75001", "75002", "Paris", "Rivoli"}},
		{"75001 Paris,  12 rue de Rivoli", "75001 Paris,  12 rue de Rivoli", []string{"75001", "Paris", "Rivoli"}},
		{"75001 Paris , 12 rue de Rivoli", "75001 Paris , 12 rue de Rivoli", []string{"75001", "Paris", "Rivoli"}},
		{"id=28013 Madrid, Calle Mayor 12", "28013 Madrid, Calle Mayor 12", []string{"28013", "Madrid", "Mayor"}},
		{"order 28013 Madrid, Calle Mayor 12", "28013 Madrid, Calle Mayor 12", []string{"28013", "Madrid", "Mayor"}},
		{"x 75001 Paris, 12 rue de Rivoli", "75001 Paris, 12 rue de Rivoli", []string{"75001", "Paris", "Rivoli"}},
		{"Error 10115 Berlin, Hauptstraße 5", "10115 Berlin, Hauptstraße 5", []string{"10115", "Berlin", "Hauptstraße"}},
	}
	for _, c := range cases {
		got := addressSpans(t, c.text)
		if len(got) != 1 || got[0].Text != c.want {
			t.Errorf("%q: want one address %q, got %+v", c.text, c.want, got)
			continue
		}
		checkMasked(t, c.text, c.reals)
	}
}

func TestNumberAndWordBeforeAStreetAreNotAPostcode(t *testing.T) {
	cases := []struct{ text, keep, street string }{
		{"run 12345 Done, Calle Mayor 12", "run 12345 Done, ", "Calle Mayor 12"},
		{"INFO 12345 Worker, Calle Mayor 12", "INFO 12345 Worker, ", "Calle Mayor 12"},
		{"commit 10115 Fix, Hauptstraße 5", "commit 10115 Fix, ", "Hauptstraße 5"},
		{"PR 10115 Merged, Hauptstraße 5", "PR 10115 Merged, ", "Hauptstraße 5"},
		{"Total 99999 Units, Via Roma 2", "Total 99999 Units, ", ""},
		{"build 12345 Failed, Calle Mayor 12", "build 12345 Failed, ", "Calle Mayor 12"},
		{"run 12345 Done, Calle Mayor 12", "run 12345 Done, ", "Calle Mayor 12"},
	}
	g := surrogate.NewGenerator([]byte("review"))
	for _, c := range cases {
		got := addressSpans(t, c.text)
		if c.street == "" {
			if len(got) != 0 {
				t.Errorf("%q: want no address, got %+v", c.text, got)
			}
			continue
		}
		if len(got) != 1 || got[0].Text != c.street {
			t.Errorf("%q: want only %q, got %+v", c.text, c.street, got)
			continue
		}
		if masked, _ := maskAddresses(t, g, c.text, 0); !strings.HasPrefix(masked, c.keep) {
			t.Errorf("%q: masked %q changed the log text", c.text, masked)
		}
	}
	for _, text := range []string{"vivo en 28013 Madrid, Calle Mayor 12", "Ship to 75001 Paris, 12 rue de Rivoli", "my address is 10115 Berlin, Hauptstraße 5"} {
		if got := addressSpans(t, text); len(got) != 1 || !strings.Contains(got[0].Text, ", ") {
			t.Errorf("%q: a lead word must still allow postcode-first, got %+v", text, got)
		}
	}
}

func TestOddSpacingAndRepeatedPostcodeMaskAndRoundTrip(t *testing.T) {
	cases := []struct {
		text  string
		reals []string
	}{
		{"Hauptstraße  5 ,  10115   Berlin", []string{"10115", "Berlin", "Hauptstraße"}},
		{"Hauptstraße\t5,\t10115 Berlin", []string{"10115", "Berlin", "Hauptstraße"}},
		{"12  rue de  Rivoli ,  75001  Paris", []string{"75001", "Paris", "Rivoli"}},
		{"Calle  Mayor 4 ,  28013 Madrid", []string{"28013", "Madrid", "Mayor"}},
		{"4  Calle Mayor,  28013  Madrid", []string{"28013", "Madrid", "Mayor"}},
		{"Kerkstraat  12 , 3511 AB  Utrecht", []string{"3511", "Utrecht", "Kerkstraat"}},
		{"Via  Roma 12,  20121   Milano", []string{"20121", "Milano", "Roma"}},
		{"Rua das  Flores 12 ,  4050-262 Porto", []string{"4050-262", "Porto", "Flores"}},
		{"12 rue de Rivoli, 75001 75001 Paris", []string{"75001", "Paris", "Rivoli"}},
		{"Hauptstraße 5, 10115 10115 Berlin", []string{"10115", "Berlin"}},
	}
	for _, c := range cases {
		text := "Lieferung an " + c.text + " bitte"
		got := addressSpans(t, text)
		if len(got) != 1 || got[0].Text != c.text {
			t.Errorf("%q: want the whole address as one span, got %+v", text, got)
			continue
		}
		checkMasked(t, text, c.reals)
		v, err := vault.Open(vault.NewMemoryBackend(bytes.Repeat([]byte{9}, 32)))
		if err != nil {
			t.Fatal(err)
		}
		g := surrogate.NewGenerator(v.Secret())
		sur, err := v.Assign("address", c.text, func(a int) string { return g.Address(c.text, a) })
		if err != nil {
			t.Fatal(err)
		}
		masked := text[:got[0].Start] + sur + text[got[0].End:]
		hits := v.FindSurrogates(masked)
		if len(hits) != 1 {
			t.Errorf("%q: surrogate %q not found in %q", c.text, sur, masked)
			continue
		}
		back := masked[:hits[0].Start] + hits[0].Record.Real + masked[hits[0].End:]
		if back != text {
			t.Errorf("round trip %q -> %q -> %q", text, masked, back)
		}
	}
}
