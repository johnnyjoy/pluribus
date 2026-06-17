# Cognitive Memory Engineering

Phase 11C translates **documented** cognitive-memory and agent-memory research into **executable engineering constraints** for Pluribus. This is not neuroscience theater, not a second AI inside Pluribus, and not proof that production LLM agents improve.

## Why memory is not storage

Storing text is archival. Memory engineering asks whether encoded experience can be **found later under the right conditions** and **applied without harm**. Phase 11B proved deterministic usefulness measurement; Phase 11C grounds that harness in research-backed encoding, schema, context, interference, and lifecycle rules.

## Why retrieval is not usefulness

Roediger & Karpicke (2006) show retrieval practice changes retention; for agents, the engineering lesson is narrower: **recall is an intermediate step**. The harness counts a memory helpful only when it is recalled, used, and improves the answer vs a no-memory baseline.

## Encoding cues

Tulving & Thomson (1973) — encoding specificity: retrieval succeeds when cues match encoding conditions.

Pluribus fixtures represent cues as:

| Field | Role |
|-------|------|
| `domain_tags` / `tags` | Compile and scope filters |
| `encoding_cues.retrieval_terms` | Lexical overlap with task query |
| `encoding_cues.scope` | Positive scope tag |
| `negative_scope` / `encoding_cues.negative_scope` | Explicit exclusion |
| `memory_schema_type` | Schema classification for hostile cases |

`EvaluateCueMatch` scores overlap; under-encoded memories (no cues) are flagged and must not count as helpful.

## Schema memory

Bartlett (1932) — schemas shape encoding and interpretation. Pluribus fixtures use `memory_schema_type` values including `constraint`, `failure`, `warning`, `procedure`, `refuted_guidance`, `current_guidance`. The deterministic simulator applies schema-typed memories only when fixture selection rules allow.

## Retrieval, use, and outcome

The harness distinguishes:

| State | Meaning |
|-------|---------|
| Recalled | Present in recall manifest |
| Used | In `memory-use` trace with `use_reason` |
| Ignored | Recalled but rejected with `ignore_reason` |
| Helpful | Used + answer improved + rubric pass |
| Harmful | Used + forbidden facts or misuse |

`outcome_feedback` artifacts record per-memory outcomes for benchmark inspection. Production verified-outcome loops remain a documented gap.

## Interference and near-miss suppression

Similar wrong memories can steer agents incorrectly (Xiong et al., 2025). Hostile fixtures include near-miss labels, high-authority wrong-scope decoys, and high-utility wrong-scope decoys. Metrics: `interference_failure_rate`, `near_miss_suppression_rate`, `wrong_context_use_rate` — all hard-zero gated.

## Experience replay risk

Agents may over-follow retrieved past experiences. Fixtures with `wrong_count` / `outdated_count` simulate stale harmful experience; the simulator suppresses them unless explicitly expected for historical tasks.

## Lifecycle governance

Zhang et al. (2024) survey — agent memory needs write/store/retrieve/use/demote/recover governance. Pluribus exposes `lifecycle_role`, `status`, and `utility_score` in recall manifests. Refuted and superseded rows must not guide current action while remaining historically recoverable.

## Research-to-test mapping

| Principle | Test package | Fixtures |
|-----------|--------------|----------|
| Encoding specificity | `encoding_quality.go`, cognitive tasks | `encoding_specificity_*` |
| Schema memory | `simulator.go`, cognitive tasks | `schema_*` |
| Retrieval practice | `score.go`, cognitive tasks | `retrieval_practice_*` |
| Context-dependent recall | `simulator.go` scope rules | `context_wrong_*` |
| Interference control | `interference_metrics.go` | `interference_*` |
| Reconstructive risk | manifest metadata tests | `reconstructive_metadata_exposed` |
| Experience following | `simulator.go` stale suppression | `experience_following_*` |
| Lifecycle governance | lifecycle + utility rules | `lifecycle_refuted_*` |

## Commands

```bash
make test-cognitive-memory-usefulness
make cognitive-memory-usefulness-benchmark
make proof-cognitive-memory-benefit
```

Artifact: `artifacts/cognitive-memory-usefulness-benchmark.json`

## Formation quality (Phase 11D)

Encoding specificity and schema rules are enforced at **formation time** via `control-plane/internal/formationquality/`. The evaluator applies the same research-backed constraints (cues, scope, lifecycle, provenance) before a memory can become active guidance. This closes the gap between measuring helpful recall (11B/11C) and preventing under-encoded writes.

```bash
make test-memory-formation-quality
make proof-memory-formation-quality
```

See [memory-formation-quality.md](memory-formation-quality.md) and [formation-escape-hatches.md](formation-escape-hatches.md).

## Test isolation (Phase 11E)

Tests must target the checkout codebase, not Cursor's configured MCP server. See [test-isolation.md](test-isolation.md).

## What Pluribus does not claim

- Human-like memory
- Proof of production LLM agent improvement
- Solved memory science
- Automatic truth detection
- Semantic/vector production enablement from this phase

## See also

- [agent-memory-usefulness.md](agent-memory-usefulness.md) — Phase 11B harness
- [reports/phase11c-cognitive-memory-research-prechange.md](reports/phase11c-cognitive-memory-research-prechange.md)
- [reports/phase11c-cognitive-memory-benefit-hardening-implementation-report.md](reports/phase11c-cognitive-memory-benefit-hardening-implementation-report.md)
