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
