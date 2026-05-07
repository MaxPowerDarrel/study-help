# Docker

**Status:** Shipped <!-- Draft | In Progress | Shipped | Deprecated -->
**Created:** 2026-05-04
**Last updated:** 2026-05-07
**Owner:** unassigned

> **Editor's note (2026-05-07):** the spec body below references `SESSION_SECRET`, `DATABASE_URL`, and a SQLite host-mount volume. Those were correct at the time of writing; they are no longer applicable. The auth + highlights features that needed a database were retired on 2026-05-07 (see [accounts.md](./accounts.md), [highlights.md](./highlights.md)), and the live `Dockerfile` / `compose.yaml` no longer mount a data volume or pass those env vars. Current required env: `ESV_API_KEY` and `YOUVERSION_APP_KEY`.

## Why

Running `study-help` today requires Go toolchain, Node, and the right env vars configured locally — a significant barrier to consistent deployment across environments. Packaging the application as a Docker image gives it a reproducible, portable runtime: the same binary and SPA bundle that runs in CI can run on a VPS or any container host. This supports PROJECT_CONSTITUTION.md §4 (Architectural Guardrails) — secrets stay server-side via env vars injected at runtime, never baked into the image.

## Goals

- [x] A deployer can start the full app with a single `docker run` command — no local Go or Node toolchain required.
- [x] A `compose.yaml` supports spinning up the app locally with env vars preset.

## Non-goals

- No Kubernetes manifests (Helm, Deployment YAMLs) — out of scope; plain Docker is sufficient.
- No production docker-compose — the Compose file is for local dev only.
- No image registry publishing in CI — CI builds the image to verify; it does not push.

## User-facing behavior

The deployer:

1. Pulls or builds the image.
2. Mounts a host directory for the SQLite database file as a volume.
3. Runs the container with required env vars passed through from the host shell:
   ```
   docker run \
     -e ESV_API_KEY=$ESV_API_KEY \
     -e SESSION_SECRET=$SESSION_SECRET \
     -e DATABASE_URL=/data/sqlite.db \
     -v $(pwd)/data:/data \
     -p 8080:8080 \
     study-help
   ```
4. The app is reachable on the configured port. `GET /healthz` confirms it's up.

For local development, `compose.yaml` presets the env vars and volume mount so the workflow is just `docker compose up`.

## Implementation outline

- `Dockerfile` at repo root, three stages:
  - **frontend**: Node 22 Alpine — `npm ci && npm run build`, outputs `dist/`
  - **backend**: Go 1.26 Alpine — copies `dist/` into `internal/web/dist/`, runs `go build -o study-help .`
  - **runtime**: `gcr.io/distroless/static` — copies the binary only; no shell, no OS packages
- `compose.yaml` at repo root — mounts `./data:/data`, sets env vars from `.env`, exposes port 8080
- `.dockerignore` to exclude `node_modules/`, `internal/web/dist/` (built inside the image), and test fixtures

## Open questions

- [x] What base image is planned for the runtime stage? → Resolved: `gcr.io/distroless/static`; see Implementation outline and Decisions 2026-05-04.
- [x] Is the SQLite database file expected to be mounted as a volume? → Resolved: yes, host-mounted volume; see Decisions 2026-05-04.
- [x] What is the intended deployment target? → Resolved: local machine; see Decisions 2026-05-04.
- [x] Will the private metrics port (`127.0.0.1:9090`) remain localhost-only when containerized? → Resolved: yes, Prometheus scraping is out of scope; see Decisions 2026-05-04.
- [x] Are there any Non-goals scoped to this feature? → Resolved: see Non-goals section.
- [x] What Verification steps are planned? → Resolved: see Verification section.

## Decisions

- 2026-05-04: Spec created.
- 2026-05-04: Use a three-stage multi-stage Dockerfile (Node → Go → distroless runtime). Reason: keeps the final image minimal and eliminates build toolchain from the shipped artifact.
- 2026-05-04: SQLite data directory is a host-mounted volume (`-v`). Reason: preserves data across container restarts without a separate database service.
- 2026-05-04: Deployment target is local machine. Docker packages the app so it can run without a local Go or Node toolchain; production hosting is out of scope for this feature.
- 2026-05-04: Metrics port stays bound to `127.0.0.1:9090` inside the container. Prometheus scraping is out of scope for this feature.
- 2026-05-04: Runtime base image is `gcr.io/distroless/static`. Reason: minimal attack surface, no shell, smallest possible image — acceptable because the Go binary is statically compiled and needs no libc.
- 2026-05-04: Secrets are passed to the container via `-e KEY=$KEY` (host env var passthrough), not hardcoded values. Reason: secrets already live as env vars in the deployer's shell; passthrough avoids ever writing them into a command literal.
- 2026-05-04: `compose.yaml` exposes `APP_PORT` (default `8080`) for the host-side port mapping. Reason: the deployer's local `go run .` may already hold port 8080; `APP_PORT=18080 docker compose up` lets both run side-by-side without editing the compose file.
- 2026-05-04: `Dockerfile` only copies `main.go` and `internal/` into the Go build stage (not the entire repo). Reason: keeps the build context small and explicit about what the binary depends on.
- 2026-05-04: Compose file is named `compose.yaml` (Compose Specification's preferred name) instead of the legacy `docker-compose.yml`. Reason: Compose v2+ treats `compose.yaml` as canonical; `docker-compose.yml` is kept only for backward compatibility.

## Verification

- [x] `docker run` smoke test: run the image with required env vars, hit `GET /healthz`, expect 200 — no local Go or Node toolchain installed. (Verified 2026-05-04 against `study-help:test`.)
- [x] `docker compose up` smoke test: SPA loads at `localhost:8080` and passage fetch works end-to-end. (Verified 2026-05-04 with `APP_PORT=18080`; `<title>study-help</title>` returned.)

## Related

- Constitution sections: `PROJECT_CONSTITUTION.md §4`