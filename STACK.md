# Tech Stack

This document records the backend and client technology choices for `study-help`. Unlike [`PROJECT_CONSTITUTION.md`](./PROJECT_CONSTITUTION.md) — which captures durable principles — these are *swappable* implementation choices. Change them via PR with a short rationale.

---

## Choices

| Concern | Choice | Why |
|---|---|---|
| Language / runtime | **Go 1.26** | Already scaffolded. Single static binary. |
| HTTP layer | **`net/http` (stdlib)** | Go 1.22+ routing is sufficient. No framework lock-in, no dependency to maintain. |
| Persistence | **None (stateless server)** | The auth + highlights features were retired on 2026-05-07 (see [`specs/archive/accounts.md`](./specs/archive/accounts.md), [`specs/archive/highlights.md`](./specs/archive/highlights.md)); per-user state went with them. The translation preference is the only remaining client-side preference and lives in `localStorage`. If a future feature reintroduces server-owned state, SQLite via `modernc.org/sqlite` is the prior choice and a defensible default. |
| Config | **Environment variables** (read at startup) | `ESV_API_KEY`, `YOUVERSION_APP_KEY`. No config files for v1. |
| Scripture providers | **ESV API** ([api.esv.org](https://api.esv.org/)) for ESV; **YouVersion Platform API** ([platform.youversion.com](https://platform.youversion.com/)) for NIV (`bible_id=111`) | Server-proxied through `internal/esv/` and `internal/youversion/` behind the `scripture.Provider` abstraction; upstream keys never reach the browser (`PROJECT_CONSTITUTION.md §4`). Adding a translation is a new package + a registry line — see [`specs/multi-translation.md`](./specs/multi-translation.md). |
| Deployment target | **Single static binary** | Implicit consequence of the above. With persistence retired, the binary is fully stateless — no DB volume, no Litestream backup. |
| Client framework | **React 19** (SPA in `web/`) | Familiar tooling; clean static-bundle target served by the Go binary. Decided in `specs/passage-reader.md`; bumped from 18 to 19 in `specs/reader-ui-refresh.md`. |
| Client bundler | **[Vite 8](https://vitejs.dev)** with `@vitejs/plugin-react` 6.x | Fast dev server, no server-only runtime APIs, clean static build output. Decided in `specs/passage-reader.md`; bumped from 5 to 8 in `specs/reader-ui-refresh.md`. |
| Client language | **TypeScript 6** | Strict mode on; `tsc -b` precedes `vite build` to enforce type-checking at build time. Bumped from 5.6 in `specs/reader-ui-refresh.md`. |
| Metrics | **`/metrics` endpoint, Prometheus-style exposition** | Real-time visibility into ESV/daily call volume without persistent state. Counter-only, hand-written exposition (no `prometheus/client_golang` dependency). |

## Explicitly NOT chosen

- **No web framework** (Gin, Echo, Fiber, chi). Stdlib is enough; revisit if routing becomes painful.
- **No database / ORM / migrations.** The server is stateless after the 2026-05-07 retirement of the auth + highlights features; reintroducing persistence is a deliberate decision that should be made in a spec, not slipped in by adding a dep.
- **No accounts / sessions / OAuth.** With no per-user features, identity has no purpose. Reintroduction starts from [`specs/archive/accounts.md`](./specs/archive/accounts.md) and [`specs/archive/oauth-auth.md`](./specs/archive/oauth-auth.md), both deprecated.

## When to revisit

| Trigger | Likely change |
|---|---|
| A new feature requires per-user state (sync across devices, cloud-stored highlights, etc.) | Reintroduce accounts (start from `specs/archive/accounts.md` or `specs/archive/oauth-auth.md`); pair with a SQLite (or Postgres if multi-host is in play) revival |
| Routing logic becomes hard to read in stdlib | Adopt `chi` (minimal, idiomatic) |
| Need background jobs (e.g. ESV cache warmers) | Add a worker goroutine pool; defer real queue (NATS / Asynq) until justified |

Each of these is a deliberate "later" — not an omission.
