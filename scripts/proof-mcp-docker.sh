#!/usr/bin/env bash
# Phase 1 MCP Docker proof — start stack, verify health/readiness, exercise JSON-RPC MCP.
# Safe to run repeatedly; exits nonzero on any failure.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

MCP_URL="${MCP_URL:-http://127.0.0.1:8123/v1/mcp}"
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:8123/healthz}"
READY_URL="${READY_URL:-http://127.0.0.1:8123/readyz}"
COMPOSE="${COMPOSE:-docker compose}"
WAIT_SECS="${WAIT_SECS:-120}"

# Optional host-conflict override (see docker-compose.host-conflicts.override.example.yml)
COMPOSE_FILES=(-f docker-compose.yml)
if [[ -f docker-compose.host-conflicts.override.yml ]]; then
  COMPOSE_FILES+=(-f docker-compose.host-conflicts.override.yml)
elif [[ -f docker-compose.override.yml ]]; then
  COMPOSE_FILES+=(-f docker-compose.override.yml)
fi

PASS=0
FAIL=0

log() { printf '[proof-mcp] %s\n' "$*"; }
pass() { PASS=$((PASS + 1)); log "PASS: $*"; }
fail() { FAIL=$((FAIL + 1)); log "FAIL: $*" >&2; }

mcp_post() {
  local body="$1"
  curl -fsS -X POST "$MCP_URL" \
    -H 'Content-Type: application/json' \
    -d "$body"
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

assert_rpc_error_code() {
  local resp="$1" want_code="$2" label="$3"
  local code
  code="$(echo "$resp" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('error',{}).get('code',''))")"
  if [[ "$code" == "$want_code" ]]; then
    pass "$label (code $want_code)"
  else
    fail "$label — expected code $want_code, got $code: $resp"
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

log "Starting Docker Compose stack..."
$COMPOSE "${COMPOSE_FILES[@]}" down --remove-orphans >/dev/null 2>&1 || true
if ! $COMPOSE "${COMPOSE_FILES[@]}" up -d --build; then
  fail "docker compose up failed"
  exit 1
fi

trap '$COMPOSE "${COMPOSE_FILES[@]}" down --remove-orphans >/dev/null 2>&1 || true' EXIT

log "Waiting for healthz/readyz (up to ${WAIT_SECS}s)..."
if wait_ready; then
  pass "healthz and readyz"
else
  fail "timeout waiting for healthz/readyz"
  $COMPOSE "${COMPOSE_FILES[@]}" ps || true
  exit 1
fi

log "Compose status:"
$COMPOSE "${COMPOSE_FILES[@]}" ps

HEALTH_BODY="$(curl -fsS "$HEALTH_URL")"
READY_BODY="$(curl -fsS "$READY_URL")"
log "healthz: $HEALTH_BODY"
log "readyz: $READY_BODY"

# --- MCP initialize ---
INIT_BODY='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"pluribus-phase1-proof","version":"0.1.0"}}}'
INIT_RESP="$(mcp_post "$INIT_BODY")" || { fail "initialize HTTP"; exit 1; }
assert_no_rpc_error "$INIT_RESP" "MCP initialize" || true

# --- tools/list ---
LIST_RESP="$(mcp_post '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}')" || { fail "tools/list HTTP"; exit 1; }
assert_no_rpc_error "$LIST_RESP" "MCP tools/list" || true

TOOL_COUNT="$(echo "$LIST_RESP" | python3 -c "
import json,sys
d=json.load(sys.stdin)
tools=d.get('result',{}).get('tools',[])
print(len(tools))
")"
if (( TOOL_COUNT >= 30 )); then
  pass "tools/list count=$TOOL_COUNT (>=30)"
else
  fail "tools/list count=$TOOL_COUNT expected >=30"
fi

REQUIRED_TOOLS=(recall_context record_experience enforcement_evaluate curation_pending wakeup_context)
if echo "$LIST_RESP" | python3 -c "
import json,sys
required=set('${REQUIRED_TOOLS[*]}'.split())
d=json.load(sys.stdin)
names={t.get('name') for t in d.get('result',{}).get('tools',[])}
missing=required-names
if missing:
    print('missing:', ','.join(sorted(missing)))
    sys.exit(1)
"; then
  pass "required tools present"
else
  fail "required tools missing from tools/list"
fi

# Schema non-empty check
if echo "$LIST_RESP" | python3 -c "
import json,sys
d=json.load(sys.stdin)
tools=d.get('result',{}).get('tools',[])
for t in tools:
    s=t.get('inputSchema') or {}
    if s.get('type')!='object':
        sys.exit(1)
    if 'additionalProperties' not in s:
        sys.exit(2)
    props=s.get('properties')
    if props is None and s.get('additionalProperties') is not False:
        sys.exit(3)
sys.exit(0)
"; then
  pass "all tool schemas have type=object and additionalProperties"
else
  fail "weak tool schemas in tools/list"
fi

# --- Happy-path tool calls ---
RECALL_BODY='{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"recall_context","arguments":{"task":"Phase 1 MCP docker proof recall"}}}'
RECALL_RESP="$(mcp_post "$RECALL_BODY")" || { fail "recall_context HTTP"; exit 1; }
assert_no_rpc_error "$RECALL_RESP" "tools/call recall_context" || true

RECORD_BODY='{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"record_experience","arguments":{"summary":"Phase 1 MCP docker proof recorded an advisory episode for schema and behavior validation."}}}'
RECORD_RESP="$(mcp_post "$RECORD_BODY")" || { fail "record_experience HTTP"; exit 1; }
assert_no_rpc_error "$RECORD_RESP" "tools/call record_experience" || true

ENFORCE_BODY='{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"enforcement_evaluate","arguments":{"proposal_text":"Migrate production database to SQLite without review."}}}'
ENFORCE_RESP="$(mcp_post "$ENFORCE_BODY")" || { fail "enforcement_evaluate HTTP"; exit 1; }
assert_no_rpc_error "$ENFORCE_RESP" "tools/call enforcement_evaluate" || true

CURATION_BODY='{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"curation_pending","arguments":{}}}'
CURATION_RESP="$(mcp_post "$CURATION_BODY")" || { fail "curation_pending HTTP"; exit 1; }
assert_no_rpc_error "$CURATION_RESP" "tools/call curation_pending" || true

# --- Hostile cases ---
UNKNOWN_TOOL='{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"does_not_exist","arguments":{}}}'
UNKNOWN_RESP="$(mcp_post "$UNKNOWN_TOOL")" || { fail "unknown tool HTTP"; exit 1; }
assert_rpc_error_code "$UNKNOWN_RESP" "-32602" "unknown tool rejected" || true

MISSING_ARG='{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"recall_context","arguments":{}}}'
MISSING_RESP="$(mcp_post "$MISSING_ARG")" || { fail "missing arg HTTP"; exit 1; }
assert_rpc_error_code "$MISSING_RESP" "-32602" "missing required argument" || true

EXTRA_ARG='{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"enforcement_evaluate","arguments":{"proposal_text":"x","extra":1}}}'
EXTRA_RESP="$(mcp_post "$EXTRA_ARG")" || { fail "extra arg HTTP"; exit 1; }
assert_rpc_error_code "$EXTRA_RESP" "-32602" "extra argument rejected" || true

UNKNOWN_METHOD='{"jsonrpc":"2.0","id":10,"method":"nope/method","params":{}}'
UNKNOWN_METHOD_RESP="$(mcp_post "$UNKNOWN_METHOD")" || { fail "unknown method HTTP"; exit 1; }
assert_rpc_error_code "$UNKNOWN_METHOD_RESP" "-32601" "unknown method rejected" || true

MALFORMED_HTTP="$(curl -sS -w '\n%{http_code}' -X POST "$MCP_URL" -H 'Content-Type: application/json' -d '{not json' || true)"
MALFORMED_CODE="$(echo "$MALFORMED_HTTP" | tail -n1)"
if [[ "$MALFORMED_CODE" == "200" ]]; then
  MALFORMED_BODY="$(echo "$MALFORMED_HTTP" | sed '$d')"
  if echo "$MALFORMED_BODY" | python3 -c "import json,sys; d=json.load(sys.stdin); sys.exit(0 if d.get('error') else 1)"; then
    pass "malformed JSON returns JSON-RPC error"
  else
    fail "malformed JSON did not return JSON-RPC error"
  fi
else
  pass "malformed JSON rejected (HTTP $MALFORMED_CODE)"
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
