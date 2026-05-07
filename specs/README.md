# Specs

Living feature specs. Each entry below points to a markdown file in
this directory. Update both the spec and this index when status
changes.

Status values: **Draft** (being shaped), **In Progress** (code is
landing), **Shipped** (user-visible), **Deprecated** (being removed).

| Spec | Status | Summary |
|---|---|---|
| [accounts](./accounts.md) | Deprecated | Email + password sign up / sign in / sign out — removed 2026-05-07 alongside highlights; see spec for context |
| [auto-load-daily-reading](./auto-load-daily-reading.md) | Shipped | Auto-load a daily passage on startup; toggle in settings |
| [daily-annotations](./daily-annotations.md) | Deprecated | Highlights and per-chapter chunking on the Daily tab — reverted 2026-05-07; daily-header translation picker kept |
| [deploy-aws](./deploy-aws.md) | Draft | Single-VM AWS Lightsail deployment with Caddy TLS termination and Litestream → S3 backup |
| [docker](./docker.md) | Shipped | Package the app as a Docker image for reproducible, portable deployment |
| [highlights](./highlights.md) | Deprecated | Range-based, persistent, per-user passage highlights — removed 2026-05-07; see spec for context |
| [multi-plan](./multi-plan.md) | Shipped | Multiple daily reading plans (Bible-in-One-Year + Hope 2026); checkbox picker in settings |
| [multi-translation](./multi-translation.md) | Shipped | Pluggable translation foundation; ESV at launch, localStorage-persisted preference |
| [niv](./niv.md) | Shipped | NIV translation via YouVersion Platform API; "Powered by YouVersion" attribution |
| [notes](./notes.md) | Deprecated | Private per-user written notes attached to passages — removed 2026-05-07; see spec for context |
| [oauth-auth](./oauth-auth.md) | Deprecated | Drafted as a replacement for email + password auth; deprecated 2026-05-07 when the auth layer was retired entirely |
| [passage-reader](./passage-reader.md) | Shipped | Read a chapter or contiguous passage range; ESV proxied through the server |
| [reader-ui-refresh](./reader-ui-refresh.md) | Shipped | iPad/Safari touch polish, light + dark theme, design tokens; rides a Vite/React/TS bump |
