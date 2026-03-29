#!/usr/bin/env bash
# Seed control-plane with benefit-eval memories. Requires curl, jq.
# Usage: CONTROL_PLANE_URL=http://127.0.0.1:8123 ./scripts/pluribus-benefit-eval-seed.sh
# Prints: project_id=<uuid> (last line stdout for scripts; details on stderr)
set -euo pipefail

BASE="${CONTROL_PLANE_URL:-http://127.0.0.1:8123}"
BASE="${BASE%/}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
FIXTURES="${SCRIPT_DIR}/fixtures/benefit-eval-seeds.json"
STAMP="$(date +%s)"
SLUG="benefit-eval-${STAMP}"

echo "POST ${BASE}/v1/projects (slug=${SLUG})" >&2
RESP="$(curl -sS -X POST "${BASE}/v1/projects" \
  -H 'Content-Type: application/json' \
  -d "{\"slug\":\"${SLUG}\",\"name\":\"Pluribus benefit eval\"}")"

PID="$(echo "$RESP" | jq -r '.id // empty')"
if [[ -z "$PID" || "$PID" == "null" ]]; then
  echo "Failed to create project: $RESP" >&2
  exit 1
fi

COUNT=0
while IFS= read -r line; do
  [[ -z "$line" ]] && continue
  BODY="$(echo "$line" | jq --arg pid "$PID" '. + {project_id: $pid}')"
  HTTP="$(curl -sS -o /tmp/benefit-eval-mem.json -w '%{http_code}' -X POST "${BASE}/v1/memory" \
    -H 'Content-Type: application/json' \
    -d "$BODY")"
  if [[ "$HTTP" != "200" && "$HTTP" != "201" ]]; then
    echo "memory_create failed HTTP $HTTP: $(cat /tmp/benefit-eval-mem.json)" >&2
    exit 1
  fi
  COUNT=$((COUNT + 1))
done < <(jq -c '.memories[]' "$FIXTURES")

echo "Seeded ${COUNT} memories for project ${PID}" >&2
STATE_FILE="${SCRIPT_DIR}/.benefit-eval-last-project"
echo "$PID" > "$STATE_FILE"
echo "Wrote ${STATE_FILE}" >&2
echo "project_id=${PID}"
