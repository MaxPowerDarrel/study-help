#!/usr/bin/env bash
# bootstrap.sh — first-time provisioning on a fresh Lightsail VM.
#
# Self-contained: only this file needs to be on the VM. With a GH_TOKEN
# env var set, the script sparse-checkout-clones the repo's
# deploy/lightsail/ directory into a tmpdir, copies compose.yaml,
# Caddyfile, deploy.sh, and .env.example into /opt/study-help/, and
# removes the tmpdir. The token is carried in `http.extraHeader` for
# the single clone command, so nothing lands in .git/config and there
# is no on-disk credential footprint after the script returns.
#
# Idempotent: safe to re-run.
#
# Usage:
#
#   # one-time: download bootstrap.sh from the (private) repo. The same
#   # PAT works on raw.githubusercontent.com via Authorization header.
#   ssh <user>@<host>
#   TOKEN=ghp_xxx
#   curl -fsSL -H "Authorization: Bearer $TOKEN" \
#        https://raw.githubusercontent.com/MaxPowerDarrel/study-help/main/deploy/lightsail/bootstrap.sh \
#        -o bootstrap.sh
#   chmod +x bootstrap.sh
#
#   # first run: installs docker + git, then exits asking for a relogin
#   ./bootstrap.sh
#
#   # second run (after relogin): clones artifacts, scaffolds host
#   GH_TOKEN=$TOKEN ./bootstrap.sh
#
# Env contract:
#   GH_TOKEN  required on the cloning pass; PAT or fine-grained token
#             with `contents:read` on $REPO.
#   REPO      defaults to MaxPowerDarrel/study-help.
#   REPO_REF  branch or tag to clone; defaults to main.
#   APP_DIR   install target; defaults to /opt/study-help.
#
# After this script completes you still need to:
#   1. Edit /opt/study-help/.env (set image tag, API keys)
#   2. Edit /opt/study-help/Caddyfile (set the FQDN)
#   3. cd /opt/study-help && docker compose up -d
#
# CI handles GHCR auth at deploy time (transient `docker login` over SSH
# from the deploy workflow). For ad-hoc `docker compose pull` outside of
# CI, run `docker login ghcr.io` manually first.

set -euo pipefail

REPO="${REPO:-MaxPowerDarrel/study-help}"
REPO_REF="${REPO_REF:-main}"
APP_DIR="${APP_DIR:-/opt/study-help}"

log() { printf '[bootstrap] %s\n' "$*"; }
err() { printf '[bootstrap] %s\n' "$*" >&2; }

if [ "$EUID" -eq 0 ]; then
  err "run as a non-root user with sudo, not as root"
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  log "installing docker + docker compose plugin + git"
  sudo apt-get update
  sudo apt-get install -y docker.io docker-compose-plugin git
  sudo systemctl enable --now docker
  sudo usermod -aG docker "$USER"
  log "docker installed; log out and back in for the docker group, then re-run this script with GH_TOKEN set"
  exit 0
fi

if ! docker compose version >/dev/null 2>&1; then
  err "docker compose plugin not available; install docker-compose-plugin"
  exit 1
fi

if ! command -v git >/dev/null 2>&1; then
  log "installing git"
  sudo apt-get update
  sudo apt-get install -y git
fi

if [ -z "${GH_TOKEN:-}" ]; then
  err "GH_TOKEN env var required (PAT with contents:read on $REPO)"
  err "example: GH_TOKEN=ghp_xxx ./bootstrap.sh"
  exit 1
fi

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

log "fetching deploy artifacts from $REPO @ $REPO_REF (sparse: deploy/lightsail/)"
git -c http.extraHeader="Authorization: Bearer $GH_TOKEN" \
    clone --depth=1 --no-tags --filter=blob:none --sparse \
    --branch "$REPO_REF" --quiet \
    "https://github.com/${REPO}.git" "$WORK/repo"

(cd "$WORK/repo" && git sparse-checkout set deploy/lightsail >/dev/null)

ARTIFACTS="$WORK/repo/deploy/lightsail"
for f in compose.yaml Caddyfile deploy.sh .env.example; do
  [ -f "$ARTIFACTS/$f" ] || { err "missing $f in cloned artifacts"; exit 1; }
done

log "creating $APP_DIR"
sudo mkdir -p "$APP_DIR"
sudo chown "$USER:$USER" "$APP_DIR"

log "copying artifacts"
install -m 0644 "$ARTIFACTS/compose.yaml" "$APP_DIR/compose.yaml"
install -m 0644 "$ARTIFACTS/Caddyfile"     "$APP_DIR/Caddyfile"
install -m 0755 "$ARTIFACTS/deploy.sh"     "$APP_DIR/deploy.sh"

if [ ! -f "$APP_DIR/.env" ]; then
  log "seeding .env from .env.example (mode 0600)"
  install -m 0600 "$ARTIFACTS/.env.example" "$APP_DIR/.env"
else
  log ".env already present; leaving it alone"
fi

cat <<EOF

[bootstrap] complete: $APP_DIR

Next steps:
  1. \$EDITOR $APP_DIR/.env              # fill in every CHANGEME
  2. \$EDITOR $APP_DIR/Caddyfile         # replace study.example.com
  3. cd $APP_DIR && docker compose up -d
  4. docker compose logs -f caddy       # watch the first cert issue

The deploy workflow handles GHCR auth automatically at deploy time. For
the very first \`docker compose up -d\` (or any ad-hoc \`docker compose
pull\` outside CI), run \`docker login ghcr.io\` manually first.
EOF
