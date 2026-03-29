// Package enforcement evaluates proposals against binding memory.
//
// WARNING: This system is memory-first.
// Do not introduce project/task/workspace/hive/scope concepts as memory partitions or required inputs.
package enforcement

import (
	"github.com/google/uuid"
)

// EnforcementDecision is the gate outcome for a proposal.
type EnforcementDecision string

const (
	DecisionAllow             EnforcementDecision = "allow"
	DecisionRequireReview     EnforcementDecision = "require_review"
	DecisionBlock             EnforcementDecision = "block"
	DecisionBlockOverrideable EnforcementDecision = "block_overrideable"
)

// EvaluateRequest is the body for POST /v1/enforcement/evaluate.
type EvaluateRequest struct {
	ProposalText string    `json:"proposal_text"`
	// Intent optional hint: change | plan | tooling | datastore | other
	Intent string `json:"intent,omitempty"`
	Tags   []string `json:"tags,omitempty"`
	// Rationale optional free text from the proposer.
	Rationale string `json:"rationale,omitempty"`
	// Goal is optional but recommended. Validation checks use it to assess directionality.
	Goal string `json:"goal,omitempty"`
	// AgentID optional opaque client identifier for salience / reinforcement on success paths only (not used for enforcement matching).
	AgentID string `json:"agent_id,omitempty"`
}

// EvaluateResponse is the structured enforcement result.
type EvaluateResponse struct {
	Decision          EnforcementDecision `json:"decision"`
	Explanation       string              `json:"explanation"`
	TriggeredMemories []TriggeredMemory   `json:"triggered_memories"`
	Validation        ValidationSummary   `json:"validation"`
	// EvaluationEngine is a stable identifier for the matcher (not an LLM).
	EvaluationEngine string `json:"evaluation_engine"`
	// EvaluationNote states that only shipped rules fire; absence of hits is not a safety certificate.
	EvaluationNote string `json:"evaluation_note"`
	RemediationHints  []string            `json:"remediation_hints,omitempty"`
	Override          *OverrideHint       `json:"override,omitempty"`
}

// ValidationSummary captures loop checks for Recall -> Act -> Validate -> Update -> Repeat.
type ValidationSummary struct {
	ViolatedConstraints bool   `json:"violated_constraints"`
	RepeatedFailures    bool   `json:"repeated_known_failures"`
	ContradictedDecisions bool `json:"contradicted_decisions"`
	MovesTowardGoal     bool   `json:"moves_toward_goal"`
	Passed              bool   `json:"passed"`
	NextAction          string `json:"next_action"` // proceed | revise | reject
}

// TriggeredMemory explains one binding memory that fired.
type TriggeredMemory struct {
	MemoryID         uuid.UUID `json:"memory_id"`
	Kind             string    `json:"kind"`
	Authority        int       `json:"authority"`
	StatementSnippet string    `json:"statement_snippet"`
	ReasonCode       string    `json:"reason_code"`
	Detail           string    `json:"detail"`
	Evidence         []EvidenceRef `json:"evidence,omitempty"`
}

// EvidenceRef is a compact evidence pointer for receipts.
type EvidenceRef struct {
	ID   uuid.UUID `json:"id"`
	Kind string    `json:"kind,omitempty"`
	Path string    `json:"path,omitempty"`
}

// OverrideHint describes explicit override expectations (no silent bypass).
type OverrideHint struct {
	Required bool   `json:"required"`
	Summary  string `json:"summary"`
}
