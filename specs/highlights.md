# Highlights

**Status:** Shipped
**Created:** 2026-05-04
**Last updated:** 2026-05-04
**Owner:** unassigned

> **Editor's note (2026-05-07):** historical references in this spec to "highlights and notes" predate the removal of the Notes feature on 2026-05-07 (see [notes.md](./notes.md)). The constitution quotes below reflect §1/§2 as they read at the time of this spec; both sections have since been amended to drop notes. Highlights ship and remain in scope.

## Why

Users read scripture in study-help to engage carefully with the text. Highlighting a passage is the most fundamental annotation act — it lets a reader mark what struck them without altering the text itself. This feature delivers the first layer of personal annotation described in PROJECT_CONSTITUTION.md §1 (purpose: "personal highlights and notes") and §2 (in-scope: "Highlighting passages — range-based, persistent, per-user"). It sits directly on top of the accounts feature (specs/accounts.md), using the session cookie and per-user identity already in place, and constitutes the first exercise of the server-authoritative user-data principle (§4).

## Goals

- [x] A signed-in user can select text in a passage, tap "Highlight" in a small floating toolbar, and have the selection saved as a persistent highlight.
- [x] Saved highlights are visually applied to the passage text on every page load — the user sees their prior annotations without any manual step.
- [x] A signed-in user can remove a highlight by tapping it and choosing "Remove highlight" in the toolbar.
- [x] Highlights are per-user and private — one user's highlights are never visible to another.
- [x] A guest who attempts to highlight is shown an inline prompt to sign in; reading is never interrupted.

## Non-goals

- Multiple highlight colors — *single color at v1; a color palette is its own future spec.*
- Shared or public highlights — *explicitly excluded by §5; no social features.*
- Highlights on the Daily reading tab — *deferred until Read tab implementation is stable.*
- Overlapping highlights — *v1 enforces non-overlapping (character-offset level check).*
- Notes attached to highlights — *notes are their own spec; highlights ship standalone.*
- Export or bulk management of highlights — *out of scope at v1.*
- Undo after deletion — *deferred.*

## User-facing behavior

**Selecting and saving.** On the Read tab, the user selects text in the passage by the normal OS gesture. When a non-empty selection is made within the `.passage` container, a small floating toolbar appears near the selection (above when space permits, below otherwise). The toolbar contains a single "Highlight" button. Tapping it saves the highlight, the toolbar dismisses, and the selection is immediately rendered with a yellow background. If the save fails with 409 (overlaps an existing highlight), the toolbar shows the inline error "Overlaps an existing highlight." and the highlight is not applied.

**Selection vs. existing highlight.** A new selection always wins. If the user drags a selection that overlaps one of their existing highlights, the toolbar shows "Highlight"; a tap (no drag) inside an existing highlight shows "Remove highlight." If the new selection truly overlaps, `POST` returns 409 and the inline error appears.

**iOS/iPad touch.** The toolbar is triggered on `touchend` on the passage element (as a fallback to `selectionchange`, which is unreliable on Safari touch). The toolbar appears alongside the native iOS copy/paste bar; the user taps "Highlight" in our toolbar to save.

**Viewing existing highlights.** When the passage HTML is rendered on load, the client fetches the user's highlights for the current passage and applies them as background color to the relevant text spans. Highlights are drawn without interaction affordance until the user taps one.

**Removing a highlight.** Tapping anywhere within an existing highlight re-opens the floating toolbar showing a "Remove highlight" button. Tapping it deletes the highlight from the server and removes the visual styling immediately.

**Guest experience.** If a guest selects text, the toolbar shows "Sign in to highlight" which triggers the existing `AuthPanel`. The in-progress selection is not preserved — accepted at v1.

**Toolbar placement.** Small floating element, z-indexed above the passage, positioned via `getBoundingClientRect`. Dismisses on outside click/tap, Escape, selection clear, book/chapter change, the selection's bounding rect leaving the viewport (IntersectionObserver), or window resize. Tap targets ≥ 44px (Apple HIG, `--tap-target` token).

**Highlight color.** Single semi-transparent yellow (`--color-highlight` design token, light and dark variants), readable over passage text and `.woc` red text.

## Implementation outline

**Range storage:** `(book, chapter, start_verse, start_char_offset, end_verse, end_char_offset)` tuples. Offsets are character positions within the `textContent` of verse elements identified by ESV's `include-verse-anchors` `name` attributes (e.g. `v001003016`). XPath is explicitly avoided. Offsets may drift if the user changes toggle state (`include_footnotes`, `include_headings`) after a highlight is created — v1 accepted limitation, documented here.

**API endpoints (all behind `RequireUser`):**
- `GET /api/highlights?book={book}&chapter={chapter}` — list user's highlights for a passage; each object includes `id, book, chapter, start_verse, start_offset, end_verse, end_offset, created_at`
- `POST /api/highlights` — create a highlight; body: `{book, chapter, start_verse, start_char_offset, end_verse, end_char_offset}`; body is validated against the canon allow-list reused from `internal/esv/`; returns 201 with the highlight object (including `created_at`) or 409 if overlapping
- `DELETE /api/highlights/{id}` — delete by server-assigned ID; returns 204; cross-user IDs return 404 (not 403) to avoid ID-existence enumeration

**DB schema** (`internal/db/migrations/00003_highlights.sql`):
- `highlights` table: `id`, `user_id` (FK → `users`, `ON DELETE CASCADE`), `book`, `chapter`, `start_verse`, `start_offset`, `end_verse`, `end_offset`, `created_at`
- Index on `(user_id, book, chapter)`

**Overlap check:** Character-offset level in Go handler before insert. Two highlights conflict only if their ranges actually intersect (accounting for cross-verse ranges). Enforced in handler, not via DB constraint.

**Backend:** New `internal/highlights/` package — service struct + handler methods + inline SQL, following `internal/auth/` conventions.

**Client** (`web/src/highlights/`):
- `api.ts` — discriminated-union fetchers (`createHighlight`, `deleteHighlight`, `listHighlights`)
- `useHighlights.ts` — hook: fetches on mount and whenever `(book, chapter)` changes (when `user` is non-null); re-fetches after every successful create/delete (no optimistic updates at v1); exposes create/delete
- `HighlightToolbar.tsx` — floating toolbar; handles all button states
- `applyHighlights.ts` — pure function; walks each highlight's range with `TreeWalker` and emits one `<mark class="highlight" data-highlight-id="<id>">` per text node it crosses. Cross-verse highlights produce multiple `<mark>` elements that share the same `data-highlight-id` and are treated as one logical highlight. No library needed.
- `PassageView` extracted from `App.tsx` to host `selectionchange`/`touchend` listeners, `useHighlights`, and `HighlightToolbar` as a sibling. Click handler reads `data-highlight-id` from the tapped `<mark>` and dispatches against the whole id-group (so tapping any span in a multi-verse highlight removes the entire highlight).

**Platform abstraction:** Selection API access (`window.getSelection()`, `Range`) goes through a `web/src/platform/SelectionAdapter.ts` wrapper, mirroring the `ToggleStore` pattern, per §4 platform-abstraction guardrail.

**Design token:** `--color-highlight` added to `web/src/styles/tokens.css` for light and dark themes. `<mark>` override in `web/src/styles/passage.css`.

## Open questions

- [x] ~~Does `include-verse-anchors=true` on the ESV HTML request change existing passage HTML in a way that breaks current CSS? (needs testing against real ESV output; see corresponding Verification item)~~ Resolved 2026-05-04 — flag enabled unconditionally in the server ESV client (it's a server-managed concern, not a user toggle); empty `<a name="...">` anchors don't render visually so existing `.passage` rules are unaffected. Final visual confirmation lives in the Verification list below. See Decisions.
- [x] ~~Should `GET /api/highlights` include `created_at` in the response for client-side sorting?~~ Resolved 2026-05-04 — yes, include `created_at` per object. See Decisions.
- [x] ~~Does switching book/chapter count as "passage navigation" for toolbar dismiss, or should the toolbar also dismiss when the selected text is scrolled out of view?~~ Resolved 2026-05-04 — dismiss on book/chapter change, scroll-out-of-view, and window resize. See Decisions.
- [x] ~~Is there an upper bound on highlights per user per passage at v1?~~ Resolved 2026-05-04 — no cap, no rate limit on highlight endpoints. See Decisions.
- [x] ~~Should Selection API access (`window.getSelection()`, `Range`) be wrapped in `web/src/platform/` (like `ToggleStore` wraps localStorage) per §4's platform-abstraction guardrail, or is inline use acceptable for this feature?~~ Resolved 2026-05-04 — yes, wrap in `web/src/platform/SelectionAdapter.ts`. See Decisions.
- [x] ~~Cross-verse highlights: a `Range` spanning multiple verse elements cannot be wrapped in a single `<mark>` without breaking the DOM tree. How does `applyHighlights` split a multi-verse range into multiple `<mark>` spans, and how does the "tap to remove" gesture treat the split spans as one unit?~~ Resolved 2026-05-04 — multiple `<mark data-highlight-id="<id>">` grouped by id; click handler dispatches against the whole id-group. See Decisions.
- [x] ~~How does the user distinguish "I want to make a new selection that overlaps an existing highlight" from "I tapped on an existing highlight"? Is selection-while-highlight a no-op, or does it take precedence?~~ Resolved 2026-05-04 — new selection always wins; tap (no drag) on a highlight gets "Remove highlight." See Decisions.
- [x] ~~Guest user "Sign in to highlight" flow: spec says the in-progress selection is not preserved. Is this surfaced to the user, or do they discover it by losing their selection?~~ Resolved 2026-05-04 — silent at v1; matches the existing "accepted v1 limitation" framing. See Decisions.
- [x] ~~409 on overlap: what exact inline error copy appears, and does it point the user at the conflicting existing highlight?~~ Resolved 2026-05-04 — copy is "Overlaps an existing highlight."; no conflict-span affordance at v1. See Decisions.
- [x] ~~Ownership guard for `DELETE` returns 404 — confirm 404 (not 403) for cross-user delete attempts (avoids ID enumeration); record explicitly under Decisions.~~ Resolved 2026-05-04 — confirmed 404. See Decisions.
- [x] ~~Does `POST /api/highlights` reuse `internal/esv/`'s canon allow-list validator for body validation, or duplicate it?~~ Resolved 2026-05-04 — reuse `internal/esv/`'s validator (extract/export as needed). See Decisions.

## Decisions

- 2026-05-04: Single highlight color at v1 (no palette). Revisit in a future colors spec.
- 2026-05-04: Range stored as `(book, chapter, start_verse, start_char_offset, end_verse, end_char_offset)`. XPath rejected (breaks on DOM reflow, Safari compat issues).
- 2026-05-04: Offset drift accepted as v1 limitation — if user toggles footnotes/headings after creating a highlight, offsets may shift. Documenting this rather than storing toggle state.
- 2026-05-04: Overlap check at character-offset level in Go handler. Verse-level rejected as too restrictive (would block two non-overlapping highlights in the same verse).
- 2026-05-04: Server-authoritative storage; client holds passage-scoped cache only (§4).
- 2026-05-04: Three endpoints — `GET`, `POST`, `DELETE /api/highlights`. No `PATCH` at v1.
- 2026-05-04: New `internal/highlights/` package following `internal/auth/` conventions.
- 2026-05-04: Guest users trigger existing `AuthPanel` — same callback as `AuthChip`. Selection lost on panel open; accepted at v1.
- 2026-05-04: Highlights applied client-side via `applyHighlights` (Range + Text node splitting), not server-side HTML transformation (respects §3 — passage proxy stays stateless).
- 2026-05-04: iOS touch trigger via `touchend` on the passage element as fallback to `selectionchange`, which is unreliable in Safari.
- 2026-05-04: Selection API access wrapped in `web/src/platform/SelectionAdapter.ts` following the `ToggleStore` pattern, per §4 platform-abstraction guardrail. Inline use rejected: §4 says browser APIs go through a thin interface.
- 2026-05-04: Client cache strategy — `useHighlights` re-fetches `GET /api/highlights` on `(book, chapter)` change and after every successful `POST`/`DELETE`. No optimistic updates at v1; the server stays the source of truth and the extra round-trip cost is acceptable for a per-passage cache.
- 2026-05-04: Offset drift after a footnotes/headings toggle change is silent at v1 — highlight may render at a shifted position; no inline UI notice and no drop-on-mismatch logic. Documented as a v1 limitation; revisit if users report it.
- 2026-05-04: `GET /api/highlights` includes `created_at` per object. Cheap (already stored), enables future client-side sort/group, and avoids a one-way door if omitted now.
- 2026-05-04: No cap on highlights per user per passage at v1, and no rate limit on the highlight endpoints. Trust signed-in users; revisit if abuse appears. Bounded GET payload size accepted as a v1 risk.
- 2026-05-04: `DELETE /api/highlights/{id}` returns 404 (not 403) for IDs owned by another user. Avoids ID-existence enumeration; matches the existing Verification item.
- 2026-05-04: `POST /api/highlights` reuses `internal/esv/`'s canon allow-list validator for body validation. Extract or export as needed; one source of truth avoids drift between query-string and body validation.
- 2026-05-04: Toolbar dismiss triggers expanded to include book/chapter change, the selection's bounding rect leaving the viewport (IntersectionObserver), and window resize, in addition to outside click/Escape/selection-clear.
- 2026-05-04: Selection-vs-existing-highlight precedence — a new drag selection always wins (toolbar shows "Highlight"); a tap (no drag) inside a highlight shows "Remove highlight." If the new range actually overlaps, `POST` returns 409 and the inline copy "Overlaps an existing highlight." appears. Rejected: showing both buttons (uglier, fights the user) and existing-highlight-wins (UI feels stuck).
- 2026-05-04: Guest "Sign in to highlight" flow loses the in-progress selection silently — no toolbar copy change, no selection preservation across `AuthPanel` open. Re-selecting after signin is fast; not worth UI debt at v1.
- 2026-05-04: 409 inline error copy is "Overlaps an existing highlight." No conflict-span pulse/outline at v1.
- 2026-05-04: Cross-verse highlights render as multiple `<mark class="highlight" data-highlight-id="<id>">` spans (one per text node the range crosses) sharing the same `data-highlight-id`. PassageView's click handler reads the id and treats all matching spans as one logical highlight for the "tap to remove" gesture. Rejected: single `<mark>` wrapping a multi-element Range (breaks DOM); per-verse server-side rows (breaks "one user gesture = one highlight" model).
- 2026-05-04: Character ranges are half-open `[start, end)` (Selection API convention). Two highlights touching exactly at a boundary do not conflict; the overlap check uses lexicographic comparison on `(verse, offset)`.
- 2026-05-04: `esv.LookupBook(name)` extracted as the shared canon allow-list helper (case-insensitive, alias-aware). Both `internal/esv/ValidateQuery` and `internal/highlights/` body validation route through it.
- 2026-05-04: `include-verse-anchors=true` is sent unconditionally by the server-side ESV client (not a user toggle). Verse anchors are infrastructure for highlights; exposing them as a toggle would risk highlights being un-renderable when the toggle is off. Empty `<a name="...">` anchors are visually inert in modern browsers, so passage CSS is unaffected.
- 2026-05-04: Client refactors the rendered passage out of `App.tsx` into `web/src/highlights/PassageView.tsx`, and adds `web/src/platform/SelectionAdapter.ts` (mirroring `ToggleStore`). Highlight overlay is applied via `applyHighlights` against the live DOM after each passage/highlights change; tap-to-remove dispatches by `data-highlight-id`.
- 2026-05-05: `web/src/highlights/parseSelection.ts` is now a per-`TranslationID` dispatcher (landed with [niv](./niv.md)). `listVerseAnchors`, `rangeToTuple`, `tupleToRange`, and `applyHighlights` all take a `translation` argument and route to the right anchor lister (ESV's `<a class="va" rel="...">`, YouVersion's `<span class="yv-v" v="N">`, etc.). The shared text-walking logic (`offsetWithin`, `textNodeAtOffset`) stays anchor-agnostic — adding a translation is one new lister function plus one entry in the dispatcher map. Highlights work for any registered provider with no further changes to this package.

## Verification

- [ ] Unit tests (`internal/highlights/`): insert/list round-trip; delete removes only the target row; character-offset overlap detection rejects intersecting ranges and allows non-overlapping ranges in the same verse.
- [ ] Auth guard tests: unauthenticated requests to all three endpoints return 401.
- [ ] Ownership guard: `DELETE /api/highlights/{id}` with a different user's ID returns 404.
- [ ] Client unit test: `applyHighlights` wraps correct spans given known HTML + highlight list.
- [ ] Client unit test: `HighlightToolbar` shows correct button state for each scenario (no highlight, existing highlight, guest user).
- [ ] Manual flow (signed in): select → highlight → reload → still visible → remove → reload → gone.
- [ ] Manual flow (guest): select → "Sign in to highlight" → `AuthPanel` opens.
- [ ] Manual flow on iPad Safari: touch selection triggers toolbar; tap targets ≥ 44px; scroll not blocked.
- [ ] Cross-verse highlight: select, save, reload — both verses correctly highlighted.
- [ ] Overlap rejection: POST returns 409; client shows inline error, no highlight applied.
- [ ] Dark theme: `--color-highlight` dark token readable against `--color-bg: #14130f`.
- [ ] Toggle interaction: document or verify offset behavior when `include_footnotes` is toggled.
- [ ] Manual: render a passage with `include-verse-anchors=true` and confirm `web/src/styles/passage.css` rules (verse numbering, woc, footnote refs) still match. (Resolves the corresponding Open question once verified.)
- [ ] Cross-verse rendering: a multi-verse highlight produces multiple `<mark>` elements with the same `data-highlight-id`; tapping any one of them removes the entire group server-side and visually.
- [ ] Toolbar dismiss: book/chapter change, scrolling the selection out of view (IntersectionObserver), and window resize each dismiss the toolbar.
- [ ] 409 copy: client renders the literal string "Overlaps an existing highlight." in the toolbar inline error slot.
- [ ] Selection-vs-tap precedence: dragging a selection across an existing highlight shows "Highlight" (not "Remove highlight"); tapping inside it (no drag) shows "Remove highlight."
- [ ] `GET /api/highlights` response includes `created_at` on each object (RFC3339 string).

## Related

- `specs/accounts.md` — sessions, `RequireUser`, `AuthPanel`, `useUser`
- `specs/passage-reader.md` — passage HTML structure, ESV proxy, `.passage` CSS class, verse toggle behavior
- `PROJECT_CONSTITUTION.md §1, §2, §3, §4, §5`
- ESV API `include-verse-anchors` parameter
- W3C Selection API, Safari touch selection behavior
