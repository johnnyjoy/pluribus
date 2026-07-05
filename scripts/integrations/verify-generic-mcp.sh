#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
parse_verify_args "$@"

FAIL=0
GEN="$REPO_ROOT/integrations/generic-mcp"
echo "== generic MCP pack =="

for f in README.md examples.json rules.md skill.md skills/pluribus/SKILL.md; do
  [[ -f "$GEN/$f" ]] || { echo "missing $f" >&2; FAIL=1; }
done

json_ok "$GEN/examples.json" || FAIL=1

for key in initialize tools_list tools_call_recall_context tools_call_record_experience tools_call_wakeup_context tools_call_resolve_chore; do
  python3 -c "import json; d=json.load(open('$GEN/examples.json')); assert '$key' in d" || {
    echo "examples.json missing $key" >&2
    FAIL=1
  }
done

if grep -q agent_telemetry_start_session "$GEN/examples.json" && grep -q agent_utility_evaluate_candidate "$GEN/examples.json"; then
  echo "ok: telemetry + utility examples"
else
  echo "examples.json missing Phase 11I/11K examples" >&2
  FAIL=1
fi

if grep -q '59' "$GEN/README.md" && grep -q '/v1/mcp' "$GEN/README.md"; then
  echo "ok: README documents endpoint + tool count"
else
  echo "generic README missing 59-tool note or endpoint" >&2
  FAIL=1
fi

instructions_has_loop "$REPO_ROOT/integrations/pluribus-instructions.md" || FAIL=1
instructions_has_housekeeping "$REPO_ROOT/integrations/pluribus-instructions.md" || FAIL=1

if [[ "$STATIC" -eq 0 ]] && curl -fsS -m 2 "${PLURIBUS_BASE_URL:-http://127.0.0.1:8123}/healthz" >/dev/null 2>&1; then
  INIT=$(python3 -c "import json; print(json.dumps(json.load(open('$GEN/examples.json'))['initialize']))")
  mcp_post "$INIT" >/dev/null && echo "live initialize: OK" || FAIL=1
else
  echo "(--static or server down: skipping live initialize)"
fi

[[ "$FAIL" -eq 0 ]] || { echo "verify-generic-mcp: FAILED" >&2; exit 1; }
echo "verify-generic-mcp: OK"
