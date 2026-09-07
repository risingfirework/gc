#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PROJECT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
PROD_ENV_FILE=${PROD_ENV_FILE:-$PROJECT_DIR/.env.production}
RELEASE_IMAGE=${RELEASE_IMAGE:?set an immutable backend image tag or digest}
RELEASE_TAG=${RELEASE_TAG:?set release identifier for Sentry}
DEPLOY_HEALTH_TIMEOUT_SECONDS=${DEPLOY_HEALTH_TIMEOUT_SECONDS:-120}
DEPLOY_DRAIN_SECONDS=${DEPLOY_DRAIN_SECONDS:-30}
TARGET_FILE="$PROJECT_DIR/deploy/nginx/runtime/backend-target.conf"

case "$RELEASE_IMAGE" in
  *:latest|latest) echo "Refusing mutable latest tag" >&2; exit 1 ;;
esac

compose() {
  RELEASE_IMAGE="$RELEASE_IMAGE" RELEASE_TAG="$RELEASE_TAG" docker compose \
    --env-file "$PROD_ENV_FILE" \
    -f "$PROJECT_DIR/docker-compose.prod.yml" \
    -f "$PROJECT_DIR/docker-compose.bluegreen.yml" "$@"
}

active=$(sed -n 's/^server \([^:]*\):8080.*$/\1/p' "$TARGET_FILE")
case "$active" in
  backend-blue) candidate=backend-green ;;
  backend-green|backend) candidate=backend-blue ;;
  *) echo "Unknown active backend: $active" >&2; exit 1 ;;
esac

echo "Deploying $RELEASE_IMAGE to $candidate (active: $active)"
docker pull "$RELEASE_IMAGE"

# Only expand/backward-compatible migrations are allowed before traffic switches.
docker compose --env-file "$PROD_ENV_FILE" -f "$PROJECT_DIR/docker-compose.prod.yml" \
  run --rm migrate
compose up -d --no-deps --no-build "$candidate"

container_id=$(compose ps -q "$candidate")
[ -n "$container_id" ] || { echo "Candidate container was not created" >&2; exit 1; }
deadline=$(( $(date +%s) + DEPLOY_HEALTH_TIMEOUT_SECONDS ))
while :; do
  health=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$container_id")
  [ "$health" = "healthy" ] && break
  [ "$health" != "unhealthy" ] || { compose logs "$candidate"; exit 1; }
  [ "$(date +%s)" -lt "$deadline" ] || { compose logs "$candidate"; exit 1; }
  sleep 2
done
compose exec -T "$candidate" /tka-api healthcheck

previous_file="${TARGET_FILE}.previous"
cp "$TARGET_FILE" "$previous_file"
target_tmp="${TARGET_FILE}.tmp.$$"
printf 'server %s:8080 resolve;\n' "$candidate" > "$target_tmp"
mv "$target_tmp" "$TARGET_FILE"
if ! docker compose --env-file "$PROD_ENV_FILE" -f "$PROJECT_DIR/docker-compose.prod.yml" exec -T nginx nginx -t; then
  mv "$previous_file" "$TARGET_FILE"
  exit 1
fi
docker compose --env-file "$PROD_ENV_FILE" -f "$PROJECT_DIR/docker-compose.prod.yml" exec -T nginx nginx -s reload
sleep "$DEPLOY_DRAIN_SECONDS"

case "$active" in
  backend|backend-blue|backend-green) compose stop "$active" ;;
esac
rm -f "$previous_file"
echo "Deployment complete. Active backend: $candidate ($RELEASE_TAG)"
