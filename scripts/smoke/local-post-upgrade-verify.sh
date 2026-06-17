#!/usr/bin/env bash
# Post-upgrade verification orchestrator — runs all local smoke scripts.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

usage() {
  cat <<EOF
Usage: $0 [--base-url URL] [--api-key KEY] [--help]

Runs (in order):
  local-rest-smoke.sh
  local-mcp-smoke.sh
  local-telemetry-smoke.sh
  local-utility-policy-smoke.sh

Set PLURIBUS_BASE_URL / PLURIBUS_API_KEY or pass flags through.
Non-destructive except one telemetry session row.
EOF
}

ARGS=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --help|-h) usage; exit 0 ;;
    *) ARGS+=("$1"); shift ;;
  esac
done

chmod +x "${SCRIPT_DIR}"/*.sh 2>/dev/null || true

for s in local-rest-smoke local-mcp-smoke local-telemetry-smoke local-utility-policy-smoke; do
  echo "=== ${s} ==="
  "${SCRIPT_DIR}/${s}.sh" "${ARGS[@]}"
done

echo "Post-upgrade verify: ALL PASS"
