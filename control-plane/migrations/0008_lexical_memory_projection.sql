-- Lexical projection table for BM25 (pg_textsearch). Index creation uses CREATE EXTENSION pg_textsearch
-- and is done by tooling (see cmd/pg-textsearch-eval) — not here, so CI on stock Postgres still applies.
CREATE TABLE IF NOT EXISTS lexical_memory_projection (
  memory_id UUID PRIMARY KEY REFERENCES memories(id) ON DELETE CASCADE,
  doc_text    TEXT NOT NULL
);

-- Optional comment for operators.
COMMENT ON TABLE lexical_memory_projection IS 'Derived from canonical memories; rebuildable; not source of truth.';
