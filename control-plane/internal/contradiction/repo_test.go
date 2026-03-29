package contradiction

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestRepo_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &Repo{DB: db}
	mem1 := uuid.MustParse("a1000000-0000-0000-0000-000000000001")
	mem2 := uuid.MustParse("a1000000-0000-0000-0000-000000000002")
	recID := uuid.MustParse("b2000000-0000-0000-0000-000000000001")
	now := time.Now()

	mock.ExpectQuery(`INSERT INTO contradiction_records`).
		WithArgs(sqlmock.AnyArg(), mem1, mem2, "unresolved").
		WillReturnRows(sqlmock.NewRows([]string{"id", "memory_id", "conflict_with_id", "resolution_state", "created_at", "updated_at"}).
			AddRow(recID, mem1, mem2, ResolutionUnresolved, now, now))

	rec, err := repo.Create(ctx, CreateRequest{MemoryID: mem1, ConflictWithID: mem2})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rec.ID != recID || rec.MemoryID != mem1 || rec.ConflictWithID != mem2 || rec.ResolutionState != ResolutionUnresolved {
		t.Errorf("got %+v", rec)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestRepo_ListMemoryIDsInUnresolved(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &Repo{DB: db}
	id1 := uuid.MustParse("a1000000-0000-0000-0000-000000000001")
	id2 := uuid.MustParse("a1000000-0000-0000-0000-000000000002")

	// Single UNION query returns memory_id then conflict_with_id rows; column name from first SELECT
	mock.ExpectQuery(`SELECT memory_id FROM contradiction_records`).
		WillReturnRows(sqlmock.NewRows([]string{"memory_id"}).AddRow(id1).AddRow(id2))

	ids, err := repo.ListMemoryIDsInUnresolved(ctx)
	if err != nil {
		t.Fatalf("ListMemoryIDsInUnresolved: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("len(ids) = %d, want 2 (unique)", len(ids))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestRepo_UpdateResolution(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &Repo{DB: db}
	recID := uuid.MustParse("b2000000-0000-0000-0000-000000000001")

	mock.ExpectExec(`UPDATE contradiction_records SET resolution_state`).
		WithArgs(ResolutionOverride, recID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateResolution(ctx, recID, ResolutionOverride)
	if err != nil {
		t.Fatalf("UpdateResolution: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestRepo_GetByID_notFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &Repo{DB: db}
	id := uuid.New()

	mock.ExpectQuery(`SELECT .* FROM contradiction_records WHERE id`).
		WithArgs(id).
		WillReturnError(sql.ErrNoRows)

	rec, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if rec != nil {
		t.Errorf("GetByID: want nil, got %+v", rec)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}
