package ingest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestHandlers_Commit_badUUID(t *testing.T) {
	r := chi.NewRouter()
	h := &Handlers{Service: NewService(nil)}
	r.Post("/ingest/{id}/commit", h.Commit)

	req := httptest.NewRequest(http.MethodPost, "/ingest/not-a-uuid/commit", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandlers_Commit_notFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	mock.ExpectQuery(`SELECT status FROM ingestion_records WHERE id = \$1`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"status"}))

	svc := NewService(&Repo{DB: db})
	h := &Handlers{Service: svc}
	r := chi.NewRouter()
	r.Post("/ingest/{id}/commit", h.Commit)

	req := httptest.NewRequest(http.MethodPost, "/ingest/"+id.String()+"/commit", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
