#!/usr/bin/env bash
# Logical backup of the Pluribus memory database (pg_dump custom format).
# The database is the hive mind's long-term memory: back it up before every
# upgrade and on a schedule (cron/systemd timer).
#
# Usage:
#   PLURIBUS_DB_DSN=postgres://user:pass@host:5432/controlplane?sslmode=disable \
#     ./scripts/backup-memory.sh [--out-dir DIR]
set -euo pipefail

OUT_DIR="${PLURIBUS_BACKUP_DIR:-./backups}"
DSN="${PLURIBUS_DB_DSN:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --out-dir) OUT_DIR="$2"; shift 2 ;;
    --dsn) DSN="$2"; shift 2 ;;
    --help|-h)
      grep '^#' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "Unknown option: $1" >&2; exit 2 ;;
  esac
done

[[ -n "$DSN" ]] || { echo "FAIL: set PLURIBUS_DB_DSN or pass --dsn" >&2; exit 2; }
command -v pg_dump >/dev/null || { echo "FAIL: pg_dump not found" >&2; exit 2; }
command -v pg_restore >/dev/null || { echo "FAIL: pg_restore not found" >&2; exit 2; }

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$OUT_DIR"
DUMP="$OUT_DIR/controlplane-$STAMP.dump"

MEM_COUNT="$(psql "$DSN" -Atc 'SELECT COUNT(*) FROM memories;' 2>/dev/null || echo 'unknown')"

pg_dump -Fc "$DSN" -f "$DUMP"

# Verify the dump is readable before declaring success.
pg_restore -l "$DUMP" >/dev/null

echo "PASS: backup written: $DUMP (memories at backup time: $MEM_COUNT)"
echo "$DUMP"
