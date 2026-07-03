#!/usr/bin/env bash
# Shared helpers for integration verification scripts (Phase 12D).
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export REPO_ROOT

usage() {
  cat <<EOF
Usage: $0 [--static] [--help]

  --static   File/manifest checks only (no live server)
  --help     Show this message

Live checks require PLURIBUS_BASE_URL (default http://127.0.0.1:8123).
Optional PLURIBUS_API_KEY for authenticated servers.
EOF
}

parse_verify_args() {
  STATIC=0
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --static) STATIC=1; shift ;;
      --help|-h) usage; exit 0 ;;
      *) echo "unknown arg: $1" >&2; usage; exit 2 ;;
    esac
  done
}

require_cmd() {
  local c="$1"
  command -v "$c" >/dev/null 2>&1 || { echo "missing required command: $c" >&2; exit 1; }
}

json_ok() {
  local f="$1"
  python3 -c "import json; json.load(open('$f'))" 2>/dev/null || {
    echo "invalid JSON: $f" >&2
    return 1
  }
}

grep_file() {
  local pattern="$1"
  local file="$2"
  grep -qE "$pattern" "$file" 2>/dev/null
}

CORE_LOOP_TOOLS=(
  recall_context
  record_experience
  wakeup_context
)

TELEMETRY_TOOL_PREFIX=agent_telemetry_
UTILITY_TOOL_PREFIX=agent_utility_

registry_tool_count() {
  grep -cE '\{Name: "' "$REPO_ROOT/control-plane/internal/mcp/tool_registry.go" 2>/dev/null || echo 0
}

registry_has_tool() {
  local name="$1"
  grep -q "\"$name\"" "$REPO_ROOT/control-plane/internal/mcp/tool_registry.go"
}

registry_count_prefix() {
  local prefix="$1"
  grep -c "{Name: \"${prefix}" "$REPO_ROOT/control-plane/internal/mcp/tool_registry.go" 2>/dev/null || echo 0
}

mcp_post() {
  local body="$1"
  local base="${PLURIBUS_BASE_URL:-http://127.0.0.1:8123}"
  local url="${base%/}/v1/mcp"
  local hdr=(-H "Content-Type: application/json")
  if [[ -n "${PLURIBUS_API_KEY:-}" ]]; then
    hdr+=(-H "X-API-Key: ${PLURIBUS_API_KEY}")
  fi
  curl -fsS -m 30 "${hdr[@]}" -d "$body" "$url"
}

verify_mcp_live() {
  require_cmd curl
  require_cmd python3
  local base="${PLURIBUS_BASE_URL:-http://127.0.0.1:8123}"
  echo "== live MCP @ ${base}/v1/mcp =="
  if ! curl -fsS -m 5 "${base%/}/healthz" >/dev/null 2>&1; then
    echo "healthz unreachable at ${base}" >&2
    return 1
  fi
  local init='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"verify-integrations","version":"1.0.0"}}}'
  mcp_post "$init" >/dev/null
  local list='{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
  local resp
  resp="$(mcp_post "$list")"
  local count
  count="$(echo "$resp" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('result',{}).get('tools',[])))")"
  echo "tools/list count: $count"
  if [[ "$count" != "55" ]]; then
    echo "expected 55 tools, got $count" >&2
    return 1
  fi
  local t
  for t in "${CORE_LOOP_TOOLS[@]}"; do
    echo "$resp" | python3 -c "import sys,json; d=json.load(sys.stdin); names=[x['name'] for x in d.get('result',{}).get('tools',[])]; sys.exit(0 if '$t' in names else 1)" || {
      echo "missing live tool: $t" >&2
      return 1
    }
    echo "ok tool: $t"
  done
  local telem n
  telem="$(echo "$resp" | python3 -c "import sys,json; d=json.load(sys.stdin); print(sum(1 for x in d.get('result',{}).get('tools',[]) if x['name'].startswith('agent_telemetry_')))")"
  n="$(echo "$resp" | python3 -c "import sys,json; d=json.load(sys.stdin); print(sum(1 for x in d.get('result',{}).get('tools',[]) if x['name'].startswith('agent_utility_')))")"
  echo "agent_telemetry_* count: $telem"
  echo "agent_utility_* count: $n"
  if [[ "$telem" -lt 9 ]]; then
    echo "expected >=9 agent_telemetry_* tools" >&2
    return 1
  fi
  if [[ "$n" -lt 8 ]]; then
    echo "expected >=8 agent_utility_* tools" >&2
    return 1
  fi
  echo "live MCP surface: OK"
}

instructions_has_loop() {
  local f="$1"
  grep -q recall_context "$f" && grep -q record_experience "$f" && grep -qE 'server owns|Pluribus owns|control plane|control-plane' "$f"
}
