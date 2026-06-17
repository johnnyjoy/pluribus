# Recall Quality (Phase 3)

Pluribus recall is **benchmarked** against labeled hostile cases. This document defines what “good recall” means before any scoring change.

## Relevant memory

A memory that directly helps the current task: governing constraint, prior decision, known failure, reusable pattern, project continuity, relevant user preference, or operational fact.

## Irrelevant memory

True but unhelpful for the task (e.g. Dornan Pro website copy during Pluribus MCP work).

## Harmful memory

Likely to mislead: wrong domain, obsolete/superseded constraint, high-authority but contextually wrong, contradicted memory.

## Expected hit

A labeled memory that **should** appear in top K for a benchmark case.

## Forbidden hit

A labeled memory that **must not** appear in top K.

## Acceptable background hit

Tolerable if expected hits are also present.

## Recall@K

Fraction of expected labels found in top K.

## Precision@K

Fraction of top K that are expected or acceptable.

## Forbidden-hit rate

Fraction of top K slots occupied by forbidden labels (case-level and suite average).

## Domain confusion

Returned memories from a different domain than the query’s expected domain.

## Constraint preservation

Governing constraints appear when genuinely relevant without dominating unrelated tasks.

## Bundle quality

Results land in appropriate bundle buckets (constraints, failures, patterns, continuity).

## Phase 4 scoring repair (lexical_only)

Pluribus recall scoring now uses **benchmarked relevance-first ranking**: situational affinity, wrong-domain penalties, empty-tag inference, repo-root affinity, authority as multiplier, and score-first sort. See [recall-scoring-principles.md](./recall-scoring-principles.md) and [phase4-recall-scoring-repair-report.md](./reports/phase4-recall-scoring-repair-report.md).

**Measured movement (Phase 3 → Phase 4 baseline):** Recall@K 0.875→0.91, Precision@K 0.545→0.78, forbidden-hit 0.077→0.04, gate failures 17→8.

**Acceptable:** Pluribus reduces cross-domain leakage in hostile benchmark cases using general scoring signals (not benchmark label hardcoding).

**Not acceptable unless proven:** “Pluribus always recalls the right memories” or “semantic recall fixes relevance.”

## Phase 3 limits

Benchmarks use **in-memory fixture corpus + lexical ranking** by default (`retrieval_mode=lexical`, semantic disabled). A **hybrid mode** exists for controlled evaluation with deterministic test embeddings (not a production embedding model).

## Phase 10B semantic / hybrid benchmark (not production-approved)

Semantic/hybrid recall is **disabled by default** in shipped config (`recall.semantic_retrieval.enabled: false`). Lexical recall remains the default production path.

Hybrid benchmark gates (`make recall-benchmark-hybrid-gate`) enforce:

- `lifecycle_violation_rate = 0`
- `date_bound_violation_rate = 0`
- `utility_violation_rate = 0`
- `mode_violation_rate = 0`
- `forbidden_hit_rate <= lexical` on semantic fixture cases
- hybrid candidates obey `recall_mode`, `include_status[]`, and date bounds

**Not acceptable unless proven:** “Semantic recall is production-ready,” “hybrid is better,” “vectors solve recall,” “lexical is obsolete.”

**Acceptable:** “Hybrid mode is benchmarkable under hostile lifecycle/date/utility gates,” “semantic remains off by default,” “production enablement requires passing hybrid gates with real embedder.”

Hostile semantic cases: `control-plane/testdata/recall_benchmark/semantic_cases.json`

## Honest claims

**Acceptable:** Pluribus measures recall with Recall@K, Precision@K, forbidden-hit rate, and domain confusion; regressions are detectable.

**Not proven by default:** “Always retrieves the right memory,” “prevents irrelevant memories,” “semantic recall active.”

## Commands

```bash
make test-recall-benchmark
make recall-benchmark-baseline
make recall-benchmark-report
make recall-benchmark-gate              # lexical 0/28
make recall-benchmark-hybrid-gate       # semantic hostile cases, hybrid mode
make recall-benchmark-compare           # artifacts/recall-benchmark-comparison.json
make recall-benchmark-all-modes         # lexical + hybrid + compare
make test-embedding-staleness           # Phase 10C metadata + staleness unit tests
make test-real-embedder-fallback        # Phase 10C internal fallback tests (httptest; no external SaaS)
```

Internal operator targets (Phase 10C plumbing — **not** agent-facing; skip without server config):

```bash
make recall-benchmark-real-embedder     # honest skip without internal embedder config
```

## Phase 10C internal retrieval guardrails (not production-approved)

Server-side optional retrieval plumbing: metadata, staleness detection, lexical fallback. **Semantic remains disabled by default.**

**Not acceptable unless proven:** “Semantic is production-ready,” “hybrid is better,” “agents must configure embeddings.”

**Acceptable:** “Deterministic hybrid gates pass,” “staleness/fallback exist,” “MCP and REST/API are primary interfaces.”

Policy: [embedding-staleness-policy.md](./embedding-staleness-policy.md), [agent-interface-boundary.md](./agent-interface-boundary.md)

## Phase 10D corrected scope

The aborted live-embedder product evaluation is **void**. Corrected Phase 10D is **agentic recall evaluation through MCP and REST/API**. See [phase10d-corrected-scope.md](./reports/phase10d-corrected-scope.md).

## Provider drift guardrail

Do not prescribe Ollama, OpenAI, Anthropic, or any specific embedding provider in core Pluribus docs. Agentic clients must not depend on provider-specific embedding details. Embeddings are optional **internal** plumbing.

Fixtures: `control-plane/testdata/recall_benchmark/`

## Phase 11B agent memory usefulness (not recall relevance)

Recall benchmarks prove **relevance and safety**. Phase 11B proves whether memory improves **deterministic agent-like task output** vs a no-memory baseline.

- **Useful memory** = recalled + explicitly used + output improved + rubric pass
- **Not useful** = recalled alone, or used without improvement

```bash
make test-agent-memory-usefulness
make proof-agent-memory-effectiveness
```

See [agent-memory-usefulness.md](./agent-memory-usefulness.md). Fixtures: `control-plane/testdata/agent_memory_usefulness/`. Artifact: `artifacts/agent-memory-usefulness-benchmark.json`.
