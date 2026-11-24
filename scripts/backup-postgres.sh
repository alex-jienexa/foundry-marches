#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ENV_FILE:-.env.prod}"
BACKUP_DIR="${BACKUP_DIR:-backups}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"

cd "$ROOT_DIR"
if [[ ! -f "$ENV_FILE" ]]; then
  echo "Environment file not found: $ENV_FILE" >&2
  exit 1
fi
if ! [[ "$RETENTION_DAYS" =~ ^[0-9]+$ ]] || (( RETENTION_DAYS < 1 )); then
  echo "RETENTION_DAYS must be a positive integer" >&2
  exit 1
fi

umask 077
mkdir -p "$BACKUP_DIR"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
backup_file="$BACKUP_DIR/foundry_marches_$stamp.dump"
checksum_file="$backup_file.sha256"

compose=(docker compose --env-file "$ENV_FILE" -f compose.yaml -f compose.production.yaml)

echo "Creating PostgreSQL custom-format backup: $backup_file"
"${compose[@]}" exec -T postgres pg_dump -U foundry_admin -d foundry_marches -Fc > "$backup_file"

if [[ ! -s "$backup_file" ]]; then
  rm -f "$backup_file"
  echo "Backup is empty; refusing to keep it" >&2
  exit 1
fi

pg_restore --list "$backup_file" >/dev/null
sha256sum "$backup_file" > "$checksum_file"

find "$BACKUP_DIR" -maxdepth 1 -type f -name 'foundry_marches_*.dump' -mtime "+$RETENTION_DAYS" -print -delete |
while IFS= read -r expired; do
  rm -f "$expired.sha256"
done

echo "Backup completed and validated: $backup_file"
echo "Checksum: $checksum_file"
