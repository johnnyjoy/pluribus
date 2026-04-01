# ETL / reindex / verify (pg_textsearch)

## Philosophy

- First indexing shape is **not** permanent.
- Prefer **rebuild from canonical `memories`** over fragile incremental hacks until requirements stabilize.
- Every step should be **idempotent** or **replayable**.

## Extract

- **Source:** `SELECT id, statement, kind, ... FROM memories WHERE status = 'active'` (adjust filters per policy).
- **Joins:** optional evidence snippets — keep bounded byte limits.

## Transform

- Build `doc_text` string (see [pg-textsearch-data-model.md](pg-textsearch-data-model.md)).
- Normalize whitespace; optional language-specific config matches BM25 `text_config` (e.g. `english`).

## Load

- `INSERT INTO lexical_memory_projection ...` batch or `COPY`.
- After bulk load: `CREATE INDEX ... USING bm25(doc_text) WITH (text_config='english')`.
- Optional: `SELECT bm25_force_merge('index_name');` per upstream bulk-load guidance.

## Verify

- `COUNT(*)` projection vs eligible memory rows.
- `EXPLAIN` on `ORDER BY doc_text <@> 'sample' LIMIT 10` — confirm index use when data is large enough.
- Spot-check: known `memory_id` appears for a query.

## Automation (primary path)

| Command | Purpose |
|---------|---------|
| `make pg-textsearch-eval` | **One-shot:** ephemeral Docker Postgres (pg_textsearch image) → migrate → seed → reindex → verify → query suite → writes `docs/experiments/pg-textsearch-eval-latest.md` + `artifacts/pg-textsearch/eval.json` |
| `make lexical-backfill` | Upsert projection from active `memories` (DSN: `PG_TEXTSEARCH_EVAL_DSN` or `DATABASE_URL`) |
| `make lexical-reindex` | Drop BM25 index → truncate projection → backfill → recreate index |
| `make lexical-verify` | Fail if projection row count ≠ active memories |

Implementation: `control-plane/cmd/pg-textsearch-eval` (`seed`, `backfill`, `reindex`, `verify`, `eval`, `migrate`).

**Prerequisite:** Postgres must load `pg_textsearch` in `shared_preload_libraries` (see [pg-textsearch-container.md](pg-textsearch-container.md)).

Manual SQL in `scripts/experiments/` remains optional for debugging only.

## Failure recovery

- If ETL fails mid-batch: **truncate projection** and rerun backfill (canonical data intact).
- If BM25 index corrupt: `DROP INDEX` on projection, recreate.

## Example DDL (experimental)

See [sql/lexical_projection_example.sql](sql/lexical_projection_example.sql) (same tree under `docs/experiments/sql/`).
