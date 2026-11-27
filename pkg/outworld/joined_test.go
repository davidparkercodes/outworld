package outworld

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/detect"
	"github.com/davidparkercodes/outworld/pkg/entity"
)

func noRealPieces(t *testing.T, out string, name string) {
	t.Helper()
	low := textutil.Fold(out)
	for _, w := range strings.Fields(name) {
		if f := textutil.Fold(w); strings.Contains(low, f) {
			t.Fatalf("%q keeps real %q", out, f)
		}
	}
}

func TestAccentedJoinedHandleLearned(t *testing.T) {
	for _, c := range []struct{ name, handle string }{
		{"José Martinez", "josemartinez"},
		{"Renée Dubois", "reneedubois"},
		{"Zoë Müller", "zoemuller"},
		{"Søren Østergaard", "sorenostergaard"},
	} {
		in := c.name + " wrote it. ping " + c.handle
		out := maskRoundTrip(t, newEngine(t, nil), in)
		noRealPieces(t, out, c.name)
		full, handle, ok := strings.Cut(out, " wrote it. ping ")
		if !ok {
			t.Fatalf("%q: shape changed: %q", in, out)
		}
		if want := strings.ToLower(strings.ReplaceAll(full, " ", "")); handle != want {
			t.Fatalf("%q: handle %q, want full joined %q", in, handle, want)
		}
	}
}

func TestAccentedJoinedHandleDictionary(t *testing.T) {
	withJose := func(o *Options) {
		o.Dictionary.Entries = append(o.Dictionary.Entries, detect.Entry{Value: "José Martinez", Type: entity.Person})
	}
	for _, c := range []struct{ in, prefix, suffix string }{
		{"We hired JoseMartinez last week", "We hired ", " last week"},
		{"user josemartinez logged in", "user ", " logged in"},
		{"cd /home/josemartinez/src", "cd /home/", "/src"},
	} {
		e := newEngine(t, withJose)
		out := maskRoundTrip(t, e, c.in)
		noRealPieces(t, out, "José Martinez")
		g, _ := e.Vault().Get("person.given", "José")
		f, _ := e.Vault().Get("person.family", "Martinez")
		if g == "" || f == "" {
			t.Fatalf("%q: name records missing", c.in)
		}
		mid := strings.TrimSuffix(strings.TrimPrefix(out, c.prefix), c.suffix)
		if strings.ToLower(mid) != strings.ToLower(g+f) {
			t.Fatalf("%q -> %q, want full joined %q", c.in, out, strings.ToLower(g+f))
		}
	}
}

func TestUnaccentedJoinedHandlesAreByteStable(t *testing.T) {
	for in, want := range map[string]string{
		"John Smith wrote it. ping johnsmith": "Cillian Garrity wrote it. ping cilliangarrity",
		"We hired JohnSmith last week":        "We hired Cilliangarrity last week",
	} {
		if out := maskRoundTrip(t, newEngine(t, nil), in); out != want {
			t.Fatalf("%q -> %q, want %q", in, out, want)
		}
	}
}

func TestPartlyReplacedJoinedTokenIsBlocked(t *testing.T) {
	e := newEngine(t, nil)
	if _, err := e.Vault().Assign("person.joined", "josemartinez", func(int) string { return "aidemartinez" }); err != nil {
		t.Fatal(err)
	}
	s := e.NewSession("p")
	out, err := s.Mask(ctx, "José Martinez wrote it. ping josemartinez")
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(s.Err(), ErrLeak) {
		t.Fatalf("partial handle surrogate not flagged: %q", out)
	}
	for _, v := range s.Violations() {
		if strings.Contains(strings.ToLower(v.Preview), "martinez") {
			t.Fatalf("preview shows the value: %q", v.Preview)
		}
	}
}

func TestKeepsRealPiece(t *testing.T) {
	e := newEngine(t, nil)
	sp := func(text, canon string) entity.Span {
		return entity.Span{Text: text, Type: entity.Person, Canonical: canon}
	}
	for _, c := range []struct {
		sp   entity.Span
		repl string
		want bool
	}{
		{sp("josemartinez", "José Martinez"), "aidemartinez", true},
		{sp("reneedubois", "Renée Dubois"), "peteedubois", true},
		{sp("josemartinez", "José Martinez"), "aidenrasmussen", false},
		{sp("johnsmith", "John Smith"), "cilliangarrity", false},
		{sp("jmartinez", "José Martinez"), "arasmussen", false},
		{sp("José Martinez", "José Martinez"), "Aiden Martinez", false},
		{sp("blacksmith", ""), "blacksmith", false},
		{sp("francois.lefevre", "François Lefèvre"), "kenjcois.santangelo", true},
		{sp("jose.martinez", "José Martinez"), "lude.martinez", true},
		{sp("jose.martinez", "José Martinez"), "aiden.rasmussen", false},
		{sp("john.smith42", "John Smith"), "cillian.garrity42", false},
	} {
		if got := e.keepsRealPiece(c.sp, c.repl); got != c.want {
			t.Fatalf("%q -> %q: got %v", c.sp.Text, c.repl, got)
		}
	}
}

type panicDetector struct{}

func (panicDetector) Name() string { return "panic" }

func (panicDetector) Detect(context.Context, string) ([]entity.Span, error) { panic("boom") }

func TestPanickingDetectorFailsMaskClosed(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Extra = []detect.Detector{panicDetector{}} })
	for _, in := range []string{"John Smith called 555-0100.", strings.Repeat("John Smith called again. ", 420)} {
		s := e.NewSession("p")
		out, err := s.Mask(ctx, in)
		if err == nil {
			t.Fatalf("len %d: no error, got %d bytes out", len(in), len(out))
		}
		if out != "" {
			t.Fatalf("len %d: text returned with the error", len(in))
		}
	}
}

func TestCommonWordVaultNameNeedsPersonUse(t *testing.T) {
	e := newEngine(t, nil)
	for w, sur := range map[string]string{"Mark": "Cedric", "Grant": "Osmund"} {
		if _, err := e.Vault().Assign("person.given", w, func(int) string { return sur }); err != nil {
			t.Fatal(err)
		}
	}
	for _, in := range []string{
		"Mark as deleted. Please mark the box.",
		"// Mark's position",
		"Grant access to the bucket.",
	} {
		if out := maskRoundTrip(t, e, in); out != in {
			t.Fatalf("%q -> %q", in, out)
		}
	}
	if fresh := maskRoundTrip(t, newEngine(t, nil), "Mark as deleted. Please mark the box."); fresh != "Mark as deleted. Please mark the box." {
		t.Fatalf("fresh vault changed it: %q", fresh)
	}
}

func TestRealPersonWithCommonWordNameStaysMasked(t *testing.T) {
	e := newEngine(t, nil)
	first := maskRoundTrip(t, e, "Mark Jones joined the call.")
	if strings.Contains(first, "Mark") || strings.Contains(first, "Jones") {
		t.Fatalf("full name kept: %q", first)
	}
	for _, in := range []string{"Mark said yes.", "Please ask Mark about it.", "Mark Jones signed off."} {
		if out := maskRoundTrip(t, e, in); strings.Contains(out, "Mark") {
			t.Fatalf("%q -> %q", in, out)
		}
	}
	if out := maskRoundTrip(t, e, "Mark as deleted. Please mark the box."); out != "Mark as deleted. Please mark the box." {
		t.Fatalf("common-word use masked: %q", out)
	}
}

func noRealRunes(t *testing.T, out string, name string) {
	t.Helper()
	low := textutil.Fold(out)
	for _, w := range strings.Fields(name) {
		r := []rune(textutil.Fold(w))
		for i := 0; i+3 <= len(r); i++ {
			if piece := string(r[i : i+3]); strings.Contains(low, piece) {
				t.Fatalf("%q keeps %q of real %q", out, piece, w)
			}
		}
	}
}

func TestAccentedSeparatedHandles(t *testing.T) {
	for _, c := range []struct {
		name, in string
		seps     []string
	}{
		{"François Lefèvre", "François Lefèvre, mail francois.lefevre", []string{"."}},
		{"José Martinez", "José Martinez, mail jose.martinez", []string{"."}},
		{"Renée Dubois", "Renée Dubois: renee.dubois and renee_dubois", []string{".", "_"}},
		{"Søren Østergaard", "Søren Østergaard: soren.ostergaard", []string{"."}},
		{"Søren Østergaard", "Søren Østergaard: @soren.ostergaard", []string{"."}},
	} {
		e := newEngine(t, nil)
		out := maskRoundTrip(t, e, c.in)
		noRealRunes(t, out, c.name)
		parts := strings.Fields(c.name)
		g, _ := e.Vault().Get("person.given", parts[0])
		f, _ := e.Vault().Get("person.family", parts[1])
		if g == "" || f == "" {
			t.Fatalf("%q: name records missing", c.in)
		}
		if !strings.HasPrefix(out, g+" "+f) {
			t.Fatalf("%q -> %q: full name not masked with its records", c.in, out)
		}
		for _, sep := range c.seps {
			if real := textutil.Fold(parts[0]) + sep + textutil.Fold(parts[1]); strings.Contains(textutil.Fold(out), real) {
				t.Fatalf("%q -> %q: handle %q left raw", c.in, out, real)
			}
		}
	}
}

func TestUnaccentedSeparatedHandlesAreByteStable(t *testing.T) {
	noDict := func(o *Options) { o.Dictionary = nil }
	for in, want := range map[string]string{
		"John Smith: john.smith":                 "Cillian Garrity: cillian.garrity",
		"John Smith: john_smith and @john.smith": "Cillian Garrity: cillian_garrity and @cillian.garrity",
	} {
		if out := maskRoundTrip(t, newEngine(t, noDict), in); out != want {
			t.Fatalf("%q -> %q, want %q", in, out, want)
		}
	}
}

func TestCommonWordNameAcrossRequests(t *testing.T) {
	e := newEngine(t, nil)
	first := maskRoundTrip(t, e, "Mark Jones and Rose Kim met.")
	if strings.Contains(first, "Mark") || strings.Contains(first, "Rose") {
		t.Fatalf("full names kept: %q", first)
	}
	for _, in := range []string{
		"Mark, can you review?",
		"Mark's laptop broke",
		"Mark and Rose went to lunch",
		"Rose and I went home",
		"I and Mark went home",
		"@Mark ping",
		"Mark J. signed off",
		"Mark said yes.",
	} {
		out := maskRoundTrip(t, e, in)
		if strings.Contains(out, "Mark") || strings.Contains(out, "Rose") {
			t.Fatalf("%q -> %q", in, out)
		}
	}
	if _, err := e.Vault().Assign("person.given", "Grant", func(int) string { return "Osmund" }); err != nil {
		t.Fatal(err)
	}
	if out := maskRoundTrip(t, e, "Grant will handle it"); strings.Contains(out, "Grant") {
		t.Fatalf("modal person verb not masked: %q", out)
	}
	for _, in := range []string{
		"Mark as deleted",
		"Please mark the box.",
		"Grant access to the repo",
		"A rose by any name",
		"// Mark's position",
	} {
		if out := maskRoundTrip(t, e, in); out != in {
			t.Fatalf("%q -> %q", in, out)
		}
	}
}
