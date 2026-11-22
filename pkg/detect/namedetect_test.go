package detect

import (
	"context"
	"strings"
	"testing"
)

func detectedNames(t *testing.T, text string) []string {
	t.Helper()
	spans, err := NameDetector{}.Detect(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, s := range Resolve(spans) {
		out = append(out, s.Text)
	}
	return out
}

func TestNameDetectorFinds(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"John Smith called Mary Ann Johnson yesterday.", []string{"John Smith", "Mary Ann Johnson"}},
		{"Emergency contact: May Lindenbrook, 509.555.0153", []string{"May Lindenbrook"}},
		{"ONCALL: ABELARD ASHDOWN\nINCIDENT LEAD: INES OKAFOR-LIND", []string{"ABELARD ASHDOWN", "INES OKAFOR-LIND"}},
		{"- 02:31: faith found the expired credential", []string{"faith"}},
		{"ping Chase to chase the vendor, and ask Hope about it", []string{"Chase", "Hope"}},
		{"My sister Ottoline will drive me.", []string{"Ottoline"}},
		{"Our new manager, Kwabena Fairweather, starts Monday.", []string{"Kwabena Fairweather"}},
		{"To: Juniper O'Dell <juniper@halvorsen-bio.example>", []string{"Juniper O'Dell"}},
		{"See https://wiki.example/people/abelard-ashdown for details", []string{"abelard-ashdown"}},
		{"[09:14] ravindra: morning\n[09:15] frank_o: yes", []string{"ravindra", "frank_o"}},
		{"Rose Thornbury is the tenant.", []string{"Rose Thornbury"}},
		{"Thanks,\nPatricia", []string{"Patricia"}},
		{"Attendees: Sarah Lee, Tom Becker and Anita Patel", []string{"Sarah Lee", "Tom Becker", "Anita Patel"}},
	}
	for _, c := range cases {
		got := detectedNames(t, c.in)
		if strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf("%q\n got %q\nwant %q", c.in, got, c.want)
		}
	}
}

func TestNameDetectorLeavesAlone(t *testing.T) {
	for _, in := range []string{
		"Will you check this before Friday?",
		"Will Congress act on this? May Google release it in May?",
		"You are Claude Code, Anthropic's official CLI for Claude.",
		"We use Ruby on Rails with Jenkins and Travis, written in Julia and Haskell.",
		"Grant access to the repo, then mark as done. Chase the vendor.",
		"Mark the task complete. Bill the client on June 3.",
		"I have faith that hope found a way. The bill paid itself.",
		"09:14:01 sshd: accepted key\n[12:00] ERROR: disk full",
		"Sterling work on the draft, by the way.",
		"The Adam optimizer and BERT model; see README.md and max(a, b) or Max.value",
		"Subject: How To Mark Items And Grant Access",
		"From: Ivorygate Legal LLP Accounts <billing@ivorygate.example>",
		"See https://x.example/users/admin-panel and https://x.example/team/data-platform",
		"The trip covers Paris, Georgia and the Jordan River.",
		"grace.hopper@navy.example wrote the func Victor_Init(john_id int)",
	} {
		if got := detectedNames(t, in); len(got) != 0 {
			t.Errorf("%q: unexpected names %q", in, got)
		}
	}
}

func TestTitleCaseShouting(t *testing.T) {
	cases := map[string]string{
		"ONCALL: ABELARD ASHDOWN":         "Oncall: Abelard Ashdown",
		"Use the API and HTTP client":     "Use the API and HTTP client",
		"NOTICE TO ÉLODIE MARTIN, TENANT": "Notice To Élodie Martin, Tenant",
		"plain text stays":                "plain text stays",
	}
	for in, want := range cases {
		got := TitleCaseShouting(in)
		if got != want {
			t.Errorf("%q: got %q want %q", in, got, want)
		}
		if len([]rune(got)) != len([]rune(in)) {
			t.Errorf("%q: rune length changed", in)
		}
	}
}
