package detect

import (
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

func TestCueGluedOntoTheValue(t *testing.T) {
	for _, c := range []struct {
		text, want string
		typ        entity.Type
	}{
		{"SSN_536-22-8471", "536-22-8471", entity.GovID},
		{"SSN536228471", "536228471", entity.GovID},
		{"ssn_536228471 on file", "536228471", entity.GovID},
		{"id: SSN536-22-8471.", "536-22-8471", entity.GovID},
		{"TEL090-1234-5678", "090-1234-5678", entity.Phone},
		{"tel_090-1234-5678", "090-1234-5678", entity.Phone},
		{"PHONE804-555-1234", "804-555-1234", entity.Phone},
		{"call804-555-1234", "804-555-1234", entity.Phone},
		{"電話804-555-1234", "804-555-1234", entity.Phone},
	} {
		if got := spanOf(t, c.text, c.typ); got != c.want {
			t.Errorf("%q: %s span %q, want %q", c.text, c.typ, got, c.want)
		}
	}
	for _, c := range []struct {
		text string
		typ  entity.Type
	}{
		{"HOSTEL090-1234-5678", entity.Phone},
		{"motel804-555-1234", entity.Phone},
		{"lessn123456789", entity.GovID},
		{"ASSN_536228471", entity.GovID},
		{"tel_id_8045551234", entity.Phone},
		{"SSN_900-22-8471", entity.GovID},
		{"SSN000228471", entity.GovID},
		{"Ssn536228471", entity.GovID},
		{"ssn536-22-8471x", entity.GovID},
	} {
		if got := spanOf(t, c.text, c.typ); got != "" {
			t.Errorf("%q: unexpected %s %q %+v", c.text, c.typ, got, regexSpans(t, c.text))
		}
	}
}

func cardSourceSpans(t *testing.T, text string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, sp := range regexSpans(t, text) {
		if sp.Type == entity.Financial {
			out[sp.Text] = sp.Source
		}
	}
	return out
}

func TestExpiryAloneIsNotACard(t *testing.T) {
	for _, text := range []string{"exp 12/27", "expires 03/2028", "valid thru 12/27", "good thru 03/2028", "expiry: 12/27", "exp 12/27 cvv 123", "expires 03/2028, cvc 456"} {
		got := cardSourceSpans(t, text)
		for v, src := range got {
			if v == "12/27" || v == "03/2028" {
				t.Errorf("%q: expiry stripped as a card by %s", text, src)
			}
		}
	}
	line := cardSourceSpans(t, "card 4111 1111 1111 1111 exp 12/27 cvv 123")
	for _, want := range []string{"4111 1111 1111 1111", "12/27", "123"} {
		if _, ok := line[want]; !ok {
			t.Errorf("card line: %q not stripped (%v)", want, line)
		}
	}
	pinned := cardSourceSpans(t, "cc: 5500-0000-0000-0004 exp 12/27")
	if _, ok := pinned["12/27"]; !ok {
		t.Errorf("expiry next to a card is no longer stripped: %v", pinned)
	}
	if _, ok := pinned["5500-0000-0000-0004"]; !ok {
		t.Errorf("card no longer stripped: %v", pinned)
	}
	if _, ok := cardSourceSpans(t, "Visa card, expiry 09/29")["09/29"]; !ok {
		t.Errorf("an expiry with a card word stopped being stripped")
	}
}
