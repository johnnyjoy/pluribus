-- Phase 10C: embedding metadata for staleness and model drift detection.

ALTER TABLE memories ADD COLUMN IF NOT EXISTS embedding_model TEXT;
ALTER TABLE memories ADD COLUMN IF NOT EXISTS embedding_provider TEXT;
ALTER TABLE memories ADD COLUMN IF NOT EXISTS embedding_dimension INT;
ALTER TABLE memories ADD COLUMN IF NOT EXISTS embedding_source_hash TEXT;
ALTER TABLE memories ADD COLUMN IF NOT EXISTS embedding_created_at TIMESTAMPTZ;
ALTER TABLE memories ADD COLUMN IF NOT EXISTS embedding_updated_at TIMESTAMPTZ;
ALTER TABLE memories ADD COLUMN IF NOT EXISTS embedding_status TEXT;

CREATE INDEX IF NOT EXISTS idx_memories_embedding_status ON memories(embedding_status)
  WHERE embedding IS NOT NULL;
