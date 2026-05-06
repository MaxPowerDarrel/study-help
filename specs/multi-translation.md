# Multi-translation

**Status:** Shipped (foundation; only ESV registered at launch)
**Created:** 2026-05-05
**Last updated:** 2026-05-05
**Owner:** unassigned

## Why

`PROJECT_CONSTITUTION.md` previously locked v1 to ESV-only because we wanted to ship reading and annotation against a single, known-good API before introducing the variability that comes with multiple translations. With reader, accounts, highlights, and notes all shipped, the abstraction can land safely. This PR amends §2 ("Multi-translation, ESV at launch") and §5 ("No cross-translation rendering") and refactors the server, schema, and SPA so additional translations can be added in subsequent PRs without touching highlights, notes, or auth.

The constitution still says "secrets stay server-side" — every translation provider is server-side proxied; no upstream API keys reach the browser. Highlights and notes remain server-authoritative (§4).

## Goals

- [x] Server-side `TranslationProvider` abstraction with a `Registry` resolved at startup.
- [x] `internal/canon/` package shared across providers (66-book Protestant canon).
- [x] ESV refactored to live behind `scripture.Provider` without behavioral change.
- [x] Per-user `users.translation` column, surfaced through `GET /api/auth/me` and a new `PATCH /api/auth/me`.
- [x] Highlights and notes scoped per translation (`translation` column on both tables).
- [x] `/api/passage` accepts `?translation=`, resolved against the registry, falling back to user pref.
- [x] SPA `web/src/translations/` module with hook + catalog mirroring the server registry.
- [x] Picker UI surfaces the selector (disabled for guests with sign-in nudge).

## Non-goals

- **Adding NIV / KJV / WEB / etc. in this PR.** This PR ships the abstraction and persists ESV; subsequent PRs register additional providers.
- **Cross-translation rendering.** Highlights and notes are stamped with their authoring translation and only render when active. Versification + verse-text differences make re-anchoring an unbounded problem (constitution §5).
- **Multiple translations active simultaneously** (e.g., parallel reading). Out of scope.
- **localStorage fallback for translation choice.** Guests are pinned to the registry default; signed-in users get persistence via the account.
- **Per-translation render toggles.** ESV's `include_word_of_christ` (CSS-only client toggle) and any future provider-specific knobs are deferred until a second provider lands.

## User-facing behavior

**Picker.** A `Translation` `<select>` sits above the Book select in the picker pane. For signed-in users it's enabled, pre-selected from `user.translation`, and changing it triggers a `PATCH /api/auth/me` and a refetch of the current passage and highlights/notes. For guests the select is disabled with the inline hint "Sign in to choose" — same posture as the existing "Sign in to highlight" guard.

**Persistence.** On sign-in, the SPA hydrates the active translation from `/api/auth/me`. On reload, hydration is the same — there is no localStorage path. Guests always see the registry default (ESV).

**Reading.** Every passage fetch carries `?translation=`. At launch, this only takes the `ESV` value, but the wire shape is correct so adding a provider in a follow-up PR is purely additive.

**Highlights/notes.** Every list/create call carries the active translation. Cross-translation isolation: a highlight created against one translation does not appear when a different translation is active (proven by `TestCrossTranslationIsolation`).

## Implementation outline

**Server packages**
- `internal/scripture/` — `Provider` interface, `Options`, `Result`, `Registry`, `CatalogEntry`, error sentinels (`ErrRateLimited`, `ErrUpstream`, `ErrUnknown`).
- `internal/canon/` — moved from `internal/esv/canon.go`. Translation-neutral 66-book canon + `LookupBook` + `ValidateQuery`.
- `internal/esv/` — `Client` unchanged; new `Provider` adapter implements `scripture.Provider`. Future providers (e.g. `internal/kjv/`) follow the same pattern.

**Endpoints**
- `GET /api/passage?q=...&translation=...` — translation resolution: explicit param → user's account preference → registry default. Unknown explicit IDs return 400.
- `PATCH /api/auth/me` — partial update of the authenticated user. Body `{"translation":"<id>"}` (other fields no-op via `*string`). Validates against `Registry.Known`.
- `GET /api/highlights?book=&chapter=&translation=` — translation optional; falls back to user pref. Unknown → 400. Filters by translation.
- `POST /api/highlights` — body grows `translation` (optional; falls back to user pref). Stamped on the row. Overlap check is naturally per-translation because the list is.
- Notes endpoints are symmetric.

**Schema** — `internal/db/migrations/00005_translation.sql`. Adds `translation TEXT NOT NULL DEFAULT 'ESV'` to `users`, `highlights`, `notes`. Replaces `(user_id, book, chapter)` indexes with `(user_id, translation, book, chapter)`. `Down` uses `DROP COLUMN` (SQLite ≥3.35; modernc/sqlite ships current).

**SPA module `web/src/translations/`**
- `catalog.ts` — `TRANSLATIONS`, `DEFAULT_TRANSLATION`, `isKnownTranslation`. Mirrors the server registry; future PRs add entries here in the same diff that registers the provider server-side.
- `useTranslation.ts` — single source of truth. Hydrates from `user.translation`; `setTranslation(id)` calls `PATCH /api/auth/me` and updates state on 200.
- `api.ts` — discriminated-union `updateTranslation(id)` matching the auth/highlights/notes shape.

**SPA wiring** — `fetchPassage`, `listHighlights`, `createHighlight`, `listNotes`, `createNote` take a required `translation` argument (TypeScript catches missed call sites). `useHighlights`/`useNotes` add it to their effect deps. The picker `<select>` lives in `App.tsx` above the Book select.

## Open questions

- **`parseSelection.ts` ESV-coupling.** The verse-anchor parser knows ESV's `<a class="va" rel="vBCCCVVV">` shape. Fine at launch (ESV-only). When a future provider's HTML differs, register a per-`TranslationID` parser dispatcher rather than overloading the existing one.
- **`include_word_of_christ`.** Currently CSS-only suppression of ESV's `<span class="woc">`. When a non-ESV provider lands, either gate the toggle on `translation === "ESV"` or generalize to a per-translation render-toggle catalog.
- **Metrics.** `ESVCallCounter` / `esv_api_calls_total` is named for ESV. Rename and segment per-translation when a second provider is registered.
- **Per-provider knobs.** `scripture.Options.Extra map[string]string` exists as an escape hatch. The handler must never populate it from arbitrary client query params — upstream behavior fingerprinting risk. Documented in `provider.go`.

## Decisions

- **2026-05-05** — Pluggable abstraction first, ESV-only at launch. Adding a second translation is a follow-up PR. Rationale: the abstraction is the load-bearing change; subsequent providers are additive (one provider package, one catalog entry, one registration line).
- **2026-05-05** — Per-translation scoping for highlights and notes via a `translation` column. Rejected: verse-level fallback when switching translations (loses precision and complicates rendering); translation-agnostic offsets (offsets drift when verse text differs across translations).
- **2026-05-05** — Translation persists as a `users` column with `PATCH /api/auth/me`. No localStorage fallback. Rejected: localStorage-only (per-device, not per-account, and inconsistent with the server-authoritative principle); both/hybrid (more hydration code for negligible gain).
- **2026-05-05** — `internal/canon/` is its own package, not nested under `scripture`. Reused by `highlights`, `notes`, `dailyreader`, and `scripture`; nesting it under `scripture` would force these packages to import the provider machinery for the canon helper.
- **2026-05-05** — `internal/esv/` stays where it is; a new `provider.go` wraps the existing `Client`. Each future provider lives in its own peer package (`internal/kjv/`, `internal/niv/`). Rejected: moving everything under `internal/scripture/<provider>/` — bigger diff, no real benefit at this size.
- **2026-05-05** — Single migration (`00005_translation.sql`) covering all three tables. Atomic, one-file review surface. Rejected: three migrations — forces reviewers to mentally diff across files for one logical change.
- **2026-05-05** — `PATCH /api/auth/me` over `PUT /api/auth/preferences`. Keeps the auth namespace cohesive; the user record is the only mutable account-scoped resource at v1.
- **2026-05-05** — Picker shows a *disabled* `<select>` for guests with an inline "Sign in to choose" hint. Rejected: hide the picker — makes the capability undiscoverable.
- **2026-05-05** — NIV is the first follow-up provider, registered through the YouVersion Platform API. Per-translation verse-anchor dispatcher (the open question above) lands in the same change. See [`specs/niv.md`](./niv.md).

## Verification

**Go tests**
- `internal/scripture/registry_test.go` — Get/Known/Default/Catalog/duplicate-panic/unregistered-fallback-panic.
- `internal/auth/handlers_test.go` — `/me` includes translation; `PATCH /me` happy path, unknown translation → 400, absent field no-op, unknown field rejected (`DisallowUnknownFields`), guest → 401.
- `internal/highlights/handlers_test.go` — `POST` rejects unknown translation; `GET ?translation=BOGUS` → 400; cross-translation isolation (a row stamped ESV is invisible when KJV is active, and vice versa).
- `internal/server/passage_test.go` — `?translation=BOGUS` returns 400 without invoking the upstream stub.

**Manual smoke**
1. `cd web && npm install && npm run build && cd .. && ESV_API_KEY=… SESSION_SECRET=… ENV=dev go run .`
2. Sign up → `GET /api/auth/me` returns `"translation":"ESV"`.
3. Picker shows the `Translation` select; for guests it's disabled with the "Sign in to choose" hint.
4. `PATCH /api/auth/me` with `{"translation":"NIV"}` → 400 unknown.
5. `PATCH /api/auth/me` with `{"translation":"ESV"}` → 200 + updated user.
6. Reload → picker still shows ESV; existing highlights/notes still render.
