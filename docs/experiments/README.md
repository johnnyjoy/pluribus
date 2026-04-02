# Experiments

Optional subsystems and tooling that **do not** change Pluribus canonical memory as source of truth.

## pg_textsearch (BM25 lexical layer)

| Document | Purpose |
|----------|---------|
| [pg-textsearch.md](pg-textsearch.md) | **Start here** — architecture, canonical vs projection, integrated vs eval tooling |
| [pg-textsearch-eval.md](pg-textsearch-eval.md) | `make` targets, CLI, ETL, eval artifacts, harness notes |
| [pg-textsearch-container.md](pg-textsearch-container.md) | Docker image, `shared_preload_libraries`, smoke SQL |
| [pg-textsearch-rollback.md](pg-textsearch-rollback.md) | Disable API path, revert compose, drop projection |

**Code / scripts:** `docker/pg-textsearch/Dockerfile`, `docker-compose.pg-textsearch.yml`, `scripts/experiments/pg_textsearch_smoke.sql`, `scripts/pg-textsearch-eval`, `control-plane/cmd/pg-textsearch-eval`, `control-plane/internal/lexical`, `control-plane/internal/experiments/pgtextsearch`.

**Generated (local, not committed):** `artifacts/pg-textsearch/eval.json`, `artifacts/pg-textsearch/eval-summary.md` after `make pg-textsearch-eval`.
