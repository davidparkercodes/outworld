# Optional NER sidecars

Outworld works without a model. A sidecar adds recall for names, organizations and places that the patterns, the built-in name detector and your dictionary miss, at the cost of latency. Two small FastAPI services are bundled and speak the same `/detect` contract, so either goes under `ner.url`:

- [detectors/gliner](../detectors/gliner/README.md): GLiNER, multilingual, covers people, organizations, locations and addresses.
- [detectors/privacy-filter](../detectors/privacy-filter/README.md): OpenAI Privacy Filter, an open-weight PII token classifier. It has no organization or location labels.

Which one: in the gateway eval the two leak about the same amount. Privacy Filter is several times faster on CPU and better at first names, titles and nicknames, but because it has no organization label it tends to tag company names as people, so it over-masks more. GLiNER is slower and knows what an organization is. Privacy Filter can also return `account_number` and `secret` spans; map them in `ner.labels` (`account_number: financial`, `secret: secret`) to use them. That mapping is not part of the measurements in [accuracy.md](accuracy.md).
