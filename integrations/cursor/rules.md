# Pluribus — Cursor rules (memory substrate)

**Scope:** This project uses **Pluribus** as **shared institutional memory** (MCP). Pluribus is **MCP-first**; REST is for tests and service integration.

## Non-negotiables

- **Recall before substantive work:** call **`recall_context`** (alias **`memory_context_resolve`**) with raw task text. Do not treat chat as source of truth.
- **Capture learning:** after meaningful outcomes, use **`record_experience`** (alias **`mcp_episode_ingest`**) (summary) or **`memory_log_if_relevant`** when deterministic signals apply. **Ingest channel** is advisory until promotion.
- **Distill mode** appears on candidates as **`pluribus_distill_origin`**—do not confuse with ingest channel.
- **No container ontology:** do not require project/task/workspace/**scope** as memory partitions—see repo `docs/anti-regression.md`.
- **Optional depth:** curation / contradictions / relationships tools are **pull-based**, not required for the default improvement loop.

## Automation bias

- Prefer the **default tool pair** for routine work: **`recall_context`** + **`record_experience`** (compat names still work).
- Use **`correlation_id`** on ingest/recall when you want **session-tagged** continuity (`mcp:session:*` tagging server-side).
