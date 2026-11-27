package detect

import (
	"context"
	"regexp"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

var oldLinkSecretRE = regexp.MustCompile(`(?i)(reset|magic|verify|verification|confirm|activate|invite|sign-?in|log-?in|auth|token=|code=|otp|password|passwordless|one-?time|unsubscribe|[?&](?:key|sig|signature|t|k)=)`)

var linkSecretCases = []struct {
	url    string
	secret bool
}{
	{"https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html", false},
	{"https://example.com/docs/reset-password", false},
	{"https://example.com/login?next=/settings", false},
	{"https://example.com/settings?reset=true", false},
	{"https://example.com/api?token=", false},
	{"https://example.com/api?token=<YOUR_TOKEN>", false},
	{"https://example.com/api?api_key=YOUR_API_KEY", false},
	{"https://example.com/api?api_key=${API_KEY}", false},
	{"https://docs.github.com/en/authentication/keeping-your-account-and-data-secure", false},
	{"https://oauth.net/2/authorization-code/", false},
	{"https://example.com/author/jane", false},
	{"https://example.com/invite", false},
	{"https://example.com/confirm-email?next=/home", false},
	{"https://example.com/verify/email-address-change", false},
	{"https://example.com/search?q=password+manager&t=web", false},
	{"https://example.com/oauth/callback?code=ok", false},
	{"https://www.unsubscribe.example/help", false},
	{"https://user@example.com/repo.git", false},
	{"https://example.com/keys?key=primary", false},
	{"https://accounts.example.com/reset-password?token=6NKAP1QXufUSevGNn7xufZMypluBFlkn&uid=975221", true},
	{"https://hooks.example.com/esc?key=whk_5d8e2a9c1f7b4036", true},
	{"https://example.com/login?password=hunter2", true},
	{"https://example.com/x?secret_key=9fK2mQx7LpV4sT8wZb3n", true},
	{"https://example.com/x?private_key=9fK2mQx7LpV4sT8wZb3n", true},
	{"https://example.com/x?secretKey=9fK2mQx7LpV4sT8wZb3n", true},
	{"https://example.com/x?app_key=9fK2mQx7LpV4sT8wZb3n", true},
	{"https://example.com/x?client_key=9fK2mQx7LpV4sT8wZb3n", true},
	{"https://example.com/x?auth_key=9fK2mQx7LpV4sT8wZb3n", true},
	{"https://example.com/x?subscription-key=9fK2mQx7LpV4sT8wZb3n", true},
	{"https://example.com/x?session=9fK2mQx7LpV4sT8wZb3n", true},
	{"https://api.telegram.org/bot48" + "21" + "937502:AAHk3Lm9Qp2Xv7Bn4Rt8Ws1Yc6Hd0Jf5GaQ/sendMessage", true},
	{"https://example.com/x?key=abc", false},
	{"https://example.com/x?session=1", false},
	{"https://example.com/x?keyword=privacy+gateway", false},
	{"https://example.com/x?monkey=banana12", false},
	{"https://example.com/x?author=jane", false},
	{"https://example.com/x?signal=strong", false},
	{"https://example.com/x?design=minimal", false},
	{"https://example.com/x?sort_key=name", false},
	{"https://example.com/x?sort_key=created_at", false},
	{"https://example.com/x?api_key=correct_horse_battery", true},
	{"https://example.com/x?api_key=deadbeef", true},
	{"https://example.com/x?app_key=deadbeefcafe", true},
	{"https://example.com/x?license_key=deadbeefcafe", true},
	{"https://example.com/x?encryption_key=deadbeefcafe", true},
	{"https://example.com/x?master_key=deadbeefcafe", true},
	{"https://example.com/x?signing_key=deadbeefcafe", true},
	{"https://example.com/x?client_key=deadbeefcafe", true},
	{"https://example.com/x?consumer_key=deadbeefcafe", true},
	{"https://example.com/x?subscription_key=deadbeefcafe", true},
	{"https://example.com/x?primary_key=id", false},
	{"https://example.com/x?cache_key=9fK2mQx7LpV4sT8wZb3n", false},
	{"https://example.com/x?publicKey=9fK2mQx7LpV4sT8wZb3n", false},
	{"https://example.com/x?token_type=access_token", false},
	{"https://example.com/x?auth_method=password", false},
	{"https://example.com/x?app_key=my_app_name", false},
	{"https://example.com/x?secret_key=abcdefgh", true},
	{"https://example.com/x?app_key=a1b2c3d4", true},
	{"https://example.com/x?token=123456", false},
	{"https://example.com/x?token=abcdefg", false},
	{"https://example.com/x?key=abc123", false},
	{"https://example.com/x?token=1&q=hello", false},
	{"https://example.com/x?q=hello&token=9fK2mQx7LpV4sT8wZb3n#frag", true},
	{"https://example.com/login?password=secret", true},
	{"https://example.com/x?pass=1", false},
	{"https://example.com/x?sig=v1", false},
	{"https://example.com/x?token=1", false},
	{"https://example.com/x?session_token=0", false},
	{"https://example.com/x?signature=sha256", false},
	{"https://example.com/x?password=${PASSWORD}", false},
	{"https://example.com/x?auth=Zm9vOmJhcg==", true},
	{"https://example.com/x?auth=basic", false},
	{"https://example.com/token/abc123def456ghi789jkl", true},
	{"https://example.com/secret/9fK2mQx7LpV4sT8wZb3nYc6h", true},
	{"https://example.com/keys/primary", false},
	{"https://example.com/docs/tokens/rotation-guide", false},
	{"https://example.com/reset?token=abc123", true},
	{"https://app.example.com/reset-password?t=abc", true},
	{"https://example.com/auth/magic/xyz", true},
	{"https://example.com/search?t=web", false},
	{"https://example.com/magic-link", false},
	{"https://user:s3cretPass@example.com/path", true},
	{"https://example.com/cb#access_token=ya" + "29.a0AfH6SMBx3kQz9Lm2Np&token_type=Bearer", true},
	{"https://bucket.s3.amazonaws.com/f.pdf?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIAEXAMPLE%2F20240101%2Fus-east-1%2Fs3%2Faws4_request&X-Amz-Signature=3f1a9c0d2b7e4f6a8c1d3e5f7a9b0c2d4e6f8a1b3c5d7e9f0a2b4c6d8e0f1a2b", true},
	{"https://acct.blob.core.windows.net/c/f.txt?sv=2022-11-02&se=2024-01-01&sig=Zm9vYmFyYmF6cXV4MTIzNDU2Nzg5MA%3D%3D", true},
	{"https://example.com/oauth/callback?code=4/0AY0e-g7Qx2nLm9PzK3vT8wR5yH1jF6", true},
	{"https://example.com/verify/9fK2mQx7LpV4sT8wZb3nYc6h", true},
	{"https://example.com/api?client_secret=abcd1234efgh", true},
	{"https://example.com/x?id=eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U", true},
	{"https://example.com/x?ref=gh" + "p_abcdefghijklmnopqrstuvwxyz0123456789", true},
}

func TestLinkSecretOnlyForCredentialValues(t *testing.T) {
	for _, c := range linkSecretCases {
		got := linkSecret(c.url)
		t.Logf("old=%-5v new=%-5v %s", oldLinkSecretRE.MatchString(c.url), got, c.url)
		if got != c.secret {
			t.Errorf("%s: secret=%v, want %v", c.url, got, c.secret)
		}
	}
}

func TestURLsWithCredentialWordsComeThroughUnchanged(t *testing.T) {
	for _, u := range []string{"https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html", "https://example.com/docs/reset-password", "https://example.com/login?next=/settings"} {
		text := "see " + u + " for details"
		for _, sp := range detectAll(t, text) {
			if sp.Type != entity.URL || sp.Text != u {
				t.Errorf("%q: unexpected span %+v", text, sp)
			}
		}
	}
	for _, c := range linkSecretCases {
		if !c.secret {
			continue
		}
		found := false
		for _, sp := range detectAll(t, "open "+c.url+" now") {
			if sp.Type == entity.Secret {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: credential link not detected as a secret", c.url)
		}
	}
}

func TestOneURLGivesAtMostOneSecretFinding(t *testing.T) {
	for _, c := range linkSecretCases {
		text := "see " + c.url + " now"
		secrets := 0
		for _, sp := range detectAll(t, text) {
			if sp.Type == entity.Secret {
				secrets++
				if c.secret && sp.Text != c.url {
					t.Errorf("%s: secret finding %q is not the whole link", c.url, sp.Text)
				}
			}
		}
		if c.secret && secrets != 1 || !c.secret && secrets != 0 {
			t.Errorf("%s: %d secret findings, secret=%v", c.url, secrets, c.secret)
		}
	}
	for _, text := range []string{"password: hunter2&go", "token = Zx9-aQ7&rT2#mK"} {
		found := false
		for _, sp := range detectAll(t, text) {
			if sp.Type == entity.Secret {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: a labelled secret outside a URL must still be found", text)
		}
	}
}

func TestEmailsAndPhonesInsideURLs(t *testing.T) {
	cases := []struct {
		text string
		typ  entity.Type
		want string
	}{
		{"Look at https://example.com/?email=jane.doe@example.com please", entity.Email, "jane.doe@example.com"},
		{"https://mcp.example.com/jane.doe@example.com", entity.Email, "jane.doe@example.com"},
		{"https://example.com/x#jane.doe@example.com", entity.Email, "jane.doe@example.com"},
		{"https://example.com/u/555-867-5309", entity.Phone, "555-867-5309"},
		{"https://example.com/u/555-867-5309/edit", entity.Phone, "555-867-5309"},
		{"https://example.com/?tel=555-867-5309", entity.Phone, "555-867-5309"},
	}
	for _, c := range cases {
		spans, _ := NewRegexDetector().Detect(context.Background(), c.text)
		found := false
		for _, sp := range spans {
			if sp.Type == c.typ && sp.Text == c.want {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: want %s span %q", c.text, c.typ, c.want)
		}
	}
	for _, text := range []string{"https://user@example.com/path", "https://jane.doe@example.com/path", "https://example.com/v1/2024-11-05/x", "https://example.com/items/123-456", "https://example.com/u/555-867-5309.json", "https://example.com/u/123-456-7890", "https://example.com/b/2024-1115-1200"} {
		spans, _ := NewRegexDetector().Detect(context.Background(), text)
		for _, sp := range spans {
			if sp.Type == entity.Email || sp.Type == entity.Phone {
				t.Errorf("%q: unexpected %s span %q", text, sp.Type, sp.Text)
			}
		}
	}
}

func TestEncodedEmailsAndGitRemotes(t *testing.T) {
	for _, c := range []struct{ text, want string }{
		{"see https://example.com/?email=elena.petrova%40example.com now", "elena.petrova%40example.com"},
		{"https://example.com/u/elena.petrova%40example.com/edit", "elena.petrova%40example.com"},
		{"https://example.com/x#to=elena.petrova%40example.com", "elena.petrova%40example.com"},
		{"Contact: jane@example.com: urgent", "jane@example.com"},
	} {
		spans, _ := NewRegexDetector().Detect(context.Background(), c.text)
		found := false
		for _, sp := range spans {
			if sp.Type == entity.Email && sp.Text == c.want {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: want email %q", c.text, c.want)
		}
	}
	for _, text := range []string{"git remote add origin git@github.com:org/repo.git", "clone gitlab@git.example.com:team/app.git", "svn@svn.example.org:trunk/app"} {
		spans, _ := NewRegexDetector().Detect(context.Background(), text)
		for _, sp := range spans {
			if sp.Type == entity.Email {
				t.Errorf("%q: unexpected email %q", text, sp.Text)
			}
		}
	}
}

func TestPercentEncodedAndScpEmails(t *testing.T) {
	for _, c := range []struct{ text, want string }{
		{"https://x.example/?e=elena%2Bwork%40example.com", "elena%2Bwork%40example.com"},
		{"https://x.example/?e=elena%2Epetrova%40example.com", "elena%2Epetrova%40example.com"},
		{"https://x.example/?e=elena.petrova%2540example.com", "elena.petrova%2540example.com"},
		{"IMAP elena@example.com:993", "elena@example.com"},
		{"user=elena@example.com:login", "elena@example.com"},
		{"From: elena@example.com:Re: hi", "elena@example.com"},
		{"remote elena.petrova@gitlab.example.com:team/repo.git", "elena.petrova@gitlab.example.com"},
	} {
		spans, _ := NewRegexDetector().Detect(context.Background(), c.text)
		found := false
		for _, sp := range spans {
			if sp.Type == entity.Email && sp.Text == c.want {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: want email %q, got %+v", c.text, c.want, spans)
		}
	}
	for _, text := range []string{"sale 50%40off today", "git@github.com:org/repo.git", "hg@hg.example.org:proj/repo", "forgejo@code.example.org:team/app.git"} {
		spans, _ := NewRegexDetector().Detect(context.Background(), text)
		for _, sp := range spans {
			if sp.Type == entity.Email {
				t.Errorf("%q: unexpected email %q", text, sp.Text)
			}
		}
	}
}

func TestUnicodeEmailDetection(t *testing.T) {
	for _, c := range []struct{ text, want string }{
		{"mail \u00e9lena@example.com now", "\u00e9lena@example.com"},
		{"mail jos\u00e9.garc\u00eda@example.es now", "jos\u00e9.garc\u00eda@example.es"},
		{"mail \u0435\u043b\u0435\u043d\u0430@\u043f\u0440\u0438\u043c\u0435\u0440.\u0440\u0444 now", "\u0435\u043b\u0435\u043d\u0430@\u043f\u0440\u0438\u043c\u0435\u0440.\u0440\u0444"},
		{"https://x.example/?to=elena%40example.com%20john%40example.com", "john%40example.com"},
	} {
		spans, _ := NewRegexDetector().Detect(context.Background(), c.text)
		found := false
		for _, sp := range spans {
			if sp.Type == entity.Email && sp.Text == c.want {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: want email %q, got %+v", c.text, c.want, spans)
		}
	}
	for _, text := range []string{"caf\u00e9@ home is nice", "@d\u00e9corateur class Foo", "caf\u00e9@maison sans domaine"} {
		spans, _ := NewRegexDetector().Detect(context.Background(), text)
		for _, sp := range spans {
			if sp.Type == entity.Email {
				t.Errorf("%q: unexpected email %q", text, sp.Text)
			}
		}
	}
}

func TestURLQueryKeyTable(t *testing.T) {
	const strong = "9fK2mQx7LpV4sT8wZb3nYc6h"
	clean := []string{"api-version", "api_version", "apiVersion", "token_type", "auth_type", "auth_method", "auth_url", "token_endpoint", "client_id", "session_url", "secret_id", "key_id"}
	for _, k := range clean {
		u := "https://example.com/x?" + k + "=" + strong
		if linkSecret(u) {
			t.Errorf("%s: key %q must stay clean", u, k)
		}
	}
	flag := []string{"session_id", "sessionid", "api_key", "apiKey", "x-api-key", "api_token", "access_token", "auth_token", "client_secret", "secret_key", "private_key", "password", "sig", "signature", "X-Amz-Signature", "X-Amz-Credential", "X-Amz-Security-Token", "license_key", "subscription-key"}
	for _, k := range flag {
		u := "https://example.com/x?" + k + "=" + strong
		if !linkSecret(u) {
			t.Errorf("%s: key %q must flag", u, k)
		}
	}
	real := []struct {
		url    string
		secret bool
	}{
		{"https://x.openai.azure.com/openai/deployments/gpt/chat?api-version=2024-02-01", false},
		{"https://x.openai.azure.com/openai/deployments/gpt/chat/completions?api-version=2024-02-15-preview", false},
		{"https://x.openai.azure.com/openai/deployments/gpt/chat?api-version=preview", false},
		{"https://x.openai.azure.com/openai/deployments/gpt/chat?api-version=2024-02-01&api-key=" + strong, true},
		{"https://example.com/v1?version=2024-02-01", false},
		{"https://accounts.example.com/o/oauth2/v2/auth?client_id=1234567890-abcdefghijklmnop.apps.example.com&redirect_uri=https%3A%2F%2Fapp.example.com%2Fcb&response_type=code&scope=openid+email&state=xyz", false},
		{"https://app.example.com/oauth/callback?code=4/0AY0e-g7Qx2nLm9PzK3vT8wR5yH1jF6&state=xyz", true},
		{"https://app.example.com/oauth/callback?code=200", false},
		{"https://app.example.com/settings?code=en", false},
		{"https://auth.example.com/oauth/token?grant_type=authorization_code&client_id=abc123&client_secret=" + strong, true},
		{"https://auth.example.com/oauth/token?grant_type=refresh_token&refresh_token=" + strong, true},
		{"https://bucket.s3.amazonaws.com/f.pdf?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AK" + "IAIOSFODNN7EXAMPLE%2F20240101%2Fus-east-1%2Fs3%2Faws4_request&X-Amz-Date=20240101T000000Z&X-Amz-Expires=3600&X-Amz-SignedHeaders=host&X-Amz-Signature=3f1a9c0d2b7e4f6a8c1d3e5f7a9b0c2d", true},
		{"https://bucket.s3.amazonaws.com/f.pdf?X-Amz-Security-Token=IQoJb3JpZ2luX2VjEJr%2F" + strong, true},
		{"https://bucket.s3.amazonaws.com/f.pdf?AWSAccessKeyId=AK" + "IAIOSFODNN7EXAMPLE&Expires=1700000000", true},
		{"https://example.com/x?key_id=AK" + "IAIOSFODNN7EXAMPLE", true},
		{"https://maps.example.com/api/js?key=AI" + "zaSyA1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q", true},
		{"https://maps.example.com/api/js?key=primary", false},
		{"https://api.stripe.com/v1/charges?client_secret=pi_3Mtw_secret_YrKJUKribcBjcG8HVhfZluoGH", true},
		{"https://api.stripe.com/v1/charges?limit=3&starting_after=ch_123", false},
		{"https://ho" + "oks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX", true},
		{"https://hooks.slack.com/services", false},
		{"https://example.com/x?api_key=correct_horse_battery", true},
		{"https://example.com/x?sort_key=created_at", false},
		{"https://example.com/x?session_id=1", false},
	}
	for _, c := range real {
		if got := linkSecret(c.url); got != c.secret {
			t.Errorf("%s: secret=%v, want %v", c.url, got, c.secret)
		}
	}
}
