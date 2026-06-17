-- Phase 7: memory utility event ledger and bounded utility scores (separate from authority).

CREATE TABLE IF NOT EXISTS memory_utility_events (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  memory_id           UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
  event_type          TEXT NOT NULL,
  event_weight        DOUBLE PRECISION NOT NULL DEFAULT 0,
  source              TEXT NOT NULL DEFAULT 'agent',
  source_tool         TEXT,
  source_session_id   TEXT,
  correlation_id      TEXT,
  recall_bundle_id    TEXT,
  agent_loop_event_id UUID,
  reason              TEXT NOT NULL DEFAULT '',
  evidence_id         UUID REFERENCES evidence_records(id) ON DELETE SET NULL,
  payload             JSONB NOT NULL DEFAULT '{}',
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT memory_utility_events_type_check CHECK (
    event_type IN (
      'helpful', 'harmful', 'wrong', 'outdated', 'irrelevant',
      'confirmed', 'refuted', 'duplicate_seen'
    )
  )
);

CREATE INDEX IF NOT EXISTS idx_memory_utility_events_memory_id
  ON memory_utility_events(memory_id);
CREATE INDEX IF NOT EXISTS idx_memory_utility_events_event_type
  ON memory_utility_events(event_type);
CREATE INDEX IF NOT EXISTS idx_memory_utility_events_created_at
  ON memory_utility_events(created_at DESC);

CREATE TABLE IF NOT EXISTS memory_utility_scores (
  memory_id         UUID PRIMARY KEY REFERENCES memories(id) ON DELETE CASCADE,
  utility_score     DOUBLE PRECISION NOT NULL DEFAULT 0,
  helpful_count     INT NOT NULL DEFAULT 0,
  harmful_count     INT NOT NULL DEFAULT 0,
  wrong_count       INT NOT NULL DEFAULT 0,
  outdated_count    INT NOT NULL DEFAULT 0,
  irrelevant_count  INT NOT NULL DEFAULT 0,
  confirmed_count   INT NOT NULL DEFAULT 0,
  refuted_count     INT NOT NULL DEFAULT 0,
  last_positive_at  TIMESTAMPTZ,
  last_negative_at  TIMESTAMPTZ,
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT memory_utility_scores_range CHECK (utility_score >= -10 AND utility_score <= 10)
);

CREATE INDEX IF NOT EXISTS idx_memory_utility_scores_utility_score
  ON memory_utility_scores(utility_score);
