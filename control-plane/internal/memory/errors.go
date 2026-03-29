package memory

import (
	"fmt"

	"github.com/google/uuid"
)

// ErrDuplicateMemory is returned when an insert would duplicate an active/pending row
// at the same project, kind, normalized scope, and statement key (Phase C).
type ErrDuplicateMemory struct {
	ExistingID uuid.UUID
}

func (e *ErrDuplicateMemory) Error() string {
	return fmt.Sprintf("duplicate memory: existing id %s", e.ExistingID.String())
}
