#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(dirname "$SCRIPT_DIR")"
ENV_FILE="$SCRIPT_DIR/.env"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.prod.yml"

# ── Helpers ──────────────────────────────────────────
red()   { printf '\033[0;31m%s\033[0m\n' "$*"; }
green() { printf '\033[0;32m%s\033[0m\n' "$*"; }
info()  { printf '→ %s\n' "$*"; }
die()   { red "FATAL: $*" >&2; exit 1; }

# ── Load config ─────────────────────────────────────
[ -f "$ENV_FILE" ] || die "deploy/.env not found. Copy deploy/.env.example and fill in values."
set -a; source "$ENV_FILE"; set +a

for var in DEPLOY_HOST DEPLOY_DATA_DIR DEPLOY_PORT DEPLOY_HEALTHCHECK_URL YAITRACKER_SECRET; do
    [ -n "${!var:-}" ] || die "Required variable $var is not set in deploy/.env"
done

VERSION="${1:-}"
[ -n "$VERSION" ] || die "Usage: $0 <version>  (e.g. v0.6.0)"

IMAGE="yaitracker:$VERSION"
CONTAINER="yaitracker"
TIMESTAMP="$(date -u +%Y%m%d-%H%M%S)"
REMOTE_COMPOSE="/tmp/yaitracker-compose-$TIMESTAMP.yml"
BACKUP_DIR="$DEPLOY_DATA_DIR/backups"

# ── Pre-flight checks ───────────────────────────────
info "Pre-flight: checking SSH connectivity to $DEPLOY_HOST..."
ssh -o ConnectTimeout=5 "$DEPLOY_HOST" true || die "Cannot SSH to $DEPLOY_HOST"

info "Pre-flight: verifying data directory exists on remote..."
ssh "$DEPLOY_HOST" "[ -d '$DEPLOY_DATA_DIR' ]" || die "Data directory $DEPLOY_DATA_DIR does not exist on $DEPLOY_HOST"

# ── Build image ──────────────────────────────────────
info "Building Docker image $IMAGE..."
docker build -t "$IMAGE" -t yaitracker:latest "$REPO_DIR"

# ── Transfer image ───────────────────────────────────
info "Transferring image to $DEPLOY_HOST..."
docker save "$IMAGE" | ssh "$DEPLOY_HOST" 'docker load'

# ── Backup database ──────────────────────────────────
info "Backing up database..."
ssh "$DEPLOY_HOST" "mkdir -p '$BACKUP_DIR'"
DB_FILE="$DEPLOY_DATA_DIR/yaitracker.db"
if ssh "$DEPLOY_HOST" "[ -f '$DB_FILE' ]"; then
    ssh "$DEPLOY_HOST" "cp '$DB_FILE' '$BACKUP_DIR/yaitracker-$TIMESTAMP.db'"
    # Also back up WAL/SHM if present
    ssh "$DEPLOY_HOST" "[ -f '${DB_FILE}-wal' ] && cp '${DB_FILE}-wal' '$BACKUP_DIR/yaitracker-$TIMESTAMP.db-wal' || true"
    ssh "$DEPLOY_HOST" "[ -f '${DB_FILE}-shm' ] && cp '${DB_FILE}-shm' '$BACKUP_DIR/yaitracker-$TIMESTAMP.db-shm' || true"
    green "Backup: $BACKUP_DIR/yaitracker-$TIMESTAMP.db"
else
    info "No existing database to back up (first deploy?)."
fi

# ── Deploy via compose ───────────────────────────────
info "Uploading compose file to remote..."
scp -q "$COMPOSE_FILE" "$DEPLOY_HOST:$REMOTE_COMPOSE"

info "Stopping current container (if running)..."
ssh "$DEPLOY_HOST" "docker stop $CONTAINER 2>/dev/null || true"
ssh "$DEPLOY_HOST" "docker rm $CONTAINER 2>/dev/null || true"

info "Starting $IMAGE..."
ssh "$DEPLOY_HOST" "VERSION=$VERSION DEPLOY_PORT=$DEPLOY_PORT DEPLOY_DATA_DIR=$DEPLOY_DATA_DIR YAITRACKER_SECRET=$YAITRACKER_SECRET docker compose -f '$REMOTE_COMPOSE' up -d"

# ── Health check ─────────────────────────────────────
info "Waiting for health check at $DEPLOY_HEALTHCHECK_URL..."
healthy=false
for i in $(seq 1 15); do
    if curl -sf --max-time 2 "$DEPLOY_HEALTHCHECK_URL" >/dev/null 2>&1; then
        healthy=true
        break
    fi
    sleep 2
done

if [ "$healthy" = false ]; then
    red "Health check failed after 30 seconds!"
    info "Container logs:"
    ssh "$DEPLOY_HOST" "docker logs --tail 20 $CONTAINER" 2>&1 || true

    # Rollback
    if ssh "$DEPLOY_HOST" "[ -f '$BACKUP_DIR/yaitracker-$TIMESTAMP.db' ]"; then
        red "Rolling back database..."
        ssh "$DEPLOY_HOST" "docker stop $CONTAINER 2>/dev/null || true"
        ssh "$DEPLOY_HOST" "docker rm $CONTAINER 2>/dev/null || true"
        ssh "$DEPLOY_HOST" "cp '$BACKUP_DIR/yaitracker-$TIMESTAMP.db' '$DB_FILE'"
        ssh "$DEPLOY_HOST" "[ -f '$BACKUP_DIR/yaitracker-$TIMESTAMP.db-wal' ] && cp '$BACKUP_DIR/yaitracker-$TIMESTAMP.db-wal' '${DB_FILE}-wal' || true"
        ssh "$DEPLOY_HOST" "[ -f '$BACKUP_DIR/yaitracker-$TIMESTAMP.db-shm' ] && cp '$BACKUP_DIR/yaitracker-$TIMESTAMP.db-shm' '${DB_FILE}-shm' || true"
        red "Database restored. Container is stopped. Investigate and redeploy manually."
    fi
    ssh "$DEPLOY_HOST" "rm -f '$REMOTE_COMPOSE'"
    exit 1
fi
green "Health check passed."

# ── Verify data ──────────────────────────────────────
info "Verifying database file exists and is non-empty..."
DB_SIZE=$(ssh "$DEPLOY_HOST" "stat -c%s '$DB_FILE' 2>/dev/null || echo 0")
if [ "$DB_SIZE" -lt 1024 ]; then
    red "WARNING: Database file is suspiciously small ($DB_SIZE bytes). Investigate!"
else
    green "Database OK ($DB_SIZE bytes)."
fi

# ── Cleanup ──────────────────────────────────────────
info "Cleaning up..."
ssh "$DEPLOY_HOST" "rm -f '$REMOTE_COMPOSE'"
ssh "$DEPLOY_HOST" "docker image prune -f >/dev/null 2>&1 || true"

# Prune backups older than 30 days
ssh "$DEPLOY_HOST" "find '$BACKUP_DIR' -name 'yaitracker-*.db*' -mtime +30 -delete 2>/dev/null || true"

green "Deploy complete: $IMAGE on $DEPLOY_HOST"
