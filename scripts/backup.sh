#!/bin/bash
#
# Database backup script for media-server
# Creates timestamped backups of the SQLite database
# Safe for use with WAL mode
#

set -e

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/data/backups}"
DB_FILE="${DB_FILE:-/data/media-server.db}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
DATE=$(date +%Y%m%d_%H%M%S)

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR"

# Check if database exists
if [ ! -f "$DB_FILE" ]; then
    echo "Error: Database file not found at $DB_FILE"
    exit 1
fi

# Create backup using SQLite's .backup command (safe for WAL mode)
BACKUP_FILE="$BACKUP_DIR/media-server_$DATE.db"
echo "Creating backup: $BACKUP_FILE"
sqlite3 "$DB_FILE" ".backup '$BACKUP_FILE'"

# Verify backup was created
if [ -f "$BACKUP_FILE" ]; then
    SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
    echo "Backup created successfully: $BACKUP_FILE ($SIZE)"
else
    echo "Error: Backup failed"
    exit 1
fi

# Remove backups older than retention period
echo "Cleaning up backups older than $RETENTION_DAYS days..."
find "$BACKUP_DIR" -name "media-server_*.db" -type f -mtime +$RETENTION_DAYS -delete

# List recent backups
echo ""
echo "Recent backups:"
ls -lh "$BACKUP_DIR"/media-server_*.db 2>/dev/null | tail -5 || echo "No backups found"

echo ""
echo "Backup complete!"
