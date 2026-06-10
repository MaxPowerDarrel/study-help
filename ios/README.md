# study-help — iOS app & widget

The native iOS client: a **SwiftUI reader** (Read / Daily / Settings tabs)
over the same JSON API the web app consumes, plus the **home-screen widget**
that shows the day's reading assignment. The app began as a thin `WKWebView`
shell existing only to host the widget ([`../specs/native-ios.md`](../specs/native-ios.md));
the native reading surface was added per
[`../specs/native-reader.md`](../specs/native-reader.md) and is the default,
with the embedded SPA kept one switch away as a fallback. The **web app
remains the canonical reading implementation** — reading features ship
web-first (see [`../PROJECT_CONSTITUTION.md`](../PROJECT_CONSTITUTION.md) §1).

## What's here

| Path | Role |
|---|---|
| `Shared/` | Compiled into **both** targets: daily-reading API client, Codable models, plan/translation catalog, canon table, App Group cache, date helpers. Never touches scripture text. |
| `StudyHelp/` | App target — the native reader (`Reader/`, `Daily/`, `Settings/`, `State/`), the HTML→`AttributedString` parsing pipeline (`Parsing/`, SwiftSoup), the scripture API client (`Networking/`, deliberately not in `Shared/`), and the fallback `WKWebView` surface. |
| `DailyWidget/` | WidgetKit extension — `AppIntentTimelineProvider` that fetches `GET /api/daily-reading`, caches to the App Group, and refreshes at the user's next local midnight. Configurable (plan + translation) via long-press. |
| `StudyHelpTests/` | XCTest suite: fixture-driven parser tests, view-model state machines, settings/session-restore persistence, model decode tests. |
| `project.yml` | [XcodeGen](https://github.com/yonaskolb/XcodeGen) spec — the `.xcodeproj` is generated, not committed. |

The Go backend needs **no changes**; it already proxies scripture with secrets
server-side and is stateless (constitution §4). The widget consumes only daily
*references* (book + chapters), never scripture text — so caching them
on-device is fine under the ESV/YouVersion ToU. The native reader holds
passage data **in memory only** (session `NSCache`); nothing scripture-shaped
is ever persisted.

## Build & run

Prerequisites: a Mac with **Xcode 15+** and (for on-device + TestFlight) an
**Apple Developer Program** membership.

```sh
brew install xcodegen        # one-time
cd ios
xcodegen generate            # produces StudyHelp.xcodeproj from project.yml
open StudyHelp.xcodeproj
```

In Xcode:

1. Set your **Team** on both targets (StudyHelp, DailyWidgetExtension) under
   Signing & Capabilities, or set `DEVELOPMENT_TEAM` in `project.yml` and
   re-run `xcodegen generate`.
2. Confirm the **App Group** `group.com.darrelross.studyhelp` is enabled on
   both targets (it's declared in the `.entitlements` files). If you change the
   bundle prefix, update the group id in the entitlements **and**
   `Shared/AppGroup.swift` so they match.
3. Run the **StudyHelp** scheme on a simulator or device. Add the widget from
   the widget gallery; long-press → **Edit Widget** to pick a plan/translation.

## Pointing at a backend

`Config.backendBaseURL` defaults to `https://study.example.com` (the Lightsail
deployment, HTTPS via Caddy — see `../specs/deploy-aws.md`). Override it without
editing code by setting the `BACKEND_BASE_URL` environment variable in the Run
scheme.

For local dev against `go run .` on your Mac:

- Use your Mac's LAN IP (`http://192.168.x.x:8080`), not `localhost`, so a
  physical device can reach it.
- Plain HTTP needs an **App Transport Security** exception. Add to
  `StudyHelp/Info.plist` *temporarily* (do not ship it):

  ```xml
  <key>NSAppTransportSecurity</key>
  <dict>
    <key>NSAllowsLocalNetworking</key><true/>
  </dict>
  ```

## Tests

```sh
xcodegen generate
xcodebuild test -scheme StudyHelp -destination 'platform=iOS Simulator,name=iPhone 15'
```

(If that destination doesn't exist on your machine, pick any simulator id
from `xcrun simctl list devices available` and use `-destination 'id=…'`.)

### Passage fixtures

`StudyHelpTests/Fixtures/*.json` are real `GET /api/passage` /
`GET /api/crossref` responses that pin both providers' markup for the
native-reader parser (`specs/native-reader.md`). The parser tests assert
`unknownClasses` is empty, so if ESV or YouVersion change their HTML, the
suite flags it as soon as fixtures are re-recorded. Re-record any of them
against the live backend (or `go run .`) with, e.g.:

```sh
curl -s "https://study.darrel.io/api/passage?q=Psalm%2023&translation=ESV" \
  -o StudyHelpTests/Fixtures/esv-psalm23-default.json
```

Keep fixtures to short passages — they live in the repo and must stay within
ESV's quoting allowances.

## Distribution (TestFlight)

Archive the **StudyHelp** scheme (Product → Archive) and upload to App Store
Connect → TestFlight. Internal testers (incl. family) install via the
TestFlight app. No public App Store listing is required.
