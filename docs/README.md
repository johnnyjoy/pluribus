# Documentation index

**Canonical** public material. Historical and cutover narratives live under **[archive/](archive/)** — not active system truth.

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

## Optional / advisory

| Doc | Note |
|-----|------|
| [episodic-similarity.md](episodic-similarity.md) | **Advisory** only — not canonical recall authority; REST proof scenarios in **`make proof-rest`** / **`make proof-episodic`** |
| [../evidence/episodic-proof.md](../evidence/episodic-proof.md) | Episodic proof inventory, commands, limits |
| [pluribus-benefit-eval.md](pluribus-benefit-eval.md) | Eval methodology |

---

## Design depth (internal + starter)

| Doc | Purpose |
|-----|---------|
| [control-plane-design-and-starter.md](control-plane-design-and-starter.md) | Design entry (links to archive + index) |

---

## Workflows (repo conventions)

| Doc | Purpose |
|-----|---------|
| [work-order-format.md](work-order-format.md) | Work order sections |
| [memory-curation.md](memory-curation.md) | Curation style |
| [retrieval-order.md](retrieval-order.md) | Context order (retrieval ritual) |

---

## Archived (historical only)

| Doc | Note |
|-----|------|
| [archive/](archive/) | Cutover reports, gap analyses, legacy operator checklists, POC walkthroughs — each file bannered **ARCHIVED** |
