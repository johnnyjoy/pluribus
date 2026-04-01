# Pluribus — Cursor / Agent prompt snippets

**Pluribus** governed-memory loop: **`recall_context`** → plan → act → **`record_experience`** when **Pluribus** MCP tools are connected (full triggers: **[`pluribus-instructions.md`](../pluribus-instructions.md)**).

Copy into **Agent** chat (or pin in **User rules**) when you want a deliberate nudge. Keep tasks **one screen**—full text belongs in **Pluribus** **`recall_context`** / **`record_experience`** tool args, not only here.

## Before meaningful work

- **Architecture / API change:** “Call **`recall_context`** first with this task description and our usual tags. Then propose a plan. Do not edit broadly until recall returns.”
- **Incident or regression:** “**`recall_context`** with the failure summary and tags for this service area. Check constraints and known failures before proposing a fix.”
- **Unfamiliar area:** “**`recall_context`** with what I’m trying to do and which parts of the repo I’ll touch. Use the bundle to avoid contradicting existing decisions.”

## After meaningful work

- **Durable lesson:** “**`record_experience`** — short summary of what changed, what we learned, and whether anything should be promoted later.”
- **Decided something new:** “**`record_experience`** — capture the decision and alternatives rejected so the next session does not re-litigate.”

## Review / hygiene (optional)

- **Candidates:** “List or summarize episodic / candidate items relevant to these tags before we promote anything.”
- **Contradictions:** “If enforcement or recall surfaces contradictions, summarize and say whether we need curation or a new decision.”

## One-liners (Agent nudges)

- “Recall first, then edit.”
- “Record before you close the task.”
- “Empty recall is not permission to skip **`record_experience`** after real work.”
