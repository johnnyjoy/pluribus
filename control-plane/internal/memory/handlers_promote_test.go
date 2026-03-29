package memory

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlers_Promote_validationAndContract(t *testing.T) {
	h := &Handlers{Service: &Service{}}

	t.Run("bad json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/memory/promote", strings.NewReader("{"))
		w := httptest.NewRecorder()
		h.Promote(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("missing type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/memory/promote", strings.NewReader(`{"content":"x"}`))
		w := httptest.NewRecorder()
		h.Promote(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid confidence", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/memory/promote", strings.NewReader(`{"type":"decision","content":"x","confidence":2}`))
		w := httptest.NewRecorder()
		h.Promote(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("repo not configured", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/memory/promote", strings.NewReader(`{"type":"decision","content":"x","confidence":0.7}`))
		w := httptest.NewRecorder()
		h.Promote(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}
