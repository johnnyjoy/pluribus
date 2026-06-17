-- Phase 2: agent loop compliance telemetry (idempotent).

CREATE TABLE IF NOT EXISTS agent_sessions (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  started_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  client_name     TEXT NOT NULL DEFAULT '',
  client_version  TEXT NOT NULL DEFAULT '',
  transport       TEXT NOT NULL DEFAULT 'http_mcp',
  repo_root       TEXT NOT NULL DEFAULT '',
  workspace_hint  TEXT NOT NULL DEFAULT '',
  metadata        JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_agent_sessions_last_seen ON agent_sessions(last_seen_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_sessions_client ON agent_sessions(client_name, started_at DESC);

CREATE TABLE IF NOT EXISTS agent_loop_events (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id        UUID NOT NULL REFERENCES agent_sessions(id) ON DELETE CASCADE,
  occurred_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  event_type        TEXT NOT NULL,
  tool_name         TEXT NOT NULL DEFAULT '',
  loop_role         TEXT NOT NULL DEFAULT '',
  risk_level        TEXT NOT NULL DEFAULT '',
  correlation_id    TEXT NOT NULL DEFAULT '',
  request_hash      TEXT NOT NULL DEFAULT '',
  request_summary   TEXT NOT NULL DEFAULT '',
  result_status     TEXT NOT NULL DEFAULT '',
  error_code        TEXT NOT NULL DEFAULT '',
  error_message     TEXT NOT NULL DEFAULT '',
  duration_ms       INT NOT NULL DEFAULT 0,
  enforcement_decision TEXT NOT NULL DEFAULT '',
  memory_id         TEXT NOT NULL DEFAULT '',
  metadata          JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_agent_loop_events_session_time ON agent_loop_events(session_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_agent_loop_events_type ON agent_loop_events(event_type, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_loop_events_tool ON agent_loop_events(tool_name, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_loop_events_correlation ON agent_loop_events(correlation_id) WHERE correlation_id <> '';

CREATE TABLE IF NOT EXISTS agent_loop_evaluations (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id    UUID NOT NULL REFERENCES agent_sessions(id) ON DELETE CASCADE,
  evaluated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  window_start  TIMESTAMPTZ NOT NULL,
  window_end    TIMESTAMPTZ NOT NULL,
  status        TEXT NOT NULL,
  missing_steps JSONB NOT NULL DEFAULT '[]'::jsonb,
  evidence      JSONB NOT NULL DEFAULT '{}'::jsonb,
  metadata      JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_agent_loop_evaluations_session ON agent_loop_evaluations(session_id, evaluated_at DESC);

COMMENT ON TABLE agent_sessions IS 'MCP/agent session for loop compliance correlation.';
COMMENT ON TABLE agent_loop_events IS 'Telemetry for MCP loop-relevant calls (redacted summaries, no raw secrets).';
COMMENT ON TABLE agent_loop_evaluations IS 'Point-in-time compliance evaluation for a session window.';
