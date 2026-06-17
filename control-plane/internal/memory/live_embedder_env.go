package memory

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// LiveEmbedderEnvConfig is explicit opt-in configuration for real embedder evaluation.
type LiveEmbedderEnvConfig struct {
	Endpoint  string
	APIKey    string
	Model     string
	Provider  string
	Dimension int
	Source    string // env var names used (for reporting)
}

// LoadLiveEmbedderFromEnv loads real embedder settings from PLURIBUS_EMBEDDER_* variables.
// Returns error when endpoint is missing (honest opt-in gate).
func LoadLiveEmbedderFromEnv() (Embedder, LiveEmbedderEnvConfig, error) {
	cfg := LiveEmbedderEnvConfig{
		Endpoint: strings.TrimSpace(os.Getenv("PLURIBUS_EMBEDDER_ENDPOINT")),
		APIKey:   strings.TrimSpace(os.Getenv("PLURIBUS_EMBEDDER_API_KEY")),
		Model:    strings.TrimSpace(os.Getenv("PLURIBUS_EMBEDDER_MODEL")),
		Provider: strings.TrimSpace(os.Getenv("PLURIBUS_EMBEDDER_PROVIDER")),
		Source:   "PLURIBUS_EMBEDDER_*",
	}
	if cfg.Endpoint == "" {
		return NoopEmbedder{}, cfg, fmt.Errorf("live embedder: set PLURIBUS_EMBEDDER_ENDPOINT (opt-in evaluation only)")
	}
	if cfg.Model == "" {
		cfg.Model = "text-embedding-3-small"
	}
	if cfg.Provider == "" {
		cfg.Provider = "http"
	}
	dim := DefaultEmbeddingDimensions
	if s := strings.TrimSpace(os.Getenv("PLURIBUS_EMBEDDER_DIMENSION")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			dim = n
		}
	}
	cfg.Dimension = dim
	if cfg.APIKey == "" {
		cfg.APIKey = strings.TrimSpace(os.Getenv("EMBEDDING_API_KEY"))
	}
	if cfg.APIKey == "" {
		cfg.APIKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	}
	emb := &HTTPEmbedder{
		Endpoint: cfg.Endpoint,
		APIKey:   cfg.APIKey,
		Model:    cfg.Model,
		dim:      dim,
		provider: cfg.Provider,
	}
	return emb, cfg, nil
}
