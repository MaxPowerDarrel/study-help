# Deploy on AWS

**Status:** Shipped
**Created:** 2026-05-06
**Last updated:** 2026-05-10 (live at study.darrel.io)
**Owner:** unassigned

> **Editor's note (2026-05-07):** the design below assumes a SQLite-backed app and devotes most of its complexity to Litestream → S3 replication, a one-shot `restore` init container, and the IAM scaffolding around them. None of that is required anymore: the auth, highlights, and notes features that needed a database were retired on 2026-05-07 (see [accounts.md](./archive/accounts.md), [highlights.md](./archive/highlights.md), [notes.md](./archive/notes.md)), so the binary is fully stateless. The live `deploy/lightsail/` stack now runs **two services only — `app` + `caddy` — with no `restore`, `litestream`, S3 bucket, or AWS IAM key**. Caddy + DNS + a static IP remain the deployment shape; everything below tied to data persistence is preserved as the historical design record and would need to be reintroduced if a future feature brings back server-owned state.

## Why

The Docker spec ([`docker.md`](./docker.md)) explicitly scoped its
deployment target to "local machine"; production hosting was a follow-up.
This spec captures that follow-up: a single-VM AWS deployment that runs
the existing Docker image, terminates TLS, and survives a wiped disk.

The choice is constrained by the codebase's database posture: SQLite via
`modernc.org/sqlite`, embedded goose migrations, single-writer WAL. That
pins the deployment to a single instance — multi-AZ replication and
zero-downtime deploys are off the table without first migrating off
SQLite. We accept that for v1.

## Goals

- [ ] One-command deploy on a single Lightsail VM via `docker compose up -d`.
- [ ] HTTPS terminates at the host (Caddy), with auto-renewing
      Let's Encrypt cert; `ENV=prod` Secure cookies work end-to-end.
- [ ] SQLite replicated continuously to S3 via Litestream; on a fresh
      disk, the app rehydrates the DB from S3 automatically before
      starting.
- [ ] Operator-facing artifacts (`compose.yaml`, `Caddyfile`,
      `litestream.yml`, `.env.example`, `bootstrap.sh`, `deploy.sh`,
      runbook) live in `deploy/lightsail/` and are versioned alongside
      the app.
- [ ] CI builds and pushes the image to GHCR on every `main` push and
      on `v*` tags. A chained deploy job (auto on successful build via
      `workflow_run`, plus `workflow_dispatch` for manual overrides)
      SSHes into the VM, performs a transient `docker login` with the
      workflow's `GITHUB_TOKEN`, and runs `deploy.sh`, which atomically
      swaps the image, healthchecks `/healthz`, and rolls back on
      failure.
- [ ] Steady-state cost ≤ $15/mo for a personal-scale deployment.

## Non-goals

- **No HA / multi-instance.** SQLite pins the deployment to one writer.
  Revisit with a Postgres migration spec.
- **No Kubernetes / ECS / Fargate.** Single-VM Docker Compose is the
  smallest thing that works given the SQLite constraint.
- ~~**No auto-deploy on push to main.**~~ *(Reversed 2026-05-09 — see
  Decisions. Deploy now auto-fires on each successful build, gated only
  by an optional `production` environment approval.)*
- **No production observability stack.** The internal metrics port stays
  bound to `127.0.0.1:9090` inside the container (per `docker.md`); a
  later spec can add a scraper sidecar or Grafana Cloud agent.
- **No secrets management migration.** Secrets sit in a host-side
  `.env` (mode `0600`); SSM Parameter Store or Secrets Manager is a
  later, tighter posture.

## Topology

```
Route 53 (A record) ──► Lightsail static IP ──► VM
                                                 ├── caddy   :80/:443  ─► app:8080
                                                 ├── app     (SPA + API + SQLite)
                                                 ├── litestream (replicate sidecar) ─► S3
                                                 └── restore   (one-shot init)      ◄─ S3
```

Three running containers (app, caddy, litestream) plus one one-shot
init container (restore) that runs only when `/data/sqlite.db` is
missing. All four share a Docker volume holding the DB file and its
WAL companion.

## Decisions

- **2026-05-06** — **Lightsail VM, not EC2.** Predictable monthly cost,
  built-in static IP and snapshot tooling, no VPC ceremony. Trade: no
  IAM instance profile, so Litestream uses a scoped IAM access key.
- **2026-05-06** — **Caddy for TLS termination.** Auto-issuing
  Let's Encrypt cert with no extra config beats an ALB ($16/mo minimum)
  for a personal-scale deployment. Caddy v2 also handles HTTP/2 and
  compression with one directive.
- **2026-05-06** — **Litestream over scheduled snapshot.** Continuous
  WAL replication gives point-in-time restore at sub-minute granularity
  with effectively zero ops cost. Lightsail daily snapshots stay enabled
  as a coarse fallback (full-disk image, not just DB).
- **2026-05-06** — **Restore via one-shot init container, not
  app-side code.** Keeps the Go binary unaware of S3; restoration is
  the deployer's concern, not the app's. The init exits 0 when no
  replica exists, so a fresh install starts with an empty DB and
  migrations populate it.
- **2026-05-06** — **Pinned image tag (`sha-<git>`), not `:latest`.**
  Reproducible deploys, unambiguous rollbacks. Tag scheme is enforced
  by `.env`; the image registry choice (GHCR vs ECR) is operator
  preference.
- **2026-05-06** — **Production compose lives at `deploy/lightsail/`,
  separate from the repo-root `compose.yaml`.** The root file builds
  locally for dev (per `docker.md`); the deploy file pulls a registry
  image and adds the Caddy / Litestream / restore sidecars. Conflating
  them would either hide the dev `build:` directive in production or
  bake registry-only fields into the local-dev path.
- **2026-05-06** — **AWS credentials passed via environment, not via
  AWS profile / mounted credentials file.** Compose env-substitution
  is the simplest path; the credentials never leave the host's `.env`.
  Rotating means editing `.env` and `docker compose up -d`.
- **2026-05-06** — **Image build in CI, deploy operator-triggered.**
  Build runs on every push to `main` (and on `v*` tags); deploy is a
  separate `workflow_dispatch` job that SSHes into the VM and runs
  `deploy.sh`. Rationale: build cycles are cheap and the image is the
  audit trail of what could be deployed; rolling forward to production
  should be a deliberate act, not a side-effect of a merge.
  *(Superseded 2026-05-09 — deploy is now auto-chained from build.)*
- **2026-05-09** — **Deploy auto-chains from build.** `deploy.yml`
  fires via `workflow_run` on every successful `build-image` run on
  `main`, in addition to the existing `workflow_dispatch` path. The
  prior "deliberate act" argument doesn't carry weight now that the
  server is stateless (no DB to corrupt; rollback in `deploy.sh` is
  already atomic). Operators who want a click-through gate keep it by
  adding required reviewers to the `production` environment — the
  workflow already targets that environment.
- **2026-05-09** — **GHCR auth is transient, not persistent on the
  VM.** The deploy workflow forwards its own `GITHUB_TOKEN` (with
  `packages: read`) over SSH and runs a one-shot `docker login
  ghcr.io` inside the SSH session, then `docker logout`. Nothing
  long-lived lives on the host. Trade: ad-hoc `docker compose pull`
  outside of CI requires the operator to `docker login` first; cached
  images on the VM cover restarts and reboots without re-auth.
- **2026-05-09** — **Bootstrap clones its own siblings via transient
  PAT, no `scp`.** `bootstrap.sh` accepts a `GH_TOKEN` env var and
  shallow-clones the repo (`http.extraHeader` Basic auth, `mktemp -d`
  + `trap rm -rf`) to copy `deploy/lightsail/` artifacts into
  `/opt/study-help/`. The token never lands in `.git/config` or
  process args. Trade: operator must mint a fine-grained PAT scoped to
  `contents:read` on this repo. Rationale: `scp` required a laptop
  with the repo cloned; self-clone makes "ssh + curl + bash"
  sufficient from any fresh shell.
- **2026-05-10** — **Live at study.darrel.io.** Five small fixes
  surfaced during launch and were rolled in: (1) `git -c
  http.extraHeader=Authorization: Basic …` for the bootstrap clone
  (Bearer is unrecognized by GitHub's git smart-HTTP and falls back to
  the credential helper / interactive prompt), (2) drop `script_stop`
  from `appleboy/ssh-action@v1` and use `set -euo pipefail` + `trap …
  EXIT` inside the script block, (3) lowercase
  `${GITHUB_REPOSITORY}` when constructing the deploy image ref so it
  matches what `docker/metadata-action` publishes, (4) operator must
  set the `DEPLOY_HOST` / `DEPLOY_USER` / `DEPLOY_SSH_KEY` repo
  secrets before the first deploy fires, (5) Lightsail's
  instance-level firewall (Networking tab in the console) is separate
  from any OS-level firewall — both 80 and 443 must be opened
  explicitly there or the cert will issue but external HTTPS will
  time out.
- **2026-05-06** — **Healthcheck via the Caddy container, not the app
  container.** The app's distroless image has no shell or wget, so we
  can't curl `/healthz` from inside it. Caddy's alpine base has both,
  and both containers share the default bridge network — `docker
  compose exec caddy wget app:8080/healthz` is the smallest path.
  Trade: deploy.sh hard-codes a runtime dependency on the `caddy`
  service name. Acceptable since both ship together in `compose.yaml`.
- **2026-05-06** — **Rollback by re-writing `APP_IMAGE` and re-running
  compose, not by keeping a "previous" container around.** Compose
  doesn't have a first-class rollback verb; tracking `OLD_IMAGE`
  before the swap and rewriting `.env` on healthcheck failure is the
  smallest correct thing. Trade: the rollback path itself can fail
  (e.g. registry unreachable); deploy.sh logs and exits non-zero so
  CI surfaces it.

## Open questions

- ~~**Image registry: GHCR vs ECR.**~~ *Resolved 2026-05-10: GHCR.
  Auto-published by `build-image.yml`, pulled by the VM with a
  transient `GITHUB_TOKEN`.*
- ~~**Domain.**~~ *Resolved 2026-05-10: `study.darrel.io`.*
- ~~**Region.**~~ *Resolved 2026-05-10: operator's choice (Caddyfile
  / `.env` are region-agnostic).*
- **Backups for non-DB state.** Lightsail snapshots cover the whole
  disk including the Caddy `/data` directory (issued cert + ACME
  account key). On full disaster recovery, Caddy will re-issue rather
  than restore — acceptable, but worth flagging. Snapshots are
  operator-managed in the Lightsail console; not yet enabled by
  default.

## Verification

End-to-end smoke (from the deploy README):

1. **AWS setup.** Provision Lightsail VM, static IP, and DNS A record
   per `deploy/lightsail/README.md`. Open ports 80 and 443 in the
   Lightsail Networking tab; restrict 22 to the operator's IP. (No
   S3 / IAM user — server is stateless.)
2. **CI image.** Push to `main` (or trigger `build-image` manually).
   Confirm the image lands at
   `ghcr.io/<owner-lowercase>/study-help:sha-<short>` in the Actions
   tab and the Packages page.
3. **GitHub repo secrets.** Set `DEPLOY_HOST`, `DEPLOY_USER`, and
   `DEPLOY_SSH_KEY` (PEM, no passphrase) at the repo level. Pipe the
   key in via `gh secret set DEPLOY_SSH_KEY < key.pem` to preserve the
   PEM newlines.
4. **Bootstrap.** `ssh` to the VM, `curl` `bootstrap.sh` from
   `raw.githubusercontent.com` with a fine-grained PAT, run twice
   (first run installs Docker + git; re-login; second run with
   `GH_TOKEN=…` clones the artifacts into `/opt/study-help/`). Edit
   `.env` and `Caddyfile` for the chosen FQDN.
5. **First bring-up.** `docker login ghcr.io` once interactively, then
   `docker compose up -d`; watch `docker compose logs -f caddy` for
   the first Let's Encrypt issuance. Hit the FQDN in a browser and
   confirm a 200 plus a valid TLS chain.
6. **Auto-deploy chain.** Land a commit on `main`. Confirm
   `build-image` succeeds, then `deploy` auto-fires (or sits at
   "Waiting" if approval is configured), the SSH step performs a
   transient `docker login`, and `deploy.sh` reports a healthy
   `/healthz`.
7. **Manual override.** Trigger `deploy` via `workflow_dispatch` with
   an explicit tag; confirm the resolved ref is the lowercase
   `ghcr.io/<owner>/study-help:<tag>` form and the deploy succeeds.
8. **Failed-deploy drill.** Tag a known-bad image (e.g. one that
   exits on start), dispatch the workflow, confirm it exits non-zero
   AND `.env` has been reverted to the prior `APP_IMAGE`.
9. **External reachability.** From outside the VPC: `curl -I
   https://<fqdn>` returns 200; `letsdebug.net` reports green for the
   FQDN.

## Cost (steady state)

| Item                                       | Monthly  |
|--------------------------------------------|----------|
| Lightsail micro (1 vCPU, 1 GB)             | $5       |
| Lightsail daily snapshots (operator-opt-in)| ~$1      |
| DNS hosted zone                            | $0.50    |
| **Total**                                  | **~$6**  |

A `small_3_0` instance brings it to ~$11/mo with a lot more headroom.
ESV / YouVersion API calls are unchanged from local dev (free tier).
No S3 / IAM costs — server is stateless.

## Related

- [`docker.md`](./docker.md) — local-dev container packaging that this
  spec builds on; same image, same Dockerfile, additive compose.
- [`PROJECT_CONSTITUTION.md`](../PROJECT_CONSTITUTION.md) — §4 secrets
  stay server-side (env vars; never baked into the image; AWS keys
  scoped to one S3 bucket).
- [`multi-translation.md`](./multi-translation.md) — `YOUVERSION_APP_KEY`
  is required by the NIV provider and is plumbed in `.env` here.
