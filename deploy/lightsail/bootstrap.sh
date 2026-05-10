#!/usr/bin/env bash
# bootstrap.sh — first-time provisioning on a fresh Lightsail VM.
#
# Self-contained: only this file needs to be on the VM. With a GH_TOKEN
# env var set, the script shallow-clones the repo into a tmpdir, copies
# deploy/lightsail/{compose.yaml,Caddyfile,deploy.sh,.env.example} into
# /opt/study-help/, and removes the tmpdir on exit. The token is carried
# in `http.extraHeader` (Basic auth, x-access-token:$GH_TOKEN) for the
# single clone command, so nothing lands in .git/config or the clone URL
# and there is no on-disk credential footprint after the script returns.
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
#   REPO      required, format: owner/repo (e.g. owner/study-help).
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

: "${REPO:?REPO env var is required (format: owner/repo)}"
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

# GitHub's git smart-HTTP endpoint speaks Basic auth (with the PAT as
# the password), not Bearer — Bearer is for the REST API and for App
# installation tokens. We carry the auth as a one-shot extraHeader so
# the token never lands in .git/config or the clone URL, and disable
# the interactive credential prompt so any auth failure fails loudly
# instead of hanging on a tty.
AUTH_BASIC="$(printf 'x-access-token:%s' "$GH_TOKEN" | base64 | tr -d '\n')"

log "fetching deploy artifacts from $REPO @ $REPO_REF"
GIT_TERMINAL_PROMPT=0 git \
    -c "http.extraHeader=Authorization: Basic ${AUTH_BASIC}" \
    clone --depth=1 --no-tags --quiet \
    --branch "$REPO_REF" \
    "https://github.com/${REPO}.git" "$WORK/repo"

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
