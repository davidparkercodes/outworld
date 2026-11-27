package main

var itemsSecretsB = []def{
	secrets("curl-commands", "shell,bearer,basic-auth",
		usr(`Convert these curl calls into a Python requests script with the credentials read from environment variables. They are from our internal wiki page, written by {{p|kwabena|nickname}}.

# 1. list shipments
curl -s https://api.berth.lumenwharf.example/v2/shipments \
  -H "Authorization: Bearer {{s|bearer-token|bk_live_7Hq2LmX9vRt4ZpW8cYb3NfK6sJd1AeG5}}" \
  -H "X-Request-Id: {{n|uuid|0b9f6a2e-5c1d-4e7f-9a3b-2d4c6e8f0a1b}}"

# 2. carrier sandbox (basic auth)
curl -u '{{s|basic-auth|corvane_api:Wq8!nT3vKx2#Lm}}' https://sandbox.carrier.example/rates?zone=3

# 3. same thing, header form
curl -H "Authorization: Basic {{s|basic-auth|Y29ydmFuZV9hcGk6V3E4IW5UM3ZLeDIjTG0=}}" https://sandbox.carrier.example/rates?zone=3

# 4. upload to the bucket with a presigned URL (expires, but still)
curl -X PUT -T manifest.csv "{{s|presigned-url|https://berth-uploads.s3.eu-central-1.amazonaws.example/manifests/2026/manifest.csv?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AK`+`IAT4QZ7WLM2XN5RPVB%2F20260514%2Feu-central-1%2Fs3%2Faws4_request&X-Amz-Date=20260514T075231Z&X-Amz-Expires=900&X-Amz-Signature=5d41402abc4b2a76b9719d911017c592e3b0c44298fc1c149afbf4c8996fb924|5d41402abc4b2a76b9719d911017c592e3b0c44298fc1c149afbf4c8996fb924|AK`+`IAT4QZ7WLM2XN5RPVB}}"

# 5. GitLab, token in the URL because the proxy strips headers
git clone https://oauth2:{{s|gitlab-token|gl`+`pat-Xp4LtZ8cW3bYn6HsKd1A}}@git.lumenwharf.example/lumenwharf/berth-api.git

# 6. health check, no auth needed
curl -s https://status.lumenwharf.example/healthz | jq .version   # prints "{{n|version|4.18.2}}"

# 7. the docs example, not a real key
curl https://api.vendor.example/v1/ping -H "X-Api-Key: {{n|placeholder|YOUR_API_KEY}}"

# 8. Telegram alert bot
curl -s "{{s|telegram-bot-token|https://api.telegram.example/bot77`+`30`+`194825:AAHk4Vq9LmX2pRt7WcY8bN3fHsJd6AeG1uT}}/sendMessage" -d chat_id=-{{n|identifier|1004829173056}} -d text="deploy ok"`)),

	secrets("actions-workflow", "ci,yaml,secret-references",
		usr(`Security review of this workflow please. I think one credential is hard-coded but I cannot see it among all the references.

name: deploy
on:
  push:
    branches: [main]
env:
  NODE_VERSION: "{{n|version|20.11.1}}"
  REGISTRY: registry.lumenwharf.example
jobs:
  build:
    runs-on: ubuntu-{{n|version|24.04}}
    steps:
      - uses: actions/checkout@{{n|hash|b4ffde65f46336ab88eb53be808477a3936bae11}} # {{n|version|v4.1.1}}
      - uses: actions/setup-node@{{n|hash|60edb5dd545a775178f52524783378180af0d1f8}}
        with:
          node-version: {{n|template-reference|$[[lb]] env.NODE_VERSION [[rb]]}}
      - name: Login to registry
        run: echo "{{n|secret-reference|$[[lb]] secrets.REGISTRY_PASSWORD [[rb]]}}" | docker login $REGISTRY -u ci-bot --password-stdin
      - name: Publish package
        run: npm publish
        env:
          NODE_AUTH_TOKEN: {{n|secret-reference|$[[lb]] secrets.NPM_TOKEN [[rb]]}}
      - name: Notify
        run: |
          curl -X POST https://hooks.slack.example/services/{{s|slack-webhook|T04829173/B07730194/Qm3VxT9aLr6ZpW2cK8dYbN1h}} \
            -d '{"text":"deployed {{n|template-reference|$[[lb]] github.sha [[rb]]}} by {{n|template-reference|$[[lb]] github.actor [[rb]]}}"}'
      - name: Deploy
        run: ./deploy.sh
        env:
          AWS_ROLE_ARN: {{n|identifier|arn:aws:iam::123456789012:role/berth-deployer}}
          DEPLOY_KEY: {{n|secret-reference|$[[lb]] secrets.DEPLOY_KEY [[rb]]}}
          SENTRY_AUTH_TOKEN: {{s|sentry-token|sn`+`trys_eyJpYXQiOjE3NzgwMDAwMDAsInVybCI6Imh0dHBzOi8vc2VudHJ5LmV4YW1wbGUifQ_Kq3vX9LmT2pRz7WcY4bN8fHsJd1AeG6uTo5}}
          DATABASE_URL: {{n|secret-reference|$[[lb]] secrets.DATABASE_URL [[rb]]}}

# last edited by {{p|lerato}} ({{p|lerato|handle|lfakude}}) on {{d|2026-05-08|8 May 2026}}; before that {{p|svenerik|initial-last|S-E. Bergqvist}}`)),

	secrets("jwt-debugging", "jwt,tokens",
		sys(`You help developers debug authentication problems. Never repeat full tokens back to the user.`),
		usr(`The API returns 401 "token expired" but I only just logged in. Here is the access token from the browser, can you decode it and tell me what is wrong?

Authorization: Bearer {{s|jwt|eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1XzEwMTQiLCJlbWFpbCI6IndhZGUua2ltYXRoaUBsdW1lbndoYXJmLmV4YW1wbGUiLCJuYW1lIjoiV2FkZSBLaW1hdGhpIiwicm9sZSI6ImFkbWluIiwiaWF0IjoxNzc4MDAwMDAwLCJleHAiOjE3NzgwMDM2MDB9.pkY4-gRAg0mbU5KUxIgtvA-xkz6N8NCTOzUxd8DnBls|pkY4-gRAg0mbU5KUxIgtvA-xkz6N8NCTOzUxd8DnBls}}

Server time according to the Date header: {{d|2026-05-05|Tue, 05 May 2026 18:40:12 GMT}}`),
		ast(`The token has three parts. The header says HS256. The payload decodes to subject u_1014, an email and a name, role admin, issued-at {{n|number|1778000000}} and expiry {{n|number|1778003600}}, so it was valid for one hour from issue. Compare the expiry with the server time: if the server clock is ahead of the issuer by more than an hour, every fresh token looks expired. Which service issued it, and are the two clocks in sync?`),
		usr(`The issuer is our auth service. Its own token for the backup job looks like this, and it has no iat at all:

{{s|jwt|eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJodHRwczovL2F1dGgubHVtZW53aGFyZi5leGFtcGxlIiwic3ViIjoic3ZjLWJhY2t1cCIsInNjb3BlIjoic25hcHNob3RzOndyaXRlIiwiZXhwIjoxNzc4MDkwMDAwfQ.RXbj910ni2hRgkk6D1TBcTlTilrHVE81X1DRxVNwvys|RXbj910ni2hRgkk6D1TBcTlTilrHVE81X1DRxVNwvys}}

The signing secret in the service config is {{s|jwt-secret|correct-horse-battery-staple-but-longer-2026}} and the refresh token cookie is rt={{s|refresh-token|rt_4f8b1c7e9a2d3f5061728394a5b6c7d8e9f00112}}. The public JWKS key id is {{n|identifier|kid-2026-03}}, that one is fine to share. {{p|wade}} is the user who reported it; he is in {{l|Nairobi}}, three hours ahead of the server.`)),

	secrets("tokens-in-logs", "log,urls,tokens",
		usr(`Compliance wants to know what leaked into our application log on {{d|2026-06-13|13 June}}. Classify each line: credential, personal data, or harmless.

02:11:04 INFO  http  GET /v2/shipments?page=3&api_key={{s|api-key|bk_live_7Hq2LmX9vRt4ZpW8cYb3NfK6sJd1AeG5}} 200 181ms
02:11:09 INFO  http  GET /healthz 200 1ms build={{n|hash|4be1a02d}}
02:11:31 WARN  auth  password reset requested for {{e|chad.wexford@corvanefreight.example}} link={{s|reset-link|https://berth.lumenwharf.example/reset/3f506172-8394-a5b6-c7d8-e9f001122334?sig=Zk4Vq9LmX2pRt7Wc|3f506172-8394-a5b6-c7d8-e9f001122334}}
02:12:02 DEBUG mail  SMTP AUTH LOGIN user={{e|no-reply@lumenwharf.example}} pass={{s|password|hunter-is-not-2-anymore}}
02:12:40 INFO  oauth callback code={{s|oauth-code|4/0AeG5uRo0MjTf2XiNvCa9BwEyDq4UzSlKm7xVp4LtZ8cW3bYn6HsKd1}} state={{n|uuid|c9bf9e57-1685-4c89-bafb-ff5af830be8a}}
02:13:15 INFO  export tenant=t_corvane user={{p|chad|handle|chadw}} rows=48213 file=s3://berth-exports/t_corvane/{{d|2026-06-13|2026-06-13}}/export.csv
02:13:16 DEBUG s3    signed with key id {{s|aws-access-key-id|AK`+`IAT4QZ7WLM2XN5RPVB}} etag={{n|checksum|d41d8cd98f00b204e9800998ecf8427e}}
02:14:00 INFO  hook  POST https://hooks.corvanefreight.example/berth?secret={{s|webhook-secret|cvf_whk_Tf2XiNvCa9BwEyDq4UzSlKm7}} 204
02:14:22 ERROR pay   card declined pan={{f|card-visa|4539 1487 0362 9153}} cvv={{s|card-cvv|417}} holder="{{p|chad|caps}}" msg="insufficient funds"
02:14:23 ERROR pay   retry with token {{n|identifier|tok_visa_debit}} for customer {{n|identifier|cus_Qx7VpLtZ8cW3bY}}
02:15:07 INFO  sess  created session sid={{s|session-id|s%3AKq3vX9LmT2pRz7WcY4bN8fHsJd1AeG6u.To5QiNvXr0MaPwBcS3dEf7gHjLnMq}} for user 1014 ip=203.0.113.61
02:15:08 INFO  trace span={{n|hash|5e1f0c9d2a7b4c6d}} trace={{n|hash|7d3c0c1e5b9a4f2e8d6c7b1a0f9e8d7c}} sampled=1
02:16:44 INFO  i18n  loaded locale de-DE version {{n|version|44.1}} checksum {{n|checksum|efb1a25b}}`)),

	secrets("password-manager-export", "csv,passwords",
		usr(`I exported my vault to move to another password manager and the import fails on row 6. Can you see a formatting problem? (Yes I know I should not paste this. I will rotate everything afterwards.)

name,url,username,password,totp,notes
Bank,https://online.firstsonoran.example,{{p|itzel|handle|icarrasco87}},{{s|password|Paraiso!2215Tucson}},,security question: first pet = {{s|security-answer|Chispa}}
Court portal,https://efile.pimacourts.example,{{e|itzel.carrasco@pimacourts.example}},{{s|password|Lex&Order#44}},{{s|totp-seed|otpauth://totp/PimaCourts:itzel?secret=JBSWY3DPEHPK3PXP&issuer=PimaCourts|JBSWY3DPEHPK3PXP}},work
School app,https://rowanbrook.example/parents,{{e|itzel.cw@desertmail.example}},{{s|password|paloma2017}},,{{p|paloma|possessive}}'s school. {{p|desmond|nickname}} uses the same login
Netflix,https://netflix.example,{{e|itzel.cw@desertmail.example}},{{s|password|tadisthebest}},,profile PIN {{s|pin|2013}}
Router,http://192.168.1.1,{{x|username-equals-password|admin}},{{s|password|admin}},,default!! change this
"Mom's phone, unlock",,,{{s|pin|0330}},,"{{p|rosario|nickname}} forgets it. It is her saint's day, {{d|2026-03-30|March 30}}"
Wifi,,CalleParaiso-5G,{{s|wifi-password|sopaipilla con miel}},,
GitHub,https://github.example,{{p|itzel|handle|itzel-cw}},{{s|password|n7Q2x-LmV9p-Rt4Zk}},{{s|totp-seed|otpauth://totp/GitHub:itzel-cw?secret=K4VQ9LMX2PRT7WCY&issuer=GitHub|K4VQ9LMX2PRT7WCY}},recovery codes in the safe
Tax software,https://taxprep.example,{{e|itzel.cw@desertmail.example}},{{s|password|Refund$2026!}},,SSN is in the document vault not here
Example entry,https://example.com,{{n|placeholder|user@example.com}},{{n|placeholder|password123}},,delete me, came with the app`)),

	secrets("chat-paste-config-files", "chat,npmrc,netrc,docker",
		usr(`dm with a teammate, i think he just leaked half our tokens into slack. list exactly what needs rotating, in order of urgency

{{p|guilherme|nickname}} [11:02]: {{p|dawit|lower-first}} can you check why my publish fails? here's my setup

{{p|guilherme|nickname}} [11:02]:
~/.npmrc
//npm.lumenwharf.example/:_authToken={{s|npm-token|np`+`m_Zk4Vq9LmX2pRt7WcY8bN3fHsJd6AeG1uTo5Q}}
@lumenwharf:registry=https://npm.lumenwharf.example/
email={{e|gui@lumenwharf.example}}

~/.netrc
machine git.lumenwharf.example login {{p|guilherme|handle|gmascarenhas}} password {{s|password|Saudade-do-Rio-88}}
machine api.heroku.example login {{e|gui@lumenwharf.example}} password {{s|heroku-key|HR`+`KU-4f8b1c7e-9a2d-3f50-6172-8394a5b6c7d8}}

~/.docker/config.json
{"auths":{"registry.lumenwharf.example":{"auth":"{{s|base64-credentials|Y2ktYm90OmRja3JfcGF0X1prNFZxOUxtWDJwUnQ3V2NZOGJOM2ZIcw==}}"}},"credsStore":"desktop"}

~/.pypirc
[internal]
repository = https://pypi.lumenwharf.example/
username = __token__
password = {{s|pypi-token|py`+`pi-AgEIcHlwaS5vcmcCJDk4ZjFiM2E3LTVjMmQtNGU2Zi04YTFi}}

{{p|dawit}} [11:04]: {{p|guilherme|nickname|gui}}. delete that. now.
{{p|guilherme|nickname}} [11:04]: ?
{{p|dawit}} [11:04]: those are live tokens in a channel with 40 people
{{p|guilherme|nickname}} [11:05]: this is a dm no?
{{p|dawit}} [11:05]: it is not. it's #berth-dev. {{p|sue|first}} is typing
{{p|sue}} [11:05]: I have screenshotted nothing and seen everything. {{p|guilherme|first}}, rotate. {{p|svenerik|nickname}} is already revoking the npm one.
{{p|guilherme|nickname}} [11:06]: 😬 the hugging face one too? i had it in my clipboard: {{s|huggingface-token|hf_Qm7xVp4LtZ8cW3bYn6HsKd1AeG5uRo0MjT}}
{{p|dawit}} [11:06]: WHY`)),
}
