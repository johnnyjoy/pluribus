package recall

import (
	"testing"
	"time"

	"control-plane/internal/memory"

	"github.com/google/uuid"
)

func TestCompareContradictionWinner_authority(t *testing.T) {
	a := memory.MemoryObject{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Authority: 9}
	b := memory.MemoryObject{ID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), Authority: 5}
	if compareContradictionWinner(a, b) >= 0 {
		t.Fatal("higher authority should win (cmp < 0 means first wins)")
	}
	if compareContradictionWinner(b, a) <= 0 {
		t.Fatal("lower authority should lose when first")
	}
}

func TestCompareContradictionWinner_noProjectScopePreference(t *testing.T) {
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	global := memory.MemoryObject{
		ID: uuid.MustParse("10000000-0000-0000-0000-000000000001"), Authority: 5,
		UpdatedAt: t1,
	}
	project := memory.MemoryObject{
		ID: uuid.MustParse("20000000-0000-0000-0000-000000000001"), Authority: 5,
		UpdatedAt: t1,
	}
	// With same authority and timestamp, ordering is UUID tie-break only.
	if compareContradictionWinner(project, global) <= 0 {
		t.Fatal("expected UUID tie-break by ID ordering, not project/global preference")
	}
	if compareContradictionWinner(global, project) >= 0 {
		t.Fatal("expected UUID tie-break symmetry by ID ordering, not project/global preference")
	}
}

func TestCompareContradictionWinner_newerUpdatedAt(t *testing.T) {
	oldT := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	newT := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	a := memory.MemoryObject{
		ID: uuid.MustParse("30000000-0000-0000-0000-000000000001"), Authority: 5,
		UpdatedAt: newT,
	}
	b := memory.MemoryObject{
		ID: uuid.MustParse("40000000-0000-0000-0000-000000000001"), Authority: 5,
		UpdatedAt: oldT,
	}
	if compareContradictionWinner(a, b) >= 0 {
		t.Fatal("newer UpdatedAt should win")
	}
}

func TestCompareContradictionWinner_uuidTieBreak(t *testing.T) {
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	lowUUID := memory.MemoryObject{
		ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), Authority: 5,
		UpdatedAt: ts,
	}
	highUUID := memory.MemoryObject{
		ID: uuid.MustParse("ffff0000-0000-0000-0000-000000000001"), Authority: 5,
		UpdatedAt: ts,
	}
	if compareContradictionWinner(lowUUID, highUUID) >= 0 {
		t.Fatal("lexicographically smaller UUID should win when all else equal")
	}
}

func TestPickContradictionWinnerPair_ordering(t *testing.T) {
	wID := uuid.MustParse("50000000-0000-0000-0000-000000000001")
	lID := uuid.MustParse("60000000-0000-0000-0000-000000000001")
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	weak := memory.MemoryObject{ID: lID, Authority: 3, UpdatedAt: ts}
	strong := memory.MemoryObject{ID: wID, Authority: 9, UpdatedAt: ts}
	win, lose, reason := pickContradictionWinnerPair(weak, strong)
	if win.ID != wID || lose.ID != lID {
		t.Fatalf("want winner strong, loser weak; got win=%s lose=%s reason=%q", win.ID, lose.ID, reason)
	}
	// argument order should not change winner
	win2, lose2, _ := pickContradictionWinnerPair(strong, weak)
	if win2.ID != wID || lose2.ID != lID {
		t.Fatalf("winner should be independent of pair order; got win=%s lose=%s", win2.ID, lose2.ID)
	}
}
