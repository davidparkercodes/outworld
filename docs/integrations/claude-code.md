# Claude Code through Outworld

Outworld serves the Anthropic Messages API (`POST /v1/messages`, streaming and non-streaming, and `POST /v1/messages/count_tokens`), so Claude Code can use it as its gateway. Prompts, the system prompt and tool results are masked before they leave your machine. Answers and tool-call arguments are translated back before Claude Code sees them.

Any other client built on an Anthropic SDK works the same way: set its `base_url` to `http://127.0.0.1:8787`.

## Gateway config

Pick one of two ways to authenticate to Anthropic.

**A. The gateway holds an API key.** Requests are billed to that key.

```yaml
upstreams:
  anthropic:
    api: anthropic
    base_url: https://api.anthropic.com/v1
    api_key_env: ANTHROPIC_API_KEY

policy:
  default_route: masked
  anthropic:
    masked_upstream: anthropic
  dictionary: ./people.csv

memo:
  enabled: true
  max_bytes: 67108864
```

**B. The client's own credential is forwarded.** Use this with a claude.ai subscription login. Replace `api_key_env` with `forward_auth`:

```yaml
upstreams:
  anthropic:
    api: anthropic
    base_url: https://api.anthropic.com/v1
    forward_auth: true
```

With `forward_auth: true`, Outworld passes the client's `Authorization` and `x-api-key` headers to the upstream. In both modes it forwards every `anthropic-*` header (including `anthropic-beta` and `anthropic-version`) and the request query string (`?beta=true`) unchanged.

Start the gateway:

```bash
export OUTWORLD_VAULT_KEY=$(outworld keygen)   # keep this; it unlocks the vault
outworld serve --config outworld.yaml
```

## Claude Code config

Mode A. Claude Code needs a credential variable, otherwise it falls back to its own login. Give it the gateway token:

```bash
export ANTHROPIC_BASE_URL=http://127.0.0.1:8787
export ANTHROPIC_AUTH_TOKEN="$OUTWORLD_TOKEN"
```

If you run Outworld without `OUTWORLD_TOKEN`, any non-empty placeholder works, because the gateway ignores the client credential and uses its own key.

Mode B. Set only the base URL. Your saved claude.ai login stays the active credential and is forwarded:

```bash
export ANTHROPIC_BASE_URL=http://127.0.0.1:8787
```

To persist either setup, put the same variables in the `env` block of `~/.claude/settings.json`. Run `/status` in Claude Code and check that the `Anthropic base URL` line shows the gateway address.

### Gateway token

When `OUTWORLD_TOKEN` is set on the gateway, every request must present it in one of three headers: `Authorization: Bearer <token>`, `x-api-key: <token>`, or `X-Outworld-Token: <token>`. The gateway token is never forwarded upstream.

In mode B the `Authorization` header already carries the provider credential, so send the gateway token in its own header:

```bash
export ANTHROPIC_CUSTOM_HEADERS="X-Outworld-Token: $OUTWORLD_TOKEN"
```

## Verify

Send a request with a made-up name and look at the response headers (mode A shown; add `-H "Authorization: Bearer $OUTWORLD_TOKEN"` if the gateway has a token):

```bash
curl -si http://127.0.0.1:8787/v1/messages \
  -H 'content-type: application/json' \
  -H 'anthropic-version: 2023-06-01' \
  -d '{"model":"claude-sonnet-4-6","max_tokens":64,"messages":[{"role":"user","content":"Reply with one short sentence greeting Dana Whitfield, phone (804) 555-1234."}]}'
```

Expected:

```
X-Outworld-Route: masked
X-Outworld-Route-Reason: default
X-Outworld-Masked: 2
```

The reply greets Dana Whitfield, although the provider only saw a stand-in name and a fictional phone number. The gateway log has one line per request with counts only, never values (timestamp and duration omitted here):

```json
{"level":"INFO","msg":"request","path":"/messages","route":"masked","reason":"default","upstream":"anthropic","status":200,"stream":false,"masked":2,"stripped":0,"shifted":0,"generalized":0,"entities":{"person":1,"phone":1},"unknown_names":0}
```

## What to expect

- **Masked:** user messages, the system prompt, tool results (file contents, command output), and tool-call inputs replayed in the history.
- **Translated back:** answer text and tool-call arguments, in streams too. Tools run on your machine with the real values.
- **Not masked:** tool definitions (`tools`). Claude Code's attribution block at the start of the system prompt is passed through untouched and stays in its own array entry.
- **Thinking blocks pass through unmodified in both directions.** They are signed by the provider and cannot be rewritten. Visible thinking summaries therefore show surrogate names, not real ones.
- **Prompt caching keeps working.** A transcript memo re-sends every already-sent turn byte-identical, which the prompt cache and the API's preserved-thinking prefix check both depend on.
- **Secrets are stripped.** Keys, tokens and passwords pasted into the conversation or returned by a tool are removed before the request leaves and are not restored.
- **Token counts.** `count_tokens` requests are masked the same way, so `/context` keeps using real token counts instead of an estimate.
- **Blocked requests.** If the re-scan still finds something it cannot transform, the request is not sent: HTTP 422, `X-Outworld-Route: blocked-leak`, error type `outworld_leak_blocked`.
- **`metadata.user_id` is dropped** unless you set `policy.keep_metadata: true`.

## Tuning for coding agents

Code is full of capitalized words that are not people. Outworld's built-in name detector already leaves common tech names alone (Claude, Ruby, Julia, Jenkins and similar). For anything else that gets masked by mistake, add it to the never-mask list:

```yaml
policy:
  allow: ["Atlas", "Morgan Stanley"]
```

`policy.name_heuristics: false` turns the built-in name detector off entirely. Then only your dictionary, the pattern detectors and the optional NER sidecar find people.

Recommended starting point: no NER sidecar, plus a dictionary (`policy.dictionary`) of the names that matter to you: colleagues, customers, family. Detection runs on every new piece of text, so with the sidecar (`ner.url`) large tool results add latency. Each distinct text is only processed once, thanks to the memo.

To send one session down another route, add the route header. Without further config it may only make routing stricter (`block`, then `local`, then `masked`, then `passthrough`), so `local` works when `policy.anthropic.local_upstream` is set; to loosen routing, for example to `passthrough`, set `policy.allow_route_header: true` and a matching upstream under `policy.anthropic`:

```bash
export ANTHROPIC_CUSTOM_HEADERS="X-Outworld-Route: passthrough"
```

## Limits

- **Outworld only sees what goes through it.** Anything the agent sends around the proxy is out of scope: a tool that uploads a file itself, a `curl` or `git push` it runs, an MCP server that calls its own backend.
- **Server-side tools see surrogates.** A web search executed by the provider searches for the stand-in name, not the real one.
- **Images and PDFs are refused on the masked route** with HTTP 422 (`outworld_unsupported_content`) unless you set `policy.allow_images: true`. With that setting they are sent unmasked.
- **The memo is in memory.** It is lost when the gateway restarts. The next request of a running conversation may then miss the prompt cache once, or have its earlier thinking dropped by Claude Code after the API rejects the old signatures. The conversation continues.
- **Detection is best-effort.** Unusual names, people described rather than named, and identifiers inside code can slip through. See the [threat model](../threat-model.md) and [known gaps](../known-gaps.md).
- **The provider still sees your code and the substance of your prompts.** Outworld hides who and what is named in them, not what they are about.
