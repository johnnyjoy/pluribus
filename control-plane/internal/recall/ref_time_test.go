package recall

import (
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

func TestRefTimeForRanking_maxUpdatedAt(t *testing.T) {
	t1 := time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	objs := []memory.MemoryObject{
		{ID: uuid.New(), Kind: api.MemoryKindDecision, UpdatedAt: t1, Status: api.StatusActive},
		{ID: uuid.New(), Kind: api.MemoryKindDecision, UpdatedAt: t2, Status: api.StatusActive},
	}
	got := RefTimeForRanking(objs)
	if !got.Equal(t2) {
		t.Fatalf("RefTimeForRanking = %v want %v", got, t2)
	}
}

func TestRefTimeForRanking_allZeroUsesEpoch(t *testing.T) {
	objs := []memory.MemoryObject{
		{ID: uuid.New(), Kind: api.MemoryKindDecision, Status: api.StatusActive},
	}
	got := RefTimeForRanking(objs)
	if !got.Equal(epochRefTime) {
		t.Fatalf("got %v want epoch", got)
	}
}
