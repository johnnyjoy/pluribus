package recall

import (
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/internal/memorynorm"
	"control-plane/internal/similarity"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

func TestCollapseScoredForRecall_exactKeyKeepsFirst(t *testing.T) {
	stmt := "Do not skip tests"
	sk := memorynorm.StatementKey(stmt)
	canon := memorynorm.StatementCanonical(stmt)
	now := time.Now()
	id1 := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	id2 := uuid.MustParse("20000000-0000-0000-0000-000000000001")
	scored := []ScoredMemory{
		{Object: memory.MemoryObject{ID: id1, Kind: api.MemoryKindConstraint, Statement: stmt, StatementCanonical: canon, StatementKey: sk, Authority: 9, UpdatedAt: now}, Score: 1, Reason: "ok"},
		{Object: memory.MemoryObject{ID: id2, Kind: api.MemoryKindConstraint, Statement: stmt, StatementCanonical: canon, StatementKey: sk, Authority: 5, UpdatedAt: now}, Score: 1, Reason: "ok"},
	}
	out := collapseScoredForRecall(scored, 0)
	if len(out) != 1 || out[0].Object.ID != id1 {
		t.Fatalf("got %d items, first id %v; want 1 item id1", len(out), out[0].Object.ID)
	}
}

func TestCollapseScoredForRecall_nearDupThreshold(t *testing.T) {
	now := time.Now()
	// High token overlap, distinct keys; Jaccard ≥ 0.85 on canonical text.
	s1 := "one two three four five six"
	s2 := "one two three four five six seven"
	j := similarity.CanonicalTokenJaccard(memorynorm.StatementCanonical(s1), memorynorm.StatementCanonical(s2))
	if j < 0.85 {
		t.Fatalf("fixture Jaccard %v too low for test", j)
	}
	id1 := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	id2 := uuid.MustParse("20000000-0000-0000-0000-000000000001")
	scored := []ScoredMemory{
		{Object: memory.MemoryObject{ID: id1, Kind: api.MemoryKindDecision, Statement: s1, StatementCanonical: memorynorm.StatementCanonical(s1), StatementKey: memorynorm.StatementKey(s1), Authority: 5, UpdatedAt: now}, Score: 1, Reason: "ok"},
		{Object: memory.MemoryObject{ID: id2, Kind: api.MemoryKindDecision, Statement: s2, StatementCanonical: memorynorm.StatementCanonical(s2), StatementKey: memorynorm.StatementKey(s2), Authority: 5, UpdatedAt: now}, Score: 1, Reason: "ok"},
	}
	out := collapseScoredForRecall(scored, 0.85)
	if len(out) != 1 || out[0].Object.ID != id1 {
		t.Fatalf("got %d items, want first only; j=%v", len(out), j)
	}
}

func TestCollapseScoredForRecall_differentKindsIndependent(t *testing.T) {
	stmt := "shared text"
	sk := memorynorm.StatementKey(stmt)
	canon := memorynorm.StatementCanonical(stmt)
	now := time.Now()
	scored := []ScoredMemory{
		{Object: memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindConstraint, Statement: stmt, StatementCanonical: canon, StatementKey: sk, Authority: 5, UpdatedAt: now}, Score: 1, Reason: "ok"},
		{Object: memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindDecision, Statement: stmt, StatementCanonical: canon, StatementKey: sk, Authority: 5, UpdatedAt: now}, Score: 1, Reason: "ok"},
	}
	out := collapseScoredForRecall(scored, 0)
	if len(out) != 2 {
		t.Fatalf("want one per kind, got %d", len(out))
	}
}
