# Pluribus MCP discipline doctrine

This is the practical operating doctrine for agents using `pluribus-mcp`.

Goal: make correct behavior likely without turning MCP into a hidden workflow engine.

## Product stance

**Pluribus is a global memory system.** Request JSON must match the **`json` tags** on the Go structs for each route ([http-api-index.md](http-api-index.md)). [api-contract.md](api-contract.md) documents a **subset** with narrative examples — not every field on every route. Mental model: [pluribus-memory-first-ontology.md](pluribus-memory-first-ontology.md).

## Core stance

- MCP stays a **thin proxy** to control-plane HTTP.
- Tool descriptions should still teach **when** and **why** to use each tool.
- Canon vs advisory distinctions must stay explicit.

## Lifecycle map

| Phase | Primary tools | Why here | Typical next step |
|------|----------------|----------|-------------------|
| **Memory grounding** / context reset | **`recall_get`** or **`recall_compile`** first (tags + `retrieval_query` as needed) | Reconstruct governing context before substantive work | **`enforcement_evaluate`** before a risky proposal |
| Pre-change gate | **`enforcement_evaluate`** | Check bounded proposal against **binding** trusted memory | Revise proposal, seek review, or proceed |
| Post-work capture | **`curation_digest`** | Turn outcomes into pending candidates (**`work_summary`** required; other fields per **`DigestRequest`** in `internal/curation/types.go`) | Review `proposals` / `rejected` |
| Durable learning | **`curation_materialize`** | Promote validated candidate to durable memory | Optional **`recall_get`** to verify |
| Explicit seeding / admin | **`memory_create`**, **`memory_promote`** | Direct memory operations | Continue normal lifecycle |

**HTTP-only routes:** Drift, preflight, compile-multi, contradictions, evidence, ingest, advisory episodes, and several memory admin paths are **REST only** — see [http-api-index.md](http-api-index.md). The default MCP **tool list** is the narrow agent slice ([`control-plane/internal/mcp/tools.go`](../control-plane/internal/mcp/tools.go)).

## Canon vs advisory discipline

- **`enforcement_evaluate`** checks **binding** memory semantics (constraint/decision/failure/object_lesson subset, active, authority threshold, non-advisory applicability).
- **`curation_digest`** output is **candidate**, not canon.
- Canon happens at **`curation_materialize`** (or other explicit server-side promotion paths).
- Evidence can support memory interpretation; it does not become policy by itself.

## Sequence examples

### 1) Memory grounding (normal)

1. **`recall_get`** or **`recall_compile`** with **`tags`** + **`retrieval_query`** / **`query`** as needed — fields must match **`CompileRequest`** / GET query rules ([http-api-index.md](http-api-index.md)).
2. If the proposal is risky: **`enforcement_evaluate`**.

### 2) Before risky architectural change

1. **`recall_get`** / **`recall_compile`** (tags + optional execution metadata).
2. **`enforcement_evaluate`** with bounded **`proposal_text`**.
3. If `block`/`require_review`: revise or escalate before coding.

### 3) After meaningful work

1. **`curation_digest`** with **`work_summary`** (+ optional `curation_answers`, `evidence_ids`, `artifact_refs`, `signals`, `options` per **`DigestRequest`**).
2. Inspect **`proposals`** and **`rejected`**.
3. **`curation_materialize`** chosen **`candidate_id`**.
4. Optional **`recall_get`** to verify the new durable memory appears.

### 4) Evidence-supported interpretation

1. **`recall_get`** / **`recall_compile`**
2. Use returned memory context plus linked evidence via regular evidence endpoints/workflows as needed.

## Anti-skip trigger cues

- **Grounding recall cue:** before substantive recommendations, run **`recall_get`**/**`recall_compile`** when continuity/constraints/failures should govern the answer.
- **Situation-shift recall cue:** if the situation changes materially (new subsystem, new risk domain), refresh recall once before continuing.
- **Risk cue for enforcement:** if proposing architecture/datastore/policy/process changes that could conflict with trusted memory, run **`enforcement_evaluate`** before endorsing or implementing.
- **Learning cue for curation:** if work produced a reusable decision/constraint/failure/object-lesson, run **`curation_digest`**; materialize only validated candidates.

## Avoid bogus ritual

- **Do not gate trivia:** formatting-only edits, typo fixes, or low-risk local refactors usually do not need **`enforcement_evaluate`**.
- **Do not digest noise:** routine/no-op updates without reusable lessons should not be promoted into durable memory.
- **Do not confuse candidate with canon:** **`curation_digest`** output is advisory until **`curation_materialize`**.

## What this doctrine is not

- Not an orchestration engine.
- Not permission to invent fields not in server contracts.
- Not a replacement for endpoint docs.

## Related docs

- [mcp-poc-contract.md](mcp-poc-contract.md)
- [pre-change-enforcement.md](pre-change-enforcement.md)
- [curation-loop.md](curation-loop.md)
- [../archive/memory-bank/plans/workflow-discipline-proof-results-latest.md](../archive/memory-bank/plans/workflow-discipline-proof-results-latest.md)
