#!/usr/bin/env sh
set -eu

OPS_ENV_FILE=${OPS_ENV_FILE:-/etc/tka/ops.env}
if [ -f "$OPS_ENV_FILE" ]; then
  set -a
  # shellcheck disable=SC1090
  . "$OPS_ENV_FILE"
  set +a
fi

: "${BACKUP_OBJECT:?set full .dump object URI or local path}"
: "${RESTORE_DATABASE_URL:?set URL of a NEW, EMPTY recovery database}"
[ "${RESTORE_CONFIRM:-}" = "restore-to-new-empty-database" ] || {
  echo "Refusing restore. Set RESTORE_CONFIRM=restore-to-new-empty-database" >&2
  exit 1
}

if [ -n "${DATABASE_URL_FILE:-}" ] && [ -f "$DATABASE_URL_FILE" ]; then
  production_url=$(tr -d '\r\n' < "$DATABASE_URL_FILE")
  [ "$production_url" != "$RESTORE_DATABASE_URL" ] || {
    echo "Refusing to restore directly over the production database" >&2
    exit 1
  }
fi

for command_name in pg_restore psql sha256sum mktemp; do
  command -v "$command_name" >/dev/null 2>&1 || { echo "Missing command: $command_name" >&2; exit 1; }
done

restore_dir=$(mktemp -d "${TMPDIR:-/tmp}/tka-restore.XXXXXX")
trap 'rm -rf "$restore_dir"' EXIT INT TERM
backup_name=$(basename "$BACKUP_OBJECT")
backup_file="$restore_dir/$backup_name"
checksum_file="$backup_file.sha256"

case "$BACKUP_OBJECT" in
  s3://*)
    aws s3 cp "$BACKUP_OBJECT" "$backup_file" --only-show-errors
    aws s3 cp "${BACKUP_OBJECT}.sha256" "$checksum_file" --only-show-errors
    ;;
  gs://*)
    gcloud storage cp "$BACKUP_OBJECT" "$backup_file"
    gcloud storage cp "${BACKUP_OBJECT}.sha256" "$checksum_file"
    ;;
  *)
    cp "$BACKUP_OBJECT" "$backup_file"
    cp "${BACKUP_OBJECT}.sha256" "$checksum_file"
    ;;
esac

(
  cd "$restore_dir"
  sha256sum -c "${backup_name}.sha256"
)
pg_restore --list "$backup_file" >/dev/null
pg_restore --dbname="$RESTORE_DATABASE_URL" --exit-on-error --single-transaction \
  --no-owner --no-privileges "$backup_file"

psql "$RESTORE_DATABASE_URL" -v ON_ERROR_STOP=1 -c \
  "SELECT COUNT(*) AS users FROM users; SELECT COUNT(*) AS exams FROM exams; SELECT COUNT(*) AS attempts FROM user_exams;"
echo "Restore selesai pada recovery database. Belum ada traffic production yang dialihkan."
