#!/usr/bin/env bash
# Utility policy smoke: read-only policy summary + applications list. No apply mutations.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=common.sh
source "${SCRIPT_DIR}/common.sh"

usage_base() {
  cat <<EOF
Usage: $0 [--base-url URL] [--api-key KEY]

Verifies:
  GET /v1/agent/utility/policy/summary
  GET /v1/agent/utility/policy/applications

Does not call apply-candidate or apply-batch.
EOF
}

parse_smoke_args "$@"

require_cmd curl
require_cmd jq

echo "Utility policy smoke against ${BASE_URL}"

summary="$(curl_json GET /v1/agent/utility/policy/summary)"
echo "$summary" | jq -e 'type == "object"' >/dev/null || fail "policy summary invalid"
pass "utility policy summary"

apps="$(curl_json GET /v1/agent/utility/policy/applications)"
echo "$apps" | jq -e 'type == "object" or type == "array"' >/dev/null || fail "applications list invalid"
pass "utility policy applications list"

echo "Utility policy smoke: ALL PASS"
