package recall

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/internal/memorynorm"
	"control-plane/internal/tooling"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

func TestCompiler_Compile_bundleStructureAndMaxPerKind(t *testing.T) {
	// Fake memory: return 2 constraints, 3 decisions, 6 failures, 1 pattern
	fakeMemory := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: uuid.New(), Kind: api.MemoryKindConstraint, Statement: "c1", Authority: 1},
			{ID: uuid.New(), Kind: api.MemoryKindConstraint, Statement: "c2", Authority: 2},
			{ID: uuid.New(), Kind: api.MemoryKindDecision, Statement: "d1", Authority: 1},
			{ID: uuid.New(), Kind: api.MemoryKindDecision, Statement: "d2", Authority: 2},
			{ID: uuid.New(), Kind: api.MemoryKindDecision, Statement: "d3", Authority: 3},
			{ID: uuid.New(), Kind: api.MemoryKindFailure, Statement: "f1", Authority: 1},
			{ID: uuid.New(), Kind: api.MemoryKindFailure, Statement: "f2", Authority: 2},
			{ID: uuid.New(), Kind: api.MemoryKindFailure, Statement: "f3", Authority: 3},
			{ID: uuid.New(), Kind: api.MemoryKindFailure, Statement: "f4", Authority: 4},
			{ID: uuid.New(), Kind: api.MemoryKindFailure, Statement: "f5", Authority: 5},
			{ID: uuid.New(), Kind: api.MemoryKindFailure, Statement: "f6", Authority: 6},
			{ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "p1", Authority: 1},
		},
	}

	compiler := &Compiler{Memory: fakeMemory}
	req := CompileRequest{RetrievalQuery: "test situation", MaxPerKind: 2}
	bundle, err := compiler.Compile(context.Background(), req)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	if len(bundle.GoverningConstraints) != 2 {
		t.Errorf("GoverningConstraints len = %d, want 2 (max per kind)", len(bundle.GoverningConstraints))
	}
	if len(bundle.Decisions) != 2 {
		t.Errorf("Decisions len = %d, want 2 (max per kind)", len(bundle.Decisions))
	}
	if len(bundle.KnownFailures) != 2 {
		t.Errorf("KnownFailures len = %d, want 2 (max per kind)", len(bundle.KnownFailures))
	}
	if len(bundle.ApplicablePatterns) != 1 {
		t.Errorf("ApplicablePatterns len = %d, want 1", len(bundle.ApplicablePatterns))
	}
	if bundle.SlowPathApplied || bundle.BaseLimits != nil || bundle.ExpandedLimits != nil {
		t.Error("normal compile should not set slow-path metadata")
	}
}

func TestCompiler_Compile_slowPathExpansionAndMetadata(t *testing.T) {
	// Enough items so that normal trim gives 2 each, slow-path gives 2+4=6 constraints, 2+4=6 failures, 2+2=4 patterns, 2 decisions
	var objs []memory.MemoryObject
	for i := 0; i < 8; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindConstraint, Statement: "c", Authority: i})
	}
	for i := 0; i < 4; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindDecision, Statement: "d", Authority: i})
	}
	for i := 0; i < 8; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindFailure, Statement: "f", Authority: i})
	}
	for i := 0; i < 6; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "p", Authority: i})
	}
	for i := 0; i < 8; i++ {
		objs = append(objs, memory.MemoryObject{
			ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "ol", Authority: i,
			Payload: json.RawMessage(`{"polarity":"negative","experience":"x","decision":"x","outcome":"x","impact":{"severity":"high"},"directive":"x"}`),
		})
	}
	fakeMemory := &fakeMemorySearcher{objs: objs}
	compiler := &Compiler{Memory: fakeMemory}

	// Normal: maxPerKind 2 -> 2 per bucket
	normalReq := CompileRequest{RetrievalQuery: "test situation", MaxPerKind: 2}
	normalBundle, err := compiler.Compile(context.Background(), normalReq)
	if err != nil {
		t.Fatalf("normal Compile: %v", err)
	}
	if len(normalBundle.GoverningConstraints) != 2 || len(normalBundle.KnownFailures) != 2 || len(normalBundle.ApplicablePatterns) != 2 {
		t.Errorf("normal: constraints=%d failures=%d patterns=%d, want 2 each", len(normalBundle.GoverningConstraints), len(normalBundle.KnownFailures), len(normalBundle.ApplicablePatterns))
	}
	if normalBundle.SlowPathApplied || normalBundle.BaseLimits != nil {
		t.Error("normal bundle should not have slow-path metadata")
	}

	// Slow-path: maxPerKind 2 + expansion 4,4,2 -> 6 constraints, 2 decisions, 6 failures, 4 patterns
	slowReq := CompileRequest{RetrievalQuery: "test situation", MaxPerKind: 2,
		SlowPathRequired:     true,
		SlowPathReasons:      []string{"risk score exceeds threshold"},
		RecommendedExpansion: &RecommendedExpansion{ConstraintsDelta: 4, FailuresDelta: 4, PatternsDelta: 2},
	}
	slowBundle, err := compiler.Compile(context.Background(), slowReq)
	if err != nil {
		t.Fatalf("slow-path Compile: %v", err)
	}
	if len(slowBundle.GoverningConstraints) != 6 {
		t.Errorf("slow-path GoverningConstraints = %d, want 6", len(slowBundle.GoverningConstraints))
	}
	if len(slowBundle.Decisions) != 2 {
		t.Errorf("slow-path Decisions = %d, want 2 (no delta)", len(slowBundle.Decisions))
	}
	if len(slowBundle.KnownFailures) != 6 {
		t.Errorf("slow-path KnownFailures = %d, want 6", len(slowBundle.KnownFailures))
	}
	if len(slowBundle.ApplicablePatterns) != 4 {
		t.Errorf("slow-path ApplicablePatterns = %d, want 4", len(slowBundle.ApplicablePatterns))
	}
	if !slowBundle.SlowPathApplied {
		t.Error("SlowPathApplied = false, want true")
	}
	if len(slowBundle.SlowPathReasons) != 1 || slowBundle.SlowPathReasons[0] != "risk score exceeds threshold" {
		t.Errorf("SlowPathReasons = %v", slowBundle.SlowPathReasons)
	}
	if slowBundle.BaseLimits == nil || slowBundle.BaseLimits.Constraints != 2 || slowBundle.BaseLimits.Failures != 2 {
		t.Errorf("BaseLimits = %+v", slowBundle.BaseLimits)
	}
	if slowBundle.ExpandedLimits == nil || slowBundle.ExpandedLimits.Constraints != 6 || slowBundle.ExpandedLimits.Failures != 6 || slowBundle.ExpandedLimits.Patterns != 4 {
		t.Errorf("ExpandedLimits = %+v", slowBundle.ExpandedLimits)
	}
}

type fakeMemorySearcher struct {
	objs []memory.MemoryObject
	// LastSearch is set on each Search call (for assertions that compile does not vary search by agent_id).
	LastSearch memory.SearchRequest
}

func (f *fakeMemorySearcher) Search(ctx context.Context, req memory.SearchRequest) ([]memory.MemoryObject, error) {
	f.LastSearch = req
	return f.objs, nil
}

func (f *fakeMemorySearcher) SearchMemories(ctx context.Context, req memory.MemoriesSearchRequest) ([]memory.MemoryObject, error) {
	// For compiler unit tests, keep deterministic: Search and SearchMemories return the same fixed objects.
	return f.objs, nil
}

func TestCompiler_Compile_agentIDDoesNotAffectMemorySearch(t *testing.T) {
	fakeMemory := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: uuid.New(), Kind: api.MemoryKindConstraint, Statement: "c1", Authority: 1},
		},
	}
	compiler := &Compiler{Memory: fakeMemory}
	base := CompileRequest{RetrievalQuery: "test situation", MaxPerKind: 2}
	_, err := compiler.Compile(context.Background(), base)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	first := fakeMemory.LastSearch
	withAgent := base
	withAgent.AgentID = "recall-agent-1"
	_, err = compiler.Compile(context.Background(), withAgent)
	if err != nil {
		t.Fatalf("Compile with agent_id: %v", err)
	}
	second := fakeMemory.LastSearch
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("SearchRequest should match with/without agent_id: %+v vs %+v", first, second)
	}
}

func TestMergeUniqueMemoryObjects(t *testing.T) {
	a := uuid.New()
	b := uuid.New()
	c := uuid.New()
	p := []memory.MemoryObject{{ID: a, Statement: "1"}, {ID: b, Statement: "2"}}
	e := []memory.MemoryObject{{ID: b, Statement: "dup"}, {ID: c, Statement: "3"}}
	got := mergeUniqueMemoryObjects(p, e)
	if len(got) != 3 {
		t.Fatalf("len=%d want 3", len(got))
	}
	if got[2].ID != c {
		t.Fatalf("third id")
	}
}

// Recall uses a single unscoped memory search; constraints appear when the searcher returns them.
func TestCompiler_Compile_includesConstraintFromSearchResults(t *testing.T) {
	constraintID := uuid.MustParse("b0000000-0000-0000-0000-000000000001")
	localConstraint := memory.MemoryObject{
		ID: constraintID, Kind: api.MemoryKindConstraint, Statement: "binding rule", Authority: 8,
		Applicability: api.ApplicabilityGoverning, UpdatedAt: time.Now(),
	}
	fake := &fakeMemorySearcher{objs: []memory.MemoryObject{localConstraint}}
	weights := DefaultRankingWeights()
	compiler := &Compiler{
		Memory:  fake,
		Ranking: &weights,
		RIU:     &RIUConfig{Enabled: true, Weights: DefaultRIUWeights()},
	}
	bundle, err := compiler.Compile(context.Background(), CompileRequest{RetrievalQuery: "test situation", MaxPerKind: 5})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	var found bool
	for _, it := range bundle.GoverningConstraints {
		if it.ID == constraintID.String() {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("want constraint in GoverningConstraints; got %#v", bundle.GoverningConstraints)
	}
}

func TestCompiler_Compile_weightedRankingOrder(t *testing.T) {
	now := time.Now()

	// Two decisions: same authority; one has matching tag "api", one doesn't. With ranking, "api" one should appear first.
	idLow := uuid.New()
	idHigh := uuid.New()
	fakeMemory := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: idLow, Kind: api.MemoryKindDecision, Statement: "decision-no-tag", Authority: 5, UpdatedAt: now, Tags: []string{"other"}},
			{ID: idHigh, Kind: api.MemoryKindDecision, Statement: "decision-with-api", Authority: 5, UpdatedAt: now, Tags: []string{"api"}},
		},
	}
	weights := DefaultRankingWeights()
	compiler := &Compiler{
		Memory:  fakeMemory,
		Ranking: &weights,
	}
	req := CompileRequest{RetrievalQuery: "test situation", Tags: []string{"api"},
		MaxPerKind: 2,
	}
	bundle, err := compiler.Compile(context.Background(), req)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(bundle.Decisions) != 2 {
		t.Fatalf("Decisions len = %d, want 2", len(bundle.Decisions))
	}
	// First decision should be the one with tag "api" (higher score).
	if bundle.Decisions[0].Statement != "decision-with-api" {
		t.Errorf("first Decision = %q, want decision-with-api (ranked by tag match)", bundle.Decisions[0].Statement)
	}
	// Justification must be present when ranking is on (Task 73).
	for i, d := range bundle.Decisions {
		if d.Justification == nil {
			t.Errorf("Decision[%d] missing justification (RIE)", i)
		}
		if d.Justification != nil && d.Justification.Score <= 0 {
			t.Errorf("Decision[%d] justification score = %v, want positive", i, d.Justification.Score)
		}
	}
}

// fakeContradictionExclusion returns fixed memory IDs to exclude from recall (Task 78).
// Optional pairs triggers Phase E winner resolution (creative C3) when non-empty.
type fakeContradictionExclusion struct {
	excluded []uuid.UUID
	pairs    [][2]uuid.UUID
}

func (f *fakeContradictionExclusion) ListMemoryIDsInUnresolved(ctx context.Context) ([]uuid.UUID, error) {
	return f.excluded, nil
}

func (f *fakeContradictionExclusion) ListUnresolvedPairs(ctx context.Context, limit int) ([][2]uuid.UUID, error) {
	if len(f.pairs) > 0 {
		if limit > 0 && len(f.pairs) > limit {
			return f.pairs[:limit], nil
		}
		return f.pairs, nil
	}
	return nil, nil
}

func TestCompiler_Compile_excludesUnresolvedContradictions(t *testing.T) {
	excludedID := uuid.MustParse("e0000000-0000-0000-0000-000000000001")
	includedID := uuid.MustParse("e0000000-0000-0000-0000-000000000002")
	now := time.Now()

	fakeMemory := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: excludedID, Kind: api.MemoryKindConstraint, Statement: "excluded constraint", Authority: 5,
				UpdatedAt: now, Status: api.StatusActive},
			{ID: includedID, Kind: api.MemoryKindConstraint, Statement: "included constraint", Authority: 9,
				UpdatedAt: now, Status: api.StatusActive},
		},
	}
	fakeContradiction := &fakeContradictionExclusion{
		excluded: []uuid.UUID{excludedID, includedID},
		pairs:    [][2]uuid.UUID{{excludedID, includedID}},
	}
	compiler := &Compiler{
		Memory:        fakeMemory,
		Contradiction: fakeContradiction,
	}
	req := CompileRequest{RetrievalQuery: "test situation", MaxPerKind: 5}
	bundle, err := compiler.Compile(context.Background(), req)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	// Only the non-excluded constraint should appear.
	if len(bundle.GoverningConstraints) != 1 {
		t.Errorf("GoverningConstraints len = %d, want 1 (one excluded by contradiction)", len(bundle.GoverningConstraints))
	}
	if len(bundle.GoverningConstraints) > 0 && bundle.GoverningConstraints[0].ID == excludedID.String() {
		t.Errorf("excluded memory should not appear in bundle, got ID %s", bundle.GoverningConstraints[0].ID)
	}
	if len(bundle.GoverningConstraints) > 0 && bundle.GoverningConstraints[0].Statement != "included constraint" {
		t.Errorf("GoverningConstraints[0].Statement = %q, want included constraint", bundle.GoverningConstraints[0].Statement)
	}
	if len(bundle.ConflictResolutions) != 1 {
		t.Fatalf("ConflictResolutions = %d, want 1", len(bundle.ConflictResolutions))
	}
	if bundle.ConflictResolutions[0].WinnerMemoryID != includedID.String() || bundle.ConflictResolutions[0].LoserMemoryID != excludedID.String() {
		t.Fatalf("ConflictResolutions[0] = %+v, want winner=%s loser=%s", bundle.ConflictResolutions[0], includedID, excludedID)
	}
}

func TestCompiler_Compile_recallCollapseExactStatementKey(t *testing.T) {
	stmt := "duplicate governing constraint text"
	sk := memorynorm.StatementKey(stmt)
	canon := memorynorm.StatementCanonical(stmt)
	now := time.Now()
	idHi := uuid.MustParse("e0000000-0000-0000-0000-000000000021")
	idLo := uuid.MustParse("e0000000-0000-0000-0000-000000000022")
	fakeMemory := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: idHi, Kind: api.MemoryKindConstraint, Statement: stmt, StatementCanonical: canon, StatementKey: sk,
				Authority: 9, UpdatedAt: now, Status: api.StatusActive},
			{ID: idLo, Kind: api.MemoryKindConstraint, Statement: stmt, StatementCanonical: canon, StatementKey: sk,
				Authority: 3, UpdatedAt: now, Status: api.StatusActive},
		},
	}
	weights := DefaultRankingWeights()
	compiler := &Compiler{
		Memory:  fakeMemory,
		Ranking: &weights,
	}
	bundle, err := compiler.Compile(context.Background(), CompileRequest{RetrievalQuery: "test situation", MaxPerKind: 5})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(bundle.GoverningConstraints) != 1 {
		t.Fatalf("GoverningConstraints = %d, want 1 after exact-key collapse", len(bundle.GoverningConstraints))
	}
	if bundle.GoverningConstraints[0].ID != idHi.String() {
		t.Fatalf("want higher-authority row kept, got %s", bundle.GoverningConstraints[0].ID)
	}
}

func TestCompiler_Compile_contradictionLegacyExcludeWhenNoPairs(t *testing.T) {
	excludedID := uuid.MustParse("e0000000-0000-0000-0000-000000000011")
	includedID := uuid.MustParse("e0000000-0000-0000-0000-000000000012")
	now := time.Now()
	fakeMemory := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: excludedID, Kind: api.MemoryKindConstraint, Statement: "low", Authority: 3,
				UpdatedAt: now, Status: api.StatusActive},
			{ID: includedID, Kind: api.MemoryKindConstraint, Statement: "high", Authority: 9,
				UpdatedAt: now, Status: api.StatusActive},
		},
	}
	// Only flat list — no pairs — legacy excludes every ID in unresolved list.
	fakeContradiction := &fakeContradictionExclusion{excluded: []uuid.UUID{excludedID}}
	compiler := &Compiler{
		Memory:        fakeMemory,
		Contradiction: fakeContradiction,
	}
	bundle, err := compiler.Compile(context.Background(), CompileRequest{RetrievalQuery: "test situation", MaxPerKind: 5})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(bundle.GoverningConstraints) != 1 || bundle.GoverningConstraints[0].ID != includedID.String() {
		t.Fatalf("want only included constraint, got %+v", bundle.GoverningConstraints)
	}
	if len(bundle.ConflictResolutions) != 0 {
		t.Fatalf("legacy path should not set ConflictResolutions, got %d", len(bundle.ConflictResolutions))
	}
}

func TestCompiler_Compile_RIELimitsMaxTotal(t *testing.T) {
	now := time.Now()

	fakeMemory := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: uuid.New(), Kind: api.MemoryKindConstraint, Statement: "c1", Authority: 1, UpdatedAt: now},
			{ID: uuid.New(), Kind: api.MemoryKindConstraint, Statement: "c2", Authority: 2, UpdatedAt: now},
			{ID: uuid.New(), Kind: api.MemoryKindDecision, Statement: "d1", Authority: 1, UpdatedAt: now},
			{ID: uuid.New(), Kind: api.MemoryKindDecision, Statement: "d2", Authority: 2, UpdatedAt: now},
		},
	}
	compiler := &Compiler{Memory: fakeMemory}
	req := CompileRequest{RetrievalQuery: "test situation", MaxPerKind: 5,
		MaxTotal: 3,
	}
	bundle, err := compiler.Compile(context.Background(), req)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	total := len(bundle.GoverningConstraints) + len(bundle.Decisions) + len(bundle.KnownFailures) + len(bundle.ApplicablePatterns)
	if total != 3 {
		t.Errorf("MaxTotal=3: got total %d items, want 3", total)
	}
}

// TestCompiler_Compile_patternPayloadBucket verifies pattern payload memories are in patterns bucket.
func TestCompiler_Compile_objectLessonBucket(t *testing.T) {
	payload := memory.PatternPayload{
		Polarity:   "positive",
		Experience: "Documented API first",
		Decision:   "Keep OpenAPI in sync",
		Outcome:    "Fewer bugs",
		Impact:     memory.PatternImpact{Severity: "medium"},
		Directive:  "Update OpenAPI on every change.",
	}
	payloadBytes, _ := json.Marshal(payload)
	fakeMemory := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "Update OpenAPI on every change.", Authority: 5, Payload: payloadBytes},
		},
	}
	compiler := &Compiler{Memory: fakeMemory}
	req := CompileRequest{RetrievalQuery: "test situation", MaxPerKind: 5}
	bundle, err := compiler.Compile(context.Background(), req)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(bundle.ApplicablePatterns) != 1 {
		t.Fatalf("ApplicablePatterns len = %d, want 1", len(bundle.ApplicablePatterns))
	}
	if bundle.ApplicablePatterns[0].Statement != "Update OpenAPI on every change." || bundle.ApplicablePatterns[0].Kind != "pattern" {
		t.Errorf("ApplicablePatterns[0] = %+v", bundle.ApplicablePatterns[0])
	}
}

// TestCompiler_Compile_severeNegativePatternOutranksFailure verifies a high-severity negative pattern dominates over generic failure when limits apply.
func TestCompiler_Compile_severeNegativeOutranksFailure(t *testing.T) {
	now := time.Now()
	payload := memory.PatternPayload{
		Polarity:   "negative",
		Experience: "Deployed without tests",
		Decision:   "Require tests",
		Outcome:    "Regression",
		Impact:     memory.PatternImpact{Severity: "high"},
		Directive:  "Never skip tests.",
	}
	payloadBytes, _ := json.Marshal(payload)
	fakeMemory := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: uuid.New(), Kind: api.MemoryKindFailure, Statement: "generic failure", Authority: 5, UpdatedAt: now},
			{ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "Never skip tests.", Authority: 3, UpdatedAt: now, Payload: payloadBytes},
		},
	}
	weights := DefaultRankingWeights()
	compiler := &Compiler{Memory: fakeMemory, Ranking: &weights}
	req := CompileRequest{RetrievalQuery: "test situation", MaxPerKind: 5,
		MaxTotal: 1,
	}
	bundle, err := compiler.Compile(context.Background(), req)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(bundle.ApplicablePatterns) > 0 && bundle.ApplicablePatterns[0].Statement != "Never skip tests." {
		t.Errorf("ApplicablePatterns[0].Statement = %q, want Never skip tests.", bundle.ApplicablePatterns[0].Statement)
	}
}

// TestCompiler_Compile_symbolDiagnosticsInBundle verifies Task 101: bundle gets MatchedSymbols and SymbolRelevanceReason when task symbols overlap object-lesson symbols.
func TestCompiler_Compile_symbolDiagnosticsInBundle(t *testing.T) {
	payload := memory.PatternPayload{
		Polarity:   "negative",
		Experience: "Changed Foo without tests",
		Decision:   "Add tests for Foo",
		Outcome:    "Regression",
		Impact:     memory.PatternImpact{Severity: "high"},
		Directive:  "Test pkg.Foo.",
		Symbols:    []string{"pkg.Foo", "pkg.Bar"},
	}
	payloadBytes, _ := json.Marshal(payload)
	fakeMemory := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "Test pkg.Foo.", Authority: 5, UpdatedAt: time.Now(), Payload: payloadBytes},
		},
	}
	weights := DefaultRankingWeights()
	compiler := &Compiler{Memory: fakeMemory, Ranking: &weights}
	req := CompileRequest{RetrievalQuery: "test situation", Symbols: []string{"pkg.Foo", "other.Sym"},
		MaxPerKind: 5,
	}
	bundle, err := compiler.Compile(context.Background(), req)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(bundle.MatchedSymbols) == 0 {
		t.Fatal("expected MatchedSymbols when task symbols overlap pattern payload")
	}
	found := false
	for _, s := range bundle.MatchedSymbols {
		if s == "pkg.Foo" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("MatchedSymbols = %v, want to include pkg.Foo", bundle.MatchedSymbols)
	}
	if bundle.SymbolRelevanceReason == "" {
		t.Error("expected SymbolRelevanceReason when symbol overlap")
	}
	if bundle.ReferenceCount != 0 {
		t.Errorf("ReferenceCount = %d without compiler LSP, want 0", bundle.ReferenceCount)
	}
}

func TestCompiler_Compile_groupedOutputs_includeConstraintsAndExperience(t *testing.T) {
	now := time.Now()

	fakeMemory := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: uuid.New(), Kind: api.MemoryKindDecision, Statement: "d1", Authority: 8, UpdatedAt: now},
			{ID: uuid.New(), Kind: api.MemoryKindConstraint, Statement: "c1", Authority: 7, UpdatedAt: now},
			{ID: uuid.New(), Kind: api.MemoryKindFailure, Statement: "f1", Authority: 6, UpdatedAt: now},
			{ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "p1", Authority: 5, UpdatedAt: now},
		},
	}
	weights := DefaultRankingWeights()
	compiler := &Compiler{Memory: fakeMemory, Ranking: &weights}

	bundle, err := compiler.Compile(context.Background(), CompileRequest{RetrievalQuery: "test situation", MaxPerKind: 5,
		Mode: "continuity",
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(bundle.Continuity) == 0 {
		t.Fatal("expected continuity group to be populated")
	}
	if len(bundle.Constraints) == 0 {
		t.Fatal("expected constraints group to be populated")
	}
	if len(bundle.Experience) == 0 {
		t.Fatal("expected experience group to be populated")
	}
}

func TestCompiler_Compile_threadModeBiasesContinuityWindow(t *testing.T) {
	now := time.Now()

	var objs []memory.MemoryObject
	for i := 0; i < 7; i++ {
		objs = append(objs, memory.MemoryObject{
			ID:        uuid.New(),
			Kind:      api.MemoryKindDecision,
			Statement: fmt.Sprintf("decision-%d", i),
			Authority: i + 1,
			UpdatedAt: now,
		})
	}
	fakeMemory := &fakeMemorySearcher{objs: objs}
	weights := DefaultRankingWeights()
	compiler := &Compiler{Memory: fakeMemory, Ranking: &weights}

	contBundle, err := compiler.Compile(context.Background(), CompileRequest{RetrievalQuery: "test situation", MaxPerKind: 2,
		Mode: "continuity",
	})
	if err != nil {
		t.Fatalf("Compile continuity: %v", err)
	}
	threadBundle, err := compiler.Compile(context.Background(), CompileRequest{RetrievalQuery: "test situation", MaxPerKind: 2,
		Mode: "thread",
	})
	if err != nil {
		t.Fatalf("Compile thread: %v", err)
	}
	if len(threadBundle.Continuity) <= len(contBundle.Continuity) {
		t.Fatalf("thread continuity window should be wider: thread=%d continuity=%d", len(threadBundle.Continuity), len(contBundle.Continuity))
	}
}

func TestCompiler_Compile_worksWithRetrievalQueryOnly(t *testing.T) {
	fakeMemory := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: uuid.New(), Kind: api.MemoryKindDecision, Statement: "d1", Authority: 8, UpdatedAt: time.Now()},
		},
	}
	weights := DefaultRankingWeights()
	compiler := &Compiler{Memory: fakeMemory, Ranking: &weights}
	bundle, err := compiler.Compile(context.Background(), CompileRequest{
		RetrievalQuery: "situation text only",
		MaxPerKind:     5,
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(bundle.Decisions) == 0 {
		t.Fatal("expected decisions from fake memory")
	}
}

// Phase 5 mandatory test 1: continuity restore.
func TestMandatory_ContinuityRestore_containsStateAndDecision(t *testing.T) {
	now := time.Now()
	fakeMemory := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: uuid.New(), Kind: api.MemoryKindState, Statement: "current state context", Authority: 8, UpdatedAt: now},
			{ID: uuid.New(), Kind: api.MemoryKindDecision, Statement: "recent decision", Authority: 7, UpdatedAt: now},
		},
	}
	weights := DefaultRankingWeights()
	compiler := &Compiler{
		Memory:  fakeMemory,
		Ranking: &weights,
	}
	bundle, err := compiler.Compile(context.Background(), CompileRequest{RetrievalQuery: "test situation", MaxPerKind: 5,
		Mode: "continuity",
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(bundle.Continuity) < 2 {
		t.Fatalf("continuity len=%d want at least 2", len(bundle.Continuity))
	}
}

// Phase 5 mandatory test 3: pattern reuse.
func TestMandatory_PatternReuse_experienceContainsPattern(t *testing.T) {
	now := time.Now()
	patternID := uuid.New()
	fakeMemory := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: patternID, Kind: api.MemoryKindPattern, Statement: "reusable pattern", Authority: 9, UpdatedAt: now},
		},
	}
	weights := DefaultRankingWeights()
	compiler := &Compiler{
		Memory:  fakeMemory,
		Ranking: &weights,
	}
	bundle, err := compiler.Compile(context.Background(), CompileRequest{RetrievalQuery: "test situation", MaxPerKind: 5})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(bundle.Experience) == 0 || bundle.Experience[0].ID != patternID.String() {
		t.Fatalf("experience should include reusable pattern %s, got %#v", patternID, bundle.Experience)
	}
}

// Phase 5 mandatory test 4: no project required.
func TestMandatory_NoProjectRequired_returnsMeaningfulSlices(t *testing.T) {
	now := time.Now()
	fakeMemory := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: uuid.New(), Kind: api.MemoryKindDecision, Statement: "decision", Authority: 8, UpdatedAt: now},
			{ID: uuid.New(), Kind: api.MemoryKindConstraint, Statement: "constraint", Authority: 8, UpdatedAt: now},
			{ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "pattern", Authority: 8, UpdatedAt: now},
		},
	}
	weights := DefaultRankingWeights()
	compiler := &Compiler{
		Memory:  fakeMemory,
		Ranking: &weights,
	}
	bundle, err := compiler.Compile(context.Background(), CompileRequest{RetrievalQuery: "test situation", MaxPerKind: 5})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(bundle.Continuity) == 0 || len(bundle.Constraints) == 0 || len(bundle.Experience) == 0 {
		t.Fatalf("expected non-empty grouped slices (tags-only compile): continuity=%d constraints=%d experience=%d", len(bundle.Continuity), len(bundle.Constraints), len(bundle.Experience))
	}
}

// TestCompiler_Compile_referenceCountFromLSP verifies max reference count across matched symbols (LSP-backed).
func TestCompiler_Compile_referenceCountFromLSP(t *testing.T) {
	payload := memory.PatternPayload{
		Symbols: []string{"pkg.Foo", "pkg.Bar"},
	}
	payloadBytes, _ := json.Marshal(payload)
	fakeMemory := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "ol", Authority: 5, UpdatedAt: time.Now(), Payload: payloadBytes},
		},
	}
	weights := DefaultRankingWeights()
	lsp := &tooling.FakeLSPClient{
		Symbols: []tooling.Symbol{
			{Name: "pkg.Foo", Range: &tooling.Range{StartLine: 2, StartCol: 5}},
			{Name: "pkg.Bar", Range: &tooling.Range{StartLine: 10, StartCol: 1}},
		},
		RefLengths: []int{8, 20},
	}
	compiler := &Compiler{
		Memory: fakeMemory, Ranking: &weights,
		LSP: lsp, LSPRecallEnabled: true,
	}
	req := CompileRequest{RetrievalQuery: "test situation", Symbols: []string{"pkg.Foo", "pkg.Bar"},
		MaxPerKind:   5,
		RepoRoot:     "/repo",
		LSPFocusPath: "pkg/foo.go",
	}
	bundle, err := compiler.Compile(context.Background(), req)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if bundle.ReferenceCount != 20 {
		t.Errorf("ReferenceCount = %d, want 20 (max of 8 and 20)", bundle.ReferenceCount)
	}
	if lsp.FindSymbolsCalls.Load() != 1 {
		t.Fatalf("FindSymbols calls = %d, want 1", lsp.FindSymbolsCalls.Load())
	}
	if lsp.FindReferencesCalls.Load() != 2 {
		t.Fatalf("FindReferences calls = %d, want 2", lsp.FindReferencesCalls.Load())
	}
}

func TestCompiler_Compile_referenceCount_respectsExpansionLimit(t *testing.T) {
	payload := memory.PatternPayload{Symbols: []string{"pkg.Foo"}}
	payloadBytes, _ := json.Marshal(payload)
	fakeMemory := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "ol", Authority: 5, UpdatedAt: time.Now(), Payload: payloadBytes},
		},
	}
	weights := DefaultRankingWeights()
	lsp := &tooling.FakeLSPClient{
		Symbols:    []tooling.Symbol{{Name: "pkg.Foo", Range: &tooling.Range{StartLine: 1, StartCol: 0}}},
		RefLengths: []int{50},
	}
	compiler := &Compiler{
		Memory: fakeMemory, Ranking: &weights,
		LSP: lsp, LSPRecallEnabled: true, ReferenceExpansionLimit: 10,
	}
	req := CompileRequest{RetrievalQuery: "test situation", Symbols: []string{"pkg.Foo"},
		MaxPerKind:   5,
		RepoRoot:     "/repo",
		LSPFocusPath: "pkg/foo.go",
	}
	bundle, err := compiler.Compile(context.Background(), req)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if bundle.ReferenceCount != 10 {
		t.Errorf("ReferenceCount = %d, want 10 (capped)", bundle.ReferenceCount)
	}
}

func TestCompiler_Compile_referenceCount_zeroWithoutFocusPath(t *testing.T) {
	payload := memory.PatternPayload{Symbols: []string{"pkg.Foo"}}
	payloadBytes, _ := json.Marshal(payload)
	fakeMemory := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "ol", Authority: 5, UpdatedAt: time.Now(), Payload: payloadBytes},
		},
	}
	weights := DefaultRankingWeights()
	lsp := &tooling.FakeLSPClient{
		Symbols:    []tooling.Symbol{{Name: "pkg.Foo", Range: &tooling.Range{StartLine: 1, StartCol: 0}}},
		RefLengths: []int{99},
	}
	compiler := &Compiler{Memory: fakeMemory, Ranking: &weights, LSP: lsp, LSPRecallEnabled: true}
	req := CompileRequest{RetrievalQuery: "test situation", Symbols: []string{"pkg.Foo"},
		MaxPerKind: 5,
		RepoRoot:   "/repo",
	}
	bundle, err := compiler.Compile(context.Background(), req)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if bundle.ReferenceCount != 0 {
		t.Errorf("ReferenceCount = %d without lsp_focus_path, want 0", bundle.ReferenceCount)
	}
}

// TestCompiler_Compile_withVariantModifier_limits verifies variant modifier changes per-kind limits.
func TestCompiler_Compile_withVariantModifier_limits(t *testing.T) {
	maxPerKind := 5
	// Enough items: 8 failures, 6 patterns, 3 constraints, 3 decisions
	var objs []memory.MemoryObject
	for i := 0; i < 3; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindConstraint, Statement: fmt.Sprintf("constraint %d unique text", i), Authority: i, UpdatedAt: time.Now()})
	}
	for i := 0; i < 3; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindDecision, Statement: fmt.Sprintf("decision %d unique text", i), Authority: i, UpdatedAt: time.Now()})
	}
	for i := 0; i < 8; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindFailure, Statement: fmt.Sprintf("failure %d unique text", i), Authority: i, UpdatedAt: time.Now()})
	}
	for i := 0; i < 2; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: fmt.Sprintf("objectlesson %d unique text", i), Authority: i, UpdatedAt: time.Now()})
	}
	for i := 0; i < 6; i++ {
		objs = append(objs, memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: fmt.Sprintf("pattern %d unique text", i), Authority: i, UpdatedAt: time.Now()})
	}
	fakeMemory := &fakeMemorySearcher{objs: objs}
	weights := DefaultRankingWeights()
	compiler := &Compiler{Memory: fakeMemory, Ranking: &weights}

	baseReq := CompileRequest{RetrievalQuery: "test situation", MaxPerKind: maxPerKind}
	balanced, err := compiler.Compile(context.Background(), baseReq)
	if err != nil {
		t.Fatalf("balanced Compile: %v", err)
	}
	mod := VariantModifierForStrategy(StrategyFailureHeavy, maxPerKind)
	if mod == nil {
		t.Fatal("failure_heavy modifier must be non-nil")
	}
	failureHeavyReq := baseReq
	failureHeavyReq.VariantModifier = mod
	failureHeavy, err := compiler.Compile(context.Background(), failureHeavyReq)
	if err != nil {
		t.Fatalf("failure_heavy Compile: %v", err)
	}
	// failure_heavy: Failures=7, Patterns=3.
	if len(failureHeavy.KnownFailures) < len(balanced.KnownFailures) {
		t.Errorf("failure_heavy KnownFailures = %d, want >= balanced %d", len(failureHeavy.KnownFailures), len(balanced.KnownFailures))
	}
	if len(failureHeavy.ApplicablePatterns) > len(balanced.ApplicablePatterns) {
		t.Errorf("failure_heavy ApplicablePatterns = %d, want <= balanced %d", len(failureHeavy.ApplicablePatterns), len(balanced.ApplicablePatterns))
	}
	// failure_heavy allows 7 failures, 3 patterns. balanced allows 5 each.
	if len(failureHeavy.KnownFailures) != 7 {
		t.Errorf("failure_heavy KnownFailures = %d, want 7", len(failureHeavy.KnownFailures))
	}
	if len(failureHeavy.ApplicablePatterns) != 3 {
		t.Errorf("failure_heavy ApplicablePatterns = %d, want 3", len(failureHeavy.ApplicablePatterns))
	}
	if len(balanced.KnownFailures) != 5 {
		t.Errorf("balanced KnownFailures = %d, want 5", len(balanced.KnownFailures))
	}
	if len(balanced.ApplicablePatterns) != 5 {
		t.Errorf("balanced ApplicablePatterns = %d, want 5", len(balanced.ApplicablePatterns))
	}
}

type fakeExperienceLister struct {
	objs []memory.MemoryObject
}

func (f *fakeExperienceLister) ListForCompile(ctx context.Context, limit int) ([]memory.MemoryObject, error) {
	_ = limit
	return f.objs, nil
}

// TestCompiler_Compile_experienceListerPrepends verifies Phase 4: experiences merge before scoring and rank with other memories.
func TestCompiler_Compile_experienceListerPrepends(t *testing.T) {
	expID := uuid.New()
	experience := memory.MemoryObject{
		ID: expID, Kind: api.MemoryKindDecision, Statement: "promoted experience text is long enough for the test",
		Authority: 10, UpdatedAt: time.Now(),
	}
	baseObjs := []memory.MemoryObject{
		{ID: uuid.New(), Kind: api.MemoryKindDecision, Statement: "regular decision with enough text", Authority: 5, UpdatedAt: time.Now()},
	}
	fakeMemory := &fakeMemorySearcher{objs: baseObjs}
	el := &fakeExperienceLister{objs: []memory.MemoryObject{experience}}
	weights := DefaultRankingWeights()
	compiler := &Compiler{
		Memory:           fakeMemory,
		Ranking:          &weights,
		Experiences:      el,
		ExperiencesLimit: 10,
	}
	bundle, err := compiler.Compile(context.Background(), CompileRequest{RetrievalQuery: "test situation", MaxPerKind: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Decisions) < 2 {
		t.Fatalf("decisions len = %d, want >= 2", len(bundle.Decisions))
	}
	if bundle.Decisions[0].Statement != experience.Statement {
		t.Errorf("first decision = %q, want experience statement first", bundle.Decisions[0].Statement)
	}
}

type fakeMemorySearcherByToken struct {
	base []memory.MemoryObject
	// token -> extra memories
	byToken map[string][]memory.MemoryObject
}

func (f *fakeMemorySearcherByToken) Search(ctx context.Context, req memory.SearchRequest) ([]memory.MemoryObject, error) {
	_ = ctx
	_ = req
	return append([]memory.MemoryObject(nil), f.base...), nil
}

func (f *fakeMemorySearcherByToken) SearchMemories(ctx context.Context, req memory.MemoriesSearchRequest) ([]memory.MemoryObject, error) {
	_ = ctx
	if f.byToken == nil {
		return nil, nil
	}
	if extra, ok := f.byToken[req.Query]; ok {
		return append([]memory.MemoryObject(nil), extra...), nil
	}
	return nil, nil
}

func TestCompiler_Compile_lexicalTokenBridgeFindsParaphrase(t *testing.T) {
	now := time.Now()

	irrelevantID := uuid.New()
	relevantID := uuid.New()

	// Base Search is intentionally noise-only.
	// SearchMemories (token bridge) injects the paraphrased relevant constraint.
	base := []memory.MemoryObject{
		{ID: irrelevantID, Kind: api.MemoryKindConstraint, Statement: "Do not cache results", Authority: 5, UpdatedAt: now},
	}
	relevant := memory.MemoryObject{
		ID: relevantID, Kind: api.MemoryKindConstraint,
		// Paraphrase: "omit tests" instead of "skip tests", but shares key tokens.
		Statement: "Never omit tests in release path", Authority: 5, UpdatedAt: now,
	}

	fakeMemory := &fakeMemorySearcherByToken{
		base: base,
		byToken: map[string][]memory.MemoryObject{
			"never":   {relevant},
			"skip":    {relevant},
			"tests":   {relevant},
			"release": {relevant},
			"path":    {relevant},
			"ship":    nil,
		},
	}
	weights := DefaultRankingWeights() // LexicalSimilarity enabled by default.
	compiler := &Compiler{
		Memory:  fakeMemory,
		Ranking: &weights,
	}

	bundle, err := compiler.Compile(context.Background(), CompileRequest{
		RetrievalQuery: "Never skip tests in release path",
		MaxPerKind:     1,
	})

	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(bundle.GoverningConstraints) != 1 {
		t.Fatalf("GoverningConstraints len=%d want 1", len(bundle.GoverningConstraints))
	}
	if bundle.GoverningConstraints[0].ID != relevantID.String() {
		t.Fatalf("selected constraint=%q (id=%s), want paraphrased relevant=%q (id=%s)",
			bundle.GoverningConstraints[0].Statement, bundle.GoverningConstraints[0].ID,
			relevant.Statement, relevantID.String(),
		)
	}
}

func TestCompiler_Compile_authorityDominatesWhenCapped(t *testing.T) {
	now := time.Now()

	highAuthID := uuid.New()
	lowAuthID := uuid.New()

	// Base Search injects high-authority noise that lexically matches some tokens.
	highAuth := memory.MemoryObject{
		ID: highAuthID, Kind: api.MemoryKindConstraint,
		Statement: "Never cache tests in release path", Authority: 9, UpdatedAt: now,
	}
	// SearchMemories injects lower-authority relevant constraint with higher lexical match.
	lowAuth := memory.MemoryObject{
		ID: lowAuthID, Kind: api.MemoryKindConstraint,
		Statement: "Never omit tests in release path", Authority: 5, UpdatedAt: now,
	}

	fakeMemory := &fakeMemorySearcherByToken{
		base: []memory.MemoryObject{highAuth},
		byToken: map[string][]memory.MemoryObject{
			"tests":   {lowAuth},
			"release": {lowAuth},
			"path":    {lowAuth},
			"never":   {lowAuth},
		},
	}
	weights := DefaultRankingWeights()
	compiler := &Compiler{
		Memory:  fakeMemory,
		Ranking: &weights,
	}

	bundle, err := compiler.Compile(context.Background(), CompileRequest{
		RetrievalQuery: "Never skip tests in release path",
		MaxPerKind:     1,
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(bundle.GoverningConstraints) != 1 {
		t.Fatalf("GoverningConstraints len=%d want 1", len(bundle.GoverningConstraints))
	}
	if bundle.GoverningConstraints[0].ID != highAuthID.String() {
		t.Fatalf("selected constraint id=%s want high-authority=%s; selected statement=%q",
			bundle.GoverningConstraints[0].ID, highAuthID.String(), bundle.GoverningConstraints[0].Statement,
		)
	}
}

func TestCompiler_Compile_mergeCandidatesDedupesByID(t *testing.T) {
	now := time.Now()

	id1 := uuid.New()
	id2 := uuid.New()

	base := []memory.MemoryObject{
		{ID: id1, Kind: api.MemoryKindConstraint, Statement: "Never omit tests in release path", Authority: 5, UpdatedAt: now},
	}
	extra := []memory.MemoryObject{
		{ID: id1, Kind: api.MemoryKindConstraint, Statement: "Never omit tests in release path", Authority: 5, UpdatedAt: now}, // duplicate
		{ID: id2, Kind: api.MemoryKindConstraint, Statement: "Never skip tests in release path", Authority: 5, UpdatedAt: now}, // unique
	}

	fakeMemory := &fakeMemorySearcherByToken{
		base: base,
		byToken: map[string][]memory.MemoryObject{
			"tests": {extra[0], extra[1]},
		},
	}
	weights := DefaultRankingWeights()
	compiler := &Compiler{
		Memory:  fakeMemory,
		Ranking: &weights,
	}

	bundle, err := compiler.Compile(context.Background(), CompileRequest{
		RetrievalQuery: "Never skip tests in release path",
		MaxPerKind:     2,
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(bundle.GoverningConstraints) != 2 {
		t.Fatalf("GoverningConstraints len=%d want 2 (unique ids)", len(bundle.GoverningConstraints))
	}
	ids := map[string]bool{
		bundle.GoverningConstraints[0].ID: true,
		bundle.GoverningConstraints[1].ID: true,
	}
	if !ids[id1.String()] || !ids[id2.String()] {
		t.Fatalf("constraint ids=%v want %s and %s", ids, id1.String(), id2.String())
	}
}
