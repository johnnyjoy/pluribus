# Memory Formation Quality

Phase 11D adds **deterministic formation-time quality gates**. Pluribus does not write memories with an LLM. The agent curates; Pluribus evaluates whether an encoded memory is safe enough to **enter the pool**.

## Why formation quality matters

A bad memory formed at write time is expensive forever. Recall ranking cannot fix:

- vague statements
- overgeneralized always/never rules without scope
- missing retrieval cues (buried memory)
- missing provenance on high-authority guidance
- historical events presented as current doctrine
- refuted or superseded guidance marked active

Phase 11B/11C measure whether memory **helps** at recall time. Phase 11D ensures obviously bad memories do not become **active guidance** at formation time.

## What makes a good Pluribus memory

A high-quality memory is **schema-appropriate**: it includes the fields required for its `schema_type`, plus:

| Field | Role |
|-------|------|
| `schema_type` | Classifies memory for rule selection |
| `statement` | Specific, actionable text |
| `scope` / `negative_scope` | Limits where the memory applies |
| `retrieval_cues` | Encoding-specific findability |
| `provenance` / `authority_basis` | Why to trust it |
| `lifecycle_role` | Historical vs current guidance |
| `use_instruction` / `misuse_warning` | How to apply without harm |
| `occurred_at` or temporal basis | When it was true (historical events) |

Not every field is required for every schema; rules are schema-specific.

## Supported schema types

`constraint`, `decision`, `lesson`, `failure_warning`, `preference`, `fact`, `historical_event`, `procedure`, `current_guidance`, `superseded_guidance`, `refuted_guidance`

Unknown `schema_type` values **fail** evaluation.

## Formation decisions

| Decision | Meaning |
|----------|---------|
| `accept_active` | Passes quality checks; enters pool at requested/capped authority |
| `accept_pending` | Soft warnings only — **hive default: still `active` at capped authority**; `pending` status is reserved for explicit review holds (rare) |
| `needs_curation` | Hard defects — **active at low authority** under hive defaults, or `pending` only when warehouse/review mode is explicitly enabled |
| `reject_garbage` | Vague or misleading; rejected |
| `reject_dangerous` | Refuted/superseded active guidance; rejected |

Integrated into `formation.Gate` on **direct create**, **promote**, and **probationary ingest** (`record_experience`). See [formation-escape-hatches.md](formation-escape-hatches.md).

- `reject_garbage` / `reject_dangerous` → HTTP/MCP error
- `accept_pending` / `needs_curation` → **`active` at capped authority** when `hive_defaults: true` (shipped default); ranking and agents curate trust over time

## Retrieval cue quality

Cues are evaluated for **specificity**, not count. Generic cues (`project`, `work`, `misc`, …) are penalized. Misleading cues (domain mismatch) can trigger `reject_garbage`.

Rich encoding is required for governing/high-authority constraint, `current_guidance`, and `procedure` schemas.

## Scope and negative scope

Universal patterns (`Always…`, `Never…`, `All agents should…`) require explicit `scope`, `negative_scope`, or exception. Constraints without scope cannot pass as active guidance.

## Provenance and authority

`agent_inferred` preferences cannot become high-authority active guidance. Governing direct creates without provenance are flagged unsafe — **active at capped authority** under hive defaults, or **`pending` only in warehouse/review mode**. `benchmark_proven` and `user_stated` constraints expect stronger provenance fields.

## Use instructions

`current_guidance`, governing `constraint`, and active `preference` / `historical_event` memories require `use_instruction`. Historical events should include `misuse_warning` (soft) when lifecycle role is missing.

## Quality score

`quality_score` is a deterministic 0–1 penalty sum from defects and warnings. It supports benchmarking and ordering; it does **not** prove real-world agent improvement.

## MCP and REST parity

Both interfaces call the shared `formation.Gate` via `EvaluateCreateInput`. Parity fixtures in `control-plane/testdata/memory_formation_quality/cases.json` assert identical evaluator outcomes.

## Proof targets

```bash
make test-memory-formation-quality
make memory-formation-quality-benchmark
make proof-memory-formation-quality
```

Artifact: `artifacts/memory-formation-quality-benchmark.json`

Hard safety thresholds (all zero except parity = 1.0):

- `dangerous_active_memory_rate`
- `under_encoded_active_memory_rate`
- `vague_memory_acceptance_rate`
- `overgeneralized_active_memory_rate`
- `missing_provenance_active_rate`
- `missing_scope_active_rate`
- `missing_cues_active_rate`

## What Pluribus does not claim

- Pluribus does not write perfect memories
- Pluribus does not understand truth
- Pluribus does not replace agent judgment
- Formation quality does not prove production LLM improvement

## Related docs

- [memory-doctrine.md](memory-doctrine.md)
- [cognitive-memory-engineering.md](cognitive-memory-engineering.md)
- [agent-memory-usefulness.md](agent-memory-usefulness.md)
