package outworld

import (
	"context"
	"strings"
	"testing"
)

func maskAll(t *testing.T, e *Engine, newSession bool, texts ...string) []string {
	t.Helper()
	s := e.NewSession("p")
	var outs []string
	for _, in := range texts {
		if newSession {
			s = e.NewSession("p")
		}
		out, err := s.Mask(context.Background(), in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		outs = append(outs, out)
	}
	return outs
}

func TestLoneTechNameKeepsUnlessAnchoredAndKnownToVault(t *testing.T) {
	ctx := context.Background()
	for _, in := range []string{"Dear Claude,", "Dear Claude, hi", "Dear Claude!", "Hi Claude,", "Hello Claude, can you help?", "Hey Claude,", "Thanks Claude!", "Thank you Claude.", "Good morning Claude,", "Ok. Hi Claude, one more", "Claude said the API is down"} {
		if out, err := newEngine(t, nil).NewSession("p").Mask(ctx, in); err != nil || out != in {
			t.Errorf("%q: got %q (%v)", in, out, err)
		}
	}
	for _, in := range []string{"Dear Claude Monet,", "Hi Claude Shannon,"} {
		if out, _ := newEngine(t, nil).NewSession("p").Mask(ctx, in); strings.Contains(out, "Claude") {
			t.Errorf("%q: real person left raw in %q", in, out)
		}
	}
	if out, _ := newEngine(t, nil).NewSession("p").Mask(ctx, "Dear Claude and Mary,"); !strings.Contains(out, "Claude") || strings.Contains(out, "Mary") {
		t.Errorf("Dear Claude and Mary: got %q, want Claude kept and Mary masked", out)
	}

	later := maskAll(t, newEngine(t, nil), true, "Claude Shannon wrote X", "Hi Claude, can you help?", "Claude said the API is down")
	if later[1] != "Hi Claude, can you help?" || strings.Contains(later[2], "Claude") {
		t.Errorf("later requests: %q", later)
	}
	given := strings.Fields(later[0])[0]

	for _, in := range []string{"Claude Shannon wrote X. Thanks Claude, noted.", "Hi Claude, see the paper. Claude Shannon wrote it."} {
		e := newEngine(t, nil)
		s := e.NewSession("p")
		out, _ := s.Mask(ctx, in)
		if strings.Contains(out, "Claude") {
			t.Errorf("%q: got %q, want both masked", in, out)
		}
		if back := s.Rehydrate(out); back != in {
			t.Errorf("%q: restored %q", in, back)
		}
	}

	if out, _ := newEngine(t, nil).NewSession("p").Mask(ctx, "Claude S. called"); out != "Claude S. called" {
		t.Errorf("fresh vault, Claude S. called: got %q, want unchanged", out)
	}
	e := newEngine(t, nil)
	first := maskAll(t, e, true, "I met Claude Dupont.", "Claude S. called", "Hi Claude, Claude Dupont says hi")
	if strings.Contains(first[1], "Claude") || strings.Contains(first[2], "Claude") {
		t.Errorf("anchored references: %q", first)
	}

	sameRequest := maskAll(t, newEngine(t, nil), false, "Claude Shannon wrote X", "Hi Claude, can you help?")
	if strings.Contains(sameRequest[1], "Claude") {
		t.Errorf("same request: %q", sameRequest)
	}

	e2 := newEngine(t, nil)
	maskAll(t, e2, true, "Claude Shannon wrote X")
	if back := e2.NewSession("p").Rehydrate(given + " says the fix is in."); back != "Claude says the fix is in." {
		t.Errorf("echoed surrogate restored as %q", back)
	}
}
