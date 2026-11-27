package detect

import (
	"context"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

func TestVenueAndThingNamesAreNotPeople(t *testing.T) {
	expectNoPeople(t, []string{
		"Dinner at The Amber Room was fine.",
		"We met in the Lincoln Room and then the Victoria Suite.",
		"Meet at Amber Lounge tonight, then Carter Hall.",
		"The Oliver Prize is next.",
		"The Grace Hopper Award went to her.",
		"The Jennifer Awards were fun.",
		"Meet at Wilson Hall.",
		"The Lincoln Memorial and the Kennedy Center.",
		"Book the Elena Suite for Friday.",
		"Turn onto Garfield Ave. after the light.",
		"LOCATION:Room Ada (4th floor)",
		"Hurricane Maria hit the coast.",
		"Project Athena ships in spring.",
		"I stayed at Hotel Amélie.",
	})
	expectPeople(t, map[string]string{
		"Amber Hall called about the invoice.":    "Amber Hall",
		"Please ask Amber about the room.":        "Amber",
		"Thanks Amber. The Amber Room is booked.": "Amber",
		"Jennifer Hall sent the file.":            "Jennifer Hall",
		"Jennifer Park and David Hall are here.":  "Jennifer Park|David Hall",
		"I emailed Amber House yesterday.":        "Amber House",
		"Invited: Rich Tavares, Amber Hall":       "Rich Tavares|Amber Hall",
		"Maria Garcia Project Update":             "Maria Garcia",
	})
}

func TestRejectedPairsDoNotLeaveLoneNames(t *testing.T) {
	expectNoPeople(t, []string{
		"For Monte Carlo sims is Julia really faster than Python?",
		"We flew to Santa Clara, then San Jose and Saint Louis.",
		"I read Monte Cristo and visited Victoria Falls with Georgia Tech alumni.",
		"Charlotte Amalie and Virginia Beach are nice.",
		"St. Augustine was nice. Lake Victoria and Mount Vernon and Fort Wayne.",
		"A Markov chain and Bayes rule, Gauss elimination, the Newton method and the Turing test.",
		"The Darwin Awards and Darwin the macOS kernel.",
		"10/06  Hotel, Regina SK, 2 nights",
		"Return: Regina, SK, 3 Jan 2027",
		"I flew to Augusta GA then Madison WI.",
		"Should I just learn Ada or Pascal?",
		"Use the Jordan method and a Gauss filter. Newton's method and Bayes' theorem.",
		"Our El Paso and La Jolla offices.",
	})
	expectPeople(t, map[string]string{
		"Florence Nightingale was a nurse.":             "Florence Nightingale",
		"Paris Hilton called yesterday.":                "Paris Hilton",
		"Ada Lovelace wrote the first program.":         "Ada Lovelace",
		"Charles Darwin the person.":                    "Charles Darwin",
		"Julia Roberts called.":                         "Julia Roberts",
		"Claude Shannon and Homer Simpson.":             "Claude Shannon|Homer Simpson",
		"Adam Smith wrote it.":                          "Adam Smith",
		"Tell Carlo about Monte Carlo. Carlo said yes.": "Carlo|Carlo",
		"Is Regina OK? Regina called.":                  "Regina|Regina",
		"John Okafor San Francisco office.":             "John Okafor",
		"We let Elena test the build.":                  "Elena",
	})
}

func TestTechTermPairsStayTerms(t *testing.T) {
	expectNoPeople(t, []string{
		"Ruby on Rails with Pascal Case.",
		"Claude Code is nice.",
		"Jasmine Rice is on the menu.",
		"Adam Optimizer settings.",
	})
}

func TestPeopleListsAfterLabels(t *testing.T) {
	expectPeople(t, map[string]string{
		"Present: J. R. Adjei (chair), Milica Jovanović (secretary), Oksana Hrytsenko, Lane Pruitt, Thiha\nApologies: Pat N.": "J. R. Adjei|Milica Jovanović|Oksana Hrytsenko|Lane Pruitt|Thiha|Pat N.",
		"Attendees: Lane Pruitt, Maria Lopez, Grace Park":                                                                     "Lane Pruitt|Maria Lopez|Grace Park",
		"Cc: Dana Whitfield, Sales Team, Hope Lang":                                                                           "Dana Whitfield|Hope Lang",
		"To: Garcia, Elena; Lopez, Daniel":                                                                                    "Garcia, Elena|Lopez, Daniel",
		"Participants: Olu, Wren Castillo, May":                                                                               "Olu|Wren Castillo",
		"Team: Data Science, Maria Lopez, Site Reliability":                                                                   "Maria Lopez",
		"Invited: Wren Castillo, Lane Pruitt and Dana Whitfield.":                                                             "Wren Castillo|Lane Pruitt|Dana Whitfield",
		"Reviewed by Elena Garcia, Daniel Brown and Lane Pruitt, Maria Lopez.":                                                "Elena Garcia|Daniel Brown|Maria Lopez",
	})
	expectNoPeople(t, []string{
		"Team: Platform Engineering, Data Science, Site Reliability",
		"Participants: Acme Corp, Legal, Finance",
		"Present: Legal, Finance, Design",
		"Members: Lane Closure, Road Works",
	})
}

func TestSurnameFirstIgnoresPrecedingName(t *testing.T) {
	expectPeople(t, map[string]string{
		"Patient Garcia, Elena was admitted.":        "Garcia, Elena",
		"Regarding Lopez, Daniel: see file.":         "Lopez, Daniel",
		"We saw Wren Garcia, Elena and the others.":  "Elena",
		"Attendees: Lane Pruitt, Elena Novak, Thiha": "Lane Pruitt|Elena Novak|Thiha",
	})
}

func initialismTexts(t *testing.T, text string) []string {
	t.Helper()
	spans, err := NameDetector{}.Detect(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, s := range spans {
		if s.Source == "names:initials" {
			if s.Type != entity.Person || !IsInitialismOf(s.Text, s.Canonical) {
				t.Errorf("%q: bad initialism span %+v", text, s)
			}
			out = append(out, s.Text+"="+s.Canonical)
		}
	}
	return out
}

func TestInitialismsOfPeopleInTheSameText(t *testing.T) {
	for text, want := range map[string]string{
		"Present: J. R. Adjei (chair), Milica Jovanović, Oksana Hrytsenko, Lane Pruitt\n1. LP reported the council will repaint. JRA to write to the council.\n2. OH obtained two quotes; MJ to circulate.": "LP=Lane Pruitt|JRA=J. R. Adjei|MJ=Milica Jovanović",
		"Attendees: Wren Castillo, Maria Lopez\nWC: I think we ship.\nML: agreed.\nAction: draft memo (WC)":                                                                                                 "WC=Wren Castillo|ML=Maria Lopez|WC=Wren Castillo",
		"Wanda Castillo and Dana Whitfield met.\nOwner: DW\nReviewed by WC and approved.":                                                                                                                   "DW=Dana Whitfield|WC=Wanda Castillo",
		"Attendees: Wren Castillo, Maria Lopez, Oksana Kovalenko, OK, ML":                                                                                                                                   "OK=Oksana Kovalenko|ML=Maria Lopez",
		"Willem van der Heijden joined.\nWH to send the lease. WVH agreed.":                                                                                                                                 "WH=Willem van der Heijden",
	} {
		if got := strings.Join(initialismTexts(t, text), "|"); got != want {
			t.Errorf("%q: got %q, want %q", text, got, want)
		}
	}
	for _, text := range []string{
		"1. LP reported the council will repaint. JRA to write to the council.",
		"Present: Lane Pruitt, Maria Lopez\nThe US to send aid. IT to fix laptops. OK to proceed. PM to confirm. ML to review the model.",
		"Ursula Stone and Ian Tate and Oksana Kovalenko met. US to send aid. IT to fix laptops. OK to proceed.",
		"Lane Pruitt called. The LP reported gains. Our LP is happy. An LP record.",
		"Maria Lopez wrote the ML pipeline and the WC door sign.",
		"Lane Pruitt called. Check the lp settings and Lp config.",
	} {
		if got := initialismTexts(t, text); len(got) != 0 {
			t.Errorf("%q: unexpected initialisms %q", text, got)
		}
	}
}

func TestDictionaryInitialisms(t *testing.T) {
	d := &Dictionary{}
	if err := d.add("Lane Pruitt", "person", nil, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := d.add("Amber Nakagawa", "person", nil, "", ""); err != nil {
		t.Fatal(err)
	}
	find := func(text string) []string {
		spans, _ := d.Detector().Detect(context.Background(), text)
		var out []string
		for _, s := range spans {
			out = append(out, s.Text)
		}
		return out
	}
	if got := strings.Join(find("Lane Pruitt chaired.\nLP to send the notes."), "|"); got != "Lane Pruitt|Lane|Pruitt|LP" && got != "Lane Pruitt|Pruitt|LP" {
		t.Errorf("got %q", got)
	}
	if got := find("LP to send the notes."); len(got) != 0 {
		t.Errorf("initialism without the full name: %q", got)
	}
	if got := find("Dinner at The Amber Room."); len(got) != 0 {
		t.Errorf("venue matched: %q", got)
	}
	if got := strings.Join(find("Amber booked The Amber Room."), "|"); got != "Amber" {
		t.Errorf("got %q", got)
	}
}

func TestCapsSurnameFirstWithParticles(t *testing.T) {
	expectPeople(t, map[string]string{
		"I live with my partner CARVALHO DE SOUZA, João Pedro since March 2023.": "CARVALHO DE SOUZA, João Pedro",
		"Seen by DE LA CRUZ, Maria and VAN DER BERG, Pieter Jan.":                "DE LA CRUZ, Maria|VAN DER BERG, Pieter Jan",
		"PATIENT: GARCIA, Elena Sofia. Stable.":                                  "GARCIA, Elena Sofia",
		"between WILLEM VAN DER HEIJDEN and FATIMA ZAHRA EL IDRISSI today":       "WILLEM VAN DER HEIJDEN|FATIMA ZAHRA EL IDRISSI",
	})
	for _, text := range []string{"NOTE THIS, Maria.", "WARNING, Elena.", "PLEASE READ, Daniel.", "NEW YORK, Elena said."} {
		spans, _ := NameDetector{}.Detect(context.Background(), text)
		for _, s := range spans {
			if s.Source == "names:surname-first" {
				t.Errorf("%q: unexpected surname-first span %q", text, s.Text)
			}
		}
	}
	spans, _ := NameDetector{}.Detect(context.Background(), "partner CARVALHO DE SOUZA, João Pedro since")
	found := false
	for _, s := range spans {
		if s.Source == "names:surname-first" {
			found = true
			if s.Canonical != "João Pedro Carvalho de Souza" {
				t.Errorf("canonical %q", s.Canonical)
			}
		}
	}
	if !found {
		t.Error("no surname-first span")
	}
}

func TestHonorificsInSeveralLanguages(t *testing.T) {
	expectPeople(t, map[string]string{
		"Necesito escribirle al señor José Luis Ramírez Ortega, porque la caldera lleva rota.": "José Luis Ramírez Ortega",
		"La señora María Fernanda López de Aguirre llegó.":                                     "María Fernanda López de Aguirre",
		"Bonjour monsieur Jean Pierre Dupont de la Fontaine, merci.":                           "Jean Pierre Dupont de la Fontaine",
		"Sehr geehrte Frau Anna Maria Schmidt, danke.":                                         "Anna Maria Schmidt",
		"Herr von Braun kommt.":                                    "von Braun",
		"Hablé con don Pepe y doña Carmen ayer.":                   "Pepe|Carmen",
		"Estimado Sr. Ramírez: hola.":                              "Ramírez",
		"Gentile signora Giulia Bianchi, grazie.":                  "Giulia Bianchi",
		"Prezado senhor Paulo Henrique Oliveira Santos, obrigado.": "Paulo Henrique Oliveira Santos",
		"Ich habe Frau Schmidt Bescheid gegeben.":                  "Schmidt",
	})
	expectNoPeople(t, []string{
		"He will don the hat.",
		"The frau said nothing.",
		"Turn left at the sr. center.",
	})
}

func TestNameUseOKForThingsAndTechTerms(t *testing.T) {
	for text, want := range map[string]bool{
		"Dinner at The Amber Room.":        false,
		"Amber Hall called.":               true,
		"Should I learn Ada or Pascal?":    false,
		"I asked Ada to review it.":        true,
		"Ada said yes.":                    true,
		"The Darwin Awards are back.":      false,
		"Darwin the macOS kernel is old.":  false,
		"Tell Darwin about it.":            true,
		"A hotel in Regina SK for a week.": false,
		"Regina called.":                   true,
		"The Monte Carlo method.":          false,
	} {
		for _, name := range []string{"Amber", "Ada", "Darwin", "Regina", "Carlo"} {
			i := strings.Index(text, name)
			if i < 0 {
				continue
			}
			if got := NameUseOK(text, i, i+len(name)); got != want {
				t.Errorf("%q %q: got %v, want %v", text, name, got, want)
			}
		}
	}
}
