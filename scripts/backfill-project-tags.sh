#!/usr/bin/env bash
# Backfill project:* and repo:* tags on existing memories (audit Task A).
#
# Uses POST /v1/memories/search with keyword queries, classifies rows by statement
# text, and merges tags via PUT /v1/memory/{id}/tags (idempotent).
#
# Usage:
#   PLURIBUS_URL=http://10.1.1.79:8123 ./scripts/backfill-project-tags.sh [--dry-run] [--apply]
#   ./scripts/backfill-project-tags.sh --dry-run   # default: dry-run only
#
# Requires: curl, jq, python3
set -euo pipefail

PLURIBUS_URL="${PLURIBUS_URL:-http://127.0.0.1:8123}"
PLURIBUS_URL="${PLURIBUS_URL%/}"
DRY_RUN=1
APPLY=0
MAX_PER_QUERY=200

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) DRY_RUN=1; APPLY=0; shift ;;
    --apply) DRY_RUN=0; APPLY=1; shift ;;
    --url) PLURIBUS_URL="${2%/}"; shift 2 ;;
    --help|-h)
      grep '^#' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *) echo "Unknown option: $1" >&2; exit 2 ;;
  esac
done

for tool in curl jq python3; do
  command -v "$tool" >/dev/null || { echo "FAIL: need $tool" >&2; exit 1; }
done

if ! curl -fsS -m 5 "$PLURIBUS_URL/healthz" >/dev/null; then
  echo "FAIL: $PLURIBUS_URL/healthz unreachable" >&2
  exit 1
fi

AUTH=()
if [[ -n "${PLURIBUS_API_KEY:-}" ]]; then
  AUTH=(-H "X-API-Key: ${PLURIBUS_API_KEY}")
fi

# Keywords to pull candidate rows (union by memory id).
KEYWORDS=(phprax Pluribus pluribus Phase upgrade-e2e orac-live-probe)

search_keyword() {
  local kw="$1"
  curl -fsS -m 30 "${AUTH[@]}" -X POST "$PLURIBUS_URL/v1/memories/search" \
    -H 'Content-Type: application/json' \
    -d "{\"query\":\"$kw\",\"max\":$MAX_PER_QUERY}"
}

classify_tags() {
  python3 -c "
import json, sys, re
raw = sys.stdin.read()
try:
    rows = json.loads(raw)
except json.JSONDecodeError:
    sys.exit(0)
if not isinstance(rows, list):
    sys.exit(0)
for row in rows:
    stmt = (row.get('statement') or '') + ' ' + (row.get('statement_canonical') or '')
    tags = row.get('tags') or []
    has_project = any(isinstance(t, str) and t.startswith('project:') for t in tags)
    if has_project:
        continue
    mid = row.get('id')
    if not mid:
        continue
    s = stmt.lower()
    project = None
    if 'phprax' in s:
        project = 'phprax'
    elif re.search(r'\bpluribus\b', s) or re.search(r'\bphase\b', s):
        project = 'pluribus'
    if not project:
        continue
    add = [f'project:{project}', f'repo:{project}']
    print(json.dumps({'id': mid, 'statement': row.get('statement', '')[:120], 'add': add}))
"
}

PLANNED=0
APPLIED=0
SKIPPED=0
declare -A SEEN

echo "== backfill project tags @ $PLURIBUS_URL (dry_run=$DRY_RUN) =="

for kw in "${KEYWORDS[@]}"; do
  echo "-- search: $kw"
  resp="$(search_keyword "$kw" || true)"
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    id="$(echo "$line" | jq -r '.id')"
    [[ -n "${SEEN[$id]:-}" ]] && continue
    SEEN[$id]=1
    add_tags="$(echo "$line" | jq -c '.add')"
    stmt="$(echo "$line" | jq -r '.statement')"
    PLANNED=$((PLANNED + 1))
    echo "  plan $id tags=$add_tags | ${stmt:0:80}"
    if [[ "$APPLY" -eq 1 ]]; then
      body="$(jq -nc --argjson tags "$add_tags" '{tags: $tags}')"
      if curl -fsS -m 20 "${AUTH[@]}" -X PUT "$PLURIBUS_URL/v1/memory/$id/tags" \
        -H 'Content-Type: application/json' -d "$body" >/dev/null; then
        APPLIED=$((APPLIED + 1))
      else
        echo "  FAIL apply $id" >&2
        SKIPPED=$((SKIPPED + 1))
      fi
    fi
  done < <(echo "$resp" | classify_tags)
done

echo "== summary: planned=$PLANNED applied=$APPLIED skipped=$SKIPPED =="
if [[ "$DRY_RUN" -eq 1 && "$APPLY" -eq 0 ]]; then
  echo "Dry-run only. Re-run with --apply to merge tags."
fi
