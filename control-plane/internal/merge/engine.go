package merge

import (
	"context"
	"strings"

	"control-plane/internal/runmulti"
)

// Run executes merge pipeline: valid runs → segment → classify → synth → drift → fallback.
func Run(ctx context.Context, in EngineInput) MergeResult {
	logf := func(format string, args ...interface{}) {
		if in.Log != nil {
			in.Log.Printf(format, args...)
		}
	}
	logf("merge: input runs=%d", len(in.Runs))

	valid := filterValid(in.Runs)
	logf("merge: valid runs=%d", len(valid))

	selOut := ""
	if in.Selected != nil {
		selOut = in.Selected.Output
	}

	if len(valid) == 0 {
		logf("merge: fallback (0 valid runs)")
		return MergeResult{
			MergedOutput: selOut,
			FallbackUsed: true,
			Debug:        MergeDebug{SegmentsIn: 0, ConflictsDropped: 0, UniquesCapped: 0},
		}
	}

	if len(valid) == 1 {
		merged := strings.TrimSpace(valid[0].Output)
		final, drift, fb, err := ValidateMerged(ctx, merged, in.DriftCheck, in.Selected)
		if err != nil {
			logf("merge: single valid drift error: %v", err)
		}
		if fb {
			logf("merge: fallback after single-run drift")
			return MergeResult{
				MergedOutput: final,
				Drift:        drift,
				UsedVariants: []string{valid[0].Variant},
				FallbackUsed: true,
				Debug:        MergeDebugFromRuns(valid),
			}
		}
		return MergeResult{
			MergedOutput: final,
			Drift:        drift,
			UsedVariants: []string{valid[0].Variant},
			FallbackUsed: false,
			Debug:        MergeDebugFromRuns(valid),
		}
	}

	segments := ExtractSegments(valid)
	strict := false
	if in.Options != nil {
		strict = in.Options.StrictConflicts
	}
	conflictTexts, bad := markConflictsWithStrict(segments, strict)
	logf("merge: segments=%d conflicts_marked=%d", len(segments), len(bad))

	clusters := clusterSegments(segments, bad)
	agreements, rawUniques := AgreementsUniquesFromClusters(clusters)
	uniques, uCap := applyUniquesPipeline(rawUniques, agreements, in.Options)
	used := UsedVariantsFromClusters(clusters, uniques)
	logf("merge: agreements=%d uniques=%d (raw=%d) uniques_capped=%d variants=%v", len(agreements), len(uniques), len(rawUniques), uCap, used)

	merged := Synthesize(agreements, uniques, used)
	dbg := buildMergeDebug(segments, bad, clusters, agreements, uniques, conflictTexts, uCap)
	final, drift, fb, err := ValidateMerged(ctx, merged, in.DriftCheck, in.Selected)
	if err != nil {
		logf("merge: drift check error: %v", err)
	}
	if fb {
		logf("merge: fallback (empty merged or drift violations)")
		return MergeResult{
			MergedOutput: fallbackOutput(in.Selected),
			Drift:        drift,
			UsedVariants: used,
			Agreements:   agreements,
			Unique:       uniques,
			Conflicts:    conflictTexts,
			FallbackUsed: true,
			Debug:        dbg,
		}
	}

	return MergeResult{
		MergedOutput: final,
		Drift:        drift,
		UsedVariants: used,
		Agreements:   agreements,
		Unique:       uniques,
		Conflicts:    conflictTexts,
		FallbackUsed: false,
		Debug:        dbg,
	}
}

func filterValid(runs []runmulti.RunResult) []runmulti.RunResult {
	var out []runmulti.RunResult
	for _, r := range runs {
		if r.Rejected {
			continue
		}
		if len(r.Drift.Violations) > 0 {
			continue
		}
		out = append(out, r)
	}
	return out
}
