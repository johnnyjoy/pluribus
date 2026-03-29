package recall

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"control-plane/internal/memory"
	"control-plane/internal/runmulti"
	"control-plane/internal/signal"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

type fakeRunMultiRunner struct {
	out       *runmulti.RunMultiResult
	err       error
	lastInput *runmulti.RunMultiInput
}

func (f fakeRunMultiRunner) Run(ctx context.Context, input runmulti.RunMultiInput) (*runmulti.RunMultiResult, error) {
	f.lastInput = &input
	return f.out, f.err
}

type fakeRunMultiRunnerPtr struct {
	out       *runmulti.RunMultiResult
	err       error
	lastInput *runmulti.RunMultiInput
}

func (f *fakeRunMultiRunnerPtr) Run(ctx context.Context, input runmulti.RunMultiInput) (*runmulti.RunMultiResult, error) {
	f.lastInput = &input
	return f.out, f.err
}

type fakePromoter struct {
	called         bool
	last           memory.PromoteRequest
	responseStatus string // empty => active
}

func (f *fakePromoter) Promote(ctx context.Context, req memory.PromoteRequest) (*memory.PromoteResponse, error) {
	f.called = true
	f.last = req
	st := f.responseStatus
	if st == "" {
		st = "active"
	}
	return &memory.PromoteResponse{Promoted: true, ID: "x", Status: st}, nil
}

type fakeUsageReinforcerRM struct {
	called bool
	ids    []uuid.UUID
	metas  []memory.ReinforceMeta
}

func (f *fakeUsageReinforcerRM) ReinforceRecallUsage(ctx context.Context, ids []uuid.UUID) error {
	f.called = true
	f.ids = append([]uuid.UUID(nil), ids...)
	return nil
}

func (f *fakeUsageReinforcerRM) ReinforceRecallUsageWithMeta(ctx context.Context, ids []uuid.UUID, meta memory.ReinforceMeta) error {
	f.called = true
	f.ids = append([]uuid.UUID(nil), ids...)
	f.metas = append(f.metas, meta)
	return nil
}

type fakeEvidenceScore struct {
	score float64
	err   error
}

func (f *fakeEvidenceScore) ScoreEvidenceIDs(ctx context.Context, ids []uuid.UUID) (float64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.score, nil
}

type fakeRunMultiMemorySearcher struct {
	list []memory.MemoryObject
}

func (f fakeRunMultiMemorySearcher) Search(ctx context.Context, req memory.SearchRequest) ([]memory.MemoryObject, error) {
	return f.list, nil
}

func (f fakeRunMultiMemorySearcher) SearchMemories(ctx context.Context, req memory.MemoriesSearchRequest) ([]memory.MemoryObject, error) {
	// Run-multi tests only require stable memory inputs.
	return f.list, nil
}

type fakeContradictionLister struct {
	ids []uuid.UUID
}

func (f fakeContradictionLister) ListMemoryIDsInUnresolved(ctx context.Context) ([]uuid.UUID, error) {
	return f.ids, nil
}

func TestService_RunMulti_requiresRunner(t *testing.T) {
	svc := &Service{}
	_, err := svc.RunMulti(context.Background(), RunMultiRequest{Query: "x"})
	if err != ErrRunMultiNotConfigured {
		t.Fatalf("expected ErrRunMultiNotConfigured, got %v", err)
	}
}

func TestService_RunMulti_returnsScoresAndSelected(t *testing.T) {
	svc := &Service{
		RunMultiRunner: fakeRunMultiRunner{
			out: &runmulti.RunMultiResult{
				Runs: []runmulti.RunResult{
					{Variant: "balanced", Score: 2, Rejected: false, Output: "a"},
					{Variant: "failure_heavy", Score: 1, Rejected: false, Output: "b"},
				},
				Selected: &runmulti.RunResult{Variant: "failure_heavy", Score: 1, Rejected: false, Output: "b"},
			},
		},
	}
	resp, err := svc.RunMulti(context.Background(), RunMultiRequest{Query: "x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Selected == nil {
		t.Fatalf("expected selected")
	}
	if len(resp.Scores) != 2 {
		t.Fatalf("expected 2 scores, got %d", len(resp.Scores))
	}
	if resp.Confidence <= 0 {
		t.Fatalf("expected positive confidence")
	}
	if resp.Debug.SignalBreakdown == nil || resp.Debug.FilterReasons == nil || resp.Debug.PromotionDecision == nil || resp.Debug.Orchestration == nil {
		t.Fatalf("expected non-nil debug maps")
	}
	if resp.Debug.Orchestration["variants_requested"] != 3 {
		t.Fatalf("orchestration variants_requested = %v, want 3", resp.Debug.Orchestration["variants_requested"])
	}
	if resp.Debug.PromotionDecision["attempted"] != false {
		t.Fatalf("expected promotion not attempted without promote flag")
	}
	if resp.Debug.PromotionDecision["rejection_code"] != RejectionCodePromoteNotAttempted {
		t.Fatalf("rejection_code = %v, want %s", resp.Debug.PromotionDecision["rejection_code"], RejectionCodePromoteNotAttempted)
	}
}

func TestService_RunMulti_autoInjectsRecallOnRiskWhenTriggeredNotExplicit(t *testing.T) {
	runner := &fakeRunMultiRunnerPtr{
		out: &runmulti.RunMultiResult{
			Runs: []runmulti.RunResult{
				{Variant: "a", Score: 1, Rejected: false, Output: "do safe deploy"},
			},
			Selected: &runmulti.RunResult{Variant: "a", Score: 1, Rejected: false, Output: "do safe deploy"},
		},
	}
	svc := &Service{
		RunMultiRunner: runner,
		TriggerRecall:  &TriggerRecallConfig{Enabled: true, EnableRisk: true, EnableDecision: true, EnableSimilarity: true, MinContextTokens: 4, MaxTriggersPerRequest: 2},
		Compiler: &Compiler{
			Memory: fakeRunMultiMemorySearcher{list: []memory.MemoryObject{
				{ID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), Kind: api.MemoryKindConstraint, Statement: "never skip tests in release path", Authority: 9},
			}},
		},
	}
	resp, err := svc.RunMulti(context.Background(), RunMultiRequest{
		Query: "deploy to production now with schema migration",
		Merge: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner.lastInput == nil {
		t.Fatalf("expected runner input to be captured")
	}
	if strings.TrimSpace(runner.lastInput.RetrievalQuery) == "" {
		t.Fatalf("expected retrieval query to be injected")
	}
	raw, ok := resp.Debug.FilterReasons["interventions"]
	if !ok {
		t.Fatalf("expected interventions in debug filter reasons")
	}
	list, ok := raw.([]map[string]any)
	if !ok || len(list) == 0 {
		t.Fatalf("expected non-empty intervention list, got %#v", raw)
	}
}

func TestService_RunMulti_doesNotInjectRecallForLowSignalShortQuery(t *testing.T) {
	runner := &fakeRunMultiRunnerPtr{
		out: &runmulti.RunMultiResult{
			Runs: []runmulti.RunResult{
				{Variant: "a", Score: 1, Rejected: false, Output: "ok"},
			},
			Selected: &runmulti.RunResult{Variant: "a", Score: 1, Rejected: false, Output: "ok"},
		},
	}
	svc := &Service{
		RunMultiRunner: runner,
		TriggerRecall:  &TriggerRecallConfig{Enabled: true, EnableRisk: true, EnableDecision: true, EnableSimilarity: true, MinContextTokens: 4, MaxTriggersPerRequest: 2},
	}
	_, err := svc.RunMulti(context.Background(), RunMultiRequest{
		Query: "quick edit",
		Merge: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner.lastInput == nil {
		t.Fatalf("expected runner input to be captured")
	}
	if strings.TrimSpace(runner.lastInput.RetrievalQuery) != "" {
		t.Fatalf("expected no retrieval query injection for low-signal query, got %q", runner.lastInput.RetrievalQuery)
	}
}

func TestService_RunMulti_mergeIncludesMergedPayload(t *testing.T) {
	svc := &Service{
		RunMultiRunner: fakeRunMultiRunner{
			out: &runmulti.RunMultiResult{
				Runs: []runmulti.RunResult{
					{Variant: "a", Score: 1, Rejected: false, Output: "do x\n\n- keep y"},
					{Variant: "b", Score: 1.5, Rejected: false, Output: "do x\n\n- add z"},
				},
				Selected: &runmulti.RunResult{Variant: "a", Score: 1, Rejected: false, Output: "do x\n\n- keep y"},
			},
		},
	}
	resp, err := svc.RunMulti(context.Background(), RunMultiRequest{Query: "x", Merge: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Merged == nil {
		t.Fatalf("expected merged payload")
	}
	if _, ok := resp.Merged["merged_output"]; !ok {
		t.Fatalf("expected merged_output in merged payload")
	}
	if v, ok := resp.Debug.SignalBreakdown["total_signal"]; !ok {
		t.Fatalf("expected signal_breakdown.total_signal after merge")
	} else if _, ok := v.(float64); !ok {
		t.Fatalf("expected total_signal float64, got %T", v)
	}
}

func TestService_RunMulti_promotePassesRequireReviewFromPolicy(t *testing.T) {
	p := &fakePromoter{}
	outA := "Implement the payment retry flow with clear steps and validation.\n\n- Use exponential backoff\n- Surface errors to the user"
	outB := "Implement the payment retry flow with clear steps and validation.\n\n- Use exponential backoff\n- Log failures for support"
	svc := &Service{
		RunMultiRunner: fakeRunMultiRunner{
			out: &runmulti.RunMultiResult{
				Runs: []runmulti.RunResult{
					{Variant: "a", Score: 0.5, Rejected: false, Output: outA},
					{Variant: "b", Score: 1, Rejected: false, Output: outB},
				},
				Selected: &runmulti.RunResult{Variant: "a", Score: 0.5, Rejected: false, Output: outA},
			},
		},
		MemoryPromoter: p,
		Promotion:      &PromotionPolicy{RequireReview: true},
	}
	_, err := svc.RunMulti(context.Background(), RunMultiRequest{
		Query:     "payment retry flow",
		Merge:     true,
		Promote:   true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.last.RequireReview {
		t.Fatalf("expected RequireReview=true on PromoteRequest, got %+v", p.last)
	}
}

func TestService_RunMulti_promotionDecisionIncludesMemoryStatus(t *testing.T) {
	p := &fakePromoter{responseStatus: "pending"}
	outA := "Implement the payment retry flow with clear steps and validation.\n\n- Use exponential backoff\n- Surface errors to the user"
	outB := "Implement the payment retry flow with clear steps and validation.\n\n- Use exponential backoff\n- Log failures for support"
	svc := &Service{
		RunMultiRunner: fakeRunMultiRunner{
			out: &runmulti.RunMultiResult{
				Runs: []runmulti.RunResult{
					{Variant: "a", Score: 0.5, Rejected: false, Output: outA},
					{Variant: "b", Score: 1, Rejected: false, Output: outB},
				},
				Selected: &runmulti.RunResult{Variant: "a", Score: 0.5, Rejected: false, Output: outA},
			},
		},
		MemoryPromoter: p,
	}
	resp, err := svc.RunMulti(context.Background(), RunMultiRequest{
		Query:     "payment retry flow",
		Merge:     true,
		Promote:   true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Debug.PromotionDecision["memory_status"] != "pending" {
		t.Fatalf("memory_status = %v", resp.Debug.PromotionDecision["memory_status"])
	}
}

func TestService_RunMulti_promoteUsesMemoryPromoter(t *testing.T) {
	p := &fakePromoter{}
	reinf := &fakeUsageReinforcerRM{}
	// Shared long agreement line so merge yields agreements + total_signal >= default threshold.
	outA := "Implement the payment retry flow with clear steps and validation.\n\n- Use exponential backoff\n- Surface errors to the user"
	outB := "Implement the payment retry flow with clear steps and validation.\n\n- Use exponential backoff\n- Log failures for support"
	svc := &Service{
		RunMultiRunner: fakeRunMultiRunner{
			out: &runmulti.RunMultiResult{
				Runs: []runmulti.RunResult{
					{Variant: "a", Score: 0.5, Rejected: false, Output: outA},
					{Variant: "b", Score: 1, Rejected: false, Output: outB},
				},
				Selected: &runmulti.RunResult{Variant: "a", Score: 0.5, Rejected: false, Output: outA},
			},
		},
		MemoryPromoter:  p,
		UsageReinforcer: reinf,
		Compiler: &Compiler{
			Memory: fakeRunMultiMemorySearcher{list: []memory.MemoryObject{
				{ID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), Kind: api.MemoryKindDecision, Statement: "d1", Authority: 8},
			}},
		},
	}
	resp, err := svc.RunMulti(context.Background(), RunMultiRequest{
		Query:     "payment retry flow",
		Merge:     true,
		Promote:   true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.called {
		t.Fatalf("expected promoter to be called")
	}
	if !resp.Promoted {
		t.Fatalf("expected promoted=true, debug=%+v", resp.Debug.PromotionDecision)
	}
	if resp.Debug.PromotionDecision["reason"] != "all promotion gates passed" {
		t.Fatalf("expected gates passed reason, got %#v", resp.Debug.PromotionDecision["reason"])
	}
	if resp.Debug.PromotionDecision["rejection_code"] != RejectionCodeOK {
		t.Fatalf("rejection_code = %v, want %s", resp.Debug.PromotionDecision["rejection_code"], RejectionCodeOK)
	}
	if !reinf.called {
		t.Fatalf("expected usage reinforcer to be called on successful promote")
	}
}

func TestService_RunMulti_promotionBlockedByBehaviorValidation(t *testing.T) {
	p := &fakePromoter{}
	reinf := &fakeUsageReinforcerRM{}
	outA := "Implement the payment retry flow with clear steps and validation.\n\n- Use exponential backoff\n- Surface errors to the user"
	outB := "Implement the payment retry flow with clear steps and validation.\n\n- Use exponential backoff\n- Log failures for support"
	constraintStmt := "Implement the payment retry flow with clear steps and validation"
	svc := &Service{
		RunMultiRunner: fakeRunMultiRunner{
			out: &runmulti.RunMultiResult{
				Runs: []runmulti.RunResult{
					{Variant: "a", Score: 0.5, Rejected: false, Output: outA},
					{Variant: "b", Score: 1, Rejected: false, Output: outB},
				},
				Selected: &runmulti.RunResult{Variant: "a", Score: 0.5, Rejected: false, Output: outA},
			},
		},
		MemoryPromoter:  p,
		UsageReinforcer: reinf,
		Compiler: &Compiler{
			Memory: fakeRunMultiMemorySearcher{list: []memory.MemoryObject{
				{ID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), Kind: api.MemoryKindConstraint, Statement: constraintStmt, Authority: 9},
			}},
		},
	}
	resp, err := svc.RunMulti(context.Background(), RunMultiRequest{
		Query:     "payment retry flow",
		Merge:     true,
		Promote:   true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.called {
		t.Fatalf("expected promoter not to be called when behavior validation fails")
	}
	if resp.Promoted {
		t.Fatalf("expected promoted=false")
	}
	if !resp.Blocked {
		t.Fatalf("expected Blocked=true when validation fails")
	}
	if resp.Debug.PromotionDecision["rejection_code"] != RejectionCodeBehaviorValidation {
		t.Fatalf("rejection_code = %v, want %s", resp.Debug.PromotionDecision["rejection_code"], RejectionCodeBehaviorValidation)
	}
	if !reinf.called {
		t.Fatalf("expected usage reinforcer to be called for validation outcome learning")
	}
	foundHigh := false
	for _, m := range reinf.metas {
		if m.Impact == "high" && m.Reason == "constraint_block" {
			foundHigh = true
			break
		}
	}
	if !foundHigh {
		t.Fatalf("expected high-impact constraint_block reinforcement meta, got %+v", reinf.metas)
	}
}

func TestService_RunMulti_promoteSkippedWhenSignalLow(t *testing.T) {
	p := &fakePromoter{}
	svc := &Service{
		RunMultiRunner: fakeRunMultiRunner{
			out: &runmulti.RunMultiResult{
				Runs: []runmulti.RunResult{
					{Variant: "a", Score: 0.5, Rejected: false, Output: "do x"},
					{Variant: "b", Score: 1, Rejected: false, Output: "do y"},
				},
				Selected: &runmulti.RunResult{Variant: "a", Score: 0.5, Rejected: false, Output: "do x"},
			},
		},
		MemoryPromoter:       p,
		RunMultiSignalConfig: &signal.SignalConfig{MinTotalScore: 1000}, // impossible threshold
	}
	resp, err := svc.RunMulti(context.Background(), RunMultiRequest{
		Query:     "x",
		Merge:     true,
		Promote:   true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.called {
		t.Fatalf("expected promoter not to be called when signal gate fails")
	}
	if resp.Promoted {
		t.Fatalf("expected promoted=false")
	}
	if resp.Debug.PromotionDecision["reason"] != "signal below promotion threshold" {
		t.Fatalf("expected signal failure reason, got %#v", resp.Debug.PromotionDecision["reason"])
	}
	if resp.Debug.PromotionDecision["rejection_code"] != RejectionCodeSignalLow {
		t.Fatalf("rejection_code = %v, want %s", resp.Debug.PromotionDecision["rejection_code"], RejectionCodeSignalLow)
	}
}

func TestService_RunMulti_promoteEvidenceRequiredWithoutIDs(t *testing.T) {
	p := &fakePromoter{}
	svc := &Service{
		RunMultiRunner: fakeRunMultiRunner{
			out: &runmulti.RunMultiResult{
				Runs: []runmulti.RunResult{
					{Variant: "a", Score: 0.5, Rejected: false, Output: "Implement the payment retry flow with clear steps and validation.\n\n- Use exponential backoff\n- Surface errors"},
					{Variant: "b", Score: 1, Rejected: false, Output: "Implement the payment retry flow with clear steps and validation.\n\n- Use exponential backoff\n- Log failures"},
				},
				Selected: &runmulti.RunResult{Variant: "a", Score: 0.5, Rejected: false, Output: "Implement the payment retry flow with clear steps and validation.\n\n- Use exponential backoff\n- Surface errors"},
			},
		},
		MemoryPromoter: p,
		Promotion:      &PromotionPolicy{RequireEvidence: true},
		Evidence:       &fakeEvidenceScore{score: 1.0},
	}
	resp, err := svc.RunMulti(context.Background(), RunMultiRequest{
		Query:     "payment retry flow",
		Merge:     true,
		Promote:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.called {
		t.Fatal("promoter should not run without evidence_ids")
	}
	if resp.Debug.PromotionDecision["rejection_code"] != RejectionCodeEvidenceRequired {
		t.Fatalf("code = %v", resp.Debug.PromotionDecision["rejection_code"])
	}
}

func TestService_RunMulti_promotePassesEvidenceIDsToPromoter(t *testing.T) {
	p := &fakePromoter{}
	eid := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	outA := "Implement the payment retry flow with clear steps and validation.\n\n- Use exponential backoff\n- Surface errors to the user"
	outB := "Implement the payment retry flow with clear steps and validation.\n\n- Use exponential backoff\n- Log failures for support"
	svc := &Service{
		RunMultiRunner: fakeRunMultiRunner{
			out: &runmulti.RunMultiResult{
				Runs: []runmulti.RunResult{
					{Variant: "a", Score: 0.5, Rejected: false, Output: outA},
					{Variant: "b", Score: 1, Rejected: false, Output: outB},
				},
				Selected: &runmulti.RunResult{Variant: "a", Score: 0.5, Rejected: false, Output: outA},
			},
		},
		MemoryPromoter: p,
		Promotion:      &PromotionPolicy{RequireEvidence: true},
		Evidence:       &fakeEvidenceScore{score: 1.0},
	}
	_, err := svc.RunMulti(context.Background(), RunMultiRequest{
		Query:       "payment retry flow",
		Merge:       true,
		Promote:     true,
		EvidenceIDs: []uuid.UUID{eid},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !p.called {
		t.Fatal("expected promoter called")
	}
	if len(p.last.EvidenceIDs) != 1 || p.last.EvidenceIDs[0] != eid {
		t.Fatalf("EvidenceIDs = %+v", p.last.EvidenceIDs)
	}
}

func TestService_RunMulti_promoteSkippedWhenEvidenceScoreLow(t *testing.T) {
	p := &fakePromoter{}
	eid := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	outA := "Implement the payment retry flow with clear steps and validation.\n\n- Use exponential backoff\n- Surface errors to the user"
	outB := "Implement the payment retry flow with clear steps and validation.\n\n- Use exponential backoff\n- Log failures for support"
	svc := &Service{
		RunMultiRunner: fakeRunMultiRunner{
			out: &runmulti.RunMultiResult{
				Runs: []runmulti.RunResult{
					{Variant: "a", Score: 0.5, Rejected: false, Output: outA},
					{Variant: "b", Score: 1, Rejected: false, Output: outB},
				},
				Selected: &runmulti.RunResult{Variant: "a", Score: 0.5, Rejected: false, Output: outA},
			},
		},
		MemoryPromoter: p,
		Promotion:      &PromotionPolicy{MinEvidenceScore: 0.95},
		Evidence:       &fakeEvidenceScore{score: 0.2},
	}
	resp, err := svc.RunMulti(context.Background(), RunMultiRequest{
		Query:       "payment retry flow",
		Merge:       true,
		Promote:     true,
		EvidenceIDs: []uuid.UUID{eid},
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.called {
		t.Fatal("promoter should not run when evidence score too low")
	}
	if resp.Debug.PromotionDecision["rejection_code"] != RejectionCodeEvidenceScoreLow {
		t.Fatalf("code = %v", resp.Debug.PromotionDecision["rejection_code"])
	}
}

func TestService_RunMulti_promoteSkippedWhenPolicyCompositeLow(t *testing.T) {
	p := &fakePromoter{}
	outA := "Implement the payment retry flow with clear steps and validation.\n\n- Use exponential backoff\n- Surface errors to the user"
	outB := "Implement the payment retry flow with clear steps and validation.\n\n- Use exponential backoff\n- Log failures for support"
	svc := &Service{
		RunMultiRunner: fakeRunMultiRunner{
			out: &runmulti.RunMultiResult{
				Runs: []runmulti.RunResult{
					{Variant: "a", Score: 0.5, Rejected: false, Output: outA},
					{Variant: "b", Score: 1, Rejected: false, Output: outB},
				},
				Selected: &runmulti.RunResult{Variant: "a", Score: 0.5, Rejected: false, Output: outA},
			},
		},
		MemoryPromoter: p,
		Promotion: &PromotionPolicy{
			MinPolicyComposite: 0.99,
			WeightConfidence:   1,
			WeightSignal:       0,
			WeightEvidence:     0,
		},
	}
	resp, err := svc.RunMulti(context.Background(), RunMultiRequest{
		Query:     "payment retry flow",
		Merge:     true,
		Promote:   true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.called {
		t.Fatalf("expected promoter not to be called when policy composite gate fails")
	}
	if resp.Debug.PromotionDecision["rejection_code"] != RejectionCodePolicyCompositeLow {
		t.Fatalf("rejection_code = %v, want %s", resp.Debug.PromotionDecision["rejection_code"], RejectionCodePolicyCompositeLow)
	}
	gates, ok := resp.Debug.PromotionDecision["gates"].(map[string]any)
	if !ok || gates["policy_composite"] == nil {
		t.Fatalf("expected policy_composite in gates, got %#v", resp.Debug.PromotionDecision["gates"])
	}
	pi, ok := resp.Debug.PromotionDecision["policy_inputs"].(map[string]any)
	if !ok || pi["total_signal"] == nil {
		t.Fatalf("expected policy_inputs.total_signal, got %#v", resp.Debug.PromotionDecision["policy_inputs"])
	}
}

func TestService_RunMulti_promoteSkippedWhenConfidenceBelowMin(t *testing.T) {
	p := &fakePromoter{}
	outA := "Implement the payment retry flow with clear steps and validation.\n\n- Use exponential backoff\n- Surface errors to the user"
	outB := "Implement the payment retry flow with clear steps and validation.\n\n- Use exponential backoff\n- Log failures for support"
	svc := &Service{
		RunMultiRunner: fakeRunMultiRunner{
			out: &runmulti.RunMultiResult{
				Runs: []runmulti.RunResult{
					{Variant: "a", Score: 0.5, Rejected: false, Output: outA},
					{Variant: "b", Score: 1, Rejected: false, Output: outB},
				},
				Selected: &runmulti.RunResult{Variant: "a", Score: 0.5, Rejected: false, Output: outA},
			},
		},
		MemoryPromoter: p,
		Promotion:      &PromotionPolicy{MinPromoteConfidence: 0.99},
	}
	resp, err := svc.RunMulti(context.Background(), RunMultiRequest{
		Query:     "payment retry flow",
		Merge:     true,
		Promote:   true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.called {
		t.Fatalf("expected promoter not to be called when confidence gate fails")
	}
	if resp.Promoted {
		t.Fatalf("expected promoted=false")
	}
	if resp.Debug.PromotionDecision["rejection_code"] != RejectionCodeConfidenceBelowMinimum {
		t.Fatalf("rejection_code = %v", resp.Debug.PromotionDecision["rejection_code"])
	}
}

func TestService_RunMulti_observabilityIncludesMemoryAndContradictions(t *testing.T) {
	mem1 := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	mem2 := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	excluded := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	svc := &Service{
		RunMultiRunner: fakeRunMultiRunner{
			out: &runmulti.RunMultiResult{
				Runs: []runmulti.RunResult{
					{Variant: "a", Score: 0.7, Rejected: false, Output: "x"},
					{Variant: "b", Score: 1.2, Rejected: false, Output: "y"},
				},
				Selected: &runmulti.RunResult{Variant: "a", Score: 0.7, Rejected: false, Output: "x"},
			},
		},
		Compiler: &Compiler{
			Memory: fakeRunMultiMemorySearcher{list: []memory.MemoryObject{
				{ID: mem1, Kind: api.MemoryKindDecision, Statement: "d1", Authority: 8},
				{ID: mem2, Kind: api.MemoryKindConstraint, Statement: "c1", Authority: 7},
			}},
			Contradiction: fakeContradictionLister{ids: []uuid.UUID{mem1, excluded}},
		},
	}
	resp, err := svc.RunMulti(context.Background(), RunMultiRequest{
		Query:     "q",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.MemoriesUsed) == 0 {
		t.Fatalf("expected memories_used")
	}
	if len(resp.Contradictions) != 2 {
		t.Fatalf("expected 2 contradictions, got %d", len(resp.Contradictions))
	}
	foundExcluded := false
	for _, id := range resp.Excluded {
		if id == excluded.String() {
			foundExcluded = true
			break
		}
	}
	if !foundExcluded {
		t.Fatalf("expected excluded unresolved id %s, got %#v", excluded.String(), resp.Excluded)
	}
}

type captureRunMultiRunner struct {
	last runmulti.RunMultiInput
	out  *runmulti.RunMultiResult
}

func (c *captureRunMultiRunner) Run(ctx context.Context, in runmulti.RunMultiInput) (*runmulti.RunMultiResult, error) {
	c.last = in
	return c.out, nil
}

func TestService_RunMulti_mapsToRunnerInput(t *testing.T) {
	cap := &captureRunMultiRunner{
		out: &runmulti.RunMultiResult{
			Runs:     []runmulti.RunResult{{Variant: "balanced", Score: 0, Rejected: false}},
			Selected: &runmulti.RunResult{Variant: "balanced", Score: 0, Rejected: false},
		},
	}
	svc := &Service{RunMultiRunner: cap}
	ch := 5
	_, err := svc.RunMulti(context.Background(), RunMultiRequest{
		Query:                "query",
		RetrievalQuery:       "situation for compile-multi",
		Variants:             2,
		Strategy:             "failure_heavy",
		Tags:                 []string{"t1"},
		Symbols:              []string{"Sym"},
		MaxPerKind:           7,
		MaxTotal:             100,
		MaxTokens:            200,
		ChangedFilesCount:    &ch,
		SlowPathRequired:     true,
		SlowPathReasons:      []string{"r"},
		RecommendedExpansion: &RecommendedExpansion{ConstraintsDelta: 1, FailuresDelta: 2, PatternsDelta: 3},
		RepoRoot:             "/repo",
		LSPFocusPath:         "pkg/x.go",
		LSPFocusLine:         4,
		LSPFocusColumn:       2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cap.last.RetrievalQuery != "situation for compile-multi" {
		t.Fatalf("RetrievalQuery: %+v", cap.last)
	}
	if cap.last.Variants != 2 || cap.last.Strategy != "failure_heavy" {
		t.Fatalf("variants/strategy: %+v", cap.last)
	}
	if len(cap.last.Tags) != 1 || cap.last.Tags[0] != "t1" {
		t.Fatalf("tags: %+v", cap.last.Tags)
	}
	if len(cap.last.Symbols) != 1 || cap.last.Symbols[0] != "Sym" {
		t.Fatalf("symbols: %+v", cap.last.Symbols)
	}
	if cap.last.MaxPerKind != 7 || cap.last.MaxTotal != 100 || cap.last.MaxTokens != 200 {
		t.Fatalf("rie: %+v", cap.last)
	}
	if cap.last.CompileChangedFilesCount == nil || *cap.last.CompileChangedFilesCount != 5 {
		t.Fatalf("compile changed files: %+v", cap.last.CompileChangedFilesCount)
	}
	if !cap.last.SlowPathRequired {
		t.Fatal("slow path")
	}
	if cap.last.RecommendedExpansion == nil || cap.last.RecommendedExpansion.ConstraintsDelta != 1 {
		t.Fatalf("expansion: %+v", cap.last.RecommendedExpansion)
	}
	if cap.last.RepoRoot != "/repo" || cap.last.LSPFocusPath != "pkg/x.go" {
		t.Fatalf("LSP context: %+v", cap.last)
	}
	if cap.last.LSPFocusLine != 4 || cap.last.LSPFocusColumn != 2 {
		t.Fatalf("LSP focus position: %+v", cap.last)
	}
}

func TestService_RunMulti_orchestrationLSPRecallFlags(t *testing.T) {
	svc := &Service{
		RunMultiRunner: fakeRunMultiRunner{
			out: &runmulti.RunMultiResult{
				Runs:     []runmulti.RunResult{{Variant: "balanced", Score: 0, Rejected: false}},
				Selected: &runmulti.RunResult{Variant: "balanced", Score: 0, Rejected: false},
			},
		},
	}
	resp, err := svc.RunMulti(context.Background(), RunMultiRequest{
		Query:          "q",
		RepoRoot:       "/repo",
		LSPFocusPath:   "a.go",
		LSPFocusLine:   1,
		LSPFocusColumn: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	o := resp.Debug.Orchestration
	if o["lsp_recall_repo_root_set"] != true {
		t.Fatalf("lsp_recall_repo_root_set = %v", o["lsp_recall_repo_root_set"])
	}
	if o["lsp_recall_focus_path_set"] != true {
		t.Fatalf("lsp_recall_focus_path_set = %v", o["lsp_recall_focus_path_set"])
	}
	if o["lsp_recall_focus_position_set"] != true {
		t.Fatalf("lsp_recall_focus_position_set = %v", o["lsp_recall_focus_position_set"])
	}
	resp2, err := svc.RunMulti(context.Background(), RunMultiRequest{Query: "q"})
	if err != nil {
		t.Fatal(err)
	}
	o2 := resp2.Debug.Orchestration
	if o2["lsp_recall_repo_root_set"] != false || o2["lsp_recall_focus_path_set"] != false || o2["lsp_recall_focus_position_set"] != false {
		t.Fatalf("expected false LSP flags, got %+v", o2)
	}
}

func TestService_RunMulti_mergeDriftUsesSlowPathAndTagsWhenFlagged(t *testing.T) {
	var lastDrift []byte
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/drift/check", func(w http.ResponseWriter, r *http.Request) {
		lastDrift, _ = io.ReadAll(r.Body)
		_ = json.NewEncoder(w).Encode(runmulti.DriftResult{Passed: true, RiskLevel: "low"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := &Service{
		RunMultiRunner: fakeRunMultiRunner{
			out: &runmulti.RunMultiResult{
				Runs: []runmulti.RunResult{
					{Variant: "a", Score: 1, Rejected: false, Output: "Implement flow with steps.\n\n- Step one\n- Step two"},
					{Variant: "b", Score: 1.5, Rejected: false, Output: "Implement flow with steps.\n\n- Step one\n- Step three"},
				},
				Selected: &runmulti.RunResult{Variant: "a", Score: 1, Rejected: false, Output: "Implement flow with steps.\n\n- Step one\n- Step two"},
			},
		},
		RunMultiBaseURL: srv.URL,
	}
	_, err := svc.RunMulti(context.Background(), RunMultiRequest{
		Query:            "x",
		Merge:            true,
		SlowPathRequired: true,
		Tags:             []string{"api", "v1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var dec struct {
		SlowPathRequired bool     `json:"slow_path_required"`
		Tags             []string `json:"tags"`
	}
	if err := json.Unmarshal(lastDrift, &dec); err != nil {
		t.Fatalf("drift body: %v", err)
	}
	if !dec.SlowPathRequired {
		t.Errorf("merge drift: want slow_path_required true, got %+v", dec)
	}
	if len(dec.Tags) != 2 || dec.Tags[0] != "api" {
		t.Errorf("merge drift tags = %#v", dec.Tags)
	}
}

func TestService_RunMulti_mergeDriftOmitsSlowPathWithoutSignal(t *testing.T) {
	var lastDrift []byte
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/drift/check", func(w http.ResponseWriter, r *http.Request) {
		lastDrift, _ = io.ReadAll(r.Body)
		_ = json.NewEncoder(w).Encode(runmulti.DriftResult{Passed: true, RiskLevel: "low"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	svc := &Service{
		RunMultiRunner: fakeRunMultiRunner{
			out: &runmulti.RunMultiResult{
				Runs: []runmulti.RunResult{
					{Variant: "a", Score: 1, Rejected: false, Output: "Shared line.\n\n- a"},
					{Variant: "b", Score: 1.5, Rejected: false, Output: "Shared line.\n\n- b"},
				},
				Selected: &runmulti.RunResult{Variant: "a", Score: 1, Rejected: false, Output: "Shared line.\n\n- a"},
			},
		},
		RunMultiBaseURL: srv.URL,
	}
	_, err := svc.RunMulti(context.Background(), RunMultiRequest{
		Query: "x",
		Merge: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var dec struct {
		SlowPathRequired bool `json:"slow_path_required"`
	}
	if err := json.Unmarshal(lastDrift, &dec); err != nil {
		t.Fatal(err)
	}
	if dec.SlowPathRequired {
		t.Errorf("merge drift: want slow_path_required false by default, got %+v", dec)
	}
}
