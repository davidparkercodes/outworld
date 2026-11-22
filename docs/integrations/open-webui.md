# Open WebUI through Outworld

Open WebUI talks to any OpenAI-compatible endpoint. Point it at Outworld and every chat is masked on its way to the provider and translated back on its way to the user.

## Add the connection

In Open WebUI, as an admin:

1. Open **Admin Settings > Connections** and find **Manage OpenAI API Connections**.
2. Click **Add Connection**.
3. **URL:** `http://127.0.0.1:8787/v1` (or `http://outworld:8787/v1` in the Compose setup below; if Open WebUI runs in Docker and Outworld on the host, use `http://host.docker.internal:8787/v1`).
4. **API Key:** the value of Outworld's `OUTWORLD_TOKEN`. If Outworld runs without a token, leave it blank or use a placeholder.
5. Click **Verify Connection**, then **Save**.

Open WebUI fills its model picker by calling `/models` on the connection. Outworld answers `GET /v1/models` by passing the request to its default upstream, so the picker shows your provider's models.

## Docker Compose

Outworld sits next to Open WebUI on the Compose network. It listens on all interfaces inside that network, but its port is not published, so only Open WebUI can reach it.

```yaml
services:
  outworld:
    build: .                      # the Outworld repo; or image: outworld:local
    environment:
      OUTWORLD_CONFIG: /config/outworld.yaml
      OUTWORLD_LISTEN: 0.0.0.0:8787
      OUTWORLD_VAULT_KEY: ${OUTWORLD_VAULT_KEY:?generate one with: outworld keygen}
      OUTWORLD_TOKEN: ${OUTWORLD_TOKEN:?}
      CLOUD_API_KEY: ${CLOUD_API_KEY:?}
    volumes:
      - ./outworld.yaml:/config/outworld.yaml:ro
      - ./people.csv:/config/people.csv:ro
      - outworld-data:/data
    restart: unless-stopped

  open-webui:
    image: ghcr.io/open-webui/open-webui:main
    ports:
      - "127.0.0.1:3000:8080"
    environment:
      OPENAI_API_BASE_URL: http://outworld:8787/v1
      OPENAI_API_KEY: ${OUTWORLD_TOKEN}
      ENABLE_OLLAMA_API: "false"
    volumes:
      - open-webui:/app/backend/data
    depends_on:
      - outworld
    restart: unless-stopped

volumes:
  outworld-data:
  open-webui:
```

`outworld.yaml`:

```yaml
vault:
  path: /data/outworld.vault      # the config directory is mounted read-only

upstreams:
  cloud:
    base_url: https://api.example.com/v1
    api_key_env: CLOUD_API_KEY

policy:
  default_route: masked
  masked_upstream: cloud
  dictionary: ./people.csv
```

Notes:

- `OPENAI_API_BASE_URL` and `OPENAI_API_KEY` are read on first launch and then stored in Open WebUI's own database. To change them later, edit the connection in Admin Settings.
- `ENABLE_OLLAMA_API: "false"` stops Open WebUI from talking to an Ollama server directly, which would bypass Outworld. Local models are reached through Outworld instead (next section).
- The provider key exists only in the Outworld container. Open WebUI holds the gateway token, which Outworld never forwards.

## Local models for sensitive chats

Add Ollama as Outworld's local upstream and a rule that sends matching chats there unmasked. Nothing in those chats leaves your hardware.

```yaml
upstreams:
  cloud:
    base_url: https://api.example.com/v1
    api_key_env: CLOUD_API_KEY
  local:
    base_url: http://ollama:11434/v1
    model: llama3.1:8b

policy:
  default_route: masked
  masked_upstream: cloud
  local_upstream: local
  rules:
    - {name: health-and-legal, match_any: ["diagnosis", "lawsuit", "prescription"], route: local}
```

Add an `ollama` service (`image: ollama/ollama`) to the Compose file, without publishing its port.

- Set `model` on the local upstream. The user picked a cloud model in the UI, and Ollama needs the name of a model it has.
- `match_any` matches whole words anywhere in the conversation, so once a chat matches, its later turns stay local.
- Rules can also match on `match_regex` or on detected entity types, for example `{entities: ["financial"], route: block}`.
- The user is not told in the UI which route answered. The Outworld log shows `route` and `reason` (`rule:health-and-legal`) for every request.

## What to expect

- **Users see real names.** Answers are rehydrated before they reach Open WebUI, so the chat reads normally.
- **The provider sees stand-ins.** Names, emails, phones and addresses become consistent surrogates, dates are shifted, secrets are stripped and not restored.
- **Streaming works.** Tokens appear as they arrive. Outworld holds back a few characters at a time when the tail of a chunk could be the start of a surrogate.
- **Model list** comes from the default upstream through `GET /v1/models`.
- **Blocked requests show as errors in the chat.** A leak-check block is HTTP 422, a `block` rule is HTTP 403. The message contains no personal values.
- **One vault for all users.** The same real name maps to the same surrogate for everyone on the instance.

## Limits

- **Open WebUI's stored history holds real text.** Outworld protects the hop to the provider, not the UI's database. Chats, titles and uploaded files in the `open-webui` volume are unmasked. Protect and back up that volume accordingly.
- **Image uploads are refused on the masked route** (HTTP 422, `outworld_unsupported_content`), because images cannot be masked. Use a local route for vision models, or set `policy.allow_images: true` to send images unmasked.
- **Chat endpoints only.** Outworld serves `/v1/chat/completions`, `/v1/completions` and `/v1/models`. Embeddings, image generation, speech and transcription are not proxied. Keep those features on local engines or give them a separate connection, and remember that a separate connection is not masked.
- **Extra connections bypass Outworld.** Any other OpenAI or Ollama connection added in Admin Settings goes straight to that server. Remove them if every chat must be masked.
- **Web search and tools run in Open WebUI.** Queries the UI sends to a search engine or tool server do not pass through Outworld.
- **Detection is best-effort.** Add a dictionary of the people who matter, and read the threat model in the [README](../../README.md).
