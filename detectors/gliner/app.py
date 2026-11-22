import math
import os
import threading

from fastapi import FastAPI
from pydantic import BaseModel, Field

from chunking import chunk_spans, merge_entities

MODEL_ID = os.environ.get("GLINER_MODEL", "urchade/gliner_multi_pii-v1")
CHUNK_CHARS = int(os.environ.get("GLINER_CHUNK_CHARS", "1500"))
CHUNK_OVERLAP = int(os.environ.get("GLINER_CHUNK_OVERLAP", "200"))
DEFAULT_LABELS = ["person", "organization", "location", "address"]

app = FastAPI(title="outworld-gliner")
_model = None
_lock = threading.Lock()


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


def model():
    global _model
    with _lock:
        if _model is None:
            import torch
            from gliner import GLiNER

            threads = cpu_threads()
            if threads:
                torch.set_num_threads(threads)
            _model = GLiNER.from_pretrained(MODEL_ID)
        return _model


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


@app.get("/health")
def health():
    return {"status": "ok", "model": MODEL_ID, "loaded": _model is not None, "threads": cpu_threads()}


@app.post("/detect", response_model=DetectResponse)
def detect(req: DetectRequest):
    m = model()
    found = []
    for start, end in chunk_spans(req.text, CHUNK_CHARS, CHUNK_OVERLAP):
        chunk = req.text[start:end]
        if not chunk.strip():
            continue
        for e in m.predict_entities(chunk, req.labels, threshold=req.threshold):
            found.append((start + e["start"], start + e["end"], e["label"], float(e["score"])))
    return DetectResponse(entities=[Entity(start=s, end=e, label=l, score=sc) for s, e, l, sc in merge_entities(found)])
