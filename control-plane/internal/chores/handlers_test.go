package chores

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
)

func newTestHandlers(t *testing.T) (*Handlers, sqlmock.Sqlmock, func()) {
	t.Helper()
	svc, mock, done := newTestService(t)
	return &Handlers{Service: svc}, mock, done
}

func TestHandlers_List_ok(t *testing.T) {
	h, mock, done := newTestHandlers(t)
	defer done()
	// List syncs review chores first (contradictions + quarantine), then lists.
	mock.ExpectQuery(`FROM contradiction_records`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "memory_id", "conflict_with_id"}))
	mock.ExpectQuery(`FROM memories`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status_reason"}))
	mock.ExpectQuery(`FROM curation_chores c`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "chore_type", "subject_memory_id", "related_memory_id", "evidence",
			"state", "resolution_action", "created_at", "resolved_at",
			"subject_statement", "related_statement", "vote_count",
		}))

	r := chi.NewRouter()
	r.Get("/v1/curation/chores", h.List)
	req := httptest.NewRequest(http.MethodGet, "/v1/curation/chores", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var resp ListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Chores == nil {
		t.Fatal("chores must serialize as an empty array, not null")
	}
}

func TestHandlers_Resolve_badID(t *testing.T) {
	h, _, done := newTestHandlers(t)
	defer done()
	r := chi.NewRouter()
	r.Post("/v1/curation/chores/{id}/resolve", h.Resolve)
	req := httptest.NewRequest(http.MethodPost, "/v1/curation/chores/not-a-uuid/resolve",
		strings.NewReader(`{"agent_id":"a","action":"coexist"}`))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
