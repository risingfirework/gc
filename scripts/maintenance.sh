#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PROJECT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
OPS_ENV_FILE=${OPS_ENV_FILE:-/etc/tka/ops.env}
if [ -f "$OPS_ENV_FILE" ]; then
  set -a
  # shellcheck disable=SC1090
  . "$OPS_ENV_FILE"
  set +a
fi

PROD_ENV_FILE=${PROD_ENV_FILE:-$PROJECT_DIR/.env.production}
LOG_DIR=${LOG_DIR:-/var/log/tka}
LOG_RETENTION_DAYS=${LOG_RETENTION_DAYS:-14}
REDIS_EXAM_CLEANUP_GRACE_SECONDS=${REDIS_EXAM_CLEANUP_GRACE_SECONDS:-86400}
REDIS_CLEANUP_MAX_KEYS=${REDIS_CLEANUP_MAX_KEYS:-10000}

case "$LOG_DIR" in
  /var/log/tka|/var/log/tka/*|"$PROJECT_DIR"/logs|"$PROJECT_DIR"/logs/*) ;;
  *) echo "Refusing unsafe LOG_DIR: $LOG_DIR" >&2; exit 1 ;;
esac

if [ -d "$LOG_DIR" ]; then
  find "$LOG_DIR" -type f \( -name '*.log' -o -name '*.log.*' \) \
    -mtime "+$LOG_RETENTION_DAYS" -delete
fi

compose() {
  docker compose --env-file "$PROD_ENV_FILE" -f "$PROJECT_DIR/docker-compose.prod.yml" "$@"
}

redis_cli() {
  node_name=$1
  shift
  compose exec -T "$node_name" sh -ec \
    'export REDISCLI_AUTH="$(cat /run/secrets/redis_password)"; exec redis-cli --no-auth-warning --raw "$@"' \
    redis-cli "$@"
}

now_ms=$(( $(date +%s) * 1000 ))
grace_ms=$(( REDIS_EXAM_CLEANUP_GRACE_SECONDS * 1000 ))
cutoff_ms=$(( now_ms - grace_ms ))
deleted=0

for node_name in redis-node-1 redis-node-2 redis-node-3 redis-node-4 redis-node-5 redis-node-6; do
  role=$(redis_cli "$node_name" INFO replication | tr -d '\r' | sed -n 's/^role://p')
  [ "$role" = "master" ] || continue
  cursor=0
  while :; do
    scan_result=$(redis_cli "$node_name" SCAN "$cursor" MATCH 'EXAM_TIMER:*' COUNT 250)
    cursor=$(printf '%s\n' "$scan_result" | sed -n '1p' | tr -d '\r')
    keys=$(printf '%s\n' "$scan_result" | sed '1d')
    for timer_key in $keys; do
      expires_at=$(redis_cli "$node_name" HGET "$timer_key" expires_at_unix_ms || true)
      case "$expires_at" in ''|*[!0-9]*) continue ;; esac
      if [ "$expires_at" -lt "$cutoff_ms" ]; then
        attempt_id=$(printf '%s' "$timer_key" | sed -n 's/^EXAM_TIMER:{\(.*\)}$/\1/p')
        [ -n "$attempt_id" ] || continue
        redis_cli "$node_name" UNLINK "$timer_key" "EXAM_ANSWERS:{$attempt_id}" >/dev/null
        deleted=$((deleted + 1))
        [ "$deleted" -lt "$REDIS_CLEANUP_MAX_KEYS" ] || break 3
      fi
    done
    [ "$cursor" = "0" ] && break
  done
done

echo "[$(date -u +%FT%TZ)] Maintenance complete; stale exam sessions removed: $deleted"
