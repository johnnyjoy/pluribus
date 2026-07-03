package contradiction

import (
	"context"
	"errors"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

var ErrSelfContradiction = errors.New("memory_id and conflict_with_id must differ")

// MemoryForConflict is used for attribute-based conflict detection (Task 78). *memory.Repo implements it.
type MemoryForConflict interface {
	GetByID(ctx context.Context, id uuid.UUID) (*memory.MemoryObject, error)
	GetAttributes(ctx context.Context, memoryID uuid.UUID) (map[string]string, error)
}

// Service handles contradiction records and conflict detection (Task 78).
type Service struct {
	Repo       *Repo
	MemoryRepo MemoryForConflict // for GetByID and GetAttributes (conflict detection); can be *memory.Repo
	// Utility optional: records refuted demotion on contradiction create (Phase 7).
	Utility UtilityDemoter
}

// UtilityDemoter records bounded utility demotion when contradictions are created.
type UtilityDemoter interface {
	RecordContradictionDemotion(ctx context.Context, memoryID, conflictWithID uuid.UUID) error
}

// Create creates a contradiction record.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Record, error) {
	if req.MemoryID == req.ConflictWithID {
		return nil, ErrSelfContradiction
	}
	rec, err := s.Repo.Create(ctx, req)
	if err != nil {
		return nil, err
	}
	if s.Utility != nil {
		_ = s.Utility.RecordContradictionDemotion(ctx, req.MemoryID, req.ConflictWithID)
	}
	return rec, nil
}

// GetByID returns a contradiction record by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Record, error) {
	return s.Repo.GetByID(ctx, id)
}

// List returns contradiction records matching the request.
func (s *Service) List(ctx context.Context, req ListRequest) ([]Record, error) {
	return s.Repo.List(ctx, req)
}

// UpdateResolution updates the resolution state of a contradiction record.
func (s *Service) UpdateResolution(ctx context.Context, id uuid.UUID, state string) error {
	if !validResolutionState(state) {
		state = ResolutionUnresolved
	}
	return s.Repo.UpdateResolution(ctx, id, state)
}

// ListMemoryIDsInUnresolved returns all memory IDs involved in unresolved contradictions (for recall exclusion).
func (s *Service) ListMemoryIDsInUnresolved(ctx context.Context) ([]uuid.UUID, error) {
	return s.Repo.ListMemoryIDsInUnresolved(ctx)
}

// ListUnresolvedPairs returns memory_id/conflict_with_id pairs for unresolved records (RIU bounded_pair).
func (s *Service) ListUnresolvedPairs(ctx context.Context, limit int) ([][2]uuid.UUID, error) {
	return s.Repo.ListUnresolvedPairs(ctx, limit)
}

// DetectConflict returns true if the two active memories have overlapping attribute keys with different values (Task 78).
// If either memory is missing or not active, returns false. Uses memory_attributes for constraint-style comparison.
func (s *Service) DetectConflict(ctx context.Context, memoryID1, memoryID2 uuid.UUID) (bool, error) {
	if memoryID1 == memoryID2 {
		return false, nil
	}
	m1, err := s.MemoryRepo.GetByID(ctx, memoryID1)
	if err != nil || m1 == nil || m1.Status != api.StatusActive {
		return false, err
	}
	m2, err := s.MemoryRepo.GetByID(ctx, memoryID2)
	if err != nil || m2 == nil || m2.Status != api.StatusActive {
		return false, err
	}
	a1, err := s.MemoryRepo.GetAttributes(ctx, memoryID1)
	if err != nil {
		return false, err
	}
	a2, err := s.MemoryRepo.GetAttributes(ctx, memoryID2)
	if err != nil {
		return false, err
	}
	for k, v1 := range a1 {
		if v2, ok := a2[k]; ok && v1 != v2 {
			return true, nil
		}
	}
	return false, nil
}

// RecordDetected creates an unresolved contradiction link between two memories.
// Used by write-path detection (memory.Service contradiction-on-write): both rows
// are then held back from recall until an operator resolves the pair.
func (s *Service) RecordDetected(ctx context.Context, memoryID, conflictWithID uuid.UUID) error {
	_, err := s.Create(ctx, CreateRequest{
		MemoryID:        memoryID,
		ConflictWithID:  conflictWithID,
		ResolutionState: ResolutionUnresolved,
	})
	return err
}

// DetectAndRecord runs DetectConflict and, if conflict is found, creates an unresolved contradiction record.
// Returns the record if created, nil if no conflict.
func (s *Service) DetectAndRecord(ctx context.Context, memoryID, conflictWithID uuid.UUID) (*Record, error) {
	ok, err := s.DetectConflict(ctx, memoryID, conflictWithID)
	if err != nil || !ok {
		return nil, err
	}
	return s.Create(ctx, CreateRequest{
		MemoryID:        memoryID,
		ConflictWithID:  conflictWithID,
		ResolutionState: ResolutionUnresolved,
	})
}
