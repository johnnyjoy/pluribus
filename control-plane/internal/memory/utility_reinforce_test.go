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

func TestDuplicateCreateDoesNotBlindlyIncreaseAuthority(t *testing.T) {
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
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at", "agent_id"}).
			AddRow(existingID, api.MemoryKindDecision, stmt, memorynorm.StatementCanonical(stmt), sk, 5, "governing", "active", nil, nil, nil, time.Now(), time.Now(), nil, nil))

	svc := &Service{Repo: &Repo{DB: db}, ReinforceDuplicateAuthority: false}
	obj, err := svc.Create(ctx, CreateRequest{Kind: api.MemoryKindDecision, Statement: stmt})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if obj.Authority != 5 {
		t.Fatalf("authority=%d want 5", obj.Authority)
	}
}

func TestDuplicateCreateLegacyReinforceWhenEnabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	existingID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	stmt := "Legacy reinforce"
	sk := memorynorm.StatementKey(stmt)
	mock.ExpectQuery(`SELECT id FROM memories`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(existingID))
	mock.ExpectQuery(`SELECT id, kind, statement`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at", "agent_id"}).
			AddRow(existingID, api.MemoryKindDecision, stmt, memorynorm.StatementCanonical(stmt), sk, 5, "governing", "active", nil, nil, nil, time.Now(), time.Now(), nil, nil))
	mock.ExpectExec(`UPDATE memories SET authority`).WithArgs(6, existingID).WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &Service{Repo: &Repo{DB: db}, ReinforceDuplicateAuthority: true}
	obj, err := svc.Create(ctx, CreateRequest{Kind: api.MemoryKindDecision, Statement: stmt})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if obj.Authority != 6 {
		t.Fatalf("legacy reinforce authority=%d", obj.Authority)
	}
}
