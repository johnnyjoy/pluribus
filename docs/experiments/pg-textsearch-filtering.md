# Filtering and query shape (pg_textsearch + Pluribus)

Upstream [pg_textsearch](https://github.com/timescale/pg_textsearch) documents **pre-filtering** (B-tree / selective `WHERE` then `ORDER BY ... LIMIT`) vs **post-filtering** (BM25 top-k then `WHERE`), with tradeoffs similar to pgvector.

## Pluribus mapping

| Filter | Prefer | Notes |
|--------|--------|--------|
| **Tags** | Pre-filter if selective (e.g. `<10%` of corpus) | B-tree index on tag column in projection |
| **Kind** (constraint/failure/pattern) | Pre-filter when few kinds | Low cardinality — often selective |
| **Entity** | Pre-filter if selective | Else risk expensive BM25 on huge sets |
| **Authority / applicability** | **After** candidate IDs | These are governance rules, not BM25 helpers |
| **Applicability** | Post-filter or merge in recall layer | Avoid encoding policy in raw text |

## LIMIT placement

- **BM25 top-k:** `ORDER BY doc_text <@> $query LIMIT $k` in SQL.
- **Application-level:** if post-filtering removes rows, **over-fetch** `LIMIT` (e.g. 3×) then `LIMIT` again in Go — document in handlers.

## Anti-patterns

- Huge `WHERE` match + BM25 on millions of rows without selective pre-index.
- Pushing **authority** into `doc_text` to “make BM25 work” — governance belongs in recall logic.
- `ORDER BY` without `LIMIT` on large corpora.

## References

- Upstream README: filtering and `ORDER BY ... LIMIT` sections.
- pgvector filtering analogy (linked from pg_textsearch README).
