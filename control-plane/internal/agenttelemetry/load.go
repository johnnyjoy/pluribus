package agenttelemetry

import (
	"encoding/json"
	"os"
	"path/filepath"

	"control-plane/internal/agentobedience"
)

// PersistenceCase is one hostile telemetry persistence fixture.
type PersistenceCase struct {
	CaseID                     string                      `json:"case_id"`
	Interface                  string                      `json:"interface"`
	AgentMode                  string                      `json:"agent_mode"`
	TaskID                     string                      `json:"task_id"`
	TaskTags                   []string                    `json:"task_tags,omitempty"`
	InputMemories              []agentobedience.CaseMemory `json:"input_memories"`
	ViolationBehaviors         []string                    `json:"violation_behaviors,omitempty"`
	ExpectedObediencePassed    bool                        `json:"expected_obedience_passed"`
	ExpectedViolationCodes     []string                    `json:"expected_violation_codes,omitempty"`
	ExpectedOutputFacts        []string                    `json:"expected_output_facts,omitempty"`
	ForbiddenOutputFacts       []string                    `json:"forbidden_output_facts,omitempty"`
	RequiredConstraintMemoryID string                      `json:"required_constraint_memory_id,omitempty"`
	RejectStep                 string                      `json:"reject_step,omitempty"`
	SelfReportObedient         bool                        `json:"self_report_obedient,omitempty"`
	RecallOnly                 bool                        `json:"recall_only,omitempty"`
	SkipEvaluate               bool                        `json:"skip_evaluate,omitempty"`
	ExternalMemoryID           string                      `json:"external_memory_id,omitempty"`
	ExpectedPersistedEvents    []string                    `json:"expected_persisted_events"`
	ExpectedQueries            []string                    `json:"expected_queries"`
	ExpectedMCPRESTParity      bool                        `json:"expected_mcp_rest_parity,omitempty"`
}

type persistenceCasesFile struct {
	Cases []PersistenceCase `json:"cases"`
}

// LoadPersistenceCases loads hostile telemetry persistence fixtures.
func LoadPersistenceCases(path string) ([]PersistenceCase, error) {
	if path == "" {
		path = filepath.Join(repoRoot(), "control-plane", "testdata", "agent_telemetry_persistence", "cases.json")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f persistenceCasesFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, err
	}
	return f.Cases, nil
}

func repoRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "control-plane", "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

// ToObedienceCase converts persistence fixture to obedience case for scripted agent.
func (c PersistenceCase) ToObedienceCase() agentobedience.ObedienceCase {
	return agentobedience.ObedienceCase{
		ID:                            c.CaseID,
		TaskID:                        c.TaskID,
		Interface:                     c.Interface,
		AgentMode:                       c.AgentMode,
		TaskTags:                      c.TaskTags,
		InputMemories:                 c.InputMemories,
		ViolationBehaviors:            c.ViolationBehaviors,
		ExpectedObediencePassed:       c.ExpectedObediencePassed,
		ExpectedViolationCodes:        c.ExpectedViolationCodes,
		ExpectedOutputFacts:           c.ExpectedOutputFacts,
		ForbiddenOutputFacts:          c.ForbiddenOutputFacts,
		RequiredConstraintMemoryID:    c.RequiredConstraintMemoryID,
	}
}
