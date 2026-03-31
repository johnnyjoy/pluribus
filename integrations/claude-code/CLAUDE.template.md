# CLAUDE.md — Pluribus (Claude Code)

Copy this file to **`CLAUDE.md`** in your **project root**. Claude Code reads **`CLAUDE.md`** for project-specific instructions ([Claude Code documentation](https://docs.anthropic.com/en/docs/claude-code)).

Append **`snippets/context-prime.txt`** from this pack below the behavior block.

**Canonical** behavior text (keep in sync): [`pluribus-instructions.md`](../pluribus-instructions.md).

---

# Pluribus — MCP behavior

Before non-trivial reasoning or changes → **`recall_context`**.  
Then **plan → act** (tools, edits, deliverables).  
After meaningful work → **`record_experience`**.  
Do not proceed on complex work without recall when Pluribus MCP is available.  
Legacy tools **`memory_context_resolve`**, **`mcp_episode_ingest`**. No **scope** memory partitions—tags + situation only (`docs/anti-regression.md`).
