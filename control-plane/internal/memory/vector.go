package memory

import (
	"fmt"
	"strings"
)

// DefaultEmbeddingDimensions matches migrations/0027_memory_embedding_pgvector.sql (vector(1536)).
const DefaultEmbeddingDimensions = 1536

// FormatVectorLiteral returns a pgvector literal string for casting to ::vector in SQL.
func FormatVectorLiteral(v []float32) string {
	if len(v) == 0 {
		return "[]"
	}
	var b strings.Builder
	b.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%.8g", f)
	}
	b.WriteByte(']')
	return b.String()
}
