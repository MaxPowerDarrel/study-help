# Daily annotations

**Status:** Shipped
**Created:** 2026-05-05
**Last updated:** 2026-05-06
**Owner:** unassigned

## Why

The Daily tab (shipped per [auto-load-daily-reading](./auto-load-daily-reading.md)) currently renders the day's passages as a plain HTML blob — no selection wiring, no highlight overlay, no notes drawer, and no visible translation picker. A reader who wants to annotate the day's reading must switch back to the Read tab, navigate to the same passage, and re-orient. This feature brings the Read-tab annotation surface and translation control to the Daily tab so the daily reading is a first-class study surface, not a read-only preview. Supports PROJECT_CONSTITUTION.md §1 (purpose: personal highlights and notes), §2 (in-scope: highlights, notes, multi-translation), and §3 (Respect the text — annotations annotate, never mutate).

## Goals

- [x] On the Daily tab, a signed-in user can select text in any rendered chapter and tap "Highlight" or "Add note" in the same floating toolbar used on the Read tab.
- [x] Saved highlights are visually applied to the matching chapter on every load of the Daily tab.
- [x] The Notes drawer, when opened on the Daily tab, lists notes from every chapter in the active pill (e.g., Genesis 1 + 2 + 3 for "Genesis 1-3"). Tap-to-scroll navigates to the right per-chapter container.
- [x] A signed-in user can change the active translation directly from the Daily tab via a picker in the daily header. The choice persists (`PATCH /api/auth/me`) and is mirrored on the Read tab.
- [x] Annotations created on the Daily tab are visible on the Read tab (and vice versa), because both surfaces read from the same `(user_id, translation, book, chapter)`-scoped server data.
- [x] A guest who attempts to highlight or change translation on the Daily tab gets the same affordances they get on the Read tab (sign-in prompt for the toolbar; disabled picker with "Sign in to choose" hint).

## Non-goals

- Annotations on the daily plan summary itself (the day-level header card) — *annotations attach to scripture only.*
- Cross-chapter selection (e.g., dragging a selection from Genesis 2:30 into Genesis 3:1) — *each chapter renders into its own container, so selections cannot cross containers; matches Read-tab single-chapter posture.*
- A new aggregated annotations API endpoint — *no `/api/notes?chapters=1,2,3`; client fans out to existing per-chapter endpoints.*
- New annotation types (multi-color highlights, tags, threads) — *each is its own future spec.*
- Push notifications, daily-reading reminders, or shared/social annotations — *§5 non-goals; out of scope.*
- Lazy/virtualized rendering of per-chapter `PassageView`s — *up-front render is simpler and matches Read-tab posture; revisit only if 3+ chapter days reveal a perf problem.*
- Server-side daily-reading schedule changes — *daily plan source remains the embedded markdown; this feature is a client-side annotation surface, not a content change.*

## User-facing behavior

**Translation picker on Daily.** The daily header (currently date nav + plan name) gains a compact `<select>` listing the same translations as the Read-tab sidebar picker (`web/src/translations/catalog.ts`). For signed-in users, changing it triggers `PATCH /api/auth/me`, re-fetches all visible chapter passages in the new translation, and the highlights/notes for that translation render (per-translation isolation per [multi-translation](./multi-translation.md)). For guests the picker is disabled with the same "Sign in to choose" inline hint as the Read tab.

**Per-chapter rendering.** Each pill (OT/NT) expands its `chapters[]` into a stack of chapter blocks. Each block is a full `PassageView` instance — its own selection wiring, its own highlight overlay, its own toolbar. A pill labelled "Genesis 1-3" renders three stacked blocks (Genesis 1, Genesis 2, Genesis 3); a pill labelled "Romans 1" renders one. Per-chapter passage fetches are issued concurrently and each block shows its own spinner until its fetch resolves; chapters render as they arrive (a slow chapter does not block earlier chapters).

**Selection → Highlight / Add note.** Identical to the Read tab. Selecting text in any chapter opens the floating `HighlightToolbar` near the selection, with "Highlight" and "Add note" buttons. Tap "Highlight" → selection saved as a highlight scoped to that chapter, mark applied immediately. Tap "Add note" → notes drawer opens with composer pre-anchored to the selection's `(book, chapter, start_verse, start_offset, end_verse, end_offset)`. Existing-highlight tap shows "Remove highlight" exactly as on the Read tab.

**Notes drawer.** The header "Notes" toggle (currently in `web/src/App.tsx`'s app-header right cluster) opens the drawer on the Daily tab too. The drawer lists notes from **every chapter visible in the active pill**, sorted (proposed: by chapter ascending, then `created_at` ascending). Tapping an entry scrolls to that note's anchor in the right per-chapter container. If the anchor's chapter is not currently rendered (the note belongs to the other pill — e.g., a Romans 1 note while the Genesis 1-3 pill is active), the drawer auto-switches to the pill that contains the chapter and then scrolls. Inline edit and delete behave as on the Read tab.

**Pill switch.** Switching OT ↔ NT swaps the visible chapter set. The drawer re-aggregates over the new pill's chapters. In-progress note composition is discarded on pill switch (matches Read-tab book/chapter discard).

**Date navigation.** Existing ←/→/date input behavior is unchanged. Switching dates clears the daily state and re-fetches; in-progress note composition is discarded.

**Annotation parity across surfaces.** A highlight created on the Daily tab in Genesis 2:5 is the same row as a highlight created on the Read tab in Genesis 2:5 (same `(user_id, translation, book, chapter, start_verse, start_offset, end_verse, end_offset)`). No tab-scoped annotations.

## Implementation outline

**Server.** No changes. `/api/passage` already accepts `?translation=` and is chapter-scoped. `/api/highlights` and `/api/notes` already filter by `(user_id, translation, book, chapter)`. `/api/auth/me` PATCH already persists translation.

**Client (`web/src/`).**

- **`App.tsx` daily fetch effect** (currently lines 107–187): replace per-pill "one fetch for combined `assembleQ(p)`" with "one fetch per `(book, chapter)` in `p.chapters[]`". State shape moves from `DailyTabState.{q, html, loading, error}` to `DailyTabState.chapters: Array<{book, chapter, html, loading, error}>`. `Promise.all` over the chapter list per pill; per-chapter loading/error tracked independently.

- **`App.tsx` `DailyPanel`** (currently lines 559–695): replace the plain `<article dangerouslySetInnerHTML>` with a stack of `PassageView` instances. Each chapter has its own `articleRef` held in a `Map<string, RefObject>` keyed by `${book}:${chapter}`. Pass the existing `PassageView` props (`html`, `book`, `chapter`, `translation`, `isSignedIn`, `showWordsOfChrist`, `onGuestSignin`, `onAddNote`, `articleRef`) per instance.

- **`onAddNote` signature change**: `PassageView`'s `onAddNote` callback gains `book` and `chapter` arguments — `(tuple, book, chapter) => void`. Today the chapter is implicit from `App.tsx:ref.chapter`; making it explicit lets the Daily tab route the pending note to the correct per-chapter `useNotes` instance. `HighlightToolbar` propagates the change. Read-tab call sites pass `book.name` and `ref.chapter`.

- **`web/src/notes/useDailyNotes.ts` (new)**: composing hook with the same surface as `useNotes`. Signature: `useDailyNotes(book: string | null, chapters: number[], translation, enabled)`. Fans out `listNotes(book, c, translation)` for each chapter; merges results into one sorted `Note[]`; exposes `create / update / remove` that route to the existing per-note API and refresh the aggregate on success. Re-fetches on `(book, chapters.join(','), translation, enabled)` change.

- **`web/src/notes/NotesDrawer.tsx`**: replace the single `articleRef` prop with `findArticle(book: string, chapter: number): HTMLElement | null`. The Read tab supplies a function that returns its single ref; the Daily tab supplies a function that looks up the per-chapter ref map. Tap-to-scroll uses `findArticle(n.book, n.chapter)` then `tupleToRange` against that container.

- **Translation picker on Daily**: a compact `<select>` rendered inside `DailyPanel`'s header, alongside `dateNav`. Reuses the existing `translation` and `setTranslation` already threaded via prop (`App.tsx:572`). Same disabled-for-guests treatment as Read.

- **CSS**: add daily-header picker styles (mirror Read-tab sidebar picker); add vertical spacing between stacked `PassageView`s and per-chapter chapter labels (so the user can tell "this is Genesis 2" when scrolling).

**No DB changes; no migration.** All annotation rows already carry `translation, book, chapter`.

## Open questions

- [x] ~~Per-chapter loading affordance: single pill-level spinner until all chapter fetches resolve, or per-chapter spinners that let earlier chapters render as soon as they arrive?~~ Resolved 2026-05-05 — per-chapter spinners. See Decisions.
- [x] ~~When tap-to-scroll targets a chapter that's not rendered (e.g., note's chapter belongs to a different pill), should the drawer toast "Switch to {pill} to view"? Or silently no-op?~~ Resolved 2026-05-05 — auto-switch pill, then scroll. See Decisions.
- [x] ~~Multi-pill (rare) days: are there days with three or more passages?~~ Resolved 2026-05-05 — no. The daily-reader markdown table has only OT and NT columns (`internal/dailyreader/dailyreader.go:14–18,43–49`); maximum two pills, possibly one if a row has only OT or only NT. See Decisions.
- [ ] On pill switch (OT → NT), should the drawer remain open and re-aggregate, or close? Read-tab equivalent is book/chapter change which keeps it open.
- [ ] Note sort order in the aggregated drawer — by `(chapter ASC, created_at ASC)`, or by `created_at` only (so newest at bottom regardless of chapter)? The first respects the reading flow; the second matches Read-tab today (single-chapter, so the question doesn't arise there).
- [ ] Should each per-chapter `PassageView` render a chapter heading (e.g., "Genesis 2") above the passage HTML, or rely on the ESV/YouVersion HTML's own chapter markers? Provider HTML may or may not include headings depending on toggles.
- [ ] Translation picker on Daily for guests: mirror Read-tab disabled-with-hint, or hide entirely? Mirror is consistent.
- [ ] Should the `onAddNote(tuple, book, chapter)` signature change land as a dedicated refactor commit before the Daily wiring, so Read-tab regressions are isolable?
- [ ] Is per-chapter concurrency bounded? `Promise.all` over an unbounded `chapters[]` could fan out widely on hypothetical large-chapter days; is a small concurrency cap warranted?
- [ ] When a user changes translation on Daily, are pending in-progress notes/highlights discarded the same way pill-switch and date-switch discard them?

## Decisions

- 2026-05-05: Render multi-chapter daily pills as a stack of per-chapter `PassageView`s, not a single combined article. Reason: existing verse-anchor parser dedupes by verse only (no chapter awareness), so a combined container would mis-attach highlights when the same verse number repeats across chapters (1:1, 2:1, 3:1 in "Genesis 1-3"). Per-chapter fetches parallelize via `Promise.all`, so wall-clock latency is unchanged. Rejected alternatives: chapter-aware refactor of `parseSelection` (touches Read-tab code paths for no Read-tab benefit); disabling annotations on multi-chapter days (most daily readings are multi-chapter, so this would gut the feature).
- 2026-05-05: Aggregate notes across visible chapters in the active pill via a new `useDailyNotes` composing hook that fans out `listNotes` per chapter. Read-tab `useNotes` unchanged. Rejected: extending the server with a `?chapters=1,2,3` parameter (premature; client fan-out is small and parallel).
- 2026-05-05: Translation picker placement — Daily header next to date nav. Rejected: shared app-header picker (bigger UI refactor, affects Read-tab layout); gear menu (less discoverable; users won't realize translation is changeable on Daily).
- 2026-05-05: `onAddNote(tuple, book, chapter)` — chapter context flows from `PassageView` → drawer explicitly instead of being implicit from `App.tsx:ref.chapter`. Read tab call sites pass current ref values. Rejected: Daily-tab-only callback overload (asymmetric, easier to drift).
- 2026-05-05: `findArticle(book, chapter)` lookup replaces the single `articleRef` prop on `NotesDrawer`. Read tab supplies a function returning its single ref; Daily supplies a function reading from the per-chapter ref map. Rejected: passing a `Map<string, RefObject>` directly (couples Read tab to a multi-chapter abstraction it doesn't need).
- 2026-05-05: Per-chapter `PassageView`s are rendered up front (not lazily). Daily readings are 1–3 chapters; lazy rendering is premature optimization. Re-evaluate if a future plan introduces 5+ chapter days.
- 2026-05-05: Per-chapter spinners while chapters load. Each block renders its own spinner; chapters appear as their fetches resolve. Rejected: single pill-level spinner (a slow chapter would block the rest of the pill from rendering). `DailyTabState.chapters[]` carries a per-entry `loading` flag.
- 2026-05-05: Drawer tap-to-scroll on a hidden chapter auto-switches to the pill that contains the chapter and then scrolls. Rejected: silent no-op (looks broken); toast "Switch to {pill} to view" (extra tap for no benefit). Implementation: drawer needs a `findPillForChapter(book, chapter)` helper supplied by `DailyPanel`.
- 2026-05-05: Maximum two pills per day (OT, NT). The dailyreader plan markdown only has OT and NT columns (`internal/dailyreader/dailyreader.go`), and a row may have one or both populated. No 3+ pill case to spec.
- 2026-05-06: Normalize daily-reader book names to canonical canon names in `internal/dailyreader/splitPassage` via `canon.LookupBook`. The plan markdown uses common abbreviations ("Num.", "Matt.", "Rev.", "1 Sam.", "Song of Songs") that the ESV API silently accepts but YouVersion's USFM mapping rejects (no entry for "num.", etc.). Pre-existing latent bug — surfaced by the new Daily-tab translation picker because the picker now lets users select NIV from Daily, triggering YouVersion fetches for abbreviated names. Fix is at the source so every provider sees canonical names. New test `TestSplitPassageNormalizesEveryPlanRow` walks the entire plan and asserts every book resolves to its canonical name.
- 2026-05-06: Implementation landed in PR #30 (`feat(daily): highlights, notes, and translation picker on Daily tab`). Status flipped to Shipped.
- 2026-05-06: Post-ship cleanup (PR #35) extracted the daily-tab logic out of `App.tsx` into a new `web/src/daily/` directory: `useDailyTab.ts` owns the load/reset effects, `fetchId` race guard, per-chapter article-ref `Map<string, RefObject>`, and active-pill switching; `DailyPanel.tsx` (with `DailyChapterBlock`) owns the panel render. The ref map and date helpers no longer live in `App.tsx`. The `Implementation outline` above describes the pre-ship design — file/line refs in that section are historical. `App.tsx` dropped from 941 to 450 lines.

## Verification

- [ ] Manual flow (signed in): Daily tab, pick 01/01/26 ("Genesis 1-3" / "Romans 1"). Confirm three OT chapter blocks stacked, one NT block, translation picker visible in daily header.
- [ ] Manual flow (signed in): select text in Genesis 2:5 → toolbar appears → "Highlight" → mark renders, persists on reload.
- [ ] Manual flow (signed in): select text in Genesis 3:1 → "Add note" → drawer opens → save → entry appears in drawer.
- [ ] Manual flow: drawer aggregation — confirm notes from Genesis 1, 2, and 3 are all listed in the drawer when the OT pill is active.
- [ ] Manual flow: drawer tap-to-scroll — tap a note from Genesis 2 → page scrolls to its anchor inside the Genesis 2 container.
- [ ] Manual flow: pill switch OT → NT → drawer re-aggregates over Romans 1 only.
- [ ] Manual flow: translation switch ESV → NIV — passages re-render in NIV; ESV-translation annotations hide; NIV-translation annotations (if any) appear.
- [ ] Cross-surface parity: highlight created on Daily/Genesis 2 visible on Read tab when navigating to Genesis 2.
- [ ] Manual flow (guest): selection prompts sign-in; translation picker disabled with hint.
- [ ] Vitest: `useDailyNotes` aggregates listNotes calls across chapter list and refreshes on mutation.
- [ ] Vitest: `DailyPanel` renders N `PassageView` instances for an N-chapter pill.
- [ ] Vitest: `NotesDrawer` `findArticle` lookup chooses the correct per-chapter container.
- [ ] `go test ./...` unchanged (no server changes).
- [ ] Latency: a 3-chapter day's pill should resolve in roughly the time of one passage fetch (parallel), not 3x.
- [ ] Mobile/touch: floating toolbar and drawer scroll behave correctly across stacked chapter blocks (no scroll-jacking between containers).

## Related

- [auto-load-daily-reading](./auto-load-daily-reading.md) — daily fetch flow, pill UX, date navigation
- [highlights](./highlights.md) — range model, toolbar, `applyHighlights`, `SelectionAdapter`
- [notes](./notes.md) — drawer, composer, `useNotes`, `NotesDrawer`, `tupleToRange`
- [multi-translation](./multi-translation.md) — per-translation isolation, picker UX, `Attribution`
- [niv](./niv.md) — per-`TranslationID` anchor dispatcher in `parseSelection.ts`
- `PROJECT_CONSTITUTION.md §1, §2, §3, §4`
