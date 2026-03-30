package similarity

import (
	"context"

	"github.com/google/uuid"
)

// AutoDistiller runs post-ingest distillation (same logic as POST /v1/episodes/distill) when enabled in config.
type AutoDistiller interface {
	DistillAfterAdvisoryIngest(ctx context.Context, episodeID uuid.UUID) error
}
