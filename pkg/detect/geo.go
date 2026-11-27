package detect

import (
	"sort"
	"strings"
)

var usStates = map[string]string{
	"AL": "Alabama", "AK": "Alaska", "AZ": "Arizona", "AR": "Arkansas", "CA": "California",
	"CO": "Colorado", "CT": "Connecticut", "DE": "Delaware", "FL": "Florida", "GA": "Georgia",
	"HI": "Hawaii", "ID": "Idaho", "IL": "Illinois", "IN": "Indiana", "IA": "Iowa",
	"KS": "Kansas", "KY": "Kentucky", "LA": "Louisiana", "ME": "Maine", "MD": "Maryland",
	"MA": "Massachusetts", "MI": "Michigan", "MN": "Minnesota", "MS": "Mississippi", "MO": "Missouri",
	"MT": "Montana", "NE": "Nebraska", "NV": "Nevada", "NH": "New Hampshire", "NJ": "New Jersey",
	"NM": "New Mexico", "NY": "New York", "NC": "North Carolina", "ND": "North Dakota", "OH": "Ohio",
	"OK": "Oklahoma", "OR": "Oregon", "PA": "Pennsylvania", "RI": "Rhode Island", "SC": "South Carolina",
	"SD": "South Dakota", "TN": "Tennessee", "TX": "Texas", "UT": "Utah", "VT": "Vermont",
	"VA": "Virginia", "WA": "Washington", "WV": "West Virginia", "WI": "Wisconsin", "WY": "Wyoming",
	"DC": "the District of Columbia",
}

func StateName(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if n, ok := usStates[strings.ToUpper(s)]; ok && len(s) == 2 {
		return n, true
	}
	for _, n := range usStates {
		if strings.EqualFold(n, s) {
			return n, true
		}
	}
	return "", false
}

func stateAlternation() string {
	var parts []string
	for abbr, name := range usStates {
		parts = append(parts, abbr)
		if !strings.HasPrefix(name, "the ") {
			parts = append(parts, strings.ReplaceAll(name, " ", `\s+`))
		}
	}
	sort.Slice(parts, func(i, j int) bool {
		if len(parts[i]) != len(parts[j]) {
			return len(parts[i]) > len(parts[j])
		}
		return parts[i] < parts[j]
	})
	return strings.Join(parts, "|")
}

func StateIn(s string) (string, bool) {
	i := strings.LastIndex(s, ",")
	if i < 0 {
		return "", false
	}
	tail := strings.TrimSpace(s[i+1:])
	f := strings.Fields(tail)
	for len(f) > 0 {
		if n, ok := StateName(strings.Join(f, " ")); ok {
			return n, true
		}
		f = f[:len(f)-1]
	}
	return "", false
}

var ukPostcodeAreas = wordSet(`AB AL B BA BB BD BH BL BN BR BS BT CA CB CF CH CM CO CR CT CV CW DA DD DE DG DH DL DN DT DY E EC EH EN EX FK FY G GL GU GY HA HD HG HP HR HS HU HX IG IM IP IV JE KA KT KW KY L LA LD LE LL LN LS LU M ME MK ML N NE NG NN NP NR NW OL OX PA PE PH PL PO PR RG RH RM S SA SE SG SK SL SM SN SO SP SR SS ST SW SY TA TD TF TN TQ TR TS TW UB W WA WC WD WF WN WR WS WV YO ZE`)

var addrCountries = phraseSet(`españa|espana|spain|deutschland|germany|allemagne|alemania|germania|france|frankreich|francia|italia|italy|italien|italie|portugal|nederland|netherlands|the netherlands|holland|united kingdom|uk|england|scotland|wales|northern ireland|great britain|reino unido|royaume-uni|österreich|austria|autriche|schweiz|suisse|svizzera|switzerland|belgique|belgië|belgie|belgium|luxembourg|luxemburg|ireland|polska|poland|danmark|denmark|sverige|sweden|norge|norway`)

var addrUKCountries = phraseSet(`united kingdom|uk|england|scotland|wales|northern ireland|great britain`)

var addrCityLeads = wordSet(`san santa santo são saint sainte st ste sankt skt bad gran grande la le les el los las den nova novo nueva nuevo new vila villa ciudad puerto porto castel castelo monte mont sint oud nieuw port bourg château villeneuve pont val bois neuville castro torre puente villanueva ponte póvoa aldeia borgo rocca neu alt groß klein ober hohen nieder kirch noord zuid groot hoog laag`)

var addrLoneLeads = wordSet(`porto`)

var addrTwoWordTowns = phraseSet(`reggio emilia|reggio calabria|ascoli piceno|vibo valentia|massa carrara|busto arsizio|cinisello balsamo|sesto fiorentino|lamezia terme|montecatini terme|abano terme|civitanova marche|martina franca|gioia tauro|vittorio veneto|mogliano veneto|ponta delgada|torres vedras|torres novas|castelo branco|medina sidonia|baden baden|wiener neustadt|bergisch gladbach|schwäbisch hall|schwäbisch gmünd|milton keynes|palma mallorca|porto alegre|porto torres|porto santo|porto empedocle|porto recanati`)

var addrDotLeads = wordSet(`st ste s sta sto skt`)

var addrRivers = wordSet(`main rhein neckar donau elbe weser saale inn lahn ruhr havel oder mosel isar lech aller spree ems nahe sieg fulda werra tauber brenz enz glan ilm rur wupper see bodensee taunus harz breisgau allgäu pfalz höhe rijn maas ijssel vecht waal amstel lek zee zoom linge dommel schie duurstede`)

var addrAfterCity = wordSet(`tel tél telefon telefono teléfono telefone telefoon phone fax mobil mobile handy email e-mail mail web www attn ust vat iban bic kvk btw nif cif cell`)

func phraseSet(s string) map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Split(s, "|") {
		m[w] = true
	}
	return m
}
