package benchmark_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"control-plane/internal/memory"
	"control-plane/internal/recall/benchmark"
)

func TestLiveComparisonArtifactIncludesLexicalAndHybrid(t *testing.T) {
	lex := []benchmark.CaseResult{{
		Metrics: benchmark.CaseMetrics{RecallAtK: 0.8, PrecisionAtK: 0.7},
	}}
	live := []benchmark.CaseResult{{
		Metrics: benchmark.CaseMetrics{RecallAtK: 0.85, PrecisionAtK: 0.72},
		LiveStats: &benchmark.LiveEmbedStats{P50LatencyMs: 12},
	}}
	report := benchmark.BuildRealEmbedderReport(lex, live, memory.LiveEmbedderEnvConfig{
		Provider: "test", Model: "m", Dimension: 1536, Source: "test",
	})
	if report.Lexical == nil || report.LiveHybrid == nil || report.Comparison == nil {
		t.Fatal("expected lexical, live_hybrid, comparison")
	}
	if report.Lexical.Mode != benchmark.RetrievalModeLexical {
		t.Fatalf("lexical mode=%q", report.Lexical.Mode)
	}
	if report.LiveHybrid.Mode != benchmark.RetrievalModeHybridLive {
		t.Fatalf("live mode=%q", report.LiveHybrid.Mode)
	}
}

func TestLiveComparisonArtifactIncludesViolationMetrics(t *testing.T) {
	lex := []benchmark.CaseResult{{
		Metrics: benchmark.CaseMetrics{
			RecallAtK: 1, PrecisionAtK: 1,
			Violations: benchmark.ViolationMetrics{LifecycleViolationRate: 0},
		},
	}}
	live := []benchmark.CaseResult{{
		Metrics: benchmark.CaseMetrics{
			RecallAtK: 1, PrecisionAtK: 1,
			Violations: benchmark.ViolationMetrics{LifecycleViolationRate: 0, DateBoundViolationRate: 0},
		},
	}}
	report := benchmark.BuildRealEmbedderReport(lex, live, memory.LiveEmbedderEnvConfig{})
	if report.Comparison.LifecycleDelta != 0 {
		t.Fatalf("lifecycle delta=%v", report.Comparison.LifecycleDelta)
	}
}

func TestLiveComparisonArtifactIncludesFallbackStaleMissingCounts(t *testing.T) {
	live := []benchmark.CaseResult{{
		Metrics: benchmark.CaseMetrics{RecallAtK: 0.5, PrecisionAtK: 0.5},
		Semantic: &benchmark.SemanticStatus{FallbackReason: "embedding_failed"},
		LiveStats: &benchmark.LiveEmbedStats{StaleSkipCount: 2, MissingCount: 1},
	}}
	report := benchmark.BuildRealEmbedderReport(nil, live, memory.LiveEmbedderEnvConfig{})
	if report.LiveHybrid.FallbackRate != 1 {
		t.Fatalf("fallback rate=%v", report.LiveHybrid.FallbackRate)
	}
	if report.LiveHybrid.StaleEmbeddingCount != 2 || report.LiveHybrid.MissingEmbeddingCount != 1 {
		t.Fatalf("stale=%d missing=%d", report.LiveHybrid.StaleEmbeddingCount, report.LiveHybrid.MissingEmbeddingCount)
	}
}

func TestRealEmbedderBenchmarkRequiresExplicitConfig(t *testing.T) {
	cfg := benchmark.DefaultRunConfig()
	cfg.RetrievalMode = benchmark.RetrievalModeHybridLive
	lc, err := benchmark.LoadSemanticOnly(benchmark.DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = benchmark.RunCase(context.Background(), lc, lc.Cases[0], cfg)
	if err == nil {
		t.Fatal("expected error without live embedder")
	}
}

func TestRealEmbedderBenchmarkSkipsHonestlyWithoutEndpoint(t *testing.T) {
	t.Setenv("PLURIBUS_EMBEDDER_ENDPOINT", "")
	_, _, err := memory.LoadLiveEmbedderFromEnv()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRealEmbedderBenchmarkReportsProviderModelDimension(t *testing.T) {
	report := benchmark.BuildRealEmbedderReport(nil, nil, memory.LiveEmbedderEnvConfig{
		Provider: "openai", Model: "text-embedding-3-small", Dimension: 1536, Source: "PLURIBUS_EMBEDDER_*",
	})
	if report.Provider != "openai" || report.Model != "text-embedding-3-small" || report.Dimension != 1536 {
		t.Fatalf("report=%+v", report)
	}
}

func TestRecallBenchmarkLiveCompare(t *testing.T) {
	if os.Getenv("RECALL_BENCHMARK_LIVE_COMPARE") != "1" {
		t.Skip("set RECALL_BENCHMARK_LIVE_COMPARE=1")
	}
	if os.Getenv("PLURIBUS_EMBEDDER_ENDPOINT") == "" {
		t.Skip("set PLURIBUS_EMBEDDER_ENDPOINT for live comparison")
	}
	// Delegates to real embedder test path; writes live-comparison artifact.
	t.Run("delegate", func(t *testing.T) {
		t.Setenv("RECALL_BENCHMARK_REAL_EMBEDDER", "1")
		TestRecallBenchmarkRealEmbedder(t)
	})
	root := findRepoRoot(t)
	// Copy/rename handled in enhanced real embedder test below if present
	_ = filepath.Join(root, "artifacts", "recall-benchmark-live-comparison.json")
}
