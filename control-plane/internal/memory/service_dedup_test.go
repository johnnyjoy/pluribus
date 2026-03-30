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

func TestService_Create_duplicateBeforeInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	existingID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	stmt := "Same canonical statement"
	sk := memorynorm.StatementKey(stmt)
	dedup := DedupKey()
	mock.ExpectQuery(`SELECT id FROM memories`).
		WithArgs(string(api.MemoryKindDecision), dedup, sk).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(existingID))
	mock.ExpectQuery(`SELECT id, kind, statement`).
		WithArgs(existingID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at"}).
			AddRow(existingID, api.MemoryKindDecision, stmt, memorynorm.StatementCanonical(stmt), sk, 5, "governing", "active", nil, nil, nil, time.Now(), time.Now(), nil))
	mock.ExpectExec(`UPDATE memories SET authority`).
		WithArgs(6, existingID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &Service{Repo: &Repo{DB: db}}
	obj, err := svc.Create(ctx, CreateRequest{
		Kind:      api.MemoryKindDecision,
		Statement: stmt,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if obj == nil || obj.ID != existingID || obj.Authority != 6 {
		t.Fatalf("want reinforced existing memory, got %+v", obj)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestService_Create_dedupDisabled_skipsLookup(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	newID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	stmt := "Fresh row"
	dedupOff := false

	dedup := DedupKey()
	mock.ExpectQuery(`INSERT INTO memories`).
		WithArgs(sqlmock.AnyArg(), "decision", stmt, memorynorm.StatementCanonical(stmt), memorynorm.StatementKey(stmt), dedup, 0, "governing", "active", nil, nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at"}).
			AddRow(newID, api.MemoryKindDecision, stmt, memorynorm.StatementCanonical(stmt), memorynorm.StatementKey(stmt), 0, "governing", "active", nil, nil, nil, time.Now(), time.Now(), nil))
	svc := &Service{Repo: &Repo{DB: db}, Dedup: &DedupConfig{Enabled: &dedupOff}}
	_, err = svc.Create(ctx, CreateRequest{
		Kind:      api.MemoryKindDecision,
		Statement: stmt,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
