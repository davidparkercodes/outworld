# How it works

Outworld is a privacy gateway that speaks both the OpenAI and the Anthropic Messages API. Point a client at it, and it lets you use cloud LLMs on personal data (email, notes, contacts, records, the files a coding agent reads) without the provider seeing the real personal details. Names, emails, phones and addresses are swapped for realistic stand-ins, dates are shifted, secrets are removed, and the model's answer is translated back before it reaches you.

It works with any OpenAI-compatible provider, with the Anthropic API (including as `ANTHROPIC_BASE_URL` for Claude Code), and with local servers (Ollama, llama.cpp, vLLM) for the requests you never want to leave your machine.

What changed in each version, and what to check before upgrading, is in [CHANGELOG.md](../CHANGELOG.md).

**Status:** 0.3.0, an early release. Over 900 tests, including scale tests that keep every detector linear-time. Leak rates on a blind corpus are published in [accuracy.md](accuracy.md), and every known gap in [known-gaps.md](known-gaps.md). Read [Threat model and honest limits](threat-model.md) before you rely on it.

```
  your app / agent / script
            |
            |  POST /v1/chat/completions or /v1/messages
            v
  +-----------------------------------------------------------+
  |  OUTWORLD                                                 |
  |                                                           |
  |  1. route     rules + X-Outworld-Route header             |
  |               masked | local | passthrough | block        |
  |                                                           |
  |  2. detect    regex (email, phone, address, ids, cards,   |
  |               keys, OTP codes, magic links, dates)        |
  |               + built-in name detector (no model needed)  |
  |               + your dictionary (contacts, aliases)       |
  |               + optional NER sidecar                      |
  |                                                           |
  |  3. transform per YOUR policy, per entity type:           |
  |               keep | mask | shift | generalize | strip    |
  |                                                           |
  |  4. leak check  re-scan the outbound payload; known real  |
  |               values and detector hits are masked in      |
  |               place and re-scanned; anything that still   |
  |               cannot be transformed = request blocked     |
  |               (fail closed)                               |
  |                                                           |
  |      +-------------------+                                |
  |      | encrypted vault   |  real <-> surrogate, stable    |
  |      | (AES-256-GCM)     |  across requests and corpora   |
  |      +-------------------+                                |
  |                                                           |
  |  6. rehydrate  surrogates and shifted dates translated    |
  |               back, including streamed SSE chunks and     |
  |               tool-call arguments                         |
  +-----------------------------------------------------------+
            |                                   ^
            | masked request                    | 5. response (surrogates)
            v                                   |
       cloud model  ----------------------------+
       ("Tamsin Wetherby emailed you on Dec 25")
```
