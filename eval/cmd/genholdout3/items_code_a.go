package main

var itemsCodeA = []def{
	code("git-log", "git,branch-names",
		usr(`Can you turn this into release notes grouped by person? Skip the merge commits.

$ git log --since="2 weeks ago" --format=medium origin/main
commit {{n|hash|9f3c2e1a7b4d5f60718293a4b5c6d7e8f9012345}}
Merge: {{n|hash|4be1a02}} {{n|hash|c77d9f3}}
Author: {{p|takumi}} <{{e|takumi.nakashima@lumenwharf.example}}>
Date:   {{d|2026-03-09|Mon Mar 9 18:42:11 2026 +0900}}

    Merge branch 'fix/{{p|kwabena|handle|kobby}}/berth-timeout' into main

commit {{n|hash|c77d9f3e02b14a5d8c6f7a9b0e1d2c3b4a5f6e7d}}
Author: {{p|kwabena|full|Kwabena Boateng-Asare}} <{{e|kobby@lumenwharf.example}}>
Date:   {{d|2026-03-09|Mon Mar 9 09:15:02 2026 +0000}}

    berth-api: raise upstream timeout to 30s

    {{p|priyanka|first}} saw the 504s first, see her notes in BERTH-2291.
    Reviewed-by: {{p|priyanka}} <{{e|pvenkat@lumenwharf.example}}>
    Co-authored-by: {{p|joaquin|ascii-fold|Joaquin Larranaga}} <{{e|jlarranaga@lumenwharf.example}}>

commit {{n|hash|4be1a02d9c8e7f6a5b4c3d2e1f0a9b8c7d6e5f4a}}
Author: {{p|joaquin}} <{{e|jlarranaga@lumenwharf.example}}>
Date:   {{d|2026-03-06|Fri Mar 6 16:03:48 2026 -0300}}

    web: fix date picker in es-AR locale (thanks {{p|joaquin|nickname}} of last year for the TODO)

commit {{n|hash|1d0c9b8a7f6e5d4c3b2a1f0e9d8c7b6a5f4e3d2c}}
Author: {{p|mirela|ascii-fold|mirela kovacevic}} <{{e|mirela@users.noreply.example}}>
Date:   {{d|2026-03-04|Wed Mar 4 11:27:30 2026 +0100}}

    ci: pin node to {{n|version|20.11.1}}, bump pnpm to {{n|version|9.1.4}}

    Signed-off-by: {{p|mirela}} <{{e|mirela.kovacevic@lumenwharf.example}}>

$ git branch -r --sort=-committerdate | head -6
  origin/main
  origin/fix/{{p|kwabena|handle|kobby}}/berth-timeout
  origin/{{p|priyanka|handle|pvenkat}}/BERTH-2291-504s
  origin/feature/{{p|mirela|lower-first}}-node20
  origin/wip-{{p|takumi|handle|tnakashima}}-do-not-merge
  origin/dependabot/npm_and_yarn/vite-{{n|version|5.2.8}}`)),

	code("python-traceback", "stack-trace,paths",
		usr(`This started after I pulled this morning. Why is it trying to email someone from a migration?

(venv) {{p|rosalind|handle|rcheesewright}}@devbox:~/src/berth-api$ python manage.py migrate
Operations to perform:
  Apply all migrations: accounts, billing, sessions
Running migrations:
  Applying accounts.0042_backfill_owner_email...Traceback (most recent call last):
  File "/home/{{p|rosalind|path|rcheesewright}}/src/berth-api/manage.py", line 22, in <module>
    main()
  File "/home/{{p|rosalind|path|rcheesewright}}/.pyenv/versions/{{n|version|3.12.2}}/lib/python3.12/site-packages/django/db/migrations/executor.py", line 255, in apply_migration
    state = migration.apply(state, schema_editor)
  File "/home/{{p|rosalind|path|rcheesewright}}/src/berth-api/accounts/migrations/0042_backfill_owner_email.py", line 31, in backfill
    notify_owner(account)
  File "/home/{{p|rosalind|path|rcheesewright}}/src/berth-api/accounts/notify.py", line 12, in notify_owner
    raise SMTPRecipientsRefused({account.owner_email: (550, b"mailbox unavailable")})
smtplib.SMTPRecipientsRefused: {'{{e|d.ferreira-stone@corvanefreight.example}}': (550, b'mailbox unavailable')}

During handling of the above exception, another exception occurred:

  File "/home/{{p|rosalind|path|rcheesewright}}/src/berth-api/accounts/notify.py", line 19, in notify_owner
    log.error("could not notify %s (%s) for account %s", account.owner_name, account.owner_phone, account.id)
ValueError: owner_name='{{p|dale}}' owner_phone='{{ph|+1 (312) 555-0147}}' account={{n|identifier|acct_7Hq2LmX9}} is missing a locale

The blame says {{p|august|nickname}} added notify_owner on {{d|2026-02-27|Feb 27}} and I do not want to ping him on his day off if this is my local data.`)),

	code("compose-file", "config,yaml",
		usr(`Review this compose file before I push it. Anything that should not be committed?

# docker-compose.staging.yml
# owner: {{p|svenerik}} ({{p|svenerik|nickname}}), ask in #berth-infra
# last touched {{d|2026-01-19}} by {{p|lerato|initial-last|L. Fakude}}
services:
  api:
    image: registry.lumenwharf.example/berth-api:{{n|version|4.18.2}}
    labels:
      maintainer: "{{p|svenerik|ascii-fold|Sven Erik Bergqvist}} <{{e|sven-erik.bergqvist@lumenwharf.example}}>"
      oncall.phone: "{{ph|+46 70 174 06 25}}"
    environment:
      DATABASE_URL: {{s|connection-string|postgres://berth_app:Vq7!mRz2pLx9@db.staging.lumenwharf.example:5432/berth|Vq7!mRz2pLx9}}
      REDIS_URL: redis://cache:6379/0
      SENTRY_DSN: {{s|sentry-dsn|https://b3f1c9a27d5e4f60a1b2c3d4e5f60718@o482913.ingest.sentry.example/6021934|b3f1c9a27d5e4f60a1b2c3d4e5f60718}}
      SMTP_PASSWORD: "{{s|password|hunter-is-not-2-anymore}}"
      FEATURE_FLAGS_SDK_KEY: ${FLAGS_SDK_KEY}
      ADMIN_SEED_EMAIL: {{e|lerato.fakude@lumenwharf.example}}
      LOG_LEVEL: info
  db:
    image: postgres:{{n|version|16.2}}@sha256:{{n|checksum|4f8b1c7e9a2d3f5061728394a5b6c7d8e9f00112233445566778899aabbccdde}}
    environment:
      POSTGRES_PASSWORD: {{s|password|Vq7!mRz2pLx9}}
      POSTGRES_USER: berth_app
  cache:
    image: redis:{{n|version|7.2.4}}-alpine`)),

	code("sql-dump-users", "sql,data",
		usr(`I need a script that anonymises this table for the demo environment but keeps the row count and the plan distribution. Here is the dump of the first rows.

-- PostgreSQL database dump, table public.app_user
-- dumped by {{p|dawit|handle|dawitg}} on {{d|2026-04-02}}
COPY public.app_user (id, full_name, email, phone, plan, created_at, last_login_ip) FROM stdin;
1041	{{p|marguerite}}	{{e|m.delacroix-benoit@corvanefreight.example}}	{{ph|+33 1 99 00 42 17}}	enterprise	{{d|2024-11-05|2024-11-05 08:14:22+00}}	203.0.113.40
1042	{{p|obinna}}	{{e|obinna.c@corvanefreight.example}}	{{ph|+234 803 555 0142}}	enterprise	{{d|2024-11-05|2024-11-05 08:15:01+00}}	203.0.113.41
1043	{{p|siobhan|ascii-fold|Siobhan Ni Bhroin}}	{{e|siobhan.nib@corvanefreight.example}}	{{ph|+353 20 915 5012}}	team	{{d|2024-12-12|2024-12-12 17:40:09+00}}	198.51.100.7
1044	{{p|jurgen}}	{{e|juergen.weissgerber@corvanefreight.example}}	{{ph|+49 30 23125 118}}	team	{{d|2025-01-20|2025-01-20 06:02:55+00}}	198.51.100.23
1045	{{p|oisin|sql-escaped|Oisin O''Daly}}	{{e|oisin@lumenwharf.example}}	\N	internal	{{d|2025-02-01|2025-02-01 10:00:00+00}}	192.0.2.10
1046	{{p|maianh|caps|TRAN THI MAI ANH}}	{{e|maianh.tran@corvanefreight.example}}	{{ph|+84 28 5550 0193}}	free	{{d|2025-03-18|2025-03-18 02:31:44+00}}	203.0.113.77
\.

INSERT INTO public.audit_note (user_id, note) VALUES (1045, 'internal test account, owner {{p|oisin|nickname}} ({{p|oisin|ascii-fold|Oisin O Dalaigh}}), do not bill');
INSERT INTO public.audit_note (user_id, note) VALUES (1043, 'renamed at customer request, was "{{p|siobhan|alias|Siobhan Byrne}}"');`),
		ast(`A deterministic approach keeps the row count and the plan column and replaces every personal column with a value derived from the id, so reruns give the same result: full_name becomes "User 1041", email becomes {{n|placeholder|user1041@demo.invalid}}, phone becomes a number from a reserved range, and last_login_ip is set to a documentation address. The audit_note table also contains names in free text ("{{p|oisin|nickname}}", "{{p|siobhan|alias|Siobhan Byrne}}"), so it needs either truncating or a rewrite. Do you want the internal row for {{p|oisin|ascii-fold|Oisin O'Daly}} kept as it is?`),
		usr(`Keep row 1045 untouched, {{p|oisin|first}} uses it for the demo himself. Truncate audit_note. And one request from {{p|dawit|first}}: the generated names should keep the same script and rough length as the original, because the demo is about label layout, so {{p|maianh|caps|TRAN THI MAI ANH}} should not turn into "User 1046". Can the script do that without a lookup table of real names?`)),

	code("ci-log-short", "ci,log",
		usr(`CI is red on my PR and I cannot tell if it is my change. Log from the failing job:

Run actions/checkout@{{n|hash|b4ffde65f46336ab88eb53be808477a3936bae11}}
Syncing repository: lumenwharf/berth-web
Triggered by: {{p|yevheniia|handle|ytkachenko}} ({{p|yevheniia}}) on refs/heads/{{p|yevheniia|nickname|zhenya}}/price-table
##[group]Run pnpm install --frozen-lockfile
Lockfile is up to date, resolution step is skipped
Packages: +{{n|number|1412}}
Done in 21.4s
##[endgroup]
##[group]Run pnpm test -- --ci
 FAIL  src/billing/PriceTable.test.tsx
  PriceTable
    x renders the annual price for the seeded account (48 ms)

    expect(received).toBe(expected)
    Expected: "{{n|money|€1.188,00}}"
    Received: "{{n|money|€1,188.00}}"

      at Object.<anonymous> (src/billing/PriceTable.test.tsx:41:29)
      seed: { owner: "{{p|anneliese}}", email: "{{e|a.vogelsang@lumenwharf.example}}", locale: "de-DE" }
##[endgroup]
##[error]Process completed with exit code 1.
Run curl -s -X POST -H "Authorization: Bearer {{s|slack-token|xo`+`xb-4829173056-7730194825561-Qm3VxT9aLr6ZpW2cK8dYbN1h}}" https://slack.example/api/chat.postMessage -d channel=C04BERTH -d text="build failed for {{p|yevheniia|handle|ytkachenko}}"
Post job cleanup. Finished {{d|2026-05-14|2026-05-14T07:52:31Z}}`),
		ast(`The failure is a locale formatting difference, not a logic error: the test expects German grouping and the runner produced English grouping, which usually means the Node image lacks full ICU data. Separately, the last step prints a Slack bot token in clear text. That token should be rotated and moved to a masked secret.`),
		usr(`Good catch on the token, I will ask {{p|svenerik|nickname}} to rotate it. For the ICU thing, {{p|anneliese|nickname}} wrote that seed, should the fix be in her fixture or in the workflow image?`)),

	code("review-thread", "code-review",
		usr(`Summarise where this review landed and what is still open. I have to decide whether to merge before the freeze.

berth-api #1873 "Idempotency keys for label printing" opened by {{p|hyunwoo|handle|hwseo}}

{{p|hyunwoo|handle|hwseo}} commented {{d|2026-04-20|Apr 20}}
First pass at BERTH-2310. cc @{{p|thandiwe|handle|thandi-mp}} @{{p|ibrahim|handle|ielsayed}} since you both touched the printer client.

{{p|thandiwe|handle|thandi-mp}} reviewed {{d|2026-04-21|Apr 21}}  (changes requested)
> key := sha256(tenant + body)
This collides when two warehouses print the same pallet label. {{p|ibrahim|first}} hit exactly this at {{o|Corvane Freight}} last year. Please include the device id.

{{p|ibrahim|handle|ielsayed}} commented {{d|2026-04-21|Apr 21}}
+1 to {{p|thandiwe|nickname}}. Also the TTL: 24h is too long, {{p|hyunwoo|lower|seo hyun-woo}} and I agreed on 15 min in the design doc (see comment from "{{p|ibrahim|alias|Ibrahim Elsayed}}" there, old display name sorry).

{{p|hyunwoo|handle|hwseo}} commented {{d|2026-04-22|Apr 22}}
Fixed both in {{n|hash|e41f9a2}}. @{{p|clay|handle|cwhitlock}} can you check the migration? It adds a unique index on a table with a lot of {{x|common-word|clay}} in it. Sorry, a lot of rows in it. Autocorrect.

{{p|clay|handle|cwhitlock}} reviewed {{d|2026-04-23|Apr 23}}  (approved)
Index is fine, built concurrently. Nit: {{p|hyunwoo|first}}, your name is spelled "{{p|hyunwoo|misspelled|Hyun-wo Seo}}" in the AUTHORS diff.

{{p|thandiwe|handle|thandi-mp}} reviewed {{d|2026-04-24|Apr 24}}  (approved)
LGTM. {{p|clay|first}} already covered the index. Ship it before {{p|sue|possessive}}'s freeze on the 30th.`)),

	code("todo-comments", "source,comments",
		usr(`Clean up the comments in this file. Keep the ones that explain why, drop stale TODOs, and tell me which ones I need to ask a human about.

package manifest

// TODO({{p|takumi|handle|tnakashima}}): remove after the {{l|Osaka}} warehouse migrates, see BERTH-1998
// {{p|takumi|nickname}} said {{d|2025-09-30|2025-09-30}} at the latest. It is now well past that.
const legacyOsakaFormat = true

// FIXME {{p|cormac}} {{d|2025-11-14|14/11/2025}}: this regexp is wrong for Irish Eircodes,
// {{p|oisin|first}} has the list of test codes. Do NOT ship to {{o|An Post}} without fixing.
var eircode = regexp.MustCompile({{bt}}^[A-Z]\d{2} ?[A-Z0-9]{4}${{bt}})

// HACK: {{p|nilufar|possessive}}'s customer ({{o|Samarqand Silk Road Logistics}}) sends weights in poods.
// 1 pood = {{n|measurement|16.38 kg}}. Yes, really. Ask {{p|nilufar|nickname}}.
const poodKg = 16.38

// NOTE({{p|farzaneh|initials|fm}}): the retry budget below was tuned by {{p|farzaneh|initials|FM}} with {{p|bartosz|first}} ({{p|bartosz|handle|bwisniewski}})
// during the {{d|2026-01-08|8 Jan 2026}} incident. Do not change without rereading the postmortem.
const retryBudget = 3

// XXX temporary: hard-coded contact until the directory service is up
// {{p|teodora|ascii-fold|Teodora Vaduva}}, {{e|teodora.vaduva@lumenwharf.example}}, desk {{ph|+40 21 555 0176}}
var escalationContact = "{{e|teodora.vaduva@lumenwharf.example}}"

// {{n|eponym|Pascal}} case on purpose: generated clients expect it. See RFC {{n|number|7231}}.
type ManifestLine struct {
	SKU    string  // e.g. {{n|part-number|WX-4471-B}}
	Weight float64 // kilograms
}`)),

	code("shell-history", "shell,secrets-inline",
		usr(`I am writing the runbook for this migration from my shell history. Turn it into numbered steps and flag anything that should not go into a wiki page.

  501  ssh {{p|wade|handle|wkimathi}}@bastion.lumenwharf.example
  502  scp {{p|wade|handle|wkimathi}}@bastion.lumenwharf.example:/home/{{p|wade|path|wkimathi}}/dumps/berth_{{d|2026-06-01|2026-06-01}}.sql.gz .
  503  gunzip berth_{{d|2026-06-01|2026-06-01}}.sql.gz
  504  psql "{{s|connection-string|postgresql://migrator:r4Wd0g-Tangent-88@pg-new.lumenwharf.example/berth?sslmode=require|r4Wd0g-Tangent-88}}" -f berth_{{d|2026-06-01|2026-06-01}}.sql
  505  curl -u {{p|wade|handle|wkimathi}}:{{s|password|Kilimanjaro#5895m}} https://artifacts.lumenwharf.example/api/storage/berth/ | jq .
  506  export GITHUB_TOKEN={{s|github-token|gh`+`p_R7kV2mQ9xLp4ZtW8cYb3NfH6sJd1AeG5uTo0}}
  507  gh pr merge 1873 --squash --subject "Idempotency keys (#1873)"
  508  git log -1 --format='%an <%ae>'
  509  # -> {{p|hyunwoo|alias|Hyunwoo Seo}} <{{e|hyunwoo.seo@lumenwharf.example}}>
  510  mysql -h legacy-db.lumenwharf.example -u root --password={{s|password|Tr0ub4dor&3xtra}} -e "select count(*) from shipments"
  511  echo "done, tell {{p|sunny|first}} and {{p|guilherme|nickname}}" | wall
  512  history | grep -i passw
  513  echo -n hello | md5sum   # -> {{n|checksum|5d41402abc4b2a76b9719d911017c592}}
  514  sudo -u postgres createuser --pwprompt {{p|sunny|handle|sunnyb}}
  515  kubectl -n berth get pods -l app=api`)),
}
