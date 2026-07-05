package proofscenarios

import (
	"strings"
	"testing"
)

func TestCollectHonestyReport_includesSuiteAndScenarioLimits(t *testing.T) {
	scenarios := []Scenario{{
		ID:           "simulated-multi-agent-continuity",
		BenefitClaim: "Shared pool recall parity",
		DoesNotProve: []string{"MCP agent loop"},
	}}
	rows := []ResultRow{{ScenarioID: "simulated-multi-agent-continuity", Pass: true}}
	report := CollectHonestyReport(scenarios, rows)
	if !strings.Contains(report, "Honesty") {
		t.Fatal("missing honesty header")
	}
	if !strings.Contains(report, "HTTP REST clients only") {
		t.Fatal("missing suite note")
	}
	if !strings.Contains(report, "Does **not** prove: MCP agent loop") {
		t.Fatal("missing scenario does_not_prove")
	}
}
