package utilitypolicy

import (
	"time"

	"github.com/google/uuid"
)

const PolicyVersion = "phase11k-v1"

// Decision values returned by the policy engine.
const (
	DecisionApplyPositive  = "apply_positive"
	DecisionApplyNegative  = "apply_negative"
	DecisionRecordOnly     = "record_only"
	DecisionReviewRequired = "review_required"
	DecisionReject         = "reject"
)

// CandidateClass is deterministic classification before policy gates.
const (
	ClassPositiveApplyable  = "positive_applyable"
	ClassNegativeApplyable  = "negative_applyable"
	ClassNeutralRecordOnly  = "neutral_record_only"
	ClassReviewRequired     = "review_required"
	ClassRejected           = "rejected"
)

// PolicyConfig bounds guarded utility application.
type PolicyConfig struct {
	MaxPositiveDelta               float64       `json:"max_positive_delta"`
	MaxNegativeDelta               float64       `json:"max_negative_delta"`
	MaxPositivePerSessionPerMemory int           `json:"max_positive_per_session_per_memory"`
	MaxPositivePerAgentPerMemory   int           `json:"max_positive_per_agent_per_memory"`
	StaleCandidateWindow           time.Duration `json:"stale_candidate_window"`
	AllowHistoricalPositive        bool          `json:"allow_historical_positive"`
}

// DefaultPolicyConfig returns conservative Phase 11K defaults.
func DefaultPolicyConfig() PolicyConfig {
	return PolicyConfig{
		MaxPositiveDelta:               0.5,
		MaxNegativeDelta:               1.0,
		MaxPositivePerSessionPerMemory: 2,
		MaxPositivePerAgentPerMemory:   3,
		StaleCandidateWindow:           7 * 24 * time.Hour,
		AllowHistoricalPositive:        false,
	}
}

// CandidateInput is the policy-facing view of a utility candidate.
type CandidateInput struct {
	CandidateID    uuid.UUID `json:"candidate_id"`
	MemoryID       string    `json:"memory_id"`
	EvaluationID   uuid.UUID `json:"evaluation_id"`
	SignalType     string    `json:"signal_type"`
	SignalStrength float64   `json:"signal_strength"`
	SafeToApply    bool      `json:"safe_to_apply"`
	Reason         string    `json:"reason"`
	ViolationCodes []string  `json:"violation_codes,omitempty"`
	AgentID        string    `json:"agent_id,omitempty"`
	ClientID       string    `json:"client_id,omitempty"`
	SessionID      uuid.UUID `json:"session_id,omitempty"`
	TaskID         string    `json:"task_id,omitempty"`
	EvaluatorPassed bool     `json:"evaluator_passed"`
	CreatedAt      time.Time `json:"created_at"`
}

// PolicyDecision is a deterministic policy outcome (evaluate or apply).
type PolicyDecision struct {
	CandidateID   string   `json:"candidate_id"`
	MemoryID      string   `json:"memory_id"`
	Decision      string   `json:"decision"`
	Delta         float64  `json:"delta"`
	Reason        string   `json:"reason"`
	PolicyVersion string   `json:"policy_version"`
	Evidence      []string `json:"evidence"`
	RollbackToken string   `json:"rollback_token,omitempty"`
	Classification string  `json:"classification,omitempty"`
}

// ApplicationRecord is a persisted policy application audit row.
type ApplicationRecord struct {
	ApplicationID        uuid.UUID  `json:"application_id"`
	CandidateID          uuid.UUID  `json:"candidate_id"`
	MemoryID             string     `json:"memory_id"`
	EvaluationID         uuid.UUID  `json:"evaluation_id"`
	Decision             string     `json:"decision"`
	Delta                float64    `json:"delta"`
	PreviousUtilityScore float64    `json:"previous_utility_score"`
	NewUtilityScore      float64    `json:"new_utility_score"`
	PolicyVersion        string     `json:"policy_version"`
	Reason               string     `json:"reason"`
	Evidence             []string   `json:"evidence"`
	RollbackToken        string     `json:"rollback_token"`
	AppliedBy            string     `json:"applied_by"`
	SessionID            uuid.UUID  `json:"session_id,omitempty"`
	AgentID              string     `json:"agent_id,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	RevertedAt           *time.Time `json:"reverted_at,omitempty"`
	RevertReason         string     `json:"revert_reason,omitempty"`
}

// EvaluateCandidateRequest is POST /v1/agent/utility/policy/evaluate-candidate.
type EvaluateCandidateRequest struct {
	CandidateID uuid.UUID `json:"candidate_id"`
}

// ApplyCandidateRequest is POST /v1/agent/utility/policy/apply-candidate.
type ApplyCandidateRequest struct {
	CandidateID uuid.UUID `json:"candidate_id"`
	AppliedBy   string    `json:"applied_by,omitempty"`
}

// ApplyBatchRequest applies multiple candidates with shared caps.
type ApplyBatchRequest struct {
	CandidateIDs []uuid.UUID `json:"candidate_ids"`
	AppliedBy    string      `json:"applied_by,omitempty"`
}

// RevertApplicationRequest rolls back one application.
type RevertApplicationRequest struct {
	RollbackToken string `json:"rollback_token"`
	RevertReason  string `json:"revert_reason,omitempty"`
	AppliedBy     string `json:"applied_by,omitempty"`
}

// PolicySummary aggregates application outcomes.
type PolicySummary struct {
	TotalApplications   int            `json:"total_applications"`
	PositiveApplications int           `json:"positive_applications"`
	NegativeApplications int           `json:"negative_applications"`
	RecordOnlyCount     int            `json:"record_only_count"`
	ReviewRequiredCount int            `json:"review_required_count"`
	RejectedCount       int            `json:"rejected_count"`
	RevertedCount       int            `json:"reverted_count"`
	PolicyVersion       string         `json:"policy_version"`
}

// BenchmarkMetrics are emitted to artifacts/guarded-utility-policy-benchmark.json.
type BenchmarkMetrics struct {
	PolicyDecisionPassRate              float64 `json:"policy_decision_pass_rate"`
	PositiveApplyPrecisionRate          float64 `json:"positive_apply_precision_rate"`
	NegativeApplyPrecisionRate          float64 `json:"negative_apply_precision_rate"`
	RecordOnlyNoMutationRate            float64 `json:"record_only_no_mutation_rate"`
	ReviewRequiredNoMutationRate        float64 `json:"review_required_no_mutation_rate"`
	RejectedNoMutationRate              float64 `json:"rejected_no_mutation_rate"`
	RecallOnlyPositiveApplyRate         float64 `json:"recall_only_positive_apply_rate"`
	SelfReportOnlyPositiveApplyRate     float64 `json:"self_report_only_positive_apply_rate"`
	HistoricalOnlyPositiveApplyRate     float64 `json:"historical_only_positive_apply_rate"`
	UnsupportedOutputPositiveApplyRate  float64 `json:"unsupported_output_positive_apply_rate"`
	WrongScopePositiveApplyRate         float64 `json:"wrong_scope_positive_apply_rate"`
	RefutedPositiveApplyRate            float64 `json:"refuted_positive_apply_rate"`
	SupersededPositiveApplyRate         float64 `json:"superseded_positive_apply_rate"`
	DuplicateApplyRejectionRate         float64 `json:"duplicate_apply_rejection_rate"`
	SessionCapEnforcementRate           float64 `json:"session_cap_enforcement_rate"`
	AgentCapEnforcementRate             float64 `json:"agent_cap_enforcement_rate"`
	ScoreBoundsPreservationRate         float64 `json:"score_bounds_preservation_rate"`
	RollbackSuccessRate                 float64 `json:"rollback_success_rate"`
	AuditRecordRate                     float64 `json:"audit_record_rate"`
	MCPRESTPolicyParityRate             float64 `json:"mcp_rest_policy_parity_rate"`
	PostgresPolicyPersistenceRate       float64 `json:"postgres_policy_persistence_rate"`
}

// PostgresProofMetrics for artifacts/guarded-utility-policy-postgres-proof.json.
type PostgresProofMetrics struct {
	PostgresPolicySchemaMigrationRate     float64 `json:"postgres_policy_schema_migration_rate"`
	PostgresCandidateQueryRate            float64 `json:"postgres_candidate_query_rate"`
	PostgresApplicationPersistenceRate    float64 `json:"postgres_application_persistence_rate"`
	PostgresUtilityEventPersistenceRate   float64 `json:"postgres_utility_event_persistence_rate"`
	PostgresDuplicateApplyRejectionRate   float64 `json:"postgres_duplicate_apply_rejection_rate"`
	PostgresRollbackRate                  float64 `json:"postgres_rollback_rate"`
	PostgresTransactionAtomicityRate      float64 `json:"postgres_transaction_atomicity_rate"`
	PostgresJSONEvidenceRoundtripRate     float64 `json:"postgres_json_evidence_roundtrip_rate"`
}
