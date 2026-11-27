package main

func longGenerated() []def {
	return []def{
		long("long-agent-session", "long,agent,tool-calls",
			sys(`You are a coding agent in the user's repository. You may run commands through tool calls. Explain what you find briefly and ask before changing files.`),
			usr(`The test suite on main has gone flaky since Friday. Run it, find what is actually failing, and tell me who I should talk to. {{p|priyanka}} thinks it is the exports tests again.`),
			ast(`I will run the suite once with verbose output and durations.

[tool_call] pytest -v --durations=0 -p no:cacheprovider`),
			usr(`[tool_result]
============================= test session starts ==============================
platform linux -- Python {{n|version|3.12.3}}, pytest-{{n|version|8.2.0}}, pluggy-{{n|version|1.5.0}}
rootdir: /home/{{p|priyanka|path|pvenkat}}/src/berth-api
configfile: pyproject.toml
collected 130 items

`+pytestLog(132, map[int]string{
				23:  `FAILED tests/exports/test_exports_023.py::test_paginates_export_in_stable_order - AssertionError: row 2 owner expected '{{p|oluwaseun}}' but got '{{p|oluwaseun|nickname}} {{p|oluwaseun|last}}' (fixture tenants.yaml, tenant {{o|Lagos Freightline}})`,
				24:  `    E   first differing element 2: {'owner': '{{p|oluwaseun}}', 'email': '{{e|seun@lagosfreightline.example}}', 'phone': '{{ph|+234 1 555 0167}}'}`,
				57:  `FAILED tests/auth/test_auth_057.py::test_refuses_token_signed_with_rotated_key - jwt.exceptions.InvalidSignatureError: key id kid-2026-03 not found`,
				58:  `    E   token subject {{e|wade.kimathi@lumenwharf.example}} issued {{d|2026-05-05|2026-05-05T18:40:12Z}}, fixture written by {{p|wade|handle|wkimathi}}`,
				90:  `SKIPPED tests/labels/test_labels_090.py::test_renders_address_block_within_label_bounds - needs the physical printer at {{l|Avonmouth}}, ask {{p|tevita|first}} ({{ph|07700 900388}})`,
				101: `FAILED tests/exports/test_exports_101.py::test_paginates_export_in_stable_order - AssertionError: 48213 rows expected, 48212 returned; missing consignee '{{p|gudrun}}' ({{a|Laugavegur 118, 105 Reykjavík}})`,
				117: `ERROR tests/billing/test_billing_117.py::test_writes_one_ledger_row_per_charge - psycopg.OperationalError: connection to server at "db.staging.lumenwharf.example", port 5432 failed: FATAL: password authentication failed for user "berth_app" (dsn {{s|connection-string|postgres://berth_app:Vq7!mRz2pLx9@db.staging.lumenwharf.example:5432/berth|Vq7!mRz2pLx9}})`,
			})+`
=========================== short test summary info ============================
3 failed, 125 passed, 1 skipped, 1 error in 44.61s`),
			ast(`Three failures and one error. Two failures are in the exports tests and both come from fixture data that is compared against a sorted export: one tenant owner is stored under a nickname, and one consignee with non-ASCII letters is dropped. The auth failure is a rotated signing key that the fixture token still references. The billing error is an expired staging password, and that connection string with its password is printed in the test output, which is worth fixing on its own.

To see who changed the exports fixtures recently I will check the history.

[tool_call] git log -6 --date=short --format="%h %ad %an <%ae> %s" -- tests/exports fixtures/tenants.yaml`),
			usr(`[tool_result]
{{n|hash|e41f9a2}} {{d|2026-05-08|2026-05-08}} {{p|hyunwoo|alias|Hyunwoo Seo}} <{{e|hyunwoo.seo@lumenwharf.example}}> exports: sort by owner then id (BERTH-2310 follow-up)
{{n|hash|93be7d0}} {{d|2026-05-07|2026-05-07}} {{p|dawit}} <{{e|dawit.g@lumenwharf.example}}> fixtures: add Icelandic consignee for the label overflow regression
{{n|hash|d7719b4}} {{d|2026-05-06|2026-05-06}} {{p|nilufar|ascii-fold|Nilufar Bakhtiyorova}} <{{e|nilufar@lumenwharf.example}}> fixtures: tenant owners use display names
{{n|hash|2c6f8e1}} {{d|2026-04-30|2026-04-30}} {{p|priyanka}} <{{e|pvenkat@lumenwharf.example}}> exports: stable pagination cursor
{{n|hash|b83d5f7}} {{d|2026-04-22|2026-04-22}} {{p|hyunwoo|alias|Hyunwoo Seo}} <{{e|hyunwoo.seo@lumenwharf.example}}> exports: idempotency keys (#1873)
{{n|hash|5a0e9c3}} {{d|2026-04-02|2026-04-02}} {{p|clay}} <{{e|clay.whitlock@lumenwharf.example}}> sql: unique index on export_job`),
			ast(`The timeline fits. {{p|nilufar}} changed tenant owners to display names on {{d|2026-05-06|6 May}}, {{p|dawit}} added the Icelandic consignee on {{d|2026-05-07|7 May}}, and {{p|hyunwoo|western-order|Hyun-woo Seo}} changed the sort order on {{d|2026-05-08|8 May}}, which was the Friday the flakiness began. The sort now depends on a field that two fixtures made unstable. I would talk to {{p|hyunwoo|first}} first, with {{p|nilufar|first}} and {{p|dawit|first}} for the fixtures. The auth failure belongs to {{p|wade}}, whose fixture token predates the key rotation.

Would you like me to make the sort deterministic by adding the id as a tiebreaker under a locale-independent collation, and regenerate the auth fixture?`),
			usr(`Yes to both, but do not touch {{p|dawit|possessive}}'s Icelandic row, {{p|gudrun|first}} is a real customer contact at {{o|Corvane Freight}}'s Icelandic partner and legal wants that address out of the repo anyway. I will ask {{p|dawit|first}} to replace it. For the billing error, the new staging password is in my .env.test, here:

DATABASE_URL={{s|connection-string|postgres://berth_app:nW4-tangerine-Quay-19@db.staging.lumenwharf.example:5432/berth|nW4-tangerine-Quay-19}}
TEST_ADMIN_EMAIL={{e|pvenkat@lumenwharf.example}}
TEST_ADMIN_PASSWORD={{s|password|only-for-staging-7731}}
TEST_SMS_NUMBER={{n|placeholder|+15005550006}}

Use it for this run only and do not write it into any file. Ping me, {{p|priyanka|nickname}}, when it is green.`)).on("2026-05-06", "nilufar", "2026-05-07", "dawit", "2026-05-08", "hyunwoo"),

		long("long-ci-log", "long,ci,log",
			usr(`The nightly pipeline failed again. Full log below. Give me the one-paragraph cause, the failing test, and anything in the log that should not be there (I already suspect the submodule step).

`+buildLog(168, 7260, map[int]string{
				0:   `Pipeline #48213 for lumenwharf/berth-rates@{{n|hash|a3f9c1d2}} started {{d|2026-03-18|2026-03-18}} by schedule, last commit by {{p|bartosz}} <{{e|bartosz.wisniewski@lumenwharf.example}}>`,
				3:   `Fetching submodule vendor/zone-data from {{s|url-credentials|https://tvaduva:gl` + `pat-Ro0MjTf2XiNvCa9BwEyD@git.lumenwharf.example/lumenwharf/zone-data.git|gl` + `pat-Ro0MjTf2XiNvCa9BwEyD}}`,
				4:   `Submodule path 'vendor/zone-data': checked out '{{n|hash|7be02f4c9d8e7f6a5b4c3d2e1f0a9b8c7d6e5f4a}}' (author {{p|teodora}}, {{d|2026-03-17|Tue Mar 17 22:41:09 2026 +0200}})`,
				70:  `warning: crates/zones/src/overrides.rs:14:5: TODO({{p|wenjing|handle|hwenjing}}): remove the {{l|Shenzhen}} override after {{d|2026-03-31|31 March}}, agreed with {{p|sunny|first}}`,
				71:  `warning: crates/zones/src/overrides.rs:22:5: FIXME ask {{p|bartosz|nickname}} (on leave, mobile {{ph|+48 601 555 013}}) whether prefix 81 covers all of {{l|Gdynia}}`,
				120: `test zones::case_120::returns_expected_zone_for_known_prefix ... FAILED`,
				121: `---- zones::case_120 stdout ---- thread 'zones::case_120' panicked at crates/zones/src/lookup.rs:88:9: no zone for postcode 81-389, consignee "{{p|agnieszka}}" <{{e|a.szczepanska@baltyk-cargo.example}}>, street "{{a|ul. Świętojańska 71/4}}"`,
				122: `note: run with RUST_BACKTRACE=1 environment variable to display a backtrace; fixture row added by {{p|teodora|handle|tvaduva}} in {{n|hash|7be02f4c}}`,
				160: `Uploading coverage report with CODECOV_TOKEN={{s|codecov-token|4f8b1c7e-9a2d-4f50-8172-8394a5b6c7d8}} to https://coverage.lumenwharf.example`,
				165: `test result: FAILED. 83 passed; 1 failed; 0 ignored; finished in 9.41s`,
				166: `Sending failure mail to {{e|berth-rates-dev@lumenwharf.example}}, {{e|bartosz.wisniewski@lumenwharf.example}}, {{e|teodora.vaduva@lumenwharf.example}}`,
				167: `Job failed: exit code 101. Retry policy: none. Runner {{n|identifier|ci-runner-07}}, image rust:{{n|version|1.77.2}}@sha256:{{n|checksum|2be8daddbb82756f7d1f2c7ece706aadcb284bf6ab6d769ea695cc3ed6016743}}`,
			}))),

		long("long-lockfile", "long,lockfile,hashes",
			usr(`npm ci works on my machine and fails in CI with EINTEGRITY. I have pasted the config and the whole lock file section for the eslint tree. Find the entry that is wrong. Please do not reformat or "fix" any integrity value or version, I need to diff your answer against the file.

--- .npmrc ---
registry=https://registry.npmjs.org/
@lumenwharf:registry=https://npm.lumenwharf.example/
//npm.lumenwharf.example/:_authToken={{s|npm-token|np`+`m_Zk4Vq9LmX2pRt7WcY8bN3fHsJd6AeG1uTo5Q}}
email={{e|adaeze.okpara-lindqvist@lumenwharf.example}}

--- package.json (top) ---
{
  "name": "@lumenwharf/berth-web",
  "version": "{{n|version|7.12.0}}",
  "author": "{{p|adaeze}} <{{e|adaeze.okpara-lindqvist@lumenwharf.example}}>",
  "contributors": ["{{p|yevheniia}} <{{e|yevheniia.tkachenko@lumenwharf.example}}>", "{{p|anneliese}} <{{e|a.vogelsang@lumenwharf.example}}>"],
  "private": true
}

--- package-lock.json ---
{
  "name": "@lumenwharf/berth-web",
  "version": "{{n|version|7.12.0}}",
  "lockfileVersion": 3,
  "requires": true,
  "packages": {
`+lockEntries(128, map[int]string{
				40: `    "node_modules/@lumenwharf/berth-label-kit": {
      "version": "{{n|version|3.4.0-rc.2}}",
      "resolved": "{{s|url-credentials|https://npm.lumenwharf.example/@lumenwharf/berth-label-kit/-/berth-label-kit-3.4.0-rc.2.tgz?authToken=np` + `m_Zk4Vq9LmX2pRt7WcY8bN3fHsJd6AeG1uTo5Q|np` + `m_Zk4Vq9LmX2pRt7WcY8bN3fHsJd6AeG1uTo5Q}}",
      "integrity": "{{n|checksum|sha512-Qm9vdHN0cmFwIGlzIG5vdCBhIHNlY3JldCBidXQgaXQgbG9va3MgbGlrZSBvbmUgdG8gYSBzY2FubmVyIQ==}}",
      "license": "UNLICENSED"
    },
`,
			})+`    "node_modules/zod": {
      "version": "{{n|version|3.23.8}}",
      "resolved": "https://registry.npmjs.org/zod/-/zod-3.23.8.tgz",
      "integrity": "{{n|checksum|sha512-XBx9AXhXktjUqnepgTiE5flcKIYWi/rme0Eaj+5Y0lftuGBq+jyRu/md4WnuxqgP1ubdpNCsYEYPxrzVHD8d6g==}}",
      "license": "MIT"
    }
  }
}

{{p|adaeze|nickname}} regenerated the lock on {{d|2026-05-28|28 May}} from her laptop in {{l|Gothenburg}}; CI runs in {{l|Frankfurt}}. {{p|yevheniia|nickname}} says it worked for her on the 29th.`)),

		long("long-app-log", "long,log,service",
			usr(`This is the gateway and worker log around the incident. Build me a timeline of the customer-visible events only (ignore the routine lines) and list every line that contains data we should not be logging.

--- berth, eu-central, {{d|2026-06-13|2026-06-13}}, level>=debug ---
`+svcLog(236, 7860, map[int]string{
				12:  `level=info svc=gateway msg="login" user={{e|chad.wexford@corvanefreight.example}} tenant=t_corvane ip=203.0.113.61 mfa=totp`,
				13:  `level=info svc=gateway msg="export requested" user={{e|chad.wexford@corvanefreight.example}} name="{{p|chad}}" rows=48213 format=csv`,
				40:  `level=warn svc=worker msg="export slow" tenant=t_corvane requested_by="{{p|chad|last-first}}" took=41000ms`,
				41:  `level=debug svc=worker msg="row sample" consignee="{{p|aino}}" phone="{{ph|+358 40 555 0149}}" street="{{a|Satamakatu 14 B, 33200 Tampere}}"`,
				77:  `level=error svc=labels msg="render failed" shipment={{n|identifier|SHP-00482917}} recipient="{{p|gudrun|ascii-fold|Gudrun Vilhjalmsdottir}}" email={{e|gudrun.vil@postur.example}} err="address block overflow"`,
				78:  `level=error svc=labels msg="render failed" shipment={{n|identifier|SHP-00482918}} recipient="{{p|bjarni}}" email={{e|bjarni@postur.example}} err="address block overflow"`,
				110: `level=warn svc=gateway msg="throttle applied" tenant=t_corvane by={{p|thandiwe|handle|thandi-mp}} reason="export storm" ticket=INC-2026-0613`,
				111: `level=info svc=gateway msg="pager ack" oncall="{{p|thandiwe}}" phone={{ph|+27 82 555 0126}} after_s=62`,
				150: `level=debug svc=ledger msg="dsn" value={{s|connection-string|postgres://ledger_rw:Hs6-Kd1AeG5u-Ro0@pg-ledger.lumenwharf.example/ledger|Hs6-Kd1AeG5u-Ro0}}`,
				151: `level=error svc=ledger msg="charge failed" customer="{{p|jordan}}" card={{f|card-visa|4012 8888 8888 1881}} iban={{f|gb-iban|GB09 LOYD 3096 1745 8203 61}} err="do not honour"`,
				190: `level=info svc=gateway msg="support note" author={{p|leilani|handle|lei-k}} text="called {{p|chad|first}} on {{ph|0117 496 0188}}, he will rerun the export after 20:00; cc {{p|jordan|first}} and {{p|ibrahim|first}}"`,
				220: `level=info svc=gateway msg="password reset" user={{e|jordan.mwangi-hale@corvanefreight.example}} link={{s|reset-link|https://berth.lumenwharf.example/reset/a5b6c7d8-e9f0-0112-2334-455667788990?sig=Tf2XiNvCa9BwEyDq|a5b6c7d8-e9f0-0112-2334-455667788990}}`,
				235: `level=info svc=gateway msg="incident closed" by="{{p|ibrahim}}" postmortem_due={{d|2026-06-17|2026-06-17}}`,
			}))).on("2026-06-17", "ibrahim"),

		long("long-sql-dump", "long,sql,dump",
			usr(`We are moving this schema to the new warehouse. Write the migration for the two small tables (customer_contact and driver) so that personal fields end up in a separate, access-controlled table, and tell me whether shipment_event contains anything personal. Dump follows.

-- dump taken {{d|2026-07-01|2026-07-01 03:00:02 UTC}} by {{p|clay|handle|cwhitlock}} from berth-prod replica
CREATE TABLE customer_contact (id integer PRIMARY KEY, tenant text, full_name text, email text, phone text, street text, city text, country char(2), tax_id text, note text);
INSERT INTO customer_contact VALUES (1, 't_corvane', '{{p|marguerite}}', '{{e|m.delacroix-benoit@corvanefreight.example}}', '{{ph|07700 900102}}', '{{a|220 Quayside Road}}', '{{l|Bristol}}', 'GB', NULL, 'primary');
INSERT INTO customer_contact VALUES (2, 't_corvane', '{{p|chad}}', '{{e|chad.wexford@corvanefreight.example}}', '{{ph|0117 496 0188}}', '{{a|220 Quayside Road}}', '{{l|Bristol}}', 'GB', NULL, 'finance, runs the June export');
INSERT INTO customer_contact VALUES (3, 't_baltyk', '{{p|katarzyna}}', '{{e|k.wojciechowska@baltyk-cargo.example}}', '{{ph|+48 58 555 0102}}', '{{a|ul. Polska 17}}', '{{l|Gdynia}}', 'PL', '{{g|pl-pesel|79021423875}}', 'sole trader until 2024, PESEL kept by mistake');
INSERT INTO customer_contact VALUES (4, 't_duna', '{{p|zoltan|family-first|Kerekes Zoltán}}', '{{e|zoli@dunaspedicio.example}}', '{{ph|+36 1 555 0128}}', '{{a|Váci út 152}}', '{{l|Budapest}}', 'HU', NULL, NULL);
INSERT INTO customer_contact VALUES (5, 't_pohjola', '{{p|aino}}', '{{e|aino.makiranta@pohjolanrahti.example}}', '{{ph|+358 40 555 0149}}', '{{a|Satamakatu 14 B}}', '{{l|Tampere}}', 'FI', '{{g|fi-hetu|140390-152K}}', 'do not call before 9');
INSERT INTO customer_contact VALUES (6, 't_marmara', '{{p|oguz}}', '{{e|oguzhan@marmaralojistik.example}}', '{{ph|+90 212 555 01 58}}', '{{a|Bağdat Caddesi No: 318}}', '{{l|İstanbul}}', 'TR', '{{g|tr-tckn|51739082690}}', 'owner');
INSERT INTO customer_contact VALUES (7, 't_illimani', '{{p|fernando}}', '{{e|fernando.iturralde@illimani.example}}', '{{ph|+591 2 555 0138}}', '{{a|Av. 6 de Agosto 2440}}', '{{l|La Paz}}', 'BO', NULL, 'angry, see #2291');
INSERT INTO customer_contact VALUES (8, 't_lagos', '{{p|oluwaseun|sql-escaped|Oluwaseun ''Seun'' Adegbenro}}', '{{e|seun@lagosfreightline.example}}', '{{ph|+234 1 555 0167}}', '{{a|Plot 7, Admiralty Way}}', '{{l|Lagos}}', 'NG', '{{g|ng-nin|48291730564}}', 'spouse works at Corvane');
INSERT INTO customer_contact VALUES (9, 't_aegean', '{{p|stavros}}', '{{e|s.papadimitrakopoulos@aegeanreefer.example}}', '{{ph|+30 21 0555 0136}}', '{{a|Akti Miaouli 85}}', '{{l|Piraeus}}', 'GR', '{{g|gr-afm|048291739}}', 'name is truncated on labels');
INSERT INTO customer_contact VALUES (10, 't_addis', '{{p|tigist}}', '{{e|tigist.h@addisair.example}}', '{{ph|+251 11 555 0152}}', NULL, '{{l|Addis Ababa}}', 'ET', NULL, 'no family name; W/ro {{p|tigist|first}}');
INSERT INTO customer_contact VALUES (11, 't_tbk', '{{p|maarten}}', '{{e|maarten@tbkcoldchain.example}}', '{{ph|06 5550 0172}}', '{{a|Nijverheidsweg 31}}', '{{l|Barneveld}}', 'NL', '{{g|nl-bsn|217849362}}', 'BSN from the supplier form, delete');
INSERT INTO customer_contact VALUES (12, 't_nile', '{{p|mostafa}}', '{{e|mostafa.shalaby@nilecargo.example}}', '{{ph|+20 2 5550 0171}}', '{{a|18 Talaat Harb Street}}', '{{l|Cairo}}', 'EG', '{{g|eg-nid|29001011234567}}', NULL);

CREATE TABLE driver (id integer PRIMARY KEY, name text, licence_no text, dob date, mobile text, depot text, bank_iban text);
INSERT INTO driver VALUES (1, '{{p|baraka}}', '{{g|gb-driving-licence|OLESA851127B99XX}}', '{{d|1985-11-27|1985-11-27}}', '{{ph|07700 900514}}', 'Avonmouth', '{{f|gb-iban|GB78 BARC 2004 1583 9026 47}}');
INSERT INTO driver VALUES (2, '{{p|hemi}}', '{{g|gb-driving-licence|TUHOR911220H99ZZ}}', '{{d|1990-11-22|1990-11-22}}', '{{ph|07700 900755}}', 'Avonmouth', NULL);
INSERT INTO driver VALUES (3, '{{p|jurgen}}', '{{g|de-fuehrerschein|B072RRE2I55}}', '{{d|1968-05-09|1968-05-09}}', '{{ph|+49 171 39200 18}}', 'Duisburg', '{{f|de-iban|DE53 2105 0170 0012 3485 67}}');
INSERT INTO driver VALUES (4, '{{p|torbjorn}}', '{{g|no-fnr|19087534106}}', '{{d|1975-08-19|1975-08-19}}', '{{ph|+47 55 55 50 62}}', 'Bergen', '{{f|no-iban|NO17 3624 0751 928}}');

CREATE TABLE shipment_event (id bigint PRIMARY KEY, shipment_id bigint, kind text, hub text, seq integer, weight_g integer, scanned_by text, note text);
`+sqlEventRows(96, map[int]string{
				30: `INSERT INTO shipment_event (id, shipment_id, kind, hub, seq, weight_g, scanned_by, note) VALUES (700390, 48417, 'exception', 'GDN1', 3, 12400, '{{p|bartosz|handle|bwisniewski}}', 'manual: consignee {{p|agnieszka}} refused, call {{ph|+48 58 555 0127}}');`,
				61: `INSERT INTO shipment_event (id, shipment_id, kind, hub, seq, weight_g, scanned_by, note) VALUES (700793, 48291, 'delivered', 'ATH1', 6, 8800, 'scanner-22', 'signed by {{p|ludovica|initial-last|L. Sanfilippo}} on {{d|2026-06-29|29/06/2026}}, left with neighbour at {{a|Akti Miaouli 87}}');`,
				88: `INSERT INTO shipment_event (id, shipment_id, kind, hub, seq, weight_g, scanned_by, note) VALUES (701144, 48733, 'customs_hold', 'LOS1', 2, 15250, 'scanner-09', 'auto: lane 17, belt 5, tote 381; broker {{p|chidinma}} <{{e|chidinma@lagosbrokers.example}}>');`,
			})+`
-- end of dump, {{n|number|112}} rows`)),

		long("long-migration-chat", "long,agent,source-file",
			usr(`I inherited this import script from {{p|cormac}} (he left in December) and I need to run it on a customer's legacy export tomorrow. First question: does anything in it look unsafe for personal data? Here is the module, then five sample rows from the customer's file.

# legacy_import.py
# author: {{p|cormac}} <{{e|cormac.fitzgibbon@lumenwharf.example}}>, {{d|2025-06-12|12 June 2025}}
# reviewed: {{p|sue|initial-last|S. Harrowgate}}
`+etlModule()+`

# sample.csv (semicolon separated, from {{o|Ten Brinke Koeltransport B.V.}})
consignee_name;consignee_street;consignee_city;consignee_postcode;consignee_country;consignee_phone;consignee_email;declared_value;currency;weight_kg
{{p|femke}};{{a|Assendorperstraat 212}};{{l|Zwolle}};8012 CE;NL;{{ph|06 5550 0129}};{{e|femke.vdmeulen@zwollepost.example}};1250,00;EUR;18,5
{{p|joris|caps|JORIS DE BRUIJN-ACKERMANS}};{{a|Grote Markt 7}};{{l|Zwolle}};8011 LW;NL;{{ph|038 555 0142}};{{e|j.debruijn@notariszwolle.example}};90,00;EUR;0,4
{{p|anouk|last-first|Hoogendoorn, Anouk}};{{a|Nijverheidsweg 31}};{{l|Barneveld}};3771 ME;NL;{{ph|+31 342 555 013}};{{e|debiteuren@tbkcoldchain.example}};n/a;EUR;2,0
{{p|willem|lower|willem van der meulen}};{{a|Assendorperstraat 212}};{{l|Zwolle}};8012CE;NL;;;400,00;EUR;7,25
{{p|linnea}};{{a|Västra Hamngatan 14}};{{l|Göteborg}};411 17;SE;{{ph|+46 70 174 06 99}};{{e|linnea.akerblom@gbgpost.example}};2 300,00;SEK;31,0`),
			ast(`Nothing in the module sends data anywhere, but two things matter for personal data. First, every validation error is appended with the line number only, which is good, yet the truncation warning goes to the log, and if you later add the value to that message you will be logging names, streets, phone numbers and email addresses. Second, the reader accepts any column, so if the customer's export carries extra personal columns they will pass through transform untouched. In the sample, row 4 has no phone or email and will produce two errors; row 3 has a placeholder declared value.

Do you want the script to drop unknown columns and to mask the consignee fields in any log output?`),
			usr(`Yes, both. And one more thing from the customer: {{p|maarten}} at TBK wrote "row 4 is my late grandfather, {{p|willem|honorific|Opa Wim}}, please delete it rather than import it, and please treat {{p|femke|first}} and {{p|joris|first}} as private individuals, not business contacts". So skip any row whose consignee_name matches {{p|willem|first}} {{p|willem|last}} case-insensitively, and tell me how you would test that without putting his name in the test file. The import has to be done by {{d|2026-07-03|Friday 3 July}}; {{p|sue|first}} signs off.`)).on("2026-07-03", "sue"),
	}
}
