# Routing

Each request takes exactly one route:

- `masked`: transform, leak-check, send to `policy.masked_upstream`, rehydrate the answer.
- `local`: send raw to `policy.local_upstream` (Ollama, llama.cpp, vLLM on your hardware).
- `passthrough`: send raw to a named upstream. Intended for confidential-compute / TEE endpoints you trust with raw data.
- `block`: refuse with HTTP 403.

Order of evaluation: any `block` rule that matches wins; then the first matching rule, else `default_route`; then the `X-Outworld-Route` request header, as described below. Rules match by whole-word keywords (`match_any`), regexes (`match_regex`) or detected entity types (`entities`). They see exactly the texts the masked route rewrites or leak-checks (see [What is masked](what-is-masked.md)), never image data.

The route header has three settings:

- `allow_route_header` unset (the default): the header may only make the decision stricter. Routes rank `block`, then `local`, then `masked`, then `passthrough`. A header that ranks above the decided route replaces it (so `local` on a masked request sends it raw to `policy.local_upstream`); one that ranks the same keeps the decided route and its upstream; one that ranks lower is refused with 403 `outworld_route_error`. The header may name an upstream only when it is the one that route would use anyway (`local:local`); any other upstream is refused with 403. So `masked` cannot move a request that a rule or `default_route` sent `local` to the cloud.
- `allow_route_header: true`: the header overrides the rules and `default_route` and may choose any route and any configured upstream, e.g. `local` or `passthrough:tee`. Only a matching `block` rule still wins. This means a client can override `default_route: block` and send raw text to any upstream in your config; anyone holding the gateway token can skip masking. Use it only when every client is trusted.
- `allow_route_header: false`: the header is ignored.

An unknown route name in the header is refused with 400 `outworld_route_error`, and so is a route or upstream the header asks for that has no matching upstream for that API. 502 `outworld_route_error` is kept for configuration problems in the rules or `default_route` and for detector failures.

Changed in this release: the default used to let the header choose any route, including `passthrough`, ahead of the rules. Configs that rely on the header to loosen routing (`passthrough`, or `masked` over a `local` rule) must now set `allow_route_header: true`.
