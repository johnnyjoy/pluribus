# Documentation index

**Canonical** public material. Historical and cutover narratives live under **[archive/](archive/)** — not active system truth.

**Public vs local:** Editor workflows and phase audit reports are **not** in the public repo. See **[public-documentation-policy.md](public-documentation-policy.md)** and **[local/README.md](local/README.md)** (local tree is gitignored on your machine).

---

## New here?

**→ [get-started.md](get-started.md)** (three steps + links).

Everything below is **reference** — use the table when you need a specific topic.

---

## Experiments (non-canonical)

| Path | Purpose |
|------|---------|
| [experiments/README.md](experiments/README.md) | Index of exploration docs (e.g. pg_textsearch BM25) |

## Public release surface (start here)

| Doc | Purpose |
|-----|---------|
| **[get-started.md](get-started.md)** | **Minimal path** — run, connect agent, optional doctrine |
| **[memory-doctrine.md](memory-doctrine.md)** | **Canonical product model — highest authority** |
| [product-doctrine.md](product-doctrine.md) | Product implementation doctrine (agents, interfaces, no second-AI trap) |
| [agent-interface-boundary.md](agent-interface-boundary.md) | MCP + REST/API primary interfaces; embedder is internal plumbing |
| [anti-regression.md](anti-regression.md) | Reviewer enforcement; banned patterns; CI guard pointers |
| [architecture.md](architecture.md) | System shape aligned with the doctrine |
| [pluribus-quickstart.md](pluribus-quickstart.md) | First-run — run, verify, HTTP smoke, MCP pointers |
| [authentication.md](authentication.md) | Auth model for technical preview |
| [mcp-usage.md](mcp-usage.md) | **MCP client setup** — Cursor, Claude Desktop, HTTP vs stdio; workflow; troubleshooting |
| **[integrations/README.md](integrations/README.md)** | **AI editors & agent systems** — platform matrix, [`pluribus-instructions.md`](../integrations/pluribus-instructions.md), native templates, `skill.md`, MCP examples ([integrations/](../integrations/) artifacts) |
| **[integrations/usage.md](integrations/usage.md)** | **Adoption layer** — behavioral loop, MCP vs REST, verification, links (not duplicate of ensuring-agent) |
| **[integrations/skills-model.md](integrations/skills-model.md)** | Four behavioral intents → MCP tools; one **`skills/pluribus/`** pack per platform |
| **[usage/ensuring-agent-usage.md](usage/ensuring-agent-usage.md)** | **Operational depth** — MCP + rules + REST fallback; failure modes; [snippets](usage/snippets/) |
| [evaluation.md](evaluation.md) | **Canonical verification:** `make proof-rest` + `make proof-episodic` + eval/stress/CI targets |
| [walkthrough-single-agent.md](walkthrough-single-agent.md) | Continuity walkthrough |
| [walkthrough-multi-agent.md](walkthrough-multi-agent.md) | Multi-agent coordination walkthrough |
| [walkthrough-constraint-enforcement.md](walkthrough-constraint-enforcement.md) | Constraint enforcement walkthrough |
| [http-api-index.md](http-api-index.md) | **Canonical HTTP + MCP route map** (every shipped path) |
| [rest-test-matrix.md](rest-test-matrix.md) | **REST boundary** — behavior matrix + integration test map (service truth) |
| [api-contract.md](api-contract.md) | **RC1 HTTP subset** — narrative examples for core integrator paths |
| [pluribus-public-architecture.md](pluribus-public-architecture.md) | **One** public architecture story |
| [pluribus-memory-first-ontology.md](pluribus-memory-first-ontology.md) | **Memory vs execution metadata** |
| [pluribus-container-install.md](pluribus-container-install.md) | Compose-first install — GHCR image |
| [pluribus-image-release-policy.md](pluribus-image-release-policy.md) | Registry, tags, CI gates |
| [pluribus-operational-guide.md](pluribus-operational-guide.md) | Config, health, migrations, CI |
| [pluribus-proof-index.md](pluribus-proof-index.md) | Proof bundle — receipts and links |
| [pluribus-release-scope.md](pluribus-release-scope.md) | In scope vs deferred |
| [pluribus-release-readiness.md](pluribus-release-readiness.md) | Release gate + operator smoke |
| [pluribus-post-release-roadmap.md](pluribus-post-release-roadmap.md) | Future work fence |
| [pluribus-lsp-mcp-boundary.md](pluribus-lsp-mcp-boundary.md) | Pluribus ≠ editor LSP |

---

## Architecture (proposed)

| Doc | Purpose |
|-----|---------|
| [architecture/layered-memory-L0-L3.md](architecture/layered-memory-L0-L3.md) | **Proposed:** layered context (L0–L3) as threshold views on one substrate |
| [architecture/situational-affinity-ranking.md](architecture/situational-affinity-ranking.md) | Situational affinity ranking (query/repo/tag boost; not partitions) |

---

## Core product (canonical)

| Topic | Doc |
|-------|-----|
| MCP service-first | [mcp-service-first.md](mcp-service-first.md) |
| MCP ↔ HTTP contract | [mcp-poc-contract.md](mcp-poc-contract.md) |
| Discipline / lifecycle | [mcp-discipline-doctrine.md](mcp-discipline-doctrine.md) |
| Pre-change enforcement | [pre-change-enforcement.md](pre-change-enforcement.md) |
| Curation loop | [curation-loop.md](curation-loop.md) |
| Proof scenarios | [proof-scenarios.md](proof-scenarios.md) |
| Deployment runbook | [deployment-poc.md](deployment-poc.md) |

---

## Compatibility / migration (secondary)

| Doc | Use when |
|-----|----------|
| [mcp-migration-stdio-to-http.md](mcp-migration-stdio-to-http.md) | Migrating from stdio MCP |

---

## Prompts & resources (MCP surface)

| Doc | Purpose |
|-----|---------|
| [mcp-prompt-resource-audit.md](mcp-prompt-resource-audit.md) | Inventory |
| [mcp-prompt-resource-proof.md](mcp-prompt-resource-proof.md) | Proof map |
| [mcp-prompt-resource-versioning.md](mcp-prompt-resource-versioning.md) | SurfaceVersion |

---

## Phase 11 — agent memory usefulness, contract, telemetry, utility (canonical)

| Doc | Purpose |
|-----|---------|
| [agent-memory-usefulness.md](agent-memory-usefulness.md) | Phase 11B cognitive usefulness harness doctrine |
| [cognitive-memory-engineering.md](cognitive-memory-engineering.md) | Phase 11C research-backed benefit hardening |
| [memory-formation-quality.md](memory-formation-quality.md) | Phase 11D formation quality gate |
| [formation-escape-hatches.md](formation-escape-hatches.md) | Phase 11E escape hatches + codebase test isolation |
| [agent-facing-memory-contract.md](agent-facing-memory-contract.md) | Phase 11F agent-facing memory contract |
| [agent-memory-contract-field-matrix.md](agent-memory-contract-field-matrix.md) | Phase 11G field-level contract matrix |
| [agent-memory-contract-endpoint-coverage.md](agent-memory-contract-endpoint-coverage.md) | Phase 11G endpoint coverage map |
| [agent-contract-obedience.md](agent-contract-obedience.md) | Phase 11H obedience telemetry |
| [memory-use-telemetry.md](memory-use-telemetry.md) | Phase 11I persisted telemetry + live loop |
| [automatic-recall-telemetry.md](automatic-recall-telemetry.md) | Phase 11J automatic recall hooks + Postgres proof |
| [guarded-utility-policy.md](guarded-utility-policy.md) | Phase 11K guarded utility application policy |
| [recall-quality.md](recall-quality.md) | Recall benchmark gates and hybrid scoring |

---

## Optional / advisory

| Doc | Note |
|-----|------|
| [episodic-similarity.md](episodic-similarity.md) | **Advisory** only — not canonical recall authority; REST proof scenarios in **`make proof-rest`** / **`make proof-episodic`** |
| [../evidence/episodic-proof.md](../evidence/episodic-proof.md) | Episodic proof inventory, commands, limits |

Benefit eval and Cursor verify protocols are **local-only** — see [public-documentation-policy.md](public-documentation-policy.md).

---

## Design depth (internal + starter)

| Doc | Purpose |
|-----|---------|
| [control-plane-design-and-starter.md](control-plane-design-and-starter.md) | Design entry (links to archive + index) |

---

## Workflows (repo conventions)

| Doc | Purpose |
|-----|---------|
| [memory-curation.md](memory-curation.md) | Curation style |
| [pre-change-enforcement.md](pre-change-enforcement.md) | Pre-change enforcement product doc |

Work-order, constitution, and retrieval-order formats are **local-only** — [local/README.md](local/README.md).

---

## Archived (historical only)

| Doc | Note |
|-----|------|
| [archive/](archive/) | Cutover reports, gap analyses, legacy operator checklists, POC walkthroughs — each file bannered **ARCHIVED** |
