# CLAUDE.md
This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Project direction, principles, guardrails, and non-goals live in [`PROJECT_CONSTITUTION.md`](./PROJECT_CONSTITUTION.md). Read it before proposing features or architectural changes.

## Status

Reader feature shipped per `specs/passage-reader.md`. The Go server proxies ESV and YouVersion (NIV) passage requests, exposes a Prometheus counter on a localhost-only metrics port, and serves the React SPA from an embedded Vite build. Daily reading shipped (`specs/auto-load-daily-reading.md`) with multi-plan support per `specs/multi-plan.md` (Bible-in-One-Year + Hope 2026; checkbox picker in settings); each daily pill renders as one HTML blob, with a daily-header translation picker and per-day date navigation. The Notes feature was removed on 2026-05-07 (privacy concern over storing free-form user prose); see `specs/archive/notes.md`. The Accounts and Highlights features (and the SQLite database that backed them) were retired on 2026-05-07 — no per-user feature remained, so the entire identity layer went with them. The server is now stateless: no DB, no sessions, no rate limiting. ESV cross-references shipped (`specs/cross-references.md`): an off-by-default toggle adds inline `.cf` markers to ESV passages, and clicking one opens a verse-text popover backed by `GET /api/crossref` (ESV-only; NIV emits no markers). Translation preference, formatting toggles, plan selection, theme, **active tab, last read passage, and last-pinned daily date** all persist in `localStorage` (per device); see `specs/restore-last-location.md` for the session-restoration design. See `specs/archive/accounts.md`, `specs/archive/highlights.md`, and `specs/archive/daily-annotations.md` for the historical record.

**Layout:**

- `main.go` — wires config → public server + private metrics server, with graceful SIGINT/SIGTERM shutdown.
- `internal/config/` — `Config` struct loaded from env vars: `ADDR` (default `:8080`), `ESV_API_KEY` (required), `YOUVERSION_APP_KEY` (required, used by the NIV provider), `ENV` (default `prod`; `dev` is informational only now that there's no `Secure` cookie to gate).
- `internal/server/` — public `*http.Server` (stdlib `http.ServeMux`) exposing `GET /healthz`, `GET /api/passage`, `GET /api/crossref` (ESV-only cross-reference lookup for the popover; verse-level refs validated by `canon.ValidateRefList`), `GET /api/daily-reading`, and the embedded SPA at `/`. Plus a private metrics server bound to `127.0.0.1:9090` exposing `GET /metrics` (Prometheus text exposition, counter-only).
- `internal/canon/` — book/chapter table and `ValidateQuery` (allow-list of `<book> <chapter>`, ranges, and verse refs).
- `internal/scripture/` — provider abstraction; `Registry` dispatches by translation ID.
- `internal/esv/` — server-side ESV API client (`api.esv.org/v3/passage/html/`) plus the `Provider` adapter. The ESV API key never reaches the browser (constitution §4).
- `internal/youversion/` — YouVersion Platform API client for NIV (`bible_id=111`) plus the `Provider` adapter; envelope-rewraps the YouVersion response into the same `{passages: [...]}` shape the SPA expects from ESV.
- `internal/dailyreader/` — embedded reading-plan parsers (Bible-in-One-Year markdown, 2026 Hope plan markdown). Returns the day's passages per plan; the SPA fetches each pill against the active translation.
- `internal/web/` — embeds the Vite build output (`internal/web/dist/`) into the Go binary. The directory is populated by `npm run build` in `web/`.
- `web/` — React SPA (Vite + TypeScript). Picker pane, reading surface with formatting toggles, light/dark/system theme, daily-tab with multi-plan pills. Component styles live in `*.module.css` (CSS Modules); design tokens in `web/src/styles/tokens.css`; ESV/NIV-rendered HTML styled globally in `web/src/styles/passage.css`. Theme persistence at `web/src/theme.ts`, wired through the localStorage-backed `ToggleStore` at `web/src/platform/ToggleStore.ts` (consumed via `useSyncExternalStore` so multiple subscribers stay in sync). `web/src/platform/TimezoneProvider.ts` is the abstraction over `Intl.DateTimeFormat().resolvedOptions().timeZone`. Translation state lives in `web/src/translations/`: `useTranslation.ts` (localStorage-backed), `store.ts` (read/write helpers), `catalog.ts` (registry). Daily-tab logic lives under `web/src/daily/`: `useDailyTab.ts` owns the load/reset effects, fetchId race guard, and `passageQueries` (fans comma-separated chapters into multiple fetches that are concatenated into one HTML blob per pill); `DailyPanel.tsx` renders one active pill plus the daily header. Both the Read tab (`App.tsx`) and Daily tab render passage HTML through the shared `web/src/PassageArticle.tsx`, which also owns the cross-reference click → popover behavior (`fetchCrossref` → `/api/crossref`). Session restoration (`web/src/restore.ts`) persists the active tab, last read passage, and last-pinned daily date to localStorage so reopening the app drops the user back where they were; the daily date snaps to today on a new calendar day via a `{date, savedOn}` shape. `App.tsx` owns these via `useStoredTab` / `useStoredReadRef` / `useStoredDailyDate` and passes `selectedDate` down into `useDailyTab`. Tests via vitest + @testing-library/react: `npm test` in `web/`.

**Other repo artifacts:**

- [`PROJECT_CONSTITUTION.md`](./PROJECT_CONSTITUTION.md) — durable principles, guardrails, non-goals. Read before proposing features.
- [`STACK.md`](./STACK.md) — current backend tech choices (Go 1.26, stdlib HTTP, stateless). Swappable; change via PR.
- [`specs/`](./specs/) — living feature specs. One markdown file per feature. Index in `specs/README.md`.
- `.env.example` — template for required env vars.
- `Dockerfile`, `compose.yaml`, `.dockerignore` — three-stage build (Node SPA → Go binary → distroless runtime) for running the app locally without a host Go/Node toolchain. See [`specs/docker.md`](./specs/docker.md).

## Commands

- Build SPA: `cd web && npm install && npm run build` (outputs to `internal/web/dist/`)
- Build server: `go build ./...`
- Run: `go run .` (also serves the embedded SPA; metrics on `127.0.0.1:9090`)
- Test: `go test ./...`
- Run a single test: `go test -run TestName ./path/to/pkg`
- SPA dev mode: `cd web && npm run dev` (proxies `/api` to `localhost:8080`)
- SPA tests: `cd web && npm test` (vitest + @testing-library/react; jsdom environment)
- Format SPA: `cd web && npm run format` (Prettier; also runs automatically via PostToolUse hook in `.claude/settings.json`)
- Build & run via Docker: `docker compose up --build` (set `ESV_API_KEY` and `YOUVERSION_APP_KEY` in your shell first; override `APP_PORT` if 8080 is taken)

## Workflow

No feature work on `main`. Every change lands on a feature branch and merges via pull request.  **Always create a branch before doing any work**
