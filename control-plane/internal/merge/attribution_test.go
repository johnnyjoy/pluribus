package merge

import (
	"testing"

	"control-plane/internal/runmulti"
)

func TestBuildMergeDebug_multiPath_hasAttribution(t *testing.T) {
	overlap := "We recommend using the repository pattern for data access with clear boundaries and sufficient length."
	runs := []runmulti.RunResult{
		{Variant: "v1", Output: overlap + "\n\nExtra unique detail from variant one here.", Rejected: false, Drift: runmulti.DriftResult{}},
		{Variant: "v2", Output: overlap + "\n\nDifferent unique insight from variant two here.", Rejected: false, Drift: runmulti.DriftResult{}},
	}
	valid := filterValid(runs)
	segments := ExtractSegments(valid)
	conflictTexts, bad := markConflicts(segments)
	clusters := clusterSegments(segments, bad)
	agreements, uniques := AgreementsUniquesFromClusters(clusters)
	dbg := buildMergeDebug(segments, bad, clusters, agreements, uniques, conflictTexts, 0)

	if dbg.SegmentsIn != len(segments) {
		t.Fatalf("segments_in: got %d want %d", dbg.SegmentsIn, len(segments))
	}
	if dbg.ConflictsDropped != len(conflictTexts) {
		t.Fatalf("conflicts_dropped: got %d want %d", dbg.ConflictsDropped, len(conflictTexts))
	}
	if len(agreements) > 0 {
		found := false
		for _, line := range dbg.Attribution {
			if line.Role == AttributionAgreement {
				found = true
				if len(line.Variants) < 2 {
					t.Fatalf("agreement should list 2+ variants: %+v", line)
				}
			}
		}
		if !found {
			t.Fatal("expected at least one agreement attribution line")
		}
	}
}
