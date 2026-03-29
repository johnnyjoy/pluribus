package evidence

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// LinkPromotedEvidence validates each evidence ID exists, then links to memoryID.
// Duplicate IDs in evidenceIDs are skipped after first successful link (CreateLink is idempotent).
func (s *Service) LinkPromotedEvidence(ctx context.Context, memoryID uuid.UUID, evidenceIDs []uuid.UUID) error {
	if len(evidenceIDs) == 0 {
		return nil
	}
	if s.Repo == nil {
		return ErrNoRepo
	}
	seen := make(map[uuid.UUID]struct{})
	for _, eid := range evidenceIDs {
		if eid == uuid.Nil {
			return fmt.Errorf("%w: nil evidence id", ErrNotFound)
		}
		if _, ok := seen[eid]; ok {
			continue
		}
		seen[eid] = struct{}{}
		rec, err := s.Repo.GetByID(ctx, eid)
		if err != nil {
			return err
		}
		if rec == nil {
			return fmt.Errorf("%w: %s", ErrNotFound, eid)
		}
		if err := s.LinkEvidenceToMemory(ctx, memoryID, eid); err != nil {
			return err
		}
	}
	return nil
}

// ScoreEvidenceIDs returns the average BaseScore for the given evidence records.
func (s *Service) ScoreEvidenceIDs(ctx context.Context, evidenceIDs []uuid.UUID) (float64, error) {
	if len(evidenceIDs) == 0 {
		return 0, nil
	}
	if s.Repo == nil {
		return 0, ErrNoRepo
	}
	seen := make(map[uuid.UUID]struct{})
	var sum float64
	var n int
	for _, eid := range evidenceIDs {
		if eid == uuid.Nil {
			return 0, fmt.Errorf("%w: nil evidence id", ErrNotFound)
		}
		if _, ok := seen[eid]; ok {
			continue
		}
		seen[eid] = struct{}{}
		rec, err := s.Repo.GetByID(ctx, eid)
		if err != nil {
			return 0, err
		}
		if rec == nil {
			return 0, fmt.Errorf("%w: %s", ErrNotFound, eid)
		}
		sum += BaseScore(rec.Kind)
		n++
	}
	if n == 0 {
		return 0, nil
	}
	return sum / float64(n), nil
}
