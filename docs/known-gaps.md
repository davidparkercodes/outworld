# Known gaps

Every item here was found by tests, probes or review and is still open. Unless it says otherwise, the value goes to the provider unmasked. A dictionary entry closes most of the name items, and an NER sidecar closes some.

Names:

- Nicknames you have not listed, first names that are not in the built-in tables, and people who are described instead of named (`my sister's husband`).
- Names in non-Latin scripts (Cyrillic, Arabic, Chinese, Japanese, Korean, Greek, Hebrew, Devanagari) without a sidecar, names with case endings (`Bartosza Wiśniewskiego`), patronymic forms of address, and names damaged by OCR.
- A Latin-script name or an email address glued directly after CJK text, with no space: in `请联系John Smith` the name is sent.
- A family-name-first name whose family name is an ordinary word, when nothing in the sentence treats it as a person (`Ma Long's serve`, or `He Wei` at the start of a sentence).
- A single all-caps surname before a comma when the surname is a common word or shorter than four letters: in `KING, Stephen` only `Stephen` is masked.
- A lone common first name at the start of a sentence with no dictionary and no other name context.
- A double given name without a hyphen: `Lily Rose Depp said hi.` is missed on a fresh vault, and in `I met Lily Rose Depp yesterday.` the surname `Depp` is sent.
- Initials and initialisms: masked only when the full name appears in the same text. One that is also a common abbreviation or a state or country code (`OH`, `IT`, `US`) is masked only as a speaker label or list entry. Lone initials with no full name nearby (`LH will send notes`) are left as they are.
- Handles: a bare handle in prose needs a cue word right before it and a name shape (`ask kchen`); after a weak cue such as `per` a short handle is missed (`per jdoe`). A handle with no recognizable name in it (`@zorgblat`) is missed outside home directories and git trailers. A mention with extra characters attached (`@jsmithLane`) is left alone. Inside `TODO(...)`, a name that is also a month or a technology term (`TODO(ruby)`) is left alone. A lone name as a branch prefix after `git log` or `git diff`, or in a dotted fork reference, is missed.
- A handle is linked to its owner's stand-in only when the same request or your dictionary shows the full name. Otherwise the model sees the handle and the person as two different people.
- A person who shares a name with a listed company is masked only when the text treats the name as a person and nowhere as a company: `Ask Edward Jones about fees.` masks, `Notes from Edward Jones.` keeps.
- A full name directly in front of `Award`, `Memorial`, `Building` and similar words is treated as the name of the thing unless the sentence shows a person (`The John Smith Memorial Fund opened.` is kept). A listed multi-word place or term is never a person (`Monte Carlo called.` is kept).
- A technology word that is also a surname, used as a first name, keeps on a fresh vault: in `Docker Jones called`, only `Jones` is masked.
- An assistant name used alone is kept unless the sentence treats it as a person, even when the vault knows a person with that name: after `Claude Shannon` has been masked, `The Claude answer was wrong.` keeps `Claude`. Greetings to an assistant name keep too (`Hi Alexa,`, `Hi Bard,`).
- Masked when it should not be (the answer is still restored): once the vault knows a person whose first name is also a technology name, a lone use keeps only on strong technical evidence right next to it (a version, a technical noun right after it, an install-style verb right before it, a code span), so some technical sentences lose the word: `Ruby case naming` becomes a stand-in name when the vault knows a Ruby. A first name the vault knows is masked at the start of a product or theme name (`Rose Pine theme is installed.` when the vault knows a Rose). A greeting to a technology word that is also a surname masks it (`Hello Docker,`). Generic greeting targets that are not on the built-in list mask (`Hi Sales,`, `Hi Store,`), and so do surname-like words before a role word (`Hi Hall Manager,`, `Hi Park Team,`). Two names joined by an em dash with no spaces are read as one person, so the first word is masked even when it is not a name (a sign-off of `Thanks`, an em dash and `John` masks `Thanks`) and the second name gets a surname stand-in. Companies, places and products that look like people are masked too, about one in five on the blind corpus; `with` in front of a name plus a building word reads as a person (`the merger with Wilson Hall`).

Phones, IDs and secrets:

- A phone number with letters glued to it, even one the vault already knows: `x804-555-1234` and `804-555-1234xyz` are sent as they are.
- A label glued to a number in a national format (`Mobile07700 900123`), unlike glued labels on dashed numbers such as `TEL090-1234-5678`, which are masked.
- A continuation label on the next line: in `phone 804~555~1234` followed by a line `work 703~555~2468`, the second number is sent. The same pair on one line is masked.
- `faxes`, `text` or `reach` before a number separated by `*`: in `He faxes 804*555*1234 daily.` the number is sent. `call` and `phone` before it work.
- A detected phone number is never sent as it is, but when no stand-in can be shown to restore in its exact place, it is replaced by `[redacted phone]` and is not restored in the answer. This happens rarely: when digits or a percent escape are glued to the number or other phone numbers run into it, mostly for a number that already has many stored spellings (see Vault below).
- A phone number written in styled digits (mathematical bold `𝟖𝟎𝟒-𝟓𝟓𝟓-𝟏𝟐𝟑𝟒`, full-width `８０４-５５５-１２３４`) comes back in plain ASCII digits (`804-555-1234`). Superscript and subscript digits next to a number restore exactly.
- A card expiry and CVV with no card number nearby: in `exp 12/27 cvv 123` the CVV is stripped, but the expiry is treated as a date and shifted rather than stripped.
- Passwords made of plain words or with no symbols when no credential word is next to them (`Sommer2024`), and any password more than about 70 bytes from its cue word.
- ID numbers with no label and no checksum to recognize them by, labels in languages that are not listed, values longer than 32 characters after a label, and account numbers after ambiguous words (`cuenta`, `conto`, `compte`).
- Secrets inside base64 text, seed phrases with no label, AWS secret keys and 64-character hex keys with no label, and some less common crypto address formats.
- On very long single lines a cue word or label only counts within about 256 bytes of the value.
- A bare number that is both a valid ID and phone-shaped can be replaced as a phone number instead of stripped. It is still masked, but restored in the answer.
- Masked when it should not be: numbers that look like phone numbers, a number joined to a word by a dot (`hotel.804-555-1234` is masked, while `motel804-555-1234` is left alone), and documentation example keys (`AKIAIOSFODNN7EXAMPLE` is stripped like a real one).

Dates:

- Month and year only (`March 2024`), bare years, weekdays and relative dates (`next Tuesday`) are not shifted. Dates in non-Latin scripts or other calendars are not detected.
- Dates the model invents are shifted back like any other date.

Addresses:

- A number-first street with no postcode (`12 Via Roma`, `5 Hauptstraße`) when the street word is an ordinary word. A UK street and town with no postcode or country keeps its town (`14 Church Lane, Leeds` masks the street only). A US address with the state spelled out (`123 Calle Ocho, Miami, Florida 33135`) has only its street line replaced; the usual `City, ST 12345` form is replaced whole.
- Addresses written entirely in lowercase, Nordic postcodes with no street line, and countries outside the list in [Surrogates](policy.md#surrogates) (those that are detected get a generic stand-in).
- A two-word town that is not in the built-in list loses its second word from the span when another capitalized word follows.
- A town followed directly by a personal name in a non-US address can get the whole request refused with 422 instead of masked (`Via Roma 10, 00184 Roma Zoë Müller-Öztürk`). A full stop after the town (`00184 Roma. Zoë Müller-Öztürk`) avoids it.
- Stand-ins do not always agree in grammar (an article may not match the gender of the new town), and a rare layout can produce an odd one: a town that is also a first name, directly followed by a person's name, may be replaced as part of the name.

Gateway:

- Numbers sent as JSON numbers are not inspected, in the request body or inside tool call arguments: `"seed": 8045551234` and `{"phone_ext": 8045551234}` as arguments go out as they are. JSON text inside a decoded opaque value is read as text, so `{"phone_ext": 2025550123}` there is caught.
- Opaque values (thinking `signature`, `redacted_thinking.data`, `encrypted_content`, `encrypted_index`) in binary form: a phone number with no cue word next to it is not checked (`a804-555-1234b` inside random bytes passes), and neither is a name with a date (`Elena Petrova born 03/14/1985`).
- Opaque values whose decoded bytes are 80 to 97% printable and also parse as protobuf: single-word names, IP addresses, street addresses and some secret shapes are not checked there. Cards, SSNs, phones and dates next to a cue word, email addresses and full names are.
- A Luhn-valid card number glued to a hex-looking run of 24 or more characters in text-like decoded content passes (`deadbeef4111111111111111`).
- CBOR data: a phone number under a phone key passes when the value is an indefinite-length string sent in chunks, when a tag sits between key and value, or when the key is an integer; a card number in a chunked string passes too. A phone number in a byte string instead of a text string sometimes passes.
- Email addresses in binary data with a one-character local part (`a@example.com`) or a top-level domain that is not on the built-in list.
- Protobuf data full of 16-digit numeric ids is refused with 422 when an id happens to pass the card checksum (it fails closed). Other random protobuf-like values are refused now and then too (a few per 1,000 in a measurement at 4 KB, fewer for smaller values), mostly from fragments that look like handles, phone numbers or assigned secrets.
- Tool definitions are masked like message text, so a value in a tool description that looks personal is replaced there too (in a test with coding-agent style tools, only a placeholder `noreply@example.com` address was). The memo keeps the result byte-stable across requests, but the model sees the stand-in in that description. A schema property name that looks personal blocks every request that carries that tool.
- Identifiers: a value with a known id prefix and a long random tail (`user_a8F3kL9qZ2xW7vB1nM4tY6uR0pE5sD3gH8jK2lQ9`) cannot be told from a secret and is sent under the id check, except under keys ending in `key`. A name joined in lowercase without a separator (`user_elenapetrova`) or a bare digit run (`8045551234123456789`) in an id field is checked but usually not recognized.
- `model` gets only the pattern, dictionary and vault checks: a model value such as `claude-opus-elena-petrova` or `claude:elena@petrova` is not recognized.
- With `policy.allow_images: true`, base64 PDFs, images and OpenAI `file_data` are checked only for their magic bytes and passed unread: image metadata, PDF text and bytes appended after the image data are not inspected.
- The OpenAI Responses API (`/v1/responses`) is not supported. Clients that default to it must be set to use chat completions.
- Detection time grows with text size: a 250 KB single message takes about 2 seconds and a 600 KB one about 10 seconds without a sidecar, on the test host. Text already sent in the same conversation is not detected again.

Streaming and restoring answers:

- Stripped values (secrets, IDs, card numbers) come back as `[redacted ...]`, by design. A value that was only partly recognized and repaired by the leak check can leave a mangled remainder, for example a URL with part of its path redacted.
- If the model writes a real acronym that happens to equal a stand-in initialism from the same vault, and the acronym was not in your request, it is rewritten.

Vault:

- A number keeps up to 32 stored phone spellings, and up to 256 in all when a spelling only restores in its context. Past that, a further unusual spelling of the same number that needs a new stored form is sent as `[redacted phone]` rather than as a stand-in that would restore.
- Text that was already sent is not re-masked when the vault later learns a new name (see [Stable transcripts](endpoints.md#stable-transcripts)).
- Stand-ins for new non-US addresses and new family-name-first names differ from v0.2.0 (see the 0.3.0 notes). Values already in a vault keep the stand-in they had. For roughly one conversation scope in 370, the date shift changed by one day to fix a bug that left dates without a year unshifted. A vault written by this version should not be opened by an older one (see [CHANGELOG.md](../CHANGELOG.md)).

Not measured yet:

- Whether answers are as good with masking as without (needs a real model and a provider key; the harness supports it).
- A full interactive agent session through the gateway with a subscription login.
