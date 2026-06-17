package utility

import (
	"context"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// MemoryRepoAdapter implements MemoryReader using memory.Repo.
type MemoryRepoAdapter struct {
	Repo *memory.Repo
}

// GetByID reports existence and key fields for utility demotion logic.
func (a *MemoryRepoAdapter) GetByID(ctx context.Context, id uuid.UUID) (bool, api.MemoryKind, api.Applicability, api.Status, error) {
	if a == nil || a.Repo == nil {
		ok, err := false, ErrNoRepo
		return ok, "", "", "", err
	}
	obj, err := a.Repo.GetByID(ctx, id)
	if err != nil {
		return false, "", "", "", err
	}
	if obj == nil {
		return false, "", "", "", nil
	}
	return true, obj.Kind, obj.Applicability, obj.Status, nil
}

// SetStatusPending moves memory to pending review.
func (a *MemoryRepoAdapter) SetStatusPending(ctx context.Context, id uuid.UUID) error {
	if a == nil || a.Repo == nil {
		return ErrNoRepo
	}
	return a.Repo.UpdateStatus(ctx, id, api.StatusPending)
}
