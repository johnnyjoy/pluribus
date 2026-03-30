package memory

import (
	"testing"
	"time"

	"control-plane/pkg/api"

	"github.com/google/uuid"
)

func TestEffectiveRecencyTime_occurredAtPreferred(t *testing.T) {
	up := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ev := time.Date(2019, 6, 1, 0, 0, 0, 0, time.UTC)
	m := MemoryObject{ID: uuid.New(), Kind: api.MemoryKindDecision, UpdatedAt: up, OccurredAt: &ev}
	if !EffectiveRecencyTime(m).Equal(ev) {
		t.Fatalf("effective = %v want %v", EffectiveRecencyTime(m), ev)
	}
}

func TestEffectiveRecencyTime_fallsBackToUpdatedAt(t *testing.T) {
	up := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	m := MemoryObject{ID: uuid.New(), Kind: api.MemoryKindDecision, UpdatedAt: up}
	if !EffectiveRecencyTime(m).Equal(up) {
		t.Fatalf("effective = %v want %v", EffectiveRecencyTime(m), up)
	}
}
