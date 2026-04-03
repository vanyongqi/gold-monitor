#!/usr/bin/env bash

set -euo pipefail

IMAGE="${IMAGE:-ghcr.io/vanyongqi/gold-monitor:latest}"
CONTAINER_NAME="${CONTAINER_NAME:-gold-monitor}"
HOST_PORT="${HOST_PORT:-127.0.0.1:18090:8080}"
STORAGE_DIR="${STORAGE_DIR:-/opt/gold-monitor/storage}"

/usr/bin/docker pull "$IMAGE"

CURRENT_IMAGE_ID=$(/usr/bin/docker inspect --format "{{.Image}}" "$CONTAINER_NAME" 2>/dev/null || true)
TARGET_IMAGE_ID=$(/usr/bin/docker image inspect "$IMAGE" --format "{{.Id}}")

if [ "$CURRENT_IMAGE_ID" = "$TARGET_IMAGE_ID" ] && [ "${1:-}" != "--force" ]; then
  exit 0
fi

/usr/bin/docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
/usr/bin/docker run -d \
  --name "$CONTAINER_NAME" \
  --restart unless-stopped \
  -p "$HOST_PORT" \
  -v "$STORAGE_DIR:/app/storage" \
  "$IMAGE" >/tmp/"$CONTAINER_NAME"-container.id
