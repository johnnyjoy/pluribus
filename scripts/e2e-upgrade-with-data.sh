#!/usr/bin/env bash
# E2E durability gate (hostile-audit C1): memories MUST survive an in-place upgrade.
#
# What it does, fully self-contained:
#   1. Starts an ephemeral pgvector Postgres container on a random port.
#   2. Builds the controlplane binary and "installs" it into a temp dir.
#   3. Starts the server against the ephemeral DB, seeds N memories via REST.
#   4. Runs scripts/upgrade-in-place.sh with a freshly built "new" binary
#      against the SAME database.
#   5. Asserts every seeded memory is still readable afterwards.
#   6. Cleans up container, processes, and temp files.
#
# Usage:
#   ./scripts/e2e-upgrade-with-data.sh [--seed-count 25] [--keep] [--pg-image IMAGE]
#
# The Postgres image major version must not exceed the host pg_dump version
# (backup-memory.sh uses the host client tools). Default: pgvector pg16.
#
# Requires: docker, go, psql, curl, jq.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

SEED_COUNT=25
KEEP=0
PG_IMAGE="pgvector/pgvector:pg16"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --seed-count) SEED_COUNT="$2"; shift 2 ;;
    --keep) KEEP=1; shift ;;
    --pg-image) PG_IMAGE="$2"; shift 2 ;;
    --help|-h) grep '^#' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "Unknown option: $1" >&2; exit 2 ;;
  esac
done

for tool in docker go psql curl jq; do
  command -v "$tool" >/dev/null || { echo "FAIL: $tool not found" >&2; exit 1; }
done

WORK="$(mktemp -d /tmp/pluribus-upgrade-e2e.XXXXXX)"
PG_CONTAINER="pluribus-upgrade-e2e-$$"
PG_PORT="$(( 20000 + RANDOM % 20000 ))"
HTTP_PORT="$(( 20000 + RANDOM % 20000 ))"
BASE_URL="http://127.0.0.1:${HTTP_PORT}"
DSN="postgres://controlplane:controlplane@127.0.0.1:${PG_PORT}/controlplane?sslmode=disable"

cleanup() {
  local code=$?
  pkill -f "^$WORK/install/controlplane" 2>/dev/null || true
  docker rm -f "$PG_CONTAINER" >/dev/null 2>&1 || true
  if [[ "$KEEP" -ne 1 ]]; then rm -rf "$WORK"; else echo "kept workdir: $WORK"; fi
  exit $code
}
trap cleanup EXIT

fail() { echo "FAIL: $*" >&2; exit 1; }

echo "== 1. Ephemeral Postgres (port $PG_PORT)"
docker run -d --name "$PG_CONTAINER" \
  -e POSTGRES_DB=controlplane -e POSTGRES_USER=controlplane -e POSTGRES_PASSWORD=controlplane \
  -p "${PG_PORT}:5432" "$PG_IMAGE" >/dev/null
for i in $(seq 1 30); do
  if psql "$DSN" -Atc 'SELECT 1' >/dev/null 2>&1; then break; fi
  sleep 1
  [[ $i -lt 30 ]] || fail "postgres did not become ready"
done

echo "== 2. Build + install binary"
( cd "$REPO_ROOT/control-plane" && go build -o "$WORK/controlplane-v1" ./cmd/controlplane )
mkdir -p "$WORK/install" "$WORK/backups"
cp "$WORK/controlplane-v1" "$WORK/install/controlplane"

cat > "$WORK/config.yaml" <<EOF
server:
  bind: "127.0.0.1:${HTTP_PORT}"
postgres:
  dsn: "${DSN}"
EOF

# Anchored pattern: match only the server process itself, not scripts that
# carry the install path in their argument list (e.g. upgrade-in-place.sh).
start_server_cmd="CONFIG=$WORK/config.yaml nohup $WORK/install/controlplane >> $WORK/server.log 2>&1 & sleep 1"
stop_server_cmd="pkill -f '^$WORK/install/controlplane' || true; sleep 1"

echo "== 3. Start server + seed ${SEED_COUNT} memories"
bash -c "$start_server_cmd"
for i in $(seq 1 30); do
  if curl -sS -m 2 "$BASE_URL/healthz" 2>/dev/null | grep -q ok; then break; fi
  sleep 1
  [[ $i -lt 30 ]] || { tail -n 40 "$WORK/server.log" >&2 || true; fail "server did not become healthy"; }
done

SEEDED_IDS=()
for i in $(seq 1 "$SEED_COUNT"); do
  resp="$(curl -sS -X POST "$BASE_URL/v1/memory/" -H 'Content-Type: application/json' \
    -d "{\"kind\":\"decision\",\"statement\":\"upgrade-e2e seeded memory number $i must survive the in-place upgrade\",\"authority\":5,\"tags\":[\"upgrade-e2e\"]}")"
  id="$(echo "$resp" | jq -r '.id // empty')"
  [[ -n "$id" ]] || fail "seed $i failed: $resp"
  SEEDED_IDS+=("$id")
done
PRE_COUNT="$(psql "$DSN" -Atc 'SELECT COUNT(*) FROM memories;')"
echo "seeded ${#SEEDED_IDS[@]} memories (table count: $PRE_COUNT)"

echo "== 4. In-place upgrade with a new build"
( cd "$REPO_ROOT/control-plane" && go build -o "$WORK/controlplane-v2" ./cmd/controlplane )
PLURIBUS_DB_DSN="$DSN" "$SCRIPT_DIR/upgrade-in-place.sh" \
  --new-binary "$WORK/controlplane-v2" \
  --install-path "$WORK/install/controlplane" \
  --base-url "$BASE_URL" \
  --backup-dir "$WORK/backups" \
  --stop-cmd "$stop_server_cmd" \
  --start-cmd "$start_server_cmd" \
  --skip-smoke

echo "== 5. Assert every seeded memory survived"
POST_COUNT="$(psql "$DSN" -Atc 'SELECT COUNT(*) FROM memories;')"
(( POST_COUNT >= PRE_COUNT )) || fail "memory count shrank: $PRE_COUNT -> $POST_COUNT"
missing=0
for id in "${SEEDED_IDS[@]}"; do
  got="$(psql "$DSN" -Atc "SELECT COUNT(*) FROM memories WHERE id = '$id';")"
  if [[ "$got" != "1" ]]; then
    echo "MISSING after upgrade: $id" >&2
    missing=$((missing + 1))
  fi
done
(( missing == 0 )) || fail "$missing seeded memories missing after upgrade"

# API-level sanity: the upgraded server can still search the seeded rows
# (direct-create decisions land pending under the formation gate).
FOUND=0
for st in pending active; do
  n="$(curl -sS -X POST "$BASE_URL/v1/memory/search" -H 'Content-Type: application/json' \
    -d "{\"tags\":[\"upgrade-e2e\"],\"status\":\"$st\",\"max\":100}" | jq 'length' 2>/dev/null || echo 0)"
  FOUND=$(( FOUND + n ))
done
(( FOUND >= 1 )) || fail "upgraded server returned no seeded memories via search"

echo "PASS: all ${#SEEDED_IDS[@]} seeded memories survived the in-place upgrade ($PRE_COUNT -> $POST_COUNT)"
