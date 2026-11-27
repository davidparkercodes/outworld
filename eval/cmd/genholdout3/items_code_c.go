package main

var itemsCodeC = []def{
	code("dockerfile-requirements", "manifest,hashes",
		usr(`pip refuses to install with "hashes do not match". Here are the Dockerfile and the requirements file. Which line is wrong? Keep every hash exactly as it is.

# Dockerfile
FROM python:{{n|version|3.12.3}}-slim@sha256:{{n|checksum|2be8daddbb82756f7d1f2c7ece706aadcb284bf6ab6d769ea695cc3ed6016743}}
LABEL org.opencontainers.image.authors="{{p|lerato}} <{{e|lerato.fakude@lumenwharf.example}}>"
LABEL org.opencontainers.image.revision="{{n|hash|8c1f4e2a9b7d6c5e4f3a2b1c0d9e8f7a6b5c4d3e}}"
ARG PIP_INDEX_URL={{s|url-credentials|https://ci-reader:py`+`pi-AgEIcHlwaS5vcmcCJDk4ZjFiM2E3LTVjMmQtNGU2Zi04YTFi@pypi.lumenwharf.example/simple|py`+`pi-AgEIcHlwaS5vcmcCJDk4ZjFiM2E3LTVjMmQtNGU2Zi04YTFi}}
RUN pip install --require-hashes -r requirements.txt
# {{p|lerato|nickname}}: do not bump gunicorn until {{p|svenerik|first}} confirms the worker class change ({{d|2026-05-20|20 May}})

# requirements.txt, compiled by {{p|lerato|handle|lfakude}} with pip-compile {{n|version|7.4.1}}
django=={{n|version|5.0.6}} \
    --hash=sha256:{{n|checksum|8363ac062bb4ef7c3f12d078f6fa5d154031d129a15170a1066412af49d30905}}
gunicorn=={{n|version|22.0.0}} \
    --hash=sha256:{{n|checksum|350679f91b24062c86e386e198a15438d53a7a8207235a78ba1b53df4c4378d9}}
psycopg[binary]=={{n|version|3.1.19}} \
    --hash=sha256:{{n|checksum|dca5e5521c859f6606686432ae1c94e8766d29cc91f2ee595378c510cc5b0731}}
phonenumbers=={{n|version|8.13.37}} \
    --hash=sha256:{{n|checksum|4ea00ef5012422c08c7955c21131e7ae5baa9a3ef52cf2d561e963f023006b80}}`)),

	code("crash-report-paths", "paths,windows,macos",
		usr(`Same crash on two laptops, mine and {{p|farzaneh|possessive}}'s. Can you spot the common frame? Hers is the Windows one.

--- macOS ---
Process:               berth-cli [41822]
Path:                  /Users/{{p|cormac|path|cormacfitzgibbon}}/go/bin/berth-cli
Identifier:            berth-cli
Version:               {{n|version|0.19.4}} ({{n|hash|a91c3f7}})
Date/Time:             {{d|2026-01-27|2026-01-27 14:03:55.118 +0000}}
OS Version:            macOS {{n|version|14.3}} (23D56)
User:                  {{p|cormac}} (uid 501)
Crashed Thread:        0  Dispatch queue: com.apple.main-thread
Exception Type:        EXC_BAD_ACCESS (SIGSEGV)
0   berth-cli   0x0000000100a3f2c4 runtime.sigpanic + 480
1   berth-cli   0x0000000100b81a10 github.com/lumenwharf/berth-cli/config.Load + 212
2   berth-cli   0x0000000100b7ff3c main.main + 96
config file: /Users/{{p|cormac|path|cormacfitzgibbon}}/Library/Application Support/berth/config.toml
keychain item: "berth-cli ({{e|cormac.fitzgibbon@lumenwharf.example}})"

--- Windows ---
Faulting application name: berth-cli.exe, version: {{n|version|0.19.4.0}}
Faulting module path: C:\Users\{{p|farzaneh|path|Farzaneh.Moghaddam-Roth}}\AppData\Local\Programs\berth\berth-cli.exe
Report Id: {{n|uuid|d2b7e5c1-3f4a-4b6c-8d9e-0a1b2c3d4e5f}}
User: LUMENWHARF\{{p|farzaneh|handle|fmoghaddam}}
panic: runtime error: invalid memory address or nil pointer dereference
goroutine 1 [running]:
github.com/lumenwharf/berth-cli/config.Load({0xc00001e0c0, 0x52})
	C:/Users/{{p|farzaneh|path|Farzaneh.Moghaddam-Roth}}/src/berth-cli/config/load.go:58 +0x1d4
main.main()
	C:/Users/{{p|farzaneh|path|Farzaneh.Moghaddam-Roth}}/src/berth-cli/main.go:31 +0x60
config file: C:\Users\{{p|farzaneh|path|Farzaneh.Moghaddam-Roth}}\AppData\Roaming\berth\config.toml (OneDrive - {{o|Lumenwharf}}\{{p|farzaneh|nickname}} backup)`)),

	code("reflog-conflict", "git,branch-names",
		usr(`I think I lost {{p|nilufar|possessive}}'s commits during a rebase. Reflog and the conflict are below. How do I get her work back without redoing mine?

$ git reflog -8
{{n|hash|f41c2aa}} HEAD@{0}: rebase (continue): zones: add UZ postcodes
{{n|hash|93be7d0}} HEAD@{1}: rebase (pick): rates: carrier fallback hook
{{n|hash|5a0e9c3}} HEAD@{2}: rebase (start): checkout origin/main
{{n|hash|d7719b4}} HEAD@{3}: pull origin {{p|nilufar|handle|nbakhtiyorova}}/uz-postcodes: Fast-forward
{{n|hash|2c6f8e1}} HEAD@{4}: commit: wip: {{p|nilufar|nickname}} pairing session, tests red
{{n|hash|b83d5f7}} HEAD@{5}: checkout: moving from main to {{p|dawit|lower-first}}/carrier-fallback
{{n|hash|5a0e9c3}} HEAD@{6}: pull: Fast-forward
{{n|hash|e19a4d6}} HEAD@{7}: clone: from git.lumenwharf.example:lumenwharf/berth-api.git

$ git status
interactive rebase in progress; onto {{n|hash|5a0e9c3}}
Unmerged paths:
	both modified:   internal/rates/zone_uz.go

<<<<<<< HEAD
// Author: {{p|dawit}} <{{e|dawit.g@lumenwharf.example}}>
var uzPrefixes = []string{"100", "140", "200"}
=======
// Author: {{p|nilufar|ascii-fold|Nilufar Bakhtiyorova}} ({{p|nilufar|native-script|Нилуфар Бахтиёрова}}) <{{e|nilufar@lumenwharf.example}}>
// Source: postcode list from {{o|O'zbekiston Pochtasi}}, checked {{d|2026-04-28|28.04.2026}}
var uzPrefixes = []string{"100", "110", "111", "140", "200", "210"}
>>>>>>> {{n|hash|d7719b4}} (zones: add UZ postcodes)

$ git stash list
stash@{0}: WIP on {{p|dawit|lower-first}}/carrier-fallback: {{n|hash|b83d5f7}} before pairing with {{p|nilufar|lower-first}}`)),

	code("github-issue", "issue,customer-report",
		usr(`Draft a reply to this issue. Be kind, he is a paying customer and clearly frustrated. Do not promise a date.

Issue #2291: Label PDFs time out for large manifests
opened {{d|2026-03-05|Mar 5, 2026}} by {{p|fernando|handle|fer-iturralde}}

**Describe the bug**
Since last week every manifest over ~400 lines fails with a 504. We are {{o|Transportes Illimani}} in {{l|La Paz}}, about 30 trucks. My dispatcher {{p|xiadani}} ({{p|xiadani|nickname}}) has been printing labels in batches of 50 by hand, which costs her two hours every morning.

**To reproduce**
1. Log in as {{e|fernando.iturralde@illimani.example}} (you have my permission to impersonate, ticket {{n|identifier|SUP-55102}})
2. Open manifest {{n|identifier|MAN-2026-03-0417}}
3. Print all

**Expected** a PDF. **Actual** a spinner and then "Gateway timeout".

**Additional context**
I already spoke to **{{p|leilani|markdown|Leilani}} {{p|leilani|last}}** from your support team on {{d|2026-03-03|the 3rd of March}}, she said engineering knew. Then nothing. If it helps you can call me, {{p|fernando}}, on {{ph|+591 2 555 0138}} between 9 and 17 {{l|La Paz}} time, or WhatsApp {{ph|+591 7 155 50138}}. Our account number is {{n|identifier|CUST-004417}}.

---
{{p|leilani|handle|lei-k}} commented {{d|2026-03-05|Mar 5, 2026}}
Confirming I spoke with Mr. {{p|fernando|title-last|Iturralde}}. Linked to BERTH-2291. @{{p|kwabena|handle|kobby}} is on it.

{{p|fernando|handle|fer-iturralde}} commented {{d|2026-03-08|Mar 8, 2026}}
Any news? {{p|xiadani|first}} is still printing by hand.`)),

	code("api-json-debug", "json,api",
		usr(`The mobile app crashes parsing this response. Which field violates the schema I pasted yesterday (ids are strings, dates are ISO 8601, phone is E.164 or null)?

HTTP/1.1 200 OK
x-request-id: {{n|uuid|6fa459ea-ee8a-3ca4-894e-db77e160355e}}
etag: W/"{{n|hash|5e1f0c9d2a}}"

{
  "data": [
    {
      "id": "usr_01HV7Q",
      "name": {"given": "{{p|valentina|first}}", "family": "{{p|valentina|last}}", "display": "{{p|valentina|nickname}} {{p|valentina|name-part|Sepúlveda}}"},
      "email": "{{e|vsepulveda@corvanefreight.example}}",
      "phone": "{{ph|+56 2 2555 0171}}",
      "created": "{{d|2025-08-14|2025-08-14T12:01:33Z}}",
      "manager": {"id": "usr_01HV2A", "name": "{{p|ingrid}}"}
    },
    {
      "id": 1047,
      "name": {"given": "{{p|baraka|first}}", "family": "{{p|baraka|last}}", "display": "{{p|baraka}}"},
      "email": "{{e|baraka.sankale@corvanefreight.example}}",
      "phone": "{{ph|0722 555 014}}",
      "created": "{{d|2025-09-02|02/09/2025}}",
      "manager": {"id": "usr_01HV2A", "name": "{{p|ingrid|initial-last|I. Aasgaard-Thorsen}}"}
    },
    {
      "id": "usr_01HV9Z",
      "name": {"given": "{{p|wulan|first}}", "family": null, "display": "{{p|wulan|nickname}}"},
      "email": "{{e|sulistyowati@corvanefreight.example}}",
      "phone": null,
      "created": "{{d|2025-10-27|2025-10-27T00:00:00+07:00}}",
      "manager": null
    }
  ],
  "next_cursor": "{{n|identifier|eyJvZmZzZXQiOjN9}}"
}`)),

	code("oncall-alert", "alert,chat",
		usr(`Pasting the alert thread so you can help me write the incident summary. Keep it to five lines.

[#berth-alerts] {{d|2026-06-13|Sat 13 Jun 2026}}
02:11 PagerBot: FIRING [sev2] berth-api p99 latency > 5s (eu-central). Runbook: https://wiki.lumenwharf.example/runbooks/latency
02:11 PagerBot: Paging primary on-call: {{p|ibrahim}} ({{ph|+20 10 5550 0143}})
02:16 PagerBot: No ack after 5 min. Paging secondary: {{p|thandiwe}} ({{ph|+27 82 555 0126}})
02:17 {{p|thandiwe|handle|thandi-mp}}: ack. looking.
02:24 {{p|thandiwe|handle|thandi-mp}}: db cpu pegged. one tenant, tenant_id={{n|identifier|t_corvane}}. somebody is exporting everything.
02:26 {{p|ibrahim|handle|ielsayed}}: sorry, phone was on silent. here now. it's {{p|chad|lower-first}} at corvane, he runs the quarterly export every june, same as last year
02:27 {{p|thandiwe|handle|thandi-mp}}: {{l|chad|country}} the country or {{p|chad|lower-first}} the person
02:27 {{p|ibrahim|handle|ielsayed}}: person. {{p|chad}}, their finance lead. {{e|chad.wexford@corvanefreight.example}}
02:31 {{p|thandiwe|handle|thandi-mp}}: throttled his session to 2 concurrent queries. latency recovering.
02:40 PagerBot: RESOLVED [sev2] berth-api p99 latency > 5s (eu-central). Duration 29m.
02:41 {{p|ibrahim|handle|ielsayed}}: i'll email {{p|chad|title-last|Wexford}} in the morning and ask {{p|jordan|first}} (their admin) to schedule these off-peak. thx {{p|thandiwe|nickname|thandi}}
02:42 {{p|thandiwe|handle|thandi-mp}}: np. going back to bed. postmortem owner = you, due {{d|2026-06-17|Wed 17 June}}`)),

	code("pyproject-poetry", "manifest,lockfile",
		usr(`Poetry says the lock file is out of date but I did not touch dependencies. Compare the two and explain. Versions and hashes must stay byte for byte.

# pyproject.toml
[tool.poetry]
name = "berth-rates"
version = "{{n|version|2.7.1}}"
description = "Rate tables and zone lookups"
authors = ["{{p|bartosz}} <{{e|bartosz.wisniewski@lumenwharf.example}}>", "{{p|teodora}} <{{e|teodora.vaduva@lumenwharf.example}}>"]
maintainers = ["{{p|wenjing}} <{{e|hwenjing@lumenwharf.example}}>"]
license = "Proprietary"

[tool.poetry.dependencies]
python = "{{n|version|^3.12}}"
pydantic = "{{n|version|^2.7.1}}"
httpx = "{{n|version|0.27.0}}"
babel = "{{n|version|2.15.0}}"

[[tool.poetry.source]]
name = "internal"
url = "https://pypi.lumenwharf.example/simple"

# poetry.lock (excerpt)
[[package]]
name = "babel"
version = "{{n|version|2.14.0}}"
files = [
    {file = "Babel-2.14.0-py3-none-any.whl", hash = "sha256:{{n|checksum|efb1a25b7118e67ce3a259bed20545c29cb68be8ad2c784c83689981b7a57287}}"},
]

[[package]]
name = "pydantic"
version = "{{n|version|2.7.1}}"
files = [
    {file = "pydantic-2.7.1-py3-none-any.whl", hash = "sha256:{{n|checksum|e029badca45266732a9a79898a15ae2e8b14840b1eabbb25844be28f0b33f3d5}}"},
]

[metadata]
lock-version = "{{n|version|2.0}}"
python-versions = "{{n|version|^3.12}}"
content-hash = "{{n|checksum|7d3c0c1e5b9a4f2e8d6c7b1a0f9e8d7c6b5a4f3e2d1c0b9a8f7e6d5c4b3a2f1e}}"`)),

	code("terraform-plan", "infra,plan-output",
		usr(`Is this plan safe to apply on a Friday? I only meant to rename the bucket tag.

Terraform will perform the following actions:

  # aws_iam_user.deployer will be updated in-place
  ~ resource "aws_iam_user" "deployer" {
        id   = "berth-deployer"
      ~ tags = {
          ~ "owner" = "{{p|svenerik|lower|sven-erik bergqvist}}" -> "{{p|lerato|lower}}"
            "team"  = "infra"
        }
    }

  # aws_iam_access_key.deployer must be replaced
-/+ resource "aws_iam_access_key" "deployer" {
      ~ id     = "{{s|aws-access-key-id|AK`+`IAT4QZ7WLM2XN5RPVB}}" -> (known after apply)
      ~ secret = (sensitive value)
        user   = "berth-deployer"
    }

  # aws_db_instance.berth will be updated in-place
  ~ resource "aws_db_instance" "berth" {
        identifier     = "berth-prod"
      ~ engine_version = "{{n|version|16.1}}" -> "{{n|version|16.2}}"
      ~ password       = (sensitive value)
        username       = "berth_admin"
    }

  # aws_route53_record.status will be created
  + resource "aws_route53_record" "status" {
      + name    = "status.lumenwharf.example"
      + records = ["198.51.100.140"]
      + ttl     = {{n|number|300}}
      + type    = "A"
    }

Plan: 2 to add, 2 to change, 1 to destroy.

Changes to Outputs:
  ~ deployer_secret = "{{s|aws-secret-access-key|q9Vx2LmT7pRz4KcW8bYn3HsJd6AfG1uEo5QiNvXt}}" -> (known after apply)
  ~ last_applied_by = "{{p|svenerik|handle|sebergqvist}}@{{d|2026-04-10|2026-04-10}}" -> "{{p|lerato|handle|lfakude}}@{{d|2026-05-08|2026-05-08}}"`)),
}
