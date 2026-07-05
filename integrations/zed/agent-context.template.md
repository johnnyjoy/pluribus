# Zed — Pluribus agent context

Zed does not use a single repo-wide instruction file like Cursor’s **`pluribus.mdc`**. Paste the block below into your **Agent** custom instructions / default prompt (wording varies by Zed version), or keep it in project docs and reference it from your team’s onboarding.  
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
