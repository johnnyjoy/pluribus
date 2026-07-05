# Pluribus — proof index (trust bundle)

This page **surfaces receipts** for what the system has actually demonstrated in-repo — not aspirations. Follow links for methodology and limitations.

---

## Canonical memory-substrate proof (REST)

| Receipt | What it proves | Where |
|---------|----------------|--------|
| **REST adversarial proof harness** | Memory/recall/enforcement **invariants** over **HTTP only**, with **determinism** (two full passes, matching signature); enforcement **rule scope** + semantic **fallback visibility** scenarios included | **`cd control-plane && TEST_PG_DSN='…' make proof-rest`** — scenarios `internal/eval/scenarios/proof-*.json`; artifact [evidence/memory-proof.md](../evidence/memory-proof.md); narrative [evaluation.md](evaluation.md) |
| **Episodic advisory + distillation proof** | Advisory ingest, time/entity **similar**, inverted window errors, weak distill suppression, repetition merge, full chain **distill → review → materialize → recall → enforcement** (modeled normative case); adversarial Go sprint (conflict, time skew, boundary, soak) | **`make proof-episodic`** (root or `control-plane/`); [evidence/episodic-proof.md](../evidence/episodic-proof.md); [episodic-similarity.md](episodic-similarity.md) |

**MCP / LSP** are **not** this layer’s primary proof surfaces — prove the **service** at REST first.

---

## Automated gates (CI / regression)

| Receipt | What it proves | Where |
|---------|----------------|--------|
| **Proof scenario suite** | Memory, recall, curation, enforcement, **continuity** behaviors against real Postgres + full router | `make regression` → `TestIntegration_proofScenarioSuite` — [proof-scenarios.md](proof-scenarios.md), [`control-plane/proof-scenarios/`](../control-plane/proof-scenarios/) |
| **YAML + integration runners** | Structured pass/fail per scenario id (not hand-wavy demos) | [`proof_scenarios_integration_test.go`](../control-plane/cmd/controlplane/proof_scenarios_integration_test.go) |

## Deployed benefit receipts (live control-plane)

| Receipt | What it proves | Where |
|---------|----------------|--------|
| **Deployed benefit receipts** | Same scenario suite against a **running** server (`CONTROL_PLANE_URL` / `PLURIBUS_PROOF_BASE_URL`) under **real** formation/recall policy — closes CI ↔ deploy gap | `./scripts/proof-deployed-benefit-receipts.sh` or `make proof-deployed-benefit-receipts` — latest: [artifacts/deployed-benefit-receipts-latest.md](../artifacts/deployed-benefit-receipts-latest.md) |

CI receipts use proof-friendly formation (seeds **active** at capped authority). Deployed receipts fail when the **shared pool** cannot **recall** or **bind** governing memory the scenarios create — usually formation drift (warehouse defaults, consolidate into non-recallable sinks, or writes that fail hostile tag verification), not because `pending` is a separate tier.

Each integration scenario documents **`does_not_prove`** in YAML; PASS receipts include an [honesty appendix](proof-scenarios.md#honesty-contract-hostile--skeptical-testing) in [`artifacts/deployed-benefit-receipts-latest.md`](../artifacts/deployed-benefit-receipts-latest.md).

---

## Scenario categories (integration)

| Category | Example scenario id | Benefit (short) |
|----------|---------------------|-----------------|
| **Enforcement** | `enforcement-sqlite-forbidden`, `enforcement-unrelated-allow` | Blocks normative conflicts; allows unrelated work |
| **Recall** | `recall-binding-constraint-surfaces`, `recall-decision-relevant-to-work` | Governing memory surfaces in bundles |
| **Curation** | `curation-digest-materialize-durable`, `curation-then-recall-continuity` | Digest → materialize → recall sees durable memory |
| **Continuity** | `continuity-second-step-from-first` | Second recall sees same truth |
| **Simulated multi-agent continuity** | `simulated-multi-agent-continuity` | Two HTTP clients; **shared situation tags + retrieval text** (no UUID handoff); hostile write verification | [pluribus-simulated-multi-agent-continuity-proof-results-20260327.md](../archive/memory-bank/plans/pluribus-simulated-multi-agent-continuity-proof-results-20260327.md) |
| **Anti-drift** | `anti-drift-known-bad-pattern` | Receipt variant tied to enforcement |

**Manual protocol (not automated in CI):**

| Receipt | Doc / YAML |
|---------|------------|
| Slug continuity (two clients, YAML) | [`passive-continuity-same-slug-two-clients.yaml`](../control-plane/proof-scenarios/passive-continuity-same-slug-two-clients.yaml), [archive/passive-continuity-architecture.md](archive/passive-continuity-architecture.md) (**archived**) |

---

## Prompts and resources (MCP surface)

| Receipt | Where |
|---------|--------|
| Audit + proof map + versioning | [mcp-prompt-resource-audit.md](mcp-prompt-resource-audit.md), [mcp-prompt-resource-proof.md](mcp-prompt-resource-proof.md), [mcp-prompt-resource-versioning.md](mcp-prompt-resource-versioning.md) |
| Discipline / lifecycle doctrine | [mcp-discipline-doctrine.md](mcp-discipline-doctrine.md) |

---

## Workflow and pre-change discipline

| Topic | Doc |
|-------|-----|
| Pre-change enforcement (product) | [pre-change-enforcement.md](pre-change-enforcement.md) |
| Curation loop | [curation-loop.md](curation-loop.md) |

---

## Benefit evaluation (A/B / baseline)

| Topic | Doc |
|-------|-----|
| Foundational beta baseline | [archive/foundational-beta-benefit-baseline.md](archive/foundational-beta-benefit-baseline.md) (**archived**) |
| Pluribus benefit eval (commands) | **Local-only** — [public-documentation-policy.md](public-documentation-policy.md) |

---

## What is *not* proven by this bundle

- **Embeddings alone as the authority layer** — authority remains explicit on memory rows; vectors **rank candidates**. Fallback is **explicit** (`[SEMANTIC FALLBACK]`, bundle **`semantic_retrieval`**). **`make proof-rest`** includes **`proof-semantic-fallback-001`**; it proves the **substrate**, not your embedding endpoint wiring.
- **Agent housekeeping loop end-to-end** — integration receipts are **HTTP**; `resolve_chore` / `memory_feedback` are documented and grep-checked in packs, not yet a CI proof scenario.
- **Honesty scope** — each scenario's YAML **`does_not_prove`** lists what PASS does not mean; see [proof-scenarios.md — Honesty contract](proof-scenarios.md#honesty-contract-hostile--skeptical-testing).
- Operational semantic defaults and fallback: [pluribus-operational-guide.md](pluribus-operational-guide.md), [evaluation.md](evaluation.md), [pluribus-release-scope.md](pluribus-release-scope.md).
- **True multi-host** continuity under network partitions — proof uses distinct HTTP clients against one server; see scenario **`does_not_prove`**.
- **Full multi-tenant isolation** — exercise your auth model if you deploy shared infrastructure.
- **Deployed benefit by default** — CI pass does not imply live benefit. Re-run **deployed benefit receipts** after deploy; see [artifacts/deployed-benefit-receipts-latest.md](../artifacts/deployed-benefit-receipts-latest.md).

---

## Optional results artifact

```bash
RECALL_PROOF_RESULTS_OUT=/path/to/proof-scenario-results-latest.md \
  TEST_PG_DSN='postgres://...' \
  go test -tags=integration -count=1 ./control-plane/cmd/controlplane -run TestIntegration_proofScenarioSuite
```

See [`archive/memory-bank/plans/proof-scenario-results-latest.md`](../archive/memory-bank/plans/proof-scenario-results-latest.md).
