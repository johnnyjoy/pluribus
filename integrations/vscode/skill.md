# Pluribus — portable skill (**Pluribus** MCP)

**Prime:** **`snippets/context-prime.txt`** into Copilot / workspace instructions.

**Canonical:** **[`pluribus-instructions.md`](../pluribus-instructions.md)** — mandatory **Pluribus** loop when tools connect.

| Step | DO |
|------|-----|
| 1 | **`recall_context`** or **`wakeup_context`** |
| 1b | If **`housekeeping`** / open chores → **`resolve_chore`** with **`agent_id`** (or defer with reason in step 5) |
| 2 | **Plan / reason** |
| 3 | **Act** |
| 3b | If recall surfaced memories you used or rejected → **`memory_feedback`** |
| 4 | **`record_experience`** |

See **[§ Housekeeping](../pluribus-instructions.md#housekeeping-when-chores-exist)**.

**Forbidden:** Skip 1 or 4 on substantive **Pluribus** runs. **Legacy:** **`memory_context_resolve`**, **`mcp_episode_ingest`**.
