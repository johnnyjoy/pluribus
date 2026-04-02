# pg_textsearch (BM25 lexical layer)

This document is the **entry point** for lexical retrieval in Pluribus: architecture, boundaries, and pointers to setup, eval tooling, and rollback.

---

## Problem

Recall combines authority, triggers, and optional **semantic** retrieval (pgvector). **Lexical** gaps remain: near-duplicate phrasing, short operator-style queries, and exact constraint/failure wording without embedding overlap.

[pg_textsearch](https://github.com/timescale/pg_textsearch) adds BM25 inside PostgreSQL (17/18): `USING bm25(...)`, `ORDER BY doc_text <@> 'query'`. It stays next to pgvector in one database.

---

## Canonical memory vs projection (non-negotiable)

| Layer | Role |
|--------|------|
| **`memories` and related tables** | **Source of truth** — authority, applicability, relationships, provenance. |
| **`lexical_memory_projection`** | **Derived** — rebuildable; BM25 indexes `doc_text` only. |
| **pg_textsearch** | Retrieval/index extension — **not** canonical storage. |

> **Truth line:** Pluribus memory rows remain the source of truth. `pg_textsearch` is an auxiliary lexical index layer.

---

## What is integrated vs evaluation tooling

| Category | Components |
|----------|------------|
| **Production-path infrastructure** | Migration `0008_lexical_memory_projection.sql`; `internal/lexical` (`Search`, optional HTTP); `LexicalConfig` + `/v1/experimental/lexical/search` when enabled. |
| **Development / eval** | `cmd/pg-textsearch-eval`; `internal/experiments/pgtextsearch` (seed, ETL, suite); `make pg-textsearch-eval`, `make lexical-*`; Docker image + compose overlay. |

Eval seed data and query suites are **for harnesses**, not a claim about production recall behavior.

---

## Data model (summary)

- **Projection table** (default `lexical_memory_projection`): `memory_id` → `memories(id)`, `doc_text` for BM25.
- **Typical `doc_text`:** `kind` + `statement` + flattened tags (see ETL implementation). Prefer projection over BM25 directly on `memories` to keep index lifecycle separate from truth DDL.
- **Updates:** backfill/reindex from canonical rows; truncate projection is safe — `memories` unchanged.

---

## Future work (not merged into core recall)

- **Hybrid:** union lexical + semantic candidates, then rank fusion (e.g. RRF) — see notes in [pg-textsearch-eval.md](pg-textsearch-eval.md#hybrid-and-filtering-notes).
- **Filtering:** pre-filter (selective `WHERE` + BM25 top-k) vs post-filter — upstream pg_textsearch docs; authority/applicability belong in recall logic, not stuffed into `doc_text`.

---

## Where to read next

| Doc | Purpose |
|-----|---------|
| [pg-textsearch-eval.md](pg-textsearch-eval.md) | Commands, CLI, ETL, eval artifacts |
| [pg-textsearch-container.md](pg-textsearch-container.md) | Docker image, `shared_preload_libraries`, smoke SQL |
| [pg-textsearch-rollback.md](pg-textsearch-rollback.md) | Disable lexical path, revert compose, drop projection |

Example DDL (non-authoritative): [sql/lexical_projection_example.sql](sql/lexical_projection_example.sql).
