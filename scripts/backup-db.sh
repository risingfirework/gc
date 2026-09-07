#!/usr/bin/env sh
set -eu

OPS_ENV_FILE=${OPS_ENV_FILE:-/etc/tka/ops.env}
if [ -f "$OPS_ENV_FILE" ]; then
  set -a
  # shellcheck disable=SC1090
  . "$OPS_ENV_FILE"
  set +a
fi

BACKUP_PROVIDER=${BACKUP_PROVIDER:-s3}
BACKUP_DB_MODE=${BACKUP_DB_MODE:-compose}
BACKUP_DIR=${BACKUP_DIR:-/var/backups/tka}
BACKUP_URI=${BACKUP_URI:?set BACKUP_URI, e.g. s3://bucket/tka or gs://bucket/tka}
BACKUP_LOCAL_RETENTION_DAYS=${BACKUP_LOCAL_RETENTION_DAYS:-2}
BACKUP_LOCK_FILE=${BACKUP_LOCK_FILE:-/var/lock/tka-postgres-backup.lock}
BACKUP_STATE_FILE=${BACKUP_STATE_FILE:-/var/lib/tka/backup-last-success}
BACKUP_METRICS_FILE=${BACKUP_METRICS_FILE:-/var/lib/node_exporter/textfile_collector/tka_backup.prom}

for command_name in pg_restore sha256sum flock; do
  command -v "$command_name" >/dev/null 2>&1 || { echo "Missing command: $command_name" >&2; exit 1; }
done

case "$BACKUP_DIR" in
  ""|/|/var|/var/) echo "Unsafe BACKUP_DIR: $BACKUP_DIR" >&2; exit 1 ;;
esac
mkdir -p "$BACKUP_DIR" "$(dirname "$BACKUP_LOCK_FILE")" "$(dirname "$BACKUP_STATE_FILE")" "$(dirname "$BACKUP_METRICS_FILE")"
exec 9>"$BACKUP_LOCK_FILE"
flock -n 9 || { echo "Backup lain masih berjalan" >&2; exit 0; }

timestamp=$(date -u +%Y%m%dT%H%M%SZ)
day_path=$(date -u +%Y/%m/%d)
backup_name="tka-${timestamp}.dump"
partial_file="$BACKUP_DIR/.${backup_name}.partial"
backup_file="$BACKUP_DIR/$backup_name"
checksum_file="$backup_file.sha256"
backup_success=0

write_metrics() {
  last_success=0
  [ -f "$BACKUP_STATE_FILE" ] && last_success=$(cat "$BACKUP_STATE_FILE")
  metrics_tmp="${BACKUP_METRICS_FILE}.tmp.$$"
  {
    echo '# HELP tka_database_backup_last_run_success Whether the most recent backup run succeeded.'
    echo '# TYPE tka_database_backup_last_run_success gauge'
    echo "tka_database_backup_last_run_success $backup_success"
    echo '# HELP tka_database_backup_last_success_timestamp_seconds Unix timestamp of the last successful backup.'
    echo '# TYPE tka_database_backup_last_success_timestamp_seconds gauge'
    echo "tka_database_backup_last_success_timestamp_seconds $last_success"
  } > "$metrics_tmp"
  mv "$metrics_tmp" "$BACKUP_METRICS_FILE"
}

cleanup() {
  exit_code=$?
  rm -f "$partial_file"
  write_metrics
  exit "$exit_code"
}
trap cleanup EXIT INT TERM

echo "[$(date -u +%FT%TZ)] Starting PostgreSQL backup"
case "$BACKUP_DB_MODE" in
  compose)
    command -v docker >/dev/null 2>&1 || { echo "Missing command: docker" >&2; exit 1; }
    SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
    PROJECT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
    PROD_ENV_FILE=${PROD_ENV_FILE:-$PROJECT_DIR/.env.production}
    docker compose --env-file "$PROD_ENV_FILE" -f "$PROJECT_DIR/docker-compose.prod.yml" \
      exec -T postgres pg_dump --username="${POSTGRES_USER:-tka}" \
      --dbname="${POSTGRES_DB:-tka}" --format=custom --compress=zstd:9 \
      --no-owner --no-privileges > "$partial_file"
    ;;
  url)
    command -v pg_dump >/dev/null 2>&1 || { echo "Missing command: pg_dump" >&2; exit 1; }
    if [ -n "${DATABASE_URL_FILE:-}" ]; then
      DATABASE_URL=$(tr -d '\r\n' < "$DATABASE_URL_FILE")
    fi
    : "${DATABASE_URL:?set DATABASE_URL or DATABASE_URL_FILE}"
    pg_dump --dbname="$DATABASE_URL" --format=custom --compress=zstd:9 \
      --no-owner --no-privileges --file="$partial_file"
    ;;
  *) echo "BACKUP_DB_MODE must be compose or url" >&2; exit 1 ;;
esac
pg_restore --list "$partial_file" >/dev/null
mv "$partial_file" "$backup_file"
(
  cd "$BACKUP_DIR"
  sha256sum "$backup_name" > "${backup_name}.sha256"
)

destination="${BACKUP_URI%/}/postgres/$day_path"
case "$BACKUP_PROVIDER" in
  s3)
    command -v aws >/dev/null 2>&1 || { echo "Missing command: aws" >&2; exit 1; }
    if [ -n "${AWS_KMS_KEY_ID:-}" ]; then
      aws s3 cp "$backup_file" "$destination/$backup_name" --only-show-errors \
        --sse aws:kms --sse-kms-key-id "$AWS_KMS_KEY_ID"
      aws s3 cp "$checksum_file" "$destination/${backup_name}.sha256" --only-show-errors \
        --sse aws:kms --sse-kms-key-id "$AWS_KMS_KEY_ID"
    else
      aws s3 cp "$backup_file" "$destination/$backup_name" --only-show-errors --sse AES256
      aws s3 cp "$checksum_file" "$destination/${backup_name}.sha256" --only-show-errors --sse AES256
    fi
    ;;
  gcs)
    command -v gcloud >/dev/null 2>&1 || { echo "Missing command: gcloud" >&2; exit 1; }
    gcloud storage cp "$backup_file" "$checksum_file" "$destination/"
    ;;
  *) echo "BACKUP_PROVIDER must be s3 or gcs" >&2; exit 1 ;;
esac

date +%s > "$BACKUP_STATE_FILE"
backup_success=1
find "$BACKUP_DIR" -maxdepth 1 -type f -mtime "+$BACKUP_LOCAL_RETENTION_DAYS" -delete
echo "[$(date -u +%FT%TZ)] Backup uploaded: $destination/$backup_name"
