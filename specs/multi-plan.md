# Multi-plan

**Status:** Shipped
**Created:** 2026-05-06
**Last updated:** 2026-05-09
**Owner:** unassigned

## Why

The daily-reading feature ships a single embedded plan ("Bible in One Year"). The user wants to support a second plan — Pastor Nate's 2026 *Hope* plan, distributed as `internal/dailyreader/2026_Hope_Bible_Reading_Plan.md` — and let readers pick either plan, or both, from the gear panel. The two plans differ enough structurally (header columns, date format, multi-passage cells, and special "catch-up"/holiday rows) that we need a small plan registry rather than a second copy of the existing parser, and the daily panel needs to stop hardcoding the plan label.

## Goals

- [x] `internal/dailyreader` exposes a plan registry; both plans embedded; `Today([]string, tz, time.Time) ([]Day, error)` returns one `Day` per requested plan.
- [x] Hope-plan parser handles its date format (`Jan 12`), the **two** passage columns (`Reading` + `Psalm`), `&`/`;` multi-passage / multi-book cells, book-only references for short books (`Philippians` → `Philippians 1-4`), verse-range Psalms (collapsed to chapter; see Decisions), and special-day rows (catch-up, holiday, scripture-quote-only).
- [x] `GET /api/daily-reading?plans=hope,bible-year` returns `{ plans: [{ id, name, passages, message }] }`. Missing `plans` defaults to the Bible-in-One-Year plan; unknown ID → 400. **This is a breaking shape change** (was `{ passages, message }`); the only consumer is the SPA, updated in the same PR.
- [x] SPA persists the selection in `localStorage` via the existing `ToggleStore` (`reader.plans` key, JSON-stringified array). No new platform abstraction — `useToggles` already serialises a JSON value through the same store.
- [x] Settings (gear) panel exposes one checkbox per plan; empty selection coerces to the default.
- [x] Daily panel renders a single, plan-tagged pill row when more than one plan is active and shows per-plan info cards for special-day messages.
- [x] Existing single-plan behaviour (Bible-in-One-Year, OT/NT pills) is preserved when only that plan is selected.

## Non-goals

- **Server-stored plan preference.** Translation persists via `users.translation` (see [multi-translation](./multi-translation.md)); plan selection stays client-only by design (works for guests, no migration). Revisit if cross-device persistence is requested.
- **A third (or arbitrary) plan UI.** The catalog supports more, but we only ship two and the picker is checkbox-per-plan; no plan-management UI.
- **Year-aware Hope-plan keying.** The plan is 2026-specific (5-day-week schedule). We key by `MM/DD` like the Bible-in-One-Year plan; in 2027 the readings still resolve, but the day-of-week alignment drifts. Acceptable.
- **Cross-plan deduplication.** When both plans assign the same chapter on the same day, we render two pills. They share the same `PassageView` cache by translation; visual de-dup is not worth the coupling.

## User-facing behavior

**Picker.** A new "Reading plan" group appears in the gear panel beneath "Show". Each plan gets a checkbox; the selection persists across reloads in `localStorage`. If the user unchecks every plan, the panel falls back to `["bible-year"]` so the daily tab is never stuck on a blank state.

**Daily panel — single plan.** Identical to today: OT/NT pill row (or just one pill if the day has only one reading), info "No reading for today" when the plan has no entry for the date.

**Daily panel — both plans.** A single pill row, in plan order. "Plan order" means the order the plans appear in the static `PLANS` catalog (the same order the server registry uses); the SPA's `?plans=` query value is built from that catalog filtered by checkbox state, and the server returns one `Day` per requested plan in request order. Within a plan the passages stay in markdown order (Bible-in-One-Year emits OT then NT; Hope emits Reading then Psalm). Each pill shows `{book} {chapters}` plus a plan-name tag (e.g. "Hope") in place of the OT/NT tag, so the reader can tell which plan each pill belongs to. Special-day rows from either plan render as small info cards above the pill row (e.g. "Happy Thanksgiving!", "Catch-up day!", or the Christmas-day quote), one card per plan that emitted a `message`, in the same plan order.

**Date nav and translation picker.** Unchanged. The translation picker continues to live in the daily header; both plans render through the active translation.

## Implementation outline

**Server packages**

- `internal/dailyreader/embed.go` — `//go:embed` both `daily-reader.md` and `2026_Hope_Bible_Reading_Plan.md`.
- `internal/dailyreader/dailyreader.go` — `Day{PlanID, PlanName, Passages, Message}`, plan registry, `Today([]string, tz, now)`, `ErrInvalidTZ`, `ErrUnknownPlan`. Empty `planIDs` falls back to `["bible-year"]`.
- `internal/dailyreader/parser_bible_year.go` — extracted current `parsePlan` (no semantic change).
- `internal/dailyreader/parser_hope.go` — Hope-plan-specific parser with a small `bookTestament` map; reuses `splitPassage` and `canon.LookupBook`. Both columns (`Reading` and `Psalm`) flow through the same `parseHopeCell` helper, which splits on `&`/`;`, normalises en-dashes, attempts `splitPassage` first, and falls back to a book-only canon lookup (with full chapter range) for short books. **Message-row criterion:** the cell yields a `Message` (not passages) iff at least one segment fails *both* the "Book Chapters" parse (where the chapter is rejected if the book is not in canon) and the bare-canon-book lookup. Holiday markdown (`**...**`), catch-up text, and scripture-only quotes all hit that branch. The `Psalm` column is conservative: it never overrides a `Reading`-cell message — a non-parseable Psalm cell is silently dropped.
- `2026_Hope_Bible_Reading_Plan.md` — typo fix `Mathew 10` → `Matthew 10`.

**Endpoint**

- `GET /api/daily-reading?tz=&date=&plans=hope,bible-year` — returns `{ plans: [{ id, name, passages, message }] }`. `DailyCounter` increments `Hit` exactly once per request when *any* plan has passages, `Empty` exactly once per request when no plan has passages (e.g. all plans yielded a message-only special day, or both date and plans were valid but unmapped), and `Error` once per request on invalid input. Counter shape is unchanged (one increment per request, not per plan), so existing dashboards stay correct.

**SPA module**

- `web/src/daily/plans.ts` — `PLANS` catalog + `DEFAULT_PLAN_IDS`.
- `web/src/daily/usePlanSelection.ts` — `ToggleStore`-backed hook (`reader.plans` key), modeled after `web/src/toggles.ts`.
- `web/src/daily/useDailyTab.ts` — flat plan-tagged pill list (`DailyPill`), `activeIdx`, per-pill `chapterStates`, plan-prefixed article-ref keys, `planMessages`. Re-fetches on `selectedDate | translation | planIDs` change.
- `web/src/daily/DailyPanel.tsx` — drops the hardcoded "Bible in One Year" label; renders `planMessages` as info cards; pill tag becomes plan-name when more than one plan is active, otherwise stays OT/NT.
- `web/src/SettingsPane.tsx` — adds a "Reading plan" checkbox group.
- `web/src/App.tsx` — lifts `planIDs` state via `usePlanSelection`, passes through to both the settings pane and `useDailyTab`.

## Open questions

- **Pill tag when both plans are active and a Hope-plan pill happens to be a Psalm.** Plan tag wins (clarity over OT/NT signal). Reader still sees "Psalm 23" as the pill ref. Revisit if multiple users find it confusing.
- **Hope-plan year drift.** Carrying a 2026-specific schedule into 2027 misaligns weekdays — and the catalog label "Hope (2026)" will read stale. Accepted for v1; the catalog label will need to be revisited (e.g. drop the year, hide behind a flag, or replace with a generated calendar) before 2027.
- **Cross-plan fetch caching.** `useDailyTab` keys per-chapter article refs on `${planID}|${book}:${chapter}` to keep DOM nodes distinct when both plans schedule the same chapter on the same day, but the *passage HTML fetch* path (`fetchPassage`) is keyed on `(book, chapter, translation)` upstream. Result: two pills on the same chapter share the same network result but each renders its own DOM. That's deliberate; if a future cache layer prefixes by plan, it'll just cost an extra request, not break correctness.

## Decisions

- **2026-05-06** — Plan selection persists in `localStorage` via `ToggleStore`, not the user record. Rationale: works for guests, no migration, matches the existing display-toggle posture; cross-device persistence is not a requested goal.
- **2026-05-06** — One pill row with plan-name tags for the both-plans case (rejected: per-plan stacked sections with their own headers). Rationale: keeps the daily-reading layout compact and lets the existing pill bar do double duty without introducing a per-plan section block.
- **2026-05-06** — Special-day rows (catch-up, holiday, quote) render as info cards above the pill row, not as empty days. Rationale: preserves the pastoral content of the Hope plan without trying to fetch scripture for it.
- **2026-05-06** — Plan IDs are slugs (`bible-year`, `hope`) decoupled from display names. Rationale: stable across label changes; URL-safe in the `?plans=` query; safe `localStorage` value.
- **2026-05-06** — Hope-plan testament tagging via a small static `bookTestament` map in the parser package, not by adding a `Testament` field to `canon.Book`. Rationale: keeps `canon` translation-neutral and keeps the change scoped to dailyreader.
- **2026-05-09** — Marked Shipped to match the index and CLAUDE.md status paragraph; all goals above are landed.

## Related

- [`multi-translation.md`](./multi-translation.md) — translation picker is the closest precedent for a per-user reading preference; this spec deliberately diverges on persistence (client-only vs server) for the reasons in Decisions / Non-goals.
- [`daily-annotations.md`](./archive/daily-annotations.md) — established the per-pill chapter-block model on the Daily tab; multi-plan keeps that posture and just generalises pill identity from `OT`/`NT` to plan-tagged.
- [`auto-load-daily-reading.md`](./auto-load-daily-reading.md) — established the daily-reading load flow; this spec only refactors the response shape and the pill state machine, not the auto-load posture.
- [`PROJECT_CONSTITUTION.md`](../PROJECT_CONSTITUTION.md) — §1 (study-first), §3 (respect the text — plans schedule reading, never mutate scripture), §4 (server still proxies all scripture; no upstream call from client; `ToggleStore` remains the platform abstraction).

## Verification

**Go tests**
- `internal/dailyreader/parser_bible_year_test.go` — every existing row still parses (extracted from current tests).
- `internal/dailyreader/parser_hope_test.go` — passage-only row, multi-passage `&` row, Psalm with verse range, catch-up day, holiday header (`**Happy Thanksgiving!**`), Christmas quote, leap day → empty.
- `internal/dailyreader/dailyreader_test.go` — `Today(nil, ...)` defaults to bible-year; `Today([]string{"hope"}, ...)` returns one Day; `Today([]string{"bible-year","hope"}, ...)` returns two Days in order; unknown ID returns `ErrUnknownPlan`.
- `internal/server/daily_test.go` — `?plans=hope`, `?plans=hope,bible-year`, `?plans=bogus` (400); default behaviour preserved when `plans` is absent.

**SPA tests**
- `web/src/daily/usePlanSelection.test.ts` — default fallback, persistence round-trip via in-memory `ToggleStore`.
- `web/src/daily/useDailyTab.test.ts` (or extension to existing tests) — multi-plan response builds a flat pill list; toggling `planIDs` triggers a re-fetch.

**Manual smoke**
1. `cd web && npm install && npm run build && cd .. && ESV_API_KEY=… SESSION_SECRET=… YOUVERSION_APP_KEY=… ENV=dev go run .`
2. Daily tab default → Bible-in-One-Year pills, unchanged.
3. Open gear → check "Hope (2026)" → confirm pill row now shows pills from both plans, each labeled with its plan name; reload → selection persists.
4. Uncheck "Bible in One Year" → only Hope pills remain.
5. Date-pick `2026-02-20` → Hope catch-up info card; Bible-year pills (if selected) still render.
6. Date-pick `2026-12-25` → Hope Christmas-quote info card.
