package agentobedience

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAgentContractObedienceBenchmarkArtifact(t *testing.T) {
	if os.Getenv("AGENT_CONTRACT_OBEDIENCE_BENCHMARK") != "1" {
		t.Skip("set AGENT_CONTRACT_OBEDIENCE_BENCHMARK=1")
	}
	cases, err := LoadCases("")
	if err != nil {
		t.Fatal(err)
	}
	m := computeBenchmarkMetrics(cases)
	path := filepath.Join(repoRoot(), "artifacts", "agent-contract-obedience-benchmark.json")
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestProofAgentContractObedienceHardThresholds(t *testing.T) {
	if os.Getenv("PROOF_AGENT_CONTRACT_OBEDIENCE") != "1" {
		t.Skip("set PROOF_AGENT_CONTRACT_OBEDIENCE=1")
	}
	os.Setenv("AGENT_CONTRACT_OBEDIENCE_BENCHMARK", "1")
	TestAgentContractObedienceBenchmarkArtifact(t)

	raw, err := os.ReadFile(filepath.Join(repoRoot(), "artifacts", "agent-contract-obedience-benchmark.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m BenchmarkMetrics
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}

	if m.ObedientAgentPassRate != 1.0 {
		t.Fatalf("obedient_agent_pass_rate=%.3f want 1.0", m.ObedientAgentPassRate)
	}
	if m.SloppyAgentDetectionRate != 1.0 {
		t.Fatalf("sloppy_agent_detection_rate=%.3f want 1.0", m.SloppyAgentDetectionRate)
	}
	if m.BrokenAgentDetectionRate != 1.0 {
		t.Fatalf("broken_agent_detection_rate=%.3f want 1.0", m.BrokenAgentDetectionRate)
	}
	if m.MemoryUseTelemetryCompleteRate != 1.0 {
		t.Fatalf("memory_use_telemetry_complete_rate=%.3f want 1.0", m.MemoryUseTelemetryCompleteRate)
	}
	if m.UsedMemoryCitationRate != 1.0 {
		t.Fatalf("used_memory_citation_rate=%.3f want 1.0", m.UsedMemoryCitationRate)
	}
	if m.ContractFieldCitationRate != 1.0 {
		t.Fatalf("contract_field_citation_rate=%.3f want 1.0", m.ContractFieldCitationRate)
	}
	if m.UnsupportedOutputClaimRate != 0 {
		t.Fatalf("unsupported_output_claim_rate=%.3f want 0", m.UnsupportedOutputClaimRate)
	}
	if m.MCPRESTObedienceParityRate != 1.0 {
		t.Fatalf("mcp_rest_obedience_parity_rate=%.3f want 1.0", m.MCPRESTObedienceParityRate)
	}
}

func computeBenchmarkMetrics(cases []ObedienceCase) BenchmarkMetrics {
	var (
		obedientTotal, obedientPass       int
		sloppyTotal, sloppyDetected       int
		brokenTotal, brokenDetected       int
		telComplete, telTotal             int
		citationOK, citationTotal         int
		fieldCitationOK, fieldCitationTotal int
		unsupportedClaims, unsupportedDenom int
		mcpParityOK, mcpParityDenom       int
		overallPass, overallTotal         int
	)

	for _, c := range cases {
		ifaces := []string{c.Interface}
		if c.Interface == "both" {
			ifaces = []string{InterfaceREST, InterfaceMCP}
			mcpParityDenom++
			restEv := runEval(c, InterfaceREST)
			mcpEv := runEval(c, InterfaceMCP)
			if restEv.ObediencePassed && mcpEv.ObediencePassed {
				mcpParityOK++
			}
		}
		for _, iface := range ifaces {
			if iface == "both" {
				continue
			}
			tel, ev := RunCase(c, iface)
			overallTotal++
			if DetectionPassed(c, ev) {
				overallPass++
			}

			switch c.AgentMode {
			case AgentObedient:
				obedientTotal++
				if DetectionPassed(c, ev) {
					obedientPass++
				}
				if ev.ObediencePassed {
					unsupportedDenom++
				}
			case AgentSloppy:
				sloppyTotal++
				if DetectionPassed(c, ev) {
					sloppyDetected++
				}
			case AgentBroken:
				brokenTotal++
				if DetectionPassed(c, ev) {
					brokenDetected++
				}
			}

			if c.AgentMode == AgentObedient {
				telTotal++
				if tel.TelemetryComplete {
					telComplete++
				}
				for _, id := range tel.UsedMemoryIDs {
					citationTotal++
					if contains(tel.FinalOutput.MemoryCitations, id) {
						citationOK++
					}
				}
				for _, d := range tel.MemoryDecisions {
					if d.Decision == "used" {
						fieldCitationTotal++
						if len(d.ContractFieldsCited) > 0 {
							fieldCitationOK++
						}
					}
				}
				for _, f := range tel.FinalOutput.Facts {
					if f == "invented_unsupported_fact_xyz" || f == "fact_from_unrecalled_memory" {
						unsupportedClaims++
					}
				}
			}
		}
	}

	return BenchmarkMetrics{
		ObediencePassRate:                rate(overallPass, overallTotal),
		ObedientAgentPassRate:            rate(obedientPass, obedientTotal),
		SloppyAgentDetectionRate:         rate(sloppyDetected, sloppyTotal),
		BrokenAgentDetectionRate:         rate(brokenDetected, brokenTotal),
		MemoryUseTelemetryCompleteRate:   rate(telComplete, telTotal),
		UsedMemoryCitationRate:           rate(citationOK, citationTotal),
		ContractFieldCitationRate:        rate(fieldCitationOK, fieldCitationTotal),
		UnsupportedOutputClaimRate:       rate(unsupportedClaims, unsupportedDenom),
		MCPRESTObedienceParityRate:       rate(mcpParityOK, mcpParityDenom),
	}
}

func runEval(c ObedienceCase, iface string) ObedienceEvaluation {
	_, ev := RunCase(c, iface)
	return ev
}

func rate(num, denom int) float64 {
	if denom == 0 {
		return 0
	}
	return float64(num) / float64(denom)
}
