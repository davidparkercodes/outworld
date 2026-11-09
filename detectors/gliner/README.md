# GLiNER NER sidecar (optional)

Outworld works without this. Add it when you want detection of people, organizations and places that are not in your dictionary.

```bash
docker build -t outworld-gliner .
docker run --rm --memory 4g -p 127.0.0.1:8790:8790 outworld-gliner
```

The default model (`urchade/gliner_multi_pii-v1`) needs about 2 GB of RAM and takes roughly 1 to 3 seconds per 1,000 characters on a few CPU cores. The first request after start loads the model and can take a minute or more.

Then in `outworld.yaml`:

```yaml
ner:
  url: http://127.0.0.1:8790
  timeout: 30s
  threshold: 0.5
  optional: false   # true = keep going if the sidecar is down (less safe)
```

API: `POST /detect` with `{"text": "...", "labels": ["person", ...], "threshold": 0.5}` returns
`{"entities": [{"start": 0, "end": 5, "label": "person", "score": 0.93}]}`. Offsets are Unicode code point offsets.
Any service that speaks this shape can replace the sidecar.

The model runs on CPU by default. Set `GLINER_MODEL` to use another GLiNER checkpoint.
