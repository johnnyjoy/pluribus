-- EXAMPLE ONLY — not applied by control-plane migrate.Apply.
-- Run manually after CREATE EXTENSION pg_textsearch; on an experimental database.

CREATE TABLE IF NOT EXISTS lexical_memory_projection (
  memory_id UUID PRIMARY KEY REFERENCES memories(id) ON DELETE CASCADE,
  doc_text TEXT NOT NULL
);

-- After bulk load:
-- CREATE INDEX lexical_memory_projection_bm25 ON lexical_memory_projection
--   USING bm25 (doc_text) WITH (text_config = 'english');
