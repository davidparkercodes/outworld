package detect

import (
	"context"
	"github.com/davidparkercodes/outworld/internal/scaletest"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

func dictionaryPeople(t *testing.T, names, text string) []string {
	t.Helper()
	dict, err := ParseDictionaryCSV(strings.NewReader(names))
	if err != nil {
		t.Fatal(err)
	}
	spans, err := dict.Detector().Detect(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, s := range Resolve(spans) {
		if s.Type == entity.Person {
			out = append(out, s.Text)
		}
	}
	return out
}

func TestPersonalPhrasesStillMaskWithPersonEvidence(t *testing.T) {
	expectPeople(t, map[string]string{
		"Please email Tom Collins about the invoice.":       "Tom Collins",
		"Jack Russell called":                               "Jack Russell",
		"Glen Allen emailed me":                             "Glen Allen",
		"Mary Sue called":                                   "Mary Sue",
		"Jane Doe is my real name":                          "Jane Doe",
		"Mary Sue Johnson called":                           "Mary Sue Johnson",
		"Contact: Tom Collins":                              "Tom Collins",
		"Cc: Tom Collins, Maria Lopez":                      "Tom Collins|Maria Lopez",
		"See you then.\n\nThanks,\nTom Collins":             "Tom Collins",
		"Mr. Tom Collins is here":                           "Tom Collins",
		"My colleague Jack Russell is out.":                 "Jack Russell",
		"Alice Springs called me.":                          "Alice Springs",
		"George Mason will join the call.":                  "George Mason",
		"I met Vera Cruz at the office.":                    "Vera Cruz",
		"Tom Collins called. Then I ordered a Tom Collins.": "Tom Collins",
		"Winston-Salem called":                              "Winston-Salem",
	})
	expectNoPeople(t, []string{
		"I ordered a Tom Collins.",
		"She has a Jack Russell terrier.",
		"We drove to Glen Allen, VA.",
		"That character is a Mary Sue.",
		"I visited Alice Springs.",
		"We saw Alice Springs from the plane.",
		"We flew to Monte Carlo.",
		"I visited Monte Carlo and called it a day.",
		"Santa Clara called for rain.",
		"She passed the Turing test.",
		"Winston-Salem is nice.",
		"Flew to Prince George.",
	})
}

func TestRegionCodeNeedsPlaceShape(t *testing.T) {
	for text, want := range map[string]string{
		"Maria, PE class is cancelled.":                  "Maria",
		"Maria, GA build is ready.":                      "Maria",
		"Thanks Maria, PE.":                              "Maria",
		"Maria PE class is cancelled.":                   "Maria",
		"Regina Novak booked a hotel in Regina SK for.":  "Regina Novak",
		"Return: Regina, SK, 3 Jan 2027":                 "",
		"10/06  Hotel, Regina SK, 2 nights":              "",
		"We moved to Regina, SK last year.":              "",
		"Ship to Regina SK S4P 3Y2":                      "",
		"Regina, SK":                                     "",
		"Regina, SK\nCanada":                             "",
		"Regina called. She is in Regina, SK 90210 now.": "Regina",
	} {
		if got := strings.Join(dictionaryPeople(t, "Maria Garcia\nRegina Novak\n", text), "|"); got != want {
			t.Errorf("%q: got %q, want %q", text, got, want)
		}
	}
	expectPeople(t, map[string]string{
		"Elena, GA build is ready":  "Elena",
		"Thanks Elena, PE is done.": "Elena",
	})
	expectNoPeople(t, []string{
		"I flew to Augusta GA then Madison WI.",
		"We moved to Augusta, GA last year.",
		"Return: Regina, SK, 3 Jan 2027",
	})
}

func TestThingNounSurnamesWithPersonEvidence(t *testing.T) {
	expectPeople(t, map[string]string{
		"We met Mary Smith-Jones Hall yesterday":   "Mary Smith-Jones Hall",
		"joined the call with Jordan House":        "Jordan House",
		"Mary Smith Hall called about it":          "Mary Smith Hall",
		"We had lunch with Amber House yesterday.": "Amber House",
		"I spoke to Jordan Park about it.":         "Jordan Park",
	})
	expectNoPeople(t, []string{
		"We met at Jordan House",
		"The Jordan House is open.",
		"Mary Smith Hall is closed.",
		"Visit the Mary Smith Hall.",
		"The call with the Jordan House team went fine.",
	})
}

func TestCommemorativeNamesNeedPersonEvidence(t *testing.T) {
	expectNoPeople(t, []string{
		"John Smith Memorial Fund",
		"The John Smith Memorial Fund is open.",
		"Grace Hopper Award",
		"She won the Grace Hopper Award.",
		"I emailed the John Smith Memorial Fund yesterday.",
		"Donations go to the Maria Lopez Scholarship.",
	})
	expectPeople(t, map[string]string{
		"I emailed John Smith Memorial Fund yesterday": "John Smith",
		"Please thank Grace Hopper Award winners":      "Grace Hopper",
		"Dear John Smith Memorial Fund":                "John Smith",
		"Contact: John Smith Memorial Fund":            "John Smith",
		"Maria Garcia Project Update":                  "Maria Garcia",
	})
}

func surnameFirstSpans(t *testing.T, text string) []string {
	t.Helper()
	spans, err := NameDetector{}.Detect(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, s := range spans {
		if s.Source == "names:surname-first" {
			out = append(out, s.Text+"="+s.Canonical)
		}
	}
	return out
}

func TestCapsSurnameFirstTrimsLabelWords(t *testing.T) {
	for text, want := range map[string]string{
		"The patient is stable. PATIENT GARCIA LOPEZ, Maria Jose was admitted.": "GARCIA LOPEZ, Maria Jose=Maria Jose Garcia Lopez",
		"INVOICE TO GARCIA LOPEZ, Maria Jose":                                   "GARCIA LOPEZ, Maria Jose=Maria Jose Garcia Lopez",
		"London NW1 6XE GARCIA LOPEZ, Maria Jose":                               "GARCIA LOPEZ, Maria Jose=Maria Jose Garcia Lopez",
		"CLIENT NAME GARCIA, Elena":                                             "GARCIA, Elena=Elena Garcia",
		"ACME SMITH, John":                                                      "SMITH, John=John Smith",
		"GARCIA LOPEZ, Maria Jose":                                              "GARCIA LOPEZ, Maria Jose=Maria Jose Garcia Lopez",
		"DE LA CRUZ, Maria":                                                     "DE LA CRUZ, Maria=Maria de la Cruz",
		"PLEASE READ, Daniel.":                                                  "",
		"PATIENT, Maria is here.":                                               "",
		"INVOICE TOTAL, Maria.":                                                 "",
		"URGENT UPDATE, Elena.":                                                 "",
		"PATIENT NOTES, Daniel.":                                                "",
	} {
		if got := strings.Join(surnameFirstSpans(t, text), "|"); got != want {
			t.Errorf("%q: got %q, want %q", text, got, want)
		}
	}
	for _, text := range []string{"The patient is stable. PATIENT GARCIA LOPEZ, Maria Jose was admitted.", "INVOICE TO GARCIA LOPEZ, Maria Jose"} {
		for _, p := range detectedPeople(t, text) {
			if lp := strings.ToLower(p); strings.Contains(lp, "patient") || strings.Contains(lp, "invoice") {
				t.Errorf("%q: label word inside person span %q", text, p)
			}
		}
	}
}

func TestLowercaseHonorificsOnlyTakeNames(t *testing.T) {
	expectPeople(t, map[string]string{
		"frau schmidt rief an":         "schmidt",
		"frau anna schmidt rief an":    "anna schmidt",
		"herr müller kommt später":     "müller",
		"señor garcia llamó ayer":      "garcia",
		"herr Schmidt kommt":           "Schmidt",
		"frau Schmidt rief gestern an": "Schmidt",
	})
	expectNoPeople(t, []string{
		"herr will come later",
		"die frau rief an",
		"frau rief an",
		"eine frau schmidt kenne ich nicht",
		"sr developer martin",
		"the frau said nothing and left",
	})
}

func TestMentionsAreWholeTokens(t *testing.T) {
	for text, want := range map[string]string{
		"ping @jsmithLane about it":  "",
		"cc @jsmithLane, @mwilliams": "mwilliams",
		"@jsmith_dev said so":        "",
		"@jsmith9x said so":          "",
		"ping @jsmith about it":      "jsmith",
		"@jsmith's branch is stale":  "jsmith",
		"thanks @mwilliams.":         "mwilliams",
	} {
		spans, _ := NameDetector{}.Detect(context.Background(), text)
		seen := map[string]bool{}
		var got []string
		for _, s := range spans {
			if s.Source == "names:handle" && !seen[s.Text] {
				seen[s.Text] = true
				got = append(got, s.Text)
			}
		}
		if strings.Join(got, "|") != want {
			t.Errorf("%q: got %q, want %q", text, got, want)
		}
	}
}

func TestContextSearchesAreBounded(t *testing.T) {
	long := strings.Repeat("x ", 4000)
	ls, le := lineBounds(long+"Maria"+long, len(long), len(long)+5)
	if len(long)-ls > contextWindow || le-len(long)-5 > contextWindow {
		t.Errorf("line bounds %d..%d are not bounded", ls, le)
	}
	if ls, le := lineBounds("a\nMaria said\nb", 2, 7); ls != 2 || le != 12 {
		t.Errorf("line bounds %d..%d", ls, le)
	}
	a, b := sentenceBounds(long+"Maria"+long, len(long), len(long)+5)
	if len(long)-a > contextWindow || b-len(long)-5 > contextWindow {
		t.Errorf("sentence bounds %d..%d are not bounded", a, b)
	}
	parts, commas := splitList("Ann Lee and Bo Chan & Cy Dunn, Di Eng")
	if !commas || len(parts) != 4 || parts[1] != [2]int{12, 19} || parts[3] != [2]int{30, 37} {
		t.Errorf("split list %v", parts)
	}
	comps := []companyMatch{{start: 0, end: 5, single: true}, {start: 10, end: 20}, {start: 30, end: 40, single: true}}
	for _, c := range []struct {
		s, e int
		want bool
	}{{0, 5, true}, {1, 4, true}, {3, 8, false}, {8, 12, true}, {19, 25, true}, {20, 30, false}, {31, 39, true}, {29, 41, false}, {45, 50, false}} {
		if got := inCompany(comps, c.s, c.e); got != c.want {
			t.Errorf("inCompany(%d, %d) = %v", c.s, c.e, got)
		}
	}
}

type nameScaleInput struct {
	name  string
	text  string
	names string
}

func nameScaleInputs(size int) []nameScaleInput {
	return []nameScaleInput{
		{"initials_line", "Present: John Michael Doe, Jane Smith\n" + repeatTo("JMD to send. JS to review. ", size), "John Michael Doe\nJane Smith\n"},
		{"cc_and_line", "Cc:" + repeatTo(" and ", size), "Maria Garcia\n"},
		{"lone_names_line", repeatTo("Maria said Johnny and Will agreed with John. ", size), "Maria\nJohnny\nWill\nJohn\n"},
		{"listed_names_line", "Attendees: " + repeatTo("Maria Garcia, Li Na, John Smith, ", size), "Maria Garcia\n"},
		{"family_pairs_line", repeatTo("Li Na, Maria Garcia and Sun Yang or Ma Long, ", size), "Maria Garcia\n"},
		{"multibyte_names_line", repeatTo("Łukasz Żółć, Émile Zoé, Åsa Öberg, Ñuño Íñigo ", size), "Åsa Öberg\n"},
		{"multibyte_caps_line", repeatTo("ÉÀÜ ÖÄÅ ŁÓŚ ", size), "Åsa Öberg\n"},
		{"multibyte_lower_line", repeatTo("łukasz żółć émile zoé åsa öberg ", size), "Åsa Öberg\n"},
		{"multibyte_dash_line", repeatTo("José-María García-Núñez ", size), "José García\n"},
		{"cjk_line", repeatTo("王小明和李雷在北京见面。", size), "Åsa Öberg\n"},
		{"from_company_line", repeatTo("Kevin from Morgan Stanley called from Edward Jones ", size), "Maria Garcia\n"},
		{"comment_possessive_line", repeatTo("// Jordan's hack # Priya's fix /* Sam's */ ", size), "Maria Garcia\n"},
		{"dash_sign_line", repeatTo("-- Sam - Priya Raman \u2013 ", size), "Maria Garcia\n"},
		{"dash_comment_line", repeatTo("ls -- Rose said x -- Mark ", size), "Maria Garcia\n"},
		{"mid_dash_sign_lines", repeatTo("I said no -- Mark\nselect a from t -- Bill\nCheers -- Will\n", size), "Maria Garcia\n"},
		{"lower_sql_line", repeatTo("select from where -- Rose ", size), "Maria Garcia\n"},
		{"nickname_line", repeatTo("William Okafor (Bill) and Patricia (Pat) Riley ", size), "Maria Garcia\n"},
		{"cue_pair_line", repeatTo("Notes from June Carter with Will Power by Mark ", size), "Maria Garcia\n"},
		{"lower_sql_stmt_lines", repeatTo("select a b c from t -- Bill\nupdate t set a -- Will\n", size), "Maria Garcia\n"},
		{"coordinated_pairs_line", repeatTo("Maria Garcia and Bill Park, Grace Hall & Rose Lake or ", size), "Maria Garcia\n"},
		{"thing_person_pair_line", repeatTo("I met Frank Ocean. Rose Garden is open. Will Xyloquist said hi. ", size), "Maria Garcia\n"},
		{"nick_surname_line", repeatTo("Robert (Bob) Smith called the tool (\"Bill\") ", size), "Maria Garcia\n"},
		{"platform_darwin_line", repeatTo("Platform: darwin OS Version: Darwin 24.1.0 linux and Darwin said ", size), "Maria Garcia\n"},
		{"modal_pair_line", repeatTo("Ask Will Xyloquist. Will Xyloquist reviewed it. Frank Ocean, our new hire, said ", size), "Maria Garcia\n"},
		{"month_meeting_line", repeatTo("May and June went to lunch. ask June about it. Lunch with Will Power today. Rose Gold iPhone ", size), "Maria Garcia\n"},
		{"to_header_lines", repeatTo("Rose to Mark: hi May was busy\nSam to Priya: ok\n", size), "Maria Garcia\n"},
		{"hyphen_names_line", repeatTo("email Jean-Luc Picard and Siobhan\u2010Nakamura, well-known Smith\u2212Jones ", size), "Maria Garcia\n"},
		{"dash_comment_code_line", repeatTo("SELECT a = 1; -- Mark as x -- Rose was ", size), "Maria Garcia\n"},
		{"bullet_names_lines", repeatTo("- Sam\n- Alex\n- Jordan\n", size), "Maria Garcia\n"},
		{"bullet_labelled_lines", repeatTo("Team:\n- Sam\n- Tom\n1. Sarah\n2. Mark\n", size), "Maria Garcia\n"},
		{"numbered_long_line", "1. " + repeatTo("Sarah Mark ", size), "Maria Garcia\n"},
		{"bullet_long_line", "- " + repeatTo("Alex Sam ", size), "Maria Garcia\n"},
		{"pr_number_line", repeatTo("PR #412 from kchen/fix ", size), "Maria Garcia\n"},
		{"company_line", repeatTo("Morgan Stanley and Goldman Sachs with JP Morgan ", size), "Maria Garcia\n"},
		{"label_list_line", "To: " + repeatTo("Maria Garcia, John Smith, ", size), "Maria Garcia\n"},
		{"paren_list_line", "Cc: x" + repeatTo(" (a) (b)", size), "Maria Garcia\n"},
		{"minified_json", minifiedNameJSON(size), "Maria Garcia\nJohn Smith\n"},
	}
}

func minifiedNameJSON(size int) string {
	var b strings.Builder
	b.WriteString(`{"people":[`)
	for b.Len() < size {
		b.WriteString(`{"name":"Maria Garcia","owner":"John Smith","manager":"Priya Raman","note":"Maria said Johnny will call JS about the Lincoln Room","cc":"Tom Okafor and Li Na"},`)
	}
	b.WriteString(`{}]}`)
	return b.String()
}

var nameGrowthFactors = map[string]int{"label_list_line": 2, "family_pairs_line": 2, "dash_comment_code_line": 2, "multibyte_lower_line": 4, "multibyte_names_line": 4, "cjk_line": 4, "minified_json": 4, "initials_line": 4, "coordinated_pairs_line": 2, "listed_names_line": 2}

func TestNameDetectorsScaleLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("scale test skipped in short mode")
	}
	size := scaletest.Size(nameGrowthSize)
	smalls := nameScaleInputs(size / scaletest.Input)
	for k, in := range nameScaleInputs(size) {
		dict, err := ParseDictionaryCSV(strings.NewReader(in.names))
		if err != nil {
			t.Fatal(err)
		}
		large, small, grown := in.text, smalls[k].text, size
		if f := nameGrowthFactors[in.name]; f > 1 {
			grown = size * f
			large, small = nameScaleInputs(grown)[k].text, nameScaleInputs(grown / scaletest.Input)[k].text
		}
		build := func(n int) string {
			if n < grown {
				return small
			}
			return large
		}
		for _, d := range []Detector{NameDetector{}, dict.Detector()} {
			d := d
			checkGrowth(t, in.name+" "+d.Name(), build, grown, func(text string) {
				if _, err := d.Detect(context.Background(), text); err != nil {
					t.Fatalf("%s: %v", in.name, err)
				}
			})
		}
	}
}
