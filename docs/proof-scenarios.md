# Pluribus proof scenarios (benefit receipts)

This repo uses a **small, scenario-driven proof layer** so we can show that memory, recall, curation, and enforcement deliver **real benefit** — not just that endpoints return 200.

It is **not** a general AI eval framework, embedding benchmark, or second product. Prefer a few strong scenarios over many weak ones.

## Layers

| Layer | Where | Purpose |
|-------|--------|---------|
| **automated_core** | `go test ./...` (default tags) | Cheap validation: YAML loads, ids unique, required fields present (`internal/proofscenarios`). |
| **integration** | **`make regression`** (Docker + Postgres) | Receipts against real DB + full API router (`TestIntegration_proofScenarioSuite`) with **proof-friendly formation defaults** (seeded memories land **active**). |
| **deployed benefit receipts** | **`./scripts/proof-deployed-benefit-receipts.sh`** | Same scenario suite against a **live** `CONTROL_PLANE_URL` / `PLURIBUS_PROOF_BASE_URL` — real formation/recall policy, not proof defaults. Artifact: [`artifacts/deployed-benefit-receipts-latest.md`](../artifacts/deployed-benefit-receipts-latest.md). |
| **manual** | Operator / release | Documented checks not worth automating yet; keep rare. |

## Scenario files

- **Location:** [`control-plane/proof-scenarios/`](../control-plane/proof-scenarios/) — one `.yaml` per scenario.
- **Skip local-only templates:** filenames starting with `_` are ignored by the loader.

### Minimal YAML shape

| Field | Required | Description |
|-------|----------|-------------|
| `id` | yes | Stable slug (`enforcement-sqlite-forbidden`). |
| `version` | yes | Start at `1`. |
| `title` | yes | Human-readable title. |
| `mode` | yes | `automated_core` \| `integration` \| `manual`. |
| `category` | yes | e.g. `recall`, `enforcement`, `curation`, `continuity`, `anti_drift`. |
| `benefit_claim` | yes | Why this run is a **receipt** (product benefit), not only an API check. |
| `seed` / `stimulus` / `expected` | optional | Documentation and future generic runners; integration tests map `id` → Go runner. |

## Running

```bash
# Fast: validate scenario files + parse
cd control-plane && go test ./internal/proofscenarios/ -count=1

# Full receipts (same as CI regression)
cd /path/to/recall && make regression

# Deployed benefit receipts (live control-plane; real formation policy)
CONTROL_PLANE_URL=http://host:8123 ./scripts/proof-deployed-benefit-receipts.sh
# or: make proof-deployed-benefit-receipts CONTROL_PLANE_URL=http://host:8123
```

Integration proof suite entrypoint: `TestIntegration_proofScenarioSuite` in [`cmd/controlplane/proof_scenarios_integration_test.go`](../control-plane/cmd/controlplane/proof_scenarios_integration_test.go). The API is wired via [`internal/apiserver`](../control-plane/internal/apiserver/router.go) (same as `cmd/controlplane`).

**Deployed vs CI:** CI boots an in-process server with proof-friendly formation so seeds are immediately **active** at capped authority. Deployed receipts hit a running server as configured; warehouse defaults or consolidate into non-recallable sinks break benefit claims — not because `pending` is a separate tier from recall.

### Optional results file

```bash
RECALL_PROOF_RESULTS_OUT=/path/to/proof-scenario-results-latest.md \
  TEST_PG_DSN='postgres://...' \
  go test -tags=integration -count=1 ./cmd/controlplane -run TestIntegration_proofScenarioSuite
```

See [`archive/memory-bank/plans/proof-scenario-results-latest.md`](../archive/memory-bank/plans/proof-scenario-results-latest.md) for the canonical artifact path in this repo.

## Adding a scenario

1. Copy an existing YAML in `control-plane/proof-scenarios/`.
2. Set a new `id` and a clear **`benefit_claim`**.
3. Choose **`mode`**: prefer `integration` if it needs Postgres; use `manual` only when automation is not yet justified.
4. Add a runner in `proof_scenarios_integration_test.go` (`runners` map + `func runProof...`) **or** extend the generic harness later.
5. Run `go test ./internal/proofscenarios/` and **`make regression`**.

## What this is not

- Not a huge benchmark suite or embeddings evaluation.
- Not a replacement for unit tests in `internal/enforcement`, `internal/recall`, etc.
- Not dependent on a human grader each run — assertions are structured (decisions, kinds, substrings).

## Honesty contract (hostile / skeptical testing)

**PASS is a receipt for a narrow claim, not proof the product is “done.”**

Every **`mode: integration`** scenario **must** document `does_not_prove` in its YAML. `go test ./internal/proofscenarios/` fails if any integration scenario omits it.

| Mechanism | Purpose |
|-----------|---------|
| **`does_not_prove` in YAML** | Explicit scope boundary per scenario — what green does *not* mean |
| **`SuiteHonestyNotes` in code** | Limitations that apply to every run (REST vs MCP, CI vs deployed policy) |
| **Honesty appendix in results** | `RECALL_PROOF_RESULTS_OUT` markdown includes a “what PASS does not mean” section |
| **`proofRequireVerifiableWrite`** | Fails when `consolidated=true` but situation tag search cannot find the marker (no “success” without verifiable write) |
| **Decision + curation scenarios** | Unique `signals` situation tag per run; digest→materialize and direct POST both hostile-verify before recall |
| **Deployed vs CI** | Documented gap: proof-friendly formation in CI ≠ deployed formation policy on live servers |

When adding a scenario, write **`does_not_prove` first** — if you cannot list honest gaps, the scenario claim is probably too broad.

Chosen gaps we may intentionally not close (documented, not hidden):

- No automated proof of **`resolve_chore`** / **`memory_feedback`** agent loops yet (static grep only in verify scripts).
- Integration runners use **HTTP**, not MCP tools — tool wiring is covered separately by `verify-integrations-mcp`.
- **Ranking quality** under a long-lived shared pool is only indirectly stressed (tag-scoped compiles in some scenarios).

## Continuity (manual + integration)

- **Integration (CI):** [`simulated-multi-agent-continuity.yaml`](../control-plane/proof-scenarios/simulated-multi-agent-continuity.yaml) — two distinct HTTP clients; **shared situation tags + retrieval text** (no UUID handoff); hostile write verification. Results: [`archive/memory-bank/plans/pluribus-simulated-multi-agent-continuity-proof-results-20260327.md`](../archive/memory-bank/plans/pluribus-simulated-multi-agent-continuity-proof-results-20260327.md).
- **Manual protocol:** [`passive-continuity-same-slug-two-clients.yaml`](../control-plane/proof-scenarios/passive-continuity-same-slug-two-clients.yaml) — shared tags + retrieval text across two notional clients; see [archive/passive-continuity-architecture.md](archive/passive-continuity-architecture.md) (**archived**).
- **Manual (MCP workflow):** [`functional-quality-workflow.yaml`](../control-plane/proof-scenarios/functional-quality-workflow.yaml) — recall → enforcement → curation tool order; see [mcp-usage.md](mcp-usage.md).

**Index of all proof receipts:** [pluribus-proof-index.md](pluribus-proof-index.md).

## References

- Plan: [`archive/memory-bank/plans/plan-pluribus-proof-scenario-system-20260317.md`](../archive/memory-bank/plans/plan-pluribus-proof-scenario-system-20260317.md)
- Prior enforcement proof: [`docs/pre-change-enforcement.md`](pre-change-enforcement.md)
