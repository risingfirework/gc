#!/bin/sh
# Mengonfigurasi SMTP untuk email reset kata sandi pada instalasi production.
#
# Pemakaian (jalankan di VPS, dari root repo):
#
#   SMTP_HOST=smtp.resend.com \
#   SMTP_USERNAME=resend \
#   SMTP_FROM=no-reply@siap-siap.my.id \
#   sh scripts/configure-smtp.sh
#
# Skrip akan meminta password/API key (input tersembunyi), menulisnya ke file
# secret, memperbarui .env.production (dengan backup), lalu me-restart layanan
# api. Nilai yang sudah ada tetap dipakai bila variabel tidak diberikan.
set -eu

PROJECT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
ENV_FILE=${ENV_FILE:-$PROJECT_DIR/.env.production}

fail() { printf 'GAGAL: %s\n' "$1" >&2; exit 1; }
info() { printf '%s\n' "$1"; }

[ -f "$ENV_FILE" ] || fail "environment production tidak ditemukan: $ENV_FILE"

# Nilai lama dipakai sebagai default agar skrip bisa dijalankan ulang.
env_value() {
  sed -n "s|^$1=||p" "$ENV_FILE" | tail -n 1
}

SMTP_HOST=${SMTP_HOST:-$(env_value SMTP_HOST)}
SMTP_PORT=${SMTP_PORT:-$(env_value SMTP_PORT)}
SMTP_USERNAME=${SMTP_USERNAME:-$(env_value SMTP_USERNAME)}
SMTP_FROM=${SMTP_FROM:-$(env_value SMTP_FROM)}
SMTP_PASSWORD_FILE=${SMTP_PASSWORD_FILE:-$(env_value SMTP_PASSWORD_FILE)}

SMTP_PORT=${SMTP_PORT:-587}
SMTP_PASSWORD_FILE=${SMTP_PASSWORD_FILE:-./secrets/smtp_password.txt}

[ -n "$SMTP_HOST" ] || fail "SMTP_HOST kosong. Contoh Resend: smtp.resend.com"
[ -n "$SMTP_FROM" ] || fail "SMTP_FROM kosong. Contoh: no-reply@siap-siap.my.id"
case "$SMTP_HOST" in
  mail.example.com|example.com|smtp.example.com) fail "SMTP_HOST masih placeholder: $SMTP_HOST" ;;
esac

case "$SMTP_PASSWORD_FILE" in
  /*) PASS_PATH=$SMTP_PASSWORD_FILE ;;
  *) PASS_PATH=$PROJECT_DIR/$SMTP_PASSWORD_FILE ;;
esac

if [ -z "${SMTP_PASSWORD:-}" ]; then
  printf 'Password/API key SMTP (tidak ditampilkan): '
  if stty -echo 2>/dev/null; then
    read -r SMTP_PASSWORD
    stty echo
    printf '\n'
  else
    read -r SMTP_PASSWORD
  fi
fi
[ -n "$SMTP_PASSWORD" ] || fail "password SMTP kosong"

if [ "$(printf '%s' "$SMTP_PASSWORD" | wc -l | tr -d ' ')" -ne 0 ]; then
  fail "password SMTP tidak boleh berisi baris baru"
fi

umask 077
mkdir -p -- "$(dirname -- "$PASS_PATH")"
printf '%s' "$SMTP_PASSWORD" > "$PASS_PATH"
chmod 600 "$PASS_PATH" 2>/dev/null || true
info "PASS: kredensial SMTP disimpan di $PASS_PATH"

BACKUP="$ENV_FILE.bak.$(date +%Y%m%d%H%M%S)"
cp -- "$ENV_FILE" "$BACKUP"
info "PASS: backup .env.production -> $BACKUP"

upsert() {
  key=$1
  value=$2
  escaped=$(printf '%s' "$value" | sed 's/[&|\\]/\\&/g')
  if grep -q "^${key}=" "$ENV_FILE"; then
    sed -i "s|^${key}=.*|${key}=${escaped}|" "$ENV_FILE"
  else
    printf '%s=%s\n' "$key" "$value" >> "$ENV_FILE"
  fi
}

upsert SMTP_HOST "$SMTP_HOST"
upsert SMTP_PORT "$SMTP_PORT"
upsert SMTP_USERNAME "$SMTP_USERNAME"
upsert SMTP_FROM "$SMTP_FROM"
upsert SMTP_PASSWORD_FILE "$SMTP_PASSWORD_FILE"
info "PASS: .env.production diperbarui (SMTP_HOST=$SMTP_HOST, SMTP_PORT=$SMTP_PORT)"

if ! command -v docker >/dev/null 2>&1; then
  info "docker tidak ditemukan; lewati restart. Jalankan manual:"
  info "  docker compose --env-file .env.production -f docker-compose.prod.yml up -d api"
  exit 0
fi

info "Me-restart layanan api..."
docker compose --env-file "$ENV_FILE" -f "$PROJECT_DIR/docker-compose.prod.yml" up -d api

info ""
info "Selesai. Cek log startup (harus ada 'email reset kata sandi aktif'):"
info "  docker compose --env-file .env.production -f docker-compose.prod.yml logs --tail=40 api"
info ""
info "Uji dari mesin lain (202 = email terkirim, 502 = gagal kirim, 503 = SMTP belum aktif):"
info "  curl -s -o /dev/null -w '%{http_code}\\n' -X POST https://<domain>/api/v1/auth/forgot-password \\"
info "    -H 'Content-Type: application/json' -d '{\"email\":\"email-akun-anda@contoh.com\"}'"
