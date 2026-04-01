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

- **No** automated benchmark in CI yet; smoke SQL proves correctness only.
- Add harness when projection table is populated in dev.
