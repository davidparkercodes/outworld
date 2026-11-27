# Security policy

## Reporting a vulnerability

Report security problems privately through GitHub: open the repository's **Security** tab and choose **Report a vulnerability**. Please do not open a public issue for anything that could expose real data.

Useful things to include:

- the Outworld version (`outworld version`) and how it is run (binary, Docker, Go library);
- the route and API involved (OpenAI or Anthropic, streaming or not);
- a minimal request that shows the problem, using invented data only.

You should get a first answer within a week. Fixes are released as a new version, with the issue described in [CHANGELOG.md](CHANGELOG.md) once a fix is available.

## What counts as a vulnerability

- A value that Outworld detects reaching the upstream anyway, or appearing in a 422 response, a log line or an error.
- A way for a client to loosen routing, skip the leak check or read the vault without the key.
- Anything that weakens the vault encryption or lets a crafted request corrupt the vault.
- Denial of service through inputs that make detection or rehydration much slower than linear.

A value that no detector recognizes (an unusual name, an ID format that is not covered) is a detection gap, not a vulnerability. Those are welcome as ordinary issues; the current list is in [docs/known-gaps.md](docs/known-gaps.md).

## Supported versions

Only the latest release gets security fixes.
