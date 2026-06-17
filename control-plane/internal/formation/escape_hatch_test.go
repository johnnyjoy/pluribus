package formation_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"control-plane/internal/formation"
	"control-plane/internal/formationquality"
	"control-plane/pkg/api"
)

func loadEscapeCases(t *testing.T) []formation.EscapeCase {
	t.Helper()
	f, err := formation.LoadEscapeCases(formation.DefaultEscapeCasesDir())
	if err != nil {
		t.Fatal(err)
	}
	return f.Cases
}

func findEscape(t *testing.T, cases []formation.EscapeCase, id string) formation.EscapeCase {
	t.Helper()
	for _, c := range cases {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("case %q not found", id)
	return formation.EscapeCase{}
}

func gateEval(t *testing.T, c formation.EscapeCase) (formation.Decision, error) {
	t.Helper()
	g := formation.NewGate(nil)
	in := formation.InputFromEscapeCase(c)
	return g.EvaluateCreateInput(in)
}

func TestFormationQualityFixtureLoads(t *testing.T) {
	if len(loadEscapeCases(t)) < 18 {
		t.Fatalf("want >= 18 escape cases")
	}
}

func TestPromoteAppliesFormationQualityLayer(t *testing.T) {
	c := findEscape(t, loadEscapeCases(t), "promote_valid_cue_rich_candidate_accepted")
	d, err := gateEval(t, c)
	if err != nil {
		t.Fatal(err)
	}
	if d.Quality == nil {
		t.Fatal("promote must run quality layer")
	}
}

func TestPromoteRejectsVagueCandidate(t *testing.T) {
	c := findEscape(t, loadEscapeCases(t), "promote_vague_candidate_rejected")
	_, err := gateEval(t, c)
	if err == nil {
		t.Fatal("expected reject")
	}
}

func TestPromoteBlocksRefutedActiveGuidance(t *testing.T) {
	c := findEscape(t, loadEscapeCases(t), "promote_refuted_active_rejected")
	_, err := gateEval(t, c)
	if err == nil {
		t.Fatal("expected reject")
	}
}

func TestPromoteBlocksSupersededActiveGuidance(t *testing.T) {
	c := findEscape(t, loadEscapeCases(t), "promote_superseded_active_rejected")
	_, err := gateEval(t, c)
	if err == nil {
		t.Fatal("expected reject")
	}
}

func TestPromoteKeepsUnderEncodedCandidatePending(t *testing.T) {
	c := findEscape(t, loadEscapeCases(t), "promote_under_encoded_candidate_pending")
	d, err := gateEval(t, c)
	if err != nil {
		t.Fatal(err)
	}
	if d.Outcome != formation.OutcomePending {
		t.Fatalf("outcome=%s", d.Outcome)
	}
}

func TestPromoteRequiresScopeForConstraint(t *testing.T) {
	c := findEscape(t, loadEscapeCases(t), "promote_constraint_without_scope_pending")
	d, err := gateEval(t, c)
	if err != nil {
		t.Fatal(err)
	}
	if d.Outcome != formation.OutcomePending {
		t.Fatalf("outcome=%s", d.Outcome)
	}
}

func TestPromoteRequiresProvenanceForHighAuthority(t *testing.T) {
	c := findEscape(t, loadEscapeCases(t), "promote_high_authority_without_provenance_pending")
	d, err := gateEval(t, c)
	if err != nil {
		t.Fatal(err)
	}
	if d.Outcome != formation.OutcomePending {
		t.Fatalf("outcome=%s", d.Outcome)
	}
}

func TestPromoteDoesNotRegressValidCandidatePromotion(t *testing.T) {
	c := findEscape(t, loadEscapeCases(t), "promote_valid_cue_rich_candidate_accepted")
	d, err := gateEval(t, c)
	if err != nil {
		t.Fatal(err)
	}
	if d.Outcome == formation.OutcomeReject {
		t.Fatal("valid promote regressed")
	}
}

func TestProbationaryIngestAppliesFormationQualityLayer(t *testing.T) {
	c := findEscape(t, loadEscapeCases(t), "probationary_valid_advisory_accepted")
	g := formation.NewGate(nil)
	in := &formation.CreateInput{
		Path:      formation.PathProbationaryIngest,
		Kind:      api.MemoryKindFailure,
		Authority: 2,
		Statement: c.Input.Statement,
	}
	d, err := g.EvaluateCreateInput(in)
	if err != nil {
		t.Fatal(err)
	}
	if d.Quality == nil {
		t.Fatal("probationary must run quality layer")
	}
}

func TestProbationaryIngestRejectsVagueSummary(t *testing.T) {
	c := findEscape(t, loadEscapeCases(t), "probationary_vague_rejected")
	_, err := gateEval(t, c)
	if err == nil {
		t.Fatal("expected reject")
	}
}

func TestProbationaryIngestMarksUnderEncodedPending(t *testing.T) {
	c := findEscape(t, loadEscapeCases(t), "probationary_under_encoded_pending")
	d, err := gateEval(t, c)
	if err != nil {
		t.Fatal(err)
	}
	if d.Outcome != formation.OutcomePending {
		t.Fatalf("outcome=%s", d.Outcome)
	}
}

func TestProbationaryIngestPreventsUnsafeInfluence(t *testing.T) {
	c := findEscape(t, loadEscapeCases(t), "probationary_agent_inferred_high_authority_blocked")
	in := escapeInput(t, c)
	if in.Authority > 2 {
		t.Fatalf("authority=%d want <=2", in.Authority)
	}
}

func TestProbationaryIngestAllowsConcreteAdvisoryExperience(t *testing.T) {
	c := findEscape(t, loadEscapeCases(t), "probationary_valid_advisory_accepted")
	d, err := gateEval(t, c)
	if err != nil {
		t.Fatal(err)
	}
	if d.Outcome == formation.OutcomeReject {
		t.Fatal("concrete advisory rejected")
	}
}

func TestProbationaryIngestQualityScoreRecordedOrReturned(t *testing.T) {
	c := findEscape(t, loadEscapeCases(t), "probationary_valid_advisory_accepted")
	d, err := gateEval(t, c)
	if err != nil {
		t.Fatal(err)
	}
	if d.Quality == nil || d.Quality.QualityScore <= 0 {
		t.Fatalf("quality=%+v", d.Quality)
	}
}

func TestProbationaryIngestDoesNotRegressRecordExperience(t *testing.T) {
	g := formation.NewGate(nil)
	stmt := "Build failed because memory_create allowed authority 10 governing constraints without review."
	reject, _ := g.RejectRecordExperienceSummary(stmt)
	if reject {
		t.Fatal("concrete failure summary rejected")
	}
}

func TestFormationEscapeHatchBenchmarkArtifact(t *testing.T) {
	if os.Getenv("FORMATION_ESCAPE_HATCH_BENCHMARK") != "1" {
		t.Skip("set FORMATION_ESCAPE_HATCH_BENCHMARK=1")
	}
	report := formation.RunEscapeBenchmark(formation.NewGate(nil), loadEscapeCases(t))
	if err := formation.WriteEscapeReport(&report, escapeArtifactPath()); err != nil {
		t.Fatal(err)
	}
}

func TestFormationEscapeHatchGate(t *testing.T) {
	if os.Getenv("FORMATION_ESCAPE_HATCH_GATE") != "1" && os.Getenv("PROOF_FORMATION_ESCAPE_HATCHES") != "1" {
		t.Skip("set FORMATION_ESCAPE_HATCH_GATE=1")
	}
	report := formation.RunEscapeBenchmark(formation.NewGate(nil), loadEscapeCases(t))
	if err := formation.WriteEscapeReport(&report, escapeArtifactPath()); err != nil {
		t.Fatal(err)
	}
	if !report.GatePassed {
		t.Fatalf("gate failed: %v summary=%+v", report.GateFailures, report.Summary)
	}
}

func TestDirectCreateRegressionStillProtected(t *testing.T) {
	report := formation.RunEscapeBenchmark(formation.NewGate(nil), loadEscapeCases(t))
	for _, c := range report.Cases {
		if c.Path != "direct_create" {
			continue
		}
		if !c.Passed {
			t.Fatalf("direct create regression: %s", c.ID)
		}
	}
}

func escapeInput(t *testing.T, c formation.EscapeCase) *formation.CreateInput {
	t.Helper()
	g := formation.NewGate(nil)
	in := formation.InputFromEscapeCase(c)
	_, err := g.EvaluateCreateInput(in)
	if err != nil && c.ExpectedGateOutcome != "reject" {
		t.Fatal(err)
	}
	return in
}

func escapeArtifactPath() string {
	if p := os.Getenv("FORMATION_ESCAPE_HATCH_ARTIFACT"); p != "" {
		return p
	}
	return filepath.Join(repoRootEscape(), "artifacts", "formation-escape-hatch-benchmark.json")
}

func repoRootEscape() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	dir := filepath.Dir(file)
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "control-plane", "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}

func TestFormationQualityScoreDeterministicEscape(t *testing.T) {
	eval := formationquality.NewEvaluator()
	in := formationquality.Input{
		Path: "record_experience", Kind: "failure",
		Statement: "Build failed because memory_create allowed authority 10 governing constraints without review.",
		Authority: 2,
	}
	r1 := eval.Evaluate(in)
	r2 := eval.Evaluate(in)
	if r1.QualityScore != r2.QualityScore || r1.Decision != r2.Decision {
		t.Fatalf("non-deterministic %+v vs %+v", r1, r2)
	}
}
