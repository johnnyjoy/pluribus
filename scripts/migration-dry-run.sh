#!/usr/bin/env bash
# Disposable Postgres migration dry-run — does NOT touch the user's local server.
# Writes artifacts/local-upgrade-migration-dry-run.json on success.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ARTIFACT="${ROOT}/artifacts/local-upgrade-migration-dry-run.json"
IMAGE="${INTEGRATION_PG_IMAGE:-pgvector/pgvector:pg18}"
mkdir -p "${ROOT}/artifacts"

cleanup() {
  [[ -n "${CID:-}" ]] && docker rm -f "$CID" >/dev/null 2>&1 || true
}
trap cleanup EXIT

if [[ "${1:-}" == "--help" ]]; then
  cat <<'EOF'
migration-dry-run.sh — apply embedded SQL to ephemeral Postgres and verify Phase 11 tables.

Does not connect to your local Pluribus server. Safe to run anytime.

Environment:
  INTEGRATION_PG_IMAGE  Postgres image (default pgvector/pgvector:pg18)
  INTEGRATION_PG_PORT   Optional fixed host port

Output:
  artifacts/local-upgrade-migration-dry-run.json
EOF
  exit 0
fi

echo "migration-dry-run: starting ephemeral Postgres..."
if [[ -n "${INTEGRATION_PG_PORT:-}" ]]; then
  PUBLISH=(-p "127.0.0.1:${INTEGRATION_PG_PORT}:5432")
else
  PUBLISH=(-p "127.0.0.1::5432")
fi

CID="$(docker run -d \
  -e POSTGRES_USER=controlplane \
  -e POSTGRES_PASSWORD=controlplane \
  -e POSTGRES_DB=controlplane \
  "${PUBLISH[@]}" \
  "$IMAGE")"

if [[ -n "${INTEGRATION_PG_PORT:-}" ]]; then
  PORT="${INTEGRATION_PG_PORT}"
else
  PORT="$(docker port "$CID" 5432 | head -1 | awk -F: '{print $NF}')"
fi

for _ in $(seq 1 45); do
  docker exec "$CID" pg_isready -U controlplane -d controlplane >/dev/null 2>&1 && break
  sleep 1
done
docker exec "$CID" pg_isready -U controlplane -d controlplane >/dev/null 2>&1 || {
  echo "migration-dry-run: Postgres not ready" >&2
  exit 1
}

export TEST_PG_DSN="postgres://controlplane:controlplane@127.0.0.1:${PORT}/controlplane?sslmode=disable"
export SCHEMA_PROOF_JSON="$ARTIFACT"

cd "${ROOT}/control-plane"
go run ./cmd/schema-proof
echo "migration-dry-run: wrote ${ARTIFACT}"
