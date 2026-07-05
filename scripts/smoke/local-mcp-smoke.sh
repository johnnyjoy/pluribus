#!/usr/bin/env bash
# MCP smoke: JSON-RPC initialize + tools/list count. Non-destructive.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=common.sh
source "${SCRIPT_DIR}/common.sh"

usage_base() {
  cat <<EOF
Usage: $0 [--base-url URL] [--api-key KEY]

Verifies POST /v1/mcp:
  initialize
  tools/list (expects >= 50 tools including agent_telemetry_* and agent_utility_*)

Does not call mutating tools.
EOF
}

parse_smoke_args "$@"

require_cmd curl
require_cmd jq

MCP_URL="${BASE_URL}/v1/mcp"
auth=()
[[ -n "$API_KEY" ]] && auth=(-H "X-API-Key: ${API_KEY}")

rpc() {
  local id="$1" method="$2" params="${3:-{}}"
  curl -sS -f -X POST "$MCP_URL" \
    "${auth[@]}" \
    -H "Content-Type: application/json" \
    -d "{\"jsonrpc\":\"2.0\",\"id\":${id},\"method\":\"${method}\",\"params\":${params}}"
}

echo "MCP smoke against ${MCP_URL}"

init="$(rpc 1 initialize '{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"smoke","version":"0"}}')"
echo "$init" | jq -e '.result.serverInfo.name' >/dev/null || fail "initialize failed"
pass "initialize"

tools="$(rpc 2 tools/list '{}')"
count="$(echo "$tools" | jq '.result.tools | length')"
[[ "$count" -ge 50 ]] || fail "tools/list count ${count} < 50"
echo "$tools" | jq -e '.result.tools[] | select(.name=="recall_context")' >/dev/null || fail "recall_context missing"
echo "$tools" | jq -e '.result.tools[] | select(.name=="agent_telemetry_start_session")' >/dev/null || fail "agent_telemetry_start_session missing"
echo "$tools" | jq -e '.result.tools[] | select(.name=="agent_utility_evaluate_candidate")' >/dev/null || fail "agent_utility_evaluate_candidate missing"
pass "tools/list count=${count}"

echo "MCP smoke: ALL PASS"
