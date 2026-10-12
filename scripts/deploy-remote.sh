#!/bin/bash
#
# Deploy the current commit to the Linux machine that runs the server.
# Run this on the Mac, from anywhere inside the repository:
#
#   ./scripts/deploy-remote.sh
#
# It sends HEAD to the server over SSH, backs up the database, fast-forwards
# the server's checkout, rebuilds the Docker image and restarts the container.
#
# The server cannot reach GitHub, so the code travels Mac -> server directly.
# Its checkout carries local edits that are not in the repository (media path,
# token lifetime, shared channels). They are left in place: the update is a
# fast-forward, which git refuses if it would touch a locally edited file, and
# a patch of them is saved beside the database backup each time.
#
# Override the target with DEPLOY_HOST and DEPLOY_DIR.

set -euo pipefail

HOST="${DEPLOY_HOST:-stephencjuliano@192.168.4.46}"
DIR="${DEPLOY_DIR:-/home/stephencjuliano/media-server}"
REF="refs/deploy/incoming"

cd "$(git rev-parse --show-toplevel)"

if ! git diff --quiet || ! git diff --cached --quiet; then
    echo "Note: uncommitted changes here are not deployed, only HEAD is."
fi

COMMIT="$(git rev-parse --short HEAD)"
echo "=== Deploying ${COMMIT} to ${HOST}:${DIR} ==="

# A ref of its own, because git refuses a push to the branch the server has
# checked out.
git push --force "ssh://${HOST}${DIR}" "HEAD:${REF}"

ssh "$HOST" "DIR='${DIR}' REF='${REF}' COMMIT='${COMMIT}' bash -s" <<'REMOTE'
set -euo pipefail
cd "$DIR"

stamp="$(date +%Y%m%d_%H%M%S)"
mkdir -p backups

echo "--- Backing up the database"
backup="backups/media-server_pre-deploy_${stamp}.db"
sqlite3 data/media-server.db ".backup '${backup}'"
if [ "$(sqlite3 "$backup" 'PRAGMA integrity_check;')" != "ok" ]; then
    echo "Backup failed its integrity check, stopping before anything changes." >&2
    exit 1
fi
echo "    ${backup}"

if ! git diff --quiet; then
    git diff > "backups/local-changes_${stamp}.patch"
    echo "    local edits saved to backups/local-changes_${stamp}.patch"
fi

echo "--- Updating the checkout"
if ! git merge-base --is-ancestor HEAD "$REF"; then
    echo "The server's checkout has commits that are not in ${COMMIT}, stopping." >&2
    echo "Bring them into the repository first: git -C $DIR log $REF..HEAD" >&2
    exit 1
fi
git merge --ff-only "$REF"

# Build before stopping anything, so a build failure leaves the old server up.
echo "--- Building"
# The first name lookup for Docker Hub regularly times out on this machine and
# the next one succeeds, so a failed build gets two more tries.
built=""
for attempt in 1 2 3; do
    if docker compose build --build-arg GIT_COMMIT="$COMMIT"; then
        built=yes
        break
    fi
    echo "    build attempt ${attempt} failed" >&2
    sleep 5
done
if [ -z "$built" ]; then
    echo "The build failed three times. The old server is still running." >&2
    exit 1
fi

echo "--- Restarting"
docker compose up -d

echo "--- Waiting for the server"
for _ in $(seq 1 30); do
    if curl -fs -m 2 http://localhost:8080/health > /dev/null; then
        curl -s -m 5 http://localhost:8080/api/deploy/status
        echo
        echo "=== Deployed ${COMMIT} ==="
        exit 0
    fi
    sleep 2
done

echo "The server did not come back within a minute. Recent logs:" >&2
docker compose logs --tail 40 >&2
exit 1
REMOTE
