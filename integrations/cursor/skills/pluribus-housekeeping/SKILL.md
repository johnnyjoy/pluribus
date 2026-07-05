---
name: pluribus-housekeeping
description: Vote on Pluribus curation chores when recall or wakeup surfaces housekeeping.
---

# Pluribus housekeeping (Cursor)

When **`recall_context`**, **`wakeup_context`**, or **`list_chores`** shows hive maintenance work:

1. Read chore type, statements, allowed **`actions`**.
2. Call **`resolve_chore`** with **`chore_id`**, **`action`**, and stable **`agent_id`** (e.g. `cursor:<hostname>`).
3. If you cannot judge, note deferral in the next **`record_experience`**.

| Chore type | Actions |
|------------|---------|
| `quarantine_review` | `release`, `delete` |
| `contradiction` | `keep_subject`, `keep_related`, `coexist` |
| `duplicate_pair` | `consolidate`, `distinct` |

Corroboration: **`min_resolvers`** distinct agents must agree; a memory's author never counts.

Canonical: [pluribus-instructions.md](../../pluribus-instructions.md#housekeeping-when-chores-exist)
