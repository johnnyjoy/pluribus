# Formation Escape Hatches

Phase 11E closes formation paths that could bypass Phase 11D quality gates.

## Why direct create was not enough

Phase 11D integrated `formationquality` into **direct create** (`memory_create` / `POST /v1/memory`). Two paths could still advance low-quality memory:

1. **Promote** (`POST /v1/memory/promote`) — candidate → canonical memory
2. **Probationary ingest** (`record_experience` / advisory episodes) — inline probationary memory

## Phase 11E integration

`formation.Gate.EvaluateCreateInput` now calls `applyQualityLayer` for:

| Path | Constant | Quality layer |
|------|----------|---------------|
| Direct create | `PathDirectCreate` | Phase 11D (unchanged) |
| Promote | `PathPromote` | **Phase 11E** |
| Probationary ingest | `PathProbationaryIngest` | **Phase 11E** |

### Promote behavior

- Vague / refuted / superseded active → **reject**
- Under-encoded / missing scope / missing provenance → **pending**
- Valid cue-rich candidates → **allow** (subject to Phase 5 authority/review rules)
- Authority cap preserved

### Probationary ingest behavior

- Junk / vague → **reject** (junk gate + quality)
- Under-encoded / overgeneralized / missing scope → **pending**
- Agent-inferred high authority → **authority capped** to ≤2
- Concrete advisory failures/decisions → **allow** with quality score on `Decision.Quality`

## MCP and REST parity

| Formation Path | MCP | REST/API | Shared code | Quality |
|----------------|-----|----------|-------------|---------|
| Direct create | Yes | Yes | `formation.Gate` | Yes |
| record_experience | Yes | Yes (advisory) | `formation.Gate` + `vet.Service` | Yes |
| Promote | No | Yes | `memory.Service.Promote` → gate | Yes |

Promote has no MCP tool; REST-only path is documented.

## Proof targets

```bash
make test-formation-escape-hatches
make formation-escape-hatch-benchmark
make proof-formation-escape-hatches
```

Artifact: `artifacts/formation-escape-hatch-benchmark.json`

Fixtures: `control-plane/testdata/formation_escape_hatches/cases.json` (18 hostile cases)

## Hard thresholds

```text
escape_hatch_count = 0
unsafe_promote_acceptance_rate = 0
unsafe_probationary_influence_rate = 0
direct_create_quality_coverage_rate = 1.0
promote_quality_coverage_rate = 1.0
probationary_ingest_quality_coverage_rate = 1.0
```

## See also

- [test-isolation.md](test-isolation.md)
- [memory-formation-quality.md](memory-formation-quality.md)
