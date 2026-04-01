# Rollback (pg_textsearch experiment)

## Runtime (API)

- Set `lexical.experimental_http: false` or remove `lexical:` from config (default).
- **Restart controlplane** — `POST /v1/experimental/lexical/search` is not registered.

## Docker / Compose

- Remove overlay: use only `docker-compose.yml` with `image: pgvector/pgvector:pg18` (no `docker-compose.pg-textsearch.yml`).
- **Existing data volume:** if you enabled `shared_preload_libraries=pg_textsearch` and `CREATE EXTENSION`, switching back to stock pgvector without preload is fine for **data** that never used the extension; if extension objects exist, drop them before switching images:

```sql
DROP INDEX IF EXISTS ...;  -- bm25 indexes on projection
DROP TABLE IF EXISTS lexical_memory_projection;
DROP EXTENSION IF EXISTS pg_textsearch;
```

## Canonical memory

- **Never** `DROP` memories for this rollback.
- Projection is **disposable**.

## Code

- Feature branch can be abandoned; `internal/lexical` is isolated behind config and does not run on default config.

## CI

- Default workflow does **not** require pg_textsearch image; optional workflow builds it for the branch only.
