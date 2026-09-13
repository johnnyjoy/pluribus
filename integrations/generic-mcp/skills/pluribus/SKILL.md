---
name: pluribus
description: Memory loop — recall before work, record after outcomes.
---

# Pluribus Skill

## Loop

1. **`wakeup_context`** or **`recall_context`** (before work)
2. Assemble working context from **`agent_grounding`** (keep only what constrains this task). Do not treat the JSON dump as context.
3. If **`housekeeping`** or open chores → **`resolve_chore`** with **`agent_id`** (see [pluribus-instructions.md](../../../../pluribus-instructions.md))
4. plan / act
5. Kept and used → **`used_memory_ids`** on **`record_experience`** (server upvotes). Do not also call **`memory_feedback`** helpful for the same IDs. Misled/wrong/outdated → **`memory_feedback`**
6. **`record_experience`** (after outcome). Optional **`occurred_at`** (RFC3339). One dense claim, not an essay. Clock questions: pass **`occurred_after` / `occurred_before`**.

## Use When

- Any non-trivial task
- Debugging
- Repeated issues
- Learning new patterns

## Principle

If you do not recall, you risk repeating mistakes.  
If you do not record, you lose what you learned.
