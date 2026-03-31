---
name: pluribus
description: Default agent loop when Pluribus MCP is connected—recall_context (pre-action), then plan and act, then record_experience (post-action). Apply to non-trivial coding, refactors, incidents, and multi-step tasks.
---

# Pluribus

Chat is not memory.

## Loop (mandatory)

| Step | Tool / phase |
|------|----------------|
| 1 | **`recall_context`** — before plan or edits |
| 2 | **Plan / reason** using `mcp_context` + `recall_bundle` |
| 3 | **Act** — tools, code, answer |
| 4 | **`record_experience`** — after meaningful outcome |

## WHEN → DO

| WHEN | DO |
|------|-----|
| Task is non-trivial | Run step 1 first; obey constraints/failures from recall |
| Work produced a durable lesson | Run step 4 before closing |
| Recall empty | Still run 1 → 4; empty recall is not a skip signal |

## Forbidden

- Act on complex work before step 1  
- Skip step 4 after real outcomes  

## Compatibility

**`memory_context_resolve`**, **`mcp_episode_ingest`**.

## Doctrine

Tags + situation text only—no **scope** partitions. **`docs/anti-regression.md`**.
