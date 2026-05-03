# Auto-load Daily Reading

**Status:** Draft
**Created:** 2026-05-03
**Last updated:** 2026-05-03
**Owner:** unassigned

---

## Why

Let users jump directly to today's daily reading without manual navigation, reducing friction for users who follow the "Bible in One Year" plan.

**Constitutional support:**
- PROJECT_CONSTITUTION.md §2: "In scope: Reading a chapter or a contiguous passage range" — this feature enables users to read passages from the daily plan.
- PROJECT_CONSTITUTION.md §3: "Study-first UX. The reading surface is the product... must justify the space it takes." — Auto-load eliminates friction, making the reading surface immediately accessible and prioritized.

## Goals

- [ ] Users can enable/disable auto-load in settings (one-click toggle)
- [ ] When enabled, today's daily passages (OT and NT) load automatically on app startup without blocking UI rendering
- [ ] User preference persists across sessions (same device, localStorage)
- [ ] Endpoint latency does not noticeably delay the app's interactivity (non-blocking background load)

## Non-goals

- Push notifications or reminders — *deferred; platform feature requiring abstraction*
- Server-side daily reading schedule — *out of scope at v1; daily reading is client-determined*
- Multi-user or shared reading plans — *user data is personal*

## User-facing behavior

A toggle in the settings pane labeled "Auto-load daily reading". When enabled:
- On app startup, today's OT and NT readings load automatically (from `daily-reader.md`)
- E.g., on 2026-05-03, the app loads "Num. 15-16" and "Rev. 20" without user action
- The user can still manually select a different passage at any time
- When disabled, the app opens to the default state (picker pane)

## Implementation outline

- **Backend:** New API endpoint `GET /api/daily-reading` that parses `daily-reader.md` and returns today's passages with book abbreviations as-is. Response: `[{book: 'Num.', chapters: '15-16', testament: 'OT'}, {book: 'Rev.', chapters: '20', testament: 'NT'}]`. Computed at request time or cached.
- **Backend:** Parse `daily-reader.md` (markdown table with columns: `Date`, `OT Reading`, `NT Reading`). Match current date (MM/DD/YY format) to return today's row. Handle edge cases: dates with missing OT or NT reading (several dates have empty cells); return only non-null passages.
- **Backend:** API endpoint `GET /api/daily-reading` response format: `[{book: 'Num.', chapters: '15-16', testament: 'OT'}, {book: 'Rev.', chapters: '20', testament: 'NT'}]`. Parse passage strings from daily-reader.md as-is.
- **Backend:** Parsing strategy (caching vs. request-time) — open question below.
- **Frontend:** Toggle in the settings pane wired to `ToggleStore` (localStorage, client-only)
- **Frontend:** On app startup (App.tsx or similar), check toggle and call `GET /api/daily-reading` in background (non-blocking). Show picker pane immediately while passages load.
- **Frontend:** On successful load, display both OT and NT passages in tabs or side-by-side layout. User can switch between them.
- **Frontend:** Graceful degradation if endpoint fails; user can still pick passages manually.
- **Data:** `daily-reader.md` (confirmed to exist in project root; Bible in One Year 2026 plan in markdown table format)

## Open questions

- [ ] Should we cache the daily-reader.md parse result server-side, or parse at request time? (Tradeoff: caching faster, but invalidation on file change)
- [ ] How should we handle timezone edge cases when determining "today"? (e.g., user in PST, server in UTC)
- [ ] Does toggling the setting on immediately trigger a load of today's reading, or does it only affect future app startups?
- [ ] Should the frontend retry or gracefully degrade if `GET /api/daily-reading` fails at startup?
- [ ] How will dates be matched to today? (Parse MM/DD/YY from daily-reader.md and compare to current date)

## Decisions

- 2026-05-03: Daily reading comes from the hardcoded `daily-reader.md` (Bible in One Year 2026 plan). Reason: Plan already exists in the project root and provides a structured source.
- 2026-05-03: Toggle preference is client-side only (localStorage), not synced across devices. Reason: Simplicity; user can set per-device preference.
- 2026-05-03: Daily reading loads in background on startup (non-blocking). Reason: Prevents UX delay; picker pane shows immediately, passages load while user can interact.
- 2026-05-03: `GET /api/daily-reading` returns book abbreviations as-is from daily-reader.md (e.g., 'Num.', 'Rev.'). Reason: Maintains data fidelity from source; no normalization overhead.
- 2026-05-03: Both OT and NT passages display in tabs or side-by-side layout. Reason: Allows user to toggle between them without losing context.
- 2026-05-03: Auto-load is opt-in (disabled by default). Reason: Respects user choice; non-disruptive to first-time users.

## Verification

- [ ] Manual test: toggle on → close and reopen app → daily passages load in background, picker visible, passages appear when ready
- [ ] Manual test: toggle off → app opens to picker pane (no auto-load)
- [ ] Manual test: preference persists across browser refresh
- [ ] Manual test: API endpoint `GET /api/daily-reading` returns correct passages for today (check daily-reader.md)
- [ ] Manual test: endpoint failure gracefully degrades (app still usable, user can pick passages manually)
- [ ] Performance: background load doesn't block UI interaction

## Related

- `[passage-reader](./passage-reader.md)` — base reading feature this extends
- `PROJECT_CONSTITUTION.md §2` (in-scope) — §3 (Study-first UX)
