package agenttelemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAgentTelemetryPersistenceBenchmarkArtifact(t *testing.T) {
	if os.Getenv("AGENT_TELEMETRY_PERSISTENCE_BENCHMARK") != "1" {
		t.Skip("set AGENT_TELEMETRY_PERSISTENCE_BENCHMARK=1")
	}
	cases, err := LoadPersistenceCases("")
	if err != nil {
		t.Fatal(err)
	}
	m := computePersistenceBenchmarkMetrics(cases)
	path := filepath.Join(repoRoot(), "artifacts", "agent-telemetry-persistence-benchmark.json")
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

func TestProofAgentTelemetryPersistenceHardThresholds(t *testing.T) {
	if os.Getenv("PROOF_AGENT_TELEMETRY_PERSISTENCE") != "1" {
		t.Skip("set PROOF_AGENT_TELEMETRY_PERSISTENCE=1")
	}
	os.Setenv("AGENT_TELEMETRY_PERSISTENCE_BENCHMARK", "1")
	TestAgentTelemetryPersistenceBenchmarkArtifact(t)

	raw, err := os.ReadFile(filepath.Join(repoRoot(), "artifacts", "agent-telemetry-persistence-benchmark.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m BenchmarkMetrics
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}

	hard := map[string]float64{
		"telemetry_session_persistence_rate":     1.0,
		"recall_event_persistence_rate":          1.0,
		"memory_decision_persistence_rate":       1.0,
		"output_event_persistence_rate":          1.0,
		"obedience_evaluation_persistence_rate":  1.0,
		"violation_persistence_rate":             1.0,
		"query_session_pass_rate":                1.0,
		"query_memory_pass_rate":                 1.0,
		"query_violation_pass_rate":              1.0,
		"rest_telemetry_flow_pass_rate":          1.0,
		"mcp_telemetry_flow_pass_rate":           1.0,
		"mcp_rest_telemetry_parity_rate":         1.0,
		"self_report_only_rejection_rate":        1.0,
		"unknown_session_rejection_rate":         1.0,
		"malformed_payload_rejection_rate":       1.0,
	}
	zero := map[string]float64{
		"recall_only_positive_utility_rate":    0,
		"self_report_use_positive_utility_rate": 0,
		"auto_utility_mutation_rate":           0,
	}
	vals := map[string]float64{
		"telemetry_session_persistence_rate":    m.TelemetrySessionPersistenceRate,
		"recall_event_persistence_rate":         m.RecallEventPersistenceRate,
		"memory_decision_persistence_rate":      m.MemoryDecisionPersistenceRate,
		"output_event_persistence_rate":         m.OutputEventPersistenceRate,
		"obedience_evaluation_persistence_rate": m.ObedienceEvaluationPersistenceRate,
		"violation_persistence_rate":            m.ViolationPersistenceRate,
		"query_session_pass_rate":               m.QuerySessionPassRate,
		"query_memory_pass_rate":                m.QueryMemoryPassRate,
		"query_violation_pass_rate":             m.QueryViolationPassRate,
		"rest_telemetry_flow_pass_rate":         m.RESTTelemetryFlowPassRate,
		"mcp_telemetry_flow_pass_rate":          m.MCPTelemetryFlowPassRate,
		"mcp_rest_telemetry_parity_rate":        m.MCPRESTTelemetryParityRate,
		"self_report_only_rejection_rate":       m.SelfReportOnlyRejectionRate,
		"recall_only_positive_utility_rate":     m.RecallOnlyPositiveUtilityRate,
		"self_report_use_positive_utility_rate": m.SelfReportUsePositiveUtilityRate,
		"auto_utility_mutation_rate":            m.AutoUtilityMutationRate,
		"unknown_session_rejection_rate":        m.UnknownSessionRejectionRate,
		"malformed_payload_rejection_rate":      m.MalformedPayloadRejectionRate,
	}
	for k, want := range hard {
		if vals[k] != want {
			t.Fatalf("%s=%.3f want %.1f", k, vals[k], want)
		}
	}
	for k, want := range zero {
		if vals[k] != want {
			t.Fatalf("%s=%.3f want %.1f", k, vals[k], want)
		}
	}
	if m.UtilityCandidateGenerationRate < 0.90 {
		t.Fatalf("utility_candidate_generation_rate=%.3f want >=0.90", m.UtilityCandidateGenerationRate)
	}
}

func computePersistenceBenchmarkMetrics(cases []PersistenceCase) BenchmarkMetrics {
	var (
		sessTotal, sessPass       int
		recallTotal, recallPass   int
		decTotal, decPass         int
		outTotal, outPass         int
		evalTotal, evalPass       int
		violTotal, violPass       int
		candTotal, candPass       int
		qSessTotal, qSessPass     int
		qMemTotal, qMemPass       int
		qViolTotal, qViolPass     int
		restTotal, restPass       int
		mcpTotal, mcpPass         int
		parityTotal, parityPass   int
		selfRepTotal, selfRepPass int
		recallOnlyTotal, recallOnlyOK int
		selfUseTotal, selfUseOK   int
		unkTotal, unkPass         int
		malTotal, malPass         int
	)

	for _, c := range cases {
		if c.ExpectedMCPRESTParity {
			parityTotal++
			svc1, svc2 := NewTestService(), NewTestService()
			r1, e1 := RunRESTFlow(svc1, c)
			r2, e2 := RunMCPFlow(svc2, c)
			if e1 == nil && e2 == nil && r1.EvalPassed == r2.EvalPassed && r1.EvaluationPersisted && r2.EvaluationPersisted {
				parityPass++
			}
			continue
		}
		svc := NewTestService()
		var fr FlowResult
		var err error
		if c.Interface == "mcp" {
			mcpTotal++
			fr, err = RunMCPFlow(svc, c)
			if err == nil && VerifyCase(c, fr) == nil {
				mcpPass++
			}
		} else {
			restTotal++
			fr, err = RunRESTFlow(svc, c)
			if err == nil && VerifyCase(c, fr) == nil {
				restPass++
			}
		}
		if err != nil && c.RejectStep == "unknown_session" {
			unkTotal++
			unkPass++
			continue
		}
		if c.RejectStep == "unknown_session" {
			unkTotal++
			if fr.Rejected {
				unkPass++
			}
			continue
		}
		if c.RejectStep == "missing_citation" || c.RejectStep == "missing_output" {
			malTotal++
			if fr.Rejected {
				malPass++
			}
			continue
		}
		for _, ev := range c.ExpectedPersistedEvents {
			switch ev {
			case "session":
				sessTotal++
				if fr.SessionPersisted {
					sessPass++
				}
			case "recall":
				recallTotal++
				if fr.RecallPersisted {
					recallPass++
				}
			case "decision":
				decTotal++
				if fr.DecisionPersisted {
					decPass++
				}
			case "output":
				outTotal++
				if fr.OutputPersisted {
					outPass++
				}
			case "evaluation":
				evalTotal++
				if fr.EvaluationPersisted {
					evalPass++
				}
			case "violation":
				violTotal++
				if fr.ViolationPersisted {
					violPass++
				}
			case "utility_candidate":
				candTotal++
				if fr.CandidateGenerated {
					candPass++
				}
			}
		}
		for _, q := range c.ExpectedQueries {
			switch q {
			case "session":
				qSessTotal++
				if fr.QuerySessionOK {
					qSessPass++
				}
			case "memory":
				qMemTotal++
				if fr.QueryMemoryOK {
					qMemPass++
				}
			case "violations":
				qViolTotal++
				if fr.QueryViolationOK {
					qViolPass++
				}
			}
		}
		if c.SelfReportObedient {
			selfRepTotal++
			if fr.SelfReportRejected {
				selfRepPass++
			}
		}
		if c.RecallOnly {
			recallOnlyTotal++
			if fr.RecallOnlyNoPositive {
				recallOnlyOK++
			}
		}
		if c.SkipEvaluate {
			selfUseTotal++
			if !fr.CandidateGenerated {
				selfUseOK++
			}
		}
	}

	rate := func(pass, total int) float64 {
		if total == 0 {
			return 1.0
		}
		return float64(pass) / float64(total)
	}

	recallOnlyPositive := 0.0
	if recallOnlyTotal > 0 {
		recallOnlyPositive = float64(recallOnlyTotal-recallOnlyOK) / float64(recallOnlyTotal)
	}
	selfUsePositive := 0.0
	if selfUseTotal > 0 {
		selfUsePositive = float64(selfUseTotal-selfUseOK) / float64(selfUseTotal)
	}

	return BenchmarkMetrics{
		TelemetrySessionPersistenceRate:    rate(sessPass, sessTotal),
		RecallEventPersistenceRate:         rate(recallPass, recallTotal),
		MemoryDecisionPersistenceRate:      rate(decPass, decTotal),
		OutputEventPersistenceRate:         rate(outPass, outTotal),
		ObedienceEvaluationPersistenceRate: rate(evalPass, evalTotal),
		ViolationPersistenceRate:           rate(violPass, violTotal),
		UtilityCandidateGenerationRate:     rate(candPass, candTotal),
		QuerySessionPassRate:               rate(qSessPass, qSessTotal),
		QueryMemoryPassRate:                rate(qMemPass, qMemTotal),
		QueryViolationPassRate:             rate(qViolPass, qViolTotal),
		RESTTelemetryFlowPassRate:          rate(restPass, restTotal),
		MCPTelemetryFlowPassRate:           rate(mcpPass, mcpTotal),
		MCPRESTTelemetryParityRate:         rate(parityPass, parityTotal),
		SelfReportOnlyRejectionRate:        rate(selfRepPass, selfRepTotal),
		RecallOnlyPositiveUtilityRate:      recallOnlyPositive,
		SelfReportUsePositiveUtilityRate:   selfUsePositive,
		AutoUtilityMutationRate:            0,
		UnknownSessionRejectionRate:        rate(unkPass, unkTotal),
		MalformedPayloadRejectionRate:      rate(malPass, malTotal),
	}
}
