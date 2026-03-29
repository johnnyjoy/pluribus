package drift

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestRepo_CreateCheck(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &Repo{DB: db}
	violations := []DriftIssue{{Code: "constraint", Statement: "no globals"}}

	mock.ExpectExec(`INSERT INTO drift_checks`).
		WithArgs(sqlmock.AnyArg(), false, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	id, err := repo.CreateCheck(ctx, false, violations, nil)
	if err != nil {
		t.Fatalf("CreateCheck: %v", err)
	}
	if id == uuid.Nil {
		t.Error("expected non-nil id")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}
