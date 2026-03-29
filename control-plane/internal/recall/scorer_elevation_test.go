package recall

import (
	"encoding/json"
	"testing"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

func TestBuildPatternSupersessionMap(t *testing.T) {
	e1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	e2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	p1 := memory.PatternPayload{Polarity: "positive", Experience: "a", Decision: "b", Outcome: "c",
		Impact: memory.PatternImpact{Severity: "medium"}, Directive: "d", SupersededBy: e2.String()}
	b1, _ := json.Marshal(&p1)
	objs := []memory.MemoryObject{
		{ID: e1, Kind: api.MemoryKindPattern, Authority: 5, Payload: b1},
	}
	m := BuildPatternSupersessionMap(objs)
	if m[e1] != e2 {
		t.Fatalf("supersession map: got %v want %v", m[e1], e2)
	}
}

func TestScore_elevationSuppression(t *testing.T) {
	e1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	e2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	p1 := memory.PatternPayload{Polarity: "positive", Experience: "a", Decision: "b", Outcome: "c",
		Impact: memory.PatternImpact{Severity: "medium"}, Directive: "d", SupersededBy: e2.String()}
	b1, _ := json.Marshal(&p1)
	p2 := memory.PatternPayload{Polarity: "positive", Experience: "a2", Decision: "b2", Outcome: "c2",
		Impact: memory.PatternImpact{Severity: "medium"}, Directive: "elevated"}
	b2, _ := json.Marshal(&p2)
	objs := []memory.MemoryObject{
		{ID: e1, Kind: api.MemoryKindPattern, Authority: 7, Payload: b1},
		{ID: e2, Kind: api.MemoryKindPattern, Authority: 8, Payload: b2},
	}
	w := RankingWeights{Authority: 1, Recency: 0, TagMatch: 0, ElevationSuppression: 2.0}
	req := ScoreRequest{
		Supersession: BuildPatternSupersessionMap(objs),
		CandidateSet: BuildCandidateSet(objs),
	}
	s1 := scoreAt(objs[0], req, w, 10, objs[0].UpdatedAt)
	s2 := scoreAt(objs[1], req, w, 10, objs[1].UpdatedAt)
	if s1 >= s2 {
		t.Fatalf("superseded row should score lower than elevated: s1=%v s2=%v", s1, s2)
	}
}
