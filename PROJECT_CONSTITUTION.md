# Project Constitution — study-help

This document is the canonical answer to *"why is this project shaped this way?"* It is opinionated, deliberately narrow, and intended to be enforced. Amendments happen via PR that updates this file with a short rationale.

---

## 1. Purpose

`study-help` is a Bible reader optimized for **focused study** of scripture a chapter or section at a time. It is a web application, served as a static SPA from a Go binary. Safari on iPad is a first-class target, but it is reached as a regular browser — there is no native shell.

The product exists to support careful, slow reading — not to be a search engine, a commentary library, or a social platform.

## 2. Users & Scope

- **Single-user, no accounts.** Originally multi-user with per-account highlights; the auth and highlights features were retired on 2026-05-07 (see [`specs/accounts.md`](./specs/accounts.md), [`specs/highlights.md`](./specs/highlights.md)). The server holds no user state. Reintroducing per-user data requires re-amending this section.
- **In scope:**
  - Reading a chapter or a contiguous passage range
  - Daily reading plans (Bible-in-One-Year, Hope 2026)
- **Multi-translation, ESV at launch.** Scripture is fetched through a server-side `Provider` abstraction. The first registered provider is ESV; additional translations can be added without schema changes. The active translation persists in `localStorage` on the client.

## 3. Core Principles

- **Study-first UX.** The reading surface is the product. Every other UI element is in service of it and must justify the space it takes.
- **Simplicity over breadth.** Fewer features, done well. Three good capabilities beat ten mediocre ones.
- **Respect the text.** Scripture is read-only. Users annotate; they never mutate.
- **Responsiveness over richness.** Page-turn and chapter-switch must feel instant. Anything that compromises that is deferred or rejected.

## 4. Architectural Guardrails

These keep the codebase clean, testable, and decoupled. They reflect how a small, focused webapp earns its longevity: a clear API contract, a frontend that doesn't sprawl into the backend, and secrets that never leak to the client.

- **Backend is a JSON API.** The application API surface returns data, not views — no server-rendered HTML for app content. Operational and observability endpoints (e.g. `/healthz`, `/metrics`) are not part of this surface and may use formats appropriate to their tooling (text exposition, plain JSON, etc.); they should be bound to a non-public listener whenever practical.
- **Frontend is decoupled.** The web client makes no assumptions about Node-only or server-only runtime APIs. It builds to a static bundle that the Go binary serves.
- **Platform features behind an abstraction.** Anything browser-API-touching — local storage, notifications, share sheets, file pickers — goes through a thin interface so feature code stays portable and testable without mocking globals.
- **Secrets stay server-side.** Translation API keys never reach the client. The server proxies scripture requests; the client never calls upstream translation APIs (e.g. `api.esv.org`) directly.
- **The server is stateless.** Since the 2026-05-07 retirement of accounts and highlights, the server holds no user state and no database. The only client-side preference (translation choice) lives in `localStorage`. Reintroducing server-side state should be a deliberate spec-level decision, not a side effect of a feature.

## 5. Non-Goals

A constitution without non-goals is a wishlist. The following are explicitly **out of scope**, and proposing them requires amending this document first.

- **No bundled commentary or study-note library.** This is a reader, not a library.
- **No social, community, or sharing features.** No comments, no public profiles, no shared highlights.
- **No original-language tooling.** No Greek/Hebrew interlinears, lexicons, or parsing aids.
- **No personal annotations.** Highlights and notes were retired on 2026-05-07 (see [`specs/highlights.md`](./specs/highlights.md), [`specs/notes.md`](./specs/notes.md)). Free-form per-user prose attached to scripture is sensitive data we chose not to retain; range-based highlights were retired alongside the rest of the auth layer when no other per-user feature remained. Reintroducing either requires re-amending this section.
- **No offline-first sync engine at v1.** Offline is a best-effort cache, not a guarantee.

## 6. Decision Rules

Before adding any feature or accepting any dependency, answer these:

1. **Does it serve focused study, or is it a distraction?** If it doesn't help someone read scripture more carefully, it doesn't belong.
2. **Can it be removed in a single PR if it doesn't pan out?** If it tangles into the foundation, it's too risky for an unproven idea.
3. **Is this the simplest implementation that works?** If a senior engineer would call it overcomplicated, simplify before merging.

If any answer is "no," the default is to reject or redesign — not to negotiate exceptions.

## 7. Amendments

This document is checked into the repo at `PROJECT_CONSTITUTION.md`. Changes happen via pull request. The PR description must include:

- **What** changed (added/removed/modified principle, guardrail, or non-goal)
- **Why** — the concrete situation that exposed the gap or made the prior rule wrong

Amendments should be rare. The constitution is meant to outlive individual features.
