package compliance

import "time"

// Tool loop classification (mirrors mcp/tool_registry.go roles; kept here to avoid import cycles).

var recallTools = map[string]bool{
	"recall_context": true, "memory_context_resolve": true, "wakeup_context": true,
}

var recordTools = map[string]bool{
	"record_experience": true, "mcp_episode_ingest": true,
	"memory_log_if_relevant": true, "auto_log_episode_if_relevant": true,
}

var enforcementTools = map[string]bool{
	"enforcement_evaluate": true,
}

var diagnosticTools = map[string]bool{
	"health": true, "compliance_summary": true, "compliance_session_get": true,
	"compliance_session_events": true, "compliance_evaluate": true,
	"memory_preflight_check": true, "memory_list_contradictions": true,
	"episode_search_similar": true, "evidence_list": true,
	"memory_relationships_get": true, "curation_pending": true,
	"curation_promotion_suggestions": true, "curation_strengthened": true,
	"curation_review_candidate": true,
}

var readOnlyRecallTools = map[string]bool{
	"recall_compile": true, "recall_get": true, "memory_recall_advanced": true,
}

var materialMutators = map[string]bool{
	"record_experience": true, "mcp_episode_ingest": true,
	"memory_log_if_relevant": true, "auto_log_episode_if_relevant": true,
	"memory_create": true, "memory_promote": true,
	"curation_digest": true, "curation_materialize": true, "curation_promote_candidate": true,
	"curation_reject_candidate": true, "curation_auto_promote": true,
	"episode_distill_explicit": true, "memory_detect_contradictions": true,
	"evidence_attach": true, "memory_relationships_create": true,
}

var highRiskMutators = map[string]bool{
	"memory_create": true, "memory_promote": true,
	"curation_materialize": true, "curation_promote_candidate": true,
	"curation_auto_promote": true,
}

// ClassifyTool returns loop-relevant event type for a successful tool call.
func ClassifyTool(toolName string) string {
	if recallTools[toolName] {
		return EventRecallCalled
	}
	if enforcementTools[toolName] {
		return EventEnforcementCalled
	}
	if recordTools[toolName] {
		return EventRecordCalled
	}
	if toolName == "curation_digest" || toolName == "curation_materialize" || toolName == "curation_promote_candidate" ||
		toolName == "curation_reject_candidate" || toolName == "curation_auto_promote" || toolName == "episode_distill_explicit" {
		return EventCurationCalled
	}
	if materialMutators[toolName] {
		return EventMemoryMutation
	}
	if diagnosticTools[toolName] || readOnlyRecallTools[toolName] {
		return EventDiagnosticCalled
	}
	return EventMCPToolCallCompleted
}

// IsMaterialChange returns true when tool mutates state or is high/critical risk.
func IsMaterialChange(toolName string) bool {
	return materialMutators[toolName] || highRiskMutators[toolName]
}

// IsSubstantiveActivity returns true when MCP-visible work goes beyond diagnostics.
func IsSubstantiveActivity(toolName string) bool {
	if diagnosticTools[toolName] || readOnlyRecallTools[toolName] {
		return false
	}
	if toolName == "health" {
		return false
	}
	return recallTools[toolName] || recordTools[toolName] || enforcementTools[toolName] ||
		materialMutators[toolName] || highRiskMutators[toolName] ||
		toolName == "recall_run_multi" || toolName == "memory_detect_contradictions"
}

// IsRecallTool returns true for recall/context tools that satisfy pre-work recall.
func IsRecallTool(toolName string) bool {
	return recallTools[toolName]
}

// IsRecordTool returns true for post-outcome recording tools.
func IsRecordTool(toolName string) bool {
	return recordTools[toolName]
}

// IsEnforcementTool returns true for enforcement_evaluate.
func IsEnforcementTool(toolName string) bool {
	return enforcementTools[toolName]
}

// IsDiagnosticOnlySession returns true when all loop tool activity is diagnostic/read-only.
func IsDiagnosticOnlySession(events []Event) bool {
	hasActivity := false
	for _, e := range events {
		if !isSuccessfulLoopEvent(e) {
			continue
		}
		tool := e.ToolName
		if tool == "" || tool == "health" {
			continue
		}
		hasActivity = true
		if IsSubstantiveActivity(tool) {
			return false
		}
	}
	return hasActivity
}

func lastRecallTime(events []Event) (zero bool, t time.Time) {
	for _, e := range events {
		if e.EventType == EventRecallCalled || (e.EventType == EventMCPToolCallCompleted && IsRecallTool(e.ToolName)) {
			if t.IsZero() || e.OccurredAt.After(t) {
				t = e.OccurredAt
			}
		}
	}
	return t.IsZero(), t
}

func firstMaterialChangeAfter(events []Event, after time.Time) (bool, time.Time) {
	for _, e := range events {
		if e.EventType != EventMCPToolCallCompleted {
			continue
		}
		if !IsMaterialChange(e.ToolName) {
			continue
		}
		if !after.IsZero() && e.OccurredAt.Before(after) {
			continue
		}
		return true, e.OccurredAt
	}
	return false, time.Time{}
}

// LoopRoleForTool returns registry-equivalent loop role string for telemetry.
func LoopRoleForTool(toolName string) string {
	switch {
	case recallTools[toolName]:
		return "session_start"
	case enforcementTools[toolName]:
		return "pre_change"
	case recordTools[toolName]:
		return "post_outcome"
	case diagnosticTools[toolName] || readOnlyRecallTools[toolName]:
		return "diagnostic"
	default:
		if materialMutators[toolName] || highRiskMutators[toolName] {
			return "admin"
		}
		return "none"
	}
}

// RiskForTool returns coarse risk label for telemetry.
func RiskForTool(toolName string) string {
	if highRiskMutators[toolName] {
		return "high"
	}
	if materialMutators[toolName] {
		return "medium"
	}
	return "low"
}
