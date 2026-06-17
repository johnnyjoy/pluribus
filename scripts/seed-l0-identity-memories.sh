#!/usr/bin/env bash
# Seed minimal Pluribus L0 identity (kind=state) for wake-up / wakeup_context.
# Run once per empty pool (or when identity should be refreshed). Uses POST /v1/memories
# only—no plugin-local identity. Re-runs may hit dedup/authority merge per server policy.
set -euo pipefail

BASE="${PLURIBUS_BASE_URL:-http://127.0.0.1:8123}"
HDR=(-H "Content-Type: application/json")
if [[ -n "${PLURIBUS_API_KEY:-}" ]]; then
  HDR+=(-H "X-API-Key: ${PLURIBUS_API_KEY}")
fi

# Shared tags: filter in ops/queries; wake-up does not require tags unless you pass them in WakeupRequest.
TAGS_JSON='["pluribus","l0-identity","doctrine"]'

# Bash 3.2–compatible (no readarray): one line per statement.
STATEMENTS=(
  'Pluribus is the governed memory control plane: one shared durable pool; recall and enforcement use tags and situation text—not workspace or partition IDs as the memory truth boundary.'
  'Agent loop: recall relevant memory before substantive multi-step work; record meaningful outcomes after fixes, failures, or discoveries; the server owns ranking, authority, supersession, and applicability.'
  'Session-start L0 identity in wake-up is composed only from active kind=state rows in this pool; clients display them and do not define parallel identity outside Pluribus.'
)

echo "== Seeding ${#STATEMENTS[@]} state memories at ${BASE} ==" >&2
for stmt in "${STATEMENTS[@]}"; do
  [[ -z "${stmt// }" ]] && continue
  BODY=$(jq -n --arg s "$stmt" --argjson tags "$TAGS_JSON" \
    '{kind:"state",statement:$s,authority:8,tags:$tags}')
  out=$(curl -sS -w "\n%{http_code}" -X POST "${BASE}/v1/memories" "${HDR[@]}" -d "$BODY")
  code=$(echo "$out" | tail -n1)
  body=$(echo "$out" | sed '$d')
  if [[ "$code" != "200" ]]; then
    echo "ERROR: HTTP $code for statement (first 80 chars): ${stmt:0:80}..." >&2
    echo "$body" >&2
    exit 1
  fi
  id=$(echo "$body" | jq -r '.id // empty')
  echo "  ok: $id" >&2
done
echo "== Done. Verify: curl -s POST ${BASE}/v1/mcp -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"wakeup_context\",\"arguments\":{}}}' | jq ." >&2
