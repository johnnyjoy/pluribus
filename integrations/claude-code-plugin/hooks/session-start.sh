#!/usr/bin/env bash
# SessionStart: optional Pluribus wake-up (HTTP MCP tools/call wakeup_context) + short orientation.
# Wake-up JSON is identical to MCP over POST /v1/mcp; the plugin only formats and injects text.
# Pluribus owns selection, authority, and truth; this script does not rank or classify memory.
set -euo pipefail

BASE="${PLURIBUS_BASE_URL:-http://127.0.0.1:8123}"
HDR=(-H "Content-Type: application/json")
if [[ -n "${PLURIBUS_API_KEY:-}" ]]; then
  HDR+=(-H "X-API-Key: ${PLURIBUS_API_KEY}")
fi

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
- Use MCP tools \`recall_context\` before substantive work; \`record_experience\` after meaningful outcomes; \`wakeup_context\` at session start is loaded below when the server is up.
- Optional hook \`UserPromptSubmit\` may inject a recall preview from \`POST /v1/recall/compile\` (set \`PLURIBUS_HOOK_RECALL=off\` to disable).
"

WAKE_BLOCK=""
if [[ "${PLURIBUS_HOOK_WAKEUP:-on}" != "off" ]] \
  && [[ "$STATUS" == "reachable" ]] \
  && command -v curl >/dev/null 2>&1 \
  && command -v jq >/dev/null 2>&1; then

  MCP_BODY='{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"wakeup_context","arguments":{}}}'
  if MCP_RESP=$(curl -fsS -m 25 -X POST "${BASE}/v1/mcp" "${HDR[@]}" -d "$MCP_BODY" 2>/dev/null); then
    if echo "$MCP_RESP" | jq -e '.result.isError == false' >/dev/null 2>&1; then
      WAKE_JSON=$(echo "$MCP_RESP" | jq -c '.result.content[0].text | fromjson' 2>/dev/null) || WAKE_JSON=""
      if [[ -n "$WAKE_JSON" ]]; then
        MI="${PLURIBUS_HOOK_WAKEUP_MAX_IDENTITY:-4}"
        MG="${PLURIBUS_HOOK_WAKEUP_MAX_GOVERNING:-8}"
        [[ "$MI" =~ ^[0-9]+$ ]] || MI=4
        [[ "$MG" =~ ^[0-9]+$ ]] || MG=8
        WAKE_BLOCK=$(echo "$WAKE_JSON" | jq -r --argjson mi "$MI" --argjson mg "$MG" '
          def stmt(i): (i.statement // "") | gsub("\r"; "") | gsub("\n"; " ")
            | if length > 160 then .[0:160] + "…" else . end;
          [
            "### Pluribus wake-up",
            "",
            "**Identity** *(from Pluribus kind=state only)*",
            (if (.identity | type) == "array" and (.identity | length) > 0 then
              .identity[0:$mi][] | "- \(stmt(.))"
            else
              "- *(empty — run repo \`scripts/seed-l0-identity-memories.sh\` once per pool if you want L0 text here)*"
            end),
            "",
            "**Governing memory** *(server-selected, bounded)*",
            (if (.governing_memory | type) == "array" and (.governing_memory | length) > 0 then
              .governing_memory[0:$mg][] | "- \(.kind): \(stmt(.))"
            else
              "- *(none)*"
            end)
          ] | join("\n")
        ' 2>/dev/null) || WAKE_BLOCK=""
      fi
    fi
  fi
fi

FULL_TEXT="$TEXT"
if [[ -n "$WAKE_BLOCK" ]]; then
  FULL_TEXT="${TEXT}

${WAKE_BLOCK}
"
fi

# Cap total injected context (hooks stay bounded).
FULL_TEXT=$(printf '%s' "$FULL_TEXT" | head -c 9000)

if command -v jq >/dev/null 2>&1; then
  jq -n \
    --arg base "$BASE" \
    --arg status "$STATUS" \
    --arg text "$FULL_TEXT" \
    '{
      hookSpecificOutput: {
        hookEventName: "SessionStart",
        additionalContext: $text
      }
    }'
else
  printf '%s' "$FULL_TEXT"
fi
