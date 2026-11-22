import importlib.util
import pathlib
import random

_spec = importlib.util.spec_from_file_location(
    "gliner_chunking", pathlib.Path(__file__).with_name("chunking.py")
)
chunking = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(chunking)
chunk_spans = chunking.chunk_spans
merge_entities = chunking.merge_entities


def test_short_text_is_one_chunk():
    assert chunk_spans("hello world", 100, 20) == [(0, 11)]
    assert chunk_spans("", 100, 20) == []


def test_chunks_cover_text_and_overlap():
    text = "word " * 2000
    spans = chunk_spans(text, 500, 100)
    assert spans[0][0] == 0 and spans[-1][1] == len(text)
    for (s, e), (ns, ne) in zip(spans, spans[1:]):
        assert e - s <= 500
        assert s < ns < e < ne
        assert e - ns >= 50


def test_prefers_newline_then_sentence_then_space():
    text = "a" * 60 + "\n" + "b" * 20 + ". " + "c" * 10 + " " + "d" * 200
    assert chunk_spans(text, 100, 0)[0] == (0, 61)
    text = "a" * 60 + ". " + "b" * 20 + " " + "c" * 200
    assert chunk_spans(text, 100, 0)[0] == (0, 62)
    text = "a" * 60 + " " + "b" * 200
    assert chunk_spans(text, 100, 0)[0] == (0, 61)
    text = "a" * 300
    assert chunk_spans(text, 100, 0)[0] == (0, 100)


def test_next_chunk_starts_on_a_line_or_word_boundary():
    text = ("x" * 30 + "\n") * 40
    for s, _ in chunk_spans(text, 200, 80)[1:]:
        assert text[s - 1] == "\n"
    text = "lorem ipsum " * 200
    for s, _ in chunk_spans(text, 200, 80)[1:]:
        assert text[s - 1] == " "


def test_entity_straddling_a_cut_is_whole_in_some_chunk():
    rng = random.Random(7)
    alphabet = "abcdefghij      \n.é😀"
    for _ in range(200):
        n = rng.randint(1, 3000)
        text = "".join(rng.choice(alphabet) for _ in range(n))
        size = rng.randint(8, 400)
        overlap = rng.randint(0, 200)
        spans = chunk_spans(text, size, overlap)
        assert spans[0][0] == 0 and spans[-1][1] == n
        for (s, e), (ns, ne) in zip(spans, spans[1:]):
            assert s < ns <= e < ne
        guaranteed = min(overlap, size // 4) // 2
        if guaranteed == 0:
            continue
        for _ in range(50):
            a = rng.randint(0, n - 1)
            b = min(n, a + rng.randint(1, guaranteed))
            assert any(s <= a and b <= e for s, e in spans), (a, b, spans)


def test_name_at_boundary_survives_chunking():
    name = "Zofia Brzezinska"
    text = "x" * 190 + " " + name + " " + "y" * 300
    start = text.index(name)
    spans = chunk_spans(text, 200, 80)
    assert not (spans[0][0] <= start and start + len(name) <= spans[0][1])
    assert any(s <= start and start + len(name) <= e for s, e in spans)


def test_merge_drops_duplicates_and_joins_partial_spans():
    merged = merge_entities(
        [
            (10, 15, "person", 0.6),
            (10, 20, "person", 0.9),
            (10, 20, "person", 0.8),
            (40, 45, "person", 0.7),
            (12, 18, "address", 0.5),
        ]
    )
    assert merged == [(10, 20, "person", 0.9), (12, 18, "address", 0.5), (40, 45, "person", 0.7)]


def test_merge_keeps_adjacent_spans_apart():
    assert merge_entities([(0, 5, "person", 0.9), (5, 9, "person", 0.8)]) == [
        (0, 5, "person", 0.9),
        (5, 9, "person", 0.8),
    ]
