# Project Constitution — study-help

This document is the canonical answer to *"why is this project shaped this way?"* It is opinionated, deliberately narrow, and intended to be enforced. Amendments happen via PR that updates this file with a short rationale.

---

## 1. Purpose

`study-help` is a Bible reader optimized for **focused study** of scripture a chapter or section at a time, with personal highlights and notes. It is web-first, but architected so a native iPad shell can wrap the same client without rework.

The product exists to support careful, slow reading — not to be a search engine, a commentary library, or a social platform.

## 2. Users & Scope

- **Multi-user with accounts.** Each user has private highlights and notes. Server-side storage is the source of truth.
- **In scope:**
  - Reading a chapter or a contiguous passage range
  - Highlighting passages (range-based, persistent, per-user)
  - Notes attached to passages (per-user)
  - Account management (sign up, sign in, sign out)
- **ESV-only at v1.** Scripture content is fetched from the ESV API.

## 3. Core Principles

- **Study-first UX.** The reading surface is the product. Every other UI element is in service of it and must justify the space it takes.
- **Simplicity over breadth.** Fewer features, done well. Three good capabilities beat ten mediocre ones.
- **Respect the text.** Scripture is read-only. Users annotate; they never mutate.
- **Responsiveness over richness.** Page-turn and chapter-switch must feel instant. Anything that compromises that is deferred or rejected.

## 4. Architectural Guardrails

These exist primarily to preserve the *web-now, native-shell-later* portability requirement. Violating them costs us the iPad path.

- **Backend is a JSON API.** The application API surface returns data, not views — no server-rendered HTML for app content. This is the contract a future native client will share with the web client. Operational and observability endpoints (e.g. `/healthz`, `/metrics`) are not part of this surface and may use formats appropriate to their tooling (text exposition, plain JSON, etc.); they should be bound to a non-public listener whenever practical.
- **Frontend is decoupled.** The web client makes no assumptions about Node-only or server-only runtime APIs. It must be loadable as a static bundle inside a WebView.
- **Platform features behind an abstraction.** Anything platform-touching — local storage, notifications, biometrics, share sheets, file pickers — goes through a thin interface so a native shell can substitute its own implementation without touching feature code.
- **Secrets stay server-side.** The ESV API key, session secrets, and database credentials never reach the client. The server proxies scripture requests; the client never calls `api.esv.org` directly.
- **User data is server-authoritative.** Accounts, highlights, and notes are persisted server-side. Clients hold cache, not source of truth.
- **Auth works inside a WebView.** Session-based auth. Avoid flows that hard-require browser-only redirects without a fallback that works inside a wrapped client.

## 5. Non-Goals

A constitution without non-goals is a wishlist. The following are explicitly **out of scope**, and proposing them requires amending this document first.

- **No bundled commentary or study-note library.** This is a reader, not a library.
- **No social, community, or sharing features.** No comments, no public profiles, no shared highlights.
- **No original-language tooling.** No Greek/Hebrew interlinears, lexicons, or parsing aids.
- **No multi-translation switching at v1.** ESV only.
- **No offline-first sync engine at v1.** Offline is a best-effort cache, not a guarantee.
- **No parallel native iPad codebase.** The web client is the single client until the day we wrap it. We do not maintain a separate SwiftUI app.

## 6. Decision Rules

Before adding any feature or accepting any dependency, answer these:

1. **Does it serve focused study, or is it a distraction?** If it doesn't help someone read scripture more carefully, it doesn't belong.
2. **Does it preserve native-shell portability?** If it binds us to browser-only behavior with no escape hatch, no.
3. **Can it be removed in a single PR if it doesn't pan out?** If it tangles into the foundation, it's too risky for an unproven idea.
4. **Is this the simplest implementation that works?** If a senior engineer would call it overcomplicated, simplify before merging.

If any answer is "no," the default is to reject or redesign — not to negotiate exceptions.

## 7. Amendments

This document is checked into the repo at `PROJECT_CONSTITUTION.md`. Changes happen via pull request. The PR description must include:

- **What** changed (added/removed/modified principle, guardrail, or non-goal)
- **Why** — the concrete situation that exposed the gap or made the prior rule wrong

Amendments should be rare. The constitution is meant to outlive individual features.
