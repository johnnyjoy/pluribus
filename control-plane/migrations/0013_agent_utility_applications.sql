-- Phase 11K: guarded utility application audit ledger (idempotent).

CREATE TABLE IF NOT EXISTS agent_utility_applications (
  application_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  candidate_id          UUID NOT NULL,
  memory_id             TEXT NOT NULL,
  evaluation_id         UUID NOT NULL,
  decision              TEXT NOT NULL,
  delta                 DOUBLE PRECISION NOT NULL DEFAULT 0,
  previous_utility_score DOUBLE PRECISION NOT NULL DEFAULT 0,
  new_utility_score     DOUBLE PRECISION NOT NULL DEFAULT 0,
  policy_version        TEXT NOT NULL DEFAULT 'phase11k-v1',
  reason                TEXT NOT NULL DEFAULT '',
  evidence_json         JSONB NOT NULL DEFAULT '[]'::jsonb,
  rollback_token        TEXT NOT NULL DEFAULT '',
  applied_by            TEXT NOT NULL DEFAULT 'system',
  session_id            UUID,
  agent_id              TEXT NOT NULL DEFAULT '',
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  reverted_at           TIMESTAMPTZ,
  revert_reason         TEXT NOT NULL DEFAULT '',
  CONSTRAINT agent_utility_applications_decision_check CHECK (
    decision IN ('apply_positive', 'apply_negative', 'record_only', 'review_required', 'reject')
  ),
  CONSTRAINT agent_utility_applications_candidate_unique UNIQUE (candidate_id)
);

CREATE INDEX IF NOT EXISTS idx_agent_utility_applications_memory
  ON agent_utility_applications(memory_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_utility_applications_evaluation
  ON agent_utility_applications(evaluation_id);
CREATE INDEX IF NOT EXISTS idx_agent_utility_applications_session
  ON agent_utility_applications(session_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_utility_applications_rollback
  ON agent_utility_applications(rollback_token) WHERE rollback_token <> '';
