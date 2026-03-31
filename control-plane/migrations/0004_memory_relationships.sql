-- Lightweight typed edges between canonical memories (not a graph DB; additive context only).

CREATE TABLE IF NOT EXISTS memory_relationships (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  from_memory_id UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
  to_memory_id UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
  relationship_type TEXT NOT NULL CHECK (relationship_type IN (
    'supports',
    'contradicts',
    'supersedes',
    'same_pattern_family',
    'derived_from'
  )),
  reason TEXT,
  source TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT memory_relationships_no_self CHECK (from_memory_id != to_memory_id),
  CONSTRAINT memory_relationships_unique_edge UNIQUE (from_memory_id, to_memory_id, relationship_type)
);

CREATE INDEX IF NOT EXISTS idx_memory_relationships_from ON memory_relationships(from_memory_id);
CREATE INDEX IF NOT EXISTS idx_memory_relationships_to ON memory_relationships(to_memory_id);
CREATE INDEX IF NOT EXISTS idx_memory_relationships_type ON memory_relationships(relationship_type);
