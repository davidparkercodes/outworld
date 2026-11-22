package surrogate

import (
	"regexp"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/textutil"
)

var usAddressGolden = []struct{ real, first, eighth string }{
	{"123 Main Street, Apt 4B, Richmond, VA 23220", "403 Glenmore Street, Apt 1F, Kingsbridge, VA 86123", "319 Coppervale Street, Apt 8B, Fairhollow, VA 49737"},
	{"123 Main Street", "510 Kestrel Street", "766 Magnolia Street"},
	{"4500 Calle Real, Santa Barbara, CA 93110", "7404 Calle Glenmore, Coldwater Eastholm, CA 45300", "5389 Calle Elmhurst, Birchfield Elmstead, CA 30762"},
	{"2000 Paseo Verde Parkway, Henderson, NV 89052", "2235 Southfield Parkway, Ashborough, NV 46339", "6764 Alderbrook Parkway, Cedarburg, NV 94664"},
	{"2207 HOLLOWAY RD STE 14\nRALEIGH NC 27603", "4534 FOXGLOVE RD STE 77\nWILLOWBROOK NC 70553", "5491 SYCAMORE RD STE 30\nMAPLESTEAD NC 23141"},
	{"915 N ORCHARD AVE, BOISE ID 83706-1120", "606 N RAVENWOOD AVE, MAPLESTEAD ID 18858-3630", "861 N TAMARACK AVE, KINGSBRIDGE ID 74687-2674"},
	{"1234 NW 5th St, Portland, OR 97209", "7556 PV 3th St, Woodmere, OR 58225", "2364 IZ 6th St, Glenhaven, OR 12663"},
	{"200 Avenue of the Stars, Los Angeles, CA 90067", "807 Heatherfield Avenue, Southmere, CA 66238", "388 Clearwater Avenue, Maplestead, CA 18113"},
	{"500 AVENUE OF THE AMERICAS\nNEW YORK NY 10001", "521 AVENUE COPPERVALE THE REDBUD\nSTONEHAVEN GREYSTONE NY 95117", "769 AVENUE CYPRESS THE PINECREST\nCLEARBROOK LARKFIELD NY 46712"},
	{"77 Via Verde Dr, Tucson, AZ 85704", "93 Sandpiper Dr, Thornfield, AZ 33980", "31 Mistletoe Dr, Coldwater, AZ 41441"},
	{"One Microsoft Way", "239 Orchard Lane", "795 Rosewood Lane"},
	{"P.O. Box 1234", "P.O. Box 3424", "P.O. Box 1199"},
	{"88 Spring Garden Road, Unit 12, Austin, TX 78701", "82 Whispering Garden Road, Unit 28, Westhollow, TX 77703", "74 Tidewater Garden Road, Unit 10, Fernvale, TX 27971"},
	{"1600 Pennsylvania Ave NW, Washington, DC 20500", "8367 Windmill Ave, Stonehaven, DC 09809", "5447 Fernhill Ave, Woodmere, DC 96115"},
	{"350 Fifth Avenue, Suite 3300, New York, NY 10118-0110", "158 Glenmore Avenue, Suite 4812, Silverlake, NY 42396-6693", "818 Primrose Avenue, Suite 8677, Thornfield, NY 91848-3274"},
	{"12 Platz Court, New Ulm, MN 56073", "60 Dovecote Court, Elmstead, MN 17516", "52 Millbrook Court, Larkfield, MN 83543"},
	{"742 Evergreen Terrace", "597 Glenmore Terrace", "887 Ashgrove Terrace"},
	{"31 Spooner St.", "60 Meadowlark St.", "20 Oakmere St."},
	{"100 MAIN ST APT B2 3RD FL\nBROOKLYN NY 11201", "305 CHESTNUT ST APT U0 4GV FL\nBELLHAVEN NY 08656", "139 PINECREST ST APT I5 4NN FL\nMILLHAVEN NY 32120"},
	{"Suite 200, 1 Market Street, San Francisco, CA 94105", "Suite 999, 9 Coppervale Street, Birchfield Dunmere, CA 40788", "Suite 989, 1 Bramblewood Street, Millhaven Oakvale, CA 30526"},
	{"9 Camino del Rio S, San Diego, CA 92108", "9 Camino del Thornbury S, Quarrytown Marshfield, CA 51017", "2 Camino del Ravenwood S, Marshfield Westhollow, CA 20584"},
	{"18 Via Roma Way, Naples, FL 34102", "33 Lakeview Way, Pinehurst, FL 83764", "40 Wisteria Way, Pinehurst, FL 51262"},
	{"655 Rue Royale Blvd, New Orleans, LA 70130", "458 Larchmont Blvd, Coldwater, LA 90922", "337 Pinecrest Blvd, Eastholm, LA 17324"},
	{"42 Avenue B, New York, NY 10009", "20 Silverbirch Avenue, Fairhollow, NY 59525", "18 Riverside Avenue, Silverlake, NY 89801"},
}

func TestUSAddressesAreByteStable(t *testing.T) {
	g := NewGenerator([]byte("golden"))
	for _, c := range usAddressGolden {
		if got := g.Address(c.real, 0); got != c.first {
			t.Errorf("%q attempt 0: got %q, want %q", c.real, got, c.first)
		}
		if got := g.Address(c.real, 7); got != c.eighth {
			t.Errorf("%q attempt 7: got %q, want %q", c.real, got, c.eighth)
		}
		if addrLocaleOf(c.real) != nil {
			t.Errorf("%q must not be read as a non-US address", c.real)
		}
	}
}

func localeWords(l *addrLocale) map[string]bool {
	m := map[string]bool{}
	for _, list := range l.streets {
		for _, w := range list {
			m[w] = true
		}
	}
	return m
}

func placeCode(l *addrLocale, name string) (string, bool) {
	for _, p := range append(append([]addrPlace(nil), l.places...), l.alt...) {
		if p.name == name {
			return p.code, true
		}
	}
	return "", false
}

func TestAddressesFollowTheirLocale(t *testing.T) {
	g := NewGenerator([]byte("shape"))
	cases := []struct {
		real   string
		loc    *addrLocale
		want   string
		street int
		post   int
		city   int
	}{
		{"12 rue de Rivoli, 75001 Paris", addrFR, `^\d\d rue de (\p{Lu}\p{L}+), (\d{5}) (\p{Lu}\p{L}+)$`, 1, 2, 3},
		{"14 rue des Lilas, 69003 Lyon", addrFR, `^\d\d rue des (\p{Lu}\p{L}+), (\d{5}) (\p{Lu}\p{L}+)$`, 1, 2, 3},
		{"3 place de la République, 35000 Rennes, France", addrFR, `^\d place de la (\p{Lu}\p{L}+), (\d{5}) (\p{Lu}\p{L}+), France$`, 1, 2, 3},
		{"12 Avenue Victor Hugo, 75016 Paris", addrFR, `^\d\d Avenue (\p{Lu}\p{L}+), (\d{5}) (\p{Lu}\p{L}+)$`, 1, 2, 3},
		{"25 rue Jean-Jaurès, 13100 Aix-en-Provence", addrFR, `^\d\d rue (\p{Lu}\p{L}+), (\d{5}) (\p{Lu}\p{L}+)$`, 1, 2, 3},
		{"8 chemin du Moulin, 17000 La Rochelle", addrFR, `^\d chemin du (\p{Lu}\p{L}+), (\d{5}) (\p{Lu}\p{L}+)$`, 1, 2, 3},
		{"12 RUE DES LILAS 69003 LYON", addrFR, `^\d\d RUE DES \p{Lu}+ \d{5} \p{Lu}+$`, 0, 0, 0},
		{"Calle del Olmo 5, 2º B, 28012 Madrid", addrES, `^Calle del (\p{Lu}\p{L}+) \d, \dº [A-Z], (\d{5}) (\p{Lu}\p{L}+)$`, 1, 2, 3},
		{"Avenida de la Constitución 14, 18001 Granada, España", addrES, `^Avenida de la (\p{Lu}\p{L}+) \d\d, (\d{5}) (\p{Lu}\p{L}+), España$`, 1, 2, 3},
		{"Plaza de España 3, 41013 Sevilla", addrES, `^Plaza de (\p{Lu}\p{L}+) \d, (\d{5}) (\p{Lu}\p{L}+)$`, 1, 2, 3},
		{"Rua das Flores 12, 4050-262 Porto", addrPT, `^Rua das (\p{Lu}\p{L}+) \d\d, (\d{4}-\d{3}) (\p{Lu}\p{L}+)$`, 1, 2, 3},
		{"Avenida da Liberdade 110, 1250-146 Lisboa", addrPT, `^Avenida da (\p{Lu}\p{L}+) \d{3}, (\d{4}-\d{3}) (\p{Lu}\p{L}+)$`, 1, 2, 3},
		{"Via Garibaldi 30, 10122 Torino", addrIT, `^Via (\p{Lu}\p{L}+) \d\d, (\d{5}) (\p{Lu}\p{L}+)$`, 1, 2, 3},
		{"Piazza della Repubblica 7, 50123 Firenze", addrIT, `^Piazza della (\p{Lu}\p{L}+) \d, (\d{5}) (\p{Lu}\p{L}+)$`, 1, 2, 3},
		{"Lindenstraße 8, 50674 Köln", addrDE, `^\p{Lu}\p{L}+straße \d, (\d{5}) (\p{Lu}\p{L}+)$`, 0, 1, 2},
		{"Hauptstr. 5, 80331 München", addrDE, `^\p{Lu}\p{L}+str\. \d, (\d{5}) (\p{Lu}\p{L}+)$`, 0, 1, 2},
		{"Berliner Straße 44, 60311 Frankfurt am Main", addrDE, `^(\p{Lu}\p{L}+) Straße \d\d, (\d{5}) (\p{Lu}\p{L}+)$`, 1, 2, 3},
		{"An der Mühle 12, D-28195 Bremen", addrDE, `^An der (\p{Lu}\p{L}+) \d\d, D-(\d{5}) (\p{Lu}\p{L}+)$`, 1, 2, 3},
		{"Mariahilfer Straße 88, 1070 Wien", addrDE, `^(\p{Lu}\p{L}+) Straße \d\d, (\d{4}) (\p{Lu}\p{L}+)$`, 1, 2, 3},
		{"Bahnhofstrasse 21, CH-8001 Zürich", addrDE, `^\p{Lu}\p{L}+strasse \d\d, CH-(\d{4}) (\p{Lu}\p{L}+)$`, 0, 1, 2},
		{"Herengracht 401, 1017 BP Amsterdam", addrNL, `^\p{Lu}\p{L}+gracht \d{3}, (\d{4} [A-Z]{2}) (\p{Lu}\p{L}+)$`, 0, 1, 2},
		{"Kerkstraat 12, 3511AB Utrecht", addrNL, `^\p{Lu}\p{L}+straat \d\d, (\d{4}[A-Z]{2}) (\p{Lu}\p{L}+)$`, 0, 1, 2},
		{"Stationsweg 7, 2515 BM Den Haag", addrNL, `^\p{Lu}\p{L}+weg \d, (\d{4} [A-Z]{2}) (\p{Lu}\p{L}+)$`, 0, 1, 2},
		{"Flat 4, 27 Glenorchy Road, Perth PH1 5XY", addrUK, `^Flat \d, \d\d (\p{Lu}\p{L}+) Road, (\p{Lu}\p{L}+) ([A-Z]{2}\d \d[A-Z]{2})$`, 1, 3, 2},
		{"88 Canal Street\nManchester\nM1 3HE", addrUK, `^\d\d (\p{Lu}\p{L}+) Street\n(\p{Lu}\p{L}+)\n([A-Z]\d \d[A-Z]{2})$`, 1, 3, 2},
		{"12 Park View Road, Burgess Hill RH15 9AA", addrUK, `^\d\d (\p{Lu}\p{L}+) Road, (\p{Lu}\p{L}+) ([A-Z]{2}\d\d \d[A-Z]{2})$`, 1, 3, 2},
	}
	for _, c := range cases {
		if got := addrLocaleOf(c.real); got != c.loc {
			t.Errorf("%q: wrong locale", c.real)
			continue
		}
		re := regexp.MustCompile(c.want)
		words := localeWords(c.loc)
		realWords := map[string]bool{}
		for _, w := range regexp.MustCompile(`\p{L}{3,}`).FindAllString(c.real, -1) {
			realWords[textutil.Fold(w)] = true
		}
		seen := map[string]bool{}
		for attempt := 0; attempt < 60; attempt++ {
			got := g.Address(c.real, attempt)
			if got != g.Address(c.real, attempt) {
				t.Fatalf("%q: not deterministic", c.real)
			}
			seen[got] = true
			m := re.FindStringSubmatch(got)
			if m == nil {
				t.Errorf("%q: surrogate %q does not match %s", c.real, got, c.want)
				break
			}
			if c.street > 0 && !words[m[c.street]] {
				t.Errorf("%q: street word %q in %q is not from the %s lists", c.real, m[c.street], got, c.loc.id)
			}
			if c.city > 0 {
				code, ok := placeCode(c.loc, m[c.city])
				if !ok {
					t.Errorf("%q: city %q in %q is not a %s place", c.real, m[c.city], got, c.loc.id)
				} else if !strings.HasPrefix(m[c.post], code) {
					t.Errorf("%q: postal code %q does not fit %q (%s)", c.real, m[c.post], m[c.city], code)
				}
			}
			for _, w := range regexp.MustCompile(`\p{L}{3,}`).FindAllString(got, -1) {
				low := strings.ToLower(w)
				if realWords[textutil.Fold(w)] && !c.loc.types[low] && !addrConnWords[low] && !addrUnitWords[low] && !addrCountryWords[low] && !addrUKTypes[low] && !addrGermanicWords[low] {
					t.Errorf("%q: surrogate %q reuses %q", c.real, got, w)
				}
			}
		}
		if len(seen) < 50 {
			t.Errorf("%q: only %d distinct surrogates in 60 attempts", c.real, len(seen))
		}
	}
}

func TestLocalePlacesAreNeverTheRealOnes(t *testing.T) {
	g := NewGenerator([]byte("shape"))
	for _, c := range []struct{ real, word string }{
		{"4 rue des Lilas, 49000 Angers", "Angers"},
		{"4 rue des Lilas, 49000 Angers", "Lilas"},
		{"Via Verdi 3, 37121 Verona", "Verona"},
		{"Via Verdi 3, 37121 Verona", "Verdi"},
		{"Kerkstraat 3, 8011 AB Zwolle", "Zwolle"},
		{"Kerkstraat 3, 8011 AB Zwolle", "Kerk"},
		{"Lindenallee 3, 48143 Münster", "Linden"},
		{"Lindenallee 3, 48143 MUENSTER", "Münster"},
		{"9 Church Lane, York YO1 7HH", "York"},
		{"9 Church Lane, York YO1 7HH", "Church"},
	} {
		for attempt := 0; attempt < 300; attempt++ {
			if got := g.Address(c.real, attempt); strings.Contains(got, c.word) {
				t.Fatalf("%q attempt %d: %q reuses %q", c.real, attempt, got, c.word)
			}
		}
	}
}

func TestLocaleListsAreClean(t *testing.T) {
	for _, l := range []*addrLocale{addrFR, addrES, addrPT, addrIT, addrDE, addrNL, addrUK} {
		for key, list := range l.streets {
			seen := map[string]bool{}
			for _, w := range list {
				if seen[w] || strings.ToLower(w) == "bulwark" {
					t.Errorf("%s %q: bad or duplicate entry %q", l.id, key, w)
				}
				seen[w] = true
			}
		}
		seen := map[string]bool{}
		for _, p := range l.places {
			if seen[p.name] || p.code == "" {
				t.Errorf("%s: bad or duplicate place %q", l.id, p.name)
			}
			seen[p.name] = true
		}
	}
	if addrDE.alt[addrAustrianPlaces-1].name != "Feldkirch" || addrDE.alt[addrAustrianPlaces].name != "Bern" {
		t.Error("Austrian and Swiss places are out of step with addrAustrianPlaces")
	}
}
