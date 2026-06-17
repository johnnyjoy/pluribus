package benchmark

import (
	"context"

	"github.com/google/uuid"
)

// utilityScoreStub supplies fixture utility scores for benchmark ranking.
type utilityScoreStub struct {
	scores map[uuid.UUID]float64
}

func (u *utilityScoreStub) GetScoresForMemories(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]float64, error) {
	out := map[uuid.UUID]float64{}
	for _, id := range ids {
		if s, ok := u.scores[id]; ok {
			out[id] = s
		}
	}
	return out, nil
}

func buildUtilityStub(lc *LoadedCorpus) *utilityScoreStub {
	scores := map[uuid.UUID]float64{}
	for id, fix := range lc.IDToFixture {
		if fix.UtilityScore != 0 {
			scores[id] = fix.UtilityScore
		} else if fix.Refuted {
			scores[id] = -0.5
		}
	}
	if len(scores) == 0 {
		return nil
	}
	return &utilityScoreStub{scores: scores}
}

// contradictionExcludeStub excludes refuted fixture memories from recall.
type contradictionExcludeStub struct {
	exclude map[uuid.UUID]bool
}

func (c *contradictionExcludeStub) ListMemoryIDsInUnresolved(_ context.Context) ([]uuid.UUID, error) {
	var out []uuid.UUID
	for id := range c.exclude {
		out = append(out, id)
	}
	return out, nil
}

func buildContradictionStub(lc *LoadedCorpus) *contradictionExcludeStub {
	ex := map[uuid.UUID]bool{}
	for id, fix := range lc.IDToFixture {
		if fix.Refuted {
			ex[id] = true
		}
	}
	if len(ex) == 0 {
		return nil
	}
	return &contradictionExcludeStub{exclude: ex}
}
