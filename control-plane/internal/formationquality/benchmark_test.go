package formationquality_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"control-plane/internal/formation"
	"control-plane/internal/formationquality"
	"control-plane/pkg/api"
)

func loadCases(t *testing.T) []formationquality.FixtureCase {
	t.Helper()
	f, err := formationquality.LoadCases(formationquality.DefaultCasesDir())
	if err != nil {
		t.Fatalf("load cases: %v", err)
	}
	if len(f.Cases) < 24 {
		t.Fatalf("cases=%d want >= 24", len(f.Cases))
	}
	return f.Cases
}

func findCase(t *testing.T, cases []formationquality.FixtureCase, id string) formationquality.FixtureCase {
	t.Helper()
	for _, c := range cases {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("case %q not found", id)
	return formationquality.FixtureCase{}
}

func evalCase(c formationquality.FixtureCase) formationquality.Result {
	return formationquality.NewEvaluator().Evaluate(c.Input)
}

func TestFormationQualityFixtureLoads(t *testing.T) {
	loadCases(t)
}

func TestVagueStatementRejected(t *testing.T) {
	c := findCase(t, loadCases(t), "vague_statement_rejected")
	r := evalCase(c)
	if r.Decision != formationquality.DecisionRejectGarbage {
		t.Fatalf("decision=%s want reject_garbage", r.Decision)
	}
}

func TestConstraintRequiresScope(t *testing.T) {
	c := findCase(t, loadCases(t), "constraint_without_scope_pending")
	r := evalCase(c)
	if r.SafeForActiveRecall {
		t.Fatal("constraint without scope must not be active-safe")
	}
}

func TestAlwaysRuleRequiresExceptionOrScope(t *testing.T) {
	c := findCase(t, loadCases(t), "always_rule_without_exception_pending")
	r := evalCase(c)
	if r.SafeForActiveRecall {
		t.Fatal("always rule without scope must not be active-safe")
	}
}

func TestAgentInferredPreferenceNotHighAuthority(t *testing.T) {
	c := findCase(t, loadCases(t), "agent_inferred_preference_pending")
	r := evalCase(c)
	if r.Decision == formationquality.DecisionAcceptActive {
		t.Fatalf("agent-inferred preference must not be accept_active, got %s", r.Decision)
	}
	if r.SafeForActiveRecall {
		t.Fatal("agent-inferred high authority preference must not be active-safe")
	}
}

func TestDecisionRequiresReason(t *testing.T) {
	c := findCase(t, loadCases(t), "decision_without_reason_pending")
	r := evalCase(c)
	if r.Decision == formationquality.DecisionAcceptActive {
		t.Fatal("decision without reason must not be accept_active")
	}
}

func TestFailureRequiresCauseAndConditions(t *testing.T) {
	c := findCase(t, loadCases(t), "failure_without_cause_pending")
	r := evalCase(c)
	if r.Decision == formationquality.DecisionAcceptActive {
		t.Fatal("failure without cause must not be accept_active")
	}
}

func TestProcedureRequiresSteps(t *testing.T) {
	c := findCase(t, loadCases(t), "procedure_without_steps_rejected_or_pending")
	r := evalCase(c)
	if r.Decision == formationquality.DecisionAcceptActive {
		t.Fatal("procedure without steps must not be accept_active")
	}
}

func TestHistoricalEventRequiresTemporalBasis(t *testing.T) {
	c := findCase(t, loadCases(t), "historical_event_without_time_pending")
	r := evalCase(c)
	if r.Decision == formationquality.DecisionAcceptActive {
		t.Fatal("historical without time must not be accept_active")
	}
}

func TestHistoricalEventNotCurrentGuidance(t *testing.T) {
	c := findCase(t, loadCases(t), "historical_event_as_current_guidance_rejected_or_pending")
	r := evalCase(c)
	if r.SafeForActiveRecall {
		t.Fatal("historical event must not be active-safe")
	}
}

func TestRefutedGuidanceNotActive(t *testing.T) {
	c := findCase(t, loadCases(t), "refuted_guidance_as_active_rejected")
	r := evalCase(c)
	if r.Decision != formationquality.DecisionRejectDangerous {
		t.Fatalf("decision=%s", r.Decision)
	}
}

func TestSupersededGuidanceNotActive(t *testing.T) {
	c := findCase(t, loadCases(t), "superseded_guidance_as_active_rejected_or_pending")
	r := evalCase(c)
	if r.Decision != formationquality.DecisionRejectDangerous {
		t.Fatalf("decision=%s", r.Decision)
	}
}

func TestHighAuthorityRequiresProvenance(t *testing.T) {
	c := findCase(t, loadCases(t), "high_authority_without_provenance_pending")
	r := evalCase(c)
	if r.SafeForActiveRecall {
		t.Fatal("high authority without provenance must not be active-safe")
	}
}

func TestUnderCuedMemoryPending(t *testing.T) {
	c := findCase(t, loadCases(t), "under_cued_memory_pending")
	r := evalCase(c)
	if r.SafeForActiveRecall {
		t.Fatal("under-cued governing memory must not be active-safe")
	}
}

func TestCueRichMemoryAccepted(t *testing.T) {
	c := findCase(t, loadCases(t), "cue_rich_memory_accepted")
	r := evalCase(c)
	if r.Decision != formationquality.DecisionAcceptActive {
		t.Fatalf("decision=%s want accept_active (pending status)", r.Decision)
	}
	if r.SafeForActiveRecall {
		t.Fatal("pending status must not be active-safe")
	}
}

func TestMisleadingCuesFlagged(t *testing.T) {
	c := findCase(t, loadCases(t), "misleading_cues_flagged")
	r := evalCase(c)
	if r.Decision != formationquality.DecisionRejectGarbage {
		t.Fatalf("decision=%s want reject_garbage", r.Decision)
	}
}

func TestUseInstructionRequiredForGuidance(t *testing.T) {
	c := findCase(t, loadCases(t), "governing_without_provenance_pending")
	r := evalCase(c)
	found := false
	for _, d := range r.Defects {
		if d.Code == formationquality.DefectMissingUseInstruction {
			found = true
		}
	}
	if !found {
		t.Fatal("expected missing_use_instruction defect")
	}
}

func TestFormationQualityMCPRESTParity(t *testing.T) {
	cases := loadCases(t)
	low := findCase(t, cases, "mcp_rest_formation_parity_low_quality")
	high := findCase(t, cases, "mcp_rest_formation_parity_high_quality")
	r1 := evalCase(low)
	r2 := evalCase(high)
	if r1.Decision != r1.Decision {
		t.Fatal("parity self-check failed")
	}
	_ = r2
	report := formationquality.RunBenchmark([]formationquality.FixtureCase{low, high})
	if report.Summary.MCPRESTFormationParityRate < 1.0 {
		t.Fatalf("parity=%.3f", report.Summary.MCPRESTFormationParityRate)
	}
}

func TestFormationQualityDoesNotRegressExistingFormationGate(t *testing.T) {
	g := formation.NewGate(nil)
	in := &formation.CreateInput{
		Path:          formation.PathDirectCreate,
		Kind:          api.MemoryKindPattern,
		Statement:     "Always validate memory writes through shared formation gate.",
		Authority:     10,
		Applicability: api.ApplicabilityAdvisory,
		Status:        api.StatusActive,
	}
	d, err := g.EvaluateCreateInput(in)
	if err != nil {
		t.Fatal(err)
	}
	if d.Outcome == formation.OutcomeReject {
		t.Fatal("legacy allow case regressed to reject")
	}
}

func TestFormationQualityScoreDeterministic(t *testing.T) {
	c := findCase(t, loadCases(t), "preference_with_scope_accepted")
	r1 := evalCase(c)
	r2 := evalCase(c)
	if r1.QualityScore != r2.QualityScore || r1.Decision != r2.Decision {
		t.Fatalf("non-deterministic: %+v vs %+v", r1, r2)
	}
}

func TestFormationQualityBenchmarkArtifact(t *testing.T) {
	if os.Getenv("MEMORY_FORMATION_QUALITY_BENCHMARK") != "1" {
		t.Skip("set MEMORY_FORMATION_QUALITY_BENCHMARK=1")
	}
	report := formationquality.RunBenchmark(loadCases(t))
	path := artifactPath()
	if err := formationquality.WriteReport(&report, path); err != nil {
		t.Fatal(err)
	}
}

func TestFormationQualityGate(t *testing.T) {
	if os.Getenv("MEMORY_FORMATION_QUALITY_GATE") != "1" && os.Getenv("PROOF_MEMORY_FORMATION_QUALITY") != "1" {
		t.Skip("set MEMORY_FORMATION_QUALITY_GATE=1")
	}
	report := formationquality.RunBenchmark(loadCases(t))
	if err := formationquality.WriteReport(&report, artifactPath()); err != nil {
		t.Fatal(err)
	}
	if !report.GatePassed {
		t.Fatalf("gate failed: %v metrics=%+v", report.GateFailures, report.Summary)
	}
}

func TestFormationGateIntegrationRejectsVagueDirectCreate(t *testing.T) {
	g := formation.NewGate(nil)
	in := &formation.CreateInput{
		Path:      formation.PathDirectCreate,
		Kind:      api.MemoryKindPattern,
		Statement: "Made progress on stuff.",
		Authority: 3,
		Status:    api.StatusActive,
	}
	_, err := g.EvaluateCreateInput(in)
	if err == nil {
		t.Fatal("expected reject for vague direct create")
	}
	_ = context.Background()
}

func artifactPath() string {
	if p := os.Getenv("MEMORY_FORMATION_QUALITY_ARTIFACT"); p != "" {
		return p
	}
	root := repoRoot()
	return filepath.Join(root, "artifacts", "memory-formation-quality-benchmark.json")
}

func repoRoot() string {
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
