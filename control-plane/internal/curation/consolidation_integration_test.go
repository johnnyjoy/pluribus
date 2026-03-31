//go:build integration

package curation

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	_ "github.com/lib/pq"
)

// TestIntegration_canonicalConsolidation_exactKeyStrengthens exercises materialize when an identical
// statement_key already exists: one canonical row, second promotion reinforces it (no new id).
func TestIntegration_canonicalConsolidation_exactKeyStrengthens(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql open: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}

	ctx := context.Background()
	curationRepo := &Repo{DB: db}
	memoryRepo := &memory.Repo{DB: db}
	memorySvc := &memory.Service{Repo: memoryRepo}

	stmt := "Canonical consolidation must strengthen existing memory when statement key matches exactly for integration testing."
	p1 := ProposalPayloadV1{
		V:                 1,
		Kind:              api.MemoryKindPattern,
		Statement:         stmt,
		Reason:            "first",
		Tags:              []string{"entity:testsvc", "domain:integration"},
		ProposedAuthority: 5,
	}
	pj1, _ := json.Marshal(p1)
	cand1, err := curationRepo.CreateDigest(ctx, stmt, 0.8, pj1)
	if err != nil {
		t.Fatalf("candidate1: %v", err)
	}

	ccfg := memory.NormalizeCanonicalConsolidation(&memory.CanonicalConsolidationConfig{
		Enabled:                 true,
		NearDuplicateJaccardMin: 0.88,
		NegationGuard:           true,
	})
	svc := &Service{
		Repo:         curationRepo,
		Memory:       memorySvc,
		MemoryDup:    memoryRepo,
		Promotion:    &PromotionDigestConfig{CanonicalConsolidation: ccfg},
		DigestLimits: defaultDigestLimits(),
	}

	out1, err := svc.Materialize(ctx, cand1.ID)
	if err != nil {
		t.Fatalf("materialize1: %v", err)
	}
	if out1 == nil || out1.Memory == nil {
		t.Fatal("expected memory from first materialize")
	}
	if !out1.Created {
		t.Fatalf("first materialize should create new row: %+v", out1)
	}
	memID := out1.Memory.ID

	p2 := p1
	p2.Reason = "second reinforcement"
	p2.Tags = []string{"entity:testsvc", "domain:integration", "source:repeat"}
	pj2, _ := json.Marshal(p2)
	cand2, err := curationRepo.CreateDigest(ctx, stmt, 0.81, pj2)
	if err != nil {
		t.Fatalf("candidate2: %v", err)
	}

	out2, err := svc.Materialize(ctx, cand2.ID)
	if err != nil {
		t.Fatalf("materialize2: %v", err)
	}
	if out2.Created {
		t.Fatalf("second materialize should consolidate, not insert: %+v", out2)
	}
	if !out2.Strengthened || out2.ConsolidatedIntoMemoryID == nil || *out2.ConsolidatedIntoMemoryID != memID.String() {
		t.Fatalf("expected strengthen into same id: %+v", out2)
	}
	if out2.Memory == nil || out2.Memory.ID != memID {
		t.Fatalf("expected returned memory id %s, got %+v", memID, out2.Memory)
	}
	final, err := memoryRepo.GetByID(ctx, memID)
	if err != nil || final == nil {
		t.Fatalf("reload memory: %v", err)
	}
	var pay map[string]any
	if err := json.Unmarshal(final.Payload, &pay); err != nil {
		t.Fatalf("payload json: %v", err)
	}
	cons, ok := pay["pluribus_consolidation"].(map[string]any)
	if !ok {
		t.Fatalf("expected pluribus_consolidation in payload, got %v", pay)
	}
	if sc, ok := cons["support_count"].(float64); !ok || sc < 1 {
		t.Fatalf("expected support_count >= 1, got %v", cons["support_count"])
	}
}
