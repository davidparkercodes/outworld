package detect

import (
	"context"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

func detectedPeople(t *testing.T, text string) []string {
	t.Helper()
	spans, err := NameDetector{}.Detect(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	var people []entity.Span
	for _, s := range spans {
		if s.Type == entity.Person {
			people = append(people, s)
		}
	}
	var out []string
	for _, s := range Resolve(people) {
		out = append(out, s.Text)
	}
	return out
}

func checkPeople(t *testing.T, in string, want []string) {
	t.Helper()
	got := detectedPeople(t, in)
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("%q\n got %q\nwant %q", in, got, want)
	}
}

func TestHandlesFound(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"TODO(mwilliams): clean this up", []string{"mwilliams"}},
		{"// FIXME(kchen)", []string{"kchen"}},
		{"# HACK(j.okafor): remove after the migration", []string{"j.okafor"}},
		{"NOTE(dthompson42) revisit", []string{"dthompson42"}},
		{"I pushed the branch kchen/fix-auth last night", []string{"kchen"}},
		{"git checkout -b jsmith/retry-logic", []string{"jsmith"}},
		{"rebased onto origin/tnakamura/fix-2 this morning", []string{"tnakamura"}},
		{"@mwilliams can you review", []string{"mwilliams"}},
		{"cc @anita.patel and @okonkwo", []string{"anita.patel", "okonkwo"}},
		{"Reviewed-by: kchen", []string{"kchen"}},
		{"Signed-off-by: tbecker\nAcked-by: @hnakamura", []string{"tbecker", "hnakamura"}},
		{"Co-authored-by: Ines Okafor-Lind <ines@corp.example>", []string{"Ines Okafor-Lind"}},
		{"open /home/jdoe/notes.txt", []string{"jdoe"}},
		{"cd /Users/jdoe/src", []string{"jdoe"}},
		{`at C:\Users\tbecker\Documents\plan.docx`, []string{"tbecker"}},
		{"key in /home/xqzrt/.ssh/id_ed25519", []string{"xqzrt"}},
	}
	for _, c := range cases {
		checkPeople(t, c.in, c.want)
	}
}

func TestHandlesLeftAlone(t *testing.T) {
	for _, in := range []string{
		"TODO(perf): cache this\nTODO(security)\nTODO(v2)\nNOTE(important)\nHACK(cleanup)\nTODO(along)",
		"git checkout feature/login && git merge release/1.2",
		"switch to fix/auth, then branch from hotfix/urgent",
		"git diff origin/main origin/release/1.2 origin/dependabot/npm_and_yarn/lodash",
		"dependabot/npm opened the branch renovate/all-minor",
		"npm i @types/node @babel/core @angular/core",
		"/** @param id the id\n * @returns the row */\n@Override\npublic void run() {}",
		"@media screen { }\n@import url(x.css);\n@font-face { }\n@keyframes spin { }",
		"@property\n@staticmethod\n@dataclass\n@app.route('/x')\n@pytest.mark.parametrize\n@name.setter",
		"@user = User.find(1)\n@price = 3\n@store.save",
		"thanks @team @here @channel @everyone",
		"/home/ubuntu/app /home/runner/work /home/node/.npm /home/user/x /home/app/src /home/git/repo",
		`/Users/Shared/tools /Users/runner/work C:\Users\Public\Desktop`,
		"Signed-off-by: dependabot[bot] <support@github.example>\nTested-by: ci-bot\nReported-by: root",
	} {
		if got := detectedPeople(t, in); len(got) != 0 {
			t.Errorf("%q: unexpected %q", in, got)
		}
	}
}

func TestCompaniesAreNotPeople(t *testing.T) {
	for _, in := range []string{
		"Morgan Stanley reported earnings today.",
		"Goldman Sachs and Wells Fargo both declined to comment.",
		"She moved from Charles Schwab to Raymond James last year.",
		"J.P. Morgan and JP Morgan Chase are the same bank.",
		"Johnson & Johnson and Johnson and Johnson are the same company.",
		"Ernst & Young LLP audited Procter & Gamble.",
		"Rolls-Royce and Rolls Royce engines; a Harley-Davidson; a Mercedes Benz.",
		"He studied at Johns Hopkins and Carnegie Mellon.",
		"Ralph Lauren, Calvin Klein and Tommy Hilfiger were on sale.",
		"Edward Jones Inc sent the statement. Morgan Stanley's forecast changed.",
		"Ben & Jerry's opened a store next to Marks & Spencer.",
		"Counsel is Baker McKenzie; the consultants are Booz Allen Hamilton.",
		"Maria Garcia Holdings bought the building.",
		"Thomas Becker LLP filed the motion. Anita Patel, Inc. hired two people.",
		"John Smith Capital and Sarah Lee Group are both investors.",
		"Anna Brandt University, Brandt Law and Brandt & Sons are unrelated.",
		"The Thomas Becker Foundation funds the Anita Patel Hospital.",
		"Client: Goldman Sachs",
	} {
		if got := detectedPeople(t, in); len(got) != 0 {
			t.Errorf("%q: unexpected %q", in, got)
		}
	}
}

func TestCompanyGuardKeepsPeople(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"Contact: Morgan Stanley", []string{"Morgan Stanley"}},
		{"My brother Raymond James is visiting.", []string{"Raymond James"}},
		{"Thanks,\nMary Kay", []string{"Mary Kay"}},
		{"Greg Abbott spoke first.", []string{"Greg Abbott"}},
		{"Dear Tom Becker, Associates will call you.", []string{"Tom Becker"}},
		{"Morgan Ellis left Morgan Stanley in March.", []string{"Morgan Ellis"}},
		{"Maria Garcia joined the group chat.", []string{"Maria Garcia"}},
	}
	for _, c := range cases {
		checkPeople(t, c.in, c.want)
	}
}

func TestCompanyOrgSpan(t *testing.T) {
	spans, err := NameDetector{}.Detect(context.Background(), "Morgan Stanley reported earnings and Maria Okafor Partners agreed.")
	if err != nil {
		t.Fatal(err)
	}
	orgs := map[string]bool{}
	for _, s := range spans {
		if s.Type == entity.Org {
			orgs[s.Text] = true
		}
	}
	if !orgs["Morgan Stanley"] || !orgs["Maria Okafor Partners"] {
		t.Fatalf("orgs = %v", orgs)
	}
}

func TestFamilyFirstFound(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"Please ask Nguyen Van An about the invoice.", []string{"Nguyen Van An"}},
		{"Tran Thi Mai called twice.", []string{"Tran Thi Mai"}},
		{"Signed by Nguyen Thi Minh Khai.", []string{"Nguyen Thi Minh Khai"}},
		{"Zhang Wei and Wang Fang approved it.", []string{"Zhang Wei", "Wang Fang"}},
		{"Forward it to Li Xiaoming today.", []string{"Li Xiaoming"}},
		{"Kim Min-jun joined the call.", []string{"Kim Min-jun"}},
		{"Park Ji Sung and Lee Min Ho are on the list.", []string{"Park Ji Sung", "Lee Min Ho"}},
		{"Le Van Tam and Do Van Long signed.", []string{"Le Van Tam", "Do Van Long"}},
		{"Hi Li Wei, thanks for the update.", []string{"Li Wei"}},
		{"Zhang Wei's laptop is missing.", []string{"Zhang Wei"}},
		{"Vu Anh Tuan emailed Choi Min-sik.", []string{"Vu Anh Tuan", "Choi Min-sik"}},
	}
	for _, c := range cases {
		checkPeople(t, c.in, c.want)
	}
	spans, _ := NameDetector{}.Detect(context.Background(), "Please ask Nguyen Van An about the invoice.")
	found := false
	for _, s := range spans {
		if s.Source == "names:family-first" {
			found = true
			if s.Canonical != "Van An Nguyen" {
				t.Fatalf("canonical = %q", s.Canonical)
			}
		}
	}
	if !found {
		t.Fatal("no family-first span")
	}
}

func TestFamilyFirstLeftAlone(t *testing.T) {
	for _, in := range []string{
		"Do Not Disturb is on.",
		"He Said She Said is a film.",
		"Park Avenue is closed. Park Lane too.",
		"Sun Microsystems made it. Sun Valley is nice.",
		"He Can Run. He Sang Song.",
		"Long Beach and Long Island.",
		"Tae Kwon Do class. Dang It.",
		"Ho Chi Minh City is large. Tran Hung Dao Street is busy.",
		"Wang Data and Sun China are placeholders.",
		"Ma Long won. Sun Yang swam.",
	} {
		if got := detectedPeople(t, in); len(got) != 0 {
			t.Errorf("%q: unexpected %q", in, got)
		}
	}
	checkPeople(t, "Kim Kardashian posted.", []string{"Kim Kardashian"})
}

func TestSplitHandle(t *testing.T) {
	cases := []struct {
		in, given, family, digits string
		ok                        bool
	}{
		{"mwilliams", "m", "williams", "", true},
		{"kchen", "k", "chen", "", true},
		{"jdoe", "j", "doe", "", true},
		{"dthompson42", "d", "thompson", "42", true},
		{"anitapatel", "anita", "patel", "", true},
		{"xqzrt", "", "", "", false},
		{"perf", "", "", "", false},
		{"priya", "", "", "", false},
	}
	for _, c := range cases {
		g, f, d, ok := SplitHandle(c.in)
		if ok != c.ok || (ok && (g != c.given || f != c.family || d != c.digits)) {
			t.Errorf("SplitHandle(%q) = %q %q %q %v", c.in, g, f, d, ok)
		}
	}
}
