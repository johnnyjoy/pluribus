#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
parse_verify_args "$@"

FAIL=0
EXT="$REPO_ROOT/integrations/vscode/extension"
echo "== VS Code extension static =="

for f in package.json tsconfig.json src/extension.ts src/orchestrator.ts src/api.ts README.md; do
  [[ -f "$EXT/$f" ]] || { echo "missing $f" >&2; FAIL=1; }
done

json_ok "$EXT/package.json" || FAIL=1
grep -q 'pluribus-ai' "$EXT/package.json" || FAIL=1
grep -q 'pluribus.verifyMandatoryLoop' "$EXT/package.json" || { echo "missing verifyMandatoryLoop command" >&2; FAIL=1; }
grep -q 'recallOnTaskStart' "$EXT/package.json" || FAIL=1
grep -q '/v1/recall/compile' "$EXT/src/api.ts" || FAIL=1
grep -q '/v1/advisory-episodes' "$EXT/src/api.ts" || FAIL=1

if grep -qE 'VSIX|no marketplace|not published.*Marketplace|Install from VSIX' "$REPO_ROOT/integrations/vscode/README.md"; then
  echo "ok: honest marketplace/VSIX docs"
else
  echo "vscode README should document VSIX-only distribution" >&2
  FAIL=1
fi

if grep -q 'server owns\|control plane' "$EXT/README.md"; then
  echo "ok: server owns memory documented"
else
  echo "extension README missing server-owns-truth" >&2
  FAIL=1
fi

if [[ "$STATIC" -eq 0 ]]; then
  if command -v npm >/dev/null 2>&1 && [[ -f "$EXT/package-lock.json" ]]; then
    (cd "$EXT" && npm run compile) || FAIL=1
    echo "npm compile: OK"
  else
    echo "skip compile (npm or lockfile missing)"
  fi
fi

[[ "$FAIL" -eq 0 ]] || { echo "verify-vscode-extension: FAILED" >&2; exit 1; }
echo "verify-vscode-extension: OK"
