package outworld

import (
	"context"
	"strings"
	"testing"
)

func TestCardInQueryStringKeepsTheURLShape(t *testing.T) {
	for _, c := range []struct{ text, prefix, suffix string }{
		{"see https://shop.example/pay?cc=5555555555554444&cvv=123&x=1 now", "see https://shop.example/pay?cc=", "&x=1 now"},
		{"see https://shop.example/pay?x=1&card_number=4111111111111111&y=2 now", "see https://shop.example/pay?x=1&card_number=", "&y=2 now"},
		{"see https://shop.example/pay;pan=4111111111111111?z=3 now", "see https://shop.example/pay;pan=", "?z=3 now"},
	} {
		e := newEngine(t, nil)
		s := e.NewSession("card")
		m, err := s.Mask(context.Background(), c.text)
		if err != nil || s.Err() != nil {
			t.Fatalf("%q: %v %v", c.text, err, s.Err())
		}
		if strings.Contains(m, "5555555555554444") || strings.Contains(m, "4111111111111111") || strings.Contains(m, "cvv=123") {
			t.Errorf("%q: masked %q keeps card data", c.text, m)
		}
		if !strings.HasPrefix(m, c.prefix) || !strings.HasSuffix(m, c.suffix) {
			t.Errorf("%q: masked %q changed the URL shape", c.text, m)
		}
		if back := s.Rehydrate(m); back != m {
			t.Errorf("%q: rehydrate changed the stripped text: %q", c.text, back)
		}
	}
}
