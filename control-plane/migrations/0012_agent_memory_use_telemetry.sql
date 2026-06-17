-- Phase 11I: agent memory-use telemetry persistence (idempotent).

CREATE TABLE IF NOT EXISTS agent_telemetry_sessions (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  started_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  ended_at        TIMESTAMPTZ,
  interface       TEXT NOT NULL DEFAULT 'rest',
  agent_id        TEXT NOT NULL DEFAULT '',
  client_name     TEXT NOT NULL DEFAULT '',
  tags            JSONB NOT NULL DEFAULT '[]'::jsonb,
  metadata_json   JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_agent_telemetry_sessions_started ON agent_telemetry_sessions(started_at DESC);

CREATE TABLE IF NOT EXISTS agent_recall_events (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id          UUID NOT NULL REFERENCES agent_telemetry_sessions(id) ON DELETE CASCADE,
  task_id             TEXT NOT NULL DEFAULT '',
  interface           TEXT NOT NULL DEFAULT 'rest',
  recall_request_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  recall_bundle_id    TEXT NOT NULL DEFAULT '',
  recalled_memory_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  recall_bundle_json  JSONB NOT NULL DEFAULT '{}'::jsonb,
  recall_mode         TEXT NOT NULL DEFAULT 'current',
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_recall_events_session ON agent_recall_events(session_id, created_at DESC);

CREATE TABLE IF NOT EXISTS agent_memory_decisions (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  recall_event_id       UUID NOT NULL REFERENCES agent_recall_events(id) ON DELETE CASCADE,
  memory_id             TEXT NOT NULL,
  decision              TEXT NOT NULL,
  reason                TEXT NOT NULL DEFAULT '',
  contract_fields_cited JSONB NOT NULL DEFAULT '[]'::jsonb,
  output_facts_supported JSONB NOT NULL DEFAULT '[]'::jsonb,
  violation_codes       JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_memory_decisions_recall ON agent_memory_decisions(recall_event_id);
CREATE INDEX IF NOT EXISTS idx_agent_memory_decisions_memory ON agent_memory_decisions(memory_id);

CREATE TABLE IF NOT EXISTS agent_output_events (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id       UUID NOT NULL REFERENCES agent_telemetry_sessions(id) ON DELETE CASCADE,
  task_id          TEXT NOT NULL DEFAULT '',
  recall_event_id  UUID REFERENCES agent_recall_events(id) ON DELETE SET NULL,
  output_facts     JSONB NOT NULL DEFAULT '[]'::jsonb,
  output_actions   JSONB NOT NULL DEFAULT '[]'::jsonb,
  memory_citations JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_output_events_session ON agent_output_events(session_id, created_at DESC);

CREATE TABLE IF NOT EXISTS agent_obedience_evaluations (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id       UUID NOT NULL REFERENCES agent_telemetry_sessions(id) ON DELETE CASCADE,
  task_id          TEXT NOT NULL DEFAULT '',
  recall_event_id  UUID NOT NULL REFERENCES agent_recall_events(id) ON DELETE CASCADE,
  output_id        UUID REFERENCES agent_output_events(id) ON DELETE SET NULL,
  obedience_passed BOOLEAN NOT NULL DEFAULT false,
  obedience_score  DOUBLE PRECISION NOT NULL DEFAULT 0,
  violations       JSONB NOT NULL DEFAULT '[]'::jsonb,
  evaluator_version TEXT NOT NULL DEFAULT 'phase11i',
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_obedience_eval_session ON agent_obedience_evaluations(session_id, created_at DESC);

CREATE TABLE IF NOT EXISTS agent_memory_use_violations (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  evaluation_id  UUID NOT NULL REFERENCES agent_obedience_evaluations(id) ON DELETE CASCADE,
  memory_id      TEXT NOT NULL DEFAULT '',
  violation_code TEXT NOT NULL,
  severity       TEXT NOT NULL DEFAULT 'error',
  details_json   JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_memory_use_violations_eval ON agent_memory_use_violations(evaluation_id);
CREATE INDEX IF NOT EXISTS idx_agent_memory_use_violations_memory ON agent_memory_use_violations(memory_id);
CREATE INDEX IF NOT EXISTS idx_agent_memory_use_violations_code ON agent_memory_use_violations(violation_code);

CREATE TABLE IF NOT EXISTS agent_utility_candidates (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  memory_id      TEXT NOT NULL,
  evaluation_id  UUID NOT NULL REFERENCES agent_obedience_evaluations(id) ON DELETE CASCADE,
  signal_type    TEXT NOT NULL,
  signal_strength DOUBLE PRECISION NOT NULL DEFAULT 0,
  safe_to_apply  BOOLEAN NOT NULL DEFAULT false,
  reason         TEXT NOT NULL DEFAULT '',
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_utility_candidates_memory ON agent_utility_candidates(memory_id);
CREATE INDEX IF NOT EXISTS idx_agent_utility_candidates_eval ON agent_utility_candidates(evaluation_id);

COMMENT ON TABLE agent_telemetry_sessions IS 'Phase 11I memory-use telemetry session.';
COMMENT ON TABLE agent_recall_events IS 'Recall bundle exposure events for obedience evaluation.';
COMMENT ON TABLE agent_memory_decisions IS 'Agent-reported memory use decisions (validated on evaluate).';
COMMENT ON TABLE agent_output_events IS 'Agent output facts/actions linked to recall.';
COMMENT ON TABLE agent_obedience_evaluations IS 'Deterministic obedience evaluation results.';
COMMENT ON TABLE agent_memory_use_violations IS 'Persisted contract violation rows.';
COMMENT ON TABLE agent_utility_candidates IS 'Evaluator-validated utility signals; not auto-applied.';
