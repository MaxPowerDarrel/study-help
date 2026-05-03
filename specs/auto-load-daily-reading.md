# Auto-load Daily Reading

**Status:** Draft
**Created:** 2026-05-03
**Last updated:** 2026-05-03 (open questions resolved)
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
- [ ] Picker pane is interactive within 250ms of app startup; daily passages render within 2s of startup on a typical connection

## Non-goals

- Push notifications or reminders — *deferred; platform feature requiring abstraction*
- Server-side daily reading schedule — *out of scope at v1; daily reading is client-determined*
- Multi-user or shared reading plans — *user data is personal*

## User-facing behavior

A new settings pane (opened via a gear icon in the header) hosts a toggle labeled "Auto-load daily reading". When enabled:
- On app startup, today's OT and NT readings load automatically (from `daily-reader.md`) and overwrite any prior manual selection
- E.g., on 2026-05-03, the app loads "Num. 15-16" and "Rev. 20" without user action
- The user can still manually select a different passage at any time during the session
- When disabled, the app opens to the default state (picker pane)
- The toggle takes effect on the next app launch; no in-line messaging tells the user this (silent next-launch behavior)

## Implementation outline

- **Backend:** New endpoint `GET /api/daily-reading?tz=<timezone>` (e.g., `?tz=America/Los_Angeles`). Query param specifies client timezone for date lookup. Returns 200 in all non-error cases (200 with empty `passages` for "no reading"); invalid `tz` returns 400.
- **Backend:** New `internal/dailyreader/` package owns parsing and lookup. `daily-reader.md` is embedded into the binary via `embed.FS` in this package (mirrors the `internal/web/dist` embed).
- **Backend:** Parse the embedded markdown at request time (no caching layer). Table columns: `Date`, `OT Reading`, `NT Reading`. Match MM/DD against the client's local date.
- **Backend:** Year wrap-around — if the query date's year is outside the plan's year, fall back to the same MM/DD of the plan's year (e.g., 2027-01-01 → 2026-01-01 reading). Lets the feature work indefinitely without yearly content updates.
- **Backend:** Response format: `{passages: [{book: 'Num.', chapters: '15-16', testament: 'OT'}, {book: 'Rev.', chapters: '20', testament: 'NT'}]}`. If the matched row is missing or both OT/NT cells are empty, return `{passages: [], message: "No reading for today"}` with status 200.
- **Backend:** Increment a Prometheus counter on the private metrics server, labeled by outcome: `hit` (passages returned), `empty` (no reading), `error` (4xx/5xx). Mirrors existing `/api/passage` instrumentation.
- **Frontend:** Toggle in the new settings pane (gear icon in the header) wired to `ToggleStore` (localStorage, client-only).
- **Frontend:** On app startup, obtain client timezone via a new `TimezoneProvider` interface in `web/src/platform/` (mirrors `ToggleStore`; web impl uses `Intl.DateTimeFormat().resolvedOptions().timeZone`, native shell substitutes its own); call `GET /api/daily-reading?tz=<detected-tz>` in background (non-blocking).
- **Frontend:** Show picker pane immediately while passages load.
- **Frontend:** On success, assemble `q` from each passage's `book` + `chapters` (e.g., `'Num. 15-16'`) and call the existing `/api/passage` proxy, which validates via the existing allow-list in `internal/esv/`. Render OT and NT passages on a single reading surface with an OT/NT tab switcher (same layout on all viewports). When auto-load is enabled, today's reading overwrites any prior manual selection.
- **Frontend:** On failure or no passages, show error: "Daily reading unavailable; pick a passage manually." Allow user to proceed manually.
- **Data:** `daily-reader.md` (Bible in One Year 2026 plan, markdown table format) is embedded into the `internal/dailyreader/` package at build time.

## Open questions

All resolved on 2026-05-03 — see Decisions below for each resolution.

- [x] Where does the settings pane live today, and what does the toggle look like? The reader UI as shipped (per CLAUDE.md) has a picker pane and a reading surface — is there an existing settings surface, or does this feature also introduce one? → see Decision 2026-05-03 (settings pane / gear icon).
- [x] When auto-load is enabled and the user has already manually picked a passage in a prior session, does the next startup overwrite that selection with today's reading, or only populate when no prior selection exists? → see Decision 2026-05-03 (always overwrite on startup).
- [x] `daily-reader.md` lives in the project root, not under `internal/`. How is it shipped to the running server — embedded via `embed.FS` (and in which package), or read from disk at runtime? Parse-at-request-time is decided, but the source-of-bytes is not. → see Decision 2026-05-03 (`internal/dailyreader/` embed).
- [x] The 2026 plan in `daily-reader.md` covers one calendar year. What is the behavior on 2027-01-01? Empty response with the "No reading for today" message, or a hard error surfaced to ops? → see Decision 2026-05-03 (year wrap-around).
- [x] Response shape uses `book: 'Num.'` and `chapters: '15-16'` as raw strings. The existing `q` allow-list validator in `internal/esv/` expects a particular canonical form — does the frontend re-assemble these into a valid `q` before calling `/api/passage`, and is there a test that every row in `daily-reader.md` round-trips through the validator? → see Decision 2026-05-03 (frontend assembles `q` + round-trip test).
- [x] Should the new endpoint increment a Prometheus counter on the private metrics server, consistent with the existing `/api/passage` instrumentation? → see Decision 2026-05-03 (counter labeled by outcome).
- [x] "Toggling auto-load only affects the next app startup" is decided — should the settings UI tell the user that, or is silent next-launch behavior acceptable? → see Decision 2026-05-03 (silent next-launch behavior).
- [x] Verification has no automated tests listed (only manual). Should the date-matching and timezone-adjustment logic have a Go unit test with table-driven cases (DST boundaries, invalid tz, missing row, empty OT/NT)? → see Decision 2026-05-03 (Go unit tests required).
- [x] What HTTP status and body shape is returned on "no reading for today" — 200 with empty `passages` (as the outline implies), or 404? The frontend error path needs to distinguish "no reading" from "endpoint failed". → see Decision 2026-05-03 (200 with empty array).

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
- 2026-05-03: Latency budget set at 250ms-to-interactive picker, 2s-to-rendered passages. Reason: Verifiable target consistent with §3 (Responsiveness over richness); leaves headroom for SPA hydration on slow connections.
- 2026-05-03: Timezone detection goes through a TimezoneProvider platform shim in `web/src/platform/`, mirroring ToggleStore. Reason: §4 (Platform features behind an abstraction) — a native shell can supply its own timezone source without touching feature code.
- 2026-05-03: OT/NT passages display via a tab switcher on a single reading surface (no side-by-side, no responsive split). Reason: Simpler to build, consistent across viewports, removes "or" ambiguity flagged by review.
- 2026-05-03: Settings toggle lives in a new dedicated settings pane opened via a gear icon in the header. Reason: Clean home for this and future toggles; keeps the picker pane focused.
- 2026-05-03: When auto-load is enabled, today's reading always overwrites any prior manual selection on startup. Reason: Predictable behavior for daily-plan followers; simpler than tracking a last-selection date.
- 2026-05-03: `daily-reader.md` is embedded via `embed.FS` in a new `internal/dailyreader/` package. Reason: Compile-time embed; single binary; mirrors how `internal/web/dist` is embedded; clean separation from the HTTP layer.
- 2026-05-03: `/api/daily-reading` returns 200 with `{passages: [], message: "No reading for today"}` when no row matches or both readings are empty. Reason: "No reading" is a valid state, not an error; frontend distinguishes empty array from HTTP failure.
- 2026-05-03: When the query date falls outside the plan's calendar year, wrap around to the same MM/DD of the plan's year (e.g., 2027-01-01 → 2026-01-01 reading). Reason: Bible-in-One-Year is a rolling annual schedule; wrapping keeps the feature working indefinitely without requiring yearly content updates.
- 2026-05-03: `/api/daily-reading` increments a Prometheus counter labeled by outcome (`hit`, `empty`, `error`) on the private metrics server. Reason: Spot tz bugs and plan-exhaustion; matches existing instrumentation pattern on `/api/passage`.
- 2026-05-03: Settings UI does not include messaging about the toggle taking effect on next launch. Reason: Cleaner UI; behavior is discoverable on first reload.
- 2026-05-03: Date-matching, timezone-adjustment, parser, and year-wrap logic require Go unit tests (table-driven: DST boundaries, invalid tz, missing row, empty OT/NT, year wrap-around) before shipping. Reason: TZ/date math is bug-prone; raises the bar before user-visible regressions ship.
- 2026-05-03: Frontend assembles `q` (e.g., `'Num. 15-16'`) from `book` + `chapters` and calls `/api/passage`, which validates via the existing allow-list in `internal/esv/`. A Go test verifies every row in `daily-reader.md` round-trips through the validator. Reason: Preserves the existing validation boundary; round-trip test catches data-file typos at build time.

## Verification

- [ ] Go unit tests (table-driven) in `internal/dailyreader/` covering: parser correctness, date matching, invalid `tz` (400), missing row, empty OT/NT, DST boundaries, and year wrap-around
- [ ] Go round-trip test: every row in the embedded `daily-reader.md` produces a `q` value that passes the existing `internal/esv/` allow-list validator
- [ ] Manual test: toggle on → close and reopen app → daily passages load in background, picker visible, passages appear when ready
- [ ] Manual test: toggle off → app opens to picker pane (no auto-load)
- [ ] Manual test: preference persists across browser refresh
- [ ] Manual test: with auto-load on and a manually-selected passage from a prior session, next startup overwrites with today's reading
- [ ] Manual test: API endpoint `GET /api/daily-reading` returns correct passages for today (check daily-reader.md)
- [ ] Manual test: endpoint failure gracefully degrades (app still usable, user can pick passages manually)
- [ ] Manual test: query for a date outside the plan year (mock or wait until 2027-01-01) wraps to the equivalent prior-year reading
- [ ] Performance: picker interactive within 250ms of startup; passages rendered within 2s on a typical connection
- [ ] Metrics: `hit`, `empty`, and `error` counter outcomes all observable on `127.0.0.1:9090/metrics`

## Related

- `[passage-reader](./passage-reader.md)` — base reading feature this extends
- `PROJECT_CONSTITUTION.md §2` (in-scope) — §3 (Study-first UX)
