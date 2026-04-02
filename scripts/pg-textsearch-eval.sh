#!/usr/bin/env bash
# One-shot pg_textsearch evaluation: ephemeral Postgres (pg18 + pg_textsearch) + migrate + seed + reindex + query suite + artifacts.
#
# From repo root:
#   ./scripts/pg-textsearch-eval.sh
#
# Requires: Docker, Go 1.22+, built image pluribus-postgres-pg-textsearch:local (built automatically).
# Optional: PG_TEXTSEARCH_EVAL_PORT=15432 to pin host port; otherwise an ephemeral port is chosen.
#
# Does not fetch runtime data from the network (Postgres image must exist locally or be built here).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
IMAGE="${PG_TEXTSEARCH_IMAGE:-pluribus-postgres-pg-textsearch:local}"

log() { printf '[LEXICAL] %s\n' "$*"; }

cleanup() {
  if [[ -n "${CID:-}" ]]; then
    docker rm -f "$CID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

if [[ ! "$(docker images -q "$IMAGE" 2>/dev/null)" ]]; then
  log "building $IMAGE"
  (cd "$ROOT" && make pg-textsearch-image)
fi

if [[ -n "${PG_TEXTSEARCH_EVAL_PORT:-}" ]]; then
  PUBLISH=(-p "127.0.0.1:${PG_TEXTSEARCH_EVAL_PORT}:5432")
else
  PUBLISH=(-p "127.0.0.1::5432")
fi

log "starting ephemeral Postgres with pg_textsearch"
CID="$(docker run -d \
  -e POSTGRES_USER=controlplane \
  -e POSTGRES_PASSWORD=controlplane \
  -e POSTGRES_DB=controlplane \
  "${PUBLISH[@]}" \
  "$IMAGE" \
  postgres -c shared_preload_libraries=pg_textsearch)"

if [[ -n "${PG_TEXTSEARCH_EVAL_PORT:-}" ]]; then
  PORT="${PG_TEXTSEARCH_EVAL_PORT}"
else
  PORT="$(docker port "$CID" 5432 | head -1 | awk -F: '{print $NF}')"
fi

DSN="postgres://controlplane:controlplane@127.0.0.1:${PORT}/controlplane?sslmode=disable"
export PG_TEXTSEARCH_EVAL_DSN="$DSN"

for i in $(seq 1 60); do
  if docker exec "$CID" pg_isready -U controlplane -d controlplane >/dev/null 2>&1; then
    break
  fi
  if [[ "$i" -eq 60 ]]; then
    log "postgres failed to become ready"
    exit 1
  fi
  sleep 0.5
done

log "running eval (DSN host=127.0.0.1 port=${PORT})"
cd "$ROOT/control-plane"
go run ./cmd/pg-textsearch-eval -dsn="$DSN" eval

log "done — artifacts: $ROOT/artifacts/pg-textsearch/eval.json"
log "summary: $ROOT/artifacts/pg-textsearch/eval-summary.md"
