# Outworld with LiteLLM

LiteLLM Proxy and Outworld both speak the OpenAI API, so they stack in either order. Which one goes in front decides who sees real text.

| Placement | Order | LiteLLM sees | Use it when |
|---|---|---|---|
| A | client, LiteLLM, Outworld, provider | real text | you already run LiteLLM for keys, budgets and teams, and want masking for some or all models |
| B | client, Outworld, LiteLLM, providers | surrogates only | you want one privacy layer in front of many providers, and nothing downstream should hold real text |

In placement A, LiteLLM's own request logs, spend logs and logging callbacks receive the real prompt. If those must not hold personal data, use placement B.

## Placement A: LiteLLM in front

Outworld is an OpenAI-compatible `api_base` for LiteLLM. Outworld config:

```yaml
upstreams:
  cloud:
    base_url: https://api.example.com/v1
    api_key_env: CLOUD_API_KEY

policy:
  default_route: masked
  masked_upstream: cloud
  dictionary: ./people.csv
```

LiteLLM `config.yaml`:

```yaml
model_list:
  - model_name: private-chat
    litellm_params:
      model: openai/your-model
      api_base: http://127.0.0.1:8787/v1
      api_key: os.environ/OUTWORLD_TOKEN
```

```bash
litellm --config config.yaml
```

- The name after `openai/` is the `model` Outworld receives and passes to its upstream (unless that upstream sets `model`).
- `api_key` reaches Outworld as `Authorization: Bearer`. It must equal Outworld's `OUTWORLD_TOKEN`. If Outworld runs without a token, use any placeholder string.
- The provider key lives in Outworld (`api_key_env`), not in LiteLLM.

### Choosing a route per model

`X-Outworld-Route` can pick any route only when `policy.allow_route_header` is true; when it is unset (the default) the header may only make routing stricter (`block`, then `local`, then `masked`, then `passthrough`) and is refused with 403 otherwise. To fix a route for one LiteLLM model, set the header in `litellm_params`:

```yaml
model_list:
  - model_name: private-chat
    litellm_params:
      model: openai/your-model
      api_base: http://127.0.0.1:8787/v1
      api_key: os.environ/OUTWORLD_TOKEN
  - model_name: private-local
    litellm_params:
      model: openai/your-model
      api_base: http://127.0.0.1:8787/v1
      api_key: os.environ/OUTWORLD_TOKEN
      extra_headers: {"X-Outworld-Route": "local"}
```

Clients then choose by model name. `local` needs `policy.local_upstream` in the Outworld config.

### Choosing a route per request

LiteLLM does not forward client headers to the provider by default. Turn it on for the models that point at Outworld:

```yaml
litellm_settings:
  model_group_settings:
    forward_client_headers_to_llm_api:
      - private-chat
```

With this setting LiteLLM forwards client headers that start with `x-`, so a client can send `X-Outworld-Route: local` (or `masked`, `passthrough:<upstream>`, `block`) on a single request. With `policy.allow_route_header: true`, anyone who can call that model can then pick the route, including `passthrough`. Leave it unset so clients can only tighten routing, or set it to `false` to ignore the header.

## Placement B: Outworld in front

LiteLLM is Outworld's upstream. Clients point at Outworld.

```yaml
upstreams:
  litellm:
    base_url: http://127.0.0.1:4000
    api_key_env: LITELLM_API_KEY

policy:
  default_route: masked
  masked_upstream: litellm
  dictionary: ./people.csv
```

- `LITELLM_API_KEY` is a LiteLLM key. Outworld sends it as `Authorization: Bearer`. Clients never hold it; they present Outworld's `OUTWORLD_TOKEN` instead, which is not forwarded.
- The client's `model` passes through, so LiteLLM's `model_name` values are what clients request. `GET /v1/models` on Outworld returns LiteLLM's model list.
- Budgets and spend tracking in LiteLLM then apply to the single key Outworld uses, not to individual end users. Outworld drops the OpenAI `user` field on the masked route.

## What to expect

- Streaming works in both placements. Outworld rehydrates SSE chunks and tool-call arguments before they reach the next hop.
- Clients get real names back. The provider gets stand-ins.
- A request Outworld blocks returns HTTP 422 with error type `outworld_leak_blocked`. A `block` route returns 403. In placement A the client sees these as provider errors relayed by LiteLLM.
- Outworld's log has one line per request with the route and counts per entity type, never values. This is the reliable place to confirm masking in placement A.

## Limits

- **Response headers in placement A.** Outworld sets `X-Outworld-Route`, `X-Outworld-Route-Reason`, `X-Outworld-Masked` and `X-Outworld-Unknown-Names` on its responses. Do not rely on LiteLLM returning them to the client unchanged; read the Outworld log.
- **Retries and fallbacks.** If LiteLLM falls back from an Outworld-backed model to a model that calls a provider directly, that request is not masked. Keep fallbacks for private models inside the set of Outworld-backed models.
- **Caching in placement A.** A LiteLLM response cache stores real text, because it sits on the client side of Outworld.
- **Endpoints.** Outworld serves `/v1/chat/completions`, `/v1/completions` and `/v1/models` on the OpenAI side. Embeddings, images and audio are not proxied; route those models past Outworld or keep them local.
- **Images and files** in chat messages are refused on the masked route (HTTP 422) unless `policy.allow_images: true`.

## Relation to LiteLLM's Presidio guardrail

LiteLLM ships its own PII guardrail based on Microsoft Presidio, which masks detected entities with typed placeholders such as `<PERSON>` and can restore them in the output; Outworld differs in that it substitutes realistic, consistent surrogates, re-scans the outbound payload and blocks the request if something remains, and restores streamed chunks and tool-call arguments. Use one of them per route. If both run, the second one only sees the first one's output.
