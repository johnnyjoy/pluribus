# Embedding Staleness Policy

Phase 10C policy for pgvector embeddings in Pluribus. **Semantic recall remains disabled by default.**

## What text is embedded

Embedding input is `EmbeddingTextForMemory(kind, statementCanonical, statement)`:

```text
"{kind}: {normalized statement}"
```

Normalization: trim, collapse whitespace; canonical statement preferred over raw statement.

## Source hash

`embedding_source_hash` is SHA-256 (hex) of the exact embedding text above. Used to detect statement/kind drift without re-reading the vector.

## When embedding is created

- On memory create, when semantic retrieval is **enabled in config** and configured internal embedding plumbing is present, `maybeEmbedOnCreate` computes the vector and writes metadata (`embedding_model`, `embedding_provider`, `embedding_dimension`, `embedding_source_hash`, `embedding_created_at`, `embedding_updated_at`, `embedding_status=valid`).
- Benchmark live mode (`hybrid_live`) embeds fixture rows at stub init using `PLURIBUS_EMBEDDER_*` env (evaluation only).

## When embedding becomes stale

An embedding is stale if any of:

| Condition | Reason code |
|-----------|-------------|
| No stored vector | `missing_embedding` |
| Metadata missing source hash | `missing_metadata` |
| Source hash ≠ current embedding text | `source_hash_mismatch` |
| Model ≠ configured model | `model_mismatch` |
| Dimension ≠ configured dimension | `dimension_mismatch` |
| Status `stale` or `failed` | `status_invalid` |

## Search behavior

- Production vector SQL (`SearchSimilar`) excludes rows where `embedding_status != 'valid'` or `embedding_source_hash IS NULL` (`embeddingFreshSQL`).
- Live hybrid benchmark stub skips stale/missing labels and reports counts.
- Stale embeddings must not be silently trusted in live benchmark passes (`stale_embedding_count` must be 0 for gate pass).

## Model change

When the configured embedding model or dimension changes, existing rows are stale until re-embedded. Detection is via `model_mismatch` / `dimension_mismatch`; no automatic backfill in Phase 10C.

## Embedder failure

- Query embed failure → lexical-only recall; `semantic_retrieval.fallback_reason` set (e.g. `embedding_failed`, `dimension_mismatch`).
- Core lexical recall must continue; embedder outage is not a recall outage.

## Lexical fallback

Lexical retrieval is always available. Semantic/hybrid paths merge extra candidates when embedding succeeds; on failure, compile proceeds with lexical candidates only.

## Production semantic

**Disabled by default.** Deterministic hybrid gates (Phase 10B) do not enable production semantic. Internal operator benchmarks (`make recall-benchmark-real-embedder`) are opt-in server-side checks only; they skip without configured internal plumbing and are **not** agent-facing.

## Re-embed (minimal)

No production backfill pipeline in Phase 10C. Re-embed is operational: update memory text or model config, detect staleness, re-run embed on create/update paths when implemented. Optional future: `embeddings-check-stale` admin command.
