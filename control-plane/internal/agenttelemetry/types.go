package agenttelemetry

import (
	"time"

	"github.com/google/uuid"
)

const EvaluatorVersion = "phase11i-v1"

// TelemetrySession is a persisted agent telemetry session.
type TelemetrySession struct {
	ID          uuid.UUID      `json:"session_id"`
	StartedAt   time.Time      `json:"started_at"`
	EndedAt     *time.Time     `json:"ended_at,omitempty"`
	Interface   string         `json:"interface"`
	AgentID     string         `json:"agent_id,omitempty"`
	ClientName  string         `json:"client_name,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// RecallEvent is a persisted recall exposure.
type RecallEvent struct {
	ID                uuid.UUID      `json:"recall_event_id"`
	SessionID         uuid.UUID      `json:"session_id"`
	TaskID            string         `json:"task_id"`
	Interface         string         `json:"interface"`
	RecallRequestJSON map[string]any `json:"recall_request,omitempty"`
	RecallBundleID    string         `json:"recall_bundle_id"`
	RecalledMemoryIDs []string       `json:"recalled_memory_ids"`
	RecallBundleJSON  map[string]any `json:"recall_bundle,omitempty"`
	RecallMode        string         `json:"recall_mode"`
	CreatedAt         time.Time      `json:"created_at"`
}

// MemoryDecisionRow is a persisted memory decision.
type MemoryDecisionRow struct {
	ID                   uuid.UUID `json:"decision_id"`
	RecallEventID        uuid.UUID `json:"recall_event_id"`
	MemoryID             string    `json:"memory_id"`
	Decision             string    `json:"decision"`
	Reason               string    `json:"reason"`
	ContractFieldsCited  []string  `json:"contract_fields_cited"`
	OutputFactsSupported []string  `json:"output_facts_supported"`
	ViolationCodes       []string  `json:"violation_codes,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
}

// OutputEvent is a persisted agent output.
type OutputEvent struct {
	ID              uuid.UUID `json:"output_id"`
	SessionID       uuid.UUID `json:"session_id"`
	TaskID          string    `json:"task_id"`
	RecallEventID   uuid.UUID `json:"recall_event_id,omitempty"`
	OutputFacts     []string  `json:"output_facts"`
	OutputActions   []string  `json:"output_actions"`
	MemoryCitations []string  `json:"memory_citations"`
	CreatedAt       time.Time `json:"created_at"`
}

// ObedienceEvaluationRow is a persisted evaluation.
type ObedienceEvaluationRow struct {
	ID              uuid.UUID `json:"evaluation_id"`
	SessionID       uuid.UUID `json:"session_id"`
	TaskID          string    `json:"task_id"`
	RecallEventID   uuid.UUID `json:"recall_event_id"`
	OutputID        uuid.UUID `json:"output_id,omitempty"`
	ObediencePassed bool      `json:"obedience_passed"`
	ObedienceScore  float64   `json:"obedience_score"`
	Violations      []string  `json:"violations"`
	EvaluatorVersion string   `json:"evaluator_version"`
	CreatedAt       time.Time `json:"created_at"`
}

// ViolationRow is a persisted violation.
type ViolationRow struct {
	ID            uuid.UUID      `json:"violation_id"`
	EvaluationID  uuid.UUID      `json:"evaluation_id"`
	MemoryID      string         `json:"memory_id"`
	ViolationCode string         `json:"violation_code"`
	Severity      string         `json:"severity"`
	Details       map[string]any `json:"details,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}

// UtilityCandidate is an evaluator-validated utility signal (not auto-applied).
type UtilityCandidate struct {
	ID             uuid.UUID `json:"candidate_id"`
	MemoryID       string    `json:"memory_id"`
	EvaluationID   uuid.UUID `json:"evaluation_id"`
	SignalType     string    `json:"signal_type"`
	SignalStrength float64   `json:"signal_strength"`
	SafeToApply    bool      `json:"safe_to_apply"`
	Reason         string    `json:"reason"`
	CreatedAt      time.Time `json:"created_at"`
}

// SessionSummary aggregates persisted telemetry for queries.
type SessionSummary struct {
	Session      TelemetrySession         `json:"session"`
	RecallEvents []RecallEvent            `json:"recall_events"`
	Decisions    []MemoryDecisionRow      `json:"memory_decisions"`
	Outputs      []OutputEvent            `json:"output_events"`
	Evaluations  []ObedienceEvaluationRow `json:"obedience_evaluations"`
	Violations   []ViolationRow           `json:"violations"`
	Candidates   []UtilityCandidate       `json:"utility_candidates"`
}

// MemorySummary is per-memory telemetry aggregates.
type MemorySummary struct {
	MemoryID         string  `json:"memory_id"`
	RecallCount      int     `json:"recall_count"`
	UsedCount        int     `json:"used_count"`
	IgnoredCount     int     `json:"ignored_count"`
	ViolationCount   int     `json:"violation_count"`
	ObediencePassRate float64 `json:"obedience_pass_rate"`
}

// API request types

type StartSessionRequest struct {
	SessionID  string         `json:"session_id,omitempty"`
	Interface  string         `json:"interface"`
	AgentID    string         `json:"agent_id,omitempty"`
	ClientName string         `json:"client_name,omitempty"`
	Tags       []string       `json:"tags,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type RecordRecallRequest struct {
	SessionID         string         `json:"session_id"`
	TaskID            string         `json:"task_id"`
	Interface         string         `json:"interface"`
	RecallRequest     map[string]any `json:"recall_request,omitempty"`
	RecallBundleID    string         `json:"recall_bundle_id"`
	RecalledMemoryIDs []string       `json:"recalled_memory_ids"`
	RecallBundle      map[string]any `json:"recall_bundle"`
	RecallMode        string         `json:"recall_mode,omitempty"`
}

type RecordDecisionRequest struct {
	SessionID     string `json:"session_id"`
	RecallEventID string `json:"recall_event_id"`
	Decisions     []struct {
		MemoryID             string   `json:"memory_id"`
		Decision             string   `json:"decision"`
		Reason               string   `json:"reason"`
		ContractFieldsCited  []string `json:"contract_fields_cited"`
		OutputFactsSupported []string `json:"output_facts_supported"`
	} `json:"decisions"`
}

type RecordOutputRequest struct {
	SessionID       string   `json:"session_id"`
	TaskID          string   `json:"task_id"`
	RecallEventID   string   `json:"recall_event_id"`
	OutputFacts     []string `json:"output_facts"`
	OutputActions   []string `json:"output_actions"`
	MemoryCitations []string `json:"memory_citations"`
}

type EvaluateRequest struct {
	SessionID       string   `json:"session_id"`
	TaskID          string   `json:"task_id"`
	RecallEventID   string   `json:"recall_event_id"`
	OutputID        string   `json:"output_id,omitempty"`
	ExpectedFacts   []string `json:"expected_facts,omitempty"`
	ForbiddenFacts  []string `json:"forbidden_facts,omitempty"`
	TaskTags            []string `json:"task_tags,omitempty"`
	AgentMode           string   `json:"agent_mode,omitempty"`
	ViolationBehaviors  []string `json:"violation_behaviors,omitempty"`
	// Client must not set these to bypass evaluator:
	ObediencePassed *bool `json:"obedience_passed,omitempty"`
}

type EvaluateResponse struct {
	Evaluation      ObedienceEvaluationRow `json:"evaluation"`
	Violations      []ViolationRow         `json:"violations"`
	UtilityCandidates []UtilityCandidate   `json:"utility_candidates"`
	EvaluatorRejectedSelfReport bool       `json:"evaluator_rejected_self_report,omitempty"`
}

// BenchmarkMetrics for Phase 11I proof artifact.
type BenchmarkMetrics struct {
	TelemetrySessionPersistenceRate    float64 `json:"telemetry_session_persistence_rate"`
	RecallEventPersistenceRate         float64 `json:"recall_event_persistence_rate"`
	MemoryDecisionPersistenceRate      float64 `json:"memory_decision_persistence_rate"`
	OutputEventPersistenceRate         float64 `json:"output_event_persistence_rate"`
	ObedienceEvaluationPersistenceRate float64 `json:"obedience_evaluation_persistence_rate"`
	ViolationPersistenceRate           float64 `json:"violation_persistence_rate"`
	UtilityCandidateGenerationRate   float64 `json:"utility_candidate_generation_rate"`
	QuerySessionPassRate               float64 `json:"query_session_pass_rate"`
	QueryMemoryPassRate                float64 `json:"query_memory_pass_rate"`
	QueryViolationPassRate             float64 `json:"query_violation_pass_rate"`
	RESTTelemetryFlowPassRate          float64 `json:"rest_telemetry_flow_pass_rate"`
	MCPTelemetryFlowPassRate           float64 `json:"mcp_telemetry_flow_pass_rate"`
	MCPRESTTelemetryParityRate         float64 `json:"mcp_rest_telemetry_parity_rate"`
	SelfReportOnlyRejectionRate        float64 `json:"self_report_only_rejection_rate"`
	RecallOnlyPositiveUtilityRate      float64 `json:"recall_only_positive_utility_rate"`
	SelfReportUsePositiveUtilityRate   float64 `json:"self_report_use_positive_utility_rate"`
	AutoUtilityMutationRate            float64 `json:"auto_utility_mutation_rate"`
	UnknownSessionRejectionRate        float64 `json:"unknown_session_rejection_rate"`
	MalformedPayloadRejectionRate      float64 `json:"malformed_payload_rejection_rate"`
}

// AutoRecallBenchmarkMetrics for Phase 11J automatic recall telemetry artifact.
type AutoRecallBenchmarkMetrics struct {
	AutoRecallHookCoverageRate           float64 `json:"auto_recall_hook_coverage_rate"`
	AgentFacingRecallSurfaceHookedRate   float64 `json:"agent_facing_recall_surface_hooked_rate"`
	RecallEventIDResponseRate            float64 `json:"recall_event_id_response_rate"`
	RecallBundleIDResponseRate           float64 `json:"recall_bundle_id_response_rate"`
	RecallRequestHashResponseRate        float64 `json:"recall_request_hash_response_rate"`
	RecallEventPersistenceRate           float64 `json:"recall_event_persistence_rate"`
	RecalledMemoryIDsPersistenceRate     float64 `json:"recalled_memory_ids_persistence_rate"`
	EvaluateFromPersistedRecallRate      float64 `json:"evaluate_from_persisted_recall_rate"`
	UnknownRecallEventRejectionRate      float64 `json:"unknown_recall_event_rejection_rate"`
	WrongSessionRecallEventRejectionRate float64 `json:"wrong_session_recall_event_rejection_rate"`
	TamperedRecalledIDsRejectionRate     float64 `json:"tampered_recalled_ids_rejection_rate"`
	TelemetryDisabledBackwardCompatRate  float64 `json:"telemetry_disabled_backward_compat_rate"`
	RecallOnlyPositiveUtilityRate        float64 `json:"recall_only_positive_utility_rate"`
	AutoRecallAutoUtilityMutationRate    float64 `json:"auto_recall_auto_utility_mutation_rate"`
	MCPRESTAutoRecallParityRate          float64 `json:"mcp_rest_auto_recall_parity_rate"`
}

// PostgresProofMetrics for Phase 11J Postgres proof artifact.
type PostgresProofMetrics struct {
	PostgresSchemaMigrationRate             float64 `json:"postgres_schema_migration_rate"`
	PostgresSessionPersistenceRate          float64 `json:"postgres_session_persistence_rate"`
	PostgresRecallEventPersistenceRate      float64 `json:"postgres_recall_event_persistence_rate"`
	PostgresDecisionPersistenceRate         float64 `json:"postgres_decision_persistence_rate"`
	PostgresOutputPersistenceRate           float64 `json:"postgres_output_persistence_rate"`
	PostgresEvaluationPersistenceRate       float64 `json:"postgres_evaluation_persistence_rate"`
	PostgresViolationPersistenceRate        float64 `json:"postgres_violation_persistence_rate"`
	PostgresUtilityCandidatePersistenceRate float64 `json:"postgres_utility_candidate_persistence_rate"`
	PostgresQuerySessionPassRate            float64 `json:"postgres_query_session_pass_rate"`
	PostgresQueryMemoryPassRate             float64 `json:"postgres_query_memory_pass_rate"`
	PostgresQueryViolationPassRate          float64 `json:"postgres_query_violation_pass_rate"`
	PostgresTransactionRollbackPassRate     float64 `json:"postgres_transaction_rollback_pass_rate"`
	PostgresJSONRoundtripPassRate           float64 `json:"postgres_json_roundtrip_pass_rate"`
}
