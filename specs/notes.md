# Notes

**Status:** Shipped <!-- Draft | In Progress | Shipped | Deprecated -->
**Created:** 2026-05-04
**Last updated:** 2026-05-05
**Owner:** Darrel

## Why

Lets a signed-in user attach a private written note to a passage so
they can record observations, questions, and connections during slow
reading and find them again on subsequent visits to the same chapter.
Supports `PROJECT_CONSTITUTION.md` §2 (in-scope: notes attached to
passages, per-user), §3 (Study-first UX, Respect the text — notes
annotate, never mutate scripture), and §4 (User data is
server-authoritative — notes persist server-side).

## Goals

User-visible outcomes that define success. Bullets, not paragraphs.

- [ ] A signed-in user can attach a note to a selected verse range within a chapter and have it persist across sessions and devices.
- [ ] Notes for the current chapter are discoverable via an on-demand drawer that does not crowd the reading surface when closed (§3 Study-first UX).
- [ ] A user can edit and delete their own notes from the drawer.
- [ ] Notes are private — only the author sees them (§2 per-user, §5 no-sharing).

## Non-goals

Things explicitly excluded from this feature, with one-line rationale
each. Forces scope honesty — what we're *not* doing is as important as
what we are.

- Rich text / markdown rendering — *plain text keeps the surface simple at v1; can revisit if a real need surfaces*
- Sharing or exporting notes — *§5 explicitly excludes social/sharing features*
- Cross-passage linking or tags — *adds taxonomy complexity that doesn't serve focused study at v1*
- Attachments (images, audio) — *out of scope for a text-first reader*
- Offline editing with conflict resolution — *§5 no offline-first sync engine at v1*

## User-facing behavior

Concrete description of what the user sees and does. Not implementation.
A senior PM should be able to read this and know what shipping looks
like.

A signed-in user selecting text inside the reading surface gets an
"Add note" affordance alongside the existing highlights toolbar; a
note anchors to the same verse-offset range scheme highlights use
(book, chapter, start verse + offset, end verse + offset). Notes
and highlights are independent — selecting "Add note" creates only
a note, never a highlight. A toggle in the reading-surface chrome
opens an on-demand drawer that lists the current chapter's notes,
ordered by anchor verse so drawer entries match reading flow; the
drawer is closed by default so the reading surface stays unchanged
(§3 Study-first UX). Each drawer entry shows the anchored passage
reference, the author's plain-text body, the created date (and an
"edited" indicator when the note has been updated), and edit/delete
controls for *their own* notes. Editing happens inline in the
drawer entry — Edit replaces the body with a textarea and Save /
Cancel buttons, no separate modal or pane. Tapping a drawer entry
scrolls the reading surface to its anchor range. A guest (not
signed in) sees no "Add note" affordance and the drawer toggle is
hidden — same posture as highlights. The drawer's empty / loading /
error states match the rest of the app: silent empty when a chapter
has no notes, a centered spinner while loading, and a generic toast
on failure ("Service is busy, try again in a moment" on 429;
"Something went wrong, try again" on other errors).

## Implementation outline

High-level shape only:

- New package `internal/notes/` mirroring `internal/highlights/`: handler, store, canon validation via `esv.LookupBook`. Behind `auth.RequireUser`.
- New endpoints: `GET /api/notes?book&chapter` (returns notes ordered by `start_verse ASC, start_offset ASC`), `POST /api/notes`, `PATCH /api/notes/{id}` (edit body; bumps `updated_at`), `DELETE /api/notes/{id}`. All four behind `auth.RequireUser`; cross-user access returns 404 (no ID enumeration), same posture as highlights. Body length cap of 16 KB enforced server-side on POST/PATCH; over-cap requests rejected with HTTP 400. No rate limiting at v1.
- New goose migration `00004_notes.sql`. Minimum viable schema sketch: `notes(id INTEGER PK, user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE, book INTEGER NOT NULL, chapter INTEGER NOT NULL, start_verse INTEGER NOT NULL, start_offset INTEGER NOT NULL, end_verse INTEGER NOT NULL, end_offset INTEGER NOT NULL, body TEXT NOT NULL, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL)` with index `(user_id, book, chapter)`. Half-open range semantics match `internal/highlights/`. Unlike highlights, notes do **not** require non-overlapping ranges — multiple notes on the same passage are explicitly allowed.
- Client: `web/src/notes/` with `api.ts` (discriminated-union fetchers for the four endpoints), `useNotes.ts` (per-passage cache; refetches on book/chapter change and after every successful mutation), `NotesDrawer.tsx` (the on-demand drawer + entry list + edit/delete controls), and a small create-note path that reuses `web/src/highlights/parseSelection.ts` + `SelectionAdapter` to convert the user's Range into the verse-offset tuple sent to `POST /api/notes`.
- No new platform abstraction needed beyond what `SelectionAdapter` and `ToggleStore` already provide.

Pointers, not full designs. The spec is durable; the design notes can
stay in the PR.

## Open questions

Questions that need to be resolved. When answered, move the resolution
under **Decisions** and leave a pointer here (don't delete the question
— its existence is part of the history).

- [x] Attachment grain: per-chapter, per-verse, or per-range (like highlights)? Each grain implies a different schema and a different UI affordance. — *resolved 2026-05-04 (per-range, see Decisions)*
- [x] Reading-surface integration: inline indicators next to the verse (badges, margin marks), a side panel that lists notes for the chapter, an on-demand drawer triggered by a button — or some combination? — *resolved 2026-05-04 (on-demand drawer, see Decisions)*
- [x] How many notes can a single user attach to one passage / chapter? Cap at v1 or unlimited? — *resolved 2026-05-04 (unlimited at v1, see Decisions)*
- [x] Note body length cap (e.g. 4 KB, 16 KB) — and what happens on overflow (reject 400, truncate, multi-note)? — *resolved 2026-05-04 (16 KB, reject 400, see Decisions)*
- [x] Plain text only at v1, or basic markdown? (Non-goals lists plain text; this is the formal Open question to confirm.) — *resolved 2026-05-04 (plain text confirmed, see Decisions)*
- [x] Editing model: in-place inline editor, modal, dedicated edit panel? Does editing bump a `updated_at`? — *resolved 2026-05-04 (in-drawer inline editor; PATCH bumps `updated_at`, see Decisions)*
- [x] Do notes interact with highlights (e.g. attach a note to an existing highlight) or are the two surfaces independent at v1? — *resolved 2026-05-04 (independent at v1, see Decisions)*
- [x] Rate-limiting posture for `POST /api/notes` — reuse the per-IP + per-account dual bucket from auth, or no limit at v1 like `/api/passage`? — *resolved 2026-05-04 (no rate limit, matches highlights precedent, see Decisions)*
- [x] What does the empty/loading/error UX look like on the notes surface — silent empty, spinner on load, generic toast on failure (matching the rest of the app)? — *resolved 2026-05-04 (match existing app patterns, see Decisions)*
- [x] Does `GET /api/notes` accept the same `book&chapter` shape as highlights, and what's the response order — created, updated, or anchor-verse ascending? — *resolved 2026-05-04 (same `?book&chapter` shape; ordered by `start_verse, start_offset` ascending, see Decisions)*

## Decisions

Append-only log. Most recent at the bottom. Never rewrite past entries;
if a decision is reversed, add a new entry that supersedes it.

- 2026-05-04: Spec drafted; status Draft. Owner: Darrel.
- 2026-05-04: Note attachment grain is per-range — book, chapter, start verse + offset, end verse + offset (half-open), matching the `internal/highlights/` scheme. Reason: gives the richest study-first UX (notes can pin to a sentence or clause, not just a chapter), and reuses `SelectionAdapter` + `parseSelection.ts` so the create path is a known pattern. Unlike highlights, notes do **not** require non-overlapping ranges — multiple notes on the same passage are intentional. Resolves Open question on grain.
- 2026-05-04: Reading-surface integration is an on-demand drawer toggled from the reading-surface chrome; closed by default. Reason: keeps the reading surface unchanged when not in use (§3 Study-first UX), gives notes a dedicated home without consuming layout, and avoids inline indicator clutter on chapters with many notes. Inline indicators were considered and rejected at v1; they can be added later without changing the schema. Resolves Open question on reading-surface integration.
- 2026-05-04: Note body cap is 16 KB; server rejects POST/PATCH bodies that exceed it with HTTP 400 (no truncation, no multi-note splitting). Reason: 16 KB is well above any realistic single-note length while keeping storage and validation cheap; reject-on-overflow is the simplest enforcement and matches the rest of the app's error posture. Resolves Open question on body length cap.
- 2026-05-04: Note body format is plain text at v1; no markdown parsing or rendering. Locks in the existing Non-goals entry. Reason: simplest end-to-end (no parser, no renderer, no XSS surface, no styling decisions); revisit only if a real reader use-case surfaces. Resolves Open question on plain text vs. markdown.
- 2026-05-04: Editing model is an in-drawer inline editor — tapping Edit on a drawer entry replaces its body with a textarea + Save/Cancel; everything stays inside the drawer. Successful PATCH bumps `updated_at`; the drawer entry shows "edited" state once `updated_at > created_at`. Reason: keeps the drawer the single home for note interactions, avoids introducing a third surface (modal or pane), and keeps the reading surface untouched (§3). Resolves Open question on editing model.
- 2026-05-04: Notes and highlights are independent surfaces at v1 — both can anchor to the same range, but neither references the other in schema, API, or UI. Reason: simplest; no foreign-key tangle, no cross-feature coupling, ships independently. A "note attached to a highlight" relation can be added later without breaking either feature. Resolves Open question on highlights interaction.
- 2026-05-04: No rate limit on the four `/api/notes` endpoints at v1, matching the existing `internal/highlights/` precedent. Reason: writes are behind `auth.RequireUser`, no concrete abuse model, and the dual-bucket limiter from `internal/auth/` exists for credential-stuffing surfaces specifically. Revisit if abuse appears, same trigger as the rest of the app. Resolves Open question on rate limiting.
- 2026-05-04: No cap on number of notes per user per chapter at v1. Reason: matches the v1 simplicity posture and the no-rate-limit decision; pathological creation would surface as a separate problem worth real signal. Resolves Open question on per-chapter cap.
- 2026-05-04: Drawer empty / loading / error UX matches the existing app patterns: silent empty (no nag when the chapter has zero notes), spinner on load, generic toast on failure (same copy posture as the reader: 429 → "Service is busy, try again in a moment"; 5xx/network → "Something went wrong, try again"). Reason: no new pattern to maintain, consistency across surfaces (§3 Study-first UX). Resolves Open question on drawer states.
- 2026-05-04: `GET /api/notes` accepts `?book&chapter` (same shape as `GET /api/highlights`) and returns notes ordered by `start_verse ASC, start_offset ASC`. Reason: drawer order follows reading order, so tap-to-scroll feels natural; matching the highlights query shape lets the client share validation and reduces surface-area cognitive load. Resolves Open question on GET shape and ordering.
- 2026-05-04: Implementation landed on branch `feat/notes`. Server-side `internal/notes/` package mirrors `internal/highlights/` with a `body TEXT` column, `updated_at`, no overlap rejection, and a PATCH endpoint; 16 KB body cap enforced via `http.MaxBytesReader` + post-decode length check. Client-side `web/src/notes/` (api / useNotes / NotesDrawer) plus a header-chrome "Notes" toggle button (hidden for guests). The selection toolbar (`HighlightToolbar`) gained an "Add note" button alongside "Highlight"; clicking it opens the drawer with a composer pre-anchored to the selected range so no empty notes are persisted. Tap-on-entry scrolls the reading surface to the anchor via `parseSelection.tupleToRange` against an `articleRef` lifted to `App.tsx`. Status flipped to In Progress; will move to Shipped on merge.
- 2026-05-05: Status flipped to Shipped on merge of `feat/notes` to `main`. End-to-end verified locally against the Docker deploy after a separate `deploy.sh` fix to pass `ENV=dev` (cookies were being issued with `Secure` over plain HTTP, which intermittently broke session lookup); the deploy fix rode along on this PR.
- 2026-05-05: Notes now work against any registered translation. The `parseSelection.tupleToRange` call used by tap-to-scroll (lifted to `App.tsx` against the shared `articleRef`) takes a `translation: TranslationID` argument after the per-translation verse-anchor dispatcher landed with [niv](./niv.md); `App.tsx` passes the active translation through. No notes-package changes were required — the dispatcher lives in `web/src/highlights/parseSelection.ts` and notes simply consume it.

## Verification

How we'll know it's working: tests, manual flows, metrics, screenshots.

- [ ] Round-trip test: signed-in user creates a note on a chapter, reloads, and sees it attached to the same passage.
- [ ] Cross-user isolation: user B's `GET /api/notes?book&chapter` does not see user A's notes; user B's `DELETE /api/notes/{A's id}` returns 404.
- [ ] Guest posture: unauthenticated `GET/POST/PATCH/DELETE /api/notes*` return 401; the SPA hides both the "Add note" affordance and the drawer toggle.
- [ ] Edit + delete round-trip: PATCH updates the body and bumps `updated_at`; the drawer entry shows the "edited" indicator afterwards. DELETE removes the note and subsequent GET no longer returns it.
- [ ] Canon validation: `POST /api/notes` with an unknown book or out-of-range chapter returns 400 (reuses `esv.LookupBook`).
- [ ] Body cap enforcement: `POST` and `PATCH` with a body > 16 KB return 400; bodies ≤ 16 KB succeed.
- [ ] Response order: `GET /api/notes?book&chapter` returns notes sorted by `start_verse ASC, start_offset ASC`.
- [ ] Reading-surface UX: with the drawer closed, the reading surface looks identical to a chapter with no notes; with the drawer open, the reading surface remains the focal element (§3 Study-first UX) — verified manually until automated coverage exists.

## Related

- Other specs this depends on or extends: [accounts](./accounts.md), [highlights](./highlights.md), [passage-reader](./passage-reader.md)
- Constitution sections: `PROJECT_CONSTITUTION.md §2`, `§3`, `§4`, `§5`
- External references: ESV API docs (https://api.esv.org/docs/) — only relevant if note anchoring uses verse anchors like highlights does
