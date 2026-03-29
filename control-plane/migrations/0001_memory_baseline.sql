-- Pluribus greenfield baseline: global tag-first memory (no container tables or columns).

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS memories (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  kind                 TEXT NOT NULL,
  statement            TEXT NOT NULL,
  statement_canonical  TEXT NOT NULL DEFAULT '',
  statement_key        TEXT NOT NULL DEFAULT '',
  dedup_key            TEXT NOT NULL DEFAULT '',
  payload              JSONB,
  authority            INT NOT NULL DEFAULT 5,
  applicability        TEXT NOT NULL DEFAULT 'governing',
  status               TEXT NOT NULL DEFAULT 'active',
  deprecated_at        TIMESTAMPTZ,
  ttl_seconds          INT,
  embedding            vector(1536),
  created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_memories_kind_status ON memories(kind, status);

CREATE TABLE IF NOT EXISTS memories_tags (
  memory_id UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
  tag       TEXT NOT NULL,
  PRIMARY KEY (memory_id, tag)
);

CREATE INDEX IF NOT EXISTS idx_memories_tags_tag ON memories_tags(tag);

CREATE TABLE IF NOT EXISTS memory_links (
  from_memory_id UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
  to_memory_id   UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
  link_type      TEXT NOT NULL,
  PRIMARY KEY (from_memory_id, to_memory_id, link_type)
);

CREATE TABLE IF NOT EXISTS evidence_records (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  digest      TEXT NOT NULL,
  path        TEXT NOT NULL,
  kind        TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_evidence_records_digest ON evidence_records(digest);

CREATE TABLE IF NOT EXISTS memory_evidence_links (
  memory_id   UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
  evidence_id UUID NOT NULL REFERENCES evidence_records(id) ON DELETE CASCADE,
  PRIMARY KEY (memory_id, evidence_id)
);

CREATE TABLE IF NOT EXISTS drift_checks (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  passed      BOOLEAN NOT NULL,
  violations  JSONB,
  warnings    JSONB,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS contradiction_records (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  memory_id           UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
  conflict_with_id    UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
  resolution_state    TEXT NOT NULL DEFAULT 'unresolved' CHECK (resolution_state IN ('unresolved', 'override', 'deprecated', 'narrow_exception')),
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT contradiction_pair_unique UNIQUE (memory_id, conflict_with_id),
  CONSTRAINT contradiction_no_self CHECK (memory_id != conflict_with_id)
);

CREATE INDEX IF NOT EXISTS idx_contradiction_records_resolution ON contradiction_records(resolution_state);
CREATE INDEX IF NOT EXISTS idx_contradiction_records_memory_id ON contradiction_records(memory_id);
CREATE INDEX IF NOT EXISTS idx_contradiction_records_conflict_with ON contradiction_records(conflict_with_id);

CREATE TABLE IF NOT EXISTS memory_attributes (
  memory_id   UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
  attr_key    TEXT NOT NULL,
  attr_value  TEXT NOT NULL,
  PRIMARY KEY (memory_id, attr_key)
);

CREATE INDEX IF NOT EXISTS idx_memory_attributes_key ON memory_attributes(attr_key);

CREATE TABLE IF NOT EXISTS candidate_events (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  raw_text         TEXT NOT NULL,
  salience_score   FLOAT,
  promotion_status TEXT NOT NULL DEFAULT 'pending',
  proposal_json    JSONB,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_candidate_events_pending_created ON candidate_events (promotion_status, created_at DESC);

CREATE TABLE IF NOT EXISTS recall_bundles (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  payload    JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ingestion_records (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  temp_contributor_id TEXT NOT NULL,
  status              TEXT NOT NULL CHECK (status IN ('accepted', 'rejected', 'processed')),
  rejected_reason     TEXT,
  payload_raw         JSONB NOT NULL,
  context_window_hash TEXT,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ingestion_records_created_at ON ingestion_records(created_at DESC);

CREATE TABLE IF NOT EXISTS temp_contributor_profiles (
  temp_contributor_id TEXT PRIMARY KEY,
  trust_weight   DOUBLE PRECISION NOT NULL DEFAULT 1.0,
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS canonical_fact_extractions (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ingestion_id     UUID NOT NULL REFERENCES ingestion_records(id) ON DELETE CASCADE,
  subject_norm     TEXT NOT NULL,
  predicate_norm   TEXT NOT NULL,
  object_norm      TEXT NOT NULL,
  confidence       DOUBLE PRECISION NOT NULL DEFAULT 0,
  provenance       JSONB NOT NULL DEFAULT '{}',
  normalized_hash  TEXT NOT NULL,
  source_index     INT NOT NULL DEFAULT 0,
  priority_score   DOUBLE PRECISION NOT NULL DEFAULT 0,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_canonical_fact_extractions_ingestion ON canonical_fact_extractions(ingestion_id);
CREATE INDEX IF NOT EXISTS idx_canonical_fact_extractions_hash ON canonical_fact_extractions(normalized_hash);
CREATE INDEX IF NOT EXISTS idx_canonical_fact_extractions_subject ON canonical_fact_extractions(subject_norm);
CREATE INDEX IF NOT EXISTS idx_canonical_fact_extractions_subject_predicate ON canonical_fact_extractions(subject_norm, predicate_norm);

CREATE TABLE IF NOT EXISTS canonical_fact_lineage (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ingestion_id UUID REFERENCES ingestion_records(id) ON DELETE SET NULL,
  fact_hash    TEXT NOT NULL,
  parent_hash  TEXT,
  root_hash    TEXT NOT NULL,
  merge_type   TEXT NOT NULL CHECK (merge_type IN ('reinforce', 'similar_unify')),
  source       TEXT NOT NULL CHECK (source IN ('batch', 'db', 'promotion')),
  meta         JSONB NOT NULL DEFAULT '{}',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_canonical_fact_lineage_fact ON canonical_fact_lineage(fact_hash);
CREATE INDEX IF NOT EXISTS idx_canonical_fact_lineage_root ON canonical_fact_lineage(root_hash);
CREATE INDEX IF NOT EXISTS idx_canonical_fact_lineage_ingestion ON canonical_fact_lineage(ingestion_id);

CREATE TABLE IF NOT EXISTS canonical_fact_contradictions (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ingestion_id       UUID REFERENCES ingestion_records(id) ON DELETE SET NULL,
  subject_norm       TEXT NOT NULL,
  predicate_norm_a   TEXT NOT NULL,
  predicate_norm_b   TEXT NOT NULL,
  fact_hash_a        TEXT NOT NULL,
  fact_hash_b        TEXT NOT NULL,
  contradiction_type TEXT NOT NULL,
  confidence_delta   DOUBLE PRECISION,
  detected_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT canonical_fact_contradictions_hash_order CHECK (fact_hash_a < fact_hash_b),
  CONSTRAINT canonical_fact_contradictions_unique UNIQUE (fact_hash_a, fact_hash_b, contradiction_type)
);

CREATE INDEX IF NOT EXISTS idx_canonical_fact_contradictions_subject ON canonical_fact_contradictions(subject_norm);

CREATE TABLE IF NOT EXISTS advisory_episodes (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  summary_text      TEXT NOT NULL,
  source            TEXT NOT NULL CHECK (source IN ('manual', 'digest', 'ingestion_summary')),
  tags              JSONB NOT NULL DEFAULT '[]'::jsonb,
  related_memory_id UUID REFERENCES memories(id) ON DELETE SET NULL,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_advisory_episodes_created ON advisory_episodes(created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS uq_memories_dedup_key
ON memories (kind, dedup_key, statement_key)
WHERE status IN ('active', 'pending') AND statement_key <> '';
