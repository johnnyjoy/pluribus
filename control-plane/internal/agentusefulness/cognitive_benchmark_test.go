package agentusefulness

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func loadCognitiveCorpus(t *testing.T) *LoadedCorpus {
	t.Helper()
	lc, err := LoadCognitive(DefaultDir())
	if err != nil {
		t.Fatalf("load cognitive corpus: %v", err)
	}
	return lc
}

func TestCognitiveMemoryResearchSourcesDocumented(t *testing.T) {
	sources := DocumentedResearchSources()
	if len(sources) < 5 {
		t.Fatalf("sources=%d want >= 5", len(sources))
	}
	for _, s := range sources {
		if s.ID == "" || s.Title == "" || len(s.Principles) == 0 {
			t.Fatalf("incomplete source: %+v", s)
		}
	}
}

func TestResearchPrinciplesMappedToCases(t *testing.T) {
	lc := loadCognitiveCorpus(t)
	covered := CoveredPrinciples(lc.CognitiveTasks())
	for _, p := range RequiredResearchPrinciples() {
		if covered[p] == 0 {
			t.Fatalf("missing fixture coverage for principle %q", p)
		}
	}
}

func TestEncodingSpecificityCueMatch(t *testing.T) {
	lc := loadCognitiveCorpus(t)
	task := findTask(t, lc, "encoding_specificity_matching_cues_help")
	mem := lc.FixtureByLabel("cmem_encoded_strong_deploy")
	if mem == nil {
		t.Fatal("missing memory fixture")
	}
	match := EvaluateCueMatch(task, *mem)
	if match.MatchScore < MinCueMatchThreshold {
		t.Fatalf("match_score=%.3f want >= %.3f", match.MatchScore, MinCueMatchThreshold)
	}
}

func TestUnderEncodedMemoryDoesNotCountHelpful(t *testing.T) {
	lc := loadCognitiveCorpus(t)
	task := findTask(t, lc, "encoding_specificity_missing_cues_fail_to_help")
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if tr.MemoryHelped {
		t.Fatal("under-encoded memory must not count as helpful")
	}
}

func TestMisleadingCueRiskDetected(t *testing.T) {
	lc := loadCognitiveCorpus(t)
	task := findTask(t, lc, "encoding_specificity_misleading_cues_harm_risk")
	mem := lc.FixtureByLabel("cmem_misleading_deploy")
	if mem == nil {
		t.Fatal("missing misleading memory")
	}
	match := EvaluateCueMatch(task, *mem)
	if !match.MisleadingCue && !match.NegativeScopeHit {
		t.Fatalf("expected misleading or negative scope risk: %+v", match)
	}
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if containsLabel(tr.MemoryREST.UseTrace.UsedLabels, "cmem_misleading_deploy") {
		t.Fatal("misleading memory must not be used")
	}
}

func TestSchemaConstraintApplication(t *testing.T) {
	runCognitiveCase(t, "schema_constraint_changes_action")
}

func TestSchemaFailurePreventsRepeat(t *testing.T) {
	runCognitiveCase(t, "schema_failure_prevents_repeat")
}

func TestSchemaProcedureOrdersSteps(t *testing.T) {
	lc := loadCognitiveCorpus(t)
	task := findTask(t, lc, "schema_procedure_orders_steps")
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if !tr.MemoryREST.Score.AnswerPass {
		t.Fatalf("procedure task failed: %v", tr.MemoryREST.Failures)
	}
	for _, step := range []string{"STEP_TRIAGE", "STEP_ISOLATE", "STEP_NOTIFY", "STEP_POSTMORTEM"} {
		if !containsFact(tr.MemoryREST.AnswerFacts, step) {
			t.Fatalf("missing procedure step %s in %v", step, tr.MemoryREST.AnswerFacts)
		}
	}
}

func TestWrongProjectMemoryIgnored(t *testing.T) {
	lc := loadCognitiveCorpus(t)
	task := findTask(t, lc, "context_wrong_project_ignored")
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if containsLabel(tr.MemoryREST.UseTrace.UsedLabels, "cmem_project_beta_config") {
		t.Fatal("wrong project memory used")
	}
}

func TestWrongSystemMemoryIgnored(t *testing.T) {
	lc := loadCognitiveCorpus(t)
	task := findTask(t, lc, "context_wrong_system_ignored")
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if containsLabel(tr.MemoryREST.UseTrace.UsedLabels, "cmem_system_vm_deploy") {
		t.Fatal("wrong system memory used")
	}
}

func TestSimilarButWrongMemorySuppressed(t *testing.T) {
	lc := loadCognitiveCorpus(t)
	task := findTask(t, lc, "interference_similar_but_wrong_ignored")
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if containsLabel(tr.MemoryREST.UseTrace.UsedLabels, "cmem_similar_wrong_deploy") {
		t.Fatal("similar wrong memory used")
	}
}

func TestHighAuthorityWrongScopeIgnored(t *testing.T) {
	lc := loadCognitiveCorpus(t)
	task := findTask(t, lc, "interference_high_authority_wrong_scope_ignored")
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if containsLabel(tr.MemoryREST.UseTrace.UsedLabels, "cmem_high_auth_wrong_scope") {
		t.Fatal("high authority wrong scope memory used")
	}
}

func TestHighUtilityWrongScopeIgnored(t *testing.T) {
	lc := loadCognitiveCorpus(t)
	task := findTask(t, lc, "interference_high_utility_wrong_scope_ignored")
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if containsLabel(tr.MemoryREST.UseTrace.UsedLabels, "cmem_high_utility_wrong_scope") {
		t.Fatal("high utility wrong scope memory used")
	}
}

func TestRecalledIgnoredNotHelpful(t *testing.T) {
	lc := loadCognitiveCorpus(t)
	task := findTask(t, lc, "retrieval_practice_recalled_ignored_not_helpful")
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if !tr.MemoryHelped {
		t.Fatal("expected helpful memory to help overall")
	}
	if containsLabel(tr.MemoryREST.UseTrace.UsedLabels, "cmem_recalled_noise_deploy") {
		t.Fatal("ignored recalled memory must not be used")
	}
}

func TestUsedNoImprovementNotHelpful(t *testing.T) {
	lc := loadCognitiveCorpus(t)
	task := findTask(t, lc, "retrieval_practice_used_no_improvement_not_helpful")
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if tr.MemoryHelped {
		t.Fatal("used-but-no-improvement must not count as helpful")
	}
}

func TestBadExperienceReplaySuppressed(t *testing.T) {
	lc := loadCognitiveCorpus(t)
	task := findTask(t, lc, "experience_following_bad_prior_experience_suppressed")
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if containsLabel(tr.MemoryREST.UseTrace.UsedLabels, "cmem_bad_experience_replay") {
		t.Fatal("bad prior experience replayed as guidance")
	}
}

func TestRefutedHistoricalNotGuidance(t *testing.T) {
	lc := loadCognitiveCorpus(t)
	task := findTask(t, lc, "lifecycle_refuted_historical_not_guidance")
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if containsLabel(tr.MemoryREST.UseTrace.UsedLabels, "cmem_refuted_historical_guidance") {
		t.Fatal("refuted historical memory used as guidance")
	}
}

func TestResearchPrincipleCoverageMetric(t *testing.T) {
	lc := loadCognitiveCorpus(t)
	report, err := RunCognitiveSuite(lc)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.ResearchPrincipleCoverageRate < 0.8 {
		t.Fatalf("coverage=%.3f want >= 0.8", report.Summary.ResearchPrincipleCoverageRate)
	}
}

func TestInterferenceMetricsComputed(t *testing.T) {
	lc := loadCognitiveCorpus(t)
	report, err := RunCognitiveSuite(lc)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.InterferenceFailureRate > 0 {
		t.Fatalf("interference_failure_rate=%.3f want 0", report.Summary.InterferenceFailureRate)
	}
}

func TestCognitiveMemoryBenchmarkArtifact(t *testing.T) {
	if os.Getenv("COGNITIVE_MEMORY_USEFULNESS_BENCHMARK") != "1" {
		t.Skip("set COGNITIVE_MEMORY_USEFULNESS_BENCHMARK=1")
	}
	lc := loadCognitiveCorpus(t)
	report, err := RunCognitiveSuite(lc)
	if err != nil {
		t.Fatal(err)
	}
	path := cognitiveArtifactPath()
	if err := WriteCognitiveReport(report, path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("artifact missing: %v", err)
	}
}

func TestCognitiveMemoryMCPRESTParity(t *testing.T) {
	lc := loadCognitiveCorpus(t)
	task := findTask(t, lc, "reconstructive_metadata_exposed")
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if !tr.MCPRESTParity {
		t.Fatalf("mcp/rest parity failed: %s", tr.MCPRESTParityNote)
	}
}

func TestCognitiveMemoryUsefulnessGate(t *testing.T) {
	if os.Getenv("COGNITIVE_MEMORY_USEFULNESS_GATE") != "1" && os.Getenv("PROOF_COGNITIVE_MEMORY_BENEFIT") != "1" {
		t.Skip("set COGNITIVE_MEMORY_USEFULNESS_GATE=1 or PROOF_COGNITIVE_MEMORY_BENEFIT=1")
	}
	lc := loadCognitiveCorpus(t)
	report, err := RunCognitiveSuite(lc)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteCognitiveReport(report, cognitiveArtifactPath()); err != nil {
		t.Fatal(err)
	}
	if !report.GatePassed {
		t.Fatalf("gate failed: %v summary=%+v", report.GateFailures, report.Summary)
	}
}

func runCognitiveCase(t *testing.T, id string) {
	t.Helper()
	lc := loadCognitiveCorpus(t)
	task := findTask(t, lc, id)
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if !tr.MemoryREST.Score.AnswerPass {
		t.Fatalf("task %s failed: %v", id, tr.MemoryREST.Failures)
	}
}

func containsFact(facts []string, want string) bool {
	for _, f := range facts {
		if f == want {
			return true
		}
	}
	return false
}

func cognitiveArtifactPath() string {
	if p := os.Getenv("COGNITIVE_MEMORY_USEFULNESS_ARTIFACT"); p != "" {
		return p
	}
	root := repoRoot()
	return filepath.Join(root, "artifacts", "cognitive-memory-usefulness-benchmark.json")
}
