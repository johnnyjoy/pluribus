#!/usr/bin/env bash
# UserPromptSubmit: optional HTTP recall preview (same bundle shape as server compile).
# Conservative: skips short prompts; caps output size; fails open on errors.
set -euo pipefail

if [[ "${PLURIBUS_HOOK_RECALL:-on}" == "off" ]]; then
  exit 0
fi

INPUT=$(cat)
PROMPT=$(echo "$INPUT" | jq -r '.prompt // empty' 2>/dev/null || echo "")

if [[ ${#PROMPT} -lt 45 ]]; then
  exit 0
fi

if ! command -v curl >/dev/null 2>&1 || ! command -v jq >/dev/null 2>&1; then
  exit 0
fi

BASE="${PLURIBUS_BASE_URL:-http://127.0.0.1:8123}"

# Keep retrieval text bounded (compile uses this as situation / intent text).
Q=$(printf '%s' "$PROMPT" | head -c 8000)
BODY=$(jq -n --arg q "$Q" '{"tags":["claude-code"],"retrieval_query":$q,"max_total":24}')

HDR=(-H "Content-Type: application/json")
if [[ -n "${PLURIBUS_API_KEY:-}" ]]; then
  HDR+=(-H "X-API-Key: ${PLURIBUS_API_KEY}")
fi

RESP=$(curl -fsS -m 12 -X POST "${BASE}/v1/recall/compile" "${HDR[@]}" -d "$BODY" 2>/dev/null) || exit 0

OUT=$(echo "$RESP" | jq -r '
  def nonempty(s): (s | type) == "string" and (s | length) > 0;
  [
    "## Pluribus recall (preview)",
    (if nonempty(.recall_preamble) then .recall_preamble else empty end),
    (if .agent_grounding.formatted != null and nonempty(.agent_grounding.formatted) then .agent_grounding.formatted else empty end)
  ] | map(select(. != null)) | join("\n\n")
' 2>/dev/null || echo "")

if [[ -z "${OUT// }" ]] || [[ "$OUT" == "## Pluribus recall (preview)" ]]; then
  exit 0
fi

# Cap injected context (hooks reference: ~10k; stay under that).
OUT=$(printf '%s' "$OUT" | head -c 9500)

if command -v jq >/dev/null 2>&1; then
  jq -n \
    --arg text "$OUT" \
    '{
      hookSpecificOutput: {
        hookEventName: "UserPromptSubmit",
        additionalContext: $text
      }
    }'
else
  printf '%s\n' "$OUT"
fi
