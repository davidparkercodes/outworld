# Name data

`given.txt` and `surnames.txt` are built from public-domain and CC0 sources:

- United States Social Security Administration baby-name tables (top 1,000 names per year and sex, 1880 to 2008). A name must reach a summed yearly share of 0.0005, or 0.02 when it is also an ordinary English word.
- The 12,000 most frequent surnames in the United States 2010 Census surname file.
- Wikidata (CC0) given-name items (male, female and unisex) that have at least 4 sitelinks, or 10 when the name is also an English word, and family-name items carried by at least 25 people, or 150 when the name is also an English word.
- The `sigpwned/popular-names-by-country-dataset` lists (CC0) of common forenames and surnames per country.

`merge_intl.py` merges the Wikidata and per-country lists into the two files. Its arguments are, in order: the English word list, `given.txt`, `surnames.txt`, the three Wikidata given-name exports (male, female, unisex), the Wikidata family-name export, and the two per-country CSV files.

Format: `name flags`, one name per line. Names are lowercase and folded to ASCII (`josé` is stored as `jose`, `müller` as `muller`, `łukasz` as `lukasz`); lookups fold the same way, so accented and unaccented spellings match. For given names the first flag is the majority gender (`f`, `m`, or `n` when unknown or mixed). The last flag is `w` (weak) when the name is also an ordinary English word or has three letters or fewer, otherwise `s`. Weak names only count as people in name context.

Only Latin-script names are included. Names outside the tables, nicknames and names in other scripts are the job of the dictionary and the NER sidecar.

`companies.txt` is a hand-written list of well-known organizations whose names look like a person's name (one per line, as usually written). The name detector treats a match as an organization, not a person. Matching folds case and accents, ignores `&` and `and`, and accepts a hyphen or a space between words. Keep it to names where the organization is the far more likely reading; a very common personal name does not belong here.

`phrases.txt` is a hand-written list of well-known multi-word places and terms that contain a first name or surname (`monte carlo`, `charlotte amalie`, `markov chain`, `turing test`). A name token inside a listed phrase is not read as a person. One phrase per line, lowercase, folded to ASCII, at least two words, kept in byte order (`TestPhrasesAreFoldedSortedAndMultiWord` and `TestPersonShapedPhrasesAreFlaggedOrAbsent` check this). Do not list real people (`florence nightingale` is a person, not a term) or placeholder names (`jane doe`). An entry that is also shaped like an ordinary personal name (`tom collins`, `jack russell`, `glen allen`, `mary sue`) ends in ` *`: it is still read as a person when the text gives person evidence (a person verb after it, a person-directed verb before it, a greeting, role, honorific or people label before it, or a further name word next to it), so only flag-free entries are things in every context. Places that start with `San`, `Santa`, `Saint`, `St.`, `Mount`, `Fort`, `Port`, `Lake`, `Cape`, `New`, `Los` or `Las` do not need an entry.

`handlewords.txt` lists ordinary English words that happen to look like a handle built from the name tables (an initial plus a surname such as `along`, or a first name plus a surname). It is derived from the same English word list and the two name tables, and stops those words from being read as usernames in code.

Surrogate pools for family-first names live in `pkg/surrogate/eastnames.go`, next to the other surrogate lists: common Vietnamese, Chinese and Korean family names, Vietnamese given names, Chinese given names and syllables, and Korean given-name syllables. They are hand-written from widely published lists of common names (names themselves are not copyrightable). `TestEastPoolsAreCleanAndDisjoint` checks that they do not overlap each other or the general lists and that every entry has at least four letters.
