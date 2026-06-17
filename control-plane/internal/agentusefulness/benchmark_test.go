package agentusefulness

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func loadTestCorpus(t *testing.T) *LoadedCorpus {
	t.Helper()
	lc, err := Load(DefaultDir())
	if err != nil {
		t.Fatalf("load corpus: %v", err)
	}
	return lc
}

func TestAgentUsefulnessFixtureLoads(t *testing.T) {
	lc := loadTestCorpus(t)
	if len(lc.Tasks) < 12 {
		t.Fatalf("tasks=%d want >= 12", len(lc.Tasks))
	}
	if len(lc.Objects) == 0 {
		t.Fatal("expected memories")
	}
}

func TestNoMemoryBaselineFailsExpectedCases(t *testing.T) {
	lc := loadTestCorpus(t)
	for _, task := range lc.Tasks {
		if !task.RequiresMemoryHelp {
			continue
		}
		t.Run(task.ID, func(t *testing.T) {
			facts, trace := SimulateAgent(task, lc, RunModeNoMemory, nil)
			score, _ := ScoreRun(task, RunModeNoMemory, facts, nil, trace)
			if score.AnswerPass {
				t.Fatalf("no-memory should fail for help-required task; facts=%v", facts)
			}
		})
	}
}

func TestMemoryRunImprovesExpectedCases(t *testing.T) {
	lc := loadTestCorpus(t)
	report, err := RunSuite(lc)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.HelpEligibleTasks == 0 {
		t.Fatal("expected help-eligible tasks")
	}
	if report.Summary.MemoryHelpRate < 0.5 {
		t.Fatalf("memory_help_rate=%.3f want >= 0.5", report.Summary.MemoryHelpRate)
	}
}

func TestRecallManifestCaptured(t *testing.T) {
	lc := loadTestCorpus(t)
	task := lc.Tasks[0]
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if tr.MemoryREST.Manifest == nil || len(tr.MemoryREST.Manifest.Recalled) == 0 {
		t.Fatal("expected recall manifest entries")
	}
}

func TestMemoryUseTraceCaptured(t *testing.T) {
	lc := loadTestCorpus(t)
	task := lc.Tasks[0]
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.MemoryREST.UseTrace.UsedLabels) == 0 {
		t.Fatal("expected used labels in trace")
	}
	if tr.MemoryREST.UseTrace.UseReasons == nil {
		t.Fatal("expected use reasons map")
	}
}

func TestOutcomeFeedbackGenerated(t *testing.T) {
	lc := loadTestCorpus(t)
	task := lc.Tasks[0]
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.MemoryREST.OutcomeFeedback) == 0 {
		t.Fatal("expected outcome feedback events")
	}
}

func TestMemoryHelpRequiresRecallUseAndImprovement(t *testing.T) {
	lc := loadTestCorpus(t)
	task := findTask(t, lc, "constraint_memory_prevents_bad_action")
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if !tr.MemoryHelped {
		t.Fatalf("expected memory helped; failures=%v rest=%v", tr.MemoryREST.Failures, tr.MemoryREST.Score)
	}
	if tr.NoMemory.Score.AnswerPass {
		t.Fatal("no-memory should not pass")
	}
}

func TestRecalledButIgnoredIsNotHelpful(t *testing.T) {
	lc := loadTestCorpus(t)
	task := findTask(t, lc, "irrelevant_memory_ignored")
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if containsLabel(tr.MemoryREST.UseTrace.UsedLabels, "mem_irrelevant_similar_noise") {
		t.Fatal("irrelevant memory must not be used")
	}
}

func TestHistoricalMemoryNotUsedAsCurrentGuidance(t *testing.T) {
	lc := loadTestCorpus(t)
	task := findTask(t, lc, "historical_memory_not_current_guidance")
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if containsLabel(tr.MemoryREST.UseTrace.UsedLabels, "mem_archived_deploy_pattern") {
		t.Fatal("archived memory used as current guidance")
	}
}

func TestSupersededMemoryNotUsedAsCurrentGuidance(t *testing.T) {
	lc := loadTestCorpus(t)
	task := findTask(t, lc, "superseded_memory_not_used_as_current")
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if containsLabel(tr.MemoryREST.UseTrace.UsedLabels, "mem_superseded_deploy_rule") {
		t.Fatal("superseded memory used as current guidance")
	}
}

func TestRefutedMemoryNotUsed(t *testing.T) {
	lc := loadTestCorpus(t)
	task := findTask(t, lc, "refuted_memory_not_used")
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if containsLabel(tr.MemoryREST.UseTrace.UsedLabels, "mem_refuted_deploy_belief") {
		t.Fatal("refuted memory used")
	}
}

func TestWrongDomainMemoryIgnored(t *testing.T) {
	lc := loadTestCorpus(t)
	task := findTask(t, lc, "wrong_domain_memory_must_stay_quiet")
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	recalled := RecalledLabelSet(tr.MemoryREST.Manifest)
	if _, ok := recalled["mem_payments_pci_constraint"]; ok {
		t.Fatal("wrong-domain memory recalled")
	}
}

func TestMCPRESTUsefulnessParity(t *testing.T) {
	lc := loadTestCorpus(t)
	task := findTask(t, lc, "mcp_rest_parity_same_task")
	tr, err := RunTask(context.Background(), lc, task)
	if err != nil {
		t.Fatal(err)
	}
	if !tr.MCPRESTParity {
		t.Fatalf("mcp/rest parity failed: %s", tr.MCPRESTParityNote)
	}
}

func TestUsefulnessMetricsComputed(t *testing.T) {
	report, err := RunSuite(loadTestCorpus(t))
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.TotalTasks != 12 {
		t.Fatalf("total_tasks=%d", report.Summary.TotalTasks)
	}
}

func TestAgentMemoryUsefulnessGate(t *testing.T) {
	if os.Getenv("AGENT_MEMORY_USEFULNESS_GATE") != "1" && os.Getenv("PROOF_AGENT_MEMORY_EFFECTIVENESS") != "1" {
		t.Skip("set AGENT_MEMORY_USEFULNESS_GATE=1 or PROOF_AGENT_MEMORY_EFFECTIVENESS=1")
	}
	report, err := RunSuite(loadTestCorpus(t))
	if err != nil {
		t.Fatal(err)
	}
	artifactPath := artifactPath()
	if err := WriteReport(report, artifactPath); err != nil {
		t.Fatal(err)
	}
	if !report.GatePassed {
		t.Fatalf("gate failed: %v metrics=%+v", report.GateFailures, report.Summary)
	}
}

func TestAgentMemoryUsefulnessBenchmarkWritesArtifact(t *testing.T) {
	if os.Getenv("AGENT_MEMORY_USEFULNESS_BENCHMARK") != "1" {
		t.Skip("set AGENT_MEMORY_USEFULNESS_BENCHMARK=1")
	}
	report, err := RunSuite(loadTestCorpus(t))
	if err != nil {
		t.Fatal(err)
	}
	path := artifactPath()
	if err := WriteReport(report, path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("artifact missing: %v", err)
	}
}

func findTask(t *testing.T, lc *LoadedCorpus, id string) TaskFixture {
	t.Helper()
	for _, task := range lc.Tasks {
		if task.ID == id {
			return task
		}
	}
	t.Fatalf("task %q not found", id)
	return TaskFixture{}
}

func artifactPath() string {
	if p := os.Getenv("AGENT_MEMORY_USEFULNESS_ARTIFACT"); p != "" {
		return p
	}
	root := repoRoot()
	return filepath.Join(root, "artifacts", "agent-memory-usefulness-benchmark.json")
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
