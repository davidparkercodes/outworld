package main

var itemsTech = []def{
	it("git-log", "author-line,handle,family-name-first,common-word-name,initials",
		sys(`You write release notes from git history. Credit contributors by name.`),
		usr(`commit 4f9c2e1a7b3d
Author: {{p|bogdan|full|Bogdan Ionescu}} <{{e|bogdan.ionescu@tallgrass.example}}>
Date:   {{d|2026-11-10|Tue Nov 10 16:22:41 2026}} +0200

    fix(export): handle empty rows (reported by {{p|kasia|nickname|Kasia}})

commit a81d07c55e02
Author: {{p|zhang|handle|zhangweiqi}} <{{e|zhang.weiqi@tallgrass.example}}>
Date:   {{d|2026-11-09|Mon Nov 9 11:05:13 2026}} +0800

    revert "bump jackson to 2.18"

    Co-authored-by: {{p|kasia|full|Katarzyna Dąbrowska}} <{{e|kdabrowska@tallgrass.example}}>
    Reviewed-by: {{p|adjei|full|J. R. Adjei}}

commit 0c3be9917fa4
Author: {{p|bill|full|Bill Oyelaran}} <{{e|bill@tallgrass.example}}>
Date:   {{d|2026-11-06|Fri Nov 6 09:48:02 2026}} -0500

    chore: bill of materials script, skip signed artifacts`)),

	it("code-review", "markdown-split,handle,api-key,diacritics",
		usr(`Summarize what's still open on this PR.

Review on PR #812 "proration fixes" (author: @{{p|miles|handle|miles-abara}})

{{p|eino|markdown|**Eino** Väisänen}} requested changes:
1. {{bt}}proration.ts:42{{bt}} duplicates the rounding helper that {{p|zoe|markdown|_Zoë_}} added in #790. Reuse it.
2. {{bt}}client.ts:9{{bt}} has a live key checked in: {{bt}}apiKey: "{{s|sk-key|sk`+`-proj-8Hq2LmX4vT9cRb1ZyN6wKd3FjP7sUe5A}}"{{bt}}. Rotate it and move it to the secret store.
3. Tests only cover the happy path. Ask {{p|marit|first|Marit}}, she wrote the fixtures.

@{{p|miles|handle|miles-abara}} replied:
Fixed 1 and 3. On 2, that key belongs to {{p|soren|possessive|Søren Kjeldgaard}}'s sandbox, I'll ask him to rotate. Thanks {{p|eino|markdown|**Eino Väisänen**}}!`)),

	it("codeowners", "handle,url-slug,family-name-first,first-name",
		usr(`# CODEOWNERS
/services/ledger/      @{{p|harpreet|handle|harpreet-sandhu}} @{{p|tanvir|handle|tanvirc}}
/services/notify/      @{{p|yusuf|handle|yusuf-alkhatib}}
/infra/                @{{p|soren|handle|soren-k}}
*.sql                  @{{p|jiwoo|handle|jwoo-park}}

{{p|jiwoo|first|Ji-woo}} is leaving on the 30th. Who should take *.sql, and can you draft the PR description? Her profile for context: https://git.tallgrass.example/users/{{p|jiwoo|url-slug|jwoo-park}} and the handover doc is at https://wiki.tallgrass.example/~{{p|jiwoo|url-slug|park.jiwoo}}/handover`)),

	it("git-blame", "author-line,lowercase-name,hex-key,hyphenated",
		usr(`who do I ask about the pagerduty key, and should that key even be in the repo?

$ git blame -L 10,14 config/alerts.yaml
3e1f0a92 ({{p|chidinma|full|Chidinma Okwuosa-Bello}} {{d|2026-08-03}} 10:14:55 +0100 10) receivers:
3e1f0a92 ({{p|chidinma|full|Chidinma Okwuosa-Bello}} {{d|2026-08-03}} 10:14:55 +0100 11)   - name: oncall
7ab44c10 ({{p|akwasi|lower|akwasi}}                 {{d|2026-09-21}} 18:02:07 +0000 12)     pagerduty_key: {{s|hex-key|5b2f9e0c47d1a6833c0e9f7712ab64de}}
7ab44c10 ({{p|akwasi|lower|akwasi}}                 {{d|2026-09-21}} 18:02:07 +0000 13)     slack_channel: "#ops-{{p|akwasi|lower|akwasi}}-test"
c09d2f6b ({{p|fergus|full|Fergus McAllister}}      {{d|2026-10-02}} 09:30:00 +0100 14)     send_resolved: true`)),

	it("source-fixture", "test-fixture,lowercase-name,last-first,mononym,ascii-fold,product-name",
		usr(`This test fails, dedupe returns 4. Why?

# fixtures for the dedupe tests
USERS = [
    {"id": 1, "name": "{{p|kasia|full|Katarzyna Dąbrowska}}", "email": "{{e|kasia.d@example.org}}", "city": "{{l|Kraków}}"},
    {"id": 2, "name": "{{p|kasia|ascii-fold|katarzyna dabrowska}}", "email": "{{e|KASIA.D@EXAMPLE.ORG}}", "city": "{{l|Krakow}}"},
    {"id": 3, "name": "{{p|kasia|last-first|Dąbrowska, Katarzyna}}", "email": "{{e|k.dabrowska@example.com}}", "city": "{{l|Warsaw}}"},
    {"id": 4, "name": "{{p|thiha|full|Thiha}}", "email": "{{e|thiha@example.net}}", "city": "{{l|Yangon}}"},
]

def test_dedupe_merges_case_and_diacritics():
    # TODO({{p|olive|lower|olive}}): a {{o|Jasmine|product}}-style matcher would be nicer here
    assert len(dedupe(USERS)) == 3`)),

	it("config-yaml", "config,password,connection-string,particle-name,uk-phone",
		usr(`Review this for anything that should not be committed:

# deploy/values-prod.yaml  (owner: {{p|willem|lower|willem}}, backup: {{p|pieter|handle|pieter.degroot}})
smtp:
  host: smtp.mailrelay.example
  username: {{e|alerts@tallgrass.example}}
  password: "{{s|password|c0rm0rant-Ledge-91!}}"
oncall:
  primary:
    name: {{p|willem|full|Willem van der Heijden}}
    phone: "{{ph|+44 20 7946 0321}}"
  secondary:
    name: {{p|pieter|full|Pieter de Groot}}
    phone: "{{ph|+44 7700 900412}}"
database:
  url: {{s|connection-string|postgres://ledger_app:Gannet%40Harbor7@db-prod.tallgrass.example:5432/ledger|Gannet%40Harbor7}}
jwt_secret: {{s|hex-key|3f7a1c9e5b2d4086a7c1e3f95b2d7a40c6e8f1a3}}`)),

	it("sql-insert", "sql,ssn,sql-escaped-name,two-surnames",
		usr(`Migrate this seed data to the new schema, which has a single full_name column and stores dob as text in DD/MM/YYYY.

INSERT INTO patients (id, last_name, first_name, dob, ssn, phone) VALUES
  (1001, '{{p|khalid|last|Al-Mazrouei}}', '{{p|khalid|first|Khalid}}', '{{d|1975-02-11}}', '{{g|ssn-dashed|987-65-4321}}', '{{ph|202-555-0166}}'),
  (1002, '{{p|declan|sql-escaped|O''Shaughnessy}}', '{{p|declan|first|Declan}}', '{{d|1983-09-30}}', '{{g|ssn-dashed|000-12-3456}}', '{{ph|202-555-0104}}'),
  (1003, '{{p|vale|last|Gutiérrez Paredes}}', '{{p|vale|first|Valentina}}', '{{d|1990-12-01}}', '{{g|ssn-dashed|512-00-7734}}', '{{ph|202-555-0179}}');`)),

	it("csv-rows", "csv,last-first,mononym,particle-name,diacritics",
		usr(`Give me a pivot of average salary by manager, and flag rows where the name format looks off.

employee_id,name,manager,email,start_date,salary
E-1042,"{{p|harpreet|last-first|Sandhu, Harpreet Kaur}}","{{p|amber|last-first|Nakagawa, Amber}}",{{e|hsandhu@tallgrass.example}},{{d|2026-12-07}},131000
E-0977,"{{p|eino|last-first|Väisänen, Eino}}","{{p|amber|last-first|Nakagawa, Amber}}",{{e|evaisanen@tallgrass.example}},{{d|2024-03-18}},142500
E-0611,"{{p|pieter|last-first|de Groot, Pieter}}","{{p|willem|last-first|van der Heijden, Willem}}",{{e|pdegroot@tallgrass.example}},{{d|2021-06-01}},118000
E-0003,{{p|wahyuni|full|Wahyuni}},,{{e|wahyuni@tallgrass.example}},{{d|2015-01-12}},203000`)),

	it("json-log", "json,handle,two-surnames,timestamp",
		usr(`Why would this webhook fire twice? Here is one of the two payloads:

{"event":"user.updated","ts":"{{d|2026-11-20|2026-11-20T04:11:09Z}}","actor":{"id":"u_8812","login":"{{p|mafer|handle|mafer.lopez}}","display":"{{p|mafer|full|María Fernanda López Aguirre}}"},"changes":{"email":["{{e|mafer@old-mail.example}}","{{e|mf.lopez.aguirre@example.com}}"],"phone":[null,"{{ph|+1 787 555 0137}}"]},"ip":"203.0.113.44","ua":"Mozilla/5.0"}`)),

	it("dotenv", "config,aws-key,github-token,password,lowercase-name,possessive",
		usr(`I accidentally pushed this. What do I need to rotate, in what order?

# .env.local, DO NOT COMMIT ({{p|kenji|lower|kenji}}'s machine)
AWS_ACCESS_KEY_ID={{s|aws-key-id|AK`+`IAIOSFODNN7EXAMPLE}}
AWS_SECRET_ACCESS_KEY={{s|aws-secret|wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY}}
GITHUB_TOKEN={{s|github-token|gh`+`p_R7tVq2LxN9bKc4WmZ8sYd1HfJ6pEa3UoG5iT}}
SENTRY_DSN={{s|dsn|https://a1b2c3d4e5f60718293a4b5c6d7e8f90@o000000.ingest.sentry.example/42|a1b2c3d4e5f60718293a4b5c6d7e8f90}}
ADMIN_EMAIL={{e|kenji.morishita@tallgrass.example}}
ADMIN_PASSWORD={{s|password|hunter2-but-Longer!}}`)),

	it("source-java", "code-comment,author-tag,handle,first-initial,product-name",
		usr(`Refactor this class so the contact string comes from config. Keep the javadoc.

/**
 * LedgerReconciler
 * @author {{p|tanvir|full|Tanvir Chowdhury}} <{{e|tanvir@tallgrass.example}}>
 * @author {{p|harpreet|handle|hsandhu}}
 * @since {{d|2025-04-02}}
 */
public class LedgerReconciler {
    // HACK({{p|tanvir|lower|tanvir}}): {{o|Mockito|product}} can't stub the final class, ask {{p|arjun|first-initial|Arjun V.}} before touching
    private static final String SUPPORT_CONTACT = "Call {{p|rohan|first|Rohan}} at {{ph|669-555-0183}}";

    public Result reconcile(Batch batch) {
        return Result.ok(batch.size());
    }
}`)),

	it("ssh-config", "config,file-path,handle,windows-path,lowercase-name",
		usr(`# ~/.ssh/config
Host bastion
  HostName bastion.tallgrass.example
  User {{p|lukasz|handle|lwisniewski}}
  IdentityFile /Users/{{p|lukasz|path|lukasz}}/.ssh/id_ed25519_work

Host {{p|meiling|lower|mei}}-box
  HostName 198.51.100.23
  User {{p|meiling|handle|mtsai}}
  IdentityFile C:\Users\{{p|meiling|path|Mei-Ling}}\.ssh\id_rsa

why does ssh {{p|meiling|lower|mei}}-box still ask for a password? I copied the second block from my coworker's laptop.`)),

	it("stack-trace", "file-path,home-directory,name-in-email,token",
		usr(`Getting this on the cron box after rotating tokens. I set it with: export LEDGER_TOKEN={{s|api-token|lk_live_4c9a1e7d2b8f40a6935e}}

Traceback (most recent call last):
  File "/home/{{p|akwasi|path|akwasi}}/projects/ledger-sync/sync.py", line 212, in <module>
    main()
  File "/home/{{p|akwasi|path|akwasi}}/projects/ledger-sync/sync.py", line 198, in main
    client = LedgerClient(token=os.environ["LEDGER_TOKEN"])
  File "/home/{{p|akwasi|path|akwasi}}/.venvs/ledger/lib/python3.12/site-packages/ledgerkit/client.py", line 41, in __init__
    raise AuthError(f"token rejected for user {user}")
ledgerkit.errors.AuthError: token rejected for user {{e|akwasi.boateng@tallgrass.example}}`)),

	it("shell-output", "home-directory,handle,product-name,file-path",
		usr(`which of these home directories can I safely archive? nobody has seen {{p|fatou|first|Fatou}} log in for weeks

$ ls -la /home
drwxr-x---  41 {{p|arjun|path|arjun}}    {{p|arjun|path|arjun}}    4096 {{d|2026-11-18|Nov 18}} 22:10 {{p|arjun|path|arjun}}
drwxr-x---  17 {{p|fatou|handle|fdiallo}}  {{p|fatou|handle|fdiallo}}  4096 {{d|2026-11-02|Nov  2}} 09:31 {{p|fatou|handle|fdiallo}}
drwxr-x---  23 {{p|jose|path|jose}}     {{p|jose|path|jose}}     4096 {{d|2026-10-30|Oct 30}} 14:02 {{p|jose|path|jose}}
drwxr-xr-x   3 {{o|jenkins|product}}  {{o|jenkins|product}}  4096 {{d|2025-01-14|Jan 14  2025}} {{o|jenkins|product}}`)),

	it("stack-trace", "windows-path,file-path,lowercase-name",
		usr(`Unhandled exception. System.IO.FileNotFoundException: Could not find file 'C:\Users\{{p|meiling|path|Mei-Ling}}\Documents\Tax 2025\w2_{{p|meiling|path|tsai}}.pdf'.
   at System.IO.FileStream..ctor(String path, FileMode mode)
   at TaxPrep.Importer.Load(String path) in D:\src\taxprep\Importer.cs:line 57
   at TaxPrep.Program.Main(String[] args) in D:\src\taxprep\Program.cs:line 12

the file is definitely there, I can see it in Explorer. what am I missing`)),

	it("server-log", "handle,jwt,last-first,name-in-email,diacritics",
		sys(`You are an on-call assistant. Explain log excerpts and suggest the next diagnostic step.`),
		usr(`{{d|2026-11-21}} 03:14:07,512 ERROR [http-nio-8443-exec-7] c.t.auth.SessionFilter - session expired for user={{p|siobhan|handle|siobhan.oriordan}} uid=4471 ip=192.0.2.15
{{d|2026-11-21}} 03:14:07,530 WARN  [http-nio-8443-exec-7] c.t.auth.SessionFilter - stale bearer {{s|jwt|eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI0NDcxIiwibmFtZSI6IlMuIE8nUmlvcmRhbiJ9.Qm9ndXNTaWduYXR1cmVGb3JFdmFsT25seQ}}
{{d|2026-11-21}} 03:14:09,001 INFO  [scheduler-2] c.t.mail.Digest - sent weekly digest to 3 recipients: {{e|s.oriordan@tallgrass.example}}, {{e|fergus@tallgrass.example}}, {{e|e.macleod@tallgrass.example}}
java.lang.IllegalStateException: profile missing for "{{p|siobhan|last-first|O'Riordan, Siobhán}}"
	at com.tallgrass.profile.ProfileService.load(ProfileService.java:88)
	at com.tallgrass.auth.SessionFilter.doFilter(SessionFilter.java:131)`)),

	it("shell-output", "home-directory,handle,password,mononym,file-path",
		usr(`trying to get this report over to {{p|bao|first|Bảo}}, what am I doing wrong?

~ % whoami
{{p|joao|path|joao}}
~ % pwd
/Users/{{p|joao|path|joao}}
~ % scp reports/q3.xlsx {{p|bao|handle|bao}}@198.51.100.40:/home/{{p|bao|path|bao}}/inbox/
{{p|bao|handle|bao}}@198.51.100.40's password:
Permission denied, please try again.
~ % curl -u {{p|joao|handle|joao.carvalho}}:{{s|password|Sabia-Laranja_88}} https://files.tallgrass.example/api/upload -F file=@reports/q3.xlsx
{"error":"user {{p|joao|handle|joao.carvalho}} is locked, contact admin ({{p|wahyuni|full|Wahyuni}})"}`)),
}
