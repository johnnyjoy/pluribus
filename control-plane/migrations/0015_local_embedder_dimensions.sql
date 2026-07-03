-- Local semantic retrieval (remediation plan, Phase 2 semantic-local).
--
-- The default embedder is now a local Ollama-compatible model
-- (nomic-embed-text, 768 dimensions) instead of OpenAI text-embedding-3-small
-- (1536). pgvector columns are dimension-typed, so the column is re-typed when
-- its current dimension differs. Embeddings are DERIVED data (rebuildable via
-- POST /v1/memory/embeddings/backfill); canonical memory rows are never touched.
--
-- Idempotency: guarded on the column's current typmod so replaying this file on
-- every boot never wipes embeddings that already match the target dimension.
DO $$
DECLARE
  cur_dim int;
BEGIN
  SELECT atttypmod INTO cur_dim
  FROM pg_attribute
  WHERE attrelid = 'memories'::regclass AND attname = 'embedding' AND NOT attisdropped;

  IF cur_dim IS DISTINCT FROM 768 THEN
    -- Re-typing drops old-dimension vectors (derived data; backfill rebuilds them).
    ALTER TABLE memories ALTER COLUMN embedding TYPE vector(768) USING NULL;
    UPDATE memories
       SET embedding_status      = 'pending',
           embedding_model       = NULL,
           embedding_provider    = NULL,
           embedding_dimension   = NULL,
           embedding_source_hash = NULL,
           embedding_created_at  = NULL,
           embedding_updated_at  = NULL
     WHERE embedding_status IS NOT NULL
        OR embedding_model IS NOT NULL
        OR embedding_source_hash IS NOT NULL;
  END IF;
END $$;

-- ANN index for hybrid candidate retrieval (cosine distance).
CREATE INDEX IF NOT EXISTS idx_memories_embedding_hnsw
  ON memories USING hnsw (embedding vector_cosine_ops);
