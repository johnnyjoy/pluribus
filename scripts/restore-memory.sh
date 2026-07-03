#!/usr/bin/env bash
# Restore a Pluribus memory database from a pg_dump custom-format backup.
# DESTRUCTIVE: overwrites current database contents. Stop the control plane first.
#
# Usage:
#   PLURIBUS_DB_DSN=postgres://user:pass@host:5432/controlplane?sslmode=disable \
#     ./scripts/restore-memory.sh --dump PATH --yes
set -euo pipefail

DSN="${PLURIBUS_DB_DSN:-}"
DUMP=""
CONFIRM=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dump) DUMP="$2"; shift 2 ;;
    --dsn) DSN="$2"; shift 2 ;;
    --yes) CONFIRM=1; shift ;;
    --help|-h)
      grep '^#' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "Unknown option: $1" >&2; exit 2 ;;
  esac
done

[[ -n "$DSN" ]] || { echo "FAIL: set PLURIBUS_DB_DSN or pass --dsn" >&2; exit 2; }
[[ -n "$DUMP" && -f "$DUMP" ]] || { echo "FAIL: pass --dump PATH to an existing backup file" >&2; exit 2; }
command -v pg_restore >/dev/null || { echo "FAIL: pg_restore not found" >&2; exit 2; }

pg_restore -l "$DUMP" >/dev/null || { echo "FAIL: dump unreadable: $DUMP" >&2; exit 2; }

if [[ "$CONFIRM" -ne 1 ]]; then
  echo "Refusing to restore without --yes (this OVERWRITES the current database)." >&2
  exit 2
fi

# --clean --if-exists drops objects before recreating them from the dump.
pg_restore --clean --if-exists --no-owner -d "$DSN" "$DUMP"

MEM_COUNT="$(psql "$DSN" -Atc 'SELECT COUNT(*) FROM memories;' 2>/dev/null || echo 'unknown')"
echo "PASS: restore complete (memories now: $MEM_COUNT)"
