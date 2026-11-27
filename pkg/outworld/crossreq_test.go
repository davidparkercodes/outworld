package outworld

import (
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/detect"
)

func TestRehydratedWeakDateIsShiftedAgain(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("gaps")
	if _, err := s.Mask(ctx, "Dentist on March 3, 2024, then may 4 for the cleaning."); err != nil {
		t.Fatal(err)
	}
	reply := "Sure: june 13 works."
	back := s.Rehydrate(reply)
	if back == reply {
		t.Fatalf("reply date not restored: %q", back)
	}
	again := maskRoundTrip(t, e, back)
	if again != reply {
		t.Fatalf("real date went out again: %q -> %q, want %q", back, again, reply)
	}
	other := e.NewSession("other")
	if out, err := other.Mask(ctx, back); err != nil || out != back {
		t.Fatalf("another scope changed: %q %v", out, err)
	}
	if out := maskRoundTrip(t, e, "Upgrade to version 2 on host 13."); out != "Upgrade to version 2 on host 13." {
		t.Fatalf("unrelated numbers changed: %q", out)
	}
}

func initialsOf(name string) string {
	var b strings.Builder
	for _, w := range strings.Fields(name) {
		b.WriteString(strings.ToUpper(w[:1]))
	}
	return b.String()
}

func TestInitialismFollowsPersonSurrogate(t *testing.T) {
	e := newEngine(t, nil)
	out := maskRoundTrip(t, e, "Laura Diaz joined. LD said the build is green.")
	name, rest, _ := strings.Cut(out, " joined. ")
	ini := initialsOf(name)
	if detect.CommonInitialism(ini) || rest != ini+" said the build is green." {
		t.Fatalf("got %q, want initials %q", out, ini)
	}
	if later := maskRoundTrip(t, e, "LD said hi."); later != ini+" said hi." {
		t.Fatalf("later request: %q, want %q", later, ini)
	}
}

func TestInitialismCollidingWithCommonOneStaysLinkedAndSafe(t *testing.T) {
	e := newEngine(t, nil)
	out := maskRoundTrip(t, e, "Kevin Chen joined. KC said the build is green.")
	name, rest, _ := strings.Cut(out, " joined. ")
	got, _, _ := strings.Cut(rest, " ")
	if got == "KC" || detect.CommonInitialism(got) || len(got) != 2 {
		t.Fatalf("got %q", out)
	}
	if want := initialsOf(name); !detect.CommonInitialism(want) && got != want {
		t.Fatalf("initials %q, want %q", got, want)
	}
	if later := maskRoundTrip(t, e, "KC said hi."); later != got+" said hi." {
		t.Fatalf("later request: %q, want %q", later, got)
	}
}

func TestPanickingLearnedDetectorFailsClosed(t *testing.T) {
	s := newEngine(t, nil).NewSession("p")
	s.learned = &detect.DictionaryDetector{}
	if out, err := s.Mask(ctx, "hello there"); err == nil || out != "" {
		t.Fatalf("got %q, %v", out, err)
	}
}

func TestAccentedHandlesAcrossRequests(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	seed := maskRoundTrip(t, e, "François Lefèvre joined")
	full, _, _ := strings.Cut(seed, " joined")
	accented := strings.ToLower(strings.ReplaceAll(full, " ", "."))
	first := maskRoundTrip(t, e, "mail francois.lefevre")
	want := strings.TrimPrefix(first, "mail ")
	if want == "francois.lefevre" || want == accented {
		t.Fatalf("ASCII handle not given its own stand-in: %q", first)
	}
	if out := maskRoundTrip(t, e, "ask francois.lefevre"); out != "ask "+want {
		t.Fatalf("ASCII handle stand-in not stable: %q, want %q", out, "ask "+want)
	}
	out := maskRoundTrip(t, e, "mail francois.lefevre@corp.com")
	if !strings.HasPrefix(out, "mail "+accented+"@") {
		t.Fatalf("email not linked: %q", out)
	}
	if out := maskRoundTrip(t, e, "user francoislefevre logged in"); out != "user "+strings.ReplaceAll(accented, ".", "")+" logged in" {
		t.Fatalf("joined handle: %q", out)
	}
}

func TestJoinedUsernameFromVaultRecords(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	seed := maskRoundTrip(t, e, "John Smith joined")
	full, _, _ := strings.Cut(seed, " joined")
	joined := strings.ToLower(strings.ReplaceAll(full, " ", ""))
	for in, exp := range map[string]string{
		"user johnsmith logged in": "user " + joined + " logged in",
		"account: johnsmith":       "account: " + joined,
		"@johnsmith ping":          "@" + joined + " ping",
		"cd /home/johnsmith/src":   "cd /home/" + joined + "/src",
	} {
		if out := maskRoundTrip(t, e, in); out != exp {
			t.Fatalf("%q -> %q, want %q", in, out, exp)
		}
	}
	for _, in := range []string{"user admin logged in"} {
		if out := maskRoundTrip(t, e, in); out != in {
			t.Fatalf("%q -> %q", in, out)
		}
	}
}
