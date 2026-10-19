#!/usr/bin/env bash
# Restore a gzipped mysqldump into the running mysql service.
# WARNING: overwrites the target database. Use only with explicit intent.
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <backup.sql.gz>" >&2
  exit 1
fi

BACKUP_FILE="$1"
ROOT_DIR="${ROOT_DIR:-/opt/finance}"
COMPOSE_FILE="${COMPOSE_FILE:-$ROOT_DIR/docker-compose.prod.yml}"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env.prod}"
CONTAINER="${MYSQL_CONTAINER:-$(docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" ps -q mysql | head -1)}"

set -a
source "$ENV_FILE"
set +a

echo "Restoring $BACKUP_FILE into ${DB_NAME:-income_expense} on container $CONTAINER"
gunzip -c "$BACKUP_FILE" | docker exec -i "$CONTAINER" mysql \
  -u"${DB_USER:-finance}" \
  -p"${DB_PASSWORD}" \
  "${DB_NAME:-income_expense}"

echo "restore complete"
