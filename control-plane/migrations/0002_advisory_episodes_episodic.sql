-- Advisory-only episodic fields (subordinate to canonical memories; not enforcement truth).

ALTER TABLE advisory_episodes
  ADD COLUMN IF NOT EXISTS occurred_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS entities JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE INDEX IF NOT EXISTS idx_advisory_episodes_effective_time ON advisory_episodes ((COALESCE(occurred_at, created_at)) DESC);

COMMENT ON COLUMN advisory_episodes.occurred_at IS 'When the episode occurred. NULL means unspecified (COALESCE with created_at for filters).';
COMMENT ON COLUMN advisory_episodes.entities IS 'JSON array of normalized entity strings for overlap filters. Advisory only.';
