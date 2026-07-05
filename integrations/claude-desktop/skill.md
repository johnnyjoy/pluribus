# Pluribus — portable skill (**Pluribus** MCP)

**Prime:** paste **`snippets/context-prime.txt`** into system instructions. **Template:** **`custom-instructions.template.md`**.

**Canonical:** **[`pluribus-instructions.md`](../pluribus-instructions.md)** — mandatory **Pluribus** loop when tools connect.

| Step | DO |
|------|-----|
| 1 | **`recall_context`** or **`wakeup_context`** |
| 1b | If **`housekeeping`** / open chores → **`resolve_chore`** with **`agent_id`** (or defer with reason in step 5) |
| 2 | **Plan / reason** |
| 3 | **Act** |
| 3b | If recall surfaced memories you used or rejected → **`memory_feedback`** |
| 4 | **`record_experience`** |

See **[§ Housekeeping](../pluribus-instructions.md#housekeeping-when-chores-exist)** in canonical instructions.

**Forbidden:** Skip 1 or 4 when **Pluribus** MCP is live. **Legacy:** **`memory_context_resolve`**, **`mcp_episode_ingest`**.
