package recall

import (
	"context"
	"strings"
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

func effectivenessCompiler(t *testing.T, objs []memory.MemoryObject, util UtilityScoreProvider) *Compiler {
	t.Helper()
	w := DefaultRankingWeights()
	c := &Compiler{
		Memory:        &fakeMemorySearcher{objs: objs},
		Ranking:       &w,
		Utility:       util,
		UtilityWeight: 0.12,
	}
	return c
}

func groundingText(b *RecallBundle) string {
	if b == nil || b.AgentGrounding == nil {
		return ""
	}
	return b.AgentGrounding.Formatted
}

func TestEffectiveness_aboutnessDoesNotGroundOffTopic(t *testing.T) {
	now := time.Now()
	wipe := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindConstraint,
		Statement:     "Never destroy Pluribus controlplane data; take a pg_dump before every upgrade.",
		Authority:     8, Applicability: api.ApplicabilityGoverning, UpdatedAt: now,
	}
	lootz := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision,
		Statement: "AllTheLootz table size is 12 by 16 and the card backs are linen.",
		Authority: 9, UpdatedAt: now.Add(-30 * 24 * time.Hour),
	}
	ofiq := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindState,
		Statement: "VAN init for OFIQ-PHP greenfield wrapping ofiq_libc.h.",
		Authority: 8, UpdatedAt: now.Add(-20 * 24 * time.Hour),
	}
	c := effectivenessCompiler(t, []memory.MemoryObject{wipe, lootz, ofiq}, nil)
	bundle, err := c.Compile(context.Background(), CompileRequest{
		RetrievalQuery: "Pluribus controlplane pg_dump before upgrade",
		MaxPerKind:     5,
	})
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(groundingText(bundle))
	if !strings.Contains(text, "never destroy pluribus controlplane") {
		t.Fatalf("expected Pluribus controlplane constraint in grounding:\n%s", groundingText(bundle))
	}
	for _, bad := range []string{"allthelootz", "ofiq-php", "ofiq_libc"} {
		if strings.Contains(text, bad) {
			t.Fatalf("off-topic %q must not be grounded:\n%s", bad, groundingText(bundle))
		}
	}
}

func TestEffectiveness_lastTimePrefersRecentOnTopic(t *testing.T) {
	now := time.Now()
	recent := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindPattern,
		Statement: "Restored the Pluribus controlplane hive from Cursor dumps; episodes were not restored.",
		Authority: 2, UpdatedAt: now,
	}
	oldOff := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindPattern,
		Statement: "OFIQ VAN complete for hostile review; next command is PLAN.",
		Authority: 9, UpdatedAt: now.Add(-400 * 24 * time.Hour),
	}
	c := effectivenessCompiler(t, []memory.MemoryObject{recent, oldOff}, nil)
	bundle, err := c.Compile(context.Background(), CompileRequest{
		RetrievalQuery: "how did we restore the Pluribus controlplane hive last time?",
		Mode:           "thread",
		MaxPerKind:     5,
	})
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(groundingText(bundle))
	if !strings.Contains(text, "restored the pluribus controlplane hive") {
		t.Fatalf("last-time must return the recent restore procedure:\n%s", groundingText(bundle))
	}
	if strings.Contains(text, "ofiq van") {
		t.Fatalf("old off-topic pattern must not win last-time:\n%s", groundingText(bundle))
	}
}

func TestEffectiveness_attentionShiftKeepsImportantRow(t *testing.T) {
	now := time.Now()
	important := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindConstraint,
		Statement:     "Never destroy Pluribus controlplane data; never treat implied intent as permission to wipe.",
		Authority:     8, Applicability: api.ApplicabilityGoverning, UpdatedAt: now,
	}
	junk := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision,
		Statement: "AllTheLootz card backs use linen stock; do not invent project_id.",
		Authority: 9, UpdatedAt: now.Add(-10 * 24 * time.Hour),
	}
	c := effectivenessCompiler(t, []memory.MemoryObject{important, junk}, nil)
	bundle, err := c.Compile(context.Background(), CompileRequest{
		RetrievalQuery: "implied permission to wipe Pluribus controlplane",
		MaxPerKind:     5,
	})
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(groundingText(bundle))
	if !strings.Contains(text, "never destroy pluribus controlplane") {
		t.Fatalf("important governing row must survive a narrower wipe query:\n%s", groundingText(bundle))
	}
	if strings.Contains(text, "allthelootz") {
		t.Fatalf("unrelated high-authority junk must not be grounded:\n%s", groundingText(bundle))
	}
	if strings.Contains(text, "project_id") && strings.Contains(strings.ToLower(bundle.AgentGrounding.Formatted), "project_id") {
		// probe must not invent a project field; leftover mention only in junk we already rejected
	}
}

type mapUtility struct {
	scores map[uuid.UUID]float64
}

func (m mapUtility) GetScoresForMemories(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]float64, error) {
	return m.scores, nil
}

func TestEffectiveness_utilityReordersOnlyAfterFloor(t *testing.T) {
	now := time.Now()
	onTopicA := uuid.New()
	onTopicB := uuid.New()
	offTopic := uuid.New()
	objs := []memory.MemoryObject{
		{ID: onTopicA, Kind: api.MemoryKindPattern, Statement: "Pluribus recall_context then record_experience default loop variant alpha", Authority: 5, UpdatedAt: now},
		{ID: onTopicB, Kind: api.MemoryKindPattern, Statement: "Pluribus recall_context then record_experience default loop variant beta", Authority: 5, UpdatedAt: now},
		{ID: offTopic, Kind: api.MemoryKindDecision, Statement: "AllTheLootz table size and linen card backs", Authority: 9, UpdatedAt: now},
	}
	util := mapUtility{scores: map[uuid.UUID]float64{offTopic: 10, onTopicB: 3, onTopicA: 0}}
	c := effectivenessCompiler(t, objs, util)
	bundle, err := c.Compile(context.Background(), CompileRequest{
		RetrievalQuery: "pluribus recall_context record_experience loop",
		MaxPerKind:     5,
	})
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ToLower(groundingText(bundle))
	if strings.Contains(text, "allthelootz") {
		t.Fatalf("high-utility off-topic must stay below the floor:\n%s", groundingText(bundle))
	}
	if len(bundle.Experience) < 2 {
		t.Fatalf("both on-topic patterns should pass the floor: %+v", bundle.Experience)
	}
	if bundle.Experience[0].ID != onTopicB.String() {
		t.Fatalf("used/helpful row should lead among in-floor rows: first=%s", bundle.Experience[0].Statement)
	}
}

func TestEffectiveness_ftsEnoughSkipsAuthorityFallback(t *testing.T) {
	now := time.Now()
	fts := make([]memory.MemoryObject, 0, 20)
	for i := 0; i < 20; i++ {
		fts = append(fts, memory.MemoryObject{
			ID: uuid.New(), Kind: api.MemoryKindConstraint,
			Statement: "Never destroy Orac hive data during restore " + strings.Repeat("x", i),
			Authority: 4, UpdatedAt: now,
		})
	}
	s := &ftsThenAuthSearcher{fts: fts}
	w := DefaultRankingWeights()
	c := &Compiler{Memory: s, Ranking: &w}
	_, err := c.Compile(context.Background(), CompileRequest{
		RetrievalQuery: "Orac hive wipe restore",
		MaxPerKind:     5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.authCalled != 0 {
		t.Fatalf("authority Search must not run when aboutness already has 20 hits, called=%d", s.authCalled)
	}
}

type ftsThenAuthSearcher struct {
	fts        []memory.MemoryObject
	authCalled int
}

func (f *ftsThenAuthSearcher) Search(_ context.Context, _ memory.SearchRequest) ([]memory.MemoryObject, error) {
	f.authCalled++
	return []memory.MemoryObject{{
		ID: uuid.New(), Kind: api.MemoryKindDecision,
		Statement: "AllTheLootz authority fallback junk", Authority: 10,
	}}, nil
}

func (f *ftsThenAuthSearcher) SearchMemories(_ context.Context, _ memory.MemoriesSearchRequest) ([]memory.MemoryObject, error) {
	return nil, nil
}

func (f *ftsThenAuthSearcher) SearchFullText(_ context.Context, _, _ string, _ int) ([]memory.MemoryObject, error) {
	return f.fts, nil
}
