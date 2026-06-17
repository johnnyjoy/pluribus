#!/usr/bin/env bash
# Phase 1 stdio MCP adapter proof — builds pluribus-mcp and exercises JSON-RPC over stdin/stdout.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

COMPOSE="${COMPOSE:-docker compose}"
WAIT_SECS="${WAIT_SECS:-120}"
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:8123/healthz}"
READY_URL="${READY_URL:-http://127.0.0.1:8123/readyz}"
CONTROL_PLANE_URL="${CONTROL_PLANE_URL:-http://127.0.0.1:8123}"
BIN="${BIN:-/tmp/pluribus-mcp-stdio-proof}"

COMPOSE_FILES=(-f docker-compose.yml)
if [[ -f docker-compose.host-conflicts.override.yml ]]; then
  COMPOSE_FILES+=(-f docker-compose.host-conflicts.override.yml)
elif [[ -f docker-compose.override.yml ]]; then
  COMPOSE_FILES+=(-f docker-compose.override.yml)
fi

log() { printf '[proof-mcp-stdio] %s\n' "$*"; }

wait_ready() {
  local deadline=$((SECONDS + WAIT_SECS))
  while (( SECONDS < deadline )); do
    if curl -fsS "$HEALTH_URL" >/dev/null 2>&1 && curl -fsS "$READY_URL" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  return 1
}

STARTED_STACK=0
if ! curl -fsS "$HEALTH_URL" >/dev/null 2>&1; then
  log "Starting Docker Compose stack for stdio proof..."
  $COMPOSE "${COMPOSE_FILES[@]}" up -d --build
  STARTED_STACK=1
  trap '$COMPOSE "${COMPOSE_FILES[@]}" down --remove-orphans >/dev/null 2>&1 || true' EXIT
fi

if wait_ready; then
  log "control-plane ready at $CONTROL_PLANE_URL"
else
  log "control-plane not ready"
  exit 1
fi

log "Building pluribus-mcp..."
(cd control-plane && go build -trimpath -ldflags="-s -w" -o "$BIN" ./cmd/pluribus-mcp)

log "Building and running mcp-stdio-proof helper..."
(cd control-plane && go run ./cmd/mcp-stdio-proof --binary "$BIN" --url "$CONTROL_PLANE_URL")

if (( STARTED_STACK )); then
  $COMPOSE "${COMPOSE_FILES[@]}" down --remove-orphans >/dev/null 2>&1 || true
fi
