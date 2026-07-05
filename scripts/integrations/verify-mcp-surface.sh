#!/usr/bin/env bash
set -euo pipefail
# Verify MCP registry/docs alignment; optional live tools/list against PLURIBUS_BASE_URL.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "$SCRIPT_DIR/common.sh"
parse_verify_args "$@"

FAIL=0
echo "== registry =="
REG_COUNT="$(registry_tool_count)"
echo "registry tools: $REG_COUNT"
if [[ "$REG_COUNT" != "59" ]]; then
  echo "expected 59 tools in tool_registry.go" >&2
  FAIL=1
fi
for t in "${CORE_LOOP_TOOLS[@]}"; do
  if registry_has_tool "$t"; then
    echo "ok registry: $t"
  else
    echo "missing registry tool: $t" >&2
    FAIL=1
  fi
done
TELEM="$(registry_count_prefix agent_telemetry_)"
UTIL="$(registry_count_prefix agent_utility_)"
echo "registry agent_telemetry_*: $TELEM"
echo "registry agent_utility_*: $UTIL"
if [[ "$TELEM" -lt 9 || "$UTIL" -lt 8 ]]; then
  echo "telemetry/utility tool count below expected" >&2
  FAIL=1
fi

echo "== docs/mcp-tools.md =="
DOC="$REPO_ROOT/docs/mcp-tools.md"
if [[ ! -f "$DOC" ]]; then
  echo "missing $DOC" >&2
  FAIL=1
else
  DOC_ROWS="$(grep -cE '^\| `[a-z_]+`' "$DOC" 2>/dev/null || echo 0)"
  echo "doc tool rows: $DOC_ROWS"
  if [[ "$DOC_ROWS" -lt 59 ]]; then
    echo "mcp-tools.md may be stale (<59 tool rows)" >&2
    FAIL=1
  fi
  for t in "${CORE_LOOP_TOOLS[@]}"; do
    grep -q "\`$t\`" "$DOC" || { echo "doc missing $t" >&2; FAIL=1; }
  done
  if grep -q housekeeping "$REPO_ROOT/control-plane/internal/mcp/init.go" && grep -q resolve_chore "$REPO_ROOT/control-plane/internal/mcp/init.go"; then
    echo "ok: MCP init instructions mention housekeeping"
  else
    echo "init.go missing housekeeping / resolve_chore" >&2
    FAIL=1
  fi
fi

echo "== integration MCP URLs =="
while IFS= read -r f; do
  if grep -qE '/v1/mcp' "$f" && grep -qE '8133|8080/v1/mcp' "$f" 2>/dev/null; then
    echo "suspicious port in $f" >&2
    FAIL=1
  fi
done < <(find "$REPO_ROOT/integrations" -type f \( -name '*.json' -o -name '*.md' \) 2>/dev/null)

if [[ "$STATIC" -eq 0 ]]; then
  verify_mcp_live || FAIL=1
else
  echo "(--static: skipping live MCP)"
fi

if [[ "$FAIL" -ne 0 ]]; then
  echo "verify-mcp-surface: FAILED" >&2
  exit 1
fi
echo "verify-mcp-surface: OK"
exit 0
