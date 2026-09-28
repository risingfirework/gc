#!/bin/sh
set -eu

# Bootstrap VPS sekali jalan untuk TKA production (Ubuntu, user non-root dengan sudo).
# Pemakaian: APP_DOMAIN=tka.domain.id [GIT_URL=https://github.com/risingfirework/gc.git] sh scripts/bootstrap-vps.sh
# Asumsi: DNS A record APP_DOMAIN sudah menunjuk ke public IP, port 80/443 terbuka.

APP_DOMAIN=${APP_DOMAIN:?set APP_DOMAIN, mis. tka.domain.id}
GIT_URL=${GIT_URL:-https://github.com/risingfirework/gc.git}
REPO_DIR=${REPO_DIR:-"$HOME/tka"}
EMAIL=${LE_EMAIL:-}

run() { printf '\n==> %s\n' "$*"; "$@"; }

if [ "$(id -u)" -eq 0 ]; then SUDO=; else SUDO=sudo; fi

run $SUDO apt-get update
if ! command -v docker >/dev/null 2>&1; then
  $SUDO mkdir -p /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg | $SUDO gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | $SUDO tee /etc/apt/sources.list.d/docker.list >/dev/null
  $SUDO apt-get update
  $SUDO apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
fi
$SUDO usermod -aG docker "$USER" || true

if ! command -v certbot >/dev/null 2>&1; then
  $SUDO apt-get install -y certbot
fi

[ -d "$REPO_DIR/.git" ] || run git clone "$GIT_URL" "$REPO_DIR"
cd "$REPO_DIR"
git fetch --force --prune origin
git checkout main
git pull --ff-only origin main

mkdir -p secrets deploy/nginx/certs

if [ ! -s secrets/postgres_password.txt ]; then
  gen() { openssl rand -base64 48 | tr -d '\n'; }
  PG=$(openssl rand -base64 48 | tr -dc 'A-Za-z0-9' | head -c 40)
  printf '%s' "$PG" > secrets/postgres_password.txt
  printf '%s' "$(openssl rand -base64 48 | tr -dc 'A-Za-z0-9' | head -c 40)" > secrets/redis_password.txt
  printf '%s' "$(openssl rand -base64 48 | tr -dc 'A-Za-z0-9' | head -c 48)" > secrets/jwt_secret.txt
  printf '%s' "$(openssl rand -base64 48 | tr -dc 'A-Za-z0-9' | head -c 48)" > secrets/payment_webhook_secret.txt
  printf 'postgres://tka:%s@postgres:5432/tka?sslmode=disable' "$PG" > secrets/database_url.txt
  : > secrets/xendit_secret_key.txt
  : > secrets/sentry_dsn.txt
  : > secrets/smtp_password.txt
fi

if [ ! -f .env.production ]; then
  sed \
    -e "s/^APP_DOMAIN=.*/APP_DOMAIN=$APP_DOMAIN/" \
    -e "s/^APP_RELEASE=.*/APP_RELEASE=${APP_RELEASE:-$(date +%Y.%m.%d)-1}/" \
    -e 's/^SIMPKB_CHECK_URL=.*/SIMPKB_CHECK_URL=http:\/\/simpkb-check:8080/' \
    .env.production.example > .env.production
fi

if [ ! -s deploy/nginx/certs/fullchain.pem ] || [ ! -s deploy/nginx/certs/privkey.pem ]; then
  LE_ARGS="certonly --standalone -d $APP_DOMAIN --agree-tos -m ${EMAIL:-admin@${APP_DOMAIN}}"
  # shellcheck disable=SC2086
  run $SUDO certbot $LE_ARGS
  run $SUDO cp "/etc/letsencrypt/live/$APP_DOMAIN/fullchain.pem" deploy/nginx/certs/fullchain.pem
  run $SUDO cp "/etc/letsencrypt/live/$APP_DOMAIN/privkey.pem" deploy/nginx/certs/privkey.pem
  run $SUDO chown "$(id -u):$(id -g)" deploy/nginx/certs/fullchain.pem deploy/nginx/certs/privkey.pem
fi

echo
echo "Grup docker baru aktif setelah logout/login. Tanpa login ulang, perintah"
echo "docker berikutnya harus diawali 'sudo'."

echo
echo "Sekarang ISI nilai nyata di:"
echo "  $REPO_DIR/secrets/xendit_secret_key.txt"
echo "  $REPO_DIR/secrets/sentry_dsn.txt"
echo "  $REPO_DIR/secrets/smtp_password.txt"
echo "  $REPO_DIR/.env.production (cek SMTP_*, XENDIT_SUCCESS_REDIRECT_URL)"
echo "lalu jalankan:  sh scripts/preflight-production.sh"
echo "Jangan lanjut ke up sebelum preflight PASS."