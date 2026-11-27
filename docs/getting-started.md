# Quickstart

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

## Claude Code and other Anthropic clients

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

See [docs/integrations/claude-code.md](integrations/claude-code.md) for subscription logins, gateway tokens and what to expect. Guides for [LiteLLM](integrations/litellm.md) and [Open WebUI](integrations/open-webui.md) are next to it.

## Docker

```bash
docker build -t outworld:local .
export OUTWORLD_VAULT_KEY=$(docker run --rm outworld:local keygen)
cp compose.example.yml compose.yml
docker compose up -d
```

The image is distroless, static and runs as a non-root user. The config directory is mounted read-only, so point the vault at the data volume in `outworld.yaml`: `vault: {path: /data/outworld.vault}`.
