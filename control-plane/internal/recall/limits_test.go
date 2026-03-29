package recall

import (
	"testing"
)

func TestEstimateTokensForItems(t *testing.T) {
	items := []MemoryItem{
		{Statement: "hello"},                    // ~1 token
		{Statement: "one two three four"},        // ~5 tokens
		{Statement: "x"},                         // 0 (len/4)
	}
	got := EstimateTokensForItems(items)
	// 5 runes/4 + 19/4 + 1/4 = 1 + 4 + 0 = 5
	if got < 1 || got > 10 {
		t.Errorf("EstimateTokensForItems = %d, expected rough token count 1-10", got)
	}
}

func TestApplyRIELimits_maxTotal(t *testing.T) {
	con := []MemoryItem{{ID: "1", Kind: "constraint", Statement: "c1"}, {ID: "2", Kind: "constraint", Statement: "c2"}}
	dec := []MemoryItem{{ID: "3", Kind: "decision", Statement: "d1"}}
	fail := []MemoryItem{}
	pat := []MemoryItem{}
	co, do, fo, po := ApplyRIELimits(con, dec, fail, pat, 2, 0)
	if len(co)+len(do)+len(fo)+len(po) != 2 {
		t.Errorf("max_total=2: got total %d items, want 2", len(co)+len(do)+len(fo)+len(po))
	}
	if len(co) != 2 || len(do) != 0 {
		t.Errorf("max_total=2: want 2 constraints, 0 decisions (order: con then dec); got %d con, %d dec", len(co), len(do))
	}
}

func TestApplyRIELimits_noCap(t *testing.T) {
	con := []MemoryItem{{Kind: "constraint", Statement: "a"}}
	dec := []MemoryItem{{Kind: "decision", Statement: "b"}}
	co, do, _, _ := ApplyRIELimits(con, dec, nil, nil, 0, 0)
	if len(co) != 1 || len(do) != 1 {
		t.Errorf("no cap: want 1 con, 1 dec; got %d con, %d dec", len(co), len(do))
	}
}

func TestApplyRIELimits_maxTokens(t *testing.T) {
	// Each statement ~25 runes -> ~6 tokens. 4 items = ~24 tokens. Cap at 10 tokens -> drop to 1 item (~6 tokens).
	con := []MemoryItem{{ID: "1", Kind: "constraint", Statement: "constraint one two three four five"}}
	dec := []MemoryItem{{ID: "2", Kind: "decision", Statement: "decision one two three four five"}}
	fail := []MemoryItem{{ID: "3", Kind: "failure", Statement: "failure one two three four five"}}
	pat := []MemoryItem{{ID: "4", Kind: "pattern", Statement: "pattern one two three four five"}}
	co, do, fo, po := ApplyRIELimits(con, dec, fail, pat, 0, 10)
	total := len(co) + len(do) + len(fo) + len(po)
	tokens := EstimateTokensForItems(append(append(append(append([]MemoryItem{}, co...), do...), fo...), po...))
	if tokens > 15 {
		t.Errorf("max_tokens=10: got %d estimated tokens (total %d items), should have trimmed", tokens, total)
	}
}
