# Pluribus pre-change protocol (enforcement gate)

Use **before** endorsing or implementing a **risky** change to datastore, workflow, policy, security, or architecture.

## Mission
Route **binding** (governing) memory against the **proposal** via **`enforcement_evaluate`**. This is **not** general recall refresh and **not** curation.

## What counts as risky (enforce)
- Schema/migration, auth model, data retention, multi-tenant boundaries
- Deployment topology, network policy, secrets handling
- Architectural commitments that are hard to unwind
- Workflow or policy that changes who must approve what

## What is usually not risky (do not over-gate)
- Typos, formatting, comments, renames with no behavior change
- Single-function refactors with identical external behavior when recall already covers constraints
- Test-only changes that do not change production paths

**Default under uncertainty:** If the change could plausibly violate a **constraint**, **decision**, or **failure** already in the bundle, treat it as **risky** and run **`enforcement_evaluate`**.

## Required sequence
1. Ensure you have fresh context (**`recall_get`** / **`recall_compile`** if you have not loaded memory for this proposal context).
2. Call **`enforcement_evaluate`** with **`proposal_text`** (bounded) and optional metadata only when explicitly required by the operation.
3. Interpret server **`validation`** + **`next_action`** first:
   - `proceed` -> continue
   - `revise` -> revise proposal, then rerun recall/enforcement
   - `reject` -> stop this action path
   Also read **`decision`**: **`block`** / **`require_review`** / **`allow`** (and variants).
4. If **`enforcement.enabled`** is false on the server, the API may return **403** — that is a **configuration** outcome, not a green light to ignore governing memory in your own process.

## Distinctions (do not confuse)
| Mechanism | Role |
|-----------|------|
| **`recall_get` / `recall_compile`** | Load working context; not a proposal gate |
| **`enforcement_evaluate`** | Structured gate vs **binding** trusted memory for **this proposal** |
| **`curation_digest`** | Post-work **candidates**; not pre-change gate |
| **`/v1/drift/check`** | Drift / execution risk signals; different contract than enforcement |

## Anti-failure checks
- [ ] Proposal text describes the change concretely (not vague intent).
- [ ] You did not skip enforcement because the team “agreed in chat.”
- [ ] You did not run enforcement for pure trivia (avoid bogus ritual).

## Default
> **If risky → enforce. If trivial → do not waste the gate.**

## No guessing rule
- Do not skip enforcement because context is unclear.
- If context is unclear, run recall first and only then validate.

## Resources
- `pluribus://discipline/lifecycle`
- `pluribus://discipline/canon-vs-advisory`
- `pluribus://discipline/doctrine`

---

## Pluribus doctrine (MCP)

Do not assume any project, task, or container.
This system is memory-first.
Canonical authority in-repo: `docs/memory-doctrine.md` (Recall repository).
Do not guess missing governing memory — recall again instead.
