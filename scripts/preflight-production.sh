#!/bin/sh
set -eu

PROJECT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
ENV_FILE=${ENV_FILE:-$PROJECT_DIR/.env.production}
LIVE_CHECK=${LIVE_CHECK:-0}

fail() { printf 'FAIL: %s\n' "$1" >&2; exit 1; }
pass() { printf 'PASS: %s\n' "$1"; }
require_file() { [ -s "$1" ] || fail "$2 tidak ada atau kosong: $1"; }

for command in docker openssl curl; do
  command -v "$command" >/dev/null 2>&1 || fail "command $command tidak tersedia"
done
require_file "$ENV_FILE" "environment production"

set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a

[ -n "${APP_DOMAIN:-}" ] || fail "APP_DOMAIN wajib diisi"
case "$APP_DOMAIN" in localhost|*.example.com|example.com) fail "APP_DOMAIN masih placeholder" ;; esac
[ -n "${APP_RELEASE:-}" ] && [ "$APP_RELEASE" != unknown ] || fail "APP_RELEASE wajib berupa versi immutable"

resolve_path() {
  case "$1" in /*) printf '%s' "$1" ;; *) printf '%s/%s' "$PROJECT_DIR" "$1" ;; esac
}

POSTGRES_PASSWORD_PATH=$(resolve_path "${POSTGRES_PASSWORD_FILE:-./secrets/postgres_password.txt}")
DATABASE_URL_PATH=$(resolve_path "${DATABASE_URL_FILE:-./secrets/database_url.txt}")
REDIS_PASSWORD_PATH=$(resolve_path "${REDIS_PASSWORD_FILE:-./secrets/redis_password.txt}")
JWT_SECRET_PATH=$(resolve_path "${JWT_SECRET_FILE:-./secrets/jwt_secret.txt}")
WEBHOOK_SECRET_PATH=$(resolve_path "${PAYMENT_WEBHOOK_SECRET_FILE:-./secrets/payment_webhook_secret.txt}")
XENDIT_SECRET_PATH=$(resolve_path "${XENDIT_SECRET_KEY_FILE:-./secrets/xendit_secret_key.txt}")
SENTRY_DSN_PATH=$(resolve_path "${SENTRY_DSN_FILE:-./secrets/sentry_dsn.txt}")
SMTP_PASSWORD_PATH=$(resolve_path "${SMTP_PASSWORD_FILE:-./secrets/smtp_password.txt}")
TLS_CERT_PATH=$(resolve_path "${TLS_CERT_FILE:-./deploy/nginx/certs/fullchain.pem}")
TLS_KEY_PATH=$(resolve_path "${TLS_KEY_FILE:-./deploy/nginx/certs/privkey.pem}")

require_file "$POSTGRES_PASSWORD_PATH" "PostgreSQL password"
require_file "$DATABASE_URL_PATH" "database URL"
require_file "$REDIS_PASSWORD_PATH" "Redis password"
require_file "$JWT_SECRET_PATH" "JWT secret"
require_file "$WEBHOOK_SECRET_PATH" "Xendit webhook token"
require_file "$XENDIT_SECRET_PATH" "Xendit API key"
require_file "$SENTRY_DSN_PATH" "Sentry DSN"
require_file "$SMTP_PASSWORD_PATH" "SMTP password"
require_file "$TLS_CERT_PATH" "TLS certificate"
require_file "$TLS_KEY_PATH" "TLS private key"

[ "$(tr -d '\r\n' < "$JWT_SECRET_PATH" | wc -c)" -ge 32 ] || fail "JWT secret minimal 32 karakter"
[ "$(tr -d '\r\n' < "$WEBHOOK_SECRET_PATH" | wc -c)" -ge 32 ] || fail "webhook token minimal 32 karakter"
case "$(tr -d '\r\n' < "$XENDIT_SECRET_PATH")" in xnd_*) : ;; *) fail "format Xendit API key tidak valid" ;; esac

DATABASE_URL=$(tr -d '\r\n' < "$DATABASE_URL_PATH")
case "$DATABASE_URL" in
  *@postgres:5432/*) printf '%s' "$DATABASE_URL" | grep -q 'sslmode=disable' || fail "PostgreSQL internal Compose harus eksplisit sslmode=disable" ;;
  *) printf '%s' "$DATABASE_URL" | grep -q 'sslmode=verify-full' || fail "database eksternal wajib sslmode=verify-full" ;;
esac

openssl x509 -in "$TLS_CERT_PATH" -noout -checkend 1209600 >/dev/null || fail "sertifikat TLS kedaluwarsa dalam kurang dari 14 hari"
CERT_KEY=$(openssl x509 -in "$TLS_CERT_PATH" -pubkey -noout | openssl pkey -pubin -outform DER 2>/dev/null | openssl dgst -sha256)
PRIVATE_KEY=$(openssl pkey -in "$TLS_KEY_PATH" -pubout -outform DER 2>/dev/null | openssl dgst -sha256)
[ "$CERT_KEY" = "$PRIVATE_KEY" ] || fail "sertifikat TLS dan private key tidak cocok"

docker compose --env-file "$ENV_FILE" -f "$PROJECT_DIR/docker-compose.prod.yml" config --quiet
pass "secret, TLS, database URL, dan Compose production valid"

if [ "$LIVE_CHECK" = 1 ]; then
  SMOKE_DIR=$(mktemp -d "${TMPDIR:-/tmp}/tka-smoke.XXXXXX")
  trap 'rm -rf "$SMOKE_DIR"' EXIT INT TERM
  curl --fail --silent --show-error --max-time 10 -D "$SMOKE_DIR/root.headers" "https://$APP_DOMAIN/" -o "$SMOKE_DIR/root.html"
  curl --fail --silent --show-error --max-time 10 "https://$APP_DOMAIN/privacy" -o "$SMOKE_DIR/privacy.html"
  curl --fail --silent --show-error --max-time 10 "https://$APP_DOMAIN/terms" -o "$SMOKE_DIR/terms.html"
  curl --fail --silent --show-error --max-time 10 "https://$APP_DOMAIN/api/v1/packages" -o "$SMOKE_DIR/packages.json"
  grep -qi '^strict-transport-security:' "$SMOKE_DIR/root.headers" || fail "HSTS tidak ditemukan pada respons publik"
  grep -qi '^content-security-policy:' "$SMOKE_DIR/root.headers" || fail "CSP tidak ditemukan pada respons publik"
  grep -q 'Kebijakan Privasi' "$SMOKE_DIR/privacy.html" || fail "halaman privasi tidak valid"
  grep -q 'Syarat Penggunaan' "$SMOKE_DIR/terms.html" || fail "halaman syarat tidak valid"
  grep -q '"data"' "$SMOKE_DIR/packages.json" || fail "respons katalog API tidak valid"
  pass "public web, legal pages, security headers, dan API smoke check"
else
  printf 'SKIP: live smoke check (jalankan dengan LIVE_CHECK=1 setelah deploy staging)\n'
fi
