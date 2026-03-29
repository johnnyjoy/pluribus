package recall

import (
	"time"

	"control-plane/internal/memory"
)

// epochRefTime is used when every candidate has zero UpdatedAt so recency is still deterministic.
var epochRefTime = time.Unix(0, 0).UTC()

// RefTimeForRanking returns a deterministic "as of" instant for recency scoring.
// It is the maximum UpdatedAt among candidates (the newest row in the set). Identical DB
// rows therefore yield identical recency components across repeated compiles.
// If all UpdatedAt values are zero, epochRefTime is used so behavior stays fixed.
func RefTimeForRanking(objs []memory.MemoryObject) time.Time {
	var max time.Time
	for _, o := range objs {
		if o.UpdatedAt.After(max) {
			max = o.UpdatedAt
		}
	}
	if max.IsZero() {
		return epochRefTime
	}
	return max
}
