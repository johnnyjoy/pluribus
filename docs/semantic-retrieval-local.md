# Local Semantic Retrieval (Ollama embedder)

Semantic recall is **on by default** using a local, no-API-key embedder. Memories
are embedded at write time; recall runs pgvector cosine search alongside lexical
retrieval and feeds the hybrid ranker. If the embedder is down or misconfigured,
recall **gracefully falls back to lexical** (look for `[SEMANTIC FALLBACK]` logs)
— nothing breaks.

## Setup (one-time)

1. Install [Ollama](https://ollama.com) on the Pluribus host (or any host the
   server can reach):

```bash
curl -fsSL https://ollama.com/install.sh | sh
ollama pull nomic-embed-text
```

2. Ollama serves an OpenAI-compatible `POST /v1/embeddings` on port 11434 by
   default. The default config already points at it:

```yaml
recall:
  semantic_retrieval:
    enabled: true
    embedding_endpoint: "http://127.0.0.1:11434/v1"
    embedding_model: "nomic-embed-text"
    embedding_dimensions: 768
```

3. Backfill embeddings for memories created before the embedder was enabled:

```bash
curl -sS -X POST http://127.0.0.1:8123/v1/memory/embeddings/backfill \
  -H 'Content-Type: application/json' -H "X-API-Key: $PLURIBUS_API_KEY" \
  -d '{"limit": 200}'
```

Repeat until `remaining` reaches 0 (each call processes up to `limit` rows).
The response reports `scanned`, `embedded`, `failed`, `skipped`, `remaining`.

## How it works

- **Write time**: `POST /v1/memory` (and probationary formation from
  `record_experience`) embeds `kind: statement_canonical` and stores the vector
  plus metadata (model, provider, dimension, source hash) on the row.
- **Recall time**: the compile path embeds the situation query and unions
  pgvector top-K with lexical candidates; cosine similarity feeds the
  `weight_semantic_similarity` ranking term.
- **Staleness**: rows embedded under a different model/dimension or whose text
  changed (source-hash mismatch) are excluded from vector search and picked up
  by the next backfill run.

## Changing models

The pgvector column is dimension-typed
(`migrations/0015_local_embedder_dimensions.sql` sets `vector(768)` for
`nomic-embed-text`). Switching to a model with different dimensions (e.g. OpenAI
`text-embedding-3-small` at 1536) requires a matching migration plus a full
backfill. Embeddings are derived data — safe to rebuild, never precious.

## Verifying

- `[SEMANTIC MATCH]` logs (set `log_semantic_matches: true`) show vector hits.
- The compile response's `semantic_retrieval` debug block reports
  `path: hybrid` on success or the `fallback_reason` when lexical-only.
