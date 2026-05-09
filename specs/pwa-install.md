# PWA Install

**Status:** Shipped
**Created:** 2026-05-09
**Last updated:** 2026-05-09
**Owner:** unassigned

---

## Why

`study-help` targets Safari on iPad as a first-class browser surface (PROJECT_CONSTITUTION.md §1). Today users open it as a regular tab, which means Safari's address bar and tab strip eat vertical space that should belong to the reading surface, and there is no first-class "launch the app" path on the home screen.

Adding the minimum Web App Manifest + Apple meta surface lets users **Add to Home Screen** and launch the app full-screen as a standalone window — no Safari chrome, larger reading area, an icon that looks intentional. It also unlocks the equivalent install flow on Android Chrome as a side benefit.

**Constitutional support:**

- §1 (purpose / iPad-first browser surface): standalone install gives Safari-on-iPad the focused reading shell it deserves without introducing a native app.
- §3 (Study-first UX, "the reading surface is the product"): install removes browser chrome, giving the reading surface the full viewport.
- §4 (Frontend is decoupled): the install path is pure static asset + meta tags. The Go server gains one MIME registration line and nothing else.

## Goals

- [x] iPad Safari can Add to Home Screen and launch in a standalone window with the correct app name and icon.
- [x] Android Chrome shows the "installable" badge, with both standard and maskable icon variants.
- [x] Browser tab favicon shows the project mark instead of the Vite default.
- [x] iOS status bar tints to the current theme background (`#fdfcf8` light, `#14130f` dark).
- [x] No service worker, no caching of `/api/passage` or `/api/daily-reading` responses.

## Non-goals

- **Service worker / offline cache.** Explicitly excluded — ESV and YouVersion terms-of-use restrict what scripture content may be cached, and §5 already limits offline to "best-effort cache, not a guarantee". Revisiting requires a separate spec that first reconciles ToU.
- **Push notifications.** Would require both a service worker and per-user state, the latter forbidden by §2 and §5.
- **Background sync, share targets, file handlers.** None match a reader's job.
- **Custom-designed icon.** The shipped mark is a generated italic-serif "S" on the parchment-tone accent (`#4a3a2a`); a hand-designed icon can replace it without spec changes.

## User-facing behavior

On iPad Safari:
- Share sheet → **Add to Home Screen** proposes the name "Study" and the apple-touch-icon (180×180).
- Launching from the home-screen icon opens a standalone window with no address bar or tab strip.
- The status bar tint follows the OS theme via `<meta name="theme-color">` light/dark variants.

On Android Chrome:
- The browser surfaces the install prompt (or "Install app" menu item) once the manifest + icons load.
- Adaptive-icon launchers use `icon-maskable-512.png` (content inset to ~70% safe zone); other launchers use `icon-512.png` / `icon-192.png`.

In a desktop browser:
- The tab favicon becomes the SVG mark.
- Chrome devtools → Application → Manifest panel parses the manifest with no warnings and shows an "installable" badge.

Nothing else changes. The app behaves identically inside the standalone window and inside a regular tab.

## Implementation outline

- **`web/public/manifest.webmanifest`** — minimal Web App Manifest. `display: "standalone"`, `start_url: "/"`, `scope: "/"`, `name: "study-help"`, `short_name: "Study"`, `theme_color`/`background_color` set to the light-theme background (`#fdfcf8`), and an `icons` array referencing the three PNGs below (192 + 512 with `purpose: "any"`, plus 512 with `purpose: "maskable"`). Vite copies anything in `web/public/` to the build root unchanged, so this lands at `/manifest.webmanifest`.
- **Icons in `web/public/`** — generated as a serif italic "S" on the accent color (`#4a3a2a`) via ImageMagick (`magick -draw roundrectangle … -annotate "S"`) at 1024×1024, then resized:
  - `icon-192.png` (192×192) — Android home-screen
  - `icon-512.png` (512×512) — Android splash
  - `icon-maskable-512.png` (512×512, square; content inset for the adaptive-icon safe zone)
  - `apple-touch-icon.png` (180×180) — iOS home-screen (iOS ignores manifest icons for this)
  - `favicon.svg` — browser-tab mark, scales crisply
- **`web/index.html` head additions** — manifest link, favicon link, apple-touch-icon link, and the iOS-only meta tags Safari still requires (`apple-mobile-web-app-capable`, `mobile-web-app-capable`, `apple-mobile-web-app-status-bar-style`, `apple-mobile-web-app-title`), plus a `<meta name="theme-color">` pair with `media="(prefers-color-scheme: …)"` matching the light/dark `--color-bg` tokens in `web/src/styles/tokens.css`.
- **Server: register `.webmanifest` MIME type** — Go's default `mime.TypeByExtension(".webmanifest")` is empty, so `http.FileServer` would otherwise serve the manifest as `application/octet-stream` and browsers would skip it. `internal/web/embed.go` gains a one-line `init()` calling `mime.AddExtensionType(".webmanifest", "application/manifest+json")`. No routing or handler changes — the existing `spaHandler` already serves any file present in the embedded `dist/`.
- **No CSP, no service worker, no cache headers.** No CSP middleware exists, and the browser's built-in HTTP cache handles the manifest and icons exactly as it does any other static file from the embed. Scripture-response caching behavior is unchanged.

## Open questions

None — scope is intentionally narrow.

## Decisions

- 2026-05-09: Install-only PWA, no service worker. Reason: ESV/YouVersion ToU restrict what may be cached; §5 already treats offline as best-effort. Going install-only delivers the standalone-launch win without taking on cache-invalidation risk or ToU exposure.
- 2026-05-09: `theme_color` / `background_color` in the manifest match the light-theme background (`#fdfcf8`); the live `<meta name="theme-color">` uses `prefers-color-scheme` media queries for both light and dark. Reason: manifest only supports a single static color (used for the splash and locked-in OS UI), but the iOS status-bar tint reads the live meta tag so it can follow the OS theme.
- 2026-05-09: Maskable icon uses the same mark inset to ~70% of the canvas, on the same accent background. Reason: keeps the mark inside Android's adaptive-icon safe zone without needing a separate design.
- 2026-05-09: Generate icons via ImageMagick `-draw roundrectangle` + `-annotate` rather than rsvg/SVG-text. Reason: macOS ImageMagick's MSVG delegate cannot resolve system fonts through `font-family`, so SVG-text rendering produced an empty glyph; native draw uses the font directly via `-font` and renders correctly.
- 2026-05-09: Register `.webmanifest` server-side rather than rename to `.json`. Reason: `application/manifest+json` is the W3C-recommended MIME type and `.webmanifest` is the recommended extension; one `mime.AddExtensionType` line is cheaper than fighting the spec.

## Verification

- [x] `cd web && npm run build` succeeds; `internal/web/dist/` contains `manifest.webmanifest`, `favicon.svg`, and all four icon PNGs.
- [x] `go test ./...` and `cd web && npm test` both pass — change is additive, no regressions.
- [x] `curl -I http://localhost:8765/manifest.webmanifest` → `Content-Type: application/manifest+json`.
- [x] `curl -I http://localhost:8765/apple-touch-icon.png` → `Content-Type: image/png`.
- [x] `curl -I http://localhost:8765/favicon.svg` → `Content-Type: image/svg+xml`.
- [x] `curl -s http://localhost:8765/manifest.webmanifest` returns valid JSON with the expected fields.
- [x] `curl -s http://localhost:8765/` includes the manifest link, apple-touch-icon link, and both light/dark `theme-color` meta tags in the head.
- [ ] Manual (iPad Safari): Share → Add to Home Screen proposes "Study" and the apple-touch-icon; launching from the home screen opens a standalone window with no address bar; status-bar tint matches the current theme.
- [ ] Manual (desktop Chrome): devtools → Application → Manifest parses the manifest with no warnings; "installable" badge appears.
- [ ] Manual (Android Chrome): install prompt or menu entry appears; installed app uses the maskable icon on adaptive-icon launchers.

## Related

- `[passage-reader](./passage-reader.md)` — the reading surface that benefits from the recovered viewport space.
- `[reader-ui-refresh](./reader-ui-refresh.md)` — supplies the `--color-bg` tokens used by the `theme-color` meta pair.
- `[deploy-aws](./deploy-aws.md)` — the deployment target where install actually matters; install requires HTTPS, which Caddy provides.
- `PROJECT_CONSTITUTION.md` §1 (iPad-first browser surface), §3 (Study-first UX), §4 (Frontend decoupled), §5 (no offline-first sync engine — bounds this spec).
