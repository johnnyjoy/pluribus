package agenttelemetry

import (
	"encoding/json"
	"os"
	"path/filepath"

	"control-plane/internal/agentobedience"
)

// AutoRecallCase is one hostile automatic recall telemetry fixture.
type AutoRecallCase struct {
	CaseID                    string   `json:"case_id"`
	Interface                 string   `json:"interface"`
	Surface                   string   `json:"surface"`
	TelemetryEnabled          bool     `json:"telemetry_enabled"`
	SessionInput              string   `json:"session_input,omitempty"`
	RecallRequest             map[string]any `json:"recall_request,omitempty"`
	ExpectedRecallEventPersisted bool  `json:"expected_recall_event_persisted"`
	ExpectedResponseFields    []string `json:"expected_response_fields,omitempty"`
	ExpectedRecalledMemoryIDs []string `json:"expected_recalled_memory_ids,omitempty"`
	ExpectedEvaluationBehavior string  `json:"expected_evaluation_behavior,omitempty"`
	ExpectedUtilityBehavior   string   `json:"expected_utility_behavior,omitempty"`
	ExpectedPostgresBehavior  string   `json:"expected_postgres_behavior,omitempty"`
	ExpectedMCPRESTParity     bool     `json:"expected_mcp_rest_parity,omitempty"`
	RejectStep                string   `json:"reject_step,omitempty"`
	InputMemories             []agentobedience.CaseMemory `json:"input_memories,omitempty"`
}

type autoRecallCasesFile struct {
	Cases []AutoRecallCase `json:"cases"`
}

func LoadAutoRecallCases(path string) ([]AutoRecallCase, error) {
	if path == "" {
		path = filepath.Join(repoRoot(), "control-plane", "testdata", "automatic_recall_telemetry", "cases.json")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f autoRecallCasesFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, err
	}
	return f.Cases, nil
}
