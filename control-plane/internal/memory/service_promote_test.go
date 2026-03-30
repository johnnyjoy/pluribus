package memory

import (
	"context"
	"testing"
	"time"

	"control-plane/internal/memorynorm"
	"control-plane/pkg/api"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestService_Promote_persistsMemoryObject(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := &Repo{DB: db}
	dedupOff := false
	svc := &Service{Repo: repo, Dedup: &DedupConfig{Enabled: &dedupOff}}
	dedup := DedupKey()
	content := "merged result"
	canon := memorynorm.StatementCanonical(content)
	sk := memorynorm.StatementKey(content)

	mock.ExpectQuery(`INSERT INTO memories`).
		WithArgs(sqlmock.AnyArg(), "decision", content, canon, sk, dedup, 8, "advisory", "active", nil, nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at",
		}).AddRow(uuid.New(), api.MemoryKindDecision, content, canon, sk, 8, "advisory", "active", nil, nil, nil, time.Now(), time.Now(), nil))
	for _, tag := range []string{"experience", "source:recall.run-multi", "promoted"} {
		mock.ExpectExec(`INSERT INTO memories_tags`).WithArgs(sqlmock.AnyArg(), tag).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	resp, err := svc.Promote(context.Background(), PromoteRequest{
		Type:       "decision",
		Content:    content,
		Source:     "recall.run-multi",
		Tags:       []string{"experience"},
		Confidence: 0.8,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Promoted || resp.ID == "" {
		t.Fatalf("expected promoted response with id, got %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestService_Promote_constraintUsesGoverningApplicability(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := &Repo{DB: db}
	dedupOff := false
	svc := &Service{Repo: repo, Dedup: &DedupConfig{Enabled: &dedupOff}}
	dedup := DedupKey()
	content := "Do not use SQLite for durable control-plane storage."
	canon := memorynorm.StatementCanonical(content)
	sk := memorynorm.StatementKey(content)

	mock.ExpectQuery(`INSERT INTO memories`).
		WithArgs(sqlmock.AnyArg(), "constraint", content, canon, sk, dedup, sqlmock.AnyArg(), "governing", "active", nil, nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at",
		}).AddRow(uuid.New(), api.MemoryKindConstraint, content, canon, sk, 8, "governing", "active", nil, nil, nil, time.Now(), time.Now(), nil))
	for _, tag := range []string{"source:recall.run-multi", "promoted"} {
		mock.ExpectExec(`INSERT INTO memories_tags`).WithArgs(sqlmock.AnyArg(), tag).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	_, err = svc.Promote(context.Background(), PromoteRequest{
		Type:       "constraint",
		Content:    content,
		Source:     "recall.run-multi",
		Confidence: 0.8,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestService_Promote_requireReviewCreatesPending(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := &Repo{DB: db}
	dedupOff := false
	svc := &Service{Repo: repo, Dedup: &DedupConfig{Enabled: &dedupOff}}
	dedup := DedupKey()
	content := "merged result"
	canon := memorynorm.StatementCanonical(content)
	sk := memorynorm.StatementKey(content)

	mock.ExpectQuery(`INSERT INTO memories`).
		WithArgs(sqlmock.AnyArg(), "decision", content, canon, sk, dedup, 8, "advisory", string(api.StatusPending), nil, nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at",
		}).AddRow(uuid.New(), api.MemoryKindDecision, content, canon, sk, 8, "advisory", api.StatusPending, nil, nil, nil, time.Now(), time.Now(), nil))
	for _, tag := range []string{"experience", "source:recall.run-multi", "promoted"} {
		mock.ExpectExec(`INSERT INTO memories_tags`).WithArgs(sqlmock.AnyArg(), tag).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	resp, err := svc.Promote(context.Background(), PromoteRequest{
		Type:          "decision",
		Content:       content,
		Source:        "recall.run-multi",
		Tags:          []string{"experience"},
		Confidence:    0.8,
		RequireReview: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != string(api.StatusPending) {
		t.Fatalf("status = %q, want pending", resp.Status)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
