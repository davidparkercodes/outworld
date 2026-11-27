package detect

import (
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

var cueEdgeVariants = []string{"-", "〜", "・", "／"}

func withSep(s, sep string) string { return strings.ReplaceAll(s, "-", sep) }

func spanOf(t *testing.T, text string, typ entity.Type) string {
	t.Helper()
	for _, sp := range regexSpans(t, text) {
		if sp.Type == typ {
			return sp.Text
		}
	}
	return ""
}

type cueEdgeCase struct {
	text, want string
	typ        entity.Type
}

func checkCueEdgeCases(t *testing.T, cases []cueEdgeCase) {
	t.Helper()
	for _, c := range cases {
		for _, sep := range cueEdgeVariants {
			text, want := withSep(c.text, sep), withSep(c.want, sep)
			if got := spanOf(t, text, c.typ); got != want {
				t.Errorf("%q: %s span %q, want %q", text, c.typ, got, want)
			}
		}
	}
}

func checkCueEdgeNegatives(t *testing.T, texts []string, typ entity.Type) {
	t.Helper()
	for _, base := range texts {
		for _, sep := range cueEdgeVariants {
			text := withSep(base, sep)
			if got := spanOf(t, text, typ); got != "" {
				t.Errorf("%q: unexpected %s %q", text, typ, got)
			}
		}
	}
}

func TestCueBeforeEqualsOrBracket(t *testing.T) {
	checkCueEdgeCases(t, []cueEdgeCase{
		{"phone=(804)555-1234", "(804)555-1234", entity.Phone},
		{"tel=(804)555-1234", "(804)555-1234", entity.Phone},
		{"mobile=(804)555-1234", "(804)555-1234", entity.Phone},
		{"PHONE=(804)555-1234", "(804)555-1234", entity.Phone},
		{"phone_number=(804)555-1234", "(804)555-1234", entity.Phone},
		{"?phone=(804)555-1234&x=1", "(804)555-1234", entity.Phone},
		{"contact=[(804)555-1234]", "(804)555-1234", entity.Phone},
		{"phone: (804)555-1234", "(804)555-1234", entity.Phone},
	})
	if got := spanOf(t, "see [(804)555-1234] now", entity.Phone); got != "(804)555-1234" {
		t.Errorf("enclosed bracket form: %q", got)
	}
	checkCueEdgeNegatives(t, []string{"arr[(804)-555-1234]", "arr[(804)555-1234]", "x=(804)-555-1234"}, entity.Phone)
}

func TestNumbersGluedToTheirCue(t *testing.T) {
	checkCueEdgeCases(t, []cueEdgeCase{
		{"SSN123-45-6789", "123-45-6789", entity.GovID},
		{"call804-555-1234", "804-555-1234", entity.Phone},
		{"Phone804-555-1234", "804-555-1234", entity.Phone},
		{"SSN\u200b123-45-6789", "123-45-6789", entity.GovID},
		{"call\u200b804-555-1234", "804-555-1234", entity.Phone},
		{"call\u00ad804-555-1234", "804-555-1234", entity.Phone},
		{"SSN\u2060123-45-6789", "123-45-6789", entity.GovID},
	})
	checkCueEdgeNegatives(t, []string{"recall804-555-1234", "callx804-555-1234"}, entity.Phone)
	checkCueEdgeNegatives(t, []string{"lessn123-45-6789"}, entity.GovID)
}

func TestMarkdownDelimitersAreBoundaries(t *testing.T) {
	checkCueEdgeCases(t, []cueEdgeCase{
		{"_Phone: 804-555-1234_", "804-555-1234", entity.Phone},
		{"SSN: _536-22-8471_", "536-22-8471", entity.GovID},
		{"*Phone: 804-555-1234*", "804-555-1234", entity.Phone},
		{"**call 804-555-1234**", "804-555-1234", entity.Phone},
		{"__SSN 536-22-8471__", "536-22-8471", entity.GovID},
		{"`call 804-555-1234`", "804-555-1234", entity.Phone},
	})
	for _, c := range []cueEdgeCase{
		{"_804-555-1234_", "804-555-1234", entity.Phone},
		{"*804-555-1234*", "804-555-1234", entity.Phone},
		{"**804-555-1234**", "804-555-1234", entity.Phone},
		{"__804-555-1234__", "804-555-1234", entity.Phone},
		{"`804-555-1234`", "804-555-1234", entity.Phone},
		{"_536-22-8471_", "536-22-8471", entity.GovID},
		{"**536-22-8471**", "536-22-8471", entity.GovID},
	} {
		if got := spanOf(t, c.text, c.typ); got != c.want {
			t.Errorf("%q: %s span %q, want %q", c.text, c.typ, got, c.want)
		}
	}
	if got := spanOf(t, "a_536-22-8471", entity.GovID); got != "" {
		t.Errorf("identifier a_536-22-8471 masked as %q", got)
	}
	if got := spanOf(t, "a_804-555-1234", entity.Phone); got != "804-555-1234" {
		t.Errorf("a_804-555-1234 changed behaviour: %q", got)
	}
}

func TestKeyStyleLabelsWithAnySeparator(t *testing.T) {
	checkCueEdgeCases(t, []cueEdgeCase{
		{"phone=804-555-1234", "804-555-1234", entity.Phone},
		{"phone_number: 804-555-1234", "804-555-1234", entity.Phone},
		{"contact.phone=804-555-1234", "804-555-1234", entity.Phone},
		{"ssn=536-22-8471", "536-22-8471", entity.GovID},
		{"user.ssn 536-22-8471", "536-22-8471", entity.GovID},
		{"tel_no: 804-555-1234", "804-555-1234", entity.Phone},
	})
	for _, base := range []string{"call_id=734-899-7921", "/api/v1/call/734-899-7921", "GET https://x.example/call/734-899-7921", "callback=734-899-7921"} {
		for _, sep := range cueEdgeVariants[1:] {
			text := withSep(base, sep)
			for _, sp := range regexSpans(t, text) {
				if sp.Source == "regex:phone_separated" {
					t.Errorf("%q: unexpected %q", text, sp.Text)
				}
			}
		}
	}
}

func TestCJKNeighboursAreBoundaries(t *testing.T) {
	checkCueEdgeCases(t, []cueEdgeCase{
		{"Call 804-555-1234です", "804-555-1234", entity.Phone},
		{"Call 804-555-1234中", "804-555-1234", entity.Phone},
		{"電話番号は804-555-1234です", "804-555-1234", entity.Phone},
		{"전화 804-555-1234로", "804-555-1234", entity.Phone},
		{"携帯: 804-555-1234", "804-555-1234", entity.Phone},
		{"电话804-555-1234", "804-555-1234", entity.Phone},
		{"SSN 536-22-8471です", "536-22-8471", entity.GovID},
	})
}

func TestSeparatedFormsAgreeWithDashedForms(t *testing.T) {
	for _, c := range []struct {
		dashed string
		typ    entity.Type
		forms  func(sep string) string
	}{
		{"SSN 912-70-1234", entity.GovID, func(sep string) string { return "SSN 912" + sep + "70" + sep + "1234" }},
		{"SSN 000-22-8471", entity.GovID, func(sep string) string { return "SSN 000" + sep + "22" + sep + "8471" }},
		{"ITIN 912-70-1234", entity.GovID, func(sep string) string { return "ITIN 912" + sep + "70" + sep + "1234" }},
		{"+1 804-555-1234", entity.Phone, func(sep string) string { return "+1 804" + sep + "555" + sep + "1234" }},
		{"+1-804-555-1234", entity.Phone, func(sep string) string { return "+1" + sep + "804" + sep + "555" + sep + "1234" }},
	} {
		want := spanOf(t, c.dashed, c.typ) != ""
		for _, sep := range []string{"~", "〜", "・", "／", "_", "|"} {
			text := c.forms(sep)
			if got := spanOf(t, text, c.typ) != ""; got != want {
				t.Errorf("%q: masked=%v, %q masked=%v", text, got, c.dashed, want)
			}
		}
	}
}

func TestZeroWidthInsideAPhoneIsCovered(t *testing.T) {
	text := "Call 804\u200b555\u200b1234 now"
	if got := spanOf(t, text, entity.Phone); got != "804\u200b555\u200b1234" {
		t.Errorf("span %q does not cover the zero-width characters", got)
	}
}

func TestPluralAndCompoundPhoneCues(t *testing.T) {
	checkCueEdgeCases(t, []cueEdgeCase{
		{"telephone 804-555-1234", "804-555-1234", entity.Phone},
		{"Telephone: 804-555-1234", "804-555-1234", entity.Phone},
		{"telephones: 804-555-1234", "804-555-1234", entity.Phone},
		{"cellphone 804-555-1234", "804-555-1234", entity.Phone},
		{"cell phone 804-555-1234", "804-555-1234", entity.Phone},
		{"phones: 804-555-1234", "804-555-1234", entity.Phone},
		{"mobiles: 804-555-1234", "804-555-1234", entity.Phone},
		{"contacts: 804-555-1234", "804-555-1234", entity.Phone},
		{"landline 804-555-1234", "804-555-1234", entity.Phone},
		{"hotline: 804-555-1234", "804-555-1234", entity.Phone},
		{"fax number 804-555-1234", "804-555-1234", entity.Phone},
		{"SSNs: 536-22-8471", "536-22-8471", entity.GovID},
		{"Call 11 804-555-1234", "804-555-1234", entity.Phone},
		{"phone 7 804-555-1234", "804-555-1234", entity.Phone},
	})
	for _, text := range []string{"contacts list has 804 items", "phones 3-4-5", "the hotline was busy", "landline usage 12", "telephones ring"} {
		for _, sp := range regexSpans(t, text) {
			if sp.Type == entity.Phone {
				t.Errorf("%q: unexpected phone %q", text, sp.Text)
			}
		}
	}
	for _, base := range []string{"calls=804-555-1234", "contacts_id=804-555-1234", "contacts=804-555-1234", "/api/v1/contacts/804-555-1234", "Call 111 804-555-1234", "after 11 804~555~1234"} {
		for _, sep := range cueEdgeVariants[1:] {
			text := withSep(base, sep)
			for _, sp := range regexSpans(t, text) {
				if sp.Source == "regex:phone_separated" {
					t.Errorf("%q: unexpected %q", text, sp.Text)
				}
			}
		}
	}
}

func TestCamelCaseAndListPhoneCues(t *testing.T) {
	for _, sep := range []string{"~", "〜"} {
		w := func(s string) string { return strings.ReplaceAll(s, "~", sep) }
		for _, c := range []struct{ text, want string }{
			{"cellphone 804~555~1234", "804~555~1234"},
			{"phones: 804~555~1234", "804~555~1234"},
			{"telephones 804~555~1234", "804~555~1234"},
			{"mobilePhone: 804~555~1234", "804~555~1234"},
			{`{"homePhone":"804~555~1234"}`, "804~555~1234"},
			{"callbackNumber=804~555~1234", "804~555~1234"},
			{"userPhone 804~555~1234", "804~555~1234"},
			{"telephone 804~555~1234", "804~555~1234"},
			{"Telephone: 804~555~1234", "804~555~1234"},
			{"cell phone 804~555~1234", "804~555~1234"},
			{"phoneNumber=804~555~1234", "804~555~1234"},
		} {
			text, want := w(c.text), w(c.want)
			if got := spanOf(t, text, entity.Phone); got != want {
				t.Errorf("%q: phone span %q, want %q", text, got, want)
			}
		}
		list := w("Phones 804~555~1234, 804~555~1235")
		var got []string
		for _, sp := range regexSpans(t, list) {
			if sp.Type == entity.Phone {
				got = append(got, sp.Text)
			}
		}
		if len(got) != 2 {
			t.Errorf("%q: want both numbers, got %q", list, got)
		}
		for _, text := range []string{"callId=804~555~1234", "callID: 804~555~1234", "phoneId=804~555~1234", "smsId 804~555~1234", "callSid=804~555~1234", "telephoneId=804~555~1234", "phonesCount=804~555~1234", "cellphoneModel 804~555~1234", `{"callId":"804~555~1234"}`, "callback=804~555~1234"} {
			text = w(text)
			if got := spanOf(t, text, entity.Phone); got != "" {
				t.Errorf("%q: unexpected phone %q", text, got)
			}
		}
	}
	covered := false
	for _, sp := range regexSpans(t, "Cellphone#804_555_1234") {
		if strings.Contains(sp.Text, "804_555_1234") {
			covered = true
		}
	}
	if !covered {
		t.Errorf("Cellphone#804_555_1234 is no longer masked")
	}
}
