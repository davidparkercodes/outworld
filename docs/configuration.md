# Configuration reference

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
