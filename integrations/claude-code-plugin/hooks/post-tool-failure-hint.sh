#!/usr/bin/env bash
# PostToolUseFailure: optional one-line reminder to consider record_experience after failures.
# Default off — set PLURIBUS_HOOK_FAILURE_HINT=on to enable (conservative; avoids spam).
set -euo pipefail

if [[ "${PLURIBUS_HOOK_FAILURE_HINT:-off}" != "on" ]]; then
  exit 0
fi

INPUT=$(cat)
ERR=$(printf '%s' "$INPUT" | jq -r '.error // empty' 2>/dev/null || echo "")
INTERRUPT=$(printf '%s' "$INPUT" | jq -r '.is_interrupt // false' 2>/dev/null || echo "false")

if [[ "$INTERRUPT" == "true" ]]; then
  exit 0
fi

if [[ ${#ERR} -lt 20 ]]; then
  exit 0
fi

TEXT="## Pluribus (post-failure hint)

A tool failed (see error above). If the cause was **non-obvious** or will affect others, consider **`record_experience`** after you understand what happened—the server qualifies ingest.

Disable this hook: \`PLURIBUS_HOOK_FAILURE_HINT=off\` (default)."

if command -v jq >/dev/null 2>&1; then
  jq -n \
    --arg text "$TEXT" \
    '{
      hookSpecificOutput: {
        hookEventName: "PostToolUseFailure",
        additionalContext: $text
      }
    }'
else
  printf '%s\n' "$TEXT"
fi
