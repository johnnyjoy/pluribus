package agentobedience

// Violation codes emitted by the obedience evaluator.
const (
	ViolationUsedUnrecalledMemory           = "used_unrecalled_memory"
	ViolationCitedUnrecalledMemory          = "cited_unrecalled_memory"
	ViolationMissingUseDecision             = "missing_use_decision"
	ViolationHistoricalUsedAsCurrent        = "historical_used_as_current"
	ViolationRefutedUsed                    = "refuted_used"
	ViolationSupersededUsed                 = "superseded_used"
	ViolationArchivedUsedAsCurrent          = "archived_used_as_current"
	ViolationWrongScopeUsed                 = "wrong_scope_used"
	ViolationNegativeScopeUsed              = "negative_scope_used"
	ViolationRequiredConstraintIgnored      = "required_constraint_ignored"
	ViolationRequiredFailureWarningIgnored  = "required_failure_warning_ignored"
	ViolationProcedureTriggerMismatch       = "procedure_trigger_mismatch"
	ViolationPreferenceScopeViolation       = "preference_scope_violation"
	ViolationUnsupportedOutputClaim         = "unsupported_output_claim"
	ViolationMemoryHelpWithoutImprovement   = "memory_help_without_improvement"
	ViolationMissingMemoryCitation          = "missing_memory_citation"
	ViolationMissingContractFieldCitation   = "missing_contract_field_citation"
	ViolationMissingTelemetry               = "missing_telemetry"
)

// Violation behaviors for sloppy/broken scripted agents.
const (
	BehaviorHistoricalAsCurrent       = "historical_as_current"
	BehaviorOmitTelemetry             = "omit_telemetry"
	BehaviorIgnoreNegativeScope       = "ignore_negative_scope"
	BehaviorUseWrongScope             = "use_wrong_scope"
	BehaviorIgnoreRequiredConstraint  = "ignore_required_constraint"
	BehaviorUsePreferenceOutsideScope = "use_preference_outside_scope"
	BehaviorUseRefuted                = "use_refuted"
	BehaviorUseSuperseded             = "use_superseded"
	BehaviorCiteUnrecalled            = "cite_unrecalled"
	BehaviorUseUnrecalled             = "use_unrecalled"
	BehaviorInventUnsupportedOutput   = "invent_unsupported_output"
	BehaviorClaimHelpWithoutImprovement = "claim_memory_help_without_improvement"
)
