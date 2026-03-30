# Memory doctrine (canonical authority)

This document is the **highest-authority product model** for Pluribus and this repository. If any other doc, prompt, example, or API comment disagrees, **this file wins** unless this file is explicitly updated.

**Final line:** Memory is the model. Everything that treats durable knowledge as owned by a container, silo, or ticket is wrong.

---

## A. What a memory is

Memory, in Pluribus, is:

- **Distilled** — a compact, reusable statement (constraint, decision, pattern, failure, state, experience), not raw narrative.
- **Durable** — stored in the authoritative store (Postgres via the control-plane), not in chat or logs.
- **Portable** — reusable across agents, sessions, and machines; not trapped in a private “bucket.”
- **Behavior-changing** — ranked by **authority**; **constraints** can override weaker material; recall and enforcement act on it.

---

## B. What memory is not

Memory is **not**:

- A **transcript** or chat log.
- A **log** or telemetry stream used as policy.
- **Container-owned state** — nothing “owns” governing memory except the global pool and server rules.
- **Ticket- or backlog-scoped data** — work-tracking artifacts are not the substrate of truth.

---

## C. Core principles

1. **Memory is global** — one shared pool; context is not a partition of the database into separate truths.
2. **Tags are context** — use tags (and kinds) to shape retrieval, not hidden ownership IDs.
3. **Recall is situational** — each request assembles a **situation-shaped slice** (hybrid retrieval), not “everything in a silo.”
4. **Authority determines importance** — higher authority wins in ranking and conflict handling where the model defines it.
5. **Constraints override** — binding constraints beat habit, chat momentum, and weak patterns.
6. **Patterns generalize** — patterns capture reusable success; they do not replace constraints.
7. **Experiences may be retained and distilled** — experience-shaped memory is first-class when promoted with discipline.

---

## D. Forbidden concepts (product ontology)

The following concepts **must never** be reintroduced as the **model for durable memory**, **recall partitioning**, or **required client mental load**:

| Forbidden | Why |
|-----------|-----|
| **project** (as memory partition) | Implies memory “belongs to” a software project or container. |
| **task** (as memory partition) | Implies truth is scoped to a ticket/unit of work. |
| **workspace** (as memory partition) | Implies a silo is the source of governing memory. |
| **hive** | Historical container synonym; same failure mode as workspace partitioning. |
| **scope** (as container abstraction) | Implies mandatory silo semantics for recall or storage truth. |
| **Any container abstraction** for memory | Same class of error: hiding the global pool behind ownership. |

> **These must never be reintroduced** as the way agents are taught to think about **what recall searches** or **what enforcement reasons over**.

Implementation detail may still carry **optional correlation** fields in narrow HTTP paths for legacy foreign keys; those fields **do not** define the doctrine and **must not** be required for core recall, enforcement, or memory create/search. New work must not add **required** container IDs to those flows.

---

## E. Retrieval model

- **Hybrid retrieval** — combine semantic similarity, lexical/tag overlap, authority, and constraint relevance as implemented by the server.
- **No container scoping** — recall is **not** “search inside project X”; it is “search the global pool for this **situation** (query + tags + policy).”
- **Situation, not silo** — the client describes the **situation**; the server ranks memory.

---

## F. Operational loop (behavior)

The intended loop is:

**Experience → memory (distilled) → behavior (recall + enforcement) → learning (curation / promote) → repeat.**

**Advisory episodes** may be **distilled** into **candidate** rows (`POST /v1/episodes/distill`) as *possible* structured learning; those candidates are **not** memory until curated and materialized. Recall comes **before** substantive action; enforcement gates **risky** proposals; curation captures **validated** learning, not noise.

### Controlled promotion (automation without guessing)

- **Truth stays explicit** — durable memory created from candidates carries **traceability** (`payload.pluribus_promotion`: originating `candidate_id`, supporting advisory episode ids, `distill_support_count_at_promotion`).
- **Automation is gated** — optional **`promotion.auto_promote`** with conservative thresholds (`min_support_count`, `min_salience`, `allowed_kinds`); default **off**. **`POST /v1/curation/auto-promote`** returns **403** when disabled.
- **Guardrails** apply before **any** materialize (manual or auto): validation rejects vague statements, missing evidence when required, duplicate statement keys (unless **`supersedes_memory_id`** matches the duplicate), and inconsistent signals.
- **No destructive retire** — there is no retire/archive endpoint. Memory is **permanent** in the store; influence changes through **authority, ranking, contradiction policy, supersession**, and **additive payload relationships** — not subtraction.
- **No recall / enforcement drift from candidates** — candidates do not affect recall or enforcement until **materialized**; ranking and compile behavior are unchanged by this path.

### Memory evolution (non-destructive)

- **Corrections are additive** — express change with **new** memories, stronger authority, and optional **`payload.pluribus_evolution`**: `superseded_by`, `contradicts`, `invalidated_by` (memory UUID strings). No agent-facing API archives or hides rows to “undo” truth.
- **Supersession** — **`POST /v1/memory`** accepts **`supersedes_id`**; materialize accepts **`supersedes_memory_id`** on the candidate proposal when replacing a row with the same statement key. Prior row moves to **`superseded`** (relationship + lifecycle), not deleted.
- **Invalidation signal** — `pluribus_evolution.invalidated_by` keeps the row **auditable**; recall scoring applies a **deprioritization penalty** so influence drops without erasure.

Details: [curation-loop.md](curation-loop.md) (Controlled Promotion + Memory evolution).

---

## Timing (canonical memory)

- **`created_at` / `updated_at`** — system timestamps when the row was created or last updated.
- **`occurred_at`** (optional) — when the **described event or fact** took place. Omitted means “unspecified”; ranking and recency use **`coalesce(occurred_at, updated_at)`** for temporal honesty. This does **not** turn canonical memory into a diary; it is still constraints, decisions, patterns, failures, and state — distinct from **advisory** episodic similarity ([episodic-similarity.md](episodic-similarity.md)).

---

## See also

- [anti-regression.md](anti-regression.md) — enforcement and review rules.
- [architecture.md](architecture.md) — system shape aligned with this doctrine.
- [pluribus-memory-first-ontology.md](pluribus-memory-first-ontology.md) — narrative companion (must stay consistent with this file).
- [episodic-similarity.md](episodic-similarity.md) — **advisory** episodic recall only (“what happened / when / involving whom”); subordinate to canonical memory and not enforcement truth.
