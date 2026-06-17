package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/internal/recall"
	"control-plane/internal/utility"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

type fixtureCase struct {
	ID                          string   `json:"id"`
	Category                    string   `json:"category"`
	EventType                   string   `json:"event_type"`
	Delta                       float64  `json:"delta"`
	CompareGT                   string   `json:"compare_gt"`
	CompareLT                   struct {
		A string `json:"a"`
		B string `json:"b"`
	} `json:"compare_lt"`
	UtilityA                    float64  `json:"utility_a"`
	UtilityB                    float64  `json:"utility_b"`
	ExpectAHigher               bool     `json:"expect_a_higher"`
	ReinforceOnRecall           bool     `json:"reinforce_on_recall"`
	ExpectReinforce             bool     `json:"expect_reinforce"`
	ReinforceDuplicateAuthority bool     `json:"reinforce_duplicate_authority"`
	ExpectAuthorityBump         bool     `json:"expect_authority_bump"`
	ExpectUtilityBump           bool     `json:"expect_utility_bump"`
	Repetitions                 int      `json:"repetitions"`
	Exposed                     bool     `json:"exposed"`
	RequiresReason              bool     `json:"requires_reason"`
	ExpectNotFound              bool     `json:"expect_not_found"`
	ParityEventTypes            []string `json:"parity_event_types"`
}

type fixtureFile struct {
	Cases []fixtureCase `json:"cases"`
}

func loadCases(t *testing.T) fixtureFile {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "..", "testdata", "memory_utility", "cases.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read cases: %v", err)
	}
	var f fixtureFile
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatalf("parse cases: %v", err)
	}
	return f
}

func TestMemoryUtilityBenchmarkCases(t *testing.T) {
	f := loadCases(t)
	for _, c := range f.Cases {
		t.Run(c.ID, func(t *testing.T) {
			evalCase(t, c)
		})
	}
}

func TestMemoryUtilityBenchmarkGate(t *testing.T) {
	f := loadCases(t)
	var failed int
	for _, c := range f.Cases {
		if err := evalCaseErr(c); err != nil {
			failed++
			t.Errorf("%s: %v", c.ID, err)
		}
	}
	if failed > 0 {
		t.Fatalf("memory utility gate: %d/%d failed", failed, len(f.Cases))
	}
}

func evalCase(t *testing.T, c fixtureCase) {
	t.Helper()
	if err := evalCaseErr(c); err != nil {
		t.Fatal(err)
	}
}

func evalCaseErr(c fixtureCase) error {
	switch c.Category {
	case "positive", "negative", "contradiction":
		if c.EventType != "" {
			w := utility.EventWeight(c.EventType)
			if c.Delta != 0 && w != c.Delta {
				return benchErr(fmt.Sprintf("weight %v want %v", w, c.Delta))
			}
		}
		if c.CompareGT != "" {
			if utility.EventWeight(c.EventType) <= utility.EventWeight(c.CompareGT) {
				return benchErr(fmt.Sprintf("%s not > %s", c.EventType, c.CompareGT))
			}
		}
		if c.CompareLT.A != "" {
			if utility.EventWeight(c.CompareLT.A) >= utility.EventWeight(c.CompareLT.B) {
				return benchErr(fmt.Sprintf("%s not < %s", c.CompareLT.A, c.CompareLT.B))
			}
		}
	case "ranking":
		w := recall.DefaultRankingWeights()
		req := recall.ScoreRequest{
			SituationQuery: "pluribus recall agent loop compliance",
			Tags:           []string{"pluribus"},
			UtilityWeight:  0.12,
		}
		idA := uuid.New()
		idB := uuid.New()
		objs := []memory.MemoryObject{
			{ID: idA, Kind: api.MemoryKindPattern, Statement: "pluribus recall agent loop compliance pattern", Authority: 5, Tags: []string{"pluribus"}, UpdatedAt: time.Now()},
			{ID: idB, Kind: api.MemoryKindPattern, Statement: "pluribus recall agent loop compliance pattern", Authority: 5, Tags: []string{"pluribus"}, UpdatedAt: time.Now()},
		}
		req.UtilityScores = map[uuid.UUID]float64{idA: c.UtilityA, idB: c.UtilityB}
		scored := recall.ScoreAndSortWithReason(objs, req, w, 10)
		var scoreA, scoreB float64
		for _, s := range scored {
			if s.Object.ID == idA {
				scoreA = s.Score
			}
			if s.Object.ID == idB {
				scoreB = s.Score
			}
		}
		if c.ExpectAHigher && scoreA <= scoreB {
			return benchErr(fmt.Sprintf("rank A=%v B=%v expect A higher", scoreA, scoreB))
		}
		if !c.ExpectAHigher && scoreA >= scoreB {
			return benchErr(fmt.Sprintf("rank A=%v B=%v expect A lower", scoreA, scoreB))
		}
	case "false_signal":
		if !c.ReinforceOnRecall && c.ExpectReinforce {
			return benchErr("unsafe default reinforce on recall")
		}
		if !c.ReinforceDuplicateAuthority && c.ExpectAuthorityBump {
			return benchErr("duplicate authority bump without opt-in")
		}
		if c.Repetitions > 1 && c.ExpectAuthorityBump {
			return benchErr("repetition must not promote by default")
		}
		if c.ExpectUtilityBump {
			return benchErr("utility must not auto-increase from recall/duplicate")
		}
	case "interface":
		if c.EventType == "bogus" {
			if utility.ExposedEventTypes[c.EventType] {
				return benchErr("bogus must not be exposed")
			}
			return nil
		}
		if c.Exposed && !utility.ExposedEventTypes[c.EventType] {
			return benchErr(fmt.Sprintf("%s not exposed", c.EventType))
		}
		if c.RequiresReason {
			if err := utility.ValidateFeedbackRequest(utility.FeedbackRequest{EventType: c.EventType}); err != utility.ErrReasonRequired {
				return benchErr(fmt.Sprintf("reason required for %s: %v", c.EventType, err))
			}
		}
		for _, et := range c.ParityEventTypes {
			if !utility.ExposedEventTypes[et] {
				return benchErr(fmt.Sprintf("parity missing %s", et))
			}
		}
	}
	return nil
}

type benchErr string

func (e benchErr) Error() string { return string(e) }
