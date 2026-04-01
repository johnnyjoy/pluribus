# pg_textsearch branch — summary (workstream 12)

## Files added/changed (high level)

- `docker/pg-textsearch/Dockerfile` — pg18 + pgvector + pg_textsearch `.deb`
- `docker-compose.pg-textsearch.yml` — overlay: `shared_preload_libraries`, custom image
- `scripts/experiments/pg_textsearch_smoke.sql` — extension + BM25 smoke
- `scripts/pg-textsearch-eval.sh` — ephemeral Postgres + `go run … eval`
- `docs/experiments/*.md` — design, container, ETL, hybrid, filtering, benchmarks, rollback
- `docs/experiments/pg-textsearch-eval-latest.md` — **generated** by `make pg-textsearch-eval`
- `docs/experiments/sql/lexical_projection_example.sql` — example projection DDL
- `control-plane/migrations/0008_lexical_memory_projection.sql` — projection table (no BM25 in migration)
- `control-plane/internal/experiments/pgtextsearch/` — seed, ETL, index, query suite, eval report
- `control-plane/cmd/pg-textsearch-eval` — CLI: `seed|backfill|reindex|verify|eval|migrate`
- `control-plane/internal/lexical/` — `Search` (quoted query literals for lib/pq), HTTP handlers
- `control-plane/internal/app/config.go` — `LexicalConfig`
- `control-plane/internal/apiserver/router.go` — optional `/v1/experimental/lexical/search`
- `control-plane/configs/config.example.yaml` — commented `lexical:` block
- `Makefile` — `pg-textsearch-image`, `pg-textsearch-eval`, `lexical-backfill`, `lexical-reindex`, `lexical-verify`
- `.github/workflows/pg-textsearch-exploration.yml` — optional image build

## Container strategy

**Pre-built `.deb`** from Timescale releases inside `FROM pgvector/pgvector:pg18`, plus Compose `command` for `shared_preload_libraries`.

## ETL / reindex strategy

**Automated:** `control-plane/cmd/pg-textsearch-eval` + **`make lexical-backfill`**, **`make lexical-reindex`**, **`make lexical-verify`**, **`make pg-textsearch-eval`** (full pipeline + artifacts). See [pg-textsearch-etl.md](pg-textsearch-etl.md).

## Indexing shape

**Projection table** (`lexical_memory_projection` default) — **not** direct BM25 on `memories`.

## Lexical path working

- **Library:** `lexical.Search(db, table, query, limit)`.
- **HTTP:** `POST /v1/experimental/lexical/search` when `lexical.experimental_http: true` and projection exists.

## Hybrid

- **Documented** in [pg-textsearch-hybrid-recall.md](pg-textsearch-hybrid-recall.md).
- **Not** integrated into `/v1/recall/compile` in this branch.

## Next steps

1. Tune `doc_text` and query suite from real recall traffic.
2. Wire hybrid candidate merge + RRF (documented in [pg-textsearch-hybrid-recall.md](pg-textsearch-hybrid-recall.md)).
3. Optional: enrich corpus via offline scripts (not required for `make pg-textsearch-eval`).
