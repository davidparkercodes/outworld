package outworld

import (
	"fmt"
	"github.com/davidparkercodes/outworld/internal/scaletest"
	"strings"
	"sync"
	"testing"
)

func letters(i int) string {
	var b []byte
	for {
		b = append(b, byte('a'+i%26))
		i /= 26
		if i == 0 {
			return string(b)
		}
	}
}

func TestFoldIndexUnderConcurrentWriter(t *testing.T) {
	if testing.Short() {
		t.Skip("scale test skipped in short mode")
	}
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	for i := 0; i < 4000; i++ {
		real := "Zoë" + letters(i)
		if _, err := e.Vault().Assign("person.given", real, func(a int) string { return fmt.Sprintf("Gen%da%d", i, a) }); err != nil {
			t.Fatal(err)
		}
	}
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			e.Vault().Assign("person.family", fmt.Sprintf("Writer%d", i), func(a int) string { return fmt.Sprintf("Wr%da%d", i, a) })
		}
	}()
	scope := uniqueScope("writer")
	mask := func(n int) func() {
		text := strings.Repeat("ping zoe"+letters(12)+" and francois today. ", n/32)
		return func() {
			if _, err := e.NewSession(scope()).Mask(ctx, text); err != nil {
				t.Fatal(err)
			}
		}
	}
	scaletest.Check(t, "mask with a concurrent writer", scaletest.Size(64<<10), mask)
	close(stop)
	wg.Wait()
}

func TestDateBookKeepsRealDatesUnderModelFlood(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("gaps")
	if _, err := s.Mask(ctx, "Dentist on March 3, 2024, then may 4 for the cleaning."); err != nil {
		t.Fatal(err)
	}
	if fresh := maskRoundTrip(t, newEngine(t, nil), "Sure: may 4 works."); fresh != "Sure: may 4 works." {
		t.Fatalf("strict detector already shifts this form: %q", fresh)
	}
	for i := 0; i < 3*maxModelDates+4200; i++ {
		e.realDates.add("gaps", fmt.Sprintf("june %d, %d", 1+i%28, i), false)
	}
	for i := 0; i < 4200; i++ {
		e.realDates.add(fmt.Sprintf("other-%d", i), "may 4", false)
	}
	if out := maskRoundTrip(t, e, "Sure: may 4 works."); strings.Contains(out, "may 4") {
		t.Fatalf("real date went out raw after a flood: %q", out)
	}
}

func TestDateBookSaturatesClosed(t *testing.T) {
	b := newDateBook()
	for i := 0; i <= maxRealDates; i++ {
		b.add("s", fmt.Sprintf("real %d", i), true)
	}
	if got := b.lookup("s", "never seen", false); !got.saturated {
		t.Fatal("full real tier is not reported as saturated")
	}
}

func TestDateBookScopeKeysAreHashed(t *testing.T) {
	b := newDateBook()
	b.add(strings.Repeat("s", 1<<16), "3/4", true)
	for k := range b.scopes {
		if len(k) != 32 {
			t.Fatal("scope key not hashed")
		}
	}
	if !b.lookup(strings.Repeat("s", 1<<16), "3/4", true).known {
		t.Fatal("lookup by scope failed")
	}
}

func TestWeakKnownDatesNeedContext(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("gaps")
	if _, err := s.Mask(ctx, "We meet on 3/4 at noon, then call on May 2, 2024 and again May 2."); err != nil {
		t.Fatal(err)
	}
	for _, in := range []string{"x = 3/4;", "3/4 tests passed", "(3/4)", "passed 3/4", "Ratio [3/4]", "you may 2 times"} {
		if out := maskRoundTrip(t, e, in); out != in {
			t.Errorf("%q -> %q", in, out)
		}
	}
	c := e.NewSession("gaps")
	c.Check(ctx, `{"3/4": 12}`)
	if err := c.Err(); err != nil {
		t.Fatalf("fraction key blocked: %v", err)
	}
}

func TestRehydratedWeakNumericDateIsShifted(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("gaps")
	masked, err := s.Mask(ctx, "Meeting on 3/4 with the team.")
	if err != nil {
		t.Fatal(err)
	}
	shifted := strings.Fields(masked)[2]
	back := s.Rehydrate("Done (" + shifted + ")")
	if back != "Done (3/4)" {
		t.Fatalf("rehydrated %q", back)
	}
	for _, in := range []string{"Done (3/4)", "the 3/4 sync", "3/4 works for me"} {
		if out := maskRoundTrip(t, e, in); strings.Contains(out, "3/4") {
			t.Errorf("%q went out raw: %q", in, out)
		}
	}
	c := e.NewSession("gaps")
	c.Check(ctx, `{"3/4": 12}`)
	if err := c.Err(); err != nil {
		t.Fatalf("rehydrated weak date became a violation: %v", err)
	}
}

func TestAsciiFormGetsItsOwnSurrogate(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	maskRoundTrip(t, e, "José García joined the call.")
	g, _ := e.Vault().Get("person.given", "José")
	f, _ := e.Vault().Get("person.family", "García")
	out := maskRoundTrip(t, e, "Jose Garcia joined the call.")
	if out == g+" "+f+" joined the call." || strings.Contains(out, "Jose") || strings.Contains(out, "Garcia") {
		t.Fatalf("ASCII spelling shares the accented stand-ins or leaks: %q", out)
	}

	mail, err := e.NewSession("p").Mask(ctx, "Write to jose.garcia@example.com today.")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mail, strings.ToLower(g)+"."+strings.ToLower(f)+"@") {
		t.Fatalf("email not linked: %q", mail)
	}
}

func TestQuotedUsernameFromVault(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	seed := maskRoundTrip(t, e, "Mary Jones called.")
	full, _, _ := strings.Cut(seed, " called.")
	joined := strings.ToLower(strings.ReplaceAll(full, " ", ""))
	for in, want := range map[string]string{
		`username = "maryjones"`: `username = "` + joined + `"`,
		`user "maryjones"`:       `user "` + joined + `"`,
	} {
		if out := maskRoundTrip(t, e, in); out != want {
			t.Fatalf("%q -> %q, want %q", in, out, want)
		}
	}
}

func TestCrossRequestPersonForms(t *testing.T) {
	e := seededGateEngine(t)
	for _, in := range []string{
		"Rose to send the deck", "- Rose to send deck", "Rose to VP last year.", "Rose by Friday", "// Rose to fix",
		"Bill for the hours", "PING WILL",
	} {
		name := strings.Fields(strings.NewReplacer("-", " ", "/", " ").Replace(in))[0]
		if name == "Done." || name == "PING" {
			name = map[string]string{"Done.": "Will", "PING": "WILL"}[name]
		}
		if out := maskRoundTrip(t, e, in); hasWord(out, name) {
			t.Errorf("%q -> %q", in, out)
		}
	}
	for _, in := range []string{"Will this work?", "Will it compile?", "Will we need a migration?", "Will there be tests?", "Will you check?",
		"Will check the logs.", "Will fix it tomorrow.", "Will do.", "Will need the deck.", "Will help me", "Will merge it",
		"Will stop by", "Will test the build", "Done. Will be careful.", "Will fix later", "Will be done soon.", "Will this work.",
		"Status: WILL FIX", "WILL NOT FIX", "GRANT ACCESS TO ADMIN", "Rose to the occasion"} {
		if out := maskRoundTrip(t, e, in); out != in {
			t.Errorf("%q -> %q", in, out)
		}
	}
}

func TestCapsRunWithFamilyNameMasks(t *testing.T) {
	e := seededGateEngine(t)
	if out := maskRoundTrip(t, e, "NOTES: WILL TURNER TO SEND DECK"); hasWord(out, "WILL") {
		t.Fatalf("got %q", out)
	}
}

func TestMonthNamesWithPersonContext(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	for _, in := range []string{"I'll loop in June tomorrow.", "Waiting on June.", "Blocked on June", "Reviewed by June", "approved by June", "from June", "Notes From June"} {
		out := maskRoundTrip(t, e, "June Carter joined the team. "+in)
		if hasWord(out, "June") {
			t.Errorf("%q -> %q", in, out)
		}
	}
	seeded := newEngine(t, func(o *Options) { o.Dictionary = nil })
	maskRoundTrip(t, seeded, "June Carter joined the team.")
	if _, ok := seeded.Vault().Get("person.given", "June"); !ok {
		t.Skip("June not learned as a given name")
	}
	for _, in := range []string{"in May", "May, June and July", "Release Notes For June", "See you on June 5"} {
		if out := maskRoundTrip(t, seeded, in); out != in && !strings.Contains(in, "June 5") {
			t.Errorf("%q -> %q", in, out)
		}
	}
}

func TestFoldedOrdinaryWordsNeedACue(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	maskRoundTrip(t, e, "Karl Groß and Jonas Löw met José Ruiz.")
	for _, in := range []string{"that is gross", "import jose", "set it low"} {
		if out := maskRoundTrip(t, e, in); out != in {
			t.Errorf("%q -> %q", in, out)
		}
	}
	if out, _ := e.NewSession("p").Mask(ctx, "please ask jose about it"); hasWord(out, "jose") {
		t.Errorf("cued folded name kept: %q", out)
	}
}

func TestUserCueShapes(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	maskRoundTrip(t, e, "Mary Jones called. José García joined.")
	for _, in := range []string{`user="MaryJones"`, `user "mjones"`, `user "jgarcia"`, `user "garciajose"`, `username = “maryjones”`} {
		out := maskRoundTrip(t, e, in)
		low := strings.ToLower(out)
		if strings.Contains(low, "jones") || strings.Contains(low, "garcia") {
			t.Errorf("%q -> %q", in, out)
		}
	}
}

func TestMemoRejectCountsAsMiss(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	if _, err := e.NewSession("").Mask(ctx, "Let Will know."); err != nil {
		t.Fatal(err)
	}
	maskRoundTrip(t, e, "Met Will Turner yesterday.")
	before := e.MemoStats()
	if _, err := e.NewSession("").Mask(ctx, "Let Will know."); err != nil {
		t.Fatal(err)
	}
	after := e.MemoStats()
	if after.Hits != before.Hits || after.Misses <= before.Misses {
		t.Fatalf("rejected entry counted as a hit: %+v -> %+v", before, after)
	}
}

func TestDateBookScopesAreCappedWithTombstones(t *testing.T) {
	full := newDateBook()
	next := 0
	for ; next < maxDateScopes; next++ {
		full.add(fmt.Sprintf("s%d", next), "june 3", true)
	}
	churn := func(n int) func() {
		return func() {
			for i := 0; i < n; i++ {
				full.add(fmt.Sprintf("s%d", next), "june 3", true)
				next++
			}
		}
	}
	scaletest.Check(t, "datebook scope churn", 3*maxDateScopes, churn)
	b := newDateBook()
	b.add("first", "may 4", true)
	for i := 0; i < 3*maxDateScopes; i++ {
		b.add(fmt.Sprintf("s%d", i), "june 3", true)
	}
	if len(b.scopes) > maxDateScopes {
		t.Fatalf("%d live scopes", len(b.scopes))
	}
	if got := b.lookup("first", "anything", false); !got.saturated {
		t.Fatal("evicted scope with real dates is not saturated")
	}
}

func TestDenseDatesScaleLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("scale test skipped in short mode")
	}
	e := newEngine(t, nil)
	if _, err := e.NewSession("dense").Mask(ctx, "Meeting on 3/4 with the team."); err != nil {
		t.Fatal(err)
	}
	scope := uniqueScope("dense")
	mask := func(n int) func() {
		var b strings.Builder
		for i := 0; b.Len() < n; i++ {
			fmt.Fprintf(&b, "on %d/%d ", 1+i%12, 1+i%28)
		}
		text := b.String()
		return func() {
			if _, err := e.NewSession(scope()).Mask(ctx, text); err != nil {
				t.Fatal(err)
			}
		}
	}
	scaletest.Check(t, "dense dates", scaletest.Size(64<<10), mask)
}

func TestZeroWidthInsideNameMasksAndRestoresExactly(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	in := "Then Jo\u200bhn Smith called."
	out := maskRoundTrip(t, e, in)
	if strings.Contains(out, "Smith") || strings.Contains(out, "Jo") || strings.ContainsRune(out, '\u200b') {
		t.Fatalf("got %q", out)
	}
	for _, w := range []string{"Jo", "hn"} {
		for _, k := range []string{"person.given", "person.family", "person.nick", "person.residue"} {
			if _, ok := e.Vault().Get(k, w); ok {
				t.Fatalf("stray %s record for %q", k, w)
			}
		}
	}
}

func TestLowercaseCommonFamilyNameAcrossRequests(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	maskRoundTrip(t, e, "Anna Gross joined.")
	if _, ok := e.Vault().Get("person.family", "Gross"); !ok {
		t.Fatal("seed not learned")
	}
	for _, in := range []string{"that is gross", "gross margin was low", "The gross total"} {
		s := e.NewSession("p")
		out, err := s.Mask(ctx, in)
		if err != nil || out != in {
			t.Errorf("%q -> %q %v", in, out, err)
		}
		if err := s.Err(); err != nil {
			t.Errorf("%q blocked: %v", in, err)
		}
		c := e.NewSession("p")
		c.Check(ctx, in)
		if err := c.Err(); err != nil {
			t.Errorf("%q Check blocked: %v", in, err)
		}
	}
	for _, in := range []string{"Anna Gross called again.", "Ms. Gross replied.", "Gross said yes.", "please ask gross about it", "anna gross joined"} {
		if out := maskRoundTrip(t, e, in); hasWord(out, "Gross") || hasWord(out, "gross") {
			t.Errorf("%q -> %q", in, out)
		}
	}
}

func TestLowercaseCommonGivenNamesAcrossRequests(t *testing.T) {
	e := seededGateEngine(t)
	if out := maskRoundTrip(t, e, "hey will, can you look?"); hasWord(out, "will") {
		t.Errorf("greeted lowercase name kept: %q", out)
	}
	for _, in := range []string{"the will of the people", "we will see", "a rose garden", "mark the box", "pay the bill"} {
		if out := maskRoundTrip(t, e, in); out != in {
			t.Errorf("%q -> %q", in, out)
		}
	}
}

func TestWillAndMessageHeaders(t *testing.T) {
	e := seededGateEngine(t)
	for in, names := range map[string][]string{
		"Will said yes":       {"Will"},
		"Will, can you look?": {"Will"},
		"Will Turner is here": {"Will"},
		"Rose to Mark: hi":    {"Rose", "Mark"},
		"Grant to Rose: done": {"Grant", "Rose"},
	} {
		out := maskRoundTrip(t, e, in)
		for _, n := range names {
			if hasWord(out, n) {
				t.Errorf("%q -> %q", in, out)
			}
		}
	}
}
