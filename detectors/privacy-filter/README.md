# OpenAI Privacy Filter NER sidecar (optional)

An alternative to the GLiNER sidecar. It wraps [`openai/privacy-filter`](https://huggingface.co/openai/privacy-filter) (Apache-2.0), a token classifier that tags people, addresses, emails, phones, URLs, dates, account numbers and secrets. It has no organization or location label, so use it when names and addresses are what you need, or run GLiNER instead when organizations and places matter.

```bash
docker build -t outworld-privacy-filter .
docker run --rm --cpus 4 -e TORCH_NUM_THREADS=4 --memory 8g -e HF_HUB_OFFLINE=1 -p 127.0.0.1:8791:8791 outworld-privacy-filter
```

Then in `outworld.yaml`:

```yaml
ner:
  url: http://127.0.0.1:8791
  timeout: 60s
  threshold: 0.5
```

The model is loaded at start. `GET /health` reports `"loaded": true` once it is ready.

## API

Same contract as the GLiNER sidecar: `POST /detect` with `{"text": "...", "labels": ["person", "address"], "threshold": 0.5}` returns
`{"entities": [{"start": 0, "end": 5, "label": "person", "score": 0.93}]}`. Offsets are Unicode code point offsets into the request text.

Only requested labels are returned, under the name used in the request. Labels the model does not have (`organization`, `location`) are ignored.

| Request label | Model category |
|---|---|
| `person` | `private_person` |
| `address` | `private_address` |
| `email` | `private_email` |
| `phone` | `private_phone` |
| `url` | `private_url` |
| `date` | `private_date` |
| `account_number` | `account_number` |
| `secret` | `secret` |

The model's own category names are accepted too. To use the extra categories from Outworld, map them in `ner.labels`.

`score` is the mean probability the model gives the span's category over the span's tokens. Spans below `threshold` are dropped.

## Decoding

The model emits BIOES tags (begin, inside, end, single) per token. The sidecar decodes them the way the reference implementation does: a constrained Viterbi pass that only allows valid tag sequences, using the transition biases from the model's `viterbi_calibration.json` (all zero in the published checkpoint). The generic `transformers` token classification pipeline does not do this: it takes a per-token argmax and only understands `B-`/`I-` prefixes, which fragments spans.

To trade precision for recall, point `PRIVACY_FILTER_CALIBRATION` at a JSON file in the same format with your own biases. A negative `transition_bias_background_stay` makes the decoder open spans more readily.

## Backends

| Build | Weights | Download | Notes |
|---|---|---|---|
| default (`BACKEND=onnx`) | `onnx/model.onnx` | 5.6 GB | ONNX Runtime, full precision. Fastest in our runs, same output as PyTorch. The weights are memory-mapped, so they live in the page cache rather than the process heap. |
| `--build-arg ONNX_FILE=onnx/model_quantized.onnx` | `onnx/model_quantized.onnx` | 1.6 GB | ONNX Runtime, 8-bit weights. Smaller image, recall within half a point, not faster on CPU. |
| `--build-arg BACKEND=torch` | `model.safetensors` | 2.8 GB | PyTorch through `transformers` (5.6.0 or newer has the model). Three to four times slower and about 6 GB of RAM. Kept as the reference implementation. |

`--build-arg PRELOAD=0` skips baking the weights into the image; mount a Hugging Face cache at `/models` instead and drop `HF_HUB_OFFLINE=1` for the first start.

ONNX Runtime's official Linux builds send usage telemetry by default. The image sets `ORT_DISABLE_TELEMETRY=1` and the app sets it again before importing the runtime. With the weights preloaded and `HF_HUB_OFFLINE=1` the container makes no outbound connections.

## Tuning

| Variable | Default | Meaning |
|---|---|---|
| `TORCH_NUM_THREADS` | container CPU quota, else the runtime default | Inference threads, for both backends. Set it to the number of cores you give the container (`docker run --cpus 4 -e TORCH_NUM_THREADS=4`). |
| `PRIVACY_FILTER_CHUNK_CHARS` | `2000` | Longer texts are split into windows of at most this many characters. |
| `PRIVACY_FILTER_CHUNK_OVERLAP` | `200` | Characters shared by neighbouring windows (capped at a quarter of the window). |
| `PRIVACY_FILTER_MAX_TOKENS` | `1024` | A window that tokenizes to more than this is split again. |
| `PRIVACY_FILTER_PARALLEL_WINDOWS` | `1` | ONNX backend: when a text has at least half as many windows as threads, run windows side by side on single-threaded sessions instead of one after another. Set `0` to disable. |
| `PRIVACY_FILTER_CALIBRATION` | model default | Path to a Viterbi calibration JSON. |
| `PRIVACY_FILTER_EAGER_LOAD` | `1` | Load the model at start instead of on the first request. |
| `PRIVACY_FILTER_MODEL` | `openai/privacy-filter` | Hugging Face repo id. |
| `PRIVACY_FILTER_ONNX_FILE` | `onnx/model.onnx` | ONNX export to load. |

### Why it chunks a model with a 128k token context

The model attends 128 tokens to each side per layer, so nothing more than about 1,000 tokens away can influence a token. The published implementations still build a full token-by-token attention mask, so cost per token is flat up to about 1,000 tokens and then climbs (on 8 cores: 1.4 ms per token at 512 and 1,024 tokens, 2.1 ms at 2,048), and memory grows with the square of the length. Windows of 2,000 characters are about 500 tokens of prose, which sits at the cheapest point of that curve and keeps memory flat for any input size.

Windows are cut at a newline when one is available in the second half of the window, otherwise at a sentence end, otherwise at whitespace. Each window starts inside the previous one, so a name that straddles a cut is seen whole by the next window; overlapping results with the same label are merged into one span. Any entity up to half the overlap long (100 characters by default) is guaranteed to fit in one window.

Long texts are where thread use matters most. The runtime does not scale well inside one window (8 threads are only about 3.5 times faster than 1), so for multi-window texts the ONNX backend runs one window per thread, which roughly doubles throughput on 8 cores. Short texts keep using all threads on their single window. Requests are processed one at a time so concurrent calls do not oversubscribe the cores.

## Performance

Measured on a shared 36-core Xeon host that was busy with other work (load average 12 to 24), containers limited with `--cpus`, CPU only. Corpus: 334 chat and email messages (150k characters, at most 1.3k each) with 1,255 labelled person mentions, sent with the labels Outworld sends by default, threshold 0.5. Treat the timings as rough: repeated runs of the same configuration differed by up to 60 percent.

| Sidecar | Threads | Person recall | Latency p50 / p95 per message | Seconds per 1,000 chars | 50,000 chars |
|---|---|---|---|---|---|
| Privacy Filter, ONNX full precision | 8 | 86.4% | 0.20 s / 0.56 s | 0.62 | 16 s |
| Privacy Filter, ONNX full precision | 4 | 86.4% | 0.28 s / 0.76 s | 0.87 | 29 s |
| Privacy Filter, ONNX 8-bit | 8 | 86.1% | 0.28 to 0.37 s / 0.58 to 0.98 s | 0.76 to 1.09 | 20 s |
| Privacy Filter, ONNX 8-bit | 4 | 86.1% | 0.38 to 0.44 s / 0.84 to 0.95 s | 1.05 to 1.19 | 38 s |
| Privacy Filter, PyTorch | 8 | 86.4% | 0.99 s / 1.86 s | 2.42 | 76 s |
| Privacy Filter, PyTorch | 4 | 86.4% | 0.88 s / 1.90 s | 2.31 | 70 s |
| GLiNER `gliner_multi_pii-v1` | 8 | 64.3% | 0.42 s / 1.24 s | 1.32 | 66 s |
| GLiNER `gliner_multi_pii-v1` | 4 | 64.3% | 0.61 s / 1.77 s | 2.04 | 101 s |

Recall by how the name is written (Privacy Filter / GLiNER): full name 99% / 99%, first name only 84% / 22%, title plus surname 100% / 38%, possessive 50% / 22%, all caps 70% / 89%, lowercase 56% / 33%, nickname 86% / 57%, name inside a URL slug 0% / 0%.

False positives: Privacy Filter tagged 57 non-person strings as a person (0.38 per 1,000 characters), mostly company names and fragments of email addresses. GLiNER tagged 312, of which 304 were the literal word "person" in a system prompt; without that one string it made 8 (0.05 per 1,000 characters).

These are sidecar-only numbers on the corpus the built-in detectors were tuned on. Inside the gateway, where the other detectors already catch most full names, the two sidecars end up with similar leak rates and Privacy Filter over-masks more; see the measured results in the top-level README.

Things to know:

- It does not find organizations or places. Outworld's regex, dictionary and name detectors still run alongside it.
- On text that repeats the same sentence dozens of times the model stops tagging later repeats. This is model behaviour, identical on both backends.
- Raising the threshold to 0.9 lowers recall to 81% and false positives to 0.21 per 1,000 characters; 0.3 changes nothing compared to 0.5.

## Tests

```bash
pip install pytest numpy
pytest
```

The chunker and decoder tests are pure functions and do not load the model. `PRIVACY_FILTER_TEST_LIVE=1 pytest` inside the container also runs an end-to-end offset check on text with emoji and accented characters.
