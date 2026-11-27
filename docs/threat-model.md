# Threat model and honest limits

What Outworld is for: reducing how much identifying personal data a cloud model provider receives, while keeping answers useful, with a hard stop when it notices it is about to leak something it knows about.

What it protects against:

- The provider (and its logs, retention, training pipelines, breaches) seeing the names, contact details, addresses, exact dates, account numbers, IDs and secrets that Outworld detects.
- Accidental leaks of values the vault already knows: every outbound payload is re-scanned for known real values and for unmasked detector hits. Anything found is masked in place with its existing surrogate (or a new one) and the payload is re-scanned; a known first name followed by a capitalized word is treated as a full name. Only what still cannot be transformed (for example a value in a JSON key or message `name`) blocks the request. The 422 lists each finding by type, field and position with a fully redacted preview and a hint, and never contains any character of the value ([details](blocked-requests.md)).
- Mentions split across messages: every text in a request is detected first, so a name that is only recognizable in a later message is also masked in earlier ones.
- Credentials and one-time codes that Outworld recognizes reaching any cloud model at all (stripped, never restored). Recognition is far from complete here: on the blind corpus about a third of secrets got through, mostly passwords and tokens with no label next to them. Do not rely on Outworld as a secret scanner.

What it does **not** do:

- **Detection is best-effort.** Regexes, your dictionary and an optional NER model will miss things: unusual names, nicknames you did not list, people described rather than named ("my sister's husband"), names in other scripts, typos. The leak check only blocks what some layer recognizes. Add a dictionary; add the NER sidecar; review what you send.
- **Masking is not anonymization.** Quasi-identifiers survive: a rare job title plus a small town plus a hospital stay can identify someone even with every name replaced. So can writing style, the structure of your life, and facts you chose to `keep`. Date shifting hides exact dates but keeps intervals (which is the point), and intervals can be identifying too.
- **The provider still sees the content.** Your medical question is still a medical question about *someone*. If the substance itself is what must not leave, route it `local` or to a `passthrough` confidential-compute endpoint, or `block` it. That is what routing rules are for.
- **Model output is untrusted.** Rehydration translates every surrogate and every date in the answer, including dates the model made up. Surrogate phone numbers are recognized in any formatting the model chooses (dashes, dots, spaces, parentheses, `+1`, digits only). Invented people that look like surrogates are flagged in `X-Outworld-Unknown-Names`, not removed.
- **The vault is the crown jewels.** It maps every surrogate back to a real value. It is encrypted at rest (AES-256-GCM with an authenticated, append-only record log), and Outworld refuses to run without a key. Protect the key like a password-manager master key. Anyone with both the vault and the key can reverse everything.
- **Not covered at all:** images, audio and files (refused on the masked route by default), fields outside the message text are only leak-checked, not rewritten, unless [What is masked](what-is-masked.md) lists them (a finding there blocks the request instead of being replaced; tool names, schema property names and ids are never rewritten), provider-signed thinking blocks (passed through unchanged), and anything an agent sends around the gateway on its own. The specific things detection is known to miss are listed under [Known gaps](known-gaps.md).
- Outworld logs never include real values or raw prompts: only routes, counts per entity type and timings.

When in doubt: use `local` for anything whose substance is sensitive, `masked` for everyday personal data where names and contact details are the risk, and `passthrough` to a TEE endpoint when you need a big model on raw data and trust that attestation.
