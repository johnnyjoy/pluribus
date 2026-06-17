package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"control-plane/pkg/api"
)

const (
	EmbeddingStatusValid  = "valid"
	EmbeddingStatusStale  = "stale"
	EmbeddingStatusFailed = "failed"
	EmbeddingStatusMissing = "missing"
)

// EmbeddingWriteMeta is persisted alongside a pgvector embedding.
type EmbeddingWriteMeta struct {
	Model      string
	Provider   string
	Dimension  int
	SourceHash string
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// EmbeddingMeta on a loaded memory row (optional; populated when columns present).
type EmbeddingMeta struct {
	Model      string     `json:"embedding_model,omitempty"`
	Provider   string     `json:"embedding_provider,omitempty"`
	Dimension  int        `json:"embedding_dimension,omitempty"`
	SourceHash string     `json:"embedding_source_hash,omitempty"`
	CreatedAt  *time.Time `json:"embedding_created_at,omitempty"`
	UpdatedAt  *time.Time `json:"embedding_updated_at,omitempty"`
	Status     string     `json:"embedding_status,omitempty"`
}

// EmbeddingConfigProfile is the active embedder configuration used for staleness checks.
type EmbeddingConfigProfile struct {
	Model      string
	Provider   string
	Dimension  int
}

// ProfileFromSemanticConfig builds a profile from semantic retrieval YAML.
func ProfileFromSemanticConfig(cfg *SemanticRetrievalConfig) EmbeddingConfigProfile {
	p := EmbeddingConfigProfile{Provider: "http"}
	if cfg == nil {
		return p
	}
	p.Model = strings.TrimSpace(cfg.EmbeddingModel)
	if p.Model == "" {
		p.Model = "text-embedding-3-small"
	}
	p.Dimension = cfg.EmbeddingDimensions
	if p.Dimension <= 0 {
		p.Dimension = DefaultEmbeddingDimensions
	}
	return p
}

// ComputeEmbeddingSourceHash hashes the exact text embedded for a memory.
// Documented input: EmbeddingTextForMemory(kind, statementCanonical, statement).
func ComputeEmbeddingSourceHash(kind api.MemoryKind, statementCanonical, statement string) string {
	text := EmbeddingTextForMemory(kind, statementCanonical, statement)
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// NewEmbeddingWriteMeta builds metadata for a freshly computed embedding.
func NewEmbeddingWriteMeta(kind api.MemoryKind, statementCanonical, statement string, provider, model string, dim int) EmbeddingWriteMeta {
	now := time.Now().UTC()
	if strings.TrimSpace(provider) == "" {
		provider = "http"
	}
	if strings.TrimSpace(model) == "" {
		model = "text-embedding-3-small"
	}
	return EmbeddingWriteMeta{
		Model:      model,
		Provider:   provider,
		Dimension:  dim,
		SourceHash: ComputeEmbeddingSourceHash(kind, statementCanonical, statement),
		Status:     EmbeddingStatusValid,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// StalenessReason explains why an embedding is not trustworthy.
type StalenessReason string

const (
	StalenessNone              StalenessReason = ""
	StalenessMissingEmbedding  StalenessReason = "missing_embedding"
	StalenessMissingMetadata   StalenessReason = "missing_metadata"
	StalenessSourceHashMismatch StalenessReason = "source_hash_mismatch"
	StalenessModelMismatch     StalenessReason = "model_mismatch"
	StalenessDimensionMismatch StalenessReason = "dimension_mismatch"
	StalenessStatusInvalid     StalenessReason = "status_invalid"
)

// CheckEmbeddingStalenessWithVector detects staleness including absent stored vectors.
func CheckEmbeddingStalenessWithVector(hasVector bool, obj MemoryObject, meta EmbeddingMeta, profile EmbeddingConfigProfile) StalenessReason {
	if !hasVector {
		return StalenessMissingEmbedding
	}
	return CheckEmbeddingStaleness(obj, meta, profile)
}

// CheckEmbeddingStaleness returns whether the stored embedding metadata is stale vs current memory text and config.
func CheckEmbeddingStaleness(obj MemoryObject, meta EmbeddingMeta, profile EmbeddingConfigProfile) StalenessReason {
	if meta.Status == EmbeddingStatusStale || meta.Status == EmbeddingStatusFailed {
		return StalenessStatusInvalid
	}
	if meta.SourceHash == "" {
		return StalenessMissingMetadata
	}
	wantHash := ComputeEmbeddingSourceHash(obj.Kind, obj.StatementCanonical, obj.Statement)
	if meta.SourceHash != wantHash {
		return StalenessSourceHashMismatch
	}
	if profile.Model != "" && meta.Model != "" && meta.Model != profile.Model {
		return StalenessModelMismatch
	}
	if profile.Dimension > 0 && meta.Dimension > 0 && meta.Dimension != profile.Dimension {
		return StalenessDimensionMismatch
	}
	return StalenessNone
}

// IsEmbeddingSearchable returns true when a row should participate in vector search.
func IsEmbeddingSearchable(meta EmbeddingMeta, obj MemoryObject, profile EmbeddingConfigProfile) bool {
	if meta.Status == EmbeddingStatusStale || meta.Status == EmbeddingStatusFailed {
		return false
	}
	return CheckEmbeddingStaleness(obj, meta, profile) == StalenessNone
}
