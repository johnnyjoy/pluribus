---
name: recall-context
description: Retrieve durable Pluribus context before substantive work. Use when editing multiple files, changing APIs or schema, debugging failures, handling security or compliance, or any task needing more than two sentences to explain to a teammate.
---

# Recall context (Pluribus)

Before planning or editing:

1. Call the Pluribus MCP tool **`recall_context`** (alias `memory_context_resolve` if your client lists that name).
2. If recall/wakeup surfaced **`housekeeping`** or open chores, call **`resolve_chore`** with **`agent_id`** when you can judge — see skill **`resolve-chore`**.
3. Pass **tags** (situation / topic, not workspace silos) and **`retrieval_query`** or task text that states what you are about to do.
4. Read **`recall_bundle`** / **`mcp_context`** and fold constraints and prior failures into your plan.

**Do not** skip recall because the task “seems small” once you have read a few files—if it matches the trigger list above, recall first.

**Do not** defer recall to “later” on substantive refactors or incidents when Pluribus MCP is available.

Server-side ranking and authority stay on the Pluribus control plane; you only request and apply context.
