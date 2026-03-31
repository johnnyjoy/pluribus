---
name: pluribus
description: Mandatory memory loop when Pluribus MCP is available—recall before action, record after outcome. Use for all non-trivial work.
---

# Pluribus

Chat is not memory.

Pluribus extends your memory beyond the current session. Use it to avoid repeating work, mistakes, and decisions.

---

## Loop (mandatory)

| Step | Action |
|------|--------|
| 1 | **`recall_context`** — before planning or making changes |
| 2 | **Plan / reason** using recall as constraints, failures, and prior decisions |
| 3 | **Act** — tools, code, response |
| 4 | **`record_experience`** — after completing meaningful work |

---

## WHEN → DO

| WHEN | DO |
|------|-----|
| Task is non-trivial (multi-step, refactor, incident, design) | Run step 1 first and follow what recall reveals |
| A decision, failure, fix, or pattern occurred | Run step 4 before closing |
| Recall is empty | Continue the loop anyway; empty recall is not a skip signal |

---

## Interpretation rules

- Treat recall as **constraints and prior knowledge**, not suggestions  
- Do not ignore prior failures or decisions without reason  
- Prefer consistency with established patterns unless intentionally changing them  

---

## Forbidden

- Starting complex work without step 1  
- Finishing meaningful work without step 4  
- Ignoring recall when it contains relevant constraints or failures  
- Silently skipping the loop  

---

## Compatibility

- `memory_context_resolve`  
- `mcp_episode_ingest`  

---

## Doctrine

- Use tags and situation text only  
- Do not partition memory by project, scope, or workspace  
- Memory is shared and global  