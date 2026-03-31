-- Allow advisory_episodes.source = 'mcp' for MCP-originated episodic ingestion.

ALTER TABLE advisory_episodes DROP CONSTRAINT IF EXISTS advisory_episodes_source_check;
ALTER TABLE advisory_episodes ADD CONSTRAINT advisory_episodes_source_check
  CHECK (source IN ('manual', 'digest', 'ingestion_summary', 'mcp'));
