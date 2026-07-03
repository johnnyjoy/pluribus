-- Hybrid candidate retrieval (remediation plan H1): the compile candidate pool
-- was ORDER BY authority DESC LIMIT 100, which silently drops relevant memories
-- as the pool grows. Candidates now union three slices:
--   1. authority-top slice        (existing Search)
--   2. pgvector ANN top-K         (existing SearchSimilar, hnsw index in 0015)
--   3. full-text top-K            (this migration: tsvector + GIN)
-- ranked afterwards by the existing scorer.
ALTER TABLE memories ADD COLUMN IF NOT EXISTS statement_tsv tsvector
  GENERATED ALWAYS AS (
    to_tsvector('english', coalesce(statement, '') || ' ' || coalesce(statement_canonical, ''))
  ) STORED;

CREATE INDEX IF NOT EXISTS idx_memories_statement_tsv
  ON memories USING gin (statement_tsv);
