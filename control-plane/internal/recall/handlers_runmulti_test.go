package recall

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlers_RunMulti_validationAndContract(t *testing.T) {
	h := &Handlers{Service: &Service{}}

	t.Run("bad json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/recall/run-multi", strings.NewReader("{"))
		w := httptest.NewRecorder()
		h.RunMulti(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("missing query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/recall/run-multi", strings.NewReader(`{"merge":true}`))
		w := httptest.NewRecorder()
		h.RunMulti(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("promote requires merge", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/recall/run-multi", strings.NewReader(`{"query":"x","promote":true,"merge":false}`))
		w := httptest.NewRecorder()
		h.RunMulti(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("contract defined, orchestration pending", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/recall/run-multi", strings.NewReader(`{"query":"x","merge":true}`))
		w := httptest.NewRecorder()
		h.RunMulti(w, req)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d", w.Code)
		}
	})
}

