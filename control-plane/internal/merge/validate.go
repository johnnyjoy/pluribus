package merge

import (
	"context"
	"strings"

	"control-plane/internal/runmulti"
)

// ValidateMerged drift-checks merged text; on failure or empty returns selected output and fallback=true.
func ValidateMerged(ctx context.Context, merged string, check DriftChecker, selected *runmulti.RunResult) (final string, drift runmulti.DriftResult, fallback bool, err error) {
	if strings.TrimSpace(merged) == "" {
		return fallbackOutput(selected), drift, true, nil
	}
	if check == nil {
		return merged, drift, false, nil
	}
	drift, err = check(ctx, merged)
	if err != nil {
		return fallbackOutput(selected), drift, true, err
	}
	if len(drift.Violations) > 0 {
		return fallbackOutput(selected), drift, true, nil
	}
	return merged, drift, false, nil
}

func fallbackOutput(selected *runmulti.RunResult) string {
	if selected == nil {
		return ""
	}
	return selected.Output
}
