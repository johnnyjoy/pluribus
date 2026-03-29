package recall

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestRepo_CreateBundle(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &Repo{DB: db}
	payload := RecallBundle{
		GoverningConstraints: []MemoryItem{{ID: "a", Kind: "constraint", Statement: "x", Authority: 1}},
	}

	mock.ExpectExec(`INSERT INTO recall_bundles`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	id, err := repo.CreateBundle(ctx, payload)
	if err != nil {
		t.Fatalf("CreateBundle: %v", err)
	}
	if id == uuid.Nil {
		t.Error("expected non-nil id")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}
