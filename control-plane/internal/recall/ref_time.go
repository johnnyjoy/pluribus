package recall

import (
	"time"

	"control-plane/internal/memory"
)

// epochRefTime is used when every candidate has zero effective recency time so scoring is deterministic.
var epochRefTime = time.Unix(0, 0).UTC()

// RefTimeForRanking returns a deterministic "as of" instant for recency scoring.
// It is the maximum effective recency time among candidates (COALESCE(occurred_at, updated_at)).
// If all such values are zero, epochRefTime is used so behavior stays fixed.
func RefTimeForRanking(objs []memory.MemoryObject) time.Time {
	var max time.Time
	for _, o := range objs {
		if t := memory.EffectiveRecencyTime(o); t.After(max) {
			max = t
		}
	}
	if max.IsZero() {
		return epochRefTime
	}
	return max
}
