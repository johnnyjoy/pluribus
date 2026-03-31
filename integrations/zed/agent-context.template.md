# Zed — Pluribus agent context

Zed does not use a single repo-wide instruction file like Cursor’s **`pluribus.mdc`**. Paste the block below into your **Agent** custom instructions / default prompt (wording varies by Zed version), or keep it in project docs and reference it from your team’s onboarding.  
Append **`snippets/context-prime.txt`** from this pack.

**Canonical:** [`pluribus-instructions.md`](../pluribus-instructions.md).

---

# Pluribus — MCP behavior

Before non-trivial reasoning or changes → **`recall_context`**.  
Then **plan → act** (tools, edits, deliverables).  
After meaningful work → **`record_experience`**.  
Do not proceed on complex work without recall when Pluribus MCP is available.  
Legacy tools **`memory_context_resolve`**, **`mcp_episode_ingest`**. No **scope** memory partitions—tags + situation only (`docs/anti-regression.md`).
