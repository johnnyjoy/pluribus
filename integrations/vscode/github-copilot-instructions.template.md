# GitHub Copilot — Pluribus

Copy this file to **`.github/copilot-instructions.md`** in your workspace so Copilot Chat / agent mode picks it up ([VS Code custom instructions](https://code.visualstudio.com/docs/copilot/customization/custom-instructions)).  
Append **`snippets/context-prime.txt`** from this pack.

**Canonical Pluribus behavior** (full mandatory loop): [`pluribus-instructions.md`](../pluribus-instructions.md).

---

## Pluribus (summary)

When **Pluribus** MCP tools are connected: **`recall_context`** → plan → act → **`record_experience`**. Full triggers, doctrine, and “tools unavailable” handling are in **`pluribus-instructions.md`**.

Legacy: **`memory_context_resolve`**, **`mcp_episode_ingest`**. **Pluribus** uses **tags + situation** only — not project/workspace/task memory partitions.
