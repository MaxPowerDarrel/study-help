# Native Reader (iOS)

**Status:** In Progress <!-- Draft | In Progress | Shipped | Deprecated -->
**Created:** 2026-06-09
**Last updated:** 2026-06-09
**Owner:** Darrel

## Why

The iOS app ([`native-ios.md`](./native-ios.md)) currently delegates all
reading to the hosted SPA in a `WKWebView`. That shell proved the widget and
the distribution pipeline; the reading surface itself, though, is still a web
page inside an app. Rendering passages **natively** — real `AttributedString`
text instead of embedded HTML — buys platform-quality typography, Dynamic
Type, native scrolling and text selection, and instant theme response. This
serves `PROJECT_CONSTITUTION.md` §3 ("Study-first UX", "Responsiveness over
richness") on the device the daily-reading widget already lives on.

This required a second amendment to constitution §1 (2026-06-09, in the same
PR as this spec): the web app is now the **canonical** reading implementation
rather than the *only* one. Reading features are specified and shipped
web-first; the native reader follows. The backend is untouched — the native
reader is just a second consumer of the existing JSON API
(`/api/passage`, `/api/crossref`, `/api/daily-reading`).

## Goals

Parity with the web reading surface as of 2026-06-09:

- [ ] **Read tab** — book/chapter picker over the 66-book canon, single
      chapter or chapter range, prev/next chapter navigation.
- [ ] **Daily tab** — per-plan passage pills, date back/forward + Today,
      daily-header translation picker, plan `message` days, comma-chapter
      fan-out into one concatenated pill body.
- [ ] **Six formatting toggles** matching web defaults (headings, footnotes,
      verse numbers, passage references, cross-references [ESV-only, default
      off], words-of-Christ red [client-side, no re-fetch]).
- [ ] **ESV + NIV**, with "Powered by YouVersion" rendered whenever NIV text
      is visible and ESV's in-HTML attribution preserved verbatim.
- [ ] **Cross-references** — tapping an ESV `.cf` marker opens a native
      popover backed by `GET /api/crossref`.
- [ ] **Footnotes** — tapping a footnote marker shows the note body.
- [ ] **Theme** light/dark/system.
- [ ] **Session restore** — active tab, last read location, last daily date
      (snaps to today on a new calendar day), mirroring
      [`restore-last-location.md`](./restore-last-location.md) on-device.
- [ ] The embedded-SPA web view stays reachable from Settings as a fallback.
- [ ] Each phase lands as one mergeable PR, shippable to TestFlight.

### Parity / accepted gaps (living list)

The web app is canonical; new web reading features need an iOS counterpart
*or* an entry here. Currently: none — the goals above target full parity with
the 2026-06-09 web surface.

## Non-goals

- **No backend changes.** The existing endpoints are sufficient; the server
  never learns a second client exists.
- **No persistent scripture cache.** ESV/YouVersion ToU restrict caching
  scripture content (see [`pwa-install.md`](./pwa-install.md)). Passage HTML
  and parsed text live in memory for the session only; nothing is written to
  disk. Daily *references* caching (widget) is unchanged.
- **No offline reading.** Follows from the above: §5 treats offline as a
  best-effort cache at most, and the no-persistence ToU constraint removes
  even that for scripture text.
- **No Android / macOS client.** iOS-only, building on the existing `ios/`
  project.
- **No replacement of the website.** The web app remains the canonical,
  fully-maintained reading implementation (constitution §1 as amended).
- **No widget changes.** [`native-ios.md`](./native-ios.md) remains the
  widget/shell spec; the widget still consumes only references.

## User-facing behavior

The app opens to a native three-tab layout: **Read**, **Daily**, **Settings**.
Read shows a passage picker and a natively typeset passage with prev/next
chapter controls. Daily shows the same pill bar, date navigation, and
daily-header translation picker as the web. Settings holds the translation
picker, the six formatting toggles, plan selection, theme, and a "Use web
reader" switch that swaps the whole surface back to the embedded SPA.
Tapping an ESV cross-reference letter opens a small popover with the verse
text; tapping a footnote marker shows the note. Reading position, active tab,
and daily date survive relaunch.

Until parity is reached the web view remains the default surface, gaining one
piece of native chrome: a small gear button in the top trailing corner that
opens the native Settings sheet, which hosts the "Native reader (beta)"
switch (the web app keeps its own in-page settings while it is the surface).
Flipping the switch swaps the whole surface to the native tab layout, whose
Settings tab offers the reverse "Use web reader" switch. The default flips to
native once parity lands (phase ⑥).

## Implementation outline

All new code lives under `ios/` — the feature is removable in one PR
(constitution §6, rule 2).

- **Parsing pipeline** (`ios/StudyHelp/Parsing/`): publisher HTML →
  SwiftSoup DOM → `[PassageBlock]` intermediate model (headings, paragraphs
  of attributed inline runs, collected footnotes) → `AttributedString` →
  SwiftUI `Text`. One DOM walker with two class maps: ESV (`verse-num`,
  `h2`/`h3`, `woc`, `a.cf`, footnote markup) and NIV/YouVersion (`yv-vlbl`,
  `yv-h`, `wj`, `.p`, empty `yv-v` boundary spans skipped, no cross-refs).
  **Never-drop rule:** unknown elements render as plain text with links
  preserved — the markup-drift safety net and the ESV attribution guarantee.
  Tappables (cross-refs, footnotes) are encoded as `.link` attributes with
  custom `studyhelp-crossref://` / `studyhelp-footnote://` schemes and
  intercepted via `OpenURLAction`.
- **Networking** (`ios/StudyHelp/Networking/PassageAPI.swift`): client for
  `GET /api/passage` and `GET /api/crossref` (`{canonical, passages}`
  envelope; 429 surfaced distinctly). Deliberately *not* in `ios/Shared/` so
  the widget target structurally cannot fetch scripture text.
- **Canon** (`ios/Shared/Canon.swift`): port of `web/src/canon.ts` (66 books
  + chapter counts, prev/next helpers) backing the picker.
- **Views/state** (`ios/StudyHelp/Reader/`, `Daily/`, `Settings/`,
  `State/`): `TabView` root; view models port the web's `useDailyTab` state
  machine (including comma-chapter fan-out) and `restore.ts` session
  restoration onto App Group `UserDefaults`. Daily reuses the existing
  `ios/Shared/{Models,Catalog,DailyReadingAPI,DateTZ}.swift`.
- **Dependency**: SwiftSoup via SwiftPM, declared in `ios/project.yml`
  (`StudyHelp` target only; the widget takes no new dependency). Listed in
  `STACK.md`.
- **Testing**: fixture-driven — short recorded API responses (a few verses
  each, within ESV quoting allowances) under `ios/StudyHelpTests/Fixtures/`,
  exercised by parser/canon/view-model tests following the existing
  `DailyReadingTests.swift` pattern.

Phasing (one PR each): ① this spec + constitution amendment, ② parsing
engine + API client + canon (no UI change), ③ native Read tab behind an
opt-in beta flag, ④ native Daily tab, ⑤ cross-ref popover + footnotes + full
toggle/theme parity, ⑥ session restore + flip the default to native + doc
sync.

## Open questions

- [ ] Exact ESV footnote markup (container/marker classes) — to be pinned
      down from recorded fixtures in phase ②, not from memory.
- [ ] Does `Shared/DailyReadingAPI.swift` grow a multi-plan overload (joined
      `plans=` param) or does the app loop per selected plan? Decide in
      phase ④ when porting plan selection.
- [ ] Lock-screen widget families and richer deep links remain open in
      [`native-ios.md`](./native-ios.md); a native Daily tab may make the
      `studyhelp://daily` deep link richer (specific date). Revisit in
      phase ⑥.
- [ ] Session restore on App Group `UserDefaults` vs. standard
      `UserDefaults` — App Group would let the widget read last-read state
      later (richer deep links), but standard is sufficient today. Decide in
      phase ⑥.
- [ ] Fixture matrix for phase ②: the five server-sent toggles arrive
      pre-applied in the HTML, so "toggle coverage" means recording fixtures
      at representative toggle combinations (default / all-on / all-off),
      not exercising a client-side path. Confirm the combinations suffice
      when recording.
- [ ] What concretely gates the phase-⑥ default flip: all Goals checkboxes
      checked, or an explicit owner sign-off that the "Parity / accepted
      gaps" list is empty? (Owner sign-off proposed.)

## Decisions

- 2026-06-09: **Supersedes** the 2026-06-05 decision in
  [`native-ios.md`](./native-ios.md) ("Thin web-view shell over a full native
  SwiftUI reader"). The widget and shell shipped and proved the pipeline; a
  native reading surface is now wanted for reading quality. The cost of being
  wrong is low: the web app stays canonical, the in-app web view stays as a
  fallback, and the whole feature deletes in one PR.
- 2026-06-09: **SwiftSoup** for HTML parsing, over
  `NSAttributedString(html:)` (WebKit-backed, main-thread-only, discards the
  `class` attributes that carry `.woc`/`.cf`/`.yv-vlbl` semantics) and
  Foundation `XMLParser` (strict XML; publisher HTML with named entities and
  unclosed tags aborts it). First third-party dependency in the iOS project;
  pinned via SwiftPM in `project.yml`.
- 2026-06-09: Passage-fetching and parsing code lives in `ios/StudyHelp/`,
  **not** `ios/Shared/` — keeps "the widget never touches scripture text"
  structurally true rather than merely conventional.
- 2026-06-09: In-memory, session-scoped passage caching only — an `NSCache`
  of parsed `[PassageBlock]`s keyed by query + translation + the five
  **server-sent** toggles. The words-of-Christ toggle is excluded from the
  key: it is render-time-only, so flipping it re-renders the cached blocks
  without a re-fetch or re-parse. Nothing persisted, per ToU.
- 2026-06-09: Native rendering ships **opt-in** (beta switch, web view
  default) and becomes the default only at parity (phase ⑥), so existing
  TestFlight users see no regression mid-experiment.
- 2026-06-09: Test fixtures are short recorded excerpts (a few verses each)
  kept within ESV quoting allowances.

## Verification

- [ ] Phase ②: `xcodegen generate && xcodebuild test` green; fixture tests
      cover verse-number runs per toggle, woc/wj tagging in both markup
      dialects, `.cf` hrefs lifted verbatim, attribution tail surviving the
      round-trip, malformed HTML degrading to plain text instead of throwing.
- [ ] Phases ③–⑤: side-by-side simulator vs live web app — John 3 / Psalm 23
      in ESV and NIV with every toggle flipped; a comma-chapter daily day and
      a plan-`message` day; Mark 1 cross-ref tap; NIV shows the YouVersion
      footer wherever its text renders.
- [ ] Phase ⑥: kill/relaunch restores tab + passage; daily date snaps to
      today across a simulated midnight; widget tap lands on the native Daily
      tab; the web-fallback switch round-trips cleanly.
- [ ] Widget regression check each phase (`Shared/` types untouched or
      test-covered).

## Related

- Extends: [`native-ios.md`](./native-ios.md) (app shell + widget)
- Mirrors on-device: [`passage-reader.md`](./passage-reader.md),
  [`multi-translation.md`](./multi-translation.md), [`niv.md`](./niv.md),
  [`cross-references.md`](./cross-references.md),
  [`auto-load-daily-reading.md`](./auto-load-daily-reading.md),
  [`multi-plan.md`](./multi-plan.md),
  [`restore-last-location.md`](./restore-last-location.md)
- Constitution: `PROJECT_CONSTITUTION.md` §1 (as amended 2026-06-09), §3,
  §4, §6
- External: [SwiftSoup](https://github.com/scinfu/SwiftSoup)
