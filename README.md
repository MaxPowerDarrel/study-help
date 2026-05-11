# study-help

A Bible reader optimized for **focused study** of scripture a chapter or section at a time. A web app, designed to be enjoyable in any modern browser — including Safari on iPad.

See [`PROJECT_CONSTITUTION.md`](./PROJECT_CONSTITUTION.md) for purpose, principles, and non-goals, and [`STACK.md`](./STACK.md) for the tech choices.

## About this project

This is a personal project, built and maintained by one developer for personal use and as a public reference. The code is shared openly so others can read, learn from, or fork it, but I am not actively soliciting contributions — issues and PRs may not get a timely response, and scope is intentionally narrow per the constitution. If you'd like to run your own copy, see [Quickstart](#quickstart) below.

## Status

Reader, multi-translation (ESV + NIV), and daily reading (Bible-in-One-Year + Hope 2026) have shipped — see the per-feature specs under [`specs/`](./specs/). The Go server proxies all scripture requests (upstream API keys never reach the browser), exposes a localhost-only Prometheus counter, and serves the React SPA from an embedded Vite build. The accounts, highlights, and notes features were retired on 2026-05-07; the server is now stateless.

## Stack

Go 1.26 stdlib HTTP server (stateless, no database), React + Vite SPA. See [`STACK.md`](./STACK.md) for the full list and rationale.

## Quickstart

Prerequisites: Go 1.26+, Node 20+, an [ESV API key](https://api.esv.org/), and a [YouVersion Platform key](https://platform.youversion.com/) (for NIV).

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
| `ESV_API_KEY` | Required. ESV API key — never reaches the browser. |
| `YOUVERSION_APP_KEY` | Required. YouVersion Platform key for NIV — never reaches the browser. |
| `ADDR` | Bind address (default `:8080`). |
| `ENV` | `prod` (default) or `dev`. Informational; nothing currently varies on it. |

## Development

```sh
go build ./...          # build server
go test ./...           # run tests
go test -run TestName ./path/to/pkg

cd web && npm run dev   # SPA dev server, proxies /api to localhost:8080
cd web && npm run format
```

## Layout

- `main.go` — wires config → scripture provider registry (ESV + NIV) → public + private servers, with graceful SIGINT/SIGTERM shutdown.
- `internal/config/` — env-var-driven `Config`.
- `internal/server/` — public HTTP server (`/healthz`, `/api/passage`, `/api/daily-reading`, embedded SPA) + private `127.0.0.1:9090/metrics`.
- `internal/scripture/` — translation-provider abstraction; `internal/esv/` and `internal/youversion/` are the registered providers.
- `internal/canon/` — 66-book canon: `LookupBook` and the canon-aware `q` allow-list validator.
- `internal/dailyreader/` — embedded reading-plan parsers (Bible-in-One-Year + Hope 2026).
- `internal/web/` — embeds the Vite build output (`internal/web/dist/`) into the Go binary.
- `web/` — React SPA (Vite + TypeScript). Platform-touching code (localStorage, timezone) goes through `web/src/platform/`; daily-tab logic lives under `web/src/daily/`.
- `specs/` — one markdown file per feature; index in [`specs/README.md`](./specs/README.md).

For the per-package detail (handler shapes, hook contracts, file-by-file responsibilities) see [`CLAUDE.md`](./CLAUDE.md).

## Workflow

No feature work on `main`. Every change lands on a feature branch and merges via pull request.
