package detect

import (
	"context"
	"github.com/davidparkercodes/outworld/internal/textutil"
	"sort"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

func expectPeople(t *testing.T, cases map[string]string) {
	t.Helper()
	for text, want := range cases {
		got := strings.Join(detectedPeople(t, text), "|")
		if got != want {
			t.Errorf("%q: got %q, want %q", text, got, want)
		}
	}
}

func expectNoPeople(t *testing.T, texts []string) {
	t.Helper()
	for _, text := range texts {
		if got := detectedPeople(t, text); len(got) != 0 {
			t.Errorf("%q: unexpected people %q", text, got)
		}
	}
}

func TestFamilyFirstSentenceContext(t *testing.T) {
	expectPeople(t, map[string]string{
		"Ma Long said the build is fine.":                "Ma Long",
		"Sun Yang will join the call.":                   "Sun Yang",
		"Sun Yang has left the company.":                 "Sun Yang",
		"Ma Long is out sick today.":                     "Ma Long",
		"I asked Ma Long to review it.":                  "Ma Long",
		"We met with Li Na yesterday.":                   "Li Na",
		"Lunch with Li Na, then the demo.":               "Li Na",
		"Thanks from Sun Yang, who sent it.":             "Sun Yang",
		"A note by Li Na, written last week.":            "Li Na",
		"Forwarded to Sun Yang, who will reply.":         "Sun Yang",
		"Sun Yang's review is due.":                      "Sun Yang",
		"Invite Maria Okafor, Sun Yang and Peter Novak.": "Maria Okafor|Sun Yang|Peter Novak",
		"Li Na and Peter Novak will present.":            "Li Na|Peter Novak",
	})
}

func TestFamilyFirstSentenceContextLeavesAlone(t *testing.T) {
	expectNoPeople(t, []string{
		"We drove to Long Island.",
		"Long Island said no to the plan.",
		"Sun Microsystems built it.",
		"Send it to Sun Microsystems, then wait.",
		"Ma Bell was a monopoly.",
		"He Said no.",
		"Do Not Disturb is on.",
		"Park Avenue is closed.",
		"I read Le Monde daily.",
		"Flights to Ho Chi Minh City are cheap.",
		"Han Solo shot first.",
		"Wu Tang Clan forever.",
		"Wu Tang Clan said no.",
		"We asked Long Range to help.",
		"Sun Dance said hello.",
		"Park Side's cafe is open.",
		"A report by Long Range, the planning group.",
		"Sun Yang is a common name.",
		"The Sun Yang exhibit opens today.",
	})
	expectPeople(t, map[string]string{
		"I grew up near Long Island, Peter Novak and others.": "Peter Novak",
		"Maria Okafor, Long Range Planning and the board.":    "Maria Okafor",
		"Do Min and Peter Novak know?":                        "Peter Novak",
	})
}

func TestCompanyNamedPersonEvidence(t *testing.T) {
	expectPeople(t, map[string]string{
		"Edward Jones called me.":            "Edward Jones",
		"Abbott said hi.":                    "Abbott",
		"I emailed Raymond James yesterday.": "Raymond James",
		"Ask Raymond James about the lease.": "Raymond James",
		"Please tell Edward Jones to wait.":  "Edward Jones",
		"We met with Morgan Stanley today.":  "Morgan Stanley",
		"Edward Jones is out sick.":          "Edward Jones",
		"Edward Jones left a message.":       "Edward Jones",
		"Raymond James will join at noon.":   "Raymond James",
		"Our neighbor Edward Jones waved.":   "Edward Jones",
		"cc Raymond James on the thread":     "Raymond James",
	})
}

func TestCompanyUsageStaysOrg(t *testing.T) {
	texts := []string{
		"Edward Jones reported earnings.",
		"Morgan Stanley downgraded the stock.",
		"I bought shares of Abbott.",
		"She works at Raymond James.",
		"Goldman Sachs said in a statement that it would.",
		"Goldman Sachs said it would cut jobs.",
		"Edward Jones said it expects growth.",
		"Abbott announced a recall.",
		"Morgan Stanley acquired the firm and filed the papers.",
		"Wells Fargo called me about fraud.",
		"Charles Schwab emailed me my statement.",
		"My advisor at Edward Jones called me.",
		"Edward Jones called me. Later Edward Jones reported earnings.",
		"Raymond James opened a new office.",
		"Edward Jones Inc called me.",
		"Abbott's quarter was strong.",
	}
	expectNoPeople(t, texts)
	for _, text := range texts {
		spans, _ := NameDetector{}.Detect(context.Background(), text)
		org := false
		for _, s := range spans {
			org = org || s.Type == entity.Org
		}
		if !org {
			t.Errorf("%q: no org span", text)
		}
	}
}

func TestProseHandles(t *testing.T) {
	expectPeople(t, map[string]string{
		"please ask kchen to review":  "kchen",
		"ping mwilliams about it":     "mwilliams",
		"assigned to jdoe":            "jdoe",
		"cc: kchen":                   "kchen",
		"Assignee: mwilliams":         "mwilliams",
		"reviewer kchen, owner: jdoe": "kchen|jdoe",
		"thanks sarahk for the fix":   "sarahk",
		"per kchen this is fine":      "kchen",
		"a patch from mwilliams":      "mwilliams",
		"written by kchen":            "kchen",
		"pairing with mwilliams":      "mwilliams",
		"poke kchen, loop in jsmith2": "kchen|jsmith2",
		"tag kchen and blame jdoe":    "kchen|jdoe",
		"ask @kchen to look":          "kchen",
	})
}

func TestProseHandlesLeftAlone(t *testing.T) {
	expectNoPeople(t, []string{
		"ask sparks",
		"ping google",
		"per mille",
		"from master",
		"by default",
		"with redis",
		"ask around, ping again, from scratch, by design, with care, per request",
		"blame config and tag latest, thanks everyone",
		"with numpy and from stdin by rsync",
		"with class and by signed",
		"from earlier, by signed, with normal",
		"owner: deploy",
		"assigned to nobody",
		"reviewer: dependabot",
		"ask kchen.name or kchen() or kchen_x",
		"review kchen and merge mwilliams",
		"per jdoe",
		"with Kchen",
		"from stand to stand, by manual, with safety",
		"cc: admin",
		"thanks team",
		"from brown to green",
		"by hand, with love, per diem",
	})
}

func TestTodoNames(t *testing.T) {
	expectPeople(t, map[string]string{
		"// TODO(will): fix this":       "will",
		"// FIXME(grant) broken":        "grant",
		"// TODO(may): later":           "may",
		"# XXX(bill): check":            "bill",
		"/* HACK(june): remove */":      "june",
		"// NOTE(april): see the issue": "april",
		"// TODO(will): this will fail": "will",
	})
	expectNoPeople(t, []string{
		"// TODO(perf): speed",
		"// TODO(security): audit",
		"// TODO(cleanup): x",
		"// TODO(v2): x",
		"// TODO(later): x",
		"// TODO(bug): x",
		"// TODO(ABC-123): x",
		"// TODO(august): x",
		"// TODO(jan): x",
		"// TODO(ruby): x",
		"// TODO(adam): x",
		"// TODO(page): x",
		"// TODO: will fix in may",
		"will(may)",
	})
}

func TestDictionaryJoinedHandleSkipsCodeIdentifiers(t *testing.T) {
	people := func(csv, text string) []string {
		d, err := ParseDictionaryCSV(strings.NewReader(csv))
		if err != nil {
			t.Fatal(err)
		}
		spans, _ := d.Detector().Detect(context.Background(), text)
		var out []string
		for _, s := range Resolve(spans) {
			out = append(out, s.Text)
		}
		return out
	}
	plain := "value,type,aliases\nJohn Smith,person,\n"
	for _, text := range []string{
		"class JSmith extends Base {}",
		"type JSmith struct{}",
		"func JSmith() {}",
		"the JSmith module",
		"class JohnSmith extends Base {}",
		"JohnSmith.run() and new JohnSmith()",
		"JohnSmith::create and JohnSmith<T>",
	} {
		if got := people(plain, text); len(got) != 0 {
			t.Errorf("%q: unexpected %q", text, got)
		}
	}
	for text, want := range map[string]string{
		"ask jsmith about it":           "jsmith",
		"Jsmith and JSMITH and jsmith":  "Jsmith|JSMITH|jsmith",
		"J. Smith wrote this.":          "J. Smith",
		"John Smith wrote this.":        "John Smith",
		"Signed by JohnSmith yesterday": "JohnSmith",
		"johnsmith and john.smith":      "johnsmith|john.smith",
	} {
		if got := strings.Join(people(plain, text), "|"); got != want {
			t.Errorf("%q: got %q, want %q", text, got, want)
		}
	}
	listed := "value,type,aliases\nJohn Smith,person,JSmith\n"
	if got := strings.Join(people(listed, "class JSmith extends Base {}"), "|"); got != "JSmith" {
		t.Errorf("listed alias not matched: %q", got)
	}
}

func TestBranchUserPrefixes(t *testing.T) {
	expectPeople(t, map[string]string{
		"git push origin dlopez/fix-cache":         "dlopez",
		"git push -u origin dlopez/fix-cache":      "dlopez",
		"PR from kchen/fix-auth":                   "kchen",
		"pull request from mwilliams/tidy":         "mwilliams",
		"git pull upstream mwilliams/retry-logic":  "mwilliams",
		"git fetch origin kchen/wip":               "kchen",
		"gh pr checkout kchen/fix-auth":            "kchen",
		"git switch -c dlopez/new-cache":           "dlopez",
		"git branch -D kchen/old":                  "kchen",
		"git rebase kchen/base":                    "kchen",
		"git log main..kchen/fix-auth":             "kchen",
		"git diff origin/kchen/fix-auth":           "kchen",
		"it was rebased onto dlopez/main2":         "dlopez",
		"based on kchen/x and off of mwilliams/y":  "kchen|mwilliams",
		"gh pr create --head kchen:fix-auth":       "kchen",
		"compare main...anita-patel:main":          "",
		"merged anita.patel:fix-auth this morning": "",
		"refs/heads/dlopez/fix-cache":              "dlopez",
		"the fork ref kchen:main is stale":         "kchen",
	})
	expectNoPeople(t, []string{
		"git push origin feature/login",
		"git push origin release/1.2",
		"git push origin hotfix/x",
		"git push origin renovate/deps",
		"git push origin dependabot/npm",
		"git push origin main",
		"git pull origin users/list",
		"git diff users/models.py internal/app src/x walker/x.go vendor/lib docs/a handlers/b models/c",
		"git log parker/x.go",
		"git branch main",
		"PR from feature/login, based on release/1.2, off of main",
		"note:this-thing key:value-pair time:12-30 localhost:8080 step:build-all",
		"image: redis:7-alpine and node:18-slim, user:fix-auth",
		"see http://kchen/x and scp host:path/to-file",
		"parker:fix-auth walker:main",
	})
}

func TestFromCompanyIsAffiliation(t *testing.T) {
	expectPeople(t, map[string]string{
		"Kevin from Morgan Stanley called.":                       "Kevin",
		"Kevin Smith from Morgan Stanley called.":                 "Kevin Smith",
		"He called from Morgan Stanley about the loan.":           "",
		"called from Morgan Stanley about the loan.":              "",
		"She was hired from Edward Jones last year.":              "",
		"Kevin at Morgan Stanley called.":                         "Kevin",
		"I got a call from Edward Jones today.":                   "Edward Jones",
		"Edward Jones called me.":                                 "Edward Jones",
		"Kevin from Edward Jones called. Edward Jones called me.": "Kevin",
	})
}

func TestPRNumberBeforeFrom(t *testing.T) {
	expectPeople(t, map[string]string{
		"PR #412 from kchen/fix-login is ready.":           "kchen",
		"PR 412 from kchen/fix-login is ready.":            "kchen",
		"pull request #412 from kchen/fix-login is ready.": "kchen",
		"MR !17 from kchen/fix-login is ready.":            "kchen",
		"merge request !17 from mwilliams/tidy":            "mwilliams",
	})
	expectNoPeople(t, []string{
		"PR #412 from feature/login is ready.",
		"PR 412 from release/1.2",
	})
}

func TestDashSignature(t *testing.T) {
	expectPeople(t, map[string]string{
		"Thanks\n-- Sam\n":         "Sam",
		"Thanks\n--Sam\n":          "",
		"Thanks\n-- Priya Raman\n": "Priya Raman",
		"Thanks\n- Priya\n":        "Priya",
		"Thanks\n– Sam\n":          "Sam",
		"Thanks,\n  -- Jordan\n":   "Jordan",
	})
	expectNoPeople(t, []string{
		"-- see below\n",
		"run it with\n--Sam\n",
		"- Monday\n- Apple\n",
		"Countries:\n- Jordan\n- Egypt\n",
		"-- Monday\n",
		"-- Will\n",
		"-- Virginia\n",
		"-- Python\n",
		"-- API\n",
		"- Sam the build is fine\n",
	})
}

func TestCommentPossessiveFirstName(t *testing.T) {
	expectPeople(t, map[string]string{
		"// Jordan's hack\nx := 1\n": "Jordan",
		"# Priya's workaround\n":     "Priya",
		"/* Sam's fix */\n":          "Sam",
		"x := 1 // Sam's hack\n":     "Sam",
		" * Jordan’s note\n":         "Jordan",
	})
	expectNoPeople(t, []string{
		"// Python's GIL\n",
		"// Go's scheduler\n",
		"// Monday's build\n",
		"// Amazon's API\n",
		"// April's release\n",
		"// Virginia's law\n",
		"// Rose's garden\n",
		"Jordan's border is closed.\n",
		"x#Sam's\n",
	})
}

func TestMidLineDashCommentIsNotPersonEvidence(t *testing.T) {
	expectPeople(t, map[string]string{
		"Thanks\n-- Sam":                         "Sam",
		"we met the team and Rose left":          "Rose",
		"I met them -- Rose was there":           "Rose",
		"we talked -- Mark was late":             "Mark",
		"Select one -- Rose was there":           "Rose",
		"x = 1\nwe talked -- Mark was late":      "Mark",
		"we met (again) -- Rose was there":       "Rose",
		"> we talked -- Mark was late":           "Mark",
		"**update** -- Rose was there":           "Rose",
		"we argued ;) -- Mark was late":          "Mark",
		"echo $x -- Rose here":                   "Rose",
		"we set a = b and c -- Rose was there":   "Rose",
		"SELECTED few -- Rose was there":         "Rose",
		"tried `go vet` -- Rose was right":       "Rose",
		"thanks =) -- Rose was there":            "Rose",
		"GROUP chat was quiet -- Rose was there": "Rose",
		"I will UPDATE you -- Rose was there":    "Rose",
		"score = 3 -- Rose won":                  "Rose",
		"ORDER arrived late -- Rose was there":   "Rose",
		"Re: GROUP dinner -- Rose was there":     "Rose",
		"we met at noon; -- Rose was there":      "Rose",
		"sum is a=b -- Rose was there":           "Rose",
		"x == y -- Mark was late":                "Mark",
		"Update on the launch -- Rose was there": "Rose",
		"x = 1 -- Will was here":                 "Will",
	})
	expectNoPeople(t, []string{
		"CREATE TABLE users; -- Mark as deleted",
		"SELECT * FROM t -- Bill totals",
		"select id from t; -- Rose totals",
		"DROP TABLE t -- Mark as gone",
		"x=1 -- Mark it",
		"let y = foo() -- Mark as done",
		"val = `cmd` -- Bill totals",
		"if (a) { -- Mark as seen",
		"count = 0 -- Bill totals",
		"foo(); -- Will fix",
		"  const total := 0 -- Mark as unused",
		"UPDATE users SET a = 1 -- Mark as read",
		"alter table t add c -- Bill totals",
		"x = 1 -- Will fix later",
		"x = 1 --Mark it",
		"-- Grant access",
		"-- Will was here",
	})
}

func TestBulletNameLists(t *testing.T) {
	expectPeople(t, map[string]string{
		"- Sam\n- Alex\n- Jordan\n":    "Sam|Alex|Jordan",
		"* Priya Raman\n* Sam\n":       "Priya Raman|Sam",
		"- Alex\n- Rose\n- Lily\n":     "Alex|Rose|Lily",
		"- Alex\r\n- Sam\r\n":          "Alex|Sam",
		"  - Sam\n  - Alex\n":          "Sam|Alex",
		"- Sam\n\n- Alex\n":            "Alex",
		"- Alex\n* Sam\n":              "Alex",
		"- Alex\n- Monday\n":           "Alex",
		"Notes:\n- Sam\n- Priya\ndone": "Sam|Priya",
	})
	expectNoPeople(t, []string{
		"Steps:\n- Grace period\n- Will fix\n",
		"- Monday\n- Apple\n",
		"- Jordan\n- Chad\n- Georgia\n",
		"- Sam\n",
		"- Rose\n- Lily\n- Daisy\n",
		"- Sam\n- Tom\n- Max\n",
		"Tasks:\n- Review\n- Mark\n",
		"- Apple\n- Google\n- Sam\n",
	})
}

func TestLabelledAndNumberedBulletLists(t *testing.T) {
	expectPeople(t, map[string]string{
		"Present:\n- Grace\n- Will\n- Mark\n":  "Grace|Will|Mark",
		"Attendees:\n1. Bill\n2. Rose Smith\n": "Bill|Rose Smith",
		"Team:\n- Sam\n- Tom\n- Max\n":         "Sam|Tom|Max",
		"Attendees:\r\n* Sam\r\n* Tom\r\n":     "Sam|Tom",
		"**Present:**\n- Rose\n- Lily\n":       "Rose|Lily",
		"1. Sarah\n2. Mark\n3. Rose\n":         "Sarah|Mark|Rose",
		"1) Sarah\n2) Mark\n":                  "Sarah|Mark",
		"Team:\n1. Sam\n2. Tom\n":              "Sam|Tom",
		"1. Sarah\n- Mark\n":                   "Sarah",
		"Team: Sam\n- Tom\n- Max\n":            "",
		"Team:\n- Sam\n\n- Tom\n- Max\n":       "",
	})
	expectNoPeople(t, []string{
		"Team:\n- Review\n- Deploy\n",
		"Team: - Backend/Frontend/QA\n",
		"Team:\n- Backend\n- QA\n",
		"Team:\n- Will\n- Python\n",
		"Ingredients:\n1. Rose water\n2. Sage\n",
		"Ingredients:\n1. Rose\n2. Sage\n",
		"- Grace\n- Will\n- Mark\n",
		"Team:\n- Sam\n",
		"Steps:\n- Sam\n- Tom\n- Max\n",
		"- Sam\n- Tom\n- Max\n",
		"1. Monday\n2. Apple\n",
		"1. Install\n2. Grant access\n",
		"1. Rose\n2. Lily\n3. Daisy\n",
		"Team:\n- Monday\n- Sam\n",
	})
}

func TestMidLineDashSignOff(t *testing.T) {
	expectPeople(t, map[string]string{
		"I said no -- Mark":                     "Mark",
		"ok -- Mark":                            "Mark",
		"See you soon -- Mark":                  "Mark",
		"I said no -- Priya":                    "Priya",
		"Cheers -- Will":                        "Will",
		"Cheers -- Georgia":                     "Georgia",
		"Thank you -- Bill":                     "Bill",
		"ok -- Mark Smith":                      "Mark Smith",
		"Thanks\n-- Will":                       "Will",
		"Thanks,\n-- Will":                      "Will",
		"Talk soon\n– Will":                     "Will",
		"we select -- Rose was there":           "Rose",
		"a message from them -- Rose was there": "Rose",
	})
	expectNoPeople(t, []string{
		"x = 1 -- Will",
		"-- Will",
		"we said\n-- Will",
		"it broke -- Monday",
		"fine -- Georgia",
		"--- Mark",
		"ok -- Python",
		"select * from t -- Bill totals",
		"select id from users where x -- Mark",
		"update t set a -- Will",
	})
}

func TestLowerSQLNeedsStatementShape(t *testing.T) {
	expectPeople(t, map[string]string{
		"not sure where it came from -- Mark":    "Mark",
		"we got into the update -- Rose":         "Rose",
		"I'll set up the update -- Will":         "Will",
		"I will select it from the list -- Mark": "Mark",
		"a = b -- said Mark":                     "Mark",
		"2 + 2 = 4 -- Mark":                      "Mark",
	})
	expectNoPeople(t, []string{
		"select * from t -- Bill totals",
		"  select id from users where x -- Mark",
		"update t set a -- Will",
		"insert into t values (1) -- Rose",
		"delete from t -- Mark",
		"WITH x AS (SELECT 1) -- Mark as cte",
	})
}

func TestCueWordGivenSurnamePairs(t *testing.T) {
	expectPeople(t, map[string]string{
		"Notes from June Carter about the launch.": "June Carter",
		"from Pat Smith":  "Pat Smith",
		"by Mark Twain":   "Mark Twain",
		"from Lily Allen": "Lily Allen",
	})
	expectNoPeople(t, []string{
		"from Grace Period settings",
		"Notes from Frank Ocean Records",
		"from April Fools",
		"from Monday Morning",
		"with Rose Garden",
		"from Chase Bank",
		"with Victoria Station",
		"from June Release",
	})
}

func TestParenthesisedNicknames(t *testing.T) {
	expectPeople(t, map[string]string{
		"William Okafor (Bill) joined.":     "William Okafor|Bill",
		"Alexandra \"Sandy\" Novak joined.": "Alexandra|Sandy|Novak",
		"Patricia (Pat) Riley joined.":      "Patricia|Pat|Riley",
		"Mary Smith “Bill” said":            "Mary Smith|Bill",
	})
	expectPeople(t, map[string]string{
		"Jordan Lee (QA) joined.":        "Jordan Lee",
		"Maria Garcia (Finance) joined.": "Maria Garcia",
		"Jordan Lee (Georgia) joined.":   "Jordan Lee",
		"Jordan Lee (May) joined.":       "Jordan Lee",
		"Jordan Lee (Notes) joined.":     "Jordan Lee",
		"Mary Smith \"Bill\"s":           "Mary Smith",
	})
}

func TestQuotedNicknameInParentheses(t *testing.T) {
	expectPeople(t, map[string]string{
		"William Okafor (\"Bill\") joined":         "William Okafor|Bill",
		"William Okafor (\u201cBill\u201d) joined": "William Okafor|Bill",
		"William Okafor ('Bill') joined":           "William Okafor|Bill",
		"Jordan Lee (\"QA\") joined":               "Jordan Lee",
		"Maria Garcia ('Finance') joined":          "Maria Garcia",
	})
}

func TestCoordinatedFullNamePairs(t *testing.T) {
	expectPeople(t, map[string]string{
		"Also Bill Park and Hope Diaz.":     "Bill Park|Hope Diaz",
		"Maria Garcia and Will Power":       "Maria Garcia|Will Power",
		"Maria Garcia and Grace Hall":       "Maria Garcia|Grace Hall",
		"Maria Garcia & June Carter":        "Maria Garcia|June Carter",
		"Hope Diaz, Mark Hall or Bill Park": "Hope Diaz|Mark Hall|Bill Park",
		"Rose Garden and Maria Garcia":      "Maria Garcia",
		"Central Park and Maria Garcia":     "Maria Garcia",
		"Maria Garcia or Rose Lake":         "Maria Garcia",
		"Maria Garcia and Victoria Station": "Maria Garcia",
		"Maria Garcia and Monday Morning":   "Maria Garcia",
		"Maria Garcia and Chase Bank":       "Maria Garcia",
		"Maria Garcia and Joy Street":       "Maria Garcia",
		"Maria Garcia and Rose Bay":         "Maria Garcia",
		"Maria Garcia, Frank Ocean":         "Maria Garcia",
	})
	expectNoPeople(t, []string{
		"Bill Park and Rose Garden",
	})
}

func TestReviewNamePairs(t *testing.T) {
	expectPeople(t, map[string]string{
		"Will Xyloquist said hi":                   "Will Xyloquist",
		"Hope Xyloquist said hi":                   "Hope Xyloquist",
		"Frank Ocean said hi":                      "Frank Ocean",
		"I met Frank Ocean.":                       "Frank Ocean",
		"Frank Ocean is here":                      "Frank Ocean",
		"Notes from June Carter about the launch.": "June Carter",
		"from Pat Smith":                           "Pat Smith",
		"by Mark Twain":                            "Mark Twain",
		"from Lily Allen":                          "Lily Allen",
		"from Will Smith":                          "Will Smith",
		"from Mark Green":                          "Mark Green",
		"with Will Power said hi":                  "Will Power",
	})
	expectNoPeople(t, []string{
		"Pacific Ocean was calm",
		"I visited Victoria Station.",
		"Rose Garden is open",
		"Will Xyloquist and the team",
		"with Will Power",
		"by Mark Down",
		"from Frank Talk",
		"with Rose Gold",
		"with Ray Tracing",
		"with Victoria Secret",
		"Rendered by Grant Studio",
		"from Carter Lake",
		"from Sierra Nevada",
	})
}

func TestNicknameSurnameAndWrappedNames(t *testing.T) {
	expectPeople(t, map[string]string{
		"Robert (Bob) Smith called": "Robert|Bob|Smith",
		"William (\"Bill\") Gates":  "William|Bill|Gates",
		"Robert (Bob) the builder":  "Robert|Bob",
		"Robert (Bob) Garden":       "Robert|Bob",
		"Hope Diaz (\"Bill\") said": "Hope Diaz|Bill",
	})
	expectNoPeople(t, []string{
		"the tool (\"Bill\") was",
		"call it (\"Will\")",
		"the tool (Bill) was",
	})
}

func TestProseBeforeSemicolonIsNotCode(t *testing.T) {
	expectPeople(t, map[string]string{
		"I fixed it; -- Rose": "Rose",
		"done; -- Rose":       "Rose",
		"ok, done; -- Rose":   "Rose",
		"Done; -- Mark":       "Mark",
	})
	expectNoPeople(t, []string{
		"foo(); -- Will fix",
		"CREATE TABLE users; -- Mark as deleted",
		"x = 1; -- Bill totals",
		"my_var; -- Rose",
		"obj.run; -- Rose",
	})
}

func TestPlatformDarwinIsNotAName(t *testing.T) {
	expectPeople(t, map[string]string{
		"Charles Darwin wrote it":    "Charles Darwin",
		"Darwin said hi":             "Darwin",
		"I emailed Darwin yesterday": "Darwin",
		"Ask Darwin about it":        "Darwin",
	})
	expectNoPeople(t, []string{
		"Platform: darwin\n",
		" - Platform: darwin\n - OS Version: Darwin 24.1.0\n",
		"OS: Darwin\n",
		"GOOS=darwin GOARCH=arm64",
		"darwin/arm64",
		"Darwin-x64 build",
		"Darwin arm64 build",
		"builds for linux and Darwin",
		"Darwin and Linux are kernels",
		"uname: Darwin",
	})
}

func TestLowercaseSignoffNeedsNameTable(t *testing.T) {
	expectPeople(t, map[string]string{
		"thanks, Maria":  "Maria",
		"thanks, maria":  "maria",
		"Thanks, Low":    "Low",
		"thanks, garcia": "garcia",
		"Cheers, priya":  "priya",
	})
	expectNoPeople(t, []string{
		"thanks, low",
		"thanks, young",
		"thanks, white",
		"thanks, long",
		"thanks, gross",
		"hi low",
		"hey gross",
	})
}

func TestModalGivenNamePairsWithPersonEvidence(t *testing.T) {
	expectPeople(t, map[string]string{
		"Will Xyloquist reviewed it":               "Will Xyloquist",
		"Will Xyloquist merged the PR":             "Will Xyloquist",
		"Ask Will Xyloquist about the deploy.":     "Will Xyloquist",
		"Tell Will Xyloquist":                      "Will Xyloquist",
		"May Xyloquist reviewed it":                "May Xyloquist",
		"Mark Xyloquist approved":                  "Mark Xyloquist",
		"Ask Grant Xyloquist":                      "Grant Xyloquist",
		"Will Smith":                               "Will Smith",
		"Frank Ocean, our new hire, starts Monday": "Frank Ocean",
		"Frank Ocean, the lead engineer, said":     "Frank Ocean",
	})
	expectNoPeople(t, []string{
		"Will Anyone help",
		"Will Anyone reviewed it",
		"Ask Will Anyone",
		"Will Kubernetes scale",
		"Pacific Ocean, our favourite, is cold",
		"Rose Garden, our favourite, is open",
	})
}

func TestMonthGivenNamesWithPersonEvidence(t *testing.T) {
	expectPeople(t, map[string]string{
		"May and June went to lunch":  "May|June",
		"June and Mark went to lunch": "June|Mark",
		"ask June about it":           "June",
		"tell May":                    "May",
		"loop in June":                "June",
		"cc June":                     "June",
		"June said hi":                "June",
		"June reviewed it":            "June",
	})
	expectNoPeople(t, []string{
		"in June",
		"by June",
		"the June release",
		"May, June and July",
		"from March to June",
		"June-July",
		"June 5 meeting",
		"Call June 2026",
		"see you in June",
		"visit June",
		"June went by fast",
		"We ship in May and June went smoothly",
		"May I ask",
	})
}

func TestCompoundPairsNeedPersonContext(t *testing.T) {
	expectPeople(t, map[string]string{
		"Lunch with Will Power today.":   "Will Power",
		"Meeting with Will Power at 3.":  "Will Power",
		"Notes with Will Power today.":   "Will Power",
		"Coffee with Mark Down tomorrow": "Mark Down",
		"Sync with Frank Talk at 3":      "Frank Talk",
		"Rose Gold said hi":              "Rose Gold",
		"Will Power reviewed it":         "Will Power",
	})
	expectNoPeople(t, []string{
		"with Will Power",
		"the will power to",
		"sheer will power",
		"Rose Gold iPhone",
		"Bought a Rose Gold iPhone",
		"# Will Power",
		"# Will Power\nbody text",
		"by Mark Down",
	})
}

func TestMonthCopulaNeedsPersonState(t *testing.T) {
	expectPeople(t, map[string]string{
		"May was sick":       "May",
		"June is out today":  "June",
		"June is on leave":   "June",
		"June was on call":   "June",
		"June is late again": "June",
	})
	expectNoPeople(t, []string{
		"May was busy",
		"June is busy",
		"June was hectic",
		"June will be busy",
		"June has been quiet",
		"May Day",
		"May Day parade",
	})
}

func TestMeetingContextNeedsSurnameOrTime(t *testing.T) {
	expectPeople(t, map[string]string{
		"Lunch with Will Power today.": "Will Power",
		"Lunch with Rose Smith":        "Rose Smith",
		"Sync with Frank Talk at 3":    "Frank Talk",
	})
	expectNoPeople(t, []string{
		"Lunch with Grace Notes",
		"Lunch with Mark Down",
		"Lunch with Will Power",
	})
}

func TestNameToNameHeader(t *testing.T) {
	expectPeople(t, map[string]string{
		"Rose to Mark: hi":        "Rose|Mark",
		"Sam to Priya: ok":        "Sam|Priya",
		"  Grace to Will: thanks": "Grace|Will",
	})
	expectNoPeople(t, []string{
		"Monday to Friday: open",
		"Paris to London: 2h",
		"Up to Date: yes",
		"June to August: closed",
	})
}

func TestHyphenatedNames(t *testing.T) {
	expectPeople(t, map[string]string{
		"Please email Jean-Luc Picard today.": "Jean-Luc Picard",
		"Ask Siobhan-Nakamura now.":           "Siobhan-Nakamura",
		"Hi Jean-Luc,":                        "Jean-Luc",
		"Jean-Luc said hi":                    "Jean-Luc",
		"Ask Jean-Xyloquist now.":             "Jean-Xyloquist",
		"Mary Smith-Jones said":               "Mary Smith-Jones",
		"Ask Mary Smith-Xyloquist":            "Mary Smith-Xyloquist",
	})
	expectNoPeople(t, []string{
		"Elena-she said",
		"a well-known fix",
		"the Coca-Cola account",
	})
}

func TestHyphenVariantsGiveSameSpans(t *testing.T) {
	inputs := []string{
		"Please email Jean-Luc Picard today.",
		"Ask Siobhan-Nakamura now.",
		"Hi Jean-Luc,",
		"Mary Smith-Jones said",
		"Elena-she said",
	}
	for _, in := range inputs {
		want := strings.Join(detectedPeople(t, in), "|")
		for _, h := range []string{"‐", "‑", "﹣", "－", "−"} {
			v := strings.ReplaceAll(in, "-", h)
			got := strings.ReplaceAll(strings.Join(detectedPeople(t, v), "|"), h, "-")
			if got != want {
				t.Errorf("%q with %U: got %q, want %q", in, []rune(h)[0], got, want)
			}
		}
	}
}

func TestHyphenVariantCompaniesStayOrg(t *testing.T) {
	for _, h := range []string{"-", "‐", "‑", "﹣", "－", "−"} {
		for _, name := range []string{"Mercedes-Benz", "Rolls-Royce", "Hewlett-Packard"} {
			in := "I drove the " + strings.ReplaceAll(name, "-", h) + " to work."
			if got := detectedPeople(t, in); len(got) != 0 {
				t.Errorf("%q: unexpected people %q", in, got)
			}
		}
	}
}

func TestTechSurnameAfterStrongGiven(t *testing.T) {
	for _, h := range []string{"-", "‐", "‑", "﹣", "－", "−"} {
		for _, sur := range []string{"Watson", "Swift", "Julia", "Hugo", "Homer", "Pascal"} {
			in := "Please email Mary" + h + "Jane " + sur + " today."
			want := "Mary" + h + "Jane " + sur
			if got := strings.Join(detectedPeople(t, in), "|"); got != want {
				t.Errorf("%q: got %q, want %q", in, got, want)
			}
		}
	}
	expectNoPeople(t, []string{
		"We rewrote it in Swift today.",
		"Python and Rust are fine.",
	})
}

var hyphenKeepCases = []string{
	"I moved to Winston-Salem last year.",
	"She studied in Ann-Arbor.",
	"Flights to Wilkes-Barre today.",
	"The Mercedes-Benz plant",
	"a Coca-Cola truck",
	"Rolls-Royce engines",
	"Hewlett-Packard printers",
	"Stratford-upon-Avon is nice",
	"the June-July window",
	"a Rose-colored wall",
	"a well-known fix",
}

var hyphenMaskCases = []string{
	"Please email Jean-Luc Picard today.",
	"Ask Siobhan-Nakamura now.",
	"Mary Smith-Jones said hi",
	"Hi Jean-Luc,",
	"Min-jun Kim said hi",
}

func TestHyphenVariantsKeepSameDecision(t *testing.T) {
	for _, in := range append(append([]string{}, hyphenKeepCases...), hyphenMaskCases...) {
		want := strings.Join(detectedPeople(t, in), "|")
		for _, h := range []string{"‐", "‑", "−", "﹣", "－"} {
			v := strings.ReplaceAll(in, "-", h)
			if got := strings.ReplaceAll(strings.Join(detectedPeople(t, v), "|"), h, "-"); got != want {
				t.Errorf("%q with %U: got %q, ASCII gives %q", in, []rune(h)[0], got, want)
			}
		}
	}
	for _, in := range hyphenKeepCases {
		if got := detectedPeople(t, in); len(got) != 0 {
			t.Errorf("%q: keep case masked %q", in, got)
		}
	}
	for _, in := range hyphenMaskCases {
		if got := detectedPeople(t, in); len(got) == 0 {
			t.Errorf("%q: mask case left unmasked", in)
		}
	}
}

var koreanHyphenMask = []string{
	"Seo-yeon Park signed", "Ask Seo-yeon Park today", "Hye-jin Choi signed", "Do-hyun Kang signed", "Yeon-do Park signed",
	"Su-ah Yoon signed", "Jae-won Han signed", "Min-seo Lim signed", "Ye-rin Shin signed", "Ji-hoon Jung signed",
	"Ji-woo Lee signed", "Min-jun Kim signed", "Ha-eun Jung signed", "Seung-hyun Choi called", "Eun-ji Lee called",
	"Hyun-woo Kang called", "So-yeon Yoon called", "Tae-yang Han called", "Jin-ho Lim called", "Na-rae Shin called",
	"Kyung-soo Park wrote", "Hee-jin Kim wrote",
	"Ha-neul Park signed", "Ask Ha-neul Park today", "Ha-byul Choi signed", "Da-bit Lee signed", "Han-sol Kim signed", "Bo-ram Yoon signed", "Ga-saem Jung signed", "Seo-yeon Miller called",
}

func TestKoreanFamilyFirstAfterPlace(t *testing.T) {
	expectPeople(t, map[string]string{
		"28013 Madrid Park Ji-sung": "Park Ji-sung",
		"I met Park Ji-sung":        "Park Ji-sung",
		"Madrid Park is open":       "",
		"We walked in Madrid Park.": "",
	})
}

var koreanHyphenKeep = []string{
	"Drive-in Park", "Check-in Kim", "Tae-kwon Do", "the drive-in park", "Sign-in Kim page",
	"Log-in Lee field", "Plug-in Park", "Built-in Han", "Pop-up Park", "Walk-in Choi clinic", "Look-up Kang", "Set-up Shin", "So-so Park",
	"Kim-chi Jar", "Mu-shu Pork", "the Mu-shu Pork was great", "Order the Kim-chi Fried Rice", "Tae-bo Class at 6", "Hyun-dai Motors",
	"Mi-so Soup", "So-ju Shot", "Ban-chan Bowl", "Bul-gogi Bowl", "Sam-sung Galaxy", "Bu-san Station", "Dae-jeon Line", "Seo-ul Tower",
	"In-cheon Airport", "Dae-gu Line", "Gwang-ju Station", "Je-ju Island", "Bu-san Park", "Sam-sung Lee", "Log-in Page", "Pop-up Shop",
	"Hip-hop Song", "Ping-pong Ball", "Go-kart Track", "Hong-kong Park", "Cha-cha Song", "Ha-ha Moment", "Kung-fu Panda", "Wi-fi Park",
	"Sea-side Park", "K-pop Song", "Seoul-born Park", "Jeon-ju Bibimbap", "Re-run King", "Do-it Young", "To-fu Burger", "Tae-kwon-do Class",
	"Dae-woo Matiz", "Hyun-dai Sonata", "Kim-bap Roll", "Bi-bim-bap Bowl", "Gal-bi Plate", "Man-du Soup", "Ram-yeon Cup", "Mak-geolli Bottle",
	"Gang-nam Style", "Myeong-dong Street", "Han-bok Rental", "Han-gul Font", "Su-won Station", "Ul-san Port", "Ji-mmy Choo",
	"Ping-pong Park", "Ha-ha Park", "Cha-cha Kim",
}

func TestKoreanHyphenGivenNames(t *testing.T) {
	for _, h := range []string{"-", "‐"} {
		for _, in := range koreanHyphenMask {
			v := strings.ReplaceAll(in, "-", h)
			got := detectedPeople(t, v)
			words := strings.Fields(v)
			lead := 0
			if words[0] == "Ask" {
				lead = 1
			}
			want := words[lead] + " " + words[lead+1]
			if len(got) != 1 || got[0] != want {
				t.Errorf("%q: got %q, want %q", v, got, want)
			}
		}
		for _, in := range koreanHyphenKeep {
			v := strings.ReplaceAll(in, "-", h)
			for _, p := range detectedPeople(t, v) {
				if strings.ContainsFunc(p, textutil.IsHyphen) || strings.Contains(p, " ") {
					t.Errorf("%q: hyphenated word joined into a person span %q", v, p)
				}
			}
		}
	}
}

var koreanFamilyFirstMask = []string{
	"Coach Hwang Hyun-woo said hi", "Oh Min-jun said", "Song Min-seo signed", "Moon Ji-ho called", "Bae Su-ah wrote",
	"Ahn Jae-won said", "Yoo Na-rae said", "Ryu Ha-neul said", "Nam Ji-woo said", "Yoon Seo-yeon said",
	"Jeon Hye-jin called", "Baek Do-hyun signed", "Jang Min-seo wrote", "Cho Seung-hyun said", "Kwon Ye-rin signed",
	"Noh Tae-yang said", "Ha Eun-ji called", "Heo Jin-ho said", "Yang So-yeon signed", "Hong Kyung-soo wrote",
	"Min Ji-hoon said", "Park Ji-sung scored",
}

func TestKoreanFamilyFirstNames(t *testing.T) {
	for _, h := range []string{"-", "‐", "–"} {
		for _, in := range koreanFamilyFirstMask {
			v := strings.ReplaceAll(in, "-", h)
			words := strings.Fields(v)
			lead := 0
			if words[0] == "Coach" {
				lead = 1
			}
			want := words[lead] + " " + words[lead+1]
			if got := detectedPeople(t, v); len(got) != 1 || got[0] != want {
				t.Errorf("%q: got %q, want %q", v, got, want)
			}
		}
	}
	expectPeople(t, map[string]string{
		"28013 Madrid Park Ji-sung": "Park Ji-sung",
		"Nam June Paik made it":     "Nam June Paik",
		"Park Ji‒sung":              "Park Ji‒sung",
		"Mary–Jane Watson":          "Mary–Jane Watson",
		"Jean–Luc Picard":           "Jean–Luc Picard",
		"Elena–she said":            "Elena",
	})
	expectNoPeople(t, []string{
		"Oh well, that failed.",
		"Moon landing footage",
		"Song lyrics are here",
		"Oh no",
		"Moon Pie",
		"Song Book",
		"Mon–fri hours",
		"Ha–ha that is funny",
		"Oh Kim-chi Jar",
		"Song Mu-shu Pork",
	})
}

var enDashProseCases = map[string]string{
	"London–Paris route":                "",
	"the London–Paris route":            "",
	"Flights on the London–Paris route": "",
	"Paris–Berlin train":                "",
	"Boston–Washington corridor":        "",
	"Mason–Dixon line":                  "",
	"Jordan–Pippen era":                 "",
	"Kim–Lee meeting":                   "",
	"the Dunning–Kruger effect":         "",
	"a Kaplan–Meier curve":              "",
	"Hardy–Weinberg equilibrium":        "",
	"the Gauss–Seidel method":           "",
	"the Cauchy–Schwarz inequality":     "",
	"the Black–Scholes model":           "",
	"the Navier–Stokes equations":       "",
	"Bose–Einstein statistics":          "",
	"the Watson–Crick model":            "",
	"Michaelis–Menten kinetics":         "",
	"the Smith–Waterman algorithm":      "",
	"open Monday–Friday":                "",
	"January–March figures":             "",
	"Austria–Hungary empire":            "",
	"Dallas–Fort Worth airport":         "",
	"New York–London flights":           "",
	"the Arsenal–Chelsea match":         "",
	"the Google–Apple deal":             "",
	"the Biden–Trump debate":            "",
	"the Franco–Prussian war":           "Franco",
	"the Smith–Jones report":            "Jones",
	"the Lennard–Jones potential":       "Jones",
	"Tokyo–Osaka trains":                "",
	"the Seoul–Busan line":              "",
	"the Lee–Kim talks":                 "",
	"the Park–Choi summit":              "",
	"the Merkel–Macron meeting":         "",
	"Sino–Korean relations":             "",
	"the Min–Max range":                 "",
	"the Paris–London service":          "",
	"Rome–Milan flights":                "",
	"a 3–2 Arsenal–Spurs win":           "",
	"the Ali–Frazier fight":             "Frazier",
	"Federer–Nadal rivalry":             "",
	"the Lennon–McCartney songs":        "",
	"Rodgers–Hammerstein musicals":      "",
	"the Hewlett–Packard split":         "",
	"Marx–Engels letters":               "Marx",
	"the Molotov–Ribbentrop pact":       "",
	"Sykes–Picot lines":                 "",
	"Lewis–Clark expedition":            "Lewis|Clark",
	"the Taft–Hartley act":              "Taft",
	"Mary–Kate Olsen":                   "Mary–Kate Olsen",
	"the Grace–Hope dispute":            "",
	"the Rose–Lily garden":              "",
	"the Jean–Paul theory":              "Jean|Paul",
}

func TestEnDashPairsReadAsProse(t *testing.T) {
	expectPeople(t, enDashProseCases)
}

func TestKoreanLeaderNames(t *testing.T) {
	for _, h := range []string{"-", "‐", "–"} {
		for _, in := range []string{"Moon Jae-in visited", "Ban Ki-moon visited", "Lee Jae-yong visited", "Yoon Suk-yeol visited", "Kim Jong-un visited", "Han Ji-min visited"} {
			v := strings.ReplaceAll(in, "-", h)
			want := strings.TrimSuffix(v, " visited")
			if got := strings.Join(detectedPeople(t, v), "|"); got != want {
				t.Errorf("%q: got %q, want %q", v, got, want)
			}
		}
	}
	expectNoPeople(t, []string{"Check-in Kim", "Drive-in Park", "In-ki Park", "Ki-in Lee", "Plug-in Han"})
}

func TestEnglishSyllableBeforeFamilyOnlySyllable(t *testing.T) {
	var syl []string
	for s := range givenSyllables {
		syl = append(syl, s)
	}
	for s := range extraGivenSyllables {
		syl = append(syl, s)
	}
	sort.Strings(syl)
	for _, s := range syl {
		if !englishWordSyllable(s) {
			continue
		}
		title := strings.ToUpper(s[:1]) + s[1:]
		for _, b := range []string{"in", "ki"} {
			in := "Park " + title + "-" + b + " deal"
			got := len(detectedPeople(t, in)) > 0
			if want := englishSyllableNames[s]; got != want {
				t.Errorf("%q: masked=%v, want %v", in, got, want)
			}
		}
	}
	for _, fam := range []string{"Park", "Song", "Han", "Moon", "Oh", "Kang", "Jung"} {
		expectNoPeople(t, []string{fam + " Tie-in deal"})
	}
	for _, w := range []string{"Sign", "Check", "Walk", "Plug", "Dine", "Fade", "Opt", "Sit", "Lock", "Built", "Cave", "Log", "Drive", "Break", "Weigh", "Teach", "Pen", "Pin", "Win", "Tin", "Bin", "Kin"} {
		expectNoPeople(t, []string{"Park " + w + "-in deal"})
	}
	expectPeople(t, map[string]string{
		"Moon Jae-in visited":  "Moon Jae-in",
		"Ban Ki-moon visited":  "Ban Ki-moon",
		"Choi Jin-ki sang":     "Choi Jin-ki",
		"Lee In-ho said":       "Lee In-ho",
		"Kim Ki-duk directed":  "Kim Ki-duk",
		"Park Ji-in said":      "Park Ji-in",
		"Park Do-in said":      "Park Do-in",
		"Park Young-ki said":   "Park Young-ki",
		"Park Sang-ki said":    "Park Sang-ki",
		"Son Heung-min scored": "Son Heung-min",
		"Yoon Seok-youl spoke": "Yoon Seok-youl",
	})
}

var particleCompoundWords = []string{"Tie-in", "Sign-in", "Log-in", "Plug-in", "Walk-in", "Drop-in", "Check-in", "Add-in", "Built-in", "Lock-in", "Sit-in", "Break-in", "Run-in", "Fade-in", "Write-in", "Hand-in", "Pin-up", "Sign-on", "Pop-up", "Set-up", "Go-to"}

func TestHyphenParticleCompoundIsNotSurname(t *testing.T) {
	for _, fam := range []string{"Kim", "Lee"} {
		for _, w := range particleCompoundWords {
			expectNoPeople(t, []string{fam + " " + w + " deal", "the " + fam + " " + w + " page"})
		}
		for _, w := range []string{"Tie-in", "Log-in", "Pop-up", "Go-to"} {
			expectNoPeople(t, []string{fam + " " + strings.ReplaceAll(w, "-", "–") + " deal", fam + " " + strings.ReplaceAll(w, "-", "‐") + " deal"})
		}
	}
	expectNoPeople(t, []string{
		"Kim Log-in page",
		"Lee Built-in feature",
		"Mark Sign-in page",
		"Will Check-in desk",
		"Grace Add-in menu",
		"Art Pop-up shop",
		"Kim Tie-in said",
	})
	expectPeople(t, map[string]string{
		"Kim Smith-Jones said": "Kim Smith-Jones",
		"Lee Day-Lewis acted":  "Lee Day-Lewis",
		"Kim Min-jun said":     "Kim Min-jun",
		"Lee Seo-yeon said":    "Lee Seo-yeon",
		"Mary Bowes-lyon said": "Mary Bowes-lyon",
		"Kim Ki-duk directed":  "Kim Ki-duk",
		"Moon Jae-in visited":  "Moon Jae-in",
	})
}

var koreanGridSurnames = []string{"Kim", "Lee", "Park", "Choi", "Jung", "Kang", "Cho", "Yoon", "Jang", "Lim", "Han", "Shin", "Song"}

func TestKoreanSurnameGivenGrid(t *testing.T) {
	given := []string{"Yu-na", "Ha-na", "Se-na", "Da-na", "Min-a", "Ha-ra", "Bo-ra", "Ah-in", "Ye-in", "Eun-a", "Mi-na", "Bo-gum", "Jae-in", "Ji-a", "Ji-won", "Min-jun"}
	for _, f := range koreanGridSurnames {
		for _, g := range given {
			want := f + " " + g
			for _, in := range []string{want, want + " said"} {
				if got := strings.Join(detectedPeople(t, in), "|"); got != want {
					t.Errorf("%q: got %q, want %q", in, got, want)
				}
			}
		}
	}
	for _, h := range []string{"–", "‐"} {
		for _, in := range []string{"Kim Yu-na said", "Park Bo-gum said", "Yoo Ah-in said", "Shin Min-a said", "Choi Ha-na said"} {
			v := strings.ReplaceAll(in, "-", h)
			if got := strings.Join(detectedPeople(t, v), "|"); got != strings.TrimSuffix(v, " said") {
				t.Errorf("%q: got %q", v, got)
			}
		}
	}
	expectPeople(t, map[string]string{
		"Min Min-in said": "Min Min-in",
		"Park Ha-in said": "Park Ha-in",
	})
}

func TestEnglishHyphenWordsAfterKoreanSurnames(t *testing.T) {
	words := append(append([]string{}, particleCompoundWords...), "Go-to", "Do-on", "Add-on", "Sun-in", "Mi-so", "Ha-ha", "Yo-yo", "So-so", "No-go", "Bo-ho", "Pin-up", "Run-in", "Sit-in", "Do-over")
	for _, f := range koreanGridSurnames {
		for _, w := range words {
			expectNoPeople(t, []string{f + " " + w + " deal", f + " " + w + " said"})
		}
	}
	expectNoPeople(t, []string{"Yo said hi", "Kim-chi Jar", "Mark Sign-in page", "Grace Add-in menu"})
}

var dashEquivalents = []string{"‐", "‑", "−", "﹣", "－", "‒", "–"}

var dashedNameCases = []string{
	"Please email Lily-Rose Depp today.", "Mary-Kate Olsen said hi", "Jean-Luc Picard said hi", "Anne-Marie Slaughter said hi",
	"Jean-Paul Sartre wrote it", "Ana-Maria Lopez said hi", "Please email Mary-Jane Watson today.", "Marie-Claire Dubois said hi",
	"Jean-Pierre Leclerc said hi", "Anne-Sophie Mutter played", "Sarah-Jane Smith said hi", "Mary-Ann Evans wrote it",
	"Jo-Ann Fletcher said hi", "Lee-Ann Womack sang", "Mary-Beth Hurt said hi", "Rose-Marie Dupont said hi",
	"Marie-Louise Hansen said hi", "Karl-Heinz Rummenigge said hi", "Hans-Peter Feldmann said hi", "Eva-Maria Hagen said hi",
	"Maria-Elena Gomez said hi", "Jean-Marc Vallee said hi", "Paul-Henri Spaak said hi", "Louis-Philippe Martin said hi",
	"Pierre-Yves Morel said hi", "Anne-Laure Bernard said hi", "Jean-Claude Juncker said hi", "Ask Lily-Rose Depp today",
	"Seo-yeon Park signed", "Hye-jin Choi signed", "Min-jun Kim signed", "Ji-woo Lee signed", "Park Ji-sung scored",
	"Moon Jae-in visited", "Yoo Ah-in said", "Kim Yu-na said", "Lee Ha-na said", "Ban Ki-moon visited", "Son Heung-min scored",
	"Please email Jean-Luc Picard today.",
	"Dear Lily-Rose Depp,", "Hi Jean-Luc Picard,", "Dear Ana-Maria Lopez Garcia,", "Hello Mary-Kate Olsen!",
	"Jean-Claude Juncker said hi", "Jean-Claude Van Damme said hi", "Mary-Ruby Smith said hi", "Anne-Java Jones said hi",
}

var dashedPairCases = []string{
	"the London-Paris route", "Paris-Berlin train", "Boston-Washington corridor", "Mason-Dixon line", "Jordan-Pippen era",
	"Kim-Lee meeting", "the Dunning-Kruger effect", "a Kaplan-Meier curve", "Hardy-Weinberg equilibrium", "the Gauss-Seidel method",
	"the Cauchy-Schwarz inequality", "the Black-Scholes model", "the Navier-Stokes equations", "Bose-Einstein statistics",
	"the Watson-Crick model", "Michaelis-Menten kinetics", "the Smith-Waterman algorithm", "open Monday-Friday", "January-March figures",
	"Austria-Hungary empire", "Dallas-Fort Worth airport", "the Arsenal-Chelsea match", "the Google-Apple deal", "the Biden-Trump debate",
	"the Franco-Prussian war", "the Smith-Jones report", "the Lennard-Jones potential", "Tokyo-Osaka trains", "the Seoul-Busan line",
	"the Lee-Kim talks", "the Park-Choi summit", "the Merkel-Macron meeting", "Sino-Korean relations", "the Min-Max range",
	"the Ali-Frazier fight", "Marx-Engels letters", "Lewis-Clark expedition", "the Taft-Hartley act", "Austin-Dallas Smith route",
	"Bob-Alice protocol", "Kim-chi Jar", "Park Tie-in deal", "the Jean-Paul theory",
}

var rangeDashProseReadings = map[string]bool{
	"the Franco-Prussian war": true, "the Smith-Jones report": true,
	"the Lennard-Jones potential": true, "the Ali-Frazier fight": true, "Marx-Engels letters": true,
	"Lewis-Clark expedition": true, "the Taft-Hartley act": true, "the Jean-Paul theory": true,
	"Austin-Dallas Smith route": true, "Bob-Alice protocol": true,
}

func TestDashFormsMaskLikeASCIIHyphen(t *testing.T) {
	for _, in := range append(append([]string{}, dashedNameCases...), dashedPairCases...) {
		want := strings.Join(detectedPeople(t, in), "|")
		for _, h := range dashEquivalents {
			r := []rune(h)[0]
			if textutil.IsRangeOrNameDash(r) && rangeDashProseReadings[in] {
				continue
			}
			v := strings.ReplaceAll(in, "-", h)
			if got := strings.ReplaceAll(strings.Join(detectedPeople(t, v), "|"), h, "-"); got != want {
				t.Errorf("%q with %U: got %q, ASCII gives %q", in, r, got, want)
			}
		}
	}
	for _, in := range dashedNameCases {
		if len(detectedPeople(t, in)) == 0 {
			t.Errorf("%q: name case not masked", in)
		}
	}
	expectPeople(t, map[string]string{
		"Please email Lily–Rose Depp today.": "Lily–Rose Depp",
		"Please email Lily‒Rose Depp today.": "Lily‒Rose Depp",
		"Dear Mary–Kate Olsen, hi":           "Mary–Kate Olsen",
		"Austin–Dallas Smith route":          "Dallas Smith",
		"Bob–Alice protocol":                 "Alice",
	})
}

var greetingCases = map[string]string{
	"Dear Lily-Rose Depp,":         "Lily-Rose Depp",
	"Dear Lily\u2013Rose Depp,":    "Lily\u2013Rose Depp",
	"Dear Mary Kate Olsen,":        "Mary Kate Olsen",
	"Hi Jean-Luc Picard,":          "Jean-Luc Picard",
	"Dear Ana-Maria Lopez Garcia,": "Ana-Maria Lopez Garcia",
	"Dear Lee Walk-in,":            "Lee",
	"Dear Kim Tie-in,":             "Kim",
	"Hi Kim Sign-in,":              "Kim",
	"Dear Mark Sign-in,":           "Mark",
	"Jean-Claude Juncker said":     "Jean-Claude Juncker",
	"Jean-Claude Van Damme said":   "Jean-Claude Van Damme",
	"Mary-Ruby Smith said":         "Mary-Ruby Smith",
	"Anne-Java Jones said":         "Anne-Java Jones",
}

var techAndGreetingKeeps = []string{"Claude", "Claude Code", "Claude said the API", "Use Claude Code today", "Ask Claude", "Dear Claude,", "Dear Claude!", "Hi Claude,", "Hello Claude, can you help?", "Hey Claude,", "Thanks Claude!", "Thank you Claude.", "Good morning Claude,", "Ok. Hi Claude, one more", "Dear Park Walk-in,"}

func TestGreetingNamesAndTechHalves(t *testing.T) {
	expectPeople(t, greetingCases)
	expectNoPeople(t, techAndGreetingKeeps)
	for in, want := range map[string]bool{"Lee Walk-in": false, "Kim Tie-in": false, "Lily-Rose Depp": true, "Jean-Luc Picard": true} {
		if got := greetingNameOK(in, 0, len(in)); got != want {
			t.Errorf("greetingNameOK(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestCapitalizedNameAfterSkipsEnglishCompounds(t *testing.T) {
	for in, want := range map[string]string{"Lee Walk-in,": "", "Kim Tie-in today": "", "Lee Smith-Jones said": " Smith-Jones", "Lee Walker said": " Walker"} {
		end := CapitalizedNameAfter(in, 3)
		if got := in[3:end]; got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}

type EndToEndRow struct {
	Table, Text string
	Want        []string
}

func EndToEndRows() []EndToEndRow {
	var rows []EndToEndRow
	add := func(table string, cases map[string]string) {
		for in, want := range cases {
			var spans []string
			if want != "" {
				spans = strings.Split(want, "|")
			}
			rows = append(rows, EndToEndRow{table, in, spans})
		}
	}
	add("greeting", greetingCases)
	add("greeting targets", greetingTargetCases)
	add("en dash prose", enDashProseCases)
	keeps := map[string]string{}
	for _, in := range techAndGreetingKeeps {
		keeps[in] = ""
	}
	add("tech term", keeps)
	compound := map[string]string{}
	for _, fam := range []string{"Kim", "Lee"} {
		for _, w := range particleCompoundWords {
			compound[fam+" "+w+" deal"] = ""
		}
	}
	add("hyphen compound", compound)
	return rows
}

var greetingTargetCases = map[string]string{
	"Hi Claude Code, fix this":      "",
	"Hey Claude Code, run tests":    "",
	"Dear Claude Code,":             "",
	"Hi Claude Sonnet, review":      "",
	"Hi Claude Opus,":               "",
	"Hello Claude Desktop,":         "",
	"Hi GitHub Copilot, help":       "",
	"Hi Google Gemini,":             "",
	"Dear Visual Studio,":           "",
	"Dear Microsoft Teams,":         "",
	"Hi Docker Desktop,":            "",
	"Dear Apple Support,":           "",
	"Hello Cursor Agent,":           "",
	"Hi Siri Shortcuts,":            "",
	"Hi Amazon Q,":                  "",
	"Hi Claude Haiku,":              "",
	"Dear Claude Team,":             "",
	"Hi Team Lead,":                 "",
	"Dear Hiring Manager,":          "",
	"Dear Customer Support,":        "",
	"Hello World,":                  "",
	"Hi Stack Overflow,":            "",
	"Hi All,":                       "",
	"Dear Team,":                    "",
	"Hello Everyone,":               "",
	"Hi Folks,":                     "",
	"Hello There,":                  "",
	"Hello World!":                  "",
	"Hi Support Team,":              "",
	"Hi Dev Team,":                  "",
	"Dear Valued Customer,":         "",
	"Dear Sir or Madam,":            "",
	"Hi everyone,":                  "",
	"Hey guys,":                     "",
	"Dear all,":                     "",
	"Dear Claude Monet,":            "Claude Monet",
	"Hi Claude Shannon,":            "Claude Shannon",
	"Dear Ruby Rose,":               "Ruby Rose",
	"Hi Ada Lovelace,":              "Ada Lovelace",
	"Dear Mary Smith,":              "Mary Smith",
	"Hi Mark Sign-in,":              "Mark",
	"Dear Jean-Claude,":             "Jean-Claude",
	"Dear Lily-Rose Depp,":          "Lily-Rose Depp",
	"Hi Grace Hopper,":              "Grace Hopper",
	"Dear Zofia Oyelaran,":          "Zofia Oyelaran",
	"Hi Thandiwe Nkosi,":            "Thandiwe Nkosi",
	"Hi Sterling Archer,":           "Sterling Archer",
	"Dear Hunter Biden,":            "Hunter Biden",
	"Hi Code Smith,":                "Code Smith",
	"Dear Apple Martin,":            "Apple Martin",
	"Hi Siri Patel,":                "Siri Patel",
	"Dear Paris Hilton,":            "Paris Hilton",
	"Hi Brooklyn Beckham,":          "Brooklyn Beckham",
	"Dear Mr. Okafor,":              "Okafor",
	"Hi Dr. Quetzal,":               "Quetzal",
	"Hi Wei Zhang,":                 "Wei Zhang",
	"Dear Tran Thi Mai,":            "Tran Thi Mai",
	"Hi Maria,":                     "Maria",
	"Dear Thandiwe,":                "Thandiwe",
	"Hi Will,":                      "Will",
	"Dear Grace,":                   "Grace",
	"Hi Hunter,":                    "Hunter",
	"Hi Taylor Swift,":              "Taylor Swift",
	"Dear Taylor Swift,":            "Taylor Swift",
	"Hello Taylor Swift!":           "Taylor Swift",
	"Hi Jordan Swift,":              "Jordan Swift",
	"Hi Jasper Swift,":              "Jasper Swift",
	"Hi Dev Patel,":                 "Dev Patel",
	"Hi Dev Kapoor,":                "Dev Kapoor",
	"Hi Dev Sharma,":                "Dev Sharma",
	"Hi Al,":                        "Al",
	"Hi Ty,":                        "Ty",
	"Hi Li,":                        "Li",
	"Dear Mo,":                      "Mo",
	"Hello Jade Dragonova,":         "Jade Dragonova",
	"Dear Amazon Lee,":              "Amazon Lee",
	"Hi Apple Martin,":              "Apple Martin",
	"Dear Rose Code,":               "Rose Code",
	"Hi Ruby,":                      "Ruby",
	"Hi Ada,":                       "Ada",
	"Hi Julia,":                     "Julia",
	"Hi Crystal,":                   "Crystal",
	"Hi Pascal,":                    "Pascal",
	"Hi Swift,":                     "Swift",
	"Hi Rust,":                      "Rust",
	"Hi Team Rocket,":               "Team Rocket",
	"Dear Grace Team,":              "Grace Team",
	"Hi Taylor Code,":               "Taylor Code",
	"Taylor Swift said hi":          "Taylor Swift",
	"Dev Patel said hi":             "Patel",
	"Ruby on Rails":                 "",
	"Swift code":                    "",
	"learn Ada or Pascal":           "",
	"Julia package":                 "",
	"Taylor series":                 "",
	"Jordan curve":                  "",
	"Should I learn Ada or Pascal?": "",
	"Hi Ruby, Python or Go?":        "Ruby",
	"Hi Taylor Ruby,":               "Taylor Ruby",
	"Hi Taylor Rust,":               "Taylor Rust",
	"Hi Taylor Java,":               "Taylor Java",
	"Hi Taylor Python,":             "Taylor Python",
	"Hi Taylor Perl,":               "Taylor Perl",
	"Hi Taylor Kotlin,":             "Taylor Kotlin",
	"Hi Taylor Scala,":              "Taylor Scala",
	"Hi Taylor Dart,":               "Taylor Dart",
	"Hi Taylor Julia,":              "Taylor Julia",
	"Hi Taylor Crystal,":            "Taylor Crystal",
	"Hi Taylor Elixir,":             "Taylor Elixir",
	"Hi Taylor Haskell,":            "Taylor Haskell",
	"Hi Taylor Lisp,":               "Taylor Lisp",
	"Hi Taylor Pascal,":             "Taylor Pascal",
	"Hi Taylor Ada,":                "Taylor Ada",
	"Hi Taylor Docker,":             "Taylor Docker",
	"Hi Taylor Shell,":              "Taylor Shell",
	"Hi Taylor Bash,":               "Taylor Bash",
	"Hi Taylor Node,":               "Taylor Node",
	"Hi Taylor React,":              "Taylor React",
	"Hi Taylor Angular,":            "Taylor Angular",
	"Hi Taylor Vue,":                "Taylor Vue",
	"Hi Taylor Spring,":             "Taylor Spring",
	"Hi Taylor Rails,":              "Taylor Rails",
	"Hi Taylor Flask,":              "Taylor Flask",
	"Hi Taylor Django,":             "Taylor Django",
	"Hi Taylor Lambda,":             "Taylor Lambda",
	"Hi Jordan Ruby,":               "Jordan Ruby",
	"Hi Jordan Rust,":               "Jordan Rust",
	"Hi Jordan Java,":               "Jordan Java",
	"Hi Jordan Python,":             "Jordan Python",
	"Hi Jordan Perl,":               "Jordan Perl",
	"Hi Jordan Kotlin,":             "Jordan Kotlin",
	"Hi Jordan Scala,":              "Jordan Scala",
	"Hi Jordan Dart,":               "Jordan Dart",
	"Hi Jordan Julia,":              "Jordan Julia",
	"Hi Jordan Crystal,":            "Jordan Crystal",
	"Hi Jordan Elixir,":             "Jordan Elixir",
	"Hi Jordan Haskell,":            "Jordan Haskell",
	"Hi Jordan Lisp,":               "Jordan Lisp",
	"Hi Jordan Pascal,":             "Jordan Pascal",
	"Hi Jordan Ada,":                "Jordan Ada",
	"Hi Jordan Code,":               "Jordan Code",
	"Hi Jordan Docker,":             "Jordan Docker",
	"Hi Jordan Shell,":              "Jordan Shell",
	"Hi Jordan Bash,":               "Jordan Bash",
	"Hi Jordan Node,":               "Jordan Node",
	"Hi Jordan React,":              "Jordan React",
	"Hi Jordan Angular,":            "Jordan Angular",
	"Hi Jordan Vue,":                "Jordan Vue",
	"Hi Jordan Spring,":             "Jordan Spring",
	"Hi Jordan Rails,":              "Jordan Rails",
	"Hi Jordan Flask,":              "Jordan Flask",
	"Hi Jordan Django,":             "Jordan Django",
	"Hi Jordan Lambda,":             "Jordan Lambda",
	"Hi Sam Swift,":                 "Sam Swift",
	"Hi Sam Ruby,":                  "Sam Ruby",
	"Hi Sam Rust,":                  "Sam Rust",
	"Hi Sam Java,":                  "Sam Java",
	"Hi Sam Python,":                "Sam Python",
	"Hi Sam Perl,":                  "Sam Perl",
	"Hi Sam Kotlin,":                "Sam Kotlin",
	"Hi Sam Scala,":                 "Sam Scala",
	"Hi Sam Dart,":                  "Sam Dart",
	"Hi Sam Julia,":                 "Sam Julia",
	"Hi Sam Crystal,":               "Sam Crystal",
	"Hi Sam Elixir,":                "Sam Elixir",
	"Hi Sam Haskell,":               "Sam Haskell",
	"Hi Sam Lisp,":                  "Sam Lisp",
	"Hi Sam Pascal,":                "Sam Pascal",
	"Hi Sam Ada,":                   "Sam Ada",
	"Hi Sam Code,":                  "Sam Code",
	"Hi Sam Docker,":                "Sam Docker",
	"Hi Sam Shell,":                 "Sam Shell",
	"Hi Sam Bash,":                  "Sam Bash",
	"Hi Sam Node,":                  "Sam Node",
	"Hi Sam React,":                 "Sam React",
	"Hi Sam Angular,":               "Sam Angular",
	"Hi Sam Vue,":                   "Sam Vue",
	"Hi Sam Spring,":                "Sam Spring",
	"Hi Sam Rails,":                 "Sam Rails",
	"Hi Sam Flask,":                 "Sam Flask",
	"Hi Sam Django,":                "Sam Django",
	"Hi Sam Lambda,":                "Sam Lambda",
	"Hi Grace Swift,":               "Grace Swift",
	"Hi Grace Ruby,":                "Grace Ruby",
	"Hi Grace Rust,":                "Grace Rust",
	"Hi Grace Java,":                "Grace Java",
	"Hi Grace Python,":              "Grace Python",
	"Hi Grace Perl,":                "Grace Perl",
	"Hi Grace Kotlin,":              "Grace Kotlin",
	"Hi Grace Scala,":               "Grace Scala",
	"Hi Grace Dart,":                "Grace Dart",
	"Hi Grace Julia,":               "Grace Julia",
	"Hi Grace Crystal,":             "Grace Crystal",
	"Hi Grace Elixir,":              "Grace Elixir",
	"Hi Grace Haskell,":             "Grace Haskell",
	"Hi Grace Lisp,":                "Grace Lisp",
	"Hi Grace Pascal,":              "Grace Pascal",
	"Hi Grace Ada,":                 "Grace Ada",
	"Hi Grace Code,":                "Grace Code",
	"Hi Grace Docker,":              "Grace Docker",
	"Hi Grace Shell,":               "Grace Shell",
	"Hi Grace Bash,":                "Grace Bash",
	"Hi Grace Node,":                "Grace Node",
	"Hi Grace React,":               "Grace React",
	"Hi Grace Angular,":             "Grace Angular",
	"Hi Grace Vue,":                 "Grace Vue",
	"Hi Grace Spring,":              "Grace Spring",
	"Hi Grace Rails,":               "Grace Rails",
	"Hi Grace Flask,":               "Grace Flask",
	"Hi Grace Django,":              "Grace Django",
	"Hi Grace Lambda,":              "Grace Lambda",
	"Hi Tom Swift,":                 "Tom Swift",
	"Hi Tom Ruby,":                  "Tom Ruby",
	"Hi Tom Rust,":                  "Tom Rust",
	"Hi Tom Java,":                  "Tom Java",
	"Hi Tom Python,":                "Tom Python",
	"Hi Tom Perl,":                  "Tom Perl",
	"Hi Tom Kotlin,":                "Tom Kotlin",
	"Hi Tom Scala,":                 "Tom Scala",
	"Hi Tom Dart,":                  "Tom Dart",
	"Hi Tom Julia,":                 "Tom Julia",
	"Hi Tom Crystal,":               "Tom Crystal",
	"Hi Tom Elixir,":                "Tom Elixir",
	"Hi Tom Haskell,":               "Tom Haskell",
	"Hi Tom Lisp,":                  "Tom Lisp",
	"Hi Tom Pascal,":                "Tom Pascal",
	"Hi Tom Ada,":                   "Tom Ada",
	"Hi Tom Code,":                  "Tom Code",
	"Hi Tom Docker,":                "Tom Docker",
	"Hi Tom Shell,":                 "Tom Shell",
	"Hi Tom Bash,":                  "Tom Bash",
	"Hi Tom Node,":                  "Tom Node",
	"Hi Tom React,":                 "Tom React",
	"Hi Tom Angular,":               "Tom Angular",
	"Hi Tom Vue,":                   "Tom Vue",
	"Hi Tom Spring,":                "Tom Spring",
	"Hi Tom Rails,":                 "Tom Rails",
	"Hi Tom Flask,":                 "Tom Flask",
	"Hi Tom Django,":                "Tom Django",
	"Hi Tom Lambda,":                "Tom Lambda",
	"Hi Lucy Swift,":                "Lucy Swift",
	"Hi Lucy Ruby,":                 "Lucy Ruby",
	"Hi Lucy Rust,":                 "Lucy Rust",
	"Hi Lucy Java,":                 "Lucy Java",
	"Hi Lucy Python,":               "Lucy Python",
	"Hi Lucy Perl,":                 "Lucy Perl",
	"Hi Lucy Kotlin,":               "Lucy Kotlin",
	"Hi Lucy Scala,":                "Lucy Scala",
	"Hi Lucy Dart,":                 "Lucy Dart",
	"Hi Lucy Julia,":                "Lucy Julia",
	"Hi Lucy Crystal,":              "Lucy Crystal",
	"Hi Lucy Elixir,":               "Lucy Elixir",
	"Hi Lucy Haskell,":              "Lucy Haskell",
	"Hi Lucy Lisp,":                 "Lucy Lisp",
	"Hi Lucy Pascal,":               "Lucy Pascal",
	"Hi Lucy Ada,":                  "Lucy Ada",
	"Hi Lucy Code,":                 "Lucy Code",
	"Hi Lucy Docker,":               "Lucy Docker",
	"Hi Lucy Shell,":                "Lucy Shell",
	"Hi Lucy Bash,":                 "Lucy Bash",
	"Hi Lucy Node,":                 "Lucy Node",
	"Hi Lucy React,":                "Lucy React",
	"Hi Lucy Angular,":              "Lucy Angular",
	"Hi Lucy Vue,":                  "Lucy Vue",
	"Hi Lucy Spring,":               "Lucy Spring",
	"Hi Lucy Rails,":                "Lucy Rails",
	"Hi Lucy Flask,":                "Lucy Flask",
	"Hi Lucy Django,":               "Lucy Django",
	"Hi Lucy Lambda,":               "Lucy Lambda",
	"Hi Anna Swift,":                "Anna Swift",
	"Hi Anna Ruby,":                 "Anna Ruby",
	"Hi Anna Rust,":                 "Anna Rust",
	"Hi Anna Java,":                 "Anna Java",
	"Hi Anna Python,":               "Anna Python",
	"Hi Anna Perl,":                 "Anna Perl",
	"Hi Anna Kotlin,":               "Anna Kotlin",
	"Hi Anna Scala,":                "Anna Scala",
	"Hi Anna Dart,":                 "Anna Dart",
	"Hi Anna Julia,":                "Anna Julia",
	"Hi Anna Crystal,":              "Anna Crystal",
	"Hi Anna Elixir,":               "Anna Elixir",
	"Hi Anna Haskell,":              "Anna Haskell",
	"Hi Anna Lisp,":                 "Anna Lisp",
	"Hi Anna Pascal,":               "Anna Pascal",
	"Hi Anna Ada,":                  "Anna Ada",
	"Hi Anna Code,":                 "Anna Code",
	"Hi Anna Docker,":               "Anna Docker",
	"Hi Anna Shell,":                "Anna Shell",
	"Hi Anna Bash,":                 "Anna Bash",
	"Hi Anna Node,":                 "Anna Node",
	"Hi Anna React,":                "Anna React",
	"Hi Anna Angular,":              "Anna Angular",
	"Hi Anna Vue,":                  "Anna Vue",
	"Hi Anna Spring,":               "Anna Spring",
	"Hi Anna Rails,":                "Anna Rails",
	"Hi Anna Flask,":                "Anna Flask",
	"Hi Anna Django,":               "Anna Django",
	"Hi Anna Lambda,":               "Anna Lambda",
	"Hi Dev Banks,":                 "Dev Banks",
	"Hi Dev Lane,":                  "Dev Lane",
	"Hi Dev Cloud,":                 "Dev Cloud",
	"Hi Dev Swift,":                 "Dev Swift",
	"Hi Dev Ruby,":                  "Dev Ruby",
	"Hi Dev Rust,":                  "Dev Rust",
	"Hi Dev Java,":                  "Dev Java",
	"Hi Dev Python,":                "Dev Python",
	"Hi Dev Perl,":                  "Dev Perl",
	"Hi Dev Kotlin,":                "Dev Kotlin",
	"Hi Dev Scala,":                 "Dev Scala",
	"Hi Dev Dart,":                  "Dev Dart",
	"Hi Dev Julia,":                 "Dev Julia",
	"Hi Dev Crystal,":               "Dev Crystal",
	"Hi Dev Elixir,":                "Dev Elixir",
	"Hi Dev Haskell,":               "Dev Haskell",
	"Hi Dev Lisp,":                  "Dev Lisp",
	"Hi Dev Pascal,":                "Dev Pascal",
	"Hi Dev Ada,":                   "Dev Ada",
	"Hi Dev Code,":                  "Dev Code",
	"Hi Dev Docker,":                "Dev Docker",
	"Hi Dev Shell,":                 "Dev Shell",
	"Hi Dev Bash,":                  "Dev Bash",
	"Hi Dev Node,":                  "Dev Node",
	"Hi Dev React,":                 "Dev React",
	"Hi Dev Angular,":               "Dev Angular",
	"Hi Dev Vue,":                   "Dev Vue",
	"Hi Dev Spring,":                "Dev Spring",
	"Hi Dev Rails,":                 "Dev Rails",
	"Hi Dev Flask,":                 "Dev Flask",
	"Hi Dev Django,":                "Dev Django",
	"Hi Dev Lambda,":                "Dev Lambda",
	"Dear Dev Banks,":               "Dev Banks",
	"Dear Dev Lane,":                "Dev Lane",
	"Dear Dev Cloud,":               "Dev Cloud",
	"Dear Dev Swift,":               "Dev Swift",
	"Dear Dev Ruby,":                "Dev Ruby",
	"Dear Dev Rust,":                "Dev Rust",
	"Dear Dev Java,":                "Dev Java",
	"Dear Dev Python,":              "Dev Python",
	"Dear Dev Perl,":                "Dev Perl",
	"Dear Dev Kotlin,":              "Dev Kotlin",
	"Dear Dev Scala,":               "Dev Scala",
	"Dear Dev Dart,":                "Dev Dart",
	"Dear Dev Julia,":               "Dev Julia",
	"Dear Dev Crystal,":             "Dev Crystal",
	"Dear Dev Elixir,":              "Dev Elixir",
	"Dear Dev Haskell,":             "Dev Haskell",
	"Dear Dev Lisp,":                "Dev Lisp",
	"Dear Dev Pascal,":              "Dev Pascal",
	"Dear Dev Ada,":                 "Dev Ada",
	"Dear Dev Code,":                "Dev Code",
	"Dear Dev Docker,":              "Dev Docker",
	"Dear Dev Shell,":               "Dev Shell",
	"Dear Dev Bash,":                "Dev Bash",
	"Dear Dev Node,":                "Dev Node",
	"Dear Dev React,":               "Dev React",
	"Dear Dev Angular,":             "Dev Angular",
	"Dear Dev Vue,":                 "Dev Vue",
	"Dear Dev Spring,":              "Dev Spring",
	"Dear Dev Rails,":               "Dev Rails",
	"Dear Dev Flask,":               "Dev Flask",
	"Dear Dev Django,":              "Dev Django",
	"Dear Dev Lambda,":              "Dev Lambda",
	"Hi Sales Banks,":               "Sales Banks",
	"Hi Team Banks,":                "Team Banks",
	"Hi Admin Lane,":                "Admin Lane",
	"Hi Ops Lane,":                  "Ops Lane",
	"Hi ChatGPT Plus,":              "",
	"Hi Meta AI,":                   "",
	"Hi Google Bard,":               "Google Bard",
	"Hi Code Monkey,":               "Code Monkey",
	"Hi Bug Hunter,":                "Bug Hunter",
	"Hi Data Lane,":                 "Data Lane",
	"Hi Web Banks,":                 "Web Banks",
	"Hi Wei Ai,":                    "Wei Ai",
	"Dear Li AI,":                   "Li AI",
	"Hi DevOps Team,":               "",
	"Hi Platform Team,":             "",
	"Hi Data Team,":                 "",
	"Hi Design Team,":               "",
	"Hi Product Team,":              "",
	"Hi QA Team,":                   "",
	"Hi Infra Team,":                "",
	"Hi Frontend Team,":             "",
	"Hi Backend Team,":              "",
	"Hi Growth Team,":               "",
	"Hi Marketing Team,":            "",
	"Hi Engineering Team,":          "",
	"Hi Finance Team,":              "",
	"Hi Security Team,":             "",
	"Hi Mobile Team,":               "",
	"Hi Web Team,":                  "",
	"Hi ML Team,":                   "",
	"Hi AI Team,":                   "",
	"Hi Research Team,":             "",
	"Hi Legal Team,":                "",
	"Hi HR Team,":                   "",
	"Hi IT Team,":                   "",
	"Hi UX Team,":                   "",
	"Hi UI Team,":                   "",
	"Hi SRE Team,":                  "",
	"Hi Banks Team,":                "Banks Team",
	"Hi Mobile Teams,":              "",
	"Hi Sales Team,":                "",
	"Hi Sales Leads,":               "",
	"Dear Sales Managers,":          "",
	"Hi Sales Ops,":                 "",
	"Dear Sales Department,":        "",
	"Hi Press Team,":                "",
	"Dear Press Office,":            "",
	"Hi Kitchen Staff,":             "",
	"Hi Store Manager,":             "",
	"Dear Property Manager,":        "",
	"Dear Case Worker,":             "",
	"Hi Night Nurse,":               "",
	"Dear Head Chef,":               "",
	"Hi Case Smith,":                "Case Smith",
	"Hi Head Smith,":                "Head Smith",
	"Dear Store Banks,":             "Store Banks",
	"Hi Night Owl,":                 "Night Owl",
	"Hi Kwabena Team,":              "",
	"Hi Field Manager,":             "Field Manager",
	"Hi Hunter Team,":               "Hunter Team",
	"Hi Mason Lead,":                "Mason Lead",
	"Hi Price Manager,":             "Price Manager",
	"Hi Hall Manager,":              "Hall Manager",
	"Hi Park Team,":                 "Park Team",
	"Hi Adam,":                      "Adam",
	"Dear Adam,":                    "Adam",
	"I met Adam yesterday at the conference.":                "Adam",
	"Please email Adam today about the contract.":            "Adam",
	"Adam said the build failed again.":                      "Adam",
	"Hi Adam Smith,":                                         "Adam Smith",
	"Adam Smith said hi":                                     "Adam Smith",
	"Hi Travis,":                                             "Travis",
	"Dear Travis,":                                           "Travis",
	"I met Travis yesterday at the conference.":              "Travis",
	"Please email Travis today about the contract.":          "Travis",
	"Travis said the build failed again.":                    "Travis",
	"Hi Travis Smith,":                                       "Travis Smith",
	"Travis Smith said hi":                                   "Travis Smith",
	"Hi Hugo,":                                               "Hugo",
	"Dear Hugo,":                                             "Hugo",
	"I met Hugo yesterday at the conference.":                "Hugo",
	"Please email Hugo today about the contract.":            "Hugo",
	"Hugo said the build failed again.":                      "Hugo",
	"Hi Hugo Smith,":                                         "Hugo Smith",
	"Hugo Smith said hi":                                     "Hugo Smith",
	"Hi Jasmine,":                                            "Jasmine",
	"Dear Jasmine,":                                          "Jasmine",
	"I met Jasmine yesterday at the conference.":             "Jasmine",
	"Please email Jasmine today about the contract.":         "Jasmine",
	"Jasmine said the build failed again.":                   "Jasmine",
	"Hi Jasmine Smith,":                                      "Jasmine Smith",
	"Jasmine Smith said hi":                                  "Jasmine Smith",
	"Hi Jade,":                                               "Jade",
	"Dear Jade,":                                             "Jade",
	"I met Jade yesterday at the conference.":                "Jade",
	"Please email Jade today about the contract.":            "Jade",
	"Jade said the build failed again.":                      "Jade",
	"Hi Jade Smith,":                                         "Jade Smith",
	"Jade Smith said hi":                                     "Jade Smith",
	"Dear Crystal,":                                          "Crystal",
	"I met Crystal yesterday at the conference.":             "Crystal",
	"Please email Crystal today about the contract.":         "Crystal",
	"Crystal said the build failed again.":                   "Crystal",
	"Hi Crystal Smith,":                                      "Crystal Smith",
	"Crystal Smith said hi":                                  "Crystal Smith",
	"Dear Pascal,":                                           "Pascal",
	"I met Pascal yesterday at the conference.":              "Pascal",
	"Please email Pascal today about the contract.":          "Pascal",
	"Pascal said the build failed again.":                    "Pascal",
	"Hi Pascal Smith,":                                       "Pascal Smith",
	"Pascal Smith said hi":                                   "Pascal Smith",
	"Hi Homer,":                                              "Homer",
	"Dear Homer,":                                            "Homer",
	"I met Homer yesterday at the conference.":               "Homer",
	"Please email Homer today about the contract.":           "Homer",
	"Homer said the build failed again.":                     "Homer",
	"Hi Homer Smith,":                                        "Homer Smith",
	"Homer Smith said hi":                                    "Homer Smith",
	"Hi Ernie,":                                              "Ernie",
	"Dear Ernie,":                                            "Ernie",
	"I met Ernie yesterday at the conference.":               "Ernie",
	"Please email Ernie today about the contract.":           "Ernie",
	"Ernie said the build failed again.":                     "Ernie",
	"Hi Ernie Smith,":                                        "Ernie Smith",
	"Ernie Smith said hi":                                    "Ernie Smith",
	"Hi Bert,":                                               "Bert",
	"Dear Bert,":                                             "Bert",
	"I met Bert yesterday at the conference.":                "Bert",
	"Please email Bert today about the contract.":            "Bert",
	"Bert said the build failed again.":                      "Bert",
	"Hi Bert Smith,":                                         "Bert Smith",
	"Bert Smith said hi":                                     "Bert Smith",
	"Hi Elmo,":                                               "Elmo",
	"Dear Elmo,":                                             "Elmo",
	"I met Elmo yesterday at the conference.":                "Elmo",
	"Please email Elmo today about the contract.":            "Elmo",
	"Elmo said the build failed again.":                      "Elmo",
	"Hi Elmo Smith,":                                         "Elmo Smith",
	"Elmo Smith said hi":                                     "Elmo Smith",
	"Hi Athena,":                                             "Athena",
	"Dear Athena,":                                           "Athena",
	"I met Athena yesterday at the conference.":              "Athena",
	"Please email Athena today about the contract.":          "Athena",
	"Athena said the build failed again.":                    "Athena",
	"Hi Athena Smith,":                                       "Athena Smith",
	"Athena Smith said hi":                                   "Athena Smith",
	"Hi Aurora,":                                             "Aurora",
	"Dear Aurora,":                                           "Aurora",
	"I met Aurora yesterday at the conference.":              "Aurora",
	"Please email Aurora today about the contract.":          "Aurora",
	"Aurora said the build failed again.":                    "Aurora",
	"Hi Aurora Smith,":                                       "Aurora Smith",
	"Aurora Smith said hi":                                   "Aurora Smith",
	"Hi Apollo,":                                             "Apollo",
	"Dear Apollo,":                                           "Apollo",
	"I met Apollo yesterday at the conference.":              "Apollo",
	"Please email Apollo today about the contract.":          "Apollo",
	"Apollo said the build failed again.":                    "Apollo",
	"Hi Apollo Smith,":                                       "Apollo Smith",
	"Apollo Smith said hi":                                   "Apollo Smith",
	"Hi Hermes,":                                             "Hermes",
	"Dear Hermes,":                                           "Hermes",
	"I met Hermes yesterday at the conference.":              "Hermes",
	"Please email Hermes today about the contract.":          "Hermes",
	"Hermes said the build failed again.":                    "Hermes",
	"Hi Hermes Smith,":                                       "Hermes Smith",
	"Hermes Smith said hi":                                   "Hermes Smith",
	"Hi Atlas,":                                              "Atlas",
	"Dear Atlas,":                                            "Atlas",
	"I met Atlas yesterday at the conference.":               "Atlas",
	"Please email Atlas today about the contract.":           "Atlas",
	"Atlas said the build failed again.":                     "Atlas",
	"Hi Atlas Smith,":                                        "Atlas Smith",
	"Atlas Smith said hi":                                    "Atlas Smith",
	"Hi Venus,":                                              "Venus",
	"Dear Venus,":                                            "Venus",
	"I met Venus yesterday at the conference.":               "Venus",
	"Please email Venus today about the contract.":           "Venus",
	"Venus said the build failed again.":                     "Venus",
	"Hi Venus Smith,":                                        "Venus Smith",
	"Venus Smith said hi":                                    "Venus Smith",
	"Hi Mercury,":                                            "Mercury",
	"Dear Mercury,":                                          "Mercury",
	"I met Mercury yesterday at the conference.":             "Mercury",
	"Please email Mercury today about the contract.":         "Mercury",
	"Mercury said the build failed again.":                   "Mercury",
	"Hi Mercury Smith,":                                      "Mercury Smith",
	"Mercury Smith said hi":                                  "Mercury Smith",
	"Hi Jenkins,":                                            "Jenkins",
	"Dear Jenkins,":                                          "Jenkins",
	"I met Jenkins yesterday at the conference.":             "",
	"Please email Jenkins today about the contract.":         "",
	"Jenkins said the build failed again.":                   "",
	"Hi Jenkins Smith,":                                      "Jenkins Smith",
	"Jenkins Smith said hi":                                  "",
	"Dear Ruby,":                                             "Ruby",
	"I met Ruby yesterday at the conference.":                "Ruby",
	"Please email Ruby today about the contract.":            "Ruby",
	"Ruby said the build failed again.":                      "Ruby",
	"Hi Ruby Smith,":                                         "Ruby Smith",
	"Ruby Smith said hi":                                     "Ruby Smith",
	"Dear Ada,":                                              "Ada",
	"I met Ada yesterday at the conference.":                 "Ada",
	"Please email Ada today about the contract.":             "Ada",
	"Ada said the build failed again.":                       "Ada",
	"Hi Ada Smith,":                                          "Ada Smith",
	"Ada Smith said hi":                                      "Ada Smith",
	"Dear Julia,":                                            "Julia",
	"I met Julia yesterday at the conference.":               "Julia",
	"Please email Julia today about the contract.":           "Julia",
	"Julia said the build failed again.":                     "Julia",
	"Hi Julia Smith,":                                        "Julia Smith",
	"Julia Smith said hi":                                    "Julia Smith",
	"Adam optimizer":                                         "",
	"Jasmine tests":                                          "",
	"Hugo site":                                              "",
	"Travis CI":                                              "",
	"Jenkins pipeline":                                       "",
	"Apollo client":                                          "",
	"Atlas cluster":                                          "",
	"Mercury release":                                        "",
	"Hi Claude,":                                             "",
	"Hi Docker,":                                             "Docker",
	"Dear Kubernetes,":                                       "",
	"Hi Adam Driver,":                                        "Adam Driver",
	"Dear Ernie Banks,":                                      "Ernie Banks",
	"Ruby Rose said the build failed again.":                 "Ruby Rose",
	"Hi Travis Scott,":                                       "Travis Scott",
	"Dear Jade Thirlwall,":                                   "Jade Thirlwall",
	"Crystal Reed called":                                    "Crystal Reed",
	"Jasmine Rice is on the menu.":                           "",
	"Dear Julia,\nhappy birthday from all of us!":            "Julia",
	"Hi Ruby,\nI attached my resume for the nurse position.": "Ruby",
	"Hi Grafana,":                                            "",
	"Hi Prometheus,":                                         "",
	"Hi Kibana,":                                             "",
	"Hi Elasticsearch,":                                      "",
	"Hi Datadog,":                                            "",
	"Hi Splunk,":                                             "",
	"Hi Sentry,":                                             "",
	"Hi Vercel,":                                             "",
	"Hi Netlify,":                                            "",
	"Hi Supabase,":                                           "",
	"Hi Figma,":                                              "",
	"Hi Notion,":                                             "",
	"Hi Zapier,":                                             "",
	"Hi Heroku,":                                             "",
	"Hi Firebase,":                                           "",
	"Hi Pytorch,":                                            "",
	"Hi Tensorflow,":                                         "",
	"Hi Snowflake,":                                          "",
	"Hi Databricks,":                                         "",
	"Hi Airtable,":                                           "",
	"Hi Trello,":                                             "",
	"Hi Asana,":                                              "",
}

func detectedGreetingPeople(t *testing.T, text string) []string {
	t.Helper()
	var people []entity.Span
	for _, d := range []Detector{NameDetector{}, NewRegexDetector()} {
		spans, err := d.Detect(context.Background(), text)
		if err != nil {
			t.Fatal(err)
		}
		for _, sp := range spans {
			if sp.Type == entity.Person {
				people = append(people, sp)
			}
		}
	}
	var out []string
	for _, sp := range Resolve(people) {
		out = append(out, sp.Text)
	}
	return out
}

func TestGreetingTargetsAndProductNames(t *testing.T) {
	for in, want := range greetingTargetCases {
		if got := strings.Join(detectedGreetingPeople(t, in), "|"); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}
