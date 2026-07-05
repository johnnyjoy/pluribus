package memory

import (
	"context"
	"testing"
	"time"

	"control-plane/pkg/api"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

type fixedEmbedder struct {
	vec []float32
}

func (f fixedEmbedder) Embed(context.Context, string) ([]float32, error) { return f.vec, nil }
func (f fixedEmbedder) Dimensions() int                                   { return len(f.vec) }

func TestTryMergeSemanticNearDuplicate_reinforcesDistinctAgent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	existingID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	vec := []float32{1, 0, 0, 0}
	dedupOff := false
	enabled := true

	svc := &Service{
		Repo:     &Repo{DB: db},
		Dedup:    &DedupConfig{Enabled: &dedupOff, SemanticConsolidateThreshold: 0.93},
		Semantic: &SemanticRetrievalConfig{Enabled: &enabled, EmbeddingDimensions: 4},
		Embedder: fixedEmbedder{vec: vec},
	}

	mock.ExpectQuery(`SELECT id, kind, statement`).
		WithArgs(sqlmock.AnyArg(), "active", sqlmock.AnyArg(), sqlmock.AnyArg(), 10).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "payload", "created_at", "updated_at", "occurred_at", "vec_dist",
		}).AddRow(existingID, "pattern", "Existing lesson about retries", "existing lesson about retries", "key-a", 3, "advisory", "active", nil, time.Now(), time.Now(), nil, 0.02))

	mock.ExpectQuery(`SELECT memory_id, tag FROM memories_tags`).
		WillReturnRows(sqlmock.NewRows([]string{"memory_id", "tag"}))

	mock.ExpectQuery(`SELECT id, kind, statement`).
		WithArgs(sqlmock.AnyArg(), "pending", sqlmock.AnyArg(), sqlmock.AnyArg(), 10).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "payload", "created_at", "updated_at", "occurred_at", "vec_dist",
		}))

	mock.ExpectQuery(`SELECT id, kind, statement`).
		WithArgs(existingID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at", "agent_id",
		}).AddRow(existingID, "pattern", "Existing lesson about retries", "existing lesson about retries", "key-a", 3, "advisory", "active", nil, nil, nil, time.Now(), time.Now(), nil, "author-a"))

	mock.ExpectExec(`UPDATE memories SET payload`).
		WithArgs(sqlmock.AnyArg(), existingID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE memories SET authority`).
		WithArgs(4, existingID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT id, kind, statement`).
		WithArgs(existingID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at", "agent_id",
		}).AddRow(existingID, "pattern", "Existing lesson about retries", "existing lesson about retries", "key-a", 4, "advisory", "active", nil, nil, []byte(`{"salience":{"distinct_contexts":1}}`), time.Now(), time.Now(), nil, "author-a"))

	req := CreateRequest{
		Kind:          api.MemoryKindPattern,
		Statement:     "Existing lesson about retry backoff with jitter",
		StatementKey:  "key-b",
		AgentID:       "agent-b",
		Embedding:     vec,
	}
	out, err := svc.tryMergeSemanticNearDuplicate(context.Background(), &req)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if out == nil || !out.Consolidated || out.ID != existingID || out.Authority != 4 {
		t.Fatalf("got %+v", out)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTryMergeSemanticNearDuplicate_sameAuthorNoAuthorityBump(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	existingID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	vec := []float32{0, 1, 0, 0}
	enabled := true

	svc := &Service{
		Repo:     &Repo{DB: db},
		Dedup:    &DedupConfig{SemanticConsolidateThreshold: 0.9},
		Semantic: &SemanticRetrievalConfig{Enabled: &enabled, EmbeddingDimensions: 4},
	}

	mock.ExpectQuery(`SELECT id, kind, statement`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "payload", "created_at", "updated_at", "occurred_at", "vec_dist",
		}).AddRow(existingID, "decision", "Use TLS everywhere", "use tls everywhere", "key-c", 5, "governing", "active", nil, time.Now(), time.Now(), nil, 0.05))

	mock.ExpectQuery(`SELECT memory_id, tag FROM memories_tags`).
		WillReturnRows(sqlmock.NewRows([]string{"memory_id", "tag"}))

	mock.ExpectQuery(`SELECT id, kind, statement`).
		WithArgs(sqlmock.AnyArg(), "pending", sqlmock.AnyArg(), sqlmock.AnyArg(), 10).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "payload", "created_at", "updated_at", "occurred_at", "vec_dist",
		}))

	mock.ExpectQuery(`SELECT id, kind, statement`).
		WithArgs(existingID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at", "agent_id",
		}).AddRow(existingID, "decision", "Use TLS everywhere", "use tls everywhere", "key-c", 5, "governing", "active", nil, nil, nil, time.Now(), time.Now(), nil, "author-same"))

	mock.ExpectExec(`UPDATE memories SET payload`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT id, kind, statement`).
		WithArgs(existingID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at", "agent_id",
		}).AddRow(existingID, "decision", "Use TLS everywhere", "use tls everywhere", "key-c", 5, "governing", "active", nil, nil, nil, time.Now(), time.Now(), nil, "author-same"))

	req := CreateRequest{
		Kind:         api.MemoryKindDecision,
		Statement:    "Always enable TLS for internal services",
		StatementKey: "key-d",
		AgentID:      "author-same",
		Embedding:    vec,
	}
	out, err := svc.tryMergeSemanticNearDuplicate(context.Background(), &req)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if out == nil || out.Authority != 5 {
		t.Fatalf("same author should not bump authority, got %+v", out)
	}
}

func TestTryMergeSemanticNearDuplicate_thresholdOffNoop(t *testing.T) {
	svc := &Service{Dedup: &DedupConfig{SemanticConsolidateThreshold: 0}}
	out, err := svc.tryMergeSemanticNearDuplicate(context.Background(), &CreateRequest{Embedding: []float32{1}})
	if err != nil || out != nil {
		t.Fatalf("want nil,nil got %v %v", out, err)
	}
}
