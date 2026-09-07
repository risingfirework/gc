#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PROJECT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
DEV_SEED="$PROJECT_DIR/apps/backend/db/seeds/001_development.sql"
LOAD_SEED="$PROJECT_DIR/apps/backend/db/seeds/002_load_test.sql"

run_seed() {
  seed_file=$1
  if [ -n "${DATABASE_URL:-}" ]; then
    psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$seed_file"
  else
    docker compose -f "$PROJECT_DIR/docker-compose.yml" exec -T postgres \
      psql -v ON_ERROR_STOP=1 -U "${POSTGRES_USER:-tka}" -d "${POSTGRES_DB:-tka}" < "$seed_file"
  fi
}

run_seed "$DEV_SEED"
if [ "${LOAD_TEST_SEED:-0}" = "1" ]; then
  run_seed "$LOAD_SEED"
fi

echo "Seed selesai. Akun demo memakai password 12345678"
