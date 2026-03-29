# Pluribus proof scenarios (benefit receipts)

This repo uses a **small, scenario-driven proof layer** so we can show that memory, recall, curation, and enforcement deliver **real benefit** — not just that endpoints return 200.

It is **not** a general AI eval framework, embedding benchmark, or second product. Prefer a few strong scenarios over many weak ones.

## Layers

| Layer | Where | Purpose |
|-------|--------|---------|
| **automated_core** | `go test ./...` (default tags) | Cheap validation: YAML loads, ids unique, required fields present (`internal/proofscenarios`). |
| **integration** | **`make regression`** (Docker + Postgres) | Receipts against real DB + full API router (`TestIntegration_proofScenarioSuite`). |
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
```

Integration proof suite entrypoint: `TestIntegration_proofScenarioSuite` in [`cmd/controlplane/proof_scenarios_integration_test.go`](../control-plane/cmd/controlplane/proof_scenarios_integration_test.go). The API is wired via [`internal/apiserver`](../control-plane/internal/apiserver/router.go) (same as `cmd/controlplane`).

### Optional results file

```bash
RECALL_PROOF_RESULTS_OUT=/path/to/proof-scenario-results-latest.md \
  TEST_PG_DSN='postgres://...' \
  go test -tags=integration -count=1 ./cmd/controlplane -run TestIntegration_proofScenarioSuite
```

See [`memory-bank/plans/proof-scenario-results-latest.md`](../memory-bank/plans/proof-scenario-results-latest.md) for the canonical artifact path in this repo.

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

## Continuity (manual + integration)

- **Integration (CI):** [`simulated-multi-agent-continuity.yaml`](../control-plane/proof-scenarios/simulated-multi-agent-continuity.yaml) — two distinct HTTP clients; **shared tag namespace** only (no UUID handoff); same recall marker for Agent B. Results: [`archive/memory-bank/plans/pluribus-simulated-multi-agent-continuity-proof-results-20260327.md`](../archive/memory-bank/plans/pluribus-simulated-multi-agent-continuity-proof-results-20260327.md).
- **Manual protocol:** [`passive-continuity-same-slug-two-clients.yaml`](../control-plane/proof-scenarios/passive-continuity-same-slug-two-clients.yaml) — shared tags + retrieval text across two notional clients; see [archive/passive-continuity-architecture.md](archive/passive-continuity-architecture.md) (**archived**).
- **Manual (MCP workflow):** [`functional-quality-workflow.yaml`](../control-plane/proof-scenarios/functional-quality-workflow.yaml) — recall → enforcement → curation tool order; see [mcp-usage.md](mcp-usage.md).

**Index of all proof receipts:** [pluribus-proof-index.md](pluribus-proof-index.md).

## References

- Plan: [`archive/memory-bank/plans/plan-pluribus-proof-scenario-system-20260317.md`](../archive/memory-bank/plans/plan-pluribus-proof-scenario-system-20260317.md)
- Prior enforcement proof: [`docs/pre-change-enforcement.md`](pre-change-enforcement.md)
