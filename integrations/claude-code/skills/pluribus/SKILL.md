---
name: pluribus
description: Memory loop — recall before work, housekeeping when chores exist, record after outcomes.
---

# Pluribus Skill

## Loop

1. **`recall_context`** or **`wakeup_context`** (before work)
2. Assemble working context from **`agent_grounding`** (keep only what constrains this task). Do not treat the JSON dump as context.
3. If **`housekeeping`** / open chores → **`resolve_chore`** with **`chore_id`**, **`action`**, **`agent_id`** (or defer in **`record_experience`**)
4. Plan / act
5. Kept and used → **`used_memory_ids`** on **`record_experience`** (server upvotes). Do not also call **`memory_feedback`** helpful for the same IDs. Misled/wrong/outdated → **`memory_feedback`**
6. **`record_experience`** (after outcome)

Canonical detail: [pluribus-instructions.md](../../../pluribus-instructions.md)

## Use When

- Any non-trivial task
- Debugging
- Repeated issues
- Open curation chores surfaced in recall/wakeup

## Principle

If you do not recall, you risk repeating mistakes.  
If you do not record, you lose what you learned.  
If you skip housekeeping when chores exist, the shared pool stays noisy for every agent.
