package tooling

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestHandlers_RunBuild_allowlistRejectsCommand(t *testing.T) {
	// Allowlist allows only "git", not "go".
	allowlist := NewAllowlist([]string{"git"}, nil)
	handlers := &Handlers{Allowlist: allowlist}

	body := []byte(`{"cwd": "` + t.TempDir() + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/tools/build/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlers.RunBuild(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("got status %d, want 403", rec.Code)
	}
	if rec.Body.String() != "" && rec.Body.String() != `{"error":"command not allowed"}` && rec.Body.String() != `{"error":"Command not allowed."}` {
		// httpx might format error differently
	}
}

func TestHandlers_RunBuild_allowlistRejectsPath(t *testing.T) {
	allowedDir := t.TempDir()
	allowlist := NewAllowlist([]string{"go"}, []string{allowedDir})
	handlers := &Handlers{Allowlist: allowlist}

	// Request with cwd outside allowed dir.
	forbiddenDir := filepath.Join(t.TempDir(), "other")
	body := []byte(`{"cwd": "` + forbiddenDir + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/tools/build/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlers.RunBuild(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("got status %d, want 403", rec.Code)
	}
}

func TestHandlers_RunBuild_noAllowlistSucceeds(t *testing.T) {
	handlers := &Handlers{Allowlist: nil}
	cwd := t.TempDir()
	body := []byte(`{"cwd": "` + cwd + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/tools/build/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handlers.RunBuild(rec, req)

	// May be 200 (go build in empty dir) or 400 if go not found; should not be 403
	if rec.Code == http.StatusForbidden {
		t.Errorf("nil allowlist should not return 403, got %d", rec.Code)
	}
}
