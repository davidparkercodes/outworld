package outworld

import (
	"context"
	"strings"
	"testing"
)

func TestHyphenVariantGivenNameMasksTechSurname(t *testing.T) {
	for _, h := range []string{"-", "‐", "‑", "﹣", "－", "−"} {
		for _, sur := range []string{"Watson", "Swift", "Julia", "Hugo", "Homer", "Pascal"} {
			in := "Please email Mary" + h + "Jane " + sur + " today."
			s := newEngine(t, nil).NewSession("p")
			out, err := s.Mask(context.Background(), in)
			if err != nil {
				t.Fatalf("%q: %v", in, err)
			}
			if strings.Contains(out, sur) || strings.Contains(out, "Mary") {
				t.Errorf("%q: raw name left in %q", in, out)
			}
			if back := s.Rehydrate(out); back != in {
				t.Errorf("%q: restored %q", in, back)
			}
		}
	}
}

func TestHyphenVariantsKeepSameSessionDecision(t *testing.T) {
	keep := []string{"I moved to Winston-Salem last year.", "She studied in Ann-Arbor.", "Flights to Wilkes-Barre today.", "The Mercedes-Benz plant", "a Coca-Cola truck", "Stratford-upon-Avon is nice", "the June-July window", "a Rose-colored wall", "Hewlett-Packard printers"}
	mask := []string{"Ask Siobhan-Nakamura now.", "Mary Smith-Jones said hi", "Please email Jean-Luc Picard today."}
	for _, h := range []string{"-", "‐", "‑", "−", "﹣", "－"} {
		for _, in := range keep {
			v := strings.ReplaceAll(in, "-", h)
			out, err := newEngine(t, nil).NewSession("p").Mask(context.Background(), v)
			if err != nil || out != v {
				t.Errorf("%q with %U: changed to %q (%v)", in, []rune(h)[0], out, err)
			}
		}
		for _, in := range mask {
			v := strings.ReplaceAll(in, "-", h)
			s := newEngine(t, nil).NewSession("p")
			out, err := s.Mask(context.Background(), v)
			if err != nil || out == v {
				t.Errorf("%q with %U: left unmasked (%v)", in, []rune(h)[0], err)
			}
			if back := s.Rehydrate(out); back != v {
				t.Errorf("%q with %U: restored %q", in, []rune(h)[0], back)
			}
		}
	}
}

func TestKoreanHyphenGivenNamesRestore(t *testing.T) {
	cases := []string{
		"Seo-yeon Park signed", "Ask Seo-yeon Park today", "Hye-jin Choi signed", "Do-hyun Kang signed", "Yeon-do Park signed",
		"Su-ah Yoon signed", "Jae-won Han signed", "Min-seo Lim signed", "Ye-rin Shin signed", "Ji-hoon Jung signed",
		"Ji-woo Lee signed", "Min-jun Kim signed", "Ha-eun Jung signed", "Seung-hyun Choi called", "Eun-ji Lee called",
		"Hyun-woo Kang called", "So-yeon Yoon called", "Tae-yang Han called", "Jin-ho Lim called", "Na-rae Shin called",
		"Ha-neul Park signed", "Ask Ha-neul Park today", "Da-bit Lee signed", "Han-sol Kim signed", "Bo-ram Yoon signed",
	}
	for _, h := range []string{"-", "‐"} {
		for _, in := range cases {
			v := strings.ReplaceAll(in, "-", h)
			words := strings.Fields(v)
			lead := 0
			if words[0] == "Ask" {
				lead = 1
			}
			given, family := words[lead], words[lead+1]
			s := newEngine(t, nil).NewSession("p")
			out, err := s.Mask(context.Background(), v)
			if err != nil {
				t.Fatalf("%q: %v", v, err)
			}
			for _, part := range append(strings.FieldsFunc(given, func(r rune) bool { return r == '-' || r == '‐' }), family) {
				if strings.Contains(out, part) {
					t.Errorf("%q: %q left raw in %q", v, part, out)
				}
			}
			if back := s.Rehydrate(out); back != v {
				t.Errorf("%q: restored %q", v, back)
			}
		}
	}
}

func TestKoreanFamilyFirstRestore(t *testing.T) {
	cases := []string{
		"Coach Hwang Hyun-woo said hi", "Oh Min-jun said", "Song Min-seo signed", "Moon Ji-ho called", "Bae Su-ah wrote",
		"Ahn Jae-won said", "Yoo Na-rae said", "Ryu Ha-neul said", "Nam Ji-woo said", "Yoon Seo-yeon said",
		"Jeon Hye-jin called", "Baek Do-hyun signed", "Jang Min-seo wrote", "Cho Seung-hyun said", "Kwon Ye-rin signed",
		"Noh Tae-yang said", "Ha Eun-ji called", "Heo Jin-ho said", "Yang So-yeon signed", "Hong Kyung-soo wrote",
		"Min Ji-hoon said", "Park Ji-sung scored",
	}
	for _, h := range []string{"-", "‐", "–"} {
		for _, in := range cases {
			v := strings.ReplaceAll(in, "-", h)
			words := strings.Fields(v)
			lead := 0
			if words[0] == "Coach" {
				lead = 1
			}
			family, given := words[lead], words[lead+1]
			s := newEngine(t, nil).NewSession("p")
			out, err := s.Mask(context.Background(), v)
			if err != nil {
				t.Fatalf("%q: %v", v, err)
			}
			split := func(r rune) bool { return r == ' ' || r == '-' || r == '\u2010' || r == '\u2013' }
			outWords := map[string]bool{}
			for _, w := range strings.FieldsFunc(out, split) {
				outWords[w] = true
			}
			for _, part := range append(strings.FieldsFunc(given, split), family) {
				if outWords[part] {
					t.Errorf("%q: %q left raw in %q", v, part, out)
				}
			}
			if back := s.Rehydrate(out); back != v {
				t.Errorf("%q: restored %q", v, back)
			}
		}
	}
}
