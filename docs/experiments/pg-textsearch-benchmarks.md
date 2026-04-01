# Benchmarks (lexical path)

## Intent

Answer whether BM25 improves **candidate quality** for classes of queries before wiring hybrid recall.

## Suggested categories

1. **Exact constraint recall** — verbatim or near-verbatim constraint phrases.
2. **Prior failure recall** — incident wording, error codes.
3. **Pattern recall** — paraphrased near-duplicates.
4. **Mixed hybrid** — same query run through lexical + semantic; compare ID overlap.
5. **Short / ugly queries** — operator-style fragments; BM25 often beats embedding-only.

## Deliverables (future)

- Small Go test harness under `control-plane/internal/lexical` or `internal/eval` with **fixture memories** and expected IDs.
- Or a **script** that loads fixture JSON, runs `POST /api/.../search`, asserts metrics.

## Metrics (starter)

- Precision@k / nDCG@k on labeled pairs (manual labels OK for experiments).
- Latency p50/p95 for `Search` SQL.

## Current state

- **`make pg-textsearch-eval`** runs a **branch-local query suite** (exact / failure / pattern / ugly / mixed), records per-query latency, compares top statements to naive `ILIKE` baselines, and writes **`docs/experiments/pg-textsearch-eval-latest.md`** plus **`artifacts/pg-textsearch/eval.json`**. This is an evaluation harness, not publication-grade IR metrics.
- CI does not gate on this path yet; optional workflow builds the pg_textsearch image only.
