package recall

import (
	"encoding/json"
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

func TestScore_authorityAndRecency(t *testing.T) {
	req := ScoreRequest{}
	w := DefaultRankingWeights()
	now := time.Now()
	old := now.Add(-400 * 24 * time.Hour) // ~1 year+

	highAuth := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 8,
		UpdatedAt: now,
	}
	lowAuth := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 2,
		UpdatedAt: now,
	}
	oldItem := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 5,
		UpdatedAt: old,
	}

	sHigh := Score(highAuth, req, w, 10)
	sLow := Score(lowAuth, req, w, 10)
	sOld := Score(oldItem, req, w, 10)

	if sHigh <= sLow {
		t.Errorf("high authority score %v should be > low %v", sHigh, sLow)
	}
	if sOld <= sLow {
		t.Errorf("mid authority (5) score %v should be > low authority (2) %v", sOld, sLow)
	}
	// Same authority: use one refTime for both so recency differs by UpdatedAt only.
	ref := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	recent := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 5,
		UpdatedAt: ref,
	}
	stale := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 5,
		UpdatedAt: ref.Add(-400 * 24 * time.Hour),
	}
	sRecent := scoreAt(recent, req, w, 10, ref)
	sStale := scoreAt(stale, req, w, 10, ref)
	if sRecent <= sStale {
		t.Errorf("same authority: newer UpdatedAt score %v should be > older %v", sRecent, sStale)
	}
}

func TestScore_recency_prefersOccurredAtOverUpdatedAt(t *testing.T) {
	w := DefaultRankingWeights()
	req := ScoreRequest{}
	ref := time.Date(2025, 12, 15, 12, 0, 0, 0, time.UTC)
	ingested := ref
	oldEvent := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	recentEvent := time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC)
	memOldFact := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 5,
		UpdatedAt: ingested, OccurredAt: &oldEvent,
	}
	memRecentFact := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 5,
		UpdatedAt: ingested, OccurredAt: &recentEvent,
	}
	sOld := scoreAt(memOldFact, req, w, 10, ref)
	sRecent := scoreAt(memRecentFact, req, w, 10, ref)
	if sRecent <= sOld {
		t.Fatalf("recent event time should score higher recency than old event: %v vs %v", sRecent, sOld)
	}
}

func TestScore_evolutionInvalidatedDeprioritizes(t *testing.T) {
	w := DefaultRankingWeights()
	req := ScoreRequest{}
	base := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 7,
		UpdatedAt: time.Now(),
	}
	corrID := uuid.New()
	payload, _ := json.Marshal(map[string]any{
		"pluribus_evolution": map[string]any{"invalidated_by": corrID.String()},
	})
	inv := base
	inv.Payload = payload
	sBase := Score(base, req, w, 10)
	sInv := Score(inv, req, w, 10)
	if sInv >= sBase {
		t.Fatalf("invalidated memory should score lower: base=%v inv=%v", sBase, sInv)
	}
}

func TestScore_tagMatch(t *testing.T) {
	w := DefaultRankingWeights()
	req := ScoreRequest{Tags: []string{"go", "api"}}

	noTags := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 5,
		UpdatedAt: time.Now(), Tags: nil,
	}
	oneMatch := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 5,
		UpdatedAt: time.Now(), Tags: []string{"go"},
	}
	twoMatch := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 5,
		UpdatedAt: time.Now(), Tags: []string{"go", "api"},
	}

	sNone := Score(noTags, req, w, 10)
	sOne := Score(oneMatch, req, w, 10)
	sTwo := Score(twoMatch, req, w, 10)

	if len(req.Tags) > 0 {
		if sOne <= sNone {
			t.Errorf("one tag match %v should be > no tags %v (when request has tags)", sOne, sNone)
		}
		if sTwo <= sOne {
			t.Errorf("two tag match %v should be >= one %v", sTwo, sOne)
		}
	}
}

func TestScore_sessionCorrelationBoost(t *testing.T) {
	w := DefaultRankingWeights()
	req := ScoreRequest{SessionCorrelationID: "sess-a"}
	base := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 5,
		UpdatedAt: time.Now(), Tags: []string{"other"},
	}
	sessionTagged := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 5,
		UpdatedAt: time.Now(), Tags: []string{"mcp:session:sess-a"},
	}
	sBase := Score(base, req, w, 10)
	sSess := Score(sessionTagged, req, w, 10)
	if sSess <= sBase {
		t.Fatalf("session-tagged memory should rank higher: base=%v sess=%v", sBase, sSess)
	}
}

func TestScore_failureOverlap(t *testing.T) {
	w := DefaultRankingWeights()
	req := ScoreRequest{Tags: []string{"auth"}}

	failureWithTag := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindFailure, Authority: 3,
		UpdatedAt: time.Now(), Tags: []string{"auth"},
	}
	failureNoTag := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindFailure, Authority: 3,
		UpdatedAt: time.Now(), Tags: []string{"other"},
	}

	sWith := Score(failureWithTag, req, w, 10)
	sWithout := Score(failureNoTag, req, w, 10)

	if sWith <= sWithout {
		t.Errorf("failure with tag overlap %v should be > failure without %v", sWith, sWithout)
	}
}

func TestScoreAndSort_order(t *testing.T) {
	req := ScoreRequest{Tags: []string{"x"}}
	w := DefaultRankingWeights()
	now := time.Now()

	objs := []memory.MemoryObject{
		{ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 5, UpdatedAt: now, Tags: nil},
		{ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 5, UpdatedAt: now, Tags: []string{"x"}},
		{ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 5, UpdatedAt: now, Tags: []string{"x"}},
	}
	sorted := ScoreAndSort(objs, req, w, 0)
	if len(sorted) != 3 {
		t.Fatalf("len(sorted) = %d, want 3", len(sorted))
	}
	// First should be a tagged match; last should be no-tag mismatch.
	firstScore := Score(sorted[0], req, w, 10)
	lastScore := Score(sorted[2], req, w, 10)
	if firstScore < lastScore {
		t.Errorf("first item score %v should be >= last %v", firstScore, lastScore)
	}
}

func TestTagMatchScore_emptyRequestTags(t *testing.T) {
	// When request has no tags, we treat as full match (don't penalize).
	got := tagMatchScore([]string{"a", "b"}, nil)
	if got != 1.0 {
		t.Errorf("tagMatchScore(memTags, nil) = %v, want 1.0", got)
	}
	got = tagMatchScore([]string{"a"}, []string{})
	if got != 1.0 {
		t.Errorf("tagMatchScore with empty req tags = %v, want 1.0", got)
	}
}

func TestScore_patternGeneralizationBoost(t *testing.T) {
	payload := memory.PatternPayload{
		Polarity:   "positive",
		Experience: "e",
		Decision:   "d",
		Outcome:    "o",
		Impact:     memory.PatternImpact{Severity: "medium"},
		Directive:  "dir",
		Generalization: &memory.PatternGeneralizationMeta{
			Reason:                  "near_duplicate_reinforce",
			Jaccard:                 0.9,
			SupportingStatementKeys: []string{"abc", "def"},
		},
	}
	b, _ := json.Marshal(payload)
	obj := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindPattern, Authority: 5,
		UpdatedAt: time.Now(),
		Payload:   b,
	}
	w := DefaultRankingWeights()
	w.PatternGeneralization = 0.5
	req := ScoreRequest{}
	base := Score(obj, req, w, 10)
	wOff := w
	wOff.PatternGeneralization = 0
	if Score(obj, req, wOff, 10) >= base {
		t.Errorf("generalization weight 0 should score lower than with boost")
	}
}

func TestScore_symbolOverlapBoost(t *testing.T) {
	// Task 100: when task symbols overlap payload symbols, score gets SymbolOverlap boost
	payload := memory.PatternPayload{
		Polarity:   "negative",
		Experience: "changed X",
		Decision:   "did Y",
		Outcome:    "broke Z",
		Impact:     memory.PatternImpact{Severity: "high"},
		Directive:  "avoid Y",
		Symbols:    []string{"pkg.Foo", "pkg.Bar"},
	}
	payloadBytes, _ := json.Marshal(payload)
	obj := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindPattern, Authority: 5,
		UpdatedAt: time.Now(),
		Payload: payloadBytes,
	}
	w := DefaultRankingWeights()
	w.SymbolOverlap = 0.5

	reqNoOverlap := ScoreRequest{Symbols: []string{"other.Symbol"}}
	reqOneOverlap := ScoreRequest{Symbols: []string{"pkg.Foo"}}
	reqTwoOverlap := ScoreRequest{Symbols: []string{"pkg.Foo", "pkg.Bar"}}

	sNone := Score(obj, reqNoOverlap, w, 10)
	sOne := Score(obj, reqOneOverlap, w, 10)
	sTwo := Score(obj, reqTwoOverlap, w, 10)

	if sOne <= sNone {
		t.Errorf("symbol overlap (1) should boost score: %v <= %v", sOne, sNone)
	}
	// One or two overlaps both add SymbolOverlap * min(1, count) = 0.5; allow float variance
	if sTwo < sNone+0.4 {
		t.Errorf("two symbol overlaps should also get boost: sTwo=%v sNone=%v", sTwo, sNone)
	}
	if sOne < sNone+0.4 {
		t.Errorf("expected symbol overlap boost ~0.5: sOne=%v sNone=%v", sOne, sNone)
	}
}

func TestFailureSeverityOutranksLowSeverity(t *testing.T) {
	req := ScoreRequest{SituationQuery: "deploy"}
	w := DefaultRankingWeights()
	w.FailureSeverity = 1.0
	now := time.Now()
	high := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindFailure, Authority: 5,
		Statement: "production outage caused data loss in customer databases",
		UpdatedAt: now,
	}
	low := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindFailure, Authority: 5,
		Statement: "minor typo in log message",
		UpdatedAt: now,
	}
	sh := scoreBase(high, req, w, 10, now)
	sl := scoreBase(low, req, w, 10, now)
	if sh <= sl {
		t.Fatalf("expected high severity failure to score higher: %v vs %v", sh, sl)
	}
}

func TestCrossContextSalienceBoostsScore(t *testing.T) {
	req := ScoreRequest{}
	w := DefaultRankingWeights()
	w.CrossContextSalience = 0.5
	now := time.Now()
	empty := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 5,
		UpdatedAt: now,
	}
	withSal := memory.MemoryObject{
		ID:        uuid.New(),
		Kind:      api.MemoryKindDecision,
		Authority: 5,
		Payload:   []byte(`{"salience":{"distinct_contexts":4}}`),
		UpdatedAt: now,
	}
	s0 := scoreBase(empty, req, w, 10, now)
	s1 := scoreBase(withSal, req, w, 10, now)
	if s1 <= s0 {
		t.Fatalf("expected cross-context boost: %v vs %v", s1, s0)
	}
}

func TestCrossAgentSalienceBoostsScore(t *testing.T) {
	req := ScoreRequest{}
	w := DefaultRankingWeights()
	w.CrossAgentSalience = 0.5
	now := time.Now()
	empty := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 5,
		UpdatedAt: now,
	}
	withSal := memory.MemoryObject{
		ID:        uuid.New(),
		Kind:      api.MemoryKindDecision,
		Authority: 5,
		Payload:   []byte(`{"salience":{"distinct_agents":3}}`),
		UpdatedAt: now,
	}
	s0 := scoreBase(empty, req, w, 10, now)
	s1 := scoreBase(withSal, req, w, 10, now)
	if s1 <= s0 {
		t.Fatalf("expected cross-agent boost: %v vs %v", s1, s0)
	}
}

func TestRankingWeightsFromConfig_zeroMeansDefault(t *testing.T) {
	w := RankingWeightsFromConfig(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	d := DefaultRankingWeights()
	if w.Authority != d.Authority || w.Recency != d.Recency {
		t.Errorf("RankingWeightsFromConfig(0,...) should fill defaults: got %+v", w)
	}
	if w.CrossContextSalience != d.CrossContextSalience || w.CrossAgentSalience != d.CrossAgentSalience {
		t.Errorf("RankingWeightsFromConfig(0,...) cross salience defaults: got ctx=%v agent=%v want ctx=%v agent=%v",
			w.CrossContextSalience, w.CrossAgentSalience, d.CrossContextSalience, d.CrossAgentSalience)
	}
	if w.SemanticSimilarity != d.SemanticSimilarity {
		t.Errorf("RankingWeightsFromConfig(0,...) SemanticSimilarity = %v, want %v", w.SemanticSimilarity, d.SemanticSimilarity)
	}
}

func TestResolveSemanticSimilarityWeight(t *testing.T) {
	if g := ResolveSemanticSimilarityWeight(nil); g != DefaultSemanticSimilarityWeight {
		t.Errorf("nil explicit: got %v want %v", g, DefaultSemanticSimilarityWeight)
	}
	explicit := 0.25
	if g := ResolveSemanticSimilarityWeight(&explicit); g != 0.25 {
		t.Errorf("explicit 0.25: got %v", g)
	}
	zero := 0.0
	if g := ResolveSemanticSimilarityWeight(&zero); g != 0 {
		t.Errorf("explicit 0: got %v want 0", g)
	}
}

func TestRankingWeightsFromConfig_override(t *testing.T) {
	w := RankingWeightsFromConfig(2.0, 0.3, 0, 0, 0, 0.6, 0.7, 0, 0, 0, 0, 0, 0, 0, 0)
	if w.Authority != 2.0 || w.Recency != 0.3 {
		t.Errorf("RankingWeightsFromConfig(2,0.3,...) = Authority %v Recency %v, want 2 and 0.3", w.Authority, w.Recency)
	}
	if w.PatternPriority != 0.6 {
		t.Errorf("RankingWeightsFromConfig(...,0.6,...) PatternPriority = %v, want 0.6", w.PatternPriority)
	}
	if w.LexicalSimilarity != 0.7 {
		t.Errorf("RankingWeightsFromConfig(...,lexical=0.7) LexicalSimilarity = %v, want 0.7", w.LexicalSimilarity)
	}
}

func TestScore_objectLessonPriorityOutranksComparableDecision(t *testing.T) {
	now := time.Now()
	req := ScoreRequest{}
	w := DefaultRankingWeights()
	w.PatternPriority = 0.8
	decision := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 5,
		UpdatedAt: now,
	}
	lesson := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindPattern, Authority: 5,
		UpdatedAt: now,
	}
	if Score(lesson, req, w, 10) <= Score(decision, req, w, 10) {
		t.Fatal("pattern should outrank comparable decision when PatternPriority > 0")
	}
}

func TestDominantReason(t *testing.T) {
	req := ScoreRequest{Tags: []string{"x"}}
	w := DefaultRankingWeights()

	// Tag match dominant when tags overlap
	withTag := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 2,
		UpdatedAt: time.Now(), Tags: []string{"x"},
	}
	got := DominantReason(withTag, req, w, 10)
	if got != "tag_match" {
		t.Logf("DominantReason(tag overlap) = %q (tag_match preferred)", got)
	}

	// Authority dominant when high and no tag/scope
	highAuth := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 10,
		UpdatedAt: time.Now().Add(-400 * 24 * time.Hour), Tags: nil,
	}
	got = DominantReason(highAuth, req, w, 10)
	if got != "authority" {
		t.Errorf("DominantReason(high authority) = %q, want authority", got)
	}
}

func TestPatternScoreFactor(t *testing.T) {
	base := 1.0
	// Negative + high => 2.5
	got := PatternScoreFactor(base, &memory.PatternPayload{Polarity: "negative", Impact: memory.PatternImpact{Severity: "high"}})
	if got != 2.5 {
		t.Errorf("negative+high = %v, want 2.5", got)
	}
	// Positive + medium => 1.2
	got = PatternScoreFactor(base, &memory.PatternPayload{Polarity: "positive", Impact: memory.PatternImpact{Severity: "medium"}})
	if got != 1.2 {
		t.Errorf("positive+medium = %v, want 1.2", got)
	}
	// nil payload => unchanged
	got = PatternScoreFactor(base, nil)
	if got != base {
		t.Errorf("nil payload = %v, want %v", got, base)
	}
}

func TestScore_objectLessonBoost(t *testing.T) {
	w := DefaultRankingWeights()
	req := ScoreRequest{}
	baseObj := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 5,
		UpdatedAt: time.Now(),
	}
	payloadHigh := memory.PatternPayload{Polarity: "negative", Impact: memory.PatternImpact{Severity: "high"}}
	payloadBytes, _ := json.Marshal(payloadHigh)
	objLesson := baseObj
	objLesson.Kind = api.MemoryKindPattern
	objLesson.Statement = "Never skip tests."
	objLesson.Payload = payloadBytes

	sDecision := Score(baseObj, req, w, 10)
	sLesson := Score(objLesson, req, w, 10)
	if sLesson <= sDecision {
		t.Errorf("object lesson (negative+high) score %v should be > ordinary decision %v", sLesson, sDecision)
	}
}
