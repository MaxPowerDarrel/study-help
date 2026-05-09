#!/usr/bin/env bash
# bootstrap.sh — first-time provisioning on a fresh Lightsail VM.
#
# Idempotent: safe to re-run. Installs Docker if missing, creates
# /opt/study-help/, and copies the deploy artifacts (compose.yaml,
# Caddyfile, deploy.sh) into place. Leaves .env as a copy of
# .env.example for the operator to fill in.
#
# Usage: scp this directory to the VM (no full repo clone needed),
# then on the VM:
#   cd <wherever-you-scp'd-it>
#   ./bootstrap.sh
#
# Run as a non-root user with sudo. After this script completes you
# still need to:
#   1. Edit /opt/study-help/.env (set image tag, API keys)
#   2. Edit /opt/study-help/Caddyfile (set the FQDN)
#   3. cd /opt/study-help && docker compose up -d
#
# CI handles GHCR auth at deploy time (transient `docker login` over SSH
# from the deploy workflow). For ad-hoc `docker compose pull` outside of
# CI, run `docker login ghcr.io` manually first.

set -euo pipefail

APP_DIR="${APP_DIR:-/opt/study-help}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

log() { printf '[bootstrap] %s\n' "$*"; }
err() { printf '[bootstrap] %s\n' "$*" >&2; }

if [ "$EUID" -eq 0 ]; then
  err "run as a non-root user with sudo, not as root"
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  log "installing docker + docker compose plugin"
  sudo apt-get update
  sudo apt-get install -y docker.io docker-compose-plugin
  sudo systemctl enable --now docker
  sudo usermod -aG docker "$USER"
  log "docker installed; log out and back in for the docker group to take effect, then re-run this script"
  exit 0
fi

# Sanity: docker compose plugin reachable.
if ! docker compose version >/dev/null 2>&1; then
  err "docker compose plugin not available; install docker-compose-plugin"
  exit 1
fi

log "creating $APP_DIR"
sudo mkdir -p "$APP_DIR"
sudo chown "$USER:$USER" "$APP_DIR"

log "copying artifacts"
install -m 0644 "$SCRIPT_DIR/compose.yaml"    "$APP_DIR/compose.yaml"
install -m 0644 "$SCRIPT_DIR/Caddyfile"        "$APP_DIR/Caddyfile"
install -m 0755 "$SCRIPT_DIR/deploy.sh"        "$APP_DIR/deploy.sh"

if [ ! -f "$APP_DIR/.env" ]; then
  log "seeding .env from .env.example (mode 0600)"
  install -m 0600 "$SCRIPT_DIR/.env.example" "$APP_DIR/.env"
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
