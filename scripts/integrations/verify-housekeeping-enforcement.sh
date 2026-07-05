#!/usr/bin/env bash
# Fail closed when agent-side housekeeping enforcement text drifts out of canonical + packs.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "$SCRIPT_DIR/common.sh"
parse_verify_args "$@"

FAIL=0
echo "== housekeeping enforcement (static grep) =="

CANON="$REPO_ROOT/integrations/pluribus-instructions.md"
if instructions_has_housekeeping "$CANON"; then
  echo "ok: canonical pluribus-instructions.md"
else
  echo "canonical instructions missing housekeeping loop" >&2
  FAIL=1
fi

check_file() {
  local label="$1"
  local file="$2"
  if [[ ! -f "$file" ]]; then
    echo "missing: $label ($file)" >&2
    FAIL=1
    return
  fi
  if grep -qE 'housekeeping|resolve_chore' "$file" && grep -q 'resolve_chore' "$file" && grep -q 'agent_id' "$file"; then
    echo "ok: $label"
  else
    echo "drift: $label (need housekeeping, resolve_chore, agent_id)" >&2
    FAIL=1
  fi
}

PACK_FILES=(
  "$REPO_ROOT/.cursor/rules/pluribus.mdc"
  "$REPO_ROOT/integrations/cursor/pluribus.mdc"
  "$REPO_ROOT/integrations/cursor/skills/pluribus/SKILL.md"
  "$REPO_ROOT/integrations/cursor/skills/pluribus-housekeeping/SKILL.md"
  "$REPO_ROOT/integrations/generic-mcp/skill.md"
  "$REPO_ROOT/integrations/generic-mcp/skills/pluribus/SKILL.md"
  "$REPO_ROOT/integrations/claude-code/skill.md"
  "$REPO_ROOT/integrations/claude-code/skills/pluribus/SKILL.md"
  "$REPO_ROOT/integrations/vscode/skill.md"
  "$REPO_ROOT/integrations/continue/skill.md"
  "$REPO_ROOT/integrations/continue/rules/pluribus.md"
  "$REPO_ROOT/integrations/zed/skill.md"
  "$REPO_ROOT/integrations/openclaw/skill.md"
  "$REPO_ROOT/integrations/claude-desktop/skill.md"
  "$REPO_ROOT/integrations/opencode/skills/pluribus/SKILL.md"
  "$REPO_ROOT/control-plane/internal/mcp/init.go"
)

for f in "${PACK_FILES[@]}"; do
  check_file "$(basename "$f")" "$f"
done

if grep -q 'tools_call_resolve_chore' "$REPO_ROOT/integrations/generic-mcp/examples.json"; then
  echo "ok: generic-mcp examples.json resolve_chore"
else
  echo "generic-mcp/examples.json missing tools_call_resolve_chore" >&2
  FAIL=1
fi

if grep -qE 'housekeeping|resolve_chore' "$REPO_ROOT/integrations/claude-code-plugin/hooks/session-start.sh"; then
  echo "ok: claude-code-plugin session-start housekeeping"
else
  echo "claude-code-plugin/hooks/session-start.sh missing housekeeping nudge" >&2
  FAIL=1
fi

if [[ -f "$REPO_ROOT/integrations/claude-code-plugin/skills/resolve-chore/SKILL.md" ]]; then
  echo "ok: claude-code-plugin resolve-chore skill"
else
  echo "missing integrations/claude-code-plugin/skills/resolve-chore/SKILL.md" >&2
  FAIL=1
fi

if grep -qE 'ephemeral|proof-scenario|smoke-shared-memory' "$REPO_ROOT/docs/memory-doctrine.md"; then
  echo "ok: memory-doctrine ephemeral proof policy"
else
  echo "docs/memory-doctrine.md missing ephemeral/proof hygiene" >&2
  FAIL=1
fi

if [[ "$FAIL" -ne 0 ]]; then
  echo "verify-housekeeping-enforcement: FAILED" >&2
  exit 1
fi
echo "verify-housekeeping-enforcement: OK"
exit 0
