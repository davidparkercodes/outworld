# What is masked

Each request is walked twice per API. The first walk knows the fields of the format and marks each place as masked (rewritten with stand-ins, then leak-checked), checked (leak-checked, never rewritten; a finding blocks the request with 422), or sent unchanged. A place is sent unchanged only when its value has the exact shape that field allows (listed below); any other value or type is left to the second walk. The second walk visits every string and every object key the first walk did not claim, anywhere in the body, and gives it the full check. Routing rules read the same places as the leak check.

Changed in this release: fields the gateway does not know, and protocol fields with an unexpected value, used to go upstream without any check. A request that carries personal data in such a field is now refused with 422 and names the field, and block rules now see those fields.

There are three kinds of check:

- full: every detector, including the name detector, plus the scan for real values the vault already knows.
- narrow: pattern detectors (emails, phones, IDs, cards, secrets), the date detector, the vault scan and the dictionary, without the name heuristics. For tool and other names the vault scan counts a person only when the name holds two or more of their name words (`notify_kwame_asante`, `notifyKwameAsante`), so a single-word record such as `Edit` left in an old vault does not block every request that carries an `Edit` tool. Used for identifiers chosen by the client: tool and function names, `tool_choice` names, `custom.name`, `json_schema.name`, `mcp_servers[].name`, schema `type`, `format`, `pattern` and `$ref` values, and provider tool results. `Edit` or `Bash` as a tool name passes; `call_8045551234` or a known person's name in one does not. Block rules see these fields by keyword and regex, and entity rules only through those detectors.
- id: narrow without the high-entropy, bare-hex and date patterns (the date pattern stays when the value has separators such as `-`, `_` or `:`). Used only when the value itself looks like an identifier: a known prefix (`toolu`, `srvtoolu`, `mcptoolu`, `call`, `msg`, `file`, `container`, `user`, `resp`, `fc`, `rs`, `item` and similar) plus `_` and 16 or more letters and digits with at least two digits, or `chatcmpl-` plus the same tail; a UUID; a ULID; 16 to 128 lowercase hex characters that are not all digits (hex longer than 40 characters that decodes to mostly printable text is not an id); or the Claude Code `user_<hex64>_account_<uuid>_session_<uuid>` form. It applies to `id`, `tool_call_id`, `tool_use_id`, `container`, short base64 values at the opaque positions below, and keys ending in `id`, `key` or `identifier` in any casing (`user_id`, `userId`, `user-id`), including keys inside `metadata` and tool arguments. Keys ending in `key` other than `prompt_cache_key` and `idempotency_key` keep the high-entropy pattern, since those hold secrets. Any other value in those fields gets the full check, together with a copy split at `_`, `-`, `.` and case changes (`user_ElenaPetrova` is also checked as `user Elena Petrova`) and, for long hex, the decoded text. `model` gets the narrow check without the date pattern.

OpenAI format:

- masked: message `content` (strings and `text`, `input_text`, `output_text` and `refusal` parts), `reasoning_content`, `reasoning`, `refusal`, tool call `arguments` (as JSON values, including keys such as `signature`; arguments with nothing to mask keep their original bytes), `custom` tool call `input`, `annotations[].url_citation.title`, `prompt`, `suffix`, `input`, `prediction.content`, `stop`, `metadata` values, `web_search_options.user_location`, and tool and `response_format.json_schema` schemas as below.
- checked (full): message `name`, every JSON object key, `custom` tool grammar `definition`, image URLs that are not data URLs, file names, and everything the first walk does not know, for example `documents`, `safety_identifier` values that are not hex, extra keys on content parts, and a known field holding an object instead of a string.

Anthropic format:

- masked: `system` text, text blocks and their `citations` (`cited_text`, `document_title`, `title`), `tool_use` inputs, `tool_result` and `search_result` content, `document` text sources and `text/plain` base64 documents (decoded, masked, encoded again), titles and context, `stop_sequences`, the fields of server tools such as `user_location`, and tool and `output_format` schemas as below.
- checked (full): every JSON object key, block types Outworld does not know, thinking blocks outside assistant messages, system blocks that are not `text`, image and document URL sources, and every other field the first walk does not know (extra keys on blocks, citation `url`, other `metadata` keys).
- checked (narrow, without the date pattern): provider server tool results (`web_search_tool_result`, `web_fetch_tool_result`, `code_execution_tool_result`, `bash_code_execution_tool_result`, `text_editor_code_execution_tool_result`, `tool_search_tool_result`) that come after their `server_tool_use` block in the same assistant message, so public names and dates in search results and fetched pages do not block the request while phones, emails, IDs and known names still do. A result type outside that list, one in a user message, or one before or without its call gets the full check. A client can forge the pair, so the narrow check applies to all of its text; the `encrypted_content` of a `web_search_result` gets the opaque value check described below the table. `mcp_tool_result` content is masked like `tool_result`.
- checked (narrow, without the high-entropy and bare-hex patterns): `system[0]` when it is exactly the billing line Claude Code sends, `x-anthropic-billing-header: cc_version=<x.y.z[.hex]>; cc_entrypoint=<word>; cch=<hex>;` (each pair at most once, no others). Any other text after that prefix, or the same line in another system block, is masked like any system text. `cc_entrypoint` keeps the word shape rather than a fixed list, because a wrong list would mask a real entrypoint and break prompt caching.
- sent unchanged, but read by block rules and checked: in assistant messages only, `thinking` blocks (the `thinking` text is checked against the vault, the `signature` gets the opaque value check) and `redacted_thinking` blocks (`data` gets the opaque value check). Any other key on them gets the full check. The thinking text a client sends in an assistant turn is forwarded as is, by design: it is the provider's own earlier output, signed, and changing it would break the signature check. A real value the vault already knows in it still blocks the request.
- checked (narrow): `page_age` of a web search result only when it is not a date or a relative age (`April 3, 2025`, `2025-04-03`, `3 days ago`); those are sent unchanged.

Tool schemas (`tools`, legacy `functions`, `response_format.json_schema.schema`, `output_format.schema`): `description`, `title`, `$comment`, the string values of unknown keywords, and the string values inside `enum`, `examples`, `const` and `default` (including keys such as `signature` inside them) are masked; a tool call that answers with a masked `enum` value is restored to the real one. Property names and `required` get the full check. Numbers and booleans are sent unchanged.

Media (only with `policy.allow_images: true`; otherwise refused with 422): only the known part and block types count as media (`image_url`, `input_audio` and `file` parts on OpenAI, `image` and `document` blocks on Anthropic). Base64 data, including data URLs, must decode to the magic bytes of its declared type (PNG, JPEG, GIF, WebP, PDF, WAV, MP3); data that does not match is refused with 422 `outworld_unsupported_content`. An unknown block with a `source`, or a made-up part type, gets the full check.

Sent unchanged only with this exact shape (anything else gets the full check):

| field | shape |
|---|---|
| `model` | `^[A-Za-z0-9][A-Za-z0-9._:/@\[\]-]{0,127}$`, then the narrow check without the date pattern |
| `service_tier` | `auto`, `default`, `flex`, `priority`, `scale`, `standard_only` |
| `reasoning_effort` | `none`, `minimal`, `low`, `medium`, `high`, `xhigh` |
| `verbosity` | `low`, `medium`, `high` |
| string `tool_choice` | `auto`, `none`, `required`, `any` |
| `tool_choice.type` | `auto`, `any`, `tool`, `none`, `function`, `custom`, `required`, `allowed_tools` |
| message `role` | `system`, `developer`, `user`, `assistant`, `tool`, `function` |
| `strict` (tools, functions, `json_schema`) | a boolean |
| `tools[].type` | `function`, `custom`, `web_search`, `web_search_preview`, `file_search`, `code_interpreter`, `computer_use_preview`, `image_generation`, `mcp`, `local_shell`, or a versioned name `^[a-z_]+_20\d{6}$` |
| `tool_calls[].type` | `function`, `custom` |
| `annotations[].type` | `url_citation` |
| `response_format.type` | `text`, `json_object`, `json_schema` |
| `prediction.type` | `content` |
| custom tool `format.type`, `grammar.syntax` | `text`, `grammar`; `lark`, `regex` |
| `thinking.type` | `enabled`, `disabled`, `adaptive` |
| `output_format.type` | `json_schema` |
| `mcp_servers[].type` | `url` |
| `citations[].type` | `char_location`, `page_location`, `content_block_location`, `web_search_result_location`, `search_result_location` |
| thinking `signature`, `redacted_thinking.data`, web search `encrypted_content`, `citations[].encrypted_index` (assistant messages only) | base64 (standard, or another base64 alphabet for values of 96 characters or more), then the opaque value check below. Invalid base64 gets the full check |
| `context_management.edits[].type` | a versioned name `^[a-z_]+_20\d{6}$` |
| text document `media_type` | `text/plain` |
| media `media_type` | `^[a-z]+/[a-z0-9.+-]+$`, with data that matches it |
| `image_url.detail` | `auto`, `low`, `high` |
| `input_audio.format` | `wav`, `mp3` |
| part and block `type` | the types the first walk recognizes |

Every value sent unchanged is still read by block rules (`match_any`, `match_regex`), except verified media data. Also sent unchanged: `user` on OpenAI requests and `metadata.user_id` on Anthropic requests when `keep_metadata` is false (both are removed before sending), and `mcp_servers[].authorization_token`, a credential meant for the provider, which is forwarded unread except by block rules. With `keep_metadata: true`, `metadata.user_id` gets the id check.

Response headers:

| header | meaning |
|---|---|
| `X-Outworld-Route` | `masked`, `local`, `passthrough`, `block`, or `blocked-leak` |
| `X-Outworld-Route-Reason` | `default`, `header`, or `rule:<name or index>` |
| `X-Outworld-Masked` | number of values transformed |
| `X-Outworld-Dry-Run` (request) | set to `1` to get a report of what would be sent instead of calling the upstream |
| `X-Outworld-Unknown-Names` | surrogate-looking names in the answer the vault does not know (the model probably invented a person). On streams it arrives as an HTTP trailer and an SSE comment `: outworld unknown_names=...` |
