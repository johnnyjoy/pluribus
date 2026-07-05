#!/usr/bin/env bash
# Shared-memory install smoke: proves write → recall → enforcement in plain English.
#
# Audience: someone who just ran `docker compose up` and wants one command that says
# "shared memory works" without reading formation YAML.
#
# Usage:
#   ./scripts/smoke-shared-memory.sh
#   PLURIBUS_BASE_URL=http://host:8123 ./scripts/smoke-shared-memory.sh
#   PLURIBUS_API_KEY=... ./scripts/smoke-shared-memory.sh
#
# Exit 0 on PASS; non-zero on FAIL.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=smoke/common.sh
source "${SCRIPT_DIR}/smoke/common.sh"

usage_base() {
  cat <<EOF
Usage: $0 [--base-url URL] [--api-key KEY]

Verifies shared memory end-to-end:
  1. healthz
  2. POST governing constraint (Postgres vs SQLite)
  3. POST recall/compile surfaces the constraint
  4. POST enforcement/evaluate blocks a conflicting SQLite proposal

Requires curl and jq. Mutates memory (one smoke constraint per run).
EOF
}

parse_smoke_args "$@"
require_cmd curl
require_cmd jq

RUN_ID="$(date +%s)-$$"
MARKER="smoke-shared-memory-${RUN_ID}"
CONSTRAINT_STMT="All durable project data must use Postgres; SQLite is not permitted. [${MARKER}]"

echo ""
echo "Pluribus shared memory smoke"
echo "============================"
echo "Base URL: ${BASE_URL}"
echo ""

step_ok() { echo "  ok — $*"; }
step_fail() { echo "  FAIL — $*" >&2; exit 1; }

echo "[1/4] Server health"
health_code="$(curl -sS -o /tmp/pluribus-smoke-health.txt -w '%{http_code}' "${BASE_URL}/healthz" ${API_KEY:+-H "X-API-Key: ${API_KEY}"})"
health_body="$(cat /tmp/pluribus-smoke-health.txt)"
if [[ "$health_code" != "200" ]]; then
  step_fail "healthz HTTP ${health_code}: ${health_body}"
fi
if [[ "$health_body" != "ok" ]] && ! echo "$health_body" | jq -e '.status == "ok" or .ok == true' >/dev/null 2>&1; then
  step_fail "healthz unexpected response: ${health_body}"
fi
step_ok "healthz"

echo "[2/4] Write governing constraint"
create_body="$(jq -nc \
  --arg stmt "$CONSTRAINT_STMT" \
  '{kind:"constraint",authority:9,applicability:"governing",statement:$stmt,tags:["ephemeral","smoke-shared-memory"],ttl_seconds:3600}')"
created="$(curl_json POST /v1/memory "$create_body")"
status="$(echo "$created" | jq -r '.status // empty')"
consolidated="$(echo "$created" | jq -r '.consolidated // false')"
mem_id="$(echo "$created" | jq -r '.id // empty')"
if [[ "$status" != "active" && "$status" != "pending" ]]; then
  step_fail "memory create status=${status:-<missing>} (expected active; pending still recallable)"
fi
if [[ "$consolidated" == "true" ]]; then
  search_body='{"tags":["smoke-shared-memory"],"max":30}'
  search="$(curl_json POST /v1/memory/search "$search_body")"
  tag_found="$(echo "$search" | jq -r --arg m "$MARKER" '[.[] | select((.statement // "") | contains($m))] | length')"
  if [[ "${tag_found:-0}" -lt 1 ]]; then
    step_fail "consolidated=true but smoke marker not found via tag search — write not independently verifiable"
  fi
  step_ok "constraint reinforced existing memory (consolidated; marker verified via tag search)"
else
  step_ok "constraint stored (status=${status})"
fi

echo "[3/4] Recall surfaces what we wrote"
compile_body="$(jq -nc \
  --arg q "Postgres durable storage SQLite ${MARKER}" \
  '{retrieval_query:$q,max_per_kind:5,max_total:20}')"
bundle="$(curl_json POST /v1/recall/compile "$compile_body")"
if [[ "$consolidated" == "true" ]]; then
  found="$(echo "$bundle" | jq -r '
    [.governing_constraints[]?
      | select(
          ((.statement // "") | ascii_downcase | contains("postgres"))
          and ((.statement // "") | ascii_downcase | contains("sqlite"))
        )] | length
  ')"
  echo "  note — consolidated write: recall checks Postgres/SQLite constraint text, not unique marker (see docs/proof-scenarios.md honesty contract)"
else
  found="$(echo "$bundle" | jq -r --arg m "$MARKER" --arg id "$mem_id" '
    [.governing_constraints[]?, .decisions[]?, .known_failures[]?, .applicable_patterns[]?]
    | map(select(((.statement // "") | contains($m)) or ((.id // "") == $id)))
    | length
  ')"
fi
if [[ "${found:-0}" -lt 1 ]]; then
  echo "$bundle" | jq '.governing_constraints // []' >&2 || true
  step_fail "recall compile did not surface the smoke constraint"
fi
step_ok "recall returned the constraint"

echo "[4/4] Enforcement blocks conflicting proposal"
enf_body='{"proposal_text":"We will migrate durable storage to SQLite.","intent":"datastore"}'
enf="$(curl_json POST /v1/enforcement/evaluate "$enf_body")"
decision="$(echo "$enf" | jq -r '.decision // empty')"
if [[ "$decision" != "block" ]]; then
  echo "$enf" | jq '.' >&2 || true
  step_fail "enforcement decision=${decision:-<missing>} (expected block — memory did not bind)"
fi
step_ok "enforcement blocked SQLite (decision=block)"

echo ""
echo "PASS: Shared memory works."
echo "      What you write is recalled and influences enforcement."
echo ""
exit 0
