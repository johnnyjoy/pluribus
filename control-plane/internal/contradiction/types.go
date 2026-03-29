package contradiction

import (
	"time"

	"github.com/google/uuid"
)

// ResolutionState is the state of a contradiction record.
const (
	ResolutionUnresolved     = "unresolved"
	ResolutionOverride       = "override"
	ResolutionDeprecated     = "deprecated"
	ResolutionNarrowException = "narrow_exception"
)

// Record is a contradiction between two memories (Task 78).
type Record struct {
	ID             uuid.UUID `json:"id"`
	MemoryID       uuid.UUID `json:"memory_id"`
	ConflictWithID uuid.UUID `json:"conflict_with_id"`
	ResolutionState string   `json:"resolution_state"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CreateRequest is the body for creating a contradiction record.
type CreateRequest struct {
	MemoryID       uuid.UUID `json:"memory_id"`
	ConflictWithID uuid.UUID `json:"conflict_with_id"`
	ResolutionState string   `json:"resolution_state,omitempty"` // default unresolved
}

// UpdateResolutionRequest is the body for updating resolution state.
type UpdateResolutionRequest struct {
	ResolutionState string `json:"resolution_state"` // unresolved, override, deprecated, narrow_exception
}

// ListRequest filters for listing contradiction records.
type ListRequest struct {
	ResolutionState string    `json:"resolution_state,omitempty"` // filter by state
	MemoryID        uuid.UUID `json:"memory_id,omitempty"`         // filter by memory involved
	Limit           int       `json:"limit,omitempty"`
}
