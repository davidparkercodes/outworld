# Outworld

**Your data stays in the nest. You decide what leaves, it leaves in disguise, and the answer comes back translated.**

Outworld is an OpenAI-compatible privacy gateway. Point any OpenAI-compatible client at it, and it lets you use cloud LLMs on personal data (email, notes, contacts, records) without the provider seeing the real personal details. Names, emails, phones and addresses are swapped for realistic stand-ins, dates are shifted, secrets are removed, and the model's answer is translated back before it reaches you.

It works with any OpenAI-compatible provider, and with local servers (Ollama, llama.cpp, vLLM) for the requests you never want to leave your machine.

## Why "Outworld"

In ant-keeping, the *outworld* is the foraging arena attached to the nest. The colony stays safe inside; only what you choose goes out. Outworld is that arena for your data: the nest is your machine and your vault, the outworld is the cloud model.

## How it works

```
  your app / agent / script
            |
            |  POST /v1/chat/completions   (real data)
            v
  +-----------------------------------------------------------+
  |  OUTWORLD                                                 |
  |                                                           |
  |  1. route     rules + X-Outworld-Route header             |
  |               masked | local | passthrough | block        |
  |                                                           |
  |  2. detect    regex (email, phone, address, ids, cards,   |
  |               keys, OTP codes, magic links, dates)        |
  |               + your dictionary (contacts, aliases)       |
  |               + optional NER sidecar (GLiNER)             |
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

### Docker

```bash
docker build -t outworld:local .
export OUTWORLD_VAULT_KEY=$(docker run --rm outworld:local keygen)
cp compose.example.yml compose.yml
docker compose up -d
```

The image is distroless, static and runs as a non-root user. The vault lives in the `/data` volume.

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

Defaults: people, emails, phones, addresses and IPs are masked; dates shifted; orgs, locations and plain URLs kept; secrets, government IDs and financial numbers stripped. Secret-bearing links (password resets, magic links, URLs with tokens) are classified as `secret` and stripped. Secrets also cover passwords in prose (`my temporary password was ...`), `KEY=value` / `export API_KEY=...` assignments, and long hex or base64 tokens next to words like `credential` or `token`. Any SSN-shaped number next to `SSN`, `social security` or `tax id` is a `gov_id` in any separator form (`123-45-6789`, `123 45 6789`, `123456789`), regardless of whether that range was ever issued.

### Surrogates

- Realistic, never `<PERSON_1>`: `John Smith` becomes `Kasimir Sandoval`, `John` alone becomes `Kasimir`, `JOHN SMITH` becomes `KASIMIR SANDOVAL`, `john.smith` in a URL becomes `kasimir.sandoval`, `John's` keeps its `'s`.
- Deterministic per vault (HMAC of a vault secret), consistent across requests and a whole corpus, collision-free, and never equal to any real value the vault or your dictionary knows.
- Shape-preserving: emails stay valid on reserved `example.com/.org/.net` domains, phones use the fictional `555-01xx` range with the original formatting, addresses keep number length, street suffix, unit and state, IPs use documentation ranges.
- Name lists are bundled and generated for this project; once exhausted, a syllable generator takes over.
- Gender-consistent: first names are replaced with first names of the same likely gender (a bundled list of common first names; unknown names get gender-neutral surrogates), so `Priya ... her` stays coherent. A dictionary `gender` column overrides the guess.
- One identity per person: every alias of a dictionary entry maps onto parts of the same surrogate. `Marcus Delaney` becomes `Ezra Sandoval`, `Delaney` becomes `Sandoval`, and the nickname `Marc` becomes a nickname derived from `Ezra` (for example `Ez`), so the model sees one person.
- A surrogate is never a word that appears capitalized anywhere in the same request, so a stand-in cannot collide with a real name you are sending.

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

- `POST /v1/chat/completions` streaming and non-streaming, tools and tool calls (arguments are masked and rehydrated as JSON).
- `POST /v1/completions` legacy completions.
- `GET /v1/models` passthrough to the default upstream.
- `GET /healthz`.

Set `OUTWORLD_TOKEN` (or `auth.token_env`) to require `Authorization: Bearer <token>` from clients. When set, requests without it are rejected. The client's token is never forwarded upstream; each upstream uses its own `api_key_env`.

Messages containing images, audio or files are refused on the `masked` route (they cannot be masked) unless you set `policy.allow_images: true`. Use `local` or `passthrough` for those.

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
policy:
  default_route: masked           # masked | local | passthrough | block
  masked_upstream: cloud
  local_upstream: local
  passthrough_upstream: tee
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
max_body_bytes: 33554432
```

Unknown keys are rejected so typos do not silently weaken your policy.

## Threat model and honest limits

What Outworld is for: reducing how much identifying personal data a cloud model provider receives, while keeping answers useful, with a hard stop when it notices it is about to leak something it knows about.

What it protects against:

- The provider (and its logs, retention, training pipelines, breaches) seeing the names, contact details, addresses, exact dates, account numbers, IDs and secrets that Outworld detects.
- Accidental leaks of values the vault already knows: every outbound payload is re-scanned for known real values and for unmasked detector hits. Anything found is masked in place with its existing surrogate (or a new one) and the payload is re-scanned; a known first name followed by a capitalized word is treated as a full name. Only what still cannot be transformed (for example a value in a JSON key or message `name`) blocks the request.
- Mentions split across messages: every text in a request is detected first, so a name that is only recognizable in a later message is also masked in earlier ones.
- Credentials and one-time codes reaching any cloud model at all (stripped, never restored).

What it does **not** do:

- **Detection is best-effort.** Regexes, your dictionary and an optional NER model will miss things: unusual names, nicknames you did not list, people described rather than named ("my sister's husband"), names in other scripts, typos. The leak check only blocks what some layer recognizes. Add a dictionary; add the NER sidecar; review what you send.
- **Masking is not anonymization.** Quasi-identifiers survive: a rare job title plus a small town plus a hospital stay can identify someone even with every name replaced. So can writing style, the structure of your life, and facts you chose to `keep`. Date shifting hides exact dates but keeps intervals (which is the point), and intervals can be identifying too.
- **The provider still sees the content.** Your medical question is still a medical question about *someone*. If the substance itself is what must not leave, route it `local` or to a `passthrough` confidential-compute endpoint, or `block` it. That is what routing rules are for.
- **Model output is untrusted.** Rehydration translates every surrogate and every date in the answer, including dates the model made up. Surrogate phone numbers are recognized in any formatting the model chooses (dashes, dots, spaces, parentheses, `+1`, digits only). Invented people that look like surrogates are flagged in `X-Outworld-Unknown-Names`, not removed.
- **The vault is the crown jewels.** It maps every surrogate back to a real value. It is encrypted at rest (AES-256-GCM with an authenticated, append-only record log), and Outworld refuses to run without a key. Protect the key like a password-manager master key. Anyone with both the vault and the key can reverse everything.
- **Not covered:** images, audio and files (refused on the masked route by default), single-letter initials (left as-is), month-year only dates like "March 2024" (not shifted), metadata outside the message text (Outworld drops the `user` field but does not rewrite `tools` schemas or system-level metadata your client adds).
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

## Optional NER sidecar

`detectors/gliner/` contains a small FastAPI service around GLiNER for detecting people, organizations and places that are not in your dictionary. Outworld works without it. See [detectors/gliner/README.md](detectors/gliner/README.md).

## License

Apache-2.0. See [LICENSE](LICENSE).
