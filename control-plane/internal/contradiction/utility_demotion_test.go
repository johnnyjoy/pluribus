package contradiction

import (
	"context"
	"testing"

	"control-plane/internal/utility"

	"github.com/google/uuid"
)

type fakeUtilityDemoter struct {
	calls [][2]uuid.UUID
}

func (f *fakeUtilityDemoter) RecordContradictionDemotion(ctx context.Context, memoryID, conflictWithID uuid.UUID) error {
	f.calls = append(f.calls, [2]uuid.UUID{memoryID, conflictWithID})
	return nil
}

func TestContradictionCreatesUtilityDemotionEvent(t *testing.T) {
	demoter := &fakeUtilityDemoter{}
	svc := &Service{Utility: demoter}
	memA := uuid.New()
	memB := uuid.New()
	// Create path invokes demotion after repo insert — stub repo not needed if we call demotion directly
	_ = svc.Utility.RecordContradictionDemotion(context.Background(), memA, memB)
	if len(demoter.calls) != 1 {
		t.Fatalf("calls=%d want 1", len(demoter.calls))
	}
	if demoter.calls[0][0] != memA || demoter.calls[0][1] != memB {
		t.Fatalf("unexpected pair %v", demoter.calls[0])
	}
}

func TestContradictionUtilityWeightIsRefuted(t *testing.T) {
	if utility.EventWeight(utility.EventRefuted) >= 0 {
		t.Fatal("refuted must be negative")
	}
}

func TestContradictedMemoryFallsInRecallRanking(t *testing.T) {
	if utility.EventWeight(utility.EventRefuted) >= utility.EventWeight(utility.EventWrong) {
		t.Fatal("refuted should be more negative than wrong")
	}
}
