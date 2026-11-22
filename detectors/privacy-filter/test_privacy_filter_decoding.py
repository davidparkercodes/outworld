import importlib.util
import os
import pathlib
import sys

import pytest

np = pytest.importorskip("numpy")

HERE = pathlib.Path(__file__).parent


def _load(name):
    spec = importlib.util.spec_from_file_location("privacy_filter_" + name, HERE / (name + ".py"))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


decoding = _load("decoding")

LABELS = ["O"] + [f"{tag}-{cat}" for cat in ("private_address", "private_person") for tag in "BIES"]
ID = {name: i for i, name in enumerate(LABELS)}


def decoder(**biases):
    return decoding.Decoder(dict(enumerate(LABELS)), biases)


def logprobs(rows):
    out = np.full((len(rows), len(LABELS)), -20.0, dtype=np.float32)
    for t, row in enumerate(rows):
        for name, value in row.items():
            out[t, ID[name]] = value
    return out


def test_viterbi_keeps_a_valid_path_unchanged():
    d = decoder()
    names = ["O", "B-private_person", "I-private_person", "E-private_person", "O", "S-private_person"]
    path = d.path(logprobs([{n: -0.01} for n in names]))
    assert [LABELS[i] for i in path] == names
    assert d.token_spans(path) == [("private_person", 1, 4), ("private_person", 5, 6)]


def test_viterbi_repairs_an_invalid_argmax_sequence():
    d = decoder()
    rows = [
        {"B-private_person": -0.1},
        {"O": -0.6, "E-private_person": -0.9, "I-private_person": -3.0},
        {"O": -0.1},
    ]
    path = [LABELS[i] for i in d.path(logprobs(rows))]
    assert path == ["B-private_person", "E-private_person", "O"]


def test_viterbi_never_switches_category_inside_a_span():
    d = decoder()
    rows = [{"B-private_person": -0.1}, {"E-private_address": -0.1, "E-private_person": -2.0}]
    assert [LABELS[i] for i in d.path(logprobs(rows))] == ["B-private_person", "E-private_person"]


def test_background_bias_trades_recall():
    rows = [{"O": -0.4, "S-private_person": -1.2}]
    assert decoder().token_spans(decoder().path(logprobs(rows))) == []
    eager = decoder(transition_bias_background_stay=-2.0)
    two = logprobs([{"O": -0.1}, {"O": -0.4, "S-private_person": -1.2}])
    assert eager.token_spans(eager.path(two)) == [("private_person", 1, 2)]


def test_token_spans_tolerates_broken_sequences():
    d = decoder()
    path = [ID[n] for n in ["I-private_person", "I-private_person", "B-private_address", "E-private_person"]]
    assert d.token_spans(path) == [("private_person", 0, 2), ("private_address", 2, 3), ("private_person", 3, 4)]


def test_char_spans_are_code_point_offsets_with_emoji_and_accents():
    d = decoder()
    text = "😀 Zoë Müller 👩‍👩‍👧 café"
    offsets = [(0, 1), (1, 5), (5, 12), (12, 13), (13, 18), (18, 18), (18, 23)]
    path = [ID[n] for n in ["O", "B-private_person", "E-private_person", "O", "O", "O", "O"]]
    probs = np.zeros((len(path), len(LABELS)), dtype=np.float32)
    probs[1, ID["B-private_person"]] = 0.9
    probs[2, ID["E-private_person"]] = 0.7
    probs[2, ID["I-private_person"]] = 0.1
    spans = d.char_spans(d.token_spans(path), offsets, text, probs)
    assert len(spans) == 1
    start, end, cat, score = spans[0]
    assert text[start:end] == "Zoë Müller"
    assert (start, end, cat) == (2, 12, "private_person")
    assert abs(score - 0.85) < 1e-6
    assert len(text[:start].encode("utf-8")) == 5


@pytest.mark.skipif(os.environ.get("PRIVACY_FILTER_TEST_LIVE") != "1", reason="needs the model")
def test_live_offsets_with_emoji_and_accents():
    sys.path.insert(0, str(HERE))
    import app

    line = "😀🎉 Héloïse Dupré-Müller 👩‍👩‍👧 wrote to José Ñandú about the café ☕ in Orléans.\n"
    filler = "".join(f"Note {i}: the crème brûlée order n° {i * 37} shipped 🚚 on time, naïve retries were not needed.\n" for i in range(40))
    text = line + filler + line
    assert len(text) > app.CHUNK_CHARS
    out = app.detect(app.DetectRequest(text=text, labels=["person", "organization"], threshold=0.5))
    assert [(e.label, text[e.start : e.end]) for e in out.entities] == [
        ("person", "Héloïse Dupré-Müller"),
        ("person", "José Ñandú"),
        ("person", "Héloïse Dupré-Müller"),
        ("person", "José Ñandú"),
    ]
    second = text.rindex("José Ñandú")
    assert out.entities[-1].start == second
    assert len(text[:second].encode("utf-8")) > second
