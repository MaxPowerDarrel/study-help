# Specs

Living feature specs. Each entry below points to a markdown file in
this directory. Update both the spec and this index when status
changes.

Status values: **Draft** (being shaped), **In Progress** (code is
landing), **Shipped** (user-visible), **Deprecated** (being removed).

Deprecated specs live under [`./archive/`](./archive/) as frozen
historical records — each one carries an editor's note explaining when
and why it was retired.

## Live

| Spec | Status | Summary |
|---|---|---|
| [auto-load-daily-reading](./auto-load-daily-reading.md) | Shipped | Auto-load a daily passage on startup; toggle in settings |
| [deploy-aws](./deploy-aws.md) | Shipped | Single-VM AWS Lightsail deployment with Caddy TLS termination, auto-deploy chained from `build-image`, transient GHCR + bootstrap auth; live at study.example.com |
| [docker](./docker.md) | Shipped | Package the app as a Docker image for reproducible, portable deployment |
| [multi-plan](./multi-plan.md) | Shipped | Multiple daily reading plans (Bible-in-One-Year + Hope 2026); checkbox picker in settings |
| [multi-translation](./multi-translation.md) | Shipped | Pluggable translation foundation; ESV at launch, localStorage-persisted preference |
| [niv](./niv.md) | Shipped | NIV translation via YouVersion Platform API; "Powered by YouVersion" attribution |
| [passage-reader](./passage-reader.md) | Shipped | Read a chapter or contiguous passage range; ESV proxied through the server |
| [pwa-install](./pwa-install.md) | Shipped | Install-only PWA: Web App Manifest + Apple meta tags so the app can be added to the iPad/Android home screen and launched standalone; no service worker |
| [reader-ui-refresh](./reader-ui-refresh.md) | Shipped | iPad/Safari touch polish, light + dark theme, design tokens; rides a Vite/React/TS bump |
| [restore-last-location](./restore-last-location.md) | Shipped | Reopen the app to the last-viewed tab + read passage + daily date (per device, localStorage) |

## Archive

| Spec | Status | Summary |
|---|---|---|
| [accounts](./archive/accounts.md) | Deprecated | Email + password sign up / sign in / sign out — removed 2026-05-07 alongside highlights; see spec for context |
| [daily-annotations](./archive/daily-annotations.md) | Deprecated | Highlights and per-chapter chunking on the Daily tab — reverted 2026-05-07; daily-header translation picker kept |
| [deploy-home](./archive/deploy-home.md) | Deprecated | Drafted as a self-host home variant via Duck DNS + Caddy/Let's Encrypt; deprecated 2026-05-08 due to DuckDNS NS-pool unreliability and a `caddy-dns/duckdns` plugin bug; see spec for context |
| [highlights](./archive/highlights.md) | Deprecated | Range-based, persistent, per-user passage highlights — removed 2026-05-07; see spec for context |
| [notes](./archive/notes.md) | Deprecated | Private per-user written notes attached to passages — removed 2026-05-07; see spec for context |
| [oauth-auth](./archive/oauth-auth.md) | Deprecated | Drafted as a replacement for email + password auth; deprecated 2026-05-07 when the auth layer was retired entirely |
