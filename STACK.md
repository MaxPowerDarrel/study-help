# Tech Stack

This document records the backend and client technology choices for `study-help`. Unlike [`PROJECT_CONSTITUTION.md`](./PROJECT_CONSTITUTION.md) — which captures durable principles — these are *swappable* implementation choices. Change them via PR with a short rationale.

---

## Choices

| Concern | Choice | Why |
|---|---|---|
| Language / runtime | **Go 1.26** | Already scaffolded. Single static binary. |
| HTTP layer | **`net/http` (stdlib)** | Go 1.22+ routing is sufficient. No framework lock-in, no dependency to maintain. |
| Database | **SQLite** via [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite) | Pure-Go driver — no CGO, no system libs. Single-file DB ships with the binary. Adequate for the expected scale. |
| Query layer | **[`sqlc`](https://sqlc.dev)** | Generates type-safe Go from plain SQL. No ORM magic, no runtime reflection. Migrations and queries stay in `.sql` files. |
| Migrations | **[`goose`](https://github.com/pressly/goose)** | Plain-SQL migrations, simple up/down model, plays well with `sqlc`. |
| Password hashing | **`golang.org/x/crypto/bcrypt`** | Standard, vetted, no surprises. |
| Sessions | **HTTP-only secure cookies**, server-side session store in SQLite | No JWTs, no third-party identity providers at v1. Works inside a WebView (constitutional requirement). |
| Config | **Environment variables** (read at startup) | `ESV_API_KEY`, `DATABASE_URL`, `SESSION_SECRET`. No config files for v1. |
| Deployment target | **Single static binary** + SQLite file on disk | Implicit consequence of the above. Specific host (Fly.io / Railway / VPS) deferred. |
| Client framework | **React** (SPA in `web/`) | Familiar tooling; clean static-bundle target that loads inside a WebView (§4 Frontend is decoupled). Decided in `specs/passage-reader.md`. |
| Client bundler | **[Vite](https://vitejs.dev)** | Fast dev server, no server-only runtime APIs, clean static build output for WebView. Decided in `specs/passage-reader.md`. |
| Metrics | **`/metrics` endpoint, Prometheus-style exposition** | Real-time visibility into ESV-call volume without persistent state. Library (e.g. `prometheus/client_golang`) decided in implementing PR. |

## Explicitly NOT chosen

- **No web framework** (Gin, Echo, Fiber, chi). Stdlib is enough; revisit if routing becomes painful.
- **No ORM** (GORM, ent). `sqlc` covers the type-safety win without the abstraction tax.
- **No JWTs.** Session cookies are simpler, revocable, and a better fit for a single-server app with a WebView client.
- **No Postgres at v1.** SQLite is sufficient; switching later is a real cost but a known one. Revisit if/when multi-host or managed hosting becomes a requirement.
- **No OAuth / social login at v1.** Email + password only. Adds scope without serving focused study.

## When to revisit

| Trigger | Likely change |
|---|---|
| Need to run multiple app instances | SQLite → Postgres |
| Routing logic becomes hard to read in stdlib | Adopt `chi` (minimal, idiomatic) |
| Need background jobs (e.g. ESV cache warmers) | Add a worker goroutine pool; defer real queue (NATS / Asynq) until justified |
| Users ask for "sign in with Google" | Add OAuth as a *parallel* path, not a replacement for password auth |

Each of these is a deliberate "later" — not an omission.