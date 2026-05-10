# Restore Last Location

**Status:** In Progress
**Created:** 2026-05-10
**Last updated:** 2026-05-10 (implementation landed on `restore-last-location` branch)
**Owner:** unassigned

---

## Why

A fresh load of the SPA always drops the user back to the same defaults: the Read tab, John 3, today's daily reading. Anything else they were doing — a chapter they were partway through, the Daily tab they had open — is forgotten between sessions. Restoring the user to where they were when they last closed the app removes a small but constant friction point and makes returning to study feel like resuming, not restarting.

This explicitly fulfills the "revisit" path noted in `auto-load-daily-reading.md` (Decision 2026-05-03: *"No 'remember last tab' preference at v1... If users miss the auto-open behavior, revisit with a `defaultTab` setting"*). It also extends the existing client-only-preferences pattern already used for translation, formatting toggles, plan selection, and theme.

**Constitutional support:**
- `PROJECT_CONSTITUTION.md` §3 (Study-first UX): reduces friction between sessions so the reading surface is what the user lands on, not a generic default.
- `PROJECT_CONSTITUTION.md` §4 (The server is stateless; the only client-side preference lives in localStorage): this feature extends that pattern. No server state introduced.
- `PROJECT_CONSTITUTION.md` §4 (Platform features behind an abstraction): all storage goes through the existing `ToggleStore` interface in `web/src/platform/`.

## Goals

- [ ] On reload, the app opens to the same tab the user was last on (Read or Daily).
- [ ] On reload to the Read tab, the same book + chapter the user was last viewing is shown.
- [ ] On reload to the Daily tab on the **same calendar day**, the same selected date the user had pinned is shown.
- [ ] On reload to the Daily tab on a **new calendar day**, the date snaps to today (a stale pinned date is not restored).
- [ ] All persistence is per-device, via `localStorage`, with no server round-trip and no cross-device sync.
- [ ] Corrupted or missing storage falls back to current defaults (Read tab, John 3, today) without error.

## Non-goals

- **Scroll position within the passage** — *partial-scroll restoration is iffy when toggles or translations change between sessions; outside this scope.*
- **Picker pane open/closed state** — *transient UI; doesn't belong in "where I was."*
- **Active pill index on the Daily tab** — *resets on plan/translation/date change by design (see `useDailyTab.ts`); not a stable "location."*
- **Cross-device sync** — *requires server state; explicitly out per `PROJECT_CONSTITUTION.md` §4.*
- **Restoring across plan/translation changes** — *the read-tab passage is restored verbatim; translation and toggles already persist independently and apply to the restored passage.*

## User-facing behavior

A user who navigates to Romans 8 on the Read tab and closes the browser tab, then reopens the app the next minute or the next week, lands directly on Romans 8 in the Read tab.

A user who switches to the Daily tab, then closes the browser tab, then reopens the app, lands on the Daily tab. If they had been navigating around past dates (e.g., catching up on yesterday's reading), and they reopen *the same calendar day*, they land on the same selected date. If they reopen on *a later calendar day*, the Daily tab opens to today's reading, not the stale pinned date.

First-ever launch (no stored state): same defaults as today — Read tab, John 3, today's daily reading.

Clearing browser data resets to the same first-ever-launch defaults.

There is no UI for this feature. No toggle, no settings entry — restoration is the new default behavior.

## Implementation outline

- **New** `web/src/restore.ts` — three small stores plus their hooks:
  - `useStoredTab()` → `[Tab, setter]`, key `reader.active-tab`, raw string validated against `"read" | "daily"`.
  - `useStoredReadRef()` → `[ChapterRef, setter]`, key `reader.read-location`, JSON `{ bookIndex, chapter }`, validated against `CANON` bounds.
  - `useStoredDailyDate(today)` → `[string, setter]`, key `reader.daily-date`, JSON `{ date, savedOn }`. On read, if `savedOn !== today`, returns `today` instead of the stored date (snap-to-today rule).
- **New** `web/src/restore.test.ts` — covers hydration, round-trip, invalid-storage fallback, and the snap-to-today rule.
- **Modified** `web/src/daily/useDailyTab.ts` — accepts `selectedDate` and `setSelectedDate` as parameters from the caller (rather than owning them internally). The internal `useState<string>(todayString)` goes away. `todayString()` is exported so callers can pass a single shared "today" value.
- **Modified** `web/src/App.tsx` — three small swaps replacing the existing `useState` calls for tab, ref, and (newly lifted) `selectedDate`. App.tsx now owns `selectedDate` and passes it through to `useDailyTab`.
- **All storage** flows through `defaultToggleStore` from `web/src/platform/ToggleStore.ts`, per §4.
- **Hook pattern** matches `web/src/translations/useTranslation.ts` (simple `useState` initializer + write-on-set), not the `useSyncExternalStore` pattern used by toggles/plans. Cross-tab live sync is intentionally not provided — the user's tab choice in window A shouldn't yank window B mid-reading.

## Open questions

- [ ] Should the snap-to-today rule fire on any `savedOn !== today` mismatch (including timezone changes mid-session, e.g. travel), or only on a strictly later date? Current spec says "new calendar day" but the implementation check is a string inequality.
- [ ] When `useDailyTab`'s `selectedDate` ownership lifts to `App.tsx`, does the existing fetchId race guard in `useDailyTab.ts` still behave correctly when `selectedDate` changes from a parent-driven hydration vs. a user click? Worth a targeted test.
- [ ] Should `reader.active-tab` validate against the live `Tab` union at runtime (so a future tab rename doesn't silently restore an unknown value), and what's the fallback — Read tab or "last known good"?
- [ ] If a user has a stored read-location pointing at a chapter that is valid today but a future canon table edit removes/renames a book, does the out-of-bounds fallback path log anything for debugging, or silently drop to John 3?
- [ ] Should clearing localStorage from devtools while the app is open trigger any in-app reset, or is reload-required the intentional behavior (matches the no-cross-tab-sync decision)?
- [ ] Is there a need to namespace these keys further (e.g. `reader.session.*`) to distinguish "where I was" from durable preferences like `reader.toggles` and `reader.plans`, in case a future "reset session" affordance wants to clear only the former?

## Decisions

- 2026-05-10: Persist active tab, read-tab passage, and daily-tab selected date (with the snap-to-today rule). Scroll position, picker open state, and active pill index are explicitly excluded. Reason: scope clarification with user; the three included items form a stable "where I was," while the excluded items are either transient UI or change semantics across sessions in ways that make restoration confusing.
- 2026-05-10: Three separate localStorage keys under the `reader.*` namespace (`reader.active-tab`, `reader.read-location`, `reader.daily-date`). Reason: matches the existing `reader.toggles` / `reader.plans` convention; one key per entity makes corruption isolated and validation simple.
- 2026-05-10: Daily-date entry stores `{ date, savedOn }` rather than just `date`. On read, if `savedOn !== today`, the stored value is treated as expired and the hook returns `today`. Reason: implements the user's "snap to today on a new day" rule with a self-contained validity check; doesn't require a separate cleanup pass or a TTL system.
- 2026-05-10: Hooks use the `useState` + write-on-set pattern (`useTranslation`-style), not `useSyncExternalStore` (`useToggles`/`usePlanSelection`-style). Reason: cross-tab live sync of "where you were" would be disorienting (a tab change in window A shouldn't rip window B out of its passage). The simpler pattern matches the semantics.
- 2026-05-10: All three stores live in a single `web/src/restore.ts` file rather than three sibling files. Reason: total surface area is small (~80 lines), the three entities share a theme, and a single file is easier to read than three near-identical 25-line files. Reverses the "directory per concept" pattern only because there is no concept here larger than session restoration itself.
- 2026-05-10: `useDailyTab`'s internal `selectedDate` ownership is lifted to `App.tsx`. `useDailyTab` accepts `selectedDate` and `setSelectedDate` as parameters. Reason: the daily-tab restoration hook needs to compare `savedOn` against today using the same timezone-aware date string the daily tab already uses; lifting state to the common parent keeps a single source of truth without introducing a global. `todayString()` is exported from `useDailyTab.ts` to avoid duplication.
- 2026-05-10: Supersedes `auto-load-daily-reading.md` Decision 2026-05-03 ("No 'remember last tab' preference at v1"). Reason: that decision left the door open ("If users miss the auto-open behavior, revisit"); user has now requested it. The active-tab persistence in this spec is the `defaultTab` setting that decision anticipated, but applied as automatic restoration rather than a settings toggle.

## Verification

- [ ] `web/src/restore.test.ts` — hydration on first read, round-trip on write, invalid-JSON fallback, out-of-canon-bounds fallback, unknown-tab-string fallback, snap-to-today rule when `savedOn` is yesterday.
- [ ] `cd web && npm test` passes (existing tests + new ones).
- [ ] `cd web && npm run build` succeeds (TypeScript + Vite build).
- [ ] `go test ./...` passes (no Go-side changes, but sanity-check the build still works after `internal/web/dist/` regenerates).
- [ ] Manual: navigate to Romans 8 on the Read tab → reload browser → land on Romans 8 in Read tab.
- [ ] Manual: switch to Daily tab → reload → land on Daily tab.
- [ ] Manual: on Daily tab, navigate to a past date → reload same day → past date persists.
- [ ] Manual: on Daily tab, navigate to a past date → simulate next day (override `Date` or set system clock) → reload → snaps to today, not the stale date.
- [ ] Manual: clear localStorage entirely → reload → defaults to Read tab / John 3 / today.
- [ ] Manual: corrupt storage (`localStorage.setItem("reader.read-location", "garbage")`) → reload → falls back to John 3, no console error.

## Related

- [`auto-load-daily-reading`](./auto-load-daily-reading.md) — Decision 2026-05-03 deferred a "remember last tab" preference; this spec is the revisit.
- [`passage-reader`](./passage-reader.md) — establishes the chapter picker, formatting toggles, and the localStorage-per-device pattern this spec extends.
- [`multi-plan`](./multi-plan.md) — establishes the `reader.*` localStorage namespace convention and the "client-only, no cross-device sync" guardrail.
- [`multi-translation`](./multi-translation.md) — the prior canonical example of "the only client-side preference lives in localStorage" cited in the constitution.
- `PROJECT_CONSTITUTION.md` §3 (Study-first UX), §4 (stateless server, platform abstraction).
