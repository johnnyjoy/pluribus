package similarity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrSimilarityDisabled is returned when the feature is off in config.
var ErrSimilarityDisabled = errors.New("similarity: feature disabled")

// ValidSources for advisory_episodes.source (must match DB CHECK).
var ValidSources = map[string]struct{}{
	"manual":             {},
	"digest":             {},
	"ingestion_summary": {},
}

// Record is a persisted advisory episode row.
type Record struct {
	ID              uuid.UUID  `json:"id"`
	SummaryText     string     `json:"summary_text"`
	Source          string     `json:"source"`
	Tags            []string   `json:"tags"`
	RelatedMemoryID *uuid.UUID `json:"related_memory_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	// OccurredAt is when the episode occurred (optional; advisory only).
	OccurredAt *time.Time `json:"occurred_at,omitempty"`
	// Entities are normalized strings (e.g. people, systems) for overlap filters; not a graph.
	Entities []string `json:"entities,omitempty"`
}

// CreateRequest is POST /v1/advisory-episodes body.
type CreateRequest struct {
	Summary         string     `json:"summary"`
	Tags            []string   `json:"tags,omitempty"`
	Source          string     `json:"source"`
	RelatedMemoryID *uuid.UUID `json:"related_memory_id,omitempty"`
	OccurredAt      *time.Time `json:"occurred_at,omitempty"`
	Entities        []string   `json:"entities,omitempty"`
}

// SimilarRequest is POST /v1/advisory-episodes/similar body.
type SimilarRequest struct {
	Query      string   `json:"query"`
	Tags       []string `json:"tags,omitempty"`
	MaxResults int      `json:"max_results,omitempty"`
	// OccurredAfter / OccurredBefore filter by effective time COALESCE(occurred_at, created_at). Optional; not container scoping.
	OccurredAfter  *time.Time `json:"occurred_after,omitempty"`
	OccurredBefore *time.Time `json:"occurred_before,omitempty"`
	// Entity matches if any request entity overlaps episode entities (normalized).
	Entity   string   `json:"entity,omitempty"`
	Entities []string `json:"entities,omitempty"`
}

// SimilarResponse wraps advisory results (always subordinate to canonical memory).
type SimilarResponse struct {
	AdvisorySimilarCases []AdvisorySimilarCase `json:"advisory_similar_cases"`
}

// AdvisorySimilarCase is one ranked similar episode (non-binding).
type AdvisorySimilarCase struct {
	ID                 string    `json:"id"`
	Summary            string    `json:"summary"`
	Source             string    `json:"source"`
	Tags               []string  `json:"tags,omitempty"`
	ResemblanceScore   float64   `json:"resemblance_score"`
	ResemblanceSignals []string  `json:"resemblance_signals,omitempty"` // local signals only, e.g. lexical_overlap
	Advisory           bool      `json:"advisory"`
	CreatedAt          time.Time `json:"created_at"`
	RelatedMemoryID    *string   `json:"related_memory_id,omitempty"`
	OccurredAt         *time.Time `json:"occurred_at,omitempty"`
	Entities           []string   `json:"entities,omitempty"`
}
