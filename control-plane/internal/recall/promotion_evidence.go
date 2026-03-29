package recall

import (
	"context"

	"github.com/google/uuid"
)

// EvidencePolicyChecker scores evidence IDs for a project. Implemented by *evidence.Service.
type EvidencePolicyChecker interface {
	ScoreEvidenceIDs(ctx context.Context, evidenceIDs []uuid.UUID) (float64, error)
}

func uniqueEvidenceIDCount(ids []uuid.UUID) int {
	if len(ids) == 0 {
		return 0
	}
	seen := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		seen[id] = struct{}{}
	}
	return len(seen)
}
