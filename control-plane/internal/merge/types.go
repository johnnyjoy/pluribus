// Package merge implements deterministic merge + synthesis of multi-run outputs.
package merge

import (
	"context"

	"control-plane/internal/runmulti"
)

// DriftChecker performs drift validation (e.g. POST /v1/drift/check).
type DriftChecker func(ctx context.Context, proposal string) (runmulti.DriftResult, error)

// Logger is optional; nil = no-op.
type Logger interface {
	Printf(format string, args ...interface{})
}

// EngineInput is the input to Run.
type EngineInput struct {
	Runs       []runmulti.RunResult
	Selected   *runmulti.RunResult
	DriftCheck DriftChecker
	Log        Logger
	// Options optional Phase 5.2 uniques cap / dedupe; nil = Phase 3 defaults.
	Options *MergeOptions
}

// Segment is a slice of output text attributed to a variant.
type Segment struct {
	Text    string
	Variant string
}

// SegmentClass classifies a segment for debugging (optional use in tests).
type SegmentClass int

const (
	Agreement SegmentClass = iota
	Unique
	Conflict
)

// MergeResult is the outcome of merge + optional drift validation.
type MergeResult struct {
	MergedOutput string               `json:"merged_output"`
	Drift        runmulti.DriftResult `json:"drift,omitempty"`

	UsedVariants []string `json:"used_variants,omitempty"`
	Agreements   []string `json:"agreements,omitempty"`
	Unique       []string `json:"unique,omitempty"`
	Conflicts    []string `json:"conflicts,omitempty"`

	FallbackUsed bool      `json:"fallback_used"`
	Debug        MergeDebug `json:"debug"` // segments, conflicts count, attribution
}
