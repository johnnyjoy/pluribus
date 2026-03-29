package evidence

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestService_ScoreEvidenceIDs_average(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	e1 := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	e2 := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	now := time.Now()
	mock.ExpectQuery(`SELECT id, digest, path, COALESCE`).
		WithArgs(e1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "digest", "path", "kind", "created_at"}).
			AddRow(e1, "d", "p", KindTest, now))
	mock.ExpectQuery(`SELECT id, digest, path, COALESCE`).
		WithArgs(e2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "digest", "path", "kind", "created_at"}).
			AddRow(e2, "d2", "p2", KindLog, now))
	svc := &Service{Repo: &Repo{DB: db}}
	score, err := svc.ScoreEvidenceIDs(context.Background(), []uuid.UUID{e1, e2})
	if err != nil {
		t.Fatal(err)
	}
	want := (1.0 + 0.5) / 2
	if score != want {
		t.Fatalf("score = %v, want %v", score, want)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
