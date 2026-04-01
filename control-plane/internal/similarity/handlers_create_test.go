package similarity

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

type failingAutoDistiller struct{}

func (failingAutoDistiller) DistillAfterAdvisoryIngest(ctx context.Context, episodeID uuid.UUID) error {
	return errors.New("injected auto-distill failure")
}

// TestCreate_Advisory201WhenAutoDistillFails proves POST /v1/advisory-episodes still returns 201 if background distillation errors.
func TestCreate_Advisory201WhenAutoDistillFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	epID := uuid.New()
	now := time.Now().UTC()
	tagsJSON := []byte(`["t"]`)
	entsJSON := []byte(`[]`)

	mock.ExpectQuery(`INSERT INTO advisory_experiences`).
		WithArgs("payment failure error timeout duplicate charge", "manual", tagsJSON, nil, nil, entsJSON, FormationRejected).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(epID, now))
	mock.ExpectQuery(`SELECT id, summary_text, source, tags, related_memory_id, created_at, occurred_at, entities, memory_formation_status, rejection_reason FROM advisory_experiences WHERE id = \$1`).
		WithArgs(epID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "summary_text", "source", "tags", "related_memory_id", "created_at", "occurred_at", "entities", "memory_formation_status", "rejection_reason"}).
			AddRow(epID, "payment failure error timeout duplicate charge", "manual", tagsJSON, nil, now, nil, entsJSON, FormationRejected, nil))

	svc := &Service{
		Repo:   &Repo{DB: db},
		Config: &Config{Enabled: true},
	}
	h := &Handlers{
		Service:     svc,
		AutoDistill: failingAutoDistiller{},
	}

	body := `{"summary":"payment failure error timeout duplicate charge","source":"manual","tags":["t"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/advisory-episodes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201 got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), epID.String()) {
		t.Fatalf("response should include episode id: %s", w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
