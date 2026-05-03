# ESV API — full query parameter reference

Read this file when the user asks for a passage/search option that isn't covered in the main `SKILL.md`. The values listed under "Default" are what the API uses when the parameter is omitted — match them with the **zero value** of your `Options` struct field, and only emit the query param when the caller has explicitly overridden it.

All boolean params are serialized as the literal strings `true` / `false`.

---

## `/v3/passage/text/`

| Parameter                          | Type    | Default | Notes |
|------------------------------------|---------|---------|-------|
| `q`                                | string  | —       | **Required.** Reference like `John 3:16-18`, `Rom 8`, `Ps 23`. |
| `include-passage-references`       | bool    | `true`  | Header line with the canonical reference. |
| `include-verse-numbers`            | bool    | `true`  |       |
| `include-first-verse-numbers`      | bool    | `true`  | Show the verse number on the first verse of a passage. |
| `include-footnotes`                | bool    | `true`  | Footnote markers in the body. |
| `include-footnote-body`            | bool    | `true`  | The footnote text itself, appended after the passage. |
| `include-headings`                 | bool    | `true`  | Section headings inline. |
| `include-short-copyright`          | bool    | `true`  | Short copyright line at the end. |
| `include-copyright`                | bool    | `false` | Full copyright; mutually exclusive with `include-short-copyright`. |
| `include-passage-horizontal-lines` | bool    | `false` | Horizontal rule before each passage. |
| `include-heading-horizontal-lines` | bool    | `false` | Horizontal rule before each heading. |
| `horizontal-line-length`           | int     | `55`    | Character width of the horizontal lines above. |
| `include-selahs`                   | bool    | `true`  | Include "Selah" markers in the Psalms. |
| `indent-using`                     | string  | `space` | `space` or `tab`. |
| `indent-paragraphs`                | int     | `2`     | Spaces (or tabs) for paragraph indentation. |
| `indent-poetry`                    | bool    | `true`  |       |
| `indent-poetry-lines`              | int     | `4`     |       |
| `indent-declares`                  | int     | `40`    | Indent for `[Declares the Lord]` style attributions. |
| `indent-psalm-doxology`            | int     | `30`    |       |
| `line-length`                      | int     | `0`     | `0` = no wrapping. |

---

## `/v3/passage/html/`

All parameters from `/v3/passage/text/` apply (boolean and integer ones). HTML-specific params:

| Parameter                          | Type    | Default | Notes |
|------------------------------------|---------|---------|-------|
| `include-css-link`                 | bool    | `false` | Emit a `<link>` to Crossway's stylesheet. |
| `wrapping-div`                     | bool    | `false` | Wrap each passage in `<div class="esv">`. |
| `div-classes`                      | string  | `passage` | Class names for the wrapper div. |
| `paragraph-tag`                    | string  | `p`     | Element used for paragraphs. |
| `include-book-titles`              | bool    | `false` | Emit a book-title heading at the start. |
| `include-verse-anchors`            | bool    | `false` | `<a name="...">` anchors per verse. |
| `include-chapter-numbers`          | bool    | `true`  |       |
| `include-crossrefs`                | bool    | `false` | Cross-reference markers. |
| `include-subheadings`              | bool    | `true`  |       |
| `include-surrounding-chapters`     | string  | `none`  | `none`, `all`, or `previous-only`. |
| `link-url`                         | string  | _empty_ | Base URL used by anchor links. |
| `crossref-url`                     | string  | _empty_ |       |
| `preface-url`                      | string  | _empty_ |       |

The text-style indent params (`indent-poetry-lines`, etc.) are still honored — they affect the rendered HTML's whitespace.

---

## `/v3/passage/search/`

| Parameter                          | Type    | Default | Notes |
|------------------------------------|---------|---------|-------|
| `q`                                | string  | —       | **Required.** Search query. Quote phrases for exact match: `"love your enemies"`. |
| `page-size`                        | int     | `20`    | Max `100`. |
| `page`                             | int     | `1`     |       |

The search response embeds short snippets per match; there are no formatting options like the passage endpoints.

---

## `/v3/passage/audio/`

| Parameter                          | Type    | Default | Notes |
|------------------------------------|---------|---------|-------|
| `q`                                | string  | —       | **Required.** Same reference syntax as the passage endpoints. |

Response body is `audio/mpeg` (binary). No JSON envelope, no other parameters. Stream the body to the caller as `io.ReadCloser`.

---

## Notes on edge cases

- **Multi-passage queries** (`John 1:1; John 3:16`) return multiple entries in `Passages` (or multiple `results` for search). Don't assume a single element.
- **Unparseable references** return HTTP 400 with a JSON body containing a `detail` string — surface that through `APIError.Body`.
- **Books outside the canon** (e.g. apocryphal books) return 400 with an explanatory `detail`. The ESV API only serves the 66-book Protestant canon.
- **Empty `passages` array**: the reference parsed but matched no text (rare — usually a chapter/verse out of range). Treat as a "not found" condition for the user, not as an HTTP error.
