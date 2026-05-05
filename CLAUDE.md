# CLAUDE.md
This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Project direction, principles, guardrails, and non-goals live in [`PROJECT_CONSTITUTION.md`](./PROJECT_CONSTITUTION.md). Read it before proposing features or architectural changes.

## Status

Reader feature shipped per `specs/passage-reader.md`. The Go server proxies ESV passage requests, exposes a Prometheus counter on a localhost-only metrics port, and serves the React SPA from an embedded Vite build. Accounts shipped per `specs/accounts.md`: server (signup / signin / signout / me, cookie sessions, per-IP + per-account rate limiting) + client (top-right chip, slide-in sign-in / create-account panel, sign-out menu). Highlights shipped per `specs/highlights.md`: server (`internal/highlights/` + three endpoints behind `RequireUser`) and client (`web/src/highlights/` floating toolbar, range-based selection → `<mark>` overlay, `web/src/platform/SelectionAdapter`); ESV client sends `include-verse-anchors=true` unconditionally so the client can locate selections by verse + offset. Notes still pending.

**Layout:**

- `main.go` — wires config → DB → auth service → public server + private metrics server, with graceful SIGINT/SIGTERM shutdown.
- `internal/config/` — `Config` struct loaded from env vars: `ADDR` (default `:8080`), `DATABASE_URL` (default `./sqlite.db`), `SESSION_SECRET` (required), `ESV_API_KEY` (required), `ENV` (default `prod`; set to `dev` to drop the `Secure` cookie flag for plain-HTTP localhost).
- `internal/db/` — opens SQLite (WAL, foreign keys, 5s busy timeout) via `modernc.org/sqlite` (pure-Go, no CGO) and runs goose migrations from the embedded `migrations/` directory. Migrations: `00001_users.sql`, `00002_sessions.sql`, `00003_highlights.sql`.
- `internal/server/` — public `*http.Server` (stdlib `http.ServeMux`) exposing `GET /healthz`, `GET /api/passage`, `GET /api/daily-reading`, the four `/api/auth/*` endpoints, the three `/api/highlights` endpoints, and the embedded SPA at `/`. The auth middleware is mounted on `/api/*` (not the SPA) so static assets don't trigger session lookups. Plus a private metrics server bound to `127.0.0.1:9090` exposing `GET /metrics` (Prometheus text exposition, counter-only).
- `internal/auth/` — accounts package: bcrypt password hashing (12-char minimum), SQLite-backed sessions with sha256-hashed cookie tokens (raw token never stored), 30-day sliding window, dual-bucket in-memory rate limiter (per-IP + per-account; resets on restart), session-lookup middleware. Surfaces `POST /api/auth/{signup,signin,signout}` and `GET /api/auth/me`.
- `internal/highlights/` — per-user range-based annotations against the ESV canon. Exposes `GET /api/highlights?book&chapter`, `POST /api/highlights`, `DELETE /api/highlights/{id}`, all behind `auth.RequireUser`. Half-open character ranges over `(book, chapter, start_verse, start_offset, end_verse, end_offset)`; overlap check in the Go handler before insert; cross-user delete returns 404 (no ID enumeration). Reuses `esv.LookupBook` for canon validation.
- `internal/esv/` — server-side ESV API client (`api.esv.org/v3/passage/html/`), the canon-aware `q` allow-list validator, and `LookupBook(name)` shared with `internal/highlights/`. The ESV API key never reaches the browser (constitution §4).
- `internal/web/` — embeds the Vite build output (`internal/web/dist/`) into the Go binary. The directory is populated by `npm run build` in `web/`.
- `web/` — React SPA (Vite + TypeScript). Picker pane, reading surface with formatting toggles, light/dark/system theme, accounts UI. Component styles live in `*.module.css` (CSS Modules); design tokens in `web/src/styles/tokens.css` (incl. `--color-highlight`); ESV-rendered HTML styled globally in `web/src/styles/passage.css` (incl. `mark.highlight`). Theme persistence at `web/src/theme.ts`, wired through the localStorage-backed `ToggleStore` at `web/src/platform/ToggleStore.ts`; `web/src/platform/SelectionAdapter.ts` is the matching abstraction for the browser Selection API (the two are the §4 platform-abstraction layer). Auth client lives in `web/src/auth/`: `api.ts` (discriminated-union fetchers for the four `/api/auth/*` endpoints), `useUser.ts` (cold-load `/me` hydration + signin/signup/signout actions), `AuthChip.tsx` (top-right chip + sign-out menu), `AuthPanel.tsx` (slide-in panel with sign-in default + create-account toggle). Highlights client lives in `web/src/highlights/`: `api.ts` (discriminated-union fetchers for the three `/api/highlights*` endpoints), `useHighlights.ts` (per-passage cache; re-fetches on book/chapter change and after every successful mutation), `parseSelection.ts` + `applyHighlights.ts` (Range ↔ verse-offset tuples, `<mark data-highlight-id="...">` overlay), `HighlightToolbar.tsx` (floating toolbar with selection / existing / guest / error modes), `PassageView.tsx` (extracted from `App.tsx` to host the article ref + selection wiring). Tests via vitest + @testing-library/react: `npm test` in `web/`.

**Other repo artifacts:**

- [`PROJECT_CONSTITUTION.md`](./PROJECT_CONSTITUTION.md) — durable principles, guardrails, non-goals. Read before proposing features.
- [`STACK.md`](./STACK.md) — current backend tech choices (Go 1.26, stdlib HTTP, SQLite, goose, sqlc planned, bcrypt, cookie sessions). Swappable; change via PR.
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
- Build & run via Docker: `docker compose up --build` (set `ESV_API_KEY` and `SESSION_SECRET` in your shell first; override `APP_PORT` if 8080 is taken)

## Workflow

No feature work on `main`. Every change lands on a feature branch and merges via pull request.  **Always create a branch before doing any work**