# Experiments

Exploration branches and optional subsystems that **do not** redefine Pluribus canonical memory.

| Document | Topic |
|----------|--------|
| [pg-textsearch-evaluation.md](pg-textsearch-evaluation.md) | Why BM25 / pg_textsearch; source of truth; success criteria |
| [pg-textsearch-container.md](pg-textsearch-container.md) | Docker image, `shared_preload_libraries`, rebuild |
| [pg-textsearch-data-model.md](pg-textsearch-data-model.md) | Projection table vs `memories` |
| [pg-textsearch-etl.md](pg-textsearch-etl.md) | Backfill, reindex, verify |
| [pg-textsearch-hybrid-recall.md](pg-textsearch-hybrid-recall.md) | BM25 + semantic + authority (plan) |
| [pg-textsearch-filtering.md](pg-textsearch-filtering.md) | Pre/post filter tradeoffs for Pluribus |
| [pg-textsearch-benchmarks.md](pg-textsearch-benchmarks.md) | Eval harness notes |
| [pg-textsearch-rollback.md](pg-textsearch-rollback.md) | Disable extension path, revert compose |

Related scripts: `scripts/experiments/pg_textsearch_smoke.sql`, `docker-compose.pg-textsearch.yml`, `docker/pg-textsearch/Dockerfile`.
