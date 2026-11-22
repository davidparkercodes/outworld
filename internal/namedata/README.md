# Name data

`given.txt` and `surnames.txt` are built from public-domain and CC0 sources:

- United States Social Security Administration baby-name tables (top 1,000 names per year and sex, 1880 to 2008). A name must reach a summed yearly share of 0.0005, or 0.02 when it is also an ordinary English word.
- The 12,000 most frequent surnames in the United States 2010 Census surname file.
- Wikidata (CC0) given-name items (male, female and unisex) that have at least 4 sitelinks, or 10 when the name is also an English word, and family-name items carried by at least 25 people, or 150 when the name is also an English word.
- The `sigpwned/popular-names-by-country-dataset` lists (CC0) of common forenames and surnames per country.

`merge_intl.py` merges the Wikidata and per-country lists into the two files. Its arguments are, in order: the English word list, `given.txt`, `surnames.txt`, the three Wikidata given-name exports (male, female, unisex), the Wikidata family-name export, and the two per-country CSV files.

Format: `name flags`, one name per line. Names are lowercase and folded to ASCII (`josé` is stored as `jose`, `müller` as `muller`, `łukasz` as `lukasz`); lookups fold the same way, so accented and unaccented spellings match. For given names the first flag is the majority gender (`f`, `m`, or `n` when unknown or mixed). The last flag is `w` (weak) when the name is also an ordinary English word or has three letters or fewer, otherwise `s`. Weak names only count as people in name context.

Only Latin-script names are included. Names outside the tables, nicknames and names in other scripts are the job of the dictionary and the NER sidecar.
