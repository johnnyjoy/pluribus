package memory

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"control-plane/internal/memorynorm"
	"control-plane/pkg/api"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

// Duplicate statement reinforces authority on the existing row (returns 200 + updated object).
func TestHandlers_Create_duplicateMemory_reinforces(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	existingID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	stmt := "Duplicate canonical text"
	sk := memorynorm.StatementKey(stmt)
	dedup := DedupKey()
	mock.ExpectQuery(`SELECT id FROM memories`).
		WithArgs(string(api.MemoryKindDecision), dedup, sk).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(existingID))
	mock.ExpectQuery(`SELECT id, kind, statement`).
		WithArgs(existingID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "statement", "statement_canonical", "statement_key", "authority", "applicability", "status", "deprecated_at", "ttl_seconds", "payload", "created_at", "updated_at", "occurred_at"}).
			AddRow(existingID, api.MemoryKindDecision, stmt, memorynorm.StatementCanonical(stmt), sk, 5, "governing", "active", nil, nil, nil, time.Now(), time.Now(), nil))
	mock.ExpectExec(`UPDATE memories SET authority`).
		WithArgs(6, existingID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := &Service{Repo: &Repo{DB: db}}
	h := &Handlers{Service: svc}

	body := fmt.Sprintf(`{"kind":"decision","authority":5,"statement":%q}`, stmt)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/memory", strings.NewReader(body))
	h.Create(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var obj MemoryObject
	if err := json.Unmarshal(w.Body.Bytes(), &obj); err != nil {
		t.Fatalf("json: %v", err)
	}
	if obj.ID != existingID || obj.Authority != 6 {
		t.Fatalf("got %+v", obj)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
