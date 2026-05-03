# Reader UI Refresh

**Status:** Draft
**Created:** 2026-05-03
**Last updated:** 2026-05-03
**Owner:** unassigned

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
- **No Tailwind.** — *Provisional — see Open questions.* Default lean
  is plain CSS / CSS modules with custom-property tokens.
- **No CSS-in-JS migration.** — Runtime cost and toolchain weight don't
  pay back at this surface size.
- **No new client-side state library.** — Theme state lives in
  `ToggleStore` like other toggles.

## User-facing behavior

- **First visit:** the app follows the system theme (light or dark via
  `prefers-color-scheme`). No flash of incorrect theme on cold load.
- **Theme toggle:** lives in the **settings panel**, alongside the
  existing formatting toggles and the auto-load-daily-reading
  toggle. Once the user picks light or dark, that choice sticks
  across reloads. No header chrome added for it.
- **iPad Safari:** the reader fills the viewport with safe-area insets
  honored in both orientations. Tapping a verse / picker control
  registers cleanly without zooming the page. Scrolling the passage
  has the expected momentum / rubber-band feel.
- **Desktop:** visual rhythm (spacing, type scale, colors) is
  consistent across the picker pane, reading surface, and settings.
  No more ad-hoc magic numbers.

## Implementation outline

- **Toolchain bump (prerequisite):** `web/package.json` — **Vite 5 → 7**,
  `@vitejs/plugin-react` to the matching major (5.x line), **React
  18 → 19**, **TypeScript 5.6 → current**. React 19's behavior
  changes (stricter Effects, ref-as-prop, removed legacy APIs) get
  absorbed in this bump rather than deferred. Resolve any breakage;
  rerun `npm run build` and verify the embedded SPA still renders.
- **Design tokens:** new `web/src/styles/tokens.css` (or equivalent)
  exporting CSS custom properties for color, spacing, type scale, and
  radii. Two token sets keyed by `[data-theme="light"]` and
  `[data-theme="dark"]` on `<html>`. Default selector follows
  `prefers-color-scheme` until the user makes a choice.
- **Theme persistence:** new `theme` key in the existing
  `web/src/platform/ToggleStore.ts` (per §4 platform abstraction).
  No new platform module.
- **iPad Safari polish:** update the viewport meta to include
  `viewport-fit=cover`, apply `env(safe-area-inset-*)` paddings on
  the app shell, set `font-size: 16px` on text inputs to suppress
  iOS zoom, apply `-webkit-overflow-scrolling: touch` (or the modern
  equivalent) on scroll containers.
- **Component touch-up:** picker controls, verse rows, and the
  reading surface migrate to the new tokens. No new components — this
  is a refactor of existing styles, not a redesign.
- **STACK.md update:** record the bumped Vite / React / TS majors in
  the same PR(s) (per the skill's "tech choice → update STACK.md"
  rule).

## Open questions

- [ ] CSS strategy: plain CSS files with tokens, CSS Modules, or
      something else? Rule-of-thumb is "the smallest thing that
      consolidates ad-hoc styles" — but pin the choice before
      implementation.
- [x] Where does the theme toggle live in the UI — header, settings
      panel, picker pane, or a small floating control? — *resolved
      2026-05-03 (see Decisions): settings panel.*
- [ ] Sepia / reading-mode in scope at v1, or hard-deferred to a
      follow-up?
- [ ] Auto-follow system theme by default, or explicit user choice
      from first visit?
- [x] Pin Vite to a specific major (6 or 7)? React 19 has known
      behavior changes (e.g. stricter `useEffect`, ref-as-prop) — do
      we want to absorb those now or stay on React 18 and only bump
      Vite + TS? — *resolved 2026-05-03 (see Decisions): Vite 7,
      React 19, TS current.*
- [ ] What is the namespace for the theme key in `ToggleStore`
      (e.g. `theme.mode` vs. `reader.theme`)?
- [x] Concrete pass/fail for "no measurable performance regression" —
      lighthouse score, frame-time on chapter-switch, or qualitative
      smoke test? — *resolved 2026-05-03 (see Decisions): qualitative
      side-by-side smoke test on the same iPad.*
- [ ] Does the toolchain bump land as one PR before the UI work, or
      bundled into a single PR with the refresh?
- [ ] Anything in the existing CSS that's load-bearing for the reader
      (e.g. specific selectors used by tests or screenshots) and
      shouldn't be churned?
- [ ] Are there any existing CSS selectors that are load-bearing
      (used by tests, screenshots, or external references) and must
      not be churned?
- [ ] Define the exact "modern equivalent" planned in place of
      `-webkit-overflow-scrolling: touch`, since that property is
      effectively a no-op on current iOS Safari.
- [ ] Specify the minimum tap-target dimensions ("sized for touch")
      so the iPad polish goal is verifiable.

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
