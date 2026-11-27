# Outworld

**Your data stays in the nest. You decide what leaves, it leaves in disguise, and the answer comes back translated.**

Outworld is a privacy gateway that speaks both the OpenAI and the Anthropic Messages API. Point a client at it, and it lets you use cloud LLMs on personal data (email, notes, contacts, records, the files a coding agent reads) without the provider seeing the real personal details. Names, emails, phones and addresses are swapped for realistic stand-ins, dates are shifted, secrets are removed, and the model's answer is translated back before it reaches you.

It works with any OpenAI-compatible provider, with the Anthropic API (including as `ANTHROPIC_BASE_URL` for Claude Code), and with local servers (Ollama, llama.cpp, vLLM) for the requests you never want to leave your machine.

What changed in each version, and what to check before upgrading, is in [CHANGELOG.md](CHANGELOG.md).

**Status:** 0.3.0, an early release. Over 900 tests, including scale tests that keep every detector linear-time. Leak rates on a blind corpus are published below, along with every known gap. Read [Threat model and honest limits](#threat-model-and-honest-limits) before you rely on it.

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
outworld dry-run request.json                         # what would be sent, and why it would block (exit code 3)
```

`dry-run` takes a chat request body (file or stdin, `--api openai|anthropic`) and prints the masked request, every finding and the route, without calling any upstream. The same report is available over HTTP by adding `X-Outworld-Dry-Run: 1` to a normal request. A dry run still writes the surrogates it assigns to the vault. See [blocked requests and dry runs](docs/blocked-requests.md).

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
- Locale-aware addresses: an address that reads as French, Spanish, Portuguese, Italian, German (including Austrian and Swiss forms), Dutch or UK gets a stand-in from the same country, with street names in that language's grammar, a real mid-sized town and a postal code that fits it. `12 rue de Rivoli, 75001 Paris` becomes something like `63 rue de Montreuil, 47000 Agen`, and `Lindenstraße 8, 50674 Köln` something like `Bergstraße 5, 35060 Marburg`. Multi-word street and town names keep their word count and connectives (`Jerez de la Frontera` becomes something like `Palencia de la Sierra`). Detection covers street-first and number-first forms, postcodes with a country prefix (`CH-8001`), floor and unit parts (`2º B`, `Appt 12`, `Top 3`), multi-line blocks and a trailing country line. A postcode and town with no street are masked only next to an address label or a street line. Danish, Swedish and Norwegian street forms are detected and get a generic stand-in. US-style addresses are generated exactly as before.
- Family-name-first names get stand-ins from the same origin with the same shape: `Nguyen Van An` becomes something like `Kieu Van Phuc`, and `Park Ji-hoon` something like `Hwang Sook-kwan`.
- Handles do not keep the real initial: `kchen` becomes `sradcliffe` when the same request shows the person's full name (so it matches their stand-in), and otherwise gets a different letter.
- Name lists are bundled and generated for this project; once exhausted, a syllable generator takes over.
- Gender-consistent: first names are replaced with first names of the same likely gender (a bundled list of common first names; unknown names get gender-neutral surrogates), so `Priya ... her` stays coherent. A dictionary `gender` column overrides the guess.
- One identity per person: every alias of a dictionary entry maps onto parts of the same surrogate. `Marcus Delaney` becomes `Ezra Sandoval`, `Delaney` becomes `Sandoval`, and the nickname `Marc` becomes a nickname derived from `Ezra` (for example `Ez`), so the model sees one person.
- A surrogate is never a word that appears capitalized anywhere in the same request, so a stand-in cannot collide with a real name you are sending.

### Detection layers

Detection is layered, and each layer can only add findings:

1. **Patterns.** Emails, phones, addresses, dates, links and codes; national ID formats checked by their checksum or structure, and ID values after a label in several languages; cards, IBANs (length and mod 97 per country), labelled bank numbers and crypto addresses; vendor API key shapes, `key=value` secrets, auth headers, cookies, private keys, recovery codes, and password-shaped tokens next to a credential word. No model, a few milliseconds for a typical prompt.
2. **Built-in name detector.** Finds people without a model: a known first name followed by a surname (`John Smith`, `JOHN SMITH`), a known first name on its own, names after labels (`Emergency contact: May Lindenbrook`, `ONCALL: ...`, `From: ... <email>`), after relationship words (`my sister Ottoline`), as chat speakers (`[09:14] ravi:`), and in profile URLs (`/people/abelard-ashdown`). It also handles particles (`Anna van der Berg`; a particle that ends a name is the surname, as in `Ruby John Van`, except `St` and `Ste`, which are read as a street abbreviation and stay as typed, as in `John St`), surname-first forms (`SMITH, John`), initials (`J. Smith`, `John S.`), accents (`José` and `Jose` are the same name) and names inside `first.last@` addresses. Family-name-first names are recognized for common Vietnamese, Chinese and Korean family names (`Nguyen Van An`, `Zhang Wei`, `Kim Min-jun`); family names that are also ordinary words (`Le`, `Do`, `He`, `Park`, `Sun`, `Kim`, `Lee`) need a distinctive given name, a middle marker such as `Van` or `Thi`, or name context, or a sentence that treats them as a person (`Sun Yang will join the call`, `I asked Ma Long to review it`). It also reads attendee-style lists (`Present:`, `Attendees:`, `Cc:`), initialisms of people named in the same text (`LP to send notes` after `Liam Parker` appears in the attendee list), all-caps surname-first names with particles (`CARVALHO DE SOUZA, João Pedro`), and Spanish, French, German, Italian and Portuguese honorifics (`el señor José Luis Ramírez Ortega`). First names inside venue or title names (`The Amber Room`), multi-word places and terms (`Monte Carlo`, `Turing test`) and `City XX` forms are left alone. Code handles count as people where the position makes them a username: `TODO(mwilliams)` and `FIXME(kchen)`, the user prefix of a branch (`git checkout -b jsmith/retry-logic`, `origin/kchen/fix`), `@mwilliams` mentions, `/home/jdoe/` and `/Users/jdoe/` path segments, git trailers (`Reviewed-by: kchen`), branch prefixes after git commands and in prose (`git push origin dlopez/fix-cache`, `PR from kchen/fix-auth`), and a lowercase handle in prose right after a cue word when it has a name shape (`please ask kchen to review`). Branch prefixes and mentions need a name-table match (an initial plus a known surname, or a known first name), so `feature/login`, `@types/node` and `@param` are left alone, and service accounts such as `root`, `ubuntu` and `runner` are skipped; a handle with no recognizable name in it (`@zorgblat`) is missed outside home directories and trailers. A bundled list of a few hundred well-known organizations whose names look like a person (`Morgan Stanley`, `Goldman Sachs`, `Johnson & Johnson`, `Ralph Lauren`) is treated as `org`, and so is a name followed by a corporate word (`Okafor Partners`, `Maria Garcia Holdings`, `Becker & Sons`). A real person who shares one of those names (`Edward Jones`, `Raymond James`) is masked from your dictionary, after a title (`Mr. Edward Jones`), after a relationship word, in a sign-off, after a label such as `Contact:`, or when the text treats the name as a person (`Edward Jones called me`) and nowhere as a company. It uses public-domain name tables (about 9,400 first names and 32,800 surnames from US government data, Wikidata and per-country popular-name lists, see `internal/namedata`). Coverage is best for names written in Latin script and common in Europe and the Americas; it will still miss many others, and nicknames in particular. Names that are also ordinary words (`Will`, `May`, `Grant`, `Faith`) only count in name context. Technology names that are also given names or surnames (`Ruby`, `Julia`, `Jenkins`, `Docker`, and the planet names `Mercury` to `Pluto`) follow the person rules: `Hi Ruby,` masks, `Ruby 3.3 is installed.` keeps, and once the vault knows a person called Ruby, a later lone `Ruby` masks unless strong technical evidence sits right next to it. Assistant names (`Claude`, `ChatGPT`, `Siri`, `Alexa`) are kept when used alone (`Hi Claude,`). Once the vault knows a person with that name, a later lone use masks when the sentence treats it as a person (`Claude wrote most of this module.`), and in a request that names the person in full (`Claude Shannon`) every other bare use masks too, except in a code span or identifier. Turn the layer off with `policy.name_heuristics: false`.
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

Order of evaluation: any `block` rule that matches wins; then the first matching rule, else `default_route`; then the `X-Outworld-Route` request header, as described below. Rules match by whole-word keywords (`match_any`), regexes (`match_regex`) or detected entity types (`entities`). They see exactly the texts the masked route rewrites or leak-checks (see [What is masked](#what-is-masked)), never image data.

The route header has three settings:

- `allow_route_header` unset (the default): the header may only make the decision stricter. Routes rank `block`, then `local`, then `masked`, then `passthrough`. A header that ranks above the decided route replaces it (so `local` on a masked request sends it raw to `policy.local_upstream`); one that ranks the same keeps the decided route and its upstream; one that ranks lower is refused with 403 `outworld_route_error`. The header may name an upstream only when it is the one that route would use anyway (`local:local`); any other upstream is refused with 403. So `masked` cannot move a request that a rule or `default_route` sent `local` to the cloud.
- `allow_route_header: true`: the header overrides the rules and `default_route` and may choose any route and any configured upstream, e.g. `local` or `passthrough:tee`. Only a matching `block` rule still wins. This means a client can override `default_route: block` and send raw text to any upstream in your config; anyone holding the gateway token can skip masking. Use it only when every client is trusted.
- `allow_route_header: false`: the header is ignored.

An unknown route name in the header is refused with 400 `outworld_route_error`, and so is a route or upstream the header asks for that has no matching upstream for that API. 502 `outworld_route_error` is kept for configuration problems in the rules or `default_route` and for detector failures.

Changed in this release: the default used to let the header choose any route, including `passthrough`, ahead of the rules. Configs that rely on the header to loosen routing (`passthrough`, or `masked` over a `local` rule) must now set `allow_route_header: true`.

### What is masked

Each request is walked twice per API. The first walk knows the fields of the format and marks each place as masked (rewritten with stand-ins, then leak-checked), checked (leak-checked, never rewritten; a finding blocks the request with 422), or sent unchanged. A place is sent unchanged only when its value has the exact shape that field allows (listed below); any other value or type is left to the second walk. The second walk visits every string and every object key the first walk did not claim, anywhere in the body, and gives it the full check. Routing rules read the same places as the leak check.

Changed in this release: fields the gateway does not know, and protocol fields with an unexpected value, used to go upstream without any check. A request that carries personal data in such a field is now refused with 422 and names the field, and block rules now see those fields.

There are three kinds of check:

- full: every detector, including the name detector, plus the scan for real values the vault already knows.
- narrow: pattern detectors (emails, phones, IDs, cards, secrets), the date detector, the vault scan and the dictionary, without the name heuristics. For tool and other names the vault scan counts a person only when the name holds two or more of their name words (`notify_kwame_asante`, `notifyKwameAsante`), so a single-word record such as `Edit` left in an old vault does not block every request that carries an `Edit` tool. Used for identifiers chosen by the client: tool and function names, `tool_choice` names, `custom.name`, `json_schema.name`, `mcp_servers[].name`, schema `type`, `format`, `pattern` and `$ref` values, and provider tool results. `Edit` or `Bash` as a tool name passes; `call_8045551234` or a known person's name in one does not. Block rules see these fields by keyword and regex, and entity rules only through those detectors.
- id: narrow without the high-entropy, bare-hex and date patterns (the date pattern stays when the value has separators such as `-`, `_` or `:`). Used only when the value itself looks like an identifier: a known prefix (`toolu`, `srvtoolu`, `mcptoolu`, `call`, `msg`, `file`, `container`, `user`, `resp`, `fc`, `rs`, `item` and similar) plus `_` and 16 or more letters and digits with at least two digits, or `chatcmpl-` plus the same tail; a UUID; a ULID; 16 to 128 lowercase hex characters that are not all digits (hex longer than 40 characters that decodes to mostly printable text is not an id); or the Claude Code `user_<hex64>_account_<uuid>_session_<uuid>` form. It applies to `id`, `tool_call_id`, `tool_use_id`, `container`, short base64 values at the opaque positions below, and keys ending in `id`, `key` or `identifier` in any casing (`user_id`, `userId`, `user-id`), including keys inside `metadata` and tool arguments. Keys ending in `key` other than `prompt_cache_key` and `idempotency_key` keep the high-entropy pattern, since those hold secrets. Any other value in those fields gets the full check, together with a copy split at `_`, `-`, `.` and case changes (`user_ElenaPetrova` is also checked as `user Elena Petrova`) and, for long hex, the decoded text. `model` gets the narrow check without the date pattern.

OpenAI format:

- masked: message `content` (strings and `text`, `input_text`, `output_text` and `refusal` parts), `reasoning_content`, `reasoning`, `refusal`, tool call `arguments` (as JSON values, including keys such as `signature`; arguments with nothing to mask keep their original bytes), `custom` tool call `input`, `annotations[].url_citation.title`, `prompt`, `suffix`, `input`, `prediction.content`, `stop`, `metadata` values, `web_search_options.user_location`, and tool and `response_format.json_schema` schemas as below.
- checked (full): message `name`, every JSON object key, `custom` tool grammar `definition`, image URLs that are not data URLs, file names, and everything the first walk does not know, for example `documents`, `safety_identifier` values that are not hex, extra keys on content parts, and a known field holding an object instead of a string.

Anthropic format:

- masked: `system` text, text blocks and their `citations` (`cited_text`, `document_title`, `title`), `tool_use` inputs, `tool_result` and `search_result` content, `document` text sources and `text/plain` base64 documents (decoded, masked, encoded again), titles and context, `stop_sequences`, the fields of server tools such as `user_location`, and tool and `output_format` schemas as below.
- checked (full): every JSON object key, block types Outworld does not know, thinking blocks outside assistant messages, system blocks that are not `text`, image and document URL sources, and every other field the first walk does not know (extra keys on blocks, citation `url`, other `metadata` keys).
- checked (narrow, without the date pattern): provider server tool results (`web_search_tool_result`, `web_fetch_tool_result`, `code_execution_tool_result`, `bash_code_execution_tool_result`, `text_editor_code_execution_tool_result`, `tool_search_tool_result`) that come after their `server_tool_use` block in the same assistant message, so public names and dates in search results and fetched pages do not block the request while phones, emails, IDs and known names still do. A result type outside that list, one in a user message, or one before or without its call gets the full check. A client can forge the pair, so the narrow check applies to all of its text; the `encrypted_content` of a `web_search_result` gets the opaque value check described below the table. `mcp_tool_result` content is masked like `tool_result`.
- checked (narrow, without the high-entropy and bare-hex patterns): `system[0]` when it is exactly the billing line Claude Code sends, `x-anthropic-billing-header: cc_version=<x.y.z[.hex]>; cc_entrypoint=<word>; cch=<hex>;` (each pair at most once, no others). Any other text after that prefix, or the same line in another system block, is masked like any system text. `cc_entrypoint` keeps the word shape rather than a fixed list, because a wrong list would mask a real entrypoint and break prompt caching.
- sent unchanged, but read by block rules and checked: in assistant messages only, `thinking` blocks (the `thinking` text is checked against the vault, the `signature` gets the opaque value check) and `redacted_thinking` blocks (`data` gets the opaque value check). Any other key on them gets the full check. The thinking text a client sends in an assistant turn is forwarded as is, by design: it is the provider's own earlier output, signed, and changing it would break the signature check. A real value the vault already knows in it still blocks the request.
- checked (narrow): `page_age` of a web search result only when it is not a date or a relative age (`April 3, 2025`, `2025-04-03`, `3 days ago`); those are sent unchanged.

Tool schemas (`tools`, legacy `functions`, `response_format.json_schema.schema`, `output_format.schema`): `description`, `title`, `$comment`, the string values of unknown keywords, and the string values inside `enum`, `examples`, `const` and `default` (including keys such as `signature` inside them) are masked; a tool call that answers with a masked `enum` value is restored to the real one. Property names and `required` get the full check. Numbers and booleans are sent unchanged.

Media (only with `policy.allow_images: true`; otherwise refused with 422): only the known part and block types count as media (`image_url`, `input_audio` and `file` parts on OpenAI, `image` and `document` blocks on Anthropic). Base64 data, including data URLs, must decode to the magic bytes of its declared type (PNG, JPEG, GIF, WebP, PDF, WAV, MP3); data that does not match is refused with 422 `outworld_unsupported_content`. An unknown block with a `source`, or a made-up part type, gets the full check.

Sent unchanged only with this exact shape (anything else gets the full check):

| field | shape |
|---|---|
| `model` | `^[A-Za-z0-9][A-Za-z0-9._:/@\[\]-]{0,127}$`, then the narrow check without the date pattern |
| `service_tier` | `auto`, `default`, `flex`, `priority`, `scale`, `standard_only` |
| `reasoning_effort` | `none`, `minimal`, `low`, `medium`, `high`, `xhigh` |
| `verbosity` | `low`, `medium`, `high` |
| string `tool_choice` | `auto`, `none`, `required`, `any` |
| `tool_choice.type` | `auto`, `any`, `tool`, `none`, `function`, `custom`, `required`, `allowed_tools` |
| message `role` | `system`, `developer`, `user`, `assistant`, `tool`, `function` |
| `strict` (tools, functions, `json_schema`) | a boolean |
| `tools[].type` | `function`, `custom`, `web_search`, `web_search_preview`, `file_search`, `code_interpreter`, `computer_use_preview`, `image_generation`, `mcp`, `local_shell`, or a versioned name `^[a-z_]+_20\d{6}$` |
| `tool_calls[].type` | `function`, `custom` |
| `annotations[].type` | `url_citation` |
| `response_format.type` | `text`, `json_object`, `json_schema` |
| `prediction.type` | `content` |
| custom tool `format.type`, `grammar.syntax` | `text`, `grammar`; `lark`, `regex` |
| `thinking.type` | `enabled`, `disabled`, `adaptive` |
| `output_format.type` | `json_schema` |
| `mcp_servers[].type` | `url` |
| `citations[].type` | `char_location`, `page_location`, `content_block_location`, `web_search_result_location`, `search_result_location` |
| thinking `signature`, `redacted_thinking.data`, web search `encrypted_content`, `citations[].encrypted_index` (assistant messages only) | base64 (standard, or another base64 alphabet for values of 96 characters or more), then the opaque value check below. Invalid base64 gets the full check |
| `context_management.edits[].type` | a versioned name `^[a-z_]+_20\d{6}$` |
| text document `media_type` | `text/plain` |
| media `media_type` | `^[a-z]+/[a-z0-9.+-]+$`, with data that matches it |
| `image_url.detail` | `auto`, `low`, `high` |
| `input_audio.format` | `wav`, `mp3` |
| part and block `type` | the types the first walk recognizes |

Every value sent unchanged is still read by block rules (`match_any`, `match_regex`), except verified media data. Also sent unchanged: `user` on OpenAI requests and `metadata.user_id` on Anthropic requests when `keep_metadata` is false (both are removed before sending), and `mcp_servers[].authorization_token`, a credential meant for the provider, which is forwarded unread except by block rules. With `keep_metadata: true`, `metadata.user_id` gets the id check.

Response headers:

| header | meaning |
|---|---|
| `X-Outworld-Route` | `masked`, `local`, `passthrough`, `block`, or `blocked-leak` |
| `X-Outworld-Route-Reason` | `default`, `header`, or `rule:<name or index>` |
| `X-Outworld-Masked` | number of values transformed |
| `X-Outworld-Dry-Run` (request) | set to `1` to get a report of what would be sent instead of calling the upstream |
| `X-Outworld-Unknown-Names` | surrogate-looking names in the answer the vault does not know (the model probably invented a person). On streams it arrives as an HTTP trailer and an SSE comment `: outworld unknown_names=...` |

## Endpoints

OpenAI format (upstreams with `api: openai`, the default):

- `POST /v1/chat/completions` streaming and non-streaming, tools and tool calls (arguments are masked and rehydrated as JSON). Tool definitions, `metadata`, `prediction`, `stop` and `response_format` schemas are masked as listed under [What is masked](#what-is-masked).
- The query string is forwarded unchanged.
- `POST /v1/completions` legacy completions.

Anthropic format (upstreams with `api: anthropic`, selected by `policy.anthropic.*`):

- `POST /v1/messages` streaming and non-streaming. `system`, text blocks, `tool_use` inputs, `tool_result` content, `stop_sequences` and tool definitions are masked (details under [What is masked](#what-is-masked)); text and tool inputs in the answer are rehydrated, including across `text_delta` and `input_json_delta` events.
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
  allow_route_header:             # unset: header may only make routing stricter; true: any route; false: ignored
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
max_opaque_bytes: 786432          # 768 KiB of base64; a longer opaque value (signature, encrypted_content) is refused with 422 unscanned
```

Unknown keys are rejected so typos do not silently weaken your policy.

## How well it works: measured

The numbers below come from the eval harness in [`eval/`](eval/README.md). It sends synthetic prompts with labelled sensitive values through the gateway to a fake upstream that records exactly what a provider would have received. **Leak rate** is the share of labelled sensitive values that reached the upstream unchanged. **Over-masking** is the share of things that should have been left alone (companies, cities, product names, version strings, hashes) that were changed anyway. **Blocked** is the share of requests the fail-closed leak check refused with a 422.

There are four corpora, and the difference between them matters more than any single number:

- **Tuned**: 152 items. The detectors were developed against it. Its numbers show what the detectors do on forms they were built for, and nothing more.
- **Development**: 85 items written without access to the detector code. After the first run on it, general gaps it exposed were fixed, so it is no longer blind.
- **Second held-out**: 120 items, 160 invented people, about 33,000 words, written without access to the detector code. No detector was changed after looking at its leaks, but its per-type totals steered the 0.3.0 work on secrets and IDs, so it is no longer fully blind.
- **Blind**: 131 items, 188 invented people, about 43,000 words and 4,208 labelled spans, written by hand without access to the detector code and much heavier on international names, IDs and addresses than the others. Only its totals have been looked at, and no detector was tuned against it. This is the number to believe.

One disclosure: two lines of a generator file for the second held-out corpus, and the text of one of its items, were seen by accident during maintenance. Nothing was tuned against them.

Leak rate in 0.3.0, through the OpenAI API, measured on 2026-11-26 (the Anthropic API, streaming, and repeat runs with a fresh vault land within about two points):

| Detectors | Second held-out | Blind |
|---|---|---|
| Patterns and built-in names only | 16.5% | 20.1% |
| Plus your dictionary (about half the people listed) | 10.5% | 16.2% |

On the blind corpus, by kind of value (with a dictionary): emails 0%, phones 8%, dates 9%, names 16%, secrets 16%, bank and card details 30%, government IDs 34%, addresses 54%. The addresses that get through are mostly formats from countries the patterns do not cover; the names are mostly nicknames, first names that are not in the tables, and names in non-Latin scripts.

The cost side in 0.3.0, without a sidecar:

| Corpus | Over-masking | Blocked | Added latency, median |
|---|---|---|---|
| Second held-out | 21% | 1% to 2.5% | about 65 ms |
| Blind | 16% | 4.6% | about 110 ms |

The blind corpus and the second held-out corpus are deliberately full of things that look sensitive and are not (companies named after people, cities that are also first names, commit hashes, version strings), which is why over-masking is high. Over-masking is reversible: the answer is still translated back, but the model loses what it knew about the real company or place.

With an NER sidecar (measured on 0.2.0, second held-out corpus): dictionary and GLiNER 9.3% leak, 26% over-masking, 3% blocked, 2.4 s median added latency on 8 CPU threads; dictionary and Privacy Filter 8.6% leak, 29% over-masking, 3% blocked, 0.5 s. The tuned and development corpora land at 0% to 0.3% and 3.5% to 13.5% across all configurations.

What to take from this:

- Outworld removes most of what identifies people in ordinary prompts. It does not remove all of it: on text it has never seen it misses roughly one sensitive value in six with a dictionary, and about one in ten with a dictionary and a model.
- A dictionary of the people you actually talk about is the cheapest large improvement. A sidecar is the next one.
- Restore accuracy (answers translated back correctly) is above 99% in every configuration: 99.3% on the blind corpus, 99.8% on the second held-out corpus.
- These are synthetic prompts against a fake upstream. Sidecar latencies were measured on a shared, busy CPU host with no GPU and vary a lot with prompt length; the 95th percentile on long agent transcripts is tens of seconds for GLiNER.

Reproduce any cell with `go run ./cmd/outworld-eval --corpus corpus/holdout3.json --people corpus/holdout3-people.csv --configs regex-only,dictionary` from `eval/` (or `holdout2` for the second held-out corpus). If you change a detector after reading the blind corpus results, it stops being blind; write a new one.

## Threat model and honest limits

What Outworld is for: reducing how much identifying personal data a cloud model provider receives, while keeping answers useful, with a hard stop when it notices it is about to leak something it knows about.

What it protects against:

- The provider (and its logs, retention, training pipelines, breaches) seeing the names, contact details, addresses, exact dates, account numbers, IDs and secrets that Outworld detects.
- Accidental leaks of values the vault already knows: every outbound payload is re-scanned for known real values and for unmasked detector hits. Anything found is masked in place with its existing surrogate (or a new one) and the payload is re-scanned; a known first name followed by a capitalized word is treated as a full name. Only what still cannot be transformed (for example a value in a JSON key or message `name`) blocks the request. The 422 lists each finding by type, field and position with a fully redacted preview and a hint, and never contains any character of the value ([details](docs/blocked-requests.md)).
- Mentions split across messages: every text in a request is detected first, so a name that is only recognizable in a later message is also masked in earlier ones.
- Credentials and one-time codes that Outworld recognizes reaching any cloud model at all (stripped, never restored). Recognition is far from complete here: on the blind corpus about a third of secrets got through, mostly passwords and tokens with no label next to them. Do not rely on Outworld as a secret scanner.

What it does **not** do:

- **Detection is best-effort.** Regexes, your dictionary and an optional NER model will miss things: unusual names, nicknames you did not list, people described rather than named ("my sister's husband"), names in other scripts, typos. The leak check only blocks what some layer recognizes. Add a dictionary; add the NER sidecar; review what you send.
- **Masking is not anonymization.** Quasi-identifiers survive: a rare job title plus a small town plus a hospital stay can identify someone even with every name replaced. So can writing style, the structure of your life, and facts you chose to `keep`. Date shifting hides exact dates but keeps intervals (which is the point), and intervals can be identifying too.
- **The provider still sees the content.** Your medical question is still a medical question about *someone*. If the substance itself is what must not leave, route it `local` or to a `passthrough` confidential-compute endpoint, or `block` it. That is what routing rules are for.
- **Model output is untrusted.** Rehydration translates every surrogate and every date in the answer, including dates the model made up. Surrogate phone numbers are recognized in any formatting the model chooses (dashes, dots, spaces, parentheses, `+1`, digits only). Invented people that look like surrogates are flagged in `X-Outworld-Unknown-Names`, not removed.
- **The vault is the crown jewels.** It maps every surrogate back to a real value. It is encrypted at rest (AES-256-GCM with an authenticated, append-only record log), and Outworld refuses to run without a key. Protect the key like a password-manager master key. Anyone with both the vault and the key can reverse everything.
- **Not covered at all:** images, audio and files (refused on the masked route by default), fields outside the message text are only leak-checked, not rewritten, unless [What is masked](#what-is-masked) lists them (a finding there blocks the request instead of being replaced; tool names, schema property names and ids are never rewritten), provider-signed thinking blocks (passed through unchanged), and anything an agent sends around the gateway on its own. The specific things detection is known to miss are listed under [Known gaps](#known-gaps).
- Outworld logs never include real values or raw prompts: only routes, counts per entity type and timings.

### Known gaps

Every item here was found by tests, probes or review and is still open. Unless it says otherwise, the value goes to the provider unmasked. A dictionary entry closes most of the name items, and an NER sidecar closes some.

Names:

- Nicknames you have not listed, first names that are not in the built-in tables, and people who are described instead of named (`my sister's husband`).
- Names in non-Latin scripts (Cyrillic, Arabic, Chinese, Japanese, Korean, Greek, Hebrew, Devanagari) without a sidecar, names with case endings (`Bartosza Wiśniewskiego`), patronymic forms of address, and names damaged by OCR.
- A Latin-script name or an email address glued directly after CJK text, with no space: in `请联系John Smith` the name is sent.
- A family-name-first name whose family name is an ordinary word, when nothing in the sentence treats it as a person (`Ma Long's serve`, or `He Wei` at the start of a sentence).
- A single all-caps surname before a comma when the surname is a common word or shorter than four letters: in `KING, Stephen` only `Stephen` is masked.
- A lone common first name at the start of a sentence with no dictionary and no other name context.
- A double given name without a hyphen: `Lily Rose Depp said hi.` is missed on a fresh vault, and in `I met Lily Rose Depp yesterday.` the surname `Depp` is sent.
- Initials and initialisms: masked only when the full name appears in the same text. One that is also a common abbreviation or a state or country code (`OH`, `IT`, `US`) is masked only as a speaker label or list entry. Lone initials with no full name nearby (`LH will send notes`) are left as they are.
- Handles: a bare handle in prose needs a cue word right before it and a name shape (`ask kchen`); after a weak cue such as `per` a short handle is missed (`per jdoe`). A handle with no recognizable name in it (`@zorgblat`) is missed outside home directories and git trailers. A mention with extra characters attached (`@jsmithLane`) is left alone. Inside `TODO(...)`, a name that is also a month or a technology term (`TODO(ruby)`) is left alone. A lone name as a branch prefix after `git log` or `git diff`, or in a dotted fork reference, is missed.
- A handle is linked to its owner's stand-in only when the same request or your dictionary shows the full name. Otherwise the model sees the handle and the person as two different people.
- A person who shares a name with a listed company is masked only when the text treats the name as a person and nowhere as a company: `Ask Edward Jones about fees.` masks, `Notes from Edward Jones.` keeps.
- A full name directly in front of `Award`, `Memorial`, `Building` and similar words is treated as the name of the thing unless the sentence shows a person (`The John Smith Memorial Fund opened.` is kept). A listed multi-word place or term is never a person (`Monte Carlo called.` is kept).
- A technology word that is also a surname, used as a first name, keeps on a fresh vault: in `Docker Jones called`, only `Jones` is masked.
- An assistant name used alone is kept unless the sentence treats it as a person, even when the vault knows a person with that name: after `Claude Shannon` has been masked, `The Claude answer was wrong.` keeps `Claude`. Greetings to an assistant name keep too (`Hi Alexa,`, `Hi Bard,`).
- Masked when it should not be (the answer is still restored): once the vault knows a person whose first name is also a technology name, a lone use keeps only on strong technical evidence right next to it (a version, a technical noun right after it, an install-style verb right before it, a code span), so some technical sentences lose the word: `Ruby case naming` becomes a stand-in name when the vault knows a Ruby. A first name the vault knows is masked at the start of a product or theme name (`Rose Pine theme is installed.` when the vault knows a Rose). A greeting to a technology word that is also a surname masks it (`Hello Docker,`). Generic greeting targets that are not on the built-in list mask (`Hi Sales,`, `Hi Store,`), and so do surname-like words before a role word (`Hi Hall Manager,`, `Hi Park Team,`). Two names joined by an em dash with no spaces are read as one person, so the first word is masked even when it is not a name (a sign-off of `Thanks`, an em dash and `John` masks `Thanks`) and the second name gets a surname stand-in. Companies, places and products that look like people are masked too, about one in five on the blind corpus; `with` in front of a name plus a building word reads as a person (`the merger with Wilson Hall`).

Phones, IDs and secrets:

- A phone number with letters glued to it, even one the vault already knows: `x804-555-1234` and `804-555-1234xyz` are sent as they are.
- A label glued to a number in a national format (`Mobile07700 900123`), unlike glued labels on dashed numbers such as `TEL090-1234-5678`, which are masked.
- A continuation label on the next line: in `phone 804~555~1234` followed by a line `work 703~555~2468`, the second number is sent. The same pair on one line is masked.
- `faxes`, `text` or `reach` before a number separated by `*`: in `He faxes 804*555*1234 daily.` the number is sent. `call` and `phone` before it work.
- A detected phone number is never sent as it is, but when no stand-in can be shown to restore in its exact place, it is replaced by `[redacted phone]` and is not restored in the answer. This happens rarely: when digits or a percent escape are glued to the number or other phone numbers run into it, mostly for a number that already has many stored spellings (see Vault below).
- A phone number written in styled digits (mathematical bold `𝟖𝟎𝟒-𝟓𝟓𝟓-𝟏𝟐𝟑𝟒`, full-width `８０４-５５５-１２３４`) comes back in plain ASCII digits (`804-555-1234`). Superscript and subscript digits next to a number restore exactly.
- A card expiry and CVV with no card number nearby: in `exp 12/27 cvv 123` the CVV is stripped, but the expiry is treated as a date and shifted rather than stripped.
- Passwords made of plain words or with no symbols when no credential word is next to them (`Sommer2024`), and any password more than about 70 bytes from its cue word.
- ID numbers with no label and no checksum to recognize them by, labels in languages that are not listed, values longer than 32 characters after a label, and account numbers after ambiguous words (`cuenta`, `conto`, `compte`).
- Secrets inside base64 text, seed phrases with no label, AWS secret keys and 64-character hex keys with no label, and some less common crypto address formats.
- On very long single lines a cue word or label only counts within about 256 bytes of the value.
- A bare number that is both a valid ID and phone-shaped can be replaced as a phone number instead of stripped. It is still masked, but restored in the answer.
- Masked when it should not be: numbers that look like phone numbers, a number joined to a word by a dot (`hotel.804-555-1234` is masked, while `motel804-555-1234` is left alone), and documentation example keys (`AKIAIOSFODNN7EXAMPLE` is stripped like a real one).

Dates:

- Month and year only (`March 2024`), bare years, weekdays and relative dates (`next Tuesday`) are not shifted. Dates in non-Latin scripts or other calendars are not detected.
- Dates the model invents are shifted back like any other date.

Addresses:

- A number-first street with no postcode (`12 Via Roma`, `5 Hauptstraße`) when the street word is an ordinary word. A UK street and town with no postcode or country keeps its town (`14 Church Lane, Leeds` masks the street only). A US address with the state spelled out (`123 Calle Ocho, Miami, Florida 33135`) has only its street line replaced; the usual `City, ST 12345` form is replaced whole.
- Addresses written entirely in lowercase, Nordic postcodes with no street line, and countries outside the list above (those that are detected get a generic stand-in).
- A two-word town that is not in the built-in list loses its second word from the span when another capitalized word follows.
- A town followed directly by a personal name in a non-US address can get the whole request refused with 422 instead of masked (`Via Roma 10, 00184 Roma Zoë Müller-Öztürk`). A full stop after the town (`00184 Roma. Zoë Müller-Öztürk`) avoids it.
- Stand-ins do not always agree in grammar (an article may not match the gender of the new town), and a rare layout can produce an odd one: a town that is also a first name, directly followed by a person's name, may be replaced as part of the name.

Gateway:

- Numbers sent as JSON numbers are not inspected, in the request body or inside tool call arguments: `"seed": 8045551234` and `{"phone_ext": 8045551234}` as arguments go out as they are. JSON text inside a decoded opaque value is read as text, so `{"phone_ext": 2025550123}` there is caught.
- Opaque values (thinking `signature`, `redacted_thinking.data`, `encrypted_content`, `encrypted_index`) in binary form: a phone number with no cue word next to it is not checked (`a804-555-1234b` inside random bytes passes), and neither is a name with a date (`Elena Petrova born 03/14/1985`).
- Opaque values whose decoded bytes are 80 to 97% printable and also parse as protobuf: single-word names, IP addresses, street addresses and some secret shapes are not checked there. Cards, SSNs, phones and dates next to a cue word, email addresses and full names are.
- A Luhn-valid card number glued to a hex-looking run of 24 or more characters in text-like decoded content passes (`deadbeef4111111111111111`).
- CBOR data: a phone number under a phone key passes when the value is an indefinite-length string sent in chunks, when a tag sits between key and value, or when the key is an integer; a card number in a chunked string passes too. A phone number in a byte string instead of a text string sometimes passes.
- Email addresses in binary data with a one-character local part (`a@example.com`) or a top-level domain that is not on the built-in list.
- Protobuf data full of 16-digit numeric ids is refused with 422 when an id happens to pass the card checksum (it fails closed). Other random protobuf-like values are refused now and then too (a few per 1,000 in a measurement at 4 KB, fewer for smaller values), mostly from fragments that look like handles, phone numbers or assigned secrets.
- Tool definitions are masked like message text, so a value in a tool description that looks personal is replaced there too (in a test with coding-agent style tools, only a placeholder `noreply@example.com` address was). The memo keeps the result byte-stable across requests, but the model sees the stand-in in that description. A schema property name that looks personal blocks every request that carries that tool.
- Identifiers: a value with a known id prefix and a long random tail (`user_a8F3kL9qZ2xW7vB1nM4tY6uR0pE5sD3gH8jK2lQ9`) cannot be told from a secret and is sent under the id check, except under keys ending in `key`. A name joined in lowercase without a separator (`user_elenapetrova`) or a bare digit run (`8045551234123456789`) in an id field is checked but usually not recognized.
- `model` gets only the pattern, dictionary and vault checks: a model value such as `claude-opus-elena-petrova` or `claude:elena@petrova` is not recognized.
- With `policy.allow_images: true`, base64 PDFs, images and OpenAI `file_data` are checked only for their magic bytes and passed unread: image metadata, PDF text and bytes appended after the image data are not inspected.
- The OpenAI Responses API (`/v1/responses`) is not supported. Clients that default to it must be set to use chat completions.
- Detection time grows with text size: a 250 KB single message takes about 2 seconds and a 600 KB one about 10 seconds without a sidecar, on the test host. Text already sent in the same conversation is not detected again.

Streaming and restoring answers:

- Stripped values (secrets, IDs, card numbers) come back as `[redacted ...]`, by design. A value that was only partly recognized and repaired by the leak check can leave a mangled remainder, for example a URL with part of its path redacted.
- If the model writes a real acronym that happens to equal a stand-in initialism from the same vault, and the acronym was not in your request, it is rewritten.

Vault:

- A number keeps up to 32 stored phone spellings, and up to 256 in all when a spelling only restores in its context. Past that, a further unusual spelling of the same number that needs a new stored form is sent as `[redacted phone]` rather than as a stand-in that would restore.
- Text that was already sent is not re-masked when the vault later learns a new name (see [Stable transcripts](#stable-transcripts)).
- Stand-ins for new non-US addresses and new family-name-first names differ from v0.2.0 (see the 0.3.0 notes). Values already in a vault keep the stand-in they had. For roughly one conversation scope in 370, the date shift changed by one day to fix a bug that left dates without a year unshifted. A vault written by this version should not be opened by an older one (see [CHANGELOG.md](CHANGELOG.md)).

Not measured yet:

- Whether answers are as good with masking as without (needs a real model and a provider key; the harness supports it).
- A full interactive agent session through the gateway with a subscription login.

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

`vault.Backend` is a small interface: `Load()` returns the vault secret and existing records; `Update(fn)` must give `fn` exclusive access, pass it records other writers added since the last call, and durably persist what `fn` returns. The bundled file backend implements this with `flock` and an append-only encrypted log, so on Linux, macOS and the BSDs several Outworld processes can share one vault file. On other platforms (Windows among them) a vault is single-process: the open vault holds an exclusive `<vault>.lock` file until it is closed, and a second process fails to open it with an error that names that file. A lock file left over from a crash must be deleted by hand, once no other Outworld process is running. Other stores can plug in by implementing the same interface.

## Optional NER sidecars

Outworld works without a model. A sidecar adds recall for names, organizations and places that the patterns, the built-in name detector and your dictionary miss, at the cost of latency. Two small FastAPI services are bundled and speak the same `/detect` contract, so either goes under `ner.url`:

- [detectors/gliner](detectors/gliner/README.md): GLiNER, multilingual, covers people, organizations, locations and addresses.
- [detectors/privacy-filter](detectors/privacy-filter/README.md): OpenAI Privacy Filter, an open-weight PII token classifier. It has no organization or location labels.

Which one: in the gateway eval the two leak about the same amount. Privacy Filter is several times faster on CPU and better at first names, titles and nicknames, but because it has no organization label it tends to tag company names as people, so it over-masks more. GLiNER is slower and knows what an organization is. Privacy Filter can also return `account_number` and `secret` spans; map them in `ner.labels` (`account_number: financial`, `secret: secret`) to use them. That mapping is not part of the measurements above.

## License

Apache-2.0. See [LICENSE](LICENSE).
