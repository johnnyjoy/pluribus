package enforcement

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"control-plane/internal/app"
)

func TestHandlers_Evaluate_disabledReturns403(t *testing.T) {
	off := false
	h := &Handlers{Service: &Service{
		Config: &app.EnforcementConfig{Enabled: &off},
	}}
	body := `{"proposal_text":"We will use SQLite."}`
	req := httptest.NewRequest(http.MethodPost, "/v1/enforcement/evaluate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Evaluate(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandlers_Evaluate_badJSONReturns400(t *testing.T) {
	on := true
	h := &Handlers{Service: &Service{
		Config: &app.EnforcementConfig{Enabled: &on},
		Repo:   nil,
	}}
	req := httptest.NewRequest(http.MethodPost, "/v1/enforcement/evaluate", strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Evaluate(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandlers_Evaluate_validationErrorReturns400(t *testing.T) {
	on := true
	h := &Handlers{Service: &Service{
		Config: &app.EnforcementConfig{Enabled: &on},
		Repo:   nil,
	}}
	body := `{"proposal_text":"   "}`
	req := httptest.NewRequest(http.MethodPost, "/v1/enforcement/evaluate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Evaluate(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid proposal_text, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandlers_Evaluate_nilServiceReturns503(t *testing.T) {
	h := &Handlers{Service: nil}
	body := `{"proposal_text":"ok"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/enforcement/evaluate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Evaluate(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestHandlers_Evaluate_enabledMissingRepoReturns400(t *testing.T) {
	on := true
	h := &Handlers{Service: &Service{
		Config: &app.EnforcementConfig{Enabled: &on},
		Repo:   nil,
	}}
	body := `{"proposal_text":"No conflict here."}`
	req := httptest.NewRequest(http.MethodPost, "/v1/enforcement/evaluate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Evaluate(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (repo nil), got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "memory repo") {
		t.Fatalf("expected memory repo error in body: %s", w.Body.String())
	}
}
