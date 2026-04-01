# pg_textsearch evaluation (BM25 lexical layer)

## Problem

Pluribus recall combines authority, triggers, and optional **semantic** retrieval (pgvector). **Lexical** gaps remain: near-duplicate phrasing, short operator-style queries, and exact constraint/failure wording without embedding overlap.

## Why [pg_textsearch](https://github.com/timescale/pg_textsearch)

- PostgreSQL **17 and 18** extension with **BM25** (`USING bm25(...)`) and `<@>` ordering.
- Stays **inside Postgres** next to pgvector — hybrid and governance stay in one database.
- Rebuildable indexes; bulk-load + `bm25_force_merge()` workflow documented upstream.

## What stays canonical

- **`memories`** and related tables: authority, applicability, relationships, provenance.
- Advisory / episodic pipelines unchanged in role.
- **No** migration of source truth into the search index.

## What becomes indexable

A **projection / indexing table** (name configurable; default `lexical_memory_projection`) holding:

- `memory_id` (FK to canonical row)
- `doc_text` (concatenated searchable text derived from statement + optional fields)
- Optional narrow columns for **pre-filtering** (kind, tags) — see [pg-textsearch-filtering.md](pg-textsearch-filtering.md)

The BM25 index lives on `doc_text` only; canonical rows are never “the index.”

## Migration / ETL model

- **Extract** from `memories` (and optionally normalized fields).
- **Transform** into projection row shape (string concatenation, language config).
- **Load** into projection table; **CREATE INDEX ... USING bm25**.
- **Verify** row counts and spot-check queries.
- **Rollback** by dropping projection table / disabling API — see [pg-textsearch-rollback.md](pg-textsearch-rollback.md).

## Success

- Extension runs in **containerized** dev stack (`docker-compose.pg-textsearch.yml`).
- One-command or documented path to **smoke** BM25 query.
- **ETL/rebuild** path documented and scriptable.
- **Experimental** HTTP `POST /v1/experimental/lexical/search` returns ranked `memory_id` when enabled.

## Rollback

- Turn off `lexical.experimental_http` in config.
- Stop using overlay image; revert to `pgvector/pgvector:pg18` only.
- `DROP TABLE` projection / `DROP EXTENSION` if needed — canonical `memories` unaffected.

---

> **Truth line:** Pluribus memory rows remain the source of truth. `pg_textsearch` is an auxiliary lexical index layer.
