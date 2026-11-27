# Outworld

Outworld is an open-source gateway for the OpenAI and Anthropic APIs. It replaces names, emails, phone numbers, addresses and dates with realistic stand-ins before a request reaches a cloud model, and restores the originals in the response, including streamed responses and tool calls.

You point your client, script or coding agent at Outworld instead of the provider. Everything else about the request stays the same.

## What it does

For each request, Outworld detects personal data in the prompt, the system prompt, tool definitions and tool results. It replaces each value with a stand-in from an encrypted vault, shifts dates by a secret offset, and strips secrets and IDs. It then re-scans the outgoing payload; if something it recognizes still cannot be replaced, the request is refused instead of sent. The masked request goes to the provider, and the answer is translated back to the real values before your client sees it.

## Features

- Realistic stand-ins (`John Smith` becomes `Kasimir Sandoval`), not `<PERSON_1>` placeholders, so the model still reads natural text.
- Answers restored, including streamed responses and tool-call arguments.
- Per-type policy: `keep`, `mask`, `shift`, `generalize` or `strip` for each entity type.
- Fails closed: a value it cannot replace blocks the request with a 422 that never echoes the value.
- Routing per request: `masked`, `local`, `passthrough` or `block`, by keyword, regex or detected entity type.
- Encrypted vault (AES-256-GCM) with stand-ins that stay the same across requests and conversations.
- Date shifting that keeps formats and intervals.
- Works with any OpenAI-compatible API, the Anthropic API and Claude Code, plus local servers such as Ollama, llama.cpp and vLLM.
- No model required; optional NER sidecars add recall.
- Dry run shows what would be sent and why it would block, without calling the provider.
- Logs carry routes, counts and timings, never real values.
- Single Go binary, a Docker image, and a Go library.

## Example

You send:

> Draft a reply to John Smith (john.smith@gmail.com) confirming our call on March 15, 2024. My verification code is 482913.

The provider sees:

> Draft a reply to Kasimir Sandoval (kasimir.sandoval@example.org) confirming our call on December 25, 2024. My verification code is [redacted code].

You get back a draft addressed to John Smith about March 15, 2024. The verification code was never sent and is not restored.

## Install and run

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

The `X-Outworld-Route` response header shows what happened. More in [Getting started](docs/getting-started.md).

## Use with Claude Code

Add an Anthropic upstream to `outworld.yaml` (see [Getting started](docs/getting-started.md#claude-code-and-other-anthropic-clients)), then:

```bash
export ANTHROPIC_BASE_URL=http://127.0.0.1:8787
claude
```

Subscription logins, gateway tokens and what to expect: [docs/integrations/claude-code.md](docs/integrations/claude-code.md).

## Docker

```bash
docker build -t outworld:local .
export OUTWORLD_VAULT_KEY=$(docker run --rm outworld:local keygen)
cp compose.example.yml compose.yml && docker compose up -d
```

The image is distroless and runs as a non-root user. Put the vault on the data volume: `vault: {path: /data/outworld.vault}`.

## Accuracy

On a blind corpus of 131 hand-written items that no detector was tuned against, 16.2% of labelled sensitive values reached the provider unchanged with a dictionary of known people, and 20.1% without one. Answers were restored correctly 99.3% of the time. Per-type numbers, over-masking, latency and how to reproduce them: [docs/accuracy.md](docs/accuracy.md).

## Limits

- Detection is best-effort. Unusual names, unlisted nicknames, people described rather than named, and names in non-Latin scripts can get through.
- Masking is not anonymization. The provider still sees the substance of the request, and context can identify someone even with every name replaced.
- Images, audio and files are refused on the masked route by default; route them `local` or `passthrough`.
- The vault maps every stand-in back to a real value. Protect its key like a password-manager master key.

Read the [threat model](docs/threat-model.md) and the [known gaps](docs/known-gaps.md) before you rely on it.

## Documentation

- [How it works](docs/how-it-works.md): overview and request flow diagram.
- [Getting started](docs/getting-started.md): install, Claude Code and Docker setup.
- [Configuration reference](docs/configuration.md): every config key.
- [Policy](docs/policy.md): entity actions, surrogates, detection layers, the dictionary.
- [Routing](docs/routing.md): routes, rules and the route header.
- [What is masked](docs/what-is-masked.md): every request field and how it is treated, response headers.
- [Endpoints](docs/endpoints.md): supported API endpoints, auth, stable transcripts.
- [CLI](docs/cli.md): `mask`, `unmask` and `dry-run`.
- [Blocked requests and dry runs](docs/blocked-requests.md): reading a 422 and previewing a request.
- [Accuracy](docs/accuracy.md): measured leak rates, over-masking and latency.
- [Threat model](docs/threat-model.md): what Outworld protects against and what it does not.
- [Known gaps](docs/known-gaps.md): everything detection is known to miss.
- [Go library](docs/library.md): using the engine directly, vault backends.
- [NER sidecars](docs/ner-sidecars.md): optional GLiNER and Privacy Filter models.
- Integrations: [Claude Code](docs/integrations/claude-code.md), [LiteLLM](docs/integrations/litellm.md), [Open WebUI](docs/integrations/open-webui.md).
- [Eval harness](eval/README.md), [Contributing](CONTRIBUTING.md), [Security](SECURITY.md).

## Project status

Version 0.3.0, an early release. The test suite has over 900 tests, including scale tests that keep every detector linear-time. Changes and upgrade notes are in [CHANGELOG.md](CHANGELOG.md).

## License

Apache-2.0. See [LICENSE](LICENSE).
