#!/usr/bin/env bash
# In-place Pluribus upgrade that treats the memory database as sacred.
#
# The database is NEVER dropped or recreated. Migrations are forward-only,
# idempotent, and applied automatically by the new binary on startup against
# the SAME database. This script:
#
#   1. Records the pre-upgrade memory count.
#   2. Takes a verified pg_dump backup (scripts/backup-memory.sh).
#   3. Stops the running server.
#   4. Swaps in the new controlplane binary (old one is kept beside it).
#   5. Starts the server and polls /healthz.
#   6. Asserts post-upgrade memory count >= pre-upgrade count.
#   7. On any failure: restores the database backup and the previous binary.
#
# Usage:
#   PLURIBUS_DB_DSN=postgres://user:pass@host:5432/controlplane?sslmode=disable \
#     ./scripts/upgrade-in-place.sh \
#       --new-binary control-plane/out/controlplane \
#       --install-path /usr/local/bin/controlplane \
#       [--base-url http://127.0.0.1:8123] \
#       [--stop-cmd 'systemctl stop pluribus'] [--start-cmd 'systemctl start pluribus'] \
#       [--backup-dir ./backups] [--skip-smoke]
#
# Without --stop-cmd/--start-cmd the script stops the process found by
# `pgrep -f <install-path>` and starts the binary with nohup (developer-style
# bare-metal deployment per INSTALL.md).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

DSN="${PLURIBUS_DB_DSN:-}"
NEW_BINARY=""
INSTALL_PATH=""
BASE_URL="${PLURIBUS_BASE_URL:-http://127.0.0.1:8123}"
STOP_CMD=""
START_CMD=""
BACKUP_DIR="${PLURIBUS_BACKUP_DIR:-./backups}"
SKIP_SMOKE=0
HEALTH_TIMEOUT="${PLURIBUS_HEALTH_TIMEOUT:-60}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --new-binary) NEW_BINARY="$2"; shift 2 ;;
    --install-path) INSTALL_PATH="$2"; shift 2 ;;
    --base-url) BASE_URL="$2"; shift 2 ;;
    --stop-cmd) STOP_CMD="$2"; shift 2 ;;
    --start-cmd) START_CMD="$2"; shift 2 ;;
    --backup-dir) BACKUP_DIR="$2"; shift 2 ;;
    --dsn) DSN="$2"; shift 2 ;;
    --skip-smoke) SKIP_SMOKE=1; shift ;;
    --help|-h) grep '^#' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "Unknown option: $1" >&2; exit 2 ;;
  esac
done

fail() { echo "FAIL: $*" >&2; exit 1; }

[[ -n "$DSN" ]] || fail "set PLURIBUS_DB_DSN or pass --dsn"
[[ -n "$NEW_BINARY" && -x "$NEW_BINARY" ]] || fail "pass --new-binary PATH to an executable new build"
[[ -n "$INSTALL_PATH" ]] || fail "pass --install-path PATH of the deployed controlplane binary"
command -v psql >/dev/null || fail "psql not found"
command -v curl >/dev/null || fail "curl not found"

BASE_URL="${BASE_URL%/}"

stop_server() {
  if [[ -n "$STOP_CMD" ]]; then
    bash -c "$STOP_CMD"
  else
    pkill -f "$INSTALL_PATH" 2>/dev/null || true
    sleep 2
  fi
}

start_server() {
  if [[ -n "$START_CMD" ]]; then
    bash -c "$START_CMD"
  else
    nohup "$INSTALL_PATH" >>"${BACKUP_DIR}/controlplane.log" 2>&1 &
  fi
}

wait_healthy() {
  local deadline=$(( $(date +%s) + HEALTH_TIMEOUT ))
  while (( $(date +%s) < deadline )); do
    if curl -sS -m 3 "${BASE_URL}/healthz" 2>/dev/null | grep -q ok; then
      return 0
    fi
    sleep 2
  done
  return 1
}

memory_count() {
  psql "$DSN" -Atc 'SELECT COUNT(*) FROM memories;' 2>/dev/null || echo "-1"
}

rollback() {
  echo "ROLLBACK: restoring previous binary and database backup" >&2
  stop_server
  if [[ -f "${INSTALL_PATH}.pre-upgrade" ]]; then
    cp -f "${INSTALL_PATH}.pre-upgrade" "$INSTALL_PATH"
  fi
  PLURIBUS_DB_DSN="$DSN" "$SCRIPT_DIR/restore-memory.sh" --dump "$DUMP_PATH" --yes || \
    echo "ROLLBACK WARNING: database restore failed; backup remains at $DUMP_PATH" >&2
  start_server
  wait_healthy || echo "ROLLBACK WARNING: old server did not become healthy" >&2
  fail "upgrade rolled back"
}

# --- 1. Pre-upgrade count -----------------------------------------------
PRE_COUNT="$(memory_count)"
[[ "$PRE_COUNT" != "-1" ]] || fail "cannot query memories table via DSN (is the DB reachable?)"
echo "pre-upgrade memory count: $PRE_COUNT"

# --- 2. Verified backup --------------------------------------------------
mkdir -p "$BACKUP_DIR"
DUMP_PATH="$(PLURIBUS_DB_DSN="$DSN" "$SCRIPT_DIR/backup-memory.sh" --out-dir "$BACKUP_DIR" | tail -n 1)"
[[ -f "$DUMP_PATH" ]] || fail "backup did not produce a dump file"
echo "backup: $DUMP_PATH"

# --- 3-4. Stop, keep old binary, swap in new one --------------------------
stop_server
if [[ -f "$INSTALL_PATH" ]]; then
  cp -f "$INSTALL_PATH" "${INSTALL_PATH}.pre-upgrade"
fi
cp -f "$NEW_BINARY" "$INSTALL_PATH"
chmod +x "$INSTALL_PATH"

# --- 5. Start and health check -------------------------------------------
start_server
wait_healthy || rollback

# --- 6. Data durability assertion ----------------------------------------
POST_COUNT="$(memory_count)"
echo "post-upgrade memory count: $POST_COUNT"
if [[ "$POST_COUNT" == "-1" ]] || (( POST_COUNT < PRE_COUNT )); then
  echo "DATA LOSS DETECTED: $PRE_COUNT -> $POST_COUNT" >&2
  rollback
fi

# --- 7. Optional smoke ----------------------------------------------------
if [[ "$SKIP_SMOKE" -ne 1 && -x "$SCRIPT_DIR/smoke/local-rest-smoke.sh" ]]; then
  "$SCRIPT_DIR/smoke/local-rest-smoke.sh" --base-url "$BASE_URL" || rollback
fi

echo "PASS: in-place upgrade complete; memories preserved ($PRE_COUNT -> $POST_COUNT); backup at $DUMP_PATH"
