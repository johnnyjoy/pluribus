// Package lexical implements optional BM25 retrieval via pg_textsearch (experimental).
// Canonical memory rows in `memories` remain the source of truth; see docs/experiments/pg-textsearch.md.
package lexical

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"

	"github.com/lib/pq"
)

// Identifier pattern for projection table names (SQL injection guard — dynamic FROM clause).
var identifierOK = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// Hit is one BM25-ranked row from the lexical projection table.
type Hit struct {
	MemoryID string  `json:"memory_id"`
	NegScore float64 `json:"neg_bm25_score"`
}

// DefaultProjectionTable is used when config leaves projection_table empty.
const DefaultProjectionTable = "lexical_memory_projection"

// ValidateProjectionTable returns an error if name is not a safe SQL identifier.
func ValidateProjectionTable(name string) error {
	if name == "" {
		return fmt.Errorf("lexical: empty projection table name")
	}
	if !identifierOK.MatchString(name) {
		return fmt.Errorf("lexical: invalid projection table name %q (use lowercase letters, digits, underscore)", name)
	}
	return nil
}

// Search runs BM25 ordering using pg_textsearch on column doc_text.
// Table must exist and use a bm25 index on doc_text; extension pg_textsearch must be enabled in the database.
func Search(ctx context.Context, db *sql.DB, projectionTable, query string, limit int) ([]Hit, error) {
	if err := ValidateProjectionTable(projectionTable); err != nil {
		return nil, err
	}
	if query == "" {
		return nil, fmt.Errorf("lexical: empty query")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 500 {
		limit = 500
	}
	t := pq.QuoteIdentifier(projectionTable)
	// pg_textsearch <@> with lib/pq prepared parameters can return empty result sets; use a single
	// quoted literal for both score and ORDER BY (validated projection table; query escaped as SQL string).
	qLit := pq.QuoteLiteral(query)
	q := fmt.Sprintf(`
		SELECT memory_id::text, (doc_text <@> %s) AS neg_score
		FROM %s
		ORDER BY doc_text <@> %s
		LIMIT %d`, qLit, t, qLit, limit)
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Hit
	for rows.Next() {
		var h Hit
		if err := rows.Scan(&h.MemoryID, &h.NegScore); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
