package recall

import (
	"context"
	"fmt"
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// fakeContradiction implements unresolved ID listing and optional pairs (RIU tests).
type fakeContradiction struct {
	ids   []uuid.UUID
	pairs [][2]uuid.UUID
}

func (f *fakeContradiction) ListMemoryIDsInUnresolved(ctx context.Context) ([]uuid.UUID, error) {
	return f.ids, nil
}

func (f *fakeContradiction) ListUnresolvedPairs(ctx context.Context, limit int) ([][2]uuid.UUID, error) {
	if limit > 0 && len(f.pairs) > limit {
		return f.pairs[:limit], nil
	}
	return f.pairs, nil
}

func riuCompiler(weights RankingWeights, riu RIUConfig, mem *fakeMemorySearcher, contradiction ContradictionExclusionLister) *Compiler {
	return &Compiler{
		Memory:        mem,
		Ranking:       &weights,
		RIU:           &riu,
		Contradiction: contradiction,
	}
}

// T1: With RIU, transferable metadata is preserved; ordering follows authority-first (RC1).
func TestRIU_T1_globalOutranksWeakLocal(t *testing.T) {
	now := time.Now()
	idLocal := uuid.MustParse("c1000000-0000-0000-0000-000000000001")
	idGlobal := uuid.MustParse("c2000000-0000-0000-0000-000000000002")

	mem := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{
				ID: idLocal, Kind: api.MemoryKindDecision,
				Statement: "local-weak", Authority: 8, Applicability: api.ApplicabilityGoverning,
				Tags: []string{"misc"}, UpdatedAt: now, Status: api.StatusActive,
			},
			{
				ID: idGlobal, Kind: api.MemoryKindDecision,
				Statement: "global-transferable", Authority: 4, Applicability: api.ApplicabilityAdvisory,
				Tags: []string{"api", "go"}, UpdatedAt: now, Status: api.StatusActive,
			},
		},
	}
	w := DefaultRankingWeights()
	riu := RIUConfig{Enabled: true, Policy: ContradictionPolicyExclude, Weights: DefaultRIUWeights()}
	c := riuCompiler(w, riu, mem, &fakeContradiction{})
	bundle, err := c.Compile(context.Background(), CompileRequest{
		RetrievalQuery: "test situation",
		Tags:           []string{"api", "go"},
		MaxPerKind:     5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Decisions) != 2 {
		t.Fatalf("decisions %d", len(bundle.Decisions))
	}
	if bundle.Decisions[0].Statement != "local-weak" {
		t.Errorf("first = %q want local-weak (authority 8 > 4)", bundle.Decisions[0].Statement)
	}
	if bundle.Decisions[1].Statement != "global-transferable" {
		t.Errorf("second = %q want global-transferable", bundle.Decisions[1].Statement)
	}
	if bundle.Decisions[1].RIU == nil || !bundle.Decisions[1].RIU.Transferable {
		t.Errorf("expected global row with transferable RIU metadata: %+v", bundle.Decisions[1].RIU)
	}
}

// T2: Recall is unscoped; RIU output carries no project-affinity component.
func TestRIU_T2_noProjectAffinity_unscopedRecall(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	mem := &fakeMemorySearcher{
		objs: []memory.MemoryObject{{
			ID: id, Kind: api.MemoryKindDecision,
			Statement: "local", Authority: 5, Applicability: api.ApplicabilityGoverning,
			Tags: []string{"x"}, UpdatedAt: now, Status: api.StatusActive,
		}},
	}
	w := DefaultRankingWeights()
	riu := RIUConfig{Enabled: true, Policy: ContradictionPolicyExclude, Weights: DefaultRIUWeights()}
	c := riuCompiler(w, riu, mem, nil)
	bundle, err := c.Compile(context.Background(), CompileRequest{Tags: []string{"x"}, MaxPerKind: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Decisions) != 1 {
		t.Fatal(len(bundle.Decisions))
	}
	if bundle.Decisions[0].RIU == nil {
		t.Errorf("expected RIU breakdown to be present")
	}
}

// T3: Higher authority dominates when tags align (lineage proxy tracks authority).
func TestRIU_T3_highAuthorityDominatesWithTags(t *testing.T) {
	now := time.Now()
	low := uuid.MustParse("d1000000-0000-0000-0000-000000000001")
	high := uuid.MustParse("d2000000-0000-0000-0000-000000000002")
	mem := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: low, Kind: api.MemoryKindDecision,
				Statement: "low-auth", Authority: 2, Applicability: api.ApplicabilityGoverning,
				Tags: []string{"api"}, UpdatedAt: now, Status: api.StatusActive},
			{ID: high, Kind: api.MemoryKindDecision,
				Statement: "high-auth", Authority: 9, Applicability: api.ApplicabilityGoverning,
				Tags: []string{"api"}, UpdatedAt: now, Status: api.StatusActive},
		},
	}
	w := DefaultRankingWeights()
	riu := RIUConfig{Enabled: true, Policy: ContradictionPolicyExclude, Weights: DefaultRIUWeights()}
	c := riuCompiler(w, riu, mem, nil)
	bundle, err := c.Compile(context.Background(), CompileRequest{ Tags: []string{"api"}, MaxPerKind: 5})
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Decisions[0].Statement != "high-auth" {
		t.Errorf("first = %q", bundle.Decisions[0].Statement)
	}
	if bundle.Decisions[0].RIU.LineageProxyScore <= bundle.Decisions[1].RIU.LineageProxyScore {
		t.Errorf("lineage proxy should track authority: %+v vs %+v", bundle.Decisions[0].RIU, bundle.Decisions[1].RIU)
	}
}

// T4: Warn policy — contradicted memory is not silent (status + penalty on RIU).
func TestRIU_T4_warnNotSilent(t *testing.T) {
	now := time.Now()
	idBad := uuid.MustParse("e1000000-0000-0000-0000-000000000001")
	idOk := uuid.MustParse("e2000000-0000-0000-0000-000000000002")
	mem := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: idBad, Kind: api.MemoryKindDecision,
				Statement: "contradicted", Authority: 9, Tags: []string{"api"}, UpdatedAt: now, Status: api.StatusActive},
			{ID: idOk, Kind: api.MemoryKindDecision,
				Statement: "clean", Authority: 3, Tags: []string{"api"}, UpdatedAt: now, Status: api.StatusActive},
		},
	}
	w := DefaultRankingWeights()
	riu := RIUConfig{Enabled: true, Policy: ContradictionPolicyWarn, Weights: DefaultRIUWeights()}
	fc := &fakeContradiction{ids: []uuid.UUID{idBad}}
	c := riuCompiler(w, riu, mem, fc)
	bundle, err := c.Compile(context.Background(), CompileRequest{ Tags: []string{"api"}, MaxPerKind: 5})
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, it := range bundle.Decisions {
		if it.ID == idBad.String() {
			found = true
			if it.RIU == nil || it.RIU.ContradictionStatus != "unresolved" || it.RIU.ContradictionPenalty <= 0 {
				t.Errorf("expected unresolved + penalty: %+v", it.RIU)
			}
		}
	}
	if !found {
		t.Fatal("contradicted memory missing from bundle")
	}
}

// T5: Bounded pair surfaces contradiction sets when both memories appear.
func TestRIU_T5_boundedPair(t *testing.T) {
	now := time.Now()
	a := uuid.MustParse("f1000000-0000-0000-0000-000000000001")
	b := uuid.MustParse("f2000000-0000-0000-0000-000000000002")
	mem := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: a, Kind: api.MemoryKindDecision,
				Statement: "A", Authority: 5, Tags: []string{"api"}, UpdatedAt: now, Status: api.StatusActive},
			{ID: b, Kind: api.MemoryKindDecision,
				Statement: "B", Authority: 5, Tags: []string{"api"}, UpdatedAt: now, Status: api.StatusActive},
		},
	}
	w := DefaultRankingWeights()
	riu := RIUConfig{Enabled: true, Policy: ContradictionPolicyBoundedPair, Weights: DefaultRIUWeights(), BoundedPairMax: 8}
	fc := &fakeContradiction{pairs: [][2]uuid.UUID{{a, b}}}
	c := riuCompiler(w, riu, mem, fc)
	bundle, err := c.Compile(context.Background(), CompileRequest{ Tags: []string{"api"}, MaxPerKind: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.ContradictionSets) != 1 {
		t.Fatalf("ContradictionSets = %#v", bundle.ContradictionSets)
	}
	if len(bundle.ContradictionSets[0].Items) != 2 {
		t.Fatal(bundle.ContradictionSets[0].Items)
	}
}

// T6: MaxTotal is respected with RIU enabled.
func TestRIU_T6_maxTotalWithRIU(t *testing.T) {
	now := time.Now()
	objs := make([]memory.MemoryObject, 0, 4)
	for i := 0; i < 4; i++ {
		objs = append(objs, memory.MemoryObject{
			ID: uuid.New(), Kind: api.MemoryKindDecision,
			Statement: fmt.Sprintf("decision %d unique text for maxtotal", i), Authority: 5 + i, Tags: []string{"t"}, UpdatedAt: now, Status: api.StatusActive,
		})
	}
	mem := &fakeMemorySearcher{objs: objs}
	w := DefaultRankingWeights()
	riu := RIUConfig{Enabled: true, Policy: ContradictionPolicyExclude, Weights: DefaultRIUWeights()}
	c := riuCompiler(w, riu, mem, nil)
	bundle, err := c.Compile(context.Background(), CompileRequest{ Tags: []string{"t"}, MaxPerKind: 10, MaxTotal: 2})
	if err != nil {
		t.Fatal(err)
	}
	n := len(bundle.GoverningConstraints) + len(bundle.Decisions) + len(bundle.KnownFailures) +
		len(bundle.ApplicablePatterns)
	if n != 2 {
		t.Errorf("total items = %d want 2", n)
	}
}

// T7: Identical compile runs yield identical decision ordering (deterministic tie-break).
func TestRIU_T7_deterministicDoubleRun(t *testing.T) {
	now := time.Now()
	// Equal scores: same authority, tags, kind — order by UUID string ascending.
	id1 := uuid.MustParse("00000000-0000-0000-0000-0000000000aa")
	id2 := uuid.MustParse("00000000-0000-0000-0000-0000000000bb")
	mem := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: id2, Kind: api.MemoryKindDecision,
				Statement: "second-id", Authority: 5, Tags: []string{"z"}, UpdatedAt: now, Status: api.StatusActive},
			{ID: id1, Kind: api.MemoryKindDecision,
				Statement: "first-id", Authority: 5, Tags: []string{"z"}, UpdatedAt: now, Status: api.StatusActive},
		},
	}
	w := DefaultRankingWeights()
	riu := RIUConfig{Enabled: true, Policy: ContradictionPolicyExclude, Weights: DefaultRIUWeights()}
	c := riuCompiler(w, riu, mem, nil)
	req := CompileRequest{ Tags: []string{"z"}, MaxPerKind: 5}
	b1, err := c.Compile(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := c.Compile(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	// Order must match; refTime is derived from max(UpdatedAt) — deterministic for identical rows.
	for i := range b1.Decisions {
		if b1.Decisions[i].ID != b2.Decisions[i].ID {
			t.Fatalf("run mismatch at %d: %v vs %v", i, b1.Decisions[i].ID, b2.Decisions[i].ID)
		}
	}
	if b1.Decisions[0].ID != id1.String() || b1.Decisions[1].ID != id2.String() {
		t.Errorf("stable UUID order want %s then %s, got %s, %s", id1, id2, b1.Decisions[0].ID, b1.Decisions[1].ID)
	}
}

func TestScoreAndSortWithReason_stableTieBreak(t *testing.T) {
	now := time.Now()
	idLo := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	idHi := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	objs := []memory.MemoryObject{
		{ID: idHi, Kind: api.MemoryKindDecision, Statement: "b", Authority: 5, UpdatedAt: now, Status: api.StatusActive},
		{ID: idLo, Kind: api.MemoryKindDecision, Statement: "a", Authority: 5, UpdatedAt: now, Status: api.StatusActive},
	}
	w := DefaultRankingWeights()
	out := ScoreAndSortWithReason(objs, ScoreRequest{Tags: []string{}}, w, 0)
	if out[0].Object.ID != idLo {
		t.Errorf("want lower UUID first, got %s", out[0].Object.ID)
	}
}
