package mcp

import (
	"fmt"
	"strings"
)

// CallCoverageCategory classifies how each MCP tool is directly proven via tools/call.
type CallCoverageCategory string

const (
	CoverageUnitToolsCall        CallCoverageCategory = "unit_tools_call_tested"
	CoverageIntegrationToolsCall CallCoverageCategory = "integration_tools_call_tested"
	CoverageDockerProof          CallCoverageCategory = "docker_proof_tested"
	CoverageUnsafeImpossible     CallCoverageCategory = "unsafe_or_impossible_with_justification"
)

const (
	testMemoryUUID    = "11111111-1111-4111-8111-111111111111"
	testMemoryUUID2   = "22222222-2222-4222-8222-222222222222"
	testCandidateUUID = "33333333-3333-4333-8333-333333333333"
	testSessionUUID   = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	testEpisodeSummary = "Unit test MCP tools/call coverage: recorded experience for phase 1 close-out validation."
)

// MinValidToolCallArguments returns minimal valid arguments for direct MCP tools/call tests.
func MinValidToolCallArguments(toolName string) (map[string]any, error) {
	switch toolName {
	case "recall_context", "memory_context_resolve":
		return map[string]any{"task": "unit test MCP tools/call coverage for recall"}, nil
	case "record_experience", "mcp_episode_ingest":
		return map[string]any{"summary": testEpisodeSummary}, nil
	case "memory_log_if_relevant", "auto_log_episode_if_relevant":
		return map[string]any{"text_block": "fixed regression failure in MCP validation path"}, nil
	case "enforcement_evaluate":
		return map[string]any{"proposal_text": "Migrate production database to SQLite without review."}, nil
	case "recall_compile":
		return map[string]any{"retrieval_query": "unit test recall compile"}, nil
	case "recall_get":
		return map[string]any{"retrieval_query": "unit test recall get"}, nil
	case "wakeup_context":
		return map[string]any{}, nil
	case "recall_run_multi":
		return map[string]any{"retrieval_query": "unit test run multi"}, nil
	case "memory_create":
		return map[string]any{"kind": "constraint", "statement": "Unit test memory create via MCP tools/call coverage."}, nil
	case "memory_feedback":
		return map[string]any{
			"memory_id":  testMemoryUUID,
			"event_type": "helpful",
			"reason":     "Unit test memory feedback via MCP coverage.",
		}, nil
	case "memory_promote":
		return map[string]any{"kind": "constraint", "statement": "Unit test memory promote via MCP tools/call coverage."}, nil
	case "curation_digest":
		return map[string]any{"work_summary": "Unit test curation digest work summary for MCP coverage gate."}, nil
	case "curation_pending", "curation_promotion_suggestions", "curation_auto_promote", "health":
		return map[string]any{}, nil
	case "curation_strengthened":
		return map[string]any{"min_support": 2}, nil
	case "curation_review_candidate", "curation_materialize", "curation_promote_candidate", "curation_reject_candidate":
		return map[string]any{"candidate_id": testCandidateUUID}, nil
	case "episode_search_similar":
		return map[string]any{"query": "unit test episode search"}, nil
	case "episode_distill_explicit":
		return map[string]any{"summary": testEpisodeSummary}, nil
	case "memory_recall_advanced":
		return map[string]any{"query": "unit test advanced recall"}, nil
	case "memory_preflight_check", "memory_list_contradictions":
		return map[string]any{}, nil
	case "memory_detect_contradictions":
		return map[string]any{"memory_id": testMemoryUUID, "conflict_with_id": testMemoryUUID2}, nil
	case "evidence_attach":
		return map[string]any{"memory_id": testMemoryUUID, "evidence_text": "Unit test evidence attach via MCP coverage."}, nil
	case "evidence_list":
		return map[string]any{"kind": "note"}, nil
	case "memory_relationships_get":
		return map[string]any{"memory_id": testMemoryUUID}, nil
	case "memory_relationships_create":
		return map[string]any{
			"from_memory_id":    testMemoryUUID,
			"to_memory_id":      testMemoryUUID2,
			"relationship_type": "related_to",
		}, nil
	case "compliance_summary":
		return map[string]any{}, nil
	case "compliance_session_get", "compliance_session_events":
		return map[string]any{"session_id": testSessionUUID}, nil
	case "compliance_evaluate":
		return map[string]any{"session_id": testSessionUUID}, nil
	case "agent_telemetry_start_session":
		return map[string]any{"interface": "mcp"}, nil
	case "agent_telemetry_record_recall":
		return map[string]any{
			"session_id":          testSessionUUID,
			"recalled_memory_ids": []string{"mem_curr"},
			"recall_bundle":       map[string]any{"items": []any{}},
		}, nil
	case "agent_telemetry_record_decision":
		return map[string]any{
			"session_id":      testSessionUUID,
			"recall_event_id": "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
			"decisions":       []any{map[string]any{"memory_id": "mem_curr", "decision": "used", "contract_fields_cited": []string{"scope"}}},
		}, nil
	case "agent_telemetry_record_output":
		return map[string]any{
			"session_id":      testSessionUUID,
			"recall_event_id":   "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
			"output_facts":      []string{"fact"},
			"memory_citations":  []string{"mem_curr"},
		}, nil
	case "agent_telemetry_evaluate":
		return map[string]any{
			"session_id":      testSessionUUID,
			"recall_event_id": "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
			"output_id":       "dddddddd-dddd-4ddd-8ddd-dddddddddddd",
		}, nil
	case "agent_telemetry_get_session":
		return map[string]any{"session_id": testSessionUUID}, nil
	case "agent_telemetry_get_memory":
		return map[string]any{"memory_id": "mem_curr"}, nil
	case "agent_telemetry_get_violations", "agent_telemetry_get_utility_candidates":
		return map[string]any{}, nil
	case "agent_utility_evaluate_candidate", "agent_utility_apply_candidate":
		return map[string]any{"candidate_id": testCandidateUUID}, nil
	case "agent_utility_apply_batch":
		return map[string]any{"candidate_ids": []string{testCandidateUUID}}, nil
	case "agent_utility_revert_application":
		return map[string]any{"rollback_token": "rollback-test-token"}, nil
	case "agent_utility_get_candidate":
		return map[string]any{"candidate_id": testCandidateUUID}, nil
	case "agent_utility_get_memory_history":
		return map[string]any{"memory_id": "test:mem-policy"}, nil
	case "agent_utility_get_applications", "agent_utility_get_policy_summary":
		return map[string]any{}, nil
	default:
		return nil, fmt.Errorf("no min arguments defined for tool %q", toolName)
	}
}

// CallCoverageIssues returns problems when registry tools lack coverage classification.
func CallCoverageIssues() []string {
	var issues []string
	for _, t := range toolRegistry() {
		if strings.TrimSpace(string(t.CallCoverage)) == "" {
			issues = append(issues, t.Name+": missing CallCoverage classification")
		}
		if t.CallCoverage == CoverageUnsafeImpossible && strings.TrimSpace(t.CallCoverageReason) == "" {
			issues = append(issues, t.Name+": unsafe_or_impossible requires CallCoverageReason")
		}
	}
	return issues
}

// CallCoverageLocation returns where direct tools/call coverage lives for a tool.
func CallCoverageLocation(t ToolSpec) string {
	switch t.CallCoverage {
	case CoverageUnitToolsCall:
		return "tool_call_coverage_test.go"
	case CoverageIntegrationToolsCall:
		return "cmd/controlplane/mcp_memory_formation_integration_test.go"
	case CoverageDockerProof:
		return "scripts/proof-mcp-docker.sh"
	default:
		return t.CallCoverageNote
	}
}
