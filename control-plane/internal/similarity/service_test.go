package similarity

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestCreate_Disabled(t *testing.T) {
	s := &Service{
		Config: &Config{Enabled: false},
		Repo:   &Repo{},
	}
	_, err := s.Create(context.Background(), CreateRequest{
		Summary: "hello",
		Source:  "manual",
	})
	if err != ErrSimilarityDisabled {
		t.Fatalf("err=%v want ErrSimilarityDisabled", err)
	}
}

func TestFindSimilar_DisabledReturnsEmpty(t *testing.T) {
	s := &Service{Config: &Config{Enabled: false}}
	resp, err := s.FindSimilar(context.Background(), SimilarRequest{
		Query: "q",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil || len(resp.AdvisorySimilarCases) != 0 {
		t.Fatalf("got %+v", resp)
	}
}

func TestFindSimilar_LexicalOrdersByOverlap(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	id1 := uuid.New()
	id2 := uuid.New()
	t1 := time.Now().UTC().Add(-time.Minute)
	t2 := time.Now().UTC()

	mock.ExpectQuery(`SELECT id, summary_text, source, tags, related_memory_id, created_at`).
		WithArgs(500).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "summary_text", "source", "tags", "related_memory_id", "created_at",
		}).
			AddRow(id1, "webhook debugging notes", "manual", []byte(`[]`), nil, t1).
			AddRow(id2, "payment webhook timeout retry", "manual", []byte(`[]`), nil, t2))

	s := &Service{
		Repo:   &Repo{DB: db},
		Config: &Config{Enabled: true, MinResemblance: 0.05, MaxResults: 5},
	}
	resp, err := s.FindSimilar(context.Background(), SimilarRequest{
		Query: "payment webhook timeout",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.AdvisorySimilarCases) != 2 {
		t.Fatalf("want 2 cases, got %d", len(resp.AdvisorySimilarCases))
	}
	if resp.AdvisorySimilarCases[0].Summary != "payment webhook timeout retry" {
		t.Fatalf("first=%q want payment webhook...", resp.AdvisorySimilarCases[0].Summary)
	}
	if resp.AdvisorySimilarCases[0].ResemblanceScore < resp.AdvisorySimilarCases[1].ResemblanceScore {
		t.Fatalf("ordering wrong")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFindSimilar_TagFilter(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	id1 := uuid.New()
	id2 := uuid.New()
	now := time.Now().UTC()

	mock.ExpectQuery(`SELECT id, summary_text, source, tags, related_memory_id, created_at`).
		WithArgs(500).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "summary_text", "source", "tags", "related_memory_id", "created_at",
		}).
			AddRow(id1, "alpha beta topic one", "manual", []byte(`["a"]`), nil, now).
			AddRow(id2, "gamma delta topic two", "manual", []byte(`["b","c"]`), nil, now))

	s := &Service{
		Repo:   &Repo{DB: db},
		Config: &Config{Enabled: true, MinResemblance: 0.01},
	}
	resp, err := s.FindSimilar(context.Background(), SimilarRequest{
		Query: "topic two gamma",
		Tags:  []string{"b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.AdvisorySimilarCases) != 1 || resp.AdvisorySimilarCases[0].Summary != "gamma delta topic two" {
		t.Fatalf("got %+v", resp.AdvisorySimilarCases)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestFindSimilar_MinResemblanceFiltersWeak(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	id1 := uuid.New()
	now := time.Now().UTC()

	mock.ExpectQuery(`SELECT id, summary_text, source, tags, related_memory_id, created_at`).
		WithArgs(500).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "summary_text", "source", "tags", "related_memory_id", "created_at",
		}).
			AddRow(id1, "completely unrelated xyz", "manual", []byte(`[]`), nil, now))

	s := &Service{
		Repo:   &Repo{DB: db},
		Config: &Config{Enabled: true, MinResemblance: 0.5},
	}
	resp, err := s.FindSimilar(context.Background(), SimilarRequest{
		Query: "payment webhook",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.AdvisorySimilarCases) != 0 {
		t.Fatalf("expected filter out, got %d", len(resp.AdvisorySimilarCases))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
