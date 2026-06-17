package benchmark_test

import (
	"context"
	"testing"

	"control-plane/internal/recall"
	"control-plane/internal/recall/benchmark"

	"github.com/google/uuid"
)

func TestOnguardMobileCredentialInPool(t *testing.T) {
	lc, err := benchmark.Load(benchmark.DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	var c benchmark.FixtureCase
	for _, fc := range lc.Cases {
		if fc.ID == "onguard_badge_integration" {
			c = fc
			break
		}
	}
	w := recall.DefaultRankingWeights()
	compiler := &recall.Compiler{Ranking: &w, Memory: &benchmark.MemoryStub{All: lc.Objects}}
	req := recall.CompileRequest{
		RetrievalQuery: c.Query,
		Tags:           c.Tags,
		MaxPerKind:     10,
		MaxTotal:       40,
		Mode:           "continuity",
	}
	bundle, err := compiler.Compile(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	flat := recall.FlattenBundleByScore(bundle)
	found := false
	for _, it := range flat {
		id, err := uuid.Parse(it.ID)
		if err != nil {
			t.Fatal(err)
		}
		if lc.IDToLabel[id] == "onguard_mobile_credential_decision" {
			found = true
			break
		}
	}
	if !found {
		labels := make([]string, 0, len(flat))
		for _, it := range flat {
			id, _ := uuid.Parse(it.ID)
			labels = append(labels, lc.IDToLabel[id])
		}
		t.Fatalf("mobile credential missing from flat bundle: %v", labels)
	}
}
