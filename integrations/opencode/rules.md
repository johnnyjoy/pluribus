# Pluribus — OpenCode rules (memory substrate)

**Pluribus** is **shared institutional memory** (MCP). It is **MCP-first**; REST is for tests and service integration only.

## Non-negotiables

- **Recall before substantive work:** call **`memory_context_resolve`** (via **`pluribus`** MCP tools) with the task text. Chat is not source of truth.
- **Capture learning:** after meaningful outcomes, use **`mcp_episode_ingest`** or **`memory_log_if_relevant`** when signals are clear. **Ingest channel** is advisory until promotion.
- **Distill mode** on candidates uses **`pluribus_distill_origin`**—do not confuse with ingest channel.
- **No container ontology:** do not treat project / task / workspace / **scope** as memory partitions—see Pluribus **`docs/anti-regression.md`**.
- **Optional depth:** curation / contradictions / relationships tools are pull-based, not required for the default loop.

## Automation bias

- Default pair: **`memory_context_resolve`** + **`mcp_episode_ingest`**.
- Use **`correlation_id`** when you want session-tagged continuity (**`mcp:session:*`** server-side).

## Nudge

In prompts when needed: **`use pluribus`** so OpenCode selects Pluribus MCP tools alongside built-ins.
