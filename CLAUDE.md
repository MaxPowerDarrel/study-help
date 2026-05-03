# CLAUDE.md
This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Project direction, principles, guardrails, and non-goals live in [`PROJECT_CONSTITUTION.md`](./PROJECT_CONSTITUTION.md). Read it before proposing features or architectural changes.

## Status

Reader feature shipped per `specs/passage-reader.md`. The Go server proxies ESV passage requests, exposes a Prometheus counter on a localhost-only metrics port, and serves the React SPA from an embedded Vite build. No DB-backed features yet (highlights/notes still pending their own specs).

**Layout:**

- `main.go` — wires config → DB → public server + private metrics server, with graceful SIGINT/SIGTERM shutdown.
- `internal/config/` — `Config` struct loaded from env vars: `ADDR` (default `:8080`), `DATABASE_URL` (default `./sqlite.db`), `SESSION_SECRET` (required), `ESV_API_KEY` (required).
- `internal/db/` — opens SQLite (WAL, foreign keys, 5s busy timeout) via `modernc.org/sqlite` (pure-Go, no CGO) and runs goose migrations from the embedded `migrations/` directory. **No migrations exist yet** (only `.gitkeep`).
- `internal/server/` — public `*http.Server` (stdlib `http.ServeMux`) exposing `GET /healthz`, `GET /api/passage`, and the embedded SPA at `/`. Plus a private metrics server bound to `127.0.0.1:9090` exposing `GET /metrics` (Prometheus text exposition, counter-only).
- `internal/esv/` — server-side ESV API client (`api.esv.org/v3/passage/html/`) and the canon-aware `q` allow-list validator. The ESV API key never reaches the browser (constitution §4).
- `internal/web/` — embeds the Vite build output (`internal/web/dist/`) into the Go binary. The directory is populated by `npm run build` in `web/`.
- `web/` — React SPA (Vite + TypeScript). Picker pane, reading surface with formatting toggles, light/dark/system theme. Component styles live in `*.module.css` (CSS Modules); design tokens in `web/src/styles/tokens.css`; ESV-rendered HTML styled globally in `web/src/styles/passage.css`. Theme persistence at `web/src/theme.ts`, wired through the localStorage-backed `ToggleStore` at `web/src/platform/ToggleStore.ts` (the §4 platform-abstraction layer for browser APIs).

**Other repo artifacts:**

- [`PROJECT_CONSTITUTION.md`](./PROJECT_CONSTITUTION.md) — durable principles, guardrails, non-goals. Read before proposing features.
- [`STACK.md`](./STACK.md) — current backend tech choices (Go 1.26, stdlib HTTP, SQLite, goose, sqlc planned, bcrypt, cookie sessions). Swappable; change via PR.
- [`specs/`](./specs/) — living feature specs. One markdown file per feature. Index in `specs/README.md`.
- `.env.example` — template for required env vars.

## Commands

- Build SPA: `cd web && npm install && npm run build` (outputs to `internal/web/dist/`)
- Build server: `go build ./...`
- Run: `go run .` (also serves the embedded SPA; metrics on `127.0.0.1:9090`)
- Test: `go test ./...`
- Run a single test: `go test -run TestName ./path/to/pkg`
- SPA dev mode: `cd web && npm run dev` (proxies `/api` to `localhost:8080`)
- Format SPA: `cd web && npm run format` (Prettier; also runs automatically via PostToolUse hook in `.claude/settings.json`)

## Workflow

No feature work on `main`. Every change lands on a feature branch and merges via pull request.  **Always create a branch before doing any work**