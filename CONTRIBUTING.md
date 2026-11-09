# Contributing to Outworld

Thanks for helping. Outworld handles people's private data, so correctness and honesty about limits matter more than features.

## Ground rules

- Fail closed. If a detector errors, a key is missing, or the outbound payload still contains something sensitive, refuse the request rather than send it.
- Never log real values, raw prompts or model output. Log routes, counts and timings only.
- No new runtime dependencies without a strong reason. The core is standard library plus a YAML parser.
- Bundled data (name lists, street names) must be written for this project. No licensed or scraped datasets.
- Keep code self-explanatory. Comments only for a non-obvious "why".

## Development

```bash
task test        # go test ./...
task race        # go test -race ./...
task lint        # gofmt check + go vet
task build       # ./bin/outworld
```

Requires Go 1.27+. The GLiNER sidecar needs Python 3.12 and is optional.

## Tests

Every detector, surrogate shape and policy action needs tests. Privacy bugs usually hide in edge cases, so include adversarial ones: names split across stream chunks, possessives, names inside URLs and emails, case variants, overlapping matches, and values that look sensitive but are not (zip codes, UUIDs, commit hashes, version numbers).

Gateway changes need an integration test against the in-process fake upstream that asserts no real value reaches it and that the response is rehydrated, for both streaming and non-streaming.

## Reporting a privacy or security issue

Please open a private security advisory on the repository instead of a public issue if you find a way to make Outworld send a value it should have masked, or to read the vault without the key.

## License

By contributing you agree that your contributions are licensed under Apache-2.0.
