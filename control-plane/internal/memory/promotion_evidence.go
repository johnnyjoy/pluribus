package memory

import (
	"context"

	"github.com/google/uuid"
)

// PromotionEvidenceLinker links evidence records to a memory after promote.
// Implemented by *evidence.Service.
type PromotionEvidenceLinker interface {
	LinkPromotedEvidence(ctx context.Context, memoryID uuid.UUID, evidenceIDs []uuid.UUID) error
}
