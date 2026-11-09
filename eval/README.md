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
task eval:selftest        # full pipeline against the bundled "oracle" fake gateway (sanity check)
task gencorpus            # regenerate the corpus (byte-identical for the same seed)
```

Without Task: `go run ./cmd/outworld-eval [flags]` from `eval/`.

The exit code is `0` when the gate passes, `1` when any run's leak rate exceeds `--max-leak` or its error rate exceeds `--max-error-rate`, and `2` when the harness itself could not run (gateway does not build, never becomes healthy, and so on). That makes it usable as a CI gate.

## What a run does

1. Loads `corpus/corpus.json` and `corpus/people.csv`.
2. Builds the gateway with `go build ./cmd/outworld` in `--outworld-dir` (default `..`), or uses `--gateway-bin`.
3. Starts a **fake upstream** (records every request body it receives) and a second, identical fake used as the direct-to-upstream latency baseline.
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
     dictionary: <abs path to people.csv>   # omitted in the without-dictionary run
     entities: {address: mask, date: shift, email: mask, financial: strip, gov_id: strip, location: keep, org: keep, person: mask, phone: mask, secret: strip}
   ```

   `OUTWORLD_VAULT_KEY` is a fresh random 32-byte hex key per run; `CLOUD_API_KEY` is a dummy (or the real key in real-upstream mode).
5. Sends every corpus item to `POST /v1/chat/completions`, once non-streaming and once streaming (`--modes`), then sends the same request straight to the baseline fake.
6. Scores each response against the ground truth and writes `results/<timestamp>/report.{md,json}`, `results/latest.{md,json}`, and the gateway log for each run.

By default it runs twice, with and without the people dictionary (`--dictionary with,without`), so dictionary recall can be compared. With `--ner-url http://127.0.0.1:8790` the config also gets `ner: {url: ..., timeout: 120s}` pointing at a running NER sidecar (`detectors/gliner`), and the runs are labelled `dictionary+ner` and `ner-only`.

## The fake upstream

`upstream.Fake` implements `/v1/chat/completions` (streaming and non-streaming) and `/v1/models`. Its answer is deterministic:

```
TIMELINE
1. [name] Marisol Quenby
2. [email] m.quenby@halvorsen-bio.example
3. [date] 2026-04-17
...

ECHO
<every non-system message it received, verbatim, joined by a divider>
```

The timeline lists name-like, email-like and date-like strings in the order they appear. The echo section is the important part: whatever surrogates the gateway sent upstream come straight back, so a perfect gateway turns the echo back into exactly the original text. Streaming responses are cut into small chunks of varying size (1 to 11 bytes) so surrogates are routinely split across SSE events.

## Metrics

All counting is case-insensitive and word-boundary aware. Entities are deduplicated per item by `(type, surface text)`, so "Marisol" appearing five times in one thread is one unit.

**Sensitive** types are those whose policy action is `mask`, `shift` or `strip`. **Kept** types are `keep` (org and location by default).

- **Leak rate.** For each sensitive unit, the harness counts how often its surface form appears in everything the upstream received (all string fields of all recorded request bodies) versus how often it appears in the *non-sensitive* text of the input (the input with every sensitive span blanked out). If the upstream saw it more often, it leaked. The baseline subtraction is what keeps names that are also common words ("Will", "Rose", "May", "Grant") honest: the verb in "Will you confirm" is not a leak, the name is. Obvious variants also count:
  - `verbatim` or `case-insensitive` match of the span text,
  - `variant`: alternate renderings stored with the span (the same date as ISO, `January 2, 2006` or `01/02/2006`; the street line of an address; the token inside a password-reset link; the password inside a connection string),
  - `markdown-stripped`: `**Marisol** Quenby` arriving as `Marisol Quenby`,
  - `digits`: phone, SSN and card/account numbers re-punctuated (`4242-4242-4242-4242` for `4242 4242 4242 4242`).

  Reported overall, per entity type, per person reference variant (full, first, nickname, possessive, caps, lower, title+last, last-first, markdown, url-slug, handle), per dictionary coverage, and per request mode.
- **Restore accuracy.** For units that were masked or shifted (and did not leak), the share whose original text appears in the echo section of the final answer at least as many times as it appeared in the input. Stripped types are excluded because they are not supposed to come back. Dates that only came back in a different format are counted separately.
- **Unknown surrogates.** Anything the gateway reports itself: response headers or trailers whose name contains `outworld` and `unknown`/`hallucinat`/`unresolved` (for example `X-Outworld-Unknown-Names`, as a count or a comma list), or the same keys inside a top-level `outworld` object in the JSON body or SSE chunks.
- **Surrogate residue.** A format-agnostic check for surrogates that were not rehydrated: tokens that the upstream received, that do not exist anywhere in the input, and that still appear in the echo section of the final answer. Tokens that only occur inside square-bracketed placeholders in the upstream text (`[REDACTED]`, `[redacted card]`) are strip placeholders, not surrogates, and are ignored. Tokens that show up in more than 10% of responses are listed as placeholders instead of residue.
- **Over-masking.** For kept types, the share of units the upstream saw fewer times than the input contained them, i.e. the gateway altered an org or location it was told to keep. Quasi-identifiers (free-text descriptions that identify someone without naming them) are reported as "sent intact"; they are not in the leak rate because no entity type covers them.
- **Latency.** p50/p95 of the full request through the gateway, of the same request straight to the baseline fake, and of the per-request difference (overhead), per mode.
- **Errors.** Non-200 responses, malformed bodies, streams without `[DONE]`, grouped by the `error.code` in the body (for example `outworld_leak_blocked` when the gateway's own leak check refuses to send a request). A gateway that refuses everything has no leaks, so the error gate (`--max-error-rate`, default 2%) is what catches that.

The leak rate is computed over **all** requests, including ones that errored, because what matters is what reached the upstream (a request that fails after forwarding still leaked; a request blocked before forwarding sent nothing). The report also shows the leak rate over answered requests only, and the per-variant, per-dictionary, restore and over-masking breakdowns are over answered requests.

## Real upstream mode

```bash
export OUTWORLD_EVAL_API_KEY=...
go run ./cmd/outworld-eval --real-upstream https://api.example.com/v1 --model <model-id> --limit 30
```

The gateway's `cloud` upstream points at a local recording proxy in front of the real API, so the leak rate is still measured on what actually left the machine. Restore accuracy is not computed (a real model does not echo). Instead each item with ground-truth facts is sent both directly and through the gateway, the model is asked for a JSON timeline of `{date, person}` events, and both outputs are scored against the ground truth with field-level matching (exact ISO date, person matched on full name or surname). The report shows precision, recall and F1 for both paths. Keep `--limit` small; every item costs two real requests per mode plus two for the quality comparison.

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
upstream/          fake OpenAI-compatible upstream, recorder, recording proxy
score/             leak/restore/over-mask scoring, aggregation, timeline quality matching
internal/gateway/  config writer, build, launch, health check
internal/runner/   request loop and OpenAI client (streaming and non-streaming)
internal/app/      orchestration used by the CLI and the end-to-end tests
internal/report/   Markdown and JSON reports
internal/shim/     passthrough and oracle fake gateways used for self-tests
cmd/gencorpus      corpus generator
cmd/outworld-eval  the harness CLI
cmd/fake-outworld  fake gateway binary (FAKE_OUTWORLD_MODE=passthrough|oracle)
```

The end-to-end tests run the real pipeline against the fake gateway twice: a passthrough gateway must score a 100% leak rate, and an oracle gateway that masks exactly the ground-truth spans must score 0% leaks and 100% restore. Those two bounds are the harness's own correctness check.
