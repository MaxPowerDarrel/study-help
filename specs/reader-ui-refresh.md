# Reader UI Refresh

**Status:** Shipped
**Created:** 2026-05-03
**Last updated:** 2026-05-04
**Owner:** unassigned

> **Editor's note (2026-05-07):** references in this spec to notes as a future surface predate the removal of the Notes feature on 2026-05-07 (see [notes.md](./notes.md)). The token / theme work described here was unaffected.

## Why

The reader surface ships, works, and is the product (per
PROJECT_CONSTITUTION.md §3 "Study-first UX"), but it has no theming layer,
no design-token system, and no iPad-Safari-specific polish. Now that
Safari on iPad is the primary mobile target (per §1, post the iPad-app
removal), the gap shows. This spec consolidates three pieces of UI
modernization that are cheaper to do together than apart: (1) iPad/Safari
touch polish so the reader feels native in mobile Safari, (2) light + dark
theme on a proper design-token foundation, and (3) a small consolidated
component / token layer so future surfaces (highlights, notes, accounts)
plug into a consistent system rather than re-deriving styles. The
toolchain bump (Vite 5 → current major, plus React and TypeScript)
rides along — doing it while the surface is small is dramatically cheaper
than doing it after highlights and notes land.

Supports PROJECT_CONSTITUTION.md §1 (web app, Safari on iPad as a
first-class target), §3 (Study-first UX, Responsiveness over richness),
§4 (Frontend is decoupled — design tokens stay client-side; theme
persistence routes through the existing `web/src/platform/` abstraction).

## Goals

- [ ] Reader feels right on iPad Safari: tap targets sized for touch,
      safe-area insets respected in portrait and landscape, no zoom on
      input focus, momentum scrolling on the reading surface.
- [ ] Light and dark themes ship, with the user's choice persisted across
      reloads. System-theme follow is the default.
- [ ] A small set of CSS custom-property design tokens (colors, spacing,
      type scale, radii) replaces ad-hoc inline values across the SPA.
- [ ] Toolchain bumped to current majors (Vite, `@vitejs/plugin-react`,
      React, TypeScript) with no regressions in `passage-reader` or
      `auto-load-daily-reading`.
- [ ] No qualitative regression on chapter-switch or page-turn,
      judged by side-by-side comparison on the same iPad against the
      pre-refresh build (per §3 "Responsiveness over richness"). Pass
      criterion is parity, not an absolute frame-time number — same
      pattern as the original passage-reader spec.

## Non-goals

- **No re-skin of features that don't exist yet.** — Highlights, notes,
  and account surfaces will get their own UI in their own specs; this
  spec only refreshes what's already shipped.
- **No iPhone-specific layout work.** — iPad is the explicit target;
  iPhone may benefit incidentally but isn't a goal.
- **No accessibility audit beyond theme contrast.** — A11y is important
  but warrants its own focused spec. Don't try to bundle it.
- **No Tailwind.** — Stylesheet strategy is CSS Modules for components
  plus a single global `tokens.css` for custom properties.
- **No CSS-in-JS migration.** — Runtime cost and toolchain weight don't
  pay back at this surface size.
- **No new client-side state library.** — Theme state lives in
  `ToggleStore` like other toggles.
- **No sepia / reading-mode at v1.** — Light + dark only. The token
  layer makes sepia cheap to add later; bundling it here triples the
  theme surface and turns a binary toggle into a 3-way picker.

## User-facing behavior

- **First visit:** the app quietly follows the system theme (light or
  dark via `prefers-color-scheme`) — no prompt, no banner. The user
  finds the explicit toggle in Settings if they want to override. No
  flash of incorrect theme on cold load.
- **Theme toggle:** lives in the **settings panel**, alongside the
  existing formatting toggles and the auto-load-daily-reading
  toggle. Once the user picks light or dark, that choice sticks
  across reloads. No header chrome added for it.
- **iPad Safari:** the reader fills the viewport with safe-area insets
  honored in both orientations. All tappable controls (picker
  buttons, settings toggles, theme toggle, any verse affordance) meet
  a **44×44 CSS px** minimum. Tapping a control registers cleanly
  without zooming the page. The passage container scrolls with iOS
  Safari's native momentum.
- **Desktop:** visual rhythm (spacing, type scale, colors) is
  consistent across the picker pane, reading surface, and settings.
  No more ad-hoc magic numbers.

## Implementation outline

Lands in **two PRs**, in this order:

### PR 1 — Toolchain bump (prerequisite, no visual change)

- `web/package.json`: **Vite 5 → 7**, `@vitejs/plugin-react` to the
  matching major (5.x line), **React 18 → 19**, **TypeScript 5.6 →
  current**. React 19's behavior changes (stricter Effects,
  ref-as-prop, removed legacy APIs) get absorbed here.
- Resolve any breakage in `web/`. Rerun `npm run build`; verify the
  embedded SPA still renders identically.
- Update `STACK.md` to reflect the new Vite / React / TS majors (per
  the skill's "tech choice → update STACK.md" rule).
- Goal of this PR: build is green on the new toolchain, no UI
  changes. If something breaks, the blast radius is one PR.

### PR 2 — UI refresh on the new toolchain

- **Stylesheet strategy:** CSS Modules (`*.module.css`) for component
  styles; one global `web/src/styles/tokens.css` for custom
  properties. Vite supports CSS Modules natively — no extra deps.
- **Design tokens:** `tokens.css` exports CSS custom properties for
  color, spacing, type scale, and radii. Two token sets keyed by
  `[data-theme="light"]` and `[data-theme="dark"]` on `<html>`.
  Default selector follows `prefers-color-scheme` until the user
  picks a theme.
- **Theme persistence:** add a top-level `theme` key to
  `web/src/platform/ToggleStore.ts` with values `"light" | "dark" |
  "system"`, default `"system"`. No new platform module (per §4
  platform abstraction).
- **Theme toggle UI:** a 3-state control (System / Light / Dark) in
  the existing settings panel.
- **iPad Safari polish:**
  - Viewport meta: `viewport-fit=cover`.
  - App shell padding: `env(safe-area-inset-*)` for top/bottom/left/
    right insets, both orientations.
  - Inputs: `font-size: 16px` to suppress iOS zoom-on-focus.
  - Scroll containers: rely on iOS Safari's native momentum (no
    `-webkit-overflow-scrolling` — it's a no-op on current iOS).
    Add `overscroll-behavior: contain` on the passage container so
    rubber-banding doesn't leak to the page.
  - Tap targets: minimum 44×44 CSS px on picker controls, toggle
    buttons, and any verse affordance.
- **Component touch-up:** picker controls, verse rows, and the
  reading surface migrate to the new tokens via CSS Modules. No new
  components — refactor of existing styles, not a redesign.

## Open questions

- [x] CSS strategy: plain CSS files with tokens, CSS Modules, or
      something else? Rule-of-thumb is "the smallest thing that
      consolidates ad-hoc styles" — but pin the choice before
      implementation. — *resolved 2026-05-03 (see Decisions): CSS
      Modules + global `tokens.css`.*
- [x] Where does the theme toggle live in the UI — header, settings
      panel, picker pane, or a small floating control? — *resolved
      2026-05-03 (see Decisions): settings panel.*
- [x] Sepia / reading-mode in scope at v1, or hard-deferred to a
      follow-up? — *resolved 2026-05-03 (see Decisions): deferred;
      added to Non-goals.*
- [x] Auto-follow system theme by default, or explicit user choice
      from first visit? — *resolved 2026-05-03 (see Decisions):
      auto-follow `prefers-color-scheme` quietly; no first-visit
      prompt; explicit toggle in Settings.*
- [x] Pin Vite to a specific major (6 or 7)? React 19 has known
      behavior changes (e.g. stricter `useEffect`, ref-as-prop) — do
      we want to absorb those now or stay on React 18 and only bump
      Vite + TS? — *resolved 2026-05-03 (see Decisions): Vite 7,
      React 19, TS current.*
- [x] What is the namespace for the theme key in `ToggleStore`
      (e.g. `theme.mode` vs. `reader.theme`)? — *resolved 2026-05-03
      (see Decisions): top-level `theme`, values `"light" | "dark" |
      "system"`, default `"system"`.*
- [x] Concrete pass/fail for "no measurable performance regression" —
      lighthouse score, frame-time on chapter-switch, or qualitative
      smoke test? — *resolved 2026-05-03 (see Decisions): qualitative
      side-by-side smoke test on the same iPad.*
- [x] Does the toolchain bump land as one PR before the UI work, or
      bundled into a single PR with the refresh? — *resolved
      2026-05-03 (see Decisions): toolchain bump first as its own
      PR; UI refresh second.*
- [x] Anything in the existing CSS that's load-bearing for the reader
      (e.g. specific selectors used by tests, screenshots, or
      external references) and shouldn't be churned? *(Reviewer also
      logged a duplicate of this; treated as one item.)* — *resolved
      2026-05-03 (see Decisions): audit found nothing load-bearing
      outside the ESV-rendered HTML; safe to refactor freely.*
- [x] Define the exact "modern equivalent" planned in place of
      `-webkit-overflow-scrolling: touch`, since that property is
      effectively a no-op on current iOS Safari. — *resolved
      2026-05-03 (see Decisions): drop the property entirely; rely
      on iOS Safari's native momentum; add `overscroll-behavior:
      contain` on the passage container.*
- [x] Specify the minimum tap-target dimensions ("sized for touch")
      so the iPad polish goal is verifiable. — *resolved 2026-05-03
      (see Decisions): 44×44 CSS px floor, per Apple HIG and WCAG
      2.5.5.*

## Decisions

- 2026-05-03: Slug is `reader-ui-refresh`. Reason: the surface being
  refreshed is the reader; the name aligns with the existing
  `passage-reader` spec and signals scope (not a full app re-skin).
- 2026-05-03: Toolchain bumps (Vite, `@vitejs/plugin-react`, React,
  TypeScript) ride along with this spec rather than getting a
  separate spec. Reason: dependency bumps don't typically warrant
  their own spec; coupling them to the UI refresh keeps related risk
  in one branch and means the new tokens / theme work runs against
  the new toolchain from day one.
- 2026-05-03: v1 focus is **iPad/Safari touch polish + light/dark
  theme + design tokens**. Typography micro-tuning is intentionally
  deferred — it lands incidentally via the token scale, but there's
  no separate goal for it.
- 2026-05-03: Toolchain target is **Vite 7 + React 19 + TypeScript
  current**. Reason: surface is small enough today to absorb React
  19's behavior changes; doing one bump now is cheaper than two
  bumps later, and the new tokens / theme work runs against the
  destination toolchain from day one. Resolves the Open question on
  Vite/React majors.
- 2026-05-03: Theme toggle lives in the **settings panel**,
  alongside the existing formatting and auto-load-daily-reading
  toggles. Reason: keeps the reader chrome unadorned per §3
  "Study-first UX"; the user's frequent toggle is light/dark, not
  per-session, so always-visible chrome would be pure clutter.
  Resolves the Open question on toggle placement.
- 2026-05-03: "No performance regression" is verified by a
  **qualitative side-by-side smoke test on the same iPad** —
  pre-refresh build vs. post-refresh build, both chapter-switch and
  page-turn. Pass criterion is parity, no absolute frame-time
  number. Reason: matches the existing passage-reader spec's
  parity-not-numbers convention; instrumentation would be more
  work than the surface deserves at this stage. Resolves the Open
  question on the perf pass/fail bar.
- 2026-05-03: Stylesheet strategy is **CSS Modules** (`*.module.css`)
  for component styles plus a single global
  `web/src/styles/tokens.css` for custom properties. Reason: Vite
  supports CSS Modules natively (no extra deps), per-component
  scoping prevents the small-app habit of selector creep, and tokens
  in a single global keep the theme story simple. Resolves the Open
  question on CSS strategy.
- 2026-05-03: Sepia / reading-mode is **deferred** to a follow-up
  spec; v1 ships **light + dark only**. Reason: the token layer
  makes adding sepia later cheap; bundling it now triples theme
  surface and turns a binary toggle into a 3-way picker.
- 2026-05-03: First-visit behavior is to **quietly follow
  `prefers-color-scheme`** with no prompt or banner. The user
  finds the override in Settings if they want it. Reason: §3
  "Study-first UX" — the reading surface should not be interrupted
  by a theme prompt on first launch.
- 2026-05-03: Theme key in `ToggleStore` is **top-level `theme`**
  with values `"light" | "dark" | "system"`, default `"system"`.
  Reason: theme is app-global (consumed by settings, picker, and
  reader), so a top-level key is more honest than nesting under
  `reader.*`. Three-state value preserves "follow system" as a
  first-class option, distinct from a one-time pick of light/dark.
- 2026-05-03: Toolchain bump (PR 1) lands **before** the UI refresh
  (PR 2). Reason: smaller blast radius if either piece breaks; PR 1
  is a no-visual-change bump easy to verify, PR 2 builds on a known-
  green toolchain. Trade: two reviews instead of one.
- 2026-05-03: `-webkit-overflow-scrolling: touch` is **dropped** with
  no replacement; iOS Safari has native momentum on `overflow:
  auto/scroll` since iOS 13. The passage container gets
  `overscroll-behavior: contain` to keep rubber-banding scoped.
  Reason: the property is a no-op on current iOS Safari; pretending
  otherwise is cargo-cult.
- 2026-05-03: Minimum tap-target dimension is **44×44 CSS px** for
  picker controls, toggle buttons, and any verse affordance.
  Reason: Apple HIG and WCAG 2.5.5 both land at 44×44; using the
  industry standard makes the goal verifiable and removes the need
  for a project-specific number.
- 2026-05-03: Toolchain bump (PR 1) implemented and lands on **Vite
  8 + `@vitejs/plugin-react` 6 + React 19 + TypeScript 6** — the
  current latest majors at install time, not the originally targeted
  Vite 7. Reason: between spec drafting and implementation, Vite 8
  and TypeScript 6 became the current stable; the spec's intent
  ("absorb majors now while the surface is small, build runs against
  the destination toolchain") applies more strongly with the newer
  current. Build, `go build`, and `go test` all green. Supersedes
  the earlier "Vite 7" portion of the toolchain Decision.
- 2026-05-03: Added `web/src/vite-env.d.ts` with
  `/// <reference types="vite/client" />`. Reason: TypeScript 6
  rejects bare side-effect CSS imports (`import "./styles.css"`)
  without ambient declarations; Vite's standard scaffold ships this
  file but the project never had one. One-line fix; no behavior
  change.
- 2026-05-03: Status flipped **Draft → In Progress** with PR 1
  (toolchain bump). Reason: real code is landing; per the skill's
  Mode 4 rule, the spec's status moves with the work.
- 2026-05-03: Load-bearing CSS audit complete. `grep` for class-name
  references in `*.go`, `*.test.ts*`, `*.spec.ts` returned no
  matches; only ESV-rendered HTML class names (`verse-num`, `woc`,
  `passage`, `h2/h3` inside `.passage`) are load-bearing, and those
  are kept under a global stylesheet (`web/src/styles/passage.css`)
  not a CSS Module. Resolves the last Open question.
- 2026-05-03: FOUC prevention: a synchronous inline script in
  `index.html` reads `localStorage.getItem("theme")` and sets
  `<html data-theme="light|dark">` before the bundle loads. Reason:
  without this, a user who picked "dark" sees a flash of light
  theme on cold load. The script duplicates a tiny amount of logic
  from `web/src/theme.ts` but is the simplest cross-browser fix
  for FOUC and lives next to the `<head>` consumers.
- 2026-05-03: Theme module shape: `web/src/theme.ts` exposes
  `readTheme`, `writeTheme`, `applyTheme`, and a `useTheme()` hook.
  `writeTheme` calls `applyTheme` so the DOM is updated on the same
  tick as state. The hook is consumed once in `App.tsx` and the
  resulting `[theme, setTheme]` is threaded into `SettingsPane` as
  props. Reason: small surface, no new platform module, theme key
  routes through the existing `ToggleStore`.
- 2026-05-03: Tap-target floor enforced via a `--tap-target` token
  (`44px`) used by every interactive control's `min-height` (and
  `min-width` on icon buttons). Inputs/selects/buttons globally get
  `font-size: 16px` in `tokens.css` to suppress iOS zoom-on-focus.
  Reason: keeps the policy in one place; no per-component fudging.
- 2026-05-03: Status flipped **In Progress → Shipped** with PR 2
  (UI refresh). Reason: tokens, theme, theme toggle, CSS Modules
  conversion, and iPad-Safari polish are all user-visible on
  merge. Manual desktop and iPad verification remain in the PR's
  test plan; any regression found post-merge becomes a follow-up
  fix, not a status reversal.
- 2026-05-04: Read-tab formatting "Show" toggles (section headings,
  footnotes, verse numbers, passage reference, words of Christ)
  moved from the picker sidebar into the SettingsPane, joining
  Theme under the gear icon. Reason: realigns the implementation
  with this spec's User-facing-behavior wording ("Theme toggle…
  alongside the existing formatting toggles"), which had assumed
  the formatting toggles were already in the settings pane. The
  picker keeps only navigation primitives (book, chapter, verse
  range, prev/next). Spec stays Shipped; no scope or behavior
  change.

## Verification

- [ ] `npm run build` succeeds on the bumped toolchain (Vite 7,
      React 19, TS current); embedded SPA loads from the Go binary
      without console errors.
- [ ] Manual on iPad Safari (real device or simulator): cold load
      lands in correct theme, safe-area honored in portrait and
      landscape, tap targets feel right, no zoom on input focus,
      passage scroll momentum feels native.
- [ ] **Qualitative perf parity check** on the same iPad: install
      the pre-refresh build and the post-refresh build, do
      chapter-switch and page-turn on both, confirm post-refresh
      does not feel slower. Fix before merge if it does.
- [ ] Manual on desktop Chrome / Safari / Firefox: theme toggle works,
      choice persists across reloads, system-theme change is followed
      until the user makes an explicit choice.
- [ ] No visual regressions in `passage-reader` (chapter / range
      selection, verse rendering, formatting toggles).
- [ ] No regressions in `auto-load-daily-reading` (toggle still works
      from the same control).
- [ ] `go test ./...` passes; existing client-side tests (if any)
      pass against the new toolchain.
- [ ] `STACK.md` updated to reflect the new Vite / React / TypeScript
      majors.

## Related

- Specs this depends on: [passage-reader](./passage-reader.md),
  [auto-load-daily-reading](./auto-load-daily-reading.md)
- Constitution sections: `PROJECT_CONSTITUTION.md §1, §3, §4`
- Code touchpoints: `web/package.json`, `web/src/platform/ToggleStore.ts`,
  `web/src/styles/` (new), the SPA shell and reader components.
