package ingest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlers_Cognition_badJSON(t *testing.T) {
	h := &Handlers{Service: NewService(nil)}
	req := httptest.NewRequest(http.MethodPost, "/v1/ingest/cognition", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Cognition(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
