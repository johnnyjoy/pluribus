package memory

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
)

// BackfillResult summarizes one embedding backfill run.
type BackfillResult struct {
	Scanned   int      `json:"scanned"`
	Embedded  int      `json:"embedded"`
	Failed    int      `json:"failed"`
	Skipped   int      `json:"skipped"`
	Remaining int      `json:"remaining"`
	Errors    []string `json:"errors,omitempty"`
}

const backfillMaxErrors = 10

// BackfillEmbeddings embeds active memories whose stored vector is missing or
// stale for the configured embedder profile (semantic-local remediation:
// write-time embedding covers new rows; this covers rows created before the
// embedder was enabled or under a previous model).
func (s *Service) BackfillEmbeddings(ctx context.Context, limit int) (*BackfillResult, error) {
	res := &BackfillResult{}
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("memory service unavailable")
	}
	if s.Semantic == nil || !s.Semantic.RetrievalEnabled() {
		return nil, fmt.Errorf("semantic retrieval is disabled; enable recall.semantic_retrieval first")
	}
	if s.Embedder == nil || s.Embedder.Dimensions() == 0 {
		return nil, fmt.Errorf("no embedder configured")
	}
	profile := ProfileFromSemanticConfig(s.Semantic)
	if s.Embedder.Dimensions() != profile.Dimension {
		return nil, fmt.Errorf("embedder dimensions %d do not match configured %d", s.Embedder.Dimensions(), profile.Dimension)
	}
	if limit <= 0 {
		limit = 200
	}

	rows, remaining, err := s.Repo.ListEmbeddingBackfillCandidates(ctx, profile, limit)
	if err != nil {
		return nil, err
	}
	res.Remaining = remaining
	for _, obj := range rows {
		res.Scanned++
		txt := EmbeddingTextForMemory(obj.Kind, obj.StatementCanonical, obj.Statement)
		if strings.TrimSpace(txt) == "" {
			res.Skipped++
			continue
		}
		vec, err := s.Embedder.Embed(ctx, txt)
		if err != nil || len(vec) != profile.Dimension {
			res.Failed++
			if err == nil {
				err = fmt.Errorf("got dim %d want %d", len(vec), profile.Dimension)
			}
			if len(res.Errors) < backfillMaxErrors {
				res.Errors = append(res.Errors, obj.ID.String()+": "+err.Error())
			}
			continue
		}
		model, provider := profile.Model, profile.Provider
		if he, ok := s.Embedder.(*HTTPEmbedder); ok {
			model, provider = he.ModelName(), he.ProviderName()
		}
		meta := NewEmbeddingWriteMeta(obj.Kind, obj.StatementCanonical, obj.Statement, provider, model, len(vec))
		if err := s.Repo.UpdateEmbedding(ctx, obj.ID, vec, meta); err != nil {
			res.Failed++
			if len(res.Errors) < backfillMaxErrors {
				res.Errors = append(res.Errors, obj.ID.String()+": "+err.Error())
			}
			continue
		}
		res.Embedded++
	}
	if res.Remaining >= res.Embedded {
		res.Remaining -= res.Embedded
	}
	slog.Info("[EMBED BACKFILL]", "scanned", res.Scanned, "embedded", res.Embedded,
		"failed", res.Failed, "skipped", res.Skipped, "remaining", res.Remaining)
	return res, nil
}

// ListEmbeddingBackfillCandidates returns active memories whose embedding is
// missing, unhashed, or written by a different model/dimension than profile,
// plus the total count of such rows (for progress reporting).
func (r *Repo) ListEmbeddingBackfillCandidates(ctx context.Context, profile EmbeddingConfigProfile, limit int) ([]MemoryObject, int, error) {
	where := `status = 'active' AND (
		embedding IS NULL
		OR embedding_source_hash IS NULL
		OR COALESCE(embedding_status, '') <> 'valid'
		OR ($1 <> '' AND COALESCE(embedding_model, '') <> $1)
		OR ($2 > 0 AND COALESCE(embedding_dimension, 0) <> $2)
	)`
	var total int
	if err := r.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM memories WHERE `+where, profile.Model, profile.Dimension,
	).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, kind, statement, statement_canonical FROM memories WHERE `+where+` ORDER BY created_at DESC LIMIT $3`,
		profile.Model, profile.Dimension, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []MemoryObject
	for rows.Next() {
		var obj MemoryObject
		if err := rows.Scan(&obj.ID, &obj.Kind, &obj.Statement, &obj.StatementCanonical); err != nil {
			return nil, 0, err
		}
		out = append(out, obj)
	}
	return out, total, rows.Err()
}

// UpdateEmbedding stores a freshly computed vector plus its metadata for one memory.
func (r *Repo) UpdateEmbedding(ctx context.Context, id uuid.UUID, vec []float32, meta EmbeddingWriteMeta) error {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE memories SET
			embedding = $1::vector,
			embedding_model = $2,
			embedding_provider = $3,
			embedding_dimension = $4,
			embedding_source_hash = $5,
			embedding_created_at = COALESCE(embedding_created_at, $6),
			embedding_updated_at = $7,
			embedding_status = $8
		 WHERE id = $9`,
		FormatVectorLiteral(vec), meta.Model, meta.Provider, meta.Dimension,
		meta.SourceHash, meta.CreatedAt, meta.UpdatedAt, meta.Status, id)
	return err
}
