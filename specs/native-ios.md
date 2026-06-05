# Native iOS App & Daily-Reading Widget

**Status:** In Progress <!-- Draft | In Progress | Shipped | Deprecated -->
**Created:** 2026-06-05
**Last updated:** 2026-06-05
**Owner:** Darrel

## Why

The day-to-day driver of a reading plan is knowing *what today's assignment
is*. On iOS the most direct surface for that is a **home-screen widget** —
glanceable, no app launch, no browser. The reading itself can then happen in a
physical book or in the app.

iOS home-screen widgets cannot be built with web technology; they require a
native Swift WidgetKit extension. That is the one capability the existing
web-only stack cannot deliver, and it is the entire reason this native client
exists. It does **not** reimplement the reader: the app is a thin `WKWebView`
shell over the same hosted SPA.

This required a constitution amendment. `PROJECT_CONSTITUTION.md` §1 previously
read "there is no native shell"; it was amended 2026-06-05 to permit a native
shell **solely to host the widget**, with reading delegated to the embedded
SPA. §4 (stateless server) is unchanged and explicitly reaffirmed: all
native-client state lives on-device.

## Goals

- [ ] A home-screen widget (small + medium) shows the current day's reading
      references for a chosen plan (e.g. "Genesis 1–3 · Romans 1").
- [ ] The widget is configurable via long-press: pick the reading plan and the
      translation a tap should open.
- [ ] The widget refreshes at the user's local midnight so it always shows the
      correct day.
- [ ] When offline / backend down, the widget shows the last cached references
      with a subtle "stale" indicator instead of going blank.
- [ ] Tapping the widget opens the app to the reading experience.
- [ ] The app itself loads the existing hosted SPA, so all reading features
      (Read/Daily tabs, translations, toggles, cross-references, theme,
      session restore) work with zero duplication.
- [ ] Distributable to the owner and family via TestFlight.

## Non-goals

- **No native reader.** The reading surface is the web SPA in a web view. We do
  not reimplement the daily-tab state machine, picker, or HTML rendering in
  Swift.
- **No scripture-text caching.** ESV/YouVersion ToU restrict caching scripture
  content (see `pwa-install.md`). The widget caches only daily *references*
  (book + chapters), which are not scripture text. `GET /api/passage` HTML is
  never cached natively.
- **No server-side state, accounts, or cross-device sync.** Constitution §2/§4
  stand. Widget config + cache are on-device only (App Group).
- **No public App Store listing at v1.** TestFlight only (owner's choice).
- **No backend changes.** The existing endpoints are sufficient.

## Architecture

```
iOS app (StudyHelp)              Widget extension (DailyWidget)
┌─────────────────────┐          ┌──────────────────────────────┐
│ WKWebView           │          │ AppIntentTimelineProvider     │
│   → hosted SPA      │          │   → GET /api/daily-reading    │
│ pull-to-refresh     │          │ cache → App Group             │
│ studyhelp:// open   │◄─tap─────│ widgetURL: studyhelp://daily  │
└─────────────────────┘          └──────────────────────────────┘
          │                                   │
          └───────── App Group (shared) ──────┘
                 group.com.darrelross.studyhelp

         Go backend on Lightsail (UNCHANGED)
         GET /api/daily-reading  ·  GET /api/passage  ·  …
```

Lives in [`../ios/`](../ios/). The `.xcodeproj` is generated from
`ios/project.yml` via XcodeGen (not committed). Three targets: the app, the
widget extension, and a unit-test bundle. A `Shared/` source set compiles into
both the app and the widget.

### API contract consumed (no backend change)

`GET /api/daily-reading?tz=<IANA>&plans=<id>&date=<YYYY-MM-DD>`
(`internal/server/daily.go`). Returns:

```json
{ "plans": [ { "id", "name",
  "passages": [ { "book", "chapters", "testament" } ],
  "message"? } ] }
```

- `tz` is required; the widget sends the device timezone so the server resolves
  the same calendar day the device is on.
- `chapters` is `"3"`, `"1-3"`, or `"1,2,3"`; the widget renders it verbatim
  (en-dash for ranges). Special / catch-up / no-reading days carry an empty
  `passages` array and a `message`.
- Mirrored client-side by `DailyResponse` / `DailyPlanResult` / `DailyPassage`
  in `ios/Shared/Models.swift`, kept in lockstep with `daily_test.go`.

### Widget behavior

- **Configuration** (`ConfigurationAppIntent`): `plan` and `translation`
  `AppEnum`s matching the server catalog (`PlanOption`/`TranslationOption` in
  `ios/Shared/Catalog.swift`). One plan per widget; add a second widget for a
  second plan.
- **Timeline** (`Provider`): fetch today's reading for the configured plan,
  write it to the App Group cache, emit one entry, and reload
  `.after(nextLocalMidnight)`. On failure, fall back to the cached entry with
  `isStale = true`; if no cache exists, show an "Unable to load" message.
- **Rendering** (`DailyWidgetView`): plan name header (+ stale glyph), then
  either the `message` or the reference list. Medium family adds OT/NT chips.
- **Deep link**: `widgetURL` is `studyhelp://daily?translation=<id>`. The app's
  `onOpenURL` reloads the SPA; the SPA restores to the Daily tab via its
  existing `web/src/restore.ts` session restoration.

### App behavior

`WebView` (a `UIViewRepresentable` over `WKWebView`) loads
`Config.backendBaseURL` with the **persistent** data store, so the SPA's
`localStorage` (translation, plan selection, theme, last location) survives
launches exactly as in the browser. Pull-to-refresh reloads. No other native
chrome.

## Attribution & ToU

- Reading happens inside the SPA web view, so **all ESV/YouVersion attribution
  renders exactly as on the web** (ESV ships attribution in its HTML; "Powered
  by YouVersion" appears when NIV is active — see `niv.md`,
  `passage-reader.md`). The native shell strips nothing.
- The widget shows only references (plan-derived metadata), not scripture text,
  so it carries no scripture-attribution obligation and the on-device cache of
  those references is ToU-safe.

## Rate limiting

`passage-reader.md` (2026-05-02) shipped `/api/passage` with no rate limit and
flagged "revisit if usage spikes." Native distribution is a plausible such
trigger, but for owner + family (TestFlight) it is not expected to matter.
Revisit per-IP limiting on `/api/passage` only if real usage warrants it; not a
blocker for this feature.

## Prerequisites

- Mac + Xcode 15+; Apple Developer Program ($99/yr) for App Group capability,
  1-year provisioning, and TestFlight.
- Backend reachable over **HTTPS** — already satisfied by the Lightsail + Caddy
  deployment at `study.example.com` (`deploy-aws.md`). App Transport Security
  blocks plain HTTP, so local-dev against `go run .` needs an `Info.plist`
  exception (documented in `ios/README.md`).

## Verification

1. `curl "http://localhost:8080/api/daily-reading?tz=America/New_York&plans=hope"`
   against `go run .`; note today's references.
2. `xcodebuild test -scheme StudyHelp …` runs the model decode/round-trip tests
   (`ios/StudyHelpTests/DailyReadingTests.swift`).
3. Run the widget in the Simulator (point `BACKEND_BASE_URL` at the local
   server), add it, long-press → Edit → pick Hope/ESV; confirm references match
   step 1.
4. Stop the server, force a widget reload; confirm the cached entry renders with
   the stale glyph.
5. Run the app; confirm the SPA loads, the Daily tab renders, pull-to-refresh
   works, and a widget tap cold-launches the app.
6. Cross-check the same date/plan in widget, app web view, and the live web app
   — references must match.

## Decisions

- **2026-06-05** — Thin web-view shell over a full native SwiftUI reader.
  Reason: the reading UI is already polished and maintained on the web; a native
  reimplementation would duplicate the daily-tab state machine and HTML
  rendering and require keeping two clients in sync. The web can't do widgets;
  that's the only native code we owe.
- **2026-06-05** — Widget configurable via AppIntents (plan + translation)
  rather than reading the web app's `localStorage`. Reason: the extension cannot
  read the web view's `localStorage`; AppIntents is the idiomatic, decoupled iOS
  way and keeps state on-device.
- **2026-06-05** — TestFlight (paid account) over free personal sideload.
  Reason: avoids the 7-day re-sign treadmill and allows installing for family;
  no public App Store listing needed.
- **2026-06-05** — Cache daily *references* on-device but never scripture text.
  Reason: ToU restrict caching scripture content; references are plan metadata
  and safe, and they let the widget survive offline.

## Open questions

- Should the SPA honor a `?tab=daily&date=…` / `?translation=…` deep-link query
  so a widget tap lands on a specific day/translation? Currently the tap relies
  on session-restore landing on the Daily tab. Small `web/src/restore.ts`
  change; deferred as an optional follow-up.
- Lock screen / StandBy widget families and an interactive "mark read" button
  (App Intents) are possible later additions; out of scope for v1.
