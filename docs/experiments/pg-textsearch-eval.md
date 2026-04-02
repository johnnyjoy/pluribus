# pg_textsearch — evaluation and ETL

How to run **backfill**, **reindex**, **verify**, and the **one-shot eval harness**. For architecture and boundaries, see [pg-textsearch.md](pg-textsearch.md).

---

## Commands (Makefile)

| Target | Purpose |
|--------|---------|
| `make pg-textsearch-eval` | Ephemeral Docker Postgres (pg_textsearch image) → migrate → replace-seed → reindex → verify → query suite → writes **`artifacts/pg-textsearch/eval.json`** and **`artifacts/pg-textsearch/eval-summary.md`** |
| `make lexical-backfill` | Upsert `lexical_memory_projection` from active `memories` |
| `make lexical-reindex` | Drop BM25 index → truncate projection → backfill → recreate index |
| `make lexical-verify` | Exit non-zero if projection row count ≠ active `memories` (active) |

DSN: **`PG_TEXTSEARCH_EVAL_DSN`** or **`DATABASE_URL`** (defaults in Makefile assume local `controlplane`).

---

## CLI

```bash
cd control-plane
go run ./cmd/pg-textsearch-eval -dsn="$PG_TEXTSEARCH_EVAL_DSN" <seed|backfill|reindex|verify|eval|migrate>
```

Flags: `-replace-seed`, `-skip-migrate`, `-artifact-dir`, `-markdown` (override summary path), `-query-limit`.

Implementation: `control-plane/cmd/pg-textsearch-eval`, package `internal/experiments/pgtextsearch`.

---

## ETL philosophy

- Prefer **rebuild from canonical `memories`** over fragile incremental hacks until requirements stabilize.
- **Extract:** active memories + tags → **Transform:** `doc_text` string → **Load:** upsert projection → **Index:** `CREATE INDEX … USING bm25(doc_text) WITH (text_config='english')`.
- Optional after bulk load: `bm25_force_merge` (called by reindex path when available).
- **Failure recovery:** truncate projection, rerun backfill — canonical rows untouched.

---

## Eval artifacts (generated, not committed)

After `make pg-textsearch-eval`:

- `artifacts/pg-textsearch/eval.json` — machine-readable report (gitignored).
- `artifacts/pg-textsearch/eval-summary.md` — human-readable table + recommendation (gitignored).

Use these for go/no-go on the lexical path; re-run the target to refresh.

---

## Query suite (harness)

Deterministic seeded memories (~110 rows) and categorized queries (exact constraint, failure recall, pattern, ugly operator, mixed). Records per-query latency, BM25 hits, naive `ILIKE` baseline IDs, and a **plausible** heuristic for quick signal — not publication IR metrics.

---

## Hybrid and filtering (notes)

**Hybrid (planned):** fetch top-N lexical and top-N semantic IDs; merge; consider RRF or normalized weighted fusion; apply authority/recency in recall layer — **not** wired into `/v1/recall/compile` yet.

**Filtering:** For selective filters (tags, kind), pre-filter then `ORDER BY doc_text <@> … LIMIT k`. For governance (authority, applicability), prefer **after** candidate retrieval unless benchmarks prove selective pre-indexes. Avoid encoding policy-only rules into `doc_text`. See upstream pg_textsearch README for pre- vs post-filter tradeoffs.

---

## Smoke SQL

`scripts/experiments/pg_textsearch_smoke.sql` — minimal `CREATE EXTENSION`, table, BM25 index, sample query (for manual debugging).
