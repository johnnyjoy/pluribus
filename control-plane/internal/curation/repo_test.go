package curation

import (
	"context"
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

	mock.ExpectQuery(`INSERT INTO candidate_events`).
		WithArgs(sqlmock.AnyArg(), "We must always use POST.", 0.4).
		WillReturnRows(sqlmock.NewRows([]string{"id", "raw_text", "salience_score", "promotion_status", "created_at"}).
			AddRow(uuid.MustParse("22222222-2222-2222-2222-222222222222"), "We must always use POST.", 0.4, "pending", time.Now()))

	c, err := repo.Create(ctx, "We must always use POST.", 0.4)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.PromotionStatus != "pending" {
		t.Errorf("got %+v", c)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestRepo_ListPending(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &Repo{DB: db}

	mock.ExpectQuery(`SELECT id, raw_text`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "raw_text", "salience_score", "promotion_status", "created_at", "proposal_json"}).
			AddRow(uuid.MustParse("33333333-3333-3333-3333-333333333333"), "Pending text", 0.6, "pending", time.Now(), nil))

	list, err := repo.ListPending(ctx)
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(list) != 1 || list[0].RawText != "Pending text" {
		t.Errorf("got %+v", list)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestRepo_UpdatePromotionStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &Repo{DB: db}
	id := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	mock.ExpectExec(`UPDATE candidate_events SET promotion_status`).
		WithArgs("promoted", id).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.UpdatePromotionStatus(ctx, id, "promoted"); err != nil {
		t.Fatalf("UpdatePromotionStatus: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestRepo_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := &Repo{DB: db}
	id := uuid.MustParse("55555555-5555-5555-5555-555555555555")

	mock.ExpectQuery(`SELECT id, raw_text`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "raw_text", "salience_score", "promotion_status", "created_at", "proposal_json"}).
			AddRow(id, "hello", 0.5, "pending", time.Now(), nil))

	c, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if c == nil || c.RawText != "hello" {
		t.Errorf("got %+v", c)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}
