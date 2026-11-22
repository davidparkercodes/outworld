import re

_SENTENCE_END = re.compile(r"[.!?][\"')\]]*\s")


def _cut(text, lo, hi):
    i = text.rfind("\n", lo, hi)
    if i >= 0:
        return i + 1
    last = None
    for m in _SENTENCE_END.finditer(text, lo, hi):
        last = m
    if last is not None:
        return last.end()
    for i in range(hi - 1, lo - 1, -1):
        if text[i].isspace():
            return i + 1
    return hi


def _resume(text, lo, hi):
    i = text.find("\n", lo, hi)
    if i >= 0:
        return i + 1
    for i in range(lo, hi):
        if text[i].isspace():
            return i + 1
    return lo


def chunk_spans(text, size, overlap):
    n = len(text)
    size = max(1, size)
    overlap = max(0, min(overlap, size // 4))
    out = []
    start = 0
    while start < n:
        end = n if n - start <= size else _cut(text, start + size // 2, start + size)
        out.append((start, end))
        if end >= n:
            break
        nxt = _resume(text, end - overlap, end - overlap // 2) if overlap else end
        start = max(nxt, start + 1)
    return out


def merge_entities(entities):
    out = []
    last = {}
    for start, end, label, score in sorted(entities, key=lambda e: (e[0], -e[1])):
        i = last.get(label)
        if i is not None and start < out[i][1]:
            ps, pe, _, pscore = out[i]
            out[i] = (ps, max(pe, end), label, max(pscore, score))
        else:
            last[label] = len(out)
            out.append((start, end, label, score))
    return sorted(out)
