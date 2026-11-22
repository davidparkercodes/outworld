package outworld

import (
	"strings"
	"testing"
)

func TestMemoKeepsEarlierTurnsByteStable(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	first := "note: abelard will review it on 2026-03-04"
	s1 := e.NewSession("")
	out1, _ := s1.Mask(ctx, first)
	if !strings.Contains(out1, "abelard") {
		t.Skipf("lowercase name already masked: %q", out1)
	}
	s2 := e.NewSession("")
	s2.SetPriming(true)
	s2.Mask(ctx, first)
	s2.Mask(ctx, "Abelard Ashdown signed off.")
	s2.SetPriming(false)
	again, _ := s2.Mask(ctx, first)
	if again != out1 {
		t.Fatalf("earlier turn changed between requests:\n was %q\n now %q", out1, again)
	}
	if got := s2.Rehydrate(again); got != first {
		t.Fatalf("rehydrate %q", got)
	}
	if st := e.MemoStats(); st.Hits == 0 || st.Entries == 0 {
		t.Fatalf("memo not used: %+v", st)
	}
}

func TestMemoRemembersModelText(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("")
	upstream := "Reach Alice Example at alice@corp.example before May 5, 2026."
	shown := s.Rehydrate(upstream)
	s.Remember(shown, upstream)
	next := e.NewSession("")
	out, err := next.Mask(ctx, shown)
	if err != nil || out != upstream {
		t.Fatalf("assistant turn drifted: %q %v", out, err)
	}
	other := e.NewSession("other-scope")
	if out2, _ := other.Mask(ctx, shown); out2 == upstream {
		t.Fatalf("memo leaked across scopes")
	}
}

func TestMemoDisabledAndEviction(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.MemoBytes = -1 })
	e.NewSession("").Mask(ctx, "John Smith called.")
	if st := e.MemoStats(); st.Entries != 0 {
		t.Fatalf("memo should be off: %+v", st)
	}
	m := NewMemo(600)
	for i := 0; i < 50; i++ {
		m.put(memoEntry{key: memoKeyFor("s", strings.Repeat("x", i+1)), out: strings.Repeat("y", 40)})
	}
	if st := m.Stats(); st.Bytes > 600 || st.Entries == 0 || st.Entries >= 50 {
		t.Fatalf("eviction: %+v", st)
	}
}

func TestLearnedNamePropagatesAcrossMessages(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil })
	texts := []string{
		"hope: can someone check https://wiki.example/people/hope-valdane today",
		"Hi team,\nHope Valdane will send the report.",
	}
	s := e.NewSession("")
	s.SetPriming(true)
	for _, x := range texts {
		s.Mask(ctx, x)
	}
	s.SetPriming(false)
	for _, x := range texts {
		out, err := s.Mask(ctx, x)
		if err != nil {
			t.Fatal(err)
		}
		if l := strings.ToLower(out); strings.Contains(l, "valdane") || strings.Contains(l, "hope") {
			t.Errorf("name survived: %q", out)
		}
		if back := s.Rehydrate(out); back != x {
			t.Errorf("rehydrate:\n got %q\nwant %q", back, x)
		}
	}
	if err := s.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestAllowListKeepsTerms(t *testing.T) {
	e := newEngine(t, func(o *Options) { o.Dictionary = nil; o.Policy.Allow = []string{"Grace Hopper"} })
	out, _ := e.NewSession("").Mask(ctx, "Grace Hopper inspired Margaret Hamilton.")
	if !strings.Contains(out, "Grace Hopper") || strings.Contains(out, "Margaret") {
		t.Fatalf("got %q", out)
	}
}
