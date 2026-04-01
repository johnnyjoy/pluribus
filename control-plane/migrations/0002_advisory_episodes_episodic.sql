-- Advisory-only episodic fields (subordinate to canonical memories; not enforcement truth).
-- Replay-safe: target whichever advisory table exists (episodes pre-rename, experiences after 0006).

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'advisory_episodes'
  ) THEN
    ALTER TABLE advisory_episodes
      ADD COLUMN IF NOT EXISTS occurred_at TIMESTAMPTZ,
      ADD COLUMN IF NOT EXISTS entities JSONB NOT NULL DEFAULT '[]'::jsonb;
    CREATE INDEX IF NOT EXISTS idx_advisory_episodes_effective_time ON advisory_episodes ((COALESCE(occurred_at, created_at)) DESC);
    COMMENT ON COLUMN advisory_episodes.occurred_at IS 'When the episode occurred. NULL means unspecified (COALESCE with created_at for filters).';
    COMMENT ON COLUMN advisory_episodes.entities IS 'JSON array of normalized entity strings for overlap filters. Advisory only.';
  ELSIF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'advisory_experiences'
  ) THEN
    ALTER TABLE advisory_experiences
      ADD COLUMN IF NOT EXISTS occurred_at TIMESTAMPTZ,
      ADD COLUMN IF NOT EXISTS entities JSONB NOT NULL DEFAULT '[]'::jsonb;
    CREATE INDEX IF NOT EXISTS idx_advisory_experiences_effective_time ON advisory_experiences ((COALESCE(occurred_at, created_at)) DESC);
    COMMENT ON COLUMN advisory_experiences.occurred_at IS 'When the episode occurred. NULL means unspecified (COALESCE with created_at for filters).';
    COMMENT ON COLUMN advisory_experiences.entities IS 'JSON array of normalized entity strings for overlap filters. Advisory only.';
  END IF;
END $$;
