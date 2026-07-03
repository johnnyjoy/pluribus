-- Remediation lifecycle (hostile audit 2026-07, findings C2/C3):
-- status gains two values (column is unconstrained TEXT, so no ALTER TYPE needed):
--   'quarantined' — stored but never surfaced by recall; awaiting review
--                   (harmful-advice screen at ingest, operator quarantine, or
--                    authority exhausted by repeated failure/contradiction events)
--   'deleted'     — soft-delete tombstone; excluded from all recall including
--                   historical mode. Canonical rows are never hard-deleted.
-- status_reason records why a row was quarantined or deleted.
ALTER TABLE memories ADD COLUMN IF NOT EXISTS status_reason TEXT;

CREATE INDEX IF NOT EXISTS idx_memories_remediation_status
  ON memories (status)
  WHERE status IN ('quarantined', 'deleted');
