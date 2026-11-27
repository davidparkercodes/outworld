# CLI: mask and unmask

For testing and batch pipelines, `mask` and `unmask` read stdin and write stdout using the same vault and policy:

```bash
outworld mask < notes.txt > notes.masked.txt          # exit code 3 if the leak check blocks
# ... send notes.masked.txt anywhere ...
outworld unmask < answer.txt > answer.real.txt
outworld mask --scope project-a < notes.txt           # per-scope date offset
outworld dry-run request.json                         # what would be sent, and why it would block (exit code 3)
```

`dry-run` takes a chat request body (file or stdin, `--api openai|anthropic`) and prints the masked request, every finding and the route, without calling any upstream. The same report is available over HTTP by adding `X-Outworld-Dry-Run: 1` to a normal request. A dry run still writes the surrogates it assigns to the vault. See [blocked requests and dry runs](blocked-requests.md).
