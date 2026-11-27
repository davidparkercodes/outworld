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
	{"fr", "Rendez-vous au 12 rue de la Paix, 75002 Paris demain", "12 rue de la Paix, 75002 Paris"},
	{"fr", "livraison au 12 bis rue du Général Leclerc, 17000 La Rochelle demain", "12 bis rue du Général Leclerc, 17000 La Rochelle"},
	{"fr", "il habite 25 rue Jean-Jaurès, 13100 Aix-en-Provence depuis mai", "25 rue Jean-Jaurès, 13100 Aix-en-Provence"},
	{"es", "vivo en Calle Mayor de Triana 4, 41010 Sevilla desde mayo", "Calle Mayor de Triana 4, 41010 Sevilla"},
	{"es", "oficina en Paseo de la Castellana 100, 20003 San Sebastián ahora", "Paseo de la Castellana 100, 20003 San Sebastián"},
	{"es", "piso en Calle del Olmo 5, 2º B, 28012 Madrid desde mayo", "Calle del Olmo 5, 2º B, 28012 Madrid"},
	{"pt", "mora na Avenida da Liberdade 110, 1250-146 Lisboa desde maio", "Avenida da Liberdade 110, 1250-146 Lisboa"},
	{"pt", "mora no Largo do Carmo 3, 6000-100 Castelo Branco desde maio", "Largo do Carmo 3, 6000-100 Castelo Branco"},
	{"it", "abita in Via Vittorio Emanuele II 12, 00186 Roma da maggio", "Via Vittorio Emanuele II 12, 00186 Roma"},
	{"it", "abita in Corso Giuseppe Mazzini 4, 42121 Reggio Emilia da maggio", "Corso Giuseppe Mazzini 4, 42121 Reggio Emilia"},
	{"de", "Treffpunkt: Unter den Linden 5, 10117 Berlin, um neun", "Unter den Linden 5, 10117 Berlin"},
	{"nl", "woont op Stationsweg 7, 2515 BM Den Haag nu", "Stationsweg 7, 2515 BM Den Haag"},
	{"uk", "send it to 12 Park View Road, Burgess Hill RH15 9AA please", "12 Park View Road, Burgess Hill RH15 9AA"},
	{"uk", "send it to 3 Station Road, Stoke-on-Trent ST4 2AA please", "3 Station Road, Stoke-on-Trent ST4 2AA"},
	{"fr", "il habite 18 boulevard Saint-Michel, 94100 Saint-Maur-des-Fossés depuis mai", "18 boulevard Saint-Michel, 94100 Saint-Maur-des-Fossés"},
	{"it", "abita in Via dei Mille 9, 36061 Bassano del Grappa da maggio", "Via dei Mille 9, 36061 Bassano del Grappa"},
	{"ch", "Lieferung an Bahnhofstrasse 21, 8001 Zürich bitte", "Bahnhofstrasse 21, 8001 Zürich"},
	{"nl", "woont op Prins Hendrikkade 33, 1012 TM Amsterdam nu", "Prins Hendrikkade 33, 1012 TM Amsterdam"},
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

func TestWideLocaleAddressesRoundTrip(t *testing.T) {
	cases := []struct{ locale, text, addr string }{
		{"es", "vivo en 4 Calle Mayor, 28013 Madrid desde mayo", "4 Calle Mayor, 28013 Madrid"},
		{"es", "vive en Calle Real 8, 11403 Jerez de la Frontera desde mayo", "Calle Real 8, 11403 Jerez de la Frontera"},
		{"pt", "mora na Rua Doutor António Cândido 7, 3º Esq, 4400-001 Vila Nova de Gaia desde maio", "Rua Doutor António Cândido 7, 3º Esq, 4400-001 Vila Nova de Gaia"},
		{"it", "abita in 12 Via Roma, 20121 Milano da maggio", "12 Via Roma, 20121 Milano"},
		{"de", "wohnt in der Konrad-Adenauer-Straße 12, 61348 Bad Homburg seit Mai", "Konrad-Adenauer-Straße 12, 61348 Bad Homburg"},
		{"de", "wohnt in 5 Hauptstraße, 10117 Berlin seit Mai", "5 Hauptstraße, 10117 Berlin"},
		{"ch", "Lieferung an Bahnhofstrasse 21, CH-8001 Zürich bitte", "Bahnhofstrasse 21, CH-8001 Zürich"},
		{"nl", "woont op Burgemeester de Vlugtlaan 125, 1063 BJ Amsterdam nu", "Burgemeester de Vlugtlaan 125, 1063 BJ Amsterdam"},
		{"uk", "send it to 3 Station Road, Stoke on Trent ST4 2AA please", "3 Station Road, Stoke on Trent ST4 2AA"},
		{"uk", "send it to 21 Grey Street, Newcastle upon Tyne NE1 6EE please", "21 Grey Street, Newcastle upon Tyne NE1 6EE"},
		{"uk", "send it to Flat 3, 14 Church Lane, Leeds, UK please", "Flat 3, 14 Church Lane, Leeds, UK"},
		{"uk", "send it to 14 Church Lane, Leeds, UK please", "14 Church Lane, Leeds, UK"},
		{"uk", "27 Glenorchy Road\nPerth PH1 5XY\nUnited Kingdom", "27 Glenorchy Road\nPerth PH1 5XY\nUnited Kingdom"},
		{"uk", "send it to 27 Glenorchy Road, Flat 3, Perth PH1 5XY please", "27 Glenorchy Road, Flat 3, Perth PH1 5XY"},
		{"uk", "she moved to Burgess Hill RH15 9AA last year", "Burgess Hill RH15 9AA"},
		{"pt", "mora na 12 Rua das Flores, 4000-123 Porto desde maio", "12 Rua das Flores, 4000-123 Porto"},
		{"nl", "woont op 5 Kerkstraat, 1017 GC Amsterdam nu", "5 Kerkstraat, 1017 GC Amsterdam"},
		{"at", "Lieferung an Stephansplatz 3, A-1010 Wien bitte", "Stephansplatz 3, A-1010 Wien"},
		{"at", "Lieferung an Mariahilfer Straße 88, Top 3, 1070 Wien bitte", "Mariahilfer Straße 88, Top 3, 1070 Wien"},
		{"de", "Büro: Hauptstraße 5, D-10117 Berlin, zweiter Stock", "Hauptstraße 5, D-10117 Berlin"},
		{"de", "Anschrift:\nLindenstraße 8\n50674 Köln\nDeutschland\nbitte zweimal klingeln", "Lindenstraße 8\n50674 Köln\nDeutschland"},
		{"de", "LIEFERN AN: HAUPTSTRASSE 5, 10117 BERLIN", "HAUPTSTRASSE 5, 10117 BERLIN"},
		{"fr", "livré au 12 rue de Rivoli, F-75001 Paris hier", "12 rue de Rivoli, F-75001 Paris"},
		{"fr", "il habite 14 rue des Lilas, Appt 12, 69003 Lyon depuis mai", "14 rue des Lilas, Appt 12, 69003 Lyon"},
		{"fr", "14 rue des Lilas\n69003 Lyon\nFrance", "14 rue des Lilas\n69003 Lyon\nFrance"},
		{"es", "vivo en Calle Alcalde Moreno 21, 41004 Sevilla, España desde mayo", "Calle Alcalde Moreno 21, 41004 Sevilla, España"},
		{"es", "ENVIAR A: CALLE MAYOR 4, 28013 MADRID", "CALLE MAYOR 4, 28013 MADRID"},
		{"it", "via Garibaldi 30, 10122 Torino è casa mia", "via Garibaldi 30, 10122 Torino"},
		{"it", "Via Garibaldi 30\n10122 Torino\nItalia", "Via Garibaldi 30\n10122 Torino\nItalia"},
		{"nl", "woont op Prof. Lorentzlaan 12, 3707 HC Zeist nu", "Prof. Lorentzlaan 12, 3707 HC Zeist"},
		{"nl", "Herengracht 401\n1017 BP Amsterdam\nNederland", "Herengracht 401\n1017 BP Amsterdam\nNederland"},
		{"", "Adresse: 10117 Berlin", "10117 Berlin"},
		{"", "Postfach 1234\n60311 Frankfurt am Main", "60311 Frankfurt am Main"},
	}
	for _, c := range cases {
		e := newEngine(t, func(o *Options) { o.Policy.DateShift = false })
		s := e.NewSession("")
		masked, err := s.Mask(ctx, c.text)
		if err != nil || s.Err() != nil {
			t.Fatalf("%s %q: mask error %v %v", c.locale, c.text, err, s.Err())
		}
		sur, ok := e.Vault().Get("address", c.addr)
		if !ok || sur == c.addr {
			t.Errorf("%s: %q was not masked as one address in %q", c.locale, c.addr, masked)
			continue
		}
		if want := strings.Replace(c.text, c.addr, sur, 1); masked != want {
			t.Errorf("%s: masked text %q, want %q", c.locale, masked, want)
		}
		if len(strings.Fields(sur)) != len(strings.Fields(c.addr)) {
			t.Errorf("%s: surrogate %q does not keep the word count of %q", c.locale, sur, c.addr)
		}
		if got := s.Rehydrate(masked); got != c.text {
			t.Errorf("%s: rehydrate %q -> %q, want %q", c.locale, masked, got, c.text)
		}
		if again, err := e.NewSession("").Mask(ctx, c.text); err != nil || again != masked {
			t.Errorf("%s: second mask %q differs from %q (%v)", c.locale, again, masked, err)
		}
		for size := 1; size <= 5; size++ {
			if got := streamBytes(s, masked, size, 0); got != c.text {
				t.Errorf("%s: byte stream size %d: %q -> %q, want %q", c.locale, size, masked, got, c.text)
			}
		}
	}
}

func TestAddressReviewCasesRoundTrip(t *testing.T) {
	cases := []struct {
		text     string
		gone     []string
		mayBlock bool
	}{
		{text: "Hier ist meine neue Adresse. Hauptstraße 5, 10115 Berlin", gone: []string{"Hauptstraße", "10115", "Berlin"}},
		{text: "Mein Name ist Anna. Hauptstraße 5, 10115 Berlin", gone: []string{"Hauptstraße", "10115", "Berlin"}},
		{text: "Thanks. Keizersgracht 123, 1015 CJ Amsterdam", gone: []string{"Keizersgracht", "1015 CJ", "Amsterdam"}},
		{text: "Ship it to Rua Augusta 100, 1100-048 Lisboa Maria Kowalski Lopez said hello", gone: []string{"Augusta", "1100-048", "Lisboa", "Maria", "Kowalski", "Lopez"}},
		{text: "Hauptstraße 5, 10115 Berlin Nguyễn Thị Minh Khai.", gone: []string{"Hauptstraße", "10115", "Berlin", "Nguyễn", "Minh", "Khai"}},
		{text: "Via Roma 10, 00184 Roma Zoë Müller-Öztürk", gone: []string{"00184", "Zoë", "Müller", "Öztürk"}, mayBlock: true},
		{text: "Via Roma 10, 00184 Roma. Zoë Müller-Öztürk", gone: []string{"00184", "Roma", "Zoë", "Müller", "Öztürk"}},
		{text: "Calle Mayor 12, 2º B, 28013 Madrid Park Ji-sung", gone: []string{"Mayor", "28013", "Madrid", "Park", "Ji-sung"}},
		{text: "Lieferung an Mariahilfer Straße 45/2/14, 1060 Wien bitte", gone: []string{"Mariahilfer", "45/2/14", "1060", "Wien"}},
		{text: "abita in Corso Giuseppe Mazzini 4, 42121 Reggio Emilia da maggio", gone: []string{"Mazzini", "42121", "Reggio", "Emilia"}},
		{text: "mora no Largo do Carmo 3, 6000-100 Castelo Branco desde maio", gone: []string{"Carmo", "6000-100", "Castelo", "Branco"}},
		{text: "Rua das Flores 12, 4050-262 Porto Maria Kowalski disse", gone: []string{"Flores", "4050-262", "Porto", "Maria", "Kowalski"}},
		{text: "send it to 123 Calle Ocho, Miami, Florida 33135 please"},
		{text: "send it to 123 Via Verde\nSan Diego, California 92101"},
		{text: "she lives at 123 Via Verde now"},
		{text: "send it to 20 Lindenstrasse, Fredericksburg, Texas please"},
		{text: "send it to Flat 2, 22 Court Street, Brooklyn, New York 11201 please"},
		{text: "she lives at 12345 Via Verde now"},
	}
	for _, c := range cases {
		e := newEngine(t, func(o *Options) { o.Policy.DateShift = false })
		s := e.NewSession("")
		masked, err := s.Mask(ctx, c.text)
		if err == nil && s.Err() != nil && c.mayBlock {
			t.Logf("%q: blocked instead of masked: %v", c.text, s.Err())
			continue
		}
		if err != nil || s.Err() != nil {
			t.Errorf("%q: mask error %v %v", c.text, err, s.Err())
			continue
		}
		if masked == c.text && c.gone != nil {
			t.Errorf("%q: nothing was masked", c.text)
		}
		for _, w := range c.gone {
			if strings.Contains(masked, w) {
				t.Errorf("%q: masked text %q still carries %q", c.text, masked, w)
			}
		}
		if got := s.Rehydrate(masked); got != c.text {
			t.Errorf("%q: rehydrate %q -> %q", c.text, masked, got)
		}
		if again, err := e.NewSession("").Mask(ctx, c.text); err != nil || again != masked {
			t.Errorf("%q: second mask %q differs from %q (%v)", c.text, again, masked, err)
		}
		for size := 1; size <= 4; size++ {
			if got := streamBytes(s, masked, size, 0); got != c.text {
				t.Errorf("%q: byte stream size %d: %q -> %q", c.text, size, masked, got)
			}
		}
	}
}
