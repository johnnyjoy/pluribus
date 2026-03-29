package recall

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

type fakeUsageReinforcer struct {
	called bool
	ids    []uuid.UUID
	meta   []memory.ReinforceMeta
}

func (f *fakeUsageReinforcer) ReinforceRecallUsage(ctx context.Context, ids []uuid.UUID) error {
	f.called = true
	f.ids = append([]uuid.UUID(nil), ids...)
	return nil
}

func (f *fakeUsageReinforcer) ReinforceRecallUsageWithMeta(ctx context.Context, ids []uuid.UUID, meta memory.ReinforceMeta) error {
	f.called = true
	f.ids = append([]uuid.UUID(nil), ids...)
	f.meta = append(f.meta, meta)
	return nil
}

func TestService_Compile_noCompiler(t *testing.T) {
	svc := &Service{Compiler: nil, Repo: nil}
	_, err := svc.Compile(context.Background(), CompileRequest{})
	if err != ErrNoCompiler {
		t.Errorf("got err %v, want ErrNoCompiler", err)
	}
}

func TestService_Compile_tagsAndRetrievalOnlyNoCorrelationUUIDs(t *testing.T) {
	memID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	objs := []memory.MemoryObject{
		{ID: memID, Kind: api.MemoryKindConstraint, Statement: "never skip tests", Authority: 7, UpdatedAt: time.Now()},
	}
	compiler := &Compiler{
		Memory: &fakeMemorySearcher{objs: objs},
	}
	svc := &Service{Compiler: compiler}
	bundle, err := svc.Compile(context.Background(), CompileRequest{
		Tags:           []string{"go"},
		RetrievalQuery: "shipping feature safely",
		MaxPerKind:     5,
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if bundle == nil {
		t.Fatal("nil bundle")
	}
	if len(bundle.GoverningConstraints) == 0 {
		t.Fatalf("expected at least one constraint from fake memory, got %+v", bundle.GoverningConstraints)
	}
}

func TestService_Compile_reinforcesBundleMemoryUsage(t *testing.T) {
	memID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	objs := []memory.MemoryObject{
		{ID: memID, Kind: api.MemoryKindPattern, Statement: "p", Authority: 6, UpdatedAt: time.Now()},
	}
	compiler := &Compiler{
		Memory: &fakeMemorySearcher{objs: objs},
	}
	r := &fakeUsageReinforcer{}
	svc := &Service{Compiler: compiler, UsageReinforcer: r}

	_, err := svc.Compile(context.Background(), CompileRequest{
		RetrievalQuery: "test situation",
		MaxPerKind: 5,
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if !r.called {
		t.Fatal("expected usage reinforcer to be called")
	}
	found := false
	for _, id := range r.ids {
		if id == memID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected reinforced ids to include %s, got %v", memID, r.ids)
	}
	if len(r.meta) == 0 || r.meta[0].Impact != "low" || r.meta[0].Reason != "reuse_recall" {
		t.Fatalf("expected low-impact reuse_recall meta, got %+v", r.meta)
	}
}

func TestService_Preflight(t *testing.T) {
	svc := &Service{}
	got := svc.Preflight(context.Background(), PreflightRequest{ChangedFilesCount: 15})
	if got.RiskLevel != "high" {
		t.Errorf("RiskLevel = %q, want high", got.RiskLevel)
	}
	if len(got.RequiredActions) != 2 {
		t.Errorf("RequiredActions = %v", got.RequiredActions)
	}
}

func TestService_Preflight_slowPathRequiredWhenHighRisk(t *testing.T) {
	svc := &Service{
		SlowPath: &SlowPathPreflightConfig{
			Enabled:             true,
			HighRiskThreshold:   1.0,
			ExpandConstraintsBy: 4,
			ExpandFailuresBy:    4,
			ExpandPatternsBy:    2,
		},
	}
	got := svc.Preflight(context.Background(), PreflightRequest{ChangedFilesCount: 11})
	if !got.SlowPathRequired {
		t.Error("SlowPathRequired = false, want true")
	}
	if len(got.SlowPathReasons) == 0 {
		t.Error("SlowPathReasons empty, want at least one reason")
	}
	if got.RecommendedExpansion == nil {
		t.Fatal("RecommendedExpansion = nil, want set")
	}
	if got.RecommendedExpansion.ConstraintsDelta != 4 || got.RecommendedExpansion.FailuresDelta != 4 || got.RecommendedExpansion.PatternsDelta != 2 {
		t.Errorf("RecommendedExpansion = %+v", got.RecommendedExpansion)
	}
}

func TestService_Preflight_slowPathNotRequiredWhenLowRisk(t *testing.T) {
	svc := &Service{
		SlowPath: &SlowPathPreflightConfig{Enabled: true, HighRiskThreshold: 1.0},
	}
	got := svc.Preflight(context.Background(), PreflightRequest{ChangedFilesCount: 2})
	if got.SlowPathRequired {
		t.Error("SlowPathRequired = true, want false for low risk")
	}
	if got.RecommendedExpansion != nil {
		t.Errorf("RecommendedExpansion = %+v, want nil", got.RecommendedExpansion)
	}
}

func TestService_Preflight_slowPathNilNoSlowPathFields(t *testing.T) {
	svc := &Service{SlowPath: nil}
	got := svc.Preflight(context.Background(), PreflightRequest{ChangedFilesCount: 11})
	if got.SlowPathRequired {
		t.Error("SlowPathRequired = true when SlowPath config nil, want false")
	}
}

func TestService_CompileMulti_noCompiler(t *testing.T) {
	svc := &Service{Compiler: nil}
	_, err := svc.CompileMulti(context.Background(), CompileMultiRequest{})
	if err != ErrNoCompiler {
		t.Errorf("got err %v, want ErrNoCompiler", err)
	}
}

func TestService_CompileMulti_returns_three_distinct_bundles(t *testing.T) {
	var objs []memory.MemoryObject
	for i := 0; i < 4; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindConstraint, Statement: "c", Authority: i, UpdatedAt: time.Now()})
	}
	for i := 0; i < 4; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindDecision, Statement: "d", Authority: i, UpdatedAt: time.Now()})
	}
	for i := 0; i < 8; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindFailure, Statement: "f", Authority: i, UpdatedAt: time.Now()})
	}
	for i := 0; i < 6; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "p", Authority: i, UpdatedAt: time.Now()})
	}
	for i := 0; i < 8; i++ {
		objs = append(objs, memory.MemoryObject{
			ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "ol", Authority: i, UpdatedAt: time.Now(),
			Payload: json.RawMessage(`{"polarity":"negative","experience":"x","decision":"x","outcome":"x","impact":{"severity":"high"},"directive":"x"}`),
		})
	}
	fakeMemory := &fakeMemorySearcher{objs: objs}
	weights := DefaultRankingWeights()
	compiler := &Compiler{Memory: fakeMemory, Ranking: &weights}
	svc := &Service{Compiler: compiler}

	req := CompileMultiRequest{
		RetrievalQuery: "test situation",
		Variants:  3,
		Strategy:  "default",
	}
	resp, err := svc.CompileMulti(context.Background(), req)
	if err != nil {
		t.Fatalf("CompileMulti: %v", err)
	}
	if len(resp.Bundles) != 3 {
		t.Fatalf("len(Bundles) = %d, want 3", len(resp.Bundles))
	}
	names := make(map[string]bool)
	for _, b := range resp.Bundles {
		names[b.Variant] = true
	}
	for _, want := range []string{StrategyBalanced, StrategyFailureHeavy, StrategyAuthorityHeavy} {
		if !names[want] {
			t.Errorf("missing variant %q in bundles", want)
		}
	}
	var balanced, failureHeavy *RecallBundle
	for i := range resp.Bundles {
		if resp.Bundles[i].Variant == StrategyBalanced {
			balanced = &resp.Bundles[i].Bundle
		}
		if resp.Bundles[i].Variant == StrategyFailureHeavy {
			failureHeavy = &resp.Bundles[i].Bundle
		}
	}
	if balanced == nil || failureHeavy == nil {
		t.Fatal("balanced or failure_heavy bundle missing")
	}
	// failure_heavy: more failures, fewer patterns than balanced (maxPerKind=5 → 7 failures, 3 patterns).
	if len(failureHeavy.KnownFailures) < len(balanced.KnownFailures) {
		t.Errorf("failure_heavy KnownFailures = %d, want >= balanced %d", len(failureHeavy.KnownFailures), len(balanced.KnownFailures))
	}
	if len(failureHeavy.ApplicablePatterns) > len(balanced.ApplicablePatterns) {
		t.Errorf("failure_heavy ApplicablePatterns = %d, want <= balanced %d", len(failureHeavy.ApplicablePatterns), len(balanced.ApplicablePatterns))
	}
}

// compile-multi forwards slow-path into each variant’s Compile; single balanced variant matches expanded limits.
func TestService_CompileMulti_slowPathExpansionBalanced(t *testing.T) {
	var objs []memory.MemoryObject
	for i := 0; i < 10; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindConstraint, Statement: "c", Authority: i, UpdatedAt: time.Now()})
	}
	for i := 0; i < 5; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindDecision, Statement: "d", Authority: i, UpdatedAt: time.Now()})
	}
	for i := 0; i < 10; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindFailure, Statement: "f", Authority: i, UpdatedAt: time.Now()})
	}
	for i := 0; i < 8; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "p", Authority: i, UpdatedAt: time.Now()})
	}
	for i := 0; i < 10; i++ {
		objs = append(objs, memory.MemoryObject{
			ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "ol", Authority: i, UpdatedAt: time.Now(),
			Payload: json.RawMessage(`{"polarity":"negative","experience":"x","decision":"x","outcome":"x","impact":{"severity":"high"},"directive":"x"}`),
		})
	}
	fakeMemory := &fakeMemorySearcher{objs: objs}
	compiler := &Compiler{Memory: fakeMemory}
	svc := &Service{Compiler: compiler}

	req := CompileMultiRequest{
		RetrievalQuery: "test situation",
		Variants:         1,
		Strategy:         "default",
		MaxPerKind:       5,
		SlowPathRequired: true,
		SlowPathReasons:  []string{"risk score exceeds threshold"},
		RecommendedExpansion: &RecommendedExpansion{
			ConstraintsDelta: 4, FailuresDelta: 4, PatternsDelta: 2,
		},
	}
	resp, err := svc.CompileMulti(context.Background(), req)
	if err != nil {
		t.Fatalf("CompileMulti: %v", err)
	}
	if len(resp.Bundles) != 1 || resp.Bundles[0].Variant != StrategyBalanced {
		t.Fatalf("bundles = %+v, want single balanced", resp.Bundles)
	}
	b := resp.Bundles[0].Bundle
	if len(b.GoverningConstraints) != 9 || len(b.Decisions) != 5 || len(b.KnownFailures) != 9 || len(b.ApplicablePatterns) != 7 {
		t.Errorf("counts c=%d d=%d f=%d p=%d want 9,5,9,7",
			len(b.GoverningConstraints), len(b.Decisions), len(b.KnownFailures), len(b.ApplicablePatterns))
	}
	if !b.SlowPathApplied {
		t.Error("SlowPathApplied = false, want true")
	}
	if len(b.SlowPathReasons) != 1 || b.SlowPathReasons[0] != "risk score exceeds threshold" {
		t.Errorf("SlowPathReasons = %v", b.SlowPathReasons)
	}
	if b.BaseLimits == nil || b.BaseLimits.Constraints != 5 {
		t.Errorf("BaseLimits = %+v", b.BaseLimits)
	}
	if b.ExpandedLimits == nil || b.ExpandedLimits.Constraints != 9 || b.ExpandedLimits.Failures != 9 || b.ExpandedLimits.Patterns != 7 {
		t.Errorf("ExpandedLimits = %+v", b.ExpandedLimits)
	}
}

// All default variants receive slow-path metadata when requested (variant modifiers may change effective per-bucket limits).
func TestService_CompileMulti_slowPathMetadataAllVariants(t *testing.T) {
	var objs []memory.MemoryObject
	for i := 0; i < 6; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindConstraint, Statement: "c", Authority: i, UpdatedAt: time.Now()})
	}
	for i := 0; i < 4; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindDecision, Statement: "d", Authority: i, UpdatedAt: time.Now()})
	}
	for i := 0; i < 8; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindFailure, Statement: "f", Authority: i, UpdatedAt: time.Now()})
	}
	for i := 0; i < 6; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "p", Authority: i, UpdatedAt: time.Now()})
	}
	for i := 0; i < 8; i++ {
		objs = append(objs, memory.MemoryObject{
			ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "ol", Authority: i, UpdatedAt: time.Now(),
			Payload: json.RawMessage(`{"polarity":"negative","experience":"x","decision":"x","outcome":"x","impact":{"severity":"high"},"directive":"x"}`),
		})
	}
	fakeMemory := &fakeMemorySearcher{objs: objs}
	compiler := &Compiler{Memory: fakeMemory}
	svc := &Service{Compiler: compiler}

	req := CompileMultiRequest{
		RetrievalQuery: "test situation",
		Variants:             3,
		Strategy:             "default",
		SlowPathRequired:     true,
		SlowPathReasons:      []string{"high risk"},
		RecommendedExpansion: &RecommendedExpansion{ConstraintsDelta: 1, FailuresDelta: 1, PatternsDelta: 1},
	}
	resp, err := svc.CompileMulti(context.Background(), req)
	if err != nil {
		t.Fatalf("CompileMulti: %v", err)
	}
	if len(resp.Bundles) != 3 {
		t.Fatalf("len(Bundles) = %d, want 3", len(resp.Bundles))
	}
	for _, vb := range resp.Bundles {
		if !vb.Bundle.SlowPathApplied {
			t.Errorf("variant %q: SlowPathApplied = false", vb.Variant)
		}
		if vb.Bundle.BaseLimits == nil || vb.Bundle.ExpandedLimits == nil {
			t.Errorf("variant %q: missing base/expanded limits", vb.Variant)
		}
	}
}

func TestService_CompileMulti_changedFilesCountInfersSlowPath(t *testing.T) {
	objs := []memory.MemoryObject{
		{ID: uuid.New(), Kind: api.MemoryKindConstraint, Statement: "c", Authority: 1, UpdatedAt: time.Now()},
	}
	compiler := &Compiler{
		Memory: &fakeMemorySearcher{objs: objs},
	}
	svc := &Service{
		Compiler: compiler,
		SlowPath: &SlowPathPreflightConfig{
			Enabled: true, HighRiskThreshold: 1.0,
			ExpandConstraintsBy: 4, ExpandFailuresBy: 4, ExpandPatternsBy: 2,
		},
	}
	n := 11
	resp, err := svc.CompileMulti(context.Background(), CompileMultiRequest{
		Variants: 1, Strategy: "default", MaxPerKind: 5,
		ChangedFilesCount: intPtr(n),
	})
	if err != nil {
		t.Fatalf("CompileMulti: %v", err)
	}
	if len(resp.Bundles) != 1 {
		t.Fatalf("bundles %d, want 1", len(resp.Bundles))
	}
	if !resp.Bundles[0].Bundle.SlowPathApplied {
		t.Error("expected slow path inferred from changed_files_count via Preflight")
	}
}

func TestService_CompileMulti_extraVariantsWhenSlow(t *testing.T) {
	objs := []memory.MemoryObject{{ID: uuid.New(), Kind: api.MemoryKindConstraint, Statement: "c", Authority: 1, UpdatedAt: time.Now()}}
	compiler := &Compiler{
		Memory: &fakeMemorySearcher{objs: objs},
	}
	svc := &Service{
		Compiler: compiler,
		SlowPath: &SlowPathPreflightConfig{
			Enabled: true, HighRiskThreshold: 1.0,
			ExpandConstraintsBy: 1, ExpandFailuresBy: 1, ExpandPatternsBy: 1,
			ExtraVariantsWhenSlow: 2,
		},
	}
	n := 11
	resp, err := svc.CompileMulti(context.Background(), CompileMultiRequest{
		Variants:          1,
		Strategy:          "default",
		ChangedFilesCount: intPtr(n),
	})
	if err != nil {
		t.Fatalf("CompileMulti: %v", err)
	}
	if len(resp.Bundles) != 3 {
		t.Errorf("want 3 bundles (1 + extra_variants_when_slow, capped at 3), got %d", len(resp.Bundles))
	}
}
