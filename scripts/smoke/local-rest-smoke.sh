#!/usr/bin/env bash
# REST smoke: health, ready, basic recall compile. Non-destructive.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=common.sh
source "${SCRIPT_DIR}/common.sh"

usage_base() {
  cat <<EOF
Usage: $0 [--base-url URL] [--api-key KEY]

Verifies:
  GET /healthz
  GET /readyz
  POST /v1/recall/compile (minimal query)

Does not mutate canonical memory.
EOF
}

parse_smoke_args "$@"

require_cmd curl
require_cmd jq

echo "REST smoke against ${BASE_URL}"

health="$(curl_json GET /healthz)"
echo "$health" | jq -e '.status == "ok" or . == "ok" or .ok == true' >/dev/null 2>&1 || {
  echo "$health"
  fail "healthz unexpected response"
}
pass "healthz"

ready_code="$(curl -sS -o /tmp/pluribus-ready.json -w '%{http_code}' "${BASE_URL}/readyz" ${API_KEY:+-H "X-API-Key: ${API_KEY}"})"
[[ "$ready_code" == "200" ]] || fail "readyz HTTP ${ready_code}"
pass "readyz HTTP 200"

compile_body='{"retrieval_query":"smoke test recall compile","max_per_kind":2,"max_total":5,"mode":"continuity"}'
bundle="$(curl_json POST /v1/recall/compile "$compile_body")"
echo "$bundle" | jq -e 'type == "object"' >/dev/null || fail "compile response not JSON object"
pass "recall compile"

echo "REST smoke: ALL PASS"
