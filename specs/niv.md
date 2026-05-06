# NIV translation (YouVersion Platform)

**Status:** In Progress
**Created:** 2026-05-05
**Last updated:** 2026-05-05
**Owner:** unassigned

## Why

The multi-translation foundation shipped with ESV-only ([`specs/multi-translation.md`](./multi-translation.md)) and an explicit follow-up to register a second provider. NIV is the most-requested next translation. Biblica licenses NIV through the YouVersion Platform API; YouVersion's public beta (Dec 2025) opened self-serve access on `platform.youversion.com`, so a hobby-scale OSS project can ship NIV without a partner contract.

Constitutionally: the YouVersion app key never reaches the browser (§4); all NIV requests are server-proxied through `internal/youversion/`. Highlights and notes remain server-authoritative and per-translation-scoped (multi-translation §Decisions, 2026-05-05).

## Goals

- [ ] `internal/youversion/` package implementing `scripture.Provider` against `https://api.youversion.com/v1/`.
- [ ] `YOUVERSION_APP_KEY` required env var; server fails fast at startup when unset.
- [ ] NIV registered in the runtime registry (`main.go`) alongside ESV.
- [ ] `internal/server/passage_test.go` covers two-provider routing (`?translation=ESV` vs `?translation=NIV`).
- [ ] Client `web/src/translations/catalog.ts` lists NIV; existing `<select>` picker surfaces it without UI changes.
- [ ] Per-translation verse-anchor dispatcher in `web/src/highlights/parseSelection.ts` so highlights AND notes work for NIV out of the gate.
- [ ] "Powered by YouVersion" attribution rendered when NIV is the active translation (Read tab + Daily tab).
- [ ] Vitest coverage for the NIV anchor adapter (range/tuple round-trip, applyHighlights idempotency).

## Non-goals

- **Renaming the metrics counter.** `esv_api_calls_total` increments for both providers today. Per-translation segmentation is tracked in `specs/multi-translation.md` open questions and is a separate change.
- **Per-translation render toggles.** ESV's `include_word_of_christ` (CSS-only on the SPA) does not currently apply to NIV; the toggle continues to live on Settings without gating.
- **Caching or offline storage.** Each request still proxies live to YouVersion; no edge cache. Defer until rate-limit pressure shows up.
- **Cross-translation re-anchoring.** Same posture as multi-translation: a highlight or note authored in ESV is invisible while NIV is active and vice versa (already enforced by `TestCrossTranslationIsolation`).
- **Bible-ID auto-discovery.** `bible_id=111` is a constant in `internal/youversion/client.go`; a runtime probe of `GET /v1/bibles` is part of the pre-merge checklist, not a runtime feature.

## User-facing behavior

**Picker.** The existing translation `<select>` (Read tab) automatically shows "NIV" once the catalog entry is added. Signed-in users can switch; guests stay pinned to the registry default (ESV) with the existing "Sign in to choose" hint.

**Switching to NIV.** `PATCH /api/auth/me` persists the choice; the open passage refetches against NIV. Existing ESV highlights and notes are no longer visible (cross-translation isolation), and NIV highlights/notes for the same range can be created independently.

**Attribution.** "Powered by [YouVersion](https://www.youversion.com/)" appears under the picker on the Read tab and beneath the daily article on the Daily tab whenever NIV is the active translation. Final wording / link locked during the pre-merge ToS review.

**Highlights and notes.** Selection inside the rendered NIV passage produces a verse + offset tuple via the per-translation anchor dispatcher (`<span class="yv-v" v="N">` boundaries instead of ESV's `<a class="va" rel="...">`). Round-trip through `applyHighlights` and `tupleToRange` (used by notes' tap-to-scroll) works the same as ESV. Offset semantics differ slightly: because YouVersion's boundary span sits before the verse-label span (`<span class="yv-vlbl">N</span>`), offset 0 falls on the visible "N" rather than the first character of the verse text body. Round-trips are consistent because both directions use the same anchor.

## Implementation outline

**Server packages**

- `internal/youversion/`
  - `client.go` — HTTP client. Translates canon-validated `q` (e.g. `"John 3:1-21"`) to USFM (`"JHN.3.1-JHN.3.21"`), calls `GET /v1/bibles/111/passages/<usfm>` with header `X-YVP-App-Key`, decodes the JSON `{content, human_reference}`, and rewraps it in the ESV-shaped envelope `{canonical, passages:[content]}` so the SPA's `web/src/api.ts` can consume both providers without branching. Uses `json.Encoder` with `SetEscapeHTML(false)` to match ESV's wire format. Maps 429 → `ErrRateLimited`, other 4xx/5xx → `ErrUpstream`.
  - `usfm.go` — Static 66-entry name → 3-character USFM code map (USFM 3.0 spec).
  - `provider.go` — Adapter implementing `scripture.Provider`. Mirrors `internal/esv/provider.go`.
  - `client_testing.go` — `HTTPClientForTest` mirrors the ESV helper for httptest stubs.
- `internal/scripture/provider.go` — adds `const NIV ID = "NIV"` next to `ESV`.
- `internal/config/config.go` — adds `YouVersionAppKey` (env `YOUVERSION_APP_KEY`); fails fast like `ESV_API_KEY`.
- `main.go` — registers the YouVersion provider in `scripture.NewRegistry`.
- `internal/server/passage_test.go` — adds `newMultiStubRegistry` that points ESV and NIV providers at separate `httptest.Server`s, plus `TestPassageHandlerRoutesByTranslation` asserting `?translation=ESV` hits the ESV stub and `?translation=NIV` hits the NIV stub.

**Client**

- `web/src/translations/catalog.ts` — `TranslationID` union grows `"NIV"`; new `TRANSLATIONS` entry.
- `web/src/highlights/parseSelection.ts` — refactored into a dispatcher: `listVerseAnchors(container, translation)` looks up a per-`TranslationID` anchor lister; `rangeToTuple` and `tupleToRange` take `translation` as a required arg. The shared text-walking logic stays in this file (anchor-agnostic). Two listers live alongside: `listEsvAnchors` (existing logic, ESV `<a class="va" rel="...">`) and `listYouVersionAnchors` (`<span class="yv-v" v="N">`).
- `web/src/highlights/applyHighlights.ts` — accepts `translation`, threads it into `listVerseAnchors` and `tupleToRange`.
- `web/src/highlights/PassageView.tsx`, `web/src/App.tsx` — call sites updated to pass the active `translation` into `applyHighlights`, `rangeToTuple`, and `tupleToRange`.
- `web/src/translations/Attribution.tsx` + `Attribution.module.css` — renders "Powered by YouVersion" when `translation === "NIV"`, else returns null. Slot below the picker (Read tab) and beneath the daily article (Daily tab).
- `web/src/highlights/applyHighlights.niv.test.ts` — fixture HTML matching the YouVersion structure; covers `listVerseAnchors` dispatch, `rangeToTuple` / `tupleToRange` round-trip, and `applyHighlights` idempotency. Existing `applyHighlights.test.ts` updated to thread `"ESV"` into call sites.

## Pre-merge checklist

- [ ] Sign up at `platform.youversion.com`, generate an app key, accept the NIV (Bible ID 111) license in the YouVersion console.
- [ ] Manually read `platform.youversion.com/terms` end-to-end (page is JS-rendered; web fetchers don't see content). Confirm the exact attribution wording and link target, plus any caching / branding rules. Adjust `Attribution.tsx` copy/link before merge.
- [ ] Confirm `bible_id = 111` for NIV by hitting `GET /v1/bibles` once with the real app key. Update `nivBibleID` in `internal/youversion/client.go` if YouVersion has remapped it.
- [ ] Capture a real `GET /v1/bibles/111/passages/JHN.3` response and confirm the boundary span structure matches the SDK PR (`<span class="yv-v" v="N">` plus `<span class="yv-vlbl">N</span>`). If the structure differs, update `listYouVersionAnchors` and the test fixture in `applyHighlights.niv.test.ts` before merge.

## Decisions

- **2026-05-05** — Provider lives in its own peer package `internal/youversion/`, mirroring `internal/esv/`. Rejected: nesting under `internal/scripture/youversion/` (the multi-translation spec already settled this).
- **2026-05-05** — `YOUVERSION_APP_KEY` is a required env var (fail-fast at startup). Rejected: optional / conditional registration when missing — the additional branching in `main.go` adds a code path that would only matter for ESV-only forks; if that demand surfaces, gating is a one-line change.
- **2026-05-05** — Per-translation verse-anchor dispatcher lands in the same PR as the NIV provider. Rejected: ship NIV as read-only first and refactor the dispatcher later — that would silently break highlights and notes for any user who switches to NIV, which the multi-translation spec already flagged as a risk.
- **2026-05-05** — Re-wrap YouVersion's response (`{content, human_reference}`) into the ESV envelope shape (`{canonical, passages:[content]}`) on the server so `web/src/api.ts` stays translation-agnostic. Rejected: a per-provider response decoder on the SPA — pushes provider concerns past the server boundary and grows the client surface for every new translation.
- **2026-05-05** — Match ESV's wire format using `json.Encoder` with `SetEscapeHTML(false)`. Rejected: Go's default `<,>` escaping — works functionally, but the bytes diverge from ESV's wire format and made the multi-provider routing test brittle on first run.
- **2026-05-05** — Use the `<span class="yv-v" v="N">` boundary span as the anchor (offsets count the verse label as part of the verse body). Rejected: walking forward to use `<span class="yv-vlbl">N</span>` as the anchor — would require an additional DOM walk per verse and the offset semantics aren't user-visible enough to justify the complexity. Round-trip integrity is what matters and is preserved either way.
- **2026-05-05** — Attribution slotted under the picker (Read) and below the daily article (Daily); component returns null for non-NIV translations. Rejected: a global header slot — clutters the chrome on the ESV path; a footer-only slot — less discoverable on tall articles.

## Verification

**Go tests**

```
go test ./...
```

- `internal/youversion/client_test.go` — USFM translation (`John 3` → `JHN.3`, `John 3:1-21` → `JHN.3.1-JHN.3.21`, `Psalms 119-120` → `PSA.119-PSA.120`, `1 Corinthians 13:4-7` → `1CO.13.4-1CO.13.7`, etc.); httptest happy path verifies path / `X-YVP-App-Key` header / envelope rewrite; 429 → `ErrRateLimited`; 5xx → `ErrUpstream`; malformed body → `ErrUpstream`.
- `internal/youversion/provider_test.go` — `ID()` / `DisplayName()`; error remapping `ErrRateLimited` → `scripture.ErrRateLimited`, `ErrUpstream` → `scripture.ErrUpstream`.
- `internal/server/passage_test.go::TestPassageHandlerRoutesByTranslation` — two-provider routing; both stubs invoked exactly once.

**SPA tests**

```
cd web && npm test
```

- `web/src/highlights/applyHighlights.niv.test.ts` — listVerseAnchors dispatches per-translation; rangeToTuple / tupleToRange round-trip on YouVersion markup; applyHighlights wraps the right text and is idempotent.
- `web/src/highlights/applyHighlights.test.ts` — existing ESV behavior, now with the `translation` arg threaded through call sites.

**End-to-end smoke (with real keys)**

```
cd web && npm install && npm run build && cd ..
ESV_API_KEY=… YOUVERSION_APP_KEY=… SESSION_SECRET=… ENV=dev go run .
```

1. `GET /healthz` → 200.
2. Sign in. `GET /api/auth/me` → `"translation":"ESV"`. Picker shows ESV + NIV.
3. Switch to NIV → `PATCH /api/auth/me` 200; passage refetches; "Powered by YouVersion" attribution appears under the picker.
4. Select text → "Highlight" → reload → highlight still rendered.
5. Select text → "Add note" → save → reload → tap entry → article scrolls to the right verse.
6. Switch back to ESV → existing ESV highlights/notes visible; NIV highlights hidden (cross-translation isolation).
7. Daily tab with NIV active → daily passages render in NIV; attribution appears below the article.
8. Negative: unset `YOUVERSION_APP_KEY`, restart → server fails fast at startup with `YOUVERSION_APP_KEY is required`.
