#!/usr/bin/env bash
# Phase 1 authenticated MCP Docker proof — proves API-key auth on MCP while healthz/readyz stay open.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

export PLURIBUS_API_KEY="${PLURIBUS_API_KEY:-phase1-mcp-auth-proof-key}"
MCP_URL="${MCP_URL:-http://127.0.0.1:8123/v1/mcp}"
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:8123/healthz}"
READY_URL="${READY_URL:-http://127.0.0.1:8123/readyz}"
COMPOSE="${COMPOSE:-docker compose}"
WAIT_SECS="${WAIT_SECS:-120}"

COMPOSE_FILES=(-f docker-compose.yml -f docker-compose.mcp-auth-proof.yml)
if [[ -f docker-compose.host-conflicts.override.yml ]]; then
  COMPOSE_FILES+=(-f docker-compose.host-conflicts.override.yml)
elif [[ -f docker-compose.override.yml ]]; then
  COMPOSE_FILES+=(-f docker-compose.override.yml)
fi

PASS=0
FAIL=0
log() { printf '[proof-mcp-auth] %s\n' "$*"; }
pass() { PASS=$((PASS + 1)); log "PASS: $*"; }
fail() { FAIL=$((FAIL + 1)); log "FAIL: $*" >&2; }

mcp_post() {
  local body="$1"
  local api_key="${2-}"
  local args=(-fsS -X POST "$MCP_URL" -H 'Content-Type: application/json' -d "$body")
  if [[ -n "$api_key" ]]; then
    args+=(-H "X-API-Key: $api_key")
  fi
  curl "${args[@]}"
}

assert_http_code() {
  local code="$1" want="$2" label="$3"
  if [[ "$code" == "$want" ]]; then
    pass "$label (HTTP $want)"
  else
    fail "$label — expected HTTP $want, got $code"
    return 1
  fi
}

assert_no_rpc_error() {
  local resp="$1" label="$2"
  if echo "$resp" | python3 -c "import json,sys; d=json.load(sys.stdin); sys.exit(0 if d.get('error') is None else 1)"; then
    pass "$label"
  else
    fail "$label — JSON-RPC error: $resp"
    return 1
  fi
}

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

log "Starting Docker Compose stack with PLURIBUS_API_KEY..."
$COMPOSE "${COMPOSE_FILES[@]}" down --remove-orphans >/dev/null 2>&1 || true
$COMPOSE "${COMPOSE_FILES[@]}" up -d --build
trap '$COMPOSE "${COMPOSE_FILES[@]}" down --remove-orphans >/dev/null 2>&1 || true' EXIT

if wait_ready; then
  pass "healthz and readyz without API key"
else
  fail "timeout waiting for healthz/readyz"
  exit 1
fi

HEALTH_BODY="$(curl -fsS "$HEALTH_URL")"
READY_BODY="$(curl -fsS "$READY_URL")"
log "healthz (unauthenticated): $HEALTH_BODY"
log "readyz (unauthenticated): $READY_BODY"

INIT='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"proof-auth","version":"0.1.0"}}}'

# No key — MCP must fail (401)
NO_KEY_CODE="$(curl -sS -o /tmp/mcp-auth-no-key.json -w '%{http_code}' -X POST "$MCP_URL" -H 'Content-Type: application/json' -d "$INIT" || true)"
assert_http_code "$NO_KEY_CODE" "401" "MCP without API key rejected" || true

# Wrong key — MCP must fail (403)
WRONG_KEY_CODE="$(curl -sS -o /tmp/mcp-auth-wrong-key.json -w '%{http_code}' -X POST "$MCP_URL" -H 'Content-Type: application/json' -H 'X-API-Key: wrong-key' -d "$INIT" || true)"
assert_http_code "$WRONG_KEY_CODE" "403" "MCP with wrong API key rejected" || true

# Correct key via header
INIT_RESP="$(mcp_post "$INIT" "$PLURIBUS_API_KEY")" || { fail "initialize with correct key"; exit 1; }
assert_no_rpc_error "$INIT_RESP" "MCP initialize with correct X-API-Key" || true

LIST_RESP="$(mcp_post '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' "$PLURIBUS_API_KEY")" || { fail "tools/list"; exit 1; }
assert_no_rpc_error "$LIST_RESP" "MCP tools/list with correct key" || true

RECALL='{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"recall_context","arguments":{"task":"authenticated MCP proof recall"}}}'
RECALL_RESP="$(mcp_post "$RECALL" "$PLURIBUS_API_KEY")" || { fail "recall_context"; exit 1; }
assert_no_rpc_error "$RECALL_RESP" "tools/call recall_context" || true

ENFORCE='{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"enforcement_evaluate","arguments":{"proposal_text":"Change production database to SQLite."}}}'
ENFORCE_RESP="$(mcp_post "$ENFORCE" "$PLURIBUS_API_KEY")" || { fail "enforcement_evaluate"; exit 1; }
assert_no_rpc_error "$ENFORCE_RESP" "tools/call enforcement_evaluate" || true

RECORD='{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"record_experience","arguments":{"summary":"Authenticated MCP proof recorded advisory episode for phase 1 close-out validation."}}}'
RECORD_RESP="$(mcp_post "$RECORD" "$PLURIBUS_API_KEY")" || { fail "record_experience"; exit 1; }
assert_no_rpc_error "$RECORD_RESP" "tools/call record_experience" || true

# Correct key via ?token= query (MCP-only)
TOKEN_URL="${MCP_URL}?token=${PLURIBUS_API_KEY}"
TOKEN_CODE="$(curl -sS -o /tmp/mcp-auth-token.json -w '%{http_code}' -X POST "$TOKEN_URL" -H 'Content-Type: application/json' -d "$INIT" || true)"
if [[ "$TOKEN_CODE" == "200" ]]; then
  if python3 -c "import json; d=json.load(open('/tmp/mcp-auth-token.json')); assert d.get('error') is None"; then
    pass "MCP initialize with ?token= query param"
  else
    fail "MCP ?token= returned JSON-RPC error"
  fi
else
  fail "MCP ?token= expected HTTP 200, got $TOKEN_CODE"
fi

log ""
log "======== SUMMARY ========"
log "PASS: $PASS  FAIL: $FAIL"
if (( FAIL > 0 )); then
  log "RESULT: FAIL"
  exit 1
fi
log "RESULT: PASS"
exit 0
