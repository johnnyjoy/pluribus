package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWrapWithPluribusAuth_disabled(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	h := WrapWithPluribusAuth(next, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/recall/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusTeapot {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestWrapWithPluribusAuth_healthBypass(t *testing.T) {
	secret := []byte("sk-test")
	for _, path := range []string{"/healthz", "/readyz"} {
		var saw bool
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			saw = true
			w.WriteHeader(http.StatusOK)
		})
		h := WrapWithPluribusAuth(inner, secret)
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if !saw {
			t.Fatalf("%s: inner not called", path)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: code %d", path, rec.Code)
		}
	}
}

func TestWrapWithPluribusAuth_validHeader(t *testing.T) {
	secret := []byte("sk-good")
	var saw bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		saw = true
		w.WriteHeader(http.StatusOK)
	})
	h := WrapWithPluribusAuth(next, secret)
	req := httptest.NewRequest(http.MethodGet, "/v1/recall/", nil)
	req.Header.Set("X-API-Key", "sk-good")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !saw || rec.Code != http.StatusOK {
		t.Fatalf("saw=%v code=%d", saw, rec.Code)
	}
}

func TestWrapWithPluribusAuth_missingKey(t *testing.T) {
	secret := []byte("sk-good")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next")
	})
	h := WrapWithPluribusAuth(next, secret)
	req := httptest.NewRequest(http.MethodGet, "/v1/recall/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestWrapWithPluribusAuth_invalidKey(t *testing.T) {
	secret := []byte("sk-good")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next")
	})
	h := WrapWithPluribusAuth(next, secret)
	req := httptest.NewRequest(http.MethodGet, "/v1/recall/", nil)
	req.Header.Set("X-API-Key", "wrong")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestWrapWithPluribusAuth_mcpToken(t *testing.T) {
	secret := []byte("sk-mcp")
	var saw bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		saw = true
		w.WriteHeader(http.StatusOK)
	})
	h := WrapWithPluribusAuth(next, secret)
	req := httptest.NewRequest(http.MethodPost, "/v1/mcp?token=sk-mcp", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !saw || rec.Code != http.StatusOK {
		t.Fatalf("saw=%v code=%d", saw, rec.Code)
	}
}

func TestWrapWithPluribusAuth_mcpHeaderBeatsBadToken(t *testing.T) {
	secret := []byte("sk-good")
	var saw bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		saw = true
		w.WriteHeader(http.StatusOK)
	})
	h := WrapWithPluribusAuth(next, secret)
	req := httptest.NewRequest(http.MethodPost, "/v1/mcp?token=bad", nil)
	req.Header.Set("X-API-Key", "sk-good")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !saw || rec.Code != http.StatusOK {
		t.Fatalf("saw=%v code=%d", saw, rec.Code)
	}
}

func TestWrapWithPluribusAuth_mcpTokenInvalid(t *testing.T) {
	secret := []byte("sk-mcp")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next")
	})
	h := WrapWithPluribusAuth(next, secret)
	req := httptest.NewRequest(http.MethodPost, "/v1/mcp?token=bad", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestWrapWithPluribusAuth_tokenForbiddenOffMCP(t *testing.T) {
	secret := []byte("sk-x")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next")
	})
	h := WrapWithPluribusAuth(next, secret)
	req := httptest.NewRequest(http.MethodGet, "/v1/recall/?token=sk-x", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestWrapWithPluribusAuth_legacyQuery(t *testing.T) {
	secret := []byte("sk-x")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next")
	})
	h := WrapWithPluribusAuth(next, secret)
	req := httptest.NewRequest(http.MethodGet, "/v1/recall/?api_key=x", nil)
	req.Header.Set("X-API-Key", "sk-x")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestWrapWithPluribusAuth_authorizationHeaderRejected(t *testing.T) {
	secret := []byte("sk-x")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next")
	})
	h := WrapWithPluribusAuth(next, secret)
	req := httptest.NewRequest(http.MethodGet, "/v1/recall/", nil)
	req.Header.Set("X-API-Key", "sk-x")
	req.Header.Set("Authorization", "Bearer x")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestLoadPluribusAPIKey_trimEmpty(t *testing.T) {
	t.Setenv(PluribusAPIKeyEnv, "  ")
	if LoadPluribusAPIKey() != nil {
		t.Fatal("whitespace-only should be unset")
	}
	t.Setenv(PluribusAPIKeyEnv, "sk-ok")
	k := LoadPluribusAPIKey()
	if string(k) != "sk-ok" {
		t.Fatalf("got %q", k)
	}
}

func TestSecureEqual(t *testing.T) {
	if !secureEqual([]byte("a"), "a") {
		t.Fatal("equal")
	}
	if secureEqual([]byte("a"), "b") {
		t.Fatal("neq value")
	}
	if secureEqual([]byte("ab"), "a") {
		t.Fatal("neq len")
	}
}
