# Cross-References

**Status:** Shipped <!-- Draft | In Progress | Shipped | Deprecated -->
**Created:** 2026-06-03
**Last updated:** 2026-06-03
**Owner:** Darrel

## Why

Cross-references are part of the published ESV text apparatus — the same
verse-to-verse pointers a print study Bible prints in its centre column.
Surfacing them lets a reader follow a thread of scripture (e.g. from
Genesis 1:1 out to Job 38, Psalm 33, John 1, Hebrews 11) without leaving
the reading surface. This serves `PROJECT_CONSTITUTION.md` §1 (focused
study a chapter/section at a time) and §6.1 (it helps someone read
scripture more carefully).

**Why this is not the §5 "no commentary library" non-goal.**
Cross-references are canon-internal pointers from one passage of
scripture to another — the same class of apparatus as the footnotes the
reader already ships. They are not third-party prose, and the popover
renders **ESV verse text only**, never commentary. So they sit inside
scope, not against the "no bundled commentary or study-note library"
non-goal. (If we ever rendered editorial notes *about* the text, that
would cross the line and require a §5 amendment first.)

## Goals

- [x] A reader can turn ESV cross-references on/off in Settings (off by default).
- [x] When on, ESV passages show inline superscript cross-reference markers.
- [x] Clicking a marker opens a popover showing the referenced verse text.
- [x] Works on both the Read tab and the Daily tab.

## Non-goals

- **NIV / other-translation cross-references** — *YouVersion's Platform
  API exposes no cross-reference parameter or data. NIV passages carry no
  markers; the toggle is a silent no-op there. ESV-only by construction.*
- **Navigating the reader to a cross-reference** — *cross-refs are
  verse-level (e.g. "Job 38:4-7"); the reader is chapter-level only
  (`canon.ValidateQuery` rejects verse refs, see passage-reader.md). The
  popover shows the verses in place instead of changing the open chapter.*
- **A persistent cross-reference side panel** — *the reading surface is
  the product (§3); cross-refs are an on-demand popover, not chrome.*

## User-facing behavior

- Settings → **Show** gains a "Cross-references (ESV)" checkbox, **off by
  default** (matches the ESV API default and keeps the default view
  clean).
- With it on (ESV selected), small superscript letters (a, b, c …) appear
  inline in the passage, rendered in the accent color with a pointer
  cursor.
- Clicking a marker opens a small popover anchored beneath it showing the
  referenced verse text (with verse numbers and the passage reference
  label). The popover dismisses on Escape, an outside click, or scroll.
- On NIV, no markers appear and the toggle has no effect.

## Implementation notes

**Two cooperating pieces.**

1. **Toggle → markers.** A new `include_cross_references` toggle
   (`web/src/toggles.ts`, default `false`) flows through the existing
   toggle pipeline: `togglesToQuery` → `?include_cross_references=` →
   `passageHandler` → `scripture.Options.IncludeCrossReferences` →
   `esv.Client` sets `include-crossrefs=true` (and `crossref-url=` empty).
   ESV then emits inline markers: `<sup><a class="cf" href="<refs>/"
   title="<abbrev refs>">a</a></sup>`. There is **no** footer block
   (unlike footnotes); the referenced verses ride in the `href` with full
   book names. `.cf` is styled globally in `web/src/styles/passage.css`.

2. **Click → popover.** The shared `web/src/PassageArticle.tsx` component
   (extracted from the duplicate `<article>` in `App.tsx` and
   `daily/DailyPanel.tsx`) delegates clicks on `a.cf`: it `preventDefault`s
   (the href is a bare reference string, not a navigable URL), strips the
   trailing slash, and calls `fetchCrossref` → `GET /api/crossref`.

**`/api/crossref` endpoint** (`internal/server/crossref.go`). ESV-only by
construction — markers only exist in ESV passages — so there is no
`translation` param; it fetches via `reg.Get(scripture.ESV)`. It returns
the same `{canonical, passages: ["<html>"]}` envelope as `/api/passage`
(consistent with the §4 JSON-API contract — HTML-as-data, no new
pattern), which the SPA renders with the shared `.passage` styling. The
lookup is rendered with headings/footnotes/crossrefs **off** and verse
numbers + passage reference **on**.

**Validation.** The reference string is verse-level, which
`canon.ValidateQuery` deliberately rejects. A sibling validator,
`canon.ValidateRefList`, allows verse syntax (`:`, `,`, `;`-lists, the
en-dash ESV emits) under a length cap and a restricted character set, and
requires the **first** reference to name a canonical book (ESV omits the
book name on same-book continuations, so only the leading reference is
checked). Same rationale as `ValidateQuery`: reject obvious garbage
before an upstream call; ESV is the arbiter of reference grammar. The two
validators share the `splitBookTail` helper.

**Why the href, not the title.** The marker's `title` uses abbreviations
*and omits the book name on continuations* (`Ps. 33:6; 136:5`), which
would break per-reference validation; the `href` carries full book names
and the ESV passage API re-parses the same multi-ref shorthand it
emitted.

**Metrics.** Cross-ref lookups are ESV API calls, so they increment the
existing `esv_api_calls_total` counter.

## Verification

- Go: `go test ./...` (`ValidateRefList`, ESV `include-crossrefs` param,
  `crossrefHandler`, passage-handler param forwarding).
- SPA: `cd web && npm test` (`togglesToQuery`, `PassageArticle` popover
  open/close/error).
- Manual: enable the toggle, load Genesis 1 (ESV) → markers appear; click
  one → popover shows verse text; switch to NIV → no markers.
