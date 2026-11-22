import json
import math
import os
import threading
from concurrent.futures import ThreadPoolExecutor
from contextlib import asynccontextmanager

import numpy as np
from fastapi import FastAPI
from pydantic import BaseModel, Field

from chunking import chunk_spans, merge_entities
from decoding import Decoder, biases_from_calibration

os.environ["ORT_DISABLE_TELEMETRY"] = "1"
os.environ.setdefault("HF_HUB_DISABLE_TELEMETRY", "1")

MODEL_ID = os.environ.get("PRIVACY_FILTER_MODEL", "openai/privacy-filter")
BACKEND = os.environ.get("PRIVACY_FILTER_BACKEND", "onnx")
ONNX_FILE = os.environ.get("PRIVACY_FILTER_ONNX_FILE", "onnx/model.onnx")
CHUNK_CHARS = int(os.environ.get("PRIVACY_FILTER_CHUNK_CHARS", "2000"))
CHUNK_OVERLAP = int(os.environ.get("PRIVACY_FILTER_CHUNK_OVERLAP", "200"))
MAX_TOKENS = int(os.environ.get("PRIVACY_FILTER_MAX_TOKENS", "1024"))
CALIBRATION = os.environ.get("PRIVACY_FILTER_CALIBRATION", "")
PARALLEL_WINDOWS = os.environ.get("PRIVACY_FILTER_PARALLEL_WINDOWS", "1") == "1"
EAGER_LOAD = os.environ.get("PRIVACY_FILTER_EAGER_LOAD", "1") == "1"
DEFAULT_LABELS = ["person", "address"]
ALIASES = {
    "person": "private_person",
    "address": "private_address",
    "email": "private_email",
    "phone": "private_phone",
    "url": "private_url",
    "date": "private_date",
}

_state = None
_load_lock = threading.Lock()
_infer_lock = threading.Lock()


def cpu_threads():
    raw = os.environ.get("TORCH_NUM_THREADS", "")
    if raw:
        return max(1, int(raw))
    try:
        with open("/sys/fs/cgroup/cpu.max") as f:
            quota, period = f.read().split()
        if quota != "max":
            return max(1, math.ceil(int(quota) / int(period)))
    except (OSError, ValueError):
        pass
    return 0


def load_onnx(threads):
    import onnxruntime as ort
    from huggingface_hub import snapshot_download

    ort.disable_telemetry_events()
    root = snapshot_download(MODEL_ID, allow_patterns=[ONNX_FILE + "*"])
    path = os.path.join(root, ONNX_FILE)

    def session(intra):
        opts = ort.SessionOptions()
        if intra:
            opts.intra_op_num_threads = intra
        return ort.InferenceSession(path, opts, providers=["CPUExecutionProvider"])

    def logits(sess, ids):
        arr = np.array([ids], dtype=np.int64)
        return sess.run(["logits"], {"input_ids": arr, "attention_mask": np.ones_like(arr)})[0][0]

    wide = session(threads)
    if threads < 2 or not PARALLEL_WINDOWS:
        return lambda batch: [logits(wide, ids) for ids in batch]
    narrow = session(1)
    pool = ThreadPoolExecutor(max_workers=threads)

    def run(batch):
        if len(batch) < max(2, threads // 2):
            return [logits(wide, ids) for ids in batch]
        return list(pool.map(lambda ids: logits(narrow, ids), batch))

    return run


def load_torch(threads):
    import torch
    from transformers import AutoModelForTokenClassification

    if threads:
        torch.set_num_threads(threads)
    net = AutoModelForTokenClassification.from_pretrained(MODEL_ID, dtype=torch.float32)
    net.eval()

    def logits(ids):
        batch = torch.tensor([ids], dtype=torch.long)
        with torch.inference_mode():
            return net(input_ids=batch, attention_mask=torch.ones_like(batch)).logits[0].float().numpy()

    return lambda batch: [logits(ids) for ids in batch]


def state():
    global _state
    with _load_lock:
        if _state is None:
            from huggingface_hub import snapshot_download
            from tokenizers import Tokenizer

            root = snapshot_download(MODEL_ID, allow_patterns=["config.json", "tokenizer.json", "viterbi_calibration.json"])
            with open(os.path.join(root, "config.json"), encoding="utf-8") as f:
                id2label = {int(k): v for k, v in json.load(f)["id2label"].items()}
            biases = None
            calibration = CALIBRATION or os.path.join(root, "viterbi_calibration.json")
            if os.path.exists(calibration):
                with open(calibration, encoding="utf-8") as f:
                    biases = biases_from_calibration(json.load(f))
            tokenizer = Tokenizer.from_file(os.path.join(root, "tokenizer.json"))
            tokenizer.no_truncation()
            tokenizer.no_padding()
            run = load_torch(cpu_threads()) if BACKEND == "torch" else load_onnx(cpu_threads())
            warm = tokenizer.encode("warm up", add_special_tokens=False).ids
            run([warm])
            run([warm] * 8)
            _state = (tokenizer, run, Decoder(id2label, biases))
        return _state


@asynccontextmanager
async def lifespan(_):
    if EAGER_LOAD:
        threading.Thread(target=state, daemon=True).start()
    yield


app = FastAPI(title="outworld-privacy-filter", lifespan=lifespan)


class DetectRequest(BaseModel):
    text: str
    labels: list[str] = Field(default_factory=lambda: list(DEFAULT_LABELS))
    threshold: float = 0.5


class Entity(BaseModel):
    start: int
    end: int
    label: str
    score: float


class DetectResponse(BaseModel):
    entities: list[Entity]


def wanted_categories(labels, known):
    out = {}
    for label in labels:
        cat = ALIASES.get(label.lower(), label.lower())
        if cat in known and cat not in out:
            out[cat] = label
    return out


def windows(tokenizer, text, base, size):
    for start, end in chunk_spans(text, size, CHUNK_OVERLAP):
        piece = text[start:end]
        if not piece.strip():
            continue
        enc = tokenizer.encode(piece, add_special_tokens=False)
        count = len(enc.ids)
        if count > MAX_TOKENS and len(piece) > 1:
            smaller = max(1, min(len(piece) - 1, int(len(piece) * MAX_TOKENS / count * 0.9)))
            yield from windows(tokenizer, piece, base + start, smaller)
        elif count:
            yield base + start, piece, enc


def decode_window(decoder, piece, enc, logits):
    logits = logits.astype(np.float32)
    shifted = logits - logits.max(axis=1, keepdims=True)
    logprobs = shifted - np.log(np.exp(shifted).sum(axis=1, keepdims=True))
    spans = decoder.token_spans(decoder.path(logprobs))
    if not spans:
        return []
    return decoder.char_spans(spans, enc.offsets, piece, np.exp(logprobs))


@app.get("/health")
def health():
    return {"status": "ok", "model": MODEL_ID, "backend": BACKEND, "loaded": _state is not None, "threads": cpu_threads()}


@app.post("/detect", response_model=DetectResponse)
def detect(req: DetectRequest):
    tokenizer, run, decoder = state()
    wanted = wanted_categories(req.labels, decoder.columns)
    if not wanted or not req.text.strip():
        return DetectResponse(entities=[])
    wins = list(windows(tokenizer, req.text, 0, CHUNK_CHARS))
    with _infer_lock:
        outputs = run([enc.ids for _, _, enc in wins])
    found = []
    for (offset, piece, enc), logits in zip(wins, outputs):
        for start, end, cat, score in decode_window(decoder, piece, enc, logits):
            if cat in wanted and score >= req.threshold:
                found.append((offset + start, offset + end, wanted[cat], score))
    return DetectResponse(entities=[Entity(start=s, end=e, label=l, score=sc) for s, e, l, sc in merge_entities(found)])
