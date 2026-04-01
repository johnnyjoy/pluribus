-- Advisory rows renamed to advisory_experiences; track inline memory formation vs reject bucket.
-- Idempotent for replay on boot (see internal/migrate).

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'advisory_episodes'
  ) AND NOT EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'advisory_experiences'
  ) THEN
    ALTER TABLE advisory_episodes RENAME TO advisory_experiences;
  END IF;
END $$;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
             WHERE c.relkind = 'i' AND c.relname = 'idx_advisory_episodes_created' AND n.nspname = 'public')
     AND NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
             WHERE c.relkind = 'i' AND c.relname = 'idx_advisory_experiences_created' AND n.nspname = 'public') THEN
    ALTER INDEX idx_advisory_episodes_created RENAME TO idx_advisory_experiences_created;
  END IF;
END $$;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
             WHERE c.relkind = 'i' AND c.relname = 'idx_advisory_episodes_effective_time' AND n.nspname = 'public')
     AND NOT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
             WHERE c.relkind = 'i' AND c.relname = 'idx_advisory_experiences_effective_time' AND n.nspname = 'public') THEN
    ALTER INDEX idx_advisory_episodes_effective_time RENAME TO idx_advisory_experiences_effective_time;
  END IF;
END $$;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.table_constraints
    WHERE table_schema = 'public' AND table_name = 'advisory_experiences'
      AND constraint_name = 'advisory_episodes_source_check'
  ) THEN
    ALTER TABLE advisory_experiences RENAME CONSTRAINT advisory_episodes_source_check TO advisory_experiences_source_check;
  END IF;
END $$;

ALTER TABLE advisory_experiences
  ADD COLUMN IF NOT EXISTS memory_formation_status TEXT NOT NULL DEFAULT 'none',
  ADD COLUMN IF NOT EXISTS rejection_reason TEXT;

ALTER TABLE advisory_experiences DROP CONSTRAINT IF EXISTS advisory_experiences_formation_status_check;
ALTER TABLE advisory_experiences ADD CONSTRAINT advisory_experiences_formation_status_check
  CHECK (memory_formation_status IN ('none', 'linked', 'rejected'));

COMMENT ON TABLE advisory_experiences IS 'Advisory experience ingest log: reject bucket + traceability to probationary memory when linked.';
COMMENT ON COLUMN advisory_experiences.memory_formation_status IS 'none=pending or legacy; linked=probationary memory created; rejected=low signal at ingest.';
