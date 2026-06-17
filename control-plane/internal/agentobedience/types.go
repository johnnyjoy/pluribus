package agentobedience

// Agent modes for scripted harness (not LLMs).
const (
	AgentObedient = "obedient_agent"
	AgentSloppy   = "sloppy_agent"
	AgentBroken   = "malicious_or_broken_agent"
)

const (
	InterfaceREST = "rest"
	InterfaceMCP  = "mcp"
)

// Obedience states for per-memory results.
const (
	StateUsedCorrectly           = "used_correctly"
	StateIgnoredCorrectly        = "ignored_correctly"
	StateHistoricalOnlyCorrectly = "historical_only_correctly"
	StateMisused                 = "misused"
	StateUnsafeUse               = "unsafe_use"
	StateUnsupportedClaim        = "unsupported_claim"
	StateUncitedMemoryUse        = "uncited_memory_use"
	StateMissingTelemetry        = "missing_telemetry"
)

// MemoryDecision is one memory's reported use decision in telemetry.
type MemoryDecision struct {
	MemoryID            string   `json:"memory_id"`
	Decision            string   `json:"decision"` // used|ignored|historical_only|misused|unsafe
	Reason              string   `json:"reason"`
	ContractFieldsCited []string `json:"contract_fields_cited"`
	OutputFactsSupported []string `json:"output_facts_supported"`
	ViolationCodes      []string `json:"violation_codes"`
}

// FinalOutput is the agent's machine-checkable output.
type FinalOutput struct {
	Facts            []string `json:"facts"`
	Actions          []string `json:"actions"`
	MemoryCitations  []string `json:"memory_citations"`
}

// RubricResult is deterministic expected-fact checking.
type RubricResult struct {
	Passed               bool     `json:"passed"`
	ExpectedFactsPresent []string `json:"expected_facts_present"`
	ForbiddenFactsAbsent []string `json:"forbidden_facts_absent"`
	Violations           []string `json:"violations"`
}

// MemoryUseTelemetry is the auditable memory-use record for one agent run.
type MemoryUseTelemetry struct {
	RunID                  string           `json:"run_id"`
	SessionID              string           `json:"session_id"`
	TaskID                 string           `json:"task_id"`
	Interface              string           `json:"interface"`
	AgentKind              string           `json:"agent_kind"`
	RecallRequestID        string           `json:"recall_request_id"`
	RecallBundleID         string           `json:"recall_bundle_id"`
	OutputID               string           `json:"output_id"`
	RecalledMemoryIDs      []string         `json:"recalled_memory_ids"`
	UsedMemoryIDs          []string         `json:"used_memory_ids"`
	IgnoredMemoryIDs       []string         `json:"ignored_memory_ids"`
	HistoricalOnlyMemoryIDs []string        `json:"historical_only_memory_ids"`
	MisusedMemoryIDs       []string         `json:"misused_memory_ids"`
	UnsafeMemoryIDs        []string         `json:"unsafe_memory_ids"`
	MemoryDecisions        []MemoryDecision `json:"memory_decisions"`
	FinalOutput            FinalOutput      `json:"final_output"`
	RubricResult           RubricResult     `json:"rubric_result"`
	TelemetryComplete      bool             `json:"telemetry_complete"`
}

// MemoryResult is per-memory obedience evaluation.
type MemoryResult struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

// ObedienceEvaluation is evaluator output.
type ObedienceEvaluation struct {
	ObediencePassed bool                    `json:"obedience_passed"`
	ObedienceScore  float64                 `json:"obedience_score"`
	Violations      []string                `json:"violations"`
	MemoryResults   map[string]MemoryResult `json:"memory_results"`
}

// CaseMemory is input memory for obedience fixtures.
type CaseMemory struct {
	MemoryID       string            `json:"memory_id"`
	Statement      string            `json:"statement"`
	SchemaType     string            `json:"schema_type"`
	LifecycleRole  string            `json:"lifecycle_role"`
	Status         string            `json:"status"`
	Scope          string            `json:"scope"`
	NegativeScope  []string          `json:"negative_scope"`
	UseInstruction string            `json:"use_instruction"`
	MisuseWarning  string            `json:"misuse_warning"`
	SourceType     string            `json:"source_type"`
	AuthorityBasis string            `json:"authority_basis"`
	UtilityScore   *float64          `json:"utility_score"`
	QualityState   string            `json:"quality_state"`
	QualityScore   *float64          `json:"quality_score"`
	SupersededBy   string            `json:"superseded_by"`
	OutputFacts    []string          `json:"output_facts,omitempty"`
}

// ObedienceCase is one hostile obedience fixture.
type ObedienceCase struct {
	ID                            string       `json:"id"`
	TaskID                        string       `json:"task_id"`
	Interface                     string       `json:"interface"`
	AgentMode                     string       `json:"agent_mode"`
	TaskTags                      []string     `json:"task_tags"`
	InputMemories                 []CaseMemory `json:"input_memories"`
	ViolationBehaviors            []string     `json:"violation_behaviors,omitempty"`
	ExpectedRecalledMemoryIDs     []string     `json:"expected_recalled_memory_ids"`
	ExpectedUsedMemoryIDs         []string     `json:"expected_used_memory_ids,omitempty"`
	ExpectedIgnoredMemoryIDs      []string     `json:"expected_ignored_memory_ids,omitempty"`
	ExpectedHistoricalOnlyMemoryIDs []string   `json:"expected_historical_only_memory_ids,omitempty"`
	ExpectedViolationCodes        []string     `json:"expected_violation_codes,omitempty"`
	ExpectedObediencePassed       bool         `json:"expected_obedience_passed"`
	ExpectedOutputFacts           []string     `json:"expected_output_facts,omitempty"`
	ForbiddenOutputFacts          []string     `json:"forbidden_output_facts,omitempty"`
	ExpectedTelemetryFields       []string     `json:"expected_telemetry_fields,omitempty"`
	ExpectedContractFieldCitations []string    `json:"expected_contract_field_citations,omitempty"`
	RequiredConstraintMemoryID    string       `json:"required_constraint_memory_id,omitempty"`
}

// CasesFile is the fixture file wrapper.
type CasesFile struct {
	Cases []ObedienceCase `json:"cases"`
}

// BenchmarkMetrics aggregates Phase 11H proof metrics.
type BenchmarkMetrics struct {
	ObediencePassRate                 float64 `json:"obedience_pass_rate"`
	ObedientAgentPassRate             float64 `json:"obedient_agent_pass_rate"`
	SloppyAgentDetectionRate          float64 `json:"sloppy_agent_detection_rate"`
	BrokenAgentDetectionRate          float64 `json:"broken_agent_detection_rate"`
	MemoryUseTelemetryCompleteRate    float64 `json:"memory_use_telemetry_complete_rate"`
	UsedMemoryCitationRate            float64 `json:"used_memory_citation_rate"`
	ContractFieldCitationRate         float64 `json:"contract_field_citation_rate"`
	UnsupportedOutputClaimRate        float64 `json:"unsupported_output_claim_rate"`
	HistoricalAsCurrentViolationRate  float64 `json:"historical_as_current_violation_rate"`
	RefutedUseViolationRate           float64 `json:"refuted_use_violation_rate"`
	SupersededUseViolationRate        float64 `json:"superseded_use_violation_rate"`
	WrongScopeUseRate                 float64 `json:"wrong_scope_use_rate"`
	NegativeScopeViolationRate        float64 `json:"negative_scope_violation_rate"`
	UnrecalledMemoryUseRate           float64 `json:"unrecalled_memory_use_rate"`
	UnrecalledMemoryCitationRate      float64 `json:"unrecalled_memory_citation_rate"`
	RequiredConstraintIgnoreRate      float64 `json:"required_constraint_ignore_rate"`
	MCPRESTObedienceParityRate        float64 `json:"mcp_rest_obedience_parity_rate"`
}
