-- Formation status is only linked (accepted → memory) or rejected (reject bucket). No pending/none.

UPDATE advisory_experiences
SET memory_formation_status = 'linked'
WHERE memory_formation_status = 'none'
  AND related_memory_id IS NOT NULL;

UPDATE advisory_experiences
SET memory_formation_status = 'rejected',
    rejection_reason = COALESCE(NULLIF(TRIM(rejection_reason), ''), 'migrated_legacy_none')
WHERE memory_formation_status = 'none';

ALTER TABLE advisory_experiences
  ALTER COLUMN memory_formation_status SET DEFAULT 'rejected';

ALTER TABLE advisory_experiences DROP CONSTRAINT IF EXISTS advisory_experiences_formation_status_check;

ALTER TABLE advisory_experiences ADD CONSTRAINT advisory_experiences_formation_status_check
  CHECK (memory_formation_status IN ('linked', 'rejected'));

COMMENT ON COLUMN advisory_experiences.memory_formation_status IS 'linked=probationary memory created at ingest; rejected=low signal or intake-only (not memory).';
