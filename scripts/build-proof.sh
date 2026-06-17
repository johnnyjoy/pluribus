#!/usr/bin/env bash
# Build proof for Phase 12C — writes artifacts/local-upgrade-build-proof.json
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ARTIFACT="${ROOT}/artifacts/local-upgrade-build-proof.json"
BIN_CP="${ROOT}/control-plane/controlplane"
BIN_MCP="${ROOT}/control-plane/pluribus-mcp"
mkdir -p "${ROOT}/artifacts"

cd "${ROOT}"
make build

cp_ok=false
mcp_ok=false
ver_ok=false

if [[ -x "$BIN_CP" ]]; then cp_ok=true; fi
if [[ -x "$BIN_MCP" ]]; then mcp_ok=true; fi

cp_ver="$("$BIN_CP" --version 2>/dev/null || true)"
mcp_ver="$("$BIN_MCP" --version 2>/dev/null || true)"
[[ -n "$cp_ver" && -n "$mcp_ver" ]] && ver_ok=true

cat > "$ARTIFACT" <<EOF
{
  "build_control_plane_pass": ${cp_ok},
  "build_mcp_pass": ${mcp_ok},
  "version_metadata_present": ${ver_ok},
  "artifact_paths": {
    "controlplane": "${BIN_CP}",
    "pluribus_mcp": "${BIN_MCP}"
  },
  "controlplane_version_line": $(jq -Rn --arg v "$cp_ver" '$v'),
  "pluribus_mcp_version_line": $(jq -Rn --arg v "$mcp_ver" '$v'),
  "external_service_required_for_build": false
}
EOF

echo "build-proof: wrote ${ARTIFACT}"
"$BIN_CP" --version
"$BIN_MCP" --version

if ! $cp_ok || ! $mcp_ok || ! $ver_ok; then
  echo "build-proof: FAIL" >&2
  exit 1
fi
echo "build-proof: PASS"
