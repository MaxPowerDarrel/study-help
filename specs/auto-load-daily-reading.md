# Auto-load Daily Reading

**Status:** Draft
**Created:** 2026-05-03
**Last updated:** 2026-05-03 (revised with open questions resolved)
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

- **Backend:** New endpoint `GET /api/daily-reading?tz=<timezone>` (e.g., `?tz=America/Los_Angeles`). Query param specifies client timezone for date lookup.
- **Backend:** Parse `daily-reader.md` at request time (no caching). Markdown table with columns: `Date`, `OT Reading`, `NT Reading`. Match MM/DD/YY date to client's local date (adjusted by timezone).
- **Backend:** Response format: `{passages: [{book: 'Num.', chapters: '15-16', testament: 'OT'}, {book: 'Rev.', chapters: '20', testament: 'NT'}]}`. If no match or both OT/NT empty, return `{passages: [], message: "No reading for today"}`.
- **Backend:** Handle invalid timezone with 400 error; return graceful empty response if reading row missing.
- **Frontend:** Toggle in the settings pane wired to `ToggleStore` (localStorage, client-only)
- **Frontend:** On app startup, detect client timezone via `Intl.DateTimeFormat` or `navigator.language`; call `GET /api/daily-reading?tz=<detected-tz>` in background (non-blocking).
- **Frontend:** Show picker pane immediately while passages load.
- **Frontend:** On success, display both OT and NT passages in tabs or side-by-side layout.
- **Frontend:** On failure or no passages, show error: "Daily reading unavailable; pick a passage manually." Allow user to proceed manually.
- **Data:** `daily-reader.md` (confirmed to exist in project root; Bible in One Year 2026 plan in markdown table format)

## Open questions

(none — all resolved via decision)

## Decisions

- 2026-05-03: Daily reading comes from the hardcoded `daily-reader.md` (Bible in One Year 2026 plan). Reason: Plan already exists in the project root and provides a structured source.
- 2026-05-03: Toggle preference is client-side only (localStorage), not synced across devices. Reason: Simplicity; user can set per-device preference.
- 2026-05-03: Daily reading loads in background on startup (non-blocking). Reason: Prevents UX delay; picker pane shows immediately, passages load while user can interact.
- 2026-05-03: `GET /api/daily-reading` returns book abbreviations as-is from daily-reader.md (e.g., 'Num.', 'Rev.'). Reason: Maintains data fidelity from source; no normalization overhead.
- 2026-05-03: Both OT and NT passages display in tabs or side-by-side layout. Reason: Allows user to toggle between them without losing context.
- 2026-05-03: Auto-load is opt-in (disabled by default). Reason: Respects user choice; non-disruptive to first-time users.
- 2026-05-03: Parse daily-reader.md at request time (no server-side caching). Reason: Simple; file is small; ensures fresh data on each request; no invalidation logic needed.
- 2026-05-03: Timezone handling uses client timezone (detected via browser `Intl.DateTimeFormat`). Backend receives client timezone and adjusts date lookup accordingly. Reason: Users expect "today" to match their local date, not server UTC.
- 2026-05-03: Toggling auto-load only affects the next app startup. Reason: Simple behavior; no need to reload app state mid-session.
- 2026-05-03: API failures show error message to user (e.g., "Daily reading unavailable; pick a passage manually"). Reason: User feedback; avoids silent failures; respects study-first UX (reading surface is the priority).
- 2026-05-03: Date matching: Client detects timezone, backend compares MM/DD/YY from daily-reader.md against client's local date. Edge case: if row is missing or both OT/NT are empty, return empty array and show message. Reason: Respects user's local timezone; handles sparse reading plan gracefully.

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
