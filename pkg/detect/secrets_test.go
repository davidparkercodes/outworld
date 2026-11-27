package detect

import (
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

func TestVendorTokens(t *testing.T) {
	runPositives(t, []classCase{
		{"gl" + "pat-Ab3dE5fG7hI9jK1lM3nO in ci", entity.Secret, "gl" + "pat-Ab3dE5fG7hI9jK1lM3nO"},
		{"hook wh" + "sec_Zk3Lm9Qp2Xv7Bn4Rt8Ws1Yc6Hd0Jf5Ga", entity.Secret, "wh" + "sec_Zk3Lm9Qp2Xv7Bn4Rt8Ws1Yc6Hd0Jf5Ga"},
		{"dc" + "kr_pat_Qw3rTy7uIo9pAs2dFg4hJk6lZx8", entity.Secret, "dc" + "kr_pat_Qw3rTy7uIo9pAs2dFg4hJk6lZx8"},
		{"vault hv" + "s.CAESIJx8mQ2vL9kTn4RcYw7HpJ0Ef", entity.Secret, "hv" + "s.CAESIJx8mQ2vL9kTn4RcYw7HpJ0Ef"},
		{"ts" + "key-auth-kTn4RcYw7HpJ-Zk3Lm9Qp2Xv7Bn4Rt8Ws1Yc6", entity.Secret, "ts" + "key-auth-kTn4RcYw7HpJ-Zk3Lm9Qp2Xv7Bn4Rt8Ws1Yc6"},
		{"bot 48" + "21" + "937502:AAHk3Lm9Qp2Xv7Bn4Rt8Ws1Yc6Hd0Jf5GaQ", entity.Secret, "48" + "21" + "937502:AAHk3Lm9Qp2Xv7Bn4Rt8Ws1Yc6Hd0Jf5GaQ"},
		{"post to https://ho" + "oks.slack.com/services/T04AB12CD/B05EF34GH/Zk3Lm9Qp2Xv7Bn4Rt8Ws1Yc6 now", entity.Secret, "https://ho" + "oks.slack.com/services/T04AB12CD/B05EF34GH/Zk3Lm9Qp2Xv7Bn4Rt8Ws1Yc6"},
		{"--" + "---BEGIN PGP PRIVATE KEY BLOCK-----\n\nlQOYBGVxyz0BCADk3Lm9Qp2Xv7Bn4Rt8\n=Ab1C\n-----END PGP PRIVATE KEY BLOCK-----", entity.Secret, "--" + "---BEGIN PGP PRIVATE KEY BLOCK-----\n\nlQOYBGVxyz0BCADk3Lm9Qp2Xv7Bn4Rt8\n=Ab1C\n-----END PGP PRIVATE KEY BLOCK-----"},
		{"--" + "---BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtz\n(truncated)", entity.Secret, "--" + "---BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtz\n"},
		{"AGE-SECRET-KEY-1QPZRY9X8GF2TVDW0S3JN54KHCE6MUA7LQPZRY9X8GF2TVDW0S3JN54KHCE", entity.Secret, "AGE-SECRET-KEY-1QPZRY9X8GF2TVDW0S3JN54KHCE6MUA7LQPZRY9X8GF2TVDW0S3JN54KHCE"},
	})
}

func TestStructuredSecrets(t *testing.T) {
	runPositives(t, []classCase{
		{"db_password=Xk9mQ2vL", entity.Secret, "Xk9mQ2vL"},
		{"aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYzT4h8Qe2Lm", entity.Secret, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYzT4h8Qe2Lm"},
		{`"clientSecret": "p8Q~x7Lm2-Vn4Rt8Ws1Yc6Hd0Jf5Ga"`, entity.Secret, "p8Q~x7Lm2-Vn4Rt8Ws1Yc6Hd0Jf5Ga"},
		{"Authorization: Basic ZGVwbG95OlNhYmlhLUxhcmFuamFfODg=", entity.Secret, "ZGVwbG95OlNhYmlhLUxhcmFuamFfODg="},
		{"x-api-key: 9f8a7b6c5d4e3f2a1b0c", entity.Secret, "9f8a7b6c5d4e3f2a1b0c"},
		{"Cookie: sessionid=k3j4h5g6f7d8s9a0q1w2e3r4; theme=dark", entity.Secret, "k3j4h5g6f7d8s9a0q1w2e3r4"},
		{"JSESSIONID=4A7F9C2E1B8D3F6A5C0E9B7D2F4A6C8E", entity.Secret, "4A7F9C2E1B8D3F6A5C0E9B7D2F4A6C8E"},
		{"redis://:Sup3rS3cret!@cache.example:6379/0", entity.Secret, "Sup3rS3cret!"},
		{"mongodb+srv://app:Gull!Harbor9@cluster0.example.net/db", entity.Secret, "Gull!Harbor9"},
		{"Server=db;User Id=sa;Password=Mar1in#Quay;", entity.Secret, "Mar1in#Quay"},
		{"curl -u joao.carvalho:Sabia-Laranja_88 https://files.example/api", entity.Secret, "Sabia-Laranja_88"},
		{"mysql -u root -pS3cr3tPw! prod", entity.Secret, "S3cr3tPw!"},
		{"GET /cb?state=1&access_token=Zk3Lm9Qp2Xv7 HTTP/1.1", entity.Secret, "Zk3Lm9Qp2Xv7"},
		{"seed phrase: legal winner thank year wave sausage worth useful legal winner thank yellow", entity.Secret, "legal winner thank year wave sausage worth useful legal winner thank yellow"},
		{"BitLocker recovery key 123456-234567-345678-456789-567890-678901-789012-890123", entity.Secret, "123456-234567-345678-456789-567890-678901-789012-890123"},
		{"Your backup codes:\n1. a1b2c-3d4e5\n2. f6g7h-8i9j0\n\nKeep them safe.", entity.Secret, "f6g7h-8i9j0"},
		{"recovery codes: 4821 9375, 0285 1147", entity.Secret, "4821 9375"},
		{"Votre code de vérification : 482913", entity.Secret, "482913"},
		{"Ihr Bestätigungscode lautet G-482913", entity.Secret, "G-482913"},
	})
	runNegatives(t, entity.Secret, []string{
		"max_tokens: 4096",
		"token_count = 1500",
		`tokenizer = "cl100k_base"`,
		"secret_name: db-prod-credentials",
		"password_min_length = 12",
		"primary key (id), foreign key (order_id) references orders(id)",
		`password = os.environ["DB_PASSWORD"]`,
		"password: ${DB_PASSWORD}",
		`api_key = "<your-api-key>"`,
		"API_KEY=changeme",
		"SECRET_KEY=xxxxxxxxxxxxxxxxxxxxxxxx",
		"Use `Authorization: Bearer <token>` on every request.",
		"Authorization: required for all endpoints",
		"Set the `X-Api-Key` header to your key.",
		"Cookie: theme=dark; lang=en-GB",
		"Set-Cookie: locale=de; Path=/; Max-Age=31536000; SameSite=Lax",
		"Basic understanding of internationalization is assumed",
		"recovery codes can be regenerated in settings",
		"Backup codes are shown only once. Store them safely.\nOrder 48219375 ships on 2026-03-14",
		"seed words: none configured",
		"Seed 1337 gives reproducible results; random_state=42",
		"Enter the 6-digit code from your authenticator app.",
		"curl -u \"$USER:$PASS\" https://api.example/v1",
		"curl --user <user>:<password> https://api.example/v1",
		"mysql -u root -p prod",
		"https://example.com/search?q=Hello%20World2&page=3",
	})
}

func TestProsePasswords(t *testing.T) {
	runPositives(t, []classCase{
		{"creds are in the vault, or use the break-glass one: Tr4wler!Osprey-22", entity.Secret, "Tr4wler!Osprey-22"},
		{"WLAN-Passwort für ihn ist immer noch Zimtstern#2019, falls er wieder fragt", entity.Secret, "Zimtstern#2019"},
		{"le mot de passe est Soleil2024", entity.Secret, "Soleil2024"},
		{"la contraseña es Gaviota_Puerto9", entity.Secret, "Gaviota_Puerto9"},
		{"a senha é Sabia-Laranja_88", entity.Secret, "Sabia-Laranja_88"},
		{"het wachtwoord is Otter-Gracht-92", entity.Secret, "Otter-Gracht-92"},
		{"hasło: Bocian!Wisla7", entity.Secret, "Bocian!Wisla7"},
		{"login: admin / Xy7!abcQ", entity.Secret, "Xy7!abcQ"},
		{"use Kr4ken!Tide-07 to log in", entity.Secret, "Kr4ken!Tide-07"},
		{"the creds are deploy:Mango-Tree_41", entity.Secret, "Mango-Tree_41"},
		{"temporary password Winter#Storm9", entity.Secret, "Winter#Storm9"},
		{"pw `Qu1et*Harbor`", entity.Secret, "Qu1et*Harbor"},
		{"the wifi password:\nGull!Harbor9", entity.Secret, "Gull!Harbor9"},
		{"she pasted Vr8$kLm!2pQx into the channel", entity.Secret, "Vr8$kLm!2pQx"},
		{"PIN is 4821", entity.Secret, "4821"},
		{"pw Xk9mQ2vL for the staging box", entity.Secret, "Xk9mQ2vL"},
		{"login with admin / hunter2xyz", entity.Secret, "hunter2xyz"},
		{"my secret answer is Fluffy1987", entity.Secret, "Fluffy1987"},
	})
	runNegatives(t, entity.Secret, []string{
		"The password must be at least 12 characters and include a digit.",
		"Your password was changed on 2026-03-01 from Chrome on Windows-11.",
		"The admin password is stored in 1Password under Shared-Infra.",
		"Credentials for the S3 bucket live in IAM-Role-Deploy2, not in env vars.",
		"creds expire 2026-04-01; renew via Okta-SSO2",
		"pw reset flow redesign, see Figma-Frame12",
		"Kennwort vergessen? Klicken Sie hier.",
		"WLAN-Passwort steht auf dem Router-Aufkleber (Fritz!Box 7590).",
		"reset the password as described in Ticket#88213",
		"psql -U app -p 5432 -h pg-3.internal orders",
		"redis-cli -p 6380 ping",
		"docker login -u ci-bot --password-stdin registry.example:5000",
		"código de verificación enviado al 612 345 678",
		"Bestätigungscode wurde an 0151 23456789 gesendet",
		"login attempts: 3/5, lockout after 5, cooldown 15min",
		"Passwords are hashed with bcrypt (cost 12) before storage.",
		"Forgot your password? Click the link below to reset it.",
		"Password reset emails are rate-limited to 5 per hour.",
		"Login failed for user admin from 203.0.113.7 after 3 attempts",
		"The login page loads React18 and Tailwind3 from the CDN.",
		"Sign in with Google, GitHub or Microsoft365.",
		"The wifi network is called Guest-5G and covers Floor-2.",
		"C# 12 and F# 8 ship with .NET 8; C++20 modules need CMake 3.28+",
		"Vue3+TypeScript, React18+Next14, Python3+Django5",
		"AT&T, H&M, R&D2024, P&L-Q3, Q&A session 2",
		"Issue#1234, PR#4567, Ticket#88213, Room#204, Section#3",
		"See README.md#Usage and CONTRIBUTING.md#Step2",
		"Meet@Room12 at 10:30am, lunch@12:15",
		"email me at jane.doe+Invoices2026@example.com or ops@Example.org",
		"Follow @Acme_Dev2 and #Launch2026!",
		"Wow!That2 was fast? Really!",
		"Total: $1,299.00 (incl. 19% VAT), EUR 1.199,00, US$1200",
		"Is it really 50%Off? Buy2Get1!",
		"x^2+y^2=z^2, f(x)=3x^2-2x+1, O(n^2) worst case",
		"git rebase -i HEAD~3; git diff main...feature/JIRA-1234-fix",
		"const re = /^(?=.*[A-Z])(?=.*[a-z])(?=.*\\d)(?=.*[!@#$%^&*]).{8,}$/;",
		"printf(\"%s:%d %-20s 0x%08x\\n\", file, line, name, flags);",
		"Sony WH-1000XM5, Canon EOS-R6mk2, iPhone15Pro, Galaxy-S24+",
		"Team: Platform-Infra2, Squad-7B, Tribe!Payments",
		"ssh admin@db01 and check the login banner",
		"https://example.com/docs/README.md#Step2-Install",
	})
}

func TestHighEntropyNegatives(t *testing.T) {
	runNegatives(t, entity.Secret, []string{
		`"integrity": "sha512-Qx8mT4pZr2LkVn9Yw1cEoH7uJd5sAf3gBtR6yXiK0lMzNqWvU2hC8jPbDeFoGaSrTyIuOpLkJhGfDsAqWeRtYg==",`,
		"  integrity sha512-tVQRucExLQ02Boi4vdPp49svNGcfL2GhdTCT9aldhXgCJVAI21EtRfBettiuLUwce/7r6bFdgs6JFkcdTiFttA==",
		"github.com/stretchr/testify v1.9.0 h1:HtqpIVDClZ4nwg75+f6Lvsy/wHu+3BoSGCbBAcpTsTg=",
		"requests==2.32.3 --hash=sha256:70761cfe03c773ceb22aa2f671b4757976145175cdfca038c02654d061d6dcc6",
		`<img src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==" alt="">`,
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJx8mQ2vL9kTn4RcYw7HpJ0EfGhIjKlMnOpQrStUvWx deploy@build",
		"-----BEGIN CERTIFICATE-----\nMIIDdzCCAl+gAwIBAgIEAgAAuTANBgkqhkiG9w0BAQUFADBaMQswCQYDVQQGEwJJ\nRTESMBAGA1UEChMJQmFsdGltb3JlMRMwEQYDVQQLEwpDeWJlclRydXN0MSIwIAYD\n-----END CERTIFICATE-----",
		"-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE8mQx9LkVn8Yw1cEoH7uJd5sAf3gB\n-----END PUBLIC KEY-----",
		"SHA256 fingerprint: SHA256:nThbg6kXUpJWGl7E1IGOCspRomTxdCARLviKw6E5SY8",
		"src/components/UserProfile2/EditProfileForm.tsx",
		"getUserAccountDetailsByIdAsync2 called 3 times",
		"handleOAuth2CallbackWithPkceVerifier returned 302",
		"--- PASS: TestParseDuration/P3Y6M4DT12H30M5S (0.00s)",
		"subnet-0bb1c79de3f4a5b6c vpc-0a1b2c3d4e5f67890 sg-903004f8",
		`"hash": "0x5c504ed432cb51138bcf09aa5e8a410dd4a1e204ef84bfed1be16dfba1b22060"`,
		"SGVsbG8gV29ybGQsIHRoaXMgaXMgYSBwbGFpbiB0ZXh0IG1lc3NhZ2Uu",
		"- uses: actions/checkout@b4ffde65f46336ab88eb53be808477a3936bae11 # v4.1.1",
		"com.acme.payments.settlement.SettlementBatchProcessorV2Impl",
	})
	runPositives(t, []classCase{
		{"paste Zk3Lm9Qp2Xv7Bn4Rt8Ws1Yc6Hd0Jf5Ga here", entity.Secret, "Zk3Lm9Qp2Xv7Bn4Rt8Ws1Yc6Hd0Jf5Ga"},
	})
}

func TestScannersSurviveTruncation(t *testing.T) {
	samples := []string{
		"creds are in the vault, or use the break-glass one: Tr4wler!Osprey-22",
		"the wifi password:\nGull!Harbor9",
		"use Kr4ken!Tide-07 to log in with deploy:Mango-Tree_41",
		"Passport on file (passenger 1): N01847729, NIR 2 84 05 75 116 042 17, HKID K482193(9)",
		"IBAN de89 3704 0044 0532 0130 00, card 4111 1111 1111 1111 04/27 123, BIC DEUTDEFF500",
		"Your backup codes:\n1. a1b2c-3d4e5\n2. 4821 9375\n",
		"Cookie: sessionid=k3j4h5g6f7d8s9a0q1w2e3r4; db_password=Xk9mQ2vL; mysql -u root -pS3cr3tPw!",
		"contraseña: Gaviota_Puerto9 | hasło: Bocian!Wisla7 | 居民 11010519491231002X",
		"-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE8mQx9LkVn8Yw1cEoH7uJd5sAf3gB\n-----END PUBLIC KEY-----",
	}
	for _, s := range samples {
		for i := 0; i <= len(s); i++ {
			detectAll(t, s[:i])
			detectAll(t, s[i:])
		}
	}
}

func BenchmarkRegexDetector(b *testing.B) {
	text := "Hi Dana, the invoice INV-2026-004417 for 4,800.00 is attached. Call 020 7946 0958 or mail ops@example.org.\n" +
		"Passport on file: N01847729. Wire to DE89 3704 0044 0532 0130 00, BIC DEUTDEFF500.\n" +
		"2026-03-03T14:22:10.482Z INFO [req-8f14e45f] GET /api/v2/orders/100482 200 34ms token=abc\n"
	for i := 0; i < 4; i++ {
		text += text
	}
	d := NewRegexDetector()
	b.SetBytes(int64(len(text)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := d.Detect(b.Context(), text); err != nil {
			b.Fatal(err)
		}
	}
}

func TestAgeSecretKey(t *testing.T) {
	const key = "AGE-SECRET-KEY-1QPZRY9X8GF2TVDW0S3JN54KHCE6MUA7LQPZRY9X8GF2TVDW0S3JN54KHCE"
	runPositives(t, []classCase{
		{key, entity.Secret, key},
		{"identity file:\n# public key: age1qpzry9x8gf2tvdw0s3jn54khce6mua7l\n" + key + "\n", entity.Secret, key},
		{"the key was " + key + ".", entity.Secret, key},
	})
	vendor := func(text string) bool {
		for _, s := range detectAll(t, text) {
			if s.Source == "regex:vendor_token" && strings.HasPrefix(s.Text, "AGE-SECRET-KEY-") {
				return true
			}
		}
		return false
	}
	for _, text := range []string{
		strings.Replace(key, "QPZ", "QBZ", 1),
		strings.Replace(key, "QPZ", "QIZ", 1),
		strings.Replace(key, "QPZ", "QOZ", 1),
		strings.Replace(key, "9X8", "1X8", 1),
		key[:len(key)-1],
		key + "Q",
		key + "B",
		strings.ToLower(key),
		"AGE-SECRET-KEY-1" + strings.ToLower(key[16:]),
		"age-secret-key-1" + key[16:],
	} {
		if vendor(text) {
			t.Errorf("%q: unexpected age vendor_token", text)
		}
	}
	if !vendor("key " + key + " here") {
		t.Errorf("age key in prose not found by vendor_token")
	}
}
