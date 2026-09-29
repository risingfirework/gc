#!/usr/bin/env sh
set -eu

# Backup PostgreSQL lokal untuk instance produksi (tanpa offsite S3/GCS).
# Dipakai oleh cron harian; dump ditulis ke BACKUP_DIR lalu divalidasi dengan pg_restore.
# Untuk backup offsite, gunakan scripts/backup-db.sh (butuh BACKUP_URI s3:// atau gs://).

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
PROJECT_DIR=$(CDPATH='' cd -- "$SCRIPT_DIR/.." && pwd)

BACKUP_DIR=${BACKUP_DIR:-/var/backups/tka}
BACKUP_RETENTION_DAYS=${BACKUP_RETENTION_DAYS:-7}
BACKUP_LOCK_FILE=${BACKUP_LOCK_FILE:-/var/lock/tka-postgres-backup-local.lock}
COMPOSE_FILE=${COMPOSE_FILE:-docker-compose.prod.yml}
PROD_ENV_FILE=${PROD_ENV_FILE:-$PROJECT_DIR/.env.production}

command -v docker >/dev/null 2>&1 || { echo "Missing command: docker" >&2; exit 1; }
command -v flock >/dev/null 2>&1 || { echo "Missing command: flock" >&2; exit 1; }
command -v sha256sum >/dev/null 2>&1 || { echo "Missing command: sha256sum" >&2; exit 1; }

case "$BACKUP_DIR" in
  ""|/|/var|/var/) echo "Unsafe BACKUP_DIR: $BACKUP_DIR" >&2; exit 1 ;;
esac
mkdir -p "$BACKUP_DIR" "$(dirname "$BACKUP_LOCK_FILE")"
exec 9>"$BACKUP_LOCK_FILE"
flock -n 9 || { echo "Backup lain masih berjalan" >&2; exit 0; }

compose() {
  if [ -f "$PROD_ENV_FILE" ]; then
    docker compose --env-file "$PROD_ENV_FILE" -f "$PROJECT_DIR/$COMPOSE_FILE" "$@"
  else
    docker compose -f "$PROJECT_DIR/$COMPOSE_FILE" "$@"
  fi
}

timestamp=$(date -u +%Y%m%dT%H%M%SZ)
backup_name="tka-${timestamp}.dump"
partial_file="$BACKUP_DIR/.${backup_name}.partial"
backup_file="$BACKUP_DIR/$backup_name"

cleanup() {
  rm -f "$partial_file"
}
trap cleanup EXIT INT TERM

echo "[$(date -u +%FT%TZ)] Starting local PostgreSQL backup -> $backup_file"
compose exec -T postgres pg_dump \
  --username="${POSTGRES_USER:-tka}" \
  --dbname="${POSTGRES_DB:-tka}" \
  --format=custom --compress=zstd:9 \
  --no-owner --no-privileges > "$partial_file"

[ -s "$partial_file" ] || { echo "Dump kosong" >&2; exit 1; }
compose exec -T postgres pg_restore --list < "$partial_file" >/dev/null

mv "$partial_file" "$backup_file"
(cd "$BACKUP_DIR" && sha256sum "$backup_name" > "${backup_name}.sha256")
find "$BACKUP_DIR" -maxdepth 1 -type f -name 'tka-*.dump*' -mtime "+$BACKUP_RETENTION_DAYS" -delete
echo "[$(date -u +%FT%TZ)] Local backup OK: $backup_file ($(du -h "$backup_file" | cut -f1))"
