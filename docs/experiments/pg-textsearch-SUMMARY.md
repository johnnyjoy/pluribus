# pg_textsearch branch — summary (workstream 12)

## Files added/changed (high level)

- `docker/pg-textsearch/Dockerfile` — pg18 + pgvector + pg_textsearch `.deb`
- `docker-compose.pg-textsearch.yml` — overlay: `shared_preload_libraries`, custom image
- `scripts/experiments/pg_textsearch_smoke.sql` — extension + BM25 smoke
- `docs/experiments/*.md` — design, container, ETL, hybrid, filtering, benchmarks, rollback
- `docs/experiments/sql/lexical_projection_example.sql` — example projection DDL
- `control-plane/internal/lexical/` — `Search`, `Handlers` for experimental HTTP
- `control-plane/internal/app/config.go` — `LexicalConfig`
- `control-plane/internal/apiserver/router.go` — optional `/v1/experimental/lexical/search`
- `control-plane/configs/config.example.yaml` — commented `lexical:` block
- `Makefile` — `pg-textsearch-image`
- `.github/workflows/pg-textsearch-exploration.yml` — optional image build

## Container strategy

**Pre-built `.deb`** from Timescale releases inside `FROM pgvector/pgvector:pg18`, plus Compose `command` for `shared_preload_libraries`.

## ETL / reindex strategy

**Documented** (projection table + backfill + rebuild). **Makefile targets** `lexical-backfill` / `reindex` / `verify` — **not implemented**; use manual SQL or future scripts.

## Indexing shape

**Projection table** (`lexical_memory_projection` default) — **not** direct BM25 on `memories`.

## Lexical path working

- **Library:** `lexical.Search(db, table, query, limit)`.
- **HTTP:** `POST /v1/experimental/lexical/search` when `lexical.experimental_http: true` and projection exists.

## Hybrid

- **Documented** in [pg-textsearch-hybrid-recall.md](pg-textsearch-hybrid-recall.md).
- **Not** integrated into `/v1/recall/compile` in this branch.

## Next steps

1. Populate projection via ETL script.
2. Add `make lexical-backfill` / verify scripts.
3. Offline benchmark harness.
4. Prototype hybrid candidate merge + RRF.
