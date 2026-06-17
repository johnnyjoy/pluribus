//go:build integration
// +build integration

package agenttelemetry

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	"control-plane/internal/migrate"
	"control-plane/internal/recall"
	sqlmigrations "control-plane/migrations"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func openTestPostgres(t *testing.T) *sql.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("TEST_PG_DSN"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("PLURIBUS_TEST_POSTGRES_DSN"))
	}
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrate.Apply(context.Background(), db, sqlmigrations.Files, nil); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestPostgresTelemetrySchemaAndRoundtrip(t *testing.T) {
	db := openTestPostgres(t)
	defer db.Close()
	repo := &Repo{DB: db}
	svc := NewServiceWithRepo(repo)
	ctx := context.Background()

	if repo.SchemaMigrationRate(ctx) != 1.0 {
		t.Fatal("postgres schema migration incomplete")
	}
	sid := uuid.New().String()
	if _, err := svc.StartSession(ctx, StartSessionRequest{Interface: "rest", SessionID: sid}); err != nil {
		t.Fatal(err)
	}
	bundle := &recall.RecallBundle{
		GoverningConstraints: []recall.MemoryItem{{ID: "test:pg-1", Statement: "pg", QualityState: "verified"}},
	}
	tel, err := svc.RecordAutoRecall(ctx, recall.AutoRecallInput{
		SessionID: sid, Interface: "rest",
		RecallRequest: map[string]any{"retrieval_query": "pg"},
		Bundle:        bundle,
	})
	if err != nil || tel.RecallEventID == "" {
		t.Fatalf("auto recall: %v %+v", err, tel)
	}
	rid, _ := uuid.Parse(tel.RecallEventID)
	ev, ok := repo.getRecall(ctx, rid)
	if !ok || len(ev.RecalledMemoryIDs) == 0 || ev.RecallBundleJSON == nil {
		t.Fatal("postgres recall roundtrip failed")
	}
	_, _ = svc.RecordDecision(ctx, RecordDecisionRequest{
		SessionID: sid, RecallEventID: tel.RecallEventID,
		Decisions: []struct {
			MemoryID             string   `json:"memory_id"`
			Decision             string   `json:"decision"`
			Reason               string   `json:"reason"`
			ContractFieldsCited  []string `json:"contract_fields_cited"`
			OutputFactsSupported []string `json:"output_facts_supported"`
		}{{MemoryID: "test:pg-1", Decision: "used", Reason: "r", ContractFieldsCited: []string{"statement"}}},
	})
	_, _ = svc.RecordOutput(ctx, RecordOutputRequest{
		SessionID: sid, RecallEventID: tel.RecallEventID, OutputFacts: []string{"done"},
	})
	resp, err := svc.Evaluate(ctx, EvaluateRequest{SessionID: sid, RecallEventID: tel.RecallEventID, TaskID: "pg"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Evaluation.ID == uuid.Nil {
		t.Fatal("evaluation not persisted")
	}
	if len(resp.Violations) == 0 && len(resp.UtilityCandidates) == 0 {
		// violations or candidates may be empty for obedient path; evaluation must exist
	}
	sum, err := svc.GetSessionSummary(ctx, sid)
	if err != nil || len(sum.RecallEvents) == 0 || len(sum.Evaluations) == 0 {
		t.Fatalf("session query failed: %v %+v", err, sum)
	}
}

func TestPostgresEvaluateTransactionRollback(t *testing.T) {
	db := openTestPostgres(t)
	defer db.Close()
	repo := &Repo{DB: db}
	ctx := context.Background()
	evalID := uuid.New()
	sid := uuid.New()
	rid := uuid.New()
	now := FixedNow()
	// Missing session FK should fail transaction
	err := repo.EvaluateTransactional(ctx, ObedienceEvaluationRow{
		ID: evalID, SessionID: sid, RecallEventID: rid, CreatedAt: now, EvaluatorVersion: EvaluatorVersion,
	}, []ViolationRow{{ID: uuid.New(), EvaluationID: evalID, ViolationCode: "x", Severity: "error", CreatedAt: now}}, nil)
	if err == nil {
		t.Fatal("expected FK rollback failure")
	}
	var n int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM agent_obedience_evaluations WHERE id = $1`, evalID).Scan(&n)
	if n != 0 {
		t.Fatal("partial evaluate persisted after rollback expected")
	}
}
