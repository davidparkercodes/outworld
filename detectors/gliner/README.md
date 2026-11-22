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

## Tuning

| Variable | Default | Meaning |
|---|---|---|
| `TORCH_NUM_THREADS` | container CPU quota, else all cores | Inference threads. Set it to the number of cores you give the container (`docker run --cpus 4 -e TORCH_NUM_THREADS=4`). Without a CPU quota and without this variable PyTorch starts one thread per host core, which is slower on a shared machine. |
| `GLINER_CHUNK_CHARS` | `1500` | Longer texts are split into windows of at most this many characters. |
| `GLINER_CHUNK_OVERLAP` | `200` | Characters shared by neighbouring windows (capped at a quarter of the window). |

Windows are cut at a newline when one is available in the second half of the window, otherwise at a sentence end, otherwise at whitespace. Each window starts inside the previous one, so a name that straddles a cut is seen whole by the next window; overlapping results with the same label are merged into one span. Any entity up to half the overlap long (100 characters by default) is guaranteed to fit in one window.

Cost grows linearly with length: a 50,000 character text is about 38 windows.

## Tests

```bash
pip install pytest
pytest test_gliner_chunking.py
```

The chunker tests are pure functions and do not load the model.
