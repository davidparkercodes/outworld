# How well it works: measured

The numbers below come from the eval harness in [`eval/`](../eval/README.md). It sends synthetic prompts with labelled sensitive values through the gateway to a fake upstream that records exactly what a provider would have received. **Leak rate** is the share of labelled sensitive values that reached the upstream unchanged. **Over-masking** is the share of things that should have been left alone (companies, cities, product names, version strings, hashes) that were changed anyway. **Blocked** is the share of requests the fail-closed leak check refused with a 422.

There are four corpora, and the difference between them matters more than any single number:

- **Tuned**: 152 items. The detectors were developed against it. Its numbers show what the detectors do on forms they were built for, and nothing more.
- **Development**: 85 items written without access to the detector code. After the first run on it, general gaps it exposed were fixed, so it is no longer blind.
- **Second held-out**: 120 items, 160 invented people, about 33,000 words, written without access to the detector code. No detector was changed after looking at its leaks, but its per-type totals steered the 0.3.0 work on secrets and IDs, so it is no longer fully blind.
- **Blind**: 131 items, 188 invented people, about 43,000 words and 4,208 labelled spans, written by hand without access to the detector code and much heavier on international names, IDs and addresses than the others. Only its totals have been looked at, and no detector was tuned against it. This is the number to believe.

One disclosure: two lines of a generator file for the second held-out corpus, and the text of one of its items, were seen by accident during maintenance. Nothing was tuned against them.

Leak rate in 0.3.0, through the OpenAI API, measured on 2026-11-26 (the Anthropic API, streaming, and repeat runs with a fresh vault land within about two points):

| Detectors | Second held-out | Blind |
|---|---|---|
| Patterns and built-in names only | 16.5% | 20.1% |
| Plus your dictionary (about half the people listed) | 10.5% | 16.2% |

On the blind corpus, by kind of value (with a dictionary): emails 0%, phones 8%, dates 9%, names 16%, secrets 16%, bank and card details 30%, government IDs 34%, addresses 54%. The addresses that get through are mostly formats from countries the patterns do not cover; the names are mostly nicknames, first names that are not in the tables, and names in non-Latin scripts.

The cost side in 0.3.0, without a sidecar:

| Corpus | Over-masking | Blocked | Added latency, median |
|---|---|---|---|
| Second held-out | 21% | 1% to 2.5% | about 65 ms |
| Blind | 16% | 4.6% | about 110 ms |

The blind corpus and the second held-out corpus are deliberately full of things that look sensitive and are not (companies named after people, cities that are also first names, commit hashes, version strings), which is why over-masking is high. Over-masking is reversible: the answer is still translated back, but the model loses what it knew about the real company or place.

With an NER sidecar (measured on 0.2.0, second held-out corpus): dictionary and GLiNER 9.3% leak, 26% over-masking, 3% blocked, 2.4 s median added latency on 8 CPU threads; dictionary and Privacy Filter 8.6% leak, 29% over-masking, 3% blocked, 0.5 s. The tuned and development corpora land at 0% to 0.3% and 3.5% to 13.5% across all configurations.

What to take from this:

- Outworld removes most of what identifies people in ordinary prompts. It does not remove all of it: on text it has never seen it misses roughly one sensitive value in six with a dictionary, and about one in ten with a dictionary and a model.
- A dictionary of the people you actually talk about is the cheapest large improvement. A sidecar is the next one.
- Restore accuracy (answers translated back correctly) is above 99% in every configuration: 99.3% on the blind corpus, 99.8% on the second held-out corpus.
- These are synthetic prompts against a fake upstream. Sidecar latencies were measured on a shared, busy CPU host with no GPU and vary a lot with prompt length; the 95th percentile on long agent transcripts is tens of seconds for GLiNER.

Reproduce any cell with `go run ./cmd/outworld-eval --corpus corpus/holdout3.json --people corpus/holdout3-people.csv --configs regex-only,dictionary` from `eval/` (or `holdout2` for the second held-out corpus). If you change a detector after reading the blind corpus results, it stops being blind; write a new one.
