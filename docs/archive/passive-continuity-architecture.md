ARCHIVED — NOT ACTIVE SYSTEM TRUTH

---

# Passive continuity architecture (Pluribus)

This note defines how **durable memory** stays consistent across agents, machines, and sessions **without** a workflow engine — using **shared tags** and **retrieval text** on the public HTTP/MCP surfaces.

It complements [mcp-discipline-doctrine.md](mcp-discipline-doctrine.md) and [mcp-service-first.md](mcp-service-first.md).

**Doctrine:** Pluribus is a **global memory system**. Continuity comes from **the same tag namespaces** and **situation-shaped recall** — not from passing workspace UUIDs in JSON bodies for compile or enforcement. See [pluribus-memory-first-ontology.md](pluribus-memory-first-ontology.md).

---

## What “passive leap” means

A **passive leap** is an improvement where:

- The **default** path teaches **recall and memory writes first**, not “create a workspace first.”
- **Operators** do less mistaken framing (treating Pluribus as Jira-for-agents).
- **Agents** land on **governed memory** because the easy path is **recall → act → validate → curate**, not “new chat → new container.”

It is **not** more model intelligence; it is **better defaults + honest contracts**.

---

## Memory-first continuity

- **Durable rows** live in the shared **`memories`** pool; correlate runs with **`tags`** (and optional **`agent_id`** where the API allows it).
- **Recall** compiles a **situation-shaped** bundle from that pool — not from chat transcripts.
- **Enforcement** evaluates proposals against **binding** memory.

---

## Cross-client alignment (no UUID handoff)

Two sessions (or two machines) can converge on the **same governing context** by:

1. Agreeing on a **tag string** (e.g. team or feature namespace).
2. Using **`POST /v1/recall/compile`** or **`GET /v1/recall/`** with those tags and a **`retrieval_query`** / **`query`** that describes the situation.
3. Writing durable memory with the **same tags** so later recall matches.

There is **no** requirement in the current **`internal/apiserver/router.go`** tree for a **`/v1/hives`** create/fetch step before recall.

---

## Ambient guidance (MCP)

The MCP layer provides **ambient** discipline:

- **Prompts** (e.g. **`pluribus_memory_grounding`**) encode recall-first behavior.
- **Resources** (`pluribus://discipline/*`) state doctrine, lifecycle, canon vs advisory.

MCP is **not** a workflow engine: no hidden state machine, no mandatory babysitting.

---

## Minimal surface (accurate)

| Layer | Role |
|-------|------|
| **Memory + recall + enforcement** | MCP tools: **`recall_*`**, **`memory_*`**, **`enforcement_evaluate`**, **`curation_*`**, **`health`** |
| **Tag agreement** | Operator/agent convention — same strings on create and recall |

See [mcp-poc-contract.md](mcp-poc-contract.md).

---

## Roadmap (sober)

| Status | Item |
|--------|------|
| **Current** | Memory-first docs + MCP prompts; continuity via tags + retrieval text |
| **Future** | Stronger multi-tenant namespace story if product requires it |
| **Integration** | See updated proof scenarios under **`control-plane/proof-scenarios/`** |

---

## References

- Public architecture: [pluribus-public-architecture.md](pluribus-public-architecture.md)
- Historical plan: [plan-pluribus-passive-continuity-leap-20260326.md](../../archive/memory-bank/plans/plan-pluribus-passive-continuity-leap-20260326.md)
