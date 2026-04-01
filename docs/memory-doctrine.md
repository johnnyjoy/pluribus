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

### Agent usage (MCP default loop)

Pluribus is most effective when agents use a **before/after loop**: **`recall_context`** (or equivalent compile path) **before** complex reasoning or multi-step work, and **`record_experience`** **after** meaningful outcomes, failures, or reusable discoveries. Chat is not memory; the loop connects session work to the shared pool.

### Ingestion vs ranking (probationary memory)

**Be generous at ingestion; be ruthless at ranking.** Probationary rows formed at **`POST /v1/advisory-episodes`** are accepted when the text is **plausibly useful** (keyword signals, `mcp:event:*`, experiment/benchmark language, or long situational context)—not only when confidence is already high. Clear noise still goes to the **reject bucket** (`advisory_experiences`). New ingest starts at **low authority (1–2)**; **recall ranking**, reinforcement, consolidation, and contradiction policy **separate** durable signal from weak material over time—intake filtering is not the primary quality gate.

**Advisory episodes** may be **distilled** into **candidate** rows (`POST /v1/episodes/distill`) as *possible* structured learning; those candidates are **not** memory until curated and materialized. Recall comes **before** substantive action; enforcement gates **risky** proposals; curation captures **validated** learning, not noise.

### Controlled promotion (automation without guessing)

- **Truth stays explicit** — durable memory created from candidates carries **traceability** (`payload.pluribus_promotion`: originating `candidate_id`, supporting advisory episode ids, `distill_support_count_at_promotion`).
- **Automation is gated** — optional **`promotion.auto_promote`** with conservative thresholds (`min_support_count`, `min_salience`, `allowed_kinds`); default **off**. **`POST /v1/curation/auto-promote`** returns **403** when disabled.
- **Guardrails** apply before **any** materialize (manual or auto): validation rejects vague statements, missing evidence when required, inconsistent signals, and **`supersedes_memory_id`** mismatches when a duplicate statement key is explicitly superseded. **Duplicate statement keys without supersession** are **not** rejected: promotion **converges** into the existing canonical row when **`promotion.canonical_consolidation`** is enabled (deterministic match), or **reinforces authority** on exact dedup via the memory create path when consolidation is off.
- **No destructive retire** — there is no retire/archive endpoint. Memory is **permanent** in the store; influence changes through **authority, ranking, contradiction policy, supersession**, and **additive payload relationships** — not subtraction.
- **No recall / enforcement drift from candidates** — candidates do not affect recall or enforcement until **materialized**; ranking and compile behavior are unchanged by this path.

### Canonical convergence (truth gains strength, not copies)

- **Repeated validated lessons** should **strengthen** an existing canonical statement when the server can **match** them deterministically (normalized **statement key**, bounded **lexical** similarity on canonical text, **tag** / **entity** overlap, same **kind**) — not spawn endless near-duplicate rows.
- **Non-destructive** — nothing is deleted. The **dominant** row gains **bounded** authority increments and **`payload.pluribus_consolidation`** lineage (`support_count`, `reinforcing_candidates`, last reason / jaccard). Weaker signals remain traceable through candidate ids and payload, not through row removal.
- **Contradiction** — when a guard detects **opposing** canonical statements (e.g. negation heuristic), materialize may **create a new** memory and record a **`contradicts`** edge; dominance is decided by **scoring + relationships**, not overwrite.
- **Explainable** — every merge decision is **reproducible** from inputs and config (no LLM consolidation).

### Memory evolution (non-destructive)

- **Corrections are additive** — express change with **new** memories, stronger authority, and optional **`payload.pluribus_evolution`**: `superseded_by`, `contradicts`, `invalidated_by` (memory UUID strings). No agent-facing API archives or hides rows to “undo” truth.
- **Supersession** — **`POST /v1/memory`** accepts **`supersedes_id`**; materialize accepts **`supersedes_memory_id`** on the candidate proposal when replacing a row with the same statement key. Prior row moves to **`superseded`** (relationship + lifecycle), not deleted.
- **Invalidation signal** — `pluribus_evolution.invalidated_by` keeps the row **auditable**; recall scoring applies a **deprioritization penalty** so influence drops without erasure.

Details: [curation-loop.md](curation-loop.md) (Controlled Promotion + Memory evolution).

### Lightweight memory relationships (additive, not a graph product)

- **Typed edges** between **canonical** `memories` rows are stored in **`memory_relationships`** (`supports`, `contradicts`, `supersedes`, `same_pattern_family`, `derived_from`). They **annotate** how truths connect; they **do not** replace memory as the unit of recall or enforcement.
- **REST:** `POST /v1/memory/relationships`, `GET /v1/memory/{id}/relationships`. **Automation (bounded):** creating a memory with **`supersedes_id`** also records a **`supersedes`** edge (new → prior row). No container ontology; no graph traversal requirement for basic recall.
- **Recall:** table-backed **`supersedes`** edges **merge** into the existing **pattern supersession** map used for elevation suppression — a **small**, explainable tie-in; ranking is not rewritten around arbitrary graph walks.

---

## Timing (canonical memory)

- **`created_at` / `updated_at`** — system timestamps when the row was created or last updated.
- **`occurred_at`** (optional) — when the **described event or fact** took place. Omitted means “unspecified”; ranking and recency use **`coalesce(occurred_at, updated_at)`** for temporal honesty. This does **not** turn canonical memory into a diary; it is still constraints, decisions, patterns, failures, and state — distinct from **advisory** episodic similarity ([episodic-similarity.md](episodic-similarity.md)).

---

## Terminology (ingest channel vs distill mode)

Pluribus distinguishes **how an advisory episode entered the system** from **how a pending candidate was produced by distillation**. The **JSON field names and stored values are unchanged**; this section is the canonical **conceptual** vocabulary.

### Ingest channel (advisory episodes)

**Concept:** **ingest channel** — the path or kind of ingestion for an **`advisory_episodes`** row.

**Wire / storage:** the JSON and Postgres field is still **`source`**. Examples of stored values: `manual`, `digest`, `ingestion_summary`, `mcp`. In prose, say “ingest channel **`mcp`**” rather than overloading the word “source” when explaining behavior.

### Distill mode (pending candidates)

**Concept:** **distill mode** — how **`candidate_events`** rows were created or updated by the distillation pipeline (explicit distill vs automatic run after ingest vs merge).

**Wire / storage:** the JSON field inside **`proposal_json`** is still **`pluribus_distill_origin`**.

**Conceptual mapping** (stored wire strings are fixed; names on the right are documentation-only):

| Stored `pluribus_distill_origin` | Distill mode (concept) |
|----------------------------------|-------------------------|
| `manual` | **explicit** (explicit `POST /v1/episodes/distill`) |
| `auto` | **auto_from_advisory** (after `POST /v1/advisory-episodes`) |
| `auto:mcp` | **auto_from_advisory_mcp** (after ingest when ingest channel is `mcp`) |
| `mixed` | **mixed** (merged provenance) |

### Tags

The distilled tag **`origin:mcp`** means the episode ingest channel was **`mcp`**. In documentation, **ingest:mcp** may be used as a **semantic alias** for the same idea. **Only** `origin:mcp` is emitted in `proposal_json` tags today (no dual-tag emission).

---

## See also

- [anti-regression.md](anti-regression.md) — enforcement and review rules.
- [architecture.md](architecture.md) — system shape aligned with this doctrine.
- [pluribus-memory-first-ontology.md](pluribus-memory-first-ontology.md) — narrative companion (must stay consistent with this file).
- [episodic-similarity.md](episodic-similarity.md) — **advisory** episodic recall only (“what happened / when / involving whom”); subordinate to canonical memory and not enforcement truth.
