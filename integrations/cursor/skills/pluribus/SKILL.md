---
name: pluribus
description: Memory loop — recall before work, record after outcomes.
---

# Pluribus Skill

## Loop

1. recall_context (before work)
2. Assemble working context from agent_grounding (keep only what constrains this task). Do not treat the JSON dump as context.
3. If housekeeping is present: `resolve_chore` with `chore_id`, `action`, and stable `agent_id` when you can judge, or defer with reason in `record_experience`
4. plan / act
5. Kept and used: pass ids as `used_memory_ids` on `record_experience` (server upvotes). Do not also call `memory_feedback` helpful for the same IDs. Misled/wrong/outdated: `memory_feedback` with reason.
6. record_experience (after outcome)

## Use When

- Any non-trivial task
- Debugging
- Repeated issues
- Learning new patterns

## Principle

If you do not recall, you risk repeating mistakes.  
If you do not record, you lose what you learned.
