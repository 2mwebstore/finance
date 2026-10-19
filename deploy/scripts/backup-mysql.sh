#!/usr/bin/env bash
# Daily MySQL backup. Run on the host or a sidecar with access to the mysql container.
# Example cron: 0 2 * * * /opt/finance/deploy/scripts/backup-mysql.sh
set -euo pipefail

ROOT_DIR="${ROOT_DIR:-/opt/finance}"
BACKUP_DIR="${BACKUP_DIR:-/var/backups/finance-mysql}"
RETENTION_DAYS="${RETENTION_DAYS:-14}"
COMPOSE_FILE="${COMPOSE_FILE:-$ROOT_DIR/docker-compose.prod.yml}"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env.prod}"
CONTAINER="${MYSQL_CONTAINER:-$(docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" ps -q mysql 2>/dev/null | head -1)}"

mkdir -p "$BACKUP_DIR"
STAMP=$(date -u +%Y%m%dT%H%M%SZ)
OUT="$BACKUP_DIR/income_expense_${STAMP}.sql.gz"

if [[ -z "${CONTAINER}" ]]; then
  echo "mysql container not found" >&2
  exit 1
fi

# shellcheck disable=SC1090
set -a
# Load only DB vars if possible
source "$ENV_FILE"
set +a

docker exec "$CONTAINER" mysqldump \
  -u"${DB_USER:-finance}" \
  -p"${DB_PASSWORD}" \
  --single-transaction \
  --routines \
  --triggers \
  "${DB_NAME:-income_expense}" \
  | gzip -c > "$OUT"

# Optional off-server copy (configure RSYNC_TARGET)
if [[ -n "${RSYNC_TARGET:-}" ]]; then
  rsync -az "$OUT" "$RSYNC_TARGET/"
fi

# Retention
find "$BACKUP_DIR" -name 'income_expense_*.sql.gz' -mtime +"${RETENTION_DAYS}" -delete

echo "backup written: $OUT"
ls -lh "$OUT"
