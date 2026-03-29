package recall

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlers_GetBundle_validation(t *testing.T) {
	h := &Handlers{Service: &Service{}}

	t.Run("minimal query allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/recall/", nil)
		w := httptest.NewRecorder()
		h.GetBundle(w, req)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503 without compiler, got %d", w.Code)
		}
	})

	t.Run("invalid max_per_kind", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/recall/?retrieval_query=x&max_per_kind=not-int", nil)
		w := httptest.NewRecorder()
		h.GetBundle(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("no compiler with retrieval_query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/recall/?retrieval_query=ship+feature", nil)
		w := httptest.NewRecorder()
		h.GetBundle(w, req)
		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d", w.Code)
		}
	})
}

func TestParseQueryStringList_tags(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x?tags=a,b&tags=c&tags=b", nil)
	got := parseQueryStringList(req.URL.Query(), "tags")
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("got %#v", got)
	}
}

func TestParseQueryStringList_empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if len(parseQueryStringList(req.URL.Query(), "tags")) != 0 {
		t.Fatal("expected empty")
	}
}

func TestHandlers_GetBundle_invalidMaxPerKind(t *testing.T) {
	h := &Handlers{Service: &Service{}}
	req := httptest.NewRequest(http.MethodGet, "/v1/recall/?retrieval_query=x&max_per_kind=-1", nil)
	w := httptest.NewRecorder()
	h.GetBundle(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandlers_GetBundle_usesCompiler(t *testing.T) {
	h := &Handlers{Service: &Service{
		Compiler: &Compiler{},
	}}
	req := httptest.NewRequest(http.MethodGet, "/v1/recall/?retrieval_query=ship&tags=go,api", nil)
	w := httptest.NewRecorder()
	h.GetBundle(w, req)
	if w.Code == http.StatusServiceUnavailable && strings.Contains(w.Body.String(), ErrNoCompiler.Error()) {
		t.Fatalf("unexpected no compiler with Compiler set")
	}
}

func TestHandlers_GetBundle_acceptsRetrievalQueryAndQueryAlias(t *testing.T) {
	h := &Handlers{Service: &Service{Compiler: &Compiler{}}}
	for _, raw := range []string{
		"/v1/recall/?retrieval_query=ship+the+API&tags=go",
		"/v1/recall/?query=ship+the+API&tags=go",
	} {
		req := httptest.NewRequest(http.MethodGet, raw, nil)
		w := httptest.NewRecorder()
		h.GetBundle(w, req)
		if w.Code == http.StatusServiceUnavailable && strings.Contains(w.Body.String(), ErrNoCompiler.Error()) {
			t.Fatalf("unexpected no compiler with Compiler set: %s", raw)
		}
	}
}
