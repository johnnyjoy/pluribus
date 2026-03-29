package recall

// EstimateTokensForItems returns an approximate token count for the items (statement text).
// Uses ~4 runes per token as a simple heuristic.
func EstimateTokensForItems(items []MemoryItem) int {
	var n int
	for _, it := range items {
		n += len([]rune(it.Statement)) / 4
		if n < 0 {
			return 0
		}
	}
	return n
}

// ApplyRIELimits enforces max_total and max_tokens over recall buckets.
// Order: constraints, decisions, failures, patterns.
// Zero means no cap for that limit.
func ApplyRIELimits(con, dec, fail, pat []MemoryItem, maxTotal, maxTokens int) (conOut, decOut, failOut, patOut []MemoryItem) {
	flat := make([]MemoryItem, 0, len(con)+len(dec)+len(fail)+len(pat))
	flat = append(flat, con...)
	flat = append(flat, dec...)
	flat = append(flat, fail...)
	flat = append(flat, pat...)
	if maxTotal > 0 && len(flat) > maxTotal {
		flat = flat[:maxTotal]
	}
	if maxTokens > 0 {
		for len(flat) > 0 && EstimateTokensForItems(flat) > maxTokens {
			flat = flat[:len(flat)-1]
		}
	}
	for _, it := range flat {
		switch it.Kind {
		case "constraint":
			conOut = append(conOut, it)
		case "decision":
			decOut = append(decOut, it)
		case "failure":
			failOut = append(failOut, it)
		case "pattern":
			patOut = append(patOut, it)
		}
	}
	return
}
