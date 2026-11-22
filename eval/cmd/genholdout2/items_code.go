package main

var itemsCode = []def{
	code("git-log", "git,name-in-email", usr(`Turn this into release notes for the 4.2 line, grouped by area. Drop the merge commits.

commit 9f2c1ab77e0d4c1b8a6f3e2d1c0b9a8f7e6d5c4b
Author: {{p|tomasz}} <{{e|tomasz.kowalczyk-brandt@harborline.example}}>
Date:   {{d|2026-03-03|Tue Mar 3 14:22:07 2026}} +0100

    ledger: stop double counting refunds in the nightly rollup

    Reported-by: {{p|ngozi}} <{{e|ngozi@harborline.example}}>
    Reviewed-by: {{p|bjorn|ascii-fold|Bjorn Akesson}} <{{e|bjorn.akesson@harborline.example}}>

commit 41d07be2a9c3f5e6d7b8a9c0d1e2f3a4b5c6d7e8
Merge: 7a1b2c3 e4f5a6b
Author: {{p|thuy|ascii-fold|Thuy Nguyen}} <{{e|thuy.nguyen@harborline.example}}>
Date:   {{d|2026-03-02|Mon Mar 2 09:05:51 2026}} +0700

    Merge branch 'thuy/export-csv' into main

commit e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3
Author: {{p|thuy|ascii-fold|Thuy Nguyen}} <{{e|thuy.nguyen@harborline.example}}>
Date:   {{d|2026-02-27|Fri Feb 27 18:40:12 2026}} +0700

    export: quote fields that contain a semicolon

    {{p|ngozi|first}} hit this with the Lagos customer file. Pairing with {{p|tomasz|nickname}} tomorrow on the importer side.

commit 7a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b
Author: {{p|bjorn}} <{{e|bjorn.akesson@harborline.example}}>
Date:   {{d|2026-02-26|Thu Feb 26 11:11:03 2026}} +0100

    ci: pin {{o|Gradle|tool}} to 8.7 and cache the {{o|Maven|tool}} repo

    Co-authored-by: {{p|seun}} <{{e|seun.babatunde@harborline.example}}>

commit 0c9d8e7f6a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1d
Author: {{p|seun|handle|seun-b}} <{{e|14402+seun-b@users.noreply.harborline.example}}>
Date:   {{d|2026-02-24|Tue Feb 24 22:58:40 2026}} +0100

    auth: rotate the signing key id, old tokens stay valid until {{d|2026-03-31|31 March}}`)),

	code("git-blame", "git,family-name-first", usr(`who do I ask about the retry logic here? blame below, I only joined last week

^3fa91c2 ({{p|haruto}}        {{d|2024-11-08}} 16:02:11 +0900  41) func (c *Client) do(req *Request) (*Response, error) {
^3fa91c2 ({{p|haruto}}        {{d|2024-11-08}} 16:02:11 +0900  42) 	var last error
b71e0d44 ({{p|priya}}   {{d|2025-06-19}} 10:47:55 +0530  43) 	for attempt := 0; attempt < c.maxRetries; attempt++ {
b71e0d44 ({{p|priya}}   {{d|2025-06-19}} 10:47:55 +0530  44) 		resp, err := c.http.Do(req.raw)
c0ffee12 ({{p|joao|truncated|João Pedro Figueiredo d}} {{d|2025-09-02}} 08:13:09 -0300  45) 		if err == nil && resp.StatusCode < 500 {
c0ffee12 ({{p|joao|truncated|João Pedro Figueiredo d}} {{d|2025-09-02}} 08:13:09 -0300  46) 			return wrap(resp), nil
b71e0d44 ({{p|priya}}   {{d|2025-06-19}} 10:47:55 +0530  47) 		}
9d2e77a0 ({{p|haruto}}        {{d|2026-01-15}} 19:30:42 +0900  48) 		last = err
9d2e77a0 ({{p|haruto}}        {{d|2026-01-15}} 19:30:42 +0900  49) 		time.Sleep(backoff(attempt)) // see {{p|priya|nickname}}'s note in #platform
00000000 (Not Committed Yet     {{d|2026-03-04}} 09:12:00 +0000  50) 	}

the slack note it mentions says "ask {{p|haruto|title-last|Kobayashi-san}} or {{p|joao|nickname}} before touching backoff"`)),

	code("python-traceback", "path,stack-trace", usr(`This worked yesterday. Any idea?

(venv) {{p|jiri|handle|jdvoracek}}@build-07:~/src/claims-etl$ python -m etl.run --day {{d|2026-03-01}}
Traceback (most recent call last):
  File "/home/{{p|jiri|path|jdvoracek}}/src/claims-etl/etl/run.py", line 88, in <module>
    main()
  File "/home/{{p|jiri|path|jdvoracek}}/src/claims-etl/etl/run.py", line 61, in main
    rows = load_claims(day)
  File "/home/{{p|jiri|path|jdvoracek}}/src/claims-etl/etl/load.py", line 142, in load_claims
    return [parse_row(r) for r in reader]
  File "/home/{{p|jiri|path|jdvoracek}}/src/claims-etl/etl/load.py", line 97, in parse_row
    dob = datetime.strptime(r["dob"], "%Y-%m-%d")
  File "/usr/lib/python3.12/_strptime.py", line 554, in _strptime_datetime
    tt, fraction, gmtoff_fraction = _strptime(data_string, format)
ValueError: time data '{{d|1987-04-09|09.04.1987}}' does not match format '%Y-%m-%d'

The row it choked on, from the vendor file:
claim_id=CL-2291847;insured={{p|hannelore|last-first|GRÜNWALD, Hannelore}};dob={{d|1987-04-09|09.04.1987}};tel={{ph|030 23125 441}};iban={{f|iban|DE89 3704 0044 0532 0130 00}}

Header comment in load.py says "# owner: {{p|jiri|ascii-fold|Jiri Dvoracek}} <{{e|jiri.dvoracek@northmere-insurance.example}}>, ask before changing date handling".`)),

	code("java-stacktrace", "stack-trace", usr(`Our onboarding service started throwing this in prod. Explain the root cause and suggest a fix that does not log customer data.

{{d|2026-03-05}} 07:41:19,204 ERROR [http-nio-8443-exec-17] c.b.onboarding.CustomerService - create failed for request 7f3e
com.brightquay.onboarding.DuplicateCustomerException: Duplicate customer: name='{{p|fatima}}', email='{{e|f.elidrissi@example.org}}', msisdn='{{ph|+33 6 39 98 01 27}}', nationalId='{{g|national-id|BK402915}}'
	at com.brightquay.onboarding.CustomerService.create(CustomerService.java:118)
	at com.brightquay.onboarding.api.CustomerController.post(CustomerController.java:57)
	at jdk.internal.reflect.DirectMethodHandleAccessor.invoke(DirectMethodHandleAccessor.java:103)
	at org.springframework.web.servlet.FrameworkServlet.service(FrameworkServlet.java:885)
Caused by: org.postgresql.util.PSQLException: ERROR: duplicate key value violates unique constraint "customers_email_key"
  Detail: Key (email)=({{e|f.elidrissi@example.org}}) already exists.
	at org.postgresql.core.v3.QueryExecutorImpl.receiveErrorResponse(QueryExecutorImpl.java:2713)
	... 61 common frames omitted

The date in that log line is {{d|2026-03-05|5 March}}, the same morning {{p|fatima|nickname}} called support twice. The agent ({{p|elodie|first}}) created the second record by hand.`)),

	code("dotenv", "secrets,config", usr(`convert this .env into a docker compose service with the secrets moved to docker secrets. keep variable names.

# staging, last rotated {{d|2026-02-11|11 Feb 2026}} by {{p|kobby|nickname}}
APP_ENV=staging
APP_PORT=8080
DATABASE_URL=postgres://ledger_app:{{s|password|Vq7!mZr2-Lw9xTe4|Vq7!mZr2-Lw9xTe4}}@db.internal.example:5432/ledger?sslmode=require
REDIS_URL=redis://:{{s|password|c4che-Me-0utside-88}}@cache.internal.example:6379/0
JWT_SIGNING_SECRET={{s|signing-secret|3q9XvB7mK2pL8sT4wY6zA1cD5fG0hJ3nR7uE9iO2}}
PAYMENTS_API_KEY={{s|api-key|sk_test_51Nq8ZtKf02HxWmP7cYbR4LuD9sVa3EgT6JhXo}}
AWS_ACCESS_KEY_ID={{s|aws-access-key|AKIAIOSFODNN7EXAMPLE}}
AWS_SECRET_ACCESS_KEY={{s|aws-secret-key|wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY}}
SMTP_USER={{e|notifications@harborline.example}}
SMTP_PASSWORD={{s|password|Tr0ub4dor&3-staging}}
ALERT_EMAIL={{e|kwabena.asante-boateng@harborline.example}}
SENTRY_RELEASE={{n|hash|4b825dc642cb6eb9a060e54bf8d69288fbee4904}}
LOG_LEVEL=info`)),

	code("k8s-manifest", "secrets,config", usr(`kubectl apply is rejecting this. What is wrong with the YAML?

apiVersion: v1
kind: Secret
metadata:
  name: billing-worker
  namespace: payments
  annotations:
    owner: "{{p|aino}} <{{e|aino.kaariainen@polarfjord.example}}>"
    on-call: "{{ph|+358 40 555 0142}}"
    last-rotated: "{{d|2026-01-30}}"
type: Opaque
stringData:
  DB_PASSWORD: "{{s|password|pg-Hx92!kLmQ-7741}}"
  WEBHOOK_SIGNING_SECRET: "{{s|signing-secret|whsec_9Tb3XkQ7mP2vL8sR4wY6zA1cD5fG0hJ3}}"
  GITHUB_TOKEN: "{{s|token|ghp_R7uE9iO2aS4dF6gH8jK0lZ1xC3vB5nM7qW9e}}"
    SFTP_PASSPHRASE: "{{s|password|correct horse battery staple 41}}"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: billing-worker
  labels:
    app.kubernetes.io/managed-by: {{o|Helm|tool}}
spec:
  replicas: 2
  template:
    spec:
      containers:
        - name: worker
          image: registry.polarfjord.example/billing-worker@sha256:{{n|hash|9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08}}
          envFrom:
            - secretRef:
                name: billing-worker`)),

	code("pr-description", "handles,pull-request", usr(`Tighten up this PR description, keep it under 200 words, keep the mentions.

## Fix timezone drift in appointment reminders (#2281)

Reminders for clinics east of UTC+8 went out a day early. Root cause: we stored the local date and then converted twice. Found by @{{p|akane|handle|akane-mz}} while looking at the Osaka pilot, and confirmed against the report {{p|meilin|western-order|Meilin Ouyang}} filed on {{d|2026-02-18|Feb 18}}.

### Changes
- store reminder instants in UTC, render in clinic tz
- backfill job for rows created between {{d|2026-01-05}} and {{d|2026-02-20}}
- new test fixtures (thanks {{p|kaho|nickname}})

### Testing
Staging: https://staging.carebridge.example/reminders (basic auth {{p|akane|handle|akane}} / {{s|password|Sakura-staging-2026!}})
Ran the backfill against a copy of prod, 1,412 rows touched.

### Rollout
Deploy {{d|2026-03-10|Tuesday 10 March}} after the 09:00 standup. @{{p|caglar|handle|cyildirim}} is on call; if anything looks off page him at {{ph|+90 532 555 01 42}}.

Reviewers: @{{p|meilin|handle|ouyang-meilin}} @{{p|kaho|handle|felix-tsang}}
cc {{p|caglar|first}}, {{p|kaho|western-order|Felix Tsang}}`)),

	code("commit-message", "git,trailers", usr(`Rewrite as a conventional commit with a body under 72 columns. Keep all trailers exactly.

fixed the thing where invoices for {{o|Okafor Freight|company-eponym}} were rounded wrong, {{p|ray|first}} from finance ({{p|ray}}, not the freight company lol) noticed on {{d|2026-03-02|3/2}} that totals were off by a cent on every invoice with more than 40 lines. banker's rounding now, with tests. also bumped {{o|Decimal.js|library}}.

Reported-by: {{p|ray}} <{{e|rokafor@tallowfield.example}}>
Tested-by: {{p|ria}} <{{e|eleftheria.p@tallowfield.example}}>
Co-authored-by: {{p|stefan|ascii-fold|Stefan Constantinescu}} <{{e|stefan@constantinescu.example}}>
Signed-off-by: {{p|zsofia|western-order|Zsófia Szabó}} <{{e|zsofia.szabo@tallowfield.example}}>`)),

	code("ticket", "ticket,support", usr(`Summarize this ticket for the weekly bug triage, two sentences max, then list open questions.

PLAT-4471  Password reset email never arrives for some .co.ke users
Reporter: {{p|wanjiru}}    Assignee: {{p|dilshan|initial-last|D. Wickramasinghe}}    Created: {{d|2026-02-25|25/Feb/26}} 3:14 PM
Priority: High    Labels: auth, email, customer-reported

Description
Customer {{p|selam}} ({{e|selam.g@example.net}}, mobile {{ph|+254 700 555 014}}) reports no reset mail since Monday. Reproduced with our QA mailbox.

Steps
1. Go to /login, click Forgot password
2. Enter {{e|qa+ke01@savannahpay.example}}
3. No mail. The reset link that should have gone out, from the outbox table: https://app.savannahpay.example/reset?token={{s|reset-token|rt_8fK2mQ9xLp4Zt7Wc1Yb6Nd3Vh5Sj0Ga}}

Comments
{{p|dilshan}} - {{d|2026-02-26|26/Feb/26}} 9:02 AM
The provider is bouncing on DMARC. {{p|wanjiru|nickname}}, can you ask {{p|selam|nickname}} whether she can receive from other senders?

{{p|wanjiru}} - {{d|2026-02-26|26/Feb/26}} 11:40 AM
Asked. She says yes. Temp workaround: I set her password by hand to {{s|password|Karibu#2026-temp}} and told her by phone to change it.

{{p|sbu|handle|sbu.mthembu}} - {{d|2026-02-27|27/Feb/26}} 8:15 AM
please never do that again {{p|wanjiru|lower-first|wanjiru}}. rotating now.`)),

	code("source-header", "source,author-comment,secrets", usr(`Review this file. I care most about anything that should not be in source control.

// Package notify sends transactional SMS.
//
// Copyright (c) 2025 {{o|Lindqvist Signals AB|company-eponym}}
// Author: {{p|paige}} <{{e|paige@lindqvist-signals.example}}>
// Maintainer since {{d|2025-11-01|1 Nov 2025}}: {{p|ingrid}} ({{p|ingrid|handle|isolbakken}})
package notify

import (
	"context"
	"net/http"
	"strings"
)

const (
	gatewayURL = "https://sms.gateway.example/v2/send"
	apiKey     = "{{s|api-key|gw_live_Zt7Wc1Yb6Nd3Vh5Sj0Ga8fK2mQ9xLp4}}"
	senderID   = "LINDQVIST"
	// TODO({{p|ingrid|lower-first|ingrid}}): move to the secret store before the audit
	testNumber = "{{ph|+47 400 55 014}}" // {{p|paige|possessive}}'s phone, do not spam
)

type Client struct{ hc *http.Client }

// Send delivers body to msisdn. {{p|soren|first}} asked for the retry to stay at 2,
// see the thread from {{d|2026-01-12|12. januar}}.
func (c *Client) Send(ctx context.Context, msisdn, body string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, gatewayURL, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("X-Sender", senderID)
	_, err = c.hc.Do(req)
	return err
}`)),

	code("manifest", "source,author-comment", usr(`Merge these two into one pyproject.toml using PEP 621 fields.

--- setup.cfg
[metadata]
name = tidepool-client
version = 2.14.3
author = {{p|ines}}
author_email = {{e|ines.albuquerque@tidepool-labs.example}}
maintainer = {{p|rafa}}, {{p|agus}}
maintainer_email = {{e|rafaela@tidepool-labs.example}}, {{e|aferreyra@tidepool-labs.example}}
url = https://code.tidepool-labs.example/{{p|ines|handle|inescarvalho}}/tidepool-client
license = Apache-2.0

--- package.json (the docs site)
{
  "name": "@tidepool/docs",
  "version": "0.9.0",
  "author": "{{p|rafa|nickname}} {{p|rafa|last}} <{{e|rafaela@tidepool-labs.example}}> (https://{{p|rafa|url-slug|rafaelacavalcanti}}.example.com)",
  "contributors": ["{{p|agus}}", "{{p|montse}} <{{e|montse@puigdevall.example}}>"],
  "dependencies": {
    "{{o|gatsby|library}}": "^5.13.3",
    "{{o|winston|library}}": "^3.11.0",
    "{{o|morgan|library}}": "~1.10.0",
    "{{o|luxon|library}}": "3.4.4"
  }
}`)),

	code("codeowners", "handles", usr(`Is there a way to make the payments rule win over the wildcard? Here is our CODEOWNERS and the MAINTAINERS file it was generated from.

# CODEOWNERS
*                   @{{p|lukas|handle|lvtrettenbach}}
/payments/          @{{p|shirin|handle|shirin-tf}} @{{p|yael|handle|yaelbs}}
/payments/sepa/     @{{p|maarten|handle|mvdheuvel}}
/docs/              @{{p|gudrun|handle|gudrun-s}}
*.tf                @{{p|mido|handle|mido}}

# MAINTAINERS
{{p|lukas}} <{{e|lukas.von.trettenbach@kupferhain.example}}> (@{{p|lukas|handle|lvtrettenbach}}) everything else
{{p|shirin}} <{{e|shirin@kupferhain.example}}> payments lead
{{p|yael}} <{{e|yael.ben-shoshan@kupferhain.example}}> payments, on leave until {{d|2026-04-13|13.04.2026}}
{{p|maarten|last-first|van den Heuvel, Maarten}} <{{e|m.vandenheuvel@kupferhain.example}}> SEPA
{{p|gudrun}} <{{e|gudrun@kupferhain.example}}> docs (writes as {{p|gudrun|ascii-fold|Gudrun Sigurdardottir}} on the blog)
{{p|mido}} ({{p|mido|nickname}}) <{{e|mahmoud.abdelrahman@kupferhain.example}}> infrastructure`)),

	code("terraform", "secrets,config", usr(`terraform plan wants to replace the database. Why? tfvars and the plan excerpt below.

# prod.auto.tfvars  (owner: {{p|nadine|first}}, {{e|nadine.khoury-haddad@cedarmount.example}})
region            = "eu-south-1"
db_instance_class = "db.r6g.large"
db_username       = "cedar_admin"
db_password       = "{{s|password|Xk9#vLp2$Qm7wRt5}}"
datadog_api_key   = "{{s|api-key|7d1f0c3a9b8e4f6a2c5d7e9f1a3b5c7d}}"
alert_phone       = "{{ph|+961 3 555 014}}"
tags = {
  CostCenter = "4410"
  Owner      = "{{p|nadine|handle|nkhoury}}"
  Reviewed   = "{{d|2026-02-09|2026/02/09}}"
}

  # aws_db_instance.main must be replaced
-/+ resource "aws_db_instance" "main" {
      ~ arn                = "arn:aws:rds:eu-south-1:000000000000:db:cedar-main" -> (known after apply)
      ~ engine_version     = "15.4" -> "16.2" # forces replacement
        identifier         = "cedar-main"
      ~ password           = (sensitive value)
    }`)),

	code("curl-session", "secrets,http", usr(`Why do I get a 422 here? The same body works from the web app.

$ curl -sS -X POST https://api.ferrycove.example/v3/customers \
    -H 'Authorization: Bearer {{s|bearer-token|fc_tok_Hx92kLmQ7741pgVq7mZr2Lw9xTe4a}}' \
    -H 'Content-Type: application/json' \
    -d '{"name":"{{p|tavita}}","email":"{{e|tavita.leota@example.com}}","phone":"{{ph|+685 72 55014}}","dob":"{{d|1991-07-04|04/07/1991}}","address":{"line1":"{{a|Lot 14 Vaitele Street}}","city":"{{l|Apia}}","country":"WS"}}'
{"error":{"code":"validation_failed","fields":{"dob":"expected ISO 8601 date","phone":"not a valid E.164 number"}},"request_id":"req_{{n|identifier|01HQZX7M4V8K2N5T}}"}

$ curl -sS https://api.ferrycove.example/v3/customers/cus_88213 -u {{p|kainoa|handle|kainoa}}:{{s|password|Hale'iwa-2026-shaka}}
{"id":"cus_88213","name":"{{p|kainoa}}","email":"{{e|kainoa.k@example.com}}","phone":"{{ph|(808) 555-0173}}","created":"{{d|2025-12-01|2025-12-01T19:04:11Z}}"}`)),

	code("go-panic", "stack-trace,logs", usr(`Service crashes about once an hour with this. What should I look at first?

{"level":"info","ts":"{{d|2026-03-06|2026-03-06T02:14:07.118Z}}","msg":"export requested","user":"{{e|aigerim.zh@steppeline.example}}","tenant":"steppeline","rows":48211}
{"level":"info","ts":"{{d|2026-03-06|2026-03-06T02:14:09.530Z}}","msg":"export requested","user":"{{e|enkhtuya@steppeline.example}}","tenant":"steppeline","rows":0}
panic: runtime error: index out of range [0] with length 0

goroutine 4117 [running]:
steppeline.example/exporter/internal/sheet.(*Writer).header(0xc0004be000)
	/home/{{p|enkhtuya|path|tuya}}/go/src/steppeline.example/exporter/internal/sheet/writer.go:88 +0x1d4
steppeline.example/exporter/internal/sheet.(*Writer).Flush(0xc0004be000, {0x0, 0x0, 0x0})
	/home/{{p|enkhtuya|path|tuya}}/go/src/steppeline.example/exporter/internal/sheet/writer.go:131 +0x9c
steppeline.example/exporter/internal/job.run({0x1b2c3d0, 0xc000123456}, {0xc0003fa2a0, 0x1a})
	/home/{{p|enkhtuya|path|tuya}}/go/src/steppeline.example/exporter/internal/job/job.go:57 +0x4a5
created by steppeline.example/exporter/internal/job.(*Pool).Submit in goroutine 61
	/home/{{p|enkhtuya|path|tuya}}/go/src/steppeline.example/exporter/internal/job/pool.go:44 +0xb6
exit status 2

{{p|enkhtuya|nickname}} built this binary on her laptop, which is why her path is in there. {{p|aigerim|possessive}} export always works.`)),

	code("test-fixture", "source,fixture", usr(`Our fixtures were copied from production years ago. Rewrite this file with obviously fake data but keep the same shapes and edge cases (apostrophes, diacritics, two-part surnames, missing last name).

export const customers = [
  { id: 1, first: '{{p|deirdre|first}}', last: '{{p|deirdre|last}}', email: '{{e|deirdre.ocallaghan@example.org}}', phone: '{{ph|020 915 5501}}', born: '{{d|1979-12-03}}' },
  { id: 2, first: '{{p|chema|first}}', last: '{{p|chema|last}}', email: '{{e|chema.gutierrez@example.com}}', phone: '{{ph|+52 55 5550 0142}}', born: '{{d|1968-05-22}}' },
  { id: 3, first: '{{p|wulan|first}}', last: null, email: '{{e|wulandari88@example.net}}', phone: '{{ph|+62 812 5550 142}}', born: '{{d|1988-08-17}}' },
  { id: 4, first: '{{p|gosia|first}}', last: '{{p|gosia|last}}', email: '{{e|gosia.wisniewska@example.org}}', phone: '{{ph|+48 22 555 01 42}}', born: '{{d|1993-01-30}}' },
  { id: 5, first: '{{p|wiremu|first}}', last: '{{p|wiremu|last}}', email: '{{e|wi.tewhaiti@example.com}}', phone: '{{ph|+64 21 555 0142}}', born: '{{d|2001-10-09}}' },
];

describe('displayName', () => {
  it('joins first and last', () => {
    expect(displayName(customers[0])).toBe("{{p|deirdre}}");
  });
  it('handles mononyms', () => {
    expect(displayName(customers[2])).toBe('{{p|wulan}}');
  });
  it('uppercases for the passport line', () => {
    expect(passportLine(customers[3])).toBe('{{p|gosia|caps-last-first|WIŚNIEWSKA MAŁGORZATA}}');
  });
});`)),

	code("shell-history", "secrets,shell,path", usr(`I think I leaked something into my shell history before sharing my screen. Which of these lines are a problem and how do I scrub them?

  501  cd /Users/{{p|hamish|path|hmacgillivray}}/work/glenmore-api
  502  export NPM_TOKEN={{s|token|npm_Q7mP2vL8sR4wY6zA1cD5fG0hJ3nR7uE9iO2a}}
  503  git clone https://{{p|hamish|handle|hamish-mac}}:{{s|token|glpat-xT9bK3mQ7vL2pR8sW4y}}@git.glenmore.example/platform/glenmore-api.git
  504  mysql -h db.glenmore.example -u reporting --password='{{s|password|Lochan!Dubh_77}}' analytics
  505  ssh {{p|angharad|handle|angharad}}@bastion.glenmore.example
  506  scp ~/Downloads/{{p|angharad|path|angharad_llywelyn}}_p60_{{d|2025-04-05|05-04-2025}}.pdf {{p|angharad|handle|angharad}}@bastion.glenmore.example:/tmp/
  507  openssl rand -hex 16
  508  echo {{n|hash|d41d8cd98f00b204e9800998ecf8427e}}
  509  curl -H "X-Api-Key: {{s|api-key|glm_2f7b9d1c4e6a8f0b3d5c7e9a1b2c4d6e}}" https://api.glenmore.example/v1/me
  510  history | grep -i token`)),

	code("sql-explain", "sql", usr(`This query takes 9 seconds. Plan below. Table has 40M rows.

-- written by {{p|zainab|first}} for the churn dashboard, {{d|2025-10-14|Oct 14 2025}}
EXPLAIN ANALYZE
SELECT u.id, u.full_name, u.email, max(e.created_at)
FROM users u JOIN events e ON e.user_id = u.id
WHERE u.email IN ('{{e|tahmina.chowdhury@example.com}}', '{{e|zq@indusriver.example}}')
   OR u.full_name ILIKE '%{{p|tahmina|last|chowdhury}}%'
GROUP BY 1, 2, 3;

 HashAggregate  (cost=1289441.12..1289441.14 rows=2 width=72) (actual time=9120.441..9120.447 rows=2 loops=1)
   ->  Hash Join  (cost=12.50..1289433.02 rows=1080 width=72) (actual time=3.118..9118.902 rows=2614 loops=1)
         ->  Seq Scan on events e  (cost=0.00..1183812.44 rows=40211244 width=16) (actual time=0.011..5210.331 rows=40211244 loops=1)
         ->  Hash  (cost=12.47..12.47 rows=2 width=64) (actual time=2.917..2.918 rows=2 loops=1)
               ->  Seq Scan on users u  (cost=0.00..12.47 rows=2 width=64) (actual time=0.640..2.911 rows=2 loops=1)
                     Filter: (((email)::text = ANY ('{{lb}}{{e|tahmina.chowdhury@example.com}},{{e|zq@indusriver.example}}{{rb}}'::text[])) OR ((full_name)::text ~~* '%{{p|tahmina|last|chowdhury}}%'::text))
 Planning Time: 0.412 ms
 Execution Time: 9120.513 ms

 id   | full_name          | email                         | max
------+--------------------+-------------------------------+---------------------
 4412 | {{p|tahmina}}  | {{e|tahmina.chowdhury@example.com}} | {{d|2026-03-01}} 22:10:04
 9087 | {{p|zainab}}     | {{e|zq@indusriver.example}}         | {{d|2026-03-05}} 08:44:51`)),

	code("webhook-payload", "json,financial", usr(`Write a TypeScript type for this webhook body and a function that verifies the signature header.

POST /hooks/payments HTTP/1.1
Host: api.orchardgate.example
X-Signature: t=1741251727,v1={{n|hash|5257a869e7ecebeda32affa62cdca3fa51cad7e77a0e56ff536d0ce8e108d8bd}}
X-Signing-Secret-Id: whk_live_03

{
  "id": "evt_3PqLx2",
  "type": "charge.succeeded",
  "created": 1741251727,
  "data": {
    "amount": 12900,
    "currency": "eur",
    "customer": {
      "name": "{{p|montse}}",
      "email": "{{e|montse@puigdevall.example}}",
      "phone": "{{ph|+34 600 555 014}}",
      "tax_id": "{{g|dni|00000000T}}"
    },
    "billing_address": {
      "line1": "{{a|Carrer de la Riera Alta 14, 3r 2a}}",
      "postal_code": "08001",
      "city": "{{l|Barcelona}}",
      "country": "ES"
    },
    "payment_method": {
      "type": "card",
      "card": { "brand": "visa", "last4": "4242", "number_test": "{{f|card-visa|4242 4242 4242 4242}}", "exp": "12/29" },
      "sepa_fallback": "{{f|iban|ES91 2100 0418 4502 0005 1332}}"
    }
  }
}

The signing secret for whk_live_03 is {{s|signing-secret|whsec_Lw9xTe4aKf02HxWmP7cYbR4LuD9sVa3E}} if you need it for the example.`)),

	code("issue-thread", "handles,thread", usr(`Summarize where this issue stands and who owes what.

**Crash on import when a row has a BOM** #918
opened on {{d|2026-02-12|Feb 12, 2026}} by {{p|rowan|handle|rowan-ashdown}}

> **{{p|rowan|handle|rowan-ashdown}}** commented on {{d|2026-02-12|Feb 12, 2026}}
> Importer dies with "invalid header" on files exported from {{o|Excel|product}}. Attaching one from a customer ({{p|precious}} at {{o|Ndlovu & Daughters Hardware|company-eponym}}). Stack trace in the gist.

> **{{p|basil|handle|bthorncroft}}** commented on {{d|2026-02-13|Feb 13, 2026}}
> Dup of #644? @{{p|lance|handle|lcorbin}} fixed the UTF-16 case there.

> **{{p|lance|handle|lcorbin}}** commented on {{d|2026-02-13|Feb 13, 2026}}
> Not a dup. Different code path. I can take it but not before the 20th. {{p|basil|first}}, can you get a redacted sample from {{p|precious|first}}? Her address and VAT number are in the one above (line 3: {{a|88 Jan Smuts Avenue, Rosebank, Johannesburg, 2196}}).

> **{{p|rowan|handle|rowan-ashdown}}** commented on {{d|2026-02-14|Feb 14, 2026}}
> Oops. Deleted the attachment. Thanks {{p|lance|lower-first|lance}}.
> -- {{p|rowan}}, Support Engineering

> **{{p|basil|handle|bthorncroft}}** commented on {{d|2026-03-02|Mar 2, 2026}}
> Emailed her at {{e|precious@ndlovu-hardware.example}}, no reply yet.`)),

	code("dataframe", "notebook,medical", usr(`df.head() is below. Write pandas code to compute age at admission and bucket into decades. Do not print the name columns.

In [14]: df.head(6)
Out[14]:
   mrn       patient_name              dob         admitted    phone             attending
0  00412983  {{p|shandiin|last-first|YAZZIE, SHANDIIN}}          {{d|1962-11-23|11/23/1962}}  {{d|2026-02-01}}  {{ph|505-555-0118}}     Dr. {{p|arjun|last}}
1  00412990  {{p|angel|last-first|BATISTA ROSARIO, ANGEL}}    {{d|1985-02-14|02/14/1985}}  {{d|2026-02-01}}  {{ph|787-555-0164}}     Dr. {{p|arjun|last}}
2  00413002  {{p|carol|last-first|WHITEHORSE, CAROL ANN}}     {{d|1948-07-30|07/30/1948}}  {{d|2026-02-02}}  {{ph|406-555-0131}}     Dr. {{p|harpreet|last}}
3  00413017  {{p|earl|last-first|WHITCOMBE, EARL}}           {{d|1939-03-08|03/08/1939}}  {{d|2026-02-03}}  {{ph|207-555-0109}}     Dr. {{p|harpreet|last}}
4  00413020  {{p|august|last-first|WENNERSTRÖM, AUGUST}}       {{d|1971-08-19|08/19/1971}}  {{d|2026-02-03}}  {{ph|612-555-0187}}     Dr. {{p|arjun|last}}
5  00413041  {{p|june|last-first|OYELARAN, JUNE}}            {{d|1996-06-06|06/06/1996}}  {{d|2026-02-04}}  {{ph|713-555-0155}}     Dr. {{p|harpreet|last}}

In [15]: df.attending.unique()
Out[15]: array(['Dr. {{p|arjun|last}}', 'Dr. {{p|harpreet|last}}'], dtype=object)

Context: {{p|arjun|initials-last|A. R. Iyer}} and {{p|harpreet|full}} are the two attendings on the ward; the month columns are named june and august in the wide table, which is unrelated to the patients above.`)),

	code("ldif", "directory,export", usr(`Write an ldapmodify script that moves these two entries to ou=alumni and clears the phone attributes.

dn: uid={{p|destiny|handle|dabara}},ou=people,dc=marlowe-college,dc=example
objectClass: inetOrgPerson
cn: {{p|destiny}}
sn: {{p|destiny|last}}
givenName: {{p|destiny|first}}
uid: {{p|destiny|handle|dabara}}
mail: {{e|d.abara@marlowe-college.example}}
telephoneNumber: {{ph|+234 803 555 0142}}
homePostalAddress: {{a|12 Adeola Odeku Street$Victoria Island$Lagos}}
employeeNumber: 204117

dn: uid={{p|blessing|handle|bokpara}},ou=people,dc=marlowe-college,dc=example
objectClass: inetOrgPerson
cn: {{p|blessing}}
sn: {{p|blessing|last}}
givenName: {{p|blessing|first}}
displayName: {{p|blessing|last-first|Okpara, Blessing (Registry)}}
uid: {{p|blessing|handle|bokpara}}
mail: {{e|blessing.okpara@marlowe-college.example}}
mobile: {{ph|0803 555 0177}}
manager: uid={{p|destiny|handle|dabara}},ou=people,dc=marlowe-college,dc=example
userPassword: {SSHA}{{s|password-hash|W6ph5Mm5Pz8GgiULbPgzG37mj9gnZQkLxyz8Ab==}}`)),

	code("mail-log", "logs", usr(`Which of these messages actually got delivered?

{{d|2026-03-04|Mar  4}} 06:12:41 mx1 postfix/smtpd[20114]: 4F1A2C0E3B: client=relay.example.net[192.0.2.44]
{{d|2026-03-04|Mar  4}} 06:12:41 mx1 postfix/cleanup[20120]: 4F1A2C0E3B: message-id=<20260304061240.4F1A2C0E3B@mx1.fennwick.example>
{{d|2026-03-04|Mar  4}} 06:12:42 mx1 postfix/qmgr[1187]: 4F1A2C0E3B: from=<{{e|jack.featherstonehaugh@fennwick.example}}>, size=48211, nrcpt=2 (queue active)
{{d|2026-03-04|Mar  4}} 06:12:44 mx1 postfix/smtp[20125]: 4F1A2C0E3B: to=<{{e|pearl.abernathy-singh@example.org}}>, relay=mx.example.org[198.51.100.7]:25, delay=2.9, dsn=2.0.0, status=sent (250 2.0.0 OK)
{{d|2026-03-04|Mar  4}} 06:12:45 mx1 postfix/smtp[20126]: 4F1A2C0E3B: to=<{{e|rusty.kowalewski@example.com}}>, relay=none, delay=3.4, dsn=4.4.1, status=deferred (connect to mx.example.com[203.0.113.9]:25: Connection timed out)
{{d|2026-03-04|Mar  4}} 06:14:02 mx1 dovecot: imap-login: Login: user=<{{p|jack|handle|jfeatherstonehaugh}}>, method=PLAIN, rip=192.0.2.101, TLS
{{d|2026-03-04|Mar  4}} 06:14:09 mx1 dovecot: imap-login: Aborted login (auth failed, 3 attempts in 6 secs): user=<{{p|rusty|handle|rusty}}>, rip=203.0.113.200
{{d|2026-03-04|Mar  4}} 06:20:00 mx1 postfix/qmgr[1187]: 4F1A2C0E3B: removed`)),

	code("sentry-event", "json,logs", usr(`Group these two events: same bug or different?

{
  "event_id": "{{n|hash|a3f1c9e07b2d4f58861e0c7d9b4a2e15}}",
  "timestamp": "{{d|2026-03-07|2026-03-07T11:02:55Z}}",
  "release": "web@{{n|version|7.41.0-rc.2}}+{{n|hash|1c7a9f0}}",
  "user": { "id": "u_55190", "username": "{{p|violet|handle|violet.oh}}", "email": "{{e|violet.okonkwo-hale@example.com}}", "ip_address": "198.51.100.23" },
  "exception": { "type": "TypeError", "value": "Cannot read properties of undefined (reading 'violet')" },
  "tags": { "theme": "violet", "browser": "{{o|Firefox|product}} 124.0", "os": "{{o|Fedora|product}} 40" },
  "breadcrumbs": [
    { "category": "ui.click", "message": "button#save-profile" },
    { "category": "xhr", "message": "PUT /api/profile 500", "data": { "display_name": "{{p|violet|nickname}} {{p|violet|last-part|Okonkwo}}" } }
  ]
}
{
  "event_id": "{{n|hash|0e8b6d4c2a1f3957b5d7f9e1c3a5b7d9}}",
  "timestamp": "{{d|2026-03-07|2026-03-07T11:09:13Z}}",
  "release": "web@{{n|version|7.41.0-rc.2}}+{{n|hash|1c7a9f0}}",
  "user": { "id": "u_60233", "username": "{{p|reed|handle|reedc}}", "email": "{{e|reed.castellano@example.net}}" },
  "exception": { "type": "TypeError", "value": "Cannot read properties of undefined (reading 'sage')" },
  "tags": { "theme": "sage", "browser": "{{o|Safari|product}} 17.4", "os": "{{o|macOS|product}} 14.4" }
}`)),

	code("review-comments", "code-review,nickname", usr(`Turn these review comments into a checklist for the author. Keep who asked for what.

internal/billing/proration.go
  L42  {{p|enzo|lower-first|lorenzo}}: this rounds half up, {{p|inaki|first}} said finance wants half even. can you confirm with him?
  L57  {{p|inaki}}: yes, half even. See the spec I sent {{p|enzo|nickname}} on {{d|2026-02-27|27/02}}.
  L57  {{p|jb|nickname}}: +1. Also the variable name {{n|identifier|maxRetries}} is doing two jobs here.
  L88  {{p|enzo|initials|LDB}}: nit, typo "recieve"
internal/billing/proration_test.go
  L12  {{p|marieeve}}: the fixture uses a real looking card ({{f|card-mastercard|5555 5555 5555 4444}}). Fine since it is a test number, but add a comment.
  L30  {{p|jb|first}}: can we get a case for a customer who upgrades on {{d|2028-02-29|Feb 29}}?
General
  {{p|marieeve|first}}: approving once {{p|enzo|misspelled|Lorenzo Di Benedeto}}'s rounding question is settled (sorry {{p|enzo|nickname}}, autocorrect).
  {{p|jb|full}}: LGTM after that. Ping me at {{e|jb.lefevre@atelier-nord.example}} if I miss the notification, I am offline until {{d|2026-03-09|lundi 9 mars}}.`)),

	code("regex-help", "phones,formats", usr(`I need one regex (PCRE) that matches all of these support callback numbers from our ticket export and nothing else on the page. Explain each alternation.

{{ph|(415) 555-0132}}
{{ph|415.555.0178}}
{{ph|+1 415 555 0191}}
{{ph|1-800-555-0100}}
{{ph|020 7946 0321}}
{{ph|+44 20 7946 0958}}
{{ph|07700 900461}}
{{ph|+44 7700 900 123}}
{{ph|+61 491 570 156}}
{{ph|(02) 5550 1234}}
{{ph|01 99 00 12 34}}
{{ph|+33 1 99 00 56 78}}
{{ph|030 23125 017}}
{{ph|+49 30 23125 902}}
{{ph|070-1740 605}}
{{ph|+46 8 465 004 71}}
{{ph|+353 20 915 5523}}

It must not match order numbers like {{n|number|4155-550-132}}, build ids like {{n|number|20260314.1552}}, or the ISBN {{n|number|978-0-306-40615-7}}.`)),

	code("mobile-crash", "logs,possessive", usr(`TestFlight crash from a beta tester. What is the likely cause?

Incident Identifier: {{n|identifier|6E1B0A3F-52C4-4D8B-9A77-0F2C3D4E5A6B}}
Hardware Model:      iPhone15,3
Device Name:         {{p|ximena|possessive}}'s iPhone
Process:             Tidewatch [4411]
Identifier:          example.tidewatch.app
Version:             3.8.1 (2291)
Date/Time:           {{d|2026-03-01}} 21:47:03.221 -0600
OS Version:          iPhone OS 17.4 (21E219)
Beta Tester:         {{e|xime.hernandez@example.com}}

Exception Type:  EXC_BAD_ACCESS (SIGSEGV)
Thread 0 Crashed:
0   Tidewatch   0x0000000104a3c1f8 TideChart.render(for:) + 412
1   Tidewatch   0x0000000104a3b9d0 TideChart.body.getter + 88
2   SwiftUI     0x00000001c9f0a1b4 ViewGraph.updateOutputs() + 164

Tester note: "crashes when I open the chart for {{l|Puerto Escondido}}. Signed in as {{p|ximena|handle|xime_hv}}. You can call me, {{ph|+52 1 55 5550 0167}}. {{p|ximena|nickname}}"`)),

	code("vcard-ics", "export,calendar", usr(`Parse these into JSON: one contact object and one event object with ISO dates.

BEGIN:VCARD
VERSION:3.0
N:{{p|ploy|last}};{{p|ploy|first}};;Khun;
FN:{{p|ploy}}
NICKNAME:{{p|ploy|nickname}}
ORG:{{o|Chao Phraya Analytics|company}}
TEL;TYPE=CELL:{{ph|+66 81 555 0142}}
EMAIL:{{e|siriporn.w@chaophraya-analytics.example}}
ADR;TYPE=HOME:;;{{a|88/14 Soi Sukhumvit 39}};{{l|Bangkok}};;10110;Thailand
BDAY:{{d|1990-09-12|19900912}}
END:VCARD

BEGIN:VEVENT
UID:{{n|identifier|b7c1d2e3-4f5a-6b7c-8d9e-0f1a2b3c4d5e}}@chaophraya-analytics.example
DTSTART;TZID=Asia/Bangkok:{{d|2026-03-17|20260317T140000}}
DTEND;TZID=Asia/Bangkok:{{d|2026-03-17|20260317T150000}}
SUMMARY:Quarterly review with {{p|gede|first}}
ORGANIZER;CN={{p|ploy}}:mailto:{{e|siriporn.w@chaophraya-analytics.example}}
ATTENDEE;CN={{p|gede}};PARTSTAT=ACCEPTED:mailto:{{e|gede.wirawan@example.org}}
ATTENDEE;CN={{p|aisyah}};PARTSTAT=TENTATIVE:mailto:{{e|nurul.aisyah@example.org}}
LOCATION:{{o|Baan Rim Naam|company}}, {{l|Bangkok}}
END:VEVENT`)),

	code("http-trace", "secrets,http", usr(`Login works in the browser but my script gets 401 on the second call. HAR excerpt:

> POST /api/session HTTP/2
> content-type: application/x-www-form-urlencoded
>
> username={{e|sage.delacroix%40example.com|url-encoded}}&password={{s|password|Lavande%2B1998%21}}
< HTTP/2 200
< set-cookie: sid={{s|session-cookie|s%3A7fK2mQ9xLp4Zt7Wc1Yb6.Nd3Vh5Sj0Ga8Tb3XkQ7mP2}}; Path=/; HttpOnly; Secure
< {"user":{"id":7731,"name":"{{p|sage}}","locale":"fr-CA"},"token":"{{s|jwt|eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI3NzMxIiwibmFtZSI6IlMuIEQuIiwiaWF0IjoxNzQxMjUxNzI3fQ.dGhpcy1pcy1ub3QtYS1yZWFsLXNpZ25hdHVyZQ}}"}

> GET /api/orders?since={{d|2026-02-01}} HTTP/2
> authorization: Bearer undefined
< HTTP/2 401
< {"error":"invalid_token"}`)),

	code("actions-yaml", "negative,config,secret-reference", usr(`Why does the publish job skip on tags? Workflow below.

name: release
on:
  push:
    tags: ['v*']
jobs:
  publish:
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@{{n|hash|b4ffde65f46336ab88eb53be808477a3936bae11}}
      - uses: actions/setup-node@v4
        with:
          node-version: {{n|version|20.11.1}}
          registry-url: https://registry.npmjs.org
      - run: npm ci
      - run: npm publish --provenance
        env:
          NODE_AUTH_TOKEN: ${{lb}} {{n|secret-reference|secrets.NPM_TOKEN}} {{rb}}
          {{n|identifier|GRACE_PERIOD_SECONDS}}: 30
      - name: Notify
        run: ./scripts/notify.sh "$GITHUB_REF_NAME"
        env:
          WEBHOOK_URL: ${{lb}} {{n|secret-reference|secrets.RELEASE_WEBHOOK}} {{rb}}`)),

	code("dependency-question", "negative,libraries", usr(`We are a {{o|Django|library}} shop moving a few services to {{o|Flask|library}}. Current stack: {{o|Celery|library}} with {{o|Redis|product}} as broker, {{o|Pandas|library}} for the reports, {{o|Jinja|library}} templates, {{o|Marshmallow|library}} for serialization, {{o|Faraday|library}} on the one {{o|Ruby|product}} service, {{o|Winston|library}} and {{o|Morgan|library}} for logging on the {{o|Node|product}} side, {{o|Luigi|tool}} for batch, and {{o|Travis CI|tool}} that we want to replace with {{o|Tekton|tool}} or {{o|Argo|tool}}. Search is {{o|Solr|product}}, moving to {{o|Elasticsearch|product}}. Someone suggested {{o|Otto|library}} for the JS sandbox and {{o|Apache Felix|product}} for plugins on the {{o|Java|product}} side, which sounds like a lot.

Questions:
1. Does {{o|Celery|library}} 5.4 still support {{o|Python|product}} 3.8?
2. Is {{o|Luigi|tool}} worth keeping or should we go to {{o|Airflow|tool}} or {{o|Dagster|tool}}?
3. {{o|Hermes|product}} vs {{o|V8|product}} for the mobile app: any reason to care?
4. Is {{o|Athena|product}} a sane replacement for our nightly {{o|Hive|product}} jobs?`)),

	code("clean-stacktrace", "negative,stack-trace,identifiers", usr(`Explain this panic. No user data involved, it happens at startup.

thread 'main' panicked at src/gc/{{n|identifier|mark_sweep}}.rs:214:17:
attempt to subtract with overflow
stack backtrace:
   0: rust_begin_unwind
   1: core::panicking::panic_fmt
   2: heap::gc::{{n|identifier|mark_sweep}}::Collector::{{n|identifier|mark_roots}}
             at ./src/gc/{{n|identifier|mark_sweep}}.rs:214:17
   3: heap::gc::{{n|identifier|mark_sweep}}::Collector::collect
   4: heap::alloc::Arena::{{n|identifier|grant_region}}
             at ./src/alloc/arena.rs:97:9
   5: heap::sched::{{n|identifier|round_robin}}::Scheduler::tick
   6: heap::codec::{{n|identifier|reed_solomon}}::Decoder::{{n|identifier|chase_errors}}
   7: heap::main
note: run with RUST_BACKTRACE=full for a verbose backtrace.

Relevant config: {{n|identifier|will_compact}} = true, {{n|identifier|max_pause_ms}} = 8, {{n|identifier|jack_port}} = "system:playback_1", {{n|identifier|ray_depth}} = 4.
Toolchain: rustc {{n|version|1.77.2}} ({{n|hash|25ef9e3d8}} 2024), cargo {{n|version|1.77.2}}.`)),

	code("checksums", "negative,hashes,versions", usr(`go mod verify fails on CI only. Relevant bits:

go: downloading golang.org/x/sync {{n|version|v0.6.0}}
verifying golang.org/x/sync@{{n|version|v0.6.0}}: checksum mismatch
	downloaded: h1:{{n|hash|5BMeUDZ7vkXGfEr1x9B4bRcTH4lpkTkpdh0T/J+qjbQ=}}
	go.sum:     h1:{{n|hash|f4VCD5DZ7vkXGfEr1x9B4bRcTH4lpkTkpdh0T/J+abc=}}

go.sum (excerpt)
github.com/google/uuid {{n|version|v1.6.0}} h1:{{n|hash|NIvaJDMOsjHA8n1jAhLSgzrAzy1Hgr+hNrb57e+94F0=}}
go.uber.org/zap {{n|version|v1.27.0}} h1:{{n|hash|aJMhYGrd5QSmlpLMr2MftRKl7t8J8PTZPA732ud/XR8=}}

Dockerfile base: golang:{{n|version|1.22.1-alpine3.19}}@sha256:{{n|hash|0466223b8544fb7d4ff04748acc4d75a608234bf4e79563bff208d2060c0dd79}}
Runner image build {{n|number|20240310.1.0}}, kernel {{n|version|6.5.0-1016}}, {{n|number|7946}} MiB free.
Cache key: gomod-{{n|hash|e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855}}
Public deploy key on the runner (not secret): ssh-ed25519 {{n|public-key|AAAAC3NzaC1lZDI1NTE5AAAAIJ7xQp2mK9vL4sR8wY6zA1cD5fG0hJ3nR7uE9iO2aS4d}} ci-runner`)),

	code("algorithms", "negative,eponyms", usr(`For an interview prep sheet, give me one line each on when to prefer {{n|eponym|Dijkstra}} over {{n|eponym|Bellman-Ford}}, {{n|eponym|Kruskal}} over {{n|eponym|Prim}}, and {{n|eponym|Knuth-Morris-Pratt}} over {{n|eponym|Boyer-Moore}}. Then explain a {{n|eponym|Bloom}} filter and {{n|eponym|Huffman}} coding like I am new to this, and why {{n|eponym|Levenshtein}} distance is quadratic.

Use the usual {{n|placeholder-name|Alice}} and {{n|placeholder-name|Bob}} framing for the {{n|eponym|Diffie-Hellman}} part, with {{n|placeholder-name|Eve}} listening and {{n|placeholder-name|Mallory}} tampering. Code in {{o|Haskell|product}} or {{o|Pascal|product}} is fine, I read both; avoid {{o|Erlang|product}}.`)),

	code("lookalike-numbers", "negative,versions,numbers", usr(`Sanity check my capacity notes before I paste them in the runbook:

- Postgres {{n|version|16.2}} on 3 nodes, {{n|number|192.0.2.14}} is the primary, replicas on .15 and .16
- pgbouncer pool {{n|number|120-400}}, statement timeout {{n|number|30-90}} s depending on route
- peak was {{n|number|415 550 0132}} rows scanned per hour (yes really), median {{n|number|8 005 550}}
- epoch of last vacuum: {{n|number|1741251727}}
- LSN at snapshot: {{n|number|3F/2A0001C8}}
- chart shows {{n|number|03/14}} shards rebalanced, {{n|number|12/25}} still pending
- WAL retention {{n|number|24-48}} h, ratio 4:3 between hot and warm tiers
- firmware on the controllers: {{n|version|2026.03.1}} and {{n|version|12.11.2025-lts}}
- MAC of the bad NIC: {{n|number|00:1B:44:11:3A:B7}}
- part number for the replacement: {{n|number|555-0142-REV-C}}
- RFC 5737 and RFC 2606 ranges only in examples, TLS 1.3, HTTP/2`)),
}
