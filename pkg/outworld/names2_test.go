package outworld

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/detect"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

func TestNamedThingsSurviveMasking(t *testing.T) {
	for in, kept := range map[string][]string{
		"Employee: Amber Nakagawa (E-0412).\nDinner at The Amber Room was fine.":                                 {"The Amber Room"},
		"Charles Darwin the person, the Darwin Awards and Darwin the macOS kernel.":                              {"the Darwin Awards", "Darwin the macOS kernel"},
		"Carlo Rossi ran the Monte Carlo sims.":                                                                  {"Monte Carlo sims"},
		"Regina Novak booked a hotel in Regina SK for two nights.":                                               {"Regina SK"},
		"Victoria Okafor flew to Victoria Falls, then Santa Clara and San Jose.":                                 {"Victoria Falls", "Santa Clara", "San Jose"},
		"Should I learn `Ada` or Pascal?":                                                                        {"learn `Ada` or Pascal"},
		"Alan Turing proposed the Turing test and the Turing machine.":                                           {"Turing test", "Turing machine"},
		"We met in the Lincoln Room and then the Victoria Suite.":                                                {"Lincoln Room", "Victoria Suite"},
		"The Grace Hopper Award went to her.":                                                                    {"The Grace Hopper Award"},
		"Present: Lane Pruitt, Maria Lopez\nThe US to send aid. IT to fix laptops. OK to proceed. ML to review.": {"US to send", "IT to fix", "OK to proceed", "ML to review"},
	} {
		out := maskRoundTrip(t, newEngine(t, nil), in)
		for _, k := range kept {
			if !strings.Contains(out, k) {
				t.Errorf("%q changed in %q", k, out)
			}
		}
	}
}

func TestPeopleNextToThingNounsStillMask(t *testing.T) {
	for in, reals := range map[string][]string{
		"Amber Hall called about the invoice.":                                    {"Amber", "Hall"},
		"Please ask Amber about the room.":                                        {"Amber"},
		"Florence Nightingale was a nurse. Paris Hilton called yesterday.":        {"Florence", "Nightingale", "Paris", "Hilton"},
		"Charles Darwin the person, the Darwin Awards.":                           {"Charles"},
		"Ada Lovelace wrote the first program.":                                   {"Ada", "Lovelace"},
		"Employee: Amber Nakagawa (E-0412).\nDinner at The Amber Room was fine.":  {"Nakagawa", "Employee: Amber"},
		"Regina Novak booked a hotel in Regina SK. Regina called to confirm.":     {"Novak", "Regina called"},
		"Carlo Rossi ran the Monte Carlo sims. Carlo said they converged.":        {"Rossi", "Carlo said"},
		"I live with CARVALHO DE SOUZA, João Pedro since March. João Pedro left.": {"CARVALHO", "SOUZA", "João", "Pedro"},
		"Escribo al señor José Luis Ramírez Ortega, porque la caldera está rota.": {"José", "Luis", "Ramírez", "Ortega"},
		"Bonjour monsieur Jean Pierre Dupont de la Fontaine, merci.":              {"Jean", "Pierre", "Dupont", "Fontaine"},
		"between WILLEM VAN DER HEIJDEN and FATIMA ZAHRA EL IDRISSI today":        {"WILLEM", "HEIJDEN", "FATIMA", "ZAHRA", "IDRISSI"},
		"Present: Lane Pruitt, Oksana Hrytsenko, Thiha\nApologies: Pat N.":        {"Lane", "Pruitt", "Oksana", "Hrytsenko", "Thiha", "Pat"},
	} {
		out := maskRoundTrip(t, newEngine(t, nil), in)
		for _, r := range reals {
			if strings.Contains(out, r) {
				t.Errorf("%q left in %q", r, out)
			}
		}
	}
}

func TestInitialismsMaskAndRestore(t *testing.T) {
	in := "Present: J. R. Adjei (chair), Milica Jovanović, Lane Pruitt\n1. LP reported the council will repaint. JRA to write to the council.\n2. MJ to circulate the quotes.\nLP: happy to help."
	e := newEngine(t, nil)
	out := maskRoundTrip(t, e, in)
	for _, r := range []string{"LP", "JRA", "MJ", "Pruitt", "Adjei", "Jovanović"} {
		if regexp.MustCompile(`\b` + r + `\b`).MatchString(out) {
			t.Errorf("%q left in %q", r, out)
		}
	}
	caps := regexp.MustCompile(`\b\p{Lu}{2,3}\b`).FindAllString(out, -1)
	if len(caps) != 4 {
		t.Fatalf("want 4 initialisms in %q, got %q", out, caps)
	}
	if caps[0] != caps[3] {
		t.Errorf("same initialism got two surrogates: %q", caps)
	}
	for _, c := range caps {
		if detect.CommonInitialism(c) {
			t.Errorf("surrogate %q is a common abbreviation", c)
		}
	}
	s := e.NewSession("answer")
	if _, err := s.Mask(ctx, in); err != nil {
		t.Fatal(err)
	}
	if got := s.Rehydrate("Ask " + caps[1] + " about the letter."); got != "Ask JRA about the letter." {
		t.Errorf("rehydrate: %q", got)
	}
}

func TestInitialismsNeedTheFullNameAndAPersonPosition(t *testing.T) {
	for _, in := range []string{
		"1. LP reported the council will repaint. JRA to write to the council.",
		"Lane Pruitt called. The LP reported gains and our LP is happy.",
		"Ursula Stone and Ian Tate met. US to send aid. IT to fix laptops.",
	} {
		out := maskRoundTrip(t, newEngine(t, nil), in)
		for _, k := range regexp.MustCompile(`\b\p{Lu}{2,3}\b`).FindAllString(in, -1) {
			if !strings.Contains(out, k) {
				t.Errorf("%q changed in %q", k, out)
			}
		}
	}
	out := maskRoundTrip(t, newEngine(t, nil), "Attendees: Ursula Stone, Ian Tate, US, IT\nUS: I agree.")
	if strings.Contains(out, "US") || strings.Contains(out, "IT") {
		t.Errorf("attendee and speaker initialisms left in %q", out)
	}
}

func TestDictionaryNameInsideVenueIsKept(t *testing.T) {
	e := newEngine(t, nil)
	out := maskRoundTrip(t, e, "Will Turner booked The Turner Room and the Turner Award dinner. WT to confirm.")
	if !strings.Contains(out, "The Turner Room") || !strings.Contains(out, "Turner Award") {
		t.Errorf("venue changed: %q", out)
	}
	if strings.HasPrefix(out, "Will Turner") || strings.Contains(out, "WT") {
		t.Errorf("person left: %q", out)
	}
}

func TestLeadingParticleKeepsFamilyRole(t *testing.T) {
	e := newEngine(t, nil)
	out := maskRoundTrip(t, e, "between WILLEM VAN DER HEIJDEN and the tenant, with the consent of Mr van der Heijden.")
	m := regexp.MustCompile(`between \S+ (\S+ \S+ \S+) and .* Mr (\S+ \S+ \S+)\.`).FindStringSubmatch(out)
	if m == nil || !strings.EqualFold(m[1], m[2]) {
		t.Errorf("surname surrogates differ: %q", out)
	}
}

func TestInitialismSurrogateNeverRunsOut(t *testing.T) {
	for _, name := range []string{"Milica Jovanović", "Maria Jensen", "Sam Abbott", "Carl Underwood", "Ines Tavares", "Dana Whitfield", "Gina Bell"} {
		e := newEngine(t, nil)
		in := name + " chaired the meeting.\n" + strings.Join(detect.NameInitials(name)[:1], "") + ": I will send the notes."
		out := maskRoundTrip(t, e, in)
		line := out[strings.IndexByte(out, '\n')+1:]
		sur := line[:strings.IndexByte(line, ':')]
		if sur == detect.NameInitials(name)[0] || detect.CommonInitialism(sur) {
			t.Errorf("%s: surrogate initials %q", name, sur)
		}
	}
}

func TestInitialismSurrogateAcrossVaultKeys(t *testing.T) {
	for k := 1; k <= 40; k++ {
		v, err := vault.Open(vault.NewMemoryBackend(bytes.Repeat([]byte{byte(k)}, 32)))
		if err != nil {
			t.Fatal(err)
		}
		e, err := New(Options{Vault: v, Policy: DefaultPolicy()})
		if err != nil {
			t.Fatal(err)
		}
		out := maskRoundTrip(t, e, "Attendees: Milica Jovanović, Maria Jensen, Oksana Hrytsenko\nMJ: I will send the notes.\nOH: thanks.")
		if strings.Contains(out, "MJ:") || strings.Contains(out, "OH:") {
			t.Fatalf("key %d: initialism left in %q", k, out)
		}
	}
}
