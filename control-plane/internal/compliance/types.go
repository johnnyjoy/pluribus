package compliance

import (
	"time"

	"github.com/google/uuid"
)

// Status values for compliance evaluation.
const (
	StatusCompliant          = "compliant"
	StatusPartiallyCompliant = "partially_compliant"
	StatusNonCompliant       = "non_compliant"
	StatusUnknown            = "unknown"
	StatusNotApplicable      = "not_applicable"
)

// DefaultRecallMaxAge is how long a recall remains valid before substantive work (documented default).
const DefaultRecallMaxAge = 60 * time.Minute

// Event types recorded at MCP layer.
const (
	EventMCPInitialize       = "mcp_initialize"
	EventMCPToolsList        = "mcp_tools_list"
	EventMCPToolCallStarted  = "mcp_tool_call_started"
	EventMCPToolCallCompleted = "mcp_tool_call_completed"
	EventMCPToolCallFailed   = "mcp_tool_call_failed"
	EventRecallCalled        = "recall_called"
	EventEnforcementCalled   = "enforcement_called"
	EventRecordCalled        = "record_experience_called"
	EventMemoryMutation      = "memory_mutation_called"
	EventCurationCalled      = "curation_called"
	EventDiagnosticCalled    = "diagnostic_called"
	EventMemoryFeedback      = "memory_feedback_called"
)

// Session represents an agent/MCP session row.
type Session struct {
	ID            uuid.UUID      `json:"id"`
	StartedAt     time.Time      `json:"started_at"`
	LastSeenAt    time.Time      `json:"last_seen_at"`
	ClientName    string         `json:"client_name"`
	ClientVersion string         `json:"client_version"`
	Transport     string         `json:"transport"`
	RepoRoot      string         `json:"repo_root,omitempty"`
	WorkspaceHint string         `json:"workspace_hint,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// Event is a loop telemetry row.
type Event struct {
	ID                  uuid.UUID      `json:"id"`
	SessionID           uuid.UUID      `json:"session_id"`
	OccurredAt          time.Time      `json:"occurred_at"`
	EventType           string         `json:"event_type"`
	ToolName            string         `json:"tool_name,omitempty"`
	LoopRole            string         `json:"loop_role,omitempty"`
	RiskLevel           string         `json:"risk_level,omitempty"`
	CorrelationID       string         `json:"correlation_id,omitempty"`
	RequestHash         string         `json:"request_hash,omitempty"`
	RequestSummary      string         `json:"request_summary,omitempty"`
	ResultStatus        string         `json:"result_status,omitempty"`
	ErrorCode           string         `json:"error_code,omitempty"`
	ErrorMessage        string         `json:"error_message,omitempty"`
	DurationMS          int            `json:"duration_ms,omitempty"`
	EnforcementDecision string         `json:"enforcement_decision,omitempty"`
	MemoryID            string         `json:"memory_id,omitempty"`
	Metadata            map[string]any `json:"metadata,omitempty"`
}

// Evaluation is a persisted compliance result.
type Evaluation struct {
	ID           uuid.UUID      `json:"id"`
	SessionID    uuid.UUID      `json:"session_id"`
	EvaluatedAt  time.Time      `json:"evaluated_at"`
	WindowStart  time.Time      `json:"window_start"`
	WindowEnd    time.Time      `json:"window_end"`
	Status       string         `json:"status"`
	MissingSteps []string       `json:"missing_steps"`
	Evidence     map[string]any `json:"evidence"`
	Warnings     []string       `json:"warnings,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// EvaluateRequest is POST /v1/compliance/evaluate body.
type EvaluateRequest struct {
	SessionID      string `json:"session_id"`
	RecallMaxAgeMS *int64 `json:"recall_max_age_ms,omitempty"`
}

// SummaryResponse is GET /v1/compliance/summary.
type SummaryResponse struct {
	TotalSessions   int            `json:"total_sessions"`
	ByStatus        map[string]int `json:"by_status"`
	EvaluatedWindow string         `json:"evaluated_window"`
}
