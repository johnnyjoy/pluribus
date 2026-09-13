# OpenClaw adoption

Pluribus is an optional shared store. Do not replace `MEMORY.md`, dreaming, or `memory_search`. Those stay the workspace ritual. Pluribus is what an OpenClaw agent and a Cursor agent can both see.

1. Register the MCP server from [`mcp-config.example.json`](mcp-config.example.json).
2. Set `toolFilter.include` to the loop names only: `wakeup_context`, `recall_context`, `record_experience`, `memory_feedback`, `list_chores`, `resolve_chore`, `health`.
3. Optional: a `before_prompt_build` hook may call `wakeup_context` or a tight `recall_context` with the user prompt. See [`snippets/before-prompt-build.md`](snippets/before-prompt-build.md).
4. Never set `autoCapture` / `captureMatcher` to `*`. Write after a judged outcome via `record_experience` only. Optional `occurred_at` (RFC3339). Clock questions use RFC3339 bounds on `recall_context`.
5. Cross-harness Friday 3pm is Pluribus experiences, not `memory/YYYY-MM-DD.md`.

Canonical loop: [`pluribus-instructions.md`](../pluribus-instructions.md).
