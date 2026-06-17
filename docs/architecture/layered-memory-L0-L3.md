# Pluribus architecture note — layered memory model (L0–L3)

## Status

**Proposed** (high priority — governing direction). **Not implemented** as separate APIs or storage tiers until explicitly scheduled.

This note **extends the mental model** for how agents load context; it does **not** replace **[memory-doctrine.md](../memory-doctrine.md)**. If anything here conflicts with doctrine, **doctrine wins** until doctrine is updated.

---

## Summary

Pluribus will adopt a **layered memory model** to control how context is loaded and used by agents.

This separates:

- always-on behavioral context
- task-specific recall
- deep exploratory memory

The goal is to **stabilize agent behavior without flooding context**.

---

## Core principle

> Reinforcement is global.  
> Layers are threshold-based views of the same memory system.

There is **one memory substrate**, not multiple systems. Layers are **policies for how much of the ranked pool to inject** per call (and under which filters), not separate databases or truths.

---

## Layer definitions

### L0 — Identity (always loaded)

**Purpose:** Define what the agent is.

Includes:

- role / mission
- active goals
- system identity

Characteristics:

- very small
- rarely changes
- not heavily reinforced

**Guardrail:** L0 must **not** override governing memory or constraints (L1). It frames behavior; it does not define durable truth.

---

### L1 — Governing memory (always loaded)

**Purpose:** Shape behavior.

Includes:

- high-authority constraints
- active (non-superseded) decisions
- dominant patterns
- critical failures to avoid

Characteristics:

- reinforcement-driven
- authority-gated
- strictly filtered
- stable but evolvable

---

### L2 — Task context (on demand)

**Purpose:** Support execution.

Includes:

- `recall_compile` / compile-path output
- relevant decisions, failures, patterns

Selection factors:

- relevance to task
- authority
- recency
- relationship proximity

---

### L3 — Deep recall (explicit)

**Purpose:** Exploration and investigation.

Includes:

- broader search results
- historical memory
- episodic content

Characteristics:

- least restrictive **inclusion** into the bundle (weaker filtering for breadth)
- reinforcement still applies; advisory / probationary paths follow **server** rules — “deep” is not “unqualified.”

---

## Behavioral model

- **L0 + L1** define baseline behavior.
- **L2** informs current task execution.
- **L3** supports deeper reasoning when needed.

---

## Reinforcement model

Reinforcement applies to all memory globally via:

- authority
- support_count
- success/failure signals
- recency
- supersession

Layer inclusion is determined by **thresholds**, not separate systems.

---

## Key rules

1. A memory may exist in L3 but never reach L1.
2. L1 must remain **small, stable, and high-signal**.
3. L2 is dynamic and task-dependent.
4. L0 changes rarely and deliberately.
5. Superseded memory must not appear in L1.

---

## Wake-up (L0 / L1) — shipped

### Endpoint

```http
POST /v1/recall/wakeup
```

**Returns:** **`identity`** (L0) and **`governing_memory`** (L1) — each is an array of normal **`MemoryItem`** objects (same schema as recall compile), not a parallel type system.

**Status:** **Shipped.** Implementation is a **projection** of **`Compiler.Compile`**: same memory search, authority ordering, contradiction handling, RIU/ranking when configured, and evidence-in-bundle hydration. Wake-up sets **`retrieval_query`** empty (no lexical/semantic expansion), **`skip_experience_hydration`** so promoted JSONL lines are not prepended, and **does not** hit the compile Redis cache or **`reuse_recall`** reinforcement (session start stays cheap and does not distort global reinforcement).

**Invariants:**

- **Not a second store** — no wake-up tables or shadow memories.
- **Not a second ranker** — selection is compile output + **applicability filter** + **caps**.
- **L0** — `kind: state` rows taken from the bundle’s **`continuity`** slice (compile already ranks `state` + `decision` there; wake-up keeps **state only**).
- **L1** — rows from **`governing_constraints`**, **`decisions`**, **`known_failures`**, **`applicable_patterns`** that pass **`applicability`** inclusion: **`governing`** or empty (empty matches durable defaults where unset at create). **`advisory`**, **`analogical`**, **`experimental`** are excluded from **`governing_memory`**.
- **Superseded / inactive** memories do not appear because **`Search`** uses **`status: active`** like compile.

Request caps: **`max_state`**, **`max_per_kind`**, **`max_governing_total`** (see `WakeupRequest` in `control-plane/internal/recall/wakeup.go`).

### Selection model (unchanged doctrine)

Layer membership is determined by a **single scoring / routing function** over the global pool (not separate stores):

```
layer = f(
  authority,
  validity,
  supersession,
  reinforcement,
  relevance,
  recency,
  task_fit
)
```

Wake-up adds only **post-compile thresholds** (kind filter, applicability filter, item caps). **`POST /v1/recall/compile`** remains the path for **L2-style** situational recall (non-empty situation text, optional triggered recall, larger bundles).

### API / routing

- **`POST /v1/recall/wakeup`** — session-start **L0/L1** slice.
- **`POST /v1/recall/compile`** — full **`RecallBundle`** when task text and deeper recall are needed.
- **L0** may still be **client- or session-injected** for product UX; server-side L0 is **state memories** in Pluribus only.

### Thresholds and caps

- L1: hard **token or item caps**; supersession and authority filters mandatory.
- L2: default **`recall_compile`**-style situational slice; dynamic.
- L3: broader retrieval; explicit user or agent opt-in to avoid context flood.

### Observability

- Metrics: bundle size by layer, authority distribution, superseded items excluded (L1).

---

## Enforcement direction (near-term)

Until full layering is implemented:

- Monitor **recall size growth**.
- Detect **lack of separation** between governing vs task memory in compiled outputs.
- **Prefer high-authority memory** in compiled outputs where the server already exposes authority and ranking.

---

## Rationale

Without layering:

- context grows uncontrollably
- behavior drifts
- decisions are not consistently enforced

With layering:

- agents remain grounded
- context is controlled
- memory becomes **operational**, not only searchable

---

## Decision

Adopt the layered memory model as a **core architectural direction**, but **defer full implementation** until current integration work stabilizes. Ship **incrementally** (metrics, caps, authority preference) where possible.

---

## Durable record (Pluribus)

This direction should exist in the **shared memory pool**, not only in git, so **`recall_context` / compile** can surface it during later API and recall design.

**Ingest path:** MCP **`memory_create`** or **`POST /v1/memory`** with **`applicability`: `governing`**.

| Kind | Role | Tags (typical) |
|------|------|----------------|
| **decision** | Locks in adoption of L0–L3 with reinforcement-based thresholds (one substrate; layers = thresholds). | `architecture`, `recall`, `layering` |
| **pattern** | “Reinforcement is global; layers are threshold-based views of the same memory system.” | `architecture`, `recall`, `layering`, `reinforcement` |

Use **high authority** on the **decision** (e.g. 8); **pattern** slightly lower is fine if you want rank ordering. **UUIDs are per deployment**—do not embed them in this file; re-create on a fresh database if needed.

---

## Tags

`architecture` · `memory` · `recall` · `layering` · `reinforcement`

---

## See also

- **[memory-doctrine.md](../memory-doctrine.md)** — canonical product model (one pool, tags + situation, authority); §E retrieval model (hybrid retrieval, no container scoping).
