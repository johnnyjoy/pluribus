ARCHIVED — NOT ACTIVE SYSTEM TRUTH

---

# What makes Pluribus different (sober, technical)

This is a **concise differentiation** — not marketing fluff. It states what the system does that typical “agent memory” or chat logs do **not**.

---

## 1. Continuity through shared memory + agreed tag namespaces

Continuity is **not** “whatever the model remembers this session.” **Durable memory** is **shared** in Postgres; recall compiles governing slices from that pool using **tags**, **retrieval text**, and server ranking. Integrators align sessions by reusing the **same tag strings** (and optional **`agent_id`** where supported) — there is **no** workspace UUID on the public JSON bodies for recall or enforcement in the current control-plane. **Proven** in controlled integration scenarios — [pluribus-proof-index.md](pluribus-proof-index.md). Ontology: [pluribus-memory-first-ontology.md](../pluribus-memory-first-ontology.md).

---

## 2. Governed memory, not transcript sludge

Memory items are **typed** (constraint, decision, pattern, failure, etc.), **tag- and authority-shaped**, and live in a **shared pool**. Recall **compiles** a bounded bundle for the **current situation** — not an unbounded chat dump.

---

## 3. Pre-change enforcement before the work ships

**Enforcement** evaluates proposals against **trusted** memory before change — **block**, **allow**, or structured outcomes with **reasons** tied to memory — [pre-change-enforcement.md](pre-change-enforcement.md).

---

## 4. Curation of durable learning

Work produces **candidates**; **materialization** promotes durable entries with governance — [curation-loop.md](curation-loop.md). This is **not** “auto-save every chat turn.”

---

## 5. Service-first MCP with prompts and resources

MCP is **not** a separate ad-hoc sidecar for the canonical path: **`POST /v1/mcp`** on the same deployment as REST. **Prompts** and **resources** are part of the **behavioral interface** — discipline, lifecycle, memory grounding, workspace identity — [mcp-service-first.md](mcp-service-first.md).

---

## 6. Proof-oriented discipline

**Proof scenarios** in CI are **benefit receipts** (recall, enforcement, curation, continuity), not only “API returns 200” — [proof-scenarios.md](proof-scenarios.md).

---

## Honest limitations

- **Not** a general-purpose vector database product.
- **Not** a replacement for human review of high-stakes decisions.
- **Simulated** multi-agent continuity is **not** a substitute for every **multi-host** edge case — see [pluribus-proof-index.md](pluribus-proof-index.md).
