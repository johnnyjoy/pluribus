# Memory curation: style guide and anti-patterns

Memory (constraints, decisions, patterns, failures) is **curated operational knowledge**: it should be clear, machine-recallable, and useful for the LLM and drift checks. It is **not** a diary or a dump of everything that happened.

## What to aim for: machine-recallable units

- **Decision** — One clear statement + reason; tags and authority so recall can rank and filter. Example: *"Use POST for create endpoints; GET is read-only."* with reason *"REST convention; avoids cache mutations."*
- **Constraint** — A must-follow rule. Example: *"No global mutable state in the core package."*
- **Pattern** — A preferred approach. Example: *"Return errors with wrapped context using fmt.Errorf and %w."*
- **Failure** — What not to repeat. Example: *"Do not block the event loop with sync I/O."*

Include **tags** and **scope** so recall can surface the right memories for the current work order. Prefer **one idea per memory**; split long lists into separate entries.

## Anti-patterns: what not to do

- **Long rambly summaries** — Memory is not a paragraph of prose. Keep statements to one or two sentences. Put narrative in docs or ADRs if needed.

- **Repeated prose** — Don’t copy the same text into many memories. One canonical memory per idea; reference it by tag or scope.

- **Vague aspirations** — *"We should improve performance"* is not recallable. Prefer *"Prefer batch DB calls over N+1 in list handlers"* (constraint or pattern).

- **"We discussed several options" without a decision** — If there was a discussion, record the **outcome**: the decision that was made and the reason. Unresolved discussions don’t belong in memory as-is.

- **Giant transcripts** — Do not paste meeting logs or long chat transcripts into memory. Extract decisions, constraints, or failures and write them as short, structured entries.

- **Diary-style logging** — Memory is not "what we did today." It is "what we decided, what we must follow, what we must avoid." Skip routine status updates.

## Style guide

1. **Statement** — One clear, actionable sentence. Use present tense or imperative (*"Use X"*, *"Do not Y"*).
2. **Reason** — In `data.reason` or in the statement, briefly say why (so the LLM and future readers can trust it).
3. **Tags** — Use consistent, lowercase tags (e.g. `api`, `auth`, `storage`) so recall and filters work.
4. **Authority** — Higher for human-curated or non-negotiable rules (e.g. 8–10); lower for auto-promoted or tentative (e.g. 6).

When in doubt: if it doesn’t help the LLM or drift check on a future work order, don’t put it in memory. Put it in a doc, a comment, or an ADR instead.
