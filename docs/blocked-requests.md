# Blocked requests and dry runs

Outworld re-scans every outbound payload after masking. If something sensitive is still there and cannot be fixed, the request is refused with HTTP 422 and nothing is sent upstream. This page explains how to read that response and how to preview what the gateway would send.

## What the leak check does

1. Each string is masked once (the first pass). When two detected values overlap, the longer one is replaced. If the shorter one reaches past it, the part that sticks out is not left in clear: once the rounds below have settled, any such part that still holds at least two letters or four digits is masked as its own type (a person's remaining words each get a surrogate, strip types get a `[redacted ...]` placeholder, other types get a surrogate of that type for the fragment). These show up with `source` `residue` in a dry run. Dates are the exception: a date that loses to an overlapping value is left to the scan in step 2.
2. The masked text is scanned again by every detector and against every real value the vault knows.
3. Anything the second scan finds is repaired in place and the text is scanned again, up to four rounds:
   - a finding that sits in untouched text is masked like any other value;
   - a person or strip-type finding that partly overlaps a replacement (for example a surrogate first name followed by a real surname the first pass missed) has only its uncovered words masked. The surrogate that is already there is not changed.
     A word masked this way gets its own surrogate so the answer can be translated back, but it is not remembered as a name: later requests are not searched for it. For strip types (secrets, IDs, financial values) the uncovered part is replaced by a `[redacted ...]` placeholder, which is not restored in the answer.
   - findings of other types that overlap a replacement are not repaired and block.
4. Whatever is still flagged after that blocks the request. So does any finding in a field the gateway cannot rewrite: a JSON object key or a message `name`.
5. An opaque value (thinking `signature`, `redacted_thinking.data`, `encrypted_content`, `encrypted_index`) whose base64 text is longer than `max_opaque_bytes` (default 786432, that is 768 KiB) is not scanned at all. It blocks the request with a finding of type `oversize` and source `gateway:opaque_too_large`.

A request never goes upstream with a value the final scan flags.

## Reading a 422

```json
{
  "error": {
    "code": "outworld_leak_blocked",
    "type": "outworld_leak_blocked",
    "message": "outworld: blocked outbound payload: 2 sensitive value(s) would have left unmasked (types: email, person)",
    "help": "Nothing was sent upstream. Each finding names a value the leak check still saw after masking; previews are redacted. Resend with the header X-Outworld-Dry-Run: 1 to see the masked request without calling the upstream.",
    "findings": [
      {
        "type": "email",
        "source": "regex:email",
        "message_index": 0,
        "field": "messages[0].name",
        "start": 0,
        "end": 36,
        "preview": "[***]",
        "reason": "unmaskable_field",
        "hint": "This value sits in a field the gateway cannot rewrite (a JSON object key or a message name). Move it into message text or a JSON value, or add it to policy.allow if it is not sensitive."
      }
    ]
  }
}
```

The Anthropic endpoint returns the same `findings` and `help` inside its usual `{"type": "error", "error": {"type": "outworld_leak_blocked", "message": ...}}` envelope. `code`, `type` and `message` are unchanged from earlier versions; `findings` and `help` are additions.

| Field | Meaning |
|---|---|
| `type` | Entity type: `person`, `email`, `phone`, `secret` and so on, or `oversize` for an opaque value too large to scan. |
| `source` | The detector that flagged it: `regex:<pattern>`, `names:<rule>`, `dictionary`, `ner`, `vault` (a real value the vault already knows), `<detector>:error` when a detector failed, or `gateway:opaque_too_large` for an opaque value longer than `max_opaque_bytes`. |
| `message_index` | Index into `messages`, or `-1` for text outside a message (`system`, `prompt`). |
| `field` | Where the text lives, for example `messages[2].content`, `messages[0].content[1]`, `messages[3].tool_calls[0].function.arguments.#1`, `system`, `metadata.#0`, `tools[2].input_schema.properties.#0.description`. Object keys are named by their position in sorted order (`#0`, `#1`), never by the key text, because the key can be the sensitive value; `(key)` at the end means the finding is in the key itself, and its `preview` is only `[***]`. Fields the gateway does not know start with `request.` and are named by position from the top of the body (`request.#4`). |
| `start`, `end` | Character offsets (Unicode code points) of the finding inside the masked text of that field. `-1` when there is no position. Use a dry run to see that text. |
| `preview` | Up to 32 bytes of the already-masked text on each side with the finding replaced by a mark. The mark is always `[***]`, for every entity type: no character of a flagged value is shown. Other findings inside the window are shown as `[***]` too. For a field the gateway only checks and never rewrites (object keys, names, ids and other fields it does not know), the preview is just `[***]`, so no part of that value is shown. |
| `reason` | `unmaskable_field` (object key, message name, or an opaque value too large to scan), `overlaps_replacement` (the finding is tangled with a replacement and could not be repaired), `rounds_exhausted` (new findings kept appearing), `detector_error`. |
| `hint` | One line on what to change. |

Each value is listed once. When two layers flag the same span of the same field with the same type, the finding keeps the detector's `source` rather than `vault`. A finding that lies inside another one in that field is folded into it only when both have the same type, or for a name inside an email address. Everything else stays separate, including findings without a position and a secret inside a URL, so the count in `message` is one per distinct value. Dry runs list findings the same way.

The response never contains more of your text than you sent, and previews are not written to the log. The log line for a blocked request carries only counts per entity type.

### What to do

| Situation | Fix |
|---|---|
| A person is flagged | Add the person to the dictionary (`policy.dictionary`) so every form of the name is masked in the first pass, or run an NER sidecar (`ner.url`). |
| The flagged text is not sensitive | Add the exact text to `policy.allow`. |
| You do not want a type masked at all | Set `policy.entities.<type>: keep`. |
| `unmaskable_field` | Move the value out of the object key or message `name` into message text or a JSON value. |
| `detector_error` | Check the NER sidecar, or set `ner.optional: true` if requests should continue without it. |
| `rounds_exhausted` | Send less text per message, or add the names involved to the dictionary. |
| `oversize` from `gateway:opaque_too_large` | Raise `max_opaque_bytes` in the config if thinking signatures or `encrypted_content` values that large are expected. The value is never scanned or forwarded while it is over the limit. |

## Dry run

A dry run does everything a real request does (routing, masking, leak check) and stops before the upstream call. The answer is a report instead of a model response.

### Over HTTP

Add `X-Outworld-Dry-Run: 1` to a normal request on `/v1/chat/completions`, `/v1/completions`, `/v1/messages` or `/v1/messages/count_tokens`:

```bash
curl -s http://127.0.0.1:8787/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -H 'X-Outworld-Dry-Run: 1' \
  -d '{"model":"m","messages":[{"role":"user","content":"Marisol Quenby (m.quenby@halvorsen-bio.example, 415-555-0134) starts on 2026-04-17. SSN 078-05-1120."}]}'
```

```json
{
  "outworld_dry_run": true,
  "api": "openai",
  "route": "masked",
  "route_reason": "default",
  "upstream": "cloud",
  "would_send": true,
  "request": {
    "messages": [
      {
        "content": "Helga Wainwright (m.wainwright@example.org, 394-555-0146) starts on 2026-10-03. SSN [redacted id].",
        "role": "user"
      }
    ],
    "model": "m"
  },
  "summary": [
    {"type": "date", "source": "date", "action": "shift", "count": 1},
    {"type": "email", "source": "regex:email", "action": "mask", "count": 1},
    {"type": "gov_id", "source": "regex:labelled_id_intl", "action": "strip", "count": 1},
    {"type": "person", "source": "dictionary", "action": "mask", "count": 1},
    {"type": "phone", "source": "regex:phone_nanp", "action": "mask", "count": 1}
  ],
  "replacements": [
    {"type": "person", "source": "dictionary", "action": "mask", "replacement": "Helga Wainwright", "message_index": 0, "field": "messages[0].content", "start": 0, "end": 16}
  ],
  "findings": [],
  "memo_hits": 0
}
```

(`replacements` is shortened here; it has one entry per replacement.)

- `request` is the body that would be sent, after masking and after the usual cleanup (`user` removed, upstream model override applied).
- `replacements` lists what was put in, never what was taken out: the surrogate, shifted date or `[redacted ...]` placeholder, its type, the detector that found the original, and where it sits in the masked text. `source` is `residue` for a word masked by the repair step described above, and `vault` for a known value masked by the leak check.
- `would_send` is `false` and `blocked` holds the error code when the real request would be refused. `findings` then has the same entries a 422 would carry. The status is still 200, and `X-Outworld-Route` is `blocked-leak`.
- On the `local` and `passthrough` routes the report shows the request unchanged, because those routes do not mask. On the `block` route there is no `request`.
- `memo_hits` counts strings that were served from the transcript memo. Those are sent exactly as before and have no entries in `replacements`.

The header is accepted on any request and needs no configuration. It is safe to leave available: the report goes only to the caller, who already has the original text, it sits behind the same gateway token as every other request, and no upstream is contacted. A dry run is not read-only. It writes to the vault and to the transcript memo exactly as the real request would: new surrogates are assigned and stored in the vault file, and the masked form of each string is kept in the in-memory memo, so the real request that follows sends the same bytes. A blocked dry run stores nothing in the memo for the blocked string, but surrogates assigned before the block stay in the vault. Run it against a copy of the vault if you need a preview that leaves no trace. The log line for a dry run has `"dry_run": true` and the usual counts, no text.

### From the command line

```bash
outworld dry-run --config outworld.yaml request.json     # a request body
outworld dry-run --config outworld.yaml notes.txt        # plain text, treated as one user message
echo 'Call Marisol at 415-555-0134.' | outworld dry-run  # stdin
```

Flags: `--config`, `--scope` (date-shift scope), `--api openai|anthropic` (defaults to whichever API has a masked upstream). The report is printed as indented JSON. The exit code is `3` when the request would be blocked by the leak check, the same as `outworld mask`. The command works with a config that has no `upstreams`; the route is then always `masked`.

## Library

`Session.Violations()` returns the same findings (`outworld.Violation`). Call `Session.SetLocation(index, field)` before each `Mask` or `Check` to label them, and `Session.SetTrace(true)` to collect `Session.Replacements()`.
