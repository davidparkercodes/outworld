package main

var itemsCodeB = []def{
	code("package-json", "manifest,versions",
		usr(`npm publish fails with "invalid author field". What is wrong with this package.json? Please do not change any versions.

{
  "name": "@lumenwharf/berth-label-kit",
  "version": "{{n|version|3.4.0-rc.2}}",
  "description": "Label templates for Berth",
  "author": "{{p|guilherme}} <{{e|gui@lumenwharf.example}}> (https://lumenwharf.example/~{{p|guilherme|handle|gmascarenhas}}",
  "contributors": [
    { "name": "{{p|adaeze}}", "email": "{{e|adaeze.okpara-lindqvist@lumenwharf.example}}" },
    { "name": "{{p|wenjing|western-order|Wenjing Huang}}", "email": "{{e|hwenjing@lumenwharf.example}}", "url": "https://git.lumenwharf.example/{{p|wenjing|handle|hwenjing}}" },
    "{{p|dawit}} <{{e|dawit.g@lumenwharf.example}}>"
  ],
  "repository": { "type": "git", "url": "git+ssh://{{n|identifier|git@git.lumenwharf.example}}/{{p|guilherme|handle|gmascarenhas}}/berth-label-kit.git" },
  "engines": { "node": ">={{n|version|20.11.0}}" },
  "dependencies": {
    "pdf-lib": "{{n|version|^1.17.1}}",
    "bwip-js": "{{n|version|4.3.2}}",
    "date-fns": "{{n|version|~3.6.0}}",
    "zod": "{{n|version|3.23.8}}"
  },
  "devDependencies": {
    "typescript": "{{n|version|5.4.5}}",
    "vitest": "{{n|version|1.6.0}}"
  },
  "publishConfig": { "registry": "https://npm.lumenwharf.example/" }
}`)),

	code("lockfile-excerpt", "lockfile,hashes,negative",
		usr(`Two machines resolve different trees from the same lockfile. Diff these two excerpts and tell me which entries actually differ. Do not normalise anything, I need the exact strings back.

--- machine A: pnpm-lock.yaml
  /date-fns@{{n|version|3.6.0}}:
    resolution: {integrity: {{n|checksum|sha512-fRHTG8g/Gif+kSh50gaGEdToemgfj74aRX3swtiouboip5JDLAyDE9F11nHMIcvOaXeOC6D7SpNhi7uFyB7Uww==}}}
  /zod@{{n|version|3.23.8}}:
    resolution: {integrity: {{n|checksum|sha512-XBx9AXhXktjUqnepgTiE5flcKIYWi/rme0Eaj+5Y0lftuGBq+jyRu/md4WnuxqgP1ubdpNCsYEYPxrzVHD8d6g==}}}
  /bwip-js@{{n|version|4.3.2}}:
    resolution: {integrity: {{n|checksum|sha512-Qm9vdHN0cmFwIGlzIG5vdCBhIHNlY3JldCBidXQgaXQgbG9va3MgbGlrZSBvbmUgdG8gYSBzY2FubmVyIQ==}}}
    hasBin: true

--- machine B: pnpm-lock.yaml
  /date-fns@{{n|version|3.6.0}}:
    resolution: {integrity: {{n|checksum|sha512-fRHTG8g/Gif+kSh50gaGEdToemgfj74aRX3swtiouboip5JDLAyDE9F11nHMIcvOaXeOC6D7SpNhi7uFyB7Uww==}}}
  /zod@{{n|version|3.23.6}}:
    resolution: {integrity: {{n|checksum|sha512-RTHJlZhsRbuA8Hmp/iNL7jnfc4nZishjsanDAfEY1QpDQZCahUp3xDzl+zfweE9BklxMUcgBgS1b7Lvie/ZVwA==}}}
  /bwip-js@{{n|version|4.3.2}}:
    resolution: {integrity: {{n|checksum|sha512-Qm9vdHN0cmFwIGlzIG5vdCBhIHNlY3JldCBidXQgaXQgbG9va3MgbGlrZSBvbmUgdG8gYSBzY2FubmVyIQ==}}}
    hasBin: true

go.sum on the same branch, for completeness:
github.com/jackc/pgx/v5 {{n|version|v5.5.5}} h1:{{n|checksum|amBjrZVmksIdNjxGW/IiIMzxMKZFelXbUoPNb+8sjQw=}}
github.com/jackc/pgx/v5 {{n|version|v5.5.5}}/go.mod h1:{{n|checksum|ez9gk+OAat140fv9ErkZDYFWmXLfV+++K0uAOiwgm1A=}}
golang.org/x/crypto {{n|version|v0.22.0}} h1:{{n|checksum|g1v0xeRhjcugydODzvb3mEM9SQ0HGp9s/nh3COQ/C30=}}
build id {{n|uuid|7c9e6679-7425-40de-944b-e07fc1f90ae7}} on runner {{n|identifier|gh-runner-0412}}`)),

	code("git-blame", "git,blame",
		usr(`Who should review a change to this function? Blame output below, I only know two of these people.

$ git blame -L 88,104 --date=short internal/rates/zone.go
{{n|hash|a3f9c1d2}} ({{p|bartosz|ascii-fold|Bartosz Wisniewski}}  {{d|2024-06-11}}  88) func zoneFor(postcode string, carrier Carrier) (Zone, error) {
{{n|hash|a3f9c1d2}} ({{p|bartosz|ascii-fold|Bartosz Wisniewski}}  {{d|2024-06-11}}  89) 	p := strings.ToUpper(strings.TrimSpace(postcode))
{{n|hash|7be02f4c}} ({{p|teodora}}      {{d|2025-02-03}}  90) 	if carrier.Country == "RO" && len(p) == 6 {
{{n|hash|7be02f4c}} ({{p|teodora}}      {{d|2025-02-03}}  91) 		return romaniaZone(p)
{{n|hash|7be02f4c}} ({{p|teodora}}      {{d|2025-02-03}}  92) 	}
{{n|hash|19d4e8aa}} ({{p|wenjing|lower|huang wenjing}}       {{d|2025-08-27}}  93) 	if z, ok := overrides[p]; ok {
{{n|hash|19d4e8aa}} ({{p|wenjing|lower|huang wenjing}}       {{d|2025-08-27}}  94) 		return z, nil
{{n|hash|19d4e8aa}} ({{p|wenjing|lower|huang wenjing}}       {{d|2025-08-27}}  95) 	}
{{n|hash|c0ffee12}} ({{p|sunny}}  {{d|2025-12-19}}  96) 	// {{x|common-word|sunny}}-day path: the table lookup
{{n|hash|c0ffee12}} ({{p|sunny}}  {{d|2025-12-19}}  97) 	z, ok := carrier.Table[p[:3]]
{{n|hash|00000000}} (Not Committed Yet   {{d|2026-05-02}}  98) 	if !ok && carrier.Fallback != nil {
{{n|hash|00000000}} (Not Committed Yet   {{d|2026-05-02}}  99) 		return carrier.Fallback(p)
{{n|hash|00000000}} (Not Committed Yet   {{d|2026-05-02}} 100) 	}
{{n|hash|a3f9c1d2}} ({{p|bartosz|ascii-fold|Bartosz Wisniewski}}  {{d|2024-06-11}} 101) 	if !ok {
{{n|hash|a3f9c1d2}} ({{p|bartosz|ascii-fold|Bartosz Wisniewski}}  {{d|2024-06-11}} 102) 		return Zone{}, ErrUnknownZone
{{n|hash|a3f9c1d2}} ({{p|bartosz|ascii-fold|Bartosz Wisniewski}}  {{d|2024-06-11}} 103) 	}
{{n|hash|a3f9c1d2}} ({{p|bartosz|ascii-fold|Bartosz Wisniewski}}  {{d|2024-06-11}} 104) 	return z, nil

I know {{p|bartosz|nickname}} and {{p|teodora|nickname}}. {{p|bartosz|nickname}} is on parental leave until {{d|2026-07-01|July 1st}}.`),
		ast(`From the blame, the function has four authors. {{p|bartosz|ascii-fold|Bartosz Wisniewski}} wrote the original and the error path, {{p|teodora}} added the Romanian branch, {{p|wenjing}} added the override lookup, and {{p|sunny}} touched the table lookup most recently, in December. Lines 98 to 100 are your own uncommitted change. Since your change sits between the override lookup and the table lookup, the most relevant reviewers are {{p|wenjing|western-order|Wenjing Huang}} and {{p|sunny|first}}; {{p|teodora|nickname}} is a reasonable backup while {{p|bartosz|nickname}} is away.`),
		usr(`I do not know {{p|wenjing|first}} or {{p|sunny|first}}. Which is the family name in "{{p|wenjing|lower}}", and how do I address them in the review request without getting it wrong? Also write the request for me: two sentences, mention that it must merge before {{p|bartosz|first}} is back on {{d|2026-07-01|1 July}}.`)),

	code("codeowners", "config,owners",
		usr(`Convert this CODEOWNERS plus the old MAINTAINERS file into one table: path, primary, backup, how to reach them.

# CODEOWNERS
/internal/rates/        @{{p|bartosz|handle|bwisniewski}} @{{p|teodora|handle|tvaduva}}
/internal/printer/      @{{p|thandiwe|handle|thandi-mp}} @{{p|ibrahim|handle|ielsayed}}
/web/                   @{{p|yevheniia|handle|ytkachenko}} @{{p|anneliese|handle|avogelsang}}
/deploy/                @{{p|svenerik|handle|sebergqvist}} @lumenwharf/infra
*.sql                   @{{p|clay|handle|cwhitlock}}
/docs/                  @{{p|sue|handle|sharrowgate}}

# MAINTAINERS (legacy, {{d|2023-10-02|2 October 2023}})
{{p|bartosz|caps-last}}, {{p|bartosz|first}} <{{e|bartosz.wisniewski@lumenwharf.example}}> rates, zones. Mobile {{ph|+48 22 555 0131}}
{{p|thandiwe|caps|THANDIWE MOKOENA-PRETORIUS}} <{{e|thandi@lumenwharf.example}}> printer drivers ({{l|Johannesburg}} office, UTC+2)
{{p|sue}} ({{p|sue|alias|Susan Harrowgate}} in the HR system) <{{e|sue.harrowgate@lumenwharf.example}}> docs, release notes, the freeze calendar
{{p|svenerik|initial-last|S-E. Bergqvist}} <{{e|sven-erik.bergqvist@lumenwharf.example}}> infra, pager {{ph|070-1740 625}}
Former: {{p|cormac}} (left {{d|2025-12-31|31 Dec 2025}}, mail now bounces, was {{e|cormac.fitzgibbon@lumenwharf.example}})`)),

	code("java-stack-trace", "stack-trace,customer-data",
		usr(`Production exception from the label service. Is the customer payload being logged on purpose? And what is the root cause?

{{d|2026-02-11|2026-02-11 03:14:07,512}} ERROR [label-worker-3] c.l.berth.label.LabelRenderer - render failed for shipment {{n|identifier|SHP-00482917}}
com.lumenwharf.berth.label.RenderException: address block overflow: Recipient{name='{{p|gudrun}}', street='{{a|Laugavegur 118, 105 Reykjavík}}', phone='{{ph|+354 555 0164}}', email='{{e|gudrun.vil@postur.example}}'}
	at com.lumenwharf.berth.label.AddressBlock.fit(AddressBlock.java:142)
	at com.lumenwharf.berth.label.LabelRenderer.render(LabelRenderer.java:88)
	at com.lumenwharf.berth.label.LabelWorker.lambda$run$0(LabelWorker.java:51)
	at java.base/java.util.concurrent.ThreadPoolExecutor.runWorker(ThreadPoolExecutor.java:1144)
	at java.base/java.lang.Thread.run(Thread.java:1583)
Caused by: java.lang.StringIndexOutOfBoundsException: begin 0, end 35, length 21
	at java.base/java.lang.String.checkBoundsBeginEnd(String.java:4861)
	at com.lumenwharf.berth.label.AddressBlock.truncate(AddressBlock.java:201)
	... 4 common frames omitted
{{d|2026-02-11|2026-02-11 03:14:07,530}} WARN  [label-worker-3] c.l.berth.label.LabelWorker - retry {{x|fraction|1/3}} for {{n|identifier|SHP-00482917}}, sender={{o|Corvane Freight}} contact={{p|tevita|initial-last|T. Fifita-Latu}}
{{d|2026-02-11|2026-02-11 03:14:09,101}} ERROR [label-worker-3] c.l.berth.label.LabelRenderer - render failed for shipment {{n|identifier|SHP-00482917}}
com.lumenwharf.berth.label.RenderException: address block overflow: Recipient{name='{{p|gudrun|ascii-fold|Gudrun Vilhjalmsdottir}}', street='{{a|Laugavegur 118, 105 Reykjavik}}', phone='{{ph|+354 555 0164}}', email='{{e|gudrun.vil@postur.example}}'}
	at com.lumenwharf.berth.label.AddressBlock.fit(AddressBlock.java:142)
{{d|2026-02-11|2026-02-11 03:14:09,102}} INFO  [label-worker-3] c.l.berth.label.LabelWorker - gave up, paging {{p|ibrahim|lower-last|el-sayed}} (primary) then {{p|thandiwe|lower-first}} (secondary)
JVM: OpenJDK {{n|version|21.0.2+13}}, build {{n|hash|f2ca8a1b}}, heap {{n|measurement|512 MiB}}`)),

	code("agent-grep-session", "agent,tool-calls",
		sys(`You are a coding agent working in the user's repository. You can call tools. Be concise.`),
		usr(`Find every place where we still hard-code a real person's contact details in tests or fixtures. {{p|priyanka}} wants them gone before the audit.`),
		ast(`I will search the test and fixture directories for email addresses and phone-like strings.

[tool_call] grep -rnE "@[a-z-]+\.example|\+?[0-9][0-9 ()-]{8,}" tests/ fixtures/ --include=*.py --include=*.json --include=*.yaml`),
		usr(`[tool_result]
tests/billing/test_invoice.py:14:    customer = make_customer(name="{{p|ngozi}}", email="{{e|ngozi.aderibigbe@corvanefreight.example}}")
tests/billing/test_invoice.py:15:    assert invoice.total_cents == {{n|number|1299900}}
tests/billing/test_invoice.py:40:    # regression for BERTH-1766, reported by {{p|ngozi|first}} on {{d|2025-10-09|2025-10-09}}
tests/sms/test_notify.py:8:PAGER = "{{ph|+1 415 555 0188}}"  # {{p|august|possessive}}'s real number, replace!!
tests/sms/test_notify.py:9:FAKE = "{{n|placeholder|+15005550006}}"
fixtures/users.yaml:3:  - name: {{p|hemi}}
fixtures/users.yaml:4:    email: {{e|hemi.tuhoro@corvanefreight.example}}
fixtures/users.yaml:5:    phone: "{{ph|+64 21 555 0119}}"
fixtures/users.yaml:9:  - name: {{n|placeholder-name|Test User}}
fixtures/users.yaml:10:    email: {{n|placeholder|test@example.com}}
fixtures/users.yaml:11:    phone: "{{n|placeholder|000-000-0000}}"
fixtures/carriers.json:22:    "support_line": "{{n|number|0800 00 1066}}", "name": "{{o|Hermes Parcel|company}}"
fixtures/carriers.json:57:    "account_manager": "{{p|hemi|ascii-fold|Hemi Tuhoro}} <{{e|hemi.tuhoro@corvanefreight.example}}>"`),
		ast(`Found three real people across four files: {{p|ngozi}} in tests/billing/test_invoice.py, {{p|august|possessive}}'s pager number in tests/sms/test_notify.py, and {{p|hemi}} in fixtures/users.yaml and fixtures/carriers.json. The "{{n|placeholder-name|Test User}}" entries and the carrier support line are placeholders or public numbers and can stay. Shall I replace the real ones with generated fixture data?`),
		usr(`Yes. Keep the comment on line 40 but drop her name from it, and leave {{n|placeholder|test@example.com}} and the {{n|placeholder|+15005550006}} magic number alone, the SMS sandbox needs that exact value. Tell me when {{p|august|nickname}} can stop worrying.`)),

	code("nginx-access-log", "log,query-strings",
		usr(`Write me an awk one-liner that counts 5xx per path from logs like these. Also, is anything in here personal data? Legal is asking.

203.0.113.9 - - [{{d|2026-03-21|21/Mar/2026:23:58:01 +0000}}] "GET /v2/track?ref={{n|identifier|CVF-88213-QA}}&email={{e|rashida.begum@corvanefreight.example}} HTTP/1.1" 200 1843 "-" "Mozilla/5.0"
203.0.113.9 - - [{{d|2026-03-21|21/Mar/2026:23:58:04 +0000}}] "GET /v2/track?ref={{n|identifier|CVF-88213-QA}}&email={{e|rashida.begum@corvanefreight.example}} HTTP/1.1" 502 166 "-" "Mozilla/5.0"
198.51.100.88 - {{p|reed|handle|rcalloway}} [{{d|2026-03-22|22/Mar/2026:00:00:13 +0000}}] "POST /admin/users/1044/impersonate HTTP/1.1" 302 0 "https://berth.lumenwharf.example/admin/users?q={{p|jurgen|url-slug|juergen+weissgerber}}" "curl/{{n|version|8.6.0}}"
198.51.100.88 - {{p|reed|handle|rcalloway}} [{{d|2026-03-22|22/Mar/2026:00:00:15 +0000}}] "GET /admin/users/1044 HTTP/1.1" 200 5120 "-" "curl/{{n|version|8.6.0}}"
192.0.2.61 - - [{{d|2026-03-22|22/Mar/2026:00:02:40 +0000}}] "GET /reset?token={{s|reset-token|rst_9aK2mQ7xVp4LtZ8cW3bYn6Hs}}&u={{p|penelope|url-slug|penelope.arkwright-sodhi}} HTTP/1.1" 200 912 "-" "Mozilla/5.0"
192.0.2.61 - - [{{d|2026-03-22|22/Mar/2026:00:02:59 +0000}}] "POST /reset HTTP/1.1" 500 87 "-" "Mozilla/5.0"
203.0.113.200 - - [{{d|2026-03-22|22/Mar/2026:00:03:11 +0000}}] "GET /static/app.{{n|hash|3e7a91bc}}.js HTTP/1.1" 304 0 "-" "Mozilla/5.0"
203.0.113.200 - - [{{d|2026-03-22|22/Mar/2026:00:03:12 +0000}}] "GET /v2/labels/{{n|uuid|0b9f6a2e-5c1d-4e7f-9a3b-2d4c6e8f0a1b}}/pdf?sms={{ph|%2B447700900152|url-encoded}} HTTP/1.1" 503 0 "-" "okhttp/{{n|version|4.12.0}}"`)),

	code("pandas-head", "notebook,dataframe",
		usr(`Why does my groupby give 4 groups for country when there are clearly 3? Output of the cell:

In [14]: df[["customer", "contact_email", "country", "signup", "mrr_eur"]].head(8)
Out[14]:
                      customer                         contact_email country      signup  mrr_eur
0        {{p|katarzyna}}  {{e|k.wojciechowska@baltyk-cargo.example}}      PL  {{d|2025-04-01}}    {{n|number|1250.0}}
1                 {{p|zoltan|family-first|Kerekes Zoltán}}           {{e|zoli@dunaspedicio.example}}      HU  {{d|2025-04-03}}     {{n|number|480.0}}
2  {{p|katarzyna|ascii-fold|katarzyna wojciechowska}}  {{e|k.wojciechowska@baltyk-cargo.example}}     PL   {{d|2025-04-01}}    {{n|number|1250.0}}
3          {{p|oluwaseun}}        {{e|seun@lagosfreightline.example}}      NG  {{d|2025-05-19}}     {{n|number|990.0}}
4               {{p|oluwaseun|nickname}} {{p|oluwaseun|last}}        {{e|seun@lagosfreightline.example}}      NG  {{d|2025-05-19}}     {{n|number|990.0}}
5               {{p|zoltan}}           {{e|zoli@dunaspedicio.example}}      HU  {{d|2025-06-30}}     {{n|number|480.0}}
6                       NaN                                   NaN     NaN         NaT        NaN
7          {{p|brook}}        {{e|brook.tesfaye@addisair.example}}      ET  {{d|2025-07-07}}     {{n|number|310.0}}

In [15]: df.country.unique()
Out[15]: array(['PL', 'HU', 'PL ', 'NG', nan, 'ET'], dtype=object)`)),
}
