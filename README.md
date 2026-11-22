# Outworld

**Your data stays in the nest. You decide what leaves, it leaves in disguise, and the answer comes back translated.**

Outworld is a privacy gateway that speaks both the OpenAI and the Anthropic Messages API. Point a client at it, and it lets you use cloud LLMs on personal data (email, notes, contacts, records, the files a coding agent reads) without the provider seeing the real personal details. Names, emails, phones and addresses are swapped for realistic stand-ins, dates are shifted, secrets are removed, and the model's answer is translated back before it reaches you.

It works with any OpenAI-compatible provider, with the Anthropic API (including as `ANTHROPIC_BASE_URL` for Claude Code), and with local servers (Ollama, llama.cpp, vLLM) for the requests you never want to leave your machine.

## Why "Outworld"

In ant-keeping, the *outworld* is the foraging arena attached to the nest. The colony stays safe inside; only what you choose goes out. Outworld is that arena for your data: the nest is your machine and your vault, the outworld is the cloud model.

## How it works

```
  your app / agent / script
            |
            |  POST /v1/chat/completions or /v1/messages
            v
  +-----------------------------------------------------------+
  |  OUTWORLD                                                 |
  |                                                           |
  |  1. route     rules + X-Outworld-Route header             |
  |               masked | local | passthrough | block        |
  |                                                           |
  |  2. detect    regex (email, phone, address, ids, cards,   |
  |               keys, OTP codes, magic links, dates)        |
  |               + built-in name detector (no model needed)  |
  |               + your dictionary (contacts, aliases)       |
  |               + optional NER sidecar                      |
  |                                                           |
  |  3. transform per YOUR policy, per entity type:           |
  |               keep | mask | shift | generalize | strip    |
  |                                                           |
  |  4. leak check  re-scan the outbound payload; known real  |
  |               values and detector hits are masked in      |
  |               place and re-scanned; anything that still   |
  |               cannot be transformed = request blocked     |
  |               (fail closed)                               |
  |                                                           |
  |      +-------------------+                                |
  |      | encrypted vault   |  real <-> surrogate, stable    |
  |      | (AES-256-GCM)     |  across requests and corpora   |
  |      +-------------------+                                |
  |                                                           |
  |  6. rehydrate  surrogates and shifted dates translated    |
  |               back, including streamed SSE chunks and     |
  |               tool-call arguments                         |
  +-----------------------------------------------------------+
            |                                   ^
            | masked request                    | 5. response (surrogates)
            v                                   |
       cloud model  ----------------------------+
       ("Tamsin Wetherby emailed you on Dec 25")
```

Example. You send:

> Draft a reply to John Smith (john.smith@gmail.com) confirming our call on March 15, 2024. My verification code is 482913.

The provider sees:

> Draft a reply to Kasimir Sandoval (kasimir.sandoval@example.org) confirming our call on December 25, 2024. My verification code is [redacted code].

You get back a draft addressed to John Smith about March 15, 2024. The verification code was never sent and is not restored.

## Quickstart

```bash
go install github.com/davidparkercodes/outworld/cmd/outworld@latest

export OUTWORLD_VAULT_KEY=$(outworld keygen)   # store this somewhere safe; it unlocks your vault
export CLOUD_API_KEY=...                       # key for your OpenAI-compatible provider

cp outworld.example.yaml outworld.yaml         # set upstreams.cloud.base_url
cp people.example.csv people.csv               # optional: your own people / orgs / places

outworld serve --config outworld.yaml
```

Then point your client at `http://127.0.0.1:8787/v1`:

```bash
curl -s http://127.0.0.1:8787/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"your-model","messages":[{"role":"user","content":"Summarize: John Smith called from (804) 555-1234."}]}' -i
```

Check the `X-Outworld-Route` response header to see what happened.

### Claude Code and other Anthropic clients

Add an Anthropic upstream and point the client at Outworld:

```yaml
upstreams:
  claude:
    api: anthropic
    base_url: https://api.anthropic.com/v1
    api_key_env: ANTHROPIC_API_KEY     # or forward_auth: true to pass the client's own login through
policy:
  anthropic:
    masked_upstream: claude
```

```bash
export ANTHROPIC_BASE_URL=http://127.0.0.1:8787
claude
```

See [docs/integrations/claude-code.md](docs/integrations/claude-code.md) for subscription logins, gateway tokens and what to expect. Guides for [LiteLLM](docs/integrations/litellm.md) and [Open WebUI](docs/integrations/open-webui.md) are next to it.

### Docker

```bash
docker build -t outworld:local .
export OUTWORLD_VAULT_KEY=$(docker run --rm outworld:local keygen)
cp compose.example.yml compose.yml
docker compose up -d
```

The image is distroless, static and runs as a non-root user. The config directory is mounted read-only, so point the vault at the data volume in `outworld.yaml`: `vault: {path: /data/outworld.vault}`.

### CLI: mask and unmask

For testing and batch pipelines, `mask` and `unmask` read stdin and write stdout using the same vault and policy:

```bash
outworld mask < notes.txt > notes.masked.txt          # exit code 3 if the leak check blocks
# ... send notes.masked.txt anywhere ...
outworld unmask < answer.txt > answer.real.txt
outworld mask --scope project-a < notes.txt           # per-scope date offset
```

## Policy is yours

Outworld does not decide what is sensitive. You do. Your employer may be public on your profile; your kid's school is not. Each entity type gets an action:

| action       | effect                                                                 | restored? |
|--------------|------------------------------------------------------------------------|-----------|
| `keep`       | sent as-is                                                             | n/a       |
| `mask`       | replaced with a realistic, consistent surrogate from the vault         | yes       |
| `shift`      | dates moved by a secret per-scope offset (days), format preserved      | yes       |
| `generalize` | replaced with a bucket, e.g. `Norfolk, Virginia` to `a city in Virginia` | no      |
| `strip`      | replaced with a neutral token like `[redacted code]`, never sent       | no        |

Entity types: `person`, `email`, `phone`, `address`, `date`, `org`, `location`, `url`, `ip`, `secret`, `gov_id`, `financial`.

Defaults: people, emails, phones, addresses and IPs are masked; dates shifted; orgs, locations and plain URLs kept; secrets, government IDs and financial numbers stripped. Secret-bearing links (password resets, magic links, URLs with tokens) are classified as `secret` and stripped. Secrets also cover passwords in prose (`my temporary password was ...`), `KEY=value` / `export API_KEY=...` assignments, and long hex or base64 tokens next to words like `credential` or `token`. Any SSN-shaped number next to `SSN`, `social security` or `tax id` is a `gov_id` in any separator form (`123-45-6789`, `123 45 6789`, `123456789`), regardless of whether that range was ever issued. Other pattern coverage: phone numbers in national formats with a leading `0` or `+` (`07911 123456`, `030 12345 678`, `01 23 45 67 89`) and spoken as digit words (`four one five five five five ...`); identifiers after a label such as passport, driver's license, EIN, tax ID, national ID, DNI/NIE/NIF, NINO, A-number, SIN, Medicare, MRN, policy number or member ID, plus unlabelled Spanish DNI/NIE and UK NINO shapes; values after `pass`, `secret`, `token` or `api key` followed by `:`, `=` or `is` when they do not look like a plain word or a code expression; bare 32, 40 and 64 character hex strings unless the line is git or checksum output (`commit ...`, `sha256: ...`, a hash that starts a line followed by more text) or labels it as an id; and street addresses in French, Spanish, Portuguese, Italian, German, Dutch and UK forms with their postal code and city, plus all-caps US mailing labels.

### Surrogates

- Realistic, never `<PERSON_1>`: `John Smith` becomes `Kasimir Sandoval`, `John` alone becomes `Kasimir`, `JOHN SMITH` becomes `KASIMIR SANDOVAL`, `john.smith` in a URL becomes `kasimir.sandoval`, `John's` keeps its `'s`.
- Deterministic per vault (HMAC of a vault secret), consistent across requests and a whole corpus, collision-free, and never equal to any real value the vault or your dictionary knows.
- Shape-preserving: emails stay valid on reserved `example.com/.org/.net` domains, phones use the fictional `555-01xx` range with the original formatting, addresses keep number length, street suffix, unit and state (addresses outside the US style keep their street word, layout and postal code shape), phones in national formats keep the leading `0` or country code and the grouping, spoken numbers stay digit words, IPs use documentation ranges.
- Locale-aware addresses: an address that reads as French, Spanish, Portuguese, Italian, German (including Austrian and Swiss forms), Dutch or UK gets a stand-in from the same country, with street names in that language's grammar, a real mid-sized town and a postal code that fits it. `12 rue de Rivoli, 75001 Paris` becomes something like `63 rue de Montreuil, 47000 Agen`, and `Lindenstraße 8, 50674 Köln` something like `Bergstraße 5, 35060 Marburg`. US-style addresses are generated exactly as before.
- Name lists are bundled and generated for this project; once exhausted, a syllable generator takes over.
- Gender-consistent: first names are replaced with first names of the same likely gender (a bundled list of common first names; unknown names get gender-neutral surrogates), so `Priya ... her` stays coherent. A dictionary `gender` column overrides the guess.
- One identity per person: every alias of a dictionary entry maps onto parts of the same surrogate. `Marcus Delaney` becomes `Ezra Sandoval`, `Delaney` becomes `Sandoval`, and the nickname `Marc` becomes a nickname derived from `Ezra` (for example `Ez`), so the model sees one person.
- A surrogate is never a word that appears capitalized anywhere in the same request, so a stand-in cannot collide with a real name you are sending.

### Detection layers

Detection is layered, and each layer can only add findings:

1. **Patterns.** Emails, phones, addresses, IDs, cards, keys, codes, links and dates. No model, microseconds.
2. **Built-in name detector.** Finds people without a model: a known first name followed by a surname (`John Smith`, `JOHN SMITH`), a known first name on its own, names after labels (`Emergency contact: May Lindenbrook`, `ONCALL: ...`, `From: ... <email>`), after relationship words (`my sister Ottoline`), as chat speakers (`[09:14] ravi:`), and in profile URLs (`/people/abelard-ashdown`). It also handles particles (`Anna van der Berg`), surname-first forms (`SMITH, John`), initials (`J. Smith`, `John S.`), accents (`José` and `Jose` are the same name) and names inside `first.last@` addresses. Family-name-first names are recognized for common Vietnamese, Chinese and Korean family names (`Nguyen Van An`, `Zhang Wei`, `Kim Min-jun`); family names that are also ordinary words (`Le`, `Do`, `He`, `Park`, `Sun`, `Kim`, `Lee`) need a distinctive given name, a middle marker such as `Van` or `Thi`, or name context, so `Ma Long` on its own is still missed. Code handles count as people where the position makes them a username: `TODO(mwilliams)` and `FIXME(kchen)`, the user prefix of a branch (`git checkout -b jsmith/retry-logic`, `origin/kchen/fix`), `@mwilliams` mentions, `/home/jdoe/` and `/Users/jdoe/` path segments, and git trailers (`Reviewed-by: kchen`). Branch prefixes and mentions need a name-table match (an initial plus a known surname, or a known first name), so `feature/login`, `@types/node` and `@param` are left alone, and service accounts such as `root`, `ubuntu` and `runner` are skipped; a handle with no recognizable name in it (`@zorgblat`) is missed outside home directories and trailers. A bundled list of a few hundred well-known organizations whose names look like a person (`Morgan Stanley`, `Goldman Sachs`, `Johnson & Johnson`, `Ralph Lauren`) is treated as `org`, and so is a name followed by a corporate word (`Okafor Partners`, `Maria Garcia Holdings`, `Becker & Sons`). A real person who shares one of those names (`Edward Jones`, `Raymond James`) is only masked from your dictionary, after a title (`Mr. Edward Jones`), after a relationship word, in a sign-off, or after a label such as `Contact:`. It uses public-domain name tables (about 9,400 first names and 32,800 surnames from US government data, Wikidata and per-country popular-name lists, see `internal/namedata`). Coverage is best for names written in Latin script and common in Europe and the Americas; it will still miss many others, and nicknames in particular. Names that are also ordinary words (`Will`, `May`, `Grant`, `Faith`) only count in name context. Common technology names (`Claude`, `Ruby`, `Julia`, `Jenkins`) are never treated as people. Turn the layer off with `policy.name_heuristics: false`.
3. **Your dictionary.** The people, orgs and places you know. The most precise layer.
4. **Names learned within a request.** Once any layer finds `Hope Valdane` anywhere in a request, every message in that request is re-checked for her other forms: `hope` in a chat line, `HOPE VALDANE`, `hope-valdane`, `hvaldane`.
5. **NER sidecar (optional).** A model for names, orgs and places nothing else caught. Two are bundled: [GLiNER](detectors/gliner/README.md) and [OpenAI Privacy Filter](detectors/privacy-filter/README.md). ALL CAPS runs are sent to the sidecar in title case, because NER models miss shouted names.

False positives are handled with `policy.allow`, a list of terms that must never be masked:

```yaml
policy:
  allow: ["Grace Hopper", "Lincoln Center"]
```

### Dictionary

A CSV (or JSON) of people, orgs and places you know, for example exported from your contacts or mail graph. Columns: `value,type,aliases[,generalize[,gender]]`, aliases separated by `|`. With a header row the columns can come in any order; `gender` is optional (`f`, `m` or `n`).

```csv
value,type,aliases,generalize,gender
Jordan Example,person,Jordy|J. Example,,f
Northwind Clinic,org,,a medical clinic,
```

For each person, Outworld also matches:

- first names in any case (`Jordan`, `jordan`), last names when capitalized, possessives, and the full name in any case;
- URL slugs and handles: `jordan.example`, `jordan_example`, `jordan-example`, `jordanexample`, `jexample`, `jordane`, `jordan_e`;
- single words that are also common English words (`Will`, `May`, `Rose`, `Chase`) only in name context: capitalized and not used as the word (`Will you`, `in May`, `Grant access`), or lowercase right after a greeting (`hey will`), before a verb like `said`, as a chat speaker (`will:`), or alone on a signature line. Inside handles, slugs and emails they always match.

JSON form: `[{"value": "...", "type": "person", "aliases": ["..."], "generalize": "...", "gender": "f"}]`.

## Routing

Each request takes exactly one route:

- `masked`: transform, leak-check, send to `policy.masked_upstream`, rehydrate the answer.
- `local`: send raw to `policy.local_upstream` (Ollama, llama.cpp, vLLM on your hardware).
- `passthrough`: send raw to a named upstream. Intended for confidential-compute / TEE endpoints you trust with raw data.
- `block`: refuse with HTTP 403.

Order of evaluation: any `block` rule that matches wins; then the `X-Outworld-Route` request header (if `allow_route_header`, default true), e.g. `local` or `passthrough:tee`; then the first matching rule; then `default_route`. Rules match by whole-word keywords (`match_any`), regexes (`match_regex`) or detected entity types (`entities`).

Response headers:

| header | meaning |
|---|---|
| `X-Outworld-Route` | `masked`, `local`, `passthrough`, `block`, or `blocked-leak` |
| `X-Outworld-Route-Reason` | `default`, `header`, or `rule:<name or index>` |
| `X-Outworld-Masked` | number of values transformed |
| `X-Outworld-Unknown-Names` | surrogate-looking names in the answer the vault does not know (the model probably invented a person). On streams it arrives as an HTTP trailer and an SSE comment `: outworld unknown_names=...` |

## Endpoints

OpenAI format (upstreams with `api: openai`, the default):

- `POST /v1/chat/completions` streaming and non-streaming, tools and tool calls (arguments are masked and rehydrated as JSON).
- `POST /v1/completions` legacy completions.

Anthropic format (upstreams with `api: anthropic`, selected by `policy.anthropic.*`):

- `POST /v1/messages` streaming and non-streaming. `system`, text blocks, `tool_use` inputs and `tool_result` content are masked; text and tool inputs in the answer are rehydrated, including across `text_delta` and `input_json_delta` events.
- `POST /v1/messages/count_tokens` masked the same way, so counts match what is sent.
- Every `anthropic-*` request header, the query string and unknown body fields are forwarded unchanged; upstream status, headers and error bodies come back unchanged.

Both:

- `GET /v1/models` passthrough to the default upstream.
- `GET /healthz`.

There is no translation between the two formats: a request goes to an upstream that speaks the format it arrived in.

Set `OUTWORLD_TOKEN` (or `auth.token_env`) to require a token from clients, sent as `Authorization: Bearer <token>`, `x-api-key: <token>` or `X-Outworld-Token: <token>`. When set, requests without it are rejected. The gateway token is never forwarded upstream. Each upstream uses its own `api_key_env`, unless it sets `forward_auth: true`, in which case the client's own `Authorization` and `x-api-key` headers are passed through (for clients that hold the provider login themselves).

Messages containing images, audio or files are refused on the `masked` route (they cannot be masked) unless you set `policy.allow_images: true`. Use `local` or `passthrough` for those.

### Stable transcripts

Agents and chat clients resend the whole conversation on every request. Providers cache on that prefix, and current Claude models reject a conversation whose earlier turns changed. So Outworld keeps a transcript memo: every text it has masked, and every model answer it has translated back, maps to the exact bytes the provider saw. When the client sends that text again, the provider receives the same bytes as before, and the text is not detected again, which is also what keeps long agent sessions fast.

The memo lives in memory (`memo.max_bytes`, default 64 MiB, least recently used entries dropped first) and holds masked text only. After a restart the first request of an ongoing conversation is masked from scratch, which can cost one prompt-cache miss. A consequence worth knowing: text that was already sent is not re-masked when the vault later learns a new name. It left in that form once; changing it later would not unsend it.

Provider-signed `thinking` blocks pass through untouched in both directions, so visible thinking shows stand-in names.

## Configuration reference

```yaml
listen: 127.0.0.1:8787            # env OUTWORLD_LISTEN overrides
vault:
  path: ./outworld.vault          # relative paths resolve against the config file
  key_env: OUTWORLD_VAULT_KEY     # 32 bytes as 64 hex chars or base64
  key_file: ./outworld.key        # alternative to key_env
upstreams:
  <name>:
    base_url: https://api.example.com/v1
    api_key_env: CLOUD_API_KEY    # optional
    model: some-model             # optional: override the request's model
    headers: {X-Extra: value}     # optional
    header_timeout: 180s          # optional
    api: openai                   # openai | anthropic
    forward_auth: false           # pass the client's Authorization / x-api-key upstream
policy:
  default_route: masked           # masked | local | passthrough | block
  masked_upstream: cloud
  local_upstream: local
  passthrough_upstream: tee
  anthropic:                      # upstreams for /v1/messages, all with api: anthropic
    masked_upstream: claude
    local_upstream: ""
    passthrough_upstream: ""
  keep_metadata: false            # false drops metadata.user_id on masked Anthropic requests
  name_heuristics: true           # built-in name detector
  allow: []                       # terms that are never masked
  date_shift: true
  date_shift_scope: global        # one secret offset per scope
  date_order: mdy                 # mdy | dmy for ambiguous numeric dates
  shift_min_days: 30
  shift_max_days: 400
  dictionary: ./people.csv
  allow_route_header: true
  allow_scope_header: false       # let clients pick a date scope via X-Outworld-Scope
  allow_images: false
  entities: {person: mask, email: mask, phone: mask, address: mask, date: shift,
             org: keep, location: keep, url: keep, ip: mask,
             secret: strip, gov_id: strip, financial: strip}
  rules:
    - {name: health, match_any: ["diagnosis", "lawsuit"], route: local}
    - {match_regex: ["(?i)attorney[- ]client"], route: passthrough, upstream: tee}
    - {entities: ["financial"], route: block}
ner:                              # optional sidecar, see detectors/gliner
  url: http://127.0.0.1:8790
  timeout: 10s
  threshold: 0.5
  optional: false                 # false = sidecar errors fail the request closed
  labels: {person: person, organization: org, location: location, address: address}
auth:
  token_env: OUTWORLD_TOKEN
log:
  level: info                     # debug | info | warn | error
  format: json                    # json | text
memo:
  enabled: true                   # byte-stable transcripts across requests
  max_bytes: 67108864
max_body_bytes: 33554432
```

Unknown keys are rejected so typos do not silently weaken your policy.

## How well it works: measured

The numbers below come from the eval harness in [`eval/`](eval/README.md). It sends synthetic prompts with labelled sensitive values through the gateway to a fake upstream that records exactly what a provider would have received. **Leak rate** is the share of labelled sensitive values that reached the upstream unchanged. **Over-masking** is the share of things that should have been left alone (companies, cities, product names, version strings, hashes) that were changed anyway. **Blocked** is the share of requests the fail-closed leak check refused with a 422.

There are three corpora, and the difference between them matters more than any single number:

- **Tuned**: 152 items. The detectors were developed against it. Its numbers show what the detectors do on forms they were built for, and nothing more.
- **Development**: 85 items written without access to the detector code. After the first run on it, general gaps it exposed were fixed, so it is no longer blind.
- **Blind**: 120 items, 160 invented people, about 33,000 words, written without access to the detector code. No detector was changed after looking at its results. The later additions (code handles, the company list, family-name-first names) came from hand-written probes in an outside review, not from this corpus. This is the number to believe.

Leak rate through the OpenAI API (the Anthropic API, streaming, and repeat runs with a fresh vault all land within about two points of these):

| Detectors | Tuned | Development | Blind |
|---|---|---|---|
| Patterns and built-in names only | 0.3% | 13.5% | 16.1% |
| Plus your dictionary (about half the people listed) | under 0.1% | 9.5% | 11.9% |
| Plus GLiNER sidecar, no dictionary | 0.1% | 4.3% | 11.0% |
| Dictionary and GLiNER | 0% | 3.5% | 9.3% |
| Plus Privacy Filter sidecar, no dictionary | 0.1% | 4.2% | 9.7% |
| Dictionary and Privacy Filter | 0.1% | 3.5% | 8.6% |

On the blind corpus, by kind of value (dictionary and GLiNER): names 5%, emails 0%, dates 7%, phones 9%, addresses 11%, secrets 32%, bank and card details 35%, government IDs 54%. The names that get through are mostly nicknames, first names that are not in the tables, and handles. The identifiers that get through are mostly national ID formats from countries the patterns do not cover and passwords that appear without a label.

The cost side, on the blind corpus:

| Detectors | Over-masking | Blocked | Added latency, median |
|---|---|---|---|
| Patterns and built-in names only | 21% | 8% | 30 to 55 ms |
| Plus your dictionary | 22% | 8% | 30 to 55 ms |
| Dictionary and GLiNER (8 CPU threads) | 26% | 3% | 2.4 s |
| Dictionary and Privacy Filter (8 CPU threads) | 29% | 3% | 0.5 s |

On the tuned corpus, over-masking without a sidecar is 0% and nothing is blocked. The blind corpus is deliberately full of things that look sensitive and are not (companies named after people, cities that are also first names, commit hashes, version strings), which is why its over-masking is high. Over-masking is reversible: the answer is still translated back, but the model loses what it knew about the real company or place.

What to take from this:

- Outworld removes most of what identifies people in ordinary prompts. It does not remove all of it, and on text it has never seen it misses roughly one sensitive value in ten even with a dictionary and a model.
- A dictionary of the people you actually talk about is the cheapest large improvement. A sidecar is the next one.
- Restore accuracy (answers translated back correctly) is about 99.5% or higher in every configuration.
- These are synthetic prompts against a fake upstream. Sidecar latencies were measured on a shared, busy CPU host with no GPU and vary a lot with prompt length; the 95th percentile on long agent transcripts is tens of seconds for GLiNER.

Reproduce any cell with `go run ./cmd/outworld-eval --corpus corpus/holdout2.json --people corpus/holdout2-people.csv --configs regex-only,dictionary` from `eval/`. If you change a detector after reading the blind corpus results, it stops being blind; write a new one.

## Threat model and honest limits

What Outworld is for: reducing how much identifying personal data a cloud model provider receives, while keeping answers useful, with a hard stop when it notices it is about to leak something it knows about.

What it protects against:

- The provider (and its logs, retention, training pipelines, breaches) seeing the names, contact details, addresses, exact dates, account numbers, IDs and secrets that Outworld detects.
- Accidental leaks of values the vault already knows: every outbound payload is re-scanned for known real values and for unmasked detector hits. Anything found is masked in place with its existing surrogate (or a new one) and the payload is re-scanned; a known first name followed by a capitalized word is treated as a full name. Only what still cannot be transformed (for example a value in a JSON key or message `name`) blocks the request.
- Mentions split across messages: every text in a request is detected first, so a name that is only recognizable in a later message is also masked in earlier ones.
- Credentials and one-time codes that Outworld recognizes reaching any cloud model at all (stripped, never restored). Recognition is far from complete here: on the blind corpus about a third of secrets got through, mostly passwords and tokens with no label next to them. Do not rely on Outworld as a secret scanner.

What it does **not** do:

- **Detection is best-effort.** Regexes, your dictionary and an optional NER model will miss things: unusual names, nicknames you did not list, people described rather than named ("my sister's husband"), names in other scripts, typos. The leak check only blocks what some layer recognizes. Add a dictionary; add the NER sidecar; review what you send.
- **Masking is not anonymization.** Quasi-identifiers survive: a rare job title plus a small town plus a hospital stay can identify someone even with every name replaced. So can writing style, the structure of your life, and facts you chose to `keep`. Date shifting hides exact dates but keeps intervals (which is the point), and intervals can be identifying too.
- **The provider still sees the content.** Your medical question is still a medical question about *someone*. If the substance itself is what must not leave, route it `local` or to a `passthrough` confidential-compute endpoint, or `block` it. That is what routing rules are for.
- **Model output is untrusted.** Rehydration translates every surrogate and every date in the answer, including dates the model made up. Surrogate phone numbers are recognized in any formatting the model chooses (dashes, dots, spaces, parentheses, `+1`, digits only). Invented people that look like surrogates are flagged in `X-Outworld-Unknown-Names`, not removed.
- **The vault is the crown jewels.** It maps every surrogate back to a real value. It is encrypted at rest (AES-256-GCM with an authenticated, append-only record log), and Outworld refuses to run without a key. Protect the key like a password-manager master key. Anyone with both the vault and the key can reverse everything.
- **Not covered:** images, audio and files (refused on the masked route by default), lone initials such as `LH` (left as-is), handles and usernames outside the recognized code positions (a bare `kchen` in prose is only masked once the same request has shown it in one of them), people who share a name with a listed company unless your dictionary has them, month-year only dates like "March 2024" (not shifted), metadata outside the message text (Outworld drops the `user` field and Anthropic `metadata.user_id` but does not rewrite `tools` schemas or system-level metadata your client adds), provider-signed thinking blocks and block types Outworld does not know (passed through unchanged), and anything an agent sends around the gateway on its own.
- Outworld logs never include real values or raw prompts: only routes, counts per entity type and timings.

When in doubt: use `local` for anything whose substance is sensitive, `masked` for everyday personal data where names and contact details are the risk, and `passthrough` to a TEE endpoint when you need a big model on raw data and trust that attestation.

## Using it as a library

```go
v, _ := vault.OpenFile("my.vault", key)
dict, _ := detect.LoadDictionary("people.csv")
eng, _ := outworld.New(outworld.Options{Vault: v, Policy: outworld.DefaultPolicy(), Dictionary: dict})

s := eng.NewSession("global")
masked, err := s.Mask(ctx, text)       // err = detection failure
if err := s.Err(); err != nil { ... }   // leak check (errors.Is(err, outworld.ErrLeak))
answer := s.Rehydrate(modelOutput)
r := s.NewStreamRehydrator(false)       // Push(chunk) / Flush() for streams
```

Packages: `pkg/detect` (detectors, dictionary, NER client, overlap resolution), `pkg/vault` (vault + `Backend` interface + encrypted file backend), `pkg/surrogate` (generators and name lists), `pkg/dateshift`, `pkg/entity`, `pkg/outworld` (the engine).

### Vault backends

`vault.Backend` is a small interface: `Load()` returns the vault secret and existing records; `Update(fn)` must give `fn` exclusive access, pass it records other writers added since the last call, and durably persist what `fn` returns. The bundled file backend implements this with `flock` and an append-only encrypted log, so several Outworld processes can share one vault file. Other stores can plug in by implementing the same interface.

## Optional NER sidecars

Outworld works without a model. A sidecar adds recall for names, organizations and places that the patterns, the built-in name detector and your dictionary miss, at the cost of latency. Two small FastAPI services are bundled and speak the same `/detect` contract, so either goes under `ner.url`:

- [detectors/gliner](detectors/gliner/README.md): GLiNER, multilingual, covers people, organizations, locations and addresses.
- [detectors/privacy-filter](detectors/privacy-filter/README.md): OpenAI Privacy Filter, an open-weight PII token classifier. It has no organization or location labels.

Which one: in the gateway eval the two leak about the same amount. Privacy Filter is several times faster on CPU and better at first names, titles and nicknames, but because it has no organization label it tends to tag company names as people, so it over-masks more. GLiNER is slower and knows what an organization is. Privacy Filter can also return `account_number` and `secret` spans; map them in `ner.labels` (`account_number: financial`, `secret: secret`) to use them. That mapping is not part of the measurements above.

## License

Apache-2.0. See [LICENSE](LICENSE).
