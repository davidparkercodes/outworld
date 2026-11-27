package detect

import (
	"fmt"
	"github.com/davidparkercodes/outworld/internal/scaletest"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

var addressSpanCases = []struct{ text, want string }{
	{"vivo en 4 Calle Mayor, 28013 Madrid desde mayo", "4 Calle Mayor, 28013 Madrid"},
	{"abita in 12 Via Roma, 20121 Milano da maggio", "12 Via Roma, 20121 Milano"},
	{"wohnt in 5 Hauptstraße, 10117 Berlin seit Mai", "5 Hauptstraße, 10117 Berlin"},
	{"mora na 12 Rua das Flores, 4000-123 Porto desde maio", "12 Rua das Flores, 4000-123 Porto"},
	{"woont op 5 Kerkstraat, 1017 GC Amsterdam nu", "5 Kerkstraat, 1017 GC Amsterdam"},
	{"wohnt in der Konrad-Adenauer-Straße 12, 61348 Bad Homburg seit Mai", "Konrad-Adenauer-Straße 12, 61348 Bad Homburg"},
	{"Lieferung an Bahnhofstrasse 10, CH-8001 Zürich bitte", "Bahnhofstrasse 10, CH-8001 Zürich"},
	{"Lieferung an Stephansplatz 3, A-1010 Wien bitte", "Stephansplatz 3, A-1010 Wien"},
	{"Büro: Hauptstraße 5, D-10117 Berlin, zweiter Stock", "Hauptstraße 5, D-10117 Berlin"},
	{"livré au 12 rue de Rivoli, F-75001 Paris hier", "12 rue de Rivoli, F-75001 Paris"},
	{"send it to 3 Grey Street, Newcastle upon Tyne NE1 6EE please", "3 Grey Street, Newcastle upon Tyne NE1 6EE"},
	{"send it to 3 Station Road, Stoke on Trent ST4 2AA please", "3 Station Road, Stoke on Trent ST4 2AA"},
	{"send it to 12 Park View Road, Burgess Hill RH15 9AA please", "12 Park View Road, Burgess Hill RH15 9AA"},
	{"she moved to Burgess Hill RH15 9AA last year", "Burgess Hill RH15 9AA"},
	{"send it to 14 Church Lane, Leeds, UK please", "14 Church Lane, Leeds, UK"},
	{"send it to Flat 3, 14 Church Lane, Leeds, UK please", "Flat 3, 14 Church Lane, Leeds, UK"},
	{"vive en Calle Real 8, 11403 Jerez de la Frontera desde mayo", "Calle Real 8, 11403 Jerez de la Frontera"},
	{"mora na Rua Doutor António Cândido 7, 3º Esq, 4400-001 Vila Nova de Gaia desde maio", "Rua Doutor António Cândido 7, 3º Esq, 4400-001 Vila Nova de Gaia"},
	{"wohnt in der Lindenstraße 8, 61348 Bad Homburg seit Mai", "Lindenstraße 8, 61348 Bad Homburg"},
	{"Büro: Berliner Straße 44, 60311 Frankfurt am Main, zweiter Stock", "Berliner Straße 44, 60311 Frankfurt am Main"},
	{"livraison au 12 bis rue du Général Leclerc, 17000 La Rochelle demain", "12 bis rue du Général Leclerc, 17000 La Rochelle"},
	{"il habite 25 rue Jean-Jaurès, 13100 Aix-en-Provence depuis mai", "25 rue Jean-Jaurès, 13100 Aix-en-Provence"},
	{"woont op Stationsweg 7, 2515 BM Den Haag nu", "Stationsweg 7, 2515 BM Den Haag"},
	{"abita in Via dei Mille 9, 36061 Bassano del Grappa da maggio", "Via dei Mille 9, 36061 Bassano del Grappa"},
	{"woont op Burgemeester de Vlugtlaan 125, 1063 BJ Amsterdam nu", "Burgemeester de Vlugtlaan 125, 1063 BJ Amsterdam"},
	{"woont op Prof. Lorentzlaan 12, 3707 HC Zeist nu", "Prof. Lorentzlaan 12, 3707 HC Zeist"},
	{"woont op Dr. Schaepmanstraat 4, 2032 GM Haarlem nu", "Dr. Schaepmanstraat 4, 2032 GM Haarlem"},
	{"woont op Mr. Troelstralaan 9, 1181 VD Amstelveen nu", "Mr. Troelstralaan 9, 1181 VD Amstelveen"},
	{"woont op Sint Jacobsstraat 14, 1012 NC Amsterdam nu", "Sint Jacobsstraat 14, 1012 NC Amsterdam"},
	{"woont op Koningin Wilhelminaplein 30, 1062 KR Amsterdam nu", "Koningin Wilhelminaplein 30, 1062 KR Amsterdam"},
	{"piso en Calle del Olmo 5, 2º B, 28012 Madrid desde mayo", "Calle del Olmo 5, 2º B, 28012 Madrid"},
	{"il habite 14 bis rue des Lilas, 69003 Lyon depuis mai", "14 bis rue des Lilas, 69003 Lyon"},
	{"il habite 14 rue des Lilas, Appt 12, 69003 Lyon depuis mai", "14 rue des Lilas, Appt 12, 69003 Lyon"},
	{"Lieferung an Mariahilfer Straße 88, Top 3, 1070 Wien bitte", "Mariahilfer Straße 88, Top 3, 1070 Wien"},
	{"Lieferung an Mariahilfer Straße 88/Top 3, 1070 Wien bitte", "Mariahilfer Straße 88/Top 3, 1070 Wien"},
	{"send it to 27 Glenorchy Road, Flat 3, Perth PH1 5XY please", "27 Glenorchy Road, Flat 3, Perth PH1 5XY"},
	{"vivo en Calle Alcalde Moreno 21, 41004 Sevilla, España desde mayo", "Calle Alcalde Moreno 21, 41004 Sevilla, España"},
	{"Anschrift:\nLindenstraße 8\n50674 Köln\nDeutschland\nTel. 030 12345 678", "Lindenstraße 8\n50674 Köln\nDeutschland"},
	{"14 rue des Lilas\n69003 Lyon\nFrance", "14 rue des Lilas\n69003 Lyon\nFrance"},
	{"Via Garibaldi 30\n10122 Torino\nItalia", "Via Garibaldi 30\n10122 Torino\nItalia"},
	{"Rua das Flores 12\n4050-262 Porto\nPortugal", "Rua das Flores 12\n4050-262 Porto\nPortugal"},
	{"Herengracht 401\n1017 BP Amsterdam\nNederland", "Herengracht 401\n1017 BP Amsterdam\nNederland"},
	{"27 Glenorchy Road\nPerth PH1 5XY\nUnited Kingdom", "27 Glenorchy Road\nPerth PH1 5XY\nUnited Kingdom"},
	{"Mara Ellison\nc/o Tessler Logistik\nLindenstraße 8\n50674 Köln", "Lindenstraße 8\n50674 Köln"},
	{"Mara Ellison\nBergmannstrasse 102a\n10961 Berlin\n+49 30 1234567", "Bergmannstrasse 102a\n10961 Berlin"},
	{"Mara Ellison\nCalle Mayor 4\n28013 Madrid\nmara@example.com", "Calle Mayor 4\n28013 Madrid"},
	{"ENVIAR A: CALLE MAYOR 4, 28013 MADRID", "CALLE MAYOR 4, 28013 MADRID"},
	{"LIVRER AU 12 RUE DE RIVOLI, 75001 PARIS", "12 RUE DE RIVOLI, 75001 PARIS"},
	{"LIEFERN AN: HAUPTSTRASSE 5, 10117 BERLIN", "HAUPTSTRASSE 5, 10117 BERLIN"},
	{"SHIP TO: 3 GREY STREET, NEWCASTLE UPON TYNE NE1 6EE", "3 GREY STREET, NEWCASTLE UPON TYNE NE1 6EE"},
	{"calle Mayor 4, 28013 Madrid es mi casa", "calle Mayor 4, 28013 Madrid"},
	{"via Garibaldi 30, 10122 Torino è casa mia", "via Garibaldi 30, 10122 Torino"},
	{"rua das Flores 12, 4050-262 Porto fica longe", "rua das Flores 12, 4050-262 Porto"},
	{"Wir treffen uns in der Lindenstraße 8, 50674 Köln.", "Lindenstraße 8, 50674 Köln"},
	{"(Lindenstraße 8, 50674 Köln)", "Lindenstraße 8, 50674 Köln"},
	{"Her flat (3 Grey Street, Newcastle upon Tyne NE1 6EE) is small.", "3 Grey Street, Newcastle upon Tyne NE1 6EE"},
	{"Vive en Calle Real 8, 11403 Jerez de la Frontera.", "Calle Real 8, 11403 Jerez de la Frontera"},
	{"Adresse: 10117 Berlin", "10117 Berlin"},
	{"Mara Ellison\nPostfach 1234\n60311 Frankfurt am Main", "60311 Frankfurt am Main"},
	{"Mara Ellison\nKungsbacken 12\n4400-001 Vila Nova de Gaia", "4400-001 Vila Nova de Gaia"},
	{"Send to 3 Grey Street, Newcastle upon Tyne NE1 6EE, UK today", "3 Grey Street, Newcastle upon Tyne NE1 6EE, UK"},
	{"12 rue de Rivoli. Demain on y va", "12 rue de Rivoli"},
	{"Calle Mayor 4 bis, 28013 Madrid", "Calle Mayor 4 bis, 28013 Madrid"},
	{"Lindenstraße 8 · 50674 Köln", "Lindenstraße 8 · 50674 Köln"},
	{"Via Roma 12, 20121 Milano (MI)", "Via Roma 12, 20121 Milano (MI)"},
	{"Hauptstraße 5 bis 7 sind gesperrt", "Hauptstraße 5"},
	{"my postcode is NE1 6EE thanks", "NE1 6EE"},
	{"Mara Ellison, Leeds LS1 4AP", "Leeds LS1 4AP"},
	{"Karl-Marx-Allee 34, 10178 Berlin", "Karl-Marx-Allee 34, 10178 Berlin"},
	{"Calle Luna 3, 35001 Las Palmas de Gran Canaria, Spain", "Calle Luna 3, 35001 Las Palmas de Gran Canaria, Spain"},
	{"Hauptstraße 5, 9000 St. Gallen ist schön", "Hauptstraße 5, 9000 St. Gallen"},
	{"Dorpsstraat 1, 2400 AB Alphen aan den Rijn", "Dorpsstraat 1, 2400 AB Alphen aan den Rijn"},
	{"14 Church Lane\nLeeds\nUnited Kingdom", "14 Church Lane\nLeeds\nUnited Kingdom"},
	{"Calle Mayor s/n, 28013 Madrid", "Calle Mayor s/n, 28013 Madrid"},
	{"Lindenstraße 8, 50674 Köln Tel: 0221 123456", "Lindenstraße 8, 50674 Köln"},
	{"Landlord of Prinsengracht 512-II, 1017 KH Amsterdam and the tenant", "Prinsengracht 512-II, 1017 KH Amsterdam"},
	{"From: Storgatan 14, 211 34 Malmö\nBill to: the client", "Storgatan 14, 211 34 Malmö"},
	{"Bill to: Vestergade 22, 8000 Aarhus C\nDate: soon", "Vestergade 22, 8000 Aarhus C"},
	{"piso en Carrer de Mallorca 211, 2n 1a, 08036 Barcelona des de maig", "Carrer de Mallorca 211, 2n 1a, 08036 Barcelona"},
	{"il habite 14 rue des Lilas, 2e étage, 69003 Lyon depuis mai", "14 rue des Lilas, 2e étage, 69003 Lyon"},
}

func addressSpans(t *testing.T, text string) []entity.Span {
	t.Helper()
	var out []entity.Span
	for _, s := range detectAll(t, text) {
		if s.Type == entity.Address {
			out = append(out, s)
		}
	}
	return out
}

func TestAddressSpansAreExact(t *testing.T) {
	for _, c := range addressSpanCases {
		got := addressSpans(t, c.text)
		if len(got) != 1 || got[0].Text != c.want {
			t.Errorf("%q: want one address %q, got %+v", c.text, c.want, got)
		}
	}
}

func TestAddressCareOfLineSplitsTheSpan(t *testing.T) {
	got := addressSpans(t, "Lindenstraße 8, c/o Tessler, 50674 Köln")
	if len(got) != 2 || got[0].Text != "Lindenstraße 8" || got[1].Text != "50674 Köln" {
		t.Errorf("want street and postcode-city as two spans around the c/o part, got %+v", got)
	}
}

func TestAddressNegatives(t *testing.T) {
	runNegatives(t, entity.Address, []string{
		"Via email, 20121 units shipped",
		"Calle 13 released an album",
		"the Berlin office",
		"in Frankfurt am Main we met",
		"Leeds United",
		"Leeds United won 3 Nil at home",
		"Newcastle upon Tyne is rainy",
		"running v2.10117 Berlin build",
		"Order 28013 Madrid shipped",
		"order number 4400-001 Vila",
		"Room 12, Building 4",
		"on 12 March 2024",
		"12 March 2024, London",
		"5 Star Hotel",
		"Mara Ellison\n10117 Berlin",
		"Ticket 10117 Berlin team picked it up",
		"release 2024 Paris edition",
		"5 Hauptstraße-Fans kamen",
		"Die Straße 5 ist gesperrt",
		"Version 12 Via the proxy, 20121 requests",
		"See RFC 2515 BM Den",
		"Prof. Lorentz 12 papers",
		"3 Via points, 20121 Milano total",
		"part number B2 4XY in stock",
		"In 2023, 10117 Berliners came",
		"On March 12, 10117 Berliners came",
		"Montag um 5, 10117 Berliner kamen",
		"Section 2 Via Zoom, 20121 people joined",
		"build 4400-001 Vila passed",
	})
}

func TestBareCityIsNotAnAddress(t *testing.T) {
	for _, text := range []string{"Newcastle upon Tyne is rainy", "in Frankfurt am Main we met", "the Berlin office", "Leeds United"} {
		if got := addressSpans(t, text); len(got) != 0 {
			t.Errorf("%q: a city without street or postcode must not be an address, got %+v", text, got)
		}
	}
}

func TestSlugsAreNotHighEntropySecrets(t *testing.T) {
	slugs := []string{
		"book https://cal.example:8443/team/leocadia-jarrett-q4-planning-sync-2024-11-22 today",
		"open https://cal.example/book?event=leocadia-jarrett-q4-planning-sync-2024-11-22",
		"see https://cal.example/team/mara-ellison-quarterly-planning-sync-2024 for the invite",
		"profile https://people.example:8443/u/dana_whitlock_senior_account_manager_emea_2024",
		"slug leocadia_jarrett_q4_planning_sync_2024_11_22 is taken",
	}
	for _, text := range slugs {
		for _, s := range detectAll(t, text) {
			if s.Type == entity.Secret {
				t.Errorf("%q: word slug flagged as secret %q by %s", text, s.Text, s.Source)
			}
		}
	}
	tokens := []struct{ text, want string }{
		{"get https://files.example:8443/dl/aB3dE5gH7jK9mN1pQ3sT5vX7zA9cE1gI now", "aB3dE5gH7jK9mN1pQ3sT5vX7zA9cE1gI"},
		{"get https://files.example/dl?id=x7k2m9q4-p3z8w1v6-b5n0c2d7-j4h8g3f1", "x7k2m9q4-p3z8w1v6-b5n0c2d7-j4h8g3f1"},
		{"fetch https://api.example/v1/items?access_key=Zx9Qm2Lk8Vt4Rw7Yp1Nc6Hd3Bf5Gj0Sa", "Zx9Qm2Lk8Vt4Rw7Yp1Nc6Hd3Bf5Gj0Sa"},
		{"code a1b2-c3d4-e5f6-g7h8-i9j0-k1l2-m3n4", "a1b2-c3d4-e5f6-g7h8-i9j0-k1l2-m3n4"},
	}
	for _, c := range tokens {
		spans, err := NewRegexDetector().Detect(t.Context(), c.text)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, s := range spans {
			found = found || s.Type == entity.Secret && len(s.Text) >= len(c.want) && s.Text[len(s.Text)-len(c.want):] == c.want
		}
		if !found {
			t.Errorf("%q: token %q must stay a secret, got %+v", c.text, c.want, spans)
		}
	}
}

var addressLongLineShapes = []string{
	"12345 Berlin ",
	"Leeds LS1 4AP ",
	"Hauptstraße 5, 10115 Berlin ",
	"Anna. Hauptstraße 5 ",
	"12 rue de Rivoli, 75001 Paris ",
	"Calle Mayor 4, 28013 Madrid, ",
	"3 Grey Street, Newcastle upon Tyne NE1 6EE ",
	"14 Church Lane, Leeds, UK ",
	"Adresse: 10117 Berlin, ",
	"75001 Paris, 12 rue de Rivoli, ",
	"10115 Berlin, Hauptstraße 5 ",
	"28013 Madrid, Plaza de España 3 ",
	"Hauptstraße  5 ,  10115   Berlin  ",
	"12  rue de  Rivoli ,  75001 75001  Paris\t",
	"Lindenstraße 8, c/o Tessler, 50674 Köln, ",
	"Burgemeester de Vlugtlaan ",
	"Karl-Marx-",
	"Aa ",
	"5 ",
	"St. ",
	", ",
}

func addressScan(text string) {
	lower := strings.ToLower(text)
	for _, p := range addressPatterns {
		if p.gate != nil && !p.gate(text, lower) {
			continue
		}
		p.scan(text, lower)
	}
}

func addressJSONLine(size int) string {
	var js strings.Builder
	js.WriteString(`{"records":[`)
	for i := 0; js.Len() < size; i++ {
		fmt.Fprintf(&js, `{"id":%d,"name":"Mara Ellison","street":"Lindenstraße %d","zip":"50674","city":"Köln","addr":"Calle Mayor %d, 28013 Madrid","uk":"3 Grey Street, Leeds LS1 4AP","note":"Order 28013 Madrid shipped"},`, i, i%90+1, i%90+1)
	}
	js.WriteString(`{}]}`)
	return js.String()
}

func TestAddressScansStayLinearOnOneLongLine(t *testing.T) {
	size := scaletest.Size(scaleGrowthSize)
	builders := map[string]func(int) string{"minified json": addressJSONLine}
	for _, shape := range addressLongLineShapes {
		shape := shape
		builders[shape] = func(n int) string { return strings.Repeat(shape, n/len(shape)+1) }
	}
	for name, build := range builders {
		if strings.Contains(build(size), "\n") {
			t.Fatalf("%q: input must be one line", name)
		}
		checkGrowth(t, fmt.Sprintf("%q", name), build, size, addressScan)
	}
}

func TestAddressAfterSentenceEndIsFound(t *testing.T) {
	cases := []struct{ text, want string }{
		{"Hier ist meine neue Adresse. Hauptstraße 5, 10115 Berlin", "Hauptstraße 5, 10115 Berlin"},
		{"Mein Name ist Anna. Hauptstraße 5, 10115 Berlin", "Hauptstraße 5, 10115 Berlin"},
		{"Thanks. Keizersgracht 123, 1015 CJ Amsterdam", "Keizersgracht 123, 1015 CJ Amsterdam"},
		{"Gracias. Calle Mayor 4, 28013 Madrid", "Calle Mayor 4, 28013 Madrid"},
		{"Ciao Paolo. Via Roma 12, 20121 Milano", "Via Roma 12, 20121 Milano"},
		{"Obrigado. Rua das Flores 12, 4050-262 Porto", "Rua das Flores 12, 4050-262 Porto"},
		{"Erledigt. Berliner Straße 44, 60311 Frankfurt am Main", "Berliner Straße 44, 60311 Frankfurt am Main"},
		{"Fertig. Karl-Marx-Allee 34, 10178 Berlin", "Karl-Marx-Allee 34, 10178 Berlin"},
		{"Alles klar. Am Kupfergraben 6, 10117 Berlin", "Am Kupfergraben 6, 10117 Berlin"},
		{"Merci. 12 rue de Rivoli, 75001 Paris", "12 rue de Rivoli, 75001 Paris"},
		{"Thanks. 3 Grey Street, Newcastle upon Tyne NE1 6EE", "3 Grey Street, Newcastle upon Tyne NE1 6EE"},
		{"Umzug geplant. Neue Anschrift folgt. Lindenstraße 8, 50674 Köln", "Lindenstraße 8, 50674 Köln"},
		{"Bitte an Anna. Hauptstraße 5", "Hauptstraße 5"},
	}
	for _, c := range cases {
		got := addressSpans(t, c.text)
		if len(got) != 1 || got[0].Text != c.want {
			t.Errorf("%q: want one address %q, got %+v", c.text, c.want, got)
		}
	}
	runNegatives(t, entity.Address, []string{
		"Wir nehmen die Hauptstraße. Hausnummer 5 fehlt noch",
		"Go via Calle Mayor. Room 5 is free",
		"Prendi la Via Appia. Sezione 4, 20121 unità",
		"Das ist der Lindenweg. Punkt 5 ist offen",
		"Thanks. Version 5 shipped",
		"Mein Name ist Anna. Kapitel 5 ist fertig",
	})
}

func TestAddressCityStopsBeforeAPersonName(t *testing.T) {
	cases := []struct{ text, want string }{
		{"Ship it to Rua Augusta 100, 1100-048 Lisboa Maria Kowalski Lopez said hello", "Rua Augusta 100, 1100-048 Lisboa"},
		{"Hauptstraße 5, 10115 Berlin Nguyễn Thị Minh Khai.", "Hauptstraße 5, 10115 Berlin"},
		{"Via Roma 10, 00184 Roma Zoë Müller-Öztürk", "Via Roma 10, 00184 Roma"},
		{"Calle Mayor 12, 2º B, 28013 Madrid Park Ji-sung", "Calle Mayor 12, 2º B, 28013 Madrid"},
		{"Lindenstraße 8, 50674 Köln Anna-Lena Schmidt wohnt dort", "Lindenstraße 8, 50674 Köln"},
		{"ENVIAR A: CALLE MAYOR 4, 28013 MADRID MARIA LOPEZ", "CALLE MAYOR 4, 28013 MADRID"},
		{"Adresse: 10117 Berlin Maria Kowalski", "10117 Berlin"},
		{"Corso Giuseppe Mazzini 4, 42121 Reggio Emilia Maria Rossi abita qui", "Corso Giuseppe Mazzini 4, 42121 Reggio Emilia"},
		{"Largo do Carmo 3, 6000-100 Castelo Branco Maria Lopes mora aqui", "Largo do Carmo 3, 6000-100 Castelo Branco"},
		{"Paseo de la Castellana 100, 20003 San Sebastián Maria Kowalski vive aquí", "Paseo de la Castellana 100, 20003 San Sebastián"},
		{"abita in Corso Giuseppe Mazzini 4, 42121 Reggio Emilia da maggio", "Corso Giuseppe Mazzini 4, 42121 Reggio Emilia"},
		{"Corso Giuseppe Mazzini 4, 42121 Reggio Emilia", "Corso Giuseppe Mazzini 4, 42121 Reggio Emilia"},
		{"Corso Giuseppe Mazzini 4, 42121 Reggio Emilia, Italia", "Corso Giuseppe Mazzini 4, 42121 Reggio Emilia, Italia"},
		{"Via Roma 10, 00184 Roma Italia", "Via Roma 10, 00184 Roma Italia"},
		{"Hauptstraße 5, 10115 Berlin Maria wohnt dort", "Hauptstraße 5, 10115 Berlin Maria"},
		{"Largo do Carmo 3, 6000-100 Castelo Branco", "Largo do Carmo 3, 6000-100 Castelo Branco"},
		{"Stationsweg 7, 2515 BM Den Haag Jan de Vries", "Stationsweg 7, 2515 BM Den Haag"},
		{"Rua das Flores 12, 4050-262 Porto Maria Kowalski disse", "Rua das Flores 12, 4050-262 Porto"},
		{"Rua das Flores 12, 9400-164 Porto Santo Maria Kowalski disse", "Rua das Flores 12, 9400-164 Porto Santo"},
		{"Calle Mayor 12, 31003 Pamplona Geum Beom-joon", "Calle Mayor 12, 31003 Pamplona"},
		{"Via Roma 10, 47121 Borgo Verona Abigail Desrosiers", "Via Roma 10, 47121 Borgo Verona"},
		{"Lindenstraße 8, 61348 Neu Kassel Abigail Desrosiers", "Lindenstraße 8, 61348 Neu Kassel"},
		{"Lindenstraße 8, 50674 Köln Tel: 0221 123456", "Lindenstraße 8, 50674 Köln"},
		{"Hauptstraße 5, 10115 Berlin Mitte Deutschland", "Hauptstraße 5, 10115 Berlin Mitte Deutschland"},
		{"Hauptstraße 5, 10115 Berlin Mitte. Bitte klingeln", "Hauptstraße 5, 10115 Berlin Mitte"},
		{"send it to Flat 2, 22 Court Street, Brooklyn, New York 11201 please", "22 Court Street, Brooklyn, New York 11201"},
		{"send it to Flat 2, 22 Court Street, Brooklyn, NY 11201 please", "22 Court Street, Brooklyn, NY 11201"},
		{"send it to Flat 3, 14 Church Lane, Leeds, UK please", "Flat 3, 14 Church Lane, Leeds, UK"},
	}
	for _, c := range cases {
		got := addressSpans(t, c.text)
		if len(got) != 1 || got[0].Text != c.want {
			t.Errorf("%q: want one address %q, got %+v", c.text, c.want, got)
		}
	}
}

func TestAddressStaircaseHouseNumber(t *testing.T) {
	const text = "Lieferung an Mariahilfer Straße 45/2/14, 1060 Wien bitte"
	got := addressSpans(t, text)
	if len(got) != 1 || got[0].Text != "Mariahilfer Straße 45/2/14, 1060 Wien" {
		t.Errorf("%q: want the whole address as one span, got %+v", text, got)
	}
}
