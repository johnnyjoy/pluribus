# Hybrid recall experiment plan (lexical + semantic + authority)

## Goal

Combine **BM25** (pg_textsearch), **semantic** (pgvector), **authority/recency/tags** (existing recall compiler) without a single fragile weight soup.

## Implemented now (this branch)

- **Lexical:** `internal/lexical.Search` + optional `POST /v1/experimental/lexical/search` when `lexical.experimental_http: true`.
- **Semantic + authority:** unchanged recall pipeline (`/v1/recall/compile`).

## Planned next

1. **Candidate union:** fetch top-N lexical `memory_id`s and top-N semantic IDs; merge with set semantics.
2. **Rank fusion:** evaluate **RRF** (reciprocal rank fusion) or weighted score normalization — benchmark before locking.
3. **Authority gating:** apply existing ranking / applicability **after** candidate IDs are gathered, not inside BM25 `WHERE` unless selective pre-filters are proven necessary.

## Benchmarking

- See [pg-textsearch-benchmarks.md](pg-textsearch-benchmarks.md).
- Do **not** ship production hybrid weights until offline eval shows lift on representative queries.

## Distinction

| Layer | Status |
|-------|--------|
| BM25 projection + HTTP | Scaffold |
| Recall compiler integration | Not merged — keep composable |
| RRF / hybrid | Documented only |
