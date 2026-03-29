# Work order format

Work orders live in `workorders/WO-xxxx.md` (e.g. `WO-0001.md`). The orchestrator parses and validates them before running the recall and LLM pipeline.

## Required sections

Each work order must include these sections. Use the exact labels below so the parser can extract them.

| Section | Label | Description |
|--------|--------|-------------|
| Goal | `**Goal:**` | One-line goal for the task. |
| Rationale | `**Rationale:**` | Why we are doing this. |
| Files involved | `**Files involved:**` | Paths or areas of the codebase affected (list or summary). |
| Constraints | `**Constraints:**` | What constraints or existing rules apply (or “None”). |
| Acceptance criteria | `**Acceptance criteria:**` | How we know the work is done. |
| What not to break | `**What not to break:**` | Existing behaviour or invariants that must be preserved. |

Optional but recommended:

- **Tags:** — Comma- or space-separated tags (e.g. `api`, `auth`) used for recall (filtering memories by tag).

## Example

```markdown
# WO-0001

**Goal:** Add login endpoint
**Rationale:** Users need auth.
**Tags:** api, auth
**Files involved:** pkg/auth
**Constraints:** Use existing session store.
**Acceptance criteria:** Login returns token; tests pass.
**What not to break:** Existing sessions.

---
Body or notes below the horizontal rule.
```

## Validation

The orchestrator calls `ValidateWorkOrder` after parsing. If any required section is missing or empty, the run fails with a clear error listing the missing sections (e.g. `work order validation failed: missing required sections: files involved, acceptance criteria`).
