# Deploy on AWS

**Status:** Draft
**Created:** 2026-05-06
**Last updated:** 2026-05-09 (self-cloning bootstrap)
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
  sparse-checkout clones `deploy/lightsail/` (`http.extraHeader` +
  `--filter=blob:none --sparse`) to a `mktemp -d` working tree, copies
  artifacts into `/opt/study-help/`, and removes the working tree on
  exit. The token never lands in `.git/config` or process args. Trade:
  operator must mint a fine-grained PAT scoped to `contents:read` on
  this repo. Rationale: `scp` required a laptop with the repo cloned;
  self-clone makes "ssh + curl + bash" sufficient from any fresh shell,
  and the operator already has GitHub creds — adding one more place to
  use them is cheaper than a laptop-side prerequisite.
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

- **Image registry: GHCR vs ECR.** Both work. GHCR keeps everything
  GitHub-side (no extra AWS console); ECR keeps everything AWS-side.
  Decide before wiring CI. *Default in `.env.example` is GHCR.*
- **Domain.** The Caddyfile ships with `study.example.com` as a
  placeholder. Need a real FQDN with an A record before first deploy.
- **Region.** No latency-sensitive path; pick whichever region is
  closest to the operator. The S3 bucket region is independent.
- **Backups for non-DB state.** Lightsail snapshots cover the whole
  disk including the Caddy `/data` directory (issued cert + ACME
  account key). On full disaster recovery, Caddy will re-issue rather
  than restore — acceptable, but worth flagging.

## Verification

End-to-end smoke (from the deploy README):

1. Provision Lightsail VM, static IP, S3 bucket, IAM user, Route 53 A
   record per `deploy/lightsail/README.md`.
2. CI build on push to `main` produces a `ghcr.io/.../study-help:sha-<git>`
   image. Confirm in the Actions tab.
3. SSH to the VM. Run `git clone <repo> && ./deploy/lightsail/bootstrap.sh`
   (twice, with a re-login between to pick up the docker group). Fill
   in `.env` / `Caddyfile` / `litestream.yml`.
4. `docker compose up -d`; watch `docker compose logs -f caddy` for the
   first cert issue.
5. Open the FQDN in a browser. Sign up, sign in, fetch a passage,
   highlight a verse, leave a note — confirm `Secure; HttpOnly` cookie
   in devtools.
6. Set GitHub repo secrets (`DEPLOY_HOST`, `DEPLOY_USER`,
   `DEPLOY_SSH_KEY`). Run the **deploy** workflow with `image_tag=main`.
   Confirm the workflow exits zero and the running container picks up
   the new image (`docker compose ps` on the VM).
7. Failed-deploy drill: tag a known-bad image (e.g. one that exits on
   start), trigger the deploy workflow, confirm the workflow exits
   non-zero AND `.env` has been reverted to the prior `APP_IMAGE`.
8. Disaster-recovery drill: `docker volume rm study-help_data` and
   `docker compose up -d`; the `restore` container should pull the
   latest snapshot from S3 and the app should come back with the same
   data.

## Cost (steady state)

| Item                                      | Monthly |
|-------------------------------------------|---------|
| Lightsail micro (1 vCPU, 1 GB)            | $5      |
| Lightsail daily snapshots (7-day rotation)| ~$1     |
| Route 53 hosted zone                      | $0.50   |
| S3 backup (single-digit MB, versioned)    | <$0.10  |
| **Total**                                 | **~$7** |

A `small_3_0` instance brings it to ~$12/mo with a lot more headroom.
ESV / YouVersion API calls are unchanged from local dev (free tier).

## Related

- [`docker.md`](./docker.md) — local-dev container packaging that this
  spec builds on; same image, same Dockerfile, additive compose.
- [`PROJECT_CONSTITUTION.md`](../PROJECT_CONSTITUTION.md) — §4 secrets
  stay server-side (env vars; never baked into the image; AWS keys
  scoped to one S3 bucket).
- [`multi-translation.md`](./multi-translation.md) — `YOUVERSION_APP_KEY`
  is required by the NIV provider and is plumbed in `.env` here.
