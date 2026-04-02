#!/usr/bin/env bash
# SessionStart: inject short orientation + optional health check (stdout → Claude context).
# Pluribus semantics stay on the server; this is orchestration only.
set -euo pipefail

BASE="${PLURIBUS_BASE_URL:-http://127.0.0.1:8123}"
STATUS="unknown"
if command -v curl >/dev/null 2>&1; then
  if curl -fsS -m 2 "${BASE}/healthz" >/dev/null 2>&1; then
    STATUS="reachable"
  else
    STATUS="unreachable"
  fi
else
  STATUS="curl-missing"
fi

TEXT="## Pluribus (plugin)

- API base: ${BASE} — health: ${STATUS}
- Use MCP tools \`recall_context\` before substantive work; \`record_experience\` after meaningful outcomes.
- Optional hook \`UserPromptSubmit\` may inject a recall preview from \`POST /v1/recall/compile\` when the server is up (set \`PLURIBUS_HOOK_RECALL=off\` to disable).
"

if command -v jq >/dev/null 2>&1; then
  jq -n \
    --arg base "$BASE" \
    --arg status "$STATUS" \
    --arg text "$TEXT" \
    '{
      hookSpecificOutput: {
        hookEventName: "SessionStart",
        additionalContext: $text
      }
    }'
else
  printf '%s' "$TEXT"
fi
