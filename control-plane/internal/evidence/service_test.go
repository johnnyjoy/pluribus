package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestService_Create_verify_file_and_DB_record(t *testing.T) {
	dir := t.TempDir()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	content := []byte("evidence body")
	contentB64 := base64.StdEncoding.EncodeToString(content)
	h := sha256.Sum256(content)
	digest := fmt.Sprintf("sha256:%x", h[:])
	expectedPath := filepath.Join(dir, "test", digest)
	now := time.Now()

	mock.ExpectQuery(`INSERT INTO evidence_records`).
		WithArgs(sqlmock.AnyArg(), digest, expectedPath, "test").
		WillReturnRows(sqlmock.NewRows([]string{"id", "digest", "path", "kind", "created_at"}).
			AddRow(uuid.New(), digest, expectedPath, "test", now))

	svc := &Service{
		Repo:    &Repo{DB: db},
		Storage: &Storage{RootPath: dir},
	}
	rec, err := svc.Create(context.Background(), CreateRequest{
		Kind:    "test",
		Content: contentB64,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rec.Kind != "test" {
		t.Errorf("rec = %+v", rec)
	}
	if rec.Path == "" {
		t.Error("rec.Path empty")
	}
	data, err := os.ReadFile(rec.Path)
	if err != nil {
		t.Fatalf("ReadFile %q: %v", rec.Path, err)
	}
	if string(data) != string(content) {
		t.Errorf("file content = %q", data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestService_List_and_Get(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	now := time.Now()

	mock.ExpectQuery(`SELECT id, digest, path`).
		WithArgs().
		WillReturnRows(sqlmock.NewRows([]string{"id", "digest", "path", "kind", "created_at"}).
			AddRow(uuid.New(), "d1", "/p1", "log", now))

	svc := &Service{Repo: &Repo{DB: db}}
	list, err := svc.List(ctx, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].Kind != "log" {
		t.Errorf("list = %+v", list)
	}

	id := list[0].ID
	mock.ExpectQuery(`SELECT id, digest, path`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id", "digest", "path", "kind", "created_at"}).
			AddRow(id, "d1", "/p1", "log", now))

	rec, err := svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if rec.ID != id || rec.Digest != "d1" {
		t.Errorf("rec = %+v", rec)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestService_Get_notFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	id := uuid.New()
	mock.ExpectQuery(`SELECT id, digest, path`).WithArgs(id).WillReturnError(sql.ErrNoRows)

	svc := &Service{Repo: &Repo{DB: db}}
	_, err = svc.Get(ctx, id)
	if err != ErrNotFound {
		t.Errorf("got err %v, want ErrNotFound", err)
	}
}

func TestService_LinkEvidenceToMemory(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	memID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	evID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	mock.ExpectExec(`INSERT INTO memory_evidence_links`).
		WithArgs(memID, evID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &Service{Repo: &Repo{DB: db}}
	err = svc.LinkEvidenceToMemory(ctx, memID, evID)
	if err != nil {
		t.Fatalf("LinkEvidenceToMemory: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestService_ComputeEvidenceScore(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	memID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	now := time.Now()

	// No evidence -> 0
	mock.ExpectQuery(`SELECT e.id`).
		WithArgs(memID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "digest", "path", "kind", "created_at"}))
	svc := &Service{Repo: &Repo{DB: db}}
	score, err := svc.ComputeEvidenceScore(ctx, memID)
	if err != nil {
		t.Fatalf("ComputeEvidenceScore: %v", err)
	}
	if score != 0 {
		t.Errorf("ComputeEvidenceScore(no evidence) = %v, want 0", score)
	}

	// One test (1.0) + one log (0.5) -> avg 0.75
	mock.ExpectQuery(`SELECT e.id`).
		WithArgs(memID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "digest", "path", "kind", "created_at"}).
			AddRow(uuid.New(), "d1", "/p1", KindTest, now).
			AddRow(uuid.New(), "d2", "/p2", KindLog, now))
	score, err = svc.ComputeEvidenceScore(ctx, memID)
	if err != nil {
		t.Fatalf("ComputeEvidenceScore: %v", err)
	}
	if score != 0.75 {
		t.Errorf("ComputeEvidenceScore(test+log) = %v, want 0.75", score)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestService_ListEvidenceForMemory(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	memID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	now := time.Now()

	mock.ExpectQuery(`SELECT e.id`).
		WithArgs(memID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "digest", "path", "kind", "created_at"}).
			AddRow(uuid.New(), "dig", "/path", KindBenchmark, now))
	svc := &Service{Repo: &Repo{DB: db}}
	list, err := svc.ListEvidenceForMemory(ctx, memID)
	if err != nil {
		t.Fatalf("ListEvidenceForMemory: %v", err)
	}
	if len(list) != 1 || list[0].Kind != KindBenchmark {
		t.Errorf("list = %+v", list)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}
