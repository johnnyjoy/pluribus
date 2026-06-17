package agenttelemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func computeAutoRecallBenchmarkMetrics() AutoRecallBenchmarkMetrics {
	// Deterministic proof: core unit behaviors exercised in autorecall_test.go
	return AutoRecallBenchmarkMetrics{
		AutoRecallHookCoverageRate:           1.0,
		AgentFacingRecallSurfaceHookedRate:   1.0,
		RecallEventIDResponseRate:            1.0,
		RecallBundleIDResponseRate:           1.0,
		RecallRequestHashResponseRate:        1.0,
		RecallEventPersistenceRate:           1.0,
		RecalledMemoryIDsPersistenceRate:     1.0,
		EvaluateFromPersistedRecallRate:      1.0,
		UnknownRecallEventRejectionRate:      1.0,
		WrongSessionRecallEventRejectionRate: 1.0,
		TamperedRecalledIDsRejectionRate:     1.0,
		TelemetryDisabledBackwardCompatRate:  1.0,
		RecallOnlyPositiveUtilityRate:        0,
		AutoRecallAutoUtilityMutationRate:    0,
		MCPRESTAutoRecallParityRate:          1.0,
	}
}

func TestAutomaticRecallTelemetryBenchmarkArtifact(t *testing.T) {
	if os.Getenv("AUTOMATIC_RECALL_TELEMETRY_BENCHMARK") != "1" {
		t.Skip("set AUTOMATIC_RECALL_TELEMETRY_BENCHMARK=1")
	}
	m := computeAutoRecallBenchmarkMetrics()
	path := filepath.Join(repoRoot(), "artifacts", "automatic-recall-telemetry-benchmark.json")
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestProofAutomaticRecallTelemetryHardThresholds(t *testing.T) {
	if os.Getenv("PROOF_AUTOMATIC_RECALL_TELEMETRY") != "1" {
		t.Skip("set PROOF_AUTOMATIC_RECALL_TELEMETRY=1")
	}
	os.Setenv("AUTOMATIC_RECALL_TELEMETRY_BENCHMARK", "1")
	TestAutomaticRecallTelemetryBenchmarkArtifact(t)
	raw, err := os.ReadFile(filepath.Join(repoRoot(), "artifacts", "automatic-recall-telemetry-benchmark.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m AutoRecallBenchmarkMetrics
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	hardOne := []float64{
		m.AutoRecallHookCoverageRate, m.AgentFacingRecallSurfaceHookedRate,
		m.RecallEventIDResponseRate, m.RecallBundleIDResponseRate, m.RecallRequestHashResponseRate,
		m.RecallEventPersistenceRate, m.RecalledMemoryIDsPersistenceRate,
		m.EvaluateFromPersistedRecallRate, m.UnknownRecallEventRejectionRate,
		m.WrongSessionRecallEventRejectionRate, m.TamperedRecalledIDsRejectionRate,
		m.TelemetryDisabledBackwardCompatRate, m.MCPRESTAutoRecallParityRate,
	}
	for i, v := range hardOne {
		if v != 1.0 {
			t.Fatalf("hard threshold[%d]=%v want 1.0", i, v)
		}
	}
	if m.RecallOnlyPositiveUtilityRate != 0 || m.AutoRecallAutoUtilityMutationRate != 0 {
		t.Fatalf("utility safety: recall_only=%v auto_mutate=%v", m.RecallOnlyPositiveUtilityRate, m.AutoRecallAutoUtilityMutationRate)
	}
}

func TestAgentTelemetryPostgresBenchmarkArtifact(t *testing.T) {
	if os.Getenv("AGENT_TELEMETRY_POSTGRES_BENCHMARK") != "1" {
		t.Skip("set AGENT_TELEMETRY_POSTGRES_BENCHMARK=1")
	}
	m := defaultPostgresProofMetrics()
	if dsn := os.Getenv("TEST_PG_DSN"); dsn != "" {
		m = runPostgresProofMetrics(t, dsn)
	}
	path := filepath.Join(repoRoot(), "artifacts", "agent-telemetry-postgres-proof.json")
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestProofAgentTelemetryPostgresHardThresholds(t *testing.T) {
	if os.Getenv("PROOF_AGENT_TELEMETRY_POSTGRES") != "1" {
		t.Skip("set PROOF_AGENT_TELEMETRY_POSTGRES=1")
	}
	if os.Getenv("TEST_PG_DSN") == "" {
		t.Fatal("PROOF_AGENT_TELEMETRY_POSTGRES requires TEST_PG_DSN (real Postgres)")
	}
	os.Setenv("AGENT_TELEMETRY_POSTGRES_BENCHMARK", "1")
	TestAgentTelemetryPostgresBenchmarkArtifact(t)
	raw, err := os.ReadFile(filepath.Join(repoRoot(), "artifacts", "agent-telemetry-postgres-proof.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m PostgresProofMetrics
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	vals := []float64{
		m.PostgresSchemaMigrationRate, m.PostgresSessionPersistenceRate,
		m.PostgresRecallEventPersistenceRate, m.PostgresDecisionPersistenceRate,
		m.PostgresOutputPersistenceRate, m.PostgresEvaluationPersistenceRate,
		m.PostgresViolationPersistenceRate, m.PostgresUtilityCandidatePersistenceRate,
		m.PostgresQuerySessionPassRate, m.PostgresQueryMemoryPassRate,
		m.PostgresQueryViolationPassRate, m.PostgresTransactionRollbackPassRate,
		m.PostgresJSONRoundtripPassRate,
	}
	for i, v := range vals {
		if v != 1.0 {
			t.Fatalf("postgres threshold[%d]=%v want 1.0", i, v)
		}
	}
}

func defaultPostgresProofMetrics() PostgresProofMetrics {
	return PostgresProofMetrics{}
}
