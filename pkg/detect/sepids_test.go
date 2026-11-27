package detect

import (
	"context"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

func regexSpans(t *testing.T, text string) []entity.Span {
	t.Helper()
	spans, err := NewRegexDetector().Detect(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	return Resolve(spans)
}

func TestCueSeparatedSSNsAndPhones(t *testing.T) {
	seps := []string{"\u301c", "~", "*", ".", "\u00b7", "/", "\u2014", "\u30a0", "\u3030", "\u2212", "-"}
	for _, sep := range seps {
		for _, c := range []struct {
			text, want string
			typ        entity.Type
		}{
			{"SSN 536" + sep + "22" + sep + "8471 on file", "536" + sep + "22" + sep + "8471", entity.GovID},
			{"social security number: 536" + sep + "22" + sep + "8471", "536" + sep + "22" + sep + "8471", entity.GovID},
			{"Call 804" + sep + "555" + sep + "1234 today", "804" + sep + "555" + sep + "1234", entity.Phone},
			{"phone: 804" + sep + "555" + sep + "1234", "804" + sep + "555" + sep + "1234", entity.Phone},
		} {
			found := false
			for _, sp := range regexSpans(t, c.text) {
				if sp.Type == c.typ && sp.Text == c.want {
					found = true
				}
			}
			if !found {
				t.Errorf("%q: want %s %q", c.text, c.typ, c.want)
			}
		}
	}
	for _, text := range []string{"SSN 536\u301c22\u30008471", "SSN 536~22*8471", "build 536~22~8471", "Call 104~555~1234", "version 804~555~1234", "ratio 536*22*8471", "SSN 536~22~84715"} {
		for _, sp := range regexSpans(t, text) {
			if (sp.Type == entity.GovID || sp.Type == entity.Phone) && strings.ContainsAny(sp.Text, "~*\u301c") {
				t.Errorf("%q: unexpected %s %q", text, sp.Type, sp.Text)
			}
		}
	}
}

func TestPasswordValueStopsAtQueryDelimiters(t *testing.T) {
	for _, text := range []string{"auth_method=password&q=hello", "auth_method=password#frag", "login with mode=password&next=/home"} {
		for _, sp := range regexSpans(t, text) {
			if sp.Type == entity.Secret {
				t.Errorf("%q: unexpected secret %q", text, sp.Text)
			}
		}
	}
	for _, c := range []struct{ text, want string }{
		{"password=Hunter2x9&q=hello", "Hunter2x9"},
		{"user=a&password=Hunter2x9&q=hello", "Hunter2x9"},
		{"password: Tr0ub4dor&3xQ!", "Tr0ub4dor&3xQ!"},
	} {
		found := false
		for _, sp := range regexSpans(t, c.text) {
			if sp.Type == entity.Secret {
				if sp.Text != c.want {
					t.Errorf("%q: secret %q, want %q", c.text, sp.Text, c.want)
				}
				found = true
			}
		}
		if !found {
			t.Errorf("%q: want secret %q", c.text, c.want)
		}
	}
}

func TestDescriptorPasswordKeys(t *testing.T) {
	for _, text := range []string{"password_type=hunter2", "password_mode=hunter2", "password_id=hunter2", "password_type=bcrypt", "token_type=Bearer", "secret_id=prod-db"} {
		for _, sp := range regexSpans(t, text) {
			if sp.Type == entity.Secret {
				t.Errorf("%q: unexpected secret %q", text, sp.Text)
			}
		}
	}
	for _, c := range []struct{ text, want string }{
		{"password_type=Zk3Lm9Qp2Xv7Bn4Rt8Ws1Y", "Zk3Lm9Qp2Xv7Bn4Rt8Ws1Y"},
		{"password_mode: Xq7#vB2!pL9@mK4$", "Xq7#vB2!pL9@mK4$"},
		{"secret_id=9fK2mQx7LpV4sT8wZb3nYc6h&next=1", "9fK2mQx7LpV4sT8wZb3nYc6h"},
		{"password_id=gh" + "p_abcdefghijklmnopqrstuvwxyz0123456789", "gh" + "p_abcdefghijklmnopqrstuvwxyz0123456789"},
	} {
		found := false
		for _, sp := range regexSpans(t, c.text) {
			if sp.Type == entity.Secret && sp.Text == c.want {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: want secret %q, got %+v", c.text, c.want, regexSpans(t, c.text))
		}
	}
}

func TestSeparatedCuesAreTight(t *testing.T) {
	for _, c := range []struct {
		text, want string
		typ        entity.Type
	}{
		{"SSN 536_22_8471", "536_22_8471", entity.GovID},
		{"Call 804_555_1234 now", "804_555_1234", entity.Phone},
		{"Call 804|555|1234 now", "804|555|1234", entity.Phone},
		{"Call 804:555:1234 now", "804:555:1234", entity.Phone},
		{"ITIN 912~70~1234", "912~70~1234", entity.GovID},
		{"tax id: 912~70~1234", "912~70~1234", entity.GovID},
		{"TIN 536~22~8471", "536~22~8471", entity.GovID},
		{"tin: 536~22~8471", "536~22~8471", entity.GovID},
		{"ph. 804~555~1234", "804~555~1234", entity.Phone},
		{"ph: 804~555~1234", "804~555~1234", entity.Phone},
	} {
		found := false
		for _, sp := range regexSpans(t, c.text) {
			if sp.Type == c.typ && sp.Text == c.want {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: want %s %q", c.text, c.typ, c.want)
		}
	}
	for _, text := range []string{
		"GET /api/v1/call/734*899*7921 HTTP/1.1",
		"https://example.com/api/call/734*899*7921",
		"call_id=734*899*7921",
		"callId=734*899*7921",
		"tin 536~22~8471 cans",
		"a tin of 536~22~8471",
		"graph 804~555~1234",
		"Call ext804~555~1234",
		"Contact x804*555*1234",
		"Call a804-555-1234",
		"call 804,555,1234",
		"recall 804~555~1234",
		"Call 804~555~1234abc",
	} {
		for _, sp := range regexSpans(t, text) {
			if sp.Source == "regex:ssn_separated" || sp.Source == "regex:phone_separated" {
				t.Errorf("%q: unexpected %s %q", text, sp.Source, sp.Text)
			}
		}
	}
}

func TestPasswordLabelInProseIsNotCut(t *testing.T) {
	for _, c := range []struct{ text, want string }{
		{"password: Pa55&word=xyz", "Pa55&word=xyz"},
		{"password=Hunter2x9&q=hello", "Hunter2x9"},
		{"a=1&password=Hunter2x9&q=hello", "Hunter2x9"},
	} {
		found := false
		for _, sp := range regexSpans(t, c.text) {
			if sp.Type == entity.Secret && sp.Text == c.want {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: want secret %q, got %+v", c.text, c.want, regexSpans(t, c.text))
		}
	}
}

func TestDescriptorSecretSkipsHexAndUUIDs(t *testing.T) {
	for _, text := range []string{"password_id=0123456789abcdef0123", "secret_id=550e8400-e29b-41d4-a716-446655440000", "token_id=deadbeefdeadbeefdeadbeef", "password_type=0123456789ABCDEF0123"} {
		for _, sp := range regexSpans(t, text) {
			if sp.Source == "regex:descriptor_secret" {
				t.Errorf("%q: descriptor_secret emitted %q", text, sp.Text)
			}
		}
	}
}

func TestBarePasswordWithQueryShapeStopsAtTheNextKey(t *testing.T) {
	for _, sp := range regexSpans(t, "password=Pa55&word=xyz") {
		if sp.Type == entity.Secret && sp.Text != "Pa55" {
			t.Errorf("secret %q, want the value to stop at &word=", sp.Text)
		}
	}
}

func TestBracketPhonesAfterACue(t *testing.T) {
	for _, c := range []struct{ text, want string }{
		{"Call (804)-555-1234 now", "(804)-555-1234"},
		{"Call (804)/555/1234 now", "(804)/555/1234"},
		{"Call (804)555-1234 now", "(804)555-1234"},
		{"Call +1(804)555/1234 now", "+1(804)555/1234"},
		{"phone: 1 (804)-555-1234", "1 (804)-555-1234"},
	} {
		found := false
		for _, sp := range regexSpans(t, c.text) {
			if sp.Type == entity.Phone && sp.Text == c.want {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: want phone %q, got %+v", c.text, c.want, regexSpans(t, c.text))
		}
	}
	for _, text := range []string{"Call (123)-555-1234 now", "Call (804)-155-1234 now", "Call (804)-555-12345", "call_id=(804)-555-1234", "Call (804)-555-1234x", "see https://x.example/call/(804)-555-1234", "Call (80)-555-1234", "Call +2(804)555/1234"} {
		for _, sp := range regexSpans(t, text) {
			if sp.Source == "regex:phone_separated" {
				t.Errorf("%q: unexpected phone %q", text, sp.Text)
			}
		}
	}
}
