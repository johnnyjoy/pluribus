# Pluribus authority classes (read before tools)

**One job:** keep **authority** straight so gates and curation are not misused.

## Classes

| Class | Meaning | Typical use |
|-------|---------|-------------|
| **Governing / binding** | Strong enough to shape planning and participate in **enforcement** (server selects participation) | Constraints, decisions, failures, patterns as configured |
| **Advisory** | Contextual signal; **does not** override binding semantics in enforcement | Lower authority rows, hints |
| **Candidates** | Output of **`curation_digest`** — **pending** until **`curation_materialize`** (or other promotion paths) | Review before promote |
| **Evidence** | Receipts linked to memory; supports scoring and explanation; **not** policy by itself | Links, artifacts |
| **History / transcript** | Chat and logs — **not** durable operational memory until curated and promoted | Do not treat as canon |

## Operational rules
- **Enforcement** reasons over **binding** memory vs a **proposal** — not over “whatever was said last in chat.”
- **Digest → candidate → materialize** is the default durable-learning path; **do not** skip review steps your team requires.
- When unsure if a row is binding or advisory, **assume advisory** for blocking others’ work — **use enforcement** or **human review**, not chat consensus.

## Default
> **Binding beats advisory. Candidates beat chat. Evidence supports; it does not replace policy.**

## Resources (deeper)
- Use **`resources/read`** on `pluribus://discipline/canon-vs-advisory` for the canonical **resource** body (table-aligned detail).
- `pluribus://discipline/history-not-memory`
- `pluribus://discipline/doctrine`

---

## Pluribus doctrine (MCP)

Do not assume any project, task, or container.
This system is memory-first.
Canonical authority in-repo: `docs/memory-doctrine.md` (Recall repository).
Do not guess missing governing memory — recall again instead.
