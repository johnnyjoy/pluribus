package memory

import "strings"

// DurableHistoricalTags block TTL archive when present on a memory row.
// Conservative gate: these tags signal doctrine, architecture, or audit-worthy history.
var DurableHistoricalTags = []string{
	"doctrine", "objective", "architecture", "api", "contract", "decision",
	"incident", "failure", "lesson", "phase", "report", "historical", "durable",
}

// OccurredAtMaterialDeltaSeconds is the minimum |occurred_at - created_at| gap
// that blocks TTL archive (event time materially differs from ingestion time).
const OccurredAtMaterialDeltaSeconds = 86400 // 24h

// ttlHistoricalValueExclusionSQL excludes memories with obvious historical-value signals
// from TTL expiration candidacy. Archive remains non-destructive; this gate prevents
// blind age-off of rows that already show durable or relational history.
const ttlHistoricalValueExclusionSQL = `
  AND NOT EXISTS (SELECT 1 FROM memory_utility_events e WHERE e.memory_id = m.id)
  AND NOT EXISTS (
    SELECT 1 FROM memory_utility_scores s
    WHERE s.memory_id = m.id AND s.utility_score <> 0
  )
  AND NOT EXISTS (SELECT 1 FROM memory_evidence_links el WHERE el.memory_id = m.id)
  AND NOT EXISTS (
    SELECT 1 FROM memory_relationships r
    WHERE r.from_memory_id = m.id OR r.to_memory_id = m.id
  )
  AND NOT EXISTS (
    SELECT 1 FROM contradiction_records c
    WHERE c.memory_id = m.id OR c.conflict_with_id = m.id
  )
  AND NOT (
    m.occurred_at IS NOT NULL
    AND ABS(EXTRACT(EPOCH FROM (m.occurred_at - m.created_at))) > $3
  )
  AND NOT EXISTS (
    SELECT 1 FROM memories_tags t
    WHERE t.memory_id = m.id AND lower(t.tag) = ANY($4)
  )
  AND (m.payload IS NULL OR NOT (m.payload ? 'superseded_by'))
  AND (
    m.payload IS NULL
    OR NOT (
      m.payload->'pluribus_evolution' ? 'superseded_by'
      OR m.payload->'pluribus_evolution' ? 'invalidated_by'
      OR m.payload->'pluribus_evolution' ? 'contradicts'
    )
  )`

func isDurableHistoricalTag(tag string) bool {
	t := strings.ToLower(strings.TrimSpace(tag))
	for _, d := range DurableHistoricalTags {
		if t == d {
			return true
		}
	}
	return false
}
