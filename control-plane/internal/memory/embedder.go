package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Embedder computes dense vectors for a single text input (no batching).
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Dimensions() int
}

// NoopEmbedder returns zero vectors (no-op when semantic retrieval is off).
type NoopEmbedder struct{}

func (NoopEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, nil
}

func (NoopEmbedder) Dimensions() int { return 0 }

// HTTPEmbedder calls an OpenAI-compatible POST /v1/embeddings endpoint.
type HTTPEmbedder struct {
	Endpoint   string
	APIKey     string
	Model      string
	dim        int
	HTTPClient *http.Client
}

func (e *HTTPEmbedder) Dimensions() int {
	if e == nil {
		return 0
	}
	return e.dim
}

func (e *HTTPEmbedder) client() *http.Client {
	if e != nil && e.HTTPClient != nil {
		return e.HTTPClient
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (e *HTTPEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if e == nil || strings.TrimSpace(e.Endpoint) == "" {
		return nil, fmt.Errorf("embedder: no endpoint")
	}
	if e.dim <= 0 {
		return nil, fmt.Errorf("embedder: no dimensions")
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("embedder: empty text")
	}
	// OpenAI-compatible JSON
	body := map[string]any{
		"input": text,
		"model": e.Model,
	}
	if e.Model == "" {
		body["model"] = "text-embedding-3-small"
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(e.Endpoint, "/")+"/embeddings", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if e.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.APIKey)
	}
	resp, err := e.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embedder: HTTP %d: %s", resp.StatusCode, truncateForErr(b, 200))
	}
	var wrap struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &wrap); err != nil {
		return nil, err
	}
	if len(wrap.Data) == 0 || len(wrap.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("embedder: empty embedding in response")
	}
	out := make([]float32, len(wrap.Data[0].Embedding))
	for i, v := range wrap.Data[0].Embedding {
		out[i] = float32(v)
	}
	if len(out) != e.dim {
		return nil, fmt.Errorf("embedder: got dim %d want %d", len(out), e.dim)
	}
	return out, nil
}

func truncateForErr(b []byte, limit int) string {
	s := string(b)
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "…"
}

// NewEmbedderFromConfig returns a noop embedder when disabled or misconfigured.
func NewEmbedderFromConfig(cfg *SemanticRetrievalConfig) Embedder {
	if cfg == nil || !cfg.RetrievalEnabled() {
		return NoopEmbedder{}
	}
	dim := cfg.EmbeddingDimensions
	if dim <= 0 {
		dim = DefaultEmbeddingDimensions
	}
	ep := strings.TrimSpace(cfg.EmbeddingEndpoint)
	if ep == "" {
		return NoopEmbedder{}
	}
	key := strings.TrimSpace(cfg.EmbeddingAPIKey)
	if key == "" {
		key = os.Getenv("EMBEDDING_API_KEY")
	}
	if key == "" {
		key = os.Getenv("OPENAI_API_KEY")
	}
	return &HTTPEmbedder{
		Endpoint: ep,
		APIKey:   key,
		Model:    strings.TrimSpace(cfg.EmbeddingModel),
		dim:      dim,
	}
}
