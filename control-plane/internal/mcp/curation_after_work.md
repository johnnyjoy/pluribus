# Pluribus memory curation protocol

Use **after** a unit of work that produced **durable, reusable** learning — not after every micro-edit.

## Mission
Capture **candidates** for promotion via **`curation_digest`**, review server output, then **`curation_materialize`** only what passes your validation bar. **Candidates are not canon.**

## What counts as meaningful work (digest-worthy)
- New or changed **constraints**, **decisions**, **failures**, or patterns worth reusing
- Lessons that would help **other agents** later
- Validated outcomes tied to evidence when your process requires it

## What is usually noise (do not digest)
- Routine typo fixes, comment-only edits, dependency bumps with no lesson
- “We discussed X” with no durable statement worth storing
- Raw chat or transcript dumps — **history is not memory** (see resource E).

**Default under uncertainty:** **Do not digest.** Prefer one strong later capture over habitual noise.

## Required sequence
1. **`curation_digest`** with bounded **`work_summary`** and optional fields per **`DigestRequest`**.
2. Inspect **`proposals`** and **`rejected`**. Treat **`proposals`** as **pending candidates** only.
3. **`curation_materialize`** only for candidates you **validate** (policy/evidence/review as your workflow requires — server still enforces its gates).
4. Optionally **`recall_get`** or **`recall_compile`** to confirm durable memory reflects the materialization when needed.

## Authority
| Kind | Status |
|------|--------|
| Digest output | **Candidate** — not durable canon until materialized (or other server paths) |
| Durable rows | What the server persists; **recall** assembles a **working slice**, not a dump of everything ever said |

## Anti-failure checks
- [ ] You are not digesting **trivia** to “pad” memory.
- [ ] You treat **`proposals`** as **review items**, not auto-truth.
- [ ] You did not confuse **chat log** with **curated memory**.

## Default
> **Meaningful learning → digest → review → materialize selectively. Routine churn → skip.**

## Resources
- `pluribus://discipline/lifecycle`
- `pluribus://discipline/canon-vs-advisory`
- `pluribus://discipline/history-not-memory`
- `pluribus://architecture/active-context-vs-durable-store`

---

## Pluribus doctrine (MCP)

Do not assume any project, task, or container.
This system is memory-first.
Canonical authority in-repo: `docs/memory-doctrine.md` (Recall repository).
Do not guess missing governing memory — recall again instead.
