import csv
import sys
import unicodedata

SPECIAL = {"ł": "l", "ð": "d", "đ": "d", "ß": "ss", "ø": "o", "æ": "ae", "œ": "oe", "þ": "th", "ı": "i", "ħ": "h"}


def fold(s):
    s = s.lower()
    out = []
    for ch in unicodedata.normalize("NFD", s):
        if unicodedata.combining(ch):
            continue
        out.append(SPECIAL.get(ch, ch))
    return "".join(out)


def usable(label):
    if not label or len(label) > 24:
        return False
    for ch in label:
        if not (ch.isalpha() or ch in "-'"):
            return False
    key = fold(label)
    return key.isascii() and key.replace("-", "").replace("'", "").isalpha()


def tsv_label(field):
    return field.rsplit("@", 1)[0].strip('"')


def main(given_path, surname_path, words_path, wd_given, wd_family, sig_given, sig_family):
    words = set()
    for line in open(words_path, encoding="utf-8", errors="ignore"):
        w = line.strip()
        if w and w.islower():
            words.add(w)

    given = {}
    for line in open(given_path, encoding="utf-8"):
        name, _, flags = line.strip().partition(" ")
        if name:
            given[name] = flags
    surnames = {}
    for line in open(surname_path, encoding="utf-8"):
        name, _, flag = line.strip().partition(" ")
        if name:
            surnames[name] = flag

    def weak(key):
        return key in words or len(key) <= 3

    for path, gender in wd_given:
        rows = open(path, encoding="utf-8").read().splitlines()[1:]
        for row in rows:
            parts = row.split("\t")
            if len(parts) != 3:
                continue
            label, links = tsv_label(parts[1]), int(parts[2].split("^^")[0].strip('"'))
            if not usable(label) or "-" in label or "'" in label:
                continue
            key = fold(label)
            if len(key) < 3 or key in given:
                continue
            w = weak(key)
            if links < (10 if w else 4):
                continue
            given[key] = gender + ("w" if w else "s")

    with open(sig_given, encoding="utf-8-sig") as f:
        for row in csv.DictReader(f):
            label = row["Romanized Name"].strip()
            if not usable(label) or " " in label:
                continue
            key = fold(label)
            if len(key) < 3 or key in given:
                continue
            g = {"F": "f", "M": "m"}.get(row["Gender"], "n")
            given[key] = g + ("w" if weak(key) else "s")

    for row in open(wd_family, encoding="utf-8").read().splitlines():
        parts = row.split("\t")
        if len(parts) != 2:
            continue
        label, count = parts[0], int(parts[1])
        if not usable(label):
            continue
        key = fold(label)
        if len(key) < 2 or key in surnames:
            continue
        w = weak(key)
        if count < (150 if w else 25):
            continue
        surnames[key] = "w" if w else "s"

    with open(sig_family, encoding="utf-8-sig") as f:
        for row in csv.DictReader(f):
            label = row["Romanized Name"].strip()
            if not usable(label) or " " in label:
                continue
            key = fold(label)
            if len(key) < 2 or key in surnames:
                continue
            surnames[key] = "w" if weak(key) else "s"

    with open(given_path, "w", encoding="utf-8") as f:
        for k in sorted(given):
            f.write(f"{k} {given[k]}\n")
    with open(surname_path, "w", encoding="utf-8") as f:
        for k in sorted(surnames):
            f.write(f"{k} {surnames[k]}\n")
    print(len(given), len(surnames))


if __name__ == "__main__":
    a = sys.argv[1:]
    main(a[0], a[1], a[2], [(a[3], "m"), (a[4], "f"), (a[5], "n")], a[6], a[7], a[8])
