-- Allow advisory_episodes.source = 'mcp' for MCP-originated episodic ingestion.
-- Replay-safe: table may already be renamed to advisory_experiences.

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'advisory_episodes'
  ) THEN
    ALTER TABLE advisory_episodes DROP CONSTRAINT IF EXISTS advisory_episodes_source_check;
    ALTER TABLE advisory_episodes ADD CONSTRAINT advisory_episodes_source_check
      CHECK (source IN ('manual', 'digest', 'ingestion_summary', 'mcp'));
  ELSIF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'advisory_experiences'
  ) THEN
    ALTER TABLE advisory_experiences DROP CONSTRAINT IF EXISTS advisory_episodes_source_check;
    ALTER TABLE advisory_experiences DROP CONSTRAINT IF EXISTS advisory_experiences_source_check;
    ALTER TABLE advisory_experiences ADD CONSTRAINT advisory_experiences_source_check
      CHECK (source IN ('manual', 'digest', 'ingestion_summary', 'mcp'));
  END IF;
END $$;
