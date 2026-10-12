#!/bin/bash
# Finds the media drive by UUID and updates MEDIA_PATH in .env
# The drive's UUID is stable even if the mount name changes after unexpected reboots.

MEDIA_DRIVE_UUID="149AB7C89AB7A4A0"
ENV_FILE="$(dirname "$0")/.env"

# Find mount point by UUID
MOUNT_POINT=$(lsblk -o UUID,MOUNTPOINT -nr | awk -v uuid="$MEDIA_DRIVE_UUID" '$1 == uuid { print $2 }')

if [ -z "$MOUNT_POINT" ]; then
    echo "ERROR: Media drive (UUID=$MEDIA_DRIVE_UUID) not found or not mounted." >&2
    exit 1
fi

echo "Found media drive at: $MOUNT_POINT"

# Update MEDIA_PATH in .env
if grep -q "^MEDIA_PATH=" "$ENV_FILE"; then
    sed -i "s|^MEDIA_PATH=.*|MEDIA_PATH=$MOUNT_POINT|" "$ENV_FILE"
else
    echo "MEDIA_PATH=$MOUNT_POINT" >> "$ENV_FILE"
fi

echo "Updated MEDIA_PATH=$MOUNT_POINT in .env"
