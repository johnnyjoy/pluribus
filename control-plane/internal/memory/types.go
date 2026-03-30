package memory

import (
	"encoding/json"
	"time"

	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// PatternPolarity marks whether pattern evidence is positive/negative.
type PatternPolarity string

const (
	PatternPolarityPositive PatternPolarity = "positive"
	PatternPolarityNegative PatternPolarity = "negative"
)

// PatternImpact describes impact metadata for pattern/failure payloads.
type PatternImpact struct {
	Severity string `json:"severity"` // e.g. low, medium, high
	Surface  string `json:"surface,omitempty"`
	Cost     string `json:"cost,omitempty"`
}

// PatternElevationReason is set on generalization when this row was created by cluster elevation.
const PatternElevationReason = "pattern_elevation"

// PatternGeneralizationMeta records explainable merge/elevation (optional).
type PatternGeneralizationMeta struct {
	Reason                  string   `json:"reason"` // exact_reinforce | near_duplicate_reinforce | elevated_from_cluster | pattern_elevation
	Jaccard                 float64  `json:"jaccard,omitempty"`
	TagOverlapFraction      float64  `json:"tag_overlap_fraction,omitempty"`
	SupportingStatementKeys []string `json:"supporting_statement_keys,omitempty"` // merged-in near-duplicate keys
}

// PatternPayload is optional structured payload for behavior memories.
type PatternPayload struct {
	Polarity   string        `json:"polarity"` // positive | negative
	Experience string        `json:"experience"`
	Decision   string        `json:"decision"`
	Outcome    string        `json:"outcome"`
	Impact     PatternImpact `json:"impact"`
	Directive  string        `json:"directive"`
	Files      []string      `json:"files,omitempty"`
	Symbols    []string      `json:"symbols,omitempty"`
	// SupportingMemoryIDs lists other pattern memory UUIDs that informed this row (optional).
	SupportingMemoryIDs []string `json:"supporting_memory_ids,omitempty"`
	// Generalization is set when this pattern was reinforced or merged from similar rows.
	Generalization *PatternGeneralizationMeta `json:"generalization,omitempty"`
	// SupersededBy is the elevated pattern memory id that dominates this row in recall (optional; not a new kind).
	SupersededBy string `json:"superseded_by,omitempty"`
}

// ValidPatternPolarity returns true if s is a valid PatternPolarity.
func ValidPatternPolarity(s string) bool {
	return s == string(PatternPolarityPositive) || s == string(PatternPolarityNegative)
}

// ValidatePatternPayload checks required fields and valid enum values. Returns nil if valid.
func ValidatePatternPayload(p *PatternPayload) error {
	if p == nil {
		return ErrPatternPayloadRequired
	}
	if p.Polarity == "" {
		return ErrPatternFieldRequired("polarity")
	}
	if !ValidPatternPolarity(p.Polarity) {
		return ErrPatternInvalidPolarity(p.Polarity)
	}
	if p.Experience == "" {
		return ErrPatternFieldRequired("experience")
	}
	if p.Decision == "" {
		return ErrPatternFieldRequired("decision")
	}
	if p.Outcome == "" {
		return ErrPatternFieldRequired("outcome")
	}
	if p.Impact.Severity == "" {
		return ErrPatternFieldRequired("impact.severity")
	}
	if p.Directive == "" {
		return ErrPatternFieldRequired("directive")
	}
	return nil
}

// ErrPatternPayloadRequired is returned when a required pattern payload is missing.
var ErrPatternPayloadRequired = &ValidationError{Field: "payload", Msg: "pattern payload is required"}

// ErrPatternInvalidPolarity is returned for invalid polarity.
func ErrPatternInvalidPolarity(got string) error {
	return &ValidationError{Field: "polarity", Msg: "must be positive or negative, got " + got}
}

// ErrPatternFieldRequired is returned when a required payload field is missing.
func ErrPatternFieldRequired(field string) error {
	return &ValidationError{Field: field, Msg: "required"}
}

// ValidationError is a field-level validation error (Task 97).
type ValidationError struct {
	Field string
	Msg   string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return e.Field + ": " + e.Msg
	}
	return e.Msg
}

// ReinforceMeta carries optional explainability and cross-context / cross-agent salience for authority bumps.
type ReinforceMeta struct {
	Reason     string // e.g. reuse_recall, success_enforcement_proceed, success_runmulti_promote
	ContextKey string // short opaque hash for optional usage salience; empty skips that merge path
	AgentKey   string // opaque hash from AgentUsageKey; empty skips agent salience merge
	// Impact controls authority delta sizing for this reinforcement event.
	// low: passive usage, medium: decision alignment/useful guidance, high: constraint block/pattern success.
	Impact string // high | medium | low (empty = medium)
	// SignalStrength is optional evidence count for noise control (0 = default behavior).
	SignalStrength int
}

// RecallReinforcementConfig bounds recall-driven authority reinforcement (authority salience sprint).
type RecallReinforcementConfig struct {
	// MaxAuthorityDeltaPerCompile is added per memory per reinforce call (cap 1–10). Default 1.
	MaxAuthorityDeltaPerCompile int `yaml:"max_authority_delta_per_compile"`
	// CrossContextEnabled when false disables distinct_contexts merge. Nil = true.
	CrossContextEnabled *bool `yaml:"cross_context_enabled,omitempty"`
	// MaxAgentAuthorityDeltaPerCompile caps extra authority from new distinct agents per call (default 1).
	MaxAgentAuthorityDeltaPerCompile int `yaml:"max_agent_authority_delta_per_compile"`
	// CrossAgentEnabled when false disables distinct_agents merge and agent authority. Nil = true.
	CrossAgentEnabled *bool `yaml:"cross_agent_enabled,omitempty"`
	// MaxDistinctAgentsForAuthority stops agent-based authority bumps once salience.distinct_agents reaches this (default 8).
	MaxDistinctAgentsForAuthority int `yaml:"max_distinct_agents_for_authority"`
	// Impact-tier authority deltas (bounded 0-10). Defaults: high=2 medium=1 low=0.
	ImpactHighDelta   int `yaml:"impact_high_delta"`
	ImpactMediumDelta int `yaml:"impact_medium_delta"`
	ImpactLowDelta    int `yaml:"impact_low_delta"`
	// MinSignalStrengthForLow skips low-impact reinforcement below this value. Default 1.
	MinSignalStrengthForLow int `yaml:"min_signal_strength_for_low"`
}

// MemoryObject is a durable typed memory (state, decision, constraint, failure, pattern).
type MemoryObject struct {
	ID                 uuid.UUID         `json:"id"`
	Kind               api.MemoryKind    `json:"kind"`
	Authority          int               `json:"authority"`
	Applicability      api.Applicability `json:"applicability"`
	Statement          string            `json:"statement"`
	StatementCanonical string            `json:"statement_canonical,omitempty"` // normalized form; see memorynorm
	// StatementKey is SHA-256 hex (memorynorm.StatementKey); populated from DB on search when available (Phase F recall collapse).
	StatementKey string          `json:"-"`
	Status       api.Status      `json:"status"`
	DeprecatedAt *time.Time      `json:"deprecated_at,omitempty"`
	TTLSeconds   *int            `json:"ttl_seconds,omitempty"`
	Tags         []string        `json:"tags,omitempty"`
	Payload      json.RawMessage `json:"payload,omitempty"` // optional structured metadata payload
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	// OccurredAt is when the underlying event or fact took place (optional). Omitted in JSON when unset.
	OccurredAt *time.Time `json:"occurred_at,omitempty"`
}

// CreateRequest is the payload for creating a memory object.
type CreateRequest struct {
	Kind          api.MemoryKind    `json:"kind"`
	Authority     int               `json:"authority"`
	Applicability api.Applicability `json:"applicability,omitempty"`
	Statement     string            `json:"statement"`
	// StatementCanonical is set by Service.Create before persist; not accepted from JSON.
	StatementCanonical string `json:"-"`
	// StatementKey is SHA-256 hex over canonical text (memorynorm.StatementKey); set by Service.Create before persist.
	StatementKey string           `json:"-"`
	Tags         []string         `json:"tags,omitempty"`
	SupersedesID *uuid.UUID       `json:"supersedes_id,omitempty"` // mark the referenced memory as superseded
	TTLSeconds   int              `json:"ttl_seconds,omitempty"`   // 0 = no expiration
	Payload      *json.RawMessage `json:"payload,omitempty"`       // optional structured metadata payload
	// Status optional on create; empty = active; pending when review queue applies.
	Status api.Status `json:"status,omitempty"`
	// Embedding optional dense vector for pgvector (set by Service when semantic retrieval is enabled); not accepted from JSON.
	Embedding []float32 `json:"-"`
	// SkipPatternNearMerge when true skips tryMergeNearDuplicatePattern (internal: pattern elevation create).
	SkipPatternNearMerge bool `json:"-"`
	// OccurredAt is optional event time (when the fact/event occurred), RFC3339 on the wire.
	OccurredAt *time.Time `json:"occurred_at,omitempty"`
}

// SearchRequest is the payload for POST /memory/search.
type SearchRequest struct {
	Tags   []string `json:"tags,omitempty"`   // match objects that have any of these tags
	Status string   `json:"status,omitempty"` // default "active"
	Max    int      `json:"max,omitempty"`    // default 20
	// Kinds when non-empty restricts rows to these memory kinds (e.g. pattern-only scan).
	Kinds []api.MemoryKind `json:"kinds,omitempty"`
}

// ListBindingRequest loads memory rows eligible for pre-change enforcement (binding only).
type ListBindingRequest struct {
	MinAuthority int
	Max          int
	// Kinds filters kinds; if empty, defaults to behavior kinds.
	Kinds []api.MemoryKind
}

// PromoteRequest is the payload for POST /v1/memory/promote (Pluribus Phase A).
type PromoteRequest struct {
	Type       string    `json:"type"`                 // logical memory type (state/decision/constraint/failure/pattern)
	Content    string    `json:"content"`              // promoted content
	Tags       []string  `json:"tags,omitempty"`       // optional tag hints
	Source     string    `json:"source,omitempty"`     // optional source attribution
	Confidence float64   `json:"confidence,omitempty"` // optional confidence score [0,1]
	// EvidenceIDs optional; linked to the created memory after insert (global evidence pool).
	EvidenceIDs []uuid.UUID `json:"evidence_ids,omitempty"`
	// RequireReview when true creates the memory with status pending; omitted/false = active.
	RequireReview bool `json:"require_review,omitempty"`
	// OccurredAt optional event time for the promoted content.
	OccurredAt *time.Time `json:"occurred_at,omitempty"`
}

// PromoteResponse is the response for POST /v1/memory/promote (Pluribus Phase A).
type PromoteResponse struct {
	Promoted bool   `json:"promoted"`
	ID       string `json:"id,omitempty"`
	Reason   string `json:"reason,omitempty"`
	// Status is the persisted memories.status (e.g. pending when require_review).
	Status string `json:"status,omitempty"`
}

// MemoriesCreateRequest is the body for POST /v1/memories.
type MemoriesCreateRequest struct {
	Kind       api.MemoryKind   `json:"kind"`
	Statement  string           `json:"statement"`
	Tags       []string         `json:"tags,omitempty"`
	Authority  int              `json:"authority,omitempty"`
	Payload    *json.RawMessage `json:"payload,omitempty"`
	Status     api.Status       `json:"status,omitempty"`
	OccurredAt *time.Time       `json:"occurred_at,omitempty"`
}

// MemoriesSearchRequest is the body for POST /v1/memories/search (shared pool; filter by tags/text — not a project partition).
type MemoriesSearchRequest struct {
	Query  string   `json:"query,omitempty"`
	Tags   []string `json:"tags,omitempty"`
	Status string   `json:"status,omitempty"`
	Max    int      `json:"max,omitempty"`
}
