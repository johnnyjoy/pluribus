package evidence

import (
	"context"
	"errors"
	"math"

	"control-plane/internal/memory"

	"github.com/google/uuid"
)

var (
	ErrContentRequired   = errors.New("evidence: content is required")
	ErrNoStorage         = errors.New("evidence: storage not configured")
	ErrNoRepo            = errors.New("evidence: repo not configured")
	ErrNotFound          = errors.New("evidence: record not found")
	ErrLinkIDsRequired   = errors.New("evidence: memory_id and evidence_id required for link")
)

// MemoryAuthorityUpdater is used to adjust memory authority from evidence score (Task 79). *memory.Repo implements it.
type MemoryAuthorityUpdater interface {
	GetByID(ctx context.Context, id uuid.UUID) (*memory.MemoryObject, error)
	UpdateAuthority(ctx context.Context, id uuid.UUID, authority int) error
}

// Service provides evidence create, list, get, link, and scoring use cases.
type Service struct {
	Repo               *Repo
	Storage            *Storage
	Memory             MemoryAuthorityUpdater  // optional: adjust memory authority when linking evidence
	AuthorityFactor    float64                 // evidence_score * factor added to authority (0 = no adjustment); default 0.1
}

// Create decodes content, saves to storage (by kind+digest), then creates DB record.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Record, error) {
	if req.Content == "" {
		return nil, ErrContentRequired
	}
	if s.Storage == nil {
		return nil, ErrNoStorage
	}
	path, digest, err := s.Storage.Save(req.Kind, req.Digest, req.Content)
	if err != nil {
		return nil, err
	}
	if s.Repo == nil {
		return nil, ErrNoRepo
	}
	return s.Repo.Create(ctx, digest, path, req.Kind)
}

// List returns evidence records, optionally filtered by kind (query param).
func (s *Service) List(ctx context.Context, kind string) ([]Record, error) {
	if s.Repo == nil {
		return nil, ErrNoRepo
	}
	return s.Repo.List(ctx, kind)
}

// Get returns one evidence record by id, or ErrNotFound.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Record, error) {
	if s.Repo == nil {
		return nil, ErrNoRepo
	}
	rec, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, ErrNotFound
	}
	return rec, nil
}

// LinkEvidenceToMemory links an evidence record to a memory object.
// If Memory is set and AuthorityFactor > 0, recomputes evidence score for the memory and updates its authority (Task 79).
func (s *Service) LinkEvidenceToMemory(ctx context.Context, memoryID, evidenceID uuid.UUID) error {
	if s.Repo == nil {
		return ErrNoRepo
	}
	if memoryID == uuid.Nil || evidenceID == uuid.Nil {
		return ErrLinkIDsRequired
	}
	if err := s.Repo.CreateLink(ctx, memoryID, evidenceID); err != nil {
		return err
	}
	if s.Memory != nil && s.AuthorityFactor > 0 {
		score, err := s.ComputeEvidenceScore(ctx, memoryID)
		if err != nil {
			return err
		}
		obj, err := s.Memory.GetByID(ctx, memoryID)
		if err != nil || obj == nil {
			return nil // link succeeded; skip authority update if memory missing
		}
		delta := score * s.AuthorityFactor
		newAuth := int(math.Round(float64(obj.Authority) + delta))
		if newAuth < 0 {
			newAuth = 0
		}
		if newAuth > 10 {
			newAuth = 10
		}
		if newAuth != obj.Authority {
			_ = s.Memory.UpdateAuthority(ctx, memoryID, newAuth)
		}
	}
	return nil
}

// ListEvidenceForMemory returns all evidence linked to the memory (Task 79: traceability).
func (s *Service) ListEvidenceForMemory(ctx context.Context, memoryID uuid.UUID) ([]Record, error) {
	if s.Repo == nil {
		return nil, ErrNoRepo
	}
	return s.Repo.ListEvidenceByMemory(ctx, memoryID)
}

// ComputeEvidenceScore returns the average base score of all evidence linked to the memory (Task 79).
// Returns 0 if the memory has no linked evidence.
func (s *Service) ComputeEvidenceScore(ctx context.Context, memoryID uuid.UUID) (float64, error) {
	if s.Repo == nil {
		return 0, ErrNoRepo
	}
	list, err := s.Repo.ListEvidenceByMemory(ctx, memoryID)
	if err != nil {
		return 0, err
	}
	if len(list) == 0 {
		return 0, nil
	}
	var sum float64
	for _, rec := range list {
		sum += BaseScore(rec.Kind)
	}
	return sum / float64(len(list)), nil
}
