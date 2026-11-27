package outworld

import (
	"regexp"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/detect"
	"github.com/davidparkercodes/outworld/pkg/surrogate"
)

func TestSentenceContextNamesMaskedAndRestored(t *testing.T) {
	for in, reals := range map[string][]string{
		"Sun Yang will join the call.":                   {"Sun", "Yang"},
		"I asked Ma Long to review it.":                  {"Ma Long"},
		"Lunch with Li Na, then the demo.":               {"Li Na"},
		"Sun Yang's review is due.":                      {"Sun", "Yang"},
		"Edward Jones called me.":                        {"Edward", "Jones"},
		"Abbott said hi.":                                {"Abbott"},
		"I emailed Raymond James yesterday.":             {"Raymond", "James"},
		"please ask kchen to review":                     {"kchen"},
		"ping mwilliams about it":                        {"mwilliams"},
		"assigned to jdoe":                               {"jdoe"},
		"cc: kchen":                                      {"kchen"},
		"// TODO(will): this will fail when will is nil": {"(will)"},
		"// TODO(may): later":                            {"(may)"},
	} {
		out := maskRoundTrip(t, newEngine(t, nil), in)
		for _, r := range reals {
			if strings.Contains(out, r) {
				t.Errorf("%q left in %q", r, out)
			}
		}
	}
	out := maskRoundTrip(t, newEngine(t, nil), "// TODO(will): this will fail when will is nil")
	if !strings.HasSuffix(out, "): this will fail when will is nil") {
		t.Errorf("ordinary word changed: %q", out)
	}
}

func TestOrdinaryTextUntouched(t *testing.T) {
	for _, in := range []string{
		"We drove to Long Island. Sun Microsystems built it. Ma Bell was a monopoly.",
		"He Said no. Do Not Disturb is on. Park Avenue is closed. I read Le Monde daily.",
		"Flights to Ho Chi Minh City are cheap. Han Solo shot first. Wu Tang Clan forever.",
		"Edward Jones reported earnings. Morgan Stanley downgraded the stock.",
		"I bought shares of Abbott. She works at Raymond James.",
		"Goldman Sachs said in a statement that it would.",
		"ask sparks, ping google, per mille, from master, by default, with redis",
		"// TODO(perf): a TODO(security): b TODO(cleanup): c TODO(v2): d TODO(later): e TODO(bug): f TODO(ABC-123): g",
		"class KChen extends Base {}",
		"type KChen struct{}",
		"func KChen() {}",
		"the KChen module",
	} {
		if out := maskRoundTrip(t, newEngine(t, nil), in); out != in {
			t.Errorf("changed %q to %q", in, out)
		}
	}
}

func TestCodeIdentifiersAreNotDictionaryHandles(t *testing.T) {
	for _, in := range []string{
		"class JSmith extends Base {}",
		"type JSmith struct{}",
		"func JSmith() {}",
		"the JSmith module",
		"class JohnSmith extends Base {}",
	} {
		if out := maskRoundTrip(t, newEngine(t, nil), in); out != in {
			t.Errorf("changed %q to %q", in, out)
		}
	}
	out := maskRoundTrip(t, newEngine(t, nil), "jsmith pushed it. Then class JSmith was added by J. Smith and John Smith.")
	if strings.Contains(out, "jsmith") || strings.Contains(out, "Smith and") || strings.Contains(out, "John") || !strings.Contains(out, "class JSmith was") {
		t.Errorf("got %q", out)
	}
	d, err := detect.ParseDictionaryCSV(strings.NewReader("value,type,aliases\nJohn Smith,person,JSmith\n"))
	if err != nil {
		t.Fatal(err)
	}
	e := newEngine(t, func(o *Options) { o.Dictionary = d })
	if out := maskRoundTrip(t, e, "class JSmith extends Base {}"); strings.Contains(out, "JSmith") {
		t.Errorf("listed alias left in %q", out)
	}
}

func TestFamilyFirstSurrogateKeepsOriginAndShape(t *testing.T) {
	e := newEngine(t, nil)
	out := maskRoundTrip(t, e, "Nguyen Van An joined.")
	fam, _ := e.Vault().Get("person.family", "Nguyen")
	last, _ := e.Vault().Get("person.given", "An")
	if fam == "" || last == "" || out != fam+" Van "+last+" joined." {
		t.Fatalf("got %q (family %q, given %q)", out, fam, last)
	}
	if _, ok := e.Vault().Get("person.family", "An"); ok {
		t.Fatal("given token stored as a family name")
	}
	if _, ok := e.Vault().Get("person.given", "Van"); ok {
		t.Fatal("middle marker stored as a name")
	}
	if !surrogate.IsBundledEastName(fam) || !surrogate.IsBundledEastName(last) {
		t.Fatalf("not from the origin pools: %q %q", fam, last)
	}
	if again := maskRoundTrip(t, newEngine(t, nil), "Nguyen Van An joined."); again != out {
		t.Fatalf("not deterministic: %q then %q", out, again)
	}
	later := maskRoundTrip(t, e, "Nguyen Van An joined. Then Van An Nguyen signed.")
	if later != fam+" Van "+last+" joined. Then Van "+last+" "+fam+" signed." {
		t.Fatalf("given-first mention not consistent: %q", later)
	}
	out = maskRoundTrip(t, newEngine(t, nil), "Tran Thi Mai called. Then Mai Tran left.")
	m := regexp.MustCompile(`^(\p{Lu}\p{Ll}+) Thi (\p{Lu}\p{Ll}+) called\. Then (\p{Lu}\p{Ll}+) (\p{Lu}\p{Ll}+) left\.$`).FindStringSubmatch(out)
	if m == nil || m[1] != m[4] || m[2] != m[3] || !surrogate.IsBundledEastName(m[1]) || !surrogate.IsBundledEastName(m[2]) {
		t.Fatalf("got %q", out)
	}
}

func TestFamilyFirstGivenKeepsHyphenAndCase(t *testing.T) {
	lower := regexp.MustCompile(`^(\p{Lu}\p{Ll}+) (\p{Lu}\p{Ll}+)-(\p{Ll}+) joined\.$`)
	upper := regexp.MustCompile(`^(\p{Lu}\p{Ll}+) (\p{Lu}\p{Ll}+)-(\p{Lu}\p{Ll}+) joined\.$`)
	joined := regexp.MustCompile(`^(\p{Lu}\p{Ll}+) (\p{Lu}\p{Ll}+)() joined\.$`)
	for in, re := range map[string]*regexp.Regexp{"Kim Min-jun joined.": lower, "Kim Min-Jun joined.": upper, "Seo Ji-hoon joined.": lower, "Kim Minjun joined.": joined, "Li Xiaoming joined.": joined, "Zhang Wei joined.": joined} {
		out := maskRoundTrip(t, newEngine(t, nil), in)
		m := re.FindStringSubmatch(out)
		if m == nil {
			t.Fatalf("%q: shape of %q", in, out)
		}
		if !surrogate.IsBundledEastName(m[1]) || (m[3] != "" && (!surrogate.IsBundledEastName(m[2]) || !surrogate.IsBundledEastName(m[3]))) {
			t.Errorf("%q: not from the origin pools: %q", in, out)
		}
		if surrogate.IsBundledFamilyName(m[1]) || surrogate.IsBundledGivenName(m[2]) {
			t.Errorf("%q: general list used in %q", in, out)
		}
		if again := maskRoundTrip(t, newEngine(t, nil), in); again != out {
			t.Errorf("not deterministic: %q then %q", out, again)
		}
	}
	e := newEngine(t, nil)
	out := maskRoundTrip(t, e, "Kim Min-jun joined. Later Min-jun replied and Kim Min-jun left.")
	first, _ := e.Vault().Get("person.given", "Min")
	second, _ := e.Vault().Get("person.given", "jun")
	if first == "" || second == "" || strings.Count(out, first+"-"+strings.ToLower(second)) != 3 {
		t.Fatalf("hyphenated given not consistent: %q", out)
	}
}

func TestFamilyFirstSurrogateAvoidsRealNames(t *testing.T) {
	reals := []string{"Nguyen", "Tran", "Pham", "Hoang", "Huynh", "Phan", "Dang", "Duong", "Bui", "Ngo"}
	d, err := detect.ParseDictionaryCSV(strings.NewReader("value,type,aliases\nKieu Linh,person,\n"))
	if err != nil {
		t.Fatal(err)
	}
	e := newEngine(t, func(o *Options) { o.Dictionary = d })
	var b strings.Builder
	for _, r := range reals {
		b.WriteString(r + " Van Hai joined. ")
	}
	out := maskRoundTrip(t, e, b.String())
	for _, r := range append(reals, "Kieu", "Linh", "Hai") {
		if strings.Contains(out, r) {
			t.Errorf("real name %q in %q", r, out)
		}
	}
	if strings.Count(out, " Van ") != len(reals) {
		t.Errorf("marker moved in %q", out)
	}
}

func TestOtherNameShapesKeepTheirRoles(t *testing.T) {
	e := newEngine(t, nil)
	out := maskRoundTrip(t, e, "Anna Dupont met Mary Smith-Jones and Kim Rose.")
	for word, kind := range map[string]string{"Anna": "person.given", "Dupont": "person.family", "Mary": "person.given", "Jones": "person.family", "Kim": "person.given", "Rose": "person.family"} {
		sur, ok := e.Vault().Get(kind, word)
		if !ok || surrogate.IsBundledEastName(sur) {
			t.Errorf("%s not stored as %s with a general surrogate in %q", word, kind, out)
		}
	}
}

func TestHandleInitialComesFromTheSurrogate(t *testing.T) {
	e := newEngine(t, nil)
	out := maskRoundTrip(t, e, "Karen Chen opened it. Then kchen pushed, k.chen replied and k_chen merged.")
	giv, _ := e.Vault().Get("person.given", "Karen")
	fam, _ := e.Vault().Get("person.family", "Chen")
	if giv == "" || fam == "" {
		t.Fatalf("missing surrogates in %q", out)
	}
	i, f := strings.ToLower(giv[:1]), strings.ToLower(fam)
	if out != giv+" "+fam+" opened it. Then "+i+f+" pushed, "+i+"."+f+" replied and "+i+"_"+f+" merged." {
		t.Fatalf("handle not consistent with the full name: %q", out)
	}
	shapes := map[string]*regexp.Regexp{
		"please ask kchen to review":   regexp.MustCompile(`^please ask ([a-z])[a-z]+ to review$`),
		"TODO(k.chen): fix":            regexp.MustCompile(`^TODO\(([a-z])\.[a-z]+\): fix$`),
		"TODO(jdoe2): later":           regexp.MustCompile(`^TODO\(([a-z])[a-z]+2\): later$`),
		"thanks sarahk for the fix":    regexp.MustCompile(`^thanks [a-z]+([a-z]) for the fix$`),
		`open C:\Users\JDoe\Desktop\a`: regexp.MustCompile(`^open C:\\Users\\([A-Z])[a-z]+\\Desktop\\a$`),
	}
	reals := map[string]string{"please ask kchen to review": "k", "TODO(k.chen): fix": "k", "TODO(jdoe2): later": "j", "thanks sarahk for the fix": "k", `open C:\Users\JDoe\Desktop\a`: "J"}
	for in, re := range shapes {
		out := maskRoundTrip(t, newEngine(t, nil), in)
		m := re.FindStringSubmatch(out)
		if m == nil {
			t.Errorf("%q: shape of %q", in, out)
			continue
		}
		if m[1] == reals[in] {
			t.Errorf("%q: real initial kept in %q", in, out)
		}
		if again := maskRoundTrip(t, newEngine(t, nil), in); again != out {
			t.Errorf("not deterministic: %q then %q", out, again)
		}
	}
	for _, in := range []string{"J. Smith wrote this.", "Smith, J. wrote this."} {
		if out := maskRoundTrip(t, newEngine(t, nil), in); !strings.Contains(out, "J.") {
			t.Errorf("prose initial changed in %q", out)
		}
	}
}

func TestBranchUserPrefixesMasked(t *testing.T) {
	for in, handle := range map[string]string{
		"git push origin dlopez/fix-cache":        "dlopez",
		"PR from kchen/fix-auth":                  "kchen",
		"pull request from mwilliams/tidy":        "mwilliams",
		"git pull upstream mwilliams/retry-logic": "mwilliams",
		"git fetch origin kchen/wip":              "kchen",
		"gh pr checkout kchen/fix-auth":           "kchen",
		"git switch -c dlopez/new-cache":          "dlopez",
		"git branch -D kchen/old":                 "kchen",
		"git rebase kchen/base":                   "kchen",
		"git log main..kchen/fix-auth":            "kchen",
		"git diff origin/kchen/fix-auth":          "kchen",
		"it was rebased onto dlopez/main2":        "dlopez",
		"based on kchen/x and off of jsmith/y":    "kchen",
		"gh pr create --head kchen:fix-auth":      "kchen",
		"refs/heads/dlopez/fix-cache":             "dlopez",
	} {
		out := maskRoundTrip(t, newEngine(t, nil), in)
		if strings.Contains(out, handle) {
			t.Errorf("handle %q left in %q", handle, out)
		}
	}
	for _, in := range []string{
		"git push origin feature/login",
		"git push origin release/1.2 hotfix/x renovate/deps dependabot/npm",
		"git push origin main",
		"git diff users/models.py internal/app src/x walker/x.go vendor/lib docs/a handlers/b models/c",
		"git log parker/x.go and git branch main",
		"note:this-thing key:value-pair time:12-30 localhost:8080 step:build-all",
		"image: redis:7-alpine and node:18-slim, user:fix-auth",
		"PR from feature/login, based on release/1.2, off of main",
	} {
		if out := maskRoundTrip(t, newEngine(t, nil), in); out != in {
			t.Errorf("changed %q to %q", in, out)
		}
	}
}

func TestFourNameGapsMasked(t *testing.T) {
	for in, reals := range map[string][]string{
		"Kevin from Morgan Stanley called.":      {"Kevin"},
		"PR #412 from kchen/fix-login is ready.": {"kchen"},
		"Thanks\n-- Sam\n":                       {"Sam"},
		"// Jordan's hack\n":                     {"Jordan"},
		"# Priya's workaround\n":                 {"Priya"},
		"/* Sam's fix */\n":                      {"Sam"},
	} {
		out := maskRoundTrip(t, newEngine(t, nil), in)
		for _, r := range reals {
			if strings.Contains(out, r) {
				t.Errorf("%q left in %q", r, out)
			}
		}
	}
	for in, keep := range map[string]string{
		"Kevin from Morgan Stanley called.":             "Morgan Stanley",
		"He called from Morgan Stanley about the loan.": "Morgan Stanley",
		"called from Morgan Stanley about the loan.":    "Morgan Stanley",
	} {
		if out := maskRoundTrip(t, newEngine(t, nil), in); !strings.Contains(out, keep) {
			t.Errorf("%q masked in %q", keep, out)
		}
	}
	for _, in := range []string{
		"-- see below\n",
		"- Monday\n- Apple\n",
		"// Python's GIL\n// Go's scheduler\n// Monday's build\n// Amazon's API\n",
	} {
		if out := maskRoundTrip(t, newEngine(t, nil), in); out != in {
			t.Errorf("changed %q to %q", in, out)
		}
	}
}
