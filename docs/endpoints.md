# Endpoints

OpenAI format (upstreams with `api: openai`, the default):

- `POST /v1/chat/completions` streaming and non-streaming, tools and tool calls (arguments are masked and rehydrated as JSON). Tool definitions, `metadata`, `prediction`, `stop` and `response_format` schemas are masked as listed under [What is masked](what-is-masked.md).
- The query string is forwarded unchanged.
- `POST /v1/completions` legacy completions.

Anthropic format (upstreams with `api: anthropic`, selected by `policy.anthropic.*`):

- `POST /v1/messages` streaming and non-streaming. `system`, text blocks, `tool_use` inputs, `tool_result` content, `stop_sequences` and tool definitions are masked (details under [What is masked](what-is-masked.md)); text and tool inputs in the answer are rehydrated, including across `text_delta` and `input_json_delta` events.
- `POST /v1/messages/count_tokens` masked the same way, so counts match what is sent.
- Every `anthropic-*` request header, the query string and unknown body fields are forwarded unchanged; upstream status, headers and error bodies come back unchanged.

Both:

- `GET /v1/models` passthrough to the default upstream.
- `GET /healthz`.

There is no translation between the two formats: a request goes to an upstream that speaks the format it arrived in.

Set `OUTWORLD_TOKEN` (or `auth.token_env`) to require a token from clients, sent as `Authorization: Bearer <token>`, `x-api-key: <token>` or `X-Outworld-Token: <token>`. When set, requests without it are rejected. The gateway token is never forwarded upstream. Each upstream uses its own `api_key_env`, unless it sets `forward_auth: true`, in which case the client's own `Authorization` and `x-api-key` headers are passed through (for clients that hold the provider login themselves).

Messages containing images, audio or files are refused on the `masked` route (they cannot be masked) unless you set `policy.allow_images: true`. Use `local` or `passthrough` for those.

## Stable transcripts

Agents and chat clients resend the whole conversation on every request. Providers cache on that prefix, and current Claude models reject a conversation whose earlier turns changed. So Outworld keeps a transcript memo: every text it has masked, and every model answer it has translated back, maps to the exact bytes the provider saw. When the client sends that text again, the provider receives the same bytes as before, and the text is not detected again, which is also what keeps long agent sessions fast.

The memo lives in memory (`memo.max_bytes`, default 64 MiB, least recently used entries dropped first) and holds masked text only. After a restart the first request of an ongoing conversation is masked from scratch, which can cost one prompt-cache miss. A consequence worth knowing: text that was already sent is not re-masked when the vault later learns a new name. It left in that form once; changing it later would not unsend it.

Provider-signed `thinking` blocks pass through untouched in both directions, so visible thinking shows stand-in names.
