package detect

import (
	"strings"
	"testing"
)

func TestOrdinaryWordUse(t *testing.T) {
	ordinary := []string{
		"Mark as deleted", "Please mark the box", "mark the box", "Grant access to the repo", "A rose by any name",
		"// Mark's position", "Mark down the price", "Mark my words", "Rose to the occasion", "Prices rose and fell",
		"The rose and the lily", "Mark and sweep GC", "Mark the date", "Grant writing tips", "Rose gold iPhone",
		"I will go", "in May", "the rose garden", "pay the bill", "Grace period", "Will this work?", "Will it compile?",
		"We will ship it", "Free will is debated", "The will was read aloud", "Bill of materials", "Will we need a migration?",
		"Will there be tests?", "Will you check?", "in May", "May, June and July", "Release Notes For June", "Status: WILL FIX",
		"WILL NOT FIX", "GRANT ACCESS TO ADMIN", "Rose to the occasion", "Rose to prominence",
		"Will fix later", "Will check the logs.", "Will do.", "Will be done soon.", "Will this work.", "Will merge it",
		"WILL NOT FIX", "Hope you are well.", "Rose 3% overnight", "- [ ] Mark complete", "Will-o'-the-wisp",
		"The Will to Power",
	}
	person := []string{
		"Let Will know.", "Can Will join?", "Does Will agree?", "Is Will free?", "Will (accounting)", "Will -- see above",
		"I'll loop in Will tomorrow.", "DM Will", "Give Will a call", "Cheers, Will", "-- Will", "Will wants the deck",
		"Will has the keys", "Will got back to me", "Will was there", "<Will> ok", "Will?", "Will from accounting",
		"Will on the design team", "Sarah, Will and Raj", "* Will, can you review?", "# Will, status?", "## Mark's notes",
		"-- Will, can you review?", "WILL wants the deck", "Let WILL know.", "Mark's laptop broke",
		"Thanks Will", "Will's PR is ready", "WILL: on it", "Hope said yes", "- Mark", "- [ ] Mark to review the doc",
		"Hope, are you free?", "# Mark, as noted", "Rose to send the deck", "- Rose to send deck", "Rose to VP last year.",
		"Rose by Friday", "// Rose to fix", "Bill for the hours", "Will said yes", "Will, can you look?", "Will Turner called", "Rose to Mark: hi",
		"I'll loop in June tomorrow.", "Waiting on June.",
		"Blocked on June", "Reviewed by June", "approved by June", "from June", "Notes From June", "PING WILL",
	}
	names := []string{"Will", "WILL", "will", "Mark", "mark", "Grant", "Rose", "rose", "Grace", "Bill", "bill", "May", "Hope", "June", "GRANT"}
	find := func(text string) (int, int) {
		for _, tok := range strings.FieldsFunc(text, func(r rune) bool {
			return r == ' ' || r == '?' || r == ',' || r == '.' || r == '(' || r == '<' || r == '>' || r == '\'' || r == '-' || r == ':' || r == '%'
		}) {
			for _, n := range names {
				if tok == n {
					i := strings.Index(text, n)
					for i >= 0 && ((i > 0 && isLetter(text[i-1])) || (i+len(n) < len(text) && isLetter(text[i+len(n)]))) {
						j := strings.Index(text[i+1:], n)
						if j < 0 {
							i = -1
							break
						}
						i += j + 1
					}
					return i, i + len(n)
				}
			}
		}
		return -1, -1
	}
	for _, text := range ordinary {
		s, e := find(text)
		if s < 0 || !OrdinaryWordUse(text, s, e) {
			t.Fatalf("%q at %d: want ordinary use", text, s)
		}
	}
	for _, text := range person {
		s, e := find(text)
		if s < 0 || OrdinaryWordUse(text, s, e) {
			t.Fatalf("%q at %d: want person use", text, s)
		}
	}
}

func isLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

func TestUsernamePositions(t *testing.T) {
	cases := map[string]string{
		"user johnsmith logged in": "johnsmith",
		"login: jdoe42 failed":     "jdoe42",
		"cd /home/kchen/src":       "kchen",
		"ping @mlopez now":         "mlopez",
		"user admin logged in":     "",
		"the account is closed":    "",
	}
	for text, want := range cases {
		var got []string
		for _, p := range UsernamePositions(text) {
			got = append(got, text[p[0]:p[1]])
		}
		if strings.Join(got, ",") != want {
			t.Fatalf("%q: got %v, want %q", text, got, want)
		}
	}
}

func TestUsernamePositionsQuoted(t *testing.T) {
	for text, want := range map[string]string{
		`username = "maryjones"`:    "maryjones",
		`user "johnsmith"`:          "johnsmith",
		`{"username": "maryjones"}`: "maryjones",
		"login: `kchen`":            "kchen",
	} {
		ps := UsernamePositions(text)
		if len(ps) != 1 || text[ps[0][0]:ps[0][1]] != want {
			t.Fatalf("%q: %v", text, ps)
		}
	}
}

func TestCoordinatedMonthPairs(t *testing.T) {
	for text, want := range map[string]bool{
		"May and June went to lunch":   false,
		"Then May and June met Raj.":   false,
		"May, June and July":           true,
		"May and June are busy months": true,
		"Sales in May and June rose":   true,
	} {
		for _, w := range []string{"May", "June"} {
			i := strings.Index(text, w)
			if got := OrdinaryWordUse(text, i, i+len(w)); got != want {
				t.Errorf("%q %s: got %v, want %v", text, w, got, want)
			}
		}
	}
}
