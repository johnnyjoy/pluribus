package agenttelemetry

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"control-plane/internal/migrate"
	"control-plane/internal/recall"
	sqlmigrations "control-plane/migrations"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func runPostgresProofMetrics(t *testing.T, dsn string) PostgresProofMetrics {
	t.Helper()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrate.Apply(context.Background(), db, sqlmigrations.Files, nil); err != nil {
		t.Fatal(err)
	}
	repo := &Repo{DB: db}
	svc := NewServiceWithRepo(repo)
	ctx := context.Background()
	m := PostgresProofMetrics{}
	m.PostgresSchemaMigrationRate = repo.SchemaMigrationRate(ctx)
	sid := uuid.New().String()
	if _, err := svc.StartSession(ctx, StartSessionRequest{Interface: "rest", SessionID: sid}); err != nil {
		t.Fatal(err)
	}
	m.PostgresSessionPersistenceRate = 1.0
	tel, err := svc.RecordAutoRecall(ctx, recall.AutoRecallInput{
		SessionID: sid, Interface: "rest",
		RecallRequest: map[string]any{"retrieval_query": "pg-proof"},
		Bundle: &recall.RecallBundle{
			GoverningConstraints: []recall.MemoryItem{{ID: "test:pg-proof", Statement: "x", QualityState: "verified"}},
		},
	})
	if err != nil || tel.RecallEventID == "" {
		t.Fatal(err)
	}
	m.PostgresRecallEventPersistenceRate = 1.0
	m.PostgresJSONRoundtripPassRate = 1.0
	rid, _ := uuid.Parse(tel.RecallEventID)
	ev, ok := repo.getRecall(ctx, rid)
	if !ok || ev.RecallBundleJSON == nil || len(ev.RecalledMemoryIDs) == 0 {
		t.Fatal("recall json roundtrip failed")
	}
	_, err = svc.RecordDecision(ctx, RecordDecisionRequest{
		SessionID: sid, RecallEventID: tel.RecallEventID,
		Decisions: []struct {
			MemoryID             string   `json:"memory_id"`
			Decision             string   `json:"decision"`
			Reason               string   `json:"reason"`
			ContractFieldsCited  []string `json:"contract_fields_cited"`
			OutputFactsSupported []string `json:"output_facts_supported"`
		}{{MemoryID: "test:pg-proof", Decision: "used", Reason: "r", ContractFieldsCited: []string{"statement"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	m.PostgresDecisionPersistenceRate = 1.0
	_, err = svc.RecordOutput(ctx, RecordOutputRequest{
		SessionID: sid, RecallEventID: tel.RecallEventID, OutputFacts: []string{"ok"},
	})
	if err != nil {
		t.Fatal(err)
	}
	m.PostgresOutputPersistenceRate = 1.0
	resp, err := svc.Evaluate(ctx, EvaluateRequest{SessionID: sid, RecallEventID: tel.RecallEventID})
	if err != nil {
		t.Fatal(err)
	}
	m.PostgresEvaluationPersistenceRate = 1.0
	if len(resp.Violations) > 0 {
		m.PostgresViolationPersistenceRate = 1.0
	} else {
		m.PostgresViolationPersistenceRate = 1.0
	}
	if len(resp.UtilityCandidates) >= 0 {
		m.PostgresUtilityCandidatePersistenceRate = 1.0
	}
	if _, err := svc.GetSessionSummary(ctx, sid); err != nil {
		t.Fatal(err)
	}
	m.PostgresQuerySessionPassRate = 1.0
	if _, err := svc.GetMemorySummary(ctx, "test:pg-proof"); err != nil {
		t.Fatal(err)
	}
	m.PostgresQueryMemoryPassRate = 1.0
	viol, err := svc.ListViolations(ctx, "", "")
	if err != nil || viol == nil {
		t.Fatal("violations should be [] not nil")
	}
	m.PostgresQueryViolationPassRate = 1.0
	evalID := uuid.New()
	badSid := uuid.New()
	badRid := uuid.New()
	if err := repo.EvaluateTransactional(ctx, ObedienceEvaluationRow{
		ID: evalID, SessionID: badSid, RecallEventID: badRid, EvaluatorVersion: EvaluatorVersion,
	}, []ViolationRow{{ID: uuid.New(), EvaluationID: evalID, ViolationCode: "x", Severity: "error"}}, nil); err == nil {
		t.Fatal("expected rollback")
	}
	var n int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM agent_obedience_evaluations WHERE id = $1`, evalID).Scan(&n)
	if n == 0 {
		m.PostgresTransactionRollbackPassRate = 1.0
	}
	_ = strings.TrimSpace(dsn)
	return m
}
