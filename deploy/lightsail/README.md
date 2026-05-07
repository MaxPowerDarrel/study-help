# Lightsail deploy

Single-VM AWS deployment for `study-help`: Caddy fronts the app on `:443`
and reverse-proxies to the stateless Go binary.

See [`specs/deploy-aws.md`](../../specs/deploy-aws.md) for the rationale
and decisions; this README is the operator runbook.

## What gets deployed

```
┌───────────── Lightsail VM ──────────────┐
│                                         │
│   caddy :80/:443  ──►  app :8080        │
│                                         │
└─────────────────────────────────────────┘
        ▲
        │ HTTPS
   Route 53  ──  A record  ──►  Lightsail static IP
```

The server has no database — see `STACK.md` for context. Auth, highlights,
and notes were retired on 2026-05-07; the binary is fully stateless, so
this stack carries no per-app volumes (only Caddy's cert / state caches).

## One-time AWS setup

1. **Lightsail instance** — Ubuntu 22.04 LTS, `micro_3_0` ($5/mo, 1 GB) or
   `small_3_0` ($10/mo, 2 GB) for headroom. Pick a region near you.
2. **Static IP** — attach to the instance (free while attached).
3. **Networking** — open ports 80 and 443. Restrict 22 to your IP.
4. **Route 53** — hosted zone for your domain, A record at the static IP.

## Image distribution

Build the image in CI (or locally), push to a registry, pin the tag in
`.env`. Two reasonable registries:

- **GHCR** — `ghcr.io/<you>/study-help:sha-<git>`. Free, lives next to the
  source. Authenticate with a GitHub personal access token; on the host
  run `docker login ghcr.io` once.
- **ECR private** — `<account>.dkr.ecr.<region>.amazonaws.com/study-help:sha-<git>`.
  Roughly free at this scale. Authenticate with `aws ecr get-login-password`.

Avoid building on the Lightsail VM itself — the multi-stage Dockerfile pulls
Node and Go and the build is heavy on a 1 GB instance.

## First deploy

On the Lightsail VM (one-time host setup):

```bash
git clone https://github.com/<you>/study-help.git
cd study-help
./deploy/lightsail/bootstrap.sh
```

`bootstrap.sh` is idempotent. On first run it installs Docker and the
Compose plugin, then asks you to log out / log back in for the docker
group to take effect. Re-run after re-login and it scaffolds
`/opt/study-help/` with `compose.yaml`, `Caddyfile`, `deploy.sh`, and a
starter `.env` (mode `0600`).

Fill in the placeholders, then bring the stack up:

```bash
cd /opt/study-help
$EDITOR .env                            # fill in every CHANGEME
$EDITOR Caddyfile                       # replace study.example.com

docker login ghcr.io                    # if pulling a private image

# DNS must resolve to the static IP before this step or Caddy can't get
# a cert. Verify with `dig +short study.example.com`.
docker compose pull
docker compose up -d
docker compose logs -f caddy            # watch the first cert issue
```

The app should be reachable at `https://study.example.com/`.

## Day-2 ops

### Deploy a new version (CI-driven)

CI builds and pushes the image to GHCR on every push to `main` and on
`v*` tags (see `.github/workflows/build-image.yml`). To roll a new
version onto the VM:

1. Open the GitHub Actions UI → **deploy** workflow → **Run workflow**.
2. Enter the image tag (e.g. `sha-abc1234`, `v1.2.3`, or `main`).
3. The workflow SSHes into the VM and runs `/opt/study-help/deploy.sh`,
   which atomically updates `APP_IMAGE` in `.env`, pulls the new image,
   restarts the `app` container, polls `/healthz`, and rolls back to
   the previous image if the healthcheck fails.

Repository secrets required by the deploy workflow:

| Secret           | Value                                                  |
|------------------|--------------------------------------------------------|
| `DEPLOY_HOST`    | Lightsail static IP or FQDN                            |
| `DEPLOY_USER`    | SSH user on the VM (e.g. `ubuntu`)                     |
| `DEPLOY_SSH_KEY` | PEM-encoded private key (no passphrase)                |

For an extra approval gate, configure a `production` environment in
the repo settings with required reviewers.

### Deploy by hand

`deploy.sh` works the same way from an SSH session:

```bash
ssh ubuntu@<host>
/opt/study-help/deploy.sh ghcr.io/<you>/study-help:sha-abc1234
```

### Logs and metrics

```bash
docker compose logs -f app           # application logs
docker compose exec app wget -qO- 127.0.0.1:9090/metrics
```

The metrics endpoint is bound to `127.0.0.1:9090` inside the container by
design — only reachable via `docker exec`. For a real dashboard, point a
Grafana Cloud free-tier agent at it.

## Files

| File              | Purpose                                                                     |
|-------------------|-----------------------------------------------------------------------------|
| `compose.yaml`    | Service definitions (app + caddy).                                          |
| `Caddyfile`       | Reverse proxy + automatic Let's Encrypt cert.                               |
| `.env.example`    | Template for `.env` (image tag, API keys).                                  |
| `bootstrap.sh`    | One-shot host provisioning: installs Docker, scaffolds `/opt/study-help/`. |
| `deploy.sh`       | VM-side deploy: atomic `.env` bump, pull, restart, healthcheck, rollback.   |

The CI workflows in `../../.github/workflows/` (`build-image.yml`,
`deploy.yml`) drive image build/push and remote deploy.
