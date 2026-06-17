package benchmark_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"control-plane/internal/recall/benchmark"

	"github.com/google/uuid"
)

func TestBenchmarkFixtureLoad(t *testing.T) {
	lc, err := benchmark.Load(benchmark.DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(lc.Memories) < 40 {
		t.Fatalf("expected >=40 memories, got %d", len(lc.Memories))
	}
	if len(lc.Cases) < 22 {
		t.Fatalf("expected >=22 cases, got %d", len(lc.Cases))
	}
}

func TestBenchmarkMetrics_math(t *testing.T) {
	c := benchmark.FixtureCase{
		K:                   5,
		ExpectedLabels:      []string{"a", "b"},
		AcceptableLabels:    []string{"c"},
		ForbiddenLabels:     []string{"x"},
		MinimumRecallAtK:    0.5,
		MinimumPrecisionAtK: 0.4,
		MaximumForbiddenHits: 0,
	}
	hits := []benchmark.RankedHit{
		{Rank: 1, Label: "a", Expected: true},
		{Rank: 2, Label: "x", Forbidden: true},
		{Rank: 3, Label: "c", Acceptable: true},
	}
	m := benchmark.ComputeMetrics(c, hits)
	if m.RecallAtK != 0.5 {
		t.Fatalf("recall=%v want 0.5", m.RecallAtK)
	}
	if m.ForbiddenHitCount != 1 {
		t.Fatalf("forbidden=%d want 1", m.ForbiddenHitCount)
	}
	ok, reason := benchmark.GateCase(c, m, len(hits))
	if ok || reason == "" {
		t.Fatalf("expected gate fail, ok=%v reason=%q", ok, reason)
	}
}

func TestRecallBenchmarkGate(t *testing.T) {
	if os.Getenv("RECALL_BENCHMARK_GATE") != "1" {
		t.Skip("set RECALL_BENCHMARK_GATE=1 to enforce per-case thresholds (expected to fail until recall scoring improves)")
	}
	lc, err := benchmark.Load(benchmark.DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := benchmark.DefaultRunConfig()
	cfg.Gate = true
	results, err := benchmark.RunAll(context.Background(), lc, cfg)
	if err != nil {
		t.Fatal(err)
	}
	var failures []benchmark.CaseResult
	for _, r := range results {
		if !r.Passed {
			failures = append(failures, r)
			t.Logf("\n%s", benchmark.FormatCaseFailure(r))
		}
	}
	if len(failures) > 0 {
		t.Fatalf("%d/%d benchmark cases failed gate thresholds", len(failures), len(results))
	}
}

func TestRecallBenchmarkBaseline(t *testing.T) {
	if os.Getenv("RECALL_BENCHMARK_BASELINE") != "1" {
		t.Skip("set RECALL_BENCHMARK_BASELINE=1 to write baseline")
	}
	lc, err := benchmark.Load(benchmark.DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := benchmark.DefaultRunConfig()
	cfg.Gate = false
	results, err := benchmark.RunAll(context.Background(), lc, cfg)
	if err != nil {
		t.Fatal(err)
	}
	sum := benchmark.Summarize(results)
	root := findRepoRoot(t)
	jsonPath := filepath.Join(root, "artifacts", "recall-benchmark-baseline.json")
	if err := benchmark.WriteBaselineJSON(jsonPath, sum); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(root, "docs", "reports", "phase3-recall-baseline-report.md")
	judgment := "Baseline capture — gate thresholds not evaluated."
	if err := benchmark.WriteMarkdownReport(reportPath, sum, judgment); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote baseline %s and %s", jsonPath, reportPath)
}

func TestRecallBenchmarkRegression(t *testing.T) {
	lc, err := benchmark.Load(benchmark.DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := benchmark.DefaultRunConfig()
	cfg.Gate = false
	current, err := benchmark.RunAll(context.Background(), lc, cfg)
	if err != nil {
		t.Fatal(err)
	}
	curSum := benchmark.Summarize(current)
	root := findRepoRoot(t)
	basePath := filepath.Join(root, "artifacts", "recall-benchmark-baseline.json")
	base, err := benchmark.LoadBaselineJSON(basePath)
	if err != nil {
		t.Skipf("no baseline at %s — run RECALL_BENCHMARK_BASELINE=1 first", basePath)
	}
	const eps = 0.05
	if curSum.OverallRecallAtK+eps < base.OverallRecallAtK {
		t.Fatalf("recall regression: current=%.2f baseline=%.2f", curSum.OverallRecallAtK, base.OverallRecallAtK)
	}
	if curSum.ForbiddenHitRate > base.ForbiddenHitRate+eps {
		t.Fatalf("forbidden-hit regression: current=%.2f baseline=%.2f", curSum.ForbiddenHitRate, base.ForbiddenHitRate)
	}
}

func TestDomainConfusionMatrix(t *testing.T) {
	lc, err := benchmark.Load(benchmark.DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	results, err := benchmark.RunAll(context.Background(), lc, benchmark.DefaultRunConfig())
	if err != nil {
		t.Fatal(err)
	}
	m := benchmark.DomainConfusionMatrix(results)
	if len(m) == 0 {
		t.Fatal("empty confusion matrix")
	}
}

func TestRecallBenchmarkHybridGate(t *testing.T) {
	if os.Getenv("RECALL_BENCHMARK_HYBRID") != "1" {
		t.Skip("set RECALL_BENCHMARK_HYBRID=1 to enforce hybrid semantic gate")
	}
	lc, err := benchmark.LoadSemanticOnly(benchmark.DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := benchmark.DefaultRunConfig()
	cfg.RetrievalMode = benchmark.RetrievalModeHybrid
	cfg.Gate = true
	cfg.HybridGate = true
	results, err := benchmark.RunAll(context.Background(), lc, cfg)
	if err != nil {
		t.Fatal(err)
	}
	var failures []benchmark.CaseResult
	for _, r := range results {
		if !r.Passed {
			failures = append(failures, r)
			t.Logf("\n%s", benchmark.FormatCaseFailure(r))
		}
	}
	if len(failures) > 0 {
		t.Fatalf("%d/%d hybrid benchmark cases failed", len(failures), len(results))
	}
}

func TestRecallBenchmarkCompare(t *testing.T) {
	if os.Getenv("RECALL_BENCHMARK_COMPARE") != "1" {
		t.Skip("set RECALL_BENCHMARK_COMPARE=1 to write comparison artifact")
	}
	lc, err := benchmark.LoadSemanticOnly(benchmark.DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	baseCfg := benchmark.DefaultRunConfig()
	baseCfg.Gate = false
	lexCfg := baseCfg
	lexCfg.RetrievalMode = benchmark.RetrievalModeLexical
	hybCfg := baseCfg
	hybCfg.RetrievalMode = benchmark.RetrievalModeHybrid
	lexical, err := benchmark.RunAll(context.Background(), lc, lexCfg)
	if err != nil {
		t.Fatal(err)
	}
	hybrid, err := benchmark.RunAll(context.Background(), lc, hybCfg)
	if err != nil {
		t.Fatal(err)
	}
	rep := benchmark.BuildComparisonReport(lexical, hybrid)
	root := findRepoRoot(t)
	outPath := filepath.Join(root, "artifacts", "recall-benchmark-comparison.json")
	if err := benchmark.WriteComparisonJSON(outPath, rep); err != nil {
		t.Fatal(err)
	}
	if err := benchmark.GateHybridComparison(rep); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote comparison %s", outPath)
}

func TestRecallBenchmarkRunsLexicalMode(t *testing.T) {
	lc, err := benchmark.LoadSemanticOnly(benchmark.DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := benchmark.DefaultRunConfig()
	cfg.RetrievalMode = benchmark.RetrievalModeLexical
	results, err := benchmark.RunAll(context.Background(), lc, cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.Semantic != nil && r.Semantic.Path == "semantic_hybrid" {
			t.Fatalf("case %s used semantic_hybrid in lexical mode", r.Case.ID)
		}
	}
}

func TestRecallBenchmarkRunsHybridMode(t *testing.T) {
	lc, err := benchmark.LoadSemanticOnly(benchmark.DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := benchmark.DefaultRunConfig()
	cfg.RetrievalMode = benchmark.RetrievalModeHybrid
	results, err := benchmark.RunAll(context.Background(), lc, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected hybrid cases")
	}
	hybridPath := 0
	for _, r := range results {
		if r.Semantic != nil && r.Semantic.Path == "semantic_hybrid" {
			hybridPath++
		}
	}
	if hybridPath == 0 {
		t.Fatal("expected at least one semantic_hybrid path in hybrid mode")
	}
}

func TestRecallBenchmarkReportsLifecycleViolations(t *testing.T) {
	c := benchmark.FixtureCase{K: 3, RecallMode: "current"}
	hits := []benchmark.RankedHit{{Rank: 1, Label: "archived_one"}}
	id := uuid.New()
	lc := &benchmark.LoadedCorpus{
		LabelToID:   map[string]uuid.UUID{"archived_one": id},
		IDToFixture: map[uuid.UUID]benchmark.FixtureMemory{id: {Label: "archived_one", Status: "archived"}},
	}
	v := benchmark.ComputeViolations(c, lc, hits)
	if v.LifecycleViolationCount == 0 {
		t.Fatal("expected lifecycle violation for archived in current mode")
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "artifacts")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, "control-plane", "testdata", "recall_benchmark")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("repo root not found")
	return ""
}
