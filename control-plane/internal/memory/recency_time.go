package memory

import "time"

// EffectiveRecencyTime is the instant used for recall ranking recency and contradiction tie-breaks:
// the underlying event time when set, otherwise the last system update time.
// This matches SQL COALESCE(occurred_at, updated_at) for persisted rows.
func EffectiveRecencyTime(m MemoryObject) time.Time {
	if m.OccurredAt != nil && !m.OccurredAt.IsZero() {
		return *m.OccurredAt
	}
	return m.UpdatedAt
}
