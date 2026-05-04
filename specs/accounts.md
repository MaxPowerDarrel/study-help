# Accounts

**Status:** In Progress <!-- Draft | In Progress | Shipped | Deprecated -->
**Created:** 2026-05-04
**Last updated:** 2026-05-04 <!-- PR 1 (server) implementation 2026-05-04 -->
**Owner:** unassigned

## Why

Highlights and notes are per-user (PROJECT_CONSTITUTION.md §2), and user data is server-authoritative (§4). Before either can ship, the app needs accounts: a way for a person to register, sign in, and sign out so the server can attribute reads, highlights, and notes to a specific user. This spec covers the foundational identity layer; highlights and notes will sit on top of it.

## Goals

User-visible outcomes that define success. Bullets, not paragraphs.

- [ ] A new visitor can create an account with email + password and is signed in on success.
- [ ] An existing user can sign in with email + password from any device.
- [ ] A signed-in user can sign out, ending their session on the server.
- [ ] Sessions persist across page reloads and browser restarts on a 30-day sliding window (see Decisions).
- [ ] Reader functionality remains usable while signed out (study-first UX, §3); only per-user features require an account.

## Non-goals

Things explicitly excluded from this feature, with one-line rationale each.

- Third-party identity providers (Google, Apple, etc.) — *§4 says no third-party IdPs at v1.*
- Password reset via email — *out of scope until we have a transactional-email story; defer to its own spec.*
- Email verification — *same reason; tracks with password reset.*
- Two-factor authentication — *unjustified complexity at v1.*
- Account deletion / data export — *valuable, but its own spec; not required for highlights/notes to ship.*
- Profile management (display name, avatar, preferences UI) — *minimal account row only; preferences live in their own feature when needed.*
- Admin UI / role system — *single user role at v1.*
- CAPTCHA / proof-of-work on signup — *skipped at v1; per-IP rate limit covers volume attacks. Add reactively if abuse appears.*

## User-facing behavior

Concrete description of what the user sees and does. Not implementation.

- The auth affordance is a top-right header chip: a "Sign in" button when guest, an avatar/email + menu when signed in. It stays out of the reading surface (§3).
- **Sign up:** email + password form. Email is the login identifier (no separate username). Password requires a minimum of 12 characters; no other complexity rules. On success, user is signed in and lands back on the reader.
- **Sign in:** email + password form. On success, user lands back on the reader (or the page they were attempting to use).
- **Sign out:** one click; ends the current session only (other devices stay signed in) and returns the user to the reader as a guest. Idempotent — clicking sign-out from an already-expired or missing session still succeeds.
- **Guest experience:** reading works fully without an account. Attempting a per-user action (highlight, note) shows an inline popover anchored to the action — "Sign in to save highlights" with sign-in / sign-up links — preserving any in-progress selection.
- **Errors:** invalid credentials, duplicate email at signup, and weak passwords surface as inline messages — no modal walls.

## Implementation outline

High-level shape only.

- One new package `internal/auth/` covers user records, password hashing (bcrypt per STACK.md), session lifecycle, and the session-lookup middleware. No separate `internal/accounts/` package at v1.
- New endpoints under `/api/auth/`:
  - `POST /api/auth/signup` — returns the user object on success (no extra `/me` round-trip needed).
  - `POST /api/auth/signin` — returns the user object on success.
  - `POST /api/auth/signout` — idempotent. Always returns `204 No Content`. If a valid session cookie is present, the row is deleted and the cookie is cleared; if it's missing/expired/unknown, still 204.
  - `GET  /api/auth/me` — returns the current user, or 401. Used for cold-load hydration.
- HTTP-only session cookie set by the server. Cookie attributes:
  - Production: `Secure`, `HttpOnly`, `SameSite=Lax`.
  - Dev (HTTP localhost): `Secure` flag dropped so the cookie sets; controlled by an env-derived flag.
  - `Max-Age` mirrors the session row's sliding `expires_at` — set on signin and re-issued on each authenticated request that bumps the row, so cookie expiry tracks server-side expiry exactly.
- Session store backed by SQLite. Lifetime: 30-day sliding window — each authenticated request bumps `expires_at`.
- Rate limiting on the auth endpoints:
  - Per-IP token-bucket (~10 attempts / 15 min, keyed off `RemoteAddr`) on `POST /api/auth/signin` and `POST /api/auth/signup` — guards against volume attacks.
  - Per-account counter on `POST /api/auth/signin` (~5 failures / 15 min, keyed off the submitted email) — so a shared-NAT user can't be locked out by another user's failures from the same IP.
  - Trusted-proxy header (`X-Forwarded-For`) support is future work; documented as a known limitation when deployed behind a reverse proxy.
- New goose migrations:
  - `users` table — email is the unique identifier; password hash; timestamps.
  - `sessions` table — `id`, `user_id`, `token_hash` (sha256 of cookie value), `created_at`, `last_seen_at`, `expires_at`. Raw cookie value is never stored.
- Client surface: header chip + sign-in / sign-up forms in `web/`, plus a session-aware fetch wrapper that routes 401s into the inline-prompt flow.
- Session secret already plumbed via `SESSION_SECRET` env var (see `internal/config/`).

## Open questions

Questions that need to be resolved. When answered, move the resolution under **Decisions** and leave a pointer here.

- [x] ~~Email as the login identifier, or a separate username?~~ — resolved 2026-05-04, see Decisions.
- [x] ~~Minimum password policy (length, complexity, breach check via HIBP, none)?~~ — resolved 2026-05-04, see Decisions.
- [x] ~~Session lifetime (e.g. 30 days sliding, 7 days absolute, "remember me" checkbox)?~~ — resolved 2026-05-04, see Decisions.
- [x] ~~Where does the auth control live in the UI?~~ — resolved 2026-05-04, see Decisions.
- [x] ~~What does "attempting a per-user action while signed out" do?~~ — resolved 2026-05-04, see Decisions.
- [x] ~~Rate limiting on sign-in / sign-up — in scope here, or separate?~~ — resolved 2026-05-04, see Decisions.
- [x] ~~CAPTCHA or proof-of-work on signup?~~ — resolved 2026-05-04, see Decisions / Non-goals.
- [x] ~~Does the API return the user object on signin/signup, or always call /me?~~ — resolved 2026-05-04, see Decisions.
- [x] ~~`sessions` table — hashed token vs raw, and column set?~~ — resolved 2026-05-04, see Decisions.
- [x] ~~`internal/accounts/` and `internal/auth/` — merged or split?~~ — resolved 2026-05-04, see Decisions.
- [x] ~~Cookie `SameSite` / `Secure` / `HttpOnly` posture in dev vs production?~~ — resolved 2026-05-04, see Decisions.
- [x] ~~Does sign-out invalidate the current session row, or all sessions for that user?~~ — resolved 2026-05-04, see Decisions.
- [x] ~~How is the rate limiter keyed (RemoteAddr vs trusted proxy header), and is a per-account counter also needed?~~ — resolved 2026-05-04, see Decisions.
- [x] ~~Cookie `Max-Age` — pinned to 30-day, refreshed each request, or session cookie?~~ — resolved 2026-05-04, see Decisions.
- [x] ~~`POST /api/auth/signout` response when no/expired/unknown cookie?~~ — resolved 2026-05-04, see Decisions.
- [ ] Inline "Sign in" popover: render the form inline within the popover, or open a sign-in surface and lose the in-progress selection? Current Decision claims the selection is preserved — so inline form is implied but not stated.
- [ ] On successful signup, is the user auto-signed-in by issuing the session cookie in the same response, or does the client follow up with a signin call? (Goal says "signed in on success" — should be explicit in the endpoint contract.)
- [ ] Email normalization / uniqueness rule: lowercase before uniqueness check? Trim whitespace? Treat `+`-aliases as distinct? Affects the `users` unique index and the duplicate-email error path.
- [ ] Does `GET /api/auth/me` count as an "authenticated request" that bumps `last_seen_at` / `expires_at`, or is it read-only against the session row?
- [ ] Should the per-IP limiter also cover `GET /api/auth/me`, or only the two POSTs?
- [ ] What happens to a user's existing sessions when their password is changed in a future feature? (Record intent now even though password change is out of scope here.)
- [ ] Cookie `Path` and `Domain` attributes — default `/` and host-only is probably fine, but worth pinning explicitly.
- [ ] Session ID generation: how many bytes of entropy from `crypto/rand`, and what encoding for the cookie value (base64url, hex)?
- [ ] Are sign-in errors deliberately ambiguous ("invalid email or password") to avoid user enumeration? Note that signup's "duplicate email" message is itself an enumeration vector — was that tradeoff considered?
- [ ] iPad-Safari verification should also confirm `SameSite=Lax` survives ITP / cross-site nuances even though the app is same-origin.

## Decisions

Append-only log. Most recent at the bottom. Never rewrite past entries; if a decision is reversed, add a new entry that supersedes it.

- 2026-05-04: Use email + password for v1 (no third-party IdPs, per constitution §4). Bcrypt for password hashing (per STACK.md). Session cookies, server-side session store in SQLite (per §4).
- 2026-05-04: Email is the login identifier; no separate username field. Reason: simplest v1, one uniqueness namespace, matches the spirit of "fewer features, done well" (§3).
- 2026-05-04: Password policy is minimum length only — 12+ characters. No complexity rules, no rotation, no HIBP check at v1. Reason: NIST-aligned guidance favors length over complexity; HIBP adds a network dependency we can layer in later if abuse appears.
- 2026-05-04: Session lifetime is a 30-day sliding window — each authenticated request bumps `expires_at`. No "remember me" toggle. Reason: feels persistent to daily users without being immortal; one model is simpler than two.
- 2026-05-04: Auth control is a top-right header chip ("Sign in" when guest; avatar/menu when signed in). Reason: discoverable for new visitors, signed-in state is visible at a glance, and it stays out of the reading surface (§3).
- 2026-05-04: Guests who attempt a per-user action see an inline popover anchored to the action ("Sign in to save highlights"). Reason: preserves the in-progress selection and reading flow; modal/redirect both break context.
- 2026-05-04: `sessions` table stores a sha256 hash of the cookie value, never the raw token. Columns: `id`, `user_id`, `token_hash`, `created_at`, `last_seen_at`, `expires_at`. Reason: a DB leak doesn't yield live sessions; `last_seen_at` keeps a future "active devices" view possible without another migration.
- 2026-05-04: Cookie attributes are `Secure`, `HttpOnly`, `SameSite=Lax` in production; `Secure` is dropped in dev (HTTP localhost) via an env-derived flag. Reason: `Lax` is the right default for a same-origin SPA with no cross-site auth flows; dropping `Secure` in dev avoids HTTPS setup friction without weakening prod.
- 2026-05-04: Per-IP token-bucket rate limiter on `POST /api/auth/signin` and `POST /api/auth/signup` (~10 attempts / 15 min) ships with this spec, not as a separate cross-cutting middleware. Reason: cheap, narrowly scoped to the credential-stuffing risk, and avoids leaving auth exposed while a generic limiter is being designed.
- 2026-05-04: No CAPTCHA or proof-of-work on signup at v1 (added to Non-goals). Reason: niche app, low expected bot pressure; per-IP limit covers volume; revisit reactively if abuse appears.
- 2026-05-04: `POST /api/auth/signin` and `POST /api/auth/signup` return the user object directly. `GET /api/auth/me` remains for cold-load hydration. Reason: saves a round-trip on the most common transition without removing the canonical hydration path.
- 2026-05-04: Single package `internal/auth/` covers user records, password hashing, sessions, and middleware — no separate `internal/accounts/`. Reason: matches the actual scope; avoids a thin wrapper package that just forwards SQL.
- 2026-05-04: Sign-out invalidates only the current session row; other devices stay signed in. Reason: matches user expectation; a future "sign out everywhere" affordance can be carved out cleanly when needed.
- 2026-05-04: Rate limiter is keyed off `RemoteAddr` per-IP for both auth POSTs (~10 / 15 min), with an additional per-account counter on signin (~5 failures / 15 min, keyed off submitted email). Trusted-proxy header (`X-Forwarded-For`) is future work. Reason: per-IP alone over-locks shared NAT; per-account counter prevents one user's failures from locking another sharing the same egress IP.
- 2026-05-04: Cookie `Max-Age` mirrors the session row's sliding `expires_at` — set on signin and re-issued on each authenticated request that bumps the row. Reason: cookie expiry tracks server-side expiry exactly; avoids the subtle mismatch where the row is fresh but the cookie has aged out (or vice versa).
- 2026-05-04: `POST /api/auth/signout` is idempotent and always returns `204 No Content`. Valid session: row deleted, cookie cleared, 204. Missing/expired/unknown: still 204. Reason: simplest client contract; safe to call defensively from the session-aware fetch wrapper without special-casing the "already signed out" state.
- 2026-05-04 (PR 1): Email normalization is `strings.ToLower(strings.TrimSpace(...))` on input; stored normalized; `+`-aliases treated as distinct. Reason: simplest correct rule; no surprises for capitalized input or trailing whitespace.
- 2026-05-04 (PR 1): `GET /api/auth/me` is treated as an authenticated request — the middleware bumps `last_seen_at`/`expires_at` and re-issues `Set-Cookie` on it. Reason: it's the cold-load hydration call; treating it as a heartbeat keeps active users signed in without extra requests.
- 2026-05-04 (PR 1): Per-IP limiter covers only the two POSTs (`signup`, `signin`); `GET /api/auth/me` is excluded. Reason: `/me` is idempotent and low-cost; rate-limiting it would block legitimate cold-loads.
- 2026-05-04 (PR 1): Session ID is 32 random bytes from `crypto/rand`, encoded `base64.RawURLEncoding` for the cookie value. Stored as `sha256(rawBytes)` (32-byte BLOB). Reason: hashing the *bytes* (not the encoded string) keeps storage independent of any future encoding change; 256 bits of entropy is well above guess-resistance needs.
- 2026-05-04 (PR 1): Cookie attributes (concrete): `Name=sh_session`, `Path=/`, no `Domain` (host-only), `HttpOnly`, `SameSite=Lax`, `Secure=!IsDev`, `Expires`/`Max-Age` mirror the session row's sliding `expires_at`. Reason: host-only is correct for a same-origin SPA; Lax is the safe default.
- 2026-05-04 (PR 1): Signup auto-signs the user in — the response carries Set-Cookie + user JSON in one round-trip (`201 Created`). Reason: matches the "signed in on success" goal without an extra signin call.
- 2026-05-04 (PR 1): Signin error is deliberately ambiguous — `401 Unauthorized` with body `"invalid email or password"` regardless of whether the email exists. Signup duplicate-email remains explicit (`409 Conflict` with `"email already in use"`) — a known enumeration vector accepted as a UX tradeoff at v1.
- 2026-05-04 (PR 1): `SESSION_SECRET` is *not* used by this PR. The cookie value is random and validated server-side via sha256 lookup — no HMAC needed. The env var stays plumbed for future signed-cookie / CSRF use.
- 2026-05-04 (PR 1): Auth middleware is scoped to `/api/*` only (not the SPA). Reason: avoids per-asset session lookups on SPA cold loads; static assets don't care about identity. The middleware skips its cookie-refresh pass on the auth POSTs to avoid double `Set-Cookie` headers.
- 2026-05-04 (PR 1): Single package `internal/auth/` houses user records, password hashing, sessions, rate limiter, middleware, and handlers — no separate `internal/accounts/`. The package is small (~10 files) and the boundary between "identity records" and "credentials/sessions" doesn't carry its weight at v1.
- 2026-05-04 (PR 1): Existing-sessions on password change — out of scope here; intent recorded for the future password-change feature: it deletes all sessions for the affected user.

## Verification

How we'll know it's working.

- [ ] Unit tests: password hashing/verification round-trip; session create/lookup/expire.
- [ ] Integration tests against a real SQLite DB (no mocks): signup, signin, signout, `/me` for guest vs signed-in.
- [ ] Manual flow: sign up → sign out → sign in → reload → still signed in → sign out → reload → signed out.
- [ ] Manual flow on iPad Safari: cookie persists across browser restart.
- [ ] Negative cases: wrong password, duplicate email signup, expired session — all surface user-friendly messages.
- [ ] Rate limiter test: 11th sign-in attempt from the same IP within 15 min is rejected.
- [ ] Session-row test: `token_hash` in DB is not equal to the cookie value; auth still succeeds via hash comparison.
- [ ] Sign-out test: signing out on session A does not invalidate session B for the same user.
- [ ] Idempotent signout test: calling `POST /api/auth/signout` with no cookie returns 204; with an expired/unknown cookie returns 204.
- [ ] Per-account rate-limit test: 6 failed sign-ins for the same email from different IPs trip the per-account limiter on the 6th.
- [ ] Cookie max-age test: each authenticated request that bumps `expires_at` also re-issues `Set-Cookie` with the new `Max-Age`.

## Related

- Constitution sections: `PROJECT_CONSTITUTION.md §2` (Users & Scope — accounts), `§4` (session cookies, server-side store, no third-party IdPs), `§3` (study-first UX — auth must not crowd the reader).
- Stack: `STACK.md` (bcrypt, cookie sessions, SQLite session store).
- Downstream specs (not yet drafted): `highlights.md`, `notes.md` — both depend on this.
