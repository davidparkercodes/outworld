package outworld

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/detect"
	"github.com/davidparkercodes/outworld/pkg/entity"
)

type reDetector struct {
	name string
	typ  entity.Type
	re   *regexp.Regexp
	skip string
}

func (d reDetector) Name() string { return d.name }

func (d reDetector) Detect(_ context.Context, text string) ([]entity.Span, error) {
	var out []entity.Span
	for _, m := range d.re.FindAllStringIndex(text, -1) {
		t := text[m[0]:m[1]]
		if d.skip != "" && strings.Contains(t, d.skip) {
			continue
		}
		out = append(out, entity.Span{Start: m[0], End: m[1], Type: d.typ, Label: string(d.typ), Text: t, Source: d.name, Priority: 50, Score: 1})
	}
	return out, nil
}

func residueEngine(t *testing.T, extra ...detect.Detector) *Engine {
	return newEngine(t, func(o *Options) {
		o.Dictionary = nil
		o.Policy.NoNameHeuristics = true
		o.Policy.Actions[entity.Location] = entity.Mask
		o.Extra = extra
	})
}

var loneAmber = reDetector{name: "test:first", typ: entity.Location, re: regexp.MustCompile(`\bAmber\b`)}

func residueRecord(e *Engine, real string) bool {
	for _, r := range e.vault.Records() {
		if r.Kind == residueKind && r.Real == real {
			return true
		}
	}
	return false
}

func TestResidueNextToSurrogateIsMaskedNotBlocked(t *testing.T) {
	pair := reDetector{name: "test:pair", typ: entity.Person, re: regexp.MustCompile(`\b[A-Z][a-z]+ Room\b`), skip: "Amber"}
	e := residueEngine(t, loneAmber, pair)
	s := e.NewSession("")
	in := "Dinner with the client at The Amber Room, table for four."
	out, err := s.Mask(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Err(); err != nil {
		t.Fatalf("a word next to a surrogate must be masked, not blocked: %v (out=%q)", err, out)
	}
	if strings.Contains(out, "Amber") || strings.Contains(out, "Room") {
		t.Fatalf("residue left in %q", out)
	}
	if !strings.HasPrefix(out, "Dinner with the client at The ") || !strings.HasSuffix(out, ", table for four.") {
		t.Fatalf("text outside the finding changed: %q", out)
	}
	if back := s.Rehydrate(out); back != in {
		t.Fatalf("rehydrate: %q", back)
	}
	if !residueRecord(e, "Room") {
		t.Fatalf("expected the repair to go through the residue path: %+v", e.vault.Records())
	}
}

func TestResidueOfRealNameIsMasked(t *testing.T) {
	surnameFirst := reDetector{name: "test:surname-first", typ: entity.Person, re: regexp.MustCompile(`\bPruitt, [A-Z][a-z]+`), skip: "Thiha"}
	thiha := reDetector{name: "test:first", typ: entity.Location, re: regexp.MustCompile(`\bThiha\b`)}
	e := residueEngine(t, thiha, surnameFirst)
	s := e.NewSession("")
	in := "Present: Lane Pruitt, Thiha"
	out, _ := s.Mask(ctx, in)
	if err := s.Err(); err != nil {
		t.Fatalf("unexpected block: %v (out=%q)", err, out)
	}
	if strings.Contains(out, "Pruitt") || strings.Contains(out, "Thiha") {
		t.Fatalf("real name part reached the output: %q", out)
	}
	if back := s.Rehydrate(out); back != in {
		t.Fatalf("rehydrate: %q", back)
	}
	if !residueRecord(e, "Pruitt") {
		t.Fatalf("expected the repair to go through the residue path: %+v", e.vault.Records())
	}
}

func TestStripResidueIsRedacted(t *testing.T) {
	token := reDetector{name: "test:token", typ: entity.Secret, re: regexp.MustCompile(`key=\S+`), skip: "Amber"}
	e := residueEngine(t, loneAmber, token)
	s := e.NewSession("")
	out, _ := s.Mask(ctx, "use key=Amber-tail9931 today")
	if err := s.Err(); err != nil {
		t.Fatalf("unexpected block: %v (out=%q)", err, out)
	}
	if strings.Contains(out, "tail9931") || strings.Contains(out, "Amber") || !strings.Contains(out, "[redacted") {
		t.Fatalf("secret residue left in %q", out)
	}
}

func TestUnrepairableResidueStillBlocks(t *testing.T) {
	phone := reDetector{name: "test:phone", typ: entity.Phone, re: regexp.MustCompile(`\b[A-Z][a-z]+ x\d{4}\b`), skip: "Amber"}
	e := residueEngine(t, loneAmber, phone)
	s := e.NewSession("")
	in := "Call the desk and ask for Amber x4471 before noon."
	out, _ := s.Mask(ctx, in)
	err := s.Err()
	if err == nil || !errors.Is(err, ErrLeak) {
		t.Fatalf("a finding that cannot be repaired must block, got %v (out=%q)", err, out)
	}
	if strings.Contains(err.Error(), "4471") {
		t.Fatal("leak error must not contain the value")
	}
	vs := s.Violations()
	if len(vs) != 1 {
		t.Fatalf("violations: %+v", vs)
	}
	v := vs[0]
	if v.Type != entity.Phone || v.Source != "test:phone" || v.Reason != ReasonOverlap || v.Hint == "" {
		t.Fatalf("violation detail: %+v", v)
	}
	r := []rune(out)
	if v.Start < 0 || v.End > len(r) || !strings.HasSuffix(string(r[v.Start:v.End]), " x4471") {
		t.Fatalf("offsets %d-%d do not point at the finding in %q", v.Start, v.End, out)
	}
	if strings.Contains(v.Preview, "4471") || !strings.Contains(v.Preview, "***") || !strings.Contains(v.Preview, "before noon") {
		t.Fatalf("preview: %q", v.Preview)
	}
}

type firstOnly struct{ reDetector }

func (d firstOnly) Detect(c context.Context, text string) ([]entity.Span, error) {
	spans, err := d.reDetector.Detect(c, text)
	if len(spans) > 1 {
		spans = spans[:1]
	}
	return spans, err
}

func TestFindingsThatKeepAppearingBlockAfterBoundedRounds(t *testing.T) {
	drip := firstOnly{reDetector{name: "test:drip", typ: entity.Secret, re: regexp.MustCompile(`\bq[a-z]+\b`)}}
	e := residueEngine(t, drip)
	s := e.NewSession("")
	out, _ := s.Mask(ctx, "qone qtwo qthree qfour qfive qsix qseven qeight qnine")
	if err := s.Err(); err == nil {
		t.Fatalf("masking must stop after a bounded number of rounds and block (out=%q)", out)
	}
	vs := s.Violations()
	if len(vs) == 0 || vs[0].Reason != ReasonRounds {
		t.Fatalf("violations: %+v", vs)
	}
	if !strings.Contains(out, "qnine") {
		t.Fatalf("test expects residue to remain, got %q", out)
	}
}

func TestCheckViolationCarriesLocationAndRedactedPreview(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("")
	s.SetLocation(2, "messages[2].name")
	s.Check(ctx, "maria.garcia@corp.test")
	vs := s.Violations()
	if len(vs) == 0 {
		t.Fatal("expected a violation")
	}
	for _, v := range vs {
		if v.Message != 2 || v.Field != "messages[2].name" || v.Reason != ReasonField || v.Hint == "" || v.Start < 0 {
			t.Fatalf("violation detail: %+v", v)
		}
		if strings.Contains(strings.ToLower(v.Preview), "maria") || strings.Contains(v.Preview, "garcia") || strings.Contains(v.Preview, "corp") {
			t.Fatalf("preview leaks the value: %q", v.Preview)
		}
	}
}

func TestPreviewShowsNoCharactersOfAnyType(t *testing.T) {
	text := "alpha SECRETONE beta SECRETTWO gamma"
	a, b := strings.Index(text, "SECRETONE"), strings.Index(text, "SECRETTWO")
	p := redactedPreview(text, a, a+9, [][2]int{{a, a + 9}, {b, b + 9}})
	if p != "alpha [***] beta [***] gamma" {
		t.Fatal(p)
	}
	e := newEngine(t, nil)
	for _, in := range []string{"maria.garcia@corp.test", "Maria Garcia", "(804) 555-1234", "123 Main Street, Richmond, VA 23220", "10.0.0.5", "123-45-6789"} {
		s := e.NewSession("")
		s.Check(ctx, in)
		vs := s.Violations()
		if len(vs) == 0 {
			t.Fatalf("%q: expected a violation", in)
		}
		for _, v := range vs {
			rest := strings.ReplaceAll(v.Preview, "[***]", "")
			for _, tok := range regexp.MustCompile(`[\p{L}\p{N}]+`).FindAllString(in, -1) {
				if len(tok) >= 2 && strings.Contains(rest, tok) {
					t.Errorf("%q: preview %q shows %q", in, v.Preview, tok)
				}
			}
			if regexp.MustCompile(`\[[^*\]]+\*\*\*`).MatchString(v.Preview) || regexp.MustCompile(`\*\*\*[^\]]+\]`).MatchString(v.Preview) {
				t.Errorf("%q: preview %q shows part of the value", in, v.Preview)
			}
		}
	}
}

func TestTraceListsReplacementsWithoutReals(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("")
	s.SetTrace(true)
	s.SetLocation(1, "messages[1].content")
	out, _ := s.Mask(ctx, "Email John Smith at john.smith@gmail.com about SSN 123-45-6789.")
	reps := s.Replacements()
	if len(reps) < 3 {
		t.Fatalf("replacements: %+v", reps)
	}
	r := []rune(out)
	for _, rp := range reps {
		if string(r[rp.Start:rp.End]) != rp.Replacement || rp.Message != 1 || rp.Source == "" {
			t.Fatalf("replacement %+v does not match %q", rp, out)
		}
		assertNoReals(t, rp.Replacement)
	}
	plain := e.NewSession("")
	plain.Mask(ctx, "Email John Smith")
	if len(plain.Replacements()) != 0 {
		t.Fatal("replacements must only be collected when tracing")
	}
}

func TestResidueWordDoesNotBecomeAKnownName(t *testing.T) {
	pair := reDetector{name: "test:pair", typ: entity.Person, re: regexp.MustCompile(`\b[A-Z][a-z]+ Room\b`), skip: "Amber"}
	e := residueEngine(t, loneAmber, pair)
	s := e.NewSession("")
	if _, err := s.Mask(ctx, "Dinner at The Amber Room."); err != nil || s.Err() != nil {
		t.Fatalf("%v %v", err, s.Err())
	}
	if !residueRecord(e, "Room") {
		t.Fatalf("expected a residue record: %+v", e.vault.Records())
	}
	later := e.NewSession("")
	in := "Meet in Room Ada on the 4th floor, the Room is booked."
	out, _ := later.Mask(ctx, in)
	if err := later.Err(); err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("a word masked only as residue must not be hunted in later requests: %q", out)
	}
}
