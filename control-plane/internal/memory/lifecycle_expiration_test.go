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

func TestService_Create_supersedes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	oldID := uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	newID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	dedup := DedupKey()
	canon := memorynorm.StatementCanonical("New decision")
	sk := memorynorm.StatementKey("New decision")

	mock.ExpectQuery(`INSERT INTO memories`).
		WithArgs(sqlmock.AnyArg(), "decision", "New decision", canon, sk, dedup, 5, "governing", "active", nil, nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at"}).
			AddRow(newID, api.MemoryKindDecision, "New decision", canon, sk, 5, "governing", "active", nil, nil, nil, time.Now(), time.Now(), nil))
	mock.ExpectQuery(`SELECT id, kind, statement`).WithArgs(oldID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at"}).
			AddRow(oldID, api.MemoryKindDecision, "Old decision", memorynorm.StatementCanonical("Old decision"), memorynorm.StatementKey("Old decision"), 5, "governing", "active", nil, nil, nil, time.Now(), time.Now(), nil))
	mock.ExpectQuery(`SELECT tag FROM memories_tags`).WithArgs(oldID).WillReturnRows(sqlmock.NewRows([]string{"tag"}))
	mock.ExpectExec(`UPDATE memories SET status = 'superseded', deprecated_at`).WithArgs(sqlmock.AnyArg(), oldID).WillReturnResult(sqlmock.NewResult(0, 1))

	dedupOff := false
	svc := &Service{Repo: &Repo{DB: db}, Dedup: &DedupConfig{Enabled: &dedupOff}}
	obj, err := svc.Create(ctx, CreateRequest{
		Kind:         api.MemoryKindDecision,
		Authority:    5,
		Statement:    "New decision",
		SupersedesID: &oldID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if obj.ID != newID {
		t.Errorf("Create: id = %v", obj.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestRepo_MarkSuperseded(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	id := uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	now := time.Now()
	mock.ExpectExec(`UPDATE memories SET status = 'superseded', deprecated_at`).WithArgs(now, id).WillReturnResult(sqlmock.NewResult(0, 1))

	err = (&Repo{DB: db}).MarkSuperseded(context.Background(), id, now)
	if err != nil {
		t.Fatalf("MarkSuperseded: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestRepo_UpdateStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	id := uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	mock.ExpectExec(`UPDATE memories SET status`).WithArgs(api.StatusArchived, id).WillReturnResult(sqlmock.NewResult(0, 1))

	err = (&Repo{DB: db}).UpdateStatus(context.Background(), id, api.StatusArchived)
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestService_ExpireMemories(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	id := uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	ttl := 60

	mock.ExpectQuery(`SELECT id, kind, statement`).
		WithArgs(2, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at"}).
			AddRow(id, api.MemoryKindDecision, "Low auth", memorynorm.StatementCanonical("Low auth"), memorynorm.StatementKey("Low auth"), 1, "governing", "active", nil, ttl, nil, time.Now().Add(-2*time.Hour), time.Now(), nil))
	mock.ExpectQuery(`SELECT tag FROM memories_tags`).WithArgs(id).WillReturnRows(
		sqlmock.NewRows([]string{"tag"}).AddRow("archive"))
	mock.ExpectExec(`UPDATE memories SET status`).WithArgs(api.StatusArchived, id).WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &Service{Repo: &Repo{DB: db}, Lifecycle: &LifecycleConfig{ExpirationAuthorityThreshold: 2}}
	count, err := svc.ExpireMemories(ctx, time.Now())
	if err != nil {
		t.Fatalf("ExpireMemories: %v", err)
	}
	if count != 1 {
		t.Errorf("ExpireMemories: archived = %d, want 1", count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}
