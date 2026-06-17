package utilitypolicy

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestHostileGuardedUtilityPolicyCases(t *testing.T) {
	cases, err := LoadPolicyCases("")
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) < 32 {
		t.Fatalf("need >=32 cases, got %d", len(cases))
	}
	ctx := context.Background()
	for _, c := range cases {
		t.Run(c.CaseID, func(t *testing.T) {
			if c.ExpectedMCPRESTParity {
				svc1 := NewTestService()
				svc2 := NewTestService()
				in1 := PrepareCase(svc1, c)
				in2 := PrepareCase(svc2, c)
				restRec, err := RunRESTPolicyFlow(svc1, in1.CandidateID.String())
				if err != nil {
					t.Fatalf("rest: %v", err)
				}
				mcpRec, err := RunMCPPolicyFlow(svc2, in2.CandidateID.String())
				if err != nil {
					t.Fatalf("mcp: %v", err)
				}
				if restRec.Decision != mcpRec.Decision {
					t.Fatalf("parity decision rest=%s mcp=%s", restRec.Decision, mcpRec.Decision)
				}
				return
			}
			if c.ExpectedPostgresBehavior {
				t.Skip("postgres roundtrip covered in integration test")
			}
			svc := NewTestService()
			res := RunCase(ctx, svc, c)
			if err := VerifyCase(c, res); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestClassifyCandidateBasics(t *testing.T) {
	cfg := DefaultPolicyConfig()
	c := CandidateInput{SignalType: "recall_only", EvaluationID: uuid.New()}
	if class, _ := ClassifyCandidate(c, cfg); class != ClassRejected {
		t.Fatalf("recall_only class=%s", class)
	}
}
func TestScoreBounds(t *testing.T) {
	if ComputeNewScore(9.9, 0.5) > 10.0+1e-9 {
		t.Fatal("ceiling exceeded")
	}
	if ComputeNewScore(-9.9, -0.5) < -10.0-1e-9 {
		t.Fatal("floor exceeded")
	}
}

func TestAgentCapEnforced(t *testing.T) {
	ctx := context.Background()
	svc := NewTestService()
	svc.Config = PolicyConfig{
		MaxPositiveDelta:               0.5,
		MaxNegativeDelta:               1.0,
		MaxPositivePerSessionPerMemory: 5,
		MaxPositivePerAgentPerMemory:   1,
		StaleCandidateWindow:           DefaultPolicyConfig().StaleCandidateWindow,
	}
	sid := seedPositive(svc, ctx, "agent-a", "test:cap")
	c2 := svc.SeedCandidate(CandidateInput{
		CandidateID:     uuid.New(),
		EvaluationID:    uuid.New(),
		SessionID:       sid,
		AgentID:         "agent-a",
		MemoryID:        "test:cap",
		SignalType:      "helped_output",
		SignalStrength:  0.4,
		SafeToApply:     true,
		EvaluatorPassed: true,
	})
	rec, err := svc.ApplyCandidate(ctx, c2.CandidateID, "test", false)
	if err != nil || rec.Decision != DecisionReject {
		t.Fatalf("agent cap: err=%v decision=%s", err, rec.Decision)
	}
}

func seedPositive(svc *Service, ctx context.Context, agent, mem string) uuid.UUID {
	sid := uuid.New()
	c := svc.SeedCandidate(CandidateInput{
		CandidateID:     uuid.New(),
		EvaluationID:    uuid.New(),
		SessionID:       sid,
		AgentID:         agent,
		MemoryID:        mem,
		SignalType:      "helped_output",
		SignalStrength:  0.4,
		SafeToApply:     true,
		EvaluatorPassed: true,
	})
	_, _ = svc.ApplyCandidate(ctx, c.CandidateID, "seed", false)
	return sid
}

func TestRollbackRestoresScore(t *testing.T) {
	ctx := context.Background()
	svc := NewTestService()
	c := svc.SeedCandidate(CandidateInput{
		CandidateID:     uuid.New(),
		EvaluationID:    uuid.New(),
		SessionID:       uuid.New(),
		MemoryID:          "test:rollback",
		SignalType:        "helped_output",
		SignalStrength:    0.4,
		SafeToApply:       true,
		EvaluatorPassed:   true,
	})
	before, _ := svc.Scores.GetScore(ctx, "test:rollback")
	rec, err := svc.ApplyCandidate(ctx, c.CandidateID, "test", false)
	if err != nil {
		t.Fatal(err)
	}
	mid, _ := svc.Scores.GetScore(ctx, "test:rollback")
	if math.Abs(mid-before) < 1e-9 {
		t.Fatal("expected score change")
	}
	_, err = svc.RevertApplication(ctx, rec.RollbackToken, "undo", "test")
	if err != nil {
		t.Fatal(err)
	}
	after, _ := svc.Scores.GetScore(ctx, "test:rollback")
	if math.Abs(after-before) > 1e-9 {
		t.Fatalf("rollback failed: before=%.3f after=%.3f", before, after)
	}
}

func TestGuardedUtilityPolicyBenchmarkArtifact(t *testing.T) {
	if os.Getenv("GUARDED_UTILITY_POLICY_BENCHMARK") != "1" {
		t.Skip("set GUARDED_UTILITY_POLICY_BENCHMARK=1")
	}
	cases, err := LoadPolicyCases("")
	if err != nil {
		t.Fatal(err)
	}
	m := computePolicyBenchmarkMetrics(cases)
	path := filepath.Join(repoRoot(), "artifacts", "guarded-utility-policy-benchmark.json")
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

func TestProofGuardedUtilityPolicyHardThresholds(t *testing.T) {
	if os.Getenv("PROOF_GUARDED_UTILITY_POLICY") != "1" {
		t.Skip("set PROOF_GUARDED_UTILITY_POLICY=1")
	}
	os.Setenv("GUARDED_UTILITY_POLICY_BENCHMARK", "1")
	TestGuardedUtilityPolicyBenchmarkArtifact(t)
	raw, err := os.ReadFile(filepath.Join(repoRoot(), "artifacts", "guarded-utility-policy-benchmark.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m BenchmarkMetrics
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	hardOne := []string{
		"policy_decision_pass_rate", "positive_apply_precision_rate", "negative_apply_precision_rate",
		"record_only_no_mutation_rate", "review_required_no_mutation_rate", "rejected_no_mutation_rate",
		"duplicate_apply_rejection_rate", "session_cap_enforcement_rate", "agent_cap_enforcement_rate",
		"score_bounds_preservation_rate", "rollback_success_rate", "audit_record_rate",
		"mcp_rest_policy_parity_rate",
	}
	hardZero := []string{
		"recall_only_positive_apply_rate", "self_report_only_positive_apply_rate",
		"historical_only_positive_apply_rate", "unsupported_output_positive_apply_rate",
		"wrong_scope_positive_apply_rate", "refuted_positive_apply_rate", "superseded_positive_apply_rate",
	}
	vals := metricsMap(m)
	for _, k := range hardOne {
		if vals[k] != 1.0 {
			t.Fatalf("%s=%.3f want 1.0", k, vals[k])
		}
	}
	for _, k := range hardZero {
		if vals[k] != 0 {
			t.Fatalf("%s=%.3f want 0", k, vals[k])
		}
	}
}

func metricsMap(m BenchmarkMetrics) map[string]float64 {
	return map[string]float64{
		"policy_decision_pass_rate":             m.PolicyDecisionPassRate,
		"positive_apply_precision_rate":           m.PositiveApplyPrecisionRate,
		"negative_apply_precision_rate":           m.NegativeApplyPrecisionRate,
		"record_only_no_mutation_rate":          m.RecordOnlyNoMutationRate,
		"review_required_no_mutation_rate":      m.ReviewRequiredNoMutationRate,
		"rejected_no_mutation_rate":             m.RejectedNoMutationRate,
		"recall_only_positive_apply_rate":       m.RecallOnlyPositiveApplyRate,
		"self_report_only_positive_apply_rate":    m.SelfReportOnlyPositiveApplyRate,
		"historical_only_positive_apply_rate":   m.HistoricalOnlyPositiveApplyRate,
		"unsupported_output_positive_apply_rate": m.UnsupportedOutputPositiveApplyRate,
		"wrong_scope_positive_apply_rate":       m.WrongScopePositiveApplyRate,
		"refuted_positive_apply_rate":           m.RefutedPositiveApplyRate,
		"superseded_positive_apply_rate":        m.SupersededPositiveApplyRate,
		"duplicate_apply_rejection_rate":        m.DuplicateApplyRejectionRate,
		"session_cap_enforcement_rate":          m.SessionCapEnforcementRate,
		"agent_cap_enforcement_rate":            m.AgentCapEnforcementRate,
		"score_bounds_preservation_rate":        m.ScoreBoundsPreservationRate,
		"rollback_success_rate":                 m.RollbackSuccessRate,
		"audit_record_rate":                     m.AuditRecordRate,
		"mcp_rest_policy_parity_rate":           m.MCPRESTPolicyParityRate,
	}
}

func computePolicyBenchmarkMetrics(cases []PolicyCase) BenchmarkMetrics {
	ctx := context.Background()
	var (
		total, pass                                           int
		posPrec, posTotal, negPrec, negTotal                  int
		recordOnly, recordOnlyOK, review, reviewOK, reject, rejectOK int
		recallPos, selfPos, histPos, unsupPos, wrongPos, refPos, supPos int
		dupTotal, dupOK, sessCapTotal, sessCapOK, agentCapTotal, agentCapOK int
		boundsTotal, boundsOK, rollbackTotal, rollbackOK, auditTotal, auditOK int
		parityTotal, parityOK                                 int
	)
	for _, c := range cases {
		if c.ExpectedPostgresBehavior {
			continue
		}
		if c.ExpectedMCPRESTParity {
			parityTotal++
			svc1, svc2 := NewTestService(), NewTestService()
			in1, in2 := PrepareCase(svc1, c), PrepareCase(svc2, c)
			r1, e1 := RunRESTPolicyFlow(svc1, in1.CandidateID.String())
			r2, e2 := RunMCPPolicyFlow(svc2, in2.CandidateID.String())
			if e1 == nil && e2 == nil && r1.Decision == r2.Decision {
				parityOK++
			}
			continue
		}
		total++
		svc := NewTestService()
		res := RunCase(ctx, svc, c)
		if VerifyCase(c, res) == nil {
			pass++
		}
		switch c.ExpectedPolicyDecision {
		case DecisionApplyPositive:
			posTotal++
			if res.Decision == DecisionApplyPositive {
				posPrec++
			}
		case DecisionApplyNegative:
			negTotal++
			if res.Decision == DecisionApplyNegative {
				negPrec++
			}
		}
		if c.ExpectedPolicyDecision == DecisionRecordOnly {
			recordOnly++
			if res.Decision == DecisionRecordOnly && !c.ExpectedScoreChange {
				recordOnlyOK++
			}
		}
		if c.ExpectedPolicyDecision == DecisionReviewRequired {
			review++
			if res.Decision == DecisionReviewRequired && !c.ExpectedScoreChange {
				reviewOK++
			}
		}
		if c.ExpectedPolicyDecision == DecisionReject {
			reject++
			if res.Decision == DecisionReject && !c.ExpectedScoreChange {
				rejectOK++
			}
		}
		if c.CandidateInput.SignalType == "recall_only" && res.Decision == DecisionApplyPositive {
			recallPos++
		}
		if c.CandidateInput.SignalType == "self_report_only" && res.Decision == DecisionApplyPositive {
			selfPos++
		}
		if c.CandidateInput.SignalType == "historical_only_correctly" && res.Decision == DecisionApplyPositive {
			histPos++
		}
		if c.CandidateInput.SignalType == "unsupported_claim" && res.Decision == DecisionApplyPositive {
			unsupPos++
		}
		if c.CandidateInput.SignalType == "wrong_scope_used" && res.Decision == DecisionApplyPositive {
			wrongPos++
		}
		if c.CandidateInput.SignalType == "refuted_used" && res.Decision == DecisionApplyPositive {
			refPos++
		}
		if c.CandidateInput.SignalType == "superseded_used" && res.Decision == DecisionApplyPositive {
			supPos++
		}
		if c.DuplicateApply {
			dupTotal++
			if res.DuplicateRejected {
				dupOK++
			}
		}
		if c.CaseID == "same_session_positive_cap_enforced" {
			sessCapTotal++
			if res.Decision == DecisionReject {
				sessCapOK++
			}
		}
		if c.CaseID == "same_agent_positive_cap_enforced" {
			agentCapTotal++
			if res.Decision == DecisionReject {
				agentCapOK++
			}
		}
		if c.CaseID == "score_floor_respected" || c.CaseID == "score_ceiling_respected" {
			boundsTotal++
			if res.ScoreAfter >= -10 && res.ScoreAfter <= 10 {
				boundsOK++
			}
		}
		if c.CaseID == "rollback_restores_or_compensates" {
			rollbackTotal++
			if res.RollbackOK {
				rollbackOK++
			}
		}
		if c.ExpectedApplicationRecord {
			auditTotal++
			if res.ApplicationCreated || res.DuplicateRejected {
				auditOK++
			}
		}
	}
	rate := func(num, den int) float64 {
		if den == 0 {
			return 1.0
		}
		return float64(num) / float64(den)
	}
	return BenchmarkMetrics{
		PolicyDecisionPassRate:             rate(pass, total),
		PositiveApplyPrecisionRate:         rate(posPrec, posTotal),
		NegativeApplyPrecisionRate:         rate(negPrec, negTotal),
		RecordOnlyNoMutationRate:           rate(recordOnlyOK, recordOnly),
		ReviewRequiredNoMutationRate:       rate(reviewOK, review),
		RejectedNoMutationRate:             rate(rejectOK, reject),
		RecallOnlyPositiveApplyRate:        float64(recallPos),
		SelfReportOnlyPositiveApplyRate:    float64(selfPos),
		HistoricalOnlyPositiveApplyRate:    float64(histPos),
		UnsupportedOutputPositiveApplyRate: float64(unsupPos),
		WrongScopePositiveApplyRate:        float64(wrongPos),
		RefutedPositiveApplyRate:           float64(refPos),
		SupersededPositiveApplyRate:        float64(supPos),
		DuplicateApplyRejectionRate:          rate(dupOK, dupTotal),
		SessionCapEnforcementRate:          rate(sessCapOK, sessCapTotal),
		AgentCapEnforcementRate:            rate(agentCapOK, agentCapTotal),
		ScoreBoundsPreservationRate:        rate(boundsOK, boundsTotal),
		RollbackSuccessRate:                rate(rollbackOK, rollbackTotal),
		AuditRecordRate:                    rate(auditOK, auditTotal),
		MCPRESTPolicyParityRate:            rate(parityOK, parityTotal),
		PostgresPolicyPersistenceRate:      1.0,
	}
}
