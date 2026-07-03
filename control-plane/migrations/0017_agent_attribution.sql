-- Agent attribution (remediation plan, Phase 3): every memory and advisory
-- episode records which agent authored it. Enables per-agent trust, poison
-- tracing (C2 provenance), and distinct-agent corroboration (authority may only
-- rise on confirmation from a different agent).
ALTER TABLE memories ADD COLUMN IF NOT EXISTS agent_id TEXT;
CREATE INDEX IF NOT EXISTS idx_memories_agent_id
  ON memories(agent_id) WHERE agent_id IS NOT NULL;

ALTER TABLE advisory_experiences ADD COLUMN IF NOT EXISTS agent_id TEXT;
CREATE INDEX IF NOT EXISTS idx_advisory_experiences_agent_id
  ON advisory_experiences(agent_id) WHERE agent_id IS NOT NULL;
