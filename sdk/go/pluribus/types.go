package pluribus

import "time"

// MemoryItem is the minimal recall memory shape useful to agents.
type MemoryItem struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Statement string `json:"statement"`
	Authority int    `json:"authority"`
}

// RecallBundle is returned by POST /v1/recall/compile.
type RecallBundle struct {
	GoverningConstraints []MemoryItem `json:"governing_constraints"`
	Decisions            []MemoryItem `json:"decisions"`
	KnownFailures        []MemoryItem `json:"known_failures"`
	ApplicablePatterns   []MemoryItem `json:"applicable_patterns"`
	Continuity           []MemoryItem `json:"continuity"`
	Constraints          []MemoryItem `json:"constraints"`
	Experience           []MemoryItem `json:"experience"`
	RecallPreamble       string       `json:"recall_preamble,omitempty"`
}

// AdvisoryEpisode is returned by POST /v1/advisory-episodes.
type AdvisoryEpisode struct {
	ID           string     `json:"id"`
	SummaryText  string     `json:"summary_text"`
	Source       string     `json:"source"`
	Tags         []string   `json:"tags"`
	Entities     []string   `json:"entities"`
	CreatedAt    *time.Time `json:"created_at,omitempty"`
	Deduplicated bool       `json:"deduplicated,omitempty"`
}

// CandidateEvent is one pending candidate row from curation.
type CandidateEvent struct {
	ID                 string  `json:"id"`
	RawText            string  `json:"raw_text"`
	SalienceScore      float64 `json:"salience_score,omitempty"`
	PromotionStatus    string  `json:"promotion_status"`
	StructuredKind     string  `json:"structured_kind,omitempty"`
	StatementPreview   string  `json:"statement_preview,omitempty"`
	PromotionReadiness string  `json:"promotion_readiness,omitempty"`
	ReadinessReason    string  `json:"readiness_reason,omitempty"`
}

// CandidateReview is the response from GET /v1/curation/candidates/{id}/review.
type CandidateReview struct {
	CandidateID        string `json:"candidate_id"`
	PromotionStatus    string `json:"promotion_status"`
	PromotionReadiness string `json:"promotion_readiness,omitempty"`
	ReadinessReason    string `json:"readiness_reason,omitempty"`
	Explanation        string `json:"explanation,omitempty"`
	SignalStrength     string `json:"signal_strength,omitempty"`
	SignalDetail       string `json:"signal_detail,omitempty"`
}

// MaterializeOutcome is the response from candidate materialization.
type MaterializeOutcome struct {
	Memory                   map[string]any `json:"memory"`
	Created                  bool           `json:"created"`
	Strengthened             bool           `json:"strengthened"`
	ConsolidatedIntoMemoryID *string        `json:"consolidated_into_memory_id,omitempty"`
	ConsolidationReason      string         `json:"consolidation_reason,omitempty"`
	ContradictsMemoryID      *string        `json:"contradicts_memory_id,omitempty"`
}
