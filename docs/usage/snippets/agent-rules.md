# Agent rules — minimal copy-paste (Pluribus)

Paste into editor rules, `AGENTS.md`, `instructions`, or system prompts. Tighten for your stack.

---

## Memory is default, not optional

- **Before** architectural or multi-step work, call **`recall_context`** (MCP; alias **`memory_context_resolve`**) or **`POST /v1/recall/compile`** (REST) with real task text and relevant **tags**. Do not rely on chat history alone.
- **After** meaningful outcomes (fixes, decisions, incidents, repeated confusion), record an advisory episode: **`record_experience`** (MCP; alias **`mcp_episode_ingest`**) or **`POST /v1/advisory-episodes`** (REST). Prefer short, factual summaries.

## Shared substrate

- Treat Pluribus as **one global pool** shaped by **tags** and **retrieval text**—not a private scratchpad per session. See [memory-doctrine.md](../../memory-doctrine.md).
- Do **not** use project / task / workspace / **scope** as required memory partitions—see [anti-regression.md](../../anti-regression.md).

## When uncertain

- If requirements conflict or risk is high, **recall first**, then **`enforcement_evaluate`** before large edits when the product calls for it.

## Non-destructive evolution

- Candidates and promotion are how learning becomes durable; there is no “delete the bad memory” user story—see doctrine.
