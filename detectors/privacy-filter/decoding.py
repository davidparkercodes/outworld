import numpy as np

NEG = -1e9
BIAS_KEYS = (
    "transition_bias_background_stay",
    "transition_bias_background_to_start",
    "transition_bias_inside_to_continue",
    "transition_bias_inside_to_end",
    "transition_bias_end_to_background",
    "transition_bias_end_to_start",
)


def biases_from_calibration(payload):
    raw = payload["operating_points"]["default"]["biases"]
    return {k: float(raw[k]) for k in BIAS_KEYS}


class Decoder:
    def __init__(self, id2label, biases=None):
        b = {k: 0.0 for k in BIAS_KEYS}
        b.update(biases or {})
        n = len(id2label)
        self.tags = []
        self.cats = []
        for i in range(n):
            name = id2label[i]
            if name == "O":
                self.tags.append(None)
                self.cats.append(None)
            else:
                tag, cat = name.split("-", 1)
                self.tags.append(tag)
                self.cats.append(cat)
        self.columns = {}
        for i, cat in enumerate(self.cats):
            if cat is not None:
                self.columns.setdefault(cat, []).append(i)
        self.start = np.full(n, NEG, dtype=np.float32)
        self.end = np.full(n, NEG, dtype=np.float32)
        self.trans = np.full((n, n), NEG, dtype=np.float32)
        for i in range(n):
            tag, cat = self.tags[i], self.cats[i]
            if tag in (None, "B", "S"):
                self.start[i] = 0.0
            if tag in (None, "E", "S"):
                self.end[i] = 0.0
            for j in range(n):
                ntag, ncat = self.tags[j], self.cats[j]
                if tag in ("B", "I"):
                    if ncat == cat and ntag == "I":
                        self.trans[i, j] = b["transition_bias_inside_to_continue"]
                    elif ncat == cat and ntag == "E":
                        self.trans[i, j] = b["transition_bias_inside_to_end"]
                elif tag is None:
                    if ntag is None:
                        self.trans[i, j] = b["transition_bias_background_stay"]
                    elif ntag in ("B", "S"):
                        self.trans[i, j] = b["transition_bias_background_to_start"]
                else:
                    if ntag is None:
                        self.trans[i, j] = b["transition_bias_end_to_background"]
                    elif ntag in ("B", "S"):
                        self.trans[i, j] = b["transition_bias_end_to_start"]

    def path(self, logprobs):
        steps = logprobs.shape[0]
        if steps == 0:
            return []
        scores = logprobs[0] + self.start
        back = np.empty((steps - 1, logprobs.shape[1]), dtype=np.int64)
        for t in range(1, steps):
            cand = scores[:, None] + self.trans
            back[t - 1] = cand.argmax(axis=0)
            scores = cand.max(axis=0) + logprobs[t]
        scores = scores + self.end
        if scores.max() <= NEG / 2:
            return logprobs.argmax(axis=1).tolist()
        last = int(scores.argmax())
        out = [last]
        for t in range(steps - 2, -1, -1):
            last = int(back[t, last])
            out.append(last)
        out.reverse()
        return out

    def token_spans(self, path):
        out = []
        cur = None
        start = 0
        for i, idx in enumerate(path):
            tag, cat = self.tags[idx], self.cats[idx]
            if cur is not None and (tag in (None, "B", "S") or cat != cur):
                out.append((cur, start, i))
                cur = None
            if tag is None:
                continue
            if tag == "S":
                out.append((cat, i, i + 1))
            elif tag == "E":
                if cur is None:
                    out.append((cat, i, i + 1))
                else:
                    out.append((cur, start, i + 1))
                    cur = None
            elif cur is None:
                cur = cat
                start = i
        if cur is not None:
            out.append((cur, start, len(path)))
        return out

    def char_spans(self, token_spans, offsets, text, probs):
        out = []
        for cat, a, b in token_spans:
            start = min(offsets[t][0] for t in range(a, b))
            end = max(offsets[t][1] for t in range(a, b))
            while start < end and text[start].isspace():
                start += 1
            while end > start and text[end - 1].isspace():
                end -= 1
            if end <= start:
                continue
            score = float(probs[a:b][:, self.columns[cat]].sum(axis=1).mean())
            out.append((start, end, cat, min(1.0, score)))
        return out
