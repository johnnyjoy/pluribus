package memory

import (
	"fmt"
	"strings"

	"control-plane/pkg/api"
)

// SemanticRetrievalConfig gates pgvector-backed semantic candidate retrieval.
// When Enabled is omitted (nil), retrieval defaults to on (situation-first recall).
// Set enabled: false to disable semantic expansion and embedding calls.
type SemanticRetrievalConfig struct {
	Enabled *bool `yaml:"enabled,omitempty"`
	// EmbeddingEndpoint is the OpenAI-compatible API base (e.g. https://api.openai.com/v1).
	EmbeddingEndpoint string `yaml:"embedding_endpoint"`
	EmbeddingModel    string `yaml:"embedding_model"`
	EmbeddingDimensions int  `yaml:"embedding_dimensions"`
	// EmbeddingAPIKey optional; falls back to EMBEDDING_API_KEY or OPENAI_API_KEY.
	EmbeddingAPIKey string `yaml:"embedding_api_key"`
	// MaxSemanticCandidates caps vector search rows (default 50).
	MaxSemanticCandidates int `yaml:"max_semantic_candidates"`
	// MinCosineSimilarity in [0,1]; default 0.35
	MinCosineSimilarity float64 `yaml:"min_cosine_similarity"`
	// LogSemanticMatches enables [SEMANTIC MATCH] slog lines.
	LogSemanticMatches bool `yaml:"log_semantic_matches"`
}

// EmbeddingTextForMemory returns normalized text for storing a memory embedding.
func EmbeddingTextForMemory(kind api.MemoryKind, statementCanonical, statement string) string {
	s := strings.TrimSpace(statementCanonical)
	if s == "" {
		s = strings.TrimSpace(statement)
	}
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return ""
	}
	return fmt.Sprintf("%s: %s", kind, s)
}

// RetrievalEnabled reports whether semantic retrieval should run. Nil Enabled defaults to true.
func (c *SemanticRetrievalConfig) RetrievalEnabled() bool {
	if c == nil {
		return false
	}
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}
