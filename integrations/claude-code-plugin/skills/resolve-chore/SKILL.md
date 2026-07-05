---
name: resolve-chore
description: Vote on Pluribus curation chores when wakeup or recall surfaces housekeeping.
---

# Resolve chore (Pluribus housekeeping)

When **`wakeup_context`**, **`recall_context`**, or **`list_chores`** shows open **pool maintenance** (curation chores):

1. Read the chore type, statement snippets, and allowed **`actions`**.
2. Call **`resolve_chore`** with **`chore_id`**, **`action`**, and stable **`agent_id`** (e.g. `claude-code:<hostname>`).
3. If you cannot judge, skip the vote and note deferral in the next **`record_experience`**.

## Actions by type

| Chore type | Actions |
|------------|---------|
| `quarantine_review` | `release` (→ pending) or `delete` |
| `contradiction` | `keep_subject`, `keep_related`, `coexist` |
| `duplicate_pair` | `consolidate` or `distinct` |

Corroboration: **`min_resolvers`** distinct agents must agree; a memory's author never counts.

Canonical reference: [pluribus-instructions.md](../../../pluribus-instructions.md#housekeeping-when-chores-exist)
