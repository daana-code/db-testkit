#!/usr/bin/env bash
# wait-for-healthy.sh - Wait for Docker container to report healthy status
#
# Usage: ./wait-for-healthy.sh CONTAINER_NAME [TIMEOUT_SECONDS]
#
# This script uses Docker health events to efficiently wait for a container
# to become healthy, avoiding arbitrary sleep delays.
#
# Exit codes:
#   0 - Container is healthy
#   1 - Container not found or timeout reached
#   2 - Invalid arguments

set -euo pipefail

# Check for timeout command (GNU coreutils)
TIMEOUT_CMD="timeout"
if ! command -v timeout >/dev/null 2>&1; then
    # macOS with GNU coreutils installed via brew
    if command -v gtimeout >/dev/null 2>&1; then
        TIMEOUT_CMD="gtimeout"
    else
        echo "Error: 'timeout' command not found. Install GNU coreutils (brew install coreutils on macOS)" >&2
        exit 2
    fi
fi

CONTAINER_NAME="${1}"
TIMEOUT="${2:-90}"  # Default 90 seconds timeout

# Validate arguments
if [ -z "$CONTAINER_NAME" ]; then
    echo "Error: Container name is required" >&2
    echo "Usage: $0 CONTAINER_NAME [TIMEOUT_SECONDS]" >&2
    exit 2
fi

# Check if container exists
if ! docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo "Error: Container '${CONTAINER_NAME}' not found" >&2
    exit 1
fi

# Check if container has health check configured
HAS_HEALTHCHECK=$(docker inspect "$CONTAINER_NAME" --format='{{if .State.Health}}true{{else}}false{{end}}')

if [ "$HAS_HEALTHCHECK" != "true" ]; then
    echo "Warning: Container '${CONTAINER_NAME}' has no health check configured" >&2
    echo "Falling back to pg_isready check..." >&2
    
    # Extract postgres user from environment if available
    PG_USER=$(docker inspect "$CONTAINER_NAME" --format='{{range .Config.Env}}{{println .}}{{end}}' | grep POSTGRES_USER | cut -d= -f2 || echo "postgres")
    
    # Poll with pg_isready
    RETRIES=$((TIMEOUT / 2))
    for i in $(seq 1 $RETRIES); do
        if docker exec "$CONTAINER_NAME" pg_isready -U "$PG_USER" >/dev/null 2>&1; then
            echo "✅ Container '${CONTAINER_NAME}' is ready (pg_isready)"
            exit 0
        fi
        sleep 2
    done
    echo "Error: Container '${CONTAINER_NAME}' did not become ready within ${TIMEOUT}s" >&2
    exit 1
fi

# Check current health status
CURRENT_HEALTH=$(docker inspect "$CONTAINER_NAME" --format='{{.State.Health.Status}}')

if [ "$CURRENT_HEALTH" = "healthy" ]; then
    echo "✅ Container '${CONTAINER_NAME}' is already healthy"
    exit 0
fi

echo "⏳ Waiting for container '${CONTAINER_NAME}' to become healthy (timeout: ${TIMEOUT}s)..."

# Get container ID for event filtering
CONTAINER_ID=$(docker inspect "$CONTAINER_NAME" --format='{{.Id}}')

# Subscribe to health events with timeout using a FIFO for proper process control
FIFO=$(mktemp -u)
mkfifo "$FIFO"
trap 'rm -f "$FIFO"' EXIT

# Start docker events in background
$TIMEOUT_CMD "$TIMEOUT" docker events \
    --filter "container=${CONTAINER_ID}" \
    --filter "event=health_status" \
    --format '{{.Status}}' > "$FIFO" &
DOCKER_PID=$!

# Read from FIFO
while read -r status; do
    if [ "$status" = "health_status: healthy" ]; then
        echo "✅ Container '${CONTAINER_NAME}' is healthy"
        kill $DOCKER_PID 2>/dev/null || true
        wait $DOCKER_PID 2>/dev/null || true
        exit 0
    elif [ "$status" = "health_status: unhealthy" ]; then
        echo "⚠️  Container '${CONTAINER_NAME}' reported unhealthy, continuing to wait..." >&2
    fi
done < "$FIFO"

# If we get here, docker events exited (timeout or error)
wait $DOCKER_PID 2>/dev/null || EXIT_CODE=$?
if [ "${EXIT_CODE:-0}" -eq 124 ]; then
    echo "Error: Timeout waiting for '${CONTAINER_NAME}' to become healthy after ${TIMEOUT}s" >&2
    echo "Tip: Check container logs with: docker logs ${CONTAINER_NAME}" >&2
fi
exit 1

