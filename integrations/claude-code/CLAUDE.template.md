# CLAUDE.md — Pluribus (Claude Code)

Copy this file to **`CLAUDE.md`** in your **project root**. Claude Code reads **`CLAUDE.md`** for project-specific instructions ([Claude Code documentation](https://docs.anthropic.com/en/docs/claude-code)).

Append **`snippets/context-prime.txt`** from this pack.

**Canonical Pluribus behavior** (full mandatory loop, keep in sync): [`pluribus-instructions.md`](../pluribus-instructions.md).

---

## Pluribus (summary)

When **Pluribus** MCP tools are connected: **`recall_context`** → plan → act → **`record_experience`**. Full triggers, doctrine, and “tools unavailable” handling are in **`pluribus-instructions.md`**.

Legacy: **`memory_context_resolve`**, **`mcp_episode_ingest`**. **Pluribus** uses **tags + situation** only — not project/workspace/task memory partitions.
