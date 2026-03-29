# Pluribus — public release scope

This document **fences** what this release claims versus what is explicitly **deferred** (future research or optional layers).

---

## In scope (this release)

- **Service-first architecture** — control-plane HTTP API as the authority.
- **Direct HTTP** — all `/v1/*` REST surfaces for agents and integrations.
- **MCP over HTTP** — `POST /v1/mcp` with tools, **prompts**, and **resources** as first-class surface.
- **Global memory system** — durable constraints, decisions, failures, patterns; recall, enforcement, curation; **tags** and shared pool semantics ([pluribus-memory-first-ontology.md](pluribus-memory-first-ontology.md)).
- **Internal DB FKs** — Postgres may link evidence and candidates for persistence; **not** a partition model for recall ([memory-doctrine.md](memory-doctrine.md)).
- **HTTP surface** — full route map [http-api-index.md](http-api-index.md); **MCP** exposes a **subset** of tools ([mcp-poc-contract.md](mcp-poc-contract.md)).
- **Recall compilation** — structured bundles for **situation-shaped** context (optional execution metadata on the request).
- **Pre-change enforcement** — evaluate proposals against trusted memory.
- **Curation** — digest → candidate → materialize into durable memory.
- **Drift / contradictions** — structured checks.
- **Proof scenario system** — YAML + integration receipts ([proof-scenarios.md](proof-scenarios.md), [pluribus-proof-index.md](pluribus-proof-index.md)).
- **Simulated multi-agent continuity proof** — controlled integration scenario (not multi-host).

---

## Out of scope (deferred / non-authoritative)

| Topic | Why |
|-------|-----|
| **Embeddings** as canonical recall | Not the authority layer; **vector search** is not the continuity gate. |
| **Semantic / embedding redesign** | Not part of this release. |
| **Major recall ranking redesign** | Current bundle + scoring rules are in-product; **no** commitment to RAG-style overhaul. |
| **Orchestration engines** | No workflow engine in the product. |
| **Stdio MCP binary** | **Compatibility** path; not the canonical deployment story — [mcp-service-first.md](mcp-service-first.md). |

---

## Optional / subordinate (shipped but clearly secondary)

| Topic | Position |
|-------|----------|
| **Advisory episodic similarity** | **Advisory only** — does not replace canonical memory — [episodic-similarity.md](episodic-similarity.md). |
| **Evidence in bundles** | Supporting links when present — [evidence-in-recall.md](evidence-in-recall.md). |

---

## Messaging rule

Public materials should **not** imply embeddings or vector DBs are required for **continuity** or **governance**. Those are **future** or **optional** enhancements, not the current release bar.
