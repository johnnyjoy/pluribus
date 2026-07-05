# GitHub Copilot — Pluribus

Copy this file to **`.github/copilot-instructions.md`** in your workspace so Copilot Chat / agent mode picks it up ([VS Code custom instructions](https://code.visualstudio.com/docs/copilot/customization/custom-instructions)).  
Append **`snippets/context-prime.txt`** from this pack.

**Canonical Pluribus behavior** (full mandatory loop): [`pluribus-instructions.md`](../pluribus-instructions.md).

---

## Pluribus (summary)

When **Pluribus** MCP tools are connected:

1. **`recall_context`** or **`wakeup_context`** before substantive work
2. If **`housekeeping`** / open chores → **`resolve_chore`** with **`agent_id`** (or defer in **`record_experience`**)
3. Plan → act → **`memory_feedback`** when recall items helped or misled
4. **`record_experience`** after meaningful outcomes

Full triggers: **`pluribus-instructions.md`**. Legacy: **`memory_context_resolve`**, **`mcp_episode_ingest`**. **Tags + situation** only.
