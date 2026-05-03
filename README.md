# study-help

A Bible reader optimized for **focused study** of scripture a chapter or section at a time, with personal highlights and notes. A web app, designed to be enjoyable in any modern browser — including Safari on iPad.

See [`PROJECT_CONSTITUTION.md`](./PROJECT_CONSTITUTION.md) for purpose, principles, and non-goals, and [`STACK.md`](./STACK.md) for the tech choices.

## Status

Passage reader shipped (see [`specs/passage-reader.md`](./specs/passage-reader.md)). The Go server proxies ESV passage requests, exposes a localhost-only Prometheus counter, and serves the React SPA from an embedded Vite build. Highlights, notes, and accounts are not yet implemented — each will get its own spec under [`specs/`](./specs/).

## Stack

Go 1.26 stdlib HTTP server, SQLite via `modernc.org/sqlite` (pure-Go, no CGO), goose migrations, React + Vite SPA. See [`STACK.md`](./STACK.md) for the full list and rationale.

## Quickstart

Prerequisites: Go 1.26+, Node 20+, an [ESV API key](https://api.esv.org/).

```sh
# 1. Configure environment (copy and fill in)
cp .env.example .env
# export the vars in your shell, or use direnv — the app does not auto-load .env

# 2. Build the SPA into the Go embed directory
cd web && npm install && npm run build && cd ..

# 3. Run the server
go run .
```

The app listens on `:8080` by default (override with `ADDR`). A private metrics endpoint is exposed on `127.0.0.1:9090/metrics`.

### Required env vars

| Var | Purpose |
|---|---|
| `SESSION_SECRET` | Random 32+ byte string used to sign session cookies |
| `ESV_API_KEY` | ESV API key — never reaches the browser |
| `ADDR` | Bind address (default `:8080`) |
| `DATABASE_URL` | SQLite file path (default `./sqlite.db`) |

## Development

```sh
go build ./...          # build server
go test ./...           # run tests
go test -run TestName ./path/to/pkg

cd web && npm run dev   # SPA dev server, proxies /api to localhost:8080
cd web && npm run format
```

## Layout

- `main.go` — wires config → DB → public server + private metrics server, with graceful SIGINT/SIGTERM shutdown.
- `internal/config/` — env-var-driven `Config`.
- `internal/db/` — opens SQLite (WAL, foreign keys, 5s busy timeout) and runs embedded goose migrations.
- `internal/server/` — public `*http.Server` exposing `/healthz`, `/api/passage`, and the embedded SPA; plus the private metrics server.
- `internal/esv/` — server-side ESV API client and the canon-aware `q` allow-list validator.
- `internal/web/` — embeds the Vite build output (`internal/web/dist/`) into the Go binary.
- `web/` — React SPA (Vite + TypeScript). Platform-touching code (localStorage, timezone, etc.) goes through `web/src/platform/` to keep feature code portable and testable.
- `specs/` — one markdown file per feature; index in [`specs/README.md`](./specs/README.md).

## Workflow

No feature work on `main`. Every change lands on a feature branch and merges via pull request.
