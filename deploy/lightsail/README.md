# Lightsail deploy

Single-VM AWS deployment for `study-help`: Caddy fronts the app on `:443`,
Litestream replicates the SQLite database to S3, and a one-shot restore
service rehydrates the DB from S3 if the disk is empty.

See [`specs/deploy-aws.md`](../../specs/deploy-aws.md) for the rationale
and decisions; this README is the operator runbook.

## What gets deployed

```
┌─────────────────────────── Lightsail VM ───────────────────────────┐
│                                                                    │
│   caddy :80/:443  ──►  app :8080  ──►  /data/sqlite.db (volume)    │
│                                            ▲                       │
│                                            │ shared volume         │
│                                       litestream  ──►  S3 (replica)│
│                                       restore   ◄──   (one-shot)   │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
        ▲
        │ HTTPS
   Route 53  ──  A record  ──►  Lightsail static IP
```

## One-time AWS setup

1. **Lightsail instance** — Ubuntu 22.04 LTS, `micro_3_0` ($5/mo, 1 GB) or
   `small_3_0` ($10/mo, 2 GB) for headroom. Pick a region near you.
2. **Static IP** — attach to the instance (free while attached).
3. **Networking** — open ports 80 and 443. Restrict 22 to your IP.
4. **Automatic snapshots** — enable daily, 7-day retention (~$1/mo). Belt
   and suspenders against the Litestream replica.
5. **S3 bucket** — `study-help-litestream-<random>`, block all public
   access, enable versioning. Same region as the VM is fine but not
   required (replication latency is irrelevant).
6. **IAM user** for Litestream — least-privilege policy:
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [{
       "Effect": "Allow",
       "Action": [
         "s3:GetObject",
         "s3:PutObject",
         "s3:DeleteObject",
         "s3:ListBucket"
       ],
       "Resource": [
         "arn:aws:s3:::study-help-litestream-XXXXXX",
         "arn:aws:s3:::study-help-litestream-XXXXXX/*"
       ]
     }]
   }
   ```
   Issue an access key for this user. Lightsail VMs do not get IAM instance
   profiles, so a scoped access key is the path.
7. **Route 53** — hosted zone for your domain, A record at the static IP.

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
sudo apt-get update
sudo apt-get install -y docker.io docker-compose-plugin
sudo systemctl enable --now docker
sudo usermod -aG docker ubuntu          # log out and back in
```

Then:
```bash
sudo mkdir -p /opt/study-help
sudo chown ubuntu:ubuntu /opt/study-help
cd /opt/study-help

# Copy compose.yaml, Caddyfile, litestream.yml, .env.example from this
# directory to /opt/study-help on the VM, then:
cp .env.example .env
chmod 600 .env
$EDITOR .env                            # fill in every CHANGEME
$EDITOR Caddyfile                       # replace study.example.com
$EDITOR litestream.yml                  # replace bucket / region

# DNS must resolve to the static IP before this step or Caddy can't get
# a cert. Verify with `dig +short study.example.com`.
docker compose pull
docker compose up -d
docker compose logs -f caddy            # watch the first cert issue
```

The app should be reachable at `https://study.example.com/`. Sign in,
and confirm `Secure` cookies are set (browser devtools → Application →
Cookies). The `ENV=prod` setting requires HTTPS; if the cookie isn't
`Secure`, sign-in will silently fail.

## Day-2 ops

### Deploy a new version

Bump `APP_IMAGE` in `.env` to the new tag, then:
```bash
docker compose pull && docker compose up -d
```
~1 second of downtime as the app container is replaced. Caddy and the
Litestream sidecar are unaffected.

### Restore from S3 (disaster recovery drill)

The `restore` service runs automatically at startup whenever the DB file
is missing. To exercise it:

```bash
docker compose down              # stops everything; data volume persists
docker volume rm study-help_data # destroys the on-disk DB
docker compose up -d             # restore service rehydrates from S3
docker compose logs restore      # confirm "restored snapshot" message
```

Practice once on a throwaway box before you need it for real.

### Logs and metrics

```bash
docker compose logs -f app           # application logs
docker compose logs -f litestream    # replication progress / errors
docker compose exec app wget -qO- 127.0.0.1:9090/metrics
```

The metrics endpoint is bound to `127.0.0.1:9090` inside the container by
design — only reachable via `docker exec`. For a real dashboard, point a
Grafana Cloud free-tier agent at it.

### Rotating the AWS access key

```bash
# in AWS console: create a new key for the litestream user
$EDITOR .env                          # update AWS_ACCESS_KEY_ID + SECRET
docker compose up -d                  # picks up the new env on restart
# then deactivate the old key in AWS console after confirming replication
```

## Files

| File              | Purpose                                                    |
|-------------------|------------------------------------------------------------|
| `compose.yaml`    | Service definitions (app + caddy + litestream + restore).  |
| `Caddyfile`       | Reverse proxy + automatic Let's Encrypt cert.              |
| `litestream.yml`  | What to replicate, where, and how often.                   |
| `.env.example`    | Template for `.env` (image tag, secrets, AWS creds).       |
