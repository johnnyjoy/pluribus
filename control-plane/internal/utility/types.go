package utility

import "time"

import "github.com/google/uuid"

// Event types for memory utility feedback.
const (
	EventHelpful    = "helpful"
	EventHarmful    = "harmful"
	EventWrong      = "wrong"
	EventOutdated   = "outdated"
	EventIrrelevant = "irrelevant"
	EventConfirmed  = "confirmed"
	EventRefuted    = "refuted"
	EventDuplicateSeen = "duplicate_seen"
)

// ExposedEventTypes are accepted via REST/MCP in Phase 7.
var ExposedEventTypes = map[string]bool{
	EventHelpful:    true,
	EventHarmful:    true,
	EventWrong:      true,
	EventOutdated:   true,
	EventIrrelevant: true,
}

// NegativeEventTypes require a non-empty reason.
var NegativeEventTypes = map[string]bool{
	EventHarmful:  true,
	EventWrong:    true,
	EventOutdated: true,
	EventRefuted:  true,
}

// MinUtilityScore and MaxUtilityScore bound stored utility_score.
const (
	MinUtilityScore = -10.0
	MaxUtilityScore = 10.0
)

// Event is one append-only utility ledger row.
type Event struct {
	ID                uuid.UUID      `json:"id"`
	MemoryID          uuid.UUID      `json:"memory_id"`
	EventType         string         `json:"event_type"`
	EventWeight       float64        `json:"event_weight"`
	Source            string         `json:"source"`
	SourceTool        string         `json:"source_tool,omitempty"`
	SourceSessionID   string         `json:"source_session_id,omitempty"`
	CorrelationID     string         `json:"correlation_id,omitempty"`
	RecallBundleID    string         `json:"recall_bundle_id,omitempty"`
	AgentLoopEventID  *uuid.UUID     `json:"agent_loop_event_id,omitempty"`
	Reason            string         `json:"reason"`
	EvidenceID        *uuid.UUID     `json:"evidence_id,omitempty"`
	Payload           map[string]any `json:"payload,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
}

// Score aggregates utility for one memory.
type Score struct {
	MemoryID        uuid.UUID  `json:"memory_id"`
	UtilityScore    float64    `json:"utility_score"`
	HelpfulCount    int        `json:"helpful_count"`
	HarmfulCount    int        `json:"harmful_count"`
	WrongCount      int        `json:"wrong_count"`
	OutdatedCount   int        `json:"outdated_count"`
	IrrelevantCount int        `json:"irrelevant_count"`
	ConfirmedCount  int        `json:"confirmed_count"`
	RefutedCount    int        `json:"refuted_count"`
	LastPositiveAt  *time.Time `json:"last_positive_at,omitempty"`
	LastNegativeAt  *time.Time `json:"last_negative_at,omitempty"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// FeedbackRequest is POST /v1/memory/{id}/feedback body.
type FeedbackRequest struct {
	EventType       string         `json:"event_type"`
	Reason          string         `json:"reason"`
	Source          string         `json:"source"`
	SourceTool      string         `json:"source_tool"`
	SourceSessionID string         `json:"source_session_id"`
	CorrelationID   string         `json:"correlation_id"`
	RecallBundleID  string         `json:"recall_bundle_id"`
	EvidenceID      *uuid.UUID     `json:"evidence_id"`
	Payload         map[string]any `json:"payload"`
}

// FeedbackResponse is returned after recording feedback.
type FeedbackResponse struct {
	MemoryID        uuid.UUID `json:"memory_id"`
	EventID         uuid.UUID `json:"event_id"`
	EventType       string    `json:"event_type"`
	NewUtilityScore float64   `json:"new_utility_score"`
	Counts          Score     `json:"counts"`
}
