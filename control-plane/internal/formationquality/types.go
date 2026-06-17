package formationquality

// SchemaType classifies memory for formation quality rules.
type SchemaType string

const (
	SchemaConstraint         SchemaType = "constraint"
	SchemaDecision           SchemaType = "decision"
	SchemaLesson             SchemaType = "lesson"
	SchemaFailureWarning     SchemaType = "failure_warning"
	SchemaPreference         SchemaType = "preference"
	SchemaFact               SchemaType = "fact"
	SchemaHistoricalEvent    SchemaType = "historical_event"
	SchemaProcedure          SchemaType = "procedure"
	SchemaCurrentGuidance    SchemaType = "current_guidance"
	SchemaSupersededGuidance SchemaType = "superseded_guidance"
	SchemaRefutedGuidance    SchemaType = "refuted_guidance"
)

// Severity of a quality defect.
type Severity string

const (
	SeverityHard Severity = "hard"
	SeveritySoft Severity = "soft"
	SeverityInfo Severity = "info"
)

// Decision is the formation quality outcome.
type Decision string

const (
	DecisionAcceptActive    Decision = "accept_active"
	DecisionAcceptPending   Decision = "accept_pending"
	DecisionRejectGarbage   Decision = "reject_garbage"
	DecisionRejectDangerous Decision = "reject_dangerous"
	DecisionNeedsCuration   Decision = "needs_curation"
)

// DefectCode identifies a specific quality problem.
type DefectCode string

const (
	DefectMissingSchemaType          DefectCode = "missing_schema_type"
	DefectUnknownSchemaType          DefectCode = "unknown_schema_type"
	DefectMissingScope               DefectCode = "missing_scope"
	DefectMissingNegativeScope       DefectCode = "missing_negative_scope"
	DefectMissingRetrievalCues       DefectCode = "missing_retrieval_cues"
	DefectMissingProvenance          DefectCode = "missing_provenance"
	DefectMissingAuthorityBasis      DefectCode = "missing_authority_basis"
	DefectMissingLifecycleRole       DefectCode = "missing_lifecycle_role"
	DefectMissingUseInstruction      DefectCode = "missing_use_instruction"
	DefectMissingTemporalBasis       DefectCode = "missing_temporal_basis"
	DefectOvergeneralizedStatement   DefectCode = "overgeneralized_statement"
	DefectVagueStatement             DefectCode = "vague_statement"
	DefectAmbiguousActor             DefectCode = "ambiguous_actor"
	DefectAmbiguousDomain            DefectCode = "ambiguous_domain"
	DefectConflictingLifecycle       DefectCode = "conflicting_lifecycle"
	DefectHistoricalAsCurrentRisk    DefectCode = "historical_as_current_risk"
	DefectPreferenceAsRuleRisk       DefectCode = "preference_as_rule_risk"
	DefectFailureWithoutCause        DefectCode = "failure_without_cause"
	DefectDecisionWithoutReason      DefectCode = "decision_without_reason"
	DefectProcedureWithoutSteps      DefectCode = "procedure_without_steps"
	DefectConstraintWithoutScope     DefectCode = "constraint_without_scope"
	DefectUnsafeDirectGoverning      DefectCode = "unsafe_direct_governing_memory"
	DefectMisleadingCues             DefectCode = "misleading_cues"
	DefectRefutedAsActive            DefectCode = "refuted_guidance_as_active"
	DefectSupersededAsActive         DefectCode = "superseded_guidance_as_active"
	DefectAgentInferredHighAuthority DefectCode = "agent_inferred_preference_cannot_be_high_authority"
)

// Defect is a quality problem found during evaluation.
type Defect struct {
	Code     DefectCode `json:"code"`
	Severity Severity   `json:"severity"`
	Message  string     `json:"message"`
}

// Warning is a non-blocking quality note.
type Warning struct {
	Code     DefectCode `json:"code"`
	Severity Severity   `json:"severity"`
	Message  string     `json:"message"`
}

// Result is the deterministic quality evaluation output.
type Result struct {
	QualityScore       float64    `json:"quality_score"`
	SchemaType         SchemaType `json:"schema_type"`
	Passed             bool       `json:"passed"`
	Decision           Decision   `json:"decision"`
	Defects            []Defect   `json:"defects"`
	Warnings           []Warning  `json:"warnings"`
	SuggestedStatus    string     `json:"suggested_status"`
	SuggestedApplicability string `json:"suggested_applicability,omitempty"`
	SafeForActiveRecall bool      `json:"safe_for_active_recall"`
}

// Input is the memory candidate evaluated at formation time.
type Input struct {
	Path            string   `json:"path"`
	Kind            string   `json:"kind"`
	Statement       string   `json:"statement"`
	Authority       int      `json:"authority"`
	Applicability   string   `json:"applicability"`
	Status          string   `json:"status"`
	Tags            []string `json:"tags,omitempty"`
	SchemaType      string   `json:"schema_type,omitempty"`
	Scope           string   `json:"scope,omitempty"`
	NegativeScope   []string `json:"negative_scope,omitempty"`
	RetrievalCues   []string `json:"retrieval_cues,omitempty"`
	UseInstruction  string   `json:"use_instruction,omitempty"`
	MisuseWarning   string   `json:"misuse_warning,omitempty"`
	SourceType      string   `json:"source_type,omitempty"`
	AuthorityBasis  string   `json:"authority_basis,omitempty"`
	LifecycleRole   string   `json:"lifecycle_role,omitempty"`
	OccurredAt      string   `json:"occurred_at,omitempty"`
	Reason          string   `json:"reason,omitempty"`
	ProvenanceFields int     `json:"provenance_fields,omitempty"`
}
