#!/bin/bash
#
# Manual deployment script
# Run this on your server to pull and restart the media server
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
COMPOSE_FILE="${PROJECT_DIR}/docker-compose.prod.yml"

echo "=== Media Server Deployment ==="
echo "Project directory: ${PROJECT_DIR}"
echo "Compose file: ${COMPOSE_FILE}"
echo ""

# Check if compose file exists
if [ ! -f "$COMPOSE_FILE" ]; then
    echo "Error: docker-compose.prod.yml not found"
    exit 1
fi

# Check if .env exists
if [ ! -f "${PROJECT_DIR}/.env" ]; then
    echo "Warning: .env file not found. Using .env.example as template..."
    if [ -f "${PROJECT_DIR}/.env.example" ]; then
        cp "${PROJECT_DIR}/.env.example" "${PROJECT_DIR}/.env"
        echo "Please edit .env with your actual values!"
    fi
fi

cd "$PROJECT_DIR"

# Create backup directory if it doesn't exist
mkdir -p "${PROJECT_DIR}/backups"

# Create pre-deployment backup
echo "Creating pre-deployment backup..."
if docker ps --format '{{.Names}}' | grep -q "media-server"; then
    docker exec media-server /bin/sh -c 'sqlite3 /data/media-server.db ".backup /data/backups/media-server_pre-deploy_$(date +%Y%m%d_%H%M%S).db"' 2>/dev/null || \
    echo "Warning: Could not create backup inside container, trying local backup..."
fi

# Also create a local backup if database exists
if [ -f "${PROJECT_DIR}/data/media-server.db" ]; then
    BACKUP_FILE="${PROJECT_DIR}/backups/media-server_pre-deploy_$(date +%Y%m%d_%H%M%S).db"
    sqlite3 "${PROJECT_DIR}/data/media-server.db" ".backup '$BACKUP_FILE'" 2>/dev/null && \
    echo "Local backup created: $BACKUP_FILE" || \
    echo "Warning: Local backup failed (database may be locked)"
fi

echo "Pulling latest image..."
docker compose -f docker-compose.prod.yml pull

echo "Stopping current container..."
docker compose -f docker-compose.prod.yml down

echo "Starting new container..."
docker compose -f docker-compose.prod.yml up -d

echo "Cleaning up old images..."
docker image prune -f

echo ""
echo "=== Deployment complete ==="
echo ""
echo "Check status with: docker compose -f docker-compose.prod.yml ps"
echo "View logs with: docker compose -f docker-compose.prod.yml logs -f"
