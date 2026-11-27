package detect

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

func spanTexts(spans []entity.Span, typ entity.Type) []string {
	var out []string
	for _, s := range spans {
		if s.Type == typ {
			out = append(out, s.Text)
		}
	}
	return out
}

func TestCredScoreDollarAtEnd(t *testing.T) {
	for _, text := range []string{"Total due: 1234567890$ thanks", "login abc12345$", "pw x$", "login $"} {
		detectAll(t, text)
	}
	if got := credScore("abc12345$", false); got == 0 {
		t.Errorf("trailing dollar should still score, got %d", got)
	}
	if got := credScore("Price$5each", false); got != 0 {
		t.Errorf("dollar amount scored %d", got)
	}
	runPositives(t, []classCase{{"the password is Hunter2x$", entity.Secret, "Hunter2x$"}})
}

func TestPatternPanicBecomesError(t *testing.T) {
	boom := pattern{name: "boom", scan: func(string, string) []entity.Span { panic("secret value") }}
	fine := pattern{name: "fine", typ: entity.Secret, scan: func(string, string) []entity.Span { return []entity.Span{{Start: 0, End: 1}} }}
	for _, text := range []string{"short text", strings.Repeat("long text ", 2*parallelMin/10)} {
		spans, err := (&RegexDetector{pats: []pattern{fine, boom, fine}}).Detect(context.Background(), text)
		if err == nil || spans != nil {
			t.Fatalf("len %d: want error and no spans, got %d spans, err %v", len(text), len(spans), err)
		}
		if strings.Contains(err.Error(), "secret value") {
			t.Errorf("error carries the panic value: %v", err)
		}
		spans, err = (&RegexDetector{pats: []pattern{fine, fine}}).Detect(context.Background(), text)
		if err != nil || len(spans) != 2 {
			t.Errorf("len %d: healthy patterns gave %d spans, err %v", len(text), len(spans), err)
		}
	}
}

func TestKeyValueSecretRunsToTokenEnd(t *testing.T) {
	long := []byte(strings.Repeat("a1B2", 50))
	long[159] = '$'
	accented := "a1" + strings.Repeat("é", 100)
	runPositives(t, []classCase{
		{"db_password=" + string(long), entity.Secret, string(long)},
		{"db_password=" + string(long) + "\nnext line", entity.Secret, string(long)},
		{"db_password = " + accented, entity.Secret, accented},
		{"db_password = " + accented + " # prod", entity.Secret, accented},
	})
	for _, text := range []string{"db_password = " + accented, "x db_password=" + string(long) + " y"} {
		for _, s := range detectAll(t, text) {
			if !utf8.ValidString(s.Text) || !utf8.ValidString(text[:s.Start]) {
				t.Errorf("span splits a rune: %q", s.Text)
			}
		}
	}
	for _, text := range []string{"db_password=short1", "db_password=" + strings.Repeat("é", 7), "db_password = ${DB_PASSWORD_FROM_THE_VAULT}"} {
		if got := scanKeyValueSecrets(text, strings.ToLower(text)); len(got) != 0 {
			t.Errorf("%q: unexpected key=value secret %+v", text, got)
		}
	}
}

func TestCredentialSpansExcludeCarriageReturn(t *testing.T) {
	runPositives(t, []classCase{
		{"password Hunter2x\r\n", entity.Secret, "Hunter2x"},
		{"Password:\r\nHunter2x!9\r\nthanks", entity.Secret, "Hunter2x!9"},
		{"backup codes:\r\na1b2c-3d4e5\r\nf6g7h-8i9j0\r\n", entity.Secret, "f6g7h-8i9j0"},
	})
	for _, text := range []string{"password Hunter2x\r\n", "login admin Hunter2x!\r\nnext", "backup codes: a1b2c-3d4e5\r\n", "wifi Tr0ub4dor&3x\r\n"} {
		for _, s := range detectAll(t, text) {
			if strings.ContainsAny(s.Text, "\r\n") {
				t.Errorf("%q: span %q from %s holds a line break", text, s.Text, s.Source)
			}
		}
	}
}

func TestOpenPrivateKeyTakesShortLastLine(t *testing.T) {
	body := "MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7VJTUt9Us8cKj"
	head := "--" + "---BEGIN RSA PRIVATE KEY-----\n" + body + "\n"
	runPositives(t, []classCase{
		{head + "AbCd12==\n", entity.Secret, head + "AbCd12=="},
		{head + "Qx9", entity.Secret, head + "Qx9"},
		{head + "AbCd12==\r\nsent from my phone", entity.Secret, head + "AbCd12=="},
		{`{"key":"--` + `---BEGIN RSA PRIVATE KEY-----\n` + body + `\nAbCd12=="}`, entity.Secret, `--` + `---BEGIN RSA PRIVATE KEY-----\n` + body + `\nAbCd12==`},
	})
	for _, tail := range []string{"Thanks\n", "Regards", "OK\n", "\nv2\n", "Call me at 5\n", "see ticket42 for more\n"} {
		for _, s := range detectAll(t, head+tail) {
			if s.Source == "regex:private_key_open" && strings.TrimRight(s.Text, "\n") != strings.TrimRight(head, "\n") {
				t.Errorf("tail %q: key span grew to %q", tail, s.Text)
			}
		}
	}
}

func TestIntlPhoneStopsBeforeNextNumber(t *testing.T) {
	runPositives(t, []classCase{
		{"+44 20 7946 0958\n10.0.0.5", entity.Phone, "+44 20 7946 0958"},
		{"+44 20 7946 0958\n10.0.0.5", entity.IP, "10.0.0.5"},
		{"+44 20 7946 0958 10.0.0.5", entity.Phone, "+44 20 7946 0958"},
		{"+44 20 7946 0958 10.0.0.5", entity.IP, "10.0.0.5"},
		{"+44 20 7946 0958 (804) 555-1234", entity.Phone, "+44 20 7946 0958"},
		{"+44 20 7946 0958 (804) 555-1234", entity.Phone, "(804) 555-1234"},
		{"+44 20 7946 0958\t+33 1 23 45 67 89", entity.Phone, "+33 1 23 45 67 89"},
		{"+44 20 7946 0958 1234 5678 9012", entity.Phone, "+44 20 7946 0958"},
		{"call +1 (804) 555-1234 now", entity.Phone, "+1 (804) 555-1234"},
		{"call +49 (0)30 12345 678", entity.Phone, "+49 (0)30 12345 678"},
		{"fax +33 1 23 45 67 89", entity.Phone, "+33 1 23 45 67 89"},
	})
	runNegatives(t, entity.Phone, []string{"offset +12 34\n56 78", "delta +5 10 15", "rows +123 4\n5678 9012"})
}

func TestOrgSuffixStaysOnOneLine(t *testing.T) {
	text := "Attendees: Laura Palmer, Liam Parker\nLP to send notes."
	for _, s := range detectAll(t, text) {
		if s.Type == entity.Org || strings.Contains(s.Text, "\n") {
			t.Errorf("unexpected span %q (%s) from %s", s.Text, s.Type, s.Source)
		}
	}
	runPositives(t, []classCase{
		{"invoice from Harbor Freight LP today", entity.Org, "Harbor Freight LP"},
		{"Smith and Jones LLP signed", entity.Org, "Smith and Jones LLP"},
		{"Acme\tWidgets Ltd.", entity.Org, "Acme\tWidgets Ltd."},
	})
	runNegatives(t, entity.Org, []string{"Thanks, Maria\nInc is not here", "Owner: Dana Reyes\nCo to follow"})
}

func TestAddressHouseWithStairAndDoor(t *testing.T) {
	runPositives(t, []classCase{
		{"Mariahilfer Straße 45/2/14, 1060 Wien", entity.Address, "Mariahilfer Straße 45/2/14, 1060 Wien"},
		{"Mariahilfer Straße 45/2, 1060 Wien", entity.Address, "Mariahilfer Straße 45/2, 1060 Wien"},
		{"Mariahilfer Straße 45, 1060 Wien", entity.Address, "Mariahilfer Straße 45, 1060 Wien"},
	})
}
