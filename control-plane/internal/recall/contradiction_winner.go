package recall

import (
	"fmt"

	"control-plane/internal/memory"
	"github.com/google/uuid"

)

// pickContradictionWinnerPair applies creative C3: higher authority, then newer effective time (occurred_at else updated_at), then UUID tie-break.
func pickContradictionWinnerPair(a, b memory.MemoryObject) (winner, loser memory.MemoryObject, reason string) {
	cmp := compareContradictionWinner(a, b)
	if cmp < 0 {
		return a, b, contradictionReason(a, b)
	}
	return b, a, contradictionReason(b, a)
}

func compareContradictionWinner(a, b memory.MemoryObject) int {
	if a.Authority != b.Authority {
		if a.Authority > b.Authority {
			return -1
		}
		return 1
	}
	ta := memory.EffectiveRecencyTime(a)
	tb := memory.EffectiveRecencyTime(b)
	if ta.After(tb) {
		return -1
	}
	if tb.After(ta) {
		return 1
	}
	if a.ID.String() < b.ID.String() {
		return -1
	}
	return 1
}

func contradictionReason(winner, loser memory.MemoryObject) string {
	if winner.Authority != loser.Authority {
		return fmt.Sprintf("authority %d vs %d", winner.Authority, loser.Authority)
	}
	return "newer_effective_time"
}

// filterMemoryObjectsRemovingLosers drops memories whose IDs are in losers.
func filterMemoryObjectsRemovingLosers(objs []memory.MemoryObject, losers map[uuid.UUID]bool) []memory.MemoryObject {
	if len(losers) == 0 {
		return objs
	}
	out := objs[:0]
	for _, o := range objs {
		if !losers[o.ID] {
			out = append(out, o)
		}
	}
	return out
}
