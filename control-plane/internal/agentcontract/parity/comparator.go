package parity

import (
	"sort"

	"control-plane/internal/recall"
)

// FieldMismatch records one field-level parity failure.
type FieldMismatch struct {
	MemoryID string `json:"memory_id"`
	Field    string `json:"field"`
	Code     string `json:"code"`
	REST     any    `json:"rest,omitempty"`
	MCP      any    `json:"mcp,omitempty"`
}

// MemoryParityResult is per-memory parity scoring.
type MemoryParityResult struct {
	ParityScore float64 `json:"parity_score"`
}

// ParityResult is the deterministic REST/MCP field parity output.
type ParityResult struct {
	ParityPassed          bool                          `json:"parity_passed"`
	FieldMismatches       []FieldMismatch               `json:"field_mismatches"`
	MissingInREST         []string                      `json:"missing_in_rest"`
	MissingInMCP          []string                      `json:"missing_in_mcp"`
	NormalizedDifferences []string                      `json:"normalized_differences"`
	AllowedOmissions      []string                      `json:"allowed_omissions"`
	MemoryResults         map[string]MemoryParityResult `json:"memory_results"`
}

// CompareMemoryItems compares REST and MCP memory slices field-by-field keyed by memory id.
func CompareMemoryItems(restItems, mcpItems []recall.MemoryItem) ParityResult {
	restByID := indexByID(restItems)
	mcpByID := indexByID(mcpItems)

	out := ParityResult{
		ParityPassed:    true,
		MemoryResults:   map[string]MemoryParityResult{},
		AllowedOmissions: []string{},
	}

	allIDs := map[string]struct{}{}
	for id := range restByID {
		allIDs[id] = struct{}{}
	}
	for id := range mcpByID {
		allIDs[id] = struct{}{}
	}

	for id := range allIDs {
		restIt, restOK := restByID[id]
		mcpIt, mcpOK := mcpByID[id]
		if !restOK {
			out.ParityPassed = false
			out.MissingInREST = append(out.MissingInREST, id)
			out.MemoryResults[id] = MemoryParityResult{ParityScore: 0}
			continue
		}
		if !mcpOK {
			out.ParityPassed = false
			out.MissingInMCP = append(out.MissingInMCP, id)
			out.MemoryResults[id] = MemoryParityResult{ParityScore: 0}
			continue
		}

		mismatches := compareOneMemory(restIt, mcpIt)
		score := 1.0
		if len(mismatches) > 0 {
			out.ParityPassed = false
			score = 0.0
			out.FieldMismatches = append(out.FieldMismatches, mismatches...)
		}
		out.MemoryResults[id] = MemoryParityResult{ParityScore: score}
	}

	sort.Strings(out.MissingInREST)
	sort.Strings(out.MissingInMCP)
	sort.Slice(out.FieldMismatches, func(i, j int) bool {
		if out.FieldMismatches[i].MemoryID == out.FieldMismatches[j].MemoryID {
			return out.FieldMismatches[i].Field < out.FieldMismatches[j].Field
		}
		return out.FieldMismatches[i].MemoryID < out.FieldMismatches[j].MemoryID
	})
	return out
}

func compareOneMemory(restIt, mcpIt recall.MemoryItem) []FieldMismatch {
	var out []FieldMismatch
	for _, field := range ComparedFields {
		rv, rPresent := restFieldValue(restIt, field)
		mv, mPresent := mcpFieldValue(mcpIt, field)

		// Optional omissions: empty optional strings on both sides are allowed.
		if !rPresent && !mPresent {
			continue
		}
		if !rPresent {
			out = append(out, FieldMismatch{
				MemoryID: restIt.ID,
				Field:    field,
				Code:     MismatchMissingFieldInREST,
				MCP:      mv,
			})
			continue
		}
		if !mPresent {
			out = append(out, FieldMismatch{
				MemoryID: restIt.ID,
				Field:    field,
				Code:     MismatchMissingFieldInMCP,
				REST:     rv,
			})
			continue
		}
		if !valuesEqual(field, rv, mv) {
			out = append(out, FieldMismatch{
				MemoryID: restIt.ID,
				Field:    field,
				Code:     fieldMismatchCode(field),
				REST:     rv,
				MCP:      mv,
			})
		}
	}
	return out
}

func indexByID(items []recall.MemoryItem) map[string]recall.MemoryItem {
	out := make(map[string]recall.MemoryItem, len(items))
	for _, it := range items {
		if it.ID == "" {
			continue
		}
		out[it.ID] = it
	}
	return out
}

// FieldMismatchCount returns the number of field mismatches in a parity result.
func FieldMismatchCount(r ParityResult) int {
	return len(r.FieldMismatches)
}

// FieldParityPassRate returns 1.0 when parity passed with zero mismatches.
func FieldParityPassRate(results []ParityResult) float64 {
	if len(results) == 0 {
		return 0
	}
	pass := 0
	for _, r := range results {
		if r.ParityPassed && FieldMismatchCount(r) == 0 {
			pass++
		}
	}
	return float64(pass) / float64(len(results))
}
