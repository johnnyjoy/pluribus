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

func TestListExpiredCandidates_excludesHistoricalValueSignals(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// Hardened query returns no rows when historical-value exclusions apply.
	mock.ExpectQuery(`SELECT m.id`).
		WithArgs(2, sqlmock.AnyArg(), OccurredAtMaterialDeltaSeconds, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at",
		}))

	list, err := (&Repo{DB: db}).ListExpiredCandidates(context.Background(), 2, time.Now(), nil)
	if err != nil {
		t.Fatalf("ListExpiredCandidates: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("len=%d want 0", len(list))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestService_ExpireMemories_skipsHistoricalValueCandidates(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(`SELECT m.id`).
		WithArgs(2, sqlmock.AnyArg(), OccurredAtMaterialDeltaSeconds, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at",
		}))

	svc := &Service{Repo: &Repo{DB: db}, Lifecycle: &LifecycleConfig{ExpirationAuthorityThreshold: 2}}
	count, err := svc.ExpireMemories(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("ExpireMemories: %v", err)
	}
	if count != 0 {
		t.Errorf("archived=%d want 0", count)
	}
}

func TestService_ExpireMemories_archivesDisposableCandidate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	id := uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	ttl := 60

	mock.ExpectQuery(`SELECT m.id`).
		WithArgs(2, sqlmock.AnyArg(), OccurredAtMaterialDeltaSeconds, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at",
		}).AddRow(id, api.MemoryKindDecision, "Low auth disposable", memorynorm.StatementCanonical("Low auth disposable"), memorynorm.StatementKey("Low auth disposable"), 1, "governing", "active", nil, ttl, nil, time.Now().Add(-2*time.Hour), time.Now(), nil))
	mock.ExpectQuery(`SELECT tag FROM memories_tags`).WithArgs(id).WillReturnRows(
		sqlmock.NewRows([]string{"tag"}).AddRow("ephemeral"))
	mock.ExpectExec(`UPDATE memories SET status`).WithArgs(api.StatusArchived, id).WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &Service{Repo: &Repo{DB: db}, Lifecycle: &LifecycleConfig{ExpirationAuthorityThreshold: 2}}
	count, err := svc.ExpireMemories(ctx, time.Now())
	if err != nil {
		t.Fatalf("ExpireMemories: %v", err)
	}
	if count != 1 {
		t.Errorf("archived=%d want 1", count)
	}
}

func TestListExpiredCandidates_includesNullPayloadDisposable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	id := uuid.MustParse("b0000000-0000-0000-0000-000000000001")
	created := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT m.id`).
		WithArgs(2, sqlmock.AnyArg(), OccurredAtMaterialDeltaSeconds, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at",
		}).AddRow(id, api.MemoryKindState, "Disposable null payload", "disposable null payload", "disposable", 1, "advisory", "active", nil, 60, nil, created, created, nil))
	mock.ExpectQuery(`SELECT tag FROM memories_tags`).WithArgs(id).WillReturnRows(
		sqlmock.NewRows([]string{"tag"}).AddRow("ephemeral"))

	list, err := (&Repo{DB: db}).ListExpiredCandidates(context.Background(), 2, time.Now(), nil)
	if err != nil {
		t.Fatalf("ListExpiredCandidates: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len=%d want 1 (NULL payload must not block TTL candidacy)", len(list))
	}
}

func TestArchiveDoesNotDeleteMemory_rowPreserved(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	id := uuid.MustParse("c0000000-0000-0000-0000-000000000001")
	mock.ExpectExec(`UPDATE memories SET status`).WithArgs(api.StatusArchived, id).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := (&Repo{DB: db}).UpdateStatus(context.Background(), id, api.StatusArchived); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}
