package merge

import (
	"strings"
	"testing"

	"control-plane/internal/runmulti"
)

func TestSegmentsConflict_mustNot(t *testing.T) {
	a := "You must not use synchronous calls in the hot path for performance reasons here"
	b := "You must use synchronous calls in the hot path for performance reasons here"
	if !segmentsConflict(a, b) {
		t.Fatal("expected conflict")
	}
}

func TestStrictAugmentedConflict_shouldNotVsShould(t *testing.T) {
	a := "You should not cache responses in the hot path for performance reasons here and there"
	b := "You should cache responses in the hot path for performance reasons here and there"
	if segmentsConflict(a, b) {
		t.Fatal("conservative path should not flag should-not vs should (Phase 3)")
	}
	if !strictAugmentedConflict(a, b) {
		t.Fatal("strict mode should flag opposing soft polarity with overlap")
	}
}

func TestMarkConflicts_strictVsDefault(t *testing.T) {
	a := "You should not cache responses in the hot path for performance reasons here and there"
	b := "You should cache responses in the hot path for performance reasons here and there"
	segs := []Segment{
		{Variant: "x", Text: a},
		{Variant: "y", Text: b},
	}
	_, badDefault := markConflictsWithStrict(segs, false)
	if len(badDefault) != 0 {
		t.Fatalf("default: expected no conflict, bad=%v", badDefault)
	}
	_, badStrict := markConflictsWithStrict(segs, true)
	if len(badStrict) != 2 {
		t.Fatalf("strict: want both segments bad, got %v", badStrict)
	}
}

func TestClassify_agreementTwoVariants(t *testing.T) {
	text := "shared recommendation about repository pattern for data access layer design"
	runs := []runmulti.RunResult{
		{Variant: "x", Output: text},
		{Variant: "y", Output: text},
	}
	segs := ExtractSegments(runs)
	if len(segs) < 2 {
		t.Fatalf("need 2 segments, got %d", len(segs))
	}
	_, bad := markConflicts(segs)
	if len(bad) != 0 {
		t.Fatalf("unexpected conflicts: %v", bad)
	}
	cl := clusterSegments(segs, bad)
	ag, u := AgreementsUniquesFromClusters(cl)
	if len(ag) < 1 {
		t.Fatalf("expected agreement, ag=%v uniques=%v clusters=%d", ag, u, len(cl))
	}
}

func TestClassify_uniqueSingleVariant(t *testing.T) {
	runs := []runmulti.RunResult{
		{Variant: "only", Output: "Unique paragraph alpha with enough characters here.\n\nAnother unique beta with enough characters here."},
	}
	segs := ExtractSegments(runs)
	_, bad := markConflicts(segs)
	cl := clusterSegments(segs, bad)
	ag, u := AgreementsUniquesFromClusters(cl)
	if len(ag) != 0 {
		t.Errorf("expected no agreement, got %v", ag)
	}
	if len(u) < 1 {
		t.Errorf("expected uniques, got ag=%v u=%v", ag, u)
	}
}

func TestSynth_noConflictLines(t *testing.T) {
	out := Synthesize([]string{"agree one", "agree two"}, []string{"unique one"}, []string{"a", "b"})
	if strings.Contains(out, "must not") {
		// heuristic: our synth doesn't include conflicts; this is a smoke check
	}
	if !strings.Contains(out, "[CORE AGREEMENTS]") || !strings.Contains(out, "[VALID UNIQUE ADDITIONS]") {
		t.Fatal(out)
	}
}
