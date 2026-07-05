# CLAUDE.md — Pluribus (Claude Code)

Copy this file to **`CLAUDE.md`** in your **project root**. Claude Code reads **`CLAUDE.md`** for project-specific instructions ([Claude Code documentation](https://docs.anthropic.com/en/docs/claude-code)).

Append **`snippets/context-prime.txt`** from this pack.

**Canonical Pluribus behavior** (full mandatory loop, keep in sync): [`pluribus-instructions.md`](../pluribus-instructions.md).

---

## Pluribus (summary)

When **Pluribus** MCP tools are connected:

1. **`recall_context`** or **`wakeup_context`** before substantive work
2. If **`housekeeping`** / open chores → **`resolve_chore`** with **`agent_id`** (or defer in **`record_experience`**)
3. Plan → act → **`memory_feedback`** when recall items helped or misled
4. **`record_experience`** after meaningful outcomes

Full triggers: **`pluribus-instructions.md`**. Legacy: **`memory_context_resolve`**, **`mcp_episode_ingest`**. **Tags + situation** only.
