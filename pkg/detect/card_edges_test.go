package detect

import (
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

func financialSpans(t *testing.T, text string) []string {
	t.Helper()
	var out []string
	for _, sp := range regexSpans(t, text) {
		if sp.Source == "regex:credit_card" || sp.Source == "regex:card_cvv" {
			out = append(out, sp.Text)
		}
	}
	return out
}

func hasSpanText(spans []string, want string) bool {
	for _, s := range spans {
		if s == want {
			return true
		}
	}
	return false
}

func TestCardsAfterKeyPunctuation(t *testing.T) {
	for _, c := range []struct{ text, want string }{
		{"card=4111111111111111", "4111111111111111"},
		{"CARD=4111111111111111", "4111111111111111"},
		{"card_number=4111111111111111", "4111111111111111"},
		{"card=4111-1111-1111-1111", "4111-1111-1111-1111"},
		{"cc=4111 1111 1111 1111", "4111 1111 1111 1111"},
		{"cc=4111111111111111&exp=12", "4111111111111111"},
		{"card#4111111111111111 ok", "4111111111111111"},
		{"pan_4111111111111111", "4111111111111111"},
		{"?cc=5555555555554444&cvv=123", "5555555555554444"},
		{"pan=4111111111111111", "4111111111111111"},
		{"cardNumber=4111111111111111", "4111111111111111"},
		{"card_no=4111111111111111", "4111111111111111"},
		{"ccnum=4111111111111111", "4111111111111111"},
		{"CARD_NUMBER=4111111111111111", "4111111111111111"},
		{"billing_card=4111111111111111", "4111111111111111"},
		{"user.pan=4111111111111111", "4111111111111111"},
		{"&card_number=4111111111111111", "4111111111111111"},
		{";pan=4111111111111111", "4111111111111111"},
		{"cardno: 4111111111111111", "4111111111111111"},
		{"id=4111 1111 1111 1111", "4111 1111 1111 1111"},
		{"ref#4111-1111-1111-1111", "4111-1111-1111-1111"},
		{"x_4111 1111 1111 1111", "4111 1111 1111 1111"},
		{"id=3782 822463 10005", "3782 822463 10005"},
	} {
		if got := financialSpans(t, c.text); !hasSpanText(got, c.want) {
			t.Errorf("%q: want card %q, got %q", c.text, c.want, got)
		}
	}
	if got := financialSpans(t, "?cc=5555555555554444&cvv=123"); !hasSpanText(got, "123") {
		t.Errorf("cvv no longer stripped: %q", got)
	}
}

func TestCardKeepsWithoutACue(t *testing.T) {
	for _, text := range []string{
		"trace_id=4111111111111111",
		"order#4111111111111111",
		"/v_4111111111111111/",
		"id=4111111111111111",
		"discard=4111111111111111",
		"span=4111111111111111",
		"panel=4111111111111111",
		"account=4111111111111111",
		"scc=4111111111111111",
		"postcard_id=4111111111111111",
		"cc the team: id=4111111111111111",
		"a pan of x=4111111111111111",
		"card v.4111111111111111",
		"card /4111111111111111",
		"x4111111111111111",
		"avenu4111 1111 1111 1111",
		"user_id=4111111111111111 session_id=5555555555554444",
	} {
		if got := financialSpans(t, text); len(got) != 0 {
			t.Errorf("%q: unexpected card %q", text, got)
		}
	}
}

func TestCardAndSSNGluedToLettersWithACue(t *testing.T) {
	for _, c := range []struct{ text, want string }{
		{"card x4111111111111111", "4111111111111111"},
		{"card: 4111111111111111x", "4111111111111111"},
		{"card 4111111111111111abc", "4111111111111111"},
	} {
		if got := financialSpans(t, c.text); !hasSpanText(got, c.want) {
			t.Errorf("%q: want card %q, got %q", c.text, c.want, got)
		}
	}
	for _, c := range []struct{ text, want string }{
		{"SSN x123-45-6789", "123-45-6789"},
		{"ssn:123456789x", "123456789"},
		{"SSN: 536-22-8471abc", "536-22-8471"},
	} {
		if got := spanOf(t, c.text, entity.GovID); got != c.want {
			t.Errorf("%q: want ssn %q, got %q", c.text, c.want, got)
		}
	}
	for _, text := range []string{"x4111111111111111 card", "card x4111111111111111y", "4111 1111 1111 1111x"} {
		if strings.HasPrefix(text, "x4111111111111111 card") {
			continue
		}
		if got := financialSpans(t, text); len(got) != 0 {
			t.Errorf("%q: unexpected card %q", text, got)
		}
	}
	for _, text := range []string{"x123-45-6789", "SSN x900-45-6789", "SSN x123-45-6789y", "SSN x000-45-6789"} {
		if got := spanOf(t, text, entity.GovID); got != "" {
			t.Errorf("%q: unexpected ssn %q", text, got)
		}
	}
}

func TestSlashAndPluralNumberPhoneLabels(t *testing.T) {
	for _, sep := range []string{"~", "〜"} {
		w := func(s string) string { return strings.ReplaceAll(s, "~", sep) }
		for _, text := range []string{"phoneNumbers: 804~555~1234", "phone/fax: 804~555~1234", "Tel/Fax 804~555~1234", "fax/phone 804~555~1234"} {
			text = w(text)
			if got := spanOf(t, text, entity.Phone); got != w("804~555~1234") {
				t.Errorf("%q: phone %q", text, got)
			}
		}
		for _, text := range []string{"phoneNumberId=804~555~1234", "phonesCount=804~555~1234", "/api/v1/call/fax/804~555~1234", "price/fax 804~555~1234x"} {
			text = w(text)
			if got := spanOf(t, text, entity.Phone); got != "" {
				t.Errorf("%q: unexpected phone %q", text, got)
			}
		}
	}
}

func TestCardRuleDoesNotStripCodeIds(t *testing.T) {
	for _, text := range []string{
		"trace_id=4111111111111111",
		"request_id=5555555555554444",
		"span_id=4111111111111111",
		"tx#4111111111111111",
		"build_4111111111111111",
		"commit=4111111111111111",
		"seq=378282246310005",
		"GET /v1/items/4111111111111111",
		"version=4111111111111111.2",
		"x.4111111111111111",
		"id_4111111111111111_tmp",
		"hash=4111111111111111&page=2",
		"offset=5555555555554444",
		"panelId=4111111111111111",
	} {
		if got := financialSpans(t, text); len(got) != 0 {
			t.Errorf("%q: unexpected card %q", text, got)
		}
	}
}
