#!/usr/bin/env bash
# Telemetry smoke: session start + violations list (read-only after session create). Minimal mutation.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=common.sh
source "${SCRIPT_DIR}/common.sh"

usage_base() {
  cat <<EOF
Usage: $0 [--base-url URL] [--api-key KEY]

Verifies:
  POST /v1/agent/telemetry/session/start
  GET  /v1/agent/telemetry/violations

Creates one telemetry session row (low-impact). Does not evaluate or apply utility.
EOF
}

parse_smoke_args "$@"

require_cmd curl
require_cmd jq

echo "Telemetry smoke against ${BASE_URL}"

session_body='{"client_id":"smoke-test","transport":"rest-smoke","repo_root":"/projects/pluribus"}'
session="$(curl_json POST /v1/agent/telemetry/session/start "$session_body")"
sid="$(echo "$session" | jq -r '.session_id // .id // empty')"
[[ -n "$sid" ]] || fail "session start missing session_id"
pass "telemetry session start id=${sid}"

violations="$(curl_json GET "/v1/agent/telemetry/violations?session_id=${sid}")"
echo "$violations" | jq -e 'type == "object" or type == "array"' >/dev/null || fail "violations response invalid"
pass "telemetry violations query"

echo "Telemetry smoke: ALL PASS"
