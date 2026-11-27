package main

var itemsSecretsA = []def{
	secrets("dotenv-vendors", "env,api-keys,placeholders",
		usr(`I am splitting this .env into .env.example (safe to commit) and a secrets file. Which lines are real secrets and which are fine to leave in the example?

# berth-api local config, copied from {{p|takumi|possessive}}'s machine {{d|2026-03-02|2026-03-02}}
APP_ENV=development
APP_VERSION={{n|version|4.18.2}}
APP_BUILD_SHA={{n|hash|9f3c2e1a7b4d5f60718293a4b5c6d7e8f9012345}}
REQUEST_ID_NAMESPACE={{n|uuid|6ba7b810-9dad-11d1-80b4-00c04fd430c8}}
LOG_LEVEL=debug
PORT=8080

OPENAI_API_KEY={{s|openai-key|sk`+`-proj-Qm7xVp4LtZ8cW3bYn6HsKd1AeG5uRo0MjTf2XiNvCa9BwEyDq4UzSl}}
ANTHROPIC_API_KEY={{s|anthropic-key|sk`+`-ant-api03-Zk4Vq9LmX2pRt7WcY8bN3fHsJd6AeG1uTo5QiKvXr0MaPw-bYcS3dEf7gHjLnMq}}
STRIPE_SECRET_KEY={{s|stripe-key|sk`+`_live_51Qx7VpLtZ8cW3bYn6HsKd1AeG5uRo0MjTf2XiNvCa9BwEyDq4UzSlKm}}
STRIPE_PUBLISHABLE_KEY={{n|public-key|pk`+`_live_51Qx7VpLtZ8cW3bYn6HsKd1AeG5uRo0MjTf2XiNvCa9BwEyDq4UzSlPb}}
STRIPE_WEBHOOK_SECRET={{s|stripe-webhook-secret|wh`+`sec_Tf2XiNvCa9BwEyDq4UzSlKm7xVp4LtZ8}}
SENDGRID_API_KEY={{s|sendgrid-key|SG`+`.Qm7xVp4LtZ8cW3bYn6HsKd.1AeG5uRo0MjTf2XiNvCa9BwEyDq4UzSlKm7xVp4LtZ8}}
TWILIO_ACCOUNT_SID=AC`+`4f8b1c7e9a2d3f5061728394a5b6c7d8
TWILIO_AUTH_TOKEN={{s|twilio-auth-token|7d3c0c1e5b9a4f2e8d6c7b1a0f9e8d7c}}
GOOGLE_MAPS_API_KEY={{s|google-api-key|AI`+`zaSyD4Vq9LmX2pRt7WcY8bN3fHsJd6AeG1uTo}}
AWS_ACCESS_KEY_ID={{s|aws-access-key-id|AK`+`IAT4QZ7WLM2XN5RPVB}}
AWS_SECRET_ACCESS_KEY={{s|aws-secret-access-key|q9Vx2LmT7pRz4KcW8bYn3HsJd6AfG1uEo5QiNvXt}}
AWS_REGION=eu-central-1
GITHUB_TOKEN={{s|github-token|gh`+`p_R7kV2mQ9xLp4ZtW8cYb3NfH6sJd1AeG5uTo0}}
SLACK_BOT_TOKEN={{s|slack-token|xo`+`xb-4829173056-7730194825561-Qm3VxT9aLr6ZpW2cK8dYbN1h}}
SLACK_SIGNING_SECRET={{s|slack-signing-secret|e029badca45266732a9a79898a15ae2e}}
DATADOG_API_KEY={{n|placeholder|<your-datadog-key>}}
SENTRY_AUTH_TOKEN={{n|placeholder|changeme}}
JWT_SECRET={{s|jwt-secret|correct-horse-battery-staple-but-longer-2026}}
DATABASE_URL={{s|connection-string|postgres://berth_app:Vq7!mRz2pLx9@localhost:5432/berth_dev|Vq7!mRz2pLx9}}
REDIS_URL=redis://localhost:6379/1
SMTP_HOST=smtp.lumenwharf.example
SMTP_USER={{e|no-reply@lumenwharf.example}}
SMTP_PASSWORD={{n|secret-reference|${SMTP_PASSWORD}}}
ADMIN_EMAIL={{e|takumi.nakashima@lumenwharf.example}}
FEATURE_NEW_LABELS=true
ENCRYPTION_KEY_BASE64={{s|encryption-key|Kq3vX9LmT2pRz7WcY4bN8fHsJd1AeG6uTo5QiNvXr0M=}}
ENCRYPTION_KEY_ID=key-2026-03`),
		ast(`Real secrets, to move out and rotate if this file was ever shared: the OpenAI, Anthropic, Stripe secret, Stripe webhook, SendGrid, Twilio auth token, Google Maps, AWS secret access key (and its key id), GitHub, Slack bot token and signing secret, the JWT secret, the database URL with its password, and the base64 encryption key. Safe for .env.example: environment, version, build SHA, the namespace UUID, log level, port, region, the Stripe publishable key, the Twilio account SID, hosts, feature flags and the key id. The Datadog and Sentry lines are placeholders already, and SMTP_PASSWORD is a reference, not a value. The two email addresses are not secrets, but one is a named person, so I would replace it in the example.`),
		usr(`Thanks. {{p|takumi|nickname}} says the Slack signing secret "is just a hash, it is fine". Is he right? And replace {{p|takumi|possessive}}'s address with something generic in the example file but keep {{e|no-reply@lumenwharf.example}}.`)),

	secrets("passwords-in-prose", "passwords,prose",
		usr(`My dad wrote all his logins on a sheet of paper and I am typing it up to put in a password manager for him. Tell me which of these are weak and need changing first. Typed exactly as he wrote it:

"Bank: user {{p|herb|handle|htillinghast47}} password {{s|password|Tillinghast1947}}. The security word is the dog's name ({{s|security-answer|Biscuit}}).
Email ({{e|herb.tillinghast@desertmail.example}}): {{s|password|Margaret&Herb1969!}}
The pharmacy one is the same as email but with a 2 on the end.
Wifi at home, network {{p|herb|caps-last}}-5G, key: {{s|wifi-password|bluecactus4471}}
Garage keypad {{s|pin|1947}}, alarm {{s|pin|4471}} then press AWAY. The alarm company will ask for a verbal password, it is {{s|password|roadrunner}}.
Medicare site: I used the long one {{p|itzel|first}} made me, {{s|password|vT9#kLm2$Qx7pRz4}}, I cannot remember it which I suppose is the point.
Phone unlock {{s|pin|194700}}. Voicemail same as garage.
Streaming: {{p|thaddeus|nickname}} set it up, ask him. I think it is {{s|password|tadisthebest}} all one word."

He also has a sticky note that just says "{{s|password|Qx7!pRz4}}" with no label, and one that says "password" which I am fairly sure is not a password but a reminder to ask me for one. His date of birth is {{d|1945-03-11|March 11, 1945}} if that helps you spot the guessable ones, and yes, the dog really is called that.`)),

	secrets("k8s-secret-yaml", "yaml,kubernetes,base64",
		usr(`kubectl apply fails with "illegal base64 data at input byte 8". Which value is not valid base64? I ran everything through echo -n ... | base64 except one.

apiVersion: v1
kind: Secret
metadata:
  name: berth-api-secrets
  namespace: berth
  annotations:
    owner: {{p|lerato|handle|lfakude}}
    rotated: "{{d|2026-05-08|2026-05-08}}"
    checksum/config: {{n|checksum|4f8b1c7e9a2d3f5061728394a5b6c7d8e9f00112233445566778899aabbccdde}}
type: Opaque
data:
  DB_BASIC_AUTH: {{s|base64-credentials|YmVydGhfYXBwOlZxNyFtUnoycEx4OQ==}}
  ADMIN_PASSWORD: {{s|base64-password|UzNjdXJlLUhhcmJvci0yMDI2IQ==}}
  REGISTRY_AUTH: {{s|base64-credentials|Y2ktYm90OmRja3JfcGF0X1prNFZxOUxtWDJwUnQ3V2NZOGJOM2ZIcw==}}
  SESSION_KEY: {{s|session-key|Kq3vX9Lm!T2pRz7WcY4bN8fHsJd1AeG6}}
stringData:
  MONGO_URL: "{{s|connection-string|mongodb+srv://berth_rw:Xp4LtZ8cW3bY@cluster0.ab1cd.mongodb.example/berth?retryWrites=true|Xp4LtZ8cW3bY}}"
  AMQP_URL: "{{s|connection-string|amqps://labels:n6HsKd1AeG5u@mq.lumenwharf.example:5671/prod|n6HsKd1AeG5u}}"
  MSSQL: "{{s|connection-string|Server=tcp:legacy-sql.lumenwharf.example,1433;Database=Shipments;User Id=sa;Password=Ro0MjTf2XiNv!;Encrypt=true|Ro0MjTf2XiNv!}}"
  JDBC: "{{s|connection-string|jdbc:mysql://legacy-db.lumenwharf.example:3306/shipments?user=root&password=Tr0ub4dor%263xtra|Tr0ub4dor%263xtra}}"
  PUBLIC_BASE_URL: "https://berth.lumenwharf.example"
  SUPPORT_EMAIL: "{{e|support@lumenwharf.example}}"
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: berth-api-config
data:
  GIT_SHA: "{{n|hash|8c1f4e2a9b7d6c5e4f3a2b1c0d9e8f7a6b5c4d3e}}"
  IMAGE_DIGEST: "sha256:{{n|checksum|2be8daddbb82756f7d1f2c7ece706aadcb284bf6ab6d769ea695cc3ed6016743}}"
  API_KEY_HEADER_NAME: "X-Api-Key"
  PASSWORD_MIN_LENGTH: "{{n|number|12}}"`)),

	secrets("pem-keys", "private-key,public-key",
		usr(`ssh says "Load key: invalid format". I think I pasted the wrong half somewhere. Which of these blocks is the private key, and is it complete? My colleague {{p|svenerik|first}} says never to paste private keys into a chat but I am stuck.

--- file 1: id_ed25519.pub ---
ssh-ed25519 {{n|public-key|AAAAC3NzaC1lZDI1NTE5AAAAIK3vX9LmT2pRz7WcY4bN8fHsJd1AeG6uTo5QiNvXr0Ma}} {{p|lerato|handle|lfakude}}@devbox

--- file 2: id_ed25519 ---
{{s|private-key|--`+`---BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
XWR3uOcRsX3MsR8JRjmS31xoklDR3Ag6Bvioux8V9TRK+kXSeypA9liRExxIonI5DjfE0I
cdYvUaYz4WGT9l5datWF7pbhSL/5ObMAzKUvvF9C38KmJ8X9QfbeO+n3WC0H4zSoCaPtg/
KJnqzi1TyzEzstFHDxxJjVgMajXopUx4XDGlrVyCnzZiM1wsvIKqTIoitNwGSKGEOBxJOJ
-----END OPENSSH PRIVATE KEY-----|XWR3uOcRsX3MsR8JRjmS31xoklDR3Ag6Bvioux8V9TRK+kXSeypA9liRExxIonI5DjfE0I}}

--- file 3: deploy.pem (from the old CI box) ---
{{s|private-key|--`+`---BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAb4eTyd2XYNkwU1qq+jBA/OAYq2rUwn8fU0bQEbKAj22R8uIimh95
V/PAmcHrT/vYzJS2YGOuGEpCJHUNSklAkgkw5SrySLX2OWjZfAhxzhAARg+1/CSrs0LV
HTskuGwKfDw0dTVK/3avwB3CK9+HhVPsdVpTGorRQxf4uuKXcYmhDJ/tJakvSCh9GzBw|MIIEowIBAAKCAQEAb4eTyd2XYNkwU1qq+jBA/OAYq2rUwn8fU0bQEbKAj22R8uIimh95}}
(file ends here, no END line)

--- file 4: server.crt ---
-----BEGIN CERTIFICATE-----
{{n|public-key|MIIDdzCCAl+gAwIBAgIEAgAAuTANBgkqhkiG9w0BAQUFADBaMQswCQYDVQQGEwJJ}}
RTESMBAGA1UEChMJQmFsdGltb3JlMRMwEQYDVQQLEwpDeWJlclRydXN0MSIwIAYD
-----END CERTIFICATE-----
Fingerprint: SHA256:{{n|public-key|n7Q2xLmV9pRt4ZkW8cYb3NfH6sJd1AeG5uTo0XaBcDe}}

--- file 5: age key someone left in the repo ---
# created: {{d|2025-11-30|2025-11-30T21:14:05+01:00}}
# public key: {{n|public-key|age1qx7vp4ltz8cw3byn6hskd1aeg5uro0mjtf2xinvca9bweydq4uzs7k3m9p}}
{{s|age-secret-key|AGE-SECRET-KEY-1QX7VP4LTZ8CW3BYN6HSKD1AEG5URO0MJTF2XINVCA9BWEYDQ4UZSLKM7XVP}}`)),

	secrets("otp-recovery-codes", "otp,recovery-codes",
		usr(`I am locked out and trying to work out which of these codes is still usable. Explain the difference between them and which one I should try first.

Text from {{n|number|22395}}, {{d|2026-07-19|19 Jul}} 21:04: "Your {{o|Lumenwharf}} verification code is {{s|otp|604 281}}. It expires in 10 minutes. Do not share it."
Text from {{n|number|22395}}, {{d|2026-07-19|19 Jul}} 21:09: "{{s|otp|177530}} is your Lumenwharf code."
Email, {{d|2026-07-19|19 Jul}} 21:11: "Use this link to sign in: {{s|magic-link|https://berth.lumenwharf.example/auth/magic?token=mlk_Zk4Vq9LmX2pRt7WcY8bN3fHsJd6AeG1u&uid=1014|mlk_Zk4Vq9LmX2pRt7WcY8bN3fHsJd6AeG1u}}"

Recovery codes I saved when I set up 2FA on {{d|2025-01-06|6 January 2025}} (crossed out = used):
  {{s|recovery-code|4f8b-1c7e-9a2d}}   (used)
  {{s|recovery-code|3f50-6172-8394}}
  {{s|recovery-code|a5b6-c7d8-e9f0}}
  {{s|recovery-code|0112-2334-4556}}   (used)
  {{s|recovery-code|6778-899a-abbc}}

Authenticator setup key from the same day, in case it helps: {{s|totp-seed|JBSW Y3DP EHPK 3PXP K4VQ 9LMX}}
Backup phone on the account: {{ph|+254 722 555 014}}. Account email: {{e|wade.kimathi@lumenwharf.example}}.

Support ({{p|leilani|first}}, case {{n|identifier|SUP-56120}}) told me to "read out the six digit code" on the phone, which I refused to do. Was that the right call? She said her name was {{p|leilani}} and gave employee number {{n|identifier|E-20755}}, which is not what the staff directory shows for her.`)),

	secrets("lookalikes-only", "lookalikes,negative",
		usr(`Our secret scanner flagged every line of this file and the team wants to turn it off. Go through it and tell me which hits, if any, are real. I believe none are.

# scanner-fixtures/should_not_match.txt
git commit:            {{n|hash|2c6f8e1d7719b45a0e9c393be7d0f41c2aab83d5}}
short sha:             {{n|hash|f41c2aa}}
tree object:           {{n|hash|4b825dc642cb6eb9a060e54bf8d69288fbee4904}}
uuid v4:               {{n|uuid|f47ac10b-58cc-4372-a567-0e02b2c3d479}}
uuid v7:               {{n|uuid|018f4e2a-9b7d-7c5e-8f3a-2b1c0d9e8f7a}}
md5 of empty string:   {{n|checksum|d41d8cd98f00b204e9800998ecf8427e}}
sha1 of empty string:  {{n|checksum|da39a3ee5e6b4b0d3255bfef95601890afd80709}}
sha256 of empty:       {{n|checksum|e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855}}
npm integrity:         {{n|checksum|sha512-XBx9AXhXktjUqnepgTiE5flcKIYWi/rme0Eaj+5Y0lftuGBq+jyRu/md4WnuxqgP1ubdpNCsYEYPxrzVHD8d6g==}}
docker digest:         sha256:{{n|checksum|2be8daddbb82756f7d1f2c7ece706aadcb284bf6ab6d769ea695cc3ed6016743}}
semver:                {{n|version|12.4.0-beta.3+build.20417}}
calver:                {{n|version|2026.05.1}}
ssh public key:        ssh-rsa {{n|public-key|AAAAB3NzaC1yc2EAAAADAQABAAABAQC7vX9LmT2pRz7WcY4bN8fHsJd1AeG6uTo5QiNvXr0Ma}} ci@build
gpg fingerprint:       {{n|public-key|3AA5 C343 71567 BD2 4829 1730 5F6A 9B0C 2D1E 8F7A}}
placeholder:           {{n|placeholder|<your-api-key>}}
placeholder:           {{n|placeholder|YOUR_API_KEY_HERE}}
placeholder:           {{n|placeholder|changeme}}
placeholder:           {{n|placeholder|xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx}}
placeholder:           {{n|placeholder|sk_test_REPLACE_ME}}
template reference:    {{n|secret-reference|$[[lb]] secrets.STRIPE_SECRET_KEY [[rb]]}}
env reference:         {{n|secret-reference|$DATABASE_PASSWORD}}
vault path:            {{n|secret-reference|vault:secret/data/berth/prod#db_password}}
arn:                   {{n|identifier|arn:aws:iam::123456789012:role/berth-deployer}}
css colour:            {{n|identifier|#4f8b1c}}
mac address (docs):    {{n|identifier|00:00:5E:00:53:AF}}
ipv6 (docs):           {{n|identifier|2001:db8:85a3::8a2e:370:7334}}
base64 of "hello":     {{n|identifier|aGVsbG8=}}
isbn:                  {{n|number|978-3-16-148410-0}}
tracking number:       {{n|order-number|1Z999AA10123456784}}
lorem:                 Lorem ipsum dolor sit amet, consectetur adipiscing elit
word "password":       the password field must not be empty
word "secret":         {{o|Victoria's Secret|company-eponym}} is a retailer; a secret santa is a tradition`)),
}
