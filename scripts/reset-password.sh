#!/bin/bash
#
# Password reset utility for media-server
# Usage: ./reset-password.sh <username> <new_password>
#

set -e

DB_FILE="${DB_FILE:-/data/media-server.db}"

if [ "$#" -lt 2 ]; then
    echo "Usage: $0 <username> <new_password>"
    echo ""
    echo "Examples:"
    echo "  $0 admin newpassword123"
    echo "  $0 juliano mypassword"
    echo ""
    echo "Current users:"
    sqlite3 "$DB_FILE" "SELECT username, email FROM users;" 2>/dev/null || echo "Could not connect to database"
    exit 1
fi

USERNAME="$1"
NEW_PASSWORD="$2"

# Check if database exists
if [ ! -f "$DB_FILE" ]; then
    echo "Error: Database file not found at $DB_FILE"
    echo "Try setting DB_FILE environment variable"
    exit 1
fi

# Check if user exists
USER_EXISTS=$(sqlite3 "$DB_FILE" "SELECT COUNT(*) FROM users WHERE username = '$USERNAME';")
if [ "$USER_EXISTS" -eq 0 ]; then
    echo "Error: User '$USERNAME' not found"
    echo ""
    echo "Available users:"
    sqlite3 "$DB_FILE" "SELECT username, email FROM users;"
    exit 1
fi

# Generate bcrypt hash using Python
HASH=$(python3 -c "import bcrypt; print(bcrypt.hashpw(b'$NEW_PASSWORD', bcrypt.gensalt()).decode())")

if [ -z "$HASH" ]; then
    echo "Error: Failed to generate password hash"
    echo "Make sure Python 3 and bcrypt module are installed:"
    echo "  pip3 install bcrypt"
    exit 1
fi

# Update password
sqlite3 "$DB_FILE" "UPDATE users SET password_hash = '$HASH' WHERE username = '$USERNAME';"

echo "Password updated successfully for user: $USERNAME"
