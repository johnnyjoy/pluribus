ARCHIVED — NOT ACTIVE SYSTEM TRUTH

---

# Pluribus — hard memory control (system behavior)

**Audience:** people **editing the Pluribus codebase** — control plane, recall, enforcement, curation, MCP.

**Not:** a Cursor agent rule file. **Not:** “how to prompt the assistant.” This describes **how Pluribus must behave** as a **durable memory system**: contracts, ordering, and invariants to preserve in code, tests, and APIs.

Framing: **durable memory (Pluribus)** — global, situationally retrieved, authority-ranked; **not** container-scoped chat history.

Cross-reference: [pluribus-memory-first-ontology.md](pluribus-memory-first-ontology.md), [pluribus-agent-surface.mdc](../.cursor/rules/pluribus-agent-surface.mdc) (agent HTTP/MCP surface: no UUID scaffolding on default recall paths).

---

## Role (system)

Pluribus is **not**:

- a stateless request/response pipe
- a chat transcript store
- a project/task/workflow executor as the **primary** memory model

Pluribus **is**:

> **a system that uses memory to maintain continuity, avoid past failures, and reuse successful patterns**

---

## Prime directive

> **If memory is not used when it applies, the behavior is invalid.**

Implementations must make **recall** available and **cheap** enough that clients are not pushed toward guessing.

---

## Core loop (mandatory ordering)

The system must support this lifecycle end-to-end:

```text
Recall → Ground → Act → Validate → Update
```

No step should be **unnecessarily** optional in product design; orchestration and APIs should **not** block recall on container IDs (see agent-surface rule).

---

## Step 1 — Recall (required first capability)

Before meaningful downstream work, the client must be able to **retrieve** governing memory for the **situation**.

**Contract:** situation-shaped recall uses **`retrieval_query`** and **`tags`** on:

- **`POST /v1/recall/compile`** (JSON body), and
- **`GET /v1/recall/?retrieval_query=...&tags=...`** (query param **`retrieval_query`**, alias **`query`**), and
- **MCP `recall_get`**: JSON arguments **`retrieval_query`** or **`query`** are forwarded to that GET URL as **`retrieval_query`**.

**Failure condition (system):** If the stack allows substantive action **without** a recall path, or gates recall on **hive/workspace** IDs on the default path, that violates Pluribus memory-first behavior.

---

## Step 2 — Ground in memory (mandatory semantics)

Recall outputs must be structured so clients can extract:

- **continuity** — what is currently true / in progress
- **constraints** — what must not be violated
- **experience** — what has worked before (patterns, failures, decisions)

Downstream reasoning should **explicitly** use this structure (bundles, slices, authority).

**Failure condition:** APIs that return opaque blobs with no **governed** / **advisory** distinction encourage guessing.

---

## Step 3 — Act (only after grounding)

The system permits proposals, code changes, and responses only in ways that:

- **respect** constraints
- **reuse** relevant patterns
- **preserve** decisions (or surface conflict explicitly)

**Failure conditions:** Violating a stored constraint, ignoring a known failure, or silently redoing a settled decision without contradiction handling.

---

## Step 4 — Validate (required for risk)

Before risky or impactful changes, **`enforcement_evaluate`** (or HTTP equivalent) must be available with **bounded** `proposal_text` and clear **next_action** semantics.

**Failure condition:** Proceeding after **reject** without a new proposal path.

---

## Step 5 — Update memory (required when meaningful)

After meaningful work, **`curation_digest`** with **`work_summary`** must be supported; candidates are **not** canon until materialized per policy.

**What qualifies as durable memory:** decisions, failures, constraints, patterns, state that affects future behavior.

**What does not:** transcripts, chatter, trivial edits, vague summaries, “we talked about X.”

---

## Memory doctrine

### Memory is not history

Memory is:

> **distilled experience that changes future behavior**

### Lessons vs memory

Lessons are **effects** of memory; memory exists to **enable behavior change**.

**Correct model:**

```text
Experience → Memory → Behavior Change → Learning (Lessons)
```

**Not:**

```text
Experience → Store Lesson → Done
```

---

## Domain rule (critical)

Pluribus must **not** hard-code a single domain (coding-only, tickets-only, etc.). The same contracts should work for NPCs, chatbots, email, ops, brainstorming, OpenClaw, Python clients, **anything**.

**Failure condition:** Core APIs or ranking assume **project / task / ticket / repo** as the **primary** memory container.

---

## No containers rule

Memory is **not** organized by project, task, or workflow unit as **truth**. Memory is:

> **global, shared, and situationally retrieved**

Orchestration rows may exist for operators; they **must not** become prerequisites for governed recall on the agent path.

---

## Recall rule

Retrieval is driven by **similarity of situation**, **relevance**, **authority**, **constraints** — **not** membership in a container.

---

## Authority rule

Stronger memory must dominate: repeated success → stronger pattern; repeated failure → stronger constraint; cross-context use → higher importance (within configured ranking/reinforcement).

---

## Constraint rule

If memory says **do not X**, enforcement and clients must treat that as **binding** per applicability rules.

---

## Anti-guessing rule

If memory is incomplete: **recall again**, **refine query/tags** — the system should **not** incentivize inventing governing context.

---

## Intervention awareness

If the system injects memory (triggered recall), clients should **accept and use** it — implementation should make injections visible and attributable.

---

## Hard failure conditions (for tests and reviews)

Behavior is **wrong** for Pluribus if:

- recall is skipped where applicable
- constraints are ignored
- known failures repeat without escalation
- applicable patterns are ignored
- noise is stored as memory
- memory is treated as transcript
- project/task structure is used as the **primary** memory boundary

---

## Success condition

Pluribus succeeds when:

> **downstream behavior is improved by memory** — measurably, not cosmetically.

---

## Final directive

> **Pluribus is not here to echo chat.  
> It is here to remember, recall, and act accordingly.**

---

## Final line

> **If memory did not change behavior, the system failed.**
