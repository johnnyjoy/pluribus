package recall

import (
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

func TestRecallRankingUtilityHelpfulBoost(t *testing.T) {
	w := DefaultRankingWeights()
	req := ScoreRequest{
		SituationQuery: "pluribus recall agent loop compliance",
		Tags:           []string{"pluribus"},
		UtilityScores:  map[uuid.UUID]float64{},
		UtilityWeight:  0.12,
	}
	idHelp := uuid.New()
	idNeutral := uuid.New()
	objs := []memory.MemoryObject{
		{ID: idHelp, Kind: api.MemoryKindPattern, Statement: "pluribus recall agent loop compliance pattern", Authority: 5, Tags: []string{"pluribus"}, UpdatedAt: time.Now()},
		{ID: idNeutral, Kind: api.MemoryKindPattern, Statement: "pluribus recall agent loop compliance pattern", Authority: 5, Tags: []string{"pluribus"}, UpdatedAt: time.Now()},
	}
	req.UtilityScores[idHelp] = 3.0
	req.UtilityScores[idNeutral] = 0

	scored := ScoreAndSortWithReason(objs, req, w, 10)
	if len(scored) < 2 {
		t.Fatal("need 2 scored")
	}
	var helpScore, neutralScore float64
	for _, s := range scored {
		if s.Object.ID == idHelp {
			helpScore = s.Score
		}
		if s.Object.ID == idNeutral {
			neutralScore = s.Score
		}
	}
	if helpScore <= neutralScore {
		t.Fatalf("helpful utility should rank higher: help=%v neutral=%v", helpScore, neutralScore)
	}
}

func TestRecallRankingUtilityWrongDemotion(t *testing.T) {
	w := DefaultRankingWeights()
	req := ScoreRequest{
		SituationQuery: "pluribus enforcement evaluate",
		Tags:           []string{"pluribus"},
		UtilityWeight:  0.12,
	}
	idWrong := uuid.New()
	idNeutral := uuid.New()
	objs := []memory.MemoryObject{
		{ID: idWrong, Kind: api.MemoryKindConstraint, Statement: "pluribus enforcement evaluate before change", Authority: 5, Tags: []string{"pluribus"}, UpdatedAt: time.Now()},
		{ID: idNeutral, Kind: api.MemoryKindConstraint, Statement: "pluribus enforcement evaluate before change", Authority: 5, Tags: []string{"pluribus"}, UpdatedAt: time.Now()},
	}
	req.UtilityScores = map[uuid.UUID]float64{idWrong: -6.0, idNeutral: 0}

	scored := ScoreAndSortWithReason(objs, req, w, 10)
	var wrongScore, neutralScore float64
	for _, s := range scored {
		if s.Object.ID == idWrong {
			wrongScore = s.Score
		}
		if s.Object.ID == idNeutral {
			neutralScore = s.Score
		}
	}
	if wrongScore >= neutralScore {
		t.Fatalf("wrong utility should rank lower: wrong=%v neutral=%v", wrongScore, neutralScore)
	}
}

func TestRecallRankingUtilityDoesNotOverrideRelevance(t *testing.T) {
	w := DefaultRankingWeights()
	req := ScoreRequest{
		SituationQuery: "onguard mobile credential",
		Tags:           []string{"onguard"},
		UtilityWeight:  0.12,
	}
	idWrongDomain := uuid.New()
	idRelevant := uuid.New()
	objs := []memory.MemoryObject{
		{ID: idWrongDomain, Kind: api.MemoryKindDecision, Statement: "pluribus mcp recall compliance", Authority: 9, Tags: []string{"pluribus"}, UpdatedAt: time.Now()},
		{ID: idRelevant, Kind: api.MemoryKindDecision, Statement: "onguard mobile credential reader deployment", Authority: 3, Tags: []string{"onguard"}, UpdatedAt: time.Now()},
	}
	req.UtilityScores = map[uuid.UUID]float64{idWrongDomain: 10.0, idRelevant: -5.0}

	scored := ScoreAndSortWithReason(objs, req, w, 10)
	if scored[0].Object.ID != idRelevant {
		t.Fatalf("relevance should beat utility boost on wrong domain: top=%v", scored[0].Object.Statement)
	}
}

func TestUtilityRankingTermBounded(t *testing.T) {
	if utilityRankingTerm(100, 0.12) > 0.18 {
		t.Fatal("boost capped")
	}
	if utilityRankingTerm(-100, 0.12) < -0.30 {
		t.Fatal("penalty capped")
	}
}
