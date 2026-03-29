package eval

import (
	"context"
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/internal/recall"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

func TestProofScenariosEmbedded(t *testing.T) {
	sc, err := LoadProofScenarios()
	if err != nil {
		t.Fatal(err)
	}
	if len(sc) < 9 {
		t.Fatalf("expected at least 9 proof-*.json scenarios, got %d", len(sc))
	}
}

func TestEvaluationHarness(t *testing.T) {
	report, err := RunAllScenarios()
	if err != nil {
		t.Fatalf("run harness: %v", err)
	}
	t.Log("\n" + FormatRunReportSummary(report))
	for _, dr := range report.DualReports {
		t.Log("\n" + FormatDualScenarioReport(dr))
	}
	if !report.AllPassed {
		t.Fatal("memory effectiveness evaluation failed for one or more scenarios (explicit or triggered arm)")
	}
}

func TestDualModeDeterminism(t *testing.T) {
	scenarios, err := LoadScenarios()
	if err != nil {
		t.Fatal(err)
	}
	if len(scenarios) == 0 {
		t.Fatal("no scenarios")
	}
	s := scenarios[0]
	writes := RunInitialScenarioExtraction(s)
	_ = ValidateExtraction(s, writes)
	ctx := context.Background()
	svc := newEvalRecallService(writes)

	_, meta1, err := runScenarioTriggered(ctx, svc, s)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runScenarioExplicit(ctx, svc, s); err != nil {
		t.Fatal(err)
	}
	svc2 := newEvalRecallService(writes)
	_, meta2, err := runScenarioTriggered(ctx, svc2, s)
	if err != nil {
		t.Fatal(err)
	}
	if meta1 == nil || meta2 == nil {
		t.Fatal("nil metadata")
	}
	if meta1.SkippedReason != meta2.SkippedReason {
		t.Fatalf("skipped_reason mismatch: %q vs %q", meta1.SkippedReason, meta2.SkippedReason)
	}
	if len(meta1.Triggers) != len(meta2.Triggers) {
		t.Fatalf("trigger count mismatch: %d vs %d", len(meta1.Triggers), len(meta2.Triggers))
	}
	if meta1.RetrievalQueryEffective != meta2.RetrievalQueryEffective {
		t.Fatalf("effective query mismatch: %q vs %q", meta1.RetrievalQueryEffective, meta2.RetrievalQueryEffective)
	}
}

func TestTriggeredDoesNotExceedTriggerCap(t *testing.T) {
	cfg := evalTriggerConfig()
	if cfg == nil {
		t.Fatal("nil config")
	}
	max := cfg.MaxTriggersPerRequest
	scenarios, err := LoadScenarios()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, s := range scenarios {
		writes := RunInitialScenarioExtraction(s)
		svc := newEvalRecallService(writes)
		_, meta, err := runScenarioTriggered(ctx, svc, s)
		if err != nil {
			t.Fatalf("%s: %v", s.ID, err)
		}
		if len(meta.Triggers) > max {
			t.Fatalf("%s: triggers %d > cap %d", s.ID, len(meta.Triggers), max)
		}
	}
}

func TestDetectTriggersFromScenarios(t *testing.T) {
	scenarios, err := LoadScenarios()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range scenarios {
		req := buildCompileRequest(s)
		in := recall.TriggerInput{ProposalText: req.ProposalText, ExistingQuery: req.RetrievalQuery}
		d := recall.DetectTriggers(in, 4)
		t.Logf("%s: detect_triggers=%d", s.ID, len(d))
	}
}

func TestCompileTriggeredMetadataMatchesDetectTriggers(t *testing.T) {
	scenarios, err := LoadScenarios()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	cfg := evalTriggerConfig()
	for _, s := range scenarios {
		writes := RunInitialScenarioExtraction(s)
		svc := newEvalRecallService(writes)
		req := buildCompileRequest(s)
		_, meta, err := svc.CompileTriggered(ctx, req)
		if err != nil {
			t.Fatalf("%s: %v", s.ID, err)
		}
		if meta == nil {
			t.Fatalf("%s: nil meta", s.ID)
		}
		raw := recall.DetectTriggers(recall.TriggerInput{
			ProposalText:  req.ProposalText,
			ExistingQuery: req.RetrievalQuery,
		}, cfg.MinContextTokens)
		if len(raw) > 0 && len(meta.Triggers) == 0 {
			t.Errorf("%s: expected some triggers after detection (raw=%d)", s.ID, len(raw))
		}
		if cfg != nil && len(meta.Triggers) > cfg.MaxTriggersPerRequest {
			t.Errorf("%s: too many triggers", s.ID)
		}
	}
}

func TestNoRegressionWhenExplicitPasses(t *testing.T) {
	scenarios, err := LoadScenarios()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, s := range scenarios {
		writes := RunInitialScenarioExtraction(s)
		svc := newEvalRecallService(writes)
		bundleEx, err := runScenarioExplicit(ctx, svc, s)
		if err != nil {
			t.Fatalf("%s: %v", s.ID, err)
		}
		recallEx := ValidateRecall(s, bundleEx)
		outEx := RunFollowupBehavior(s, bundleEx)
		behEx := ValidateBehavior(s, bundleEx, outEx)
		exAll := recallEx.Pass && behEx.Pass
		if !exAll {
			continue
		}
		bundleTr, _, err := runScenarioTriggered(ctx, svc, s)
		if err != nil {
			t.Fatalf("%s: %v", s.ID, err)
		}
		recallTr := ValidateRecall(s, bundleTr)
		outTr := RunFollowupBehavior(s, bundleTr)
		behTr := ValidateBehavior(s, bundleTr, outTr)
		if !behTr.AvoidedFailure && behEx.AvoidedFailure {
			t.Fatalf("%s: triggered lost avoided_failure", s.ID)
		}
		if !behTr.AppliedPattern && behEx.AppliedPattern {
			t.Fatalf("%s: triggered lost applied_pattern", s.ID)
		}
		if !behTr.AlignedWithDecision && behEx.AlignedWithDecision {
			t.Fatalf("%s: triggered lost aligned_with_decision", s.ID)
		}
		if !recallTr.Pass && recallEx.Pass {
			t.Fatalf("%s: triggered recall failed when explicit passed", s.ID)
		}
	}
}

func TestValidateRecall_mustBeFirst_crossAgentDominates(t *testing.T) {
	id1 := uuid.MustParse("f0000000-0000-0000-0000-000000000001")
	id2 := uuid.MustParse("f0000000-0000-0000-0000-000000000002")
	now := time.Now()
	writes := []memory.MemoryObject{
		{ID: id1, Kind: api.MemoryKindPattern, Statement: "noise pattern without salience", Authority: 8, UpdatedAt: now},
		{ID: id2, Kind: api.MemoryKindPattern, Statement: "shared rollout victor with agents", Authority: 8, UpdatedAt: now, Payload: []byte(`{"salience":{"distinct_agents":4}}`)},
	}
	s := Scenario{
		ID: "must-be-first-test",
		RecallExpectations: struct {
			Query       string   `json:"query"`
			MustInclude []string `json:"must_include"`
			MustBeFirst []string `json:"must_be_first,omitempty"`
		}{
			Query:       "rollout shared agents",
			MustBeFirst: []string{"experience::shared rollout victor"},
		},
	}
	b, err := RunRecall(s, writes)
	if err != nil {
		t.Fatal(err)
	}
	res := ValidateRecall(s, b)
	if !res.Pass {
		t.Fatalf("ValidateRecall: %v", res.Details)
	}
}
