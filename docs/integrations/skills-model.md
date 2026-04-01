# Skills model — four intents, one skill pack

Integrations use **one** Agent Skill folder per platform (**`skills/pluribus/SKILL.md`**) so users are not buried in copies. The **four behavioral intents** below map to **MCP tools** (and optional HTTP).

| Intent | Tool(s) | When |
|--------|---------|------|
| **Use memory before acting** | **`recall_context`** · **`memory_context_resolve`** | Substantive work per [`pluribus-instructions.md`](../../integrations/pluribus-instructions.md) |
| **Record experience** | **`record_experience`** · **`mcp_episode_ingest`** | After meaningful outcomes |
| **Review learnings** | **`curation_pending`** · **`curation_review_candidate`** · **`curation_digest`** · recall bundles | Inspect candidates before promotion; optional |
| **Promote stable knowledge** | **`curation_materialize`** · **`memory_promote`** (and related curation tools) | After validation; durable memory |

**Pack contents:** each **`integrations/<platform>/skills/pluribus/SKILL.md`** encodes the mandatory table + link to **`pluribus-instructions.md`**. Do not split into four separate skill products unless a platform requires multiple skill IDs.

**Doctrine:** tags + situation only — [memory-doctrine.md](../memory-doctrine.md).
