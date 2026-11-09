import os
import threading

from fastapi import FastAPI
from pydantic import BaseModel, Field

MODEL_ID = os.environ.get("GLINER_MODEL", "urchade/gliner_multi_pii-v1")
CHUNK_CHARS = int(os.environ.get("GLINER_CHUNK_CHARS", "1500"))
DEFAULT_LABELS = ["person", "organization", "location", "address"]

app = FastAPI(title="outworld-gliner")
_model = None
_lock = threading.Lock()


def model():
    global _model
    with _lock:
        if _model is None:
            from gliner import GLiNER

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


def chunks(text: str):
    start = 0
    n = len(text)
    while start < n:
        end = min(n, start + CHUNK_CHARS)
        if end < n:
            cut = text.rfind(" ", start + CHUNK_CHARS // 2, end)
            if cut > start:
                end = cut
        yield start, text[start:end]
        start = end


@app.get("/health")
def health():
    return {"status": "ok", "model": MODEL_ID, "loaded": _model is not None}


@app.post("/detect", response_model=DetectResponse)
def detect(req: DetectRequest):
    m = model()
    out = []
    for offset, chunk in chunks(req.text):
        if not chunk.strip():
            continue
        for e in m.predict_entities(chunk, req.labels, threshold=req.threshold):
            out.append(Entity(start=offset + e["start"], end=offset + e["end"], label=e["label"], score=float(e["score"])))
    return DetectResponse(entities=out)
