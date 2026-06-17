package benchmark_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"control-plane/internal/memory"
	"control-plane/internal/recall/benchmark"
)

func fakeLiveEmbedderServer(t *testing.T, dim int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vec := make([]float64, dim)
		for i := range vec {
			vec[i] = float64(i+1) / float64(dim)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"embedding": vec}},
		})
	}))
}

func TestBenchmarkLiveEmbedderRequiresExplicitConfig(t *testing.T) {
	cfg := benchmark.DefaultRunConfig()
	cfg.RetrievalMode = benchmark.RetrievalModeHybridLive
	cfg.LiveEmbedder = nil
	lc, err := benchmark.LoadSemanticOnly(benchmark.DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = benchmark.RunCase(context.Background(), lc, lc.Cases[0], cfg)
	if err == nil {
		t.Fatal("expected error when live embedder not configured")
	}
}

func TestBenchmarkLiveEmbedderReportsModel(t *testing.T) {
	dim := 8
	srv := fakeLiveEmbedderServer(t, dim)
	defer srv.Close()
	emb := memory.NewHTTPEmbedderForTest(srv.URL+"/v1", "phase10c-test-model", dim, "http")
	lc, err := benchmark.LoadSemanticOnly(benchmark.DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := benchmark.DefaultRunConfig()
	cfg.RetrievalMode = benchmark.RetrievalModeHybridLive
	cfg.LiveEmbedder = emb
	cfg.LiveProfile = memory.EmbeddingConfigProfile{Model: "phase10c-test-model", Provider: "http", Dimension: dim}
	res, err := benchmark.RunCase(context.Background(), lc, lc.Cases[0], cfg)
	if err != nil {
		t.Fatal(err)
	}
	if res.LiveStats == nil {
		t.Fatal("expected live stats")
	}
}

func TestBenchmarkLiveEmbedderReportsFallbackRate(t *testing.T) {
	dim := 8
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	emb := memory.NewHTTPEmbedderForTest(srv.URL+"/v1", "fail-model", dim, "http")
	lc, err := benchmark.LoadSemanticOnly(benchmark.DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := benchmark.DefaultRunConfig()
	cfg.RetrievalMode = benchmark.RetrievalModeHybridLive
	cfg.LiveEmbedder = emb
	cfg.LiveProfile = memory.EmbeddingConfigProfile{Model: "fail-model", Dimension: dim}
	_, err = benchmark.NewLiveHybridMemoryStub(lc, emb, cfg.LiveProfile)
	if err == nil {
		t.Fatal("expected init failure when embedder returns 500")
	}
}

func TestBenchmarkLiveEmbedderReportsStaleEmbeddingCount(t *testing.T) {
	dim := 8
	srv := fakeLiveEmbedderServer(t, dim)
	defer srv.Close()
	emb := memory.NewHTTPEmbedderForTest(srv.URL+"/v1", "m", dim, "http")
	lc, err := benchmark.LoadSemanticOnly(benchmark.DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	stub, err := benchmark.NewLiveHybridMemoryStub(lc, emb, memory.EmbeddingConfigProfile{Model: "m", Dimension: dim})
	if err != nil {
		t.Fatal(err)
	}
	var staleLbl string
	for _, o := range stub.All {
		lbl := stub.IDToLabel[o.ID]
		if lbl == "" {
			continue
		}
		st := string(o.Status)
		if st == "" {
			st = "active"
		}
		if st == "active" {
			staleLbl = lbl
			break
		}
	}
	if staleLbl == "" {
		t.Fatal("no fixture labels embedded")
	}
	stub.MarkLabelStale(staleLbl, memory.StalenessSourceHashMismatch)
	before := stub.Stats().StaleSkipCount
	query := make([]float32, dim)
	for i := range query {
		query[i] = 1
	}
	_, _, err = stub.SearchSimilarCandidates(context.Background(), query, memory.SearchRequest{Statuses: []string{"active"}}, 10, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	if stub.Stats().StaleSkipCount <= before {
		t.Fatalf("expected stale skip count increase, before=%d after=%d", before, stub.Stats().StaleSkipCount)
	}
}

func TestRecallBenchmarkRealEmbedder(t *testing.T) {
	if os.Getenv("RECALL_BENCHMARK_REAL_EMBEDDER") != "1" {
		t.Skip("set RECALL_BENCHMARK_REAL_EMBEDDER=1 for live embedder benchmark")
	}
	emb, env, err := memory.LoadLiveEmbedderFromEnv()
	if err != nil {
		t.Skip(err.Error())
	}
	lc, err := benchmark.LoadSemanticOnly(benchmark.DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	lexCfg := benchmark.DefaultRunConfig()
	lexCfg.RetrievalMode = benchmark.RetrievalModeLexical
	liveCfg := benchmark.DefaultRunConfig()
	liveCfg.RetrievalMode = benchmark.RetrievalModeHybridLive
	liveCfg.LiveEmbedder = emb
	liveCfg.LiveProfile = memory.EmbeddingConfigProfile{
		Model: env.Model, Provider: env.Provider, Dimension: env.Dimension,
	}
	liveCfg.LiveEnv = env
	lexical, err := benchmark.RunAll(context.Background(), lc, lexCfg)
	if err != nil {
		t.Fatal(err)
	}
	live, err := benchmark.RunAll(context.Background(), lc, liveCfg)
	if err != nil {
		t.Fatal(err)
	}
	report := benchmark.BuildRealEmbedderReport(lexical, live, env)
	root := findRepoRoot(t)
	realPath := filepath.Join(root, "artifacts", "recall-benchmark-real-embedder.json")
	comparePath := benchmark.DefaultLiveComparisonPath(root)
	if err := benchmark.WriteRealEmbedderJSON(report, realPath); err != nil {
		t.Fatal(err)
	}
	if err := benchmark.WriteLiveComparisonJSON(report, comparePath); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s and %s provider=%s model=%s dim=%d passes=%v",
		realPath, comparePath, env.Provider, env.Model, env.Dimension, report.Comparison.LivePassesThreshold)
}

func TestRecallBenchmarkRealEmbedderSkipsWithoutEnv(t *testing.T) {
	t.Setenv("PLURIBUS_EMBEDDER_ENDPOINT", "")
	_, _, err := memory.LoadLiveEmbedderFromEnv()
	if err == nil {
		t.Fatal("expected error without endpoint")
	}
}
