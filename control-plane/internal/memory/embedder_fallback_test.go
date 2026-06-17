package memory_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"control-plane/internal/memory"
)

func fakeEmbedderServer(t *testing.T, status int, body string, delay time.Duration) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if delay > 0 {
			time.Sleep(delay)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

func okEmbeddingJSON(dim int) string {
	vec := make([]float64, dim)
	for i := range vec {
		vec[i] = float64(i+1) / float64(dim)
	}
	b, _ := json.Marshal(map[string]any{
		"data": []map[string]any{{"embedding": vec}},
	})
	return string(b)
}

func TestEmbedderTimeoutFallsBackLexical(t *testing.T) {
	srv := fakeEmbedderServer(t, 200, okEmbeddingJSON(8), 200*time.Millisecond)
	defer srv.Close()
	emb := memory.NewHTTPEmbedderForTest(srv.URL+"/v1", "test", 8, "http")
	emb.HTTPClient = &http.Client{Timeout: 5 * time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := emb.Embed(ctx, "query")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestEmbedderBadJSONFallsBackLexical(t *testing.T) {
	srv := fakeEmbedderServer(t, 200, "{not-json", 0)
	defer srv.Close()
	emb := memory.NewHTTPEmbedderForTest(srv.URL+"/v1", "test", 8, "http")
	_, err := emb.Embed(context.Background(), "query")
	if err == nil {
		t.Fatal("expected json error")
	}
}

func TestEmbedderDimensionMismatchRejected(t *testing.T) {
	srv := fakeEmbedderServer(t, 200, okEmbeddingJSON(4), 0)
	defer srv.Close()
	emb := memory.NewHTTPEmbedderForTest(srv.URL+"/v1", "test", 8, "http")
	_, err := emb.Embed(context.Background(), "query")
	if err == nil || !strings.Contains(err.Error(), "dim") {
		t.Fatalf("expected dimension error, got %v", err)
	}
}

func TestEmbedderEmptyVectorRejected(t *testing.T) {
	srv := fakeEmbedderServer(t, 200, `{"data":[{"embedding":[]}]}`, 0)
	defer srv.Close()
	emb := memory.NewHTTPEmbedderForTest(srv.URL+"/v1", "test", 8, "http")
	_, err := emb.Embed(context.Background(), "query")
	if err == nil {
		t.Fatal("expected empty embedding error")
	}
}

func TestEmbedderServerErrorFallsBackLexical(t *testing.T) {
	srv := fakeEmbedderServer(t, 500, `{"error":"boom"}`, 0)
	defer srv.Close()
	emb := memory.NewHTTPEmbedderForTest(srv.URL+"/v1", "test", 8, "http")
	_, err := emb.Embed(context.Background(), "query")
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected 500 error, got %v", err)
	}
}
