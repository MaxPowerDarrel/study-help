# OAuth auth (Google)

**Status:** Draft
**Created:** 2026-05-06
**Last updated:** 2026-05-06
**Owner:** unassigned
**Supersedes:** [`accounts.md`](./accounts.md) (email + password is being retired)

## Why

The [`accounts.md`](./accounts.md) spec shipped email + password because v1
deliberately deferred third-party identity providers (PROJECT_CONSTITUTION.md §4,
[`STACK.md`](../STACK.md)). With deployment imminent ([`deploy-aws.md`](./deploy-aws.md))
the downstream cost of running passwords in production becomes concrete: password reset email,
breach-monitoring posture, hash-algorithm rotation, brute-force scaling, eventual
WebAuthn migration. Replacing the password path with Google OAuth eliminates all of
those line items in one move. The remaining auth surface — session cookies, server
store, per-IP rate limiting, the existing `sessions` table — stays exactly as it is.

This spec amends both the constitution (§4 must allow third-party IdPs as the
primary auth path) and `STACK.md` (which says OAuth should be a *parallel* path,
not a replacement). Those text edits ship with the implementation PR; the
rationale is logged in Decisions below.

## Goals

- [ ] A new visitor signs in with one Google click; account is created on first
      sign-in.
- [ ] An existing user (signed up via email + password before this change) signs
      in with Google using the same email and keeps every highlight, note, and
      preference on their account.
- [ ] The `users` row is keyed by Google's stable `sub` claim once linked, so a
      future Google email change doesn't break sign-in.
- [ ] Session cookies, server-side `sessions` store, sign-out flow, `/api/auth/me`
      hydration, and the per-IP rate limiter are unchanged from `accounts.md`.
- [ ] Email + password endpoints (`/signup`, `/signin`) and password storage are
      removed; the `password_hash` column is dropped; bcrypt is removed from the
      module graph.
- [ ] OAuth `client_secret` lives only in the server-side environment
      (constitution §4); the browser never sees it.

## Non-goals

- **Multiple OAuth providers** (GitHub, Apple, etc.). Single-provider keeps
  account-linking out of the picture; multi-provider is a separate, additive spec.
- **Account-linking UI.** First Google sign-in matches by lower-cased email; if
  no row matches, a new user is created. There's no "link a second login" flow.
- **Account deletion / data export.** Out of scope, as in `accounts.md`.
- **PKCE for a public client.** This is a confidential-client flow (the client
  secret is server-side); PKCE is added belt-and-suspenders but no public-client
  variant is provided.
- **"Sign in with Apple."** Requires an Apple Developer account ($99/yr) and a
  particular flow; defer until iOS shell exists.
- **Self-hosted IdP** (Keycloak, Authentik, etc.). The simplification we're
  buying is exactly *not* operating an identity service.

## User-facing behavior

**Sign-in panel.** The `AuthPanel` slide-in becomes a single button: "Continue
with Google". The email/password form, "Create account" toggle, and password
strength helper are gone. The sign-in nudge popover reads "Sign in with Google
to save highlights" (and "Sign in with Google to save notes" for the notes
nudge); the link inside it jumps straight to the OAuth start URL.

**Sign-in flow.**
1. User clicks **Continue with Google** → server redirects to Google with a
   freshly-minted `state` cookie (HttpOnly, SameSite=Lax) and a `nonce` claim.
2. Google authenticates the user and redirects back to
   `GET /api/auth/oauth/callback?code=...&state=...`.
3. Server validates `state` against the cookie, exchanges the code for tokens,
   verifies the ID token (Google's JWKS, audience = our client_id, nonce match,
   `email_verified=true`), then either finds-or-creates the `users` row and
   issues a session cookie. Final redirect: `/`.

**Migration of existing email-password users.** On first OAuth sign-in, the
server normalises Google's claimed email through the same
`strings.ToLower(strings.TrimSpace(email))` rule that `accounts.md` PR 1
established (2026-05-04), then looks for a `users` row with that already-
normalised email. If found, it stamps `oauth_provider=google` and
`oauth_sub=<sub>` onto that row, leaving `user_id` (and therefore every
highlight, note, and translation preference) intact. If no row matches, a
new row is created. This runs once per user; subsequent sign-ins key on
`(oauth_provider, oauth_sub)`.

**Sign-out.** Unchanged — `POST /api/auth/signout` deletes the local session
row and clears the cookie. We don't try to sign the user out of Google.

**Guest experience.** Unchanged — reading and the daily tab work fully without
an account; per-user actions (highlight, note) show the existing inline nudge.

**Errors.** OAuth-specific failure modes get inline messages on a new
`/api/auth/oauth/error` landing route: `state` mismatch, `email_verified=false`
on the ID token, Google denial, and "user closed Google before completing"
(no code in callback). One generic "couldn't complete sign-in, try again" for
everything else; details go to the server log.

## Implementation outline

### Server

**New package `internal/oauth/`** — encapsulates the OAuth dance:
- `provider.go` — `Provider` interface (`AuthURL(state, nonce)`, `Exchange(code)`,
  `VerifyIDToken(token, nonce)`); concrete `googleProvider` built from
  `coreos/go-oidc/v3` (preferred over Google-specific `idtoken` for forward-compat).
- `state.go` — random `state` + `nonce` generation; HttpOnly state cookie helpers.
- `handlers.go` — `oauthStart` and `oauthCallback`, mounted under
  `/api/auth/oauth/*` next to existing `/api/auth/*` routes.

**Removed from `internal/auth/`:**
- `password.go` + `password_test.go` (bcrypt + verify).
- `service.go` Signup / Signin code paths and their handlers in `handlers.go`.
- The per-account rate-limit bucket (per-IP stays — applied to
  `/api/auth/oauth/start`).
- `golang.org/x/crypto/bcrypt` from `go.mod`.

**Kept in `internal/auth/`:** users package CRUD (now with two new fields),
sessions package, cookie helpers, middleware, per-IP rate limiter.

### Endpoints

| Method | Path                              | Purpose                                  |
|--------|-----------------------------------|------------------------------------------|
| `GET`  | `/api/auth/oauth/start`           | Mint state cookie, redirect to Google.   |
| `GET`  | `/api/auth/oauth/callback`        | Validate, mint session, redirect to `/`. |
| `POST` | `/api/auth/signout`               | **Unchanged.**                           |
| `GET`  | `/api/auth/me`                    | **Unchanged.**                           |
| `PATCH`| `/api/auth/me`                    | **Unchanged.** (translation pref)        |

`POST /api/auth/signup` and `POST /api/auth/signin` are removed (404 to anyone
still calling them, including any cached SPA in the wild).

### Schema migration — `00006_oauth.sql`

Single migration; the server ships and the SPA ship in the same release
(server 404s `/signup`/`/signin` immediately, SPA no longer calls them).
A two-phase migration would only be worth the extra mechanism if there
were a meaningful in-flight password user base; today's reality is a
single-developer dataset so a one-shot rollout is the smaller risk.

```sql
-- +goose Up
ALTER TABLE users ADD COLUMN oauth_provider TEXT;
ALTER TABLE users ADD COLUMN oauth_sub TEXT;

CREATE UNIQUE INDEX idx_users_oauth_identity
  ON users (oauth_provider, oauth_sub)
  WHERE oauth_provider IS NOT NULL;

-- Drop the password column after backfilling oauth_provider/sub from a
-- one-shot script (see scripts/oauth_migrate.go) for any pre-existing
-- rows. The column is nullable in 00001_users.sql for SQLite < 3.35
-- compatibility, so DROP COLUMN works without a table rebuild.
ALTER TABLE users DROP COLUMN password_hash;

-- +goose Down
ALTER TABLE users ADD COLUMN password_hash TEXT;
DROP INDEX idx_users_oauth_identity;
ALTER TABLE users DROP COLUMN oauth_sub;
ALTER TABLE users DROP COLUMN oauth_provider;
```

**Pre-migration check.** `users.email` already carries a `UNIQUE` constraint
(see `00001_users.sql`), so the email-keyed migration step can rely on at
most one row per address. The migration runs an explicit
`SELECT count(*)…GROUP BY lower(email) HAVING count > 1` guard before
DROP COLUMN; non-zero rows abort the migration with a clear error.

**Rollback.** Goose `Down` re-adds `password_hash` as nullable and drops
the OAuth columns. Existing sessions still work after rollback (sessions
key on `user_id`); sign-*in* on the rolled-back binary fails because no
row has a `password_hash` to verify against. That's expected — the
rollback path exists for "the OAuth code is broken", not "we want to
re-enable passwords."

### Client

- `web/src/auth/AuthPanel.tsx` — replaces the form with a single
  "Continue with Google" button (an `<a href="/api/auth/oauth/start">`).
  The `signin`/`signup`/password-strength affordances are deleted.
- `web/src/auth/api.ts` — drop `signin` / `signup`; keep `signout` and `me`.
- `web/src/auth/useUser.ts` — drop `signin` / `signup` actions.
- `web/src/auth/AuthChip.tsx` — copy on the "Sign in" chip stays the same;
  click target is now the OAuth start URL via `AuthPanel`.

### Configuration

New env vars (added to `.env.example`, `internal/config/Config`):
- `GOOGLE_OAUTH_CLIENT_ID` — required.
- `GOOGLE_OAUTH_CLIENT_SECRET` — required.
- `OAUTH_REDIRECT_URL` — required, e.g. `https://study.example.com/api/auth/oauth/callback`
  in prod, `http://localhost:8080/api/auth/oauth/callback` in dev.

`SESSION_SECRET`, `ESV_API_KEY`, `YOUVERSION_APP_KEY` unchanged.

## Open questions

*None blocking implementation.* Three earlier questions were closed in the
2026-05-06 decisions log below; the remaining frictions (e.g. iPad-Safari
cookie behaviour through the OAuth redirect, what happens when the user
disconnects our app from their Google Account settings) surface during
implementation rather than spec.

## Decisions

- **2026-05-06** — **Replace email + password with Google OAuth, not add it
  alongside.** Reverses `accounts.md` (Decisions, 2026-05-04: "Use email +
  password for v1") and overrides `STACK.md`'s "Add OAuth as a *parallel*
  path, not a replacement" guidance. Rationale: deployment is imminent, the
  password path's prod-grade follow-on costs (reset email, breach monitoring,
  hash rotation) outweigh the simplicity of email+password v1, and a parallel
  path means twice the auth surface to test and reason about. This is a
  simplification, not an addition.
- **2026-05-06** — **Single provider (Google), not multi.** Rationale: avoids
  account-linking design (same-email collisions across providers); Google
  covers the audience an unverified app actually serves. Adding a second
  provider becomes its own spec.
- **2026-05-06** — **Identity is `(oauth_provider, oauth_sub)`, with a
  one-time email-based migration step.** `sub` is durable across email
  changes on Google's side; email matches the existing-user data set during
  cutover. Lower-case both sides of the email match for the migration.
- **2026-05-06** — **Authorization Code flow with confidential client (server
  exchanges the code), plus PKCE.** Rationale: keeps `client_secret` server-side
  (constitution §4); PKCE adds defence against authorization-code injection
  even though the client is confidential — it's free.
- **2026-05-06** — **Library: `github.com/coreos/go-oidc/v3`.** Battle-tested,
  treats Google as an ordinary OIDC provider, makes adding a second provider
  trivial. Considered: `google.golang.org/api/idtoken` — simpler, but
  Google-specific and would need to be replaced if multi-provider lands.
- **2026-05-06** — **ID-token claim verification list.** Beyond signature:
  `iss ∈ { "https://accounts.google.com", "accounts.google.com" }`,
  `aud == GOOGLE_OAUTH_CLIENT_ID`, `exp` not in the past (with the default
  `go-oidc` skew), `iat` not in the future, `nonce` matches the value bound
  to the state cookie, and `email_verified == true`. `go-oidc` enforces the
  signature, `iss`, `aud`, `exp`, and `iat` checks; the handler enforces
  the rest.
- **2026-05-06** — **Single-release cutover, not a two-step migration.**
  Server stops accepting `/signup` / `/signin` (404), SPA stops calling
  them, and `password_hash` is dropped in one migration (`00006_oauth.sql`)
  in the same release. Rationale: pre-prod user base is the developer
  alone, so the staged window a two-step migration buys has no value to
  trade for the extra mechanism. The migration includes a pre-DROP
  `count(*) GROUP BY lower(email)` guard against pre-existing duplicates.
- **2026-05-06** — **State + nonce in a single HttpOnly cookie.** Avoids a
  second cookie or a server-side nonce table; the cookie is single-use
  (cleared on first callback, success or failure).
- **2026-05-06** — **Per-account rate limiter is removed; per-IP stays.**
  Rationale: there's no password attempt to brute-force at our endpoint
  anymore; Google handles credential abuse. The per-IP limiter still guards
  the `/oauth/*` endpoints against state-cookie spam.
- **2026-05-06** — **Per-IP threshold on OAuth endpoints is ~60 requests
  / 15 min, applied uniformly to `/oauth/start` and `/oauth/callback`.**
  Higher than the password-era 10/15min because OAuth flows aren't
  credential-stuffing targets and shared-NAT users (office, library)
  legitimately retry. Single threshold across both endpoints keeps
  configuration small; tighten only if abuse appears.
- **2026-05-06** — **Same-email collision rejects with 409.** If a Google
  account presents `email_verified=true` for an email already linked to a
  different `oauth_sub`, the callback fails with `409 Conflict` and the
  message "This email is already linked to a different Google account."
  Rationale: keeps `users.email` unique post-migration (the migration
  guard relied on it; the runtime invariant matches); protects against
  upstream account-ownership changes silently transferring data. Trade:
  a user who legitimately switched Google accounts is locked out — a
  future "relink" flow can address this when it becomes a real complaint.
- **2026-05-06** — **Final redirect after callback is always `/`.** The
  SPA has no client-side routing today (URL is always `/`, book/chapter/
  tab are SPA state), so there are no deep links to preserve. A `next=`
  parameter and the open-redirect validation it implies will land in the
  same change that introduces SPA routing.
- **2026-05-06** — **Email-changed-on-Google case is unhandled at v1.**
  We key on `oauth_sub` after first link, so a Google email change is
  invisible to us. If a user re-signs-in to a different Google account that
  shares an email with their existing record, the `(provider, sub)` index
  prevents collision (different sub = different account), but they'd lose
  access to their old data. Acceptable given audience size; revisit if it
  becomes a real complaint.
- **2026-05-06** — **Trust Google's `email_verified=true` claim without
  additional verification.** Rationale: Google's bugs in this area have
  been isolated, not systemic; every mitigation (domain allowlist, our own
  verification email, requiring Workspace) is hostile to legitimate users
  for negligible safety gain — and an own-verification email pulls in a
  transactional-email dependency that `accounts.md` explicitly deferred.
  The residual risk (an upstream-faulty `email_verified=true` letting an
  attacker inherit an existing user's data) is documented as a known
  small risk.
- **2026-05-06** — **Email casing on the migration match inherits
  `accounts.md`'s normalisation.** `strings.ToLower(strings.TrimSpace(...))`
  on Google's claimed email, compared against the already-normalised
  `users.email` column (per `accounts.md` Decisions PR 1, 2026-05-04). No
  new normalisation code path; the migration step cites the existing
  contract directly.
- **2026-05-06** — **Guest sign-in nudge names the provider.** The
  popover reads "Sign in with Google to save highlights" / "Sign in with
  Google to save notes" instead of the prior "Sign in to save…" form.
  Rationale: removes the password-field expectation users carry over
  from the email + password era; matches the "Continue with Google"
  button label inside `AuthPanel` so users don't see two different
  framings of the same action.

## Verification

**Unit tests** (Go) with a fake OIDC provider built on `httptest`:
- Round-trip `oauthStart` → `oauthCallback` with valid state, nonce, ID token →
  session minted, cookie set.
- New-user case: no pre-existing row matching the verified email → fresh
  `users` row with `email`, `oauth_provider=google`, `oauth_sub`, and the
  default translation populated.
- Existing-user-by-email migration: pre-create a user with email
  `nate@example.com` (and a `password_hash`, in the world before 00006);
  first OAuth sign-in stamps `oauth_sub` and reuses the same `user_id`;
  existing highlights/notes survive.
- Same-email collision: pre-create an OAuth-linked user, then run the flow
  with a *different* `sub` for the same verified email → 409 (or the chosen
  decision per Open questions).
- State cookie missing on callback → 400.
- State cookie / query param mismatch → 400.
- State cookie single-use: first callback succeeds, second callback with the
  same cookie fails (cookie cleared on first use).
- ID token nonce mismatch → 400.
- ID token `email_verified=false` → 400.
- ID token `iss` not in the allow-list → 400.
- ID token `aud` not equal to `GOOGLE_OAUTH_CLIENT_ID` → 400.
- ID token signed by an unknown key → 400.
- ID token expired → 400.
- `POST /api/auth/signup` and `/signin` return 404.

**Manual smoke prerequisites**
1. Configure the Google OAuth client in test mode and **register both
   redirect URIs** (`http://localhost:8080/api/auth/oauth/callback` for dev,
   `https://study.example.com/api/auth/oauth/callback` for prod) in the
   Google console. The redirect URI must match exactly; this is an
   out-of-band step.

**Manual smoke**
1. Configure a Google OAuth client in test mode with both dev and prod
   redirect URIs.
2. Local dev (`ENV=dev`): click "Continue with Google", sign in, hit `/api/auth/me`
   — returns the new user with `email` and `translation` populated.
3. Sign out, reload, confirm guest state and the daily tab still works.
4. Sign in again, place a highlight, sign out, sign in again — highlight still
   there.
5. Production: same flow against the deployed `https://study.example.com/`.
6. Rollback drill: revert the deploy to the prior image (which still has the
   password endpoints); `/api/auth/me` works for active sessions because the
   sessions table is unchanged. Sign-in is broken on the rolled-back binary
   (no longer the goal), but data is intact.

## Constitution and STACK.md amendments

These edits ship with the implementation PR; the wording is pinned here so
the principle change can be reviewed before code lands.

**`PROJECT_CONSTITUTION.md` §4** — current text:
> Auth uses session cookies. HTTP-only secure cookies, server-side session
> store. No JWTs, no third-party identity providers at v1.

Replace with:
> Auth uses session cookies. HTTP-only secure cookies, server-side session
> store. No JWTs in cookies. Identity is established once per session via a
> third-party OIDC provider (Google at v1; the OIDC token is verified once
> and discarded — only the session cookie persists). The OAuth client
> secret stays server-side; the browser never sees it.

**`STACK.md`** — three rows change:
- *Sessions* row: replace "No JWTs, no third-party identity providers at
  v1" with "No JWTs in cookies. Identity established via OIDC at sign-in
  (Google at v1); session cookie persists, ID token is discarded."
- *Explicitly NOT chosen* row "No OAuth / social login at v1. Email +
  password only.": delete (this is the bullet being reversed).
- *When to revisit* row "Users ask for sign in with Google → Add OAuth as
  a *parallel* path": replace with "Users ask for a second IdP (GitHub,
  Apple) → write a multi-provider spec; account-linking is the load-bearing
  design choice."
- Add a new dependency row: `OIDC client | github.com/coreos/go-oidc/v3 |
  Google at v1; trivially extends to a second provider.`
- Remove `bcrypt` row.

## Related

- [`accounts.md`](./accounts.md) — superseded; password storage is being
  retired. Update its Status to **Deprecated** when this ships, and strike
  through its password-specific Verification rows (the email-flow rows
  remain valid as historical record).
- [`PROJECT_CONSTITUTION.md`](../PROJECT_CONSTITUTION.md) — §4 amended (text
  above).
- [`STACK.md`](../STACK.md) — three rows changed (text above).
- [`multi-translation.md`](./multi-translation.md) — `users.translation` is
  unchanged; `PATCH /api/auth/me` still works because the user record key
  (`user_id`) survives the migration. New OAuth-created users get the same
  default translation (`'ESV'`) the migration sets.
- [`highlights.md`](./highlights.md), [`notes.md`](./notes.md) — both
  reference `users.id` via a foreign key; nothing changes for them.
- [`deploy-aws.md`](./deploy-aws.md) — `.env` for the Lightsail deploy gains
  `GOOGLE_OAUTH_CLIENT_ID`, `GOOGLE_OAUTH_CLIENT_SECRET`, `OAUTH_REDIRECT_URL`.
