#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
parse_verify_args "$@"

FAIL=0
PLUGIN="$REPO_ROOT/integrations/claude-code-plugin"
echo "== Claude Code plugin =="

bash "$PLUGIN/scripts/verify-plugin.sh" || FAIL=1

for f in .mcp.json hooks/hooks.json .claude-plugin/plugin.json; do
  json_ok "$PLUGIN/$f" || FAIL=1
done
json_ok "$REPO_ROOT/integrations/.claude-plugin/marketplace.json" || FAIL=1

grep -q '/v1/mcp' "$PLUGIN/.mcp.json" || { echo ".mcp.json missing /v1/mcp" >&2; FAIL=1; }

if grep -q 'fail open\|fail-open\|unreachable' "$PLUGIN/README.md" && grep -q 'remote' "$PLUGIN/README.md"; then
  echo "ok: failure + remote limitations documented"
else
  echo "README missing failure-open or remote docs" >&2
  FAIL=1
fi

if grep -q 'recall_context' "$PLUGIN/skills/recall-context/SKILL.md" && grep -q 'record_experience' "$PLUGIN/skills/record-experience/SKILL.md"; then
  echo "ok: skills reference loop tools"
else
  echo "skills missing loop tool names" >&2
  FAIL=1
fi

if grep -qE 'housekeeping|resolve_chore' "$PLUGIN/hooks/session-start.sh" && [[ -f "$PLUGIN/skills/resolve-chore/SKILL.md" ]]; then
  echo "ok: housekeeping hook + resolve-chore skill"
else
  echo "plugin missing housekeeping enforcement" >&2
  FAIL=1
fi

for sh in hooks/session-start.sh hooks/user-prompt-recall.sh hooks/post-tool-failure-hint.sh; do
  test -f "$PLUGIN/$sh" || { echo "missing $sh" >&2; FAIL=1; }
done

if grep -qE 'curation/chores|housekeeping|resolve_chore' "$PLUGIN/hooks/user-prompt-recall.sh"; then
  echo "ok: user-prompt-recall housekeeping/chores hint"
else
  echo "user-prompt-recall.sh missing chore hint" >&2
  FAIL=1
fi

if [[ "$STATIC" -eq 0 ]] && curl -fsS -m 2 "${PLURIBUS_BASE_URL:-http://127.0.0.1:8123}/healthz" >/dev/null 2>&1; then
  echo "== hook dry-run health =="
  PLURIBUS_BASE_URL="${PLURIBUS_BASE_URL:-http://127.0.0.1:8123}" bash "$PLUGIN/hooks/session-start.sh" | head -c 200 >/dev/null || FAIL=1
  echo "session-start.sh runs"
else
  echo "(--static or server down: skipping hook dry-run)"
fi

[[ "$FAIL" -eq 0 ]] || { echo "verify-claude-code-plugin: FAILED" >&2; exit 1; }
echo "verify-claude-code-plugin: OK"
