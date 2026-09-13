#!/usr/bin/env sh
set -eu

# Buat akun owner pertama (bootstrap) terhadap database yang berjalan.
# Pemakaian: CREATE_OWNER_PASSWORD='rahasia-kuat' scripts/create-owner.sh admin@domain.com
# Secara default memakai compose produksi; override dengan COMPOSE_FILE=docker-compose.yml untuk lokal.

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PROJECT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

EMAIL=${1:-}
if [ -z "$EMAIL" ]; then
  echo "Pemakaian: CREATE_OWNER_PASSWORD='...' scripts/create-owner.sh EMAIL"
  exit 1
fi
: "${CREATE_OWNER_PASSWORD:?set CREATE_OWNER_PASSWORD (8-72 karakter)}"

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"

docker compose -f "$PROJECT_DIR/$COMPOSE_FILE" exec -T backend \
  /tka-api create-owner --email "$EMAIL" --password "$CREATE_OWNER_PASSWORD"