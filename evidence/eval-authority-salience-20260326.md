# Authority salience sprint — implementation evidence (2026-03-26)

## Summary

Implemented **bounded recall reinforcement** with optional **cross-context salience** (`payload.salience`), **scorer-only failure severity** (keyword heuristic), **configurable ranking weights** (`recall.ranking`), **enforcement success** hooks, **run-multi promote** success reasons, and **`[AUTHORITY UPDATE]`** structured logs.

## Formula (recall ranking)

`scoreBase` sums: weighted normalized authority, recency, tag match, failure overlap lexical similarity, pattern priority, optional **failure severity** (failure kind only), optional **cross-context** term from `payload.salience.distinct_contexts`:

`cross_context_term = weight_cross_context_salience * min(1, log1p(distinct_contexts) / k)` with `k = weight_cross_context_salience_k` or **3** when zero.

Pattern rows still apply `PatternScoreFactor` / generalization in `scoreAt` after `scoreBase`.

## Tests

- `go test ./...` in `control-plane/` — green.
- `internal/memory`: `TestMergeSalienceForContext_distinct`.
- `internal/recall`: `TestFailureSeverityOutranksLowSeverity`, `TestCrossContextSalienceBoostsScore`.

## Config

- `memory.recall_reinforcement`: `max_authority_delta_per_compile`, `cross_context_enabled`.
- `recall.ranking`: `weight_failure_severity`, `weight_cross_context_salience`, `weight_cross_context_salience_k` (defaults **0** = off for new terms).

## Remaining gaps

- Enforcement success reinforcement only fires when `validation.next_action == proceed`, `validation.passed`, and `triggered_memories` is non-empty (rare path).
- Failure severity list is **keyword v1**; extend buckets as product requires.
