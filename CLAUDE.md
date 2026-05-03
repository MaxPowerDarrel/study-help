# CLAUDE.md
This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Project direction, principles, guardrails, and non-goals live in [`PROJECT_CONSTITUTION.md`](./PROJECT_CONSTITUTION.md). Read it before proposing features or architectural changes.

## Status

Early scaffolding. The HTTP server boots and serves one health endpoint; no feature endpoints, no migrations, no client, no tests yet.

**Layout:**

- `main.go` — wires config → DB → server, with graceful SIGINT/SIGTERM shutdown.
- `internal/config/` — `Config` struct loaded from env vars: `ADDR` (default `:8080`), `DATABASE_URL` (default `./sqlite.db`), `SESSION_SECRET` (required), `ESV_API_KEY` (required).
- `internal/db/` — opens SQLite (WAL, foreign keys, 5s busy timeout) via `modernc.org/sqlite` (pure-Go, no CGO) and runs goose migrations from the embedded `migrations/` directory. **No migrations exist yet** (only `.gitkeep`).
- `internal/server/` — `*http.Server` with stdlib `http.ServeMux`, logging middleware, sensible timeouts. Currently exposes only `GET /healthz` (DB ping → JSON).

**Other repo artifacts:**

- [`PROJECT_CONSTITUTION.md`](./PROJECT_CONSTITUTION.md) — durable principles, guardrails, non-goals. Read before proposing features.
- [`STACK.md`](./STACK.md) — current backend tech choices (Go 1.26, stdlib HTTP, SQLite, goose, sqlc planned, bcrypt, cookie sessions). Swappable; change via PR.
- [`specs/`](./specs/) — living feature specs. One markdown file per feature. Index in `specs/README.md`.
- `.env.example` — template for required env vars.

There is no frontend, no static assets, no `sqlc`-generated code, and no tests yet. When code is added, update this file with the actual structure — do not invent content describing things that do not exist.

## Commands

- Build: `go build ./...`
- Run: `go run .`
- Test (once tests exist): `go test ./...`
- Run a single test: `go test -run TestName ./path/to/pkg`

## Workflow

No feature work on `main`. Every change lands on a feature branch and merges via pull request.