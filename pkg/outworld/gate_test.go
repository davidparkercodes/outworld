package outworld

import (
	"fmt"
	"github.com/davidparkercodes/outworld/internal/scaletest"
	"regexp"
	"strings"
	"testing"
)

var gateLeaks = []string{
	"Let Will know.", "Can Will join?", "Does Will agree?", "Is Will free?", "Will (accounting)", "Will -- see above",
	"I'll loop in Will tomorrow.", "DM Will", "Give Will a call", "Cheers, Will", "-- Will", "Will wants the deck",
	"Will has the keys", "Will got back to me", "Will was there", "<Will> ok", "Will?", "Will from accounting",
	"Will on the design team", "Sarah Lindqvist, Will and Raj Patel met.", "* Will, can you review?", "# Will, status?",
	"-- Will, can you review?", "Mark's laptop broke", "## Mark's notes",
}

var gateNegatives = []string{
	"Mark as deleted", "Please mark the box", "mark the box", "Grant access to the repo", "A rose by any name",
	"// Mark's position", "Mark down the price", "Mark my words", "Rose to the occasion", "Prices rose and fell",
	"The rose and the lily", "Mark and sweep GC", "Mark the date", "Grant writing tips", "Rose gold iPhone",
	"I will go", "in May", "the rose garden", "pay the bill", "Grace period", "Will this work?", "Will it compile?",
	"We will ship it", "Free will is debated", "The will was read aloud", "Bill of materials",
}

var gateSeeds = []string{
	"Notes from Will Turner about the launch.", "Notes from Mark Jones about the launch.", "Notes from Rose Kim about the launch.",
	"Notes from Grace Okafor about the launch.", "Notes from Frank Novak about the launch.", "Notes from June Park about the launch.",
	"Notes from Bill Hughes about the launch.", "Notes from Pat Morrow about the launch.", "Notes from Grant Lee about the launch.",
	"Notes from May Chen about the launch.", "Notes from Hope Diaz about the launch.", "Notes from June Carter about the launch.",
}

func seededGateEngine(t *testing.T) *Engine {
	t.Helper()
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	for _, in := range gateSeeds {
		maskRoundTrip(t, e, in)
	}
	return e
}

func hasWord(text, w string) bool {
	return regexp.MustCompile(`\b` + regexp.QuoteMeta(w) + `\b`).MatchString(text)
}

func TestCommonWordNamesMaskAcrossRequests(t *testing.T) {
	e := seededGateEngine(t)
	for _, name := range []string{"Will", "Rose", "Grace", "Mark", "Frank", "Bill", "Pat", "Grant", "June"} {
		for _, in := range gateLeaks {
			in = strings.ReplaceAll(in, "Will", name)
			if name != "Mark" {
				in = strings.ReplaceAll(in, "Mark", name)
			}
			if out := maskRoundTrip(t, e, in); hasWord(out, name) {
				t.Errorf("%q -> %q", in, out)
			}
		}
	}
	for _, in := range []string{"WILL wants the deck", "Let WILL know."} {
		if out := maskRoundTrip(t, e, in); hasWord(out, "WILL") {
			t.Errorf("%q -> %q", in, out)
		}
	}
}

func TestCommonWordNegativesOnPoisonedVault(t *testing.T) {
	e := seededGateEngine(t)
	for _, in := range gateNegatives {
		if out := maskRoundTrip(t, e, in); out != in {
			t.Errorf("%q -> %q", in, out)
		}
	}
}

func TestMemoDoesNotLockInAnUnknownName(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	first := e.NewSession("")
	if _, err := first.Mask(ctx, "Met Will Turner yesterday."); err != nil {
		t.Fatal(err)
	}
	fresh := newEngine(t, func(o *Options) { o.Dictionary = nil })
	s := fresh.NewSession("")
	out, err := s.Mask(ctx, "Let Will know.")
	if err != nil {
		t.Fatal(err)
	}
	if out != "Let Will know." {
		t.Fatalf("unknown name masked on a fresh vault: %q", out)
	}
	s = fresh.NewSession("")
	if _, err := s.Mask(ctx, "Met Will Turner yesterday."); err != nil {
		t.Fatal(err)
	}
	if again, _ := s.Mask(ctx, "Let Will know."); hasWord(again, "Will") {
		t.Fatalf("memo returned the raw name: %q", again)
	}
	if again, _ := fresh.NewSession("").Mask(ctx, "Let Will know."); hasWord(again, "Will") {
		t.Fatalf("later request returned the raw name: %q", again)
	}
}

func uniqueScope(prefix string) func() string {
	n := 0
	return func() string {
		n++
		return fmt.Sprintf("%s-%d", prefix, n)
	}
}

func TestCommonWordGateScalesLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("scale test skipped in short mode")
	}
	e := seededGateEngine(t)
	gate := func(n int) func() {
		text := strings.Repeat("xq Will. ", n/9)
		s := e.NewSession("scale")
		return func() {
			hits := 0
			for _, h := range e.Vault().FindReals(text) {
				if _, ok := s.realHit(text, h, nil, nil); ok {
					hits++
				}
			}
			if hits < len(text)/9 {
				t.Fatalf("only %d of %d hits masked", hits, len(text)/9)
			}
		}
	}
	scaletest.Check(t, "common-word gate", scaletest.Size(128<<10), gate)
	scope := uniqueScope("gate-mask")
	mask := func(n int) func() {
		in := strings.Repeat("xq Will. ", n/9)
		return func() {
			out, err := e.NewSession(scope()).Mask(ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			if hasWord(out, "Will") {
				t.Fatal("raw name left")
			}
		}
	}
	scaletest.Check(t, "common-word mask", scaletest.Size(32<<10), mask)
}

func TestViolationPreviewsScaleLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("scale test skipped in short mode")
	}
	e := newEngine(t, nil)
	scope := uniqueScope("previews")
	check := func(n int) func() {
		text := strings.Repeat("mail ann.lee@corp.test now ", n/27)
		return func() {
			s := e.NewSession(scope())
			s.Check(ctx, text)
			v := s.Violations()
			if len(v) == 0 {
				t.Fatal("no violations")
			}
			if strings.Contains(v[len(v)-1].Preview, "ann.lee") {
				t.Fatal("preview shows the value")
			}
		}
	}
	scaletest.Check(t, "violation previews", scaletest.Size(128<<10), check)
}

func TestMemoThreeRequestRepro(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	if _, err := e.NewSession("").Mask(ctx, "Met Will Turner yesterday."); err != nil {
		t.Fatal(err)
	}
	r2, err := e.NewSession("").Mask(ctx, "Let Will know.")
	if err != nil || hasWord(r2, "Will") {
		t.Fatalf("r2 %q %v", r2, err)
	}
	s := e.NewSession("")
	for _, in := range []string{"Met Will Turner yesterday.", "Let Will know."} {
		out, err := s.Mask(ctx, in)
		if err != nil || hasWord(out, "Will") {
			t.Fatalf("r3 %q -> %q %v", in, out, err)
		}
	}
	if s.MemoHits() == 0 {
		t.Fatal("memo not used for masked strings")
	}
}

func TestTitleCaseNeverSwallowsTheNextWord(t *testing.T) {
	for _, e := range []*Engine{seededGateEngine(t), newEngine(t, func(o *Options) { o.Dictionary = nil })} {
		for in, kept := range map[string]string{
			"Grant Proposals Are Due Friday": "Proposals Are Due Friday",
			"Rose Petal Theme":               "Petal Theme",
		} {
			if out := maskRoundTrip(t, e, in); !strings.HasSuffix(out, kept) || strings.Count(out, " ") != strings.Count(in, " ") {
				t.Errorf("%q -> %q", in, out)
			}
		}
	}
	e := seededGateEngine(t)
	if out := maskRoundTrip(t, e, "Grant Lee signed the form."); hasWord(out, "Grant") || hasWord(out, "Lee") {
		t.Fatalf("known full name kept: %q", out)
	}
}

func TestMoreOrdinaryUsesStayOnPoisonedVault(t *testing.T) {
	e := seededGateEngine(t)
	for _, in := range []string{"Hope you are well.", "Rose 3% overnight", "- [ ] Mark complete", "Will-o'-the-wisp", "The Will to Power", "The Will of the People"} {
		if out := maskRoundTrip(t, e, in); out != in {
			t.Errorf("%q -> %q", in, out)
		}
	}
}

func TestMorePersonUsesMask(t *testing.T) {
	e := seededGateEngine(t)
	for in, name := range map[string]string{
		"Thanks Will": "Will", "Will's PR is ready": "Will", "WILL: on it": "WILL", "Assigned to Will": "Will",
		"Reviewer: Grant": "Grant", "Hope and I will join": "Hope", "Hope said yes": "Hope", "- Mark": "Mark",
		"- [ ] Mark to review the doc": "Mark", "Hope, are you free?": "Hope", "# Mark, as noted": "Mark",
		"Let Will know.": "Will", "Can Will join?": "Will", "Will wants the deck": "Will", "Sarah, Will and Raj": "Will",
	} {
		if out := maskRoundTrip(t, e, in); hasWord(out, name) {
			t.Errorf("%q -> %q", in, out)
		}
	}
}

func TestJuneCarterSeedAndPersonVerbs(t *testing.T) {
	e := seededGateEngine(t)
	if _, ok := e.Vault().Get("person.given", "June"); !ok {
		t.Fatal("June Carter seed not masked")
	}
	for _, in := range []string{"Let June know.", "Ask with June", "I told June", "June -- see above", "June said yes", "Will thinks so"} {
		name := "June"
		if strings.HasPrefix(in, "Will") {
			name = "Will"
		}
		if out := maskRoundTrip(t, e, in); hasWord(out, name) {
			t.Errorf("%q -> %q", in, out)
		}
	}
	for _, in := range []string{"Grant Proposals Are Due Friday", "Rose Petal Theme"} {
		if out := maskRoundTrip(t, e, in); out != in {
			t.Errorf("%q -> %q", in, out)
		}
	}
}

func TestCoordinatedMonthNamesAcrossRequests(t *testing.T) {
	e := seededGateEngine(t)
	if out := maskRoundTrip(t, e, "May and June went to lunch"); hasWord(out, "May") || hasWord(out, "June") {
		t.Errorf("person pair kept: %q", out)
	}
	for _, in := range []string{"May, June and July", "May and June are busy months"} {
		if out := maskRoundTrip(t, e, in); out != in {
			t.Errorf("%q -> %q", in, out)
		}
	}
}
