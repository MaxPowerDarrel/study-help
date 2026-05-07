#!/usr/bin/env bash
# deploy.sh — runs on the Lightsail VM. Atomically swap APP_IMAGE in
# /opt/study-help/.env, pull the new image, restart the app container,
# wait for /healthz, and roll back to the previous image on failure.
#
# Usage:   deploy.sh <image-ref>
# Example: deploy.sh ghcr.io/you/study-help:sha-abc1234
#
# Invoked by .github/workflows/deploy.yml via SSH; safe to run by hand
# from an SSH session too.

set -euo pipefail

NEW_IMAGE="${1:?usage: deploy.sh <image-ref>}"
APP_DIR="${APP_DIR:-/opt/study-help}"
HEALTH_TIMEOUT_SEC="${HEALTH_TIMEOUT_SEC:-60}"

log() { printf '[deploy] %s\n' "$*"; }
err() { printf '[deploy] %s\n' "$*" >&2; }

cd "$APP_DIR"

[ -f .env ] || { err "missing $APP_DIR/.env"; exit 1; }
[ -f compose.yaml ] || { err "missing $APP_DIR/compose.yaml"; exit 1; }

# Snapshot the current image for potential rollback.
OLD_IMAGE="$(grep -E '^APP_IMAGE=' .env | head -n1 | cut -d= -f2-)"
log "current: ${OLD_IMAGE:-<unset>}"
log "target:  ${NEW_IMAGE}"

if [ "$OLD_IMAGE" = "$NEW_IMAGE" ]; then
  log "image unchanged; running compose up to ensure desired state"
fi

# Atomically rewrite .env with the new APP_IMAGE.
write_app_image() {
  local image="$1"
  local tmp=".env.deploy.tmp"
  if grep -qE '^APP_IMAGE=' .env; then
    sed "s|^APP_IMAGE=.*|APP_IMAGE=${image}|" .env > "$tmp"
  else
    cat .env > "$tmp"
    printf 'APP_IMAGE=%s\n' "$image" >> "$tmp"
  fi
  chmod 600 "$tmp"
  mv "$tmp" .env
}

write_app_image "$NEW_IMAGE"

log "pulling image"
docker compose pull app

log "restarting app"
docker compose up -d app

# Healthcheck: exec into the caddy container (alpine, has wget) and
# call the app's /healthz over the bridge network. The app's distroless
# image has no shell, so we can't curl from inside it directly.
log "waiting for /healthz (timeout: ${HEALTH_TIMEOUT_SEC}s)"
deadline=$((SECONDS + HEALTH_TIMEOUT_SEC))
healthy=0
while (( SECONDS < deadline )); do
  if docker compose exec -T caddy wget -qO- http://app:8080/healthz >/dev/null 2>&1; then
    healthy=1
    break
  fi
  sleep 1
done

if (( healthy == 1 )); then
  log "deploy ok: $NEW_IMAGE"
  exit 0
fi

err "healthcheck failed; rolling back to ${OLD_IMAGE:-<previous>}"

if [ -z "$OLD_IMAGE" ]; then
  err "no previous APP_IMAGE recorded in .env; nothing to roll back to"
  exit 1
fi

write_app_image "$OLD_IMAGE"
docker compose pull app || true
docker compose up -d app
err "rolled back to $OLD_IMAGE"
exit 1
