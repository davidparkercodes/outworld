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
	{"123 Calle Ocho, Miami, Florida 33135", "374 Calle Larchmont, Brookvale, Meadowbrook 38707", "323 Calle Yarrow, Thornfield, Whitbury 57242"},
	{"123 Via Verde\nSan Diego, California 92101", "996 Via Mistletoe\nCedarburg Pinehurst, Harborview 77200", "671 Via Northgate\nRidgemont Kingsbridge, Silverlake 28869"},
	{"123 Via Verde", "608 Via Primrose", "903 Via Silverbirch"},
	{"20 Lindenstrasse, Fredericksburg, Texas", "51 Primrosestrasse, Greystone, Marshfield", "86 Chestnutstrasse, Lakemont, Stonehaven"},
	{"Flat 2, 22 Court Street, Brooklyn, New York 11201", "Flat 7, 25 Court Street, Eastholm, Westhollow Whitbury 30268", "Flat 8, 12 Court Street, Glenhaven, Maplestead Coldwater 62374"},
	{"12345 Via Verde", "18588 Via Clearwater", "19732 Via Hazelwood"},
	{"4 Calle Mayor, Laredo, Texas, USA", "9 Calle Redbud, Cedarburg, Stonehaven, CEDARBURG", "6 Calle Cypress, Ravensbrook, Brookvale, ELMSTEAD"},
	{"12 Kirchweg, New Braunfels, TX 78130, United States", "21 Sagebrushweg, Willowbrook Millhaven, TX 82644, Woodmere Larkfield", "96 Tamarackweg, Brookvale Millhaven, TX 87570, Kingsbridge Dunmere"},
	{"77 Hauptstrasse", "83 Redbudstrasse", "76 Northgatestrasse"},
	{"8 Rua das Flores, Newark 07105", "7 Rua das Northgate, Westhollow 29753", "3 Rua das Pinecrest, Hollowmere 52301"},
	{"Apt 3, 250 Paseo del Sol, Santa Fe, New Mexico", "Apt 8, 281 Paseo del Tidewater, Crestwood Whitbury, Oakvale Dunmere", "Apt 2, 635 Paseo del Dovecote, Lakemont Coldwater, Hollowmere Millhaven"},
	{"14 Church Lane, Boston, Massachusetts", "41 Pinecrest Lane, Crestwood, Massachusetts", "93 Elmhurst Lane, Whitbury, Massachusetts"},
	{"PO Box 77, Laredo, Texas 78040", "PO Box 40, Coppervale, Lakeview 50265", "PO Box 22, Cobblestone, Alderbrook 04359"},
	{"5 Kerkstraat, Holland, Michigan", "5 Fernhillstraat, Elmstead, Northbury", "1 Silverbirchstraat, Millhaven, Birchfield"},
	{"Flat 3, 14 Church Lane, Leeds", "Flat 9, 95 Ashgrove Lane, Quarrytown", "Flat 5, 67 Birchmont Lane, Ravensbrook"},
	{"Flat 2, 191 Cedar High Avenue", "Flat 4, 495 Wisteria Millbrook Avenue", "Flat 6, 702 Glenmore Magnolia Avenue"},
	{"Unit 5, 12 Kirchweg, Hermann, Missouri 65041", "Unit 5, 61 Pinecrestweg, Harborview, Northbury 19627", "Unit 4, 37 Hollyhockweg, Quarrytown, Millhaven 58928"},
	{"4 Calle Mayor, Madrid 28013", "6 Calle Stonebridge, Elmstead 83012", "4 Calle Glenmore, Ashborough 16754"},
	{"Suite 200, 77 Paseo del Sol, Santa Fe, NM 87501, USA", "Suite 438, 10 Paseo del Primrose, Hollowmere Brookvale, NM 25274, BELLHAVEN", "Suite 296, 86 Paseo del Fernhill, Fairhollow Maplestead, NM 07112, BIRCHFIELD"},
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
		{"12 Avenue Victor Hugo, 75016 Paris", addrFR, `^\d\d Avenue \p{Lu}\p{L}+ (\p{Lu}\p{L}+), (\d{5}) (\p{Lu}\p{L}+)$`, 1, 2, 3},
		{"25 rue Jean-Jaurès, 13100 Aix-en-Provence", addrFR, `^\d\d rue \p{Lu}\p{L}+-(\p{Lu}\p{L}+), (\d{5}) (\p{Lu}\p{L}+)-en-\p{Lu}\p{L}+$`, 1, 2, 3},
		{"8 chemin du Moulin, 17000 La Rochelle", addrFR, `^\d chemin du (\p{Lu}\p{L}+), (\d{5}) La (\p{Lu}\p{L}+)$`, 1, 2, 3},
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
		{"Berliner Straße 44, 60311 Frankfurt am Main", addrDE, `^(\p{Lu}\p{L}+) Straße \d\d, (\d{5}) (\p{Lu}\p{L}+) am \p{Lu}\p{L}+$`, 1, 2, 3},
		{"An der Mühle 12, D-28195 Bremen", addrDE, `^An der (\p{Lu}\p{L}+) \d\d, D-(\d{5}) (\p{Lu}\p{L}+)$`, 1, 2, 3},
		{"Mariahilfer Straße 88, 1070 Wien", addrDE, `^(\p{Lu}\p{L}+) Straße \d\d, (\d{4}) (\p{Lu}\p{L}+)$`, 1, 2, 3},
		{"Bahnhofstrasse 21, CH-8001 Zürich", addrDE, `^\p{Lu}\p{L}+strasse \d\d, CH-(\d{4}) (\p{Lu}\p{L}+)$`, 0, 1, 2},
		{"Herengracht 401, 1017 BP Amsterdam", addrNL, `^\p{Lu}\p{L}+gracht \d{3}, (\d{4} [A-Z]{2}) (\p{Lu}\p{L}+)$`, 0, 1, 2},
		{"Kerkstraat 12, 3511AB Utrecht", addrNL, `^\p{Lu}\p{L}+straat \d\d, (\d{4}[A-Z]{2}) (\p{Lu}\p{L}+)$`, 0, 1, 2},
		{"Stationsweg 7, 2515 BM Den Haag", addrNL, `^\p{Lu}\p{L}+weg \d, (\d{4} [A-Z]{2}) Den (\p{Lu}\p{L}+)$`, 0, 1, 2},
		{"Flat 4, 27 Glenorchy Road, Perth PH1 5XY", addrUK, `^Flat \d, \d\d (\p{Lu}\p{L}+) Road, (\p{Lu}\p{L}+) ([A-Z]{2}\d \d[A-Z]{2})$`, 1, 3, 2},
		{"88 Canal Street\nManchester\nM1 3HE", addrUK, `^\d\d (\p{Lu}\p{L}+) Street\n(\p{Lu}\p{L}+)\n([A-Z]\d \d[A-Z]{2})$`, 1, 3, 2},
		{"12 Park View Road, Burgess Hill RH15 9AA", addrUK, `^\d\d (\p{Lu}\p{L}+) View Road, \p{Lu}\p{L}+ (\p{Lu}\p{L}+) ([A-Z]{2}\d\d \d[A-Z]{2})$`, 1, 3, 2},
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

var addrKeptWordRE = regexp.MustCompile(`^(?:[IVX]{2,5}|Flat|Apt|Top)$`)

func addrKeptWord(l *addrLocale, w string) bool {
	low := strings.ToLower(w)
	return l.types[low] || addrConnWords[low] || addrUnitWords[low] || addrCountryWords[low] || addrUKTypes[low] || addrGermanicWords[low] || addrKeptWordRE.MatchString(w) || addrAttachedRE.MatchString(w)
}

func TestMultiWordAddressesKeepTheirShape(t *testing.T) {
	const w = `\p{Lu}\p{L}+`
	cases := []struct {
		real string
		loc  *addrLocale
		want string
	}{
		{"12 rue de la Paix, 75002 Paris", addrFR, `^\d\d rue de la ` + w + `, \d{5} ` + w + `$`},
		{"12 bis rue du Général Leclerc, 17000 La Rochelle", addrFR, `^\d\d bis rue du ` + w + ` ` + w + `, \d{5} La ` + w + `$`},
		{"5 rue du Général de Gaulle, 44000 Nantes", addrFR, `^\d rue du ` + w + ` de ` + w + `, \d{5} ` + w + `$`},
		{"18 boulevard Saint-Michel, 94100 Saint-Maur-des-Fossés", addrFR, `^\d\d boulevard ` + w + `-` + w + `, \d{5} ` + w + `-` + w + `-des-` + w + `$`},
		{"3 rue de l'Abbé Grégoire, 59650 Villeneuve-d'Ascq", addrFR, `^\d rue de l'` + w + ` ` + w + `, \d{5} ` + w + `-d'[AEIOU]\p{L}+$`},
		{"9 quai de la Loire, Appt 12, 13100 Aix-en-Provence", addrFR, `^\d quai de la ` + w + `, Appt \d\d, \d{5} ` + w + `-en-` + w + `$`},
		{"Calle Mayor de Triana 4, 41010 Sevilla", addrES, `^Calle ` + w + ` de ` + w + ` \d, \d{5} ` + w + `$`},
		{"Paseo de la Castellana 100, 20003 San Sebastián", addrES, `^Paseo de la ` + w + ` \d{3}, \d{5} ` + w + ` ` + w + `$`},
		{"Calle del Doctor Fleming 8, 2º B, 11403 Jerez de la Frontera", addrES, `^Calle del ` + w + ` ` + w + ` \d, \dº [A-Z], \d{5} ` + w + ` de la ` + w + `$`},
		{"4 Calle Mayor de Triana, 41010 Sevilla", addrES, `^\d Calle ` + w + ` de ` + w + `, \d{5} ` + w + `$`},
		{"Avenida da Liberdade 110, 1250-146 Lisboa", addrPT, `^Avenida da ` + w + ` \d{3}, \d{4}-\d{3} ` + w + `$`},
		{"Rua Doutor António Cândido 7, 3º Esq, 4400-001 Vila Nova de Gaia", addrPT, `^Rua ` + w + ` ` + w + ` ` + w + ` \d, \dº Esq, \d{4}-\d{3} ` + w + ` ` + w + ` de ` + w + `$`},
		{"Largo do Carmo 3, 6000-100 Castelo Branco", addrPT, `^Largo do ` + w + ` \d, \d{4}-\d{3} ` + w + ` ` + w + `$`},
		{"Via Vittorio Emanuele II 12, 00186 Roma", addrIT, `^Via ` + w + ` ` + w + ` II \d\d, \d{5} ` + w + `$`},
		{"Corso Giuseppe Mazzini 4, 42121 Reggio Emilia", addrIT, `^Corso ` + w + ` ` + w + ` \d, \d{5} ` + w + ` ` + w + `$`},
		{"Via dei Mille 9, 36061 Bassano del Grappa", addrIT, `^Via dei ` + w + ` \d, \d{5} ` + w + ` del ` + w + `$`},
		{"Via XX Settembre 14, 67100 L'Aquila", addrIT, `^Via XX ` + w + ` \d\d, \d{5} L'[AEIOU]\p{L}+$`},
		{"12 Via Vittorio Emanuele II, 00186 Roma", addrIT, `^\d\d Via ` + w + ` ` + w + ` II, \d{5} ` + w + `$`},
		{"Unter den Linden 5, 10117 Berlin", addrDE, `^Unter den ` + w + ` \d, \d{5} ` + w + `$`},
		{"Konrad-Adenauer-Straße 12, 61348 Bad Homburg", addrDE, `^` + w + `-` + w + `-Straße \d\d, \d{5} ` + w + ` ` + w + `$`},
		{"Berliner Straße 44, 60311 Frankfurt am Main", addrDE, `^` + w + ` Straße \d\d, \d{5} ` + w + ` am ` + w + `$`},
		{"Hauptstraße 5, Top 3, 5020 Salzburg", addrDE, `^` + w + `straße \d, Top \d, \d{4} ` + w + `$`},
		{"Burgemeester de Vlugtlaan 125, 1063 BJ Amsterdam", addrNL, `^` + w + ` de ` + w + `laan \d{3}, \d{4} [A-Z]{2} ` + w + `$`},
		{"Stationsweg 7, 2515 BM Den Haag", addrNL, `^` + w + `weg \d, \d{4} [A-Z]{2} Den ` + w + `$`},
		{"Grote Markt 3, 4611NT Bergen op Zoom", addrNL, `^` + w + ` Markt \d, \d{4}[A-Z]{2} ` + w + ` op ` + w + `$`},
		{"12 Park View Road, Burgess Hill RH15 9AA", addrUK, `^\d\d ` + w + ` View Road, ` + w + ` ` + w + ` [A-Z]{2}\d\d \d[A-Z]{2}$`},
		{"3 Station Road, Stoke on Trent ST4 2AA", addrUK, `^\d ` + w + ` Road, ` + w + ` on ` + w + ` [A-Z]{2}\d \d[A-Z]{2}$`},
		{"3 Station Road, Stoke-on-Trent ST4 2AA", addrUK, `^\d ` + w + ` Road, ` + w + `-on-` + w + ` [A-Z]{2}\d \d[A-Z]{2}$`},
		{"Flat 3, 21 Grey Street, Newcastle upon Tyne NE1 6EE", addrUK, `^Flat \d, \d\d ` + w + ` Street, ` + w + ` upon ` + w + ` [A-Z]{2}\d \d[A-Z]{2}$`},
		{"Apt 2, 9 Mill Lane\nMilton Keynes\nMK9 3XS", addrUK, `^Apt \d, \d ` + w + ` Lane\n` + w + ` ` + w + `\n[A-Z]{2}\d \d[A-Z]{2}$`},
	}
	wordRE := regexp.MustCompile(`\p{L}{3,}`)
	for _, c := range cases {
		if got := addrLocaleOf(c.real); got != c.loc {
			t.Errorf("%q: wrong locale", c.real)
			continue
		}
		re := regexp.MustCompile(c.want)
		realWords := map[string]bool{}
		for _, word := range wordRE.FindAllString(c.real, -1) {
			realWords[textutil.Fold(word)] = true
		}
		for _, secret := range []string{"shape", "golden", "third"} {
			g := NewGenerator([]byte(secret))
			seen := map[string]bool{}
			for attempt := 0; attempt < 40; attempt++ {
				got := g.Address(c.real, attempt)
				seen[got] = true
				if got == c.real || got != g.Address(c.real, attempt) {
					t.Fatalf("%q: surrogate %q equals the original or is not deterministic", c.real, got)
				}
				if !re.MatchString(got) {
					t.Errorf("%q: surrogate %q does not match %s", c.real, got, c.want)
					break
				}
				if len(strings.Fields(got)) != len(strings.Fields(c.real)) || strings.Count(got, "-") != strings.Count(c.real, "-") {
					t.Errorf("%q: surrogate %q changed the word count", c.real, got)
				}
				words := wordRE.FindAllString(got, -1)
				for i, word := range words {
					if realWords[textutil.Fold(word)] && !addrKeptWord(c.loc, word) {
						t.Errorf("%q: surrogate %q reuses %q", c.real, got, word)
					}
					for _, other := range words[:i] {
						if other == word && !addrKeptWord(c.loc, word) {
							t.Errorf("%q: surrogate %q repeats %q", c.real, got, word)
						}
					}
				}
			}
			if len(seen) < 35 {
				t.Errorf("%q: only %d distinct surrogates in 40 attempts", c.real, len(seen))
			}
		}
	}
}

func TestAddressLocaleDetection(t *testing.T) {
	cases := []struct {
		real string
		loc  *addrLocale
		want string
	}{
		{"4 Calle Mayor, 2º B, 28013 Madrid", addrES, `^\d Calle \p{Lu}\p{L}+, \dº [A-Z], \d{5} \p{Lu}\p{L}+$`},
		{"4500 Avenida de la Constitución, 18001 Granada", addrES, `^\d{4} Avenida de la \p{Lu}\p{L}+, \d{5} \p{Lu}\p{L}+$`},
		{"110 Avenida da Liberdade, 1250-146 Lisboa", addrPT, `^\d{3} Avenida da \p{Lu}\p{L}+, \d{4}-\d{3} \p{Lu}\p{L}+$`},
		{"12 Rua das Flores, 4050-262 Porto", addrPT, `^\d\d Rua das \p{Lu}\p{L}+, \d{4}-\d{3} \p{Lu}\p{L}+$`},
		{"12 Via Roma, 20121 Milano", addrIT, `^\d\d Via \p{Lu}\p{L}+, \d{5} \p{Lu}\p{L}+$`},
		{"12345 Via Roma, 20121 Milano", addrIT, `^\d{5} Via \p{Lu}\p{L}+, \d{5} \p{Lu}\p{L}+$`},
		{"5 Hauptstraße\n10117 Berlin", addrDE, `^\d \p{Lu}\p{L}+straße\n\d{5} \p{Lu}\p{L}+$`},
		{"5 Hauptstr. 12, 8010 Graz", addrDE, `^\d \p{Lu}\p{L}+str\. \d\d, \d{4} \p{Lu}\p{L}+$`},
		{"5 Berliner Straße, 10117 Berlin", addrDE, `^\d \p{Lu}\p{L}+ Straße, \d{5} \p{Lu}\p{L}+$`},
		{"5 Lindenallee, 10117 Berlin", addrDE, `^\d \p{Lu}\p{L}+allee, \d{5} \p{Lu}\p{L}+$`},
		{"5 Kerkstraat, 1017 GC Amsterdam", addrNL, `^\d \p{Lu}\p{L}+straat, \d{4} [A-Z]{2} \p{Lu}\p{L}+$`},
		{"14 Church Lane, Leeds, UK", addrUK, `^\d\d [A-Z][a-z]+ Lane, [A-Z][a-z]+, UK$`},
		{"14 Church Lane, Leeds UK", addrUK, `^\d\d [A-Z][a-z]+ Lane, [A-Z][a-z]+ UK$`},
		{"14 Church Lane\nLeeds\nUnited Kingdom", addrUK, `^\d\d [A-Z][a-z]+ Lane\n[A-Z][a-z]+\nUnited Kingdom$`},
		{"14 Church Lane, Leeds, England", addrUK, `^\d\d [A-Z][a-z]+ Lane, [A-Z][a-z]+, England$`},
		{"Flat 3, 14 Church Lane, Leeds, UK", addrUK, `^Flat \d, \d\d [A-Z][a-z]+ Lane, [A-Z][a-z]+, UK$`},
		{"Flat 3, 14 Church Lane, Leeds", nil, ``},
		{"Flat 2, 191 Cedar High Avenue", nil, ``},
		{"14 Church Lane, Leeds", nil, ``},
		{"45 Church Street, Boston", nil, ``},
		{"123 Main St, Springfield, IL 62704", nil, ``},
		{"12 George Street, Sydney, New South Wales", nil, ``},
		{"Flat 3, 123 Main St, Springfield, IL 62704", nil, ``},
		{"4500 Calle Real", nil, ``},
		{"4 Calle Mayor", nil, ``},
		{"4 Calle Mayor, Madrid 28013", nil, ``},
		{"12 Rua das Flores", nil, ``},
		{"12 Via Roma", nil, ``},
		{"12 Via Roma, Milano", nil, ``},
		{"12345 Via Verde", nil, ``},
		{"5 Hauptstraße", nil, ``},
		{"5 Hauptstr. 12", nil, ``},
		{"5 Kerkstraat", nil, ``},
		{"123 Calle Ocho, Miami, Florida 33135", nil, ``},
		{"123 Calle Ocho, Miami, Florida", nil, ``},
		{"123 Calle Ocho, 33135 Miami, USA", nil, ``},
		{"123 Calle Ocho, 33135 Miami, United States", nil, ``},
		{"123 Via Verde\nSan Diego, California 92101", nil, ``},
		{"20 Lindenstrasse, Fredericksburg, Texas", nil, ``},
		{"20 Lindenstrasse, 78624 Fredericksburg, Texas", nil, ``},
		{"5 Kerkstraat, Holland, Michigan", nil, ``},
		{"Flat 2, 22 Court Street, Brooklyn, New York 11201", nil, ``},
		{"Flat 2, 22 Court Street, Brooklyn, New York", nil, ``},
		{"Flat 2, 22 Court Street, Brooklyn NY", nil, ``},
		{"Flat 2, 22 Court Street, Brooklyn, USA", nil, ``},
		{"14 Church Lane, Boston, Massachusetts", nil, ``},
		{"77 Via Verde Dr", nil, ``},
		{"18 Via Roma Way", nil, ``},
		{"9 Camino del Rio S", nil, ``},
		{"4 Calle Mayor, Laredo, TX", nil, ``},
		{"2000 Paseo Verde Parkway", nil, ``},
		{"12 Platz Court", nil, ``},
		{"5 Spring St", nil, ``},
		{"100 Spring Street", nil, ``},
		{"100 Kettering Rd, Dayton", nil, ``},
		{"5 Lindenallee", nil, ``},
		{"12 Stockade Ln", nil, ``},
		{"742 Evergreen Terrace", nil, ``},
	}
	g := NewGenerator([]byte("shape"))
	for _, c := range cases {
		if got := addrLocaleOf(c.real); got != c.loc {
			id := "none"
			if got != nil {
				id = got.id
			}
			t.Errorf("%q: read as %s", c.real, id)
			continue
		}
		if c.loc == nil {
			continue
		}
		re := regexp.MustCompile(c.want)
		for attempt := 0; attempt < 40; attempt++ {
			got := g.Address(c.real, attempt)
			if got == c.real || !re.MatchString(got) {
				t.Errorf("%q: surrogate %q does not match %s", c.real, got, c.want)
				break
			}
		}
	}
}

func TestLocalePostalCodesKeepTheirShape(t *testing.T) {
	cases := []struct{ real, want string }{
		{"12 rue de la Paix, 75002 Paris", `, \d{5} `},
		{"Calle Mayor 4, 28013 Madrid", `, \d{5} `},
		{"Rua das Flores 12, 4050-262 Porto", `, \d{4}-\d{3} `},
		{"Via Garibaldi 30, 10122 Torino", `, \d{5} `},
		{"Lindenstraße 8, 50674 Köln", `, \d{5} `},
		{"Mariahilfer Straße 88, 1070 Wien", `, \d{4} `},
		{"Mariahilfer Straße 88, A-1070 Wien", `, A-\d{4} `},
		{"Bahnhofstrasse 21, CH-8001 Zürich", `, CH-\d{4} `},
		{"Bahnhofstrasse 21, 8001 Zürich", `, \d{4} `},
		{"Herengracht 401, 1017 BP Amsterdam", `, \d{4} [A-Z]{2} `},
		{"Kerkstraat 12, 3511AB Utrecht", `, \d{4}[A-Z]{2} `},
		{"9 Church Lane, York YO1 7HH", ` [A-Z]{2}\d \d[A-Z]{2}$`},
		{"9 Church Lane, Leeds LS12 7HH", ` [A-Z]{2}\d\d \d[A-Z]{2}$`},
		{"88 Canal Street, Manchester M1 3HE", ` [A-Z]\d \d[A-Z]{2}$`},
		{"88 Canal Street, Manchester M13HE", ` [A-Z]\d\d[A-Z]{2}$`},
	}
	for _, secret := range []string{"shape", "golden"} {
		g := NewGenerator([]byte(secret))
		for _, c := range cases {
			re := regexp.MustCompile(c.want)
			for attempt := 0; attempt < 40; attempt++ {
				if got := g.Address(c.real, attempt); !re.MatchString(got) {
					t.Errorf("%q: postal code shape lost in %q", c.real, got)
					break
				}
			}
		}
	}
}

func TestPostalSpanFindsPostcodeAtStart(t *testing.T) {
	cases := []struct {
		loc  *addrLocale
		real string
		want string
	}{
		{addrFR, "75001 Paris, 12 rue de Rivoli", "75001"},
		{addrES, "28013 Madrid, Calle Mayor 4", "28013"},
		{addrIT, "20121 Milano, Via Roma 12", "20121"},
		{addrPT, "1250-146 Lisboa, Avenida da Liberdade 110", "1250-146"},
		{addrDE, "10117 Berlin", "10117"},
		{addrDE, "1070 Wien, Mariahilfer Straße 88", "1070"},
		{addrNL, "1017 BP Amsterdam, Herengracht 401", "1017 BP"},
		{addrUK, "SW1A 1AA", "SW1A 1AA"},
		{addrES, "4 Calle Mayor, Madrid 28013", "28013"},
		{addrDE, "Hauptstr. 5, 80331 München", "80331"},
	}
	for _, c := range cases {
		s, e := c.loc.postalSpan(c.real)
		if s < 0 || c.real[s:e] != c.want {
			t.Errorf("%s %q: postal span %d:%d, want %q", c.loc.id, c.real, s, e, c.want)
		}
	}
}

func TestPostcodeFirstAddressesKeepShapeAndFittingTown(t *testing.T) {
	g := NewGenerator([]byte("shape"))
	cases := []struct {
		real string
		loc  *addrLocale
		want string
	}{
		{"75001 Paris, 12 rue de Rivoli", addrFR, `^(\d{5}) (\p{Lu}\p{L}+), \d\d rue de \p{Lu}\p{L}+$`},
		{"28013 Madrid, Calle Mayor 12", addrES, `^(\d{5}) (\p{Lu}\p{L}+), Calle \p{Lu}\p{L}+ \d\d$`},
		{"20121 Milano, Via Roma 12", addrIT, `^(\d{5}) (\p{Lu}\p{L}+), Via \p{Lu}\p{L}+ \d\d$`},
		{"1250-146 Lisboa, Avenida da Liberdade 110", addrPT, `^(\d{4}-\d{3}) (\p{Lu}\p{L}+), Avenida da \p{Lu}\p{L}+ \d{3}$`},
		{"10115 Berlin, Hauptstraße 5", addrDE, `^(\d{5}) (\p{Lu}\p{L}+), \p{Lu}\p{L}+straße \d$`},
		{"1070 Wien, Mariahilfer Straße 88", addrDE, `^(\d{4}) (\p{Lu}\p{L}+), \p{Lu}\p{L}+ Straße \d\d$`},
		{"CH-8001 Zürich, Bahnhofstrasse 21", addrDE, `^CH-(\d{4}) (\p{Lu}\p{L}+), \p{Lu}\p{L}+strasse \d\d$`},
		{"1017 BP Amsterdam, Herengracht 401", addrNL, `^(\d{4} [A-Z]{2}) (\p{Lu}\p{L}+), \p{Lu}\p{L}+gracht \d{3}$`},
	}
	for _, c := range cases {
		if got := addrLocaleOf(c.real); got != c.loc {
			t.Errorf("%q: wrong locale", c.real)
			continue
		}
		re := regexp.MustCompile(c.want)
		for attempt := 0; attempt < 30; attempt++ {
			got := g.Address(c.real, attempt)
			m := re.FindStringSubmatch(got)
			if m == nil {
				t.Errorf("%q: surrogate %q does not match %s", c.real, got, c.want)
				break
			}
			code, ok := placeCode(c.loc, m[2])
			if !ok || !strings.HasPrefix(m[1], code) {
				t.Errorf("%q: %q is not a %s town with a fitting postcode", c.real, got, c.loc.id)
			}
		}
	}
	for _, real := range []string{"12345 Via Roma, 20121 Milano", "1234 NW 5th St, Portland, OR 97209", "1600 Pennsylvania Ave NW, Washington, DC 20500"} {
		if _, ok := addrPostcodeFirst(real); ok {
			t.Errorf("%q: must not read as postcode-first", real)
		}
	}
}

func TestPostcodeFirstFrenchAddressKeepsAFittingTown(t *testing.T) {
	g := NewGenerator([]byte("shape"))
	real := "75001 Paris, 12 rue de Rivoli"
	if addrLocaleOf(real) != addrFR {
		t.Fatalf("%q: wrong locale", real)
	}
	re := regexp.MustCompile(`^(\d{5}) (\p{Lu}\p{L}+), \d\d rue de \p{Lu}\p{L}+$`)
	for attempt := 0; attempt < 30; attempt++ {
		got := g.Address(real, attempt)
		m := re.FindStringSubmatch(got)
		if m == nil {
			t.Fatalf("%q: surrogate %q has the wrong shape", real, got)
		}
		code, ok := placeCode(addrFR, m[2])
		if !ok || !strings.HasPrefix(m[1], code) {
			t.Errorf("%q: %q is not a FR town with a fitting postcode", real, got)
		}
	}
}

func TestLocaleAddressesNeverKeepTheRealHouseNumber(t *testing.T) {
	g := NewGenerator([]byte("house"))
	num := regexp.MustCompile(`\d+`)
	for _, real := range []string{"75002 Paris, 12 rue de la Paix", "12 rue de la Paix, 75002 Paris", "Rua das Flores 12, 4050-262 Porto", "Via Roma 2, 20121 Milano", "Hauptstraße 5, 10115 Berlin", "Calle del Olmo 5, 2º B, 28012 Madrid"} {
		if addrLocaleOf(real) == nil {
			t.Fatalf("%q: no locale", real)
		}
		realNums := num.FindAllString(real, -1)
		for attempt := 0; attempt < 60; attempt++ {
			got := g.Address(real, attempt)
			gotNums := num.FindAllString(got, -1)
			if len(gotNums) != len(realNums) {
				t.Errorf("%q: number groups changed in %q", real, got)
				continue
			}
			for i, n := range realNums {
				if len(n) >= 4 {
					continue
				}
				if len(gotNums[i]) != len(n) || gotNums[i] == n {
					t.Errorf("%q: surrogate %q keeps or reshapes the real number %s", real, got, n)
				}
			}
		}
	}
}
