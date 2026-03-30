-- Canonical memory: optional event time (when the underlying fact/event occurred), distinct from created_at / updated_at.

ALTER TABLE memories ADD COLUMN IF NOT EXISTS occurred_at TIMESTAMPTZ;

COMMENT ON COLUMN memories.occurred_at IS 'When the described event or fact took place. NULL means unspecified (recency falls back to updated_at).';
