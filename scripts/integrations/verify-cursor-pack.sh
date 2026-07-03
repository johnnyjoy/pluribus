#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
parse_verify_args "$@"

FAIL=0
PACK="$REPO_ROOT/integrations/cursor"
echo "== Cursor pack static =="

required=(
  README.md
  plugin-plan.md
  mcp-config.json
  pluribus.mdc
  rules.md
  skill.md
  skills/pluribus/SKILL.md
  helper/verify-mcp
  ENFORCEMENT-TIER.md
)
for f in "${required[@]}"; do
  if [[ -e "$PACK/$f" ]]; then
    echo "ok: $f"
  else
    echo "missing: $f" >&2
    FAIL=1
  fi
done

json_ok "$PACK/mcp-config.json" || FAIL=1
grep -q '/v1/mcp' "$PACK/mcp-config.json" || { echo "mcp-config missing /v1/mcp" >&2; FAIL=1; }

if instructions_has_loop "$REPO_ROOT/integrations/pluribus-instructions.md"; then
  echo "ok: canonical instructions"
else
  echo "canonical instructions drift" >&2
  FAIL=1
fi

if grep -qE 'not a VSIX|no separate VSIX|do not ship.*VSIX|We do not ship' "$PACK/plugin-plan.md" && grep -qE 'not a VSIX|no separate VSIX|do not ship.*VSIX|We do not ship' "$PACK/README.md"; then
  echo "ok: honest VSIX/marketplace labeling"
else
  echo "Cursor README/plugin-plan must state not a VSIX" >&2
  FAIL=1
fi

if grep -q 'Tier' "$PACK/ENFORCEMENT-TIER.md"; then
  echo "ok: enforcement tier doc"
else
  echo "ENFORCEMENT-TIER.md incomplete" >&2
  FAIL=1
fi

if [[ -x "$PACK/helper/verify-mcp" ]]; then
  echo "ok: verify-mcp executable"
else
  echo "helper/verify-mcp not executable" >&2
  FAIL=1
fi

if [[ "$STATIC" -eq 0 ]]; then
  if [[ -n "${PLURIBUS_BASE_URL:-}" ]]; then
    PLURIBUS_URL="${PLURIBUS_BASE_URL}" "$PACK/helper/verify-mcp" || FAIL=1
  elif curl -fsS -m 2 http://127.0.0.1:8123/healthz >/dev/null 2>&1; then
    PLURIBUS_URL="http://127.0.0.1:8123" "$PACK/helper/verify-mcp" || FAIL=1
  else
    echo "(--static or server down: skipping live verify-mcp)"
  fi
else
  echo "(--static: skipping live verify-mcp)"
fi

[[ "$FAIL" -eq 0 ]] || { echo "verify-cursor-pack: FAILED" >&2; exit 1; }
echo "verify-cursor-pack: OK"
