package detect

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

func detectAll(t *testing.T, text string) []entity.Span {
	t.Helper()
	spans, err := NewRegexDetector().Detect(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	return Resolve(spans)
}

func hasSpan(spans []entity.Span, typ entity.Type, text string) bool {
	for _, s := range spans {
		if s.Type == typ && s.Text == text {
			return true
		}
	}
	return false
}

func anyType(spans []entity.Span, typ entity.Type) bool {
	for _, s := range spans {
		if s.Type == typ {
			return true
		}
	}
	return false
}

func TestRegexPositives(t *testing.T) {
	cases := []struct {
		text string
		typ  entity.Type
		want string
	}{
		{"mail me at jane.doe+news@mail.co.uk today", entity.Email, "jane.doe+news@mail.co.uk"},
		{"call (804) 555-1234 now", entity.Phone, "(804) 555-1234"},
		{"call 804-555-1234.", entity.Phone, "804-555-1234"},
		{"call +1 804.555.1234", entity.Phone, "+1 804.555.1234"},
		{"call 8045551234", entity.Phone, "8045551234"},
		{"ring +44 20 7946 0958 please", entity.Phone, "+44 20 7946 0958"},
		{"see https://example.org/a/b?x=1.", entity.URL, "https://example.org/a/b?x=1"},
		{"(see https://example.org/wiki/Foo_(bar))", entity.URL, "https://example.org/wiki/Foo_(bar)"},
		{"click https://app.example.com/reset-password?t=abc", entity.Secret, "https://app.example.com/reset-password?t=abc"},
		{"magic link: https://example.com/auth/magic/xyz", entity.Secret, "https://example.com/auth/magic/xyz"},
		{"server 10.20.30.40 is down", entity.IP, "10.20.30.40"},
		{"v6 2001:db8:85a3::8a2e:370:7334 ok", entity.IP, "2001:db8:85a3::8a2e:370:7334"},
		{"SSN 123-45-6789", entity.GovID, "123-45-6789"},
		{"my social security number is 123456789", entity.GovID, "123456789"},
		{"passport number X12345678", entity.GovID, "X12345678"},
		{"SSN: 666827567", entity.GovID, "666827567"},
		{"my SSN is 987 65 4329 and", entity.GovID, "987 65 4329"},
		{"SSN 000-12-3456", entity.GovID, "000-12-3456"},
		{"social security no. 531.00.1297", entity.GovID, "531.00.1297"},
		{"My temporary password was Tundra$Tundra85 and", entity.Secret, "Tundra$Tundra85"},
		{"the password is now Kettle!Fable11.", entity.Secret, "Kettle!Fable11"},
		{"    export OPENAI_API_KEY=3cb9b85ddd02d8a461b4c9d70cb95ffa", entity.Secret, "3cb9b85ddd02d8a461b4c9d70cb95ffa"},
		{"ADMIN_PASSWORD=Plover$Tundra79", entity.Secret, "Plover$Tundra79"},
		{"found the expired credential: `7f848d794ecc79a2267a1a9b7df8b7cd`.", entity.Secret, "7f848d794ecc79a2267a1a9b7df8b7cd"},
		{"Dr. Ms. Halloran will see you", entity.Person, "Halloran"},
		{"card 4111 1111 1111 1111 exp", entity.Financial, "4111 1111 1111 1111"},
		{"card 4111-1111-1111-1111", entity.Financial, "4111-1111-1111-1111"},
		{"IBAN GB82 WEST 1234 5698 7654 32", entity.Financial, "GB82 WEST 1234 5698 7654 32"},
		{"routing number 021000021", entity.Financial, "021000021"},
		{"account number: 000123456789", entity.Financial, "000123456789"},
		{"key sk-proj-abcdefghijklmnopqrstuvwxyz0123", entity.Secret, "sk-proj-abcdefghijklmnopqrstuvwxyz0123"},
		{"token ghp_abcdefghijklmnopqrstuvwxyz0123456789", entity.Secret, "ghp_abcdefghijklmnopqrstuvwxyz0123456789"},
		{"aws AKIAIOSFODNN7EXAMPLE", entity.Secret, "AKIAIOSFODNN7EXAMPLE"},
		{"Authorization: Bearer abcdefABCDEF0123456789xyz", entity.Secret, "abcdefABCDEF0123456789xyz"},
		{`{"api_key": "Zx81kLq0Pm2Nb7Vc"}`, entity.Secret, "Zx81kLq0Pm2Nb7Vc"},
		{"password: hunter2!", entity.Secret, "hunter2!"},
		{"postgres://admin:s3cretPass@db.internal:5432/app", entity.Secret, "s3cretPass"},
		{"Your verification code is 482913.", entity.Secret, "482913"},
		{"482913 is your login code", entity.Secret, "482913"},
		{"Use code G-123456 to sign in", entity.Secret, "G-123456"},
		{"blob q8Zr2LmN0vX7pK4sW9tY1bC6dF3gH5jQ end", entity.Secret, "q8Zr2LmN0vX7pK4sW9tY1bC6dF3gH5jQ"},
		{"I live at 123 Main Street, Apt 4B, Richmond, VA 23220 now", entity.Address, "123 Main Street, Apt 4B, Richmond, VA 23220"},
		{"send to 42 N Oak Ave.", entity.Address, "42 N Oak Ave."},
		{"P.O. Box 1234", entity.Address, "P.O. Box 1234"},
		{"we met in Norfolk, Virginia last year", entity.Location, "Norfolk, Virginia"},
		{"works at Bluefin Robotics Inc.", entity.Org, "Bluefin Robotics Inc."},
		{"I saw Dr. Okafor yesterday", entity.Person, "Okafor"},
		{"Hi Marguerite,\nthanks", entity.Person, "Marguerite"},
	}
	for _, c := range cases {
		spans := detectAll(t, c.text)
		if !hasSpan(spans, c.typ, c.want) {
			t.Errorf("%q: expected %s %q, got %+v", c.text, c.typ, c.want, spans)
		}
	}
}

func TestRegexNegatives(t *testing.T) {
	cases := []struct {
		text string
		typ  entity.Type
	}{
		{"card 4111 1111 1111 1112", entity.Financial},
		{"call me at 555 12 3456 later", entity.GovID},
		{"password manager is great", entity.Secret},
		{"the monkey=banana flag", entity.Secret},
		{"version 1.2.3.4.5 released", entity.IP},
		{"the zip code 23220 area", entity.Secret},
		{"error code 5001 occurred", entity.Secret},
		{"id 550e8400-e29b-41d4-a716-446655440000", entity.Secret},
		{"commit 9fceb02d0ae598e95dc970b74767f19372d61af8", entity.Secret},
		{"meet at 10:30:00 tomorrow", entity.IP},
		{"the password reset email", entity.Secret},
		{"a-really-long-article-slug-about-things", entity.Secret},
		{"order 12345678901234 shipped", entity.Phone},
	}
	for _, c := range cases {
		if spans := detectAll(t, c.text); anyType(spans, c.typ) {
			t.Errorf("%q: unexpected %s in %+v", c.text, c.typ, spans)
		}
	}
}

func TestChecks(t *testing.T) {
	if !Luhn("4539 1488 0343 6467") || Luhn("4539 1488 0343 6468") {
		t.Fatal("luhn")
	}
	if !ValidIBAN("DE89370400440532013000") || ValidIBAN("DE89370400440532013001") {
		t.Fatal("iban")
	}
	if !ValidABA("021000021") || ValidABA("021000022") {
		t.Fatal("aba")
	}
}

func TestResolvePrefersLongerThenPriority(t *testing.T) {
	spans := []entity.Span{
		{Start: 0, End: 4, Type: entity.Person, Priority: 65},
		{Start: 0, End: 20, Type: entity.Email, Priority: 50},
		{Start: 25, End: 30, Type: entity.Secret, Priority: 10},
		{Start: 25, End: 30, Type: entity.Phone, Priority: 55},
		{Start: 28, End: 35, Type: entity.Date, Priority: 50},
	}
	got := Resolve(spans)
	if len(got) != 2 || got[0].Type != entity.Email || got[1].Type != entity.Date {
		t.Fatalf("got %+v", got)
	}
}

const peopleCSV = `value,type,aliases
John Smith,person,Johnny|J. Smith
Will Turner,person,
Acme Health,org,AH
Richmond,location,,a city in Virginia
`

func TestDictionaryCSV(t *testing.T) {
	d, err := ParseDictionaryCSV(strings.NewReader(peopleCSV))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Entries) != 4 || d.Entries[3].Generalize != "a city in Virginia" || d.Entries[0].Aliases[1] != "J. Smith" {
		t.Fatalf("entries %+v", d.Entries)
	}
	det := d.Detector()
	cases := []struct {
		text, want string
		typ        entity.Type
	}{
		{"I met John Smith today", "John Smith", entity.Person},
		{"JOHN SMITH signed", "JOHN SMITH", entity.Person},
		{"john smith signed", "john smith", entity.Person},
		{"John's report", "John", entity.Person},
		{"ask Smith about it", "Smith", entity.Person},
		{"Johnny said hi", "Johnny", entity.Person},
		{"see facebook.com/john.smith ok", "john.smith", entity.Person},
		{"handle @john_smith", "john_smith", entity.Person},
		{"user johnsmith logged in", "johnsmith", entity.Person},
		{"Will Turner sailed", "Will Turner", entity.Person},
		{"Will went home", "Will", entity.Person},
		{"Acme Health billed me", "Acme Health", entity.Org},
	}
	for _, c := range cases {
		spans, _ := det.Detect(context.Background(), c.text)
		spans = Resolve(spans)
		if !hasSpan(spans, c.typ, c.want) {
			t.Errorf("%q: want %q got %+v", c.text, c.want, spans)
		}
	}
	for _, neg := range []string{"I will go", "johnson is here", "the turner tool", "johnny-come-lately"} {
		spans, _ := det.Detect(context.Background(), neg)
		for _, s := range spans {
			if s.Text != "johnny" {
				t.Errorf("%q: unexpected %+v", neg, s)
			}
		}
	}
	spans, _ := det.Detect(context.Background(), "Richmond is nice")
	if len(spans) != 1 || spans[0].Generalized != "a city in Virginia" {
		t.Fatalf("generalize column: %+v", spans)
	}
}

func TestDictionaryGenderColumn(t *testing.T) {
	d, err := ParseDictionaryCSV(strings.NewReader("value,type,aliases,generalize,gender\nAlex Doe,person,,,f\nSam Roe,person\n"))
	if err != nil {
		t.Fatal(err)
	}
	if d.Entries[0].Gender != "f" || d.Entries[1].Gender != "" {
		t.Fatalf("entries %+v", d.Entries)
	}
	spans, _ := d.Detector().Detect(context.Background(), "Alex Doe called")
	if len(spans) == 0 || spans[0].Gender != "f" {
		t.Fatalf("spans %+v", spans)
	}
	if _, err := ParseDictionaryCSV(strings.NewReader("value,type,aliases,generalize,gender\nA B,person,,,q\n")); err == nil {
		t.Fatal("expected error for unknown gender")
	}
	d, err = ParseDictionaryCSV(strings.NewReader("value,type,gender,aliases\nAlex Doe,person,m,AD\n"))
	if err != nil || d.Entries[0].Gender != "m" || d.Entries[0].Aliases[0] != "AD" {
		t.Fatalf("header order: %+v %v", d.Entries, err)
	}
}

func TestDictionaryJSON(t *testing.T) {
	d, err := ParseDictionaryJSON(strings.NewReader(`[{"value":"Ana Lima","type":"person","aliases":["Aninha"]},{"value":"Beta Corp","type":"org","aliases":"BC|Beta"}]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Entries) != 2 || len(d.Entries[1].Aliases) != 2 {
		t.Fatalf("%+v", d.Entries)
	}
	if _, err := ParseDictionaryCSV(strings.NewReader("Bob,alien,\n")); err == nil {
		t.Fatal("expected unknown type error")
	}
}

func TestNERDetectorRuneOffsets(t *testing.T) {
	text := "Café owner Zoë Brandt lives in Ålesund."
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req nerRequest
		json.NewDecoder(r.Body).Decode(&req)
		runes := []rune(req.Text)
		find := func(s string) (int, int) {
			idx := strings.Index(string(runes), s)
			start := utf8.RuneCountInString(string(runes)[:idx])
			return start, start + utf8.RuneCountInString(s)
		}
		s1, e1 := find("Zoë Brandt")
		s2, e2 := find("Ålesund")
		json.NewEncoder(w).Encode(map[string]any{"entities": []map[string]any{
			{"start": s1, "end": e1, "label": "person", "score": 0.93},
			{"start": s2, "end": e2, "label": "location", "score": 0.88},
			{"start": 0, "end": 4, "label": "person", "score": 0.1},
		}})
	}))
	defer srv.Close()
	d := NewNERDetector(srv.URL, 0)
	spans, err := d.Detect(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 2 || spans[0].Text != "Zoë Brandt" || spans[1].Text != "Ålesund" {
		t.Fatalf("%+v", spans)
	}
	down := NewNERDetector("http://127.0.0.1:1", 0)
	if _, err := down.Detect(context.Background(), text); err == nil {
		t.Fatal("expected error from unreachable sidecar")
	}
}
