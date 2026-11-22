# Outworld dev notes

Privacy gateway in Go that speaks the OpenAI and Anthropic Messages APIs. Public open-source project (Apache-2.0): keep every file free of references to any private environment, machine, host or personal path.

## Layout

- `pkg/entity` entity types, policy actions, `Span`.
- `pkg/detect` detectors: `RegexDetector` (all pattern detectors, `regex.go`), `DateDetector`, `DictionaryDetector` (CSV/JSON, aliases, first/last-name-only, joined handles, optional gender column; `names.go` holds the common-word name-context guard `NameUseOK`), `NERDetector` (HTTP sidecar, code point offsets). `Resolve` picks non-overlapping spans (longest, then priority). `RunAll` runs detectors concurrently for texts of 8 KiB or more and keeps result order; `RegexDetector` does the same per pattern.
- `internal/textutil` `Fold` (lowercase, accents to ASCII); every name-table lookup goes through it.
- `internal/namedata` embedded given-name and surname tables (US government data, Wikidata, per-country lists; keys are folded ASCII; `merge_intl.py` rebuilds them); `pkg/detect/namedetect.go` is the model-free `NameDetector` built on them (full names, lone first names, particles, surname-first, initials, lowercase full names, labels, kinship, chat speakers, profile slugs, with tech-term and common-word guards).
- `pkg/vault` `Vault` (in-memory index, real/surrogate matchers) over a `Backend`. `FileBackend` = magic header + length-prefixed AES-256-GCM frames, frame index in AAD, frame 0 holds the vault secret, `flock` on `<path>.lock` for multi-process writers, truncated tail frames are dropped on next write.
- `pkg/surrogate` HMAC-seeded generators and bundled name lists.
- `pkg/dateshift` date finding (English plus es, de, fr, pt, it, nl month names, compact and spoken forms; dotted dates read day-month), format-preserving shift; `FindLoose`/`ShiftLoose` are the lenient variants used for rehydration; yearless dates shift cyclically within a 365-day year so they round-trip.
- `pkg/outworld` `Engine` + `Session` (mask, leak check, rehydrate, `StreamRehydrator`, unknown-name flags).
- `internal/config` YAML schema (strict, unknown keys rejected). The eval harness writes configs against these keys; do not rename them.
- `pkg/outworld/memo.go` transcript memo (real text to the exact masked bytes already sent, per scope, LRU by bytes); `learn.go` builds a per-session dictionary from person spans found anywhere in the request, including names inside `first.last@` email addresses.
- `internal/gateway` HTTP server, routing, request masking, response and SSE rehydration. `anthropic.go` and `anthropic_stream.go` are the Anthropic Messages path.
- `internal/app` builds an engine from config. `cmd/outworld` CLI.
- `detectors/gliner` and `detectors/privacy-filter` optional Python NER sidecars with the same `/detect` contract.
- `eval/` is a separate Go module owned by the eval harness. `corpus.json` is the tuned corpus, `holdout.json` is a development set (detectors were tuned on its misses), `holdout2.json` is blind: never read its leaks to tune detectors, or its number stops meaning anything.

## Invariants

- Vault key required; no plaintext fallback.
- `Vault.Assign` runs the generator while holding the vault lock: generator callbacks must not call back into the vault (pre-compute lookups first).
- Leak check runs on every transformed string (detector re-scan + known-real scan, skipping inserted surrogate ranges) and on untransformed fields like message `name` and JSON object keys. In `Session.Mask`, findings that do not overlap an inserted range are masked in place and the text is re-scanned (up to 4 rounds); real text equal to a person surrogate is re-masked so rehydration stays correct (this overrides a kept org span). A known-real name next to its own surrogate is not a violation when the only uncovered words are single letters or name particles. Only residual findings (and any finding from `Check`) become violations: 422, no values in the error.
- The gateway masks each request twice: a priming pass (`Session.SetPriming(true)`) detects every text, caches spans and assigns surrogates without changing anything, then the real pass masks with the vault already knowing every name in the request.
- Person surrogates: role (given/family/nickname) comes from the dictionary canonical name, else position. Nicknames get their own vault kind `person.nick`, derived from the first-name surrogate. Reals are stored in canonical casing. Given-name surrogates follow `surrogate.GenderOf` or the dictionary `gender` column.
- Rehydration restores dates via the session's exact shifted-string map first (style-ambiguous forms like `May`), and phones by digits (`Vault.ReversePhone`) in any formatting.
- Byte stability: anything already sent upstream must be re-sent byte-identical (prompt caches and Claude's preserved-thinking prefix check depend on it). `Session.Mask` consults the memo first; the gateway calls `Session.Remember(rehydrated, upstream)` for every model string it translates back. Never rehydrate or re-mask `thinking` blocks. JSON is re-encoded with sorted keys, which is stable across requests.
- Anthropic path forwards `anthropic-*` headers, query string, unknown fields and upstream error bodies unchanged.
- Logs carry counts and routes only.
- Streaming: `StreamRehydrator.safeCut` holds back any tail that could be the start of a surrogate or a date, and `avoidStraddle` never cuts inside a complete match.

## Commands

```bash
go build ./... && go vet ./... && go test ./...
go test -race ./...
```
