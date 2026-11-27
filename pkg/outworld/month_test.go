package outworld

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

var monthKept = []string{
	"by the end of June", "end of June", "the end of June release", "the June release", "Q2 (April-June)",
	"from March to June", "June-July", "June/July timeline", "Due June.", "deadline is June", "by June",
	"the first week of June", "in June", "in May", "since May", "until June", "early June", "late May", "mid-June",
	"next June", "last May", "this June", "every May", "before June", "after May", "around June", "during May",
	"through June", "June 5", "May 2024", "June 12, 2025", "May, June and July", "May and June are busy months",
	"May or June", "April through June", "June to August", "May–June", "the June timeline", "our June launch",
	"the May sprint", "June release notes", "the June newsletter", "the May update", "Release Notes For June",
	"June 2026 roadmap", "the June cohort", "by the end of May", "as of June", "the June 30 deadline",
	"Q3 starts after June", "Planned for late June", "May 1st", "June/July", "May-June window", "the June meeting",
	"from May until June", "the week of June 9", "Taxes are due in May.",
	"May was busy", "May is busy", "May will be busy", "May has been busy", "June was hot", "June is hot", "June will be hot", "June has been hot", "May was quiet", "May is quiet", "May will be quiet", "May has been quiet", "June was slow", "June is slow", "June will be slow", "June has been slow", "May was great", "May is great", "May will be great", "May has been great", "June was rainy", "June is rainy", "June will be rainy", "June has been rainy", "May was short", "May is short", "May will be short", "May has been short", "June was long", "June is long", "June will be long", "June has been long", "May was cold", "May is cold", "May will be cold", "May has been cold", "June was warm", "June is warm", "June will be warm", "June has been warm", "May was hectic", "May is hectic", "May will be hectic", "May has been hectic", "June was fun", "June is fun", "June will be fun", "June has been fun",
	"the release is June",
}

var monthPeople = []string{
	"ask June", "June said yes", "loop in June", "reviewed by June", "I told June", "Let June know.",
	"Waiting on June.", "Blocked on June", "approved by June", "from June", "Notes From June",
	"end of June's contract", "June's laptop broke", "ping June", "cc June", "thanks June", "June, can you review?",
	"talk to June", "June and I will join", "May and June went to lunch.", "written by June", "May said no",
	"Who is June?", "When is June back?", "Where is June", "How is June", "Why is June", "What is June",
	"Who is May?", "When is May back from leave?", "Is June free?", "June is sick", "May was out", "June will be away",
	"May has been on leave", "June is back", "May was late",
}

func TestMonthPhrasesOnSeededVault(t *testing.T) {
	e := seededGateEngine(t)
	if _, ok := e.Vault().Get("person.given", "June"); !ok {
		t.Fatal("June not in vault")
	}
	if _, ok := e.Vault().Get("person.given", "May"); !ok {
		t.Fatal("May not in vault")
	}
	fresh := newEngine(t, func(o *Options) { o.Dictionary = nil })
	wrong := 0
	for _, in := range monthKept {
		if out, want := maskRoundTrip(t, e, in), maskRoundTrip(t, fresh, in); out != want {
			wrong++
			t.Errorf("kept: %q -> %q", in, out)
		}
	}
	for _, in := range monthPeople {
		if out := maskRoundTrip(t, e, in); hasWord(out, "June") || hasWord(out, "May") {
			wrong++
			t.Errorf("masked: %q -> %q", in, out)
		}
	}
	if out := maskRoundTrip(t, e, "in June, May and I went"); !strings.HasPrefix(out, "in June, ") || hasWord(out, "May") {
		wrong++
		t.Errorf("coordinated with I: %q", out)
	}
	t.Logf("%d kept, %d people, %d wrong", len(monthKept), len(monthPeople), wrong)
}

func TestCamelCaseHandleEveryOrder(t *testing.T) {
	forms := []string{`username: "ElenaPetrova"`, `username: “ElenaPetrova”`, `handle: ElenaPetrova`, `user: ElenaPetrova again`, `@ElenaPetrova`, `ElenaPetrova`}
	for first := range forms[:len(forms)-1] {
		e := newEngine(t, func(o *Options) { o.Dictionary = nil })
		maskRoundTrip(t, e, "Elena Petrova joined.")
		order := append([]string{forms[first]}, append(append([]string(nil), forms[:first]...), forms[first+1:]...)...)
		for _, in := range order {
			s := e.NewSession("p")
			out, err := s.Mask(ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			if hasWord(out, "ElenaPetrova") {
				t.Errorf("order %d: %q -> %q", first, in, out)
			}
			if s.leavesKnownReal(out) {
				t.Errorf("order %d: leak re-scan still sees a real name in %q", first, out)
			}
			if err := s.Err(); err != nil {
				t.Errorf("order %d: %q blocked: %v", first, in, err)
			}
		}
	}
}

func TestMemoMissesAfterDateIsRehydrated(t *testing.T) {
	e := newEngine(t, nil)
	if out, err := e.NewSession("gaps").Mask(ctx, "Done (3/4)"); err != nil || out != "Done (3/4)" {
		t.Fatalf("fraction-like text changed before any date state: %q %v", out, err)
	}
	s := e.NewSession("gaps")
	masked, err := s.Mask(ctx, "Meeting on 3/4 with the team.")
	if err != nil {
		t.Fatal(err)
	}
	stable, _ := e.NewSession("gaps").Mask(ctx, "Meeting on 3/4 with the team.")
	if stable != masked {
		t.Fatalf("byte stability lost: %q then %q", masked, stable)
	}
	s.Rehydrate("Done (" + strings.Fields(masked)[2] + ")")
	if out, _ := e.NewSession("gaps").Mask(ctx, "Done (3/4)"); strings.Contains(out, "3/4") {
		t.Fatalf("memo returned the raw date: %q", out)
	}
	if again, _ := e.NewSession("gaps").Mask(ctx, "Meeting on 3/4 with the team."); again != masked {
		t.Fatalf("unrelated entry changed bytes: %q then %q", masked, again)
	}
}

func TestTombstoneTableEvictsOldest(t *testing.T) {
	b := newDateBook()
	for i := 0; i <= maxDateTombstones; i++ {
		var k [32]byte
		k[0], k[1], k[2] = byte(i), byte(i>>8), byte(i>>16)
		b.tombs[k] = b.tombList.PushFront(k)
	}
	for b.tombList.Len() > maxDateTombstones {
		old := b.tombList.Back()
		b.tombList.Remove(old)
		delete(b.tombs, old.Value.([32]byte))
	}
	if got := b.lookup("never-seen", "3/4", true); got.saturated {
		t.Fatal("a scope missing from the tombstone table must count as new")
	}
}

func TestMemoHitsInSaturatedScopeWithoutNewDates(t *testing.T) {
	e := newEngine(t, nil)
	for i := 0; i <= maxRealDates; i++ {
		e.realDates.add("sat", fmt.Sprintf("real %d", i), true)
	}
	in := "Meeting on March 3, 2024 with the team."
	first, err := e.NewSession("sat").Mask(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	s := e.NewSession("sat")
	again, err := s.Mask(ctx, in)
	if err != nil || again != first || s.MemoHits() == 0 {
		t.Fatalf("saturated scope with no new date state missed the memo: %q %q hits=%d", first, again, s.MemoHits())
	}
}

func TestNameFormsRestore(t *testing.T) {
	for in, want := range map[string]string{
		"Then Mary\u2014Jane Smith called.":                           "Then Mary\u2014Jane Smith called.",
		"Then Mary\u00a0Smith called.":                                "Then Mary\u00a0Smith called.",
		"Then \uff2d\uff41\uff52\uff59 Smith called.":                 "Then \uff2d\uff41\uff52\uff59 Smith called.",
		"Then \U0001d40c\U0001d41a\U0001d42b\U0001d432 Smith called.": "Then \U0001d40c\U0001d41a\U0001d42b\U0001d432 Smith called.",
	} {
		e := newEngine(t, func(o *Options) { o.Dictionary = nil })
		s := e.NewSession("p")
		out, err := s.Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, "Smith") || strings.Contains(out, "Mary") {
			t.Fatalf("%q -> %q", in, out)
		}
		if back := s.Rehydrate(out); back != want {
			t.Errorf("%q -> %q -> %q, want %q", in, out, back, want)
		}
		plain, _ := e.NewSession("p").Mask(ctx, "Then Mary Smith called.")
		later := e.NewSession("p")
		if back := later.Rehydrate(plain); back != "Then Mary Smith called." {
			t.Errorf("plain mention after %q rehydrates as %q", in, back)
		}
	}
}

func TestFullWidthNameRestoresExactly(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	s := e.NewSession("p")
	out, err := s.Mask(ctx, "Notes from Ｍａｒｋ Jones today.")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "Ｍａｒｋ") || strings.Contains(out, "Jones") {
		t.Fatalf("got %q", out)
	}
	if back := s.Rehydrate(out); back != "Notes from Ｍａｒｋ Jones today." {
		t.Fatalf("rehydrated %q", back)
	}
	later := e.NewSession("p")
	m, _ := later.Mask(ctx, "Mark said ok")
	if back := later.Rehydrate(m); back != "Mark said ok" {
		t.Fatalf("plain mention rehydrated as %q", back)
	}
}

func TestWillPolicyIsListIndependent(t *testing.T) {
	e := seededGateEngine(t)
	for _, in := range []string{"Will review the doc.", "Will send it.", "Will call later.", "Will update it.", "Will merge it.", "Will handle it."} {
		if out := maskRoundTrip(t, e, in); out != in {
			t.Errorf("%q -> %q", in, out)
		}
	}
	for _, in := range []string{"Will reviews the doc.", "Will reviewed it.", "Will said yes.", "Will, can you look?", "Will: on it", "Will Turner is here.", "Will's PR is ready.", "Will has the keys", "Will from accounting"} {
		if out := maskRoundTrip(t, e, in); hasWord(out, "Will") {
			t.Errorf("%q -> %q", in, out)
		}
	}
}

func TestWillTitleCaseHeadings(t *testing.T) {
	e := seededGateEngine(t)
	for _, in := range []string{"# Will It Scale?", "Will It Scale?", "Title: Will It Scale", "## Will It Blend", "Will This Work"} {
		if out := maskRoundTrip(t, e, in); out != in {
			t.Errorf("%q -> %q", in, out)
		}
	}
	for _, in := range []string{"Will Turner is here.", "Will Smith Joins The Team", "Will, It Works"} {
		if out := maskRoundTrip(t, e, in); hasWord(out, "Will") {
			t.Errorf("%q -> %q", in, out)
		}
	}
}

func TestURLPathDatesShiftAgain(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("urls")
	in := "Born 1985-03-14, record at https://example.com/patients/x/1985-03-14 today."
	out, err := s.Mask(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "1985-03-14") {
		t.Fatalf("raw date left: %q", out)
	}
	if back := s.Rehydrate(out); back != in {
		t.Fatalf("rehydrate %q -> %q", out, back)
	}
}

func TestVersionLineSurvivesEcho(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	s := e.NewSession("ver")
	in := "Mary Jones sent curl -H 'anthropic-version: 2023-06-01' https://api.example.com/v1/messages"
	out, err := s.Mask(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "anthropic-version: 2023-06-01") || strings.Contains(out, "Mary") {
		t.Fatalf("got %q", out)
	}
	if back := s.Rehydrate(out); back != in {
		t.Fatalf("rehydrate %q -> %q", out, back)
	}
	reply := "Use anthropic-version: 2023-06-01 as shown."
	if back := s.Rehydrate(reply); back != reply {
		t.Fatalf("echoed version line un-shifted: %q", back)
	}
	later := e.NewSession("ver")
	if again, err := later.Mask(ctx, reply); err != nil || again != reply || later.Err() != nil {
		t.Fatalf("echoed version line changed: %q %v %v", again, err, later.Err())
	}
}

func TestMemoUnderConcurrentDateLearning(t *testing.T) {
	e := newEngine(t, nil)
	const n = 40
	dates := make([]string, n)
	for i := range dates {
		dates[i] = fmt.Sprintf("%d/%d", 1+i%12, 1+i/12)
	}
	var wg sync.WaitGroup
	var raw atomic.Int64
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, d := range dates {
			e.realDates.addRehydrated("conc", d)
		}
	}()
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for round := 0; round < 30; round++ {
				d := dates[(w*7+round)%n]
				known := e.realDates.lookup("conc", d, true).rehydrated
				out, err := e.NewSession("conc").Mask(ctx, "Done ("+d+")")
				if err != nil {
					t.Error(err)
					return
				}
				if known && strings.Contains(out, "("+d+")") {
					raw.Add(1)
				}
			}
		}(w)
	}
	wg.Wait()
	if r := raw.Load(); r != 0 {
		t.Fatalf("%d re-sends carried a known date raw", r)
	}
}

func TestMemoStoresTheGenerationFromBeforeMasking(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("gen")
	text := "Done (3/4)"
	mk := s.memoMark()
	e.realDates.addRehydrated("gen", "3/4")
	s.memoStore(text, text, nil, mk)
	if out, _ := e.NewSession("gen").Mask(ctx, text); strings.Contains(out, "3/4") {
		t.Fatalf("entry stored during date learning came back raw: %q", out)
	}
}

func TestSaturatedScopeMemoHitRate(t *testing.T) {
	e := newEngine(t, nil)
	for i := 0; i <= maxRealDates; i++ {
		e.realDates.add("satrate", fmt.Sprintf("real %d", i), true)
	}
	var texts []string
	for i := 0; i < 50; i++ {
		texts = append(texts, fmt.Sprintf("Visit %d on March %d, 2024 and the review on 4/%d.", i, 1+i%28, 1+i%28))
	}
	for _, in := range texts {
		if _, err := e.NewSession("satrate").Mask(ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	hits, total := 0, 0
	for round := 0; round < 10; round++ {
		r := e.NewSession("satrate")
		r.Rehydrate(fmt.Sprintf("Booked on June %d, 2025.", 1+round))
		s := e.NewSession("satrate")
		for _, in := range texts {
			if _, err := s.Mask(ctx, in); err != nil {
				t.Fatal(err)
			}
			total++
		}
		hits += s.MemoHits()
	}
	rate := float64(hits) / float64(total)
	t.Logf("saturated scope memo hit rate %.1f%% (%d of %d)", 100*rate, hits, total)
	if rate < 0.85 {
		t.Fatalf("saturated scope hit rate %.1f%%", 100*rate)
	}
}

func TestOldFullWidthRecordRehydratesPlain(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	sur, err := e.Vault().Assign("person.given", "Ｍａｒｋ", func(int) string { return "Cedric" })
	if err != nil {
		t.Fatal(err)
	}
	s := e.NewSession("p")
	if back := s.Rehydrate(sur + " said ok"); back != "Mark said ok" {
		t.Fatalf("old full-width record rehydrated as %q", back)
	}
}

func TestMonthCopulaAndQuestions(t *testing.T) {
	e := seededGateEngine(t)
	fresh := newEngine(t, func(o *Options) { o.Dictionary = nil })
	for _, in := range []string{"June is hot", "May was busy", "June is in summer", "June is in Q2", "May is in spring",
		"What month is June?", "Which month is June?", "The launch is June, right?", "The kickoff is June?", "Is it June yet?"} {
		if out, want := maskRoundTrip(t, e, in), maskRoundTrip(t, fresh, in); out != want {
			t.Errorf("kept: %q -> %q", in, out)
		}
	}
	for _, in := range []string{"June is on PTO", "June is on vacation", "June is on holiday", "June is OOO", "June is traveling",
		"June is in Paris", "June is with a client", "June is it done?", "Will to send the deck", "// Will to fix", "Will is it done?"} {
		name := "June"
		if strings.Contains(in, "Will") {
			name = "Will"
		}
		if out := maskRoundTrip(t, e, in); hasWord(out, name) {
			t.Errorf("masked: %q -> %q", in, out)
		}
	}
	maskRoundTrip(t, fresh, "June is out of stock")
	if _, ok := fresh.Vault().Get("person.given", "June"); ok {
		t.Fatal("learned June from a stock notice")
	}
}

func TestVersionLabelledRealDateShifts(t *testing.T) {
	e := newEngine(t, nil)
	seed := e.NewSession("ver")
	masked, err := seed.Mask(ctx, "Patient born 1985-03-14 at home.")
	if err != nil || strings.Contains(masked, "1985-03-14") {
		t.Fatalf("seed: %q %v", masked, err)
	}
	shifted := strings.Fields(masked)[2]
	s := e.NewSession("ver")
	in := "config api-version: 1985-03-14 loaded"
	out, err := s.Mask(ctx, in)
	if err != nil || strings.Contains(out, "1985-03-14") || s.Err() != nil {
		t.Fatalf("known real date under a version label went out raw: %q %v %v", out, err, s.Err())
	}
	if back := s.Rehydrate(out); back != in {
		t.Fatalf("rehydrate %q -> %q", out, back)
	}
	both := "anthropic-version: 2023-06-01 and we met on 2023-06-01."
	if out, _ := e.NewSession("ver").Mask(ctx, both); strings.Contains(out, "2023-06-01") {
		t.Fatalf("date repeated outside the label went out raw: %q", out)
	}
	r := e.NewSession("ver")
	if back := r.Rehydrate("set version: " + shifted); back != "set version: 1985-03-14" {
		t.Fatalf("model echo of a shifted real date not restored: %q", back)
	}
	echo := "Use anthropic-version: 2023-06-01 as shown."
	if back := r.Rehydrate(echo); back != echo {
		t.Fatalf("plain version echo changed: %q", back)
	}
	if again, err := e.NewSession("ver-echo").Mask(ctx, echo); err != nil || again != echo {
		t.Fatalf("plain version line changed: %q %v", again, err)
	}
}

func TestVersionLabelledDateMatchesByCalendarDay(t *testing.T) {
	forms := []string{"March 14, 1985", "03/14/1985", "14 March 1985", "1985-03-14"}
	for _, dob := range forms {
		for _, order := range []bool{true, false} {
			e := newEngine(t, nil)
			in := "Patient born " + dob + ". Build version: 1985-03-14 deployed."
			if !order {
				in = "Build version: 1985-03-14 deployed. Patient born " + dob + "."
			}
			s := e.NewSession("day")
			out, err := s.Mask(ctx, in)
			if err != nil || strings.Contains(out, "1985-03-14") || strings.Contains(out, "1985") && strings.Contains(out, dob) {
				t.Errorf("same message %q -> %q %v", in, out, err)
			}
			if back := s.Rehydrate(out); back != in {
				t.Errorf("rehydrate %q -> %q", out, back)
			}
		}
		e := newEngine(t, nil)
		seed := e.NewSession("day")
		masked, err := seed.Mask(ctx, "Patient born "+dob+".")
		if err != nil {
			t.Fatal(err)
		}
		later := e.NewSession("day")
		out, err := later.Mask(ctx, "app version: 1985-03-14")
		if err != nil || strings.Contains(out, "1985-03-14") {
			t.Errorf("earlier-turn %q then version: %q %v", masked, out, err)
		}
		r := e.NewSession("day")
		shiftedISO := strings.TrimPrefix(out, "app version: ")
		if back := r.Rehydrate("set version: " + shiftedISO); back != "set version: 1985-03-14" {
			t.Errorf("echo of shifted ISO for %q restored as %q", dob, back)
		}
	}
}

func TestWillCoordinatedInTitleLine(t *testing.T) {
	e := seededGateEngine(t)
	for _, in := range []string{"Will And Sarah Review", "Will and Sarah Review", "Sarah & Will Review"} {
		if out := maskRoundTrip(t, e, in); hasWord(out, "Will") || hasWord(out, "Sarah") {
			t.Errorf("%q -> %q", in, out)
		}
	}
	for _, in := range []string{"Will It Scale?", "Will This Work"} {
		if out := maskRoundTrip(t, e, in); out != in {
			t.Errorf("%q -> %q", in, out)
		}
	}
}
