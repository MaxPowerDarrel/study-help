# syntax=docker/dockerfile:1

# Stage 1: build the SPA bundle.
# Vite is configured (web/vite.config.ts) to emit into ../internal/web/dist
# relative to the web/ directory — i.e. /app/internal/web/dist here.
FROM node:22-alpine AS frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: compile the Go binary with the SPA embedded.
# CGO_ENABLED=0 keeps the binary statically linked so it can run on
# distroless/static (no libc).
FROM golang:1.26-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY internal/ ./internal/
COPY --from=frontend /app/internal/web/dist ./internal/web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -o /study-help .

# Stage 3: minimal runtime image. No shell, no package manager, just the binary.
FROM gcr.io/distroless/static AS runtime
COPY --from=backend /study-help /study-help
EXPOSE 8080
ENTRYPOINT ["/study-help"]