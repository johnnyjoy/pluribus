-- Agent-driven curation chores: the server packages review work (unresolved
-- contradictions, quarantined rows, embedding near-duplicate pairs) as small
-- "chores" that visiting agents resolve. Nothing applies until min_resolvers
-- DISTINCT agents (AgentUsageKey hash) vote for the same action.
CREATE TABLE IF NOT EXISTS curation_chores (
    id UUID PRIMARY KEY,
    chore_type TEXT NOT NULL CHECK (chore_type IN ('contradiction', 'quarantine_review', 'duplicate_pair')),
    subject_memory_id UUID NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    related_memory_id UUID REFERENCES memories(id) ON DELETE CASCADE,
    -- evidence: type-specific context (contradiction_record_id, cosine similarity, quarantine reason...)
    evidence JSONB,
    state TEXT NOT NULL DEFAULT 'open' CHECK (state IN ('open', 'resolved', 'dismissed')),
    resolution_action TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ
);

-- At most one open chore per (type, subject, related) triple.
CREATE UNIQUE INDEX IF NOT EXISTS uq_curation_chores_open_triple
    ON curation_chores (chore_type, subject_memory_id,
        COALESCE(related_memory_id, '00000000-0000-0000-0000-000000000000'::uuid))
    WHERE state = 'open';

CREATE INDEX IF NOT EXISTS idx_curation_chores_open
    ON curation_chores (created_at) WHERE state = 'open';

CREATE TABLE IF NOT EXISTS curation_chore_votes (
    id UUID PRIMARY KEY,
    chore_id UUID NOT NULL REFERENCES curation_chores(id) ON DELETE CASCADE,
    agent_id TEXT NOT NULL,
    -- agent_hash = memory.AgentUsageKey(agent_id); distinctness is judged on the hash.
    agent_hash TEXT NOT NULL,
    action TEXT NOT NULL,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- One vote per agent per chore (no vote-changing; keeps corroboration auditable).
    CONSTRAINT uq_curation_chore_votes_agent UNIQUE (chore_id, agent_hash)
);
