# Documentation index

**Start:** [get-started.md](get-started.md) · **Truth:** [memory-doctrine.md](memory-doctrine.md) · **Wire:** [http-api-index.md](http-api-index.md)

Redirect stubs at old paths point here — **edit canonical files only**, not `MOVED` stubs.

---

## START (read these)

| Doc | Purpose |
|-----|---------|
| [get-started.md](get-started.md) | Run → connect agent → optional doctrine |
| [guides/quickstart-lab.md](guides/quickstart-lab.md) | HTTP curl lab (first five minutes) |
| [guides/container-install.md](guides/container-install.md) | GHCR / install compose |
| [mcp-usage.md](mcp-usage.md) | MCP wiring per client |
| [usage/ensuring-agent-usage.md](usage/ensuring-agent-usage.md) | Verify agents actually use memory |
| [integrations/README.md](../integrations/README.md) | Platform packs hub |

---

## TRUTH (canonical — highest authority first)

| Doc | Purpose |
|-----|---------|
| [memory-doctrine.md](memory-doctrine.md) | **Product model — wins all conflicts** |
| [anti-regression.md](anti-regression.md) | Banned patterns; CI guardrails |
| [architecture.md](architecture.md) | System shape + public summary |
| [ontology.md](ontology.md) | Memory vs execution metadata |
| [http-api-index.md](http-api-index.md) | Every shipped HTTP + MCP route |
| [rest-test-matrix.md](rest-test-matrix.md) | REST boundary behavior matrix |
| [api-contract.md](api-contract.md) | RC1 narrative subset |
| [product-doctrine.md](product-doctrine.md) | Product implementation doctrine |

---

## USE (agents & integrations)

| Doc | Purpose |
|-----|---------|
| [../integrations/pluribus-instructions.md](../integrations/pluribus-instructions.md) | **Paste into rules** — mandatory loop |
| [integrations/matrix.md](integrations/matrix.md) | Platform tiers |
| [integrations/skills-model.md](integrations/skills-model.md) | Skill intents → tools |
| [mcp-service-first.md](mcp-service-first.md) | HTTP MCP canonical |
| [mcp-poc-contract.md](mcp-poc-contract.md) | Tool ↔ route mapping |
| [mcp-discipline-doctrine.md](mcp-discipline-doctrine.md) | Lifecycle / timing |
| [pre-change-enforcement.md](pre-change-enforcement.md) | Enforcement product doc |
| [lsp-mcp-boundary.md](lsp-mcp-boundary.md) | Pluribus ≠ editor LSP |
| [walkthrough-single-agent.md](walkthrough-single-agent.md) | Continuity demo |
| [walkthrough-multi-agent.md](walkthrough-multi-agent.md) | Shared pool demo |
| [walkthrough-constraint-enforcement.md](walkthrough-constraint-enforcement.md) | Enforcement demo |

---

## OPERATE (run in production)

| Doc | Purpose |
|-----|---------|
| [operate/guide.md](operate/guide.md) | Config, health, migrations, CI |
| [authentication.md](authentication.md) | API key auth |
| [evaluation.md](evaluation.md) | `make proof-rest`, regression |
| [memory-lifecycle-semantics.md](memory-lifecycle-semantics.md) | Status, archive, historical recall |
| [curation/loop.md](curation/loop.md) | Digest → materialize |
| [curation/style-guide.md](curation/style-guide.md) | How to write recallable memories |

---

## PROOF (receipts — what we demonstrated)

| Doc | Purpose |
|-----|---------|
| [proof/README.md](proof/README.md) | Proof bundle index |
| [proof/scenarios.md](proof/scenarios.md) | YAML scenario contract + honesty |
| [../artifacts/deployed-benefit-receipts-latest.md](../artifacts/deployed-benefit-receipts-latest.md) | Latest deployed run |
| [../evidence/memory-proof.md](../evidence/memory-proof.md) | REST adversarial harness |
| [../evidence/episodic-proof.md](../evidence/episodic-proof.md) | Episodic lane |

---

## RELEASE

| Doc | Purpose |
|-----|---------|
| [release/README.md](release/README.md) | Release doc index |
| [release/scope.md](release/scope.md) | In scope vs deferred |
| [release/readiness.md](release/readiness.md) | Release gate checklist |
| [release/roadmap.md](release/roadmap.md) | Post-release fence |
| [release/images.md](release/images.md) | GHCR tags, CI gates |

---

## AGENT CONTRACT (Phase 11 — integrator depth)

| Doc | Purpose |
|-----|---------|
| [agent-contract/README.md](agent-contract/README.md) | Index |
| [agent-contract/contract.md](agent-contract/contract.md) | Agent-facing contract |
| [agent-contract/field-matrix.md](agent-contract/field-matrix.md) | Field-level matrix |
| [agent-contract/endpoint-coverage.md](agent-contract/endpoint-coverage.md) | Endpoint map |
| [agent-contract/usefulness.md](agent-contract/usefulness.md) | Usefulness harness |
| [agent-contract/obedience.md](agent-contract/obedience.md) | Obedience telemetry |
| [agent-contract/memory-telemetry.md](agent-contract/memory-telemetry.md) | Memory use telemetry |
| [agent-contract/recall-telemetry.md](agent-contract/recall-telemetry.md) | Automatic recall hooks |
| [guarded-utility-policy.md](guarded-utility-policy.md) | Guarded utility policy |
| [recall-quality.md](recall-quality.md) | Recall benchmark gates |

---

## ADVISORY (non-canonical — do not treat as product truth)

| Doc | Purpose |
|-----|---------|
| [advisory/README.md](advisory/README.md) | Index |
| [advisory/episodic-similarity.md](advisory/episodic-similarity.md) | Episodic advisory layer |
| [advisory/cognitive-memory-engineering.md](advisory/cognitive-memory-engineering.md) | Research notes |
| [advisory/audit-2026-07-hive-memory-objectives.md](advisory/audit-2026-07-hive-memory-objectives.md) | July 2026 audit (historical) |
| [advisory/audit-2026-07-hostile.md](advisory/audit-2026-07-hostile.md) | Hostile audit notes |

---

## ARCHIVE (historical — bannered, not active truth)

| Path | Note |
|------|------|
| [archive/](archive/) | Cutover reports, legacy POC docs |
| [archive/control-plane-design-and-starter.md](archive/control-plane-design-and-starter.md) | Design starter |
| [archive/deployment-poc.md](archive/deployment-poc.md) | Legacy deployment POC |

---

## META (maintainer notes — not for agents or new users)

| Path | Note |
|------|------|
| [meta/README.md](meta/README.md) | Internal index |
| [meta/adoption.md](meta/adoption.md) | Adoption strategy notes |
| [meta/active.md](meta/active.md) | Current focus |
| [meta/decisions.md](meta/decisions.md) | Decision log |
| [meta/roadmap.md](meta/roadmap.md) | Roadmap |
| [public-documentation-policy.md](public-documentation-policy.md) | What ships publicly |

---

## Experiments

| Path | Purpose |
|------|---------|
| [experiments/README.md](experiments/README.md) | Non-canonical explorations |

---

## SDK

| Path | Purpose |
|------|---------|
| [sdk/README.md](sdk/README.md) | Go + Python SDK index |

---

## MCP prompts & resources

| Doc | Purpose |
|-----|---------|
| [mcp-prompt-resource-audit.md](mcp-prompt-resource-audit.md) | Inventory |
| [mcp-prompt-resource-proof.md](mcp-prompt-resource-proof.md) | Proof map |
| [mcp-prompt-resource-versioning.md](mcp-prompt-resource-versioning.md) | SurfaceVersion |
| [mcp-tools.md](mcp-tools.md) | **Generated** tool list — not hand-edited |

---

## Local-only (gitignored on your machine)

[local/README.md](local/README.md) · [public-documentation-policy.md](public-documentation-policy.md)
