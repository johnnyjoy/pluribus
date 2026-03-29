#!/usr/bin/env bash
# POC smoke: health → create project → recall compile → GET recall → run-multi (may warn if no reasoner).
# Requires: curl, jq. Stack up + migrations applied (see docs/deployment-poc.md).
set -euo pipefail
BASE="${BASE:-http://127.0.0.1:8123}"

echo "== Health"
curl -sS "$BASE/healthz"
echo

echo "== Create project"
PROJECT_JSON="$(curl -sS -X POST "$BASE/v1/projects" \
  -H 'Content-Type: application/json' \
  -d "{\"slug\":\"poc-e2e-$(date +%s)\",\"name\":\"POC E2E\"}")"
echo "$PROJECT_JSON" | jq .
PROJECT_ID="$(echo "$PROJECT_JSON" | jq -r .id)"
if [[ -z "$PROJECT_ID" || "$PROJECT_ID" == "null" ]]; then
  echo "Failed to get project id" >&2
  exit 1
fi

echo "== POST /v1/recall/compile"
curl -sS -X POST "$BASE/v1/recall/compile" \
  -H 'Content-Type: application/json' \
  -d "{\"project_id\":\"$PROJECT_ID\",\"tags\":[\"poc\"],\"max_per_kind\":3}" | jq . | head -c 4000
echo

echo "== GET /v1/recall/"
curl -sS "$BASE/v1/recall/?project_id=$PROJECT_ID&tags=poc&max_per_kind=3" | jq . | head -c 4000
echo

echo "== POST /v1/recall/run-multi (check debug if reasoner unset)"
curl -sS -X POST "$BASE/v1/recall/run-multi" \
  -H 'Content-Type: application/json' \
  -d "{\"query\":\"POC e2e smoke\",\"project_id\":\"$PROJECT_ID\",\"merge\":false,\"promote\":false}" | jq . | head -c 8000
echo
echo "== Done (project_id=$PROJECT_ID)"
