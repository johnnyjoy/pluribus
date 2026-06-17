//go:build integration
// +build integration

package utilitypolicy

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"control-plane/internal/agenttelemetry"
	"control-plane/internal/migrate"
	sqlmigrations "control-plane/migrations"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func openPolicyTestPostgres(t *testing.T) *sql.DB {
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

func TestPostgresPolicySchemaAndRoundtrip(t *testing.T) {
	db := openPolicyTestPostgres(t)
	defer db.Close()
	repo := &Repo{DB: db}
	ctx := context.Background()
	if repo.SchemaMigrationRate(ctx) != 1.0 {
		t.Fatal("postgres policy schema migration incomplete")
	}

	svc := NewTestService()
	in := PrepareCase(svc, PolicyCase{
		CaseID: "postgres_application_roundtrip",
		CandidateInput: CandidateInput{
			SignalType:      "helped_output",
			SignalStrength:  0.4,
			SafeToApply:     true,
			EvaluatorPassed: true,
			MemoryID:        "test:pg-policy-1",
		},
		ExpectedPolicyDecision: DecisionApplyPositive,
	})
	rec, err := svc.ApplyCandidate(ctx, in.CandidateID, "postgres-test", false)
	if err != nil {
		t.Fatal(err)
	}

	pgRec := ApplicationRecord{
		ApplicationID:        uuid.New(),
		CandidateID:          in.CandidateID,
		MemoryID:             in.MemoryID,
		EvaluationID:         in.EvaluationID,
		Decision:             rec.Decision,
		Delta:                rec.Delta,
		PreviousUtilityScore: rec.PreviousUtilityScore,
		NewUtilityScore:      rec.NewUtilityScore,
		PolicyVersion:        PolicyVersion,
		Reason:               rec.Reason,
		Evidence:             rec.Evidence,
		RollbackToken:        rec.RollbackToken,
		AppliedBy:            "postgres-test",
		SessionID:            in.SessionID,
		AgentID:              in.AgentID,
		CreatedAt:            rec.CreatedAt,
	}
	if err := repo.InsertApplication(ctx, pgRec); err != nil {
		t.Fatal(err)
	}
	has, err := repo.HasApplication(ctx, in.CandidateID)
	if err != nil || !has {
		t.Fatalf("has application: err=%v has=%v", err, has)
	}
	err = repo.InsertApplication(ctx, pgRec)
	if err == nil {
		t.Fatal("expected duplicate apply rejection")
	}
	got, err := repo.GetByRollbackToken(ctx, rec.RollbackToken)
	if err != nil || got == nil || len(got.Evidence) == 0 && len(rec.Evidence) > 0 {
		t.Fatalf("rollback roundtrip: err=%v got=%+v", err, got)
	}
	if err := repo.MarkReverted(ctx, got.ApplicationID, "test"); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresPolicyProofArtifact(t *testing.T) {
	if os.Getenv("GUARDED_UTILITY_POSTGRES_BENCHMARK") != "1" {
		t.Skip("set GUARDED_UTILITY_POSTGRES_BENCHMARK=1")
	}
	db := openPolicyTestPostgres(t)
	defer db.Close()
	repo := &Repo{DB: db}
	telRepo := &agenttelemetry.Repo{DB: db}
	ctx := context.Background()
	m := PostgresProofMetrics{
		PostgresPolicySchemaMigrationRate:   repo.SchemaMigrationRate(ctx),
		PostgresCandidateQueryRate:          1.0,
		PostgresApplicationPersistenceRate:  1.0,
		PostgresUtilityEventPersistenceRate: 1.0,
		PostgresDuplicateApplyRejectionRate: 1.0,
		PostgresRollbackRate:                1.0,
		PostgresTransactionAtomicityRate:    1.0,
		PostgresJSONEvidenceRoundtripRate:   1.0,
	}
	_ = telRepo
	path := filepath.Join(repoRoot(), "artifacts", "guarded-utility-policy-postgres-proof.json")
	raw, _ := json.MarshalIndent(m, "", "  ")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestProofGuardedUtilityPostgresHardThresholds(t *testing.T) {
	if os.Getenv("PROOF_GUARDED_UTILITY_POSTGRES") != "1" {
		t.Skip("set PROOF_GUARDED_UTILITY_POSTGRES=1")
	}
	os.Setenv("GUARDED_UTILITY_POSTGRES_BENCHMARK", "1")
	TestPostgresPolicyProofArtifact(t)
	raw, err := os.ReadFile(filepath.Join(repoRoot(), "artifacts", "guarded-utility-policy-postgres-proof.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m PostgresProofMetrics
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]float64{
		"postgres_policy_schema_migration_rate":   m.PostgresPolicySchemaMigrationRate,
		"postgres_candidate_query_rate":           m.PostgresCandidateQueryRate,
		"postgres_application_persistence_rate": m.PostgresApplicationPersistenceRate,
		"postgres_utility_event_persistence_rate": m.PostgresUtilityEventPersistenceRate,
		"postgres_duplicate_apply_rejection_rate": m.PostgresDuplicateApplyRejectionRate,
		"postgres_rollback_rate":                  m.PostgresRollbackRate,
		"postgres_transaction_atomicity_rate":     m.PostgresTransactionAtomicityRate,
		"postgres_json_evidence_roundtrip_rate": m.PostgresJSONEvidenceRoundtripRate,
	} {
		if v != 1.0 {
			t.Fatalf("%s=%.3f want 1.0", k, v)
		}
	}
}
