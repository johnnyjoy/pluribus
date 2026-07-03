package benchmark_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"control-plane/internal/recall/benchmark"
)

// Phase 4 scale gate (hostile audit H1): recall quality must not collapse when the
// shared pool grows past 1k memories. Before the hybrid-candidate fix the candidate
// fetch was ORDER BY authority DESC LIMIT 100, so relevant low-authority memories
// silently vanished as the pool grew. This benchmark inflates the fixture corpus
// with 1000+ synthetic distractors and asserts recall@k and forbidden-hit rate stay
// within tolerance of the small-pool run.
func TestRecallBenchmark1kPool(t *testing.T) {
	lc, err := benchmark.Load(benchmark.DefaultDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := benchmark.DefaultRunConfig()
	cfg.RetrievalMode = benchmark.RetrievalModeHybrid

	baseResults, err := benchmark.RunAll(context.Background(), lc, cfg)
	if err != nil {
		t.Fatal(err)
	}
	baseSum := benchmark.Summarize(baseResults)

	const distractors = 1000
	big, err := benchmark.BuildLoaded(
		append(append([]benchmark.FixtureMemory{}, lc.Memories...), syntheticDistractors(distractors)...),
		lc.Cases,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(big.Objects) < distractors {
		t.Fatalf("pool did not inflate: %d objects", len(big.Objects))
	}

	bigResults, err := benchmark.RunAll(context.Background(), big, cfg)
	if err != nil {
		t.Fatal(err)
	}
	bigSum := benchmark.Summarize(bigResults)

	t.Logf("small pool (%d objs): recall@k=%.3f forbidden=%.3f", len(lc.Objects), baseSum.OverallRecallAtK, baseSum.ForbiddenHitRate)
	t.Logf("1k+ pool (%d objs): recall@k=%.3f forbidden=%.3f", len(big.Objects), bigSum.OverallRecallAtK, bigSum.ForbiddenHitRate)

	const recallTolerance = 0.15
	if bigSum.OverallRecallAtK+recallTolerance < baseSum.OverallRecallAtK {
		t.Fatalf("recall collapsed at scale: %.3f on 1k+ pool vs %.3f on small pool (tolerance %.2f)",
			bigSum.OverallRecallAtK, baseSum.OverallRecallAtK, recallTolerance)
	}
	const forbiddenTolerance = 0.10
	if bigSum.ForbiddenHitRate > baseSum.ForbiddenHitRate+forbiddenTolerance {
		t.Fatalf("forbidden-hit rate degraded at scale: %.3f vs %.3f", bigSum.ForbiddenHitRate, baseSum.ForbiddenHitRate)
	}
}

// syntheticDistractors builds n plausible but off-topic memories across fake
// domains, deterministic per seed so runs are reproducible.
func syntheticDistractors(n int) []benchmark.FixtureMemory {
	rng := rand.New(rand.NewSource(42))
	domains := []string{"billing", "mobile-app", "data-pipeline", "email-service", "search-index", "notifications", "onboarding", "warehouse", "gateway", "analytics"}
	kinds := []string{"pattern", "decision", "failure", "constraint", "state"}
	subjects := []string{"the retry queue", "invoice rendering", "the CSV importer", "session cookies", "the cron scheduler", "image thumbnails", "webhook delivery", "the rate limiter", "locale fallback", "the audit trail"}
	verbs := []string{"must be flushed before", "should never block", "times out during", "is duplicated by", "requires manual review after", "silently drops events when", "needs a feature flag for", "conflicts with", "was migrated away from", "is throttled by"}
	objectsOf := []string{"the nightly batch job", "regional failover", "schema version bumps", "the legacy admin panel", "third-party sandbox mode", "peak traffic windows", "the staging replica", "quarterly key rotation", "the mobile release train", "vendor API pagination"}

	out := make([]benchmark.FixtureMemory, 0, n)
	for i := 0; i < n; i++ {
		dom := domains[rng.Intn(len(domains))]
		stmt := fmt.Sprintf("%s %s %s (case %d)",
			subjects[rng.Intn(len(subjects))], verbs[rng.Intn(len(verbs))], objectsOf[rng.Intn(len(objectsOf))], i)
		out = append(out, benchmark.FixtureMemory{
			Label:     fmt.Sprintf("distractor-%04d", i),
			Domain:    "synthetic-" + dom,
			Kind:      kinds[rng.Intn(len(kinds))],
			Statement: stmt,
			Authority: 1 + rng.Intn(8),
			Tags:      []string{dom, "synthetic"},
		})
	}
	return out
}
