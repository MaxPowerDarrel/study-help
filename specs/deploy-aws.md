# Deploy on AWS

**Status:** Draft
**Created:** 2026-05-06
**Last updated:** 2026-05-06
**Owner:** unassigned

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
      `litestream.yml`, `.env.example`, runbook) live in
      `deploy/lightsail/` and are versioned alongside the app.
- [ ] Steady-state cost ≤ $15/mo for a personal-scale deployment.

## Non-goals

- **No HA / multi-instance.** SQLite pins the deployment to one writer.
  Revisit with a Postgres migration spec.
- **No Kubernetes / ECS / Fargate.** Single-VM Docker Compose is the
  smallest thing that works given the SQLite constraint.
- **No CI/CD wiring in this spec.** The spec assumes images get pushed
  to a registry; it does not prescribe how. A GitHub Actions workflow
  is a separate, additive change.
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
2. Push an image (`sha-<git>`) to the chosen registry.
3. SSH in, install Docker, copy `deploy/lightsail/*` to `/opt/study-help/`,
   fill in `.env` / `Caddyfile` / `litestream.yml`.
4. `docker compose up -d`; watch `docker compose logs -f caddy` for the
   first cert issue.
5. Open the FQDN in a browser. Sign up, sign in, fetch a passage,
   highlight a verse, leave a note — confirm `Secure; HttpOnly` cookie
   in devtools.
6. Disaster-recovery drill: `docker volume rm study-help_data` and
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
