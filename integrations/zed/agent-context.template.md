# Zed — Pluribus agent context

Zed does not use a single repo-wide instruction file like Cursor’s **`pluribus.mdc`**. Paste the block below into your **Agent** custom instructions / default prompt (wording varies by Zed version), or keep it in project docs and reference it from your team’s onboarding.  
Append **`snippets/context-prime.txt`** from this pack.

**Canonical Pluribus behavior** (full mandatory loop): [`pluribus-instructions.md`](../pluribus-instructions.md).

---

## Pluribus (summary)

When **Pluribus** MCP tools are connected: **`recall_context`** → plan → act → **`record_experience`**. Full triggers, doctrine, and “tools unavailable” handling are in **`pluribus-instructions.md`**.

Legacy: **`memory_context_resolve`**, **`mcp_episode_ingest`**. **Pluribus** uses **tags + situation** only — not project/workspace/task memory partitions.
