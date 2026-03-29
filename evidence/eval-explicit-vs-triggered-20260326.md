# Evidence: explicit vs triggered recall (deterministic eval harness)

**Date:** 2026-03-26  
**Scope:** `control-plane/internal/eval` dual-mode runs (explicit = `Service.Compile`, triggered = `Service.CompileTriggered`) with identical `CompileRequest` per scenario. **No** recall heuristic edits; eval uses `DefaultTriggerRecallConfig()` + `Enabled: true` (per-kind flags must be on — `{Enabled: true}` alone zeroed `EnableRisk` / `EnableDecision` / `EnableSimilarity` and filtered out all triggers; fixed in `eval_service.go`).

## How to reproduce

```bash
cd control-plane && go test ./internal/eval -v -count=1
```

Gate: `go test ./...` and `make regression` green (2026-03-26).

## Method

| Arm | Code path |
|-----|-----------|
| Explicit | `recall.Service.Compile` with `RetrievalQuery` from `recall_expectations.query` or `goal` + `context`; `ProposalText` = scenario `trap` (for request parity; compile ignores it for ranking). |
| Triggered | `recall.Service.CompileTriggered` with the **same** `CompileRequest`, in-memory memory pool, `Cache` nil. |

**Timing proxy:** `explicit_query` vs `trigger_metadata.retrieval_query_effective`; `triggers_fired` count; `skipped_reason` when cap applies.

**Delta rule (behavior-first):** `improvement=yes` only if explicit arm fails recall+behavior together and triggered arm passes both; `no` if explicit passes and triggered fails; else `same`.

## Aggregate (latest run)

| Metric | Value |
|--------|--------|
| Scenarios | 5 |
| `total_triggers` (sum of per-run trigger counts) | 8 |
| Scenarios with ≥1 trigger | 5 |
| `redundant_trigger` rows (triggers fired but recall+behavior match explicit) | 5 |
| Delta: better / same / worse | 0 / 5 / 0 |
| Dual harness `AllPassed` | true |

## Per-scenario snapshot

| Scenario | Triggers (kinds) | Capped | Redundant | Effective query vs explicit |
|----------|------------------|--------|-----------|-----------------------------|
| constraint-enforcement-001 | 2 (decision, similarity) | no | yes | Augmented with decision/similarity fragments |
| decision-enforcement-001 | 2 (risk, decision) | yes (`max_triggers_per_request`) | yes | Augmented |
| failure-avoidance-001 | 2 (risk, decision) | yes | yes | Augmented |
| pattern-reuse-001 | 1 (similarity) | no | yes | Augmented |
| state-continuity-001 | 1 (similarity) | no | yes | Augmented |

## Interpretation

1. **Behavior:** On the current JSON scenarios and validation rules, **triggered recall did not beat explicit recall** — all deltas **`same`**, with both arms passing extraction, recall `must_include`, and behavior heuristics.
2. **Noise signal:** **5/5** scenarios flagged **redundant** (triggers fired, outcomes unchanged vs explicit). That matches the charter’s “possible noise” case when augmentation does not change bundle/behavior checks.
3. **Caps:** Two scenarios hit **`max_triggers_per_request`** (default 2) with a third raw detection (e.g. risk + decision + similarity) — third kind dropped by cap, not by tuning.
4. **Recommendation:** **Keep triggered recall optional** for query enrichment and observability; **do not claim** behavior superiority from this harness alone. **Next:** add harder scenarios where explicit `RetrievalQuery` is intentionally thin vs rich `ProposalText`, or agent-loop metrics, before product conclusions. **Tune** caps/keywords only in a separate sprint (not this measurement run).
