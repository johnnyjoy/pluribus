package evidence

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
	now := time.Now()

	mock.ExpectQuery(`INSERT INTO evidence_records`).
		WithArgs(sqlmock.AnyArg(), "sha256:abc", "/path/to/file", "log").
		WillReturnRows(sqlmock.NewRows([]string{"id", "digest", "path", "kind", "created_at"}).
			AddRow(uuid.New(), "sha256:abc", "/path/to/file", "log", now))

	rec, err := repo.Create(ctx, "sha256:abc", "/path/to/file", "log")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rec.Digest != "sha256:abc" || rec.Path != "/path/to/file" {
		t.Errorf("rec = %+v", rec)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestRepo_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	repo := &Repo{DB: db}
	now := time.Now()

	mock.ExpectQuery(`SELECT id, digest`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "digest", "path", "kind", "created_at"}).
			AddRow(uuid.New(), "d1", "/p1", "log", now).
			AddRow(uuid.New(), "d2", "/p2", "test", now))

	list, err := repo.List(ctx, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d records", len(list))
	}
	if list[0].Digest != "d1" || list[1].Kind != "test" {
		t.Errorf("list = %+v", list)
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
	id := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	now := time.Now()

	mock.ExpectQuery(`SELECT id, digest, path, COALESCE`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "digest", "path", "kind", "created_at"}).
			AddRow(id, "dig", "/path", "build", now))

	rec, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if rec == nil || rec.ID != id || rec.Kind != "build" {
		t.Errorf("rec = %+v", rec)
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

	mock.ExpectQuery(`SELECT id, digest, path, COALESCE`).WithArgs(id).WillReturnError(sql.ErrNoRows)

	rec, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if rec != nil {
		t.Errorf("expected nil: %+v", rec)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestRepo_CreateLink(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	repo := &Repo{DB: db}
	memID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	evID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	mock.ExpectExec(`INSERT INTO memory_evidence_links`).
		WithArgs(memID, evID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.CreateLink(ctx, memID, evID)
	if err != nil {
		t.Fatalf("CreateLink: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestRepo_ListEvidenceByMemory(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	repo := &Repo{DB: db}
	memID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	evID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT e.id, e.digest`).
		WithArgs(memID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "digest", "path", "kind", "created_at"}).
			AddRow(evID, "d1", "/p1", KindTest, now))

	list, err := repo.ListEvidenceByMemory(ctx, memID)
	if err != nil {
		t.Fatalf("ListEvidenceByMemory: %v", err)
	}
	if len(list) != 1 || list[0].Kind != KindTest || list[0].ID != evID {
		t.Errorf("list = %+v", list)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}
