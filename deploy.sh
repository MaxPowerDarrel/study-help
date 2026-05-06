#!/bin/bash

set -e

# Configuration
IMAGE_NAME="study-help"
IMAGE_TAG="latest"
CONTAINER_NAME="study-help"
PORT="${APP_PORT:-8081}"
DATABASE_DIR="./data"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
  echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
  echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
  echo -e "${RED}[ERROR]${NC} $1"
}

# Check for required environment variables
if [ -z "$ESV_API_KEY" ] || [ -z "$SESSION_SECRET" ]; then
  if [ -f .env ]; then
    log_info "Loading environment variables from .env"
    set -a
    source .env
    set +a
  else
    log_error "ESV_API_KEY and SESSION_SECRET must be set, or .env file must exist"
    exit 1
  fi
fi

# Create data directory if it doesn't exist
if [ ! -d "$DATABASE_DIR" ]; then
  log_info "Creating data directory: $DATABASE_DIR"
  mkdir -p "$DATABASE_DIR"
fi

# Build the Docker image
log_info "Building Docker image: $IMAGE_NAME:$IMAGE_TAG"
docker build -t "$IMAGE_NAME:$IMAGE_TAG" .

# Check if a container with this name is already running
if docker ps --filter "name=^${CONTAINER_NAME}$" --format '{{.Names}}' | grep -q "$CONTAINER_NAME"; then
  log_warn "Container $CONTAINER_NAME is running. Stopping and removing it..."
  docker stop "$CONTAINER_NAME"
  docker rm "$CONTAINER_NAME"
fi

# Check if a stopped container exists
if docker ps -a --filter "name=^${CONTAINER_NAME}$" --format '{{.Names}}' | grep -q "$CONTAINER_NAME"; then
  log_warn "Stopped container $CONTAINER_NAME exists. Removing it..."
  docker rm "$CONTAINER_NAME"
fi

# Start the new container
log_info "Starting container: $CONTAINER_NAME on port $PORT"
docker run -d \
  --name "$CONTAINER_NAME" \
  -p "$PORT:8080" \
  -e "ESV_API_KEY=$ESV_API_KEY" \
  -e "SESSION_SECRET=$SESSION_SECRET" \
  -e "YOUVERSION_APP_KEY=$YOUVERSION_APP_KEY" \
  -e "DATABASE_URL=/data/sqlite.db" \
  -e "ENV=dev" \
  -v "$(pwd)/$DATABASE_DIR:/data" \
  "$IMAGE_NAME:$IMAGE_TAG"

# Wait a moment for the container to start
sleep 2

# Check if container is running
if docker ps --filter "name=^${CONTAINER_NAME}$" --format '{{.Names}}' | grep -q "$CONTAINER_NAME"; then
  log_info "Container is running!"
  docker ps --filter "name=^${CONTAINER_NAME}$"
else
  log_error "Container failed to start. Checking logs..."
  docker logs "$CONTAINER_NAME"
  exit 1
fi

log_info "Deploy complete! App is running on http://localhost:$PORT"