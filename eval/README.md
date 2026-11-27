# Outworld eval harness

A black-box harness that answers two questions about an Outworld gateway:

1. **How much real personal data reaches the upstream provider?** Measured per entity type.
2. **Does the rehydrated answer still come back correct?** Measured as restore accuracy, plus task quality against a real model when one is available.

It runs fully offline with a deterministic fake upstream, and optionally against a real OpenAI-compatible model. It is its own Go module (`github.com/davidparkercodes/outworld/eval`) so it never affects the gateway's build.

Every person, company, address, phone number and identifier in the corpus is invented.

## Quick start

```bash
cd eval
task test                 # go vet + unit tests, including an end-to-end run against a bundled fake gateway
task eval                 # build ../cmd/outworld, run the full eval, write results/
task eval -- --limit 20   # any outworld-eval flag after --
task eval -- --configs regex-only,dictionary --api openai,anthropic --out ../.work/eval-scratch
task eval:selftest        # full pipeline against the bundled "oracle" fake gateway (sanity check)
task gencorpus            # regenerate the corpus (byte-identical for the same seed)
```

Without Task: `go run ./cmd/outworld-eval [flags]` from `eval/`.

The exit code is `0` when the gate passes, `1` when any run's leak rate exceeds `--max-leak` or its error rate exceeds `--max-error-rate`, and `2` when the harness itself could not run (gateway does not build, never becomes healthy, and so on). That makes it usable as a CI gate.

## What a run does

1. Loads `corpus/corpus.json` and `corpus/people.csv`.
2. Builds the gateway with `go build ./cmd/outworld` in `--outworld-dir` (default `..`), or uses `--gateway-bin`.
3. For every API in `--api` and every detector config in `--configs`, starts a **fake upstream** (records every request body it receives) and a second, identical fake used as the direct-to-upstream latency baseline.
4. Writes a temporary config and launches `outworld serve --config <file>`, waits for `GET /healthz`:

   ```yaml
   listen: 127.0.0.1:<port>
   vault:
     path: <tmp>/<run>.vault
     key_env: OUTWORLD_VAULT_KEY
   upstreams:
     cloud: {base_url: "http://127.0.0.1:<fakeport>/v1", api_key_env: CLOUD_API_KEY}
   policy:
     default_route: masked
     masked_upstream: cloud
     date_shift: true
     dictionary: <abs path to people.csv>   # dictionary and dictionary+ner only
     entities: {address: mask, date: shift, email: mask, financial: strip, gov_id: strip, location: keep, org: keep, person: mask, phone: mask, secret: strip}
   ner: {url: <--ner-url>, timeout: 120s}   # ner and dictionary+ner only
   memo: {enabled: false}                   # true with --memo
   ```

   For `--api anthropic` the upstream line is `cloud: {base_url: ..., api_key_env: CLOUD_API_KEY, api: anthropic}` and `masked_upstream: cloud` is replaced by `anthropic: {masked_upstream: cloud}`.

   `OUTWORLD_VAULT_KEY` is a fresh random 32-byte hex key per run; `CLOUD_API_KEY` is a dummy (or the real key in real-upstream mode).
5. Sends every corpus item to `POST /v1/chat/completions` (or `POST /v1/messages` for the Anthropic API), once non-streaming and once streaming (`--modes`), then sends the same request straight to the baseline fake.
6. Scores each response against the ground truth and writes `<out>/<timestamp>/report.{md,json}`, `<out>/latest.{md,json}`, and the gateway log for each run. `--out` defaults to `results`; point it elsewhere (for example `../.work/eval-scratch`) for partial runs so `results/latest.*` is not overwritten.

## Run matrix

One invocation produces one report with a column per run.

- `--configs` picks the detector configs, comma-separated:

  | Config | Dictionary | NER sidecar |
  |---|---|---|
  | `regex-only` | no | no |
  | `dictionary` | yes | no |
  | `ner` | no | yes |
  | `dictionary+ner` | yes | yes |

  Default: `dictionary,regex-only`, or `dictionary+ner,ner` when `--ner-url` is given. `ner` and `dictionary+ner` need `--ner-url http://127.0.0.1:8790` (a running `detectors/gliner` sidecar). `--configs regex-only,dictionary,ner,dictionary+ner --ner-url ...` puts all four side by side, so the cost of NER shows against a no-NER baseline. Runs are sequential, one gateway at a time.
- The older `--dictionary with,without` still works when `--configs` is not set (`with` is `dictionary` or `dictionary+ner`, `without` is `regex-only` or `ner`). The old labels `with-dictionary`, `without-dictionary` and `ner-only` are accepted by `--configs` as aliases; reports use the new names.
- `--api openai,anthropic` (default `openai`) picks the wire format. With more than one API every config runs once per API and runs are labelled `<config>/<api>`.
- `--memo` turns on the gateway's cross-request mask memo. It is off by default because each item is sent twice (non-stream, then stream) and the second send would be a cache hit, which flatters the latency numbers. When on, the report says so in the header and next to the latency table.

The Anthropic client lifts the system prompt into the top-level `system` field, sets `max_tokens`, sends `anthropic-version: 2023-06-01` (and `x-api-key` when a key is set), and reassembles `text_delta` events until `message_stop`.

## The fake upstream

`upstream.Fake` implements `/v1/chat/completions` and `/v1/messages` (both streaming and non-streaming) and `/v1/models`. The answer is the same in both formats and deterministic:

```
TIMELINE
1. [name] Marisol Quenby
2. [email] m.quenby@halvorsen-bio.example
3. [date] 2026-04-17
...

ECHO
<every non-system message it received, verbatim, joined by a divider>
```

The timeline lists name-like, email-like and date-like strings in the order they appear. The echo section is the important part: whatever surrogates the gateway sent upstream come straight back, so a perfect gateway turns the echo back into exactly the original text. Streaming responses are cut into small chunks of varying size (1 to 11 bytes) so surrogates are routinely split across SSE events. The Anthropic stream is `message_start`, `content_block_start`, `ping`, one `content_block_delta` per chunk, `content_block_stop`, `message_delta`, `message_stop`.

## Metrics

All counting is case-insensitive and word-boundary aware. Entities are deduplicated per item by `(type, surface text)`, so "Marisol" appearing five times in one thread is one unit.

**Sensitive** types are those whose policy action is `mask`, `shift` or `strip`. **Kept** types are `keep` (org and location by default).

- **Leak rate.** For each sensitive unit, the harness counts how often its surface form appears in everything the upstream received (all string fields of all recorded request bodies) versus how often it appears in the *non-sensitive* text of the input (the input with every sensitive span blanked out). If the upstream saw it more often, it leaked. The baseline subtraction is what keeps names that are also common words ("Will", "Rose", "May", "Grant") honest: the verb in "Will you confirm" is not a leak, the name is. Obvious variants also count:
  - `verbatim` or `case-insensitive` match of the span text,
  - `variant`: alternate renderings stored with the span (the same date as ISO, `January 2, 2006` or `01/02/2006`; the street line of an address; the token inside a password-reset link; the password inside a connection string),
  - `markdown-stripped`: `**Marisol** Quenby` arriving as `Marisol Quenby`,
  - `digits`: phone, SSN and card/account numbers re-punctuated (`4242-4242-4242-4242` for `4242 4242 4242 4242`).

  Reported overall, per entity type, per person reference variant (full, first, nickname, possessive, caps, lower, title+last, last-first, markdown, url-slug, handle), per dictionary coverage, and per API and request mode.
- **Restore accuracy.** For units that were masked or shifted (and did not leak), the share whose original text appears in the echo section of the final answer at least as many times as it appeared in the input. Stripped types are excluded because they are not supposed to come back. Dates that only came back in a different format are counted separately.
- **Unknown surrogates.** Anything the gateway reports itself: response headers or trailers whose name contains `outworld` and `unknown`/`hallucinat`/`unresolved` (for example `X-Outworld-Unknown-Names`, as a count or a comma list), the same keys inside a top-level `outworld` object in the JSON body or SSE events, or the `: outworld unknown_names=a,b` SSE comment. The comment and the trailer carry the same list, so a stream counts the larger of the two.
- **Surrogate residue.** A format-agnostic check for surrogates that were not rehydrated: tokens that the upstream received, that do not exist anywhere in the input, and that still appear in the echo section of the final answer. Tokens that only occur inside square-bracketed placeholders in the upstream text (`[REDACTED]`, `[redacted card]`) are strip placeholders, not surrogates, and are ignored. Tokens that show up in more than 10% of responses are listed as placeholders instead of residue.
- **Over-masking.** For kept types, the share of units the upstream saw fewer times than the input contained them, i.e. the gateway altered an org or location it was told to keep. Quasi-identifiers (free-text descriptions that identify someone without naming them) are reported as "sent intact"; they are not in the leak rate because no entity type covers them.
- **Latency.** p50/p95 of the full request through the gateway, of the same request straight to the baseline fake, and of the per-request difference (overhead), plus the count and share of requests whose overhead exceeded 1 s. Reported per API and mode with the detector configs side by side, then in full per run. Answered requests only. The report header and the latency section state the host's CPU core count, whether NER was in use, and whether the memo was on.
- **Errors.** Non-200 responses, malformed bodies, streams without `[DONE]`, grouped by the `error.code` in the body (for example `outworld_leak_blocked` when the gateway's own leak check refuses to send a request). A gateway that refuses everything has no leaks, so the error gate (`--max-error-rate`, default 2%) is what catches that.

The leak rate is computed over **all** requests, including ones that errored, because what matters is what reached the upstream (a request that fails after forwarding still leaked; a request blocked before forwarding sent nothing). The report also shows the leak rate over answered requests only, and the per-variant, per-dictionary, restore and over-masking breakdowns are over answered requests.

## Real upstream mode

```bash
export OUTWORLD_EVAL_API_KEY=...
go run ./cmd/outworld-eval --real-upstream https://api.example.com/v1 --model <model-id> --limit 30
```

OpenAI format only: `--real-upstream` with `--api anthropic` exits with an error. The gateway's `cloud` upstream points at a local recording proxy in front of the real API, so the leak rate is still measured on what actually left the machine. Restore accuracy is not computed (a real model does not echo). Instead each item with ground-truth facts is sent both directly and through the gateway, the model is asked for a JSON timeline of `{date, person}` events, and both outputs are scored against the ground truth with field-level matching (exact ISO date, person matched on full name or surname). The report shows precision, recall and F1 for both paths. Keep `--limit` small; every item costs two real requests per mode plus two for the quality comparison.

## Corpus

`cmd/gencorpus` generates the corpus deterministically from a fixed seed (`corpus.DefaultSeed`); a test fails if the committed files drift from the generator. There are 10 templates (scheduling, onboarding, security notices, billing disputes, medical reminders, moving, incident reviews, family chat, insurance claims, team newsletters) times 14, plus 12 hand-written hard cases in `corpus/hand.go`.

Each item is a chat request (system task prompt plus one or more user/assistant messages) with ground-truth spans (`msg`, byte `start`/`end`, `type`, `value`, `canonical`, `variant`, `needles`) and, for the quality check, `facts` (`date`, `person`).

Data used:

- **people**: 48 invented people, 18 of them with names that are also English words (Will, Rose, Grant, Hope, May, Mark, Dawn, Summer, Bill, Faith, Art, Joy, Frank, Grace, Sterling, Ivy, Penny, Chase). 30 of the 48 are in `people.csv` (`value,type,aliases`, aliases `|`-separated); the rest are not, so dictionary and no-dictionary recall can be compared.
- **email**: `.example` domains and RFC 2606 names only.
- **phone**: `555-0100` to `555-0199` fictional numbers in four formats.
- **address**: invented streets.
- **date**: ten formats including no-year (`Monday, March 2`, `Mar 2`, `the 2nd of March`), dotted European and email header dates.
- **secret**: random API-key-shaped strings, OTP codes, password-reset links, passwords, a connection string, a JWT-shaped token.
- **gov_id**: SSN-format numbers from never-issued ranges only (area 666, group 00, serial 0000, and the 987-65-432x advertising block).
- **financial**: published processor test card numbers and generated `400000` numbers (all Luhn-valid), checksum-valid routing numbers, account numbers, policy numbers, and the standard documentation IBAN.
- **org** and **location**: invented companies, real city names (kept by policy).

Hard cases covered: names inside email addresses, URLs and chat handles; names split by markdown (`**First** Last`, `First _Last_`, `[Name](mailto:...)`); ALL-CAPS headers and legal text; lowercase casual names; possessives; nicknames; titles with surnames; "Last, First"; the same person referenced a dozen ways in one item; quoted reply chains; multi-turn chats where the assistant turn already contains names; an org whose name contains a person's surname; quasi-identifiers.

## Layout

```
corpus/            corpus.json, people.csv, and the generator (package corpus)
upstream/          fake OpenAI and Anthropic upstream, recorder, recording proxy
score/             leak/restore/over-mask scoring, aggregation, timeline quality matching
internal/gateway/  config writer, build, launch, health check
internal/runner/   request loop, OpenAI and Anthropic clients (streaming and non-streaming)
internal/app/      orchestration and config matrix used by the CLI and the end-to-end tests
internal/report/   Markdown and JSON reports
internal/shim/     passthrough and oracle fake gateways used for self-tests
cmd/gencorpus      corpus generator
cmd/outworld-eval  the harness CLI
cmd/fake-outworld  fake gateway binary (FAKE_OUTWORLD_MODE=passthrough|oracle)
```

The end-to-end tests run the real pipeline against the fake gateway in both API formats: a passthrough gateway must score a 100% leak rate, and an oracle gateway that masks exactly the ground-truth spans must score 0% leaks and 100% restore. Those two bounds are the harness's own correctness check. The fake gateway serves `/v1/chat/completions` and `/v1/messages`, so the Anthropic path is covered without the real gateway.

## Held-out corpus

`corpus/holdout.json` and `corpus/holdout-people.csv` are a second corpus written without sight of the detector code, to measure how well detection generalizes beyond the main corpus the detectors were tuned on. It uses the same schema, so the harness loads it unchanged:

```bash
go run ./cmd/outworld-eval --corpus corpus/holdout.json --people corpus/holdout-people.csv --out ../.work/eval-holdout
go run ./cmd/genholdout                 # regenerate both files (deterministic)
go run ./cmd/genholdout -lint -stats    # also list span counts and sensitive values that recur as plain words
```

**Never tune against it.** Detector changes must not be guided by held-out results: read the headline numbers, not the leak list, when deciding what to fix. Once the held-out corpus has been used to guide a fix it is no longer held out and should be replaced with a newly written one.

What is in it: 85 hand-written items, 85 invented people (none shared with `people.csv`), 39 of them in `holdout-people.csv`. Names are mostly not US census names (West African, South Asian, East Asian including family-name-first, Arabic, Eastern European, Nordic, two-surname Spanish and Portuguese, `van der`/`von`/`de` particles, `O'`/`Mc`/`Mac`, hyphenated, diacritics, mononyms, initials), plus English names that are also words (Rob, Sandy, Bill, Lane, Amber, Jasper, Rich, Skip, Dawn, Miles, Olive, Pat, Drew). Genres: Slack, Teams, WhatsApp, iMessage and IRC exports; voicemail transcriptions; support tickets, chats and CRM notes; recruiting and HR notes; legal letters; meeting transcripts and minutes; clinical notes; git log, blame, code review and CODEOWNERS; fixtures, YAML, `.env`, SQL, CSV, JSON and Java source; shell output and stack traces with home directory paths; travel bookings; invoices, expenses and a bank statement; calendar exports; forwarded email chains; Spanish, German and French messages; multi-turn chats with system prompts.

`cmd/genholdout` holds the texts with inline markup (`{{p|key|variant|surface}}` for a person, `{{e|...}}` email, `{{ph|...}}` phone, `{{a|...}}` address, `{{d|ISO|surface}}` date, `{{o|...}}` org, `{{l|...}}` location, `{{s|variant|value|needle...}}` secret, `{{g|variant|value}}` government ID, `{{f|variant|value}}` financial, `{{q|...}}` quasi-identifier) and computes every offset. A test fails if the committed files drift from the generator, if a span does not slice to its value, if spans overlap, or if an item has no sensitive span without being tagged `negative`.

Annotation decisions:

- Person variants add `last`, `initials`, `initial-last`, `first-initial`, `path` (home directory and file names), `ascii-fold` (diacritics dropped), `misspelled` and `sql-escaped` to the main corpus's set. A possessive span excludes the `'s`.
- Every calendar date with at least a month and a day is a `date` span, including log timestamps and `ls` output. A bare weekday, month or day of month is not annotated.
- **Hard negatives the scorer can measure** are annotated as kept types, so masking them counts as over-masking: tools and products with name-like names (Jenkins, Jackson, Cassandra, Hugo, Loki, Thanos, Jasmine, Alexa, Julia, Ada and so on) and companies containing a surname are `org`; places that are also given names (Florence, Sydney, Austin, Charlotte, Savannah, Regina, Georgia, Nancy) are `location`.
- **Hard negatives the scorer cannot measure** are left unannotated: ordinary words that are also names ("rob us of a weekend", "the feed bill", "skip the slides", "amber light"), months and weekdays, and historical or fictional public figures in general-knowledge questions (Hannibal, Marie Curie, Sherlock Holmes). The schema has no kept person type, and annotating a public figure as `person` would score a correct pass-through as a leak. Where such a word shares an item with a real person of the same name, the leak baseline already separates the two. Items tagged `public-figure` or `common-words` are there to be inspected by hand and to exercise the error and residue checks.
- Identifiers with no entity type (medical record numbers, booking references, employee IDs, the last four digits of a card) are not annotated.
- A sensitive value in a system message always appears in a later turn too, because restore accuracy is counted on the echo, which excludes system messages.

## Second held-out corpus

`corpus/holdout2.json` and `corpus/holdout2-people.csv` are a blind replacement for the first held-out corpus, which stopped being held out once detector work was guided by its misses. It is no longer fully blind either: its per-type totals (not its individual leaks) pointed the later work on secrets, government IDs and bank details, so the third held-out corpus below is the blind one. Same schema, same harness:

```bash
go run ./cmd/outworld-eval --corpus corpus/holdout2.json --people corpus/holdout2-people.csv --out ../.work/eval-holdout2
go run ./cmd/genholdout2                 # regenerate both files (deterministic)
go run ./cmd/genholdout2 -lint -stats    # span counts, plus sensitive values that recur as plain text and text that looks like an unannotated email or date
```

**Blindness rules.** The author worked only inside `eval/` and did not open the gateway packages, the detectors, the project docs, or any earlier corpus (`corpus.json`, `people.csv`, `holdout.json`, `holdout-people.csv`, or the item and people files of `cmd/genholdout`). Only the schema, the scorer and the structure of the first generator were read. The same rules apply to anyone who extends it. Read the headline numbers only: do not open the leak list for this corpus while changing detectors, and replace the corpus once it has been used to guide a fix.

**What is in it.** 120 items and about 2,070 spans, written by hand as inline markup in `cmd/genholdout2` (the first held-out markup plus `{{n|kind|text}}` for lookalikes, and person variants that derive their surface from the people table). Every offset is computed. 160 invented people, 74 of them in `holdout2-people.csv`; a test fails if any of them is in an earlier dictionary. The mix follows how assistants and coding agents are used:

| Share | Category (first tag) | Genres |
|---|---|---|
| 35% | `code` | git log and blame, stack traces and panics, `.env`, Kubernetes and Terraform files, SQL dumps and query results, CSV, TSV, JSON and LDIF exports, access and mail logs, PR descriptions, commit messages, tickets, review comments, fixtures, shell history, HTTP traces |
| 25% | `business` | email threads, meeting notes and minutes, HR plans and letters, legal letters and agreements, CRM notes, invoices, support chats, expense and vendor forms, a postmortem, a press release, a voicemail; some in German, French, Spanish, Portuguese, Dutch and romanized Japanese |
| 20% | `personal` | family and class group chats, school and medical messages, bookings, a visa form, a CV, a tax question, an obituary, a genealogy timeline, a notes-app list of logins |
| 20% | `long-or-multi-turn` | six items over 2,000 words where sensitive values are sparse (an RFC, an application log, a generated source file, a CI log, a changelog, an agent session) and 18 multi-turn conversations, most with tool calls |

Items tagged `long` have more than 2,000 words. Items tagged `negative` (17) have no sensitive span at all. The long logs, the source file and the changelog are built by small deterministic loops in `long_gen.go`, with the sensitive lines placed by hand.

**Values.** Phone numbers use published fictional ranges where a country has one (North American 555-01xx, UK Ofcom drama ranges, Australian, French, German, Swedish and Irish reserved blocks) and zero-padded, unassignable shapes elsewhere. Emails are on `.example` and RFC 2606 domains. Cards are published processor test numbers. IBANs are documentation examples. Government identifiers use never-issued ranges or invalid check digits where the scheme has them. Keys and tokens are made up; the AWS pair is the documentation example.

**Annotation decisions that differ from the first held-out corpus.**

- Tool calls are text. A message is a role and a string, so tool calls and results are written into assistant and user turns in several common transcript styles. Turns alternate and end on a user turn, which both wire formats accept.
- Hard negatives that are not organisations are annotated with type `org` and a variant that starts with `not-entity:` (`hash`, `version`, `number`, `identifier`, `common-word`, `month`, `placeholder`, `placeholder-name`, `pet-name`, `eponym`, `public-figure`, `fictional`, `public-key`, `secret-reference`). `org` is used only because it is a kept type: the scorer measures over-masking for kept types and has no "none" type. They show up in the `org` row of the over-masking table; filter on the variant in `report.json` to separate them from real organisations (`company`, `company-eponym`, `product`, `library`, `tool`).
- Historical and fictional figures are annotated the same way (`not-entity:public-figure`, `not-entity:fictional`), in items that contain no private person. Masking them is reversible and not a privacy failure, but it takes away what the model knows about them, so it is counted as over-masking. Leave those variants out if you disagree.
- A date needs a month and a day in its surface text. A bare weekday or day of month ("Friday", "the 10th") is not annotated even when the thread makes the date clear.
- There are no `facts`, so this corpus does not take part in the real-upstream quality comparison.

## Third held-out corpus

`corpus/holdout3.json` and `corpus/holdout3-people.csv` are a blind replacement for the second held-out corpus. Same schema, same harness:

```bash
go run ./cmd/outworld-eval --corpus corpus/holdout3.json --people corpus/holdout3-people.csv --out ../.work/eval-holdout3
go run ./cmd/genholdout3                 # regenerate both files (deterministic)
go run ./cmd/genholdout3 -lint -stats    # lint warnings, then counts by family, type and variant
go run ./cmd/genholdout3 -mint "es-dni:47382915?"   # fill each ? so an invented identifier passes its checksum
go test ./cmd/genholdout3                # drift, offsets, overlaps, checksums, mix, earlier names, lint
```

**Blindness rules.** The author worked in an isolated copy that held only `eval/README.md`, `corpus/corpus.go`, `score/`, `upstream/` and the machinery of `cmd/genholdout2` (`main.go`, `long_gen.go`, `holdout2_test.go`) without its item or people files. The gateway packages, the detectors, the project docs, every earlier corpus and every earlier item file were out of reach and were not read. The earlier people files were present only so the test could load them; they were never opened by hand, and the test prints nothing but the offending name from this corpus's own table. No gateway was run and no results were seen. The same rules apply to anyone who extends it. Read the headline numbers only: do not open the leak list for this corpus while changing detectors, and replace the corpus once it has been used to guide a fix.

**What is in it.** 131 items, about 43,400 words and 4,208 spans (3,041 on sensitive types, 1,163 on kept types of which 662 are lookalikes, 4 quasi-identifiers), written by hand as inline markup in `cmd/genholdout3`. Every offset is computed. 188 invented people (168 in the people table, 20 one-off relatives and clerks), 89 of them in `holdout3-people.csv` with the aliases a user would write (nicknames, ASCII folds, former names, handles, "Abuela Chayo", "Fujisawa-san"). 102 people appear in three or more items. A test fails if any name is in an earlier people file; nine first drafts were, and were renamed. The first tag of every item is its family:

| Family (first tag) | Items | Words | Spans | Genres |
|---|---|---|---|---|
| `code` | 24 | 3,700 | 569 | git log, blame, reflog and merge conflict, Python and Java stack traces, crash reports with macOS and Windows home paths, compose file, SQL dump, CI log, review thread, TODO comments, shell history, `package.json`, pnpm, Go, pip and Poetry lock data with hashes, CODEOWNERS, nginx log, notebook output, API JSON, Terraform plan, on-call alert thread, GitHub issue, an agent session with tool calls |
| `work` | 18 | 4,000 | 588 | meeting notes with initials, quoted email thread with signatures, Slack and Teams transcripts, HR file note, recruiting debrief, support ticket, CRM export, ICS invite, performance review, offer letter, expense claim, stand-up in initials only, call transcript, supplier form, newsletter, payroll diversion emails, out-of-office chain |
| `personal` | 16 | 4,200 | 493 | family group chat, GP referral, insurance denial, flight booking with passports and a machine-readable line, tax question, card dispute, solicitor's letter, school email, rental application, wedding guest list, intestacy question, rail and hotel confirmations, pharmacy message, visa form, vet summary, flatmate advice |
| `data` | 14 | 2,000 | 429 | CSV, TSV, JSON, YAML, auth log, markdown table, label and value form, OCR scan, all caps notice, unpunctuated dictation, vCard, SOAP XML, fixed-width report, NDJSON |
| `intl` | 27 | 5,300 | 786 | Spanish (Spain, Mexico, Argentina and Chile), French (France, Senegal, Québec), German (Germany, Switzerland), Portuguese (Brazil, Portugal), Italian, Dutch, Polish, Swedish, Turkish, Vietnamese, Indonesian and Tagalog, Hinglish, a Swahili note, and short items in Chinese, Japanese, Korean, Russian, Arabic, Greek, Hebrew and Hindi; onboarding sheets for Africa, Asia, Australia and New Zealand |
| `secrets` | 12 | 1,800 | 236 | `.env` with many vendors, passwords in prose, Kubernetes secret, PEM and age keys next to public keys, one-time and recovery codes, curl commands, a workflow file, JWT debugging, tokens in logs, a password manager export, config files pasted into chat, and one file of lookalikes only |
| `negatives` | 11 | 2,100 | 400 | companies named after people, places and products that are first names, eponyms, public and fictional figures, ordinary words that are names, pet names, part and order numbers with measurements, months and weekdays, arithmetic; one item mixes real colleagues with their namesakes (Paris in Paris, April in April) |
| `long` | 9 | 20,400 | 707 | nine items over 2,000 words: agent session with test output, CI log, lock file, service log, SQL dump, a conversation about an import script, an email chain with a pasted rate table, a tenancy agreement with inventory, a postmortem with its paging log |

Twelve items are tagged `negative` (no sensitive span at all). Eighteen are `multi-turn`, three have a system prompt. 38 items carry `facts` (74 date and person pairs), so this corpus takes part in the real-upstream quality comparison. The bulk of the long logs, the lock file, the SQL rows, the source file, the rate table and the inventory come from small deterministic loops in `long_gen.go`; every line that holds a sensitive value is placed by hand.

Spans by type: person 1,522, date 535, email 259, phone 236, gov_id 155, address 124, secret 118, financial 92, org 850, location 313, quasi 4.

**Values.** Everything is invented. Emails are on `.example` and RFC 2606 domains. Phone numbers use published fictional ranges where a country has one and made-up subscriber numbers elsewhere. Government and bank identifiers cover 50 country codes; wherever the scheme has a check digit the value was constructed to pass it and the test verifies it (`checks.go`: Luhn, IBAN mod 97, Spanish DNI and NIE, Italian codice fiscale, Dutch BSN, Portuguese NIF, PESEL, Swedish, Norwegian, Danish, Finnish and Icelandic numbers, Swiss AHV, French NIR, German tax ID, Czech birth number, Turkish, Greek, Russian, Irish and UK NHS numbers, US routing numbers, Canadian SIN, CURP, RFC, CLABE, CPF, RUT, CUIL, CBU, Aadhaar, Chinese, Japanese, Korean, Singaporean, Hong Kong, Taiwanese, Thai, Israeli, Emirati and South African IDs, Australian TFN and Medicare, New Zealand IRD). Formats without a public checksum are format-only. US Social Security numbers use never-issued ranges, UK National Insurance numbers the `QQ` prefix, German ID cards the specimen numbers. Cards are published processor test numbers or constructed Luhn-valid numbers. Keys and tokens follow vendor shapes with made-up bodies.

**Markup added to the second generator.** `{{x|reason|text}}` marks plain text that was reviewed and deliberately left without a span; it produces no span, the lint skips it and `-stats` counts the reasons. `[[lb]]`, `[[rb]]` and `[[pipe]]` inside a field stand for `{{`, `}}` and `|`. `.on("2026-02-05", "key", ...)` on an item adds facts. Government and financial variants are `<country>-<kind>` (`es-dni`, `gb-iban`), and their canonical form and needle drop spaces, dots, dashes and slashes.

**Lint.** `-lint` warns when a sensitive value, or any name or name part of a person in the people table, occurs as plain text; when text looks like an unannotated email, date (numeric, or a day with a month name in seven languages), phone number or IBAN; and when a run of seven or more digits has no span. It reports 0 warnings, and the test fails if that changes. Eighteen places are marked as reviewed plain text: a wrong name a customer complains about ("Stephen", 3), ordinary words next to a colleague of that name (4), a company register number, two employer tax IDs, a medical record number, a doctor's registration number, a railcard number, a username equal to its password, a fraction, and three strings in an arithmetic question that only look like phone numbers and a date.

**Annotation decisions that differ from the second held-out corpus.**

- Every street address is `address`, home or business, because a detector cannot tell them apart; a company's own address counts as sensitive here. In records with separate fields the street is `address`, the city is `location`, and a bare postcode has no span. A multi-line address is one span whose needle is the street line.
- Every email address is `email`, including role mailboxes. Placeholders (`test@example.com`) and an SSH remote (`git@host`) are lookalikes instead.
- A date needs a month and a day, in any script or calendar (`令和8年4月1日`, `13 เม.ย. 2518`, `02NOV26`, `20260609T140000`, spoken "the tenth of march"). An ISO timestamp is one span up to its zone, because the scorer needs a word boundary. Epoch seconds, bare years, months, weekdays and "the 16th" are not dates.
- A title is outside the span (`Dr {Prys}`, variant `title-last`); a kinship word or honorific that is fused with the name in speech is inside it (`{Opa Wim}`, `{Cô Hương}`, `{Fujisawa-san}`, variant `honorific`). A possessive span excludes the `'s`.
- User names, handles, home directory names and branch name parts that are built from a person's name are `person`. Initials are `person` wherever they stand for a person, including line labels.
- `secret` covers credentials of every kind, including password hashes, one-time codes, door and pickup codes, security answers, card verification codes and whole URLs that embed a credential (the needle is the credential). Public keys, fingerprints, checksums, key ids and references such as `${{ secrets.X }}` are lookalikes.
- `financial` is card, account, routing and policy numbers and payment aliases. Amounts, card expiry dates, last four digits, bank names and BIC codes are not annotated.
- Not annotated, as before: IP addresses, booking references, ticket and case numbers that are not marked as lookalikes, company tax and register numbers, medical record numbers.
- Hard negatives use `org` with a `not-entity:` variant, as in the second corpus. New kinds: `checksum`, `uuid`, `measurement`, `money`, `weekday`, `order-number`, `part-number`, `template-reference`. Real organisations are `company`, `company-eponym` or `product`; places that are also names are `location` with `place-name`, `place-eponym`, `country` or `room-name`. Filter on the variant in `report.json` to separate them.
- A value is never a secret in one place and a lookalike in another within the same item, since no consistent gateway could score well on both.
- Three items put a person next to a kept namesake on purpose (Chad the colleague and Chad the country, Paris in Paris, Clay and clay). Both sides are annotated; the leak baseline counts the kept occurrences, so only a gateway that tells them apart scores cleanly on both.

**New person variants.** Added to the earlier set (`full`, `first`, `last`, `nickname`, `possessive`, `caps`, `lower`, `lower-first`, `title-last`, `last-first`, `markdown`, `url-slug`, `handle`, `initials`, `initial-last`, `first-initial`, `path`, `ascii-fold`, `misspelled`, `sql-escaped`):

| Variant | Meaning |
|---|---|
| `lower-last`, `caps-first`, `caps-last` | one part of the name in lower or upper case |
| `caps-reversed`, `caps-truncated` | `SURNAME GIVEN` without a comma, and the same cut off by a fixed-width field |
| `family-first`, `western-order` | the other order for a name whose usual order is family name first, or the reverse |
| `native-script` | the name in Cyrillic, Arabic, Chinese, Japanese (kanji or kana), Korean, Greek, Hebrew or Devanagari |
| `patronymic` | given name and patronymic as a form of address (`Тимур Ринатович`) |
| `inflected` | a case-inflected form (`Agnieszko`, `Bartosza Wiśniewskiego`) |
| `maiden` | birth surname |
| `alias` | another legal or former rendering (`Susan` for Sue, a married name, a transliteration) |
| `name-part` | one piece of a compound name (`Kaur`, `bin Rashid`, the first of two surnames) |
| `initial-first` | initial before the given name (`S. Rajeshwari`) |
| `middle` | full name with a middle initial |
| `honorific` | name fused with a kinship word or honorific |
| `teknonym` | `Abu Khalid` style |
| `display` | chat display name, sometimes with an emoji |
| `booking` | airline reservation form (`KOBAYAKAWA/MINORI MS`) |
| `mrz` | the name inside a passport machine-readable line |
| `ocr` | a name with scanner errors |

Phone variants `spoken` (digits as words), `ocr` and `url-encoded` exist for the same reason.

**Known limits.** The lint only knows the people table and a handful of shapes, so a one-off name that is mentioned once without markup would go unnoticed; the capitalised words and digit runs outside spans were read through by hand once. People count (188) is above the planned 150 because one-off relatives were added late. Organisations are annotated where they matter as hard cases, not exhaustively.
