# GitHub Copilot — Pluribus

Copy this file to **`.github/copilot-instructions.md`** in your workspace so Copilot Chat / agent mode picks it up ([VS Code custom instructions](https://code.visualstudio.com/docs/copilot/customization/custom-instructions)).  
Append **`snippets/context-prime.txt`** from this pack.

**Canonical:** [`pluribus-instructions.md`](../pluribus-instructions.md).

---

# Pluribus — MCP behavior

Before non-trivial reasoning or changes → **`recall_context`**.  
Then **plan → act** (tools, edits, deliverables).  
After meaningful work → **`record_experience`**.  
Do not proceed on complex work without recall when Pluribus MCP is available.  
Legacy tools **`memory_context_resolve`**, **`mcp_episode_ingest`**. No **scope** memory partitions—tags + situation only (`docs/anti-regression.md`).
