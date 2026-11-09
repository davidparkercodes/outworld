package corpus

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
)

type Address struct {
	Number string
	Road   string
	Unit   string
	City   string
	State  string
	Zip    string
}

func (a Address) Street() string { return a.Number + " " + a.Road }

type Person struct {
	ID         string
	First      string
	Last       string
	Nick       string
	Title      string
	CommonWord bool
	InDict     bool
	Emails     []string
	PhoneArea  string
	PhoneLine  string
	Address    Address
	Birth      time.Time
}

func (p *Person) Full() string { return p.First + " " + p.Last }

func (p *Person) Slug() string {
	s := strings.ToLower(p.First + "-" + p.Last)
	return strings.ReplaceAll(s, "'", "")
}

type Org struct {
	Name   string
	Domain string
}

type personSeed struct {
	first, last, nick, title string
	common                   bool
}

var personSeeds = []personSeed{
	{"Marisol", "Quenby", "Mari", "Ms.", false},
	{"Thaddeus", "Wexley", "Thad", "Mr.", false},
	{"Ingrid", "Achterberg", "", "Dr.", false},
	{"Desmond", "Oyelaran", "Dez", "Mr.", false},
	{"Priyanka", "Haverkamp", "Priya", "Ms.", false},
	{"Callum", "Thistlewood", "Cal", "Mr.", false},
	{"Odalys", "Kestrelmoor", "", "Ms.", false},
	{"Benedikt", "Sorensby", "Ben", "Mr.", false},
	{"Yusra", "Pellucidi", "", "Dr.", false},
	{"Lorcan", "Quillfeather", "", "Mr.", false},
	{"Henrike", "Dunleavy", "Rike", "Ms.", false},
	{"Tobiah", "Marchetti-Rook", "Toby", "Mr.", false},
	{"Saoirse", "Nakashima-Blythe", "", "Ms.", false},
	{"Emeka", "Rautio", "", "Mr.", false},
	{"Wilhelmina", "Szczepanek", "Mina", "Ms.", false},
	{"Anselm", "Hollowell", "", "Dr.", false},
	{"Ximena", "Vasquez-Ferro", "", "Ms.", false},
	{"Corwin", "Abernathy", "", "Mr.", false},
	{"Leocadia", "Brannigan", "Cady", "Ms.", false},
	{"Ravindra", "Lindqvist", "Ravi", "Mr.", false},
	{"Juniper", "O'Dell", "June", "Ms.", false},
	{"Osric", "Pemberton-Oake", "", "Mr.", false},
	{"Delphine", "Ilunga", "Del", "Ms.", false},
	{"Kwabena", "Fairweather", "Kobby", "Mr.", false},
	{"Ottoline", "Corrigan", "Lina", "Ms.", false},
	{"Bartholomew", "Tolliver", "Bart", "Mr.", false},
	{"Ines", "Okafor-Lind", "", "Ms.", false},
	{"Fenwick", "Delacroix-Penn", "Fen", "Mr.", false},
	{"Mireille", "Varga-Holm", "", "Ms.", false},
	{"Abelard", "Ashdown", "Abe", "Mr.", false},
	{"Will", "Ashgrove", "", "Mr.", true},
	{"Rose", "Thornbury", "", "Ms.", true},
	{"Grant", "Okonedo-Hale", "", "Mr.", true},
	{"Hope", "Valdane", "", "Ms.", true},
	{"May", "Lindenbrook", "", "Ms.", true},
	{"Mark", "Fennimore", "", "Mr.", true},
	{"Dawn", "Pryzbyla", "", "Ms.", true},
	{"Summer", "Ekwueme", "", "Ms.", true},
	{"Bill", "Carraway", "", "Mr.", true},
	{"Faith", "Morrow-Quist", "", "Ms.", true},
	{"Art", "Velasquez-Moor", "", "Mr.", true},
	{"Joy", "Halloran-Pike", "", "Ms.", true},
	{"Frank", "Ostergaard", "", "Mr.", true},
	{"Grace", "Wilmot-Ayer", "", "Ms.", true},
	{"Sterling", "Adeyemi-Cole", "", "Mr.", true},
	{"Ivy", "Garroway", "", "Ms.", true},
	{"Penny", "Strathearn", "", "Ms.", true},
	{"Chase", "Nwankwo-Bell", "", "Mr.", true},
}

var orgs = []Org{
	{"Copperline Logistics", "copperline.example"},
	{"Halvorsen Biotech", "halvorsen-bio.example"},
	{"Bramblecrest Credit Union", "bramblecrest.example"},
	{"Tidewater Robotics", "tidewater-robotics.example"},
	{"Fernhollow Youth Orchestra", "fernhollow.example"},
	{"Quarry Lane Dental", "quarrylane.example"},
	{"Saltmarsh Insurance Group", "saltmarsh-ins.example"},
	{"Ivorygate Legal LLP", "ivorygate.example"},
	{"Pinecrest Veterinary Clinic", "pinecrest-vet.example"},
	{"Stonebridge Mutual Bank", "stonebridge.example"},
	{"Lanternfish Studios", "lanternfish.example"},
	{"Greywillow Property Management", "greywillow.example"},
	{"Kettlebrook Medical Associates", "kettlebrook.example"},
	{"Brightfen Software", "brightfen.example"},
}

var locations = []string{
	"Lisbon", "Rotterdam", "Halifax", "Ghent", "Bergen", "Cork",
	"Wellington", "Tallinn", "Valparaiso", "Kyoto", "Nairobi", "Montreal",
}

var roads = []string{
	"Larkspur Lane", "Wren Hollow Road", "Quillon Street", "Marrowby Avenue",
	"Tansy Court", "Old Ferrier Way", "Basalt Ridge Drive", "Gannet Terrace",
	"Coldspring Row", "Hazelmere Place", "Sorrel Bend", "Pikestaff Lane",
}

type cityInfo struct{ city, state, zip, area string }

var cities = []cityInfo{
	{"Duluth", "MN", "558", "218"},
	{"Boise", "ID", "837", "208"},
	{"Asheville", "NC", "288", "828"},
	{"Tucson", "AZ", "857", "520"},
	{"Burlington", "VT", "054", "802"},
	{"Spokane", "WA", "992", "509"},
	{"Eugene", "OR", "974", "541"},
	{"Madison", "WI", "537", "608"},
}

var freeMail = []string{"mail.example", "example.net", "inbox.example", "example.org"}

func buildPeople(r *rand.Rand) []*Person {
	var out []*Person
	for i, s := range personSeeds {
		c := cities[r.IntN(len(cities))]
		p := &Person{
			ID:         fmt.Sprintf("P%02d", i+1),
			First:      s.first,
			Last:       s.last,
			Nick:       s.nick,
			Title:      s.title,
			CommonWord: s.common,
			InDict:     i%5 < 3,
			PhoneArea:  c.area,
			PhoneLine:  fmt.Sprintf("01%02d", r.IntN(100)),
			Address: Address{
				Number: fmt.Sprintf("%d", 10+r.IntN(4980)),
				Road:   roads[r.IntN(len(roads))],
				City:   c.city,
				State:  c.state,
				Zip:    fmt.Sprintf("%s%02d", c.zip, r.IntN(100)),
			},
			Birth: time.Date(1950+r.IntN(50), time.Month(1+r.IntN(12)), 1+r.IntN(28), 0, 0, 0, 0, time.UTC),
		}
		if r.IntN(3) == 0 {
			p.Address.Unit = fmt.Sprintf("Apt %d%c", 1+r.IntN(9), 'A'+rune(r.IntN(4)))
		}
		first := strings.ToLower(s.first)
		last := strings.ToLower(strings.NewReplacer("'", "", "-", "").Replace(s.last))
		work := orgs[r.IntN(len(orgs))].Domain
		switch r.IntN(4) {
		case 0:
			p.Emails = []string{first + "." + last + "@" + work, first[:1] + last + fmt.Sprintf("%d", 10+r.IntN(89)) + "@" + freeMail[r.IntN(len(freeMail))]}
		case 1:
			p.Emails = []string{first + "@" + work, last + "." + first + "@" + freeMail[r.IntN(len(freeMail))]}
		case 2:
			p.Emails = []string{first[:1] + "." + last + "@" + work}
		default:
			p.Emails = []string{first + "_" + last + "@" + freeMail[r.IntN(len(freeMail))], strings.ToUpper(first[:1]) + first[1:] + "." + strings.ToUpper(last[:1]) + last[1:] + "@" + work}
		}
		out = append(out, p)
	}
	return out
}

func dictionary(people []*Person) []DictEntry {
	var out []DictEntry
	for _, p := range people {
		if !p.InDict {
			continue
		}
		aliases := []string{p.First, p.Last}
		if p.Nick != "" {
			aliases = append(aliases, p.Nick)
		}
		aliases = append(aliases, p.Last+", "+p.First)
		out = append(out, DictEntry{Value: p.Full(), Type: TypePerson, Aliases: aliases})
	}
	return out
}
