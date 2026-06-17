package agentcontract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type MissingFieldCase struct {
	ID                   string               `json:"id"`
	Interface            string               `json:"interface"`
	RecallMode           string               `json:"recall_mode"`
	InputMemories        []ContractCaseMemory `json:"input_memories"`
	ExpectedDefectCodes  []string             `json:"expected_defect_codes"`
	ExpectedContractPassed bool               `json:"expected_contract_passed"`
	FlattenedMCP         bool                 `json:"flattened_mcp"`
}

type MissingFieldCasesFile struct {
	Cases []MissingFieldCase `json:"cases"`
}

func loadMissingFieldCases(t *testing.T) []MissingFieldCase {
	t.Helper()
	path := filepath.Join(repoRoot(), "control-plane", "testdata", "agent_memory_contract_missing_fields", "cases.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read missing field cases: %v", err)
	}
	var f MissingFieldCasesFile
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatalf("parse missing field cases: %v", err)
	}
	return f.Cases
}

func containsAll(haystack []string, needles []string) bool {
	set := map[string]struct{}{}
	for _, h := range haystack {
		set[h] = struct{}{}
	}
	for _, n := range needles {
		if n == "" {
			continue
		}
		if _, ok := set[n]; !ok {
			return false
		}
	}
	return true
}

func allDefects(ev ContractEvaluation) []string {
	out := append([]string{}, ev.MissingRequiredFields...)
	out = append(out, ev.UnsafeOmissions...)
	return out
}

func TestMissingFieldHostileCases(t *testing.T) {
	cases := loadMissingFieldCases(t)
	if len(cases) < 24 {
		t.Fatalf("need >=24 missing-field cases, got %d", len(cases))
	}
	for _, c := range cases {
		t.Run(c.ID, func(t *testing.T) {
			bundle := buildBundleFromCase(ContractCase{
				Interface:     c.Interface,
				RecallMode:    c.RecallMode,
				InputMemories: c.InputMemories,
			})
			mode := parseRecallMode(c.RecallMode)
			ev := EvaluateBundleContract(bundle, mode, c.FlattenedMCP)
			if ev.ContractPassed != c.ExpectedContractPassed {
				t.Fatalf("contract_passed=%v want %v defects=%v", ev.ContractPassed, c.ExpectedContractPassed, allDefects(ev))
			}
			if len(c.ExpectedDefectCodes) > 0 && !containsAll(allDefects(ev), c.ExpectedDefectCodes) {
				t.Fatalf("defects %v missing expected %v", allDefects(ev), c.ExpectedDefectCodes)
			}
		})
	}
}

func TestMissingFieldDefectCoverageRate(t *testing.T) {
	cases := loadMissingFieldCases(t)
	failCases := 0
	caught := 0
	for _, c := range cases {
		if c.ExpectedContractPassed {
			continue
		}
		failCases++
		bundle := buildBundleFromCase(ContractCase{
			Interface:     c.Interface,
			RecallMode:    c.RecallMode,
			InputMemories: c.InputMemories,
		})
		ev := EvaluateBundleContract(bundle, parseRecallMode(c.RecallMode), c.FlattenedMCP)
		if !ev.ContractPassed {
			caught++
		}
	}
	if failCases == 0 {
		t.Fatal("no failing cases")
	}
	rate := float64(caught) / float64(failCases)
	if rate < 0.95 {
		t.Fatalf("missing_field_defect_coverage_rate=%.3f want >=0.95", rate)
	}
}
