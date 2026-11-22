package main

var itemsCodeData = []def{
	code("sql-dump", "sql,export,sql-escaped", usr(`The restore fails on row 6. Find the syntax error and give me the fixed statement only.

-- Dump of table members, {{d|2026-03-01|2026-03-01 03:00:02}} UTC
INSERT INTO members (id, full_name, email, phone, dob, street, city, national_id, joined) VALUES
(1041, '{{p|deirdre|sql-escaped|Deirdre O''Callaghan}}', '{{e|deirdre.ocallaghan@example.org}}', '{{ph|+353 20 915 5501}}', '{{d|1979-12-03}}', '{{a|14 Sráid an Chaisleáin}}', '{{l|Galway}}', '{{g|pps|1234567TW}}', '{{d|2019-05-14}}'),
(1042, '{{p|tavita|sql-escaped|Tavita Leota-Fa''asavalu}}', '{{e|tavita.leota@example.com}}', '{{ph|+685 72 55014}}', '{{d|1991-07-04}}', '{{a|Lot 14 Vaitele Street}}', '{{l|Apia}}', NULL, '{{d|2020-01-22}}'),
(1043, '{{p|khalid}}', '{{e|k.altamimi@example.net}}', '{{ph|+971 50 555 0142}}', '{{d|1975-03-19}}', '{{a|Villa 22, Street 14B, Al Safa 1}}', '{{l|Dubai}}', '{{g|emirates-id|784-1975-0000000-1}}', '{{d|2020-09-01}}'),
(1044, '{{p|sasha}}', '{{e|o.tkachenko@example.org}}', '{{ph|+380 44 555 01 42}}', '{{d|1994-10-28}}', '{{a|вул. Садова 14, кв. 7}}', '{{l|Kyiv}}', '{{g|tax-id|0000000000}}', '{{d|2021-03-30}}'),
(1045, '{{p|jiwoo|western-order|Ji-woo Seo}}', '{{e|jiwoo.seo@example.com}}', '{{ph|+82 10-5550-0142}}', '{{d|1998-02-11}}', '{{a|14 Teheran-ro 88-gil, Gangnam-gu}}', '{{l|Seoul}}', '{{g|rrn|980211-0000000}}', '{{d|2022-06-06}}'),
(1046, '{{p|annelize}}', '{{e|annelize@vandermerwe.example}}', '{{ph|+27 82 555 0142}}', '{{d|1983-09-09}}', '{{a|14 Dorp Street}}', '{{l|Stellenbosch}}', '{{g|sa-id|8309090000000}}', '{{d|2022-11-18}}',
(1047, '{{p|meenakshi}}', '{{e|s.meenakshi@example.net}}', '{{ph|+91 00000 55014}}', '{{d|1969-04-14}}', '{{a|14, 3rd Cross Street, Mylapore}}', '{{l|Chennai}}', '{{g|aadhaar|1234 5678 9012}}', '{{d|2023-02-02}}'),
(1048, '{{p|zarghona}}', '{{e|zarghona@example.org}}', '{{ph|+93 70 000 0142}}', '{{d|2000-01-01}}', NULL, '{{l|Herat}}', NULL, '{{d|2024-07-19}}');`)),

	code("psql-result", "sql,financial", usr(`Finance says two of these payouts went to the wrong account. Write a query that finds payouts where the IBAN country does not match the payee country.

billing=> select id, payee, country, iban, amount, paid_on from payouts where paid_on >= '{{d|2026-02-01}}' order by id;
  id  |             payee              | country |               iban                | amount  |  paid_on
------+--------------------------------+---------+-----------------------------------+---------+------------
 7701 | {{p|maarten}}          | NL      | {{f|iban|NL91 ABNA 0417 1643 00}}            | 1840.00 | {{d|2026-02-03}}
 7702 | {{p|elodie}}        | FR      | {{f|iban|FR14 2004 1010 0505 0001 3M02 606}} |  912.50 | {{d|2026-02-03}}
 7703 | {{p|hannelore}}             | AT      | {{f|iban|DE89 3704 0044 0532 0130 00}}       | 2210.00 | {{d|2026-02-10}}
 7704 | {{p|inaki}}    | ES      | {{f|iban|ES91 2100 0418 4502 0005 1332}}     |  640.75 | {{d|2026-02-10}}
 7705 | {{p|angharad}}              | GB      | {{f|iban|GB82 WEST 1234 5698 7654 32}}       | 1175.00 | {{d|2026-02-17}}
 7706 | {{p|soren}}                | DK      | {{f|iban|GB82 WEST 1234 5698 7654 32}}       |  388.20 | {{d|2026-02-17}}
 7707 | {{o|Kjærgaard Møbler ApS|company-eponym}}           | DK      | {{f|iban|DK50 0040 0440 1162 43}}            | 9120.00 | {{d|2026-02-24}}
(7 rows)

billing=> select name, sort_code, account_no from uk_accounts where name ilike '%llywelyn%';
       name        | sort_code | account_no
-------------------+-----------+------------
 {{p|angharad|initial-last|A Llywelyn}}        | {{f|sort-code|20-00-00}}  | {{f|account|55779911}}
(1 row)`)),

	code("csv-export", "csv,export,phones", usr(`Dedupe this CRM export. Same person can appear with a different spelling or name order. Output the merged CSV and a list of the merges you made.

contact_id,name,email,phone,city,country,created,owner
C-1001,"{{p|siobhan}}",{{e|siobhan.nibhraonain@example.org}},{{ph|+353 20 915 5587}},{{l|Cork}},IE,{{d|2025-03-14|14/03/2025}},{{p|daisy|handle|dmukherjee}}
C-1002,"{{p|siobhan|ascii-fold|Siobhan Ni Bhraonain}}",{{e|shiv@example.org}},{{ph|020 915 5587}},{{l|Cork}},IE,{{d|2025-06-02|02/06/2025}},{{p|daisy|handle|dmukherjee}}
C-1003,"{{p|thuy}}",{{e|thuy.nguyen@harborline.example}},{{ph|+84 28 5550 0142}},{{l|Ho Chi Minh City}},VN,{{d|2025-06-09|09/06/2025}},{{p|cliff|handle|cbarrowman}}
C-1004,"{{p|thuy|last-first|NGUYEN, Thi Thu Thuy}}",{{e|thuy.nguyen@harborline.example}},{{ph|028 5550 0142}},{{l|Saigon}},VN,{{d|2025-07-21|21/07/2025}},{{p|cliff|handle|cbarrowman}}
C-1005,"{{p|haruto|western-order|Haruto Kobayashi}}",{{e|h.kobayashi@example.com}},{{ph|+81 3-0000-0142}},{{l|Tokyo}},JP,{{d|2025-08-01|01/08/2025}},{{p|daisy|handle|dmukherjee}}
C-1006,"{{p|haruto|caps-last-first|KOBAYASHI Haruto}}",{{e|haruto@kobayashi.example}},{{ph|03-0000-0142}},{{l|Tokyo}},JP,{{d|2025-08-19|19/08/2025}},{{p|cliff|handle|cbarrowman}}
C-1007,"{{p|joao}}",{{e|joaopedro.santos@example.com}},{{ph|+55 11 90000-0142}},{{l|São Paulo}},BR,{{d|2025-09-30|30/09/2025}},{{p|daisy|handle|dmukherjee}}
C-1008,"{{p|joao|short|JP dos Santos}}",{{e|jp@example.com}},{{ph|(11) 90000-0142}},{{l|Sao Paulo}},BR,{{d|2025-10-11|11/10/2025}},{{p|cliff|handle|cbarrowman}}
C-1009,"{{p|iris}}",{{e|iris.papadakis@example.net}},{{ph|+30 21 0555 0142}},{{l|Athens}},GR,{{d|2025-11-05|05/11/2025}},{{p|daisy|handle|dmukherjee}}
C-1010,"{{p|malou|last-first|Dimaculangan, Maria Lourdes}}",{{e|malou.dimaculangan@example.com}},{{ph|+63 917 555 0142}},{{l|Quezon City}},PH,{{d|2025-12-12|12/12/2025}},{{p|cliff|handle|cbarrowman}}
C-1011,"{{p|malou|nickname-last|Malou Dimaculangan}}",{{e|malou.dimaculangan@example.com}},{{ph|0917 555 0142}},{{l|QC}},PH,{{d|2026-01-08|08/01/2026}},{{p|cliff|handle|cbarrowman}}
C-1012,"{{o|Iris Optical Ltd|company-eponym}}",{{e|accounts@iris-optical.example}},{{ph|+44 20 7946 0144}},{{l|London}},GB,{{d|2026-01-20|20/01/2026}},{{p|daisy|handle|dmukherjee}}

Owners are {{p|daisy}} and {{p|cliff}}; ask {{p|cliff|first}} about anything created before he took over the APAC book.`)),

	code("json-api", "json,gov-id", usr(`Generate a JSON Schema for this response. Mark which fields are PII in a "x-pii": true extension.

GET /v2/employees?page=1
{
  "data": [
    {
      "id": "emp_00417",
      "legal_name": {"given": "{{p|jun|first}}", "family": "{{p|jun|last}}", "suffix": "Jr."},
      "preferred_name": "{{p|jun|nickname}}",
      "work_email": "{{e|rodrigo.delapaz@brightquay.example}}",
      "personal_email": "{{e|junjun.delapaz@example.com}}",
      "mobile": "{{ph|+1 (671) 555-0142}}",
      "date_of_birth": "{{d|1984-06-30}}",
      "hire_date": "{{d|2017-09-05}}",
      "ssn": "{{g|ssn|000-12-3456}}",
      "passport": {"country": "PH", "number": "{{g|passport|P0000014A}}", "expires": "{{d|2029-11-02}}"},
      "home_address": {"line1": "{{a|217 Chalan San Antonio, Unit 4}}", "city": "{{l|Tamuning}}", "postal_code": "96913"},
      "bank": {"routing": "{{f|routing|110000000}}", "account": "{{f|account|000123456789}}"},
      "manager": "emp_00102"
    },
    {
      "id": "emp_00102",
      "legal_name": {"given": "{{p|bobby|first}}", "family": "{{p|bobby|last}}", "suffix": "III"},
      "preferred_name": "{{p|bobby|nickname}}",
      "work_email": "{{e|rtannenbaum@brightquay.example}}",
      "personal_email": null,
      "mobile": "{{ph|212-555-0196}}",
      "date_of_birth": "{{d|1961-01-17}}",
      "hire_date": "{{d|1999-04-12}}",
      "ssn": "{{g|ssn|666-45-6789}}",
      "passport": null,
      "home_address": {"line1": "{{a|400 West 119th Street, Apt 9C}}", "city": "{{l|New York}}", "postal_code": "10027"},
      "bank": {"routing": "{{f|routing|110000000}}", "account": "{{f|account|000987654321}}"},
      "manager": null
    }
  ],
  "next": "/v2/employees?page=2&cursor={{n|identifier|eyJvZmZzZXQiOjJ9}}"
}`)),

	code("access-log", "logs,http", usr(`Write an awk one-liner that counts 5xx per endpoint from logs like these, and tell me whether anything in here should have been scrubbed before I pasted it.

203.0.113.77 - - [{{d|2026-03-08|08/Mar/2026}}:23:58:01 +0000] "GET /healthz HTTP/1.1" 200 2 "-" "kube-probe/1.29"
198.51.100.12 - {{p|hunter|handle|hashby}} [{{d|2026-03-08|08/Mar/2026}}:23:58:04 +0000] "GET /admin/users?q={{p|mason|url-slug|mason+dubois}} HTTP/1.1" 200 1843 "-" "Mozilla/5.0"
198.51.100.12 - {{p|hunter|handle|hashby}} [{{d|2026-03-08|08/Mar/2026}}:23:58:09 +0000] "GET /admin/users/8812?email={{e|mason.dubois@example.com}} HTTP/1.1" 200 911 "-" "Mozilla/5.0"
192.0.2.201 - - [{{d|2026-03-08|08/Mar/2026}}:23:59:40 +0000] "POST /api/login?user={{p|robin|handle|rachterberg}}&pass={{s|password|Tulp3nveld!94}} HTTP/1.1" 302 0 "-" "curl/8.5.0"
192.0.2.201 - - [{{d|2026-03-09|09/Mar/2026}}:00:00:02 +0000] "GET /api/export.csv?api_key={{s|api-key|ak_4c3d2e1f0a9b8c7d6e5f4a3b2c1d0c9d}} HTTP/1.1" 500 133 "-" "curl/8.5.0"
192.0.2.201 - - [{{d|2026-03-09|09/Mar/2026}}:00:00:03 +0000] "GET /api/export.csv?api_key={{s|api-key|ak_4c3d2e1f0a9b8c7d6e5f4a3b2c1d0c9d}} HTTP/1.1" 500 133 "-" "curl/8.5.0"
198.51.100.90 - - [{{d|2026-03-09|09/Mar/2026}}:00:01:17 +0000] "GET /u/{{p|wade|url-slug|wade-harcourt}}/avatar.png HTTP/1.1" 304 0 "https://forum.example.org/t/mason-jar-lids/4412" "Mozilla/5.0"
198.51.100.90 - - [{{d|2026-03-09|09/Mar/2026}}:00:01:18 +0000] "GET /static/app.{{n|hash|8f14e45f}}.js HTTP/1.1" 200 48211 "-" "Mozilla/5.0"`)),

	code("payroll-tsv", "export,financial,gov-id", usr(`Convert this tab-separated payroll extract to one JSON object per line. Keep IDs as strings.

emp_no	name	country	national_id	bank_ref	net_pay	pay_date
20417	{{p|christian}}	SE	{{g|personnummer|19850230-0004}}	{{f|iban|SE45 5000 0000 0583 9825 7466}}	41 200,00	{{d|2026-02-25|2026-02-25}}
20418	{{p|april}}	US	{{g|ssn|900-55-0142}}	{{f|routing|110000000}}/{{f|account|000111222333}}	5,814.22	{{d|2026-02-27|02/27/2026}}
20419	{{p|pearl}}	GB	{{g|nino|QQ 12 34 56 C}}	{{f|sort-code|20-00-00}} {{f|account|55779911}}	3,402.10	{{d|2026-02-27|27/02/2026}}
20420	{{p|jade}}	CA	{{g|sin|000 000 018}}	{{f|account|00011-003-1234567}}	4,977.03	{{d|2026-02-27|2026-02-27}}
20421	{{p|rafa}}	BR	{{g|cpf|111.444.777-00}}	{{f|account|0001 / 00012345-6}}	9.310,40	{{d|2026-02-27|27/02/2026}}
20422	{{p|akane}}	JP	{{g|my-number|0000 0000 0142}}	{{f|account|0001-014-0000142}}	412,000	{{d|2026-02-25|2026年2月25日}}
20423	{{p|wiremu}}	NZ	{{g|ird|000-000-142}}	{{f|account|00-0000-0000142-00}}	4,105.55	{{d|2026-02-26|26 Feb 2026}}
20424	{{p|priya}}	IN	{{g|pan|ABCDE1234F}}	{{f|account|000014200001}}	1,84,500.00	{{d|2026-02-28|28-02-2026}}

Payroll contact: {{p|christian|nickname}} ({{e|payroll@lindgren-ochoa.example}}).`)),

	code("schema-only", "negative,sql,identifiers", usr(`Review this migration for locking problems on a 200M row table. No data attached, just the DDL.

ALTER TABLE customers
  ADD COLUMN {{n|identifier|first_name}} text,
  ADD COLUMN {{n|identifier|last_name}} text,
  ADD COLUMN {{n|identifier|date_of_birth}} date,
  ADD COLUMN {{n|identifier|ssn_last4}} char(4),
  ADD COLUMN {{n|identifier|phone_e164}} varchar(16),
  ADD COLUMN {{n|identifier|bill_to_address_id}} bigint REFERENCES addresses(id),
  ADD COLUMN {{n|identifier|grace_period_days}} smallint NOT NULL DEFAULT 14;

CREATE INDEX CONCURRENTLY {{n|identifier|idx_customers_last_first}} ON customers ({{n|identifier|last_name}}, {{n|identifier|first_name}});
CREATE TYPE {{n|identifier|card_brand}} AS ENUM ('visa', 'mastercard', 'amex', 'jcb');
COMMENT ON COLUMN customers.{{n|identifier|ssn_last4}} IS 'last four only, never the full number';

We run {{o|Postgres|product}} {{n|version|15.6}} behind {{o|PgBouncer|product}}, migrations through {{o|Flyway|tool}} ({{n|version|V2026.03.2}}__customer_pii_columns.sql), and the app is on {{o|Hibernate|library}} {{n|version|6.4.4}}.`)),
}
