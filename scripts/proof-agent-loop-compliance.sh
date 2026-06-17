#!/usr/bin/env bash
# Phase 2 agent-loop compliance proof — unit scenarios + optional Docker MCP exercise.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PASS=0
FAIL=0

log() { printf '[proof-agent-loop] %s\n' "$*"; }
pass() { PASS=$((PASS + 1)); log "PASS: $*"; }
fail() { FAIL=$((FAIL + 1)); log "FAIL: $*" >&2; }

log "Running compliance unit + hostile tests..."
if (cd control-plane && go test ./internal/compliance/... ./internal/mcp/ -run 'TestEvaluateSession|TestMCPTelemetry|TestAllRegisteredTools|TestProofAgentLoopScenarios' -count=1); then
  pass "Go compliance and MCP telemetry tests"
else
  fail "Go compliance and MCP telemetry tests"
fi

log "Running MCP scenario proof (in-process HTTP MCP)..."
if (cd control-plane && go test ./internal/mcp/ -run TestProofAgentLoopScenarios -count=1); then
  pass "In-process MCP loop scenarios"
else
  fail "In-process MCP loop scenarios"
fi

if [[ "${PROOF_AGENT_LOOP_SKIP_DOCKER:-}" == "1" ]]; then
  log "Skipping Docker MCP proof (PROOF_AGENT_LOOP_SKIP_DOCKER=1)"
else
  MCP_URL="${MCP_URL:-http://127.0.0.1:8123/v1/mcp}"
  HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:8123/healthz}"
  COMPOSE="${COMPOSE:-docker compose}"
  COMPOSE_FILES=(-f docker-compose.yml)
  if [[ -f docker-compose.host-conflicts.override.yml ]]; then
    COMPOSE_FILES+=(-f docker-compose.host-conflicts.override.yml)
  elif [[ -f docker-compose.override.yml ]]; then
    COMPOSE_FILES+=(-f docker-compose.override.yml)
  fi

  log "Starting Docker stack for MCP compliance smoke..."
  $COMPOSE "${COMPOSE_FILES[@]}" down --remove-orphans >/dev/null 2>&1 || true
  if $COMPOSE "${COMPOSE_FILES[@]}" up -d --build; then
    trap '$COMPOSE "${COMPOSE_FILES[@]}" down --remove-orphans >/dev/null 2>&1 || true' EXIT
    deadline=$((SECONDS + 120))
    while (( SECONDS < deadline )); do
      if curl -fsS "$HEALTH_URL" >/dev/null 2>&1; then
        break
      fi
      sleep 2
    done
    if curl -fsS "$HEALTH_URL" >/dev/null 2>&1; then
      INIT=$(curl -fsS -X POST "$MCP_URL" -H 'Content-Type: application/json' \
        -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"proof-agent-loop","version":"1"}}}')
      SID=$(echo "$INIT" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('result',{}).get('pluribus',{}).get('session_id',''))")
      if [[ -n "$SID" ]]; then
        pass "Docker MCP initialize returned session_id"
        EVAL=$(curl -fsS -X POST "http://127.0.0.1:8123/v1/compliance/evaluate" \
          -H 'Content-Type: application/json' \
          -d "{\"session_id\":\"$SID\"}")
        STATUS=$(echo "$EVAL" | python3 -c "import json,sys; print(json.load(sys.stdin).get('status',''))")
        if [[ -n "$STATUS" ]]; then
          pass "Docker compliance evaluate returned status=$STATUS"
        else
          fail "Docker compliance evaluate missing status: $EVAL"
        fi
      else
        fail "Docker MCP initialize missing pluribus.session_id: $INIT"
      fi
    else
      fail "Docker health check failed"
    fi
  else
    fail "docker compose up failed"
  fi
fi

log "Results: PASS=$PASS FAIL=$FAIL"
if (( FAIL > 0 )); then
  exit 1
fi
log "Phase 2 agent-loop compliance proof: ALL PASS"
exit 0
